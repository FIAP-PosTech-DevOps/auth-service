package main

import (
	"strings"
	"testing"
)

func TestGenerateAPIKeyFormato(t *testing.T) {
	key, err := generateAPIKey()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.HasPrefix(key, apiKeyPrefix) {
		t.Errorf("chave sem o prefixo %q: %s", apiKeyPrefix, key)
	}
	// prefixo + 32 bytes em hex (64 caracteres)
	if got, want := len(key), len(apiKeyPrefix)+64; got != want {
		t.Errorf("tamanho da chave = %d, esperado %d", got, want)
	}
}

func TestGenerateAPIKeyUnica(t *testing.T) {
	vistas := map[string]bool{}
	for i := 0; i < 100; i++ {
		key, err := generateAPIKey()
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if vistas[key] {
			t.Fatalf("chave repetida: %s", key)
		}
		vistas[key] = true
	}
}

func TestHashAPIKeyConhecido(t *testing.T) {
	// SHA-256 de "abc" (vetor de teste do FIPS 180-2). O Job de migração do
	// repositório GitOps calcula o mesmo hash no Postgres com
	// encode(sha256(convert_to(:'api_key', 'UTF8')), 'hex').
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := hashAPIKey("abc"); got != want {
		t.Errorf("hashAPIKey(abc) = %s, esperado %s", got, want)
	}
}
