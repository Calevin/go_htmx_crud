package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Instancia del router Chi
	r := chi.NewRouter()

	// Middleware que loguea las peticiones en la consola
	r.Use(middleware.Logger)

	// Ruta GET
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		// String "hola mundo" como respuesta
		_, err := w.Write([]byte("hola mundo"))
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
