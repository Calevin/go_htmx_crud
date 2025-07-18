-- sql/queries/query.sql

-- name: CreateTag :one
INSERT INTO tags (nombre, color)
VALUES (?, ?)
RETURNING *;

-- name: GetTag :one
SELECT * FROM tags
WHERE id = ? LIMIT 1;

-- name: ListTags :many
SELECT * FROM tags
ORDER BY nombre;

-- name: CreateNote :one
INSERT INTO notes (nombre, contenido)
VALUES (?, ?)
RETURNING *;

-- name: ListNotes :many
SELECT * FROM notes
ORDER BY id DESC;

-- name: GetNote :one
SELECT * FROM notes
WHERE id = ? LIMIT 1;

-- name: UpdateNote :exec
UPDATE notes
SET nombre = ?, contenido = ?
WHERE id = ?;

-- name: DeleteNote :exec
DELETE FROM notes
WHERE id = ?;

-- name: UnlinkTagsFromNote :exec
DELETE FROM note_tags
WHERE note_id = ?;

-- name: LinkTagToNote :exec
INSERT INTO note_tags (note_id, tag_id)
VALUES (?, ?);

-- name: GetTagsForNote :many
SELECT t.* FROM tags t
JOIN note_tags nt ON t.id = nt.tag_id
WHERE nt.note_id = ?;

-- name: CreateUser :one
INSERT INTO users (username, password_hash)
VALUES (?, ?)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = ? LIMIT 1;

-- name: ListNotesWithTags :many
SELECT
    n.id AS note_id,
    n.nombre AS note_nombre,
    n.contenido AS note_contenido,
    t.id AS tag_id,
    t.nombre AS tag_nombre,
    t.color AS tag_color
FROM
    notes n
        LEFT JOIN
    note_tags nt ON n.id = nt.note_id
        LEFT JOIN
    tags t ON nt.tag_id = t.id
ORDER BY
    n.id DESC;