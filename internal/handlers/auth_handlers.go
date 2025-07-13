package handlers

import (
	"github.com/Calevin/go_htmx_crud/internal/auth"
	"github.com/Calevin/go_htmx_crud/internal/db"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

// LoginHandler procesa la petición de login.
func LoginHandler(queries *db.Queries, jwtSecret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Se parsean las credenciales del formulario
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Error al parsear el formulario", http.StatusBadRequest)
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")

		// 2. Se obtiene el usuario de la BD
		user, err := queries.GetUserByUsername(r.Context(), username)
		if err != nil {
			// Mensaje genérico no revela información
			http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
			return
		}

		// 3. Se compara el hash de la contraseña con la contraseña enviada
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if err != nil {
			http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
			return
		}

		// 4. Se genera el token JWT
		tokenString, err := auth.GenerateJWT(user.Username, jwtSecret)
		if err != nil {
			http.Error(w, "Error al generar el token", http.StatusInternalServerError)
			return
		}

		// 5. Se establece el token en una cookie HttpOnly
		// HttpOnly previene que el token sea accedido por JavaScript (protección XSS)
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    tokenString,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Path:     "/", // Disponible en todo el sitio
			SameSite: http.SameSiteLaxMode,
		})

		// Se redirige a las notas usando HTMX
		w.Header().Set("HX-Redirect", "/notas")
		w.WriteHeader(http.StatusOK)
	}
}

func LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Se limpia la cookie del token
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			Expires:  time.Unix(0, 0), // Expira inmediatamente
			HttpOnly: true,
			Path:     "/",
		})
		// Se redirige al login usando HTMX
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
	}
}
