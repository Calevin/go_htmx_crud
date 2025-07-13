package handlers

import (
	"github.com/Calevin/go_htmx_crud/internal/db"
	"html/template"
	"log"
	"net/http"
)

// Render rederiza dentro de layout el template contentFile con los datos pasados como parametros
func Render(tpl *template.Template, w http.ResponseWriter, contentFile string, data any) {
	err := tpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"contentFile": contentFile,
		"data":        data,
	})
	if err != nil {
		log.Printf("Error renderizando: %v", err)
		http.Error(w, "Error del servidor", 500)
	}
}

// ListNotesHandler muestra la lista de notas del usuario
func ListNotesHandler(w http.ResponseWriter, r *http.Request, tpl *template.Template, queries *db.Queries) {
	// Lógica para obtener las notas
	notesFromDB, err := queries.ListNotes(r.Context())
	if err != nil {
		http.Error(w, "Error al obtener notas", http.StatusInternalServerError)
		return
	}
	// Estructura para pasar datos enriquecidos al template
	type NoteWithTags struct {
		db.Note
		Tags []db.Tag
	}
	var notesForTemplate []NoteWithTags
	// TODO reemplazar con join
	for _, note := range notesFromDB {
		tags, _ := queries.GetTagsForNote(r.Context(), note.ID)
		notesForTemplate = append(notesForTemplate, NoteWithTags{Note: note, Tags: tags})
	}
	// Se renderiza la página de notas, pasando los datos.
	data := make(map[string]any)
	data["Notes"] = notesForTemplate
	Render(tpl, w, "notas.html", data)
}
