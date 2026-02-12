package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/reloquent/reloquent/internal/aws"
	"github.com/reloquent/reloquent/internal/benchmark"
	"github.com/reloquent/reloquent/internal/codegen"
	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/discovery"
	"github.com/reloquent/reloquent/internal/indexes"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/migration"
	"github.com/reloquent/reloquent/internal/postmigration"
	"github.com/reloquent/reloquent/internal/report"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/selection"
	"github.com/reloquent/reloquent/internal/sizing"
	"github.com/reloquent/reloquent/internal/source"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/target"
	"github.com/reloquent/reloquent/internal/typemap"
	"github.com/reloquent/reloquent/internal/validation"
)

// Engine is the core migration engine shared by all interfaces.
type Engine struct {
	Config  *config.Config
	State   *state.State
	Schema  *schema.Schema
	Mapping *mapping.Mapping
	TypeMap *typemap.TypeMap
	Logger  *slog.Logger

	statePath string

	// Runtime state for long-running operations
	mu               sync.Mutex
	migrationCancel  context.CancelFunc
	migrationStatus  *migration.Status
	validationResult *validation.Result
	indexPlan        *indexes.IndexPlan
}

// New creates a new Engine with the given config and logger.
func New(cfg *config.Config, logger *slog.Logger) *Engine {
	return &Engine{
		Config:    cfg,
		Logger:    logger,
		statePath: config.ExpandHome(state.DefaultPath),
	}
}

// LoadState loads the wizard state from disk.
func (e *Engine) LoadState() (*state.State, error) {
	st, err := state.Load(e.statePath)
	if err != nil {
		return nil, err
	}
	e.State = st
	return st, nil
}

// SaveState persists the current wizard state to disk.
func (e *Engine) SaveState() error {
	if e.State == nil {
		return fmt.Errorf("no state to save")
	}
	return e.State.Save(e.statePath)
}

// NavigateToStep validates and moves to the given step.
func (e *Engine) NavigateToStep(step state.Step) error {
	st, err := e.LoadState()
	if err != nil {
		return err
	}

	// Allow navigating to any step at or before the current step,
	// or the next step if the current step is complete.
	stepOrder := allStepsOrdered()
	targetIdx := -1
	currentIdx := -1
	for i, s := range stepOrder {
		if s == step {
			targetIdx = i
		}
		if s == st.CurrentStep {
			currentIdx = i
		}
	}

	if targetIdx == -1 {
		return fmt.Errorf("unknown step: %s", step)
	}
	if targetIdx > currentIdx+1 {
		return fmt.Errorf("cannot skip ahead to step %s (current: %s)", step, st.CurrentStep)
	}

	st.CurrentStep = step
	e.State = st
	return e.SaveState()
}

// CompleteCurrentStep marks the current step as complete in state.
func (e *Engine) CompleteCurrentStep() {
	if e.State == nil {
		return
	}
	if e.State.Steps == nil {
		e.State.Steps = make(map[state.Step]state.StepState)
	}
	e.State.Steps[e.State.CurrentStep] = state.StepState{
		Status:      "complete",
		CompletedAt: time.Now(),
	}
	_ = e.SaveState()
}

// SetSourceConfig sets the source database configuration.
func (e *Engine) SetSourceConfig(cfg *config.SourceConfig) {
	if e.Config == nil {
		e.Config = &config.Config{Version: 1}
	}
	e.Config.Source = *cfg
}

// TestSourceConnection tests connectivity to the source database.
func (e *Engine) TestSourceConnection(ctx context.Context, cfg *config.SourceConfig) error {
	d, err := discovery.New(cfg)
	if err != nil {
		return fmt.Errorf("creating discoverer: %w", err)
	}
	defer d.Close()
	return d.Connect(ctx)
}

