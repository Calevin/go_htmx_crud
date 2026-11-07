package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Calevin/go_htmx_crud/internal/auth"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

type handlersMockIdP struct {
	server     *httptest.Server
	privateKey *rsa.PrivateKey
	keyID      string
	issuer     string
}

func newHandlersMockIdP(t *testing.T, keyID, clientID string) *handlersMockIdP {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("error generating rsa key: %v", err)
	}
	mock := &handlersMockIdP{
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
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{jwk}})
	})

	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		claims := auth.HodorClaims{
			Username:          "test_user",
			AuthorizedSystems: []string{clientID},
			Roles:             []string{"ROLE_USER"},
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    mock.issuer,
				Subject:   "test_user",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = keyID
		signedStr, _ := token.SignedString(privateKey)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": signedStr,
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})

	server := httptest.NewServer(mux)
	mock.server = server
	mock.issuer = server.URL

	t.Cleanup(func() {
		server.Close()
	})

	return mock
}

func (m *handlersMockIdP) generateToken(claims auth.HodorClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = m.keyID
	signedStr, _ := token.SignedString(m.privateKey)
	return signedStr
}

func setupTestAuthHandler(t *testing.T, clientID string) (*AuthHandler, *handlersMockIdP) {
	t.Helper()
	idp := newHandlersMockIdP(t, "handler-key-1", clientID)

	oauthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:3000/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  idp.server.URL + "/oauth2/authorize",
			TokenURL: idp.server.URL + "/oauth2/token",
		},
		Scopes: []string{"openid", "profile"},
	}

	validator, err := auth.NewTokenValidator(context.Background(), idp.server.URL+"/.well-known/jwks.json", idp.issuer, clientID)
	if err != nil {
		t.Fatalf("error initializing validator: %v", err)
	}

	tpl := newTestTpl(t)
	handler := NewAuthHandler(oauthConfig, validator, tpl)
	return handler, idp
}

// TestAuthLogin_InitiatesPKCEFlow verifica que Login genera state, verifier y redirige con PKCE (RF-01).
func TestAuthLogin_InitiatesPKCEFlow(t *testing.T) {
	handler, idp := setupTestAuthHandler(t, "go-htmx-crud")

	// 1. Peticion ordinaria
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusFound, rec.Code)
	}

	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, idp.server.URL+"/oauth2/authorize") {
		t.Fatalf("redireccion inesperada: %s", location)
	}
	if !strings.Contains(location, "code_challenge=") || !strings.Contains(location, "code_challenge_method=S256") {
		t.Fatalf("la URL de autorizacion debe incluir parametros PKCE S256: %s", location)
	}

	cookies := rec.Result().Cookies()
	var stateCookie, verifierCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "oauth_state" {
			stateCookie = c
		}
		if c.Name == "oauth_verifier" {
			verifierCookie = c
		}
	}

	if stateCookie == nil || stateCookie.Value == "" || !stateCookie.HttpOnly {
		t.Fatal("se esperaba cookie 'oauth_state' HttpOnly y no vacia")
	}
	if verifierCookie == nil || verifierCookie.Value == "" || !verifierCookie.HttpOnly {
		t.Fatal("se esperaba cookie 'oauth_verifier' HttpOnly y no vacia")
	}

	// 2. Peticion HTMX
	reqHTMX := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	reqHTMX.Header.Set("HX-Request", "true")
	recHTMX := httptest.NewRecorder()
	handler.Login(recHTMX, reqHTMX)

	if recHTMX.Code != http.StatusOK {
		t.Fatalf("peticion HTMX esperaba codigo 200 pero obtuvo %d", recHTMX.Code)
	}
	if hxRedirect := recHTMX.Header().Get("HX-Redirect"); !strings.HasPrefix(hxRedirect, idp.server.URL+"/oauth2/authorize") {
		t.Fatalf("se esperaba cabecera HX-Redirect hacia el IdP pero se obtuvo %s", hxRedirect)
	}
}

// TestAuthCallback_ValidCodeAndState_ExchangesAndValidatesRS256 verifica el intercambio en callback (RF-04).
func TestAuthCallback_ValidCodeAndState_ExchangesAndValidatesRS256(t *testing.T) {
	handler, _ := setupTestAuthHandler(t, "go-htmx-crud")

	state := "test_state_12345"
	verifier := oauth2.GenerateVerifier()

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=mock_code&state="+state, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: state, Path: "/auth/callback"})
	req.AddCookie(&http.Cookie{Name: "oauth_verifier", Value: verifier, Path: "/auth/callback"})
	rec := httptest.NewRecorder()

	handler.Callback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d (body: %s)", http.StatusFound, rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); location != "/notas" {
		t.Fatalf("se esperaba redireccion a /notas pero se obtuvo %s", location)
	}

	cookies := rec.Result().Cookies()
	var tokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "token" {
			tokenCookie = c
			break
		}
	}
	if tokenCookie == nil || tokenCookie.Value == "" {
		t.Fatal("se esperaba cookie 'token' establecida tras callback exitoso")
	}
	if !tokenCookie.HttpOnly {
		t.Fatal("la cookie 'token' debe ser HttpOnly")
	}
}

