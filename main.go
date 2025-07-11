package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	// Se importa los paquetes locales.
	"github.com/Calevin/go_htmx_crud/database"
)

func main() {
	// Se iniciala la base de datos. Esto creará el archivo 'crud.db' en la raíz.
	db := database.InitDB("./crud.db")
	defer db.Close()

	// Se insertan los datos de prueba.
	insertMockData(db)

	// Instancia del router Chi
	r := chi.NewRouter()

	// Middleware que loguea las peticiones en la consola
	r.Use(middleware.Logger)

	// Ruta GET
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("hola mundo, la base de datos está lista!"))
		if err != nil {
			log.Printf("Error escribiendo la respuesta: %v", err)
		}
	})

	// Se inicia el servidor en el puerto 3000
	log.Println("Servidor iniciado en http://localhost:3000")
	err := http.ListenAndServe(":3000", r)
	if err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

// insertMockData inserta un tag y una nota de ejemplo, y los vincula.
func insertMockData(db *sql.DB) {
	// --- 1. Insertar Tag ---
	var tagID int64
	// Primero: se verificamos si el tag ya existe
	err := db.QueryRow("SELECT id FROM tags WHERE nombre = ?", "Importante").Scan(&tagID)
	if err == sql.ErrNoRows {
		// El tag no existe, se inserta
		res, err := db.Exec("INSERT INTO tags (nombre, color) VALUES (?, ?)", "Importante", "#FF5733")
		if err != nil {
			log.Fatalf("Error insertando el tag de prueba: %v", err)
		}
		tagID, _ = res.LastInsertId()
		log.Printf("Tag de prueba 'Importante' insertado con ID: %d\n", tagID)
	} else {
		log.Println("El tag de prueba 'Importante' ya existía.")
	}

	// --- 2. Insertar Nota ---
	var noteID int64
	err = db.QueryRow("SELECT id FROM notes WHERE nombre = ?", "Mi primera Nota").Scan(&noteID)
	if err == sql.ErrNoRows {
		res, err := db.Exec("INSERT INTO notes (nombre, contenido) VALUES (?, ?)", "Mi primera Nota", "Este es el contenido de la nota inicial.")
		if err != nil {
			log.Fatalf("Error insertando la nota de prueba: %v", err)
		}
		noteID, _ = res.LastInsertId()
		log.Printf("Nota de prueba 'Mi primera Nota' insertada con ID: %d\n", noteID)

		// --- 3. Vincular Nota y Tag ---
		_, err = db.Exec("INSERT OR IGNORE INTO note_tags (note_id, tag_id) VALUES (?, ?)", noteID, tagID)
		if err != nil {
			log.Fatalf("Error vinculando nota y tag: %v", err)
		}
		log.Printf("Nota %d vinculada con Tag %d\n", noteID, tagID)
	} else {
		log.Println("La nota de prueba 'Mi primera Nota' ya existía.")
	}
}
