package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// switchDirToRepoRoot ajusta el directorio de trabajo a la raiz del repositorio si el test se ejecuta desde database/.
func switchDirToRepoRoot(t *testing.T) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("error obteniendo el directorio actual de trabajo: %v", err)
	}

	if strings.HasSuffix(origDir, "database") {
		if err := os.Chdir(".."); err != nil {
			t.Fatalf("error cambiando al directorio raiz del repositorio: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chdir(origDir)
		})
	}
}

// TestInitDB valida que InitDB inicialice correctamente una base de datos SQLite temporal y aplique el esquema.
func TestInitDB(t *testing.T) {
	switchDirToRepoRoot(t)

	tempDBPath := filepath.Join(t.TempDir(), "test_init.db")
	db := InitDB(tempDBPath)
	if db == nil {
		t.Fatal("se esperaba una conexion db no nula")
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.Ping(); err != nil {
		t.Fatalf("error en db.Ping(): %v", err)
	}

	// Verificar que las tablas canónicas existen en sqlite_master
	expectedTables := []string{"users", "tags", "notes"}
	for _, tableName := range expectedTables {
		var foundName string
		query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
		err := db.QueryRow(query, tableName).Scan(&foundName)
		if err != nil {
			t.Fatalf("se esperaba que la tabla %q existiera tras aplicar el esquema: %v", tableName, err)
		}
	}

	// Probar una insercion y lectura basica para confirmar operatividad
	_, err := db.Exec("INSERT INTO tags (nombre, color) VALUES (?, ?)", "TestTag", "#000000")
	if err != nil {
		t.Fatalf("error insertando registro de prueba: %v", err)
	}

	var tagCount int
	err = db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount)
	if err != nil || tagCount != 1 {
		t.Fatalf("error verificando conteo de etiquetas (count=%d): %v", tagCount, err)
	}
}

// TestExecuteSchema_FileNotFound valida el comportamiento de error cuando no se encuentra el archivo de esquema.
func TestExecuteSchema_FileNotFound(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("error obteniendo el directorio actual: %v", err)
	}

	// Cambiar a un directorio temporal vacio donde no existe sql/schema/schema.sql
	emptyDir := t.TempDir()
	if err := os.Chdir(emptyDir); err != nil {
		t.Fatalf("error cambiando al directorio temporal vacio: %v", err)
	}
	defer func() {
		_ = os.Chdir(origDir)
	}()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("error abriendo base de datos en memoria: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	err = executeSchema(db)
	if err == nil {
		t.Fatal("se esperaba un error al ejecutar executeSchema sin el archivo de esquema, pero retorno nil")
	}
}
