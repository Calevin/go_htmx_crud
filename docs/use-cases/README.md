# Catalogo de Casos de Uso del Sistema

Este directorio contiene la documentacion funcional de caja negra (Black-Box) del sistema, generada y mantenida de acuerdo con las especificaciones SDD verificadas deterministamente por `seb-harness`.

## Indice de Casos de Uso

| ID                                                      | Caso de Uso                                                    | Disparador Principal                              | Spec Originaria                                          | Estado  |
| :------------------------------------------------------ | :------------------------------------------------------------- | :------------------------------------------------ | :------------------------------------------------------- | :------ |
| [`UC-notes_list`](UC-notes_list.md)                     | Consulta y Visualizacion de Notas                              | `GET /notas`                                      | `specs/002_codigos_http_canonicos_spec.md`               | Vigente |
| [`UC-notes_create`](UC-notes_create.md)                 | Creacion de Nota y Renderizado de Formulario                   | `GET /crear_nota`, `POST /crear_nota`             | `specs/002_codigos_http_canonicos_spec.md`               | Vigente |
| [`UC-notes_edit`](UC-notes_edit.md)                     | Modificacion y Formulario de Edicion de Nota                   | `GET /editar_nota/{id}`, `POST /editar_nota/{id}` | `specs/002_codigos_http_canonicos_spec.md`               | Vigente |
| [`UC-notes_delete`](UC-notes_delete.md)                 | Eliminacion de Nota                                            | `DELETE /borrar_nota/{id}`                        | `specs/002_codigos_http_canonicos_spec.md`               | Vigente |
| [`UC-auth_landing`](UC-auth_landing.md)                 | Visualizacion de Pantalla de Bienvenida y Enrutamiento Inicial | `GET /`, `GET /login`                             | `specs/004_landing_login_logout_spec.md`                 | Vigente |
| [`UC-auth_session`](UC-auth_session.md)                 | Control de Sesion y Acceso a Rutas Protegidas                  | Rutas protegidas, `POST /logout`                  | `specs/004_landing_login_logout_spec.md` (rev. 002, 003) | Vigente |
| [`UC-auth_federated_login`](UC-auth_federated_login.md) | Inicio de Sesion Federado con Hodor (OIDC / PKCE)              | `GET /auth/login`, `GET /auth/callback`           | `specs/003_autenticacion_hodor_spec.md` (rev. 004)       | Vigente |

## Convenciones de Documentacion

1. **Enfoque Black-Box:** Los casos de uso describen la frontera externa de interaccion del sistema (HTTP / HTMX) sin acoplarse a detalles internos de implementacion.
2. **Nomenclatura Bilingue:**
   - La sintaxis tecnica, identificadores de DTOs, endpoints y participantes en diagramas Mermaid estan en ingles.
   - La descripcion funcional, explicaciones y tablas estan redactadas en espanol formal.
3. **Restriccion Visual:** Prohibicion estricta de emojis en artefactos de especificacion y diagramas.
4. **Trazabilidad:** Cada caso de uso referencia explicitamente la spec y los requisitos funcionales (`RF-XX`) de los que deriva.
