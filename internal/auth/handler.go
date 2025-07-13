package auth

import (
	"net/http"
	"time"

	"github.com/Calevin/go_htmx_crud/internal/db"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Se definen los claims para el token.
// Se incluye RegisteredClaims para tener los campos estándar como `ExpiresAt`.
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateJWT crea un nuevo token JWT para un usuario.
func GenerateJWT(username string, secretKey []byte) (string, error) {
	// Tiempo de expiración del token (ej. 24 horas)
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

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
		tokenString, err := GenerateJWT(user.Username, jwtSecret)
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

		// Usando HTMX se redirecciona
		w.Header().Set("HX-Redirect", "/notas")
		w.WriteHeader(http.StatusOK)
	}
}
