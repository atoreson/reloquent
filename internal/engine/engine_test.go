package engine

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/state"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	tmpDir := t.TempDir()
	e := New(&config.Config{Version: 1}, slog.Default())
	e.statePath = filepath.Join(tmpDir, "state.yaml")
	return e
}

func TestNew(t *testing.T) {
	cfg := &config.Config{Version: 1}
	e := New(cfg, slog.Default())
	if e.Config != cfg {
		t.Error("Config not set")
	}
	if e.Logger == nil {
		t.Error("Logger not set")
	}
}

func TestLoadState_Fresh(t *testing.T) {
	e := testEngine(t)
	st, err := e.LoadState()
	if err != nil {
		t.Fatalf("LoadState error: %v", err)
	}
	if st.CurrentStep != state.StepSourceConnection {
		t.Errorf("CurrentStep = %q, want %q", st.CurrentStep, state.StepSourceConnection)
	}
	if e.State != st {
		t.Error("engine.State not set after LoadState")
	}
}

func TestSaveState(t *testing.T) {
	e := testEngine(t)

	// LoadState first to populate e.State
	st, err := e.LoadState()
	if err != nil {
		t.Fatalf("LoadState error: %v", err)
	}
	st.CurrentStep = state.StepTargetConnection

	err = e.SaveState()
	if err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	// Verify it persisted
	st2, err := e.LoadState()
	if err != nil {
		t.Fatalf("second LoadState error: %v", err)
	}
	if st2.CurrentStep != state.StepTargetConnection {
		t.Errorf("after save/load: CurrentStep = %q, want %q", st2.CurrentStep, state.StepTargetConnection)
	}
}

func TestSaveState_NilState(t *testing.T) {
	e := testEngine(t)
	err := e.SaveState()
	if err == nil {
		t.Error("expected error when saving nil state")
	}
}

func TestNavigateToStep_Backward(t *testing.T) {
	e := testEngine(t)

	// Set current step to table_selection
	st, _ := e.LoadState()
	st.CurrentStep = state.StepTableSelection
	e.State = st
	if err := e.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	// Navigate backward to source_connection
	if err := e.NavigateToStep(state.StepSourceConnection); err != nil {
		t.Fatalf("NavigateToStep backward error: %v", err)
	}

	st, _ = e.LoadState()
	if st.CurrentStep != state.StepSourceConnection {
		t.Errorf("CurrentStep = %q, want %q", st.CurrentStep, state.StepSourceConnection)
	}
}

func TestNavigateToStep_Forward(t *testing.T) {
	e := testEngine(t)

	// Start at source_connection (default)
	e.LoadState()

	// Try to navigate forward — should fail
	err := e.NavigateToStep(state.StepTableSelection)
	if err == nil {
		t.Error("expected error navigating forward")
	}
}

func TestNavigateToStep_SameStep(t *testing.T) {
	e := testEngine(t)
	e.LoadState()

	// Navigate to current step — should succeed
	err := e.NavigateToStep(state.StepSourceConnection)
	if err != nil {
		t.Fatalf("NavigateToStep to current step error: %v", err)
	}
}

func TestNavigateToStep_Unknown(t *testing.T) {
	e := testEngine(t)
	e.LoadState()

	err := e.NavigateToStep("nonexistent_step")
	if err == nil {
		t.Error("expected error for unknown step")
	}
}

func TestSetSourceConfig(t *testing.T) {
	e := testEngine(t)
	cfg := &config.SourceConfig{
		Type:     "postgresql",
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
	}
	e.SetSourceConfig(cfg)

	if e.Config.Source.Type != "postgresql" {
		t.Errorf("Source.Type = %q, want %q", e.Config.Source.Type, "postgresql")
	}
	if e.Config.Source.Host != "localhost" {
		t.Errorf("Source.Host = %q", e.Config.Source.Host)
	}
}

func TestSetSourceConfig_NilConfig(t *testing.T) {
	e := &Engine{Logger: slog.Default()}
	cfg := &config.SourceConfig{Type: "postgresql", Host: "localhost"}
	e.SetSourceConfig(cfg)

	if e.Config == nil {
		t.Fatal("Config should be created")
	}
	if e.Config.Version != 1 {
		t.Errorf("Config.Version = %d, want 1", e.Config.Version)
	}
	if e.Config.Source.Host != "localhost" {
		t.Errorf("Source.Host = %q", e.Config.Source.Host)
	}
}

func TestGetSchema_Nil(t *testing.T) {
	e := testEngine(t)
	if e.GetSchema() != nil {
		t.Error("expected nil schema")
	}
}

func TestGetSchema_Set(t *testing.T) {
	e := testEngine(t)
	s := &schema.Schema{DatabaseType: "postgresql", Tables: []schema.Table{
		{Name: "users", RowCount: 100},
	}}
	e.Schema = s

	got := e.GetSchema()
	if got != s {
		t.Error("GetSchema did not return the set schema")
	}
}

