package main

// MCP server for agentctl — stdio transport, JSON-RPC 2.0.
//
// Implements the minimal subset of the Model Context Protocol needed to act
// as a tool provider for Claude Code, Codex CLI, and any other MCP client.
// Spec: https://spec.modelcontextprotocol.io/specification/2024-11-05/
//
// Usage (Claude Code ~/.claude/settings.json):
//
//	{
//	  "mcpServers": {
//	    "agentctl": {
//	      "command": "agentctl",
//	      "args": ["mcp"],
//	      "env": { "AGENTCTL_POLICY": "./agentctl.policy.yaml" }
//	    }
//	  }
//	}
//
// Usage (Codex CLI ~/.codex/config.yaml):
//
//	mcp_servers:
//	  agentctl:
//	    command: agentctl
//	    args: [mcp]
//
// When an MCP client calls one of the five agentctl tools, the request is
// evaluated against the local policy, a trace is recorded, and the verdict
// is returned as the tool result. Deny and escalate verdicts are returned as
// isError:true so the caller treats them as a blocking failure.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/chocks/agentctl/pkg/gate"
	"github.com/chocks/agentctl/pkg/schema"
	"github.com/chocks/agentctl/pkg/trace"
)

const mcpProtocolVersion = "2024-11-05"

// ── JSON-RPC 2.0 wire types ──────────────────────────────────────────────────

type jsonRPCMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"` // nil for notifications
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *jsonRPCError    `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Standard JSON-RPC 2.0 error codes.
const (
	codeParseError     = -32700
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// ── MCP protocol types ───────────────────────────────────────────────────────

type mcpInitializeParams struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    map[string]any    `json:"capabilities"`
	ClientInfo      mcpImplementation `json:"clientInfo"`
}

type mcpImplementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpInitializeResult struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    map[string]any    `json:"capabilities"`
	ServerInfo      mcpImplementation `json:"serverInfo"`
}

type mcpTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type mcpToolsListResult struct {
	Tools []mcpTool `json:"tools"`
}

type mcpToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type mcpToolCallResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError"`
}

// ── Argument structs for each tool ───────────────────────────────────────────

type mcpInstallPackageArgs struct {
	Manager string `json:"manager"`
	Package string `json:"package"`
	Version string `json:"version,omitempty"`
	Hash    string `json:"hash,omitempty"`
	Pinned  bool   `json:"pinned,omitempty"`
	Reason  string `json:"reason"`
}

type mcpRunCodeArgs struct {
	Language string `json:"language"`
	Command  string `json:"command"`
	Stdin    string `json:"stdin,omitempty"`
	Network  bool   `json:"network,omitempty"`
	Reason   string `json:"reason"`
}

