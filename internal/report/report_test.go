package report

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/reloquent/reloquent/internal/validation"
)

func TestJSON_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	report := GenerateReport(
		"postgresql", "localhost", "mydb", 10,
		"target_db", "replica_set", 5,
		"completed", "emr",
		&validation.Result{Status: "PASS"},
		8, "complete",
		[]ReadinessCheck{
			{Name: "validation", Passed: true, Message: "All checks passed"},
			{Name: "indexes", Passed: true, Message: "All indexes built"},
		},
	)

	if err := WriteJSON(report, path); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	loaded, err := ReadJSON(path)
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}

	if loaded.Version != "1" {
		t.Errorf("expected version 1, got %s", loaded.Version)
	}
	if loaded.Source.Type != "postgresql" {
		t.Errorf("expected postgresql, got %s", loaded.Source.Type)
	}
	if loaded.Target.Collections != 5 {
		t.Errorf("expected 5 collections, got %d", loaded.Target.Collections)
	}
	if !loaded.ProductionReady {
		t.Error("expected production ready")
	}
	if loaded.Validation.Status != "PASS" {
		t.Errorf("expected PASS validation, got %s", loaded.Validation.Status)
	}
}

func TestJSON_NotReady(t *testing.T) {
	report := GenerateReport(
		"oracle", "db.example.com", "prod", 20,
		"mongo_prod", "sharded", 10,
		"completed", "glue",
		&validation.Result{Status: "PARTIAL"},
		15, "complete",
		[]ReadinessCheck{
			{Name: "validation", Passed: false, Message: "Fix validation failures"},
			{Name: "indexes", Passed: true, Message: "All indexes built"},
		},
	)

	if report.ProductionReady {
		t.Error("should not be production ready with failed checks")
	}
	if len(report.NextSteps) == 0 {
		t.Error("should have next steps for failed checks")
	}
}

func TestFormatText(t *testing.T) {
	report := GenerateReport(
		"postgresql", "localhost", "mydb", 5,
		"target_db", "replica_set", 3,
		"completed", "emr",
		&validation.Result{
			Status: "PASS",
			Collections: []validation.CollectionResult{
				{Name: "users", Status: "PASS"},
			},
		},
		4, "complete",
		[]ReadinessCheck{
			{Name: "validation", Passed: true, Message: "OK"},
		},
	)

	text := FormatText(report)
	if !strings.Contains(text, "Reloquent Migration Report") {
		t.Error("should contain title")
	}
	if !strings.Contains(text, "postgresql") {
		t.Error("should contain source type")
	}
	if !strings.Contains(text, "Production Ready: YES") {
		t.Error("should show production ready status")
	}
	if !strings.Contains(text, "users: PASS") {
		t.Error("should show collection validation status")
	}
}

func TestWriteText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")

	report := GenerateReport(
		"postgresql", "localhost", "mydb", 1,
		"target", "standalone", 1,
		"completed", "",
		nil, 0, "none",
		nil,
	)

	if err := WriteText(report, path); err != nil {
		t.Fatalf("WriteText: %v", err)
	}
}
