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
	notesWithTagsFromDB, err := queries.ListNotesWithTags(r.Context())
	if err != nil {
		http.Error(w, "Error al obtener notas", http.StatusInternalServerError)
		return
	}
	// Estructura para pasar datos enriquecidos al template
	type NoteWithTags struct {
		ID        int64
		Nombre    string
		Contenido string
		Tags      []db.Tag
	}

	// Mapa para no duplicar notas y agrupar sus tags.
	notesMap := make(map[int64]*NoteWithTags)
	// Slice para mantener el orden original.
	var orderedNotes []*NoteWithTags

	for _, noteAndTag := range notesWithTagsFromDB {
		// Si no existe se agrega al mapa
		if _, ok := notesMap[noteAndTag.NoteID]; !ok {
			note := &NoteWithTags{
				ID:        noteAndTag.NoteID,
				Nombre:    noteAndTag.NoteNombre,
				Contenido: noteAndTag.NoteContenido.String,
				Tags:      []db.Tag{}, // Se inicializa el slice de tags vacío.
			}
			// se agrega al mapa y al lista ordenada
			notesMap[noteAndTag.NoteID] = note
			orderedNotes = append(orderedNotes, note)
		}

		// Si este row tiene tag se agrega a la nota correspondiente
		if noteAndTag.TagID.Valid {
			tag := db.Tag{
				ID:     noteAndTag.TagID.Int64,
				Nombre: noteAndTag.TagNombre.String,
				Color:  noteAndTag.TagColor, // `sqlc` ya maneja el NullString aca
			}
			notesMap[noteAndTag.NoteID].Tags = append(notesMap[noteAndTag.NoteID].Tags, tag)
		}
	}
	// Se renderiza la página de notas, pasando los datos.
	data := make(map[string]any)
	data["Notes"] = orderedNotes
	Render(tpl, w, "notas.html", data)
}
