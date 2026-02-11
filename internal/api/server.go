package api

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/reloquent/reloquent/internal/engine"
	"github.com/reloquent/reloquent/internal/ws"
)

// Server is the REST API server for the web UI.
type Server struct {
	engine  *engine.Engine
	hub     *ws.Hub
	logger  *slog.Logger
	port    int
	server  *http.Server
	staticFS fs.FS
	devMode  bool
}

// Option configures the API server.
type Option func(*Server)

// WithStaticFS sets the embedded filesystem for serving the React app.
func WithStaticFS(fsys fs.FS) Option {
	return func(s *Server) {
		s.staticFS = fsys
	}
}

// WithDevMode enables CORS for development.
func WithDevMode(dev bool) Option {
	return func(s *Server) {
		s.devMode = dev
	}
}

// WithHub sets the WebSocket hub.
func WithHub(hub *ws.Hub) Option {
	return func(s *Server) {
		s.hub = hub
	}
}

// New creates a new API server.
func New(eng *engine.Engine, logger *slog.Logger, port int, opts ...Option) *Server {
	s := &Server{
		engine: eng,
		logger: logger,
		port:   port,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	var handler http.Handler = mux
	if s.devMode {
		handler = s.corsMiddleware(mux)
	}

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: handler,
	}

	s.logger.Info("starting web UI server", "port", s.port, "dev_mode", s.devMode)
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// API routes
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/state", s.handleGetState)
	mux.HandleFunc("PUT /api/state/step", s.handleSetStep)
	mux.HandleFunc("GET /api/source/config", s.handleGetSourceConfig)
	mux.HandleFunc("POST /api/source/test-connection", s.handleTestSourceConnection)
	mux.HandleFunc("POST /api/source/discover", s.handleDiscover)
	mux.HandleFunc("GET /api/source/schema", s.handleGetSchema)
	mux.HandleFunc("GET /api/target/config", s.handleGetTargetConfig)
	mux.HandleFunc("POST /api/target/test-connection", s.handleTestTargetConnection)
	mux.HandleFunc("POST /api/target/detect-topology", s.handleDetectTopology)
	mux.HandleFunc("GET /api/tables", s.handleGetTables)
	mux.HandleFunc("POST /api/tables/select", s.handleSelectTables)
	mux.HandleFunc("GET /api/mapping", s.handleGetMapping)
	mux.HandleFunc("POST /api/mapping", s.handleSaveMapping)
	mux.HandleFunc("GET /api/mapping/preview", s.handleGetMappingPreview)
	mux.HandleFunc("GET /api/mapping/size-estimate", s.handleGetSizeEstimate)
	mux.HandleFunc("GET /api/typemap", s.handleGetTypeMap)
	mux.HandleFunc("POST /api/typemap", s.handleSaveTypeMap)
	mux.HandleFunc("GET /api/sizing", s.handleGetSizing)
	mux.HandleFunc("POST /api/sizing/benchmark", s.handleRunBenchmark)
	mux.HandleFunc("POST /api/aws/configure", s.handleConfigureAWS)
	mux.HandleFunc("GET /api/aws/validate", s.handleValidateAWS)
	mux.HandleFunc("POST /api/premigration/prepare", s.handlePreMigrationPrepare)
	mux.HandleFunc("GET /api/premigration/status", s.handlePreMigrationStatus)
	mux.HandleFunc("POST /api/migration/start", s.handleStartMigration)
	mux.HandleFunc("GET /api/migration/status", s.handleMigrationStatus)
	mux.HandleFunc("POST /api/migration/retry", s.handleRetryMigration)
	mux.HandleFunc("POST /api/migration/abort", s.handleAbortMigration)
	mux.HandleFunc("POST /api/validation/run", s.handleRunValidation)
	mux.HandleFunc("GET /api/validation/results", s.handleValidationResults)
	mux.HandleFunc("GET /api/indexes/plan", s.handleGetIndexPlan)
	mux.HandleFunc("POST /api/indexes/build", s.handleBuildIndexes)
	mux.HandleFunc("GET /api/indexes/status", s.handleIndexStatus)
	mux.HandleFunc("GET /api/readiness", s.handleReadiness)

	// WebSocket
	if s.hub != nil {
		mux.HandleFunc("/api/ws", s.hub.HandleWebSocket)
	}

	// SPA static file serving
	if s.staticFS != nil {
		mux.Handle("/", s.spaHandler())
	}
}

// spaHandler serves the React SPA. For any non-API, non-asset request,
// it returns index.html so client-side routing works.
func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		// Try to serve the file directly
		f, err := s.staticFS.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found — serve index.html for SPA routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// Handlers delegate to implementations in handlers.go
