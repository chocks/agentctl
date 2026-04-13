package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPaths(t *testing.T) {
	home := "/tmp/test-agentctl"
	p := NewPaths(home)

	if p.Home != home {
		t.Errorf("Home = %q, want %q", p.Home, home)
	}
	if p.Policy != filepath.Join(home, "policy.yaml") {
		t.Errorf("Policy = %q, want %q", p.Policy, filepath.Join(home, "policy.yaml"))
	}
	if p.Traces != filepath.Join(home, "traces.jsonl") {
		t.Errorf("Traces = %q, want %q", p.Traces, filepath.Join(home, "traces.jsonl"))
	}
	if p.Approvals != filepath.Join(home, "approvals.jsonl") {
		t.Errorf("Approvals = %q, want %q", p.Approvals, filepath.Join(home, "approvals.jsonl"))
	}
}

func TestDefaultPaths(t *testing.T) {
	p, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths() error = %v", err)
	}
	homeDir, _ := os.UserHomeDir()
	want := filepath.Join(homeDir, ".agentctl")
	if p.Home != want {
		t.Errorf("Home = %q, want %q", p.Home, want)
	}
}

func TestEnsureHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), ".agentctl")
	p := NewPaths(home)

	if err := p.EnsureHome(); err != nil {
		t.Fatalf("EnsureHome() error = %v", err)
	}

	info, err := os.Stat(home)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v", home, err)
	}
	if !info.IsDir() {
		t.Errorf("%q is not a directory", home)
	}

	// Idempotent
	if err := p.EnsureHome(); err != nil {
		t.Fatalf("second EnsureHome() error = %v", err)
	}
}

func TestLoadPolicy_MissingFile(t *testing.T) {
	p := NewPaths(t.TempDir())
	engine, err := p.LoadPolicy()
	if err != nil {
		t.Fatalf("LoadPolicy() error = %v, want nil (fallback to defaults)", err)
	}
	if engine == nil {
		t.Fatal("LoadPolicy() returned nil engine")
	}
}

func TestLoadPolicy_ValidFile(t *testing.T) {
	home := t.TempDir()
	p := NewPaths(home)

	policyContent := `actions:
  run_code:
    block_patterns:
      - "rm -rf /"
`
	if err := os.WriteFile(p.Policy, []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	engine, err := p.LoadPolicy()
	if err != nil {
		t.Fatalf("LoadPolicy() error = %v", err)
	}
	if engine == nil {
		t.Fatal("LoadPolicy() returned nil engine")
	}
}

func TestLoadPolicy_MalformedFile(t *testing.T) {
	home := t.TempDir()
	p := NewPaths(home)

	if err := os.WriteFile(p.Policy, []byte("not: [valid: yaml: {{"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := p.LoadPolicy()
	if err == nil {
		t.Fatal("LoadPolicy() error = nil, want error for malformed YAML")
	}
}

func TestWriteDefaultPolicy(t *testing.T) {
	home := t.TempDir()
	p := NewPaths(home)

	if err := p.WriteDefaultPolicy(); err != nil {
		t.Fatalf("WriteDefaultPolicy() error = %v", err)
	}

	// File should exist and be loadable
	engine, err := p.LoadPolicy()
	if err != nil {
		t.Fatalf("LoadPolicy() after WriteDefaultPolicy() error = %v", err)
	}
	if engine == nil {
		t.Fatal("engine is nil")
	}

	// Idempotent — should not overwrite
	if err := os.WriteFile(p.Policy, []byte("actions: {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := p.WriteDefaultPolicy(); err != nil {
		t.Fatalf("second WriteDefaultPolicy() error = %v", err)
	}
	data, _ := os.ReadFile(p.Policy)
	if string(data) != "actions: {}" {
		t.Error("WriteDefaultPolicy() overwrote existing file")
	}
}
