# MEMORY.md — Bitácora Técnica y Restricciones (seb-harness)

> **Regla de Higiene de Contexto:** Este archivo almacena lo aprendido empíricamente en el proyecto entre sesiones.
> - **Límite duro:** Máximo 150 líneas (~15 KB). Resume o elimina entradas obsoletas.
> - **Promoción:** Si un aprendizaje se vuelve una ley permanente de arquitectura, muévelo a `docs/constitution.md`.
> - **Seguridad:** Prohibido guardar credenciales, tokens, rutas privadas del host o datos personales.
> - **Invariante del Agente:** Antes de proponer un diseño en `seb-planificador` o escribir código, verifica las secciones 2 y 3 para no reintentar enfoques fallidos.

## 1. Estado Actual del Proyecto
- **Fase SDD activa:** `seb-iniciador` completado — `docs/constitution.md` generada (secciones A–F).
- **Última Spec verificada:** Ninguna (directorio `specs/` vacío).
- **Estado de la suite de verificación:** Pendiente de primera ejecución con `.seb/bin/seb verify`.
- **Comandos oficiales vigentes:** `go build ./...` / `go test ./...` / `go vet ./...` (sin tests ni linter externos aún).

## 2. Callejones sin Salida y Enfoques Descartados (NO REINTENTAR)
*Registro de librerías, patrones o configuraciones que se probaron en este proyecto y fallaron o resultaron incompatibles.*
- *(Vacío inicialmente — Formato requerido: `[Fecha] [Tarea/Spec] Qué se intentó -> Por qué falló -> Alternativa adoptada`)*

## 3. Restricciones Técnicas y Trampas Conocidas (Gotchas)
*Limitaciones del entorno (NixOS, kernel, contenedores, base de datos, versiones del compilador/runtime) o del dominio que no son obvias leyendo el código.*
- `[2026-11-03] [flake.nix]` El devshell solo expone `go` y `sqlite`: **`sqlc` y `golangci-lint` NO están instalados** -> `sqlc generate` falla fuera del PATH del sistema; no se pueden declarar como comandos oficiales de verificación hasta actualizar el flake (tarea aparte, requiere editar `flake.nix`).
- `[2026-11-03] [tests]` No existe ningún `*_test.go` en el repo -> `go test ./...` pasa vacío (no valida nada); la primera spec que toque handlers debe crear la suite base con `httptest` + interfaz `db.Queries` (`emit_interface: true` ya habilitado en `sqlc.yaml`).
- `[2026-11-03] [repo]` El binario compilado `go_htmx_crud` (~17 MB) y `crud.db` están rastreados en git pese a estar en `.gitignore` (ignoran solo archivos no rastreados) -> requieren `git rm --cached`; deuda de higiene pendiente.
- `[2026-11-03] [auth]` `main.go` crea un usuario de prueba (`testuser`/`password123`) al arrancar -> cualquier test de integración que levante `main` real chocará con esa semilla; usar BD temporal y no ejecutar `main()` completo en tests.

## 4. Riesgos Conocidos y Deuda Técnica Activa
*Riesgos arquitectónicos o casos borde postergados conscientemente para futuras specs.*
- *(Vacío inicialmente — Formato requerido: `[ID-Spec] Descripción del riesgo -> Condición de mitigación`)*

## 5. Decisiones Técnicas Relevantes (Histórico Reciente)
- *(Vacío inicialmente — Formato requerido: `[D-XX] Decisión tomada -> Alternativa descartada -> Motivo`)*