func testSchema() *schema.Schema {
	return &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "users", RowCount: 1000, SizeBytes: 100000},
			{Name: "orders", RowCount: 5000, SizeBytes: 500000,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_user", Columns: []string{"user_id"}, ReferencedTable: "users", ReferencedColumns: []string{"id"}},
				},
			},
			{Name: "products", RowCount: 200, SizeBytes: 20000},
		},
	}
}

func TestSelectTables(t *testing.T) {
	e := testEngine(t)
	e.Schema = testSchema()

	if err := e.SelectTables([]string{"users", "orders"}); err != nil {
		t.Fatalf("SelectTables error: %v", err)
	}

	st, _ := e.LoadState()
	if len(st.SelectedTables) != 2 {
		t.Fatalf("SelectedTables count = %d, want 2", len(st.SelectedTables))
	}
}

func TestSelectTables_InvalidTable(t *testing.T) {
	e := testEngine(t)
	e.Schema = testSchema()

	err := e.SelectTables([]string{"users", "nonexistent"})
	if err == nil {
		t.Error("expected error for invalid table name")
	}
}

func TestSelectTables_NoSchema(t *testing.T) {
	e := testEngine(t)
	// No schema set — should still work (no validation)
	if err := e.SelectTables([]string{"anything"}); err != nil {
		t.Fatalf("SelectTables without schema error: %v", err)
	}
}

func TestGetSelectedTables(t *testing.T) {
	e := testEngine(t)
	e.Schema = testSchema()
	e.State = &state.State{
		CurrentStep:    state.StepTableSelection,
		SelectedTables: []string{"users", "products"},
		Steps:          make(map[state.Step]state.StepState),
	}

	selected := e.GetSelectedTables()
	if len(selected) != 2 {
		t.Fatalf("GetSelectedTables() count = %d, want 2", len(selected))
	}
	names := map[string]bool{}
	for _, t := range selected {
		names[t.Name] = true
	}
	if !names["users"] || !names["products"] {
		t.Errorf("unexpected tables: %v", names)
	}
}

func TestGetSelectedTables_NilState(t *testing.T) {
	e := testEngine(t)
	e.Schema = testSchema()
	if got := e.GetSelectedTables(); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestGetSelectedTables_NilSchema(t *testing.T) {
	e := testEngine(t)
	e.State = &state.State{SelectedTables: []string{"users"}, Steps: make(map[state.Step]state.StepState)}
	if got := e.GetSelectedTables(); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestGetOrphanedReferences(t *testing.T) {
	e := testEngine(t)
	e.Schema = testSchema()
	// Select only orders (which references users)
	e.State = &state.State{
		CurrentStep:    state.StepTableSelection,
		SelectedTables: []string{"orders"},
		Steps:          make(map[state.Step]state.StepState),
	}

	orphans := e.GetOrphanedReferences()
	if len(orphans) != 1 {
		t.Fatalf("orphans count = %d, want 1", len(orphans))
	}
	if orphans[0].ReferencedTable != "users" {
		t.Errorf("orphan ref = %q, want %q", orphans[0].ReferencedTable, "users")
	}
}

func TestGetOrphanedReferences_NoOrphans(t *testing.T) {
	e := testEngine(t)
	e.Schema = testSchema()
	e.State = &state.State{
		SelectedTables: []string{"users", "orders", "products"},
		Steps:          make(map[state.Step]state.StepState),
	}

	orphans := e.GetOrphanedReferences()
	if len(orphans) != 0 {
		t.Errorf("expected 0 orphans, got %d", len(orphans))
	}
}

func TestSetMapping_GetMapping(t *testing.T) {
	e := testEngine(t)
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}
	e.SetMapping(m)

	got := e.GetMapping()
	if got != m {
		t.Error("GetMapping did not return the set mapping")
	}
}

func TestGetMapping_Nil(t *testing.T) {
	e := testEngine(t)
	if e.GetMapping() != nil {
		t.Error("expected nil mapping")
	}
}

func TestSaveMappingJSON(t *testing.T) {
	e := testEngine(t)
	// Override HOME so mapping.yaml goes to temp dir
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	m := mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}
	data, _ := json.Marshal(m)

	if err := e.SaveMappingJSON(data); err != nil {
		t.Fatalf("SaveMappingJSON error: %v", err)
	}

	if e.Mapping == nil {
		t.Fatal("Mapping not set after SaveMappingJSON")
	}
	if len(e.Mapping.Collections) != 1 {
		t.Errorf("collections count = %d, want 1", len(e.Mapping.Collections))
	}

	// Verify file was written
	mappingPath := filepath.Join(tmpDir, ".reloquent", "mapping.yaml")
	if _, err := os.Stat(mappingPath); os.IsNotExist(err) {
		t.Error("mapping.yaml not written to disk")
	}
}

