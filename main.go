package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/Calevin/go_htmx_crud/database"
	"github.com/Calevin/go_htmx_crud/internal/db"
	"github.com/Calevin/go_htmx_crud/internal/handlers"
	authMiddleware "github.com/Calevin/go_htmx_crud/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS
var tpl *template.Template

func init() {
	funcMap := template.FuncMap{
		"include": func(templateName string, data any) (template.HTML, error) {
			var buf strings.Builder
			err := tpl.ExecuteTemplate(&buf, templateName, data)
			return template.HTML(buf.String()), err
		},
	}
	tpl = template.New("").Funcs(funcMap)
	tpl = template.Must(tpl.ParseFS(templateFS, "templates/*.html"))
}

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

	// Servidor de archivos estáticos para el CSS
	r.Handle("/static/*", http.FileServer(http.FS(staticFS)))

	// --- Rutas Públicas ---
	// Endpoint que procesa el formulario de login
	r.Post("/login", handlers.LoginHandler(queries, jwtSecret))

	// Endpoint del formulario de login
	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		handlers.Render(tpl, w, "login.html", nil)
	})

	// --- Rutas Protegidas ---
	// Grupo de rutas que usarán el middleware de autenticación
	r.Group(func(r chi.Router) {
		// El middleware de autenticacion se encarga de validar la sesion
		r.Use(authMiddleware.Authenticator(jwtSecret))

		// Todas las rutas aquí dentro requerirán un JWT válido.
		// GET /notas renderiza la página de notas.
		r.Get("/notas", func(w http.ResponseWriter, r *http.Request) {
			handlers.ListNotesHandler(w, r, tpl, queries)
		})

		// POST /logout para cerrar sesión
		r.Post("/logout", handlers.LogoutHandler())

		// GET /crear_nota para mostrar el formulario
		r.Get("/crear_nota", func(w http.ResponseWriter, r *http.Request) {
			handlers.CreateNoteFormHandler(w, r, tpl, queries)
		})

		// POST /crear_nota para procesar el formulario
		r.Post("/crear_nota", func(w http.ResponseWriter, r *http.Request) {
			handlers.CreateNoteHandler(w, r, queries)
		})

		// DELETE /borrar_nota/{id} para borrar una nota
		r.Delete("/borrar_nota/{id}", func(w http.ResponseWriter, r *http.Request) {
			handlers.DeleteNoteHandler(w, r, queries)
		})

		// GET /editar_nota/{id} para mostrar el formulario de edición
		r.Get("/editar_nota/{id}", func(w http.ResponseWriter, r *http.Request) {
			handlers.EditNoteFormHandler(w, r, tpl, queries)
		})

		// POST /editar_nota/{id} para procesar el formulario de edición
		r.Post("/editar_nota/{id}", func(w http.ResponseWriter, r *http.Request) {
			handlers.UpdateNoteHandler(w, r, queries)
		})
	})

	// Redirección de la raíz a /notas (el middleware se encargara de dirigr al login si es necesario)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/notas", http.StatusFound)
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
