# Plan Técnico — tests_codigo_existente (Spec 001)

- **Spec Origen:** `specs/001_tests_codigo_existente_spec.md`
- **Verificador Asociado:** `specs/001_tests_codigo_existente_verificador.md`
- **Estado:** `PLANIFICADO`

## 1. Resumen Arquitectónico y Aislamiento de Lógica

La spec 001 establece la suite base de pruebas automatizadas para los cuatro paquetes lógicos del repositorio (`internal/auth`, `internal/middleware`, `internal/handlers`, `database`), garantizando una red de seguridad contra regresiones con una cobertura de líneas superior al 70% por paquete (`RNF-01`).

La implementación previa y verificación formal de la spec base `002_codigos_http_canonicos` ya construyó la infraestructura base de pruebas HTTP en `internal/handlers/testhelpers_test.go`, cubriendo exhaustivamente los manejadores HTTP (`internal/handlers`, cobertura actual 73.2%) y el middleware de control de acceso (`internal/middleware`, cobertura actual 86.7%).

Por ende, el trabajo técnico restante de la spec 001 se focaliza quirúrgicamente en:
1. Crear la suite de pruebas unitarias para el módulo de credenciales y tokens (`internal/auth/jwt_test.go`), validando emisión, claims, expiración y verificación de firma criptográfica.
2. Crear la suite de pruebas de persistencia para el módulo de base de datos (`database/sqlite_test.go`), validando inicialización aislada sobre directorio temporal (`t.TempDir()`), ejecución del esquema canónico (`sql/schema/schema.sql`) y manejo de errores.
3. Asegurar que los cuatro paquetes alcancen y mantengan individualmente >= 70% de cobertura de código sin dependencias externas ni estado compartido entre pruebas.

El diseño respeta estrictamente `docs/constitution.md` y `MEMORY.md`:
- Ninguna prueba ejecuta `main()` completo ni abre puertos de red reales.
- Ninguna prueba lee, escribe ni depende del archivo de base de datos `crud.db` ni de variables del entorno operador (`.env`).
- Prohibido modificar el código generado en `internal/db/` o añadir librerías externas a `go.mod`.

## 2. Mapa de Archivos a Crear o Modificar

| Archivo / Ruta | Acción | Responsabilidad Técnica | Requisitos Cubiertos |
| :------------- | :----- | :---------------------- | :------------------- |
| `internal/auth/jwt_test.go` | `CREAR` | Pruebas unitarias de generación y verificación de JWT, validación de claims (`Username`, `ExpiresAt`), firma con HMAC-SHA256, rechazo de firma ajena y tokens malformados | `RF-01`, `RF-04`, `RF-07`, `RF-10`, `RNF-01` |
| `database/sqlite_test.go` | `CREAR` | Pruebas de inicialización de SQLite con `InitDB` sobre `t.TempDir()`, verificación de tablas creadas por `schema.sql` y validación de error en `executeSchema` | `RF-01`, `RF-02`, `RF-03`, `RF-10`, `RNF-01` |
| `specs/001_tests_codigo_existente_plan.md` | `CREAR` | Plan técnico y desglose TDD de la tarea | Gobernanza SDD |
| `specs/001_tests_codigo_existente_verificador.md` | `CREAR` | Matriz de controles deterministas parseable por `.seb/bin/seb verify` | Gobernanza SDD |

*Archivos ya existentes validados en Spec 002 (sin modificaciones de producción requeridas):*
- `internal/middleware/auth_test.go` (cubre `RF-05`, `CL-03`, `RNF-01`).
- `internal/handlers/view_handlers_test.go` (cubre `RF-01`, `RF-06`, `RF-08`, `CL-01`, `CL-02`, `RNF-01`).
- `internal/handlers/auth_handlers_test.go` (cubre `RF-02`, `RF-07`, `CL-04`, `RNF-01`).
- `internal/handlers/testhelpers_test.go` (fixtures en memoria y SQLite temporal con `t.TempDir()`).

## 3. Contratos, Firmas y Diseño de Pruebas

### 3.1. Pruebas Unitarias de JWT (`internal/auth/jwt_test.go`)

```text
TestGenerateJWT_ValidClaims:
  - Generar token para username "alice" con secreto conocido.
  - Parsear el token resultante con jwt.ParseWithClaims usando el mismo secreto.
  - Afirmar que err == nil y token.Valid == true.
  - Afirmar que claims.Username == "alice".
  - Afirmar que claims.ExpiresAt está situado en una ventana razonable (+24h desde time.Now()).

TestGenerateJWT_InvalidSecretRejection:
  - Generar token con secretA.
  - Intentar validar token con secretB.
  - Afirmar que el parseo falla (token inválido / firma no coincide).

TestGenerateJWT_MalformedToken:
  - Intentar parsear una cadena arbitraria ("token-invalido-random").
  - Afirmar que el parseo devuelve error.
```

### 3.2. Pruebas de Persistencia (`database/sqlite_test.go`)

```text
TestInitDB_Success:
  - Resolver la ruta relativa hacia la raíz del repositorio si la prueba corre desde el directorio "database/".
  - Crear ruta a base de datos temporal: filepath.Join(t.TempDir(), "test_init.db").
  - Invocar db := InitDB(tempDBPath).
  - Asegurar defer db.Close().
  - Ejecutar db.Ping() y verificar err == nil.
  - Consultar sqlite_master para confirmar la creación de las tablas "users", "tags" y "notes".
  - Insertar y leer un registro para confirmar operatividad completa de lectura/escritura.

TestExecuteSchema_FileNotFound:
  - Abrir una base de datos SQLite en memoria (o archivo temporal).
  - Cambiar temporalmente el working directory a un directorio vacío (t.TempDir()).
  - Invocar executeSchema(db).
  - Afirmar que executeSchema devuelve error (cubre rama de fallo sin abortar el proceso).
```

## 4. Desglose Atómico de Tareas TDD

- **TAREA-01:** Implementar `internal/auth/jwt_test.go`
  - Escribir `TestGenerateJWT_ValidClaims`, `TestGenerateJWT_InvalidSecretRejection` y `TestGenerateJWT_MalformedToken`.
  - Ejecutar `go test -v -cover ./internal/auth` confirmando cobertura del 100%.

- **TAREA-02:** Implementar `database/sqlite_test.go`
  - Escribir `TestInitDB_Success` gestionando el cwd para ubicar `sql/schema/schema.sql` y validando tablas con `sqlite_master`.
  - Escribir `TestExecuteSchema_FileNotFound` para cubrir la rama de error de lectura de esquema.
  - Ejecutar `go test -v -cover ./database` confirmando cobertura >= 70%.

- **TAREA-03:** Verificación de Cobertura Global y Determinismo
  - Ejecutar `go test -cover ./internal/auth ./database ./internal/middleware ./internal/handlers`.
  - Auditar que cada uno de los 4 paquetes supere individualmente el umbral de 70% de cobertura de líneas.
  - Ejecutar `golangci-lint run` garantizando 0 advertencias.
  - Ejecutar suite completa con `-count=2` para certificar determinismo sin carreras ni efectos secundarios.

## 5. Estrategia de Verificación y Criterios de Aceptación

La aprobación de esta especificación depende de que el archivo `specs/001_tests_codigo_existente_verificador.md` alcance el 100% de controles en estado `[PASA]` al ejecutarse mediante `.seb/bin/seb verify tests_codigo_existente`.
