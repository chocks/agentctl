// agentctl CLI — the simplest possible entry point.
//
// Usage:
//
//	echo '{"action":"install_package","params":{...},"reason":"..."}' | agentctl gate
//	agentctl trace list --last 20
//	agentctl trace search --action install_package --since 7d
//	agentctl replay <session_id> --policy new.policy.yaml
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentctl/agentctl/pkg/gate"
	"github.com/agentctl/agentctl/pkg/policy"
	"github.com/agentctl/agentctl/pkg/schema"
	"github.com/agentctl/agentctl/pkg/trace"
)

const defaultPolicyFile = "agentctl.policy.yaml"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "gate":
		cmdGate()
	case "trace":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: agentctl trace [list|search]")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "list":
			cmdTraceList()
		case "search":
			cmdTraceSearch()
		default:
			fmt.Fprintf(os.Stderr, "unknown trace command: %s\n", os.Args[2])
			os.Exit(1)
		}
	case "replay":
		cmdReplay()
	case "approval":
		cmdApproval()
	case "serve":
		cmdServe()
	case "version":
		fmt.Println("agentctl v0.1.0")
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func cmdGate() {
	// Load policy
	pol := loadPolicy()
	traceFile := traceFilePath()

	// Set up trace store
	ensureDir(filepath.Dir(traceFile))
	tracer, err := trace.NewFileStore(traceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Create gate
	g := gate.New(pol, tracer)

	// Read action request from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		os.Exit(1)
	}

	var req schema.ActionRequest
	if err := json.Unmarshal(input, &req); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing request: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()

	// Inject context (CLI sets these, not the caller)
	req.Context = schema.RequestContext{
		SessionID: stringFlagValue("--session", fmt.Sprintf("cli_%d", now.UnixMilli())),
		Model:     stringFlagValue("--model", ""),
		Agent:     stringFlagValue("--agent", "agentctl-cli"),
		Timestamp: now,
	}

	// Evaluate
	decision, err := g.Evaluate(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := recordApprovalForDecision(approvalFilePath(), decision); err != nil {
		fmt.Fprintf(os.Stderr, "error recording approval: %v\n", err)
		os.Exit(1)
	}

	// Output decision as JSON
	out, err := json.MarshalIndent(decision, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding decision: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))

	// Exit code reflects verdict
	switch decision.Verdict {
	case schema.VerdictAllow:
		os.Exit(0)
	case schema.VerdictDeny:
		os.Exit(1)
	case schema.VerdictEscalate:
		os.Exit(2)
	}
}

func cmdTraceList() {
	limit := 20
	// Simple flag parsing
	for i, arg := range os.Args {
		if arg == "--last" && i+1 < len(os.Args) {
			if _, err := fmt.Sscanf(os.Args[i+1], "%d", &limit); err != nil {
				fmt.Fprintf(os.Stderr, "invalid --last value %q: %v\n", os.Args[i+1], err)
				os.Exit(1)
			}
		}
	}

	traces, err := trace.ReadTraces(traceFilePath(), trace.TraceFilter{Limit: limit})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Print as table
	fmt.Printf("%-20s %-20s %-10s %-5s %s\n", "TIME", "ACTION", "VERDICT", "RISK", "REASON")
	fmt.Println(repeat("-", 90))
	for _, t := range traces {
		fmt.Printf("%-20s %-20s %-10s %-5d %s\n",
			t.Timestamp.Format("15:04:05"),
			t.Request.Action,
			t.Verdict,
			t.RiskScore,
			truncate(t.Reason, 40),
		)
	}
	fmt.Printf("\n%d traces shown\n", len(traces))
}

