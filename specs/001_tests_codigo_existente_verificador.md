# Verificador Programático — tests_codigo_existente (Spec 001)

> **Contrato Inmutable (`seb-harness`):** Este archivo es ejecutado automáticamente por `.seb/bin/seb verify tests_codigo_existente` y por el hook `pre-commit` de Git.
> **Prohibición Anti-Reward-Hacking:** El agente implementador tiene terminantemente prohibido editar este archivo o relajar los tests invocados.
> **Restricción de Parser:** Ningún comando contiene el carácter `|` (rompería el parseo de filas de Markdown); las cadenas de verificación se encadenan exclusivamente con `&&`.

## 1. Matriz de Controles Ejecutables (Parseada por `seb verify`)

| Check ID | Requisito | Descripción de la Verificación | Comando Determinista |
| :------- | :-------- | :----------------------------- | :------------------- |
| CHK-01 | RF-01 | Presencia y pase de tests en los cuatro paquetes lógicos (`internal/auth`, `database`, `internal/middleware`, `internal/handlers`) | `test -f internal/auth/jwt_test.go && test -f database/sqlite_test.go && test -f internal/middleware/auth_test.go && test -f internal/handlers/view_handlers_test.go && go test -count=1 ./internal/auth ./database ./internal/middleware ./internal/handlers` |
| CHK-02 | RF-02 | Ejecución limpia de la suite completa sin servicios externos ni variables de operador | `go test -count=1 ./...` |
| CHK-03 | RF-03 | Aislamiento de persistencia con base de datos temporal por prueba (`t.TempDir()`) | `grep -q 't.TempDir()' database/sqlite_test.go && grep -q 't.TempDir()' internal/handlers/testhelpers_test.go && go test -count=1 -run '^TestInitDB$' ./database/` |
| CHK-04 | RF-04 | Contexto de autenticación autónomo y generación de JWT en memoria | `grep -q 'func TestGenerateJWT(' internal/auth/jwt_test.go && go test -count=1 -run '^TestGenerateJWT' ./internal/auth/` |
| CHK-05 | RF-05 | Middleware responde `303 See Other` a `/login` ante token ausente, inválido, ajeno o expirado | `grep -q 'func TestAuthMiddleware_NoToken_Returns303(' internal/middleware/auth_test.go && go test -count=1 -run '^TestAuthMiddleware_NoToken_Returns303$' ./internal/middleware/ && go test -count=1 -run '^TestAuthMiddleware_InvalidToken_Returns303$' ./internal/middleware/ && go test -count=1 -run '^TestAuthMiddleware_WrongSecret_Returns303$' ./internal/middleware/ && go test -count=1 -run '^TestAuthMiddleware_ExpiredToken_Returns303$' ./internal/middleware/` |
| CHK-06 | RF-06 | Códigos canónicos en operaciones de notas (listar `200`, crear/editar `302`, borrar `200`, inexistente `404`) | `grep -q 'func TestGETRoutes_Return200(' internal/handlers/view_handlers_test.go && go test -count=1 -run '^TestGETRoutes_Return200$' ./internal/handlers/ && go test -count=1 -run '^TestCreateNote_Success_Returns302$' ./internal/handlers/ && go test -count=1 -run '^TestUpdateNote_Success_Returns302$' ./internal/handlers/ && go test -count=1 -run '^TestDeleteNote_Success_Returns200RemovesNote$' ./internal/handlers/ && go test -count=1 -run '^TestEditForm_NonexistentNote_Returns404$' ./internal/handlers/` |
| CHK-07 | RF-07 | Login y logout exitosos responden `200 OK` con `HX-Redirect` y token utilizable | `grep -q 'func TestLogin_Success_Returns200WithHXRedirect(' internal/handlers/auth_handlers_test.go && go test -count=1 -run '^TestLogin_Success_Returns200WithHXRedirect$' ./internal/handlers/ && go test -count=1 -run '^TestLogout_Success_Returns200WithHXRedirect$' ./internal/handlers/` |
| CHK-08 | RF-08 | Entradas no numéricas o cuerpos malformados devuelven `400 Bad Request` sin persistir | `grep -q 'func TestNonNumericID_Returns400NoWrite(' internal/handlers/view_handlers_test.go && go test -count=1 -run '^TestNonNumericID_Returns400NoWrite$' ./internal/handlers/ && go test -count=1 -run '^TestMalformedBody_Returns400NoWrite$' ./internal/handlers/` |
| CHK-09 | RF-09 | Determinismo de la suite bajo múltiples ejecuciones consecutivas | `go test -count=2 ./internal/auth ./database ./internal/middleware ./internal/handlers` |
| CHK-10 | RF-10 | Invariante de dependencias externas en manifiesto del módulo (`go.mod` intacto) | `git diff --exit-code go.mod go.sum` |
| CHK-11 | RNF-01 | Cobertura de líneas >= 70% en paquete internal/auth | `go test -coverprofile=/tmp/cov_auth.out ./internal/auth && go tool cover -func=/tmp/cov_auth.out -o=/tmp/cov_auth.txt && awk '$1=="total:" { sub(/%/, "", $3); if (($3+0) < 70.0) exit 1 }' /tmp/cov_auth.txt` |
| CHK-12 | RNF-01 | Cobertura de líneas >= 70% en paquete database | `go test -coverprofile=/tmp/cov_db.out ./database && go tool cover -func=/tmp/cov_db.out -o=/tmp/cov_db.txt && awk '$1=="total:" { sub(/%/, "", $3); if (($3+0) < 70.0) exit 1 }' /tmp/cov_db.txt` |
| CHK-13 | RNF-01 | Cobertura de líneas >= 70% en paquete internal/middleware | `go test -coverprofile=/tmp/cov_mw.out ./internal/middleware && go tool cover -func=/tmp/cov_mw.out -o=/tmp/cov_mw.txt && awk '$1=="total:" { sub(/%/, "", $3); if (($3+0) < 70.0) exit 1 }' /tmp/cov_mw.txt` |
| CHK-14 | RNF-01 | Cobertura de líneas >= 70% en paquete internal/handlers | `go test -coverprofile=/tmp/cov_hdl.out ./internal/handlers && go tool cover -func=/tmp/cov_hdl.out -o=/tmp/cov_hdl.txt && awk '$1=="total:" { sub(/%/, "", $3); if (($3+0) < 70.0) exit 1 }' /tmp/cov_hdl.txt` |
| CHK-15 | RNF-02 | Tiempo de ejecución de suite completa inferior a 60 segundos | `go test -count=1 -timeout=60s ./...` |
| CHK-16 | RNF-03 | Compilación limpia y análisis estático oficial con cero hallazgos | `go build ./... && golangci-lint run` |

## 2. Criterio de Aprobación

- La tarea solo se considera implementada cuando `.seb/bin/seb verify tests_codigo_existente` reporta **100% de los controles (`CHK-01` a `CHK-16`) en estado `[PASA]`**.
