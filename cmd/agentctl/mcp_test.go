package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/chocks/agentctl/pkg/gate"
	"github.com/chocks/agentctl/pkg/policy"
	"github.com/chocks/agentctl/pkg/schema"
	"github.com/chocks/agentctl/pkg/trace"
)

// newTestServer creates an mcpServer backed by a configurable policy and a
// no-op trace writer. sessionID is fixed so test output is deterministic.
// Approval file writes are isolated to a temp dir via the approvalPath field.
func newTestServer(t *testing.T, policyYAML string) *mcpServer {
	t.Helper()
	pol, err := policy.LoadFromBytes([]byte(policyYAML))
	if err != nil {
		t.Fatalf("LoadFromBytes: %v", err)
	}
	return &mcpServer{
		sessionID:    "test-session",
		agent:        "test-agent",
		approvalPath: t.TempDir() + "/approvals.jsonl",
		g:            gate.New(pol, trace.NewWriterStore(&bytes.Buffer{})),
	}
}

// rpc sends a single JSON-RPC message to the server and returns the decoded response.
// msg may be multi-line for readability; it is compacted to a single line before sending
// because the server uses a line-by-line scanner (MCP stdio transport is newline-delimited).
func rpc(t *testing.T, srv *mcpServer, msg string) jsonRPCMessage {
	t.Helper()
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(msg)); err != nil {
		t.Fatalf("compact input JSON: %v", err)
	}
	var out bytes.Buffer
	srv.serve(strings.NewReader(compact.String()+"\n"), &out)
	var resp jsonRPCMessage
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("decode response: %v\nraw: %s", err, out.String())
	}
	return resp
}

// ── Protocol handshake ───────────────────────────────────────────────────────

func TestMCPInitialize(t *testing.T) {
	srv := newTestServer(t, `actions: {}`)
	resp := rpc(t, srv, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	var result mcpInitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.ProtocolVersion != mcpProtocolVersion {
		t.Errorf("protocolVersion = %q, want %q", result.ProtocolVersion, mcpProtocolVersion)
	}
	if result.ServerInfo.Name != "agentctl" {
		t.Errorf("serverInfo.name = %q, want agentctl", result.ServerInfo.Name)
	}
	if _, ok := result.Capabilities["tools"]; !ok {
		t.Error("capabilities missing tools key")
	}
}

func TestMCPPing(t *testing.T) {
	srv := newTestServer(t, `actions: {}`)
	resp := rpc(t, srv, `{"jsonrpc":"2.0","id":2,"method":"ping","params":{}}`)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
}

func TestMCPUnknownMethod(t *testing.T) {
	srv := newTestServer(t, `actions: {}`)
	resp := rpc(t, srv, `{"jsonrpc":"2.0","id":3,"method":"unknown/method","params":{}}`)
	if resp.Error == nil {
		t.Fatal("expected error for unknown method, got nil")
	}
	if resp.Error.Code != codeMethodNotFound {
		t.Errorf("error code = %d, want %d", resp.Error.Code, codeMethodNotFound)
	}
}

func TestMCPParseError(t *testing.T) {
	srv := newTestServer(t, `actions: {}`)
	// Send raw invalid JSON (cannot use rpc() which compacts valid JSON only).
	var out bytes.Buffer
	srv.serve(strings.NewReader("not valid json\n"), &out)
	var resp jsonRPCMessage
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("decode response: %v\nraw: %s", err, out.String())
	}
	if resp.Error == nil {
		t.Fatal("expected parse error, got nil")
	}
	if resp.Error.Code != codeParseError {
		t.Errorf("error code = %d, want %d", resp.Error.Code, codeParseError)
	}
}

