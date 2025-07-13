package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
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
