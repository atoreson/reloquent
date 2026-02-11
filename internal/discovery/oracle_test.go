package discovery

import (
	"strings"
	"testing"

	"github.com/reloquent/reloquent/internal/config"
)

func TestNewOracle(t *testing.T) {
	cfg := &config.SourceConfig{
		Type:     "oracle",
		Host:     "localhost",
		Port:     1521,
		Database: "ORCL",
		Username: "scott",
		Password: "tiger",
	}

	o, err := NewOracle(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Owner should default to uppercase username
	if o.owner != "SCOTT" {
		t.Errorf("expected owner SCOTT, got %q", o.owner)
	}
}

func TestNewOracle_ExplicitSchema(t *testing.T) {
	cfg := &config.SourceConfig{
		Type:     "oracle",
		Host:     "localhost",
		Port:     1521,
		Database: "ORCL",
		Username: "scott",
		Password: "tiger",
		Schema:   "HR",
	}

	o, err := NewOracle(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if o.owner != "HR" {
		t.Errorf("expected owner HR, got %q", o.owner)
	}
}

func TestOracleConnString(t *testing.T) {
	cfg := &config.SourceConfig{
		Type:     "oracle",
		Host:     "db.example.com",
		Port:     1521,
		Database: "ORCL",
		Username: "scott",
		Password: "tiger",
	}

	o, _ := NewOracle(cfg)
	connStr := o.ConnString()

	if !strings.Contains(connStr, "oracle://") {
		t.Error("connection string should start with oracle://")
	}
	if !strings.Contains(connStr, "db.example.com:1521") {
		t.Error("connection string should contain host:port")
	}
	if !strings.Contains(connStr, "/ORCL") {
		t.Error("connection string should contain database")
	}
}

func TestFactoryDispatch_Oracle(t *testing.T) {
	cfg := &config.SourceConfig{
		Type:     "oracle",
		Host:     "localhost",
		Port:     1521,
		Database: "ORCL",
		Username: "scott",
		Password: "tiger",
	}

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := d.(*Oracle); !ok {
		t.Errorf("expected *Oracle, got %T", d)
	}
}

func TestFactoryDispatch_Unsupported(t *testing.T) {
	cfg := &config.SourceConfig{Type: "mysql"}
	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported DB type")
	}
}
