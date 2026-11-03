package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Calevin/go_htmx_crud/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

var testMiddlewareSecret = []byte("middleware-test-jwt-secret")

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

// TestAuthMiddleware_NoToken_Returns303 verifica redireccion 303 a /login si la peticion carece de token (RF-06).
func TestAuthMiddleware_NoToken_Returns303(t *testing.T) {
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(testMiddlewareSecret)(spy)

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusSeeOther, recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != "/login" {
		t.Fatalf("se esperaba redireccion a /login pero se obtuvo %s", location)
	}
	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado sin token")
	}
	if strings.Contains(recorder.Body.String(), "contenido-protegido-secreto") {
		t.Fatal("la respuesta no debe incluir contenido protegido")
	}
}

// TestAuthMiddleware_InvalidToken_Returns303 verifica redireccion 303 si el token es estructuralmente invalido (RF-06).
func TestAuthMiddleware_InvalidToken_Returns303(t *testing.T) {
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(testMiddlewareSecret)(spy)

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	req.AddCookie(&http.Cookie{
		Name:  "token",
		Value: "token-invalido-malformado",
		Path:  "/",
	})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusSeeOther, recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != "/login" {
		t.Fatalf("se esperaba redireccion a /login pero se obtuvo %s", location)
	}
	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado con token invalido")
	}
	if strings.Contains(recorder.Body.String(), "contenido-protegido-secreto") {
		t.Fatal("la respuesta no debe incluir contenido protegido")
	}
}

// TestAuthMiddleware_WrongSecret_Returns303 verifica redireccion 303 si el token fue firmado con otro secreto (RF-06).
func TestAuthMiddleware_WrongSecret_Returns303(t *testing.T) {
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(testMiddlewareSecret)(spy)

	wrongSecret := []byte("clave-secreta-totalmente-distinta")
	tokenString, err := auth.GenerateJWT("testuser", wrongSecret)
	if err != nil {
		t.Fatalf("error generando token con clave ajena: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	req.AddCookie(&http.Cookie{
		Name:  "token",
		Value: tokenString,
		Path:  "/",
	})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusSeeOther, recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != "/login" {
		t.Fatalf("se esperaba redireccion a /login pero se obtuvo %s", location)
	}
	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado con firma ajena")
	}
	if strings.Contains(recorder.Body.String(), "contenido-protegido-secreto") {
		t.Fatal("la respuesta no debe incluir contenido protegido")
	}
}

// TestAuthMiddleware_ExpiredToken_Returns303 verifica redireccion 303 si el token ha expirado (RF-06, CL-03).
func TestAuthMiddleware_ExpiredToken_Returns303(t *testing.T) {
	spy, called := newProtectedSpyHandler()
	handler := Authenticator(testMiddlewareSecret)(spy)

	claims := &auth.Claims{
		Username: "expired_user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := expiredToken.SignedString(testMiddlewareSecret)
	if err != nil {
		t.Fatalf("error firmando token expirado: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/notas", nil)
	req.AddCookie(&http.Cookie{
		Name:  "token",
		Value: tokenString,
		Path:  "/",
	})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusSeeOther, recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != "/login" {
		t.Fatalf("se esperaba redireccion a /login pero se obtuvo %s", location)
	}
	if *called {
		t.Fatal("el handler protegido no debio haberse ejecutado con token expirado")
	}
	if strings.Contains(recorder.Body.String(), "contenido-protegido-secreto") {
		t.Fatal("la respuesta no debe incluir contenido protegido")
	}
}
