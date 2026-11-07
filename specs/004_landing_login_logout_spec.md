# Spec 004 — Pantalla de Bienvenida y Desacople de Redirección Automática de Login

- **Tarea ID:** `004`
- **Identificador:** `landing_login_logout`
- **Origen:** Prosa / Petición de usuario
- **Estado:** `APROBADA`

## 1. Contexto y Objetivo (El POR QUÉ)

Actualmente, el sistema acopla de manera forzada la ruta `/login` con el inicio automático del flujo federado OAuth2/OIDC hacia Hodor. Al presionar el botón de "Cerrar Sesión", la aplicación invalida la sesión local y redirige al navegador a `/login`, el cual a su vez dispara inmediatamente la autorización contra Hodor. Dado que el Proveedor de Identidad mantiene activa la sesión SSO del usuario en el navegador, se produce un bucle continuo de reautenticación que impide al usuario desvincularse localmente de la aplicación.
Esta tarea introduce una pantalla de bienvenida y aterrizaje (Landing) desacoplada en `/login` que no redirige automáticamente a Hodor, ofreciendo un punto de detención claro tras el cierre de sesión y requiriendo una acción explícita del usuario para iniciar el flujo de autenticación federada. Asimismo, formaliza el cierre de sesión bajo semántica segura HTTP POST mediante HTMX para prevenir desconexiones accidentales por prefetching.

## 2. Actores y Usuarios

- **Usuario no autenticado:** persona que accede por primera vez a la aplicación o cuyo token ha expirado, visualizando la pantalla de bienvenida con la opción explícita de iniciar sesión.
- **Usuario autenticado:** persona con una sesión local válida que interactúa con la aplicación y decide finalizar su sesión local de forma deliberada.
- **Cliente HTMX:** navegador ejecutando solicitudes AJAX asíncronas que requiere instrucciones de redirección total (`HX-Redirect`) ante el cierre de sesión o ante pérdida de sesión en operaciones parciales.
- **Proveedor de Identidad (Hodor):** servicio externo OIDC que gestiona la autenticación centralizada cuando es invocado explícitamente a través de `/auth/login`.
- **Agente SDD (`seb-verificador`):** evalúa de forma determinista el cumplimiento de los contratos de renderizado, redirecciones y códigos de estado HTTP.

## 3. Historias de Usuario

- **HU-01:** Como usuario que cierra sesión, quiero aterrizar en una pantalla de bienvenida local sin ser reenviado automáticamente al Proveedor de Identidad, para confirmar que mi sesión en esta aplicación fue terminada de forma segura.
- **HU-02:** Como visitante no autenticado, quiero ver una página de bienvenida con información de la aplicación y un botón explícito para iniciar sesión con Hodor, para decidir conscientemente cuándo iniciar el flujo de autenticación.
- **HU-03:** Como usuario ya autenticado, quiero que si navego manualmente a `/` o a `/login` se me redirija directamente al listado de notas, para no interactuar innecesariamente con una pantalla de bienvenida.
- **HU-04:** Como usuario no autenticado que intenta acceder a una ruta protegida o a la raíz `/`, quiero ser redirigido a la pantalla de bienvenida (`/login`), para comprender que requiero autenticarme antes de ingresar al sistema.

## 4. Definiciones de Dominio (Opcional)

- **Pantalla de Bienvenida (Landing Page):** vista pública de nivel superior servida en `/login` con información introductoria y un disparador interactivo explícito de inicio de sesión.
- **Cierre de Sesión Local (Local Logout):** destrucción exclusiva de la cookie de sesión del cliente en esta aplicación sin revocar la sesión central en el Proveedor de Identidad (IdP).
- **Inicio Explícito de Sesión:** acción intencional del usuario (clic en un botón o hipervínculo) que envía una petición a `/auth/login`, diferenciándose de una redirección automática no solicitada.
- **Semántica Segura de Logout (POST):** limitación estricta de la operación de cierre de sesión al método HTTP POST, rechazando peticiones GET para cumplir con la no idempotencia y evitar prefetching de navegadores.
- **Navegación HTMX Completa:** emisión de la cabecera `HX-Redirect` con código `200 OK` (en logout) o `401 Unauthorized` (en expiración de sesión) para forzar al navegador a realizar un cambio global de página en lugar de una sustitución parcial del DOM.

## 5. Requisitos Funcionales (Criterios de Aceptación en EARS)