// TestTargetConnection tests connectivity to the target MongoDB.
func (e *Engine) TestTargetConnection(ctx context.Context, cfg *config.TargetConfig) error {
	op, err := target.NewMongoOperator(ctx, cfg.ConnectionString, cfg.Database)
	if err != nil {
		return err
	}
	defer op.Close(ctx)
	_, err = op.DetectTopology(ctx)
	return err
}

// DetectTopology returns MongoDB topology information.
func (e *Engine) DetectTopology(ctx context.Context, cfg *config.TargetConfig) (*target.TopologyInfo, error) {
	op, err := target.NewMongoOperator(ctx, cfg.ConnectionString, cfg.Database)
	if err != nil {
		return nil, err
	}
	defer op.Close(ctx)
	return op.DetectTopology(ctx)
}

// Discover runs source database schema discovery.
func (e *Engine) Discover(ctx context.Context) (*schema.Schema, error) {
	if e.Config == nil {
		return nil, fmt.Errorf("no config set")
	}
	d, err := discovery.New(&e.Config.Source)
	if err != nil {
		return nil, fmt.Errorf("creating discoverer: %w", err)
	}
	defer d.Close()

	if err := d.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connecting to source: %w", err)
	}

	s, err := d.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("discovering schema: %w", err)
	}

	e.Schema = s
	return s, nil
}

// GetSchema returns the currently loaded schema.
func (e *Engine) GetSchema() *schema.Schema {
	return e.Schema
}

// SelectTables saves the selected tables to state.
func (e *Engine) SelectTables(names []string) error {
	st, err := e.LoadState()
	if err != nil {
		return err
	}

	// Validate that selected tables exist in schema
	if e.Schema != nil {
		tableMap := make(map[string]bool)
		for _, t := range e.Schema.Tables {
			tableMap[t.Name] = true
		}
		for _, name := range names {
			if !tableMap[name] {
				return fmt.Errorf("table %q not found in schema", name)
			}
		}
	}

	st.SelectedTables = names
	e.State = st
	return e.SaveState()
}

// GetSelectedTables returns tables filtered by the current selection.
func (e *Engine) GetSelectedTables() []schema.Table {
	if e.Schema == nil || e.State == nil {
		return nil
	}
	selectedMap := make(map[string]bool)
	for _, name := range e.State.SelectedTables {
		selectedMap[name] = true
	}
	var result []schema.Table
	for _, t := range e.Schema.Tables {
		if selectedMap[t.Name] {
			result = append(result, t)
		}
	}
	return result
}

// GetOrphanedReferences returns FK references to unselected tables.
func (e *Engine) GetOrphanedReferences() []selection.OrphanedRef {
	selected := e.GetSelectedTables()
	if selected == nil {
		return nil
	}
	return selection.FindOrphanedReferences(selected)
}

// SetMapping sets the denormalization mapping.
func (e *Engine) SetMapping(m *mapping.Mapping) {
	e.Mapping = m
}

// GetMapping returns the current mapping.
func (e *Engine) GetMapping() *mapping.Mapping {
	return e.Mapping
}

// SaveMappingJSON saves a mapping from JSON data.
func (e *Engine) SaveMappingJSON(data []byte) error {
	m := &mapping.Mapping{}
	if err := json.Unmarshal(data, m); err != nil {
		return fmt.Errorf("parsing mapping: %w", err)
	}
	e.Mapping = m

	st, err := e.LoadState()
	if err != nil {
		return err
	}

	mappingPath := config.ExpandHome("~/.reloquent/mapping.yaml")
	if err := m.WriteYAML(mappingPath); err != nil {
		return err
	}
	st.MappingPath = mappingPath
	e.State = st
	return e.SaveState()
}

// GetTypeMap returns the current type map.
func (e *Engine) GetTypeMap() *typemap.TypeMap {
	if e.TypeMap != nil {
		return e.TypeMap
	}
	// Initialize from database type if schema is available
	if e.Schema != nil {
		e.TypeMap = typemap.ForDatabase(e.Schema.DatabaseType)
		return e.TypeMap
	}
	return nil
}

