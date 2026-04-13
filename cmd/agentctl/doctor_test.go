package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chocks/agentctl/pkg/config"
	"github.com/chocks/agentctl/pkg/schema"
)

func TestRunDoctorFreshInstall(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	paths := config.NewPaths(filepath.Join(tempHome, ".agentctl"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runDoctor(paths, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runDoctor() code = %d, want 0\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}

	output := stdout.String()
	for _, want := range []string{
		"agentctl ",
		paths.Policy,
		"warn  run \"agentctl attach\" to set up",
		paths.Traces,
		"ok    (0 traces)",
		paths.Approvals,
		"ok    (0 pending)",
		"Agents:",
		"claude-code",
		"codex",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, output)
		}
	}
}

func TestRunDoctorMalformedPolicy(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	paths := config.NewPaths(filepath.Join(tempHome, ".agentctl"))
	if err := paths.EnsureHome(); err != nil {
		t.Fatalf("EnsureHome() error = %v", err)
	}
	if err := os.WriteFile(paths.Policy, []byte("not: [valid: yaml: {{"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runDoctor(paths, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("runDoctor() code = %d, want 1\nstdout=%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "error parse error:") {
		t.Fatalf("doctor output missing parse error:\n%s", stdout.String())
	}
}

func TestRunDoctorCountsExistingFiles(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	paths := config.NewPaths(filepath.Join(tempHome, ".agentctl"))
	if err := paths.EnsureHome(); err != nil {
		t.Fatalf("EnsureHome() error = %v", err)
	}
	if err := os.WriteFile(paths.Policy, []byte("actions:\n  run_code: {}\n  access_secret: {}\n"), 0644); err != nil {
		t.Fatalf("WriteFile(policy) error = %v", err)
	}

	decision := &schema.Decision{
		TraceID:   "trace-1",
		Verdict:   schema.VerdictEscalate,
		Timestamp: time.Now(),
		Reason:    "needs approval",
		Request: schema.ActionRequest{
			Action: schema.ActionAccessSecret,
			Context: schema.RequestContext{
				SessionID: "session-1",
			},
		},
		ApprovalRequired: true,
	}
	if err := os.WriteFile(paths.Traces, []byte(`{"trace_id":"trace-1","verdict":"allow","risk_score":10,"timestamp":"2026-04-12T12:00:00Z","request":{"action":"run_code","params":{"command":"ls"},"reason":"test","context":{"session_id":"session-1","timestamp":"2026-04-12T12:00:00Z"}},"reason":"ok","eval_duration_ms":1}`+"\n"), 0644); err != nil {
		t.Fatalf("WriteFile(traces) error = %v", err)
	}
	if err := recordApprovalForDecision(paths.Approvals, decision); err != nil {
		t.Fatalf("recordApprovalForDecision() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runDoctor(paths, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runDoctor() code = %d, want 0\nstdout=%s", code, stdout.String())
	}

	output := stdout.String()
	for _, want := range []string{
		"ok    (2 actions configured)",
		"ok    (1 traces)",
		"ok    (1 pending)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, output)
		}
	}
}
