package sizing

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Input holds the parameters needed to compute a sizing plan.
type Input struct {
	TotalDataBytes        int64   `yaml:"total_data_bytes"`
	TotalRowCount         int64   `yaml:"total_row_count"`
	DenormExpansionFactor float64 `yaml:"denorm_expansion_factor"` // default 1.4
	MaxSourceConnections  int     `yaml:"max_source_connections"`  // default 20
	CollectionCount       int     `yaml:"collection_count"`
	BenchmarkMBps         float64 `yaml:"benchmark_mbps"` // 0 = not benchmarked
}

// SizingPlan contains the complete sizing recommendations.
type SizingPlan struct {
	SparkPlan     SparkPlan     `yaml:"spark_plan" json:"spark_plan"`
	MongoPlan     MongoPlan     `yaml:"mongo_plan" json:"mongo_plan"`
	ShardPlan     *ShardingPlan `yaml:"shard_plan,omitempty" json:"shard_plan,omitempty"`
	EstimatedTime time.Duration `yaml:"estimated_time" json:"estimated_time"`
	Explanations  []Explanation `yaml:"explanations" json:"explanations"`
}

// SparkPlan describes the recommended Spark cluster configuration.
type SparkPlan struct {
	Platform     string  `yaml:"platform" json:"platform"`
	InstanceType string  `yaml:"instance_type" json:"instance_type"`
	WorkerCount  int     `yaml:"worker_count" json:"worker_count"`
	DPUCount     int     `yaml:"dpu_count" json:"dpu_count"`
	CostEstimate string  `yaml:"cost_estimate" json:"cost_estimate"`
	CostLow      float64 `yaml:"cost_low" json:"cost_low"`
	CostHigh     float64 `yaml:"cost_high" json:"cost_high"`
}

// MongoPlan describes the recommended MongoDB tier.
type MongoPlan struct {
	MigrationTier   string `yaml:"migration_tier" json:"migration_tier"`
	ProductionTier  string `yaml:"production_tier" json:"production_tier"`
	StorageGB       int64  `yaml:"storage_gb" json:"storage_gb"`
	MigrationRAMGB  int    `yaml:"migration_ram_gb" json:"migration_ram_gb"`
	ProductionRAMGB int    `yaml:"production_ram_gb" json:"production_ram_gb"`
}

// Calculate computes a complete sizing plan from the given input.
func Calculate(input Input) *SizingPlan {
	if input.DenormExpansionFactor == 0 {
		input.DenormExpansionFactor = 1.4
	}
	if input.MaxSourceConnections == 0 {
		input.MaxSourceConnections = 20
	}

	estimatedBytes := int64(float64(input.TotalDataBytes) * input.DenormExpansionFactor)

	spark := calculateEMR(estimatedBytes)
	glue := calculateGlue(estimatedBytes)

	// Prefer Glue if viable (simpler, serverless)
	if glue.DPUCount > 0 {
		spark = glue
	}

	mongo := calculateMongo(estimatedBytes, input.TotalRowCount)

	// Estimate migration time
	var estTime time.Duration
	if input.BenchmarkMBps > 0 {
		bytesPerSec := input.BenchmarkMBps * 1024 * 1024
		seconds := float64(estimatedBytes) / bytesPerSec
		estTime = time.Duration(seconds) * time.Second
	} else {
		// Conservative estimate: 50 MB/s with EMR
		bytesPerSec := 50.0 * 1024 * 1024
		seconds := float64(estimatedBytes) / bytesPerSec
		estTime = time.Duration(seconds) * time.Second
	}

	explanations := generateExplanations(input, spark, mongo, estTime)

	// Calculate sharding plan
	shardPlan := CalculateSharding(estimatedBytes, nil)

	plan := &SizingPlan{
		SparkPlan:     spark,
		MongoPlan:     mongo,
		EstimatedTime: estTime,
		Explanations:  explanations,
	}

	if shardPlan.Recommended {
		plan.ShardPlan = shardPlan
		plan.Explanations = append(plan.Explanations, shardPlan.Explanations...)
	}

	return plan
}

// WriteYAML writes the sizing plan to a YAML file.
func (sp *SizingPlan) WriteYAML(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Convert duration to string for YAML
	type yamlPlan struct {
		SparkPlan     SparkPlan     `yaml:"spark_plan"`
		MongoPlan     MongoPlan     `yaml:"mongo_plan"`
		ShardPlan     *ShardingPlan `yaml:"shard_plan,omitempty"`
		EstimatedTime string        `yaml:"estimated_time"`
		Explanations  []Explanation `yaml:"explanations"`
	}

	yp := yamlPlan{
		SparkPlan:     sp.SparkPlan,
		MongoPlan:     sp.MongoPlan,
		ShardPlan:     sp.ShardPlan,
		EstimatedTime: sp.EstimatedTime.String(),
		Explanations:  sp.Explanations,
	}

	data, err := yaml.Marshal(yp)
	if err != nil {
		return fmt.Errorf("marshaling sizing plan: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// LoadYAML reads a sizing plan from a YAML file.
func LoadYAML(path string) (*SizingPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading sizing plan: %w", err)
	}

	type yamlPlan struct {
		SparkPlan     SparkPlan     `yaml:"spark_plan"`
		MongoPlan     MongoPlan     `yaml:"mongo_plan"`
		ShardPlan     *ShardingPlan `yaml:"shard_plan,omitempty"`
		EstimatedTime string        `yaml:"estimated_time"`
		Explanations  []Explanation `yaml:"explanations"`
	}

	var yp yamlPlan
	if err := yaml.Unmarshal(data, &yp); err != nil {
		return nil, fmt.Errorf("parsing sizing plan: %w", err)
	}

	dur, err := time.ParseDuration(yp.EstimatedTime)
	if err != nil {
		return nil, fmt.Errorf("parsing estimated time: %w", err)
	}

	return &SizingPlan{
		SparkPlan:     yp.SparkPlan,
		MongoPlan:     yp.MongoPlan,
		ShardPlan:     yp.ShardPlan,
		EstimatedTime: dur,
		Explanations:  yp.Explanations,
	}, nil
}

// FormatBytes returns a human-readable byte size string.
func FormatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)
	switch {
	case b >= tb:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(tb))
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// FormatDuration returns a human-readable duration string.
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func bytesToGB(b int64) float64 {
	return float64(b) / (1024 * 1024 * 1024)
}

func bytesToTB(b int64) float64 {
	return float64(b) / (1024 * 1024 * 1024 * 1024)
}

func gbToBytes(gb float64) int64 {
	return int64(gb * 1024 * 1024 * 1024)
}

func tbToBytes(tb float64) int64 {
	return int64(tb * 1024 * 1024 * 1024 * 1024)
}

func ceilInt(f float64) int {
	return int(math.Ceil(f))
}