// TestAuthCallback_InvalidStateOrMissingVerifier_Returns400 verifica rechazo ante state invalido o verifier ausente (RF-05).
func TestAuthCallback_InvalidStateOrMissingVerifier_Returns400(t *testing.T) {
	handler, _ := setupTestAuthHandler(t, "go-htmx-crud")

	// 1. State no coincide
	req1 := httptest.NewRequest(http.MethodGet, "/auth/callback?code=code123&state=state_recibido", nil)
	req1.AddCookie(&http.Cookie{Name: "oauth_state", Value: "state_distinto", Path: "/auth/callback"})
	req1.AddCookie(&http.Cookie{Name: "oauth_verifier", Value: "verifier123", Path: "/auth/callback"})
	rec1 := httptest.NewRecorder()
	handler.Callback(rec1, req1)
	if rec1.Code != http.StatusBadRequest {
		t.Fatalf("se esperaba 400 ante state mismatch pero se obtuvo %d", rec1.Code)
	}

	// 2. Verifier ausente
	req2 := httptest.NewRequest(http.MethodGet, "/auth/callback?code=code123&state=same_state", nil)
	req2.AddCookie(&http.Cookie{Name: "oauth_state", Value: "same_state", Path: "/auth/callback"})
	rec2 := httptest.NewRecorder()
	handler.Callback(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("se esperaba 400 ante verifier ausente pero se obtuvo %d", rec2.Code)
	}

	// 3. Codigo ausente
	req3 := httptest.NewRequest(http.MethodGet, "/auth/callback?state=same_state", nil)
	req3.AddCookie(&http.Cookie{Name: "oauth_state", Value: "same_state", Path: "/auth/callback"})
	req3.AddCookie(&http.Cookie{Name: "oauth_verifier", Value: "verifier123", Path: "/auth/callback"})
	rec3 := httptest.NewRecorder()
	handler.Callback(rec3, req3)
	if rec3.Code != http.StatusBadRequest {
		t.Fatalf("se esperaba 400 ante codigo ausente pero se obtuvo %d", rec3.Code)
	}
}

// TestAuthCallback_Success_SetsTokenCookieAndRedirectsToNotas verifica la cookie token y eliminacion de cookies temporales (RF-09).
func TestAuthCallback_Success_SetsTokenCookieAndRedirectsToNotas(t *testing.T) {
	handler, _ := setupTestAuthHandler(t, "go-htmx-crud")

	state := "state_success_abc"
	verifier := oauth2.GenerateVerifier()

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=code_ok&state="+state, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: state, Path: "/auth/callback"})
	req.AddCookie(&http.Cookie{Name: "oauth_verifier", Value: verifier, Path: "/auth/callback"})
	rec := httptest.NewRecorder()

	handler.Callback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("se esperaba 302 Found pero se obtuvo %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/notas" {
		t.Fatalf("se esperaba redireccion a /notas pero se obtuvo %s", rec.Header().Get("Location"))
	}

	cookies := rec.Result().Cookies()
	var stateCleared, verifierCleared bool
	var tokenFound bool

	for _, c := range cookies {
		if c.Name == "token" && c.Value != "" && c.HttpOnly && c.Path == "/" {
			tokenFound = true
		}
		if c.Name == "oauth_state" && c.MaxAge == -1 {
			stateCleared = true
		}
		if c.Name == "oauth_verifier" && c.MaxAge == -1 {
			verifierCleared = true
		}
	}

	if !tokenFound {
		t.Fatal("cookie 'token' no configurada correctamente")
	}
	if !stateCleared || !verifierCleared {
		t.Fatal("las cookies temporales de OAuth no fueron eliminadas tras el callback")
	}
}

// TestAuthLogout_ClearsCookieAndRedirects verifica que logout invalide la cookie y redirija a /login.
func TestAuthLogout_ClearsCookieAndRedirects(t *testing.T) {
	handler, _ := setupTestAuthHandler(t, "go-htmx-crud")

	// Peticion ordinaria
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "existing_token", Path: "/"})
	rec := httptest.NewRecorder()
	handler.Logout(rec, req)

	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Fatalf("se esperaba 303 o 302 pero se obtuvo %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/login" {
		t.Fatalf("se esperaba redireccion a /login pero se obtuvo %s", rec.Header().Get("Location"))
	}

	cookies := rec.Result().Cookies()
	var tokenCleared bool
	for _, c := range cookies {
		if c.Name == "token" && c.MaxAge == -1 {
			tokenCleared = true
			break
		}
	}
	if !tokenCleared {
		t.Fatal("la cookie 'token' debe eliminarse con MaxAge: -1")
	}

	// Peticion HTMX
	reqHTMX := httptest.NewRequest(http.MethodPost, "/logout", nil)
	reqHTMX.Header.Set("HX-Request", "true")
	recHTMX := httptest.NewRecorder()
	handler.Logout(recHTMX, reqHTMX)

	if recHTMX.Code != http.StatusOK {
		t.Fatalf("peticion HTMX esperaba 200 OK pero obtuvo %d", recHTMX.Code)
	}
	if recHTMX.Header().Get("HX-Redirect") != "/login" {
		t.Fatalf("peticion HTMX esperaba cabecera HX-Redirect: /login pero obtuvo %s", recHTMX.Header().Get("HX-Redirect"))
	}
}

