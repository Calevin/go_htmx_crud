package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestGenerateJWT valida la generacion de JWT, claims y rechazo de firmas o tokens invalidos.
func TestGenerateJWT(t *testing.T) {
	t.Run("ValidClaims", func(t *testing.T) {
		username := "alice"
		secretKey := []byte("clave-secreta-de-prueba-12345")

		tokenString, err := GenerateJWT(username, secretKey)
		if err != nil {
			t.Fatalf("se esperaba que no hubiera error generando el token: %v", err)
		}

		claims := &Claims{}
		parsedToken, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Validar que el metodo de firma sea HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("metodo de firma inesperado")
			}
			return secretKey, nil
		})

		if err != nil {
			t.Fatalf("error parseando token valido: %v", err)
		}

		if !parsedToken.Valid {
			t.Fatal("se esperaba que el token fuera marcado como valido")
		}

		if claims.Username != username {
			t.Fatalf("se esperaba username %q, pero se obtuvo %q", username, claims.Username)
		}

		if claims.ExpiresAt == nil {
			t.Fatal("se esperaba que el claim ExpiresAt estuviera definido")
		}

		// Verificar que la expiracion este aproximadamente a 24 horas a futuro (margen de 2 minutos)
		expectedExpiration := time.Now().Add(24 * time.Hour)
		diff := claims.ExpiresAt.Sub(expectedExpiration)
		if diff < -2*time.Minute || diff > 2*time.Minute {
			t.Fatalf("tiempo de expiracion fuera de margen razonable: %v (esperado ~%v)", claims.ExpiresAt, expectedExpiration)
		}
	})

	t.Run("InvalidSecretRejection", func(t *testing.T) {
		secretA := []byte("clave-secreta-a")
		secretB := []byte("clave-secreta-b")

		tokenString, err := GenerateJWT("bob", secretA)
		if err != nil {
			t.Fatalf("error generando token: %v", err)
		}

		claims := &Claims{}
		_, err = jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return secretB, nil
		})

		if err == nil {
			t.Fatal("se esperaba error al validar token con una clave ajena, pero fue aceptado")
		}
	})

	t.Run("MalformedToken", func(t *testing.T) {
		secretKey := []byte("clave-secreta-prueba")
		claims := &Claims{}

		_, err := jwt.ParseWithClaims("token-completamente-invalido", claims, func(token *jwt.Token) (interface{}, error) {
			return secretKey, nil
		})

		if err == nil {
			t.Fatal("se esperaba error parseando token malformado, pero fue aceptado")
		}
	})
}
