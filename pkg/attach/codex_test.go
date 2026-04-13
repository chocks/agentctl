package attach

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttachCodex_FreshConfig(t *testing.T) {
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex")

	result, err := attachCodex(codexDir)
	if err != nil {
		t.Fatalf("attachCodex() error = %v", err)
	}
	if result.Action != "attached" {
		t.Errorf("Action = %q, want attached", result.Action)
	}

	data, err := os.ReadFile(filepath.Join(codexDir, "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "[mcp_servers.agentctl]") {
		t.Errorf("config.toml missing [mcp_servers.agentctl] section:\n%s", content)
	}
	if !strings.Contains(content, `command = "agentctl"`) {
		t.Errorf("config.toml missing command:\n%s", content)
	}
}

func TestAttachCodex_Idempotent(t *testing.T) {
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex")

	if _, err := attachCodex(codexDir); err != nil {
		t.Fatal(err)
	}
	result, err := attachCodex(codexDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "already attached" {
		t.Errorf("Action = %q, want 'already attached'", result.Action)
	}
}

func TestAttachCodex_PreservesExistingConfig(t *testing.T) {
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		t.Fatal(err)
	}

	existing := "model = \"o3\"\n\n[history]\nmax_entries = 100\n"
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := attachCodex(codexDir); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(codexDir, "config.toml"))
	content := string(data)
	if !strings.Contains(content, `model = "o3"`) {
		t.Errorf("existing model setting was lost:\n%s", content)
	}
	if !strings.Contains(content, "[mcp_servers.agentctl]") {
		t.Errorf("agentctl MCP server not added:\n%s", content)
	}
}

func TestDetachCodex(t *testing.T) {
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex")

	if _, err := attachCodex(codexDir); err != nil {
		t.Fatal(err)
	}

	result, err := detachCodex(codexDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "detached" {
		t.Errorf("Action = %q, want detached", result.Action)
	}

	data, _ := os.ReadFile(filepath.Join(codexDir, "config.toml"))
	if strings.Contains(string(data), "agentctl") {
		t.Errorf("agentctl still present after detach:\n%s", string(data))
	}
}

func TestDetachCodex_NotAttached(t *testing.T) {
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex")

	result, err := detachCodex(codexDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "not attached" {
		t.Errorf("Action = %q, want 'not attached'", result.Action)
	}
}

func TestStatusCodex(t *testing.T) {
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex")

	result, _ := statusCodex(codexDir)
	if result.Action != "not attached" {
		t.Errorf("Action = %q, want 'not attached'", result.Action)
	}

	attachCodex(codexDir) //nolint:errcheck
	result, _ = statusCodex(codexDir)
	if result.Action != "attached" {
		t.Errorf("Action = %q, want attached", result.Action)
	}
}