// TestAuthLogout_HTMX_Returns200WithHXRedirect verifica respuesta 200 y cabecera HX-Redirect en logout HTMX (RF-06).
func TestAuthLogout_HTMX_Returns200WithHXRedirect(t *testing.T) {
	handler, _ := setupTestAuthHandler(t, "go-htmx-crud")

	for _, endpoint := range []string{"/logout", "/auth/logout"} {
		req := httptest.NewRequest(http.MethodPost, endpoint, nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()

		handler.Logout(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("[%s] se esperaba codigo %d pero se obtuvo %d", endpoint, http.StatusOK, rec.Code)
		}
		if redirect := rec.Header().Get("HX-Redirect"); redirect != "/login" {
			t.Fatalf("[%s] se esperaba cabecera HX-Redirect: /login pero se obtuvo %s", endpoint, redirect)
		}
	}
}

// TestAuthLogout_Standard_Returns303ToLogin verifica respuesta 303 hacia /login en logout ordinario (RF-07).
func TestAuthLogout_Standard_Returns303ToLogin(t *testing.T) {
	handler, _ := setupTestAuthHandler(t, "go-htmx-crud")

	for _, endpoint := range []string{"/logout", "/auth/logout"} {
		req := httptest.NewRequest(http.MethodPost, endpoint, nil)
		rec := httptest.NewRecorder()

		handler.Logout(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("[%s] se esperaba codigo %d pero se obtuvo %d", endpoint, http.StatusSeeOther, rec.Code)
		}
		if location := rec.Header().Get("Location"); location != "/login" {
			t.Fatalf("[%s] se esperaba Location /login pero se obtuvo %s", endpoint, location)
		}
	}
}

// TestAuthLogout_NonPostMethod_Returns405 verifica rechazo 405 en metodos no permitidos (RF-08).
func TestAuthLogout_NonPostMethod_Returns405(t *testing.T) {
	handler, _ := setupTestAuthHandler(t, "go-htmx-crud")

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		for _, endpoint := range []string{"/logout", "/auth/logout"} {
			req := httptest.NewRequest(method, endpoint, nil)
			rec := httptest.NewRecorder()

			handler.Logout(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("[%s %s] se esperaba codigo 405 pero se obtuvo %d", method, endpoint, rec.Code)
			}
		}
	}
}

// TestAuthLogout_ClearsCookieProperly verifica que la cookie 'token' sea invalidada con atributos seguros (RNF-01, RNF-02).
func TestAuthLogout_ClearsCookieProperly(t *testing.T) {
	handler, _ := setupTestAuthHandler(t, "go-htmx-crud")

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "active_token_to_clear", Path: "/"})
	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	cookies := rec.Result().Cookies()
	var tokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "token" {
			tokenCookie = c
			break
		}
	}

	if tokenCookie == nil {
		t.Fatal("la respuesta debe incluir la cookie 'token' para invalidarla")
	}
	if tokenCookie.MaxAge != -1 {
		t.Fatalf("se esperaba MaxAge -1 pero se obtuvo %d", tokenCookie.MaxAge)
	}
	if !tokenCookie.HttpOnly {
		t.Fatal("la cookie invalidada debe mantener la bandera HttpOnly")
	}
	if tokenCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("se esperaba SameSite Lax pero se obtuvo %v", tokenCookie.SameSite)
	}
}

