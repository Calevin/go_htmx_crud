# Spec 001 — Suite de Tests para el Código Existente

- **Tarea ID:** `001`
- **Identificador:** `tests_codigo_existente`
- **Origen:** Prosa (solicitud del usuario: "agregar test al proyecto para el código existente")
- **Estado:** `APROBADA`
- **Dependencias:** `002_codigos_http_canonicos`

## 1. Contexto y Objetivo (El POR QUÉ)

El proyecto no tiene ni una sola prueba automatizada: el comando oficial de pruebas termina vacío y no valida ningún comportamiento, de modo que cualquier refactor o extensión futura del stack de referencia se hace sin red de seguridad. Esta tarea establece la suite base de pruebas del código existente (credenciales, control de acceso, manejo HTTP de notas y persistencia) para que `seb-harness` disponga de una verificación determinista antes de implementar nuevas funcionalidades, y para que el repositorio cumpla su rol de ejemplo copiable del stack Go + HTMX + sqlc mostrando también cómo se testea.

## 2. Actores y Usuarios

- **Desarrollador humano:** ejecuta la suite localmente para obtener una señal confiable de regresión antes de modificar código.
- **Agente SDD (`seb-verificador`):** consume `go test ./...` como gate determinista obligatorio de toda tarea futura.
- **Lector externo del repositorio:** estudia las pruebas de ejemplo del patrón de testeabilidad del stack.

## 3. Historias de Usuario

- **HU-01:** Como desarrollador del proyecto, quiero ejecutar una única orden y saber si el comportamiento existente se mantió intacto, para cambiar código sin romper funcionalidad de forma silenciosa.
- **HU-02:** Como lector del repositorio, quiero encontrar pruebas de ejemplo de handlers, autenticación y persistencia, para copiar el patrón en mis propios proyectos del stack.

## 4. Definiciones de Dominio (Opcional)

- **Suite de pruebas:** conjunto de pruebas automatizadas ejecutables mediante el comando oficial de pruebas del proyecto.
- **Paquete lógico:** uno de los cuatro módulos con lógica propia sujetos a cobertura: credenciales y sesiones (`internal/auth`), control de acceso (`internal/middleware`), manejo HTTP (`internal/handlers`) y persistencia (`database`). Quedan excluidos el código generado por sqlc y el paquete raíz de composición (`main`).
- **Código generado:** artefacto producido por el generador de consultas a partir del SQL declarativo; no es lógica propia del proyecto y queda excluido de la obligación de cobertura.
- **Petición de prueba:** solicitud HTTP construida y ejecutada en proceso, sin abrir puertos de red reales.
- **Base de datos temporal:** instancia de persistencia creada y destruida dentro de la ejecución de la prueba, ajena al archivo de datos del proyecto.

## 5. Requisitos Funcionales (Criterios de Aceptación en EARS)

- **RF-01 (Ubiquitous):** EL SISTEMA DEBE contener pruebas automatizadas ejecutables mediante el comando oficial `go test ./...` para los cuatro paquetes lógicos definidos en la sección 4, de modo que ninguno de ellos quede sin al menos una prueba; el código generado y el paquete raíz de composición quedan excluidos de esta obligación.
- **RF-02 (Event-Driven):** CUANDO se ejecute el comando oficial de pruebas sobre un clon limpio del repositorio, EL SISTEMA DEBE finalizar con código de salida 0 sin intervención manual y sin requisitos externos (servidor en ejecución, archivo de datos del proyecto presente, credenciales u variables de entorno del operador).
- **RF-03 (Unwanted Behavior):** SI una prueba requiere persistencia de datos, ENTONCES EL SISTEMA DEBE utilizar una base de datos temporal aislada por ejecución, garantizando que ninguna prueba lea, modifique ni dependa del archivo de datos real del proyecto ni de datos preexistentes.
- **RF-04 (Unwanted Behavior):** SI una prueba requiere contexto de ejecución (configuración de entorno, usuarios iniciales, secretos), ENTONCES EL SISTEMA DEBE proporcionarlo dentro de la propia prueba, sin arrancar el proceso completo de la aplicación ni depender del archivo de configuración del operador.
- **RF-05 (Event-Driven):** CUANDO se ejecute una petición de prueba hacia una ruta protegida sin token, con token inválido, con token firmado con un secreto distinto al configurado o con token expirado, EL SISTEMA DEBE verificar que la respuesta tiene código `303 See Other` con redirección hacia el inicio de sesión, y que no se entrega ningún contenido protegido.
- **RF-06 (Event-Driven):** CUANDO se ejecute una petición de prueba autenticada válida sobre operaciones de notas, EL SISTEMA DEBE verificar los códigos de estado exactos: listado de notas, formulario de creación y formulario de edición → `200 OK`; creación y edición (POST) → `302 Found` con redirección al listado; eliminación de nota existente → `200 OK`; operación sobre un identificador de nota inexistente (edición, actualización o eliminación) → `404 Not Found`. Las notas son globales en el modelo de datos actual.
- **RF-07 (Event-Driven):** CUANDO se ejecute una operación de prueba de inicio o cierre de sesión con credenciales válidas, EL SISTEMA DEBE verificar que la respuesta es `200 OK` con cabecera `HX-Redirect`, que el token emitido en el inicio de sesión permite acceder a una ruta protegida en la petición siguiente, y que la contraseña no aparece en claro ni en la respuesta ni en el valor almacenado.
- **RF-08 (Unwanted Behavior):** SI un identificador recibido en una petición no es numérico o el cuerpo de la petición no puede interpretarse, ENTONCES EL SISTEMA DEBE verificar que la respuesta tiene código `400 Bad Request` y que ninguna escritura llegó a la persistencia.
- **RF-09 (Ubiquitous):** EL SISTEMA DEBE ser determinista bajo el modo de ejecución por defecto del comando oficial de pruebas: dos ejecuciones consecutivas de la suite sobre el mismo estado del repositorio deben producir el mismo resultado, sin dependencia del orden de ejecución, sin esperas temporales arbitrarias, sin estado compartido entre pruebas y sin exigir ejecución en paralelo.
- **RF-10 (Ubiquitous):** EL SISTEMA DEBE construirse sin añadir dependencias externas al proyecto, usando exclusivamente la biblioteca estándar de Go y las dependencias ya declaradas en el manifiesto del módulo.

