# Verificador Programático — landing_login_logout (Spec 004)

> **Contrato Inmutable (`seb-harness`):** Este archivo es ejecutado automáticamente por `.seb/bin/seb verify landing_login_logout` y por el hook `pre-commit` de Git.
> **Prohibición Anti-Reward-Hacking:** El agente implementador tiene terminantemente prohibido editar este archivo o relajar los tests invocados.
> **Restricción de Parser:** Ningún comando contiene el carácter `|` (rompería el parseo de filas de Markdown); las cadenas de verificación se encadenan exclusivamente con `&&`.

## 1. Matriz de Controles Ejecutables (Parseada por `seb verify`)

| Check ID | Requisito | Descripción de la Verificación | Comando Determinista |
| :------- | :-------- | :----------------------------- | :------------------- |
| CHK-01 | RF-01, RF-02 | `GET /login` sin autenticación renderiza pantalla de bienvenida con código `200 OK` y botón hacia `/auth/login` | `grep -q 'func TestAuthLanding_Unauthenticated_Renders200WithHodorButton(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthLanding_Unauthenticated_Renders200WithHodorButton$' ./internal/handlers/` |
| CHK-02 | RF-03 | Usuario autenticado accediendo a `/login` o `/` es redirigido con `303 See Other` a `/notas` | `grep -q 'func TestAuthLanding_Authenticated_RedirectsToNotas(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthLanding_Authenticated_RedirectsToNotas$' ./internal/handlers/` |
| CHK-03 | RF-04 | Usuario anónimo accediendo a la raíz `/` es redirigido con `303 See Other` a `/login` | `grep -q 'func TestAuthRoot_Unauthenticated_RedirectsToLogin(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthRoot_Unauthenticated_RedirectsToLogin$' ./internal/handlers/` |
| CHK-04 | RF-05 | `GET /auth/login` inicia el flujo PKCE y redirige al Proveedor de Identidad Hodor | `grep -q 'func TestAuthLogin_InitiatesPKCEFlow(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthLogin_InitiatesPKCEFlow$' ./internal/handlers/` |
| CHK-05 | RF-06 | Petición HTMX a `/logout` o `/auth/logout` invalida cookie y responde `200 OK` con cabecera `HX-Redirect: /login` | `grep -q 'func TestAuthLogout_HTMX_Returns200WithHXRedirect(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthLogout_HTMX_Returns200WithHXRedirect$' ./internal/handlers/` |
| CHK-06 | RF-07 | Petición ordinaria a `/logout` o `/auth/logout` invalida cookie y redirige `303 See Other` hacia `/login` | `grep -q 'func TestAuthLogout_Standard_Returns303ToLogin(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthLogout_Standard_Returns303ToLogin$' ./internal/handlers/` |
| CHK-07 | RF-08 | Peticiones con método distinto de POST (GET) a `/logout` o `/auth/logout` se rechazan con `405 Method Not Allowed` | `grep -q 'func TestAuthLogout_NonPostMethod_Returns405(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthLogout_NonPostMethod_Returns405$' ./internal/handlers/` |
| CHK-08 | RF-09 | Petición ordinaria a ruta protegida sin sesión redirige `303 See Other` hacia `/login` | `grep -q 'func TestAuthMiddleware_StandardRequest_RedirectsToLogin(' internal/middleware/auth_test.go && go test -count=1 -run '^TestAuthMiddleware_StandardRequest_RedirectsToLogin$' ./internal/middleware/` |
| CHK-09 | RF-10 | Petición HTMX a ruta protegida sin sesión emite `401 Unauthorized` y cabecera `HX-Redirect: /login` | `grep -q 'func TestAuthMiddleware_HTMXRequest_Returns401WithHXRedirectToLogin(' internal/middleware/auth_test.go && go test -count=1 -run '^TestAuthMiddleware_HTMXRequest_Returns401WithHXRedirectToLogin$' ./internal/middleware/` |
| CHK-10 | RNF-01, RNF-02 | Logout elimina cookie `token` con `Max-Age: -1` y atributos de seguridad (`HttpOnly`, `Path=/`, `SameSite=Lax`) | `grep -q 'func TestAuthLogout_ClearsCookieProperly(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestAuthLogout_ClearsCookieProperly$' ./internal/handlers/` |
| CHK-11 | RNF-04 | Suite completa determinista de autenticación y handlers sin dependencias de red externas | `go test -count=1 ./internal/auth ./internal/middleware ./internal/handlers ./database` |
| CHK-12 | GLOBAL | Compilación limpia, tests globales verdes y linter oficial sin hallazgos | `go build ./... && go test -count=1 ./... && golangci-lint run` |

## 2. Criterio de Aprobación

- La tarea solo se considera implementada cuando `.seb/bin/seb verify landing_login_logout` reporta **100% de los controles (`CHK-01` a `CHK-12`) en estado `[PASA]`**.
