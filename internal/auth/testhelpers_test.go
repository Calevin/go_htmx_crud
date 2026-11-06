package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// mockJWKSServer encapsula el servidor HTTP y la clave RSA para pruebas.
type mockJWKSServer struct {
	server      *httptest.Server
	privateKey  *rsa.PrivateKey
	keyID       string
	reqCount    int64
	issuer      string
}

func newMockJWKSServer(t *testing.T, keyID string) *mockJWKSServer {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("error al generar clave RSA para test: %v", err)
	}

	mock := &mockJWKSServer{
		privateKey: privateKey,
		keyID:      keyID,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&mock.reqCount, 1)

		pub := &privateKey.PublicKey
		nBytes := pub.N.Bytes()
		eBytes := big.NewInt(int64(pub.E)).Bytes()

		jwk := map[string]any{
			"kty": "RSA",
			"kid": keyID,
			"use": "sig",
			"alg": "RS256",
			"n":   base64.RawURLEncoding.EncodeToString(nBytes),
			"e":   base64.RawURLEncoding.EncodeToString(eBytes),
		}

		resp := map[string]any{
			"keys": []any{jwk},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	mock.server = server
	mock.issuer = server.URL

	t.Cleanup(func() {
		server.Close()
	})

	return mock
}

func (m *mockJWKSServer) JWKSURL() string {
	return m.server.URL + "/.well-known/jwks.json"
}

func (m *mockJWKSServer) generateSignedToken(claims HodorClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = m.keyID
	return token.SignedString(m.privateKey)
}

func (m *mockJWKSServer) generateValidToken(username string, authorizedSystems []string) (string, error) {
	claims := HodorClaims{
		Username:          username,
		AuthorizedSystems: authorizedSystems,
		Roles:             []string{"ROLE_USER"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return m.generateSignedToken(claims)
}
