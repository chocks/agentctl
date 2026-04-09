package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"
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

func TestApprovalWaitEndpoint(t *testing.T) {
	t.Run("returns 404 for unknown approval id", func(t *testing.T) {
		approvalFile := filepath.Join(t.TempDir(), "approvals.jsonl")
		t.Setenv("AGENTCTL_APPROVAL_FILE", approvalFile)
		server := &apiServer{authToken: ""}

		req := httptest.NewRequest(http.MethodGet, "/v1/approvals/no-such-id/wait?timeout=1s", nil)
		rec := httptest.NewRecorder()
		server.routes().ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("returns 408 when approval stays pending past timeout", func(t *testing.T) {
		approvalFile := filepath.Join(t.TempDir(), "approvals.jsonl")
		t.Setenv("AGENTCTL_APPROVAL_FILE", approvalFile)

		// Write a pending approval record directly.
		rec := approvalRecord{
			ApprovalID:  "wait-test-1",
			TraceID:     "wait-test-1",
			SessionID:   "s1",
			Action:      "access_secret",
			Status:      approvalStatusPending,
			Reason:      "needs review",
			RequestedAt: time.Now(),
		}
		if err := appendApproval(approvalFile, rec); err != nil {
			t.Fatalf("appendApproval: %v", err)
		}

		server := &apiServer{authToken: ""}
		httpReq := httptest.NewRequest(http.MethodGet, "/v1/approvals/wait-test-1/wait?timeout=2s", nil)
		httpRec := httptest.NewRecorder()
		server.routes().ServeHTTP(httpRec, httpReq)

		if httpRec.Code != http.StatusRequestTimeout {
			t.Fatalf("expected 408, got %d body=%s", httpRec.Code, httpRec.Body.String())
		}
		var body approvalRecord
		if err := json.Unmarshal(httpRec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body.Status != approvalStatusPending {
			t.Fatalf("expected pending, got %q", body.Status)
		}
	})

	t.Run("returns 200 when approval is resolved while waiting", func(t *testing.T) {
		approvalFile := filepath.Join(t.TempDir(), "approvals.jsonl")
		t.Setenv("AGENTCTL_APPROVAL_FILE", approvalFile)

		rec := approvalRecord{
			ApprovalID:  "wait-test-2",
			TraceID:     "wait-test-2",
			SessionID:   "s2",
			Action:      "access_secret",
			Status:      approvalStatusPending,
			Reason:      "needs review",
			RequestedAt: time.Now(),
		}
		if err := appendApproval(approvalFile, rec); err != nil {
			t.Fatalf("appendApproval: %v", err)
		}

		// Approve asynchronously after a short delay.
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(1500 * time.Millisecond)
			if _, err := resolveApproval(approvalFile, "wait-test-2", approvalStatusApproved, "alice"); err != nil {
				fmt.Printf("resolveApproval error: %v\n", err)
			}
		}()

		server := &apiServer{authToken: ""}
		httpReq := httptest.NewRequest(http.MethodGet, "/v1/approvals/wait-test-2/wait?timeout=10s", nil)
		httpRec := httptest.NewRecorder()
		server.routes().ServeHTTP(httpRec, httpReq)
		wg.Wait()

		if httpRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", httpRec.Code, httpRec.Body.String())
		}
		var body approvalRecord
		if err := json.Unmarshal(httpRec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body.Status != approvalStatusApproved {
			t.Fatalf("expected approved, got %q", body.Status)
		}
		if body.ResolvedBy != "alice" {
			t.Fatalf("expected resolved_by=alice, got %q", body.ResolvedBy)
		}
	})
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
