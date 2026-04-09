// Package schema defines the canonical action and trace types for agentctl.
//
// This is the most important file in the project. The action schema is the
// interface contract between agents, the gate, the policy engine, and the
// trace store. Get this wrong and everything downstream is painful.
//
// Design principles:
//   - Actions are the unit of governance, not tool calls or model outputs
//   - Only risky operations get schema'd (install, exec, secret, write, call)
//   - Context is injected server-side, never trusted from the agent
//   - Every field exists because it's needed for a policy decision or trace query
package schema

import (
	"encoding/json"
	"time"
)

// Action is the canonical set of governable operations.
// We deliberately keep this small. Not every agent tool call needs governance —
// only the ones where getting it wrong has consequences.
type Action string

const (
	ActionInstallPackage  Action = "install_package"
	ActionRunCode         Action = "run_code"
	ActionAccessSecret    Action = "access_secret"
	ActionWriteFile       Action = "write_file"
	ActionCallExternalAPI Action = "call_external_api"
)

// ActionRequest is the universal input to the gate.
// Every agent framework (LangChain, CrewAI, MCP, raw function calls)
// gets normalized into this shape by the SDK middleware.
type ActionRequest struct {
	// What the agent wants to do
	Action Action          `json:"action"`
	Params json.RawMessage `json:"params"` // Action-specific, validated per type

	// Why (required, logged, used for semantic policy checks)
	Reason string `json:"reason"`

	// Injected by the SDK/gateway — agent cannot set these
	Context RequestContext `json:"context"`
}

// RequestContext is set by the agentctl SDK, not the agent or model.
// This is the equivalent of JWT claims — signed by the infrastructure,
// not the caller.
type RequestContext struct {
	SessionID string    `json:"session_id"`
	Model     string    `json:"model,omitempty"`
	Agent     string    `json:"agent,omitempty"`
	Actor     string    `json:"actor,omitempty"`
	Team      string    `json:"team,omitempty"`
	Turn      int       `json:"turn"`
	Timestamp time.Time `json:"timestamp"`
}

// Verdict is the output of the gate.
type Verdict string

const (
	VerdictAllow    Verdict = "allow"
	VerdictDeny     Verdict = "deny"
	VerdictEscalate Verdict = "escalate"
)

// Decision is the full output of a gate() call.
// This is both the runtime return value AND the trace record.
type Decision struct {
	// Core fields
	TraceID   string    `json:"trace_id"`
	Verdict   Verdict   `json:"verdict"`
	RiskScore int       `json:"risk_score"` // 0-100
	Timestamp time.Time `json:"timestamp"`

	// What was requested
	Request ActionRequest `json:"request"`

	// Why this verdict was reached
	Reason       string   `json:"reason"`
	MatchedRules []string `json:"matched_rules,omitempty"`

	// For escalation
	ApprovalRequired bool   `json:"approval_required,omitempty"`
	ApprovedBy       string `json:"approved_by,omitempty"`

	// Timing
	EvalDurationMs float64 `json:"eval_duration_ms"`
}

// ── Action-specific param types ─────────────────────────────────────────────
// These are validated when params are unmarshaled. The gate rejects
// requests with invalid params before policy evaluation.

type InstallPackageParams struct {
	Manager string `json:"manager"` // pip, npm, cargo, go
	Package string `json:"package"`
	Version string `json:"version,omitempty"`
	Hash    string `json:"hash,omitempty"`   // sha256:...
	Pinned  bool   `json:"pinned,omitempty"` // from lockfile?
}

type RunCodeParams struct {
	Language string `json:"language"` // python, bash, node, etc
	Command  string `json:"command"`
	Stdin    string `json:"stdin,omitempty"`
	Network  bool   `json:"network,omitempty"` // does this need egress?
}

type AccessSecretParams struct {
	Name  string `json:"name"`
	Scope string `json:"scope,omitempty"` // read, write, admin
	TTL   int    `json:"ttl,omitempty"`   // seconds
}

type WriteFileParams struct {
	Path      string `json:"path"`
	Operation string `json:"operation"` // create, overwrite, append
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

type CallExternalAPIParams struct {
	URL    string `json:"url"`
	Method string `json:"method"`
	Domain string `json:"domain"` // extracted, for policy matching
}
