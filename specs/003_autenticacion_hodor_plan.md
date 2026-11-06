# Plan Técnico — autenticacion_hodor (Spec 003)

- **Spec Origen:** `specs/003_autenticacion_hodor_spec.md`
- **Verificador Asociado:** `specs/003_autenticacion_hodor_verificador.md`
- **Estado:** `PLANIFICADO`

## 1. Resumen Arquitectónico y Aislamiento de Lógica

La spec 003 transforma la arquitectura de autenticación del proyecto, migrando de un esquema local monolítico (basado en credenciales almacenadas en SQLite, hash con bcrypt y tokens simétricos HS256 generados localmente) hacia un modelo federado estándar OpenID Connect / OAuth 2.0 (Authorization Code Flow con PKCE), donde **Hodor** actúa como Proveedor de Identidad (IdP) centralizado emitiendo tokens firmados asimétricamente con RS256.

La aplicación `go_htmx_crud` opera exclusivamente como **Relying Party / Resource Server**:

1. **Delegación de Credenciales:** Se descontinúa la custodia y verificación de contraseñas locales. La base de datos local SQLite conserva su esquema de datos de dominio (notas y etiquetas), pero los handlers dejan de consultar o verificar contraseñas.
2. **Validación Criptográfica Asimétrica:** La aplicación consume el endpoint público de claves (`/.well-known/jwks.json`) expuesto por Hodor mediante un cliente en memoria que cachea y refresca automáticamente las claves públicas RSA. Se prohíbe terminantemente la firma local o el uso de algoritmos simétricos (HS256).
3. **Flujo de Autorización OIDC con PKCE:**
   - `/auth/login` (y `/login`): genera desafío PKCE (`code_challenge` SHA-256) y parámetro de estado (`state`), almacenados en cookies temporales seguras (`HttpOnly`, `SameSite=Lax`), y redirige al IdP (`IDP_HODOR_URL`).
   - `/auth/callback`: valida `state`, intercambia código y `code_verifier` por el `access_token` en Hodor, valida la firma RS256 y claims mandatorios, almacena el `access_token` en la cookie de sesión `token` y redirige a `/notas`.
   - `/auth/logout` (y `/logout`): invalida la cookie local de sesión y redirige a `/login`.
4. **Integración con HTMX:** El middleware de autenticación detecta la cabecera `HX-Request: true`. Ante falta de sesión o token inválido, emite la cabecera `HX-Redirect: /auth/login` con código `401 Unauthorized`, forzando la navegación completa del navegador y evitando bloqueos CORS o inserciones del formulario de login dentro del DOM. Ante peticiones ordinarias sin HTMX, responde `303 See Other`.
5. **Control de Acceso por Sistema:** Si el token es criptográficamente válido pero el cliente configurado (`OAUTH_CLIENT_ID`) no figura en el claim `authorized_systems`, se responde `403 Forbidden`.
6. **Principio Fail-Fast y Testabilidad:** El validador aborta al arranque (`log.Fatal`) si el endpoint JWKS inicial no responde. Para garantizar pruebas 100% deterministas y autónomas (Constitución B y F), la suite automatizada utiliza un `httptest.Server` en memoria que expone un JWKS simulado con claves RSA efímeras.

## 2. Mapa de Archivos a Crear o Modificar

