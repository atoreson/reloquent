package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reloquent.yaml")

	content := `version: 1
source:
  type: postgresql
  host: localhost
  port: 5432
  database: testdb
  username: testuser
  password: testpass
target:
  type: mongodb
  connection_string: "mongodb://localhost:27017"
  database: testdb
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Source.Type != "postgresql" {
		t.Errorf("expected source type postgresql, got %s", cfg.Source.Type)
	}
	if cfg.Source.MaxConnections != 20 {
		t.Errorf("expected default max_connections 20, got %d", cfg.Source.MaxConnections)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("expected default log level info, got %s", cfg.Logging.Level)
	}
}

func TestLoadInvalidVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reloquent.yaml")

	content := `version: 99
source:
  type: postgresql
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid version")
	}
}

func TestResolveEnvSecret(t *testing.T) {
	t.Setenv("TEST_SECRET", "mysecret")
	val, err := ResolveValue("${ENV:TEST_SECRET}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "mysecret" {
		t.Errorf("expected mysecret, got %s", val)
	}
}

func TestResolvePlainValue(t *testing.T) {
	val, err := ResolveValue("plaintext")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "plaintext" {
		t.Errorf("expected plaintext, got %s", val)
	}
}

func TestMaxConnectionsCapped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reloquent.yaml")

	content := `version: 1
source:
  type: postgresql
  host: localhost
  port: 5432
  database: testdb
  username: testuser
  password: testpass
  max_connections: 100
target:
  type: mongodb
  connection_string: "mongodb://localhost:27017"
  database: testdb
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Source.MaxConnections != 50 {
		t.Errorf("expected max_connections capped at 50, got %d", cfg.Source.MaxConnections)
	}
}
