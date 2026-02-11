package wizard

import (
	"context"
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/benchmark"
	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/indexes"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/postmigration"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/sizing"
	"github.com/reloquent/reloquent/internal/source"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/target"
	"github.com/reloquent/reloquent/internal/typemap"
	"github.com/reloquent/reloquent/internal/validation"
)

// Wizard orchestrates the multi-step interactive migration setup.
type Wizard struct {
	state     *state.State
	statePath string

	// Accumulated data
	sourceConfig *config.SourceConfig
	targetConfig *config.TargetConfig
	schema       *schema.Schema
	mapping      *mapping.Mapping
	typeMap      *typemap.TypeMap

	// Phase 3 data
	sizingPlan   *sizing.SizingPlan
	shardingPlan *sizing.ShardingPlan
	benchResult  *benchmark.Result

	// Phase 4 data
	validationResult *validation.Result
	indexPlan        *indexes.IndexPlan
}

// New creates a new Wizard, loading any saved state for resume.
func New(statePath string) (*Wizard, error) {
	s, err := state.Load(statePath)
	if err != nil {
		return nil, fmt.Errorf("loading wizard state: %w", err)
	}
	return &Wizard{
		state:     s,
		statePath: statePath,
	}, nil
}

// Run executes the wizard from the current step through type mapping (Steps 1-5).
func (w *Wizard) Run() error {
	step := w.state.CurrentStep

	// Step 1: Source connection + discovery
	if step == state.StepSourceConnection {
		if err := w.runSource(); err != nil {
			return err
		}
		step = w.state.CurrentStep
	}

	// Step 2: Target connection
	if step == state.StepTargetConnection {
		if err := w.runTarget(); err != nil {
			return err
		}
		step = w.state.CurrentStep
	}

	// Step 3: Table selection
	if step == state.StepTableSelection {
		if err := w.runTableSelect(); err != nil {
			return err
		}
		step = w.state.CurrentStep
	}

	// Step 4: Denormalization design
	if step == state.StepDenormalization {
		if err := w.runDenorm(); err != nil {
			return err
		}
		step = w.state.CurrentStep
	}

	// Step 5: Type mapping review
	if step == state.StepTypeMapping {
		if err := w.runTypeMapping(); err != nil {
			return err
		}
		step = w.state.CurrentStep
	}

	// Step 6: Sizing
	if step == state.StepSizing {
		if err := w.runSizing(); err != nil {
			return err
		}
		step = w.state.CurrentStep
	}

	// Step 7: AWS Setup
	if step == state.StepAWSSetup {
		if err := w.runAWSSetup(); err != nil {
			return err
		}
		step = w.state.CurrentStep
	}

	// Step 8: Pre-Migration
	if step == state.StepPreMigration {
		if err := w.runPreMigration(); err != nil {
			return err
		}
		step = w.state.CurrentStep
	}

	// Step 8b+9: Review → Migration
	if step == state.StepReview {
		if err := w.runReview(); err != nil {
			return err
		}
		step = w.state.CurrentStep
	}

	if step == state.StepMigration {
		if err := w.runMigrate(); err != nil {
			return err
		}
		step = w.state.CurrentStep
	}

	// Step 10: Validation
	if step == state.StepValidation {
		if err := w.runValidation(); err != nil {
			return err
		}
		step = w.state.CurrentStep
	}

	// Step 11: Index Builds
	if step == state.StepIndexBuilds {
		if err := w.runIndexBuilds(); err != nil {
			return err
		}
	}

	return nil
}

func (w *Wizard) runSource() error {
	m := NewSourceModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running source wizard: %w", err)
	}

	sm := finalModel.(SourceModel)
	if sm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	result := sm.Result()
	if result == nil {
		return fmt.Errorf("no source result")
	}

	w.sourceConfig = result.Config
	w.schema = result.Schema

	// Save schema to disk
	schemaDir := filepath.Dir(config.ExpandHome(w.statePath))
	schemaPath := filepath.Join(schemaDir, "source-schema.yaml")
	if err := w.schema.WriteYAML(schemaPath); err != nil {
		return fmt.Errorf("saving schema: %w", err)
	}

	// Update state
	w.state.SourceConfig = result.Config
	w.state.SchemaPath = schemaPath
	w.state.CompleteStep(state.StepSourceConnection, state.StepTargetConnection)
	if err := w.state.Save(w.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("\nDiscovered %d tables.\n\n", len(w.schema.Tables))
	return nil
}

func (w *Wizard) runTarget() error {
	m := NewTargetModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running target wizard: %w", err)
	}

	tm := finalModel.(TargetModel)
	if tm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	result := tm.Result()
	if result == nil {
		return fmt.Errorf("no target result")
	}

	w.targetConfig = result.Config

	// Update state
	w.state.TargetConfig = result.Config
	w.state.CompleteStep(state.StepTargetConnection, state.StepTableSelection)
	if err := w.state.Save(w.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("\nConnected to MongoDB (%s).\n\n", result.Config.Database)
	return nil
}

func (w *Wizard) runTableSelect() error {
	// Load schema if we're resuming and don't have it in memory
	if w.schema == nil {
		if w.state.SchemaPath == "" {
			return fmt.Errorf("no schema available; run source discovery first")
		}
		s, err := schema.LoadYAML(w.state.SchemaPath)
		if err != nil {
			return fmt.Errorf("loading schema: %w", err)
		}
		w.schema = s
	}

	m := NewTableSelectModel(w.schema.Tables, w.state.SelectedTables)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running table selection: %w", err)
	}

	tsm := finalModel.(TableSelectModel)
	if tsm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	result := tsm.Result()
	if result == nil {
		return fmt.Errorf("no tables selected")
	}

	// Update state with selected table names
	names := make([]string, len(result.Selected))
	for i, t := range result.Selected {
		names[i] = t.Name
	}
	w.state.SelectedTables = names
	w.state.CompleteStep(state.StepTableSelection, state.StepDenormalization)
	if err := w.state.Save(w.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("\nSelected %d tables for migration.\n", len(result.Selected))
	return nil
}

// RunTableSelectStandalone runs only the table selection step.
// Used by the `reloquent select` subcommand.
func RunTableSelectStandalone(schemaPath string, statePath string) error {
	s, err := schema.LoadYAML(schemaPath)
	if err != nil {
		return fmt.Errorf("loading schema: %w", err)
	}

	st, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	m := NewTableSelectModel(s.Tables, st.SelectedTables)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running table selection: %w", err)
	}

	tsm := finalModel.(TableSelectModel)
	if tsm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	result := tsm.Result()
	if result == nil {
		return fmt.Errorf("no tables selected")
	}

	names := make([]string, len(result.Selected))
	for i, t := range result.Selected {
		names[i] = t.Name
	}
	st.SelectedTables = names
	st.CompleteStep(state.StepTableSelection, state.StepDenormalization)
	if err := st.Save(statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("Selected %d tables for migration.\n", len(result.Selected))
	return nil
}

func (w *Wizard) runDenorm() error {
	// Load schema if we're resuming and don't have it in memory
	if w.schema == nil {
		if w.state.SchemaPath == "" {
			return fmt.Errorf("no schema available; run source discovery first")
		}
		s, err := schema.LoadYAML(w.state.SchemaPath)
		if err != nil {
			return fmt.Errorf("loading schema: %w", err)
		}
		w.schema = s
	}

	// Filter schema tables to only selected ones
	selectedSet := make(map[string]bool, len(w.state.SelectedTables))
	for _, n := range w.state.SelectedTables {
		selectedSet[n] = true
	}
	var tables []schema.Table
	for _, t := range w.schema.Tables {
		if selectedSet[t.Name] {
			tables = append(tables, t)
		}
	}

	m := NewDenormModel(tables)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running denormalization designer: %w", err)
	}

	dm := finalModel.(DenormModel)
	if dm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	result := dm.BuildMapping()
	w.mapping = result

	// Save mapping to disk
	stateDir := filepath.Dir(config.ExpandHome(w.statePath))
	mappingPath := filepath.Join(stateDir, "mapping.yaml")
	if err := result.WriteYAML(mappingPath); err != nil {
		return fmt.Errorf("saving mapping: %w", err)
	}

	// Update state
	w.state.MappingPath = mappingPath
	w.state.CompleteStep(state.StepDenormalization, state.StepTypeMapping)
	if err := w.state.Save(w.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("\nMapping saved with %d collections.\n", len(result.Collections))
	return nil
}

// RunDenormStandalone runs only the denormalization designer step.
// Used by the `reloquent design` subcommand.
func RunDenormStandalone(schemaPath string, statePath string) error {
	s, err := schema.LoadYAML(schemaPath)
	if err != nil {
		return fmt.Errorf("loading schema: %w", err)
	}

	st, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// Filter to selected tables
	selectedSet := make(map[string]bool, len(st.SelectedTables))
	for _, n := range st.SelectedTables {
		selectedSet[n] = true
	}
	var tables []schema.Table
	for _, t := range s.Tables {
		if selectedSet[t.Name] {
			tables = append(tables, t)
		}
	}
	// If no selection in state, use all tables
	if len(tables) == 0 {
		tables = s.Tables
	}

	m := NewDenormModel(tables)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running denormalization designer: %w", err)
	}

	dm := finalModel.(DenormModel)
	if dm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	result := dm.BuildMapping()

	// Save mapping
	stateDir := filepath.Dir(config.ExpandHome(statePath))
	mappingPath := filepath.Join(stateDir, "mapping.yaml")
	if err := result.WriteYAML(mappingPath); err != nil {
		return fmt.Errorf("saving mapping: %w", err)
	}

	st.MappingPath = mappingPath
	st.CompleteStep(state.StepDenormalization, state.StepTypeMapping)
	if err := st.Save(statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("Mapping saved with %d collections.\n", len(result.Collections))
	return nil
}

func (w *Wizard) runTypeMapping() error {
	// Load schema if we're resuming and don't have it in memory
	if w.schema == nil {
		if w.state.SchemaPath == "" {
			return fmt.Errorf("no schema available; run source discovery first")
		}
		s, err := schema.LoadYAML(w.state.SchemaPath)
		if err != nil {
			return fmt.Errorf("loading schema: %w", err)
		}
		w.schema = s
	}

	// Determine DB type
	dbType := "postgresql"
	if w.state.SourceConfig != nil {
		dbType = w.state.SourceConfig.Type
	}

	// Load existing type map if available
	var existing *typemap.TypeMap
	if w.state.TypeMappingPath != "" {
		tm, err := typemap.LoadYAML(w.state.TypeMappingPath)
		if err == nil {
			existing = tm
		}
	}

	// Filter schema to selected tables only
	filteredSchema := w.filteredSchema()

	m := NewTypeMapModel(filteredSchema, dbType, existing)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running type mapping review: %w", err)
	}

	tmm := finalModel.(TypeMapModel)
	if tmm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	result := tmm.Result()
	if result == nil {
		return fmt.Errorf("no type mapping result")
	}

	w.typeMap = result

	// Save type mapping to disk
	stateDir := filepath.Dir(config.ExpandHome(w.statePath))
	typeMapPath := filepath.Join(stateDir, "typemap.yaml")
	if err := result.WriteYAML(typeMapPath); err != nil {
		return fmt.Errorf("saving type mapping: %w", err)
	}

	// Update state
	w.state.TypeMappingPath = typeMapPath
	w.state.CompleteStep(state.StepTypeMapping, state.StepSizing)
	if err := w.state.Save(w.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("\nType mapping saved.\n")
	fmt.Println("Run `reloquent generate` to create the PySpark migration script.")
	return nil
}

// RunTypeMapStandalone runs only the type mapping review step.
// Used by the `reloquent config type-mapping` subcommand.
func RunTypeMapStandalone(statePath string) error {
	st, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	if st.SchemaPath == "" {
		return fmt.Errorf("no schema available; run source discovery first")
	}

	s, err := schema.LoadYAML(st.SchemaPath)
	if err != nil {
		return fmt.Errorf("loading schema: %w", err)
	}

	dbType := "postgresql"
	if st.SourceConfig != nil {
		dbType = st.SourceConfig.Type
	}

	var existing *typemap.TypeMap
	if st.TypeMappingPath != "" {
		tm, loadErr := typemap.LoadYAML(st.TypeMappingPath)
		if loadErr == nil {
			existing = tm
		}
	}

	m := NewTypeMapModel(s, dbType, existing)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running type mapping review: %w", err)
	}

	tmm := finalModel.(TypeMapModel)
	if tmm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	result := tmm.Result()
	if result == nil {
		return fmt.Errorf("no type mapping result")
	}

	stateDir := filepath.Dir(config.ExpandHome(statePath))
	typeMapPath := filepath.Join(stateDir, "typemap.yaml")
	if err := result.WriteYAML(typeMapPath); err != nil {
		return fmt.Errorf("saving type mapping: %w", err)
	}

	st.TypeMappingPath = typeMapPath
	st.CompleteStep(state.StepTypeMapping, state.StepSizing)
	if err := st.Save(statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Println("Type mapping saved.")
	return nil
}

func (w *Wizard) runSizing() error {
	// Load schema for data size calculation
	if w.schema == nil && w.state.SchemaPath != "" {
		s, err := schema.LoadYAML(w.state.SchemaPath)
		if err != nil {
			return fmt.Errorf("loading schema: %w", err)
		}
		w.schema = s
	}

	// Compute sizing input from schema
	var totalBytes int64
	var totalRows int64
	for _, t := range w.filteredSchema().Tables {
		totalBytes += t.SizeBytes
		totalRows += t.RowCount
	}

	input := sizing.Input{
		TotalDataBytes:        totalBytes,
		TotalRowCount:         totalRows,
		DenormExpansionFactor: 1.4,
		CollectionCount:       len(w.state.SelectedTables),
	}
	if w.benchResult != nil {
		input.BenchmarkMBps = w.benchResult.ThroughputMBps
	}
	if w.state.SourceConfig != nil {
		input.MaxSourceConnections = w.state.SourceConfig.MaxConnections
	}

	plan := sizing.Calculate(input)
	w.sizingPlan = plan
	if plan.ShardPlan != nil {
		w.shardingPlan = plan.ShardPlan
	}

	m := NewSizingModel(plan)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running sizing step: %w", err)
	}

	sm := finalModel.(SizingModel)
	if sm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	// Save sizing plan
	stateDir := filepath.Dir(config.ExpandHome(w.statePath))
	sizingPath := filepath.Join(stateDir, "sizing.yaml")
	if err := plan.WriteYAML(sizingPath); err != nil {
		return fmt.Errorf("saving sizing plan: %w", err)
	}

	w.state.SizingPlanPath = sizingPath
	w.state.CompleteStep(state.StepSizing, state.StepAWSSetup)
	if err := w.state.Save(w.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("\nSizing plan saved.\n")
	return nil
}

func (w *Wizard) runAWSSetup() error {
	m := NewAWSSetupModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running AWS setup: %w", err)
	}

	am := finalModel.(AWSSetupModel)
	if am.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	result := am.Result()
	fmt.Printf("\nAWS: region=%s, profile=%s, bucket=%s, platform=%s\n",
		result.Region, result.Profile, result.S3Bucket, result.Platform)

	w.state.CompleteStep(state.StepAWSSetup, state.StepPreMigration)
	if err := w.state.Save(w.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	return nil
}

func (w *Wizard) runPreMigration() error {
	collections := w.state.SelectedTables

	m := NewPreMigrationModel(collections)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running pre-migration: %w", err)
	}

	pm := finalModel.(PreMigrationModel)
	if pm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	w.state.CompleteStep(state.StepPreMigration, state.StepReview)
	if err := w.state.Save(w.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("\nPre-migration setup complete.\n")
	return nil
}

func (w *Wizard) runReview() error {
	// Load sizing plan if needed
	if w.sizingPlan == nil && w.state.SizingPlanPath != "" {
		plan, err := sizing.LoadYAML(w.state.SizingPlanPath)
		if err != nil {
			return fmt.Errorf("loading sizing plan: %w", err)
		}
		w.sizingPlan = plan
	}

	m := NewReviewModel(w.sizingPlan, "")
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running review: %w", err)
	}

	rm := finalModel.(ReviewModel)
	if rm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	if !rm.Confirmed() {
		return fmt.Errorf("not confirmed")
	}

	w.state.CompleteStep(state.StepReview, state.StepMigration)
	if err := w.state.Save(w.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	return nil
}

