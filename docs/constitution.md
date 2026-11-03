# Constitución del Proyecto — go_htmx_crud

> **Ley Suprema (`seb-harness`):** Este documento define los principios innegociables del proyecto. Toda especificación (`*_spec.md`), plan (`*_plan.md`), verificador (`*_verificador.md`) e implementación debe cumplirlos estrictamente.

## A. Naturaleza del Proyecto

- **Propósito y Misión:** Proyecto de prueba y referencia (reference implementation) del stack **Go + HTMX + sqlc**. El valor principal no es el dominio, sino validar que el stack funciona de punta a punta (routing, render server-side, queries tipadas, auth) y servir como ejemplo copiable del stack. La funcionalidad de "administrador de notas" es **secundaria e ilustrativa**: cualquier decisión de diseño se justifica por demostrar una característica del stack, no por las necesidades del dominio de notas.
- **Usuarios / Consumidores:** (1) Desarrollador humano que ejecuta y navega la app en `http://localhost:3000`; (2) el propio agente de IA que extiende el proyecto bajo `seb-harness`; (3) lectores externos que usan el repo como ejemplo del stack. No hay microservicios consumidores ni API pública externa.
- **Tipo de Sistema y Alcance:** Monolito web server-side-rendered de un solo binario Go. Alcance: HTTP server con Chi, templates HTML + HTMX para interacción parcial, SQLite como única persistencia, autenticación JWT (login/logout + middleware). Fuera de alcance: microservicios, frontends SPA, múltiples bases de datos, colas, caché, despliegue a producción.

## B. Stack Tecnológico (Reglas de Implementación)

- **Lenguaje y Runtime:** Go 1.24.1+ (módulo `github.com/Calevin/go_htmx_crud`), entorno de desarrollo NixOS vía `flake.nix` + direnv (`.envrc` → `use flake`).
- **Frameworks y Librerías Core (DEBEN reutilizarse, ya presentes en `go.mod`):**
  - `github.com/go-chi/chi/v5` — enrutador HTTP (nunca `net/http` mux ni otros routers).
  - `github.com/mattn/go-sqlite3` — driver SQLite.
  - `github.com/golang-jwt/jwt/v5` — emisión/validación de JWT.
  - `golang.org/x/crypto` (bcrypt) — hash de contraseñas.
  - `github.com/joho/godotenv` — carga de `.env`.
  - sqlc — generación de código a partir de `sql/queries/*.sql` (config en `sqlc.yaml`, salida en `internal/db/`).
  - HTMX vía templates server-side (`templates/*.html` + `embed.FS`); el JS de HTMX se sirve desde `static/`.
- **Persistencia e Infraestructura:** SQLite (archivo `./crud.db` creado al arrancar desde `database/sqlite.go`); esquema canónico en `sql/schema/schema.sql`. Sin Docker, sin colas, sin caché. Variables de entorno en `.env` (`JWT_SECRET` obligatoria; nunca commitear `.env`).
- **Comandos Oficiales de Verificación Programática:**
  - **Compilación / Build:** `go build ./...`
  - **Tests Unitarios / Integración:** `go test ./...`
  - **Linter / Análisis Estático:** `golangci-lint run` (incluye `govet`; config en `.golangci.yml`, gate de 0 hallazgos).
  - **Sincronía sqlc:** tras modificar `sql/schema/*.sql` o `sql/queries/*.sql` debe ejecutarse `sqlc generate` y confirmar `git diff --exit-code internal/db/` (código generado en sync).
- **Prohibiciones de Stack:**
  - Prohibido añadir nuevas dependencias externas sin aprobación explícita en la spec/plan.
  - Prohibido reemplazar librerías existentes del proyecto por alternativas equivalentes (ej. `gorilla/mux` por Chi, `pgx` por SQLite, `templ` por `html/template`).
  - Prohibido introducir frameworks web, ORMs (GORM, sqlx) o libs de templates: la queries se generan con sqlc y las vistas se renderizan con `html/template` + HTMX.

## C. Reglas de Dominio y Lógica de Negocio

- **Invariantes del Dominio:**
  - Toda operación de escritura sobre notas requiere un JWT válido (middleware `internal/middleware.Authenticator`); no hay endpoint público que lea o escriba notas.
  - Contraseñas almacenadas **solo** como hash bcrypt; prohibido guardar o loggear contraseñas en claro.
  - Las notas pertenecen al usuario autenticado: ninguna query permite listar/editar/borrar notas de otro usuario sin filtro por usuario autenticado (si el dominio lo permite por diseño actual, documentarlo en la spec; no introducir silenciosamente multi-tenant).
  - `JWT_SECRET` debe estar definida al arrancar; si falta, el proceso aborta (`log.Fatal`).
- **Fronteras de Datos y Consistencia:**
  - SQLite opera en modo default de consistencia; las operaciones multi-query que deben ser atómicas se envuelven en transacción (`*sql.Tx` vía sqlc) — nunca asumir atomicidad entre dos llamadas sueltas a `Queries`.
  - No hay zonas horarias ni dinero: si se añaden timestamps, usar UTC y formato RFC3339.
  - Compatibilidad hacia atrás: los cambios en `sql/schema/schema.sql` que rompan el esquema existente deben contemplar migración o regeneración de `crud.db` documentada en la spec.