| Archivo / Ruta                                 | Acción      | Responsabilidad Técnica                                                                                                                                                                    | Requisitos Cubiertos                                            |
| :--------------------------------------------- | :---------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :-------------------------------------------------------------- |
| `go.mod` / `go.sum`                            | `MODIFICAR` | Agregar dependencias oficiales `golang.org/x/oauth2` y `github.com/MicahParks/keyfunc/v3`                                                                                                  | `RNF-05`                                                        |
| `internal/auth/claims.go`                      | `CREAR`     | Definir struct `HodorClaims` con campos `sub`, `roles`, `authorized_systems` y `jwt.RegisteredClaims`                                                                                      | `RF-04`, `RF-06`, `RF-08`                                       |
| `internal/auth/validator.go`                   | `CREAR`     | Validador de tokens RS256 con cliente JWKS en memoria (`keyfunc`), validación de emisor, caducidad y pertenencia al sistema autorizado                                                     | `RF-06`, `RF-07`, `RF-08`, `RF-11`, `RF-12`, `RNF-01`, `RNF-02` |
| `internal/auth/testhelpers_test.go`            | `CREAR`     | Utilidades de pruebas para generar pares de claves RSA efímeras y servidor mock de JWKS en memoria                                                                                         | `RNF-04`                                                        |
| `internal/auth/validator_test.go`              | `CREAR`     | Pruebas unitarias del validador de tokens (validación RS256, expiración, rechazo de HS256, rechazo sin sistema autorizado, caché y fail-fast)                                              | `RF-06`, `RF-07`, `RF-08`, `RF-11`, `RF-12`, `RNF-01`, `RNF-02` |
| `internal/middleware/auth.go`                  | `MODIFICAR` | Adaptar middleware `Authenticator` para recibir `*auth.TokenValidator` y `clientID`: soporte de `HX-Redirect` (`401`), `303` para peticiones estándar y `403` para usuarios desautorizados | `RF-02`, `RF-03`, `RF-08`                                       |
| `internal/middleware/auth_test.go`             | `MODIFICAR` | Pruebas de middleware cubriendo HTMX (`401`+`HX-Redirect`), estándar (`303`), `403 Forbidden` ante desautorización y paso autorizado                                                       | `RF-02`, `RF-03`, `RF-08`, `RNF-04`                             |
| `internal/handlers/auth_handlers.go`           | `MODIFICAR` | Implementar `LoginHandler`, `CallbackHandler` y `LogoutHandler` OIDC/PKCE; retirar verificación de contraseñas locales y bcrypt                                                            | `RF-01`, `RF-04`, `RF-05`, `RF-09`, `RF-10`, `RF-13`            |
| `internal/handlers/auth_handlers_test.go`      | `MODIFICAR` | Pruebas de los handlers OIDC contra servidor mock IdP (`Login`, `Callback` con estado válido e inválido, `Logout`)                                                                         | `RF-01`, `RF-04`, `RF-05`, `RF-09`, `RF-10`, `RF-13`            |
| `internal/handlers/testhelpers_test.go`        | `MODIFICAR` | Adaptar router espejo de test y generadores de token para usar firmas RS256 del IdP mock                                                                                                   | `RNF-04`                                                        |
| `main.go`                                      | `MODIFICAR` | Leer variables `IDP_HODOR_URL`, `OAUTH_CLIENT_ID`, `OAUTH_CLIENT_SECRET`; inicializar `TokenValidator` (fail-fast); montar rutas OIDC y retirar inicialización de usuario/contraseñas      | `RF-01`, `RF-12`, `RF-13`, `RNF-03`                             |
| `specs/003_autenticacion_hodor_plan.md`        | `CREAR`     | Plan técnico y desglose TDD de la tarea                                                                                                                                                    | Gobernanza SDD                                                  |
| `specs/003_autenticacion_hodor_verificador.md` | `CREAR`     | Matriz de controles deterministas parseable por `.seb/bin/seb verify`                                                                                                                      | Gobernanza SDD                                                  |

_Archivos que NO se modifican:_

- `internal/db/**` (código generado por sqlc; se mantiene intacto).
- `sql/schema/**` y `sql/queries/**` (no se alteran las consultas ni el esquema de base de datos en esta tarea).

## 3. Contratos, Firmas y Modelo de Datos

### 3.1. Estructura de Claims y Contrato del Validador (`internal/auth/`)

```go
package auth

import (
	"context"
	"errors"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrUnauthorizedSystem = errors.New("user not authorized for this system")
	ErrInvalidAlgorithm   = errors.New("invalid signing algorithm: only RS256 allowed")
)

type HodorClaims struct {
	Username          string   `json:"sub"`
	AuthorizedSystems []string `json:"authorized_systems"`
	Roles             []string `json:"roles"`
	jwt.RegisteredClaims
}

type TokenValidator struct {
	jwks     keyfunc.Keyfunc
	issuer   string
	clientID string
}

func NewTokenValidator(ctx context.Context, jwksURL, issuer, clientID string) (*TokenValidator, error)
func (v *TokenValidator) ValidateToken(tokenString string) (*HodorClaims, error)
```

