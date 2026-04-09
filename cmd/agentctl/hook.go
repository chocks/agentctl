package main

// Claude Code PreToolUse hook adapter.
//
// Usage (in ~/.claude/settings.json):
//
//	{
//	  "hooks": {
//	    "PreToolUse": [
//	      {
//	        "matcher": "Bash|Write|Edit|MultiEdit|WebFetch",
//	        "hooks": [{ "type": "command", "command": "agentctl hook claude-code" }]
//	      }
//	    ]
//	  }
//	}
//
// The hook receives a JSON event on stdin, evaluates it against the local
// agentctl policy, emits a trace, and exits:
//
//	0  — allow (tool call proceeds)
//	2  — block (stdout reason shown to user by Claude Code)

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentctl/agentctl/pkg/gate"
	"github.com/agentctl/agentctl/pkg/schema"
	"github.com/agentctl/agentctl/pkg/trace"
)

// claudeHookEvent is the JSON payload Claude Code sends to PreToolUse hooks on stdin.
type claudeHookEvent struct {
	SessionID     string          `json:"session_id"`
	HookEventName string          `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
}

// claudeHookResponse is output to stdout; Claude Code displays Reason on block.
type claudeHookResponse struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

func cmdHook() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: agentctl hook [claude-code]")
		os.Exit(1)
	}
	switch os.Args[2] {
	case "claude-code":
		cmdHookClaudeCode()
	default:
		fmt.Fprintf(os.Stderr, "unknown hook type: %s\n", os.Args[2])
		os.Exit(1)
	}
}

func cmdHookClaudeCode() {
	agent := stringFlagValue("--agent", "claude-code")
	model := stringFlagValue("--model", "")
	sessionOverride := stringFlagValue("--session", "")

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentctl hook: error reading stdin: %v\n", err)
		os.Exit(1)
	}

	var event claudeHookEvent
	if err := json.Unmarshal(input, &event); err != nil {
		// Malformed input — allow rather than block on hook errors.
		fmt.Fprintf(os.Stderr, "agentctl hook: error parsing hook event: %v\n", err)
		os.Exit(0)
	}

	req, skip := claudeEventToActionRequest(event)
	if skip {
		// Tool not governed by agentctl — allow silently.
		os.Exit(0)
	}

	sessionID := sessionOverride
	if sessionID == "" {
		sessionID = event.SessionID
	}
	if sessionID == "" {
		sessionID = fmt.Sprintf("claude-%d", time.Now().UnixMilli())
	}

	req.Context = schema.RequestContext{
		SessionID: sessionID,
		Agent:     agent,
		Model:     model,
		Timestamp: time.Now(),
	}

	pol := loadPolicy()
	traceFile := traceFilePath()
	ensureDir(filepath.Dir(traceFile))

	tracer, err := trace.NewFileStore(traceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentctl hook: error opening trace store: %v\n", err)
		os.Exit(0) // Fail open: do not block tool use on infrastructure errors.
	}

	g := gate.New(pol, tracer)
	decision, err := g.Evaluate(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentctl hook: error evaluating gate: %v\n", err)
		os.Exit(0) // Fail open.
	}

	if err := recordApprovalForDecision(approvalFilePath(), decision); err != nil {
		fmt.Fprintf(os.Stderr, "agentctl hook: error recording approval: %v\n", err)
	}

	switch decision.Verdict {
	case schema.VerdictAllow:
		os.Exit(0)

	case schema.VerdictDeny:
		hookBlock(fmt.Sprintf("agentctl deny: %s", decision.Reason))

	case schema.VerdictEscalate:
		hookBlock(fmt.Sprintf(
			"agentctl escalate (pending approval): %s\nTo approve: agentctl approval approve %s",
			decision.Reason, decision.TraceID,
		))
	}
}

// hookBlock writes a block response to stdout and exits 2.
// Claude Code shows the Reason to the user when the hook exits non-zero.
func hookBlock(reason string) {
	resp := claudeHookResponse{Decision: "block", Reason: reason}
	out, _ := json.Marshal(resp)
	fmt.Println(string(out))
	os.Exit(2)
}

// claudeEventToActionRequest maps a Claude Code PreToolUse event to an agentctl
// ActionRequest. Returns (req, skip=true) for tools that agentctl does not govern.
func claudeEventToActionRequest(event claudeHookEvent) (schema.ActionRequest, bool) {
	switch event.ToolName {
	case "Bash":
		return mapBashTool(event.ToolInput)
	case "Write":
		return mapWriteTool(event.ToolInput)
	case "Edit", "MultiEdit":
		return mapEditTool(event.ToolInput)
	case "WebFetch":
		return mapWebFetchTool(event.ToolInput)
	default:
		return schema.ActionRequest{}, true // not governed
	}
}

// mapBashTool maps a Bash tool call. Heuristically detects package installs;
// all other commands are classified as run_code.
func mapBashTool(raw json.RawMessage) (schema.ActionRequest, bool) {
	var input struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(raw, &input); err != nil || input.Command == "" {
		return schema.ActionRequest{}, true
	}

	if pkg, mgr, ok := detectPackageInstall(input.Command); ok {
		params, _ := json.Marshal(schema.InstallPackageParams{
			Manager: mgr,
			Package: pkg,
		})
		return schema.ActionRequest{
			Action: schema.ActionInstallPackage,
			Params: params,
			Reason: "install package via Bash: " + input.Command,
		}, false
	}

	params, _ := json.Marshal(schema.RunCodeParams{
		Language: "bash",
		Command:  input.Command,
		Network:  looksLikeNetworkCommand(input.Command),
	})
	return schema.ActionRequest{
		Action: schema.ActionRunCode,
		Params: params,
		Reason: "execute bash command",
	}, false
}

// mapWriteTool maps a Write tool call (create/overwrite a file).
func mapWriteTool(raw json.RawMessage) (schema.ActionRequest, bool) {
	var input struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(raw, &input); err != nil || input.FilePath == "" {
		return schema.ActionRequest{}, true
	}
	params, _ := json.Marshal(schema.WriteFileParams{
		Path:      input.FilePath,
		Operation: "overwrite",
		SizeBytes: int64(len(input.Content)),
	})
	return schema.ActionRequest{
		Action: schema.ActionWriteFile,
		Params: params,
		Reason: "write file: " + input.FilePath,
	}, false
}

// mapEditTool maps Edit and MultiEdit tool calls (in-place file edits).
func mapEditTool(raw json.RawMessage) (schema.ActionRequest, bool) {
	var input struct {
		FilePath  string `json:"file_path"`
		NewString string `json:"new_string"`
	}
	if err := json.Unmarshal(raw, &input); err != nil || input.FilePath == "" {
		return schema.ActionRequest{}, true
	}
	params, _ := json.Marshal(schema.WriteFileParams{
		Path:      input.FilePath,
		Operation: "overwrite",
		SizeBytes: int64(len(input.NewString)),
	})
	return schema.ActionRequest{
		Action: schema.ActionWriteFile,
		Params: params,
		Reason: "edit file: " + input.FilePath,
	}, false
}

// mapWebFetchTool maps a WebFetch tool call (outbound HTTP GET).
func mapWebFetchTool(raw json.RawMessage) (schema.ActionRequest, bool) {
	var input struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &input); err != nil || input.URL == "" {
		return schema.ActionRequest{}, true
	}
	parsed, err := url.Parse(input.URL)
	if err != nil || parsed.Hostname() == "" {
		return schema.ActionRequest{}, true
	}
	params, _ := json.Marshal(schema.CallExternalAPIParams{
		URL:    input.URL,
		Method: "GET",
		Domain: parsed.Hostname(),
	})
	return schema.ActionRequest{
		Action: schema.ActionCallExternalAPI,
		Params: params,
		Reason: "fetch URL: " + input.URL,
	}, false
}

// detectPackageInstall checks if a shell command is a package manager install.
// Returns (package, manager, ok). Only matches coding-tool package managers
// (pip, npm, yarn, cargo, go) — not system package managers (apt, brew, etc.).
func detectPackageInstall(command string) (string, string, bool) {
	type pattern struct {
		prefix  string
		manager string
	}
	patterns := []pattern{
		{"pip install ", "pip"},
		{"pip3 install ", "pip"},
		{"python -m pip install ", "pip"},
		{"npm install ", "npm"},
		{"npm i ", "npm"},
		{"yarn add ", "npm"},
		{"pnpm add ", "npm"},
		{"cargo add ", "cargo"},
		{"cargo install ", "cargo"},
		{"go get ", "go"},
		{"go install ", "go"},
	}

	trimmed := strings.TrimSpace(command)
	for _, p := range patterns {
		if !strings.HasPrefix(trimmed, p.prefix) {
			continue
		}
		rest := strings.TrimPrefix(trimmed, p.prefix)
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		pkg := fields[0]
		if strings.HasPrefix(pkg, "-") {
			// Flag before package name — skip for now rather than mis-classify.
			continue
		}
		return pkg, p.manager, true
	}
	return "", "", false
}

// looksLikeNetworkCommand returns true if the command likely makes outbound network calls.
func looksLikeNetworkCommand(command string) bool {
	lower := strings.ToLower(strings.TrimSpace(command))
	// Tokens that can appear mid-command.
	contains := []string{"curl ", "wget ", "http://", "https://"}
	for _, tok := range contains {
		if strings.Contains(lower, tok) {
			return true
		}
	}
	// Commands that are network tools when they start the pipeline.
	prefixes := []string{"ssh ", "scp ", "rsync "}
	for _, pfx := range prefixes {
		if strings.HasPrefix(lower, pfx) {
			return true
		}
	}
	return false
}