- **RF-01 (Event-Driven):** CUANDO un usuario no autenticado solicita acceder mediante GET a `/login`, EL SISTEMA DEBE renderizar la pantalla de bienvenida con código de estado HTTP `200 OK`, sin iniciar ni redirigir al flujo de autorización del Proveedor de Identidad.
- **RF-02 (Ubiquitous):** EL SISTEMA DEBE incluir en la pantalla de bienvenida un botón o enlace interactivo visible con el texto "Iniciar sesión con Hodor" cuyo destino apunte a `/auth/login`.
- **RF-03 (State-Driven):** MIENTRAS un usuario posea una sesión local válida y activa, SI solicita acceder a `/login` o a `/`, ENTONCES EL SISTEMA DEBE responder con una redirección HTTP `303 See Other` hacia `/notas`.
- **RF-04 (Event-Driven):** CUANDO un usuario no autenticado solicita acceder mediante GET a la raíz `/`, EL SISTEMA DEBE responder con una redirección HTTP `303 See Other` hacia `/login`.
- **RF-05 (Event-Driven):** CUANDO un usuario solicita iniciar sesión accediendo a `/auth/login`, EL SISTEMA DEBE iniciar el flujo OAuth2 con PKCE generando el desafío criptográfico y redirigiendo al endpoint de autorización del Proveedor de Identidad.
- **RF-06 (Event-Driven):** CUANDO un usuario solicita el cierre de sesión mediante una petición HTTP POST a `/logout` o `/auth/logout` con cabecera `HX-Request: true`, EL SISTEMA DEBE invalidar la cookie de sesión local (`token`) y responder con código de estado `200 OK` y la cabecera `HX-Redirect: /login`.
- **RF-07 (Event-Driven):** CUANDO un usuario solicita el cierre de sesión mediante una petición HTTP POST ordinaria (sin cabecera HTMX) a `/logout` o `/auth/logout`, EL SISTEMA DEBE invalidar la cookie de sesión local (`token`) y responder con una redirección HTTP `303 See Other` hacia `/login`.
- **RF-08 (Unwanted Behavior):** SI se recibe una petición con un método HTTP distinto de POST (como GET) en `/logout` o `/auth/logout`, ENTONCES EL SISTEMA DEBE rechazarla con código de estado `405 Method Not Allowed`.
- **RF-09 (Unwanted Behavior):** SI una petición entrante ordinaria (sin cabecera HTMX) a una ruta protegida carece de sesión válida o contiene un token inválido, ENTONCES EL SISTEMA DEBE responder con una redirección HTTP `303 See Other` hacia `/login`.
- **RF-10 (Event-Driven):** CUANDO una petición entrante desde un cliente HTMX (con cabecera `HX-Request: true`) a una ruta protegida carece de sesión válida o contiene un token inválido, EL SISTEMA DEBE responder con la cabecera `HX-Redirect: /login` y código de estado `401 Unauthorized`.

## 6. Requisitos No Funcionales (Verificables)

- **RNF-01 (Aislamiento de Sesión Local):** El cierre de sesión debe restringirse estrictamente al ámbito de la aplicación local, sin emitir peticiones de red hacia el Proveedor de Identidad externo.
- **RNF-02 (Higiene de Cookies):** La invalidación de la sesión debe garantizar la eliminación efectiva de la cookie de autenticación estableciendo una fecha de expiración en el pasado (`Max-Age: -1` o fecha pretérita) y conservando los atributos `HttpOnly`, `Path=/` y `SameSite=Lax`.
- **RNF-03 (Determinismo en Respuestas HTMX):** Toda respuesta a solicitudes HTMX de cambio de sesión o expiración debe utilizar la cabecera canónica `HX-Redirect: /login` acompañada del código HTTP exacto estipulado (`200 OK` para logout exitoso y `401 Unauthorized` para peticiones no autorizadas).
- **RNF-04 (Testabilidad Determinista y Aislamiento):** Toda la funcionalidad de la pantalla de bienvenida, despacho del flujo federado y cierre de sesión seguro debe validarse mediante pruebas unitarias e integración en memoria (`httptest.Server` y `net/http/httptest`) con cero llamadas a redes externas.

## 7. Casos Límite (Edge Cases)

- **CL-01 (Cierre de sesión idempotente):** Si un cliente sin sesión activa (o cuya cookie ya fue borrada) envía una petición POST a `/logout`, el sistema debe procesar la solicitud exitosamente respondiendo con `200 OK` (`HX-Redirect: /login`) o `303 See Other`, sin registrar errores de servidor ni excepciones.
- **CL-02 (Sesión expirada navegando a la landing):** Si un usuario presenta una cookie con un token expirado o firma corrupta e ingresa mediante GET a `/login`, el sistema debe tratar la sesión como inexistente, limpiar la cookie y renderizar la pantalla de bienvenida con código `200 OK`.
- **CL-03 (Petición GET a `/logout`):** El sistema rechaza cualquier intento de logout por GET con `405 Method Not Allowed`, obligando a usar POST mediante el botón HTMX correspondiente.
- **CL-04 (Interacción HTMX en cierre de sesión):** El botón de cierre de sesión en la barra de navegación debe emitir un `hx-post="/logout"`, desencadenando la redirección de página completa a `/login` al recibir `200 OK` con `HX-Redirect: /login`.

## 8. Fuera de Alcance (Out of Scope)

- Implementación de Single Sign-Out global (Front-Channel o Back-Channel Logout de OIDC) en Hodor.
- Rediseño estructural de los templates del CRUD de notas.
- Creación de formularios o inputs locales de usuario y contraseña en `/login` (la autenticación se delega por completo a Hodor).

## 9. Criterios de Finalización (Definition of Done)

- Todos los requisitos (`RF-01` a `RF-10`, `RNF-01` a `RNF-04`) cuentan con verificación programática en `[PASA]` ejecutada mediante `.seb/bin/seb verify landing_login_logout`.
- El flujo de logout aterriza inequívocamente en `/login` mostrando la pantalla de bienvenida y deteniendo cualquier bucle de redirección hacia Hodor.
- Cero violaciones a `docs/constitution.md`.

## 10. Dudas Abiertas y Puntos por Clarificar

Ninguna duda pendiente.
