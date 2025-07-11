package database

import (
	"database/sql"
	"log"
	"os"

	// El guion bajo _ importa el driver para que se registre en database/sql
	_ "github.com/mattn/go-sqlite3"
)

// InitDB inicializa la conexión a la base de datos y ejecuta el esquema.
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

	// Ejecuta el schema.sql para crear las tablas
	if err = executeSchema(db); err != nil {
		log.Fatalf("Error aplicando el esquema: %v", err)
	}

	return db
}

func executeSchema(db *sql.DB) error {
	// Lee el archivo de esquema
	schema, err := os.ReadFile("sql/schema/schema.sql")
	if err != nil {
		return err
	}

	// Ejecuta el SQL del archivo
	_, err = db.Exec(string(schema))
	return err
}