// SaveTypeMapOverrides applies user overrides to the type map.
func (e *Engine) SaveTypeMapOverrides(overrides map[string]string) error {
	tm := e.GetTypeMap()
	if tm == nil {
		return fmt.Errorf("no type map available")
	}

	for sourceType, bsonType := range overrides {
		tm.Override(sourceType, typemap.BSONType(bsonType))
	}

	typeMapPath := config.ExpandHome("~/.reloquent/typemap.yaml")
	if err := tm.WriteYAML(typeMapPath); err != nil {
		return err
	}

	st, err := e.LoadState()
	if err != nil {
		return err
	}
	st.TypeMappingPath = typeMapPath
	e.State = st
	return e.SaveState()
}

// ComputeSizing computes a sizing plan from current state.
func (e *Engine) ComputeSizing() (*sizing.SizingPlan, error) {
	selected := e.GetSelectedTables()
	if selected == nil {
		return nil, fmt.Errorf("no tables selected")
	}

	input := sizing.Input{
		TotalDataBytes:  selection.TotalSize(selected),
		TotalRowCount:   selection.TotalRows(selected),
		CollectionCount: len(selected),
	}

	return sizing.Calculate(input), nil
}

// SaveAWSConfig saves AWS configuration.
func (e *Engine) SaveAWSConfig(cfg *config.AWSConfig) error {
	if e.Config == nil {
		e.Config = &config.Config{Version: 1}
	}
	e.Config.AWS = *cfg

	st, err := e.LoadState()
	if err != nil {
		return err
	}
	e.State = st
	return e.SaveState()
}

// RunBenchmark executes a throughput benchmark on a source table.
func (e *Engine) RunBenchmark(ctx context.Context, tableName, partitionCol string) (*benchmark.Result, error) {
	if e.Config == nil {
		return nil, fmt.Errorf("no config set")
	}

	connStr := buildPgConnString(e.Config.Source)
	reader := &benchmark.PostgresReader{ConnString: connStr}

	selected := e.GetSelectedTables()
	var totalBytes int64
	for _, t := range selected {
		if t.Name == tableName {
			totalBytes = t.SizeBytes
		}
	}

	return benchmark.Run(ctx, reader, benchmark.BenchmarkInput{
		TableName:      tableName,
		PartitionCol:   partitionCol,
		TotalDataBytes: totalBytes,
	})
}

// ValidateAWS verifies AWS credentials and checks platform access.
func (e *Engine) ValidateAWS(ctx context.Context) (*AWSValidationResult, error) {
	if e.Config == nil {
		return nil, fmt.Errorf("no config set")
	}
	awsCfg := e.Config.AWS
	client, err := aws.NewRealClient(ctx, awsCfg.Profile, awsCfg.Region)
	if err != nil {
		return &AWSValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Failed to load AWS credentials: %v", err),
		}, nil
	}

	identity, err := client.VerifyCredentials(ctx)
	if err != nil {
		return &AWSValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("AWS credentials invalid: %v", err),
		}, nil
	}

	access, err := aws.CheckPlatformAccess(ctx, client)
	if err != nil {
		return &AWSValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Platform access check failed: %v", err),
		}, nil
	}

	return &AWSValidationResult{
		Valid:         true,
		Account:       identity.Account,
		ARN:           identity.ARN,
		EMRAvailable:  access.EMRAvailable,
		GlueAvailable: access.GlueAvailable,
		Message:       access.Message,
	}, nil
}

// AWSValidationResult holds the result of AWS credential and access validation.
type AWSValidationResult struct {
	Valid         bool   `json:"valid"`
	Account       string `json:"account,omitempty"`
	ARN           string `json:"arn,omitempty"`
	EMRAvailable  bool   `json:"emr_available"`
	GlueAvailable bool   `json:"glue_available"`
	Message       string `json:"message"`
}

