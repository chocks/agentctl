package attach

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAttachClaudeCode_FreshSettings(t *testing.T) {
	home := t.TempDir()
	claudeDir := filepath.Join(home, ".claude")

	result, err := attachClaudeCode(claudeDir)
	if err != nil {
		t.Fatalf("attachClaudeCode() error = %v", err)
	}
	if result.Action != "attached" {
		t.Errorf("Action = %q, want attached", result.Action)
	}

	data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatal("settings.hooks missing")
	}
	preToolUse, ok := hooks["PreToolUse"].([]any)
	if !ok {
		t.Fatal("settings.hooks.PreToolUse missing")
	}
	if len(preToolUse) != 1 {
		t.Fatalf("expected 1 PreToolUse hook, got %d", len(preToolUse))
	}
}

func TestAttachClaudeCode_Idempotent(t *testing.T) {
	home := t.TempDir()
	claudeDir := filepath.Join(home, ".claude")

	if _, err := attachClaudeCode(claudeDir); err != nil {
		t.Fatal(err)
	}
	result, err := attachClaudeCode(claudeDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "already attached" {
		t.Errorf("Action = %q, want 'already attached'", result.Action)
	}
}

func TestAttachClaudeCode_PreservesExistingSettings(t *testing.T) {
	home := t.TempDir()
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	existing := `{"model": "claude-opus-4-6", "permissions": {"allow": ["Read"]}}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := attachClaudeCode(claudeDir); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}

	if settings["model"] != "claude-opus-4-6" {
		t.Error("existing model setting was lost")
	}
	if _, ok := settings["permissions"]; !ok {
		t.Error("existing permissions setting was lost")
	}
}

func TestDetachClaudeCode(t *testing.T) {
	home := t.TempDir()
	claudeDir := filepath.Join(home, ".claude")

	// Attach first
	if _, err := attachClaudeCode(claudeDir); err != nil {
		t.Fatal(err)
	}

	result, err := detachClaudeCode(claudeDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "detached" {
		t.Errorf("Action = %q, want detached", result.Action)
	}

	data, _ := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}

	hooks, _ := settings["hooks"].(map[string]any)
	preToolUse, _ := hooks["PreToolUse"].([]any)
	if len(preToolUse) != 0 {
		t.Errorf("expected 0 PreToolUse hooks after detach, got %d", len(preToolUse))
	}
}

func TestDetachClaudeCode_NotAttached(t *testing.T) {
	home := t.TempDir()
	claudeDir := filepath.Join(home, ".claude")

	result, err := detachClaudeCode(claudeDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "not attached" {
		t.Errorf("Action = %q, want 'not attached'", result.Action)
	}
}

func TestStatusClaudeCode(t *testing.T) {
	home := t.TempDir()
	claudeDir := filepath.Join(home, ".claude")

	result, err := statusClaudeCode(claudeDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "not attached" {
		t.Errorf("Action = %q, want 'not attached'", result.Action)
	}

	if _, err := attachClaudeCode(claudeDir); err != nil {
		t.Fatalf("attachClaudeCode() error = %v", err)
	}
	result, _ = statusClaudeCode(claudeDir)
	if result.Action != "attached" {
		t.Errorf("Action = %q, want attached", result.Action)
	}
}
