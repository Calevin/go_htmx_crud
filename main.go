package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	// Importa el paquete de base de datos
	"github.com/Calevin/go_htmx_crud/database"
	// Importa el paquete generado por sqlc
	"github.com/Calevin/go_htmx_crud/internal/db"
)

func main() {
	// Context para las operaciones de base de datos.
	ctx := context.Background()

	// Se iniciala la base de datos. Esto creará el archivo 'crud.db' en la raíz.
	conn := database.InitDB("./crud.db")
	defer conn.Close()

	// Crea una instancia de `Queries` generada por sqlc.
	queries := db.New(conn)

	// Se insertan los datos de prueba.
	insertMockData(ctx, queries)

	// Instancia del router Chi
	r := chi.NewRouter()

	// Middleware que loguea las peticiones en la consola
	r.Use(middleware.Logger)

	// Ruta GET
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		// Prueba usando funcion creada con sqlc
		notes, err := queries.ListNotes(r.Context())
		if err != nil {
			http.Error(w, "Error al listar notas", http.StatusInternalServerError)
			return
		}

		response := "No hay notas."
		if len(notes) > 0 {
			response = "Primera nota: " + notes[0].Nombre
		}
		_, _ = w.Write([]byte(response))
	})

	// Se inicia el servidor en el puerto 3000
	log.Println("Servidor iniciado en http://localhost:3000")
	err := http.ListenAndServe(":3000", r)
	if err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

// insertMockData inserta un tag y una nota de ejemplo, y los vincula.
func insertMockData(ctx context.Context, queries *db.Queries) {
	log.Println("Insertando datos de prueba si es necesario...")

	// --- 1. Insertar Tag ---
	tag, err := queries.CreateTag(ctx, db.CreateTagParams{
		Nombre: "Importante",
		Color:  sql.NullString{String: "#FF5733", Valid: true},
	})
	if err != nil {
		// Es probable que el tag ya exista (violación de UNIQUE), así que lo buscamos.
		tag, _ = queries.GetTag(ctx, 1) // Asumimos ID 1 para el ejemplo
	}

	// --- 2. Insertar Nota ---
	var contenido sql.NullString
	contenido.Scan("Este es el contenido de la nota.")

	note, err := queries.CreateNote(ctx, db.CreateNoteParams{
		Nombre:    "Mi primera Nota con sqlc",
		Contenido: contenido,
	})
	if err != nil {
		// La nota probablemente ya existe.
		notes, _ := queries.ListNotes(ctx)
		if len(notes) > 0 {
			note = notes[0]
		}
	}

	// --- 3. Vincular Nota y Tag ---
	if tag.ID != 0 && note.ID != 0 {
		err = queries.LinkTagToNote(ctx, db.LinkTagToNoteParams{
			NoteID: note.ID,
			TagID:  tag.ID,
		})
		// Ignoramos el error, ya que el vínculo podría existir.
		if err == nil {
			log.Printf("Vínculo creado entre Nota ID %d y Tag ID %d\n", note.ID, tag.ID)
		}
	}
}
