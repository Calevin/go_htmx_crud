package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Calevin/go_htmx_crud/internal/auth"
	"github.com/Calevin/go_htmx_crud/internal/db"
	"github.com/Calevin/go_htmx_crud/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// defaultTestSecret se conserva por compatibilidad de tipos con helpers historicos.
var defaultTestSecret = []byte("test-jwt-secret-key-for-unit-testing")

var (
	testIdPMutex    sync.Mutex
	testIdPServer   *httptest.Server
	testIdPPrivKey  *rsa.PrivateKey
	testIdPKeyID    = "test-handlers-key-1"
	testIdPIssuer   string
	testValidator   *auth.TokenValidator
	testIdPInitOnce sync.Once
)

func ensureTestIdP() {
	testIdPInitOnce.Do(func() {
		var err error
		testIdPPrivKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic("failed to generate rsa key for handlers test: " + err.Error())
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
			pub := &testIdPPrivKey.PublicKey
			nBytes := pub.N.Bytes()
			eBytes := big.NewInt(int64(pub.E)).Bytes()
			jwk := map[string]any{
				"kty": "RSA",
				"kid": testIdPKeyID,
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(nBytes),
				"e":   base64.RawURLEncoding.EncodeToString(eBytes),
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{jwk}})
		})

		testIdPServer = httptest.NewServer(mux)
		testIdPIssuer = testIdPServer.URL

		testValidator, err = auth.NewTokenValidator(context.Background(), testIdPServer.URL+"/.well-known/jwks.json", testIdPIssuer, "go-htmx-crud")
		if err != nil {
			panic("failed to initialize testValidator: " + err.Error())
		}
	})
}

// newTestDB inicializa una base de datos SQLite aislada en un directorio temporal y aplica el esquema.
func newTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("error abriendo base de datos de prueba: %v", err)
	}

	schemaBytes, err := os.ReadFile("../../sql/schema/schema.sql")
	if err != nil {
		_ = conn.Close()
		t.Fatalf("error leyendo archivo schema.sql: %v", err)
	}

	if _, err := conn.Exec(string(schemaBytes)); err != nil {
		_ = conn.Close()
		t.Fatalf("error ejecutando schema.sql: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
	})

	return conn, db.New(conn)
}

// newTestTpl carga los templates HTML desde el disco replicando la configuracion de main.go.
func newTestTpl(t *testing.T) *template.Template {
	t.Helper()

	var tpl *template.Template
	funcMap := template.FuncMap{
		"include": func(templateName string, data any) (template.HTML, error) {
			var buf strings.Builder
			err := tpl.ExecuteTemplate(&buf, templateName, data)
			return template.HTML(buf.String()), err
		},
	}

	tpl = template.New("").Funcs(funcMap)
	var err error
	tpl, err = tpl.ParseGlob("../../templates/*.html")
	if err != nil {
		t.Fatalf("error parseando templates: %v", err)
	}

	return tpl
}

// newTestRouter construye un router Chi que refleja las rutas definidas en main.go.
func newTestRouter(queries *db.Queries, tpl *template.Template, _ []byte) *chi.Mux {
	ensureTestIdP()
	r := chi.NewRouter()

	// Rutas publicas
	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		Render(tpl, w, "login.html", nil)
	})

	// Rutas protegidas bajo el middleware de autenticacion
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticator(testValidator))

		r.Get("/notas", func(w http.ResponseWriter, r *http.Request) {
			ListNotesHandler(w, r, tpl, queries)
		})
		r.Post("/logout", LogoutHandler())
		r.Get("/crear_nota", func(w http.ResponseWriter, r *http.Request) {
			CreateNoteFormHandler(w, r, tpl, queries)
		})
		r.Post("/crear_nota", func(w http.ResponseWriter, r *http.Request) {
			CreateNoteHandler(w, r, queries)
		})
		r.Delete("/borrar_nota/{id}", func(w http.ResponseWriter, r *http.Request) {
			DeleteNoteHandler(w, r, queries)
		})
		r.Get("/editar_nota/{id}", func(w http.ResponseWriter, r *http.Request) {
			EditNoteFormHandler(w, r, tpl, queries)
		})
		r.Post("/editar_nota/{id}", func(w http.ResponseWriter, r *http.Request) {
			UpdateNoteHandler(w, r, queries)
		})
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/notas", http.StatusFound)
	})

	return r
}

// createTestUser inserta un usuario en la base de datos de prueba con su clave hasheada mediante bcrypt.
func createTestUser(t *testing.T, ctx context.Context, queries *db.Queries, username, password string) db.User {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("error hasheando password para usuario de prueba: %v", err)
	}

	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Username:     username,
		PasswordHash: string(hashedPassword),
	})
	if err != nil {
		t.Fatalf("error creando usuario de prueba: %v", err)
	}

	return user
}

// createTestTag inserta una etiqueta en la base de datos de prueba.
func createTestTag(t *testing.T, ctx context.Context, queries *db.Queries, name, color string) db.Tag {
	t.Helper()

	tag, err := queries.CreateTag(ctx, db.CreateTagParams{
		Nombre: name,
		Color: sql.NullString{
			String: color,
			Valid:  color != "",
		},
	})
	if err != nil {
		t.Fatalf("error creando tag de prueba: %v", err)
	}

	return tag
}

// createTestNote inserta una nota en la base de datos de prueba y opcionalmente la vincula con una etiqueta.
func createTestNote(t *testing.T, ctx context.Context, queries *db.Queries, title, content string, tagID int64) db.Note {
	t.Helper()

	note, err := queries.CreateNote(ctx, db.CreateNoteParams{
		Nombre: title,
		Contenido: sql.NullString{
			String: content,
			Valid:  content != "",
		},
	})
	if err != nil {
		t.Fatalf("error creando nota de prueba: %v", err)
	}

	if tagID > 0 {
		err = queries.LinkTagToNote(ctx, db.LinkTagToNoteParams{
			NoteID: note.ID,
			TagID:  tagID,
		})
		if err != nil {
			t.Fatalf("error vinculando tag a nota de prueba: %v", err)
		}
	}

	return note
}

// withValidAuthCookie adjunta la cookie 'token' con un JWT RS256 valido para el cliente go-htmx-crud.
func withValidAuthCookie(t *testing.T, req *http.Request, username string, _ []byte) {
	t.Helper()
	ensureTestIdP()

	testIdPMutex.Lock()
	defer testIdPMutex.Unlock()

	claims := auth.HodorClaims{
		Username:          username,
		AuthorizedSystems: []string{"go-htmx-crud"},
		Roles:             []string{"ROLE_USER"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIdPIssuer,
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testIdPKeyID
	tokenString, err := token.SignedString(testIdPPrivKey)
	if err != nil {
		t.Fatalf("error generando JWT RS256 de prueba: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
	})
}

// newExpiredToken genera un JWT expirado.
func newExpiredToken(t *testing.T, username string, _ []byte) string {
	t.Helper()
	ensureTestIdP()

	testIdPMutex.Lock()
	defer testIdPMutex.Unlock()

	claims := auth.HodorClaims{
		Username:          username,
		AuthorizedSystems: []string{"go-htmx-crud"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIdPIssuer,
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testIdPKeyID
	tokenString, err := token.SignedString(testIdPPrivKey)
	if err != nil {
		t.Fatalf("error firmando token expirado de prueba: %v", err)
	}

	return tokenString
}