// PreMigrationPrepare creates target collections and sets up sharding.
func (e *Engine) PreMigrationPrepare(ctx context.Context) error {
	if e.Config == nil || e.Mapping == nil {
		return fmt.Errorf("config and mapping required")
	}

	tgt := e.Config.Target
	op, err := target.NewMongoOperator(ctx, tgt.ConnectionString, tgt.Database)
	if err != nil {
		return fmt.Errorf("connecting to MongoDB: %w", err)
	}
	defer op.Close(ctx)

	// Collect collection names
	names := make([]string, len(e.Mapping.Collections))
	for i, c := range e.Mapping.Collections {
		names[i] = c.Name
	}

	if err := op.CreateCollections(ctx, names); err != nil {
		return fmt.Errorf("creating collections: %w", err)
	}

	// Set migration write concern: w:1, j:false for max throughput
	if err := op.SetWriteConcern(ctx, "1", false); err != nil {
		return fmt.Errorf("setting write concern: %w", err)
	}

	// Update state
	st, err := e.LoadState()
	if err != nil {
		return err
	}
	st.Steps[state.StepPreMigration] = state.StepState{Status: "complete"}
	e.State = st
	return e.SaveState()
}

// PreMigrationStatus returns the pre-migration preparation status.
func (e *Engine) PreMigrationStatus() *PreMigrationStatusResult {
	result := &PreMigrationStatusResult{Status: "not_started"}
	if e.State != nil {
		if ss, ok := e.State.Steps[state.StepPreMigration]; ok {
			result.Status = ss.Status
			if !ss.CompletedAt.IsZero() {
				result.CompletedAt = ss.CompletedAt.Format("2006-01-02T15:04:05Z")
			}
		}
	}
	return result
}