func TestMCPNotificationIsIgnored(t *testing.T) {
	// Notifications have no id — server should produce no output for them.
	srv := newTestServer(t, `actions: {}`)
	var out bytes.Buffer
	// Send notification followed by a proper request so we can tell serve() processed input.
	input := `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" +
		`{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}` + "\n"
	srv.serve(strings.NewReader(input), &out)
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected exactly 1 response line (for ping), got %d: %q", len(lines), out.String())
	}
}

// ── tools/list ───────────────────────────────────────────────────────────────

func TestMCPToolsList(t *testing.T) {
	srv := newTestServer(t, `actions: {}`)
	resp := rpc(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	var result mcpToolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}

	wantTools := []string{
		"agentctl_install_package",
		"agentctl_run_code",
		"agentctl_write_file",
		"agentctl_access_secret",
		"agentctl_call_external_api",
	}
	if len(result.Tools) != len(wantTools) {
		t.Fatalf("got %d tools, want %d", len(result.Tools), len(wantTools))
	}
	for i, want := range wantTools {
		if result.Tools[i].Name != want {
			t.Errorf("tools[%d].name = %q, want %q", i, result.Tools[i].Name, want)
		}
		if result.Tools[i].Description == "" {
			t.Errorf("tools[%d].description is empty", i)
		}
		// Verify inputSchema is valid JSON with type:object.
		var schema map[string]any
		if err := json.Unmarshal(result.Tools[i].InputSchema, &schema); err != nil {
			t.Errorf("tools[%d].inputSchema is not valid JSON: %v", i, err)
		}
		if schema["type"] != "object" {
			t.Errorf("tools[%d].inputSchema.type = %q, want object", i, schema["type"])
		}
	}
}

// ── tools/call: allow verdicts ───────────────────────────────────────────────

func TestMCPToolCallAllowInstallPackage(t *testing.T) {
	srv := newTestServer(t, `actions: {}`) // permissive policy
	resp := rpc(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{
		"name":"agentctl_install_package",
		"arguments":{"manager":"pip","package":"requests","reason":"HTTP client"}
	}}`)

	result := mustDecodeToolResult(t, resp)
	if result.IsError {
		t.Errorf("isError = true, want false (content: %v)", result.Content)
	}
	if len(result.Content) == 0 || result.Content[0].Type != "text" {
		t.Error("expected text content")
	}
	// Result text should be parseable JSON with verdict:allow.
	var verdict map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &verdict); err != nil {
		t.Fatalf("result text is not JSON: %v\ntext: %s", err, result.Content[0].Text)
	}
	if verdict["verdict"] != "allow" {
		t.Errorf("verdict = %q, want allow", verdict["verdict"])
	}
	if verdict["trace_id"] == "" {
		t.Error("trace_id is empty")
	}
}

func TestMCPToolCallAllowRunCode(t *testing.T) {
	srv := newTestServer(t, `actions: {}`)
	resp := rpc(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{
		"name":"agentctl_run_code",
		"arguments":{"language":"bash","command":"ls -la","reason":"list files"}
	}}`)

	result := mustDecodeToolResult(t, resp)
	if result.IsError {
		t.Errorf("isError = true: %v", result.Content)
	}
}

func TestMCPToolCallAllowCallExternalAPIDerivesDomain(t *testing.T) {
	srv := newTestServer(t, `
actions:
  call_external_api:
    allowed_domains:
      - "api.openai.com"`)
	resp := rpc(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{
		"name":"agentctl_call_external_api",
		"arguments":{"url":"https://api.openai.com/v1/chat","method":"POST","reason":"LLM call"}
	}}`)
	// domain not provided — should be derived from URL and allowed.
	result := mustDecodeToolResult(t, resp)
	if result.IsError {
		t.Errorf("isError = true, want false (content: %v)", result.Content)
	}
}

// ── tools/call: deny verdicts ────────────────────────────────────────────────

func TestMCPToolCallDenyInstallPackageMissingHash(t *testing.T) {
	srv := newTestServer(t, `
actions:
  install_package:
    require_hashes: true`)
	resp := rpc(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{
		"name":"agentctl_install_package",
		"arguments":{"manager":"pip","package":"requests","reason":"HTTP client"}
	}}`)

	result := mustDecodeToolResult(t, resp)
	if !result.IsError {
		t.Error("isError = false, want true for denied install")
	}
	if !strings.Contains(result.Content[0].Text, "agentctl deny") {
		t.Errorf("content = %q, want agentctl deny prefix", result.Content[0].Text)
	}
}

func TestMCPToolCallDenyBlockedDomain(t *testing.T) {
	srv := newTestServer(t, `
actions:
  call_external_api:
    blocked_domains:
      - "evil.com"`)
	resp := rpc(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{
		"name":"agentctl_call_external_api",
		"arguments":{"url":"https://evil.com/exfil","method":"POST","reason":"test"}
	}}`)

	result := mustDecodeToolResult(t, resp)
	if !result.IsError {
		t.Error("isError = false, want true for blocked domain")
	}
}

// ── tools/call: escalate verdict ─────────────────────────────────────────────

