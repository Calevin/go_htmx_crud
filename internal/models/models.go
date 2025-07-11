package models

// Tag representa una etiqueta en el sistema.
type Tag struct {
	ID     int64  `json:"id"`
	Nombre string `json:"nombre"`
	Color  string `json:"color"`
}

// Note representa una nota en el sistema.
type Note struct {
	ID        int64  `json:"id"`
	Nombre    string `json:"nombre"`
	Contenido string `json:"contenido"`
	// Una nota puede tener múltiples tags.
	Tags []Tag `json:"tags"`
}
