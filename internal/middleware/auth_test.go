package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Calevin/go_htmx_crud/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

type middlewareMockIdP struct {
	server     *httptest.Server
	privateKey *rsa.PrivateKey
	keyID      string
	issuer     string
}

func newMiddlewareMockIdP(t *testing.T, keyID string) *middlewareMockIdP {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("error al generar clave RSA para test: %v", err)
	}

	mock := &middlewareMockIdP{
		privateKey: privateKey,
		keyID:      keyID,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		pub := &privateKey.PublicKey
		nBytes := pub.N.Bytes()
		eBytes := big.NewInt(int64(pub.E)).Bytes()

		jwk := map[string]any{
			"kty": "RSA",
			"kid": keyID,
			"use": "sig",
			"alg": "RS256",
			"n":   base64.RawURLEncoding.EncodeToString(nBytes),
			"e":   base64.RawURLEncoding.EncodeToString(eBytes),
		}

		resp := map[string]any{
			"keys": []any{jwk},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	mock.server = server
	mock.issuer = server.URL

	t.Cleanup(func() {
		server.Close()
	})

	return mock
}

func (m *middlewareMockIdP) generateToken(claims auth.HodorClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = m.keyID
	tokenStr, _ := token.SignedString(m.privateKey)
	return tokenStr
}

// newProtectedSpyHandler retorna un handler de prueba que registra si fue ejecutado o no.
func newProtectedSpyHandler() (http.Handler, *bool) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("contenido-protegido-secreto"))
	})
	return handler, &called
}

func setupTestValidator(t *testing.T, clientID string) (*auth.TokenValidator, *middlewareMockIdP) {
	t.Helper()
	idp := newMiddlewareMockIdP(t, "mw-key-1")
	validator, err := auth.NewTokenValidator(context.Background(), idp.server.URL+"/.well-known/jwks.json", idp.issuer, clientID)
	if err != nil {
		t.Fatalf("error inicializando validador para test: %v", err)
	}
	return validator, idp
}

// TestAuthMiddleware_HTMXRequest_Returns401WithHXRedirect verifica respuesta ante peticion HTMX sin sesion.
func TestAuthMiddleware_HTMXRequest_Returns401WithHXRedirect(t *testing.T) {
	validator, _ := setupTestValidator(t, "go-htmx-crud")
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(validator)(spy)

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	req.Header.Set("HX-Request", "true")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusUnauthorized, recorder.Code)
	}

	redirect := recorder.Header().Get("HX-Redirect")
	if redirect != "/login" && redirect != "/auth/login" {
		t.Fatalf("se esperaba cabecera HX-Redirect hacia login pero se obtuvo %s", redirect)
	}

	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado sin sesion")
	}
}

// TestAuthMiddleware_StandardRequest_Returns303ToLogin verifica respuesta ante navegacion ordinaria sin sesion.
func TestAuthMiddleware_StandardRequest_Returns303ToLogin(t *testing.T) {
	validator, _ := setupTestValidator(t, "go-htmx-crud")
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(validator)(spy)

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusSeeOther, recorder.Code)
	}

	location := recorder.Header().Get("Location")
	if location != "/login" && location != "/auth/login" {
		t.Fatalf("se esperaba Location hacia login pero se obtuvo %s", location)
	}

	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado sin sesion")
	}
}

// TestAuthMiddleware_StandardRequest_RedirectsToLogin verifica redireccion 303 hacia /login para spec 004 (RF-09).
func TestAuthMiddleware_StandardRequest_RedirectsToLogin(t *testing.T) {
	validator, _ := setupTestValidator(t, "go-htmx-crud")
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(validator)(spy)

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusSeeOther, recorder.Code)
	}

	if location := recorder.Header().Get("Location"); location != "/login" {
		t.Fatalf("se esperaba Location /login pero se obtuvo %s", location)
	}

	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado sin sesion")
	}
}

// TestAuthMiddleware_HTMXRequest_Returns401WithHXRedirectToLogin verifica respuesta HTMX con HX-Redirect a /login (RF-10).
func TestAuthMiddleware_HTMXRequest_Returns401WithHXRedirectToLogin(t *testing.T) {
	validator, _ := setupTestValidator(t, "go-htmx-crud")
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(validator)(spy)

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	req.Header.Set("HX-Request", "true")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusUnauthorized, recorder.Code)
	}

	if redirect := recorder.Header().Get("HX-Redirect"); redirect != "/login" {
		t.Fatalf("se esperaba cabecera HX-Redirect: /login pero se obtuvo %s", redirect)
	}

	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado sin sesion")
	}
}

