package database

import (
	"database/sql"
	"log"

	// El guion bajo _ importa el driver para que se registre en database/sql
	_ "github.com/mattn/go-sqlite3"
)

// InitDB inicializa la conexión a la base de datos SQLite y crea las tablas si no existen.
func InitDB(filepath string) *sql.DB {
	// Abre la conexión con la base de datos. Si el archivo no existe, lo crea.
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatalf("Error abriendo la base de datos: %v", err)
	}

	// Ping confirma que la conexión es válida.
	if err = db.Ping(); err != nil {
		log.Fatalf("Error conectando a la base de datos: %v", err)
	}

	log.Println("Conexión a la base de datos SQLite exitosa.")

	createTables(db)

	return db
}

// createTables ejecuta el SQL para crear las tablas.
func createTables(db *sql.DB) {
	// SQL para crear la tabla de tags.
	// NOTA: "nombre" es UNIQUE para evitar tags duplicados.
	createTagsTableSQL := `
	CREATE TABLE IF NOT EXISTS tags (
		"id"    INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"nombre" TEXT NOT NULL UNIQUE,
		"color"  TEXT
	);`

	// SQL para crear la tabla de notas.
	createNotesTableSQL := `
	CREATE TABLE IF NOT EXISTS notes (
		"id"        INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"nombre"    TEXT NOT NULL,
		"contenido" TEXT
	);`

	// Tabla muchos a muchos entre notas y tags.
	createNoteTagsTableSQL := `
	CREATE TABLE IF NOT EXISTS note_tags (
		"note_id" INTEGER NOT NULL,
		"tag_id"  INTEGER NOT NULL,
		PRIMARY KEY(note_id, tag_id),
		FOREIGN KEY(note_id) REFERENCES notes(id) ON DELETE CASCADE,
		FOREIGN KEY(tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);`

	// Se ejecutan las sentencias SQL.
	for _, query := range []string{createTagsTableSQL, createNotesTableSQL, createNoteTagsTableSQL} {
		statement, err := db.Prepare(query)
		if err != nil {
			log.Fatalf("Error preparando la query de creación de tabla: %v", err)
		}
		_, err = statement.Exec()
		if err != nil {
			log.Fatalf("Error ejecutando la query de creación de tabla: %v", err)
		}
	}

	log.Println("Tablas creadas (si no existían) exitosamente.")
}
