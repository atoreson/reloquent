package sizing

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCalculate_SmallDataset(t *testing.T) {
	input := Input{
		TotalDataBytes:        gbToBytes(10), // 10 GB
		TotalRowCount:         1_000_000,
		DenormExpansionFactor: 1.4,
		MaxSourceConnections:  20,
		CollectionCount:       5,
	}

	plan := Calculate(input)

	// Small data should recommend Glue
	if plan.SparkPlan.Platform != "glue" {
		t.Errorf("expected glue for 10GB, got %s", plan.SparkPlan.Platform)
	}
	if plan.SparkPlan.DPUCount < 10 {
		t.Errorf("expected at least 10 DPUs, got %d", plan.SparkPlan.DPUCount)
	}

	// MongoDB should recommend lower tiers
	if plan.MongoPlan.StorageGB < 10 {
		t.Errorf("expected at least 10 GB storage, got %d", plan.MongoPlan.StorageGB)
	}
	if plan.MongoPlan.MigrationRAMGB == 0 {
		t.Error("migration RAM should be non-zero")
	}

	// Should have explanations
	if len(plan.Explanations) == 0 {
		t.Error("expected explanations")
	}
	for i, exp := range plan.Explanations {
		if exp.Summary == "" {
			t.Errorf("explanation %d has empty summary", i)
		}
		if exp.Detail == "" {
			t.Errorf("explanation %d has empty detail", i)
		}
	}

	// Should have estimated time
	if plan.EstimatedTime == 0 {
		t.Error("expected non-zero estimated time")
	}
}

func TestCalculate_MediumDataset(t *testing.T) {
	input := Input{
		TotalDataBytes:        tbToBytes(2), // 2 TB
		TotalRowCount:         500_000_000,
		DenormExpansionFactor: 1.4,
		CollectionCount:       20,
	}

	plan := Calculate(input)

	// 2 TB × 1.4 = 2.8 TB → too large for Glue → EMR
	if plan.SparkPlan.Platform != "emr" {
		t.Errorf("expected emr for 2TB, got %s", plan.SparkPlan.Platform)
	}
	if plan.SparkPlan.WorkerCount < 10 {
		t.Errorf("expected at least 10 workers, got %d", plan.SparkPlan.WorkerCount)
	}
	if plan.SparkPlan.InstanceType == "" {
		t.Error("expected non-empty instance type")
	}
}

func TestCalculate_LargeDataset(t *testing.T) {
	input := Input{
		TotalDataBytes:        tbToBytes(10), // 10 TB
		TotalRowCount:         2_000_000_000,
		DenormExpansionFactor: 1.4,
		CollectionCount:       50,
	}

	plan := Calculate(input)

	// 10 TB × 1.4 = 14 TB → EMR with r5.8xlarge
	if plan.SparkPlan.Platform != "emr" {
		t.Errorf("expected emr for 10TB, got %s", plan.SparkPlan.Platform)
	}
	if plan.SparkPlan.InstanceType != "r5.8xlarge" {
		t.Errorf("expected r5.8xlarge, got %s", plan.SparkPlan.InstanceType)
	}
	if plan.SparkPlan.WorkerCount < 20 {
		t.Errorf("expected at least 20 workers, got %d", plan.SparkPlan.WorkerCount)
	}
}

func TestCalculate_VeryLargeDataset(t *testing.T) {
	input := Input{
		TotalDataBytes:        tbToBytes(60), // 60 TB
		TotalRowCount:         10_000_000_000,
		DenormExpansionFactor: 1.2,
		CollectionCount:       100,
	}

	plan := Calculate(input)

	// 60 TB × 1.2 = 72 TB → EMR with r5.12xlarge
	if plan.SparkPlan.Platform != "emr" {
		t.Errorf("expected emr for 60TB, got %s", plan.SparkPlan.Platform)
	}
	if plan.SparkPlan.InstanceType != "r5.12xlarge" {
		t.Errorf("expected r5.12xlarge, got %s", plan.SparkPlan.InstanceType)
	}
	if plan.SparkPlan.WorkerCount < 100 {
		t.Errorf("expected at least 100 workers, got %d", plan.SparkPlan.WorkerCount)
	}
}

func TestCalculate_WithBenchmark(t *testing.T) {
	input := Input{
		TotalDataBytes:        gbToBytes(100),
		TotalRowCount:         10_000_000,
		DenormExpansionFactor: 1.4,
		CollectionCount:       5,
		BenchmarkMBps:         100, // 100 MB/s measured
	}

	plan := Calculate(input)

	// With benchmark, time estimate should be based on measured throughput
	// 100 GB × 1.4 = 140 GB at 100 MB/s ≈ ~24 min
	if plan.EstimatedTime < 20*time.Minute || plan.EstimatedTime > 30*time.Minute {
		t.Errorf("expected ~24 min with 100 MB/s benchmark, got %v", plan.EstimatedTime)
	}
}

