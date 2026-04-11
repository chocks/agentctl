package gate

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/chocks/agentctl/pkg/policy"
	"github.com/chocks/agentctl/pkg/schema"
	"github.com/chocks/agentctl/pkg/trace"
)

func TestEvaluateInvalidRequestDeniesAndRecordsTrace(t *testing.T) {
	var buf bytes.Buffer
	g := New(policy.DefaultEngine(), trace.NewWriterStore(&buf))

	params, err := json.Marshal(schema.InstallPackageParams{
		Manager: "pip",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	decision, err := g.Evaluate(schema.ActionRequest{
		Action: schema.ActionInstallPackage,
		Params: params,
		Reason: "test invalid install",
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if decision.Verdict != schema.VerdictDeny {
		t.Fatalf("expected deny, got %s", decision.Verdict)
	}
	if !strings.Contains(decision.Reason, "package name required") {
		t.Fatalf("expected validation reason, got %q", decision.Reason)
	}
	if buf.Len() == 0 {
		t.Fatal("expected trace to be recorded")
	}
}

func TestEvaluateEscalationSetsApprovalRequired(t *testing.T) {
	g := New(policy.DefaultEngine(), nil)

	params, err := json.Marshal(schema.AccessSecretParams{
		Name: "OPENAI_API_KEY",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	decision, err := g.Evaluate(schema.ActionRequest{
		Action: schema.ActionAccessSecret,
		Params: params,
		Reason: "need provider credentials",
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if decision.Verdict != schema.VerdictEscalate {
		t.Fatalf("expected escalate, got %s", decision.Verdict)
	}
	if !decision.ApprovalRequired {
		t.Fatal("expected approval_required to be true")
	}
}