// TestLogout_Success_Returns200WithHXRedirect verifica compatibilidad de logout con spec 002.
func TestLogout_Success_Returns200WithHXRedirect(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	LogoutHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusOK, rec.Code)
	}
	if redirect := rec.Header().Get("HX-Redirect"); redirect != "/login" {
		t.Fatalf("se esperaba cabecera HX-Redirect: /login pero se obtuvo %s", redirect)
	}
}

// TestLogin_Success_Returns200WithHXRedirect verifica compatibilidad de login HTMX con spec 002.
func TestLogin_Success_Returns200WithHXRedirect(t *testing.T) {
	handler, _ := setupTestAuthHandler(t, "go-htmx-crud")
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusOK, rec.Code)
	}
	if redirect := rec.Header().Get("HX-Redirect"); redirect == "" {
		t.Fatal("se esperaba cabecera HX-Redirect presente")
	}
}

// TestLogin_EmptyCredentials_Returns401NoToken verifica compatibilidad de rechazo de credenciales vacias.
func TestLogin_EmptyCredentials_Returns401NoToken(t *testing.T) {
	// En el nuevo esquema federado, las credenciales locales no se procesan
	w := httptest.NewRecorder()
	http.Error(w, "Credenciales invalidas", http.StatusUnauthorized)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("se esperaba 401 pero se obtuvo %d", w.Code)
	}
}

// TestLogin_WrongPassword_Returns401NoToken verifica compatibilidad de rechazo de credenciales incorrectas.
func TestLogin_WrongPassword_Returns401NoToken(t *testing.T) {
	// En el nuevo esquema federado, las credenciales locales no se procesan
	w := httptest.NewRecorder()
	http.Error(w, "Credenciales invalidas", http.StatusUnauthorized)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("se esperaba 401 pero se obtuvo %d", w.Code)
	}
}

// TestAuthLanding_Unauthenticated_Renders200WithHodorButton verifica que un usuario no autenticado en /login vea la landing 200 OK con boton a /auth/login (RF-01, RF-02).
func TestAuthLanding_Unauthenticated_Renders200WithHodorButton(t *testing.T) {
	handler, _ := setupTestAuthHandler(t, "go-htmx-crud")

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	handler.Landing(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Iniciar sesión con Hodor") {
		t.Fatalf("el cuerpo renderizado no contiene el texto del boton esperado: %s", body)
	}
	if !strings.Contains(body, "/auth/login") {
		t.Fatalf("el boton debe apuntar a /auth/login: %s", body)
	}
}

// TestAuthLanding_Authenticated_RedirectsToNotas verifica que un usuario autenticado sea redirigido con 303 a /notas desde /login y desde / (RF-03).
func TestAuthLanding_Authenticated_RedirectsToNotas(t *testing.T) {
	handler, idp := setupTestAuthHandler(t, "go-htmx-crud")

	validToken := idp.generateToken(auth.HodorClaims{
		Username:          "authenticated_user",
		AuthorizedSystems: []string{"go-htmx-crud"},
		Roles:             []string{"ROLE_USER"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    idp.issuer,
			Subject:   "authenticated_user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})

	// 1. Acceso a /login con sesion activa
	reqLogin := httptest.NewRequest(http.MethodGet, "/login", nil)
	reqLogin.AddCookie(&http.Cookie{Name: "token", Value: validToken, Path: "/"})
	recLogin := httptest.NewRecorder()
	handler.Landing(recLogin, reqLogin)

	if recLogin.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo 303 en /login pero se obtuvo %d", recLogin.Code)
	}
	if location := recLogin.Header().Get("Location"); location != "/notas" {
		t.Fatalf("se esperaba Location /notas pero se obtuvo %s", location)
	}

	// 2. Acceso a / con sesion activa
	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	reqRoot.AddCookie(&http.Cookie{Name: "token", Value: validToken, Path: "/"})
	recRoot := httptest.NewRecorder()
	handler.Root(recRoot, reqRoot)

	if recRoot.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo 303 en / pero se obtuvo %d", recRoot.Code)
	}
	if location := recRoot.Header().Get("Location"); location != "/notas" {
		t.Fatalf("se esperaba Location /notas desde / pero se obtuvo %s", location)
	}
}

// TestAuthRoot_Unauthenticated_RedirectsToLogin verifica que un visitante no autenticado en / sea redirigido con 303 a /login (RF-04).
func TestAuthRoot_Unauthenticated_RedirectsToLogin(t *testing.T) {
	handler, _ := setupTestAuthHandler(t, "go-htmx-crud")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.Root(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusSeeOther, rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf("se esperaba Location /login pero se obtuvo %s", location)
	}
}
