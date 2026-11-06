# Spec 003 — Integración de Autenticación Federada con Hodor (OIDC / RS256)

- **Tarea ID:** `003`
- **Identificador:** `autenticacion_hodor`
- **Origen:** Prosa / Documento de análisis (`implementar_auth_hodor.md`)
- **Estado:** `APROBADA`

## 1. Contexto y Objetivo (El POR QUÉ)

Actualmente, el sistema gestiona la identidad mediante un esquema local monolítico (almacenamiento de credenciales y contraseñas en la base de datos local y emisión de tokens con clave simétrica compartida HS256). Este enfoque centraliza innecesariamente la custodia de credenciales y no escala hacia arquitecturas seguras distribuidas. Esta tarea reemplaza la autenticación local por una integración federada estándar OpenID Connect y OAuth 2.0 (Authorization Code Flow con PKCE) delegada en Hodor como Proveedor de Identidad (IdP) centralizado. La aplicación pasa a actuar como cliente y servidor de recursos que valida firmas asimétricas RS256 mediante el conjunto público de claves (JWKS) expuesto por Hodor, delegando por completo el almacenamiento de contraseñas y garantizando la compatibilidad con interacciones parciales impulsadas por HTMX.

## 2. Actores y Usuarios

- **Usuario final:** interactúa con la aplicación a través de un navegador web y es redirigido a Hodor para identificarse sin exponer sus credenciales a la aplicación local.
- **Cliente HTMX:** navegador ejecutando solicitudes AJAX asíncronas que requiere redirecciones de página completa ante expiración de sesión para evitar conflictos de origen cruzado o renderizado fragmentario incorrecto.
- **Proveedor de Identidad (Hodor):** servicio externo que autentica credenciales, emite códigos de autorización, gestiona sesiones de usuario y firma asimétricamente los tokens de seguridad con algoritmo RS256.
- **Agente SDD (`seb-verificador`):** evalúa de forma determinista el cumplimiento de los contratos de autenticación, validación de tokens y respuestas HTTP.

## 3. Historias de Usuario

- **HU-01:** Como usuario de la aplicación, quiero autenticarme a través de la interfaz centralizada de Hodor mediante un flujo seguro de autorización, para utilizar una única identidad sin registrar ni ingresar contraseñas en esta aplicación.
- **HU-02:** Como usuario que navega la interfaz HTMX, quiero que al expirar mi sesión se me redirija de forma completa a la pantalla de inicio de sesión de Hodor, para no experimentar bloqueos de red ni interfaces rotas.
- **HU-03:** Como administrador del sistema, quiero que la aplicación rechace con `403 Forbidden` a cualquier usuario autenticado que no pertenezca al sistema cliente autorizado, y con `401 Unauthorized` a cualquier token manipulado, con algoritmo no permitido o expirado.

## 4. Definiciones de Dominio (Opcional)

- **Proveedor de Identidad (IdP):** servicio centralizado (Hodor) responsable de autenticar usuarios y emitir credenciales criptográficas asimétricas.
- **Authorization Code Flow con PKCE:** flujo estándar OAuth 2.0 que intercambia un código temporal de autorización por tokens utilizando un verificador criptográfico de clave efímera para mitigar intercepciones en el canal.
- **JWKS (JSON Web Key Set):** conjunto de claves públicas criptográficas publicado por el IdP que permite verificar firmas RS256 sin compartir secretos privados.
- **Sistema Autorizado (`authorized_systems`):** lista declarativa de clientes autorizados contenida en los claims del token que determina si el usuario tiene permiso explícito para acceder a este sistema específico.
- **Redirección de Navegación Completa (HX-Redirect):** instrucción HTTP enviada en cabecera de respuesta para forzar al cliente HTMX a realizar una navegación total en el navegador hacia la URL de destino.
- **Falla Rápida (Fail-Fast):** principio de diseño según el cual el proceso de la aplicación aborta inmediatamente el arranque si sus dependencias críticas de inicialización (como el JWKS del IdP) no están disponibles.

## 5. Requisitos Funcionales (Criterios de Aceptación en EARS)

