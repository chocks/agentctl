package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chocks/agentctl/pkg/gate"
	"github.com/chocks/agentctl/pkg/schema"
	"github.com/chocks/agentctl/pkg/trace"
)

type traceListResponse struct {
	Traces []schema.Decision `json:"traces"`
}

type replayRequest struct {
	SessionID  string `json:"session_id"`
	PolicyPath string `json:"policy_path"`
	Limit      int    `json:"limit"`
}

type replayResponse struct {
	SessionID string            `json:"session_id"`
	Policy    string            `json:"policy"`
	Results   []schema.Decision `json:"results"`
}

type approvalListResponse struct {
	Approvals []approvalRecord `json:"approvals"`
}

func cmdServe() {
	addr := stringFlagValue("--addr", "127.0.0.1:8080")
	server := newServer()
	if !isLoopbackListenAddr(addr) && server.authToken == "" {
		fmt.Fprintln(os.Stderr, "error: non-loopback listen addresses require AGENTCTL_AUTH_TOKEN or --auth-token")
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "agentctl server listening on http://%s\n", addr)
	if err := http.ListenAndServe(addr, server.routes()); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

type apiServer struct {
	traceFile  string
	policyFile string
	authToken  string
}

func newServer() *apiServer {
	return &apiServer{
		traceFile:  traceFilePath(),
		policyFile: defaultPolicyFile,
		authToken:  authTokenValue(),
	}
}

func (s *apiServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.Handle("/ui/", http.StripPrefix("/ui/", uiHandler()))
	mux.HandleFunc("/ui", serveUIIndex)
	mux.HandleFunc("/v1/gate", s.handleGate)
	mux.HandleFunc("/v1/traces", s.handleTraces)
	mux.HandleFunc("/v1/replay", s.handleReplay)
	mux.HandleFunc("/v1/approvals", s.handleApprovals)
	mux.HandleFunc("/v1/approvals/", s.handleApprovalResolution)
	return s.withAuth(mux)
}

func (s *apiServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *apiServer) handleGate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req schema.ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	now := time.Now()
	if req.Context.SessionID == "" {
		req.Context.SessionID = fmt.Sprintf("http_%d", now.UnixMilli())
	}
	if req.Context.Agent == "" {
		req.Context.Agent = "agentctl-http"
	}
	if req.Context.Actor == "" {
		req.Context.Actor = r.Header.Get("X-Agentctl-Actor")
	}
	if req.Context.Team == "" {
		req.Context.Team = r.Header.Get("X-Agentctl-Team")
	}
	if req.Context.Timestamp.IsZero() {
		req.Context.Timestamp = now
	}

	ensureDir(filepath.Dir(s.traceFile))

	tracer, err := trace.NewFileStore(s.traceFile)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("opening trace store: %v", err))
		return
	}

	g := gate.New(loadPolicyFromPath(s.policyFile), tracer)
	decision, err := g.Evaluate(req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("evaluating request: %v", err))
		return
	}
	if err := recordApprovalForDecision(approvalFilePath(), decision); err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("recording approval: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, decision)
}

func (s *apiServer) handleTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	filter, err := traceFilterFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	traces, err := trace.ReadTraces(s.traceFile, filter)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("reading traces: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, traceListResponse{Traces: traces})
}

func (s *apiServer) handleReplay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req replayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if req.SessionID == "" {
		writeJSONError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	policyPath := req.PolicyPath
	if policyPath == "" {
		policyPath = s.policyFile
	}

	results, err := replaySession(policyPath, s.traceFile, req.SessionID, req.Limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, replayResponse{
		SessionID: req.SessionID,
		Policy:    policyPath,
		Results:   results,
	})
}

func (s *apiServer) handleApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	status := approvalStatus(r.URL.Query().Get("status"))
	records, err := readApprovals(approvalFilePath(), status)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("reading approvals: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, approvalListResponse{Approvals: records})
}

func (s *apiServer) handleApprovalResolution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/v1/approvals/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" {
		writeJSONError(w, http.StatusNotFound, "approval route not found")
		return
	}

	var status approvalStatus
	switch parts[1] {
	case "approve":
		status = approvalStatusApproved
	case "deny":
		status = approvalStatusDenied
	default:
		writeJSONError(w, http.StatusNotFound, "approval route not found")
		return
	}

	resolvedBy := r.Header.Get("X-Agentctl-Actor")
	if resolvedBy == "" {
		resolvedBy = "api-operator"
	}

	record, err := resolveApproval(approvalFilePath(), parts[0], status, resolvedBy)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func replaySession(policyPath, traceFile, sessionID string, limit int) ([]schema.Decision, error) {
	pol := loadPolicyFromPath(policyPath)
	traces, err := trace.ReadTraces(traceFile, trace.TraceFilter{
		SessionID: sessionID,
		Limit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("reading traces: %w", err)
	}
	if len(traces) == 0 {
		return nil, fmt.Errorf("no traces found for session %q", sessionID)
	}

	sort.Slice(traces, func(i, j int) bool {
		return traces[i].Timestamp.Before(traces[j].Timestamp)
	})

	results := make([]schema.Decision, 0, len(traces))
	for _, prior := range traces {
		result := pol.Evaluate(prior.Request)
		results = append(results, schema.Decision{
			TraceID:        prior.TraceID,
			Verdict:        result.Verdict,
			RiskScore:      result.RiskScore,
			Timestamp:      time.Now(),
			Request:        prior.Request,
			Reason:         result.Reason,
			MatchedRules:   result.MatchedRules,
			EvalDurationMs: 0,
		})
	}

	return results, nil
}

func traceFilterFromRequest(r *http.Request) (trace.TraceFilter, error) {
	query := r.URL.Query()
	filter := trace.TraceFilter{
		SessionID: query.Get("session_id"),
		Action:    schema.Action(query.Get("action")),
		Verdict:   schema.Verdict(query.Get("verdict")),
		Package:   query.Get("package"),
	}

	if since := query.Get("since"); since != "" {
		parsed, err := time.Parse(time.RFC3339, since)
		if err != nil {
			return trace.TraceFilter{}, fmt.Errorf("invalid since value %q", since)
		}
		filter.Since = parsed
	}
	if until := query.Get("until"); until != "" {
		parsed, err := time.Parse(time.RFC3339, until)
		if err != nil {
			return trace.TraceFilter{}, fmt.Errorf("invalid until value %q", until)
		}
		filter.Until = parsed
	}
	if limit := query.Get("limit"); limit != "" {
		if _, err := fmt.Sscanf(limit, "%d", &filter.Limit); err != nil {
			return trace.TraceFilter{}, fmt.Errorf("invalid limit value %q", limit)
		}
	}

	return filter, nil
}

func writeMethodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *apiServer) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authToken == "" || r.URL.Path == "/healthz" || r.URL.Path == "/ui" || strings.HasPrefix(r.URL.Path, "/ui/") {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		expected := "Bearer " + s.authToken
		if authHeader != expected {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authTokenValue() string {
	if token := stringFlagValue("--auth-token", ""); token != "" {
		return token
	}
	return os.Getenv("AGENTCTL_AUTH_TOKEN")
}

func isLoopbackListenAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "" || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