### 3.2. Contrato de Handlers OIDC (`internal/handlers/`)

```go
package handlers

import (
	"net/http"
	"github.com/Calevin/go_htmx_crud/internal/auth"
	"golang.org/x/oauth2"
)

type OAuthConfig struct {
	Config   *oauth2.Config
	IDPURL   string
	ClientID string
}

func NewOIDCAuthHandler(cfg OAuthConfig, validator *auth.TokenValidator) *OIDCAuthHandler

// Handlers expuestos:
// - Login(w http.ResponseWriter, r *http.Request)
// - Callback(w http.ResponseWriter, r *http.Request)
// - Logout(w http.ResponseWriter, r *http.Request)
```

### 3.3. Contrato del Middleware de Autenticación (`internal/middleware/`)

```go
package middleware

import (
	"net/http"
	"github.com/Calevin/go_htmx_crud/internal/auth"
)

func Authenticator(validator *auth.TokenValidator) func(http.Handler) http.Handler
```

## 4. Decisiones Técnicas Justificadas

- **Decisión 1: Incorporación justificada de `golang.org/x/oauth2` y `keyfunc/v3`.**
  - _Motivo:_ La Constitución Sección B prohíbe dependencias sin aprobación en spec/plan. La spec 003 (RNF-05) autoriza formalmente las dependencias de OAuth2 y JWKS. `x/oauth2` es el paquete canónico y mantenido por el equipo oficial de Go para flujos Authorization Code con PKCE. `keyfunc/v3` es el estándar recomendado para `golang-jwt/jwt/v5` para manejo de JWKS con caché en memoria y soporte de rotación de claves (`kid`).
  - _Alternativa descartada:_ Implementar el cliente JWKS y el intercambio PKCE manualmente con `net/http` básico; incrementa innecesariamente el riesgo de errores criptográficos en el manejo de claves RSA y rotación.
- **Decisión 2: Código HTTP `403 Forbidden` ante falta de asignación en `authorized_systems`.**
  - _Motivo:_ Confirmado por el usuario en la fase de clarificación. Un usuario autenticado en Hodor pero sin el cliente `go-htmx-crud` en sus claims está plenamente identificado pero desprovisto de permisos; responder `401` provocaría una redirección en bucle al login de Hodor donde el usuario volvería a presentarse válidamente.
- **Decisión 3: Cierre de sesión local sin Single Sign-Out federado.**
  - _Motivo:_ Confirmado por el usuario. La revocación de la cookie local de sesión (`token`) desacopla el ciclo de vida del cliente del IdP, evitando flujos complejos de Front-Channel o Back-Channel Logout de OIDC en este MVP.
- **Decisión 4: Principio Fail-Fast en arranque (`log.Fatal`).**
  - _Motivo:_ Confirmado por el usuario. Si la aplicación no puede contactar al JWKS de Hodor al arrancar, ningún request podrá ser validado criptográficamente. Abortar el inicio previene desplegar un servidor en estado degradado o inconsistente.
- **Decisión 5: Mock de IdP en memoria con `httptest.Server` para pruebas deterministas.**
  - _Motivo:_ Cumplimiento estricto de la Constitución Sección B y F. La suite `go test ./...` debe finalizar limpia sin requerir un contenedor o servicio Hodor real en ejecución en el host. En los tests se generarán pares RSA de 2048 bits efímeros y un endpoint `/.well-known/jwks.json` en memoria.
- **Decisión 6: Sintaxis de comandos en `verificador.md` sin caracteres `|`.**
  - _Motivo:_ El parser de `.seb/bin/seb verify` segmenta filas de Markdown por pipes (`|`). Se encadenan los comandos con `&&` y se utilizan aserciones deterministas (`grep -q`, `go test -run`, y condiciones sh).

## 5. Secuencia de Tareas Atómicas (TDD: Test en Rojo -> Código -> Test en Verde)

- [ ] **T1. Incorporar dependencias externas y definir estructura de claims (`HodorClaims`)** (`RNF-05`, `RF-04`)
  - _Acción:_ Ejecutar `go get golang.org/x/oauth2 github.com/MicahParks/keyfunc/v3@v3.3.10` y crear `internal/auth/claims.go`.
  - _Hecho cuando:_ El proyecto compila con `go build ./...` y `go.mod` refleja las nuevas dependencias.
