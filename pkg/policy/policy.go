// Package policy implements the YAML-based policy engine for agentctl.
//
// Design decisions:
//   - YAML config, not Rego. Lower barrier. Rego can come in v2.
//   - Rules are evaluated top-to-bottom, first match wins.
//   - Risk scores are additive from base action risk + rule modifiers.
//   - The engine is stateless — no side effects during evaluation.
//   - Designed to evaluate in <1ms for typical policy files.
package policy

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/chocks/agentctl/pkg/schema"
	"gopkg.in/yaml.v3"
)

// Config is the top-level policy configuration.
// Lives in agentctl.policy.yaml in the user's repo.
type Config struct {
	Actions       map[string]ActionPolicy `yaml:"actions"`
	Notifications *NotificationConfig     `yaml:"notifications,omitempty"`
	Trust         *TrustConfig            `yaml:"trust,omitempty"`
}

type ActionPolicy struct {
	// Package install specifics
	RequireHashes             bool `yaml:"require_hashes,omitempty"`
	RequireLockfile           bool `yaml:"require_lockfile,omitempty"`
	MaxPublishAgeHours        int  `yaml:"max_publish_age_hours,omitempty"`
	BlockMaintainerChangeDays int  `yaml:"block_maintainer_change_days,omitempty"`

	// URL/domain controls
	AllowedDomains []string `yaml:"allowed_domains,omitempty"`
	BlockedDomains []string `yaml:"blocked_domains,omitempty"`

	// Code execution controls
	BlockPatterns []string `yaml:"block_patterns,omitempty"`
	Network       string   `yaml:"network,omitempty"` // "allow" or "deny"

	// Approval
	RequireApproval string `yaml:"require_approval,omitempty"` // "always", "high_risk", "never"
	MaxTTL          int    `yaml:"max_ttl,omitempty"`          // seconds, for secrets

	// Override risk
	BaseRisk *int `yaml:"base_risk,omitempty"` // override default
}

type NotificationConfig struct {
	Escalation []NotifyTarget `yaml:"escalation,omitempty"`
	Denial     []NotifyTarget `yaml:"denial,omitempty"`
}

type NotifyTarget struct {
	Slack   string `yaml:"slack,omitempty"`
	Webhook string `yaml:"webhook,omitempty"`
}

type TrustConfig struct {
	Initial    string         `yaml:"initial,omitempty"` // "new", "standard", "elevated"
	Thresholds map[string]int `yaml:"thresholds,omitempty"`
}

// EvalResult is the output of policy evaluation.
type EvalResult struct {
	Verdict      schema.Verdict
	RiskScore    int
	Reason       string
	MatchedRules []string
}

// Engine evaluates action requests against a policy config.
type Engine struct {
	config Config
}

// Default risk scores per action type.
var defaultRisk = map[schema.Action]int{
	schema.ActionInstallPackage:  60,
	schema.ActionRunCode:         50,
	schema.ActionAccessSecret:    80,
	schema.ActionWriteFile:       30,
	schema.ActionCallExternalAPI: 40,
}

// LoadFromFile loads a policy from a YAML file.
func LoadFromFile(path string) (*Engine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading policy file: %w", err)
	}
	return LoadFromBytes(data)
}

// LoadFromBytes loads a policy from YAML bytes.
func LoadFromBytes(data []byte) (*Engine, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing policy: %w", err)
	}
	return &Engine{config: config}, nil
}

// DefaultEngine returns an engine with sensible defaults:
// deny unpinned installs, deny shell pipes, deny secret access.
func DefaultEngine() *Engine {
	return &Engine{
		config: Config{
			Actions: map[string]ActionPolicy{
				"install_package": {
					RequireHashes: true,
				},
				"run_code": {
					BlockPatterns: []string{"| bash", "| sh", "| python"},
					Network:       "deny",
				},
				"access_secret": {
					RequireApproval: "always",
					MaxTTL:          300,
				},
			},
		},
	}
}