- **RF-01 (Event-Driven):** CUANDO un usuario no autenticado solicita acceder a una ruta protegida o accede a `/login` o `/auth/login`, EL SISTEMA DEBE iniciar el flujo OAuth2 con PKCE generando el desafío criptográfico (`code_challenge` derivado por SHA-256) y el parámetro de estado (`state`), almacenando el verificador y el estado en cookies seguras temporales de corta vigencia (`HttpOnly`, `SameSite=Lax`), y redirigiendo al usuario al endpoint de autorización del Proveedor de Identidad configurado en la variable de entorno `IDP_HODOR_URL`.
- **RF-02 (Event-Driven):** CUANDO una petición protegida proviene de un cliente HTMX (con cabecera `HX-Request: true`) y carece de sesión válida o presenta un token inválido, EL SISTEMA DEBE responder con la cabecera `HX-Redirect` hacia `/auth/login` y código de estado `401 Unauthorized`, forzando la navegación completa en el navegador.
- **RF-03 (Event-Driven):** CUANDO una petición protegida ordinaria (sin cabecera HTMX) carece de sesión válida o presenta un token inválido, EL SISTEMA DEBE responder con una redirección `303 See Other` hacia `/auth/login`.
- **RF-04 (Event-Driven):** CUANDO el Proveedor de Identidad redirige al usuario al endpoint de retorno (`/auth/callback`) con un código de autorización y un parámetro de estado, EL SISTEMA DEBE comprobar que el estado coincide con el valor temporal almacenado previamente, intercambiar el código y el verificador PKCE ante el IdP por el `access_token` (token JWT firmado asimétricamente con RS256 y portador de los claims `sub`, `roles` y `authorized_systems`), y validar criptográficamente dicho token.
- **RF-05 (Unwanted Behavior):** SI en la petición a `/auth/callback` el parámetro de estado no coincide con el almacenado, falta el parámetro de estado o no se encuentra el verificador PKCE, ENTONCES EL SISTEMA DEBE responder con código `400 Bad Request` y cancelar el flujo de autenticación sin contactar al IdP.
- **RF-06 (Ubiquitous):** EL SISTEMA DEBE validar la firma asimétrica de los tokens recibidos utilizando exclusivamente el algoritmo RS256 contrastado contra el conjunto de claves públicas (JWKS) expuesto por el Proveedor de Identidad en `${IDP_HODOR_URL}/.well-known/jwks.json`.
- **RF-07 (Unwanted Behavior):** SI un token presentado utiliza un algoritmo de firma distinto de RS256 (incluyendo `none` o algoritmos simétricos como HS256), tiene una firma no verificable contra el JWKS, ha expirado, o su emisor (`iss`) no coincide con el emisor configurado, ENTONCES EL SISTEMA DEBE rechazar el acceso y denegarlo con código `401 Unauthorized` (o redirección `303` para peticiones estándar sin sesión).
- **RF-08 (Unwanted Behavior):** SI un token contiene una firma y vigencia válidas pero el identificador del cliente configurado no se encuentra incluido en el claim `authorized_systems`, ENTONCES EL SISTEMA DEBE rechazar la solicitud con código `403 Forbidden`, impidiendo el acceso a los recursos protegidos.
- **RF-09 (Event-Driven):** CUANDO el intercambio de código y la validación del `access_token` son exitosos en `/auth/callback`, EL SISTEMA DEBE establecer una cookie de sesión HTTP segura (`token`, con atributos `HttpOnly`, `SameSite=Lax`, `Path=/`) con el valor del `access_token` y expiración alineada al claim `exp`, eliminar las cookies temporales del flujo OIDC/PKCE, y redirigir al usuario con código `302 Found` hacia el listado de notas (`/notas`).
- **RF-10 (Event-Driven):** CUANDO un usuario autenticado envía una petición de cierre de sesión a `/auth/logout` o `/logout`, EL SISTEMA DEBE invalidar la cookie de sesión local con expiración en el pasado y redirigir al usuario a `/login` (utilizando cabecera `HX-Redirect` si es petición HTMX o redirección HTTP si es estándar), realizando un logout de ámbito local.
- **RF-11 (Ubiquitous):** EL SISTEMA DEBE mantener en caché las claves públicas del JWKS del Proveedor de Identidad y renovarlas automáticamente ante expiración o presencia de nuevos identificadores de clave (`kid`), garantizando que la validación de peticiones ordinarias se resuelva localmente en memoria sin llamadas de red sincrónicas bloqueantes por cada petición entrante.
- **RF-12 (Unwanted Behavior):** SI al arrancar la aplicación el endpoint de claves públicas JWKS de Hodor (`${IDP_HODOR_URL}/.well-known/jwks.json`) no se encuentra accesible o falla su inicialización, ENTONCES EL SISTEMA DEBE abortar la ejecución inmediatamente con error fatal (`log.Fatal`), impidiendo que el servidor HTTP entre en servicio en un estado inconsistente.
- **RF-13 (Ubiquitous):** EL SISTEMA DEBE deshabilitar y retirar todo almacenamiento y verificación de credenciales de usuario locales (contraseñas y tablas de autenticación en la base de datos local), delegando la comprobación de credenciales exclusivamente en el Proveedor de Identidad.

## 6. Requisitos No Funcionales (Verificables)

- **RNF-01 (Seguridad Criptográfica):** Prohibición terminante de métodos de firma simétricos o algoritmos nulos. Todo token aceptado debe ser validado con RS256 y un tamaño de clave RSA igual o superior a 2048 bits.
- **RNF-02 (Rendimiento de Validación en Memoria):** La validación de tokens en rutas protegidas mediante claves cacheadas en memoria debe ejecutarse con una latencia inferior a 5 milisegundos en el percentil 95 dentro del entorno local de desarrollo.
- **RNF-03 (Configurabilidad y Variables de Entorno):** La URL base del Proveedor de Identidad debe ser configurable mediante la variable de entorno `IDP_HODOR_URL` (valor por defecto en desarrollo local: `http://localhost:8080`), junto con el identificador del cliente (`OAUTH_CLIENT_ID`) y secreto de cliente (`OAUTH_CLIENT_SECRET`).
- **RNF-04 (Testabilidad Determinista y Aislamiento):** Toda prueba automatizada ejecutada mediante `go test ./...` debe utilizar servidores HTTP simulados en memoria (`httptest.Server`) con pares de claves RSA efímeras y endpoints JWKS locales, sin realizar conexiones de red externas ni requerir una instancia real de Hodor en ejecución.
- **RNF-05 (Alineación con el Manifiesto y Gobernanza):** Toda nueva dependencia externa requerida (para flujos OAuth2 y cliente JWKS) debe quedar documentada y justificada. La aplicación debe compilar con `go build ./...`, pasar la suite `go test ./...` y registrar 0 hallazgos en `golangci-lint run`.

## 7. Casos Límite (Edge Cases)

- **CL-01 (Rotación de Claves en el IdP):** Si un token entrante posee un identificador de clave (`kid`) desconocido en la caché local, el sistema debe refrescar inmediatamente el conjunto JWKS desde el IdP antes de declarar inválido el token.
- **CL-02 (Sesión expirada durante interacción HTMX):** Un clic o envío de formulario parcial HTMX con token caducado o ausente debe emitir `HX-Redirect: /auth/login` y estado `401 Unauthorized`, evitando que la interfaz intente insertar el HTML del IdP en un contenedor interno del DOM.
- **CL-03 (Ataque de repetición o estado caducado en callback):** Intentar invocar `/auth/callback` con un `state` previamente consumido o generado hace más de 5 minutos debe ser rechazado con `400 Bad Request`.
- **CL-04 (Usuario válido en IdP pero no autorizado para el cliente):** Un usuario que se autentica exitosamente en Hodor pero no tiene asignado el sistema configurado en su claim `authorized_systems` recibe `403 Forbidden` (RF-08) sin acceso al listado de notas ni a operaciones de escritura.
- **CL-05 (Indisponibilidad inicial del IdP):** Si el endpoint de JWKS del IdP no responde en el arranque, la aplicación aborta con error fatal (`log.Fatal`) cumpliendo con el principio fail-fast (RF-12).

## 8. Fuera de Alcance (Out of Scope)

- Implementación, despliegue, configuración o modificación interna del servicio Hodor (alta de clientes OAuth, administración de usuarios, llaves privadas RSA).
- Flujos de Single Sign-Out distribuido mediante Back-Channel Logout o Front-Channel Logout de OIDC (el logout de esta iteración revoca la sesión local en la aplicación cliente).
- Modificación del esquema de notas para soportar multi-inquilino o propiedad de notas (`user_id` por nota) — las notas se mantienen globales según lo establecido en las especificaciones 001 y 002.
- Soporte para otros proveedores de identidad (Google, GitHub, Auth0) o protocolos alternativos (SAML, LDAP).
- Intercambio de tokens por Refresh Tokens para renovación silenciosa en segundo plano (se renueva iniciando el flujo cuando el token de acceso expira).

## 9. Criterios de Finalización (Definition of Done)

- Todos los requisitos (`RF-01` a `RF-13`, `RNF-01` a `RNF-05`) cuentan con verificación programática en `[PASA]` ejecutada mediante `.seb/bin/seb verify autenticacion_hodor`.
- Eliminación completa de la verificación de contraseñas locales por hash (bcrypt) y su persistencia.
- Ninguna petición HTMX rota o bloqueada por CORS en flujos de autenticación o caducidad de sesión.
- Cero violaciones a `docs/constitution.md`.

## 10. Dudas Abiertas y Puntos por Clarificar

Ninguna duda pendiente.
