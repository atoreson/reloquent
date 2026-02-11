package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/reloquent/reloquent/internal/config"
	"gopkg.in/yaml.v3"
)

const DefaultPath = "~/.reloquent/state.yaml"

// Step represents a wizard step.
type Step string

const (
	StepSourceConnection Step = "source_connection"
	StepTargetConnection Step = "target_connection"
	StepTableSelection   Step = "table_selection"
	StepDenormalization  Step = "denormalization"
	StepTypeMapping      Step = "type_mapping"
	StepSizing           Step = "sizing"
	StepAWSSetup         Step = "aws_setup"
	StepPreMigration     Step = "pre_migration"
	StepReview           Step = "review"
	StepMigration        Step = "migration"
	StepValidation       Step = "validation"
	StepIndexBuilds      Step = "index_builds"
	StepComplete         Step = "complete"
)

// State holds the current wizard progress and accumulated data.
type State struct {
	CurrentStep Step               `yaml:"current_step"`
	LastUpdated time.Time          `yaml:"last_updated"`
	Steps       map[Step]StepState `yaml:"steps,omitempty"`

	// Data accumulated across wizard steps
	SourceConfig    *config.SourceConfig `yaml:"source_config,omitempty"`
	TargetConfig    *config.TargetConfig `yaml:"target_config,omitempty"`
	SchemaPath      string               `yaml:"schema_path,omitempty"`
	SelectedTables  []string             `yaml:"selected_tables,omitempty"`
	MappingPath     string               `yaml:"mapping_path,omitempty"`
	TypeMappingPath string               `yaml:"type_mapping_path,omitempty"`
	ConfigPath      string               `yaml:"config_path,omitempty"`

	// Phase 3: sizing, AWS, and migration state
	SizingPlanPath   string `yaml:"sizing_plan_path,omitempty"`
	ShardingPlanPath string `yaml:"sharding_plan_path,omitempty"`
	AWSResourceID    string `yaml:"aws_resource_id,omitempty"`
	AWSResourceType  string `yaml:"aws_resource_type,omitempty"`
	MigrationStatus  string `yaml:"migration_status,omitempty"`
	S3ArtifactPrefix string `yaml:"s3_artifact_prefix,omitempty"`
	BenchmarkPath    string `yaml:"benchmark_path,omitempty"`

	// Phase 4: validation, indexes, production readiness
	ValidationReportPath string `yaml:"validation_report_path,omitempty"`
	IndexPlanPath        string `yaml:"index_plan_path,omitempty"`
	IndexBuildStatus     string `yaml:"index_build_status,omitempty"`
	BalancerReEnabled    bool   `yaml:"balancer_re_enabled,omitempty"`
	WriteConcernRestored bool   `yaml:"write_concern_restored,omitempty"`
	ProductionReady      bool   `yaml:"production_ready,omitempty"`
	ReportPath           string `yaml:"report_path,omitempty"`
}

// StepState tracks the state of a single wizard step.
type StepState struct {
	Status      string    `yaml:"status"` // pending, in_progress, complete, skipped
	CompletedAt time.Time `yaml:"completed_at,omitempty"`
}

// Load reads the wizard state from disk.
func Load(path string) (*State, error) {
	if path == "" {
		path = config.ExpandHome(DefaultPath)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return New(), nil
		}
		return nil, fmt.Errorf("reading state: %w", err)
	}

	s := &State{}
	if err := yaml.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}
	if s.Steps == nil {
		s.Steps = make(map[Step]StepState)
	}

	return s, nil
}

// Save writes the wizard state to disk.
func (s *State) Save(path string) error {
	if path == "" {
		path = config.ExpandHome(DefaultPath)
	}

	s.LastUpdated = time.Now()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// New creates a fresh wizard state.
func New() *State {
	return &State{
		CurrentStep: StepSourceConnection,
		LastUpdated: time.Now(),
		Steps:       make(map[Step]StepState),
	}
}

// CompleteStep marks a step as complete and advances to the next.
func (s *State) CompleteStep(step Step, next Step) {
	s.Steps[step] = StepState{
		Status:      "complete",
		CompletedAt: time.Now(),
	}
	s.CurrentStep = next
}

// IsStepComplete returns true if the given step has been completed.
func (s *State) IsStepComplete(step Step) bool {
	ss, ok := s.Steps[step]
	return ok && ss.Status == "complete"
}
