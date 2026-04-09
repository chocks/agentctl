package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIEndToEndGateTraceReplay(t *testing.T) {
	repoRoot := repoRootFromTestDir(t)
	workdir := t.TempDir()
	binaryPath := filepath.Join(workdir, "agentctl")

	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/agentctl")
	build.Dir = repoRoot
	build.Env = append(os.Environ(), inheritedGoEnv()...)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build error = %v\n%s", err, output)
	}

	writeFile(t, filepath.Join(workdir, "agentctl.policy.yaml"), `
actions:
  access_secret:
    require_approval: always
  call_external_api:
    allowed_domains:
      - "api.openai.com"
`)

	allowInput := `{"action":"call_external_api","params":{"url":"https://api.openai.com/v1/responses","method":"POST"},"reason":"call provider"}`
	_, stdout, stderr := runCLI(t, workdir, binaryPath, allowInput, "gate", "--session", "demo-1")
	if !strings.Contains(stdout, `"verdict": "allow"`) {
		t.Fatalf("expected allow output, got stdout=%q stderr=%q", stdout, stderr)
	}

	escalateInput := `{"action":"access_secret","params":{"name":"OPENAI_API_KEY"},"reason":"need credentials"}`
	exitCode, stdout, stderr := runCLI(t, workdir, binaryPath, escalateInput, "gate", "--session", "demo-1")
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for escalation, got %d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, `"verdict": "escalate"`) {
		t.Fatalf("expected escalate output, got stdout=%q stderr=%q", stdout, stderr)
	}

	exitCode, stdout, stderr = runCLI(t, workdir, binaryPath, "", "trace", "search", "--session", "demo-1")
	if exitCode != 0 {
		t.Fatalf("trace search exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, `"action": "call_external_api"`) || !strings.Contains(stdout, `"action": "access_secret"`) {
		t.Fatalf("expected both actions in trace search output, got stdout=%q stderr=%q", stdout, stderr)
	}

	writeFile(t, filepath.Join(workdir, "replay.policy.yaml"), `
actions:
  access_secret:
    require_approval: always
  call_external_api:
    blocked_domains:
      - "api.openai.com"
`)

	exitCode, stdout, stderr = runCLI(t, workdir, binaryPath, "", "replay", "demo-1", "--policy", "replay.policy.yaml")
	if exitCode != 0 {
		t.Fatalf("replay exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}

	var replay struct {
		SessionID string `json:"session_id"`
		Results   []struct {
			Verdict string `json:"verdict"`
			Request struct {
				Action string `json:"action"`
			} `json:"request"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &replay); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout=%s", err, stdout)
	}

	if replay.SessionID != "demo-1" {
		t.Fatalf("expected session demo-1, got %q", replay.SessionID)
	}
	if len(replay.Results) != 2 {
		t.Fatalf("expected 2 replay results, got %d", len(replay.Results))
	}
	if replay.Results[0].Request.Action != "call_external_api" || replay.Results[0].Verdict != "deny" {
		t.Fatalf("expected first replay result to deny external API call, got %+v", replay.Results[0])
	}
	if replay.Results[1].Request.Action != "access_secret" || replay.Results[1].Verdict != "escalate" {
		t.Fatalf("expected second replay result to escalate secret access, got %+v", replay.Results[1])
	}
}

func repoRootFromTestDir(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}

	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func inheritedGoEnv() []string {
	keys := []string{"GOCACHE", "GOMODCACHE", "GOPATH", "PATH", "HOME"}
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			values = append(values, key+"="+value)
		}
	}
	return values
}

func runCLI(t *testing.T, dir, binaryPath, stdin string, args ...string) (int, string, string) {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = append(os.Environ(), inheritedGoEnv()...)

	stdout, err := cmd.Output()
	if err == nil {
		return 0, string(stdout), ""
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("command error = %v", err)
	}

	return exitErr.ExitCode(), string(stdout), string(exitErr.Stderr)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
}
