package api

import (
	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/state"
)

// StateResponse is the API response for wizard state.
type StateResponse struct {
	CurrentStep string                       `json:"current_step"`
	Steps       map[string]StepStateResponse `json:"steps"`
	LastUpdated string                       `json:"last_updated"`
}

// StepStateResponse is the API response for a step's state.
type StepStateResponse struct {
	Status      string `json:"status"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// SetStepRequest is the request body for PUT /api/state/step.
type SetStepRequest struct {
	Step string `json:"step"`
}

// SourceConfigRequest is the request body for source connection test.
type SourceConfigRequest struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Schema   string `json:"schema,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSL      bool   `json:"ssl"`
}

// TargetConfigRequest is the request body for target connection test.
type TargetConfigRequest struct {
	ConnectionString string `json:"connection_string"`
	Database         string `json:"database"`
}

// SelectTablesRequest is the request body for table selection.
type SelectTablesRequest struct {
	Tables []string `json:"tables"`
}

// TopologyResponse is the API response for MongoDB topology detection.
type TopologyResponse struct {
	Type          string `json:"type"`
	IsAtlas       bool   `json:"is_atlas"`
	ShardCount    int    `json:"shard_count"`
	ServerVersion string `json:"server_version"`
}

// ConnectionTestResponse is the API response for connection tests.
type ConnectionTestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// AWSConfigRequest is the request body for AWS configuration.
type AWSConfigRequest struct {
	Region   string `json:"region"`
	Profile  string `json:"profile"`
	S3Bucket string `json:"s3_bucket"`
	Platform string `json:"platform"`
}

// StepInfo defines metadata for each wizard step.
type StepInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Order int    `json:"order"`
}

// AllSteps returns ordered metadata for all 12 wizard steps.
var AllSteps = []StepInfo{
	{ID: string(state.StepSourceConnection), Label: "Source Connection", Order: 1},
	{ID: string(state.StepTargetConnection), Label: "Target Connection", Order: 2},
	{ID: string(state.StepTableSelection), Label: "Table Selection", Order: 3},
	{ID: string(state.StepDenormalization), Label: "Denormalization Design", Order: 4},
	{ID: string(state.StepTypeMapping), Label: "Type Mapping", Order: 5},
	{ID: string(state.StepSizing), Label: "Sizing", Order: 6},
	{ID: string(state.StepAWSSetup), Label: "AWS Setup", Order: 7},
	{ID: string(state.StepPreMigration), Label: "Pre-Migration", Order: 8},
	{ID: string(state.StepReview), Label: "Review", Order: 9},
	{ID: string(state.StepMigration), Label: "Migration", Order: 10},
	{ID: string(state.StepValidation), Label: "Validation", Order: 11},
	{ID: string(state.StepIndexBuilds), Label: "Index Builds", Order: 12},
}

// toSourceConfig converts an API request to internal config.
func (r *SourceConfigRequest) toSourceConfig() config.SourceConfig {
	return config.SourceConfig{
		Type:     r.Type,
		Host:     r.Host,
		Port:     r.Port,
		Database: r.Database,
		Schema:   r.Schema,
		Username: r.Username,
		Password: r.Password,
		SSL:      r.SSL,
	}
}

// toTargetConfig converts an API request to internal config.
func (r *TargetConfigRequest) toTargetConfig() config.TargetConfig {
	return config.TargetConfig{
		Type:             "mongodb",
		ConnectionString: r.ConnectionString,
		Database:         r.Database,
	}
}

// BenchmarkRequest is the request body for running a benchmark.
type BenchmarkRequest struct {
	Table        string `json:"table"`
	PartitionCol string `json:"partition_col"`
}

// RetryMigrationRequest is the request body for retrying a migration.
type RetryMigrationRequest struct {
	Collections []string `json:"collections"`
}

// AsyncAcceptedResponse is the response for async operations returning 202.
type AsyncAcceptedResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
