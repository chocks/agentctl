package policy

import (
	"encoding/json"
	"testing"

	"github.com/agentctl/agentctl/pkg/schema"
)

func TestEvaluateInstallPackageRequireLockfileAddsRisk(t *testing.T) {
	engine, err := LoadFromBytes([]byte(`
actions:
  install_package:
    require_hashes: true
    require_lockfile: true
`))
	if err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}

	params, err := json.Marshal(schema.InstallPackageParams{
		Manager: "npm",
		Package: "axios",
		Hash:    "sha256:abc",
		Pinned:  false,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	result := engine.Evaluate(schema.ActionRequest{
		Action: schema.ActionInstallPackage,
		Params: params,
		Reason: "test",
	})

	if result.Verdict != schema.VerdictAllow {
		t.Fatalf("expected allow, got %s", result.Verdict)
	}
	if result.RiskScore != 80 {
		t.Fatalf("expected risk 80, got %d", result.RiskScore)
	}
}

func TestEvaluateCallExternalAPIDerivesDomainFromURL(t *testing.T) {
	engine, err := LoadFromBytes([]byte(`
actions:
  call_external_api:
    allowed_domains:
      - "api.openai.com"
`))
	if err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}

	params, err := json.Marshal(schema.CallExternalAPIParams{
		URL:    "https://api.openai.com/v1/responses",
		Method: "POST",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	result := engine.Evaluate(schema.ActionRequest{
		Action: schema.ActionCallExternalAPI,
		Params: params,
		Reason: "test",
	})

	if result.Verdict != schema.VerdictAllow {
		t.Fatalf("expected allow, got %s (%s)", result.Verdict, result.Reason)
	}
}
