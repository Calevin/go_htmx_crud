# Plan Técnico — landing_login_logout (Spec 004)

- **Spec Origen:** `specs/004_landing_login_logout_spec.md`
- **Verificador Asociado:** `specs/004_landing_login_logout_verificador.md`
- **Estado:** `PLANIFICADO`

## 1. Resumen Arquitectónico y Aislamiento de Lógica

El objetivo es solucionar el bucle de redirección hacia Hodor tras el cierre de sesión, separando la ruta pública de aterrizaje (`/login`) del disparador activo de autenticación federada OIDC/PKCE (`/auth/login`), y formalizando el cierre de sesión mediante peticiones `POST` seguras procesadas por HTMX.

Para preservar la arquitectura del proyecto estipulada en `docs/constitution.md`:
1. **Desacople en transporte HTTP (`main.go` / `internal/handlers`):**
   - La ruta `/login` pasa a ser atendida por un handler dedicado `Landing` que verifica si el usuario ya posee un token válido. Si es así, redirige con `303 See Other` a `/notas`. Si no tiene sesión (o es inválida/expirada), renderiza `templates/login.html` dentro de `templates/layout.html` respondiendo con `200 OK`.
   - La ruta `/auth/login` se mantiene como el único punto de inicio del flujo de autorización OIDC con PKCE hacia Hodor.
   - La raíz `/` (`Root`) evalúa si el token es válido: si lo es, redirige con `303 See Other` a `/notas`; de lo contrario, redirige con `303 See Other` a `/login`.
2. **Cierre de sesión seguro y determinismo HTMX:**
   - `/logout` y `/auth/logout` exigen estrictamente el método HTTP `POST`. Peticiones `GET` son rechazadas con `405 Method Not Allowed`.
   - Ante peticiones HTMX (`HX-Request: true`), el handler responde con `200 OK` y cabecera `HX-Redirect: /login`, eliminando la cookie `token` (`Max-Age: -1`).
   - Ante peticiones ordinarias `POST`, responde con `303 See Other` hacia `/login`.
3. **Middleware de autenticación (`internal/middleware`):**
   - Las peticiones sin autenticación o con token inválido hacia rutas protegidas son redirigidas a `/login` (en vez de `/auth/login`), emitiendo `303 See Other` para navegación ordinaria o `401 Unauthorized` + `HX-Redirect: /login` para HTMX.

## 2. Mapa de Archivos a Crear o Modificar

| Archivo / Ruta | Acción (`CREAR` / `MODIFICAR`) | Responsabilidad Técnica | Requisitos Cubiertos (`RF-XX`) |
| :------------- | :----------------------------- | :---------------------- | :----------------------------- |
| `internal/middleware/auth.go` | `MODIFICAR` | Actualizar `handleUnauthorized` para redirigir hacia `/login` en lugar de `/auth/login`. | `RF-09`, `RF-10`, `RNF-03` |
| `internal/middleware/auth_test.go` | `MODIFICAR` | Adaptar assertions de redirección no autenticada hacia `/login` y añadir casos deterministas. | `RF-09`, `RF-10`, `RNF-04` |
| `internal/handlers/auth_handlers.go` | `MODIFICAR` | Incorporar métodos `Landing` y `Root` en `AuthHandler`, restringir `Logout` a `POST` con código `405` en métodos no permitidos. | `RF-01`, `RF-02`, `RF-03`, `RF-04`, `RF-06`, `RF-07`, `RF-08`, `RNF-01`, `RNF-02` |
| `internal/handlers/auth_handlers_test.go` | `MODIFICAR` | Agregar tests unitarios para `Landing`, `Root`, rechazo 405 en `Logout` y validación de cookies. | `RF-01` a `RF-08`, `RNF-01` a `RNF-04` |
| `templates/login.html` | `MODIFICAR` | Transformar el formulario de usuario/contraseña en pantalla de bienvenida con botón explícito a `/auth/login`. | `RF-01`, `RF-02` |
| `main.go` | `MODIFICAR` | Conectar `r.Get("/login", authHandler.Landing)`, `r.Get("/auth/login", authHandler.Login)`, `r.Get("/", authHandler.Root)` y registrar métodos `POST` y `405` en `/logout`. | `RF-01`, `RF-03`, `RF-04`, `RF-05`, `RF-08` |

## 3. Contratos, Firmas y Modelo de Datos

### 3.1. Estructura de `AuthHandler` y nuevos métodos
```go
type AuthHandler struct {
    oauthConfig    *oauth2.Config
    tokenValidator *auth.TokenValidator
    tpl            *template.Template
}

func NewAuthHandler(oauthConfig *oauth2.Config, validator *auth.TokenValidator, tpl *template.Template) *AuthHandler

// Landing atiende GET /login: renderiza la vista de bienvenida o redirige a /notas si ya está autenticado.
func (h *AuthHandler) Landing(w http.ResponseWriter, r *http.Request)

// Root atiende GET /: redirige a /notas si está autenticado, o a /login si es anónimo.
func (h *AuthHandler) Root(w http.ResponseWriter, r *http.Request)

// Logout atiende POST /logout y POST /auth/logout: invalida cookie y responde según HTMX o estándar.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request)
```

### 3.2. Lógica Algorítmica de `Landing`
1. Extraer cookie `token`.
2. Si la cookie existe y `h.tokenValidator.ValidateToken(cookie.Value)` es exitoso:
   - Redirigir con `http.Redirect(w, r, "/notas", http.StatusSeeOther)`.
   - Retornar.
