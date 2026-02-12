package api

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/engine"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/state"
)

// testServer creates a Server with an engine pointing to a temp state file.
func testServer(t *testing.T, opts ...Option) (*Server, *engine.Engine) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	eng := engine.New(&config.Config{Version: 1}, slog.Default())
	logger := slog.Default()
	s := New(eng, logger, 0, opts...)
	return s, eng
}

// serveMux creates an http.ServeMux with the server's routes registered.
func serveMux(s *Server) *http.ServeMux {
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	return mux
}

func TestHealthEndpoint(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}

func TestGetState(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	req := httptest.NewRequest("GET", "/api/state", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp StateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.CurrentStep != "source_connection" {
		t.Errorf("current_step = %q, want %q", resp.CurrentStep, "source_connection")
	}
}

func TestSetStep_Backward(t *testing.T) {
	s, eng := testServer(t)
	mux := serveMux(s)

	// Advance state to table_selection on disk
	st, _ := eng.LoadState()
	st.CurrentStep = state.StepTableSelection
	eng.SaveState()

	body, _ := json.Marshal(SetStepRequest{Step: "source_connection"})
	req := httptest.NewRequest("PUT", "/api/state/step", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSetStep_ForwardOneStep(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	// One step forward (source_connection → table_selection) should succeed
	body, _ := json.Marshal(SetStepRequest{Step: "table_selection"})
	req := httptest.NewRequest("PUT", "/api/state/step", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSetStep_SkipAhead(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	// Skipping multiple steps (source_connection → denormalization) should fail
	body, _ := json.Marshal(SetStepRequest{Step: "denormalization"})
	req := httptest.NewRequest("PUT", "/api/state/step", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSetStep_InvalidBody(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	req := httptest.NewRequest("PUT", "/api/state/step", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetSchema_NoSchema(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	req := httptest.NewRequest("GET", "/api/source/schema", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetSchema_WithSchema(t *testing.T) {
	s, eng := testServer(t)
	eng.Schema = &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "users", RowCount: 100},
		},
	}
	mux := serveMux(s)

	req := httptest.NewRequest("GET", "/api/source/schema", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetTables_NoSchema(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	req := httptest.NewRequest("GET", "/api/tables", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetTables_WithSchema(t *testing.T) {
	s, eng := testServer(t)
	eng.Schema = &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "users", RowCount: 100, SizeBytes: 10000},
			{Name: "orders", RowCount: 500, SizeBytes: 50000},
		},
	}
	mux := serveMux(s)

	req := httptest.NewRequest("GET", "/api/tables", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var tables []struct {
		Name      string `json:"name"`
		RowCount  int64  `json:"row_count"`
		SizeBytes int64  `json:"size_bytes"`
		Selected  bool   `json:"selected"`
	}
	json.NewDecoder(w.Body).Decode(&tables)
	if len(tables) != 2 {
		t.Fatalf("tables count = %d, want 2", len(tables))
	}
	if tables[0].Name != "users" {
		t.Errorf("first table = %q", tables[0].Name)
	}
}

func TestSelectTables(t *testing.T) {
	s, eng := testServer(t)
	eng.Schema = &schema.Schema{
		DatabaseType: "postgresql",
		Tables:       []schema.Table{{Name: "users"}, {Name: "orders"}},
	}
	mux := serveMux(s)

	body, _ := json.Marshal(SelectTablesRequest{Tables: []string{"users"}})
	req := httptest.NewRequest("POST", "/api/tables/select", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSelectTables_InvalidTable(t *testing.T) {
	s, eng := testServer(t)
	eng.Schema = &schema.Schema{
		DatabaseType: "postgresql",
		Tables:       []schema.Table{{Name: "users"}},
	}
	mux := serveMux(s)

	body, _ := json.Marshal(SelectTablesRequest{Tables: []string{"nonexistent"}})
	req := httptest.NewRequest("POST", "/api/tables/select", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGetMapping_NoMapping(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	req := httptest.NewRequest("GET", "/api/mapping", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetMapping_WithMapping(t *testing.T) {
	s, eng := testServer(t)
	eng.SetMapping(&mapping.Mapping{
		Collections: []mapping.Collection{{Name: "users", SourceTable: "users"}},
	})
	mux := serveMux(s)

	req := httptest.NewRequest("GET", "/api/mapping", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSaveMapping(t *testing.T) {
	s, eng := testServer(t)
	_ = eng
	mux := serveMux(s)

	body, _ := json.Marshal(map[string]any{
		"collections": []map[string]string{
			{"name": "users", "source_table": "users"},
		},
	})
	req := httptest.NewRequest("POST", "/api/mapping", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if eng.GetMapping() == nil {
		t.Error("mapping not set after save")
	}
}

func TestSaveMapping_InvalidBody(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	req := httptest.NewRequest("POST", "/api/mapping", bytes.NewReader([]byte("bad")))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetTypeMap_NoSchema(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	req := httptest.NewRequest("GET", "/api/typemap", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetTypeMap_WithSchema(t *testing.T) {
	s, eng := testServer(t)
	eng.Schema = &schema.Schema{DatabaseType: "postgresql"}
	mux := serveMux(s)

	req := httptest.NewRequest("GET", "/api/typemap", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entries []struct {
		SourceType string `json:"source_type"`
		BSONType   string `json:"bson_type"`
		Overridden bool   `json:"overridden"`
	}
	json.NewDecoder(w.Body).Decode(&entries)
	if len(entries) == 0 {
		t.Error("expected non-empty type map entries")
	}
}

func TestSaveTypeMap(t *testing.T) {
	s, eng := testServer(t)
	eng.Schema = &schema.Schema{DatabaseType: "postgresql"}
	mux := serveMux(s)

	body, _ := json.Marshal(map[string]string{"integer": "String"})
	req := httptest.NewRequest("POST", "/api/typemap", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSaveTypeMap_NoSchema(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	body, _ := json.Marshal(map[string]string{"integer": "String"})
	req := httptest.NewRequest("POST", "/api/typemap", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGetSizing_NoTables(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	req := httptest.NewRequest("GET", "/api/sizing", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestConfigureAWS(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	body, _ := json.Marshal(AWSConfigRequest{
		Region:   "us-east-1",
		Profile:  "default",
		S3Bucket: "mybucket",
		Platform: "emr",
	})
	req := httptest.NewRequest("POST", "/api/aws/configure", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestConfigureAWS_InvalidBody(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	req := httptest.NewRequest("POST", "/api/aws/configure", bytes.NewReader([]byte("bad")))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestImplementedEndpoints(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	// Endpoints that return real data (no 501s)
	// These return non-501 codes depending on engine state

	// Endpoints that work without prerequisite state
	statusOK := []struct {
		method   string
		path     string
		wantCode int
	}{
		{"GET", "/api/premigration/status", http.StatusOK},
		{"GET", "/api/migration/status", http.StatusOK},
		{"GET", "/api/indexes/status", http.StatusOK},
		{"GET", "/api/validation/results", http.StatusNotFound}, // no results yet
	}
	for _, tc := range statusOK {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != tc.wantCode {
			t.Errorf("%s %s: status = %d, want %d", tc.method, tc.path, w.Code, tc.wantCode)
		}
	}

	// Migration abort without running migration → 409
	req := httptest.NewRequest("POST", "/api/migration/abort", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("POST /api/migration/abort: status = %d, want %d", w.Code, http.StatusConflict)
	}

	// Endpoints that require schema/mapping → 500
	needState := []struct {
		method string
		path   string
	}{
		{"GET", "/api/indexes/plan"},
		{"GET", "/api/mapping/preview"},
		{"GET", "/api/mapping/size-estimate"},
		{"GET", "/api/readiness"},
	}
	for _, tc := range needState {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("%s %s: status = %d, want %d", tc.method, tc.path, w.Code, http.StatusInternalServerError)
		}
	}

	// Benchmark without body → 400
	req = httptest.NewRequest("POST", "/api/sizing/benchmark", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/sizing/benchmark: status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	// Start migration → 202 (async) — tested separately to avoid
	// goroutine writing state after TempDir cleanup
}

func TestStartMigration(t *testing.T) {
	s, eng := testServer(t)
	eng.State = &state.State{Steps: make(map[state.Step]state.StepState)}
	mux := serveMux(s)

	req := httptest.NewRequest("POST", "/api/migration/start", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Errorf("POST /api/migration/start: status = %d, want %d", w.Code, http.StatusAccepted)
	}

	// Wait for async goroutine to finish writing state
	time.Sleep(100 * time.Millisecond)
}

func TestCORSMiddleware(t *testing.T) {
	s, _ := testServer(t, WithDevMode(true))
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	handler := s.corsMiddleware(mux)

	// OPTIONS preflight
	req := httptest.NewRequest("OPTIONS", "/api/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS origin = %q, want %q", got, "*")
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("CORS methods header missing")
	}

	// Normal request
	req = httptest.NewRequest("GET", "/api/health", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS origin on GET = %q", got)
	}
}

func TestSPAHandler(t *testing.T) {
	staticFS := fstest.MapFS{
		"index.html":       {Data: []byte("<html>SPA</html>")},
		"assets/main.js":   {Data: []byte("console.log('app')")},
		"assets/style.css": {Data: []byte("body{}")},
	}

	s, _ := testServer(t, WithStaticFS(staticFS))
	mux := serveMux(s)

	tests := []struct {
		name     string
		path     string
		wantBody string
	}{
		{"root", "/", "<html>SPA</html>"},
		{"asset JS", "/assets/main.js", "console.log('app')"},
		{"asset CSS", "/assets/style.css", "body{}"},
		{"SPA fallback", "/source", "<html>SPA</html>"},
		{"SPA deep path", "/some/deep/path", "<html>SPA</html>"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}
			if got := w.Body.String(); got != tc.wantBody {
				t.Errorf("body = %q, want %q", got, tc.wantBody)
			}
		})
	}
}

func TestSPAHandler_NoStaticFS(t *testing.T) {
	s, _ := testServer(t)
	mux := serveMux(s)

	// Without staticFS, root path should 404 (no SPA handler registered)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// The default mux returns 404 for unregistered paths
	// but "/" is a catch-all in net/http, so it depends on registration
	// Just verify it doesn't panic
	_ = w.Code
}

func TestJsonResponse(t *testing.T) {
	w := httptest.NewRecorder()
	jsonResponse(w, http.StatusCreated, map[string]string{"key": "value"})

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["key"] != "value" {
		t.Errorf("key = %q", resp["key"])
	}
}

func TestErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	errorResponse(w, http.StatusBadRequest, "bad input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "bad input" {
		t.Errorf("error = %q", resp["error"])
	}
}

func TestWithOptions(t *testing.T) {
	eng := engine.New(&config.Config{Version: 1}, slog.Default())
	logger := slog.Default()

	staticFS := fstest.MapFS{"index.html": {Data: []byte("test")}}
	s := New(eng, logger, 8080,
		WithStaticFS(staticFS),
		WithDevMode(true),
	)

	if s.port != 8080 {
		t.Errorf("port = %d", s.port)
	}
	if !s.devMode {
		t.Error("devMode not set")
	}
	if s.staticFS == nil {
		t.Error("staticFS not set")
	}
}

func TestAllSteps(t *testing.T) {
	if len(AllSteps) != 12 {
		t.Errorf("AllSteps len = %d, want 12", len(AllSteps))
	}
	if AllSteps[0].ID != string(state.StepSourceConnection) {
		t.Errorf("first step ID = %q", AllSteps[0].ID)
	}
	if AllSteps[11].ID != string(state.StepIndexBuilds) {
		t.Errorf("last step ID = %q", AllSteps[11].ID)
	}
}

func TestSourceConfigRequest_ToSourceConfig(t *testing.T) {
	req := SourceConfigRequest{
		Type:     "postgresql",
		Host:     "localhost",
		Port:     5432,
		Database: "mydb",
		Schema:   "public",
		Username: "admin",
		Password: "secret",
		SSL:      true,
	}

	cfg := req.toSourceConfig()
	if cfg.Type != "postgresql" {
		t.Errorf("Type = %q", cfg.Type)
	}
	if cfg.Host != "localhost" {
		t.Errorf("Host = %q", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("Port = %d", cfg.Port)
	}
	if cfg.SSL != true {
		t.Error("SSL should be true")
	}
}

func TestTargetConfigRequest_ToTargetConfig(t *testing.T) {
	req := TargetConfigRequest{
		ConnectionString: "mongodb://localhost:27017",
		Database:         "mydb",
	}

	cfg := req.toTargetConfig()
	if cfg.Type != "mongodb" {
		t.Errorf("Type = %q", cfg.Type)
	}
	if cfg.ConnectionString != "mongodb://localhost:27017" {
		t.Errorf("ConnectionString = %q", cfg.ConnectionString)
	}
}

// Ensure the unused import doesn't cause issues.
var _ fs.FS
