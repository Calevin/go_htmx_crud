package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestTokenValidator_ValidRS256AgainstJWKS verifica validacion exitosa de firma RS256 contra endpoint JWKS (RF-06).
func TestTokenValidator_ValidRS256AgainstJWKS(t *testing.T) {
	mockServer := newMockJWKSServer(t, "key-1")
	ctx := context.Background()

	clientID := "go-htmx-crud"
	validator, err := NewTokenValidator(ctx, mockServer.JWKSURL(), mockServer.issuer, clientID)
	if err != nil {
		t.Fatalf("se esperaba inicializacion exitosa pero fallo: %v", err)
	}

	tokenStr, err := mockServer.generateValidToken("alice", []string{clientID, "other-client"})
	if err != nil {
		t.Fatalf("error al generar token valido: %v", err)
	}

	claims, err := validator.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("se esperaba token valido pero retorno error: %v", err)
	}

	if claims.Username != "alice" {
		t.Fatalf("se esperaba username 'alice' pero se obtuvo '%s'", claims.Username)
	}
	if len(claims.AuthorizedSystems) != 2 {
		t.Fatalf("se esperaban 2 authorized_systems pero se obtuvieron %d", len(claims.AuthorizedSystems))
	}
}

// TestTokenValidator_RejectsNonRS256OrInvalidSignatureOrExpired verifica rechazo de firmas invalidas, expiradas o emisor erroneo (RF-07).
func TestTokenValidator_RejectsNonRS256OrInvalidSignatureOrExpired(t *testing.T) {
	mockServer := newMockJWKSServer(t, "key-1")
	ctx := context.Background()
	clientID := "go-htmx-crud"

	validator, err := NewTokenValidator(ctx, mockServer.JWKSURL(), mockServer.issuer, clientID)
	if err != nil {
		t.Fatalf("error al inicializar validador: %v", err)
	}

	// 1. Token con firma de otra clave RSA no publicada
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	tokenBadSig := jwt.NewWithClaims(jwt.SigningMethodRS256, HodorClaims{
		Username:          "bob",
		AuthorizedSystems: []string{clientID},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    mockServer.issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	tokenBadSig.Header["kid"] = "key-1"
	badSigStr, _ := tokenBadSig.SignedString(otherKey)

	if _, err := validator.ValidateToken(badSigStr); err == nil {
		t.Fatal("se esperaba error ante firma RSA no coincidente")
	}

	// 2. Token expirado
	expiredClaims := HodorClaims{
		Username:          "charlie",
		AuthorizedSystems: []string{clientID},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    mockServer.issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	expiredStr, _ := mockServer.generateSignedToken(expiredClaims)
	if _, err := validator.ValidateToken(expiredStr); err == nil {
		t.Fatal("se esperaba error ante token expirado")
	}

	// 3. Emisor no coincidente
	wrongIssuerClaims := HodorClaims{
		Username:          "david",
		AuthorizedSystems: []string{clientID},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "http://fake-idp.example.com",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	wrongIssuerStr, _ := mockServer.generateSignedToken(wrongIssuerClaims)
	if _, err := validator.ValidateToken(wrongIssuerStr); err == nil {
		t.Fatal("se esperaba error ante emisor invalido")
	}
}

// TestTokenValidator_EnforcesRS256Algorithm verifica el rechazo terminante de algoritmos simetricos o none (RNF-01).
func TestTokenValidator_EnforcesRS256Algorithm(t *testing.T) {
	mockServer := newMockJWKSServer(t, "key-1")
	ctx := context.Background()
	clientID := "go-htmx-crud"

	validator, err := NewTokenValidator(ctx, mockServer.JWKSURL(), mockServer.issuer, clientID)
	if err != nil {
		t.Fatalf("error al inicializar validador: %v", err)
	}

	// Token firmado con HMAC HS256
	symToken := jwt.NewWithClaims(jwt.SigningMethodHS256, HodorClaims{
		Username:          "eve",
		AuthorizedSystems: []string{clientID},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    mockServer.issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	symToken.Header["kid"] = "key-1"
	symTokenStr, _ := symToken.SignedString([]byte("shared-secret-key-1234567890123456"))

	if _, err := validator.ValidateToken(symTokenStr); err == nil {
		t.Fatal("se esperaba rechazo inmediato de token firmado con algoritmo simetrico HS256")
	}
}

// TestTokenValidator_UsesCachedJWKS verifica que las claves se cachean en memoria sin llamadas bloqueantes (RF-11).
func TestTokenValidator_UsesCachedJWKS(t *testing.T) {
	mockServer := newMockJWKSServer(t, "key-1")
	ctx := context.Background()
	clientID := "go-htmx-crud"

	validator, err := NewTokenValidator(ctx, mockServer.JWKSURL(), mockServer.issuer, clientID)
	if err != nil {
		t.Fatalf("error al inicializar validador: %v", err)
	}

	tokenStr, err := mockServer.generateValidToken("alice", []string{clientID})
	if err != nil {
		t.Fatalf("error al generar token: %v", err)
	}

	initialRequests := atomic.LoadInt64(&mockServer.reqCount)

	// Ejecutar multiples validaciones en memoria
	for i := 0; i < 5; i++ {
		if _, err := validator.ValidateToken(tokenStr); err != nil {
			t.Fatalf("validacion %d fallo: %v", i, err)
		}
	}

	finalRequests := atomic.LoadInt64(&mockServer.reqCount)
	if finalRequests != initialRequests {
		t.Fatalf("se esperaba que las validaciones usaran la cache en memoria (inicial: %d, final: %d)",
			initialRequests, finalRequests)
	}
}

// TestNewTokenValidator_UnreachableEndpoint_ReturnsError verifica falla rapida al arranque (RF-12).
func TestNewTokenValidator_UnreachableEndpoint_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// URL inexistente o no disponible
	unreachableURL := "http://127.0.0.1:54321/.well-known/jwks.json"
	_, err := NewTokenValidator(ctx, unreachableURL, "http://127.0.0.1:54321", "go-htmx-crud")
	if err == nil {
		t.Fatal("se esperaba error fatal al conectar a endpoint JWKS inaccesible")
	}

	if !strings.Contains(err.Error(), "failed to initialize JWKS client") {
		t.Fatalf("mensaje de error inesperado: %v", err)
	}
}
