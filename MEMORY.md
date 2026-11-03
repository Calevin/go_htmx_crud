# MEMORY.md — Bitácora Técnica y Restricciones (seb-harness)

> **Regla de Higiene de Contexto:** Este archivo almacena lo aprendido empíricamente en el proyecto entre sesiones.
> - **Límite duro:** Máximo 150 líneas (~15 KB). Resume o elimina entradas obsoletas.
> - **Promoción:** Si un aprendizaje se vuelve una ley permanente de arquitectura, muévelo a `docs/constitution.md`.
> - **Seguridad:** Prohibido guardar credenciales, tokens, rutas privadas del host o datos personales.
> - **Invariante del Agente:** Antes de proponer un diseño en `seb-planificador` o escribir código, verifica las secciones 2 y 3 para no reintentar enfoques fallidos.

## 1. Estado Actual del Proyecto
- **Fase SDD activa:** Spec `002_codigos_http_canonicos` implementada y verificada (`VERIFIED`). Próximo paso: retomar spec `001_tests_codigo_existente` (desbloqueada).
- **Última Spec verificada:** `002_codigos_http_canonicos` (11/11 controles en `[PASA]`).
- **Estado de la suite de verificación:** Verde (100% de controles deterministas aprobados por `.seb/bin/seb verify 002`).
- **Comandos oficiales vigentes:** `go build ./...` / `go test ./...` / `golangci-lint run` / `sqlc generate` + `git diff --exit-code internal/db/`.

## 2. Callejones sin Salida y Enfoques Descartados (NO REINTENTAR)
*Registro de librerías, patrones o configuraciones que se probaron en este proyecto y fallaron o resultaron incompatibles.*
- *(Vacío inicialmente — Formato requerido: `[Fecha] [Tarea/Spec] Qué se intentó -> Por qué falló -> Alternativa adoptada`)*

## 3. Restricciones Técnicas y Trampas Conocidas (Gotchas)
*Limitaciones del entorno (NixOS, kernel, contenedores, base de datos, versiones del compilador/runtime) o del dominio que no son obvias leyendo el código.*
- `[2026-11-03] [tests]` No existe ningún `*_test.go` en el repo -> `go test ./...` pasa vacío (no valida nada); la primera spec que toque handlers debe crear la suite base con `httptest` + interfaz `db.Queries` (`emit_interface: true` ya habilitado en `sqlc.yaml`).
- `[2026-11-03] [repo]` El binario compilado `go_htmx_crud` (~17 MB) y `crud.db` están rastreados en git pese a estar en `.gitignore` (ignoran solo archivos no rastreados) -> requieren `git rm --cached`; deuda de higiene pendiente.
- `[2026-11-03] [auth]` `main.go` crea un usuario de prueba (`testuser`/`password123`) al arrancar -> cualquier test de integración que levante `main` real chocará con esa semilla; usar BD temporal y no ejecutar `main()` completo en tests.

## 4. Riesgos Conocidos y Deuda Técnica Activa
*Riesgos arquitectónicos o casos borde postergados conscientemente para futuras specs.*
- *(Vacío inicialmente — Formato requerido: `[ID-Spec] Descripción del riesgo -> Condición de mitigación`)*

## 5. Decisiones Técnicas Relevantes (Histórico Reciente)
- *(Vacío inicialmente — Formato requerido: `[D-XX] Decisión tomada -> Alternativa descartada -> Motivo`)*
- **[2026-11-03] [Spec 001 / Discrepancias codigo-spec]** Bloqueo de planificacion 001: el codigo real no cumple la spec aprobada. (1) RF-05: middleware sin token/invalido redirige 303, la spec pide 401. (2) RF-06: crear/editar nota devuelven 302, spec pide 200. (3) RF-06/CL-03: ID inexistente en editar da 500 y en borrar da 200, spec pide 404. (4) RF-07: login devuelve 200 + header HX-Redirect, spec pide 303; ademas NO existe endpoint de registro. (5) RF-06 ownership + CL-01: la tabla notes no tiene user_id (notas globales, sin dueno); el aislamiento por usuario es imposible sin cambiar el esquema. CL-05 exige detenerse y volver a seb-clarificador: prohibido tocar produccion ni relajar pruebas.
- **[2026-11-03] [Bloqueo SDD]** Spec 001 bloqueada por dependencia de 002_codigos_http_canonicos (codigos HTTP canonicos); retomar recien cuando 002 tenga verified: true en .seb/seb.lock, reescribiendo RF-05/RF-06/RF-07/RF-08/CL-01/CL-04 segun el Contrato de Desbloqueo.
- **[2026-11-03] [Deuda Tecnica / Scope Fork]** Spec 002: validacion de campos vacios (nombre de nota en alta/edicion) y manejo de tag_id numerico inexistente postergados fuera de alcance (Opciones B y D); requieren specs futuras de validacion e integridad de datos. Login con credenciales vacias resuelto en RF-07 con 401.
- **[2026-11-03] [Decision Tecnica]** [D-01] Spec 002: pruebas usan *db.Queries concreto sobre SQLite temporal (t.TempDir) en vez de refactorizar handlers a interfaz db.Querier -> refactor queda para spec 001 segun su Contrato de Desbloqueo. Router de prueba espejo de main.go y templates parseados desde disco con funcMap include replicado, sin tocar produccion.