func (w *Wizard) runMigrate() error {
	m := NewMigrateModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running migration: %w", err)
	}

	mm := finalModel.(MigrateModel)
	if mm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	w.state.MigrationStatus = "completed"
	w.state.CompleteStep(state.StepMigration, state.StepValidation)
	if err := w.state.Save(w.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("\nMigration complete.\n")
	return nil
}

func (w *Wizard) runValidation() error {
	// Load schema and mapping if needed
	if err := w.ensureSchemaAndMapping(); err != nil {
		return err
	}

	// Build source reader
	srcReader, err := w.buildSourceReader()
	if err != nil {
		return fmt.Errorf("connecting to source for validation: %w", err)
	}
	defer srcReader.Close()

	// Build target operator
	tgtOp, err := w.buildTargetOperator()
	if err != nil {
		return fmt.Errorf("connecting to target for validation: %w", err)
	}
	defer tgtOp.Close(context.Background())

	// Infer index plan
	w.indexPlan = indexes.Infer(w.filteredSchema(), w.mapping)

	// Create orchestrator
	orch := &postmigration.Orchestrator{
		Source:     srcReader,
		Target:     tgtOp,
		Schema:     w.filteredSchema(),
		Mapping:    w.mapping,
		State:      w.state,
		StatePath:  w.statePath,
		IndexPlan:  w.indexPlan,
		SampleSize: 100,
	}

	// Create validation TUI model
	vm := NewValidationModel()

	cb := postmigration.Callbacks{
		OnValidationCheck: func(collection, checkType string, passed bool) {
			vm.AddCheck(collection, checkType, passed)
		},
	}

	// Run validation
	result, err := orch.RunValidation(context.Background(), cb)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	vm.SetResult(result)
	w.validationResult = result

	// Show the validation TUI
	p := tea.NewProgram(&vm, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running validation UI: %w", err)
	}

	fm := finalModel.(*ValidationModel)
	if fm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	w.state.CompleteStep(state.StepValidation, state.StepIndexBuilds)
	if err := w.state.Save(w.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	return nil
}

func (w *Wizard) runIndexBuilds() error {
	// Load schema and mapping if needed
	if err := w.ensureSchemaAndMapping(); err != nil {
		return err
	}

	// Infer index plan if not already done
	if w.indexPlan == nil {
		w.indexPlan = indexes.Infer(w.filteredSchema(), w.mapping)
	}

	// Build target operator
	tgtOp, err := w.buildTargetOperator()
	if err != nil {
		return fmt.Errorf("connecting to target for index builds: %w", err)
	}
	defer tgtOp.Close(context.Background())

	// Create orchestrator
	orch := &postmigration.Orchestrator{
		Source:    nil, // not needed for index builds
		Target:    tgtOp,
		Schema:    w.filteredSchema(),
		Mapping:   w.mapping,
		State:     w.state,
		StatePath: w.statePath,
		IndexPlan: w.indexPlan,
	}

	// Detect topology for post-ops
	topo, err := tgtOp.DetectTopology(context.Background())
	if err == nil {
		orch.Topology = topo
	}

	// Create index build TUI model
	ibm := NewIndexBuildModel(len(w.indexPlan.Indexes))

	cb := postmigration.Callbacks{
		OnIndexProgress: func(statuses []target.IndexBuildStatus) {
			ibm.UpdateProgress(statuses)
		},
	}

	// Run index builds
	if err := orch.RunIndexBuilds(context.Background(), cb); err != nil {
		return fmt.Errorf("index builds: %w", err)
	}
	ibm.SetFinished()

	// Run post-ops (balancer, write concern)
	if err := orch.RunPostOps(context.Background()); err != nil {
		return fmt.Errorf("post-migration ops: %w", err)
	}

	// Show the index build TUI
	p := tea.NewProgram(&ibm, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running index build UI: %w", err)
	}

	fm := finalModel.(*IndexBuildModel)
	if fm.Cancelled() {
		return fmt.Errorf("cancelled")
	}

	// Check readiness and generate report
	rpt, err := orch.CheckReadiness(context.Background())
	if err != nil {
		return fmt.Errorf("checking readiness: %w", err)
	}

	// Show readiness
	rm := NewReadinessModel()
	rm.SetReport(rpt)
	p2 := tea.NewProgram(&rm, tea.WithAltScreen())
	_, err = p2.Run()
	if err != nil {
		return fmt.Errorf("running readiness UI: %w", err)
	}

	w.state.CompleteStep(state.StepIndexBuilds, state.StepComplete)
	if err := w.state.Save(w.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	return nil
}

func (w *Wizard) ensureSchemaAndMapping() error {
	if w.schema == nil && w.state.SchemaPath != "" {
		s, err := schema.LoadYAML(w.state.SchemaPath)
		if err != nil {
			return fmt.Errorf("loading schema: %w", err)
		}
		w.schema = s
	}
	if w.mapping == nil && w.state.MappingPath != "" {
		m, err := mapping.LoadYAML(w.state.MappingPath)
		if err != nil {
			return fmt.Errorf("loading mapping: %w", err)
		}
		w.mapping = m
	}
	return nil
}

func (w *Wizard) buildSourceReader() (source.Reader, error) {
	if w.state.SourceConfig == nil {
		return nil, fmt.Errorf("no source configuration; run source discovery first")
	}
	sc := w.state.SourceConfig
	var reader source.Reader

	switch sc.Type {
	case "postgresql":
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
			sc.Username, sc.Password, sc.Host, sc.Port, sc.Database)
		if sc.SSL {
			connStr += "?sslmode=require"
		} else {
			connStr += "?sslmode=disable"
		}
		reader = source.NewPostgresReader(connStr, sc.Schema)
	case "oracle":
		connStr := fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
			sc.Username, sc.Password, sc.Host, sc.Port, sc.Database)
		reader = source.NewOracleReader(connStr, sc.Schema)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sc.Type)
	}

	if err := reader.Connect(context.Background()); err != nil {
		return nil, err
	}
	return reader, nil
}