3. Si la cookie no existe o el token es inválido:
   - Si había cookie inválida, forzar su eliminación en la respuesta (`Max-Age: -1`).
   - Renderizar `templates/login.html` mediante `Render(h.tpl, w, "login.html", nil)` con código HTTP `200 OK`.

### 3.3. Lógica Algorítmica de `Logout`
1. Si `r.Method != http.MethodPost`:
   - Responder `http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)`.
   - Retornar.
2. Establecer cookie `token` vacía con `MaxAge: -1`, `Expires: time.Unix(0, 0)`, `Path: "/"`, `HttpOnly: true`, `SameSite: http.SameSiteLaxMode`.
3. Si `r.Header.Get("HX-Request") == "true"`:
   - Establecer cabecera `HX-Redirect: /login`.
   - Responder con `w.WriteHeader(http.StatusOK)`.
   - Retornar.
4. Responder con `http.Redirect(w, r, "/login", http.StatusSeeOther)`.

## 4. Decisiones Técnicas Justificadas

- **Decisión 1: Reutilizar `Render` existente en `internal/handlers`:**
  - _Justificación:_ En `view_handlers.go` ya existe la función `Render(tpl, w, contentFile, data)` que envuelve el contenido en `layout.html`. Se reutiliza dicha función en `AuthHandler.Landing`, evitando duplicación de lógica de layouts.
- **Decisión 2: Inyección de `*template.Template` en `AuthHandler`:**
  - _Justificación:_ Permite que `AuthHandler` sea autocontenido para servir la pantalla de bienvenida y facilita las pruebas unitarias mediante templates cargados desde memoria o disco en `httptest.Server`.
- **Decisión 3: Rechazo explícito con `405 Method Not Allowed` para peticiones GET en logout:**
  - _Justificación:_ Cumple con las respuestas de clarificación del usuario y con el RFC 7231, impidiendo que extensiones o precargas automáticas del navegador cierren la sesión de forma fortuita.

## 5. Secuencia de Tareas Atómicas (TDD: Test en Rojo -> Código -> Test en Verde)

- [ ] **T1. Actualizar el Middleware de Autenticación para Redirigir a `/login`** (`RF-09`, `RF-10`, `RNF-03`)
  - _Archivos:_ `internal/middleware/auth.go`, `internal/middleware/auth_test.go`
  - _Acción:_ Modificar `handleUnauthorized` para emitir `Location: /login` (`303 See Other`) en peticiones ordinarias y `HX-Redirect: /login` (`401 Unauthorized`) en HTMX. Adaptar los tests correspondientes.
  - _Hecho cuando:_ Los comandos `CHK-08` y `CHK-09` de `specs/004_landing_login_logout_verificador.md` devuelven exit code `0`.

- [ ] **T2. Restricción del Cierre de Sesión a Método POST y Respuestas Seguras** (`RF-06`, `RF-07`, `RF-08`, `RNF-01`, `RNF-02`)
  - _Archivos:_ `internal/handlers/auth_handlers.go`, `internal/handlers/auth_handlers_test.go`
  - _Acción:_ Implementar validación de método POST en `Logout`, emisión de cabecera `HX-Redirect: /login` con código `200 OK` para HTMX, `303 See Other` hacia `/login` para estándar, rechazo 405 ante GET y eliminación estricta de la cookie `token`.
  - _Hecho cuando:_ Los comandos `CHK-05`, `CHK-06`, `CHK-07` y `CHK-10` devuelven exit code `0`.

- [ ] **T3. Implementar Handlers de Landing Page (`/login`) y Raíz (`/`)** (`RF-01`, `RF-02`, `RF-03`, `RF-04`, `RF-05`)
  - _Archivos:_ `internal/handlers/auth_handlers.go`, `internal/handlers/auth_handlers_test.go`
  - _Acción:_ Inyectar `tpl *template.Template` en `AuthHandler`. Implementar `Landing` (con evaluación de token activo -> 303 a `/notas`, o render 200 de `login.html`) y `Root` (303 a `/notas` si está autenticado, 303 a `/login` si es anónimo).
  - _Hecho cuando:_ Los comandos `CHK-01`, `CHK-02`, `CHK-03` y `CHK-04` devuelven exit code `0`.

- [ ] **T4. Actualización de Vistas HTML y Composición en `main.go`** (`RF-01`, `RF-02`, `RF-08`)
  - _Archivos:_ `templates/login.html`, `main.go`
  - _Acción:_ Reemplazar formulario de credenciales locales en `templates/login.html` por pantalla de bienvenida con botón visible "Iniciar sesión con Hodor" que dirija a `/auth/login`. En `main.go`, pasar `tpl` a `NewAuthHandler`, registrar las nuevas rutas y configurar handlers 405 para GET en `/logout` y `/auth/logout`.
  - _Hecho cuando:_ `go build ./...` compila exitosamente y los controles de integración validan el flujo de rutas.

- [ ] **T5. Verificación Determinista Integral y Limpieza de Linter** (`RNF-04`, `GLOBAL`)
  - _Archivos:_ Repositorio completo
  - _Acción:_ Ejecutar la suite completa determinista y corregir cualquier observación de `golangci-lint`.
  - _Hecho cuando:_ Los comandos `CHK-11` y `CHK-12` devuelven exit code `0`.
