package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	// Importa el paquete de base de datos
	"github.com/Calevin/go_htmx_crud/database"
	"github.com/Calevin/go_htmx_crud/internal/auth"
	"github.com/Calevin/go_htmx_crud/internal/db"
	authMiddleware "github.com/Calevin/go_htmx_crud/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Cargar variables de entorno desde el archivo .env
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró el archivo .env, usando variables de entorno del sistema.")
	}

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		log.Fatal("La variable de entorno JWT_SECRET no está definida.")
	}

	ctx := context.Background()

	// Se iniciala la base de datos. Esto creará el archivo 'crud.db' en la raíz.
	conn := database.InitDB("./crud.db")
	defer conn.Close()

	// Crea una instancia de `Queries` generada por sqlc.
	queries := db.New(conn)
	// Creamos un usuario de prueba si no existe
	createTestUser(ctx, queries)

	// Instancia del router Chi
	r := chi.NewRouter()
	// Middleware que loguea las peticiones en la consola
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// --- Rutas Públicas ---
	// Endpoint que procesa el formulario de login
	r.Post("/login", auth.LoginHandler(queries, jwtSecret))
	// Futura ruta para mostrar el formulario de login con HTMX
	r.Get("/login-page", func(w http.ResponseWriter, r *http.Request) {
		// Aquí se servira el HTML del formulario de login
		w.Write([]byte("Esta será la página de login. Envía un POST a /login."))
	})

	// --- Rutas Protegidas ---
	// Grupo de rutas que usarán el middleware de autenticación
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticator(jwtSecret))

		// Todas las rutas aquí dentro requerirán un JWT válido.
		r.Get("/notas", func(w http.ResponseWriter, r *http.Request) {
			// Se  accede a los claims desde el contexto
			claims := r.Context().Value(authMiddleware.UserContextKey).(*auth.Claims)
			response := "Bienvenido, " + claims.Username + "! Aquí estarán tus notas."
			w.Write([]byte(response))
		})
	})

	// Se inicia el servidor en el puerto 3000
	log.Println("Servidor iniciado en http://localhost:3000")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

// createTestUser crea un usuario para poder probar el login.
func createTestUser(ctx context.Context, queries *db.Queries) {
	username := "testuser"
	_, err := queries.GetUserByUsername(ctx, username)
	// Si el usuario no existe (ErrNoRows), lo creamos.
	if err != nil {
		password := "password123"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Error al hashear la contraseña: %v", err)
		}

		_, err = queries.CreateUser(ctx, db.CreateUserParams{
			Username:     username,
			PasswordHash: string(hashedPassword),
		})
		if err != nil {
			log.Fatalf("Error al crear usuario de prueba: %v", err)
		}
		log.Printf("Usuario de prueba '%s' creado con contraseña '%s'", username, password)
	}
}
