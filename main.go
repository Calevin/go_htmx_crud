package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"

	"github.com/Calevin/go_htmx_crud/database"
	"github.com/Calevin/go_htmx_crud/internal/auth"
	"github.com/Calevin/go_htmx_crud/internal/db"
	"github.com/Calevin/go_htmx_crud/internal/handlers"
	authMiddleware "github.com/Calevin/go_htmx_crud/internal/middleware"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS
var tpl *template.Template

func init() {
	funcMap := template.FuncMap{
		"include": func(templateName string, data any) (template.HTML, error) {
			var buf strings.Builder
			err := tpl.ExecuteTemplate(&buf, templateName, data)
			return template.HTML(buf.String()), err
		},
	}
	tpl = template.New("").Funcs(funcMap)
	tpl = template.Must(tpl.ParseFS(templateFS, "templates/*.html"))
}

func main() {
	// Cargar variables de entorno desde el archivo .env
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró el archivo .env, usando variables de entorno del sistema.")
	}

	idpHodorURL := os.Getenv("IDP_HODOR_URL")
	if idpHodorURL == "" {
		idpHodorURL = "http://localhost:8080"
	}

	oauthClientID := os.Getenv("OAUTH_CLIENT_ID")
	if oauthClientID == "" {
		oauthClientID = "go-htmx-crud"
	}

	oauthClientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
	if oauthClientSecret == "" {
		oauthClientSecret = "go-crud-secret"
	}

	redirectURI := os.Getenv("OAUTH_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://localhost:3000/auth/callback"
	}

	ctx := context.Background()

	// Inicializar el validador de tokens RS256 contra el JWKS del IdP (fail-fast)
	jwksURL := strings.TrimRight(idpHodorURL, "/") + "/.well-known/jwks.json"
	validator, err := auth.NewTokenValidator(ctx, jwksURL, idpHodorURL, oauthClientID)
	if err != nil {
		log.Fatalf("Error al inicializar el validador de tokens contra Hodor (%s): %v", jwksURL, err)
	}

	// Configuracion del cliente OAuth2 con PKCE
	oauthConfig := &oauth2.Config{
		ClientID:     oauthClientID,
		ClientSecret: oauthClientSecret,
		RedirectURL:  redirectURI,
		Endpoint: oauth2.Endpoint{
			AuthURL:  strings.TrimRight(idpHodorURL, "/") + "/oauth2/authorize",
			TokenURL: strings.TrimRight(idpHodorURL, "/") + "/oauth2/token",
		},
		Scopes: []string{"openid", "profile"},
	}
	authHandler := handlers.NewAuthHandler(oauthConfig, validator)

	// Se iniciala la base de datos. Esto creará el archivo 'crud.db' en la raíz.
	conn := database.InitDB("./crud.db")
	defer conn.Close()

	// Crea una instancia de `Queries` generada por sqlc.
	queries := db.New(conn)

	// Instancia del router Chi
	r := chi.NewRouter()
	// Middleware que loguea las peticiones en la consola
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Servidor de archivos estaticos para CSS y assets
	r.Handle("/static/*", http.FileServer(http.FS(staticFS)))

	// --- Rutas Publicas ---
	// Inicio de sesion federado (redireccion OIDC a Hodor con PKCE)
	r.Get("/login", authHandler.Login)
	r.Get("/auth/login", authHandler.Login)

	// Retorno de autorizacion OIDC
	r.Get("/auth/callback", authHandler.Callback)

	// Cierre de sesion local
	r.Post("/logout", authHandler.Logout)
	r.Get("/auth/logout", authHandler.Logout)

	// --- Rutas Protegidas ---
	// Grupo de rutas que usarán el middleware de autenticación
	r.Group(func(r chi.Router) {
		// Middleware de autenticacion con validacion RS256 y soporte HTMX
		r.Use(authMiddleware.Authenticator(validator))

		// GET /notas renderiza la página de notas.
		r.Get("/notas", func(w http.ResponseWriter, r *http.Request) {
			handlers.ListNotesHandler(w, r, tpl, queries)
		})

		// GET /crear_nota para mostrar el formulario
		r.Get("/crear_nota", func(w http.ResponseWriter, r *http.Request) {
			handlers.CreateNoteFormHandler(w, r, tpl, queries)
		})

		// POST /crear_nota para procesar el formulario
		r.Post("/crear_nota", func(w http.ResponseWriter, r *http.Request) {
			handlers.CreateNoteHandler(w, r, queries)
		})

		// DELETE /borrar_nota/{id} para borrar una nota
		r.Delete("/borrar_nota/{id}", func(w http.ResponseWriter, r *http.Request) {
			handlers.DeleteNoteHandler(w, r, queries)
		})

		// GET /editar_nota/{id} para mostrar el formulario de edición
		r.Get("/editar_nota/{id}", func(w http.ResponseWriter, r *http.Request) {
			handlers.EditNoteFormHandler(w, r, tpl, queries)
		})

		// POST /editar_nota/{id} para procesar el formulario de edición
		r.Post("/editar_nota/{id}", func(w http.ResponseWriter, r *http.Request) {
			handlers.UpdateNoteHandler(w, r, queries)
		})
	})

	// Redireccion raiz a /notas
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/notas", http.StatusFound)
	})

	// Se inicia el servidor en el puerto 3000
	log.Println("Servidor iniciado en http://localhost:3000")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}