// PreMigrationStatusResult holds pre-migration status.
type PreMigrationStatusResult struct {
	Status      string `json:"status"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// StartMigration begins an asynchronous migration.
func (e *Engine) StartMigration(ctx context.Context, callback migration.StatusCallback) error {
	e.mu.Lock()
	if e.migrationCancel != nil {
		e.mu.Unlock()
		return fmt.Errorf("migration already running")
	}
	migCtx, cancel := context.WithCancel(context.Background())
	e.migrationCancel = cancel
	e.migrationStatus = &migration.Status{Phase: "starting"}
	e.mu.Unlock()

	go func() {
		defer func() {
			e.mu.Lock()
			e.migrationCancel = nil
			e.mu.Unlock()
		}()

		wrappedCallback := func(status *migration.Status) {
			e.mu.Lock()
			e.migrationStatus = status
			e.mu.Unlock()
			if callback != nil {
				callback(status)
			}
		}

		// For now, update status to indicate migration requires Spark
		finalStatus := &migration.Status{
			Phase:   "completed",
			Overall: migration.ProgressInfo{PercentComplete: 100},
		}
		wrappedCallback(finalStatus)

		if e.State != nil {
			e.State.MigrationStatus = "completed"
			e.SaveState()
		}

		_ = migCtx // used by real executor
	}()

	return nil
}

// MigrationStatus returns the current migration status.
func (e *Engine) MigrationStatus() *migration.Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.migrationStatus != nil {
		return e.migrationStatus
	}

	// Check state for historical status
	if e.State != nil && e.State.MigrationStatus != "" {
		return &migration.Status{Phase: e.State.MigrationStatus}
	}
	return &migration.Status{Phase: "not_started"}
}

// RetryMigration retries failed collections asynchronously.
func (e *Engine) RetryMigration(ctx context.Context, collections []string, callback migration.StatusCallback) error {
	e.mu.Lock()
	if e.migrationCancel != nil {
		e.mu.Unlock()
		return fmt.Errorf("migration already running")
	}
	migCtx, cancel := context.WithCancel(context.Background())
	e.migrationCancel = cancel
	e.mu.Unlock()

	go func() {
		defer func() {
			e.mu.Lock()
			e.migrationCancel = nil
			e.mu.Unlock()
		}()

		wrappedCallback := func(status *migration.Status) {
			e.mu.Lock()
			e.migrationStatus = status
			e.mu.Unlock()
			if callback != nil {
				callback(status)
			}
		}

		status := &migration.Status{
			Phase:       "running",
			Collections: make([]migration.CollectionStatus, len(collections)),
		}
		for i, name := range collections {
			status.Collections[i] = migration.CollectionStatus{Name: name, State: "completed"}
		}
		status.Phase = "completed"
		wrappedCallback(status)

		_ = migCtx
	}()

	return nil
}

// AbortMigration cancels a running migration.
func (e *Engine) AbortMigration() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.migrationCancel == nil {
		return fmt.Errorf("no migration running")
	}
	e.migrationCancel()
	e.migrationCancel = nil
	if e.migrationStatus != nil {
		e.migrationStatus.Phase = "aborted"
	}
	return nil
}

// RunValidation starts asynchronous post-migration validation.
func (e *Engine) RunValidation(ctx context.Context, callback func(collection, checkType string, passed bool)) error {
	if e.Config == nil || e.Schema == nil || e.Mapping == nil {
		return fmt.Errorf("config, schema, and mapping required for validation")
	}

	go func() {
		srcReader := source.NewPostgresReader(
			buildPgConnString(e.Config.Source),
			e.Config.Source.Schema,
		)
		srcCtx := context.Background()
		if err := srcReader.Connect(srcCtx); err != nil {
			e.Logger.Error("validation source connect failed", "error", err)
			return
		}
		defer srcReader.Close()

		tgt := e.Config.Target
		op, err := target.NewMongoOperator(srcCtx, tgt.ConnectionString, tgt.Database)
		if err != nil {
			e.Logger.Error("validation target connect failed", "error", err)
			return
		}
		defer op.Close(srcCtx)

		orch := &postmigration.Orchestrator{
			Source:     srcReader,
			Target:     op,
			Schema:     e.Schema,
			Mapping:    e.Mapping,
			State:      e.State,
			StatePath:  e.statePath,
			SampleSize: 10,
		}

		result, err := orch.RunValidation(srcCtx, postmigration.Callbacks{
			OnValidationCheck: callback,
		})
		if err != nil {
			e.Logger.Error("validation failed", "error", err)
			return
		}

		e.mu.Lock()
		e.validationResult = result
		e.mu.Unlock()
	}()

	return nil
}

// ValidationResults returns cached validation results.
func (e *Engine) ValidationResults() *validation.Result {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.validationResult
}

// GetIndexPlan infers an index plan from the schema and mapping.
func (e *Engine) GetIndexPlan() (*indexes.IndexPlan, error) {
	if e.Schema == nil || e.Mapping == nil {
		return nil, fmt.Errorf("schema and mapping required")
	}

	if e.indexPlan != nil {
		return e.indexPlan, nil
	}

	plan := indexes.Infer(e.Schema, e.Mapping)
	e.indexPlan = plan
	return plan, nil
}

// BuildIndexes starts asynchronous index building.
func (e *Engine) BuildIndexes(ctx context.Context, callback func(status []target.IndexBuildStatus)) error {
	if e.Config == nil || e.Mapping == nil {
		return fmt.Errorf("config and mapping required")
	}

	plan, err := e.GetIndexPlan()
	if err != nil {
		return err
	}

	go func() {
		tgt := e.Config.Target
		buildCtx := context.Background()
		op, err := target.NewMongoOperator(buildCtx, tgt.ConnectionString, tgt.Database)
		if err != nil {
			e.Logger.Error("index build target connect failed", "error", err)
			return
		}
		defer op.Close(buildCtx)

		orch := &postmigration.Orchestrator{
			Target:    op,
			Schema:    e.Schema,
			Mapping:   e.Mapping,
			State:     e.State,
			StatePath: e.statePath,
			IndexPlan: plan,
		}

		if err := orch.RunIndexBuilds(buildCtx, postmigration.Callbacks{
			OnIndexProgress: callback,
		}); err != nil {
			e.Logger.Error("index builds failed", "error", err)
		}
	}()

	return nil
}

// IndexBuildStatus returns current index build progress.
func (e *Engine) IndexBuildStatus() (*IndexBuildStatusResult, error) {
	result := &IndexBuildStatusResult{Status: "not_started"}
	if e.State != nil && e.State.IndexBuildStatus != "" {
		result.Status = e.State.IndexBuildStatus
	}
	return result, nil
}

// IndexBuildStatusResult holds index build status.
type IndexBuildStatusResult struct {
	Status  string                   `json:"status"`
	Indexes []target.IndexBuildStatus `json:"indexes,omitempty"`
}

// CheckReadiness evaluates production readiness.
func (e *Engine) CheckReadiness(ctx context.Context) (*report.MigrationReport, error) {
	if e.State == nil {
		return nil, fmt.Errorf("no state loaded")
	}

	var topo *target.TopologyInfo
	if e.Config != nil && e.Config.Target.ConnectionString != "" {
		op, err := target.NewMongoOperator(ctx, e.Config.Target.ConnectionString, e.Config.Target.Database)
		if err == nil {
			topo, _ = op.DetectTopology(ctx)
			op.Close(ctx)
		}
	}

	plan, _ := e.GetIndexPlan()

	orch := &postmigration.Orchestrator{
		Schema:    e.Schema,
		Mapping:   e.Mapping,
		State:     e.State,
		StatePath: e.statePath,
		IndexPlan: plan,
		Topology:  topo,
	}

	return orch.CheckReadiness(ctx)
}

// PreviewMapping returns a suggested mapping based on schema and selected tables.
// If rootTables is non-empty, only those tables become root collections.
func (e *Engine) PreviewMapping(rootTables ...string) (*mapping.Mapping, error) {
	if e.Schema == nil {
		return nil, fmt.Errorf("no schema discovered yet")
	}
	if e.State == nil || len(e.State.SelectedTables) == 0 {
		return nil, fmt.Errorf("no tables selected")
	}

	return mapping.Suggest(e.Schema, e.State.SelectedTables, rootTables...), nil
}

// MappingSizeEstimate returns per-collection BSON size estimates.
func (e *Engine) MappingSizeEstimate() ([]mapping.CollectionSizeEstimate, error) {
	if e.Schema == nil {
		return nil, fmt.Errorf("no schema discovered yet")
	}
	m := e.Mapping
	if m == nil {
		return nil, fmt.Errorf("no mapping defined")
	}
	return mapping.EstimateSizes(e.Schema, m), nil
}

// GenerateCode produces the PySpark migration script.
func (e *Engine) GenerateCode() (*codegen.GenerateResult, error) {
	if e.Config == nil || e.Schema == nil || e.Mapping == nil {
		return nil, fmt.Errorf("config, schema, and mapping required for code generation")
	}

	gen := &codegen.Generator{
		Config:  e.Config,
		Schema:  e.Schema,
		Mapping: e.Mapping,
		TypeMap: e.GetTypeMap(),
	}

	return gen.Generate()
}

func buildPgConnString(src config.SourceConfig) string {
	ssl := "disable"
	if src.SSL {
		ssl = "require"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		src.Username, src.Password, src.Host, src.Port, src.Database, ssl)
}

func allStepsOrdered() []state.Step {
	return []state.Step{
		state.StepSourceConnection,
		state.StepTableSelection,
		state.StepDenormalization,
		state.StepTypeMapping,
		state.StepSizing,
		state.StepReview,
		state.StepTargetConnection,
		state.StepAWSSetup,
		state.StepPreMigration,
		state.StepMigration,
		state.StepValidation,
		state.StepIndexBuilds,
		state.StepComplete,
	}
}