// TestAuthMiddleware_UnauthorizedSystem_Returns403 verifica que usuario sin sistema asignado reciba 403 (RF-08).
func TestAuthMiddleware_UnauthorizedSystem_Returns403(t *testing.T) {
	clientID := "go-htmx-crud"
	validator, idp := setupTestValidator(t, clientID)
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(validator)(spy)

	// Token valido pero sin go-htmx-crud en authorized_systems
	tokenStr := idp.generateToken(auth.HodorClaims{
		Username:          "unauthorized_user",
		AuthorizedSystems: []string{"other-system-only"},
		Roles:             []string{"ROLE_USER"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    idp.issuer,
			Subject:   "unauthorized_user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: tokenStr, Path: "/"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusForbidden, recorder.Code)
	}

	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado para usuario no autorizado")
	}
}

// TestAuthMiddleware_NoToken_Returns303 verifica redireccion 303 si la peticion carece de token (compatibilidad).
func TestAuthMiddleware_NoToken_Returns303(t *testing.T) {
	validator, _ := setupTestValidator(t, "go-htmx-crud")
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(validator)(spy)

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusSeeOther, recorder.Code)
	}
	location := recorder.Header().Get("Location")
	if location != "/auth/login" && location != "/login" {
		t.Fatalf("se esperaba redireccion a login pero se obtuvo %s", location)
	}
	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado sin token")
	}
}

// TestAuthMiddleware_InvalidToken_Returns303 verifica redireccion 303 si el token es estructuralmente invalido.
func TestAuthMiddleware_InvalidToken_Returns303(t *testing.T) {
	validator, _ := setupTestValidator(t, "go-htmx-crud")
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(validator)(spy)

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "token-invalido-malformado", Path: "/"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusSeeOther, recorder.Code)
	}
	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado con token invalido")
	}
}

// TestAuthMiddleware_WrongSecret_Returns303 verifica redireccion 303 si el token fue firmado con otra clave.
func TestAuthMiddleware_WrongSecret_Returns303(t *testing.T) {
	validator, idp := setupTestValidator(t, "go-htmx-crud")
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(validator)(spy)

	// Token firmado con clave ajena no registrada en el IdP
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, auth.HodorClaims{
		Username:          "intruder",
		AuthorizedSystems: []string{"go-htmx-crud"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    idp.issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	token.Header["kid"] = idp.keyID
	tokenStr, _ := token.SignedString(otherKey)

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: tokenStr, Path: "/"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusSeeOther, recorder.Code)
	}
	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado con clave ajena")
	}
}

// TestAuthMiddleware_ExpiredToken_Returns303 verifica redireccion 303 si el token esta caducado.
func TestAuthMiddleware_ExpiredToken_Returns303(t *testing.T) {
	validator, idp := setupTestValidator(t, "go-htmx-crud")
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(validator)(spy)

	tokenStr := idp.generateToken(auth.HodorClaims{
		Username:          "expired_user",
		AuthorizedSystems: []string{"go-htmx-crud"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    idp.issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: tokenStr, Path: "/"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusSeeOther, recorder.Code)
	}
	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado con token caducado")
	}
}

// TestAuthMiddleware_ValidAuthorizedToken_Passes verifica paso exitoso e inyeccion de claims en el contexto.
func TestAuthMiddleware_ValidAuthorizedToken_Passes(t *testing.T) {
	clientID := "go-htmx-crud"
	validator, idp := setupTestValidator(t, clientID)

	var receivedClaims *auth.HodorClaims
	handler := Authenticator(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		val := r.Context().Value(UserContextKey)
		if claims, ok := val.(*auth.HodorClaims); ok {
			receivedClaims = claims
		}
		w.WriteHeader(http.StatusOK)
	}))

	tokenStr := idp.generateToken(auth.HodorClaims{
		Username:          "valid_user",
		AuthorizedSystems: []string{clientID},
		Roles:             []string{"ROLE_USER"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    idp.issuer,
			Subject:   "valid_user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: tokenStr, Path: "/"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusOK, recorder.Code)
	}
	if receivedClaims == nil || receivedClaims.Username != "valid_user" {
		t.Fatal("no se inyectaron los claims validos en el contexto")
	}
}