## 6. Requisitos No Funcionales (Verificables)

- **RNF-01:** Cada uno de los cuatro paquetes lógicos definidos en la sección 4 debe alcanzar una cobertura de líneas igual o superior al 70%, medida individualmente (por paquete) con la generación de perfiles de cobertura del comando oficial de pruebas; el código generado queda excluido del cálculo.
- **RNF-02:** La suite completa debe finalizar en 60 segundos o menos en el equipo de desarrollo estándar.
- **RNF-03:** La suite debe pasar con el linter del proyecto en modo de cero hallazgos, incluyendo las reglas de análisis estático ya configuradas.

## 7. Casos Límite (Edge Cases)

- **CL-01 (Notas globales sin aislamiento multiusuario):** En el esquema actual las notas carecen de asociación con un usuario específico (sin columna `user_id`); toda sesión autenticada válida accede al conjunto global de notas.
- **CL-02 (Colección vacía):** Un usuario autenticado sin notas recibe `200 OK` con lista vacía, sin error interno.
- **CL-03 (Identificador inexistente):** Operar sobre una nota que no existe produce `404 Not Found`, sin pánico ni error interno (500).
- **CL-04 (Credenciales vacías en login):** Usuario o contraseña en blanco en el formulario de inicio de sesión producen `401 Unauthorized` sin emisión de token ni acceso a rutas protegidas.
- **CL-05 (Discrepancia con el comportamiento actual):** SI durante la implementación se detecta que la aplicación ya emite un código de estado distinto del definido en RF-06 o RF-07 para alguna operación, ENTONCES el agente DEBE detener la implementación, registrar la discrepancia en `MEMORY.md` y solicitar retorno a `seb-clarificador`; está prohibido cambiar el código de producción ni relajar la prueba para forzar el pase.

## 8. Fuera de Alcance (Out of Scope)

- Añadir frameworks o librerías de pruebas externas (p. ej. librerías de aserciones o mocks de terceros): requiere aprobación explícita en una spec futura.
- Pruebas de extremo a extremo con navegador (playwright, cypress) y pruebas de rendimiento/carga.
- Pruebas de los artefactos generados por el generador de consultas.
- Pruebas del paquete raíz de composición (`main`).
- Alterar el comportamiento HTTP observable de la aplicación existente (los ajustes mínimos de composición necesarios para la inyección de dependencias deben preservar el comportamiento observable, salvo el procedimiento de la CL-05).
- Verificación de contenido específico de templates HTML: basta con código de estado + efecto observable en persistencia (decisión explícita del usuario).
- Configurar integración continua en servicios externos (el gate local `.seb/bin/seb verify` es suficiente para esta iteración).
- Mejoras funcionales de la app de notas.
- Registro de usuarios, pertenencia de notas a usuarios (multi-tenant), y validación de campos vacíos en notas (postergados formalmente a specs futuras según spec 002).

## 9. Criterios de Finalización (Definition of Done)

- Todos los requisitos (`RF-01` a `RF-10`, `RNF-01` a `RNF-03`) cuentan con verificación programática en `[PASA]` ejecutada mediante `.seb/bin/seb verify tests_codigo_existente`.
- Cero violaciones a `docs/constitution.md`.
- La suite cubre los cuatro paquetes lógicos de la sección 4 y ninguna prueba depende de la red real, del archivo de datos del proyecto ni de variables de entorno del operador.

## 10. Dudas Abiertas y Puntos por Clarificar

Ninguna duda pendiente.

## Contrato de Desbloqueo (Satisfecho por Spec 002)

La dependencia con la tarea `002_codigos_http_canonicos` ha sido satisfecha y verificada (`verified: true`, commit `6e21d20` en `.seb/seb.lock`). Las siguientes adaptaciones canónicas quedaron incorporadas en esta especificación:

1. **RF-05:** la respuesta ante token ausente, inválido, firmado con otro secreto o expirado pasa a `303 See Other` hacia el inicio de sesión.
2. **RF-06:** códigos exactos — listar y formularios → `200 OK`; crear y editar → `302 Found`; eliminar existente → `200 OK`; identificador de nota inexistente → `404 Not Found`. Se eliminó la cláusula de aislamiento por usuario: notas globales por diseño.
3. **RF-07:** inicio y cierre de sesión exitosos → `200 OK` con redirección total (`HX-Redirect`). Se eliminó toda mención a "registro" (fuera de alcance).
4. **RF-08:** acotado a identificadores no numéricos o cuerpos de petición ilegibles → `400 Bad Request` sin escritura.
5. **CL-01:** notas globales sin aislamiento multiusuario documentadas explícitamente.
6. **CL-04:** credenciales vacías en login → `401 Unauthorized` sin token.
7. **CL-05:** conservado como invariante de seguridad ante discrepancias no previstas.
