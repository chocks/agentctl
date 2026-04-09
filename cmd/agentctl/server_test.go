package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestServerGateTraceReplay(t *testing.T) {
	policyPath := filepath.Join(t.TempDir(), "agentctl.policy.yaml")
	server := &apiServer{
		traceFile:  filepath.Join(t.TempDir(), "traces.jsonl"),
		policyFile: policyPath,
		authToken:  "secret-token",
	}

	writeFile(t, policyPath, `
actions:
  access_secret:
    require_approval: always
  call_external_api:
    allowed_domains:
      - "api.openai.com"
`)
	t.Setenv("AGENTCTL_TRACE_FILE", server.traceFile)
	t.Setenv("AGENTCTL_APPROVAL_FILE", t.TempDir()+"/approvals.jsonl")

	gateBody := bytes.NewBufferString(`{"action":"call_external_api","params":{"url":"https://api.openai.com/v1/responses","method":"POST"},"reason":"call provider","context":{"session_id":"srv-1"}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/gate", gateBody)
	rec := httptest.NewRecorder()
	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without bearer token, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/gate", gateBody)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("X-Agentctl-Actor", "alice")
	req.Header.Set("X-Agentctl-Team", "platform")
	rec = httptest.NewRecorder()
	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("gate status=%d body=%s", rec.Code, rec.Body.String())
	}

	var decision struct {
		Verdict string `json:"verdict"`
		Request struct {
			Context struct {
				SessionID string `json:"session_id"`
				Actor     string `json:"actor"`
				Team      string `json:"team"`
			} `json:"context"`
		} `json:"request"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &decision); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decision.Verdict != "allow" {
		t.Fatalf("expected allow, got %q", decision.Verdict)
	}
	if decision.Request.Context.SessionID != "srv-1" {
		t.Fatalf("expected session id srv-1, got %q", decision.Request.Context.SessionID)
	}
	if decision.Request.Context.Actor != "alice" || decision.Request.Context.Team != "platform" {
		t.Fatalf("expected actor/team alice/platform, got %q/%q", decision.Request.Context.Actor, decision.Request.Context.Team)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/approvals?status=pending", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("approvals status=%d body=%s", rec.Code, rec.Body.String())
	}

	var approvals struct {
		Approvals []json.RawMessage `json:"approvals"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &approvals); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(approvals.Approvals) != 0 {
		t.Fatalf("expected 0 approvals for allow flow, got %d", len(approvals.Approvals))
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/traces?session_id=srv-1", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("traces status=%d body=%s", rec.Code, rec.Body.String())
	}

	var traces struct {
		Traces []json.RawMessage `json:"traces"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &traces); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(traces.Traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces.Traces))
	}

	replayPath := filepath.Join(t.TempDir(), "replay.policy.yaml")
	writeFile(t, replayPath, `
actions:
  call_external_api:
    blocked_domains:
      - "api.openai.com"
`)

	replayBody := bytes.NewBufferString(`{"session_id":"srv-1","policy_path":"` + replayPath + `"}`)
	req = httptest.NewRequest(http.MethodPost, "/v1/replay", replayBody)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("replay status=%d body=%s", rec.Code, rec.Body.String())
	}

	var replay struct {
		Results []struct {
			Verdict string `json:"verdict"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &replay); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(replay.Results) != 1 || replay.Results[0].Verdict != "deny" {
		t.Fatalf("expected replay deny result, got %+v", replay.Results)
	}
}

func TestServerUIIsServedWithoutAuth(t *testing.T) {
	server := &apiServer{authToken: "secret-token"}

	req := httptest.NewRequest(http.MethodGet, "/ui", nil)
	rec := httptest.NewRecorder()
	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ui status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("agentctl")) {
		t.Fatalf("expected ui body to contain agentctl, got %q", rec.Body.String())
	}
}