- [ ] **T2. TDD Validador de Tokens RS256 con cliente JWKS (`internal/auth/validator.go`)** (`RF-06`, `RF-07`, `RF-08`, `RF-11`, `RF-12`, `RNF-01`, `RNF-02`)
  - _Rojo:_ Escribir `internal/auth/validator_test.go` con utilidades de prueba en `internal/auth/testhelpers_test.go` (claves RSA y JWKS efímero).
  - _Verde:_ Implementar `TokenValidator` en `internal/auth/validator.go` resolviendo firmas RS256, expiración, emisor, validación de `authorized_systems`, caché en memoria y fail-fast.
  - _Hecho cuando:_ `CHK-06`, `CHK-07`, `CHK-11`, `CHK-12` y `CHK-14` devuelven exit code `0`.
- [ ] **T3. TDD Middleware de Autenticación con soporte HTMX y control de acceso (`internal/middleware/auth.go`)** (`RF-02`, `RF-03`, `RF-08`)
  - _Rojo:_ Actualizar `internal/middleware/auth_test.go` para verificar `401`+`HX-Redirect` en peticiones HTMX, `303` en peticiones ordinarias y `403 Forbidden` ante usuario desautorizado.
  - _Verde:_ Adaptar `internal/middleware/auth.go` para usar `*auth.TokenValidator`.
  - _Hecho cuando:_ `CHK-02`, `CHK-03` y `CHK-08` devuelven exit code `0`.
- [ ] **T4. TDD Handlers de Autenticación OIDC con PKCE (`internal/handlers/auth_handlers.go`)** (`RF-01`, `RF-04`, `RF-05`, `RF-09`, `RF-10`)
  - _Rojo:_ Actualizar `internal/handlers/auth_handlers_test.go` con casos de prueba para `Login` (generación de PKCE y cookies `oauth_*`), `Callback` (éxito, estado inválido `400`, intercambio de token) y `Logout` (invalidación de cookie `token`).
  - _Verde:_ Reescribir `internal/handlers/auth_handlers.go` implementando los nuevos handlers OIDC.
  - _Hecho cuando:_ `CHK-01`, `CHK-04`, `CHK-05`, `CHK-09` y `CHK-10` devuelven exit code `0`.
- [ ] **T5. Retiro de verificación de contraseñas locales y bcrypt en handlers (`RF-13`)**
  - _Acción:_ Asegurar que `internal/handlers/auth_handlers.go` no invoque `bcrypt` ni consulte contraseñas de la base de datos local.
  - _Hecho cuando:_ `CHK-13` devuelve exit code `0`.
- [ ] **T6. Adaptación de `main.go` para cableado de OIDC y variables de entorno (`RNF-03`, `RF-12`, `RF-13`)**
  - _Acción:_ Cargar `IDP_HODOR_URL`, `OAUTH_CLIENT_ID`, `OAUTH_CLIENT_SECRET`; inicializar `TokenValidator` con verificación fail-fast; configurar rutas Chi públicas (`/login`, `/auth/login`, `/auth/callback`, `/logout`, `/auth/logout`) y grupo protegido. Retirar la función `createTestUser` y la obligatoriedad de `JWT_SECRET`.
  - _Hecho cuando:_ `go build ./...` compila limpiamente sin errores de cableado.
- [ ] **T7. Actualización de helpers de pruebas existentes y validación de rutas de notas (`RNF-04`)**
  - _Acción:_ Adaptar `internal/handlers/testhelpers_test.go` para firmar tokens con la clave RSA efímera y validar que los tests de CRUD en `internal/handlers/view_handlers_test.go` sigan pasando sin regresiones.
  - _Hecho cuando:_ `CHK-15` (`go test -count=1 ./internal/auth ./internal/middleware ./internal/handlers ./database`) devuelve exit code `0`.
- [ ] **T8. Puerta determinista final: compilación limpia, suite verde y linter con cero hallazgos (`RNF-05`)**
  - _Acción:_ Ejecutar la verificación global determinista del repositorio.
  - _Hecho cuando:_ `CHK-16` devuelve exit code `0`.
