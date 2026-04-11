package main

import (
	"encoding/json"
	"testing"

	"github.com/chocks/agentctl/pkg/schema"
)

func TestClaudeEventToActionRequest(t *testing.T) {
	tests := []struct {
		name       string
		event      claudeHookEvent
		wantSkip   bool
		wantAction schema.Action
		// Optional field checks on the mapped params
		checkParams func(t *testing.T, params json.RawMessage)
	}{
		// ── Bash: package installs ───────────────────────────────────────────
		{
			name:       "pip install",
			event:      bashEvent("pip install requests"),
			wantAction: schema.ActionInstallPackage,
			checkParams: func(t *testing.T, raw json.RawMessage) {
				var p schema.InstallPackageParams
				mustUnmarshal(t, raw, &p)
				if p.Manager != "pip" {
					t.Errorf("manager = %q, want pip", p.Manager)
				}
				if p.Package != "requests" {
					t.Errorf("package = %q, want requests", p.Package)
				}
			},
		},
		{
			name:       "pip3 install",
			event:      bashEvent("pip3 install numpy==1.26.0"),
			wantAction: schema.ActionInstallPackage,
			checkParams: func(t *testing.T, raw json.RawMessage) {
				var p schema.InstallPackageParams
				mustUnmarshal(t, raw, &p)
				if p.Manager != "pip" {
					t.Errorf("manager = %q, want pip", p.Manager)
				}
				if p.Package != "numpy==1.26.0" {
					t.Errorf("package = %q, want numpy==1.26.0", p.Package)
				}
			},
		},
		{
			name:       "npm install",
			event:      bashEvent("npm install express"),
			wantAction: schema.ActionInstallPackage,
			checkParams: func(t *testing.T, raw json.RawMessage) {
				var p schema.InstallPackageParams
				mustUnmarshal(t, raw, &p)
				if p.Manager != "npm" {
					t.Errorf("manager = %q, want npm", p.Manager)
				}
				if p.Package != "express" {
					t.Errorf("package = %q, want express", p.Package)
				}
			},
		},
		{
			name:       "yarn add",
			event:      bashEvent("yarn add lodash"),
			wantAction: schema.ActionInstallPackage,
		},
		{
			name:       "cargo add",
			event:      bashEvent("cargo add serde"),
			wantAction: schema.ActionInstallPackage,
			checkParams: func(t *testing.T, raw json.RawMessage) {
				var p schema.InstallPackageParams
				mustUnmarshal(t, raw, &p)
				if p.Manager != "cargo" {
					t.Errorf("manager = %q, want cargo", p.Manager)
				}
			},
		},
		{
			name:       "go get",
			event:      bashEvent("go get github.com/user/pkg"),
			wantAction: schema.ActionInstallPackage,
			checkParams: func(t *testing.T, raw json.RawMessage) {
				var p schema.InstallPackageParams
				mustUnmarshal(t, raw, &p)
				if p.Manager != "go" {
					t.Errorf("manager = %q, want go", p.Manager)
				}
			},
		},
		// ── Bash: general code execution ─────────────────────────────────────
		{
			name:       "bash command",
			event:      bashEvent("ls -la /tmp"),
			wantAction: schema.ActionRunCode,
			checkParams: func(t *testing.T, raw json.RawMessage) {
				var p schema.RunCodeParams
				mustUnmarshal(t, raw, &p)
				if p.Language != "bash" {
					t.Errorf("language = %q, want bash", p.Language)
				}
				if p.Command != "ls -la /tmp" {
					t.Errorf("command = %q, want ls -la /tmp", p.Command)
				}
				if p.Network {
					t.Error("network = true, want false for non-network command")
				}
			},
		},
		{
			name:       "curl command detected as network",
			event:      bashEvent("curl https://example.com/data.json"),
			wantAction: schema.ActionRunCode,
			checkParams: func(t *testing.T, raw json.RawMessage) {
				var p schema.RunCodeParams
				mustUnmarshal(t, raw, &p)
				if !p.Network {
					t.Error("network = false, want true for curl command")
				}
			},
		},
		// ── Write ─────────────────────────────────────────────────────────────
		{
			name: "Write tool",
			event: claudeHookEvent{
				ToolName:  "Write",
				ToolInput: rawJSON(`{"file_path":"/tmp/out.txt","content":"hello"}`),
			},
			wantAction: schema.ActionWriteFile,
			checkParams: func(t *testing.T, raw json.RawMessage) {
				var p schema.WriteFileParams
				mustUnmarshal(t, raw, &p)
				if p.Path != "/tmp/out.txt" {
					t.Errorf("path = %q, want /tmp/out.txt", p.Path)
				}
				if p.Operation != "overwrite" {
					t.Errorf("operation = %q, want overwrite", p.Operation)
				}
				if p.SizeBytes != 5 {
					t.Errorf("size_bytes = %d, want 5", p.SizeBytes)
				}
			},
		},
		// ── Edit ─────────────────────────────────────────────────────────────
		{
			name: "Edit tool",
			event: claudeHookEvent{
				ToolName:  "Edit",
				ToolInput: rawJSON(`{"file_path":"/src/main.go","old_string":"foo","new_string":"bar"}`),
			},
			wantAction: schema.ActionWriteFile,
			checkParams: func(t *testing.T, raw json.RawMessage) {
				var p schema.WriteFileParams
				mustUnmarshal(t, raw, &p)
				if p.Path != "/src/main.go" {
					t.Errorf("path = %q, want /src/main.go", p.Path)
				}
			},
		},
		{
			name: "MultiEdit tool",
			event: claudeHookEvent{
				ToolName:  "MultiEdit",
				ToolInput: rawJSON(`{"file_path":"/src/util.go","new_string":"updated"}`),
			},
			wantAction: schema.ActionWriteFile,
		},
		// ── WebFetch ──────────────────────────────────────────────────────────
		{
			name: "WebFetch tool",
			event: claudeHookEvent{
				ToolName:  "WebFetch",
				ToolInput: rawJSON(`{"url":"https://api.github.com/repos/foo/bar","prompt":"summarize"}`),
			},
			wantAction: schema.ActionCallExternalAPI,
			checkParams: func(t *testing.T, raw json.RawMessage) {
				var p schema.CallExternalAPIParams
				mustUnmarshal(t, raw, &p)
				if p.Domain != "api.github.com" {
					t.Errorf("domain = %q, want api.github.com", p.Domain)
				}
				if p.Method != "GET" {
					t.Errorf("method = %q, want GET", p.Method)
				}
			},
		},
		// ── Skipped tools ─────────────────────────────────────────────────────
		{
			name:     "Read tool is skipped",
			event:    claudeHookEvent{ToolName: "Read", ToolInput: rawJSON(`{"file_path":"/etc/hosts"}`)},
			wantSkip: true,
		},
		{
			name:     "Glob tool is skipped",
			event:    claudeHookEvent{ToolName: "Glob", ToolInput: rawJSON(`{"pattern":"**/*.go"}`)},
			wantSkip: true,
		},
		{
			name:     "Grep tool is skipped",
			event:    claudeHookEvent{ToolName: "Grep", ToolInput: rawJSON(`{"pattern":"TODO"}`)},
			wantSkip: true,
		},
		{
			name:     "Agent tool is skipped",
			event:    claudeHookEvent{ToolName: "Agent", ToolInput: rawJSON(`{}`)},
			wantSkip: true,
		},
		// ── Edge cases ────────────────────────────────────────────────────────
		{
			name:     "Bash with empty command is skipped",
			event:    bashEvent(""),
			wantSkip: true,
		},
		{
			name:     "Write with empty path is skipped",
			event:    claudeHookEvent{ToolName: "Write", ToolInput: rawJSON(`{"file_path":"","content":"x"}`)},
			wantSkip: true,
		},
		{
			name:     "WebFetch with invalid URL is skipped",
			event:    claudeHookEvent{ToolName: "WebFetch", ToolInput: rawJSON(`{"url":"not-a-url"}`)},
			wantSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, skip := claudeEventToActionRequest(tt.event)
			if skip != tt.wantSkip {
				t.Fatalf("skip = %v, want %v", skip, tt.wantSkip)
			}
			if tt.wantSkip {
				return
			}
			if req.Action != tt.wantAction {
				t.Errorf("action = %q, want %q", req.Action, tt.wantAction)
			}
			if req.Reason == "" {
				t.Error("reason is empty")
			}
			if tt.checkParams != nil {
				tt.checkParams(t, req.Params)
			}
		})
	}
}