func cmdTraceSearch() {
	filter := trace.TraceFilter{}
	for i, arg := range os.Args {
		if i+1 >= len(os.Args) {
			break
		}
		switch arg {
		case "--session":
			filter.SessionID = os.Args[i+1]
		case "--action":
			filter.Action = schema.Action(os.Args[i+1])
		case "--verdict":
			filter.Verdict = schema.Verdict(os.Args[i+1])
		case "--package":
			filter.Package = os.Args[i+1]
		case "--since":
			if d, err := parseDuration(os.Args[i+1]); err == nil {
				filter.Since = time.Now().Add(-d)
			}
		case "--limit":
			if _, err := fmt.Sscanf(os.Args[i+1], "%d", &filter.Limit); err != nil {
				fmt.Fprintf(os.Stderr, "invalid --limit value %q: %v\n", os.Args[i+1], err)
				os.Exit(1)
			}
		}
	}

	traces, err := trace.ReadTraces(traceFilePath(), filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for _, t := range traces {
		out, err := json.MarshalIndent(t, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error encoding trace: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(out))
	}
	fmt.Fprintf(os.Stderr, "\n%d traces found\n", len(traces))
}

func cmdReplay() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: agentctl replay <session_id> [--policy file] [--limit N]")
		os.Exit(1)
	}

	sessionID := os.Args[2]
	policyFile := stringFlagValue("--policy", defaultPolicyFile)
	limit := intFlagValue("--limit", 0)

	results, err := replaySession(policyFile, traceFilePath(), sessionID, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	out, err := json.MarshalIndent(struct {
		SessionID string            `json:"session_id"`
		Policy    string            `json:"policy"`
		Results   []schema.Decision `json:"results"`
	}{
		SessionID: sessionID,
		Policy:    policyFile,
		Results:   results,
	}, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding replay results: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

func loadPolicy() *policy.Engine {
	return loadPolicyFromPath(defaultPolicyFile)
}

func loadPolicyFromPath(path string) *policy.Engine {
	if _, err := os.Stat(path); err == nil {
		pol, err := policy.LoadFromFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to load %s: %v (using defaults)\n", path, err)
			return policy.DefaultEngine()
		}
		return pol
	}
	return policy.DefaultEngine()
}

func printUsage() {
	fmt.Println(`agentctl — trace and control dangerous agent actions

Usage:
  agentctl gate [--session id]     Evaluate an action (JSON from stdin)
  agentctl trace list [--last N]   Show recent traces
  agentctl trace search [filters]  Search traces
  agentctl replay <session_id>     Re-evaluate a session with a policy file
  agentctl approval [subcommand]   List or resolve escalations
  agentctl serve [--addr host:port] Run local HTTP API
  agentctl version                 Print version

Gate reads an ActionRequest from stdin and outputs a Decision.
Exit codes: 0=allow, 1=deny, 2=escalate

Search filters:
  --session <id>      Filter by session id
  --action <action>    Filter by action type
  --verdict <verdict>  Filter by verdict (allow/deny/escalate)
  --package <name>     Filter install_package by package name
  --since <duration>   Filter by time (e.g., 7d, 24h)
  --limit <n>          Max results

Gate flags:
  --session <id>       Reuse a session id across actions
  --agent <name>       Annotate the trace with an agent name
  --model <name>       Annotate the trace with a model name

Replay flags:
  --policy <file>      Policy file to use for replay
  --limit <n>          Max traces to replay

Approval commands:
  approval list [--status pending|approved|denied]
  approval approve <id> [--by name]
  approval deny <id> [--by name]

Serve flags:
  --addr <host:port>   Listen address (default 127.0.0.1:8080)
  --auth-token <token> Require bearer auth for the HTTP API

Environment:
  AGENTCTL_TRACE_FILE  Override the trace file path
  AGENTCTL_APPROVAL_FILE Override the approvals file path
  AGENTCTL_AUTH_TOKEN  Bearer token for the HTTP API
  AGENTCTL_HOME        Override the trace home directory`)
}

func ensureDir(path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating %s: %v\n", path, err)
		os.Exit(1)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func repeat(s string, n int) string {
	return strings.Repeat(s, n)
}

func parseDuration(s string) (time.Duration, error) {
	// Support "7d" style durations
	if len(s) > 0 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}
	return time.ParseDuration(s)
}

func stringFlagValue(name, fallback string) string {
	for i, arg := range os.Args {
		if arg == name && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return fallback
}

func intFlagValue(name string, fallback int) int {
	for i, arg := range os.Args {
		if arg == name && i+1 < len(os.Args) {
			var value int
			if _, err := fmt.Sscanf(os.Args[i+1], "%d", &value); err == nil {
				return value
			}
		}
	}
	return fallback
}

func traceFilePath() string {
	if path := os.Getenv("AGENTCTL_TRACE_FILE"); path != "" {
		return path
	}

	home := os.Getenv("AGENTCTL_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error resolving user home directory: %v\n", err)
			os.Exit(1)
		}
		home = filepath.Join(userHome, ".agentctl")
	}

	return filepath.Join(home, "traces.jsonl")
}
