-- sql/schema/schema.sql

CREATE TABLE IF NOT EXISTS tags (
    "id"    INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "nombre" TEXT NOT NULL UNIQUE,
    "color"  TEXT
);

CREATE TABLE IF NOT EXISTS notes (
    "id"        INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "nombre"    TEXT NOT NULL,
    "contenido" TEXT
);

CREATE TABLE IF NOT EXISTS note_tags (
    "note_id" INTEGER NOT NULL,
    "tag_id"  INTEGER NOT NULL,
    PRIMARY KEY(note_id, tag_id),
    FOREIGN KEY(note_id) REFERENCES notes(id) ON DELETE CASCADE,
    FOREIGN KEY(tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS users (
    "id"            INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "username"      TEXT NOT NULL UNIQUE,
    "password_hash" TEXT NOT NULL
);