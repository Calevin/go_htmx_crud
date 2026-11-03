# Spec 002 — Estandarización de Códigos HTTP Canónicos

- **Tarea ID:** `002`
- **Identificador:** `codigos_http_canonicos`
- **Origen:** Prosa (deriva del bloqueo de planificación de la spec 001; tabla canónica confirmada por el usuario el 2026-11-03)
- **Estado:** `APROBADA`

## 1. Contexto y Objetivo (El POR QUÉ)

La aplicación emite códigos de estado HTTP inconsistentes en varios flujos: una nota inexistente produce un error interno (500) o una confirmación engañosa (200 sin efecto), y los flujos de éxito mezclan redirect y redirección total sin reglas claras. Esto impide que la spec 001 (suite de tests) pueda fijar expectativas verificables y degrada la claridad del contrato HTTP. Esta tarea define y hace cumplir el conjunto canónico de códigos de estado, elegido para ser correcto semánticamente y **compatibles con el cliente HTMX** (sin dejar al cliente sin acción observable), de modo que la spec 001 pueda retomarse y verificarse contra un comportamiento estable.

## 2. Actores y Usuarios

- **Cliente HTMX / navegador del usuario:** consume las respuestas HTTP y necesita que cada código dispare la acción correcta (swap, navegación o muestra de error).
- **Desarrollador y agente SDD:** obtienen un contrato de códigos único, estable y testeable como base de la suite de pruebas (spec 001 dependiente).

## 3. Historias de Usuario

- **HU-01:** Como usuario de la aplicación, quiero que pedir una nota que no exista me devuelva un "no encontrado" claro, para no recibir un error interno ni creer que la operación tuvo éxito.
- **HU-02:** Como desarrollador, quiero un conjunto fijo de códigos de estado compatible con HTMX, para poder escribir pruebas deterministas contra el comportamiento real.

## 4. Definiciones de Dominio (Opcional)

- **Códigos canónicos:** conjunto cerrado de códigos de estado permitidos para los escenarios de esta spec: `200`, `302`, `303`, `400`, `401`, `404`.
- **Nota existente:** nota cuyo identificador corresponde a un registro presente en la persistencia al momento de la petición.
- **Redirección total:** instrucción de respuesta que obliga al cliente HTMX a una navegación completa de página (cabecera `HX-Redirect`).
- **Ruta protegida:** toda ruta que exige sesión iniciada para entregar su contenido.

## 5. Requisitos Funcionales (Criterios de Aceptación en EARS)

- **RF-01 (Ubiquitous):** EL SISTEMA DEBE responder `200 OK` a las peticiones de lectura exitosas: listado de notas, formulario de inicio de sesión, formulario de creación de nota y formulario de edición de una nota existente.
- **RF-02 (Event-Driven):** CUANDO un inicio o cierre de sesión se procese correctamente, EL SISTEMA DEBE responder `200 OK` junto con la redirección total hacia la vista correspondiente.
- **RF-03 (Event-Driven):** CUANDO una petición de creación o actualización de una nota se procese correctamente, EL SISTEMA DEBE responder `302 Found` con redirección hacia el listado de notas (patrón Post/Redirect/Get).
- **RF-04 (Event-Driven):** CUANDO se reciba la eliminación de una nota existente, EL SISTEMA DEBE responder `200 OK` habiendo retirado la nota de la persistencia.
- **RF-05 (Unwanted Behavior):** SI se solicita el formulario de edición, la actualización o la eliminación de una nota cuyo identificador no corresponde a ninguna nota existente, ENTONCES EL SISTEMA DEBE responder `404 Not Found`, sin error interno (500) ni confirmación engañosa de éxito.
- **RF-06 (Unwanted Behavior):** SI una petición hacia una ruta protegida carece de token, tiene un token inválido, firmado con un secreto distinto al configurado o expirado, ENTONCES EL SISTEMA DEBE responder `303 See Other` hacia el inicio de sesión, sin entregar ningún contenido protegido.
- **RF-07 (Unwanted Behavior):** SI el inicio de sesión recibe credenciales vacías o que no corresponden a ningún usuario válido, ENTONCES EL SISTEMA DEBE responder `401 Unauthorized` sin emitir token de sesión alguno.
- **RF-08 (Unwanted Behavior):** SI un identificador recibido en la petición no es numérico o el cuerpo de la petición no puede interpretarse, ENTONCES EL SISTEMA DEBE responder `400 Bad Request` sin realizar ninguna escritura en la persistencia.

## 6. Requisitos No Funcionales (Verificables)

- **RNF-01:** Cada escenario cubierto por `RF-01` a `RF-08` debe emitir exactamente el código de estado definido en su requisito, verificado mediante pruebas automatizadas; ningún escenario puede emitir un código fuera del conjunto canónico salvo fallo interno real de persistencia.
- **RNF-02:** Los comandos oficiales de verificación (`go build ./...`, `go test ./...`, `golangci-lint run`) deben finalizar con código de salida 0 tras la implementación.

## 7. Casos Límite (Edge Cases)

- **CL-01 (Doble eliminación):** Eliminar la misma nota dos veces seguidas produce `200 OK` en la primera petición y `404 Not Found` en la segunda.
- **CL-02 (Identificador no numérico vs. inexistente):** Un identificador no numérico produce `400 Bad Request` (RF-08); un identificador numérico que no corresponde a ninguna nota produce `404 Not Found` (RF-05).
- **CL-03 (Token expirado):** Un token caducado produce `303 See Other` hacia el inicio de sesión, con el mismo resultado observable que un token inválido (RF-06).
- **CL-04 (Credenciales vacías en login):** Usuario o contraseña en blanco producen `401 Unauthorized` sin token, el mismo resultado observable que credenciales incorrectas (RF-07).

## 8. Fuera de Alcance (Out of Scope)

- Endpoint de registro de usuarios: funcionalidad nueva sin implementar, requiere spec propia.
- Pertenencia de notas a usuarios (multi-tenant / cambio de esquema): decisión tomada — las notas se documentan como globales por diseño en la spec 001 al retomarla; un modelo con dueño requiere spec propia futura.
- Validación de campos vacíos en formularios (decisión Opción B): el alta o edición de notas con nombre vacío conserva el comportamiento actual (`302`); exigir `400` requeriría lógica de validación nueva — postergado a spec futura.
- Identificador de etiqueta numérico inexistente al crear o editar una nota (decisión Opción D): fuera de alcance; se cubre en una spec futura de integridad de datos.
- Alterar el contenido de templates, la lógica de negocio o la estructura de datos existente.
- Añadir nuevos endpoints, flujos o códigos fuera del conjunto canónico definido en la sección 4.
- Cambiar cualquier código ya fijado en `RF-01` a `RF-08`: toda modificación futura de este contrato requiere una spec nueva.

## 9. Criterios de Finalización (Definition of Done)

- Todos los requisitos (`RF-01` a `RF-08`, `RNF-01`, `RNF-02`) cuentan con verificación programática en `[PASA]` ejecutada mediante `.seb/bin/seb verify codigos_http_canonicos`.
- Cero violaciones a `docs/constitution.md`.
- La spec 001 puede retomarse: su bloqueo por dependencia queda satisfecho al verificarse esta tarea.

## 10. Dudas Abiertas y Puntos por Clarificar

Ninguna duda pendiente.