func (w *Wizard) buildTargetOperator() (target.Operator, error) {
	if w.state.TargetConfig == nil {
		return nil, fmt.Errorf("no target configuration; run target setup first")
	}
	return target.NewMongoOperator(context.Background(),
		w.state.TargetConfig.ConnectionString,
		w.state.TargetConfig.Database)
}

// RunSizingStandalone runs only the sizing step.
func RunSizingStandalone(statePath string) (*sizing.SizingPlan, error) {
	st, err := state.Load(statePath)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	if st.SchemaPath == "" {
		return nil, fmt.Errorf("no schema available; run source discovery first")
	}

	s, err := schema.LoadYAML(st.SchemaPath)
	if err != nil {
		return nil, fmt.Errorf("loading schema: %w", err)
	}

	var totalBytes int64
	var totalRows int64
	for _, t := range s.Tables {
		totalBytes += t.SizeBytes
		totalRows += t.RowCount
	}

	input := sizing.Input{
		TotalDataBytes:        totalBytes,
		TotalRowCount:         totalRows,
		DenormExpansionFactor: 1.4,
		CollectionCount:       len(st.SelectedTables),
	}
	if st.SourceConfig != nil {
		input.MaxSourceConnections = st.SourceConfig.MaxConnections
	}

	return sizing.Calculate(input), nil
}

func (w *Wizard) filteredSchema() *schema.Schema {
	if len(w.state.SelectedTables) == 0 {
		return w.schema
	}
	selectedSet := make(map[string]bool, len(w.state.SelectedTables))
	for _, n := range w.state.SelectedTables {
		selectedSet[n] = true
	}
	var tables []schema.Table
	for _, t := range w.schema.Tables {
		if selectedSet[t.Name] {
			tables = append(tables, t)
		}
	}
	return &schema.Schema{
		DatabaseType: w.schema.DatabaseType,
		Host:         w.schema.Host,
		Database:     w.schema.Database,
		SchemaName:   w.schema.SchemaName,
		Tables:       tables,
	}
}
