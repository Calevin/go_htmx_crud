# Verificador Programático — autenticacion_hodor (Spec 003)

> **Contrato Inmutable (`seb-harness`):** Este archivo es ejecutado automáticamente por `.seb/bin/seb verify autenticacion_hodor` y por el hook `pre-commit` de Git.
> **Prohibición Anti-Reward-Hacking:** El agente implementador tiene terminantemente prohibido editar este archivo o relajar los tests invocados.
> **Restricción de Parser:** Ningún comando contiene el carácter `|` (rompería el parseo de filas de Markdown); las cadenas de verificación se encadenan exclusivamente con `&&`.

## 1. Matriz de Controles Ejecutables (Parseada por `seb verify`)

| Check ID | Requisito | Descripción de la Verificación | Comando Determinista |
| :------- | :-------- | :----------------------------- | :------------------- |
| CHK-01 | RF-01 | Inicio de flujo OIDC PKCE genera `code_challenge`, `state` en cookies y redirige al IdP | `grep -q 'func TestAuthLogin_InitiatesPKCEFlow(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthLogin_InitiatesPKCEFlow$' ./internal/handlers/` |
| CHK-02 | RF-02 | Petición HTMX sin sesión emite `401 Unauthorized` y cabecera `HX-Redirect: /auth/login` | `grep -q 'func TestAuthMiddleware_HTMXRequest_Returns401WithHXRedirect(' internal/middleware/auth_test.go && go test -count=1 -run '^TestAuthMiddleware_HTMXRequest_Returns401WithHXRedirect$' ./internal/middleware/` |
| CHK-03 | RF-03 | Petición ordinaria sin sesión responde `303 See Other` hacia `/auth/login` | `grep -q 'func TestAuthMiddleware_StandardRequest_Returns303ToLogin(' internal/middleware/auth_test.go && go test -count=1 -run '^TestAuthMiddleware_StandardRequest_Returns303ToLogin$' ./internal/middleware/` |
| CHK-04 | RF-04 | Endpoint `/auth/callback` valida `state`, intercambia código y valida token RS256 | `grep -q 'func TestAuthCallback_ValidCodeAndState_ExchangesAndValidatesRS256(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthCallback_ValidCodeAndState_ExchangesAndValidatesRS256$' ./internal/handlers/` |
| CHK-05 | RF-05 | `/auth/callback` con estado inválido o sin verificador PKCE responde `400 Bad Request` | `grep -q 'func TestAuthCallback_InvalidStateOrMissingVerifier_Returns400(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthCallback_InvalidStateOrMissingVerifier_Returns400$' ./internal/handlers/` |
| CHK-06 | RF-06 | Validador comprueba firma asimétrica RS256 contra endpoint JWKS | `grep -q 'func TestTokenValidator_ValidRS256AgainstJWKS(' internal/auth/validator_test.go && go test -count=1 -run '^TestTokenValidator_ValidRS256AgainstJWKS$' ./internal/auth/` |
| CHK-07 | RF-07 | Rechazo de algoritmos no RS256 (HS256, none), tokens expirados o emisor inválido | `grep -q 'func TestTokenValidator_RejectsNonRS256OrInvalidSignatureOrExpired(' internal/auth/validator_test.go && go test -count=1 -run '^TestTokenValidator_RejectsNonRS256OrInvalidSignatureOrExpired$' ./internal/auth/` |
| CHK-08 | RF-08 | Token sin el cliente en `authorized_systems` responde `403 Forbidden` | `grep -q 'func TestAuthMiddleware_UnauthorizedSystem_Returns403(' internal/middleware/auth_test.go && go test -count=1 -run '^TestAuthMiddleware_UnauthorizedSystem_Returns403$' ./internal/middleware/` |
| CHK-09 | RF-09 | Callback exitoso establece cookie `token` segura (`HttpOnly`, `SameSite=Lax`) y redirige `302` a `/notas` | `grep -q 'func TestAuthCallback_Success_SetsTokenCookieAndRedirectsToNotas(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthCallback_Success_SetsTokenCookieAndRedirectsToNotas$' ./internal/handlers/` |
| CHK-10 | RF-10 | Logout invalida cookie `token` y redirige a `/login` (con `HX-Redirect` si es HTMX) | `grep -q 'func TestAuthLogout_ClearsCookieAndRedirects(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthLogout_ClearsCookieAndRedirects$' ./internal/handlers/` |
| CHK-11 | RF-11 | Validador cachea claves públicas JWKS en memoria sin consultas de red por request | `grep -q 'func TestTokenValidator_UsesCachedJWKS(' internal/auth/validator_test.go && go test -count=1 -run '^TestTokenValidator_UsesCachedJWKS$' ./internal/auth/` |
| CHK-12 | RF-12 | Falla rápida: inicialización con endpoint JWKS inaccesible retorna error fatal | `grep -q 'func TestNewTokenValidator_UnreachableEndpoint_ReturnsError(' internal/auth/validator_test.go && go test -count=1 -run '^TestNewTokenValidator_UnreachableEndpoint_ReturnsError$' ./internal/auth/` |
| CHK-13 | RF-13 | Eliminación definitiva de autenticación por contraseña local y bcrypt en handlers | `if grep -q 'bcrypt.CompareHashAndPassword' internal/handlers/auth_handlers.go; then exit 1; fi && if grep -q 'GetUserByUsername' internal/handlers/auth_handlers.go; then exit 1; fi` |
| CHK-14 | RNF-01 | Verificación estricta de algoritmo RS256 y rechazo de firmas débiles | `grep -q 'func TestTokenValidator_EnforcesRS256Algorithm(' internal/auth/validator_test.go && go test -count=1 -run '^TestTokenValidator_EnforcesRS256Algorithm$' ./internal/auth/` |
| CHK-15 | RNF-04 | Determinismo y aislamiento: suite completa sin dependencias externas de red | `go test -count=1 ./internal/auth ./internal/middleware ./internal/handlers ./database` |
| CHK-16 | RNF-05 | Compilación completa, suite verde y linter oficial con cero hallazgos | `go build ./... && go test -count=1 ./... && golangci-lint run` |

## 2. Criterio de Aprobación

- La tarea solo se considera implementada cuando `.seb/bin/seb verify autenticacion_hodor` reporta **100% de los controles (`CHK-01` a `CHK-16`) en estado `[PASA]`**.
