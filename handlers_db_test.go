package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Caminhos dos handlers que consultam o banco, usando o banco falso de
// db_fake_test.go.

func TestValidateChaveValida(t *testing.T) {
	db, fake := newFakeDB(t, fakeResult{id: 7})
	app := &App{DB: db}

	rec := request(t, app, http.MethodGet, "/validate", "", map[string]string{"Authorization": "Bearer tm_key_abc"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200", rec.Code)
	}
	// O banco recebe o hash, nunca a chave em texto plano.
	if got := fake.args[0][0]; got != hashAPIKey("tm_key_abc") {
		t.Errorf("consulta recebeu %v, esperado o hash da chave", got)
	}
}

func TestValidateChaveInvalida(t *testing.T) {
	casos := map[string]fakeResult{
		"chave inexistente ou inativa": {noRows: true},
		"erro no banco":                {err: errors.New("conexão perdida")},
	}
	for nome, r := range casos {
		t.Run(nome, func(t *testing.T) {
			db, _ := newFakeDB(t, r)
			rec := request(t, &App{DB: db}, http.MethodGet, "/validate", "",
				map[string]string{"Authorization": "Bearer tm_key_abc"})
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, esperado 401", rec.Code)
			}
		})
	}
}

func TestCriarChave(t *testing.T) {
	db, fake := newFakeDB(t, fakeResult{id: 1})
	app := &App{DB: db, MasterKey: "segredo-correto"}

	rec := request(t, app, http.MethodPost, "/admin/keys", `{"name":"evaluation-service"}`,
		map[string]string{"Authorization": "Bearer segredo-correto"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, esperado 201 (corpo: %s)", rec.Code, rec.Body.String())
	}
	var resp CreateKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("resposta não é JSON: %v", err)
	}
	if resp.Name != "evaluation-service" || !strings.HasPrefix(resp.Key, apiKeyPrefix) {
		t.Errorf("resposta inesperada: %+v", resp)
	}
	// O que vai para o banco é o hash da chave devolvida, não a chave.
	if got := fake.args[0][1]; got != hashAPIKey(resp.Key) {
		t.Errorf("banco recebeu %v, esperado o hash da chave", got)
	}
}

func TestCriarChaveErroNoBanco(t *testing.T) {
	db, _ := newFakeDB(t, fakeResult{err: errors.New("disco cheio")})
	app := &App{DB: db, MasterKey: "segredo-correto"}

	rec := request(t, app, http.MethodPost, "/admin/keys", `{"name":"x"}`,
		map[string]string{"Authorization": "Bearer segredo-correto"})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, esperado 500", rec.Code)
	}
}

func TestWriteJSONComValorInvalido(t *testing.T) {
	// Um canal não vira JSON: o erro só é registrado no log, sem pânico.
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, make(chan int))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, esperado 200", rec.Code)
	}
}

func TestConnectDBSemBanco(t *testing.T) {
	// Porta 1 em localhost: a conexão é recusada na hora.
	db, err := connectDB("postgres://u:p@127.0.0.1:1/db?sslmode=disable&connect_timeout=2")
	if err == nil {
		_ = db.Close()
		t.Fatal("esperava erro ao conectar num banco inexistente")
	}
}

func TestConnectDBURLInvalida(t *testing.T) {
	if _, err := connectDB("://isto-não-é-uma-url"); err == nil {
		t.Fatal("esperava erro com URL inválida")
	}
}
