# Plan Técnico — codigos_http_canonicos (Spec 002)

- **Spec Origen:** `specs/002_codigos_http_canonicos_spec.md`
- **Verificador Asociado:** `specs/002_codigos_http_canonicos_verificador.md`
- **Estado:** `PLANIFICADO`

## 1. Resumen Arquitectónico y Aislamiento de Lógica

La spec 002 solo exige **un cambio de comportamiento real**: responder `404` cuando se opera sobre una nota inexistente (hoy `500` en edición/actualización y `200` silencioso en borrado). Todo lo demás ya se comporta según el contrato canónico y se cubre con pruebas. El diseño mantiene la arquitectura de la constitución: los handlers siguen orquestando request → `db.Queries` → respuesta; no hay lógica de negocio nueva, ni cambios de esquema, ni cambios en `main.go`, ni regeneración sqlc. Las pruebas usan `net/http/httptest` sobre un router Chi espejo de las rutas reales y una base de datos SQLite temporal por test (patrón exigido por `MEMORY.md`): nunca se ejecuta `main()` ni se toca `crud.db`.

## 2. Mapa de Archivos a Crear o Modificar

| Archivo / Ruta | Acción | Responsabilidad Técnica | Requisitos Cubiertos |
| :------------- | :----- | :---------------------- | :------------------- |
| `internal/handlers/view_handlers.go` | `MODIFICAR` | Distinguir `sql.ErrNoRows` → `404` en edición, actualización y borrado (resto de errores → `500` sigue igual) | `RF-05`, `CL-01` |
| `internal/handlers/testhelpers_test.go` | `CREAR` | BD temporal con esquema, fixtures (usuarios bcrypt, tags, notas), router Chi espejo de `main.go` con middleware, `tpl` desde disco con `funcMap include`, utilidades de cookie/token | `RF-01` a `RF-08` (infra) |
| `internal/handlers/view_handlers_test.go` | `CREAR` | Pruebas de lectura, CRUD exitoso, `404`, `400`, doble borrado | `RF-01`, `RF-03`, `RF-04`, `RF-05`, `RF-08`, `CL-01`, `CL-02` |
| `internal/handlers/auth_handlers_test.go` | `CREAR` | Pruebas de login/logout (`200`+`HX-Redirect`) y credenciales inválidas/vacías (`401` sin token) | `RF-02`, `RF-07`, `CL-04` |
| `internal/middleware/auth_test.go` | `CREAR` | Pruebas de los 4 escenarios de token → `303` a `/login` sin contenido protegido | `RF-06`, `CL-03` |

No se modifica: `main.go`, `internal/db/**` (generado), `sql/**`, `templates/**`, `database/sqlite.go`, `internal/auth/**`.

## 3. Contratos, Firmas y Modelo de Datos

**Pseudocódigo del único cambio de producción (`view_handlers.go`):**

```text
EditNoteFormHandler / UpdateNoteHandler (sobre queries.GetNote):
  err := GetNote(ctx, id)
  SI err NO es nil:
    SI errors.Is(err, sql.ErrNoRows) -> http.Error(w, ..., 404); return
    ENTONCES -> http.Error(w, ..., 500); return   // comportamiento actual conservado

DeleteNoteHandler:
  id := chi.URLParam -> ParseInt; si falla -> 400 (ya existe)
  _, err := GetNote(ctx, id)
  SI errors.Is(err, sql.ErrNoRows) -> http.Error(w, ..., 404); return
  SI err != nil -> 500; return
  err := DeleteNote(ctx, id); SI err != nil -> 500
  -> 200  // efecto verificado por el test con consulta posterior
```

**Contratos de las utilidades de prueba (lado test, no producción):**

```text
newTestDB(t) (conn *sql.DB, queries *db.Queries):
  sql.Open("sqlite3", t.TempDir()/test.db)
  leer ../../sql/schema/schema.sql y Ejecutarlo
  NO usar database.InitDB (log.Fatalf + ruta de esquema fija al CWD de main)

newTestRouter(t, queries, tpl) *chi.Mirror de rutas de main.go:
  públicas: GET /login (Render "login.html"), POST /login (LoginHandler)
  protegidas tras Authenticator(secret): GET /notas, POST /logout,
    GET|POST /crear_nota, POST /editar_nota/{id}, GET /editar_nota/{id}, DELETE /borrar_nota/{id}

newTestTpl(t) *template.Template:
  template.New("").Funcs(FuncMap{include: ...}) ParseGlob("../../templates/*.html")
  // réplica del funcMap de main.go; layout.html llama {{include .contentFile .data}}

withValidToken(t, req, secret): adjunta cookie "token" con auth.GenerateJWT(user, secret)
newExpiredToken(t, secret): jwt.NewWithClaims HS256 con ExpiresAt en el pasado, firmado con el MISMO secret
```

## 4. Decisiones Técnicas Justificadas

