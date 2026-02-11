package drivers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindOracleJDBC_Found(t *testing.T) {
	dir := t.TempDir()
	driversDir := filepath.Join(dir, ".reloquent", "drivers")
	if err := os.MkdirAll(driversDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a fake JDBC JAR
	jarPath := filepath.Join(driversDir, "ojdbc8.jar")
	if err := os.WriteFile(jarPath, []byte("fake jar"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Override HOME for this test
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	found, err := FindOracleJDBC()
	if err != nil {
		t.Fatalf("expected to find JAR: %v", err)
	}
	if !strings.Contains(found, "ojdbc8.jar") {
		t.Errorf("expected path to contain ojdbc8.jar, got %q", found)
	}
}

func TestFindOracleJDBC_NotFound(t *testing.T) {
	dir := t.TempDir()
	driversDir := filepath.Join(dir, ".reloquent", "drivers")
	if err := os.MkdirAll(driversDir, 0o755); err != nil {
		t.Fatal(err)
	}

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	_, err := FindOracleJDBC()
	if err == nil {
		t.Error("expected error when no JAR found")
	}
}

func TestFindOracleJDBC_NoDirExists(t *testing.T) {
	dir := t.TempDir()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	_, err := FindOracleJDBC()
	if err == nil {
		t.Error("expected error when drivers directory doesn't exist")
	}
}

func TestOracleJDBCGuidance(t *testing.T) {
	guidance := OracleJDBCGuidance()
	if !strings.Contains(guidance, "ojdbc8.jar") {
		t.Error("guidance should mention ojdbc8.jar")
	}
	if !strings.Contains(guidance, "oracle.com") {
		t.Error("guidance should include download URL")
	}
}