type mcpWriteFileArgs struct {
	Path      string `json:"path"`
	Operation string `json:"operation"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	Reason    string `json:"reason"`
}

type mcpAccessSecretArgs struct {
	Name   string `json:"name"`
	Scope  string `json:"scope,omitempty"`
	TTL    int    `json:"ttl,omitempty"`
	Reason string `json:"reason"`
}

type mcpCallExternalAPIArgs struct {
	URL    string `json:"url"`
	Method string `json:"method"`
	Domain string `json:"domain,omitempty"`
	Reason string `json:"reason"`
}

// ── Tool input schemas (JSON Schema, inlined as raw JSON) ────────────────────

// schemaObj is a helper to build JSON Schema objects concisely.
func schemaObj(properties map[string]any, required []string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	})
	return raw
}

func strProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func boolProp(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

func intProp(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

// mcpTools returns the five agentctl gate tools for tools/list.
func mcpTools() []mcpTool {
	return []mcpTool{
		{
			Name:        "agentctl_install_package",
			Description: "Gate a package installation through agentctl policy before it runs. Use before pip install, npm install, cargo add, go get, or equivalent. Returns allow/deny/escalate verdict and records a trace.",
			InputSchema: schemaObj(map[string]any{
				"manager": strProp("Package manager: pip, npm, yarn, cargo, go"),
				"package": strProp("Package name (may include version specifier, e.g. requests==2.31.0)"),
				"version": strProp("Explicit version string, if separate from package name"),
				"hash":    strProp("Integrity hash for verification (e.g. sha256:abc123...)"),
				"pinned":  boolProp("True if the package comes from a lockfile"),
				"reason":  strProp("Why this package is needed"),
			}, []string{"manager", "package", "reason"}),
		},
		{
			Name:        "agentctl_run_code",
			Description: "Gate a code or shell command execution through agentctl policy before it runs. Returns allow/deny/escalate verdict and records a trace.",
			InputSchema: schemaObj(map[string]any{
				"language": strProp("Execution language or runtime: bash, python, node, ruby, etc."),
				"command":  strProp("The full command or code string to execute"),
				"stdin":    strProp("Optional stdin to pipe to the command"),
				"network":  boolProp("True if the command requires outbound network access"),
				"reason":   strProp("Why this command needs to run"),
			}, []string{"language", "command", "reason"}),
		},
		{
			Name:        "agentctl_write_file",
			Description: "Gate a file write operation through agentctl policy before it runs. Returns allow/deny/escalate verdict and records a trace.",
			InputSchema: schemaObj(map[string]any{
				"path":       strProp("Absolute or relative path of the file to write"),
				"operation":  strProp("File operation: create, overwrite, or append"),
				"size_bytes": intProp("Approximate size of content in bytes, if known"),
				"reason":     strProp("Why this file needs to be written"),
			}, []string{"path", "operation", "reason"}),
		},
		{
			Name:        "agentctl_access_secret",
			Description: "Gate a secret or credential access through agentctl policy. Use before reading API keys, tokens, passwords, or other sensitive values. Returns allow/deny/escalate verdict and records a trace.",
			InputSchema: schemaObj(map[string]any{
				"name":   strProp("Secret name or identifier (e.g. OPENAI_API_KEY, db-password)"),
				"scope":  strProp("Access scope: read, write, or admin"),
				"ttl":    intProp("Requested token lifetime in seconds"),
				"reason": strProp("Why this secret is needed"),
			}, []string{"name", "reason"}),
		},
		{
			Name:        "agentctl_call_external_api",
			Description: "Gate an outbound HTTP request through agentctl policy before it is made. Returns allow/deny/escalate verdict and records a trace.",
			InputSchema: schemaObj(map[string]any{
				"url":    strProp("Full URL of the request"),
				"method": strProp("HTTP method: GET, POST, PUT, DELETE, etc."),
				"domain": strProp("Target domain (derived from URL if omitted)"),
				"reason": strProp("Why this API call is needed"),
			}, []string{"url", "method", "reason"}),
		},
	}
}

// ── Server ───────────────────────────────────────────────────────────────────

type mcpServer struct {
	sessionID string
	agent     string
	model     string
	g         *gate.Gate
}

func cmdMCP() {
	agent := stringFlagValue("--agent", "agentctl-mcp")
	model := stringFlagValue("--model", "")

	sessionID := os.Getenv("AGENTCTL_SESSION")
	if sessionID == "" {
		sessionID = fmt.Sprintf("mcp-%d", time.Now().UnixMilli())
	}

	traceFile := traceFilePath()
	ensureDir(filepath.Dir(traceFile))
	tracer, err := trace.NewFileStore(traceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentctl mcp: error opening trace store: %v\n", err)
		os.Exit(1)
	}

	srv := &mcpServer{
		sessionID: sessionID,
		agent:     agent,
		model:     model,
		g:         gate.New(loadPolicy(), tracer),
	}
	srv.serve(os.Stdin, os.Stdout)
}

// serve reads newline-delimited JSON-RPC messages from in and writes responses to out.
// It runs until in is closed (EOF).
func (s *mcpServer) serve(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	enc := json.NewEncoder(out)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg jsonRPCMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			_ = enc.Encode(errorResponse(json.RawMessage("null"), codeParseError, "parse error"))
			continue
		}

		// Notifications have no id — handle but do not respond.
		if msg.ID == nil {
			continue
		}

		resp := s.dispatch(msg)
		_ = enc.Encode(resp)
	}
}

func (s *mcpServer) dispatch(msg jsonRPCMessage) jsonRPCMessage {
	id := *msg.ID
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(id)
	case "tools/list":
		return s.handleToolsList(id)
	case "tools/call":
		return s.handleToolsCall(id, msg.Params)
	case "ping":
		result, _ := json.Marshal(map[string]any{})
		return okResponse(id, result)
	default:
		return errorResponse(id, codeMethodNotFound, "method not found: "+msg.Method)
	}
}

func (s *mcpServer) handleInitialize(id json.RawMessage) jsonRPCMessage {
	result, _ := json.Marshal(mcpInitializeResult{
		ProtocolVersion: mcpProtocolVersion,
		Capabilities:    map[string]any{"tools": map[string]any{}},
		ServerInfo:      mcpImplementation{Name: "agentctl", Version: version},
	})
	return okResponse(id, result)
}

func (s *mcpServer) handleToolsList(id json.RawMessage) jsonRPCMessage {
	result, _ := json.Marshal(mcpToolsListResult{Tools: mcpTools()})
	return okResponse(id, result)
}

func (s *mcpServer) handleToolsCall(id json.RawMessage, rawParams json.RawMessage) jsonRPCMessage {
	var params mcpToolCallParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return errorResponse(id, codeInvalidParams, "invalid params: "+err.Error())
	}

	req, err := mcpArgsToActionRequest(params.Name, params.Arguments)
	if err != nil {
		return errorResponse(id, codeInvalidParams, err.Error())
	}

	req.Context = schema.RequestContext{
		SessionID: s.sessionID,
		Agent:     s.agent,
		Model:     s.model,
		Timestamp: time.Now(),
	}

	decision, err := s.g.Evaluate(req)
	if err != nil {
		return errorResponse(id, codeInternalError, "gate evaluation failed: "+err.Error())
	}

	if err := recordApprovalForDecision(approvalFilePath(), decision); err != nil {
		fmt.Fprintf(os.Stderr, "agentctl mcp: error recording approval: %v\n", err)
	}

	result, _ := json.Marshal(verdictToToolResult(decision))
	return okResponse(id, result)
}

// ── Tool argument → ActionRequest mapping ────────────────────────────────────

// mcpArgsToActionRequest maps a tool name and its JSON arguments to an agentctl ActionRequest.
func mcpArgsToActionRequest(toolName string, rawArgs json.RawMessage) (schema.ActionRequest, error) {
	switch toolName {
	case "agentctl_install_package":
		return mapInstallPackage(rawArgs)
	case "agentctl_run_code":
		return mapRunCode(rawArgs)
	case "agentctl_write_file":
		return mapWriteFile(rawArgs)
	case "agentctl_access_secret":
		return mapAccessSecret(rawArgs)
	case "agentctl_call_external_api":
		return mapCallExternalAPI(rawArgs)
	default:
		return schema.ActionRequest{}, fmt.Errorf("unknown tool: %s", toolName)
	}
}

func mapInstallPackage(raw json.RawMessage) (schema.ActionRequest, error) {
	var args mcpInstallPackageArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return schema.ActionRequest{}, fmt.Errorf("invalid install_package arguments: %w", err)
	}
	if args.Package == "" {
		return schema.ActionRequest{}, fmt.Errorf("package is required")
	}
	params, _ := json.Marshal(schema.InstallPackageParams{
		Manager: args.Manager,
		Package: args.Package,
		Version: args.Version,
		Hash:    args.Hash,
		Pinned:  args.Pinned,
	})
	return schema.ActionRequest{
		Action: schema.ActionInstallPackage,
		Params: params,
		Reason: args.Reason,
	}, nil
}

func mapRunCode(raw json.RawMessage) (schema.ActionRequest, error) {
	var args mcpRunCodeArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return schema.ActionRequest{}, fmt.Errorf("invalid run_code arguments: %w", err)
	}
	if args.Command == "" {
		return schema.ActionRequest{}, fmt.Errorf("command is required")
	}
	params, _ := json.Marshal(schema.RunCodeParams{
		Language: args.Language,
		Command:  args.Command,
		Stdin:    args.Stdin,
		Network:  args.Network,
	})
	return schema.ActionRequest{
		Action: schema.ActionRunCode,
		Params: params,
		Reason: args.Reason,
	}, nil
}

func mapWriteFile(raw json.RawMessage) (schema.ActionRequest, error) {
	var args mcpWriteFileArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return schema.ActionRequest{}, fmt.Errorf("invalid write_file arguments: %w", err)
	}
	if args.Path == "" {
		return schema.ActionRequest{}, fmt.Errorf("path is required")
	}
	params, _ := json.Marshal(schema.WriteFileParams{
		Path:      args.Path,
		Operation: args.Operation,
		SizeBytes: args.SizeBytes,
	})
	return schema.ActionRequest{
		Action: schema.ActionWriteFile,
		Params: params,
		Reason: args.Reason,
	}, nil
}

func mapAccessSecret(raw json.RawMessage) (schema.ActionRequest, error) {
	var args mcpAccessSecretArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return schema.ActionRequest{}, fmt.Errorf("invalid access_secret arguments: %w", err)
	}
	if args.Name == "" {
		return schema.ActionRequest{}, fmt.Errorf("name is required")
	}
	params, _ := json.Marshal(schema.AccessSecretParams{
		Name:  args.Name,
		Scope: args.Scope,
		TTL:   args.TTL,
	})
	return schema.ActionRequest{
		Action: schema.ActionAccessSecret,
		Params: params,
		Reason: args.Reason,
	}, nil
}

func mapCallExternalAPI(raw json.RawMessage) (schema.ActionRequest, error) {
	var args mcpCallExternalAPIArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return schema.ActionRequest{}, fmt.Errorf("invalid call_external_api arguments: %w", err)
	}
	if args.URL == "" {
		return schema.ActionRequest{}, fmt.Errorf("url is required")
	}
	domain := args.Domain
	if domain == "" {
		if parsed, err := url.Parse(args.URL); err == nil {
			domain = parsed.Hostname()
		}
	}
	params, _ := json.Marshal(schema.CallExternalAPIParams{
		URL:    args.URL,
		Method: args.Method,
		Domain: domain,
	})
	return schema.ActionRequest{
		Action: schema.ActionCallExternalAPI,
		Params: params,
		Reason: args.Reason,
	}, nil
}

// ── Verdict → MCP tool result ────────────────────────────────────────────────

func verdictToToolResult(d *schema.Decision) mcpToolCallResult {
	switch d.Verdict {
	case schema.VerdictAllow:
		summary, _ := json.Marshal(map[string]any{
			"verdict":    string(d.Verdict),
			"trace_id":   d.TraceID,
			"risk_score": d.RiskScore,
			"reason":     d.Reason,
		})
		return mcpToolCallResult{
			Content: []mcpContent{{Type: "text", Text: string(summary)}},
			IsError: false,
		}

	case schema.VerdictDeny:
		return mcpToolCallResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf(
				"agentctl deny: %s (trace_id: %s)", d.Reason, d.TraceID,
			)}},
			IsError: true,
		}

	default: // VerdictEscalate
		return mcpToolCallResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf(
				"agentctl escalate: %s\nPending approval — run: agentctl approval approve %s",
				d.Reason, d.TraceID,
			)}},
			IsError: true,
		}
	}
}

// ── JSON-RPC helpers ─────────────────────────────────────────────────────────

func okResponse(id, result json.RawMessage) jsonRPCMessage {
	return jsonRPCMessage{JSONRPC: "2.0", ID: &id, Result: result}
}

func errorResponse(id json.RawMessage, code int, msg string) jsonRPCMessage {
	return jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Error:   &jsonRPCError{Code: code, Message: msg},
	}
}
