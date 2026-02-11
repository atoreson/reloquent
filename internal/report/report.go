package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/reloquent/reloquent/internal/validation"
)

// MigrationReport is the final migration report.
type MigrationReport struct {
	Version         string              `json:"version"`
	GeneratedAt     time.Time           `json:"generated_at"`
	Source          SourceSummary       `json:"source"`
	Target          TargetSummary       `json:"target"`
	Migration       MigrationSummary    `json:"migration"`
	Validation      *validation.Result  `json:"validation,omitempty"`
	Indexes         IndexSummary        `json:"indexes"`
	ProductionReady bool                `json:"production_ready"`
	ReadinessChecks []ReadinessCheck    `json:"readiness_checks"`
	NextSteps       []string            `json:"next_steps"`
}

// SourceSummary describes the source database.
type SourceSummary struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Database string `json:"database"`
	Tables   int    `json:"tables"`
}

// TargetSummary describes the target database.
type TargetSummary struct {
	Database    string `json:"database"`
	Topology    string `json:"topology"`
	Collections int    `json:"collections"`
}

// MigrationSummary describes the migration execution.
type MigrationSummary struct {
	Status   string `json:"status"`
	Platform string `json:"platform,omitempty"`
}

// IndexSummary describes the indexes built.
type IndexSummary struct {
	TotalIndexes int    `json:"total_indexes"`
	Status       string `json:"status"`
}

// ReadinessCheck is a single production readiness condition.
type ReadinessCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// GenerateReport creates a MigrationReport from the provided parameters.
func GenerateReport(
	sourceType, sourceHost, sourceDB string,
	sourceTableCount int,
	targetDB, targetTopology string,
	collectionCount int,
	migrationStatus, platform string,
	validationResult *validation.Result,
	indexCount int,
	indexStatus string,
	readinessChecks []ReadinessCheck,
) *MigrationReport {
	allPassed := true
	for _, rc := range readinessChecks {
		if !rc.Passed {
			allPassed = false
			break
		}
	}

	var nextSteps []string
	if !allPassed {
		for _, rc := range readinessChecks {
			if !rc.Passed {
				nextSteps = append(nextSteps, rc.Message)
			}
		}
	} else {
		nextSteps = append(nextSteps, "Switch application connection strings to MongoDB target")
		nextSteps = append(nextSteps, "Monitor MongoDB performance metrics for 24-48 hours")
		nextSteps = append(nextSteps, "Decommission source database when ready")
	}

	return &MigrationReport{
		Version:     "1",
		GeneratedAt: time.Now(),
		Source: SourceSummary{
			Type:     sourceType,
			Host:     sourceHost,
			Database: sourceDB,
			Tables:   sourceTableCount,
		},
		Target: TargetSummary{
			Database:    targetDB,
			Topology:    targetTopology,
			Collections: collectionCount,
		},
		Migration: MigrationSummary{
			Status:   migrationStatus,
			Platform: platform,
		},
		Validation: validationResult,
		Indexes: IndexSummary{
			TotalIndexes: indexCount,
			Status:       indexStatus,
		},
		ProductionReady: allPassed,
		ReadinessChecks: readinessChecks,
		NextSteps:       nextSteps,
	}
}

// WriteJSON writes the report as JSON.
func WriteJSON(report *MigrationReport, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating report directory: %w", err)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// ReadJSON reads a report from a JSON file.
func ReadJSON(path string) (*MigrationReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading report: %w", err)
	}
	r := &MigrationReport{}
	if err := json.Unmarshal(data, r); err != nil {
		return nil, fmt.Errorf("parsing report: %w", err)
	}
	return r, nil
}

// WriteText writes the report as human-readable text.
func WriteText(report *MigrationReport, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating report directory: %w", err)
	}
	return os.WriteFile(path, []byte(FormatText(report)), 0o644)
}

// FormatText renders the report as human-readable text.
func FormatText(report *MigrationReport) string {
	var b strings.Builder

	b.WriteString("=== Reloquent Migration Report ===\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", report.GeneratedAt.Format(time.RFC3339)))

	b.WriteString("Source:\n")
	b.WriteString(fmt.Sprintf("  Type:     %s\n", report.Source.Type))
	b.WriteString(fmt.Sprintf("  Host:     %s\n", report.Source.Host))
	b.WriteString(fmt.Sprintf("  Database: %s\n", report.Source.Database))
	b.WriteString(fmt.Sprintf("  Tables:   %d\n\n", report.Source.Tables))

	b.WriteString("Target:\n")
	b.WriteString(fmt.Sprintf("  Database:    %s\n", report.Target.Database))
	b.WriteString(fmt.Sprintf("  Topology:    %s\n", report.Target.Topology))
	b.WriteString(fmt.Sprintf("  Collections: %d\n\n", report.Target.Collections))

	b.WriteString("Migration:\n")
	b.WriteString(fmt.Sprintf("  Status:   %s\n", report.Migration.Status))
	if report.Migration.Platform != "" {
		b.WriteString(fmt.Sprintf("  Platform: %s\n", report.Migration.Platform))
	}
	b.WriteString("\n")

	if report.Validation != nil {
		b.WriteString(fmt.Sprintf("Validation: %s\n", report.Validation.Status))
		for _, c := range report.Validation.Collections {
			b.WriteString(fmt.Sprintf("  %s: %s\n", c.Name, c.Status))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("Indexes: %d (%s)\n\n", report.Indexes.TotalIndexes, report.Indexes.Status))

	if report.ProductionReady {
		b.WriteString("Production Ready: YES\n\n")
	} else {
		b.WriteString("Production Ready: NO\n\n")
	}

	b.WriteString("Readiness Checks:\n")
	for _, rc := range report.ReadinessChecks {
		status := "PASS"
		if !rc.Passed {
			status = "FAIL"
		}
		b.WriteString(fmt.Sprintf("  [%s] %s\n", status, rc.Name))
	}
	b.WriteString("\n")

	b.WriteString("Next Steps:\n")
	for i, s := range report.NextSteps {
		b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, s))
	}

	return b.String()
}
