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