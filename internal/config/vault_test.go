package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveVault_Success(t *testing.T) {
	// Mock Vault server returning KV v2 style response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/secret/data/myapp" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-Vault-Token") != "test-token" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"password": "s3cret",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("VAULT_ADDR", server.URL)
	t.Setenv("VAULT_TOKEN", "test-token")

	val, err := resolveVault("secret/data/myapp#password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "s3cret" {
		t.Errorf("expected 's3cret', got %q", val)
	}
}

func TestResolveVault_MissingKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"username": "admin",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("VAULT_ADDR", server.URL)
	t.Setenv("VAULT_TOKEN", "test-token")

	_, err := resolveVault("secret/data/myapp#nonexistent")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestResolveVault_InvalidFormat(t *testing.T) {
	t.Setenv("VAULT_ADDR", "http://localhost:8200")
	t.Setenv("VAULT_TOKEN", "test-token")

	_, err := resolveVault("no-hash-separator")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestResolveVault_MissingEnv(t *testing.T) {
	t.Setenv("VAULT_ADDR", "")
	t.Setenv("VAULT_TOKEN", "")

	_, err := resolveVault("secret/data/path#key")
	if err == nil {
		t.Error("expected error when VAULT_ADDR not set")
	}
}

func TestResolveValue_Vault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"db_pass": "hunter2",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("VAULT_ADDR", server.URL)
	t.Setenv("VAULT_TOKEN", "test-token")

	val, err := ResolveValue("${VAULT:secret/data/myapp#db_pass}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "hunter2" {
		t.Errorf("expected 'hunter2', got %q", val)
	}
}
