package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// TokenValidator gestiona la validacion de tokens RS256 contra las claves JWKS de Hodor.
type TokenValidator struct {
	jwks     keyfunc.Keyfunc
	issuer   string
	clientID string
}

// NewTokenValidator inicializa el cliente JWKS descargando las claves publicas al arrancar.
// Si el endpoint no responde, falla inmediatamente (principio fail-fast).
func NewTokenValidator(ctx context.Context, jwksURL, issuer, clientID string) (*TokenValidator, error) {
	// Comprobar conectividad inicial con el endpoint JWKS (fail-fast)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize JWKS client: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize JWKS client: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to initialize JWKS client: HTTP status %d", resp.StatusCode)
	}

	// keyfunc gestiona la descarga y refresco automatico de las claves publicas RSA.
	kf, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize JWKS client: %w", err)
	}

	return &TokenValidator{
		jwks:     kf,
		issuer:   issuer,
		clientID: clientID,
	}, nil
}

// ValidateToken valida firma asimetrica RS256, expiracion, emisor y pertenencia a authorized_systems.
func (v *TokenValidator) ValidateToken(tokenString string) (*HodorClaims, error) {
	claims := &HodorClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, v.jwks.Keyfunc,
		jwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token is not valid")
	}

	// Comprobacion explicita de algoritmo de firma (prohibicion estricta de simetricos/nulos)
	if token.Method.Alg() != "RS256" {
		return nil, ErrInvalidAlgorithm
	}

	// Validar emisor si esta configurado
	if v.issuer != "" && claims.Issuer != v.issuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", v.issuer, claims.Issuer)
	}

	// Validar expiracion explicitamente
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token has expired")
	}

	// Validar que el token haya sido autorizado para este cliente/sistema
	if !slices.Contains(claims.AuthorizedSystems, v.clientID) {
		return nil, ErrUnauthorizedSystem
	}

	return claims, nil
}
