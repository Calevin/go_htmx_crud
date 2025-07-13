package middleware

import (
	"context"
	"net/http"

	"github.com/Calevin/go_htmx_crud/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

// contextKey para pasar datos de forma segura entre middlewares y handlers.
type contextKey string

const UserContextKey = contextKey("user")

// Authenticator es un middleware de Chi que verifica el token JWT.
func Authenticator(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Se obtiene el token de la cookie
			cookie, err := r.Cookie("token")
			if err != nil {
				// Si no hay cookie, el usuario no está autenticado
				http.Redirect(w, r, "/login-page", http.StatusSeeOther) // Redirige a la página de login
				return
			}
			tokenString := cookie.Value

			// 2. Se parsea y valida el token
			claims := &auth.Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				// Token inválido o expirado
				http.Redirect(w, r, "/login-page", http.StatusSeeOther)
				return
			}

			// 3. Se guarda la información del usuario en el contexto de la petición
			// para que los siguientes handlers puedan acceder a ella.
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
