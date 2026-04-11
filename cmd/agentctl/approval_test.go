package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/chocks/agentctl/pkg/schema"
)

func TestRecordAndResolveApproval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "approvals.jsonl")
	decision := &schema.Decision{
		TraceID:          "trace-1",
		Timestamp:        time.Now(),
		Reason:           "needs approval",
		ApprovalRequired: true,
		Request: schema.ActionRequest{
			Action: schema.ActionAccessSecret,
			Context: schema.RequestContext{
				SessionID: "session-1",
			},
		},
	}

	if err := recordApprovalForDecision(path, decision); err != nil {
		t.Fatalf("recordApprovalForDecision() error = %v", err)
	}

	records, err := readApprovals(path, approvalStatusPending)
	if err != nil {
		t.Fatalf("readApprovals() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 pending approval, got %d", len(records))
	}

	resolved, err := resolveApproval(path, "trace-1", approvalStatusApproved, "alice")
	if err != nil {
		t.Fatalf("resolveApproval() error = %v", err)
	}
	if resolved.Status != approvalStatusApproved {
		t.Fatalf("expected approved status, got %s", resolved.Status)
	}
	if resolved.ResolvedBy != "alice" {
		t.Fatalf("expected resolved_by alice, got %q", resolved.ResolvedBy)
	}
}
