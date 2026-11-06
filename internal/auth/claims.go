package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrUnauthorizedSystem indica que el usuario no tiene asignado este cliente en su claim authorized_systems.
	ErrUnauthorizedSystem = errors.New("user not authorized for this system")

	// ErrInvalidAlgorithm indica que el token no utiliza la firma asimetrica RS256 exigida.
	ErrInvalidAlgorithm = errors.New("invalid signing algorithm: only RS256 allowed")
)

// HodorClaims estructura los claims emitidos por el Proveedor de Identidad Hodor.
type HodorClaims struct {
	Username          string   `json:"sub"`
	AuthorizedSystems []string `json:"authorized_systems"`
	Roles             []string `json:"roles"`
	jwt.RegisteredClaims
}