- **Decisión 1:** Pruebas con `*db.Queries` **concreto** sobre SQLite temporal (sin refactor a interfaz `db.Querier` en 002).
  - _Alternativa descartada:_ refactor de firmas de handlers a interfaz ahora — ampliaría 002 más allá de los códigos; el refactor queda autorizado para 001 según su Contrato de Desbloqueo (los tests de 002 se ajustarán ahí).
- **Decisión 2:** Existencia previa con `queries.GetNote` antes de borrar (y `ErrNoRows` en edit/update).
  - _Alternativa descartada:_ cambiar la query a `:execresult` + `sqlc generate` — toca `internal/db/` (generado) y `sql/queries/` sin necesidad; prohibido por la constitución editar `internal/db` a mano y el alcance no requiere regeneración.
- **Decisión 3:** Router de prueba espejo de `main.go` dentro de los tests.
  - _Alternativa descartada:_ extraer un constructor de router desde `main.go` — refactor de producción fuera del alcance de 002 (sección 8 de la spec).
- **Decisión 4:** Templates parseados desde disco con `funcMap` replicado en el helper de test.
  - _Alternativa descartada:_ reutilizar el `embed.FS` de `main.go` — inaccesible desde paquetes `internal/*` sin refactor de producción.
- **Decisión 5:** Comandos del verificador **sin caracteres `|`** (ni en shell ni en regex `-run`): el parser de `seb verify` divide las filas de la tabla por `pipe`. Cada test se encadena con `&&` como invocación `go test -run` individual más un `grep -q` que garantiza que la función de prueba existe (evita el falso positivo de `-run` sin coincidencias).
- **Decisión 6:** Prohibido `t.Skip` en cualquier prueba; toda aserción compara el código HTTP exacto (y `Location` / `HX-Redirect` / ausencia de cookie `token` donde aplique). `errcheck` está activo: todo error en tests se maneja vía helpers (`mustExec`, `mustQueries`) — solo `(*sql.DB).Close` está excluido en `.golangci.yml`.

## 5. Secuencia de Tareas Atómicas (TDD: Test en Rojo -> Código -> Test en Verde)

_Regla para el Agente Implementador: ejecutá UNA tarea a la vez en orden. Primero el test (Rojo), luego el código mínimo (Verde). Si el comportamiento actual ya cumple el requisito, el test nace en Verde: la tarea se valida igual con su CHK._

- [ ] **T1. Crear infraestructura de pruebas (BD temporal, fixtures, router espejo, tpl) y test de lectura GET → 200** (`RF-01`)
  - _Archivos:_ `internal/handlers/testhelpers_test.go`, `internal/handlers/view_handlers_test.go`
  - _Hecho cuando:_ `CHK-01` de `specs/002_codigos_http_canonicos_verificador.md` devuelve exit code `0`.
- [ ] **T2. Pruebas de middleware: sin token, inválido, secreto ajeno y expirado → `303` a `/login` sin contenido protegido** (`RF-06`, `CL-03`)
  - _Archivos:_ `internal/middleware/auth_test.go`
  - _Hecho cuando:_ `CHK-06` devuelve exit code `0`.
- [ ] **T3. Pruebas de auth: login/logout exitoso → `200` + `HX-Redirect`; credenciales vacías/incorrectas → `401` sin cookie de token** (`RF-02`, `RF-07`, `CL-04`)
  - _Archivos:_ `internal/handlers/auth_handlers_test.go`
  - _Hecho cuando:_ `CHK-02` y `CHK-07` devuelven exit code `0`.
- [ ] **T4. Pruebas de éxito de notas: crear/editar → `302` con efecto en persistencia; borrar existente → `200` con retirada verificada** (`RF-03`, `RF-04`)
  - _Archivos:_ `internal/handlers/view_handlers_test.go`
  - _Hecho cuando:_ `CHK-03` y `CHK-04` devuelven exit code `0`.
- [ ] **T5. ROJO: tests de `404` para nota inexistente (editar, actualizar, borrar) + doble borrado → VERDE: implementar `ErrNoRows`/`GetNote` previo en `view_handlers.go`** (`RF-05`, `CL-01`)
  - _Archivos:_ `internal/handlers/view_handlers_test.go` (rojo), `internal/handlers/view_handlers.go` (verde)
  - _Hecho cuando:_ `CHK-05` y `CHK-09` devuelven exit code `0`.
- [ ] **T6. Pruebas de `400` sin escritura: identificador no numérico y cuerpo ilegible** (`RF-08`, `CL-02`)
  - _Archivos:_ `internal/handlers/view_handlers_test.go`
  - _Hecho cuando:_ `CHK-08` devuelve exit code `0`.
- [ ] **T7. Suite completa sin regresiones en todos los paquetes** (`RNF-01`)
  - _Archivos:_ (correcciones mínimas de test si algo regresa)
  - _Hecho cuando:_ `CHK-10` devuelve exit code `0`.
- [ ] **T8. Compilación, suite y linter en cero hallazgos (`govet`, `errcheck`, `staticcheck`)** (`RNF-02`)
  - _Archivos:_ (correcciones mínimas de estilo/errores no manejados)
  - _Hecho cuando:_ `CHK-11` devuelve exit code `0`.
