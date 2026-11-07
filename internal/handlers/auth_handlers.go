package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"net/http"
	"time"

	"github.com/Calevin/go_htmx_crud/internal/auth"
	"golang.org/x/oauth2"
)

// AuthHandler gestiona el flujo de autenticacion federada OpenID Connect / OAuth2 con PKCE y la landing.
type AuthHandler struct {
	oauthConfig    *oauth2.Config
	tokenValidator *auth.TokenValidator
	tpl            *template.Template
}

// NewAuthHandler construye una nueva instancia de AuthHandler.
func NewAuthHandler(oauthConfig *oauth2.Config, validator *auth.TokenValidator, tpl *template.Template) *AuthHandler {
	return &AuthHandler{
		oauthConfig:    oauthConfig,
		tokenValidator: validator,
		tpl:            tpl,
	}
}

// Landing atiende GET /login: renderiza la vista de bienvenida o redirige a /notas si ya existe sesion activa.
func (h *AuthHandler) Landing(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("token"); err == nil && cookie.Value != "" {
		if _, err := h.tokenValidator.ValidateToken(cookie.Value); err == nil {
			http.Redirect(w, r, "/notas", http.StatusSeeOther)
			return
		}
		// Si la cookie contiene un token invalido o expirado, se limpia en la respuesta
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		})
	}

	w.WriteHeader(http.StatusOK)
	Render(h.tpl, w, "login.html", nil)
}

// Root atiende GET /: redirige a /notas si el usuario esta autenticado, o a /login si es anonimo.
func (h *AuthHandler) Root(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("token"); err == nil && cookie.Value != "" {
		if _, err := h.tokenValidator.ValidateToken(cookie.Value); err == nil {
			http.Redirect(w, r, "/notas", http.StatusSeeOther)
			return
		}
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Login inicia el flujo Authorization Code con PKCE.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	state, err := generateRandomString(32)
	if err != nil {
		http.Error(w, "Error generating state parameter", http.StatusInternalServerError)
		return
	}

	codeVerifier := oauth2.GenerateVerifier()

	// Guardar state y verifier en cookies temporales HttpOnly de 5 minutos
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/auth/callback",
		Expires:  time.Now().Add(5 * time.Minute),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_verifier",
		Value:    codeVerifier,
		Path:     "/auth/callback",
		Expires:  time.Now().Add(5 * time.Minute),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	authURL := h.oauthConfig.AuthCodeURL(state, oauth2.S256ChallengeOption(codeVerifier))

	// Si la peticion proviene de HTMX, forzar redireccion total via HX-Redirect
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", authURL)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback procesa la respuesta del IdP intercambiando el codigo por el token y validandolo.
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	queryState := r.URL.Query().Get("state")
	if err != nil || queryState == "" || stateCookie.Value != queryState {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}

	verifierCookie, err := r.Cookie("oauth_verifier")
	if err != nil || verifierCookie.Value == "" {
		http.Error(w, "Code verifier not found", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code missing", http.StatusBadRequest)
		return
	}

	// Intercambio del codigo y el verificador PKCE por tokens ante Hodor
	token, err := h.oauthConfig.Exchange(r.Context(), code, oauth2.VerifierOption(verifierCookie.Value))
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	// Validar criptograficamente el access_token recibido contra JWKS y claims
	claims, err := h.tokenValidator.ValidateToken(token.AccessToken)
	if err != nil {
		http.Error(w, "Received invalid token from IdP", http.StatusUnauthorized)
		return
	}

	cookieExpiry := token.Expiry
	if cookieExpiry.IsZero() && claims.ExpiresAt != nil {
		cookieExpiry = claims.ExpiresAt.Time
	}
	if cookieExpiry.IsZero() {
		cookieExpiry = time.Now().Add(24 * time.Hour)
	}

	// Persistir el token de acceso en la cookie de sesion segura
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token.AccessToken,
		Expires:  cookieExpiry,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	// Limpiar cookies temporales del flujo OIDC/PKCE
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/auth/callback",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_verifier",
		Value:    "",
		Path:     "/auth/callback",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/notas", http.StatusFound)
}

// Logout gestiona la revocacion de sesion local eliminando la cookie 'token'.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	LogoutHandler()(w, r)
}

// LogoutHandler retorna el HandlerFunc para cerrar la sesion local con semantica POST estricta.
func LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Rechazar metodos distintos de POST con 405 Method Not Allowed
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Eliminar la cookie de sesion estableciendo MaxAge -1 y expiracion preterita
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		})

		// Si proviene de HTMX, emitir cabecera HX-Redirect con estado 200 OK
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/login")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Peticion ordinaria POST: redirigir 303 See Other hacia /login
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func generateRandomString(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