func TestSaveMappingJSON_InvalidJSON(t *testing.T) {
	e := testEngine(t)
	err := e.SaveMappingJSON([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestGetTypeMap_NilSchema(t *testing.T) {
	e := testEngine(t)
	if e.GetTypeMap() != nil {
		t.Error("expected nil type map when no schema")
	}
}

func TestGetTypeMap_PostgreSQL(t *testing.T) {
	e := testEngine(t)
	e.Schema = &schema.Schema{DatabaseType: "postgresql"}

	tm := e.GetTypeMap()
	if tm == nil {
		t.Fatal("expected non-nil type map")
	}
	// Check a known postgres mapping
	if tm.Resolve("integer") != "NumberLong" {
		t.Errorf("integer maps to %q, want NumberLong", tm.Resolve("integer"))
	}
}

func TestGetTypeMap_Oracle(t *testing.T) {
	e := testEngine(t)
	e.Schema = &schema.Schema{DatabaseType: "oracle"}

	tm := e.GetTypeMap()
	if tm == nil {
		t.Fatal("expected non-nil type map")
	}
	if tm.Resolve("VARCHAR2") != "String" {
		t.Errorf("VARCHAR2 maps to %q, want String", tm.Resolve("VARCHAR2"))
	}
}

func TestGetTypeMap_Cached(t *testing.T) {
	e := testEngine(t)
	e.Schema = &schema.Schema{DatabaseType: "postgresql"}

	tm1 := e.GetTypeMap()
	tm2 := e.GetTypeMap()
	if tm1 != tm2 {
		t.Error("GetTypeMap should return cached instance")
	}
}

func TestSaveTypeMapOverrides(t *testing.T) {
	e := testEngine(t)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	e.Schema = &schema.Schema{DatabaseType: "postgresql"}

	overrides := map[string]string{
		"integer": "String",
	}
	if err := e.SaveTypeMapOverrides(overrides); err != nil {
		t.Fatalf("SaveTypeMapOverrides error: %v", err)
	}

	tm := e.GetTypeMap()
	if tm.Resolve("integer") != "String" {
		t.Errorf("after override: integer maps to %q, want String", tm.Resolve("integer"))
	}
	if !tm.IsOverridden("integer") {
		t.Error("integer should be marked as overridden")
	}

	// Verify file was written
	tmPath := filepath.Join(tmpDir, ".reloquent", "typemap.yaml")
	if _, err := os.Stat(tmPath); os.IsNotExist(err) {
		t.Error("typemap.yaml not written to disk")
	}
}

func TestSaveTypeMapOverrides_NoTypeMap(t *testing.T) {
	e := testEngine(t)
	err := e.SaveTypeMapOverrides(map[string]string{"integer": "String"})
	if err == nil {
		t.Error("expected error when no type map available")
	}
}

func TestComputeSizing(t *testing.T) {
	e := testEngine(t)
	e.Schema = testSchema()
	e.State = &state.State{
		SelectedTables: []string{"users", "orders"},
		Steps:          make(map[state.Step]state.StepState),
	}

	plan, err := e.ComputeSizing()
	if err != nil {
		t.Fatalf("ComputeSizing error: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil sizing plan")
	}
	if plan.MongoPlan.StorageGB == 0 {
		t.Error("expected non-zero storage estimate")
	}
}

func TestComputeSizing_NoTables(t *testing.T) {
	e := testEngine(t)
	_, err := e.ComputeSizing()
	if err == nil {
		t.Error("expected error when no tables selected")
	}
}

func TestSaveAWSConfig(t *testing.T) {
	e := testEngine(t)
	cfg := &config.AWSConfig{
		Region:   "us-east-1",
		Profile:  "default",
		S3Bucket: "my-bucket",
		Platform: "emr",
	}

	if err := e.SaveAWSConfig(cfg); err != nil {
		t.Fatalf("SaveAWSConfig error: %v", err)
	}

	if e.Config.AWS.Region != "us-east-1" {
		t.Errorf("AWS.Region = %q", e.Config.AWS.Region)
	}
	if e.Config.AWS.Platform != "emr" {
		t.Errorf("AWS.Platform = %q", e.Config.AWS.Platform)
	}
}

func TestSaveAWSConfig_NilConfig(t *testing.T) {
	e := testEngine(t)
	e.Config = nil
	cfg := &config.AWSConfig{Region: "eu-west-1"}

	if err := e.SaveAWSConfig(cfg); err != nil {
		t.Fatalf("SaveAWSConfig error: %v", err)
	}
	if e.Config == nil {
		t.Fatal("Config should be created")
	}
	if e.Config.AWS.Region != "eu-west-1" {
		t.Errorf("AWS.Region = %q", e.Config.AWS.Region)
	}
}

func TestAllStepsOrdered(t *testing.T) {
	steps := allStepsOrdered()
	if len(steps) != 13 {
		t.Fatalf("allStepsOrdered() len = %d, want 13", len(steps))
	}
	if steps[0] != state.StepSourceConnection {
		t.Errorf("first step = %q", steps[0])
	}
	if steps[12] != state.StepComplete {
		t.Errorf("last step = %q", steps[12])
	}
}
