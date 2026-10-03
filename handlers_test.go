package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Testes dos caminhos que não dependem do banco. O acesso ao PostgreSQL
// fica coberto pelo Job de migração e pelos testes de integração no cluster.

func request(t *testing.T, app *App, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	rec := request(t, &App{}, http.MethodGet, "/health", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Errorf("corpo inesperado: %s", rec.Body.String())
	}
}

func TestValidateSemHeader(t *testing.T) {
	rec := request(t, &App{}, http.MethodGet, "/validate", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, esperado 401", rec.Code)
	}
}

func TestAdminKeysExigeMasterKey(t *testing.T) {
	app := &App{MasterKey: "segredo-correto"}

	casos := map[string]string{
		"sem header":   "",
		"chave errada": "Bearer segredo-errado",
		"sem Bearer":   "segredo-correto-mas-sem-prefixo",
	}
	for nome, header := range casos {
		t.Run(nome, func(t *testing.T) {
			rec := request(t, app, http.MethodPost, "/admin/keys", `{"name":"x"}`,
				map[string]string{"Authorization": header})
			if rec.Code != http.StatusForbidden {
				t.Errorf("status = %d, esperado 403", rec.Code)
			}
		})
	}
}

func TestAdminKeysValidaRequisicao(t *testing.T) {
	app := &App{MasterKey: "segredo-correto"}
	auth := map[string]string{"Authorization": "Bearer segredo-correto"}

	casos := []struct {
		nome   string
		method string
		body   string
		want   int
	}{
		{"método errado", http.MethodGet, "", http.StatusMethodNotAllowed},
		{"JSON inválido", http.MethodPost, "{", http.StatusBadRequest},
		{"sem nome", http.MethodPost, `{"name":""}`, http.StatusBadRequest},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			rec := request(t, app, c.method, "/admin/keys", c.body, auth)
			if rec.Code != c.want {
				t.Errorf("status = %d, esperado %d", rec.Code, c.want)
			}
		})
	}
}
