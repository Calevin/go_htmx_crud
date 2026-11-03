package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestGETRoutes_Return200 verifica que las peticiones GET de lectura respondan 200 OK (RF-01).
func TestGETRoutes_Return200(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	// Crear fixture basico: usuario, etiqueta y nota
	user := createTestUser(t, ctx, queries, "reader_user", "password123")
	tag := createTestTag(t, ctx, queries, "Trabajo", "#ff0000")
	note := createTestNote(t, ctx, queries, "Nota Existente", "Contenido de lectura", tag.ID)

	testCases := []struct {
		name         string
		url          string
		requiresAuth bool
	}{
		{
			name:         "Formulario de login publico",
			url:          "/login",
			requiresAuth: false,
		},
		{
			name:         "Listado de notas con sesion valida",
			url:          "/notas",
			requiresAuth: true,
		},
		{
			name:         "Formulario de creacion de nota",
			url:          "/crear_nota",
			requiresAuth: true,
		},
		{
			name:         "Formulario de edicion de nota existente",
			url:          fmt.Sprintf("/editar_nota/%d", note.ID),
			requiresAuth: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			if tc.requiresAuth {
				withValidAuthCookie(t, req, user.Username, defaultTestSecret)
			}

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("se esperaba codigo %d en %s pero se obtuvo %d (cuerpo: %s)",
					http.StatusOK, tc.url, recorder.Code, recorder.Body.String())
			}
		})
	}
}

// TestCreateNote_Success_Returns302 verifica creacion exitosa con 302 y redireccion a /notas (RF-03).
func TestCreateNote_Success_Returns302(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	user := createTestUser(t, ctx, queries, "creator_user", "password123")
	tag := createTestTag(t, ctx, queries, "General", "#00ff00")

	formData := url.Values{
		"nombre":    {"Nueva Nota de Prueba"},
		"contenido": {"Contenido creado desde test"},
		"tag_id":    {fmt.Sprintf("%d", tag.ID)},
	}

	req := httptest.NewRequest(http.MethodPost, "/crear_nota", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withValidAuthCookie(t, req, user.Username, defaultTestSecret)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusFound {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d (cuerpo: %s)",
			http.StatusFound, recorder.Code, recorder.Body.String())
	}
	if location := recorder.Header().Get("Location"); location != "/notas" {
		t.Fatalf("se esperaba redireccion a /notas pero se obtuvo %s", location)
	}

	// Verificar persistencia en base de datos
	notes, err := queries.ListNotes(ctx)
	if err != nil {
		t.Fatalf("error consultando notas creadas: %v", err)
	}
	if len(notes) == 0 {
		t.Fatal("se esperaba al menos una nota en la base de datos tras la creacion")
	}
	if notes[0].Nombre != "Nueva Nota de Prueba" {
		t.Fatalf("se esperaba nombre 'Nueva Nota de Prueba' pero se obtuvo '%s'", notes[0].Nombre)
	}
}

// TestUpdateNote_Success_Returns302 verifica actualizacion exitosa con 302 y cambio persistido (RF-03).
func TestUpdateNote_Success_Returns302(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	user := createTestUser(t, ctx, queries, "updater_user", "password123")
	tag := createTestTag(t, ctx, queries, "Urgente", "#ff00ff")
	note := createTestNote(t, ctx, queries, "Titulo Original", "Contenido Original", tag.ID)

	formData := url.Values{
		"nombre":    {"Titulo Actualizado"},
		"contenido": {"Contenido Modificado"},
		"tag_id":    {fmt.Sprintf("%d", tag.ID)},
	}

	updateURL := fmt.Sprintf("/editar_nota/%d", note.ID)
	req := httptest.NewRequest(http.MethodPost, updateURL, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withValidAuthCookie(t, req, user.Username, defaultTestSecret)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusFound {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d (cuerpo: %s)",
			http.StatusFound, recorder.Code, recorder.Body.String())
	}
	if location := recorder.Header().Get("Location"); location != "/notas" {
		t.Fatalf("se esperaba redireccion a /notas pero se obtuvo %s", location)
	}

	// Verificar cambios en persistencia
	updatedNote, err := queries.GetNote(ctx, note.ID)
	if err != nil {
		t.Fatalf("error obteniendo nota actualizada: %v", err)
	}
	if updatedNote.Nombre != "Titulo Actualizado" {
		t.Fatalf("se esperaba titulo 'Titulo Actualizado' pero se obtuvo '%s'", updatedNote.Nombre)
	}
	if updatedNote.Contenido.String != "Contenido Modificado" {
		t.Fatalf("se esperaba contenido 'Contenido Modificado' pero se obtuvo '%s'", updatedNote.Contenido.String)
	}
}