- **Aislamiento de Lógica:** La lógica de negocio vive fuera de los handlers: `internal/handlers` solo orquesta request → `db.Queries` → respuesta (HTML o fragmento HTMX); `internal/db` es código generado y no se edita a mano; `internal/auth` y `internal/middleware` contienen la lógica de credenciales. Los handlers deben ser testables con `net/http/httptest` y una interfaz `db.Queries` (sqlc emite `emit_interface: true`), sin levantar el servidor en `:3000`.

## D. Estructura y Estilo de Código

- **Topología de Directorios:**
  - `main.go` — composición: env, DB, router Chi, rutas, embed de `templates/` y `static/`.
  - `database/` — apertura e inicialización de SQLite (`sqlite.go`).
  - `sql/schema/`, `sql/queries/` — fuente de verdad SQL (esquema y consultas).
  - `internal/db/` — **generado por sqlc; prohibido editar a mano.**
  - `internal/handlers/` — handlers HTTP (auth y vistas/notas).
  - `internal/middleware/`, `internal/auth/` — JWT, autenticación, bcrypt.
  - `templates/` — vistas HTML + HTMX (layout, CRUD, login).
  - `static/` — CSS y assets estáticos.
  - `docs/` — `constitution.md` (ley suprema) y afines; `specs/` — artefactos SDD; `.seb/` — harness.
- **Patrones Arquitectónicos:** Server-side rendering con HTMX sobre un monolito en capas delgadas (transporte HTTP → handler → queries sqlc → SQLite). Sin capa de repositorio adicional sobre sqlc: `db.Queries` **es** la interfaz de datos.
- **Convenciones de Nomenclatura e Idioma:** Código, identificadores y mensajes de commit en **inglés** (convención Go: `camelCase`, interfaces sin sufijo `-er` innecesario). Comentarios en el código y documentación (`docs/`, `MEMORY.md`, specs) en **español**. Nombres de rutas y templates en español (`/crear_nota`, `notas.html`) — se conservan por ser ya existentes. Sin emojis en archivos de código.
- **Prohibiciones de Diseño:**
  - Prohibido editar manualmente `internal/db/**` (es código generado).
  - Prohibido lógica de negocio en templates ni en closures anónimas de `main.go` complejas: mover a `internal/handlers`.
  - Prohibido estado global mutable compartido (más allá de `tpl` y `templateFS` de embed, que son de solo lectura tras `init`).
  - Prohibido commitear artefactos de build o datos: binarios compilados (`go_htmx_crud`), bases de datos (`crud.db`), `.env`, `.direnv/` (ver `.gitignore`).
  - Prohibido el acoplamiento circular entre paquetes `internal/*`.

## E. Manejo de Errores y Validaciones

- **Estrategia de Errores:** Errores Go con valores de retorno (`err != nil`); wrapping de contexto con `fmt.Errorf("...: %w", err)` en las fronteras. En handlers: nunca continuar con `err != nil`; responder con el código HTTP correspondiente (400 validación, 401 no autenticado, 500 interno) y renderizar la vista de error o devolver un fragmento HTMX. No se usan excepciones ni panics para control de flujo (el `Recoverer` de Chi es red de seguridad, no control de errores).
- **Validación de Entrada:** Validar y normalizar los formularios (login, crear/editar nota) en el handler **antes** de llamar a `db.Queries`; los parámetros de URL (`{id}`) se parsean y validan antes de tocar la BD (IDs inválidos → 400, no query). Nunca confiar en datos del cliente para autorización (el `user_id` sale del JWT, no del body).
- **Observabilidad y Seguridad:** Logs con `log` estándar de Go y `middleware.Logger` de Chi (formato texto, suficiente para dev). **Prohibido absolutamente** loggear `JWT_SECRET`, valores de `.env`, hashes de contraseña, tokens JWT completos ni PII. Los mensajes de error al usuario no deben exponer detalles internos (stack traces, SQL).

## F. Comportamiento del Agente de IA (Reglas SDD — `seb-harness`)

1. **La Spec Manda:** Ninguna funcionalidad se implementa sin un archivo `specs/<NNN>_<tarea>_spec.md` aprobado y sin etiquetas `[NECESITA ACLARACIÓN]`.
2. **Testabilidad Programática Obligatoria:** Ningún plan (`specs/<NNN>_<tarea>_plan.md`) es válido sin su correspondiente `specs/<NNN>_<tarea>_verificador.md` con comandos que devuelvan código de salida `0`.
3. **Anti-Reward-Hacking:** Está terminantemente prohibido modificar `docs/constitution.md`, `specs/*_spec.md` o `specs/*_verificador.md` durante la implementación, así como deshabilitar o relajar tests para forzar que pasen.
4. **Memoria Persistente (`MEMORY.md`):** Antes de planificar o codificar, el agente debe consultar `MEMORY.md` para evitar callejones sin salida ya documentados, y registrar allí cualquier nueva limitación técnica descubierta.
5. **Commits Protegidos:** Todo commit de código debe pasar la puerta determinista `.seb/bin/seb verify <tarea>`.
