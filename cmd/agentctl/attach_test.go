package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chocks/agentctl/pkg/config"
)

func TestRunAttach(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantCode        int
		wantStdoutParts []string
		wantStderrPart  string
		checkFiles      func(t *testing.T, tempHome string, paths config.Paths)
	}{
		{
			name:            "claude-code bootstraps policy and settings",
			args:            []string{"claude-code"},
			wantCode:        0,
			wantStdoutParts: []string{"claude-code attached", "config:", "policy:"},
			checkFiles: func(t *testing.T, tempHome string, paths config.Paths) {
				t.Helper()
				if _, err := os.Stat(paths.Policy); err != nil {
					t.Fatalf("policy file missing: %v", err)
				}
				settingsPath := filepath.Join(tempHome, ".claude", "settings.json")
				data, err := os.ReadFile(settingsPath)
				if err != nil {
					t.Fatalf("settings.json missing: %v", err)
				}
				if !strings.Contains(string(data), "agentctl hook claude-code") {
					t.Fatalf("settings.json missing agentctl hook:\n%s", data)
				}
			},
		},
		{
			name:           "invalid agent is rejected",
			args:           []string{"cursor"},
			wantCode:       1,
			wantStderrPart: "usage: agentctl attach <claude-code|codex>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempHome := t.TempDir()
			t.Setenv("HOME", tempHome)

			paths := config.NewPaths(filepath.Join(tempHome, ".agentctl"))
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := runAttach(paths, tt.args, &stdout, &stderr)
			if code != tt.wantCode {
				t.Fatalf("runAttach() code = %d, want %d", code, tt.wantCode)
			}

			for _, part := range tt.wantStdoutParts {
				if !strings.Contains(stdout.String(), part) {
					t.Fatalf("stdout missing %q:\n%s", part, stdout.String())
				}
			}
			if tt.wantStderrPart != "" && !strings.Contains(stderr.String(), tt.wantStderrPart) {
				t.Fatalf("stderr missing %q:\n%s", tt.wantStderrPart, stderr.String())
			}
			if tt.checkFiles != nil {
				tt.checkFiles(t, tempHome, paths)
			}
		})
	}
}

func TestRunDetach(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	paths := config.NewPaths(filepath.Join(tempHome, ".agentctl"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := runAttach(paths, []string{"codex"}, &stdout, &stderr); code != 0 {
		t.Fatalf("runAttach() code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	code := runDetach([]string{"codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runDetach() code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "codex detached") {
		t.Fatalf("stdout missing detach result:\n%s", stdout.String())
	}

	configPath := filepath.Join(tempHome, ".codex", "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config.toml missing after detach: %v", err)
	}
	if strings.Contains(string(data), "agentctl") {
		t.Fatalf("config.toml still contains agentctl after detach:\n%s", data)
	}
}