// TestDeleteNote_Success_Returns200RemovesNote verifica eliminacion exitosa con 200 y retiro de persistencia (RF-04).
func TestDeleteNote_Success_Returns200RemovesNote(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	user := createTestUser(t, ctx, queries, "deleter_user", "password123")
	tag := createTestTag(t, ctx, queries, "Temporal", "#0000ff")
	note := createTestNote(t, ctx, queries, "Nota para borrar", "Sera eliminada", tag.ID)

	deleteURL := fmt.Sprintf("/borrar_nota/%d", note.ID)
	req := httptest.NewRequest(http.MethodDelete, deleteURL, nil)
	withValidAuthCookie(t, req, user.Username, defaultTestSecret)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d (cuerpo: %s)",
			http.StatusOK, recorder.Code, recorder.Body.String())
	}

	// Verificar que la nota ya no existe en la base de datos
	_, err := queries.GetNote(ctx, note.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("se esperaba sql.ErrNoRows pero se obtuvo: %v", err)
	}
}

// TestEditForm_NonexistentNote_Returns404 verifica 404 al pedir formulario de edicion para nota inexistente (RF-05).
func TestEditForm_NonexistentNote_Returns404(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	user := createTestUser(t, ctx, queries, "edit404_user", "password123")

	req := httptest.NewRequest(http.MethodGet, "/editar_nota/999999", nil)
	withValidAuthCookie(t, req, user.Username, defaultTestSecret)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d (cuerpo: %s)",
			http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}

// TestUpdate_NonexistentNote_Returns404 verifica 404 al intentar actualizar una nota inexistente (RF-05).
func TestUpdate_NonexistentNote_Returns404(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	user := createTestUser(t, ctx, queries, "update404_user", "password123")
	tag := createTestTag(t, ctx, queries, "Tag404", "#aabbcc")

	formData := url.Values{
		"nombre":    {"Nuevo Nombre"},
		"contenido": {"Nuevo Contenido"},
		"tag_id":    {fmt.Sprintf("%d", tag.ID)},
	}

	req := httptest.NewRequest(http.MethodPost, "/editar_nota/999999", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withValidAuthCookie(t, req, user.Username, defaultTestSecret)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d (cuerpo: %s)",
			http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}

// TestDelete_NonexistentNote_Returns404 verifica 404 al intentar borrar una nota inexistente (RF-05).
func TestDelete_NonexistentNote_Returns404(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	user := createTestUser(t, ctx, queries, "delete404_user", "password123")

	req := httptest.NewRequest(http.MethodDelete, "/borrar_nota/999999", nil)
	withValidAuthCookie(t, req, user.Username, defaultTestSecret)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("se esperaba codigo %d pero se obtuvo %d (cuerpo: %s)",
			http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}

// TestDoubleDelete_Returns404OnSecond verifica 200 en primer borrado y 404 en el segundo (RF-05, CL-01).
func TestDoubleDelete_Returns404OnSecond(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	user := createTestUser(t, ctx, queries, "double_delete_user", "password123")
	tag := createTestTag(t, ctx, queries, "TagDouble", "#112233")
	note := createTestNote(t, ctx, queries, "Nota para doble borrado", "Contenido", tag.ID)

	deleteURL := fmt.Sprintf("/borrar_nota/%d", note.ID)

	// Primer intento: nota existente -> 200 OK
	req1 := httptest.NewRequest(http.MethodDelete, deleteURL, nil)
	withValidAuthCookie(t, req1, user.Username, defaultTestSecret)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("primer borrado: se esperaba codigo %d pero se obtuvo %d", http.StatusOK, rec1.Code)
	}

	// Segundo intento: nota ya eliminada -> 404 Not Found
	req2 := httptest.NewRequest(http.MethodDelete, deleteURL, nil)
	withValidAuthCookie(t, req2, user.Username, defaultTestSecret)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusNotFound {
		t.Fatalf("segundo borrado: se esperaba codigo %d pero se obtuvo %d", http.StatusNotFound, rec2.Code)
	}
}

// TestNonNumericID_Returns400NoWrite verifica 400 sin alterar la base de datos ante IDs no numericos (RF-08, CL-02).
func TestNonNumericID_Returns400NoWrite(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	user := createTestUser(t, ctx, queries, "non_numeric_user", "password123")
	tag := createTestTag(t, ctx, queries, "TagTest", "#123456")
	note := createTestNote(t, ctx, queries, "Nota Base", "Contenido Base", tag.ID)

	testCases := []struct {
		name   string
		method string
		url    string
		body   string
	}{
		{
			name:   "GET editar_nota con ID no numerico",
			method: http.MethodGet,
			url:    "/editar_nota/abc",
		},
		{
			name:   "POST editar_nota con ID no numerico",
			method: http.MethodPost,
			url:    "/editar_nota/abc",
			body:   "nombre=Nuevo&contenido=Nuevo&tag_id=1",
		},
		{
			name:   "DELETE borrar_nota con ID no numerico",
			method: http.MethodDelete,
			url:    "/borrar_nota/abc",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req = httptest.NewRequest(tc.method, tc.url, nil)
			}
			withValidAuthCookie(t, req, user.Username, defaultTestSecret)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("se esperaba codigo %d pero se obtuvo %d (cuerpo: %s)",
					http.StatusBadRequest, recorder.Code, recorder.Body.String())
			}

			// Verificar que la nota base sigue intacta
			currentNote, err := queries.GetNote(ctx, note.ID)
			if err != nil {
				t.Fatalf("error verificando nota tras peticion invalida: %v", err)
			}
			if currentNote.Nombre != "Nota Base" {
				t.Fatal("la nota fue modificada indebidamente tras peticion con ID no numerico")
			}
		})
	}
}