func TestMCPToolCallEscalateSecret(t *testing.T) {
	srv := newTestServer(t, `
actions:
  access_secret:
    require_approval: always`)
	resp := rpc(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{
		"name":"agentctl_access_secret",
		"arguments":{"name":"OPENAI_API_KEY","reason":"need credentials"}
	}}`)

	result := mustDecodeToolResult(t, resp)
	if !result.IsError {
		t.Error("isError = false, want true for escalated secret")
	}
	if !strings.Contains(result.Content[0].Text, "agentctl escalate") {
		t.Errorf("content = %q, want agentctl escalate prefix", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "agentctl approval approve") {
		t.Errorf("content = %q, want approval approve hint", result.Content[0].Text)
	}
}

// ── tools/call: invalid arguments ────────────────────────────────────────────

func TestMCPToolCallUnknownTool(t *testing.T) {
	srv := newTestServer(t, `actions: {}`)
	resp := rpc(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{
		"name":"not_a_real_tool",
		"arguments":{}
	}}`)
	if resp.Error == nil {
		t.Fatal("expected JSON-RPC error for unknown tool, got nil")
	}
	if resp.Error.Code != codeInvalidParams {
		t.Errorf("error code = %d, want %d", resp.Error.Code, codeInvalidParams)
	}
}

func TestMCPToolCallMissingRequiredField(t *testing.T) {
	srv := newTestServer(t, `actions: {}`)
	// package field is missing
	resp := rpc(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{
		"name":"agentctl_install_package",
		"arguments":{"manager":"pip","reason":"test"}
	}}`)
	if resp.Error == nil {
		t.Fatal("expected error for missing package field")
	}
}

// ── mcpArgsToActionRequest unit tests ────────────────────────────────────────

func TestMCPArgsToActionRequest(t *testing.T) {
	tests := []struct {
		name       string
		toolName   string
		args       string
		wantAction schema.Action
		wantErr    bool
	}{
		{
			name:       "install_package",
			toolName:   "agentctl_install_package",
			args:       `{"manager":"npm","package":"lodash","reason":"utility"}`,
			wantAction: schema.ActionInstallPackage,
		},
		{
			name:       "run_code",
			toolName:   "agentctl_run_code",
			args:       `{"language":"python","command":"print(1)","reason":"test"}`,
			wantAction: schema.ActionRunCode,
		},
		{
			name:       "write_file",
			toolName:   "agentctl_write_file",
			args:       `{"path":"/tmp/f.txt","operation":"create","reason":"output"}`,
			wantAction: schema.ActionWriteFile,
		},
		{
			name:       "access_secret",
			toolName:   "agentctl_access_secret",
			args:       `{"name":"DB_PASSWORD","reason":"connect"}`,
			wantAction: schema.ActionAccessSecret,
		},
		{
			name:       "call_external_api",
			toolName:   "agentctl_call_external_api",
			args:       `{"url":"https://api.github.com","method":"GET","reason":"fetch"}`,
			wantAction: schema.ActionCallExternalAPI,
		},
		{
			name:     "unknown tool returns error",
			toolName: "agentctl_unknown",
			args:     `{}`,
			wantErr:  true,
		},
		{
			name:     "install_package: missing package returns error",
			toolName: "agentctl_install_package",
			args:     `{"manager":"pip","reason":"test"}`,
			wantErr:  true,
		},
		{
			name:     "run_code: missing command returns error",
			toolName: "agentctl_run_code",
			args:     `{"language":"bash","reason":"test"}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := mcpArgsToActionRequest(tt.toolName, json.RawMessage(tt.args))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if req.Action != tt.wantAction {
				t.Errorf("action = %q, want %q", req.Action, tt.wantAction)
			}
			if req.Reason == "" {
				t.Error("reason is empty")
			}
		})
	}
}

func TestMCPCallExternalAPIDerivesDomain(t *testing.T) {
	req, err := mcpArgsToActionRequest("agentctl_call_external_api", json.RawMessage(
		`{"url":"https://api.openai.com/v1/chat","method":"POST","reason":"test"}`,
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var params schema.CallExternalAPIParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		t.Fatalf("decode params: %v", err)
	}
	if params.Domain != "api.openai.com" {
		t.Errorf("domain = %q, want api.openai.com", params.Domain)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func mustDecodeToolResult(t *testing.T, resp jsonRPCMessage) mcpToolCallResult {
	t.Helper()
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: code=%d msg=%s", resp.Error.Code, resp.Error.Message)
	}
	var result mcpToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v\nraw: %s", err, string(resp.Result))
	}
	if len(result.Content) == 0 {
		t.Fatal("tool result content is empty")
	}
	return result
}
