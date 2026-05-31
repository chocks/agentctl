package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLITraceVerify proves the end-to-end integrity path: the gate writes a
// hash-chained trace, `trace verify` confirms it, and tampering with a recorded
// decision is detected and reported at the exact record.
func TestCLITraceVerify(t *testing.T) {
	repoRoot := repoRootFromTestDir(t)
	workdir := t.TempDir()
	binaryPath := filepath.Join(workdir, "agentctl")
	fakeHome := filepath.Join(workdir, "fakehome")
	agentctlDir := filepath.Join(fakeHome, ".agentctl")
	if err := os.MkdirAll(agentctlDir, 0755); err != nil {
		t.Fatal(err)
	}

	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/agentctl")
	build.Dir = repoRoot
	build.Env = append(inheritedGoEnv(), "HOME="+os.Getenv("HOME"))
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build error = %v\n%s", err, output)
	}

	// Produce a few chained records via the gate.
	inputs := []string{
		`{"action":"write_file","params":{"path":"/tmp/a.txt","operation":"create"},"reason":"one"}`,
		`{"action":"run_code","params":{"language":"bash","command":"echo hi"},"reason":"two"}`,
		`{"action":"write_file","params":{"path":"/tmp/b.txt","operation":"create"},"reason":"three"}`,
	}
	for _, in := range inputs {
		if code, stdout, stderr := runCLI(t, workdir, binaryPath, fakeHome, in, "gate", "--session", "s1"); code > 1 {
			t.Fatalf("gate exit=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	}

	tracePath := filepath.Join(agentctlDir, "traces.jsonl")

	// A clean chain verifies and exits 0.
	code, stdout, stderr := runCLI(t, workdir, binaryPath, fakeHome, "", "trace", "verify")
	if code != 0 {
		t.Fatalf("verify clean chain exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "VERIFIED") {
		t.Fatalf("expected VERIFIED, got stdout=%q stderr=%q", stdout, stderr)
	}

	// Tamper: flip a recorded verdict in place without updating its hash.
	raw, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(raw), `"verdict":"allow"`, `"verdict":"deny"`, 1)
	if tampered == string(raw) {
		t.Fatalf("test setup: expected an allow verdict to tamper with, got:\n%s", raw)
	}
	if err := os.WriteFile(tracePath, []byte(tampered), 0644); err != nil {
		t.Fatal(err)
	}

	// The broken chain must be detected and exit non-zero.
	code, stdout, stderr = runCLI(t, workdir, binaryPath, fakeHome, "", "trace", "verify")
	if code == 0 {
		t.Fatalf("expected non-zero exit for tampered chain, got 0 stdout=%q stderr=%q", stdout, stderr)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, "BROKEN") || !strings.Contains(combined, "seq") {
		t.Fatalf("expected a BROKEN ... seq report, got stdout=%q stderr=%q", stdout, stderr)
	}
}