func TestDetectPackageInstall(t *testing.T) {
	tests := []struct {
		command     string
		wantPkg     string
		wantManager string
		wantOK      bool
	}{
		{"pip install requests", "requests", "pip", true},
		{"pip3 install numpy==1.26.0", "numpy==1.26.0", "pip", true},
		{"python -m pip install boto3", "boto3", "pip", true},
		{"npm install express", "express", "npm", true},
		{"npm i lodash", "lodash", "npm", true},
		{"yarn add react", "react", "npm", true},
		{"pnpm add vue", "vue", "npm", true},
		{"cargo add serde", "serde", "cargo", true},
		{"cargo install ripgrep", "ripgrep", "cargo", true},
		{"go get github.com/user/pkg", "github.com/user/pkg", "go", true},
		{"go install golang.org/x/tools/gopls@latest", "golang.org/x/tools/gopls@latest", "go", true},
		// Not a coding package manager
		{"apt-get install curl", "", "", false},
		{"brew install jq", "", "", false},
		// Flag before package name
		{"pip install --upgrade pip", "", "", false},
		{"npm install --save-dev jest", "", "", false},
		// Unrelated commands
		{"ls -la", "", "", false},
		{"echo hello", "", "", false},
		{"git commit -m 'msg'", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			pkg, mgr, ok := detectPackageInstall(tt.command)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v (pkg=%q mgr=%q)", ok, tt.wantOK, pkg, mgr)
			}
			if !ok {
				return
			}
			if pkg != tt.wantPkg {
				t.Errorf("pkg = %q, want %q", pkg, tt.wantPkg)
			}
			if mgr != tt.wantManager {
				t.Errorf("manager = %q, want %q", mgr, tt.wantManager)
			}
		})
	}
}

func TestLooksLikeNetworkCommand(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"curl https://example.com", true},
		{"wget http://example.com/file", true},
		{"ssh user@host", true},
		{"ls -la", false},
		{"python script.py", false},
		{"echo https://example.com", true}, // contains https://
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := looksLikeNetworkCommand(tt.command)
			if got != tt.want {
				t.Errorf("looksLikeNetworkCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

// helpers

func bashEvent(command string) claudeHookEvent {
	return claudeHookEvent{
		ToolName:  "Bash",
		ToolInput: rawJSON(`{"command":` + jsonString(command) + `}`),
	}
}

func rawJSON(s string) json.RawMessage {
	return json.RawMessage(s)
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func mustUnmarshal(t *testing.T, raw json.RawMessage, dst any) {
	t.Helper()
	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
}
