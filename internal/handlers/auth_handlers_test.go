package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// hasTokenCookie inspecciona las cookies de respuesta y retorna verdadero si 'token' tiene valor no vacio.
func hasTokenCookie(cookies []*http.Cookie) bool {
	for _, cookie := range cookies {
		if cookie.Name == "token" && cookie.Value != "" {
			return true
		}
	}
	return false
}

// TestLogin_Success_Returns200WithHXRedirect verifica login exitoso con 200, HX-Redirect y cookie token (RF-02).
func TestLogin_Success_Returns200WithHXRedirect(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	username := "valid_login_user"
	password := "correct_password_123"
	_ = createTestUser(t, ctx, queries, username, password)

	formData := url.Values{
		"username": {username},
		"password": {password},
	}

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d (cuerpo: %s)",
			http.StatusOK, recorder.Code, recorder.Body.String())
	}

	if redirect := recorder.Header().Get("HX-Redirect"); redirect != "/notas" {
		t.Fatalf("se esperaba cabecera HX-Redirect: /notas pero se obtuvo %s", redirect)
	}

	cookies := recorder.Result().Cookies()
	var tokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "token" {
			tokenCookie = c
			break
		}
	}

	if tokenCookie == nil || tokenCookie.Value == "" {
		t.Fatal("se esperaba cookie 'token' presente y no vacia")
	}
	if !tokenCookie.HttpOnly {
		t.Fatal("la cookie 'token' debe ser HttpOnly")
	}
}

// TestLogout_Success_Returns200WithHXRedirect verifica logout exitoso con 200, HX-Redirect y limpieza de cookie (RF-02).
func TestLogout_Success_Returns200WithHXRedirect(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	user := createTestUser(t, ctx, queries, "logout_user", "password123")

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	withValidAuthCookie(t, req, user.Username, defaultTestSecret)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusOK, recorder.Code)
	}

	if redirect := recorder.Header().Get("HX-Redirect"); redirect != "/login" {
		t.Fatalf("se esperaba cabecera HX-Redirect: /login pero se obtuvo %s", redirect)
	}

	cookies := recorder.Result().Cookies()
	var tokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "token" {
			tokenCookie = c
			break
		}
	}

	if tokenCookie == nil {
		t.Fatal("se esperaba cookie 'token' de expiracion en la respuesta")
	}
	if tokenCookie.Value != "" {
		t.Fatalf("se esperaba cookie 'token' vacia tras logout pero contenia: %s", tokenCookie.Value)
	}
}

// TestLogin_EmptyCredentials_Returns401NoToken verifica 401 sin cookie con credenciales vacias (RF-07, CL-04).
func TestLogin_EmptyCredentials_Returns401NoToken(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	user := createTestUser(t, ctx, queries, "registered_user", "real_password")

	testCases := []struct {
		name     string
		username string
		password string
	}{
		{
			name:     "Usuario vacio",
			username: "",
			password: "any_password",
		},
		{
			name:     "Contrasena vacia",
			username: user.Username,
			password: "",
		},
		{
			name:     "Ambos campos vacios",
			username: "",
			password: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formData := url.Values{
				"username": {tc.username},
				"password": {tc.password},
			}

			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusUnauthorized, recorder.Code)
			}

			if hasTokenCookie(recorder.Result().Cookies()) {
				t.Fatal("no debe emitirse cookie 'token' ante credenciales vacias")
			}
		})
	}
}

// TestLogin_WrongPassword_Returns401NoToken verifica 401 sin token ante credenciales incorrectas (RF-07).
func TestLogin_WrongPassword_Returns401NoToken(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	user := createTestUser(t, ctx, queries, "existing_user", "password_secreta")

	testCases := []struct {
		name     string
		username string
		password string
	}{
		{
			name:     "Contrasena equivocada para usuario existente",
			username: user.Username,
			password: "password_incorrecta",
		},
		{
			name:     "Usuario inexistente",
			username: "nonexistent_user",
			password: "any_password",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formData := url.Values{
				"username": {tc.username},
				"password": {tc.password},
			}

			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("se esperaba codigo %d pero se obtuvo %d", http.StatusUnauthorized, recorder.Code)
			}

			if hasTokenCookie(recorder.Result().Cookies()) {
				t.Fatal("no debe emitirse cookie 'token' ante credenciales invalidas")
			}
		})
	}
}