// Evaluate runs the policy against an action request.
func (e *Engine) Evaluate(req schema.ActionRequest) EvalResult {
	actionStr := string(req.Action)
	actionPolicy, exists := e.config.Actions[actionStr]

	// Base risk score
	risk := defaultRisk[req.Action]
	if exists && actionPolicy.BaseRisk != nil {
		risk = *actionPolicy.BaseRisk
	}

	var matched []string

	if !exists {
		// No specific policy — use defaults, allow with base risk
		return EvalResult{
			Verdict:   schema.VerdictAllow,
			RiskScore: risk,
			Reason:    "no specific policy, default allow",
		}
	}

	// ── Install package rules ──
	if req.Action == schema.ActionInstallPackage {
		var params schema.InstallPackageParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			if actionPolicy.RequireHashes && params.Hash == "" {
				return EvalResult{
					Verdict:      schema.VerdictDeny,
					RiskScore:    100,
					Reason:       "package install requires hash verification (require_hashes: true)",
					MatchedRules: []string{"require_hashes"},
				}
			}
			if actionPolicy.RequireLockfile && !params.Pinned {
				risk += 20
				matched = append(matched, "require_lockfile")
			}
		}
	}

	// ── Run code rules ──
	if req.Action == schema.ActionRunCode {
		var params schema.RunCodeParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			// Check block patterns
			for _, pattern := range actionPolicy.BlockPatterns {
				if strings.Contains(params.Command, pattern) {
					return EvalResult{
						Verdict:      schema.VerdictDeny,
						RiskScore:    100,
						Reason:       fmt.Sprintf("command matches blocked pattern: %q", pattern),
						MatchedRules: []string{"block_pattern:" + pattern},
					}
				}
			}
			// Network policy
			if actionPolicy.Network == "deny" && params.Network {
				return EvalResult{
					Verdict:      schema.VerdictDeny,
					RiskScore:    90,
					Reason:       "code execution with network access denied by policy",
					MatchedRules: []string{"network_deny"},
				}
			}
		}
	}

	// ── External API rules ──
	if req.Action == schema.ActionCallExternalAPI {
		var params schema.CallExternalAPIParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			if params.Domain == "" {
				params.Domain = deriveDomain(params.URL)
			}
			if len(actionPolicy.AllowedDomains) > 0 {
				allowed := false
				for _, d := range actionPolicy.AllowedDomains {
					if matchDomain(params.Domain, d) {
						allowed = true
						break
					}
				}
				if !allowed {
					return EvalResult{
						Verdict:      schema.VerdictDeny,
						RiskScore:    85,
						Reason:       fmt.Sprintf("domain %q not in allowed list", params.Domain),
						MatchedRules: []string{"domain_allowlist"},
					}
				}
			}
			for _, d := range actionPolicy.BlockedDomains {
				if matchDomain(params.Domain, d) {
					return EvalResult{
						Verdict:      schema.VerdictDeny,
						RiskScore:    95,
						Reason:       fmt.Sprintf("domain %q is blocked", params.Domain),
						MatchedRules: []string{"domain_blocklist"},
					}
				}
			}
		}
	}

	// ── Approval rules ──
	if actionPolicy.RequireApproval == "always" {
		return EvalResult{
			Verdict:      schema.VerdictEscalate,
			RiskScore:    risk,
			Reason:       "action requires human approval (require_approval: always)",
			MatchedRules: append(matched, "require_approval"),
		}
	}
	if actionPolicy.RequireApproval == "high_risk" && risk >= 70 {
		return EvalResult{
			Verdict:      schema.VerdictEscalate,
			RiskScore:    risk,
			Reason:       fmt.Sprintf("risk score %d exceeds threshold for escalation", risk),
			MatchedRules: append(matched, "high_risk_escalation"),
		}
	}

	return EvalResult{
		Verdict:      schema.VerdictAllow,
		RiskScore:    risk,
		Reason:       "policy evaluated, action allowed",
		MatchedRules: matched,
	}
}

// matchDomain checks if a domain matches a pattern.
// Supports wildcard prefix: "*.github.com" matches "api.github.com"
func matchDomain(domain, pattern string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // ".github.com"
		return strings.HasSuffix(domain, suffix) || domain == pattern[2:]
	}
	return domain == pattern
}

func deriveDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}
