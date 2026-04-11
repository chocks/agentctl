// Package gate implements the core gate() primitive.
//
// gate(action) → Decision{allow, deny, escalate}
//
// This is the single decision point for all risky agent actions.
// Every action goes through here. The gate:
//  1. Validates the action request
//  2. Evaluates policy rules
//  3. Computes risk score
//  4. Records the trace
//  5. Returns the verdict
//
// The gate is designed to be fast (<5ms for cached allow decisions)
// and never to block on external calls for allow/deny verdicts.
// Only escalation involves async (waiting for human approval).
package gate

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/chocks/agentctl/pkg/policy"
	"github.com/chocks/agentctl/pkg/schema"
	"github.com/chocks/agentctl/pkg/trace"
	"github.com/google/uuid"
)

// Gate is the central decision engine.
type Gate struct {
	policy *policy.Engine
	tracer *trace.Store
	mu     sync.RWMutex
}

// New creates a gate with the given policy engine and trace store.
func New(pol *policy.Engine, tracer *trace.Store) *Gate {
	return &Gate{
		policy: pol,
		tracer: tracer,
	}
}

// Evaluate is the core primitive. It takes an action request,
// evaluates it against policy, records the trace, and returns a decision.
func (g *Gate) Evaluate(req schema.ActionRequest) (*schema.Decision, error) {
	start := time.Now()

	// 1. Validate the request
	if err := validateRequest(req); err != nil {
		decision := &schema.Decision{
			TraceID:        uuid.New().String(),
			Verdict:        schema.VerdictDeny,
			RiskScore:      100,
			Timestamp:      time.Now(),
			Request:        req,
			Reason:         fmt.Sprintf("invalid request: %v", err),
			EvalDurationMs: float64(time.Since(start).Microseconds()) / 1000,
		}
		g.record(decision)
		return decision, nil
	}

	// 2. Evaluate policy
	g.mu.RLock()
	result := g.policy.Evaluate(req)
	g.mu.RUnlock()

	// 3. Build decision
	decision := &schema.Decision{
		TraceID:        uuid.New().String(),
		Verdict:        result.Verdict,
		RiskScore:      result.RiskScore,
		Timestamp:      time.Now(),
		Request:        req,
		Reason:         result.Reason,
		MatchedRules:   result.MatchedRules,
		EvalDurationMs: float64(time.Since(start).Microseconds()) / 1000,
	}

	if result.Verdict == schema.VerdictEscalate {
		decision.ApprovalRequired = true
	}

	// 4. Record trace (async, non-blocking)
	g.record(decision)

	return decision, nil
}

// ReloadPolicy hot-reloads the policy engine.
func (g *Gate) ReloadPolicy(pol *policy.Engine) {
	g.mu.Lock()
	g.policy = pol
	g.mu.Unlock()
}

func (g *Gate) record(d *schema.Decision) {
	if g.tracer != nil {
		g.tracer.Record(d)
	}
}

// validateRequest checks that the action is known and params are valid.
func validateRequest(req schema.ActionRequest) error {
	switch req.Action {
	case schema.ActionInstallPackage:
		var p schema.InstallPackageParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return fmt.Errorf("install_package params: %w", err)
		}
		if p.Package == "" {
			return fmt.Errorf("install_package: package name required")
		}
	case schema.ActionRunCode:
		var p schema.RunCodeParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return fmt.Errorf("run_code params: %w", err)
		}
		if p.Command == "" {
			return fmt.Errorf("run_code: command required")
		}
	case schema.ActionAccessSecret:
		var p schema.AccessSecretParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return fmt.Errorf("access_secret params: %w", err)
		}
		if p.Name == "" {
			return fmt.Errorf("access_secret: name required")
		}
	case schema.ActionWriteFile:
		var p schema.WriteFileParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return fmt.Errorf("write_file params: %w", err)
		}
		if p.Path == "" {
			return fmt.Errorf("write_file: path required")
		}
	case schema.ActionCallExternalAPI:
		var p schema.CallExternalAPIParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return fmt.Errorf("call_external_api params: %w", err)
		}
		if p.URL == "" {
			return fmt.Errorf("call_external_api: url required")
		}
	default:
		return fmt.Errorf("unknown action: %s", req.Action)
	}
	return nil
}