// TestMalformedBody_Returns400NoWrite verifica 400 sin escritura ante cuerpo malformado o tag_id no numerico (RF-08).
func TestMalformedBody_Returns400NoWrite(t *testing.T) {
	ctx := context.Background()
	_, queries := newTestDB(t)
	tpl := newTestTpl(t)
	router := newTestRouter(queries, tpl, defaultTestSecret)

	user := createTestUser(t, ctx, queries, "malformed_body_user", "password123")
	tag := createTestTag(t, ctx, queries, "TagValido", "#654321")
	note := createTestNote(t, ctx, queries, "Nota Original", "Contenido Original", tag.ID)

	testCases := []struct {
		name   string
		method string
		url    string
		body   string
	}{
		{
			name:   "POST crear_nota con tag_id no numerico",
			method: http.MethodPost,
			url:    "/crear_nota",
			body:   "nombre=Titulo&contenido=Texto&tag_id=invalido",
		},
		{
			name:   "POST editar_nota con tag_id no numerico",
			method: http.MethodPost,
			url:    fmt.Sprintf("/editar_nota/%d", note.ID),
			body:   "nombre=TituloNuevo&contenido=TextoNuevo&tag_id=invalido",
		},
		{
			name:   "POST crear_nota con cuerpo malformado urlencoded",
			method: http.MethodPost,
			url:    "/crear_nota",
			body:   "nombre=%ZZinvalido&tag_id=1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			withValidAuthCookie(t, req, user.Username, defaultTestSecret)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("se esperaba codigo %d pero se obtuvo %d (cuerpo: %s)",
					http.StatusBadRequest, recorder.Code, recorder.Body.String())
			}

			// Verificar que no se crearon notas adicionales
			notes, err := queries.ListNotes(ctx)
			if err != nil {
				t.Fatalf("error consultando notas: %v", err)
			}
			if len(notes) != 1 {
				t.Fatalf("se esperaba exactamente 1 nota en base de datos pero hay %d", len(notes))
			}

			// Verificar que la nota original no fue modificada
			currentNote, err := queries.GetNote(ctx, note.ID)
			if err != nil {
				t.Fatalf("error verificando nota original: %v", err)
			}
			if currentNote.Nombre != "Nota Original" {
				t.Fatal("la nota original fue modificada indebidamente tras peticion malformada")
			}
		})
	}
}
