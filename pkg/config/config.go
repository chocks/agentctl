// Package config resolves file paths and loads policy for agentctl.
//
// All agentctl state lives under a single home directory (~/.agentctl/).
// The Paths struct is constructed once at startup and threaded to all callers.
// No mutable global state.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chocks/agentctl/pkg/policy"
)

// Paths holds resolved file paths for all agentctl state.
// Constructed once, passed to all callers. Tests use NewPaths(t.TempDir()).
type Paths struct {
	Home      string // e.g. ~/.agentctl/
	Policy    string // e.g. ~/.agentctl/policy.yaml
	Traces    string // e.g. ~/.agentctl/traces.jsonl
	Approvals string // e.g. ~/.agentctl/approvals.jsonl
}

// DefaultPaths returns paths rooted at ~/.agentctl/.
func DefaultPaths() (Paths, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolving user home directory: %w", err)
	}
	return NewPaths(filepath.Join(homeDir, ".agentctl")), nil
}

// NewPaths returns paths rooted at an arbitrary directory.
func NewPaths(home string) Paths {
	return Paths{
		Home:      home,
		Policy:    filepath.Join(home, "policy.yaml"),
		Traces:    filepath.Join(home, "traces.jsonl"),
		Approvals: filepath.Join(home, "approvals.jsonl"),
	}
}

// EnsureHome creates the home directory if it doesn't exist.
func (p Paths) EnsureHome() error {
	if err := os.MkdirAll(p.Home, 0755); err != nil {
		return fmt.Errorf("creating agentctl home %s: %w", p.Home, err)
	}
	return nil
}

// LoadPolicy loads the policy engine from p.Policy.
// If the file doesn't exist, returns the default engine with a nil error.
// If the file exists but is malformed, returns an error.
func (p Paths) LoadPolicy() (*policy.Engine, error) {
	_, err := os.Stat(p.Policy)
	if errors.Is(err, os.ErrNotExist) {
		return policy.DefaultEngine(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("checking policy file: %w", err)
	}

	engine, err := policy.LoadFromFile(p.Policy)
	if err != nil {
		return nil, fmt.Errorf("loading policy from %s: %w", p.Policy, err)
	}
	return engine, nil
}

// WriteDefaultPolicy writes the default policy file if it doesn't already exist.
// If the file exists (even if empty), it is not overwritten.
func (p Paths) WriteDefaultPolicy() error {
	if _, err := os.Stat(p.Policy); err == nil {
		return nil // already exists
	}

	content := `# agentctl policy — edit to customize enforcement rules
# Docs: https://github.com/chocks/agentctl

actions:
  install_package:
    require_hashes: true

  run_code:
    block_patterns:
      - "| bash"
      - "| sh"
      - "| python"
    network: deny

  access_secret:
    require_approval: always
    max_ttl: 300                 # ttl > max_ttl = deny

  write_file:
    block_patterns:
      - ".env"
      - "*.pem"
      - "*.key"

  call_external_api:
    allowed_domains: []          # empty list = deny all external calls
`
	if err := os.WriteFile(p.Policy, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing default policy: %w", err)
	}
	return nil
}