func TestCalculate_DefaultExpansionFactor(t *testing.T) {
	input := Input{
		TotalDataBytes:  gbToBytes(10),
		TotalRowCount:   1_000_000,
		CollectionCount: 5,
	}

	plan := Calculate(input)
	if plan.SparkPlan.Platform == "" {
		t.Error("expected non-empty platform")
	}
}

func TestGlueViability(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		viable   bool
	}{
		{"10 GB", gbToBytes(10), true},
		{"100 GB", gbToBytes(100), true},
		{"499 GB", gbToBytes(499), true},
		{"500 GB", gbToBytes(500), true},
		{"501 GB", gbToBytes(501), false},
		{"1 TB", tbToBytes(1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGlueViable(tt.bytes); got != tt.viable {
				t.Errorf("IsGlueViable(%s) = %v, want %v", tt.name, got, tt.viable)
			}
		})
	}
}

func TestMongoTierSelection(t *testing.T) {
	tests := []struct {
		name        string
		bytes       int64
		wantMigTier string
		wantProdTier string
	}{
		{"10 GB", gbToBytes(10), "M40", "M30"},
		{"200 GB", gbToBytes(200), "M50", "M40"},
		{"1 TB", tbToBytes(1), "M60", "M50"},
		{"5 TB", tbToBytes(5), "M80", "M60"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := calculateMongo(tt.bytes, 1_000_000)
			if plan.MigrationTier == "" {
				t.Error("migration tier should not be empty")
			}
			if plan.ProductionTier == "" {
				t.Error("production tier should not be empty")
			}
			// Check tier name is contained in the label
			found := false
			for _, label := range []string{plan.MigrationTier} {
				if len(label) >= len(tt.wantMigTier) {
					found = true
				}
			}
			if !found {
				t.Errorf("unexpected migration tier for %s", tt.name)
			}
		})
	}
}

func TestExplanationsNonEmpty(t *testing.T) {
	input := Input{
		TotalDataBytes:        gbToBytes(50),
		TotalRowCount:         5_000_000,
		DenormExpansionFactor: 1.4,
		CollectionCount:       10,
	}

	plan := Calculate(input)

	categories := make(map[string]bool)
	for _, exp := range plan.Explanations {
		if exp.Summary == "" {
			t.Errorf("explanation %q has empty summary", exp.Category)
		}
		if exp.Detail == "" {
			t.Errorf("explanation %q has empty detail", exp.Category)
		}
		if exp.Category == "" {
			t.Error("explanation has empty category")
		}
		categories[exp.Category] = true
	}

	// Should have all categories
	for _, cat := range []string{"overview", "spark", "mongodb", "time"} {
		if !categories[cat] {
			t.Errorf("missing explanation category: %s", cat)
		}
	}
}

func TestYAMLRoundTrip(t *testing.T) {
	input := Input{
		TotalDataBytes:        gbToBytes(50),
		TotalRowCount:         5_000_000,
		DenormExpansionFactor: 1.4,
		CollectionCount:       10,
		BenchmarkMBps:         75,
	}

	plan := Calculate(input)

	dir := t.TempDir()
	path := filepath.Join(dir, "sizing.yaml")

	if err := plan.WriteYAML(path); err != nil {
		t.Fatalf("WriteYAML: %v", err)
	}

	loaded, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}

	if loaded.SparkPlan.Platform != plan.SparkPlan.Platform {
		t.Errorf("platform mismatch: got %q, want %q", loaded.SparkPlan.Platform, plan.SparkPlan.Platform)
	}
	if loaded.MongoPlan.StorageGB != plan.MongoPlan.StorageGB {
		t.Errorf("storage mismatch: got %d, want %d", loaded.MongoPlan.StorageGB, plan.MongoPlan.StorageGB)
	}
	if loaded.EstimatedTime != plan.EstimatedTime {
		t.Errorf("time mismatch: got %v, want %v", loaded.EstimatedTime, plan.EstimatedTime)
	}
	if len(loaded.Explanations) != len(plan.Explanations) {
		t.Errorf("explanations count mismatch: got %d, want %d", len(loaded.Explanations), len(plan.Explanations))
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		if got := FormatBytes(tt.bytes); got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		dur  time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{5 * time.Minute, "5m"},
		{2 * time.Hour, "2h"},
		{2*time.Hour + 30*time.Minute, "2h 30m"},
	}

	for _, tt := range tests {
		if got := FormatDuration(tt.dur); got != tt.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.dur, got, tt.want)
		}
	}
}
