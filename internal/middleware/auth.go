package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/Calevin/go_htmx_crud/internal/auth"
)

// contextKey para pasar datos de forma segura entre middlewares y handlers.
type contextKey string

const UserContextKey = contextKey("user")

// Authenticator es un middleware de Chi que verifica el access_token mediante TokenValidator.
func Authenticator(validator *auth.TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Obtener la cookie de sesion 'token'
			cookie, err := r.Cookie("token")
			if err != nil || cookie.Value == "" {
				handleUnauthorized(w, r)
				return
			}

			// 2. Validar criptograficamente el token con RS256 contra JWKS
			claims, err := validator.ValidateToken(cookie.Value)
			if err != nil {
				// Si el usuario es valido en Hodor pero no pertenece a authorized_systems -> 403 Forbidden
				if errors.Is(err, auth.ErrUnauthorizedSystem) {
					http.Error(w, "Forbidden: User not authorized for this system", http.StatusForbidden)
					return
				}
				// Token caducado, malformado o con firma invalida -> no autenticado
				handleUnauthorized(w, r)
				return
			}

			// 3. Inyectar claims en el contexto para handlers posteriores
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// handleUnauthorized gestiona la respuesta segun si la peticion proviene de HTMX o es ordinaria.
func handleUnauthorized(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/auth/login")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}
