package api

import (
	"encoding/json"
	"net/http"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/migration"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/target"
)

func (s *Server) handleGetStateImpl(w http.ResponseWriter, r *http.Request) {
	st, err := s.engine.LoadState()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := StateResponse{
		CurrentStep: string(st.CurrentStep),
		Steps:       make(map[string]StepStateResponse),
		LastUpdated: st.LastUpdated.Format("2006-01-02T15:04:05Z"),
	}
	for step, ss := range st.Steps {
		r := StepStateResponse{Status: ss.Status}
		if !ss.CompletedAt.IsZero() {
			r.CompletedAt = ss.CompletedAt.Format("2006-01-02T15:04:05Z")
		}
		resp.Steps[string(step)] = r
	}

	jsonResponse(w, http.StatusOK, resp)
}

func (s *Server) handleSetStepImpl(w http.ResponseWriter, r *http.Request) {
	var req SetStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.engine.NavigateToStep(state.Step(req.Step)); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetSourceConfigImpl(w http.ResponseWriter, r *http.Request) {
	cfg := s.engine.Config
	if cfg == nil {
		jsonResponse(w, http.StatusOK, SourceConfigRequest{})
		return
	}
	jsonResponse(w, http.StatusOK, SourceConfigRequest{
		Type:     cfg.Source.Type,
		Host:     cfg.Source.Host,
		Port:     cfg.Source.Port,
		Database: cfg.Source.Database,
		Schema:   cfg.Source.Schema,
		Username: cfg.Source.Username,
		Password: cfg.Source.Password,
		SSL:      cfg.Source.SSL,
	})
}

func (s *Server) handleTestSourceConnectionImpl(w http.ResponseWriter, r *http.Request) {
	var req SourceConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cfg := req.toSourceConfig()
	err := s.engine.TestSourceConnection(r.Context(), &cfg)
	if err != nil {
		jsonResponse(w, http.StatusOK, ConnectionTestResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	jsonResponse(w, http.StatusOK, ConnectionTestResponse{
		Success: true,
		Message: "Connection successful",
	})
}

func (s *Server) handleDiscoverImpl(w http.ResponseWriter, r *http.Request) {
	var req SourceConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cfg := req.toSourceConfig()
	s.engine.SetSourceConfig(&cfg)

	sch, err := s.engine.Discover(r.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Mark source_connection as complete
	s.engine.CompleteCurrentStep()

	jsonResponse(w, http.StatusOK, sch)
}

func (s *Server) handleGetSchemaImpl(w http.ResponseWriter, r *http.Request) {
	sch := s.engine.GetSchema()
	if sch == nil {
		errorResponse(w, http.StatusNotFound, "no schema discovered yet")
		return
	}
	jsonResponse(w, http.StatusOK, sch)
}

func (s *Server) handleGetTargetConfigImpl(w http.ResponseWriter, r *http.Request) {
	cfg := s.engine.Config
	if cfg == nil || cfg.Target.ConnectionString == "" {
		jsonResponse(w, http.StatusOK, TargetConfigRequest{})
		return
	}
	jsonResponse(w, http.StatusOK, TargetConfigRequest{
		ConnectionString: cfg.Target.ConnectionString,
		Database:         cfg.Target.Database,
	})
}

func (s *Server) handleTestTargetConnectionImpl(w http.ResponseWriter, r *http.Request) {
	var req TargetConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cfg := req.toTargetConfig()
	err := s.engine.TestTargetConnection(r.Context(), &cfg)
	if err != nil {
		jsonResponse(w, http.StatusOK, ConnectionTestResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	jsonResponse(w, http.StatusOK, ConnectionTestResponse{
		Success: true,
		Message: "Connection successful",
	})
}

func (s *Server) handleDetectTopologyImpl(w http.ResponseWriter, r *http.Request) {
	var req TargetConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cfg := req.toTargetConfig()
	topo, err := s.engine.DetectTopology(r.Context(), &cfg)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, TopologyResponse{
		Type:          topo.Type,
		IsAtlas:       topo.IsAtlas,
		ShardCount:    topo.ShardCount,
		ServerVersion: topo.ServerVersion,
	})
}

func (s *Server) handleGetTablesImpl(w http.ResponseWriter, r *http.Request) {
	sch := s.engine.GetSchema()
	if sch == nil {
		errorResponse(w, http.StatusNotFound, "no schema discovered yet")
		return
	}

	st, err := s.engine.LoadState()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	type tableInfo struct {
		Name      string `json:"name"`
		RowCount  int64  `json:"row_count"`
		SizeBytes int64  `json:"size_bytes"`
		Selected  bool   `json:"selected"`
	}

	selectedMap := make(map[string]bool)
	for _, t := range st.SelectedTables {
		selectedMap[t] = true
	}

	tables := make([]tableInfo, len(sch.Tables))
	for i, t := range sch.Tables {
		tables[i] = tableInfo{
			Name:      t.Name,
			RowCount:  t.RowCount,
			SizeBytes: t.SizeBytes,
			Selected:  selectedMap[t.Name],
		}
	}

	jsonResponse(w, http.StatusOK, tables)
}

func (s *Server) handleSelectTablesImpl(w http.ResponseWriter, r *http.Request) {
	var req SelectTablesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.engine.SelectTables(req.Tables); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetMappingImpl(w http.ResponseWriter, r *http.Request) {
	m := s.engine.GetMapping()
	if m == nil {
		errorResponse(w, http.StatusNotFound, "no mapping defined yet")
		return
	}
	jsonResponse(w, http.StatusOK, m)
}

func (s *Server) handleSaveMappingImpl(w http.ResponseWriter, r *http.Request) {
	var m map[string]any
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Re-encode and pass through to engine
	data, _ := json.Marshal(m)
	if err := s.engine.SaveMappingJSON(data); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetTypeMapImpl(w http.ResponseWriter, r *http.Request) {
	tm := s.engine.GetTypeMap()
	if tm == nil {
		errorResponse(w, http.StatusNotFound, "no type map available")
		return
	}

	type typeMapEntry struct {
		SourceType string `json:"source_type"`
		BSONType   string `json:"bson_type"`
		Overridden bool   `json:"overridden"`
	}

	entries := make([]typeMapEntry, 0)
	for _, st := range tm.SortedTypes() {
		entries = append(entries, typeMapEntry{
			SourceType: st,
			BSONType:   string(tm.Resolve(st)),
			Overridden: tm.IsOverridden(st),
		})
	}

	jsonResponse(w, http.StatusOK, entries)
}

func (s *Server) handleSaveTypeMapImpl(w http.ResponseWriter, r *http.Request) {
	var overrides map[string]string
	if err := json.NewDecoder(r.Body).Decode(&overrides); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.engine.SaveTypeMapOverrides(overrides); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetSizingImpl(w http.ResponseWriter, r *http.Request) {
	plan, err := s.engine.ComputeSizing()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, plan)
}

func (s *Server) handleRunBenchmarkImpl(w http.ResponseWriter, r *http.Request) {
	var req BenchmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Table == "" {
		errorResponse(w, http.StatusBadRequest, "table is required")
		return
	}

	result, err := s.engine.RunBenchmark(r.Context(), req.Table, req.PartitionCol)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleConfigureAWSImpl(w http.ResponseWriter, r *http.Request) {
	var req AWSConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cfg := config.AWSConfig{
		Region:   req.Region,
		Profile:  req.Profile,
		S3Bucket: req.S3Bucket,
		Platform: req.Platform,
	}

	if err := s.engine.SaveAWSConfig(&cfg); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleValidateAWSImpl(w http.ResponseWriter, r *http.Request) {
	result, err := s.engine.ValidateAWS(r.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handlePreMigrationPrepareImpl(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.PreMigrationPrepare(r.Context()); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handlePreMigrationStatusImpl(w http.ResponseWriter, r *http.Request) {
	result := s.engine.PreMigrationStatus()
	jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleStartMigrationImpl(w http.ResponseWriter, r *http.Request) {
	callback := func(status *migration.Status) {
		if s.hub != nil {
			s.hub.BroadcastMigrationProgress(status)
		}
	}

	if err := s.engine.StartMigration(r.Context(), callback); err != nil {
		errorResponse(w, http.StatusConflict, err.Error())
		return
	}

	jsonResponse(w, http.StatusAccepted, AsyncAcceptedResponse{
		Status:  "accepted",
		Message: "Migration started",
	})
}

func (s *Server) handleMigrationStatusImpl(w http.ResponseWriter, r *http.Request) {
	status := s.engine.MigrationStatus()
	jsonResponse(w, http.StatusOK, status)
}

func (s *Server) handleRetryMigrationImpl(w http.ResponseWriter, r *http.Request) {
	var req RetryMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	callback := func(status *migration.Status) {
		if s.hub != nil {
			s.hub.BroadcastMigrationProgress(status)
		}
	}

	if err := s.engine.RetryMigration(r.Context(), req.Collections, callback); err != nil {
		errorResponse(w, http.StatusConflict, err.Error())
		return
	}

	jsonResponse(w, http.StatusAccepted, AsyncAcceptedResponse{
		Status:  "accepted",
		Message: "Migration retry started",
	})
}

func (s *Server) handleAbortMigrationImpl(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.AbortMigration(); err != nil {
		errorResponse(w, http.StatusConflict, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "aborted"})
}

func (s *Server) handleRunValidationImpl(w http.ResponseWriter, r *http.Request) {
	callback := func(collection, checkType string, passed bool) {
		if s.hub != nil {
			s.hub.BroadcastValidationCheck(map[string]any{
				"collection": collection,
				"check_type": checkType,
				"passed":     passed,
			})
		}
	}

	if err := s.engine.RunValidation(r.Context(), callback); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusAccepted, AsyncAcceptedResponse{
		Status:  "accepted",
		Message: "Validation started",
	})
}

func (s *Server) handleValidationResultsImpl(w http.ResponseWriter, r *http.Request) {
	result := s.engine.ValidationResults()
	if result == nil {
		errorResponse(w, http.StatusNotFound, "no validation results available")
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleGetIndexPlanImpl(w http.ResponseWriter, r *http.Request) {
	plan, err := s.engine.GetIndexPlan()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, plan)
}

func (s *Server) handleBuildIndexesImpl(w http.ResponseWriter, r *http.Request) {
	callback := func(status []target.IndexBuildStatus) {
		if s.hub != nil {
			s.hub.BroadcastIndexProgress(status)
		}
	}

	if err := s.engine.BuildIndexes(r.Context(), callback); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusAccepted, AsyncAcceptedResponse{
		Status:  "accepted",
		Message: "Index build started",
	})
}

func (s *Server) handleIndexStatusImpl(w http.ResponseWriter, r *http.Request) {
	result, err := s.engine.IndexBuildStatus()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleReadinessImpl(w http.ResponseWriter, r *http.Request) {
	rpt, err := s.engine.CheckReadiness(r.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, rpt)
}

func (s *Server) handleGetMappingPreviewImpl(w http.ResponseWriter, r *http.Request) {
	m, err := s.engine.PreviewMapping()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, m)
}

func (s *Server) handleGetSizeEstimateImpl(w http.ResponseWriter, r *http.Request) {
	estimates, err := s.engine.MappingSizeEstimate()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, estimates)
}