func (s *Server) handleGetState(w http.ResponseWriter, r *http.Request) {
	s.handleGetStateImpl(w, r)
}
func (s *Server) handleSetStep(w http.ResponseWriter, r *http.Request) {
	s.handleSetStepImpl(w, r)
}
func (s *Server) handleGetSourceConfig(w http.ResponseWriter, r *http.Request) {
	s.handleGetSourceConfigImpl(w, r)
}
func (s *Server) handleTestSourceConnection(w http.ResponseWriter, r *http.Request) {
	s.handleTestSourceConnectionImpl(w, r)
}
func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
	s.handleDiscoverImpl(w, r)
}
func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	s.handleGetSchemaImpl(w, r)
}
func (s *Server) handleGetTargetConfig(w http.ResponseWriter, r *http.Request) {
	s.handleGetTargetConfigImpl(w, r)
}
func (s *Server) handleTestTargetConnection(w http.ResponseWriter, r *http.Request) {
	s.handleTestTargetConnectionImpl(w, r)
}
func (s *Server) handleDetectTopology(w http.ResponseWriter, r *http.Request) {
	s.handleDetectTopologyImpl(w, r)
}
func (s *Server) handleGetTables(w http.ResponseWriter, r *http.Request) {
	s.handleGetTablesImpl(w, r)
}
func (s *Server) handleSelectTables(w http.ResponseWriter, r *http.Request) {
	s.handleSelectTablesImpl(w, r)
}
func (s *Server) handleGetMapping(w http.ResponseWriter, r *http.Request) {
	s.handleGetMappingImpl(w, r)
}
func (s *Server) handleSaveMapping(w http.ResponseWriter, r *http.Request) {
	s.handleSaveMappingImpl(w, r)
}
func (s *Server) handleGetMappingPreview(w http.ResponseWriter, r *http.Request) {
	s.handleGetMappingPreviewImpl(w, r)
}
func (s *Server) handleGetSizeEstimate(w http.ResponseWriter, r *http.Request) {
	s.handleGetSizeEstimateImpl(w, r)
}
func (s *Server) handleGetTypeMap(w http.ResponseWriter, r *http.Request) {
	s.handleGetTypeMapImpl(w, r)
}
func (s *Server) handleSaveTypeMap(w http.ResponseWriter, r *http.Request) {
	s.handleSaveTypeMapImpl(w, r)
}
func (s *Server) handleGetSizing(w http.ResponseWriter, r *http.Request) {
	s.handleGetSizingImpl(w, r)
}
func (s *Server) handleRunBenchmark(w http.ResponseWriter, r *http.Request) {
	s.handleRunBenchmarkImpl(w, r)
}
func (s *Server) handleConfigureAWS(w http.ResponseWriter, r *http.Request) {
	s.handleConfigureAWSImpl(w, r)
}
func (s *Server) handleValidateAWS(w http.ResponseWriter, r *http.Request) {
	s.handleValidateAWSImpl(w, r)
}
func (s *Server) handlePreMigrationPrepare(w http.ResponseWriter, r *http.Request) {
	s.handlePreMigrationPrepareImpl(w, r)
}
func (s *Server) handlePreMigrationStatus(w http.ResponseWriter, r *http.Request) {
	s.handlePreMigrationStatusImpl(w, r)
}
func (s *Server) handleStartMigration(w http.ResponseWriter, r *http.Request) {
	s.handleStartMigrationImpl(w, r)
}
func (s *Server) handleMigrationStatus(w http.ResponseWriter, r *http.Request) {
	s.handleMigrationStatusImpl(w, r)
}
func (s *Server) handleRetryMigration(w http.ResponseWriter, r *http.Request) {
	s.handleRetryMigrationImpl(w, r)
}
func (s *Server) handleAbortMigration(w http.ResponseWriter, r *http.Request) {
	s.handleAbortMigrationImpl(w, r)
}
func (s *Server) handleRunValidation(w http.ResponseWriter, r *http.Request) {
	s.handleRunValidationImpl(w, r)
}
func (s *Server) handleValidationResults(w http.ResponseWriter, r *http.Request) {
	s.handleValidationResultsImpl(w, r)
}
func (s *Server) handleGetIndexPlan(w http.ResponseWriter, r *http.Request) {
	s.handleGetIndexPlanImpl(w, r)
}
func (s *Server) handleBuildIndexes(w http.ResponseWriter, r *http.Request) {
	s.handleBuildIndexesImpl(w, r)
}
func (s *Server) handleIndexStatus(w http.ResponseWriter, r *http.Request) {
	s.handleIndexStatusImpl(w, r)
}
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	s.handleReadinessImpl(w, r)
}
