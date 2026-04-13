package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chocks/agentctl/pkg/attach"
	"github.com/chocks/agentctl/pkg/config"
	"github.com/chocks/agentctl/pkg/trace"
	"gopkg.in/yaml.v3"
)

func cmdDoctor(paths config.Paths) {
	os.Exit(runDoctor(paths, os.Stdout, os.Stderr))
}

type policyFile struct {
	Actions map[string]any `yaml:"actions"`
}

func runDoctor(paths config.Paths, stdout, stderr io.Writer) int {
	_ = stderr
	hasErrors := false

	printDoctorLine(stdout, "agentctl "+version, "ok", "")

	policyStatus, policyDetail, policyErr := doctorPolicyStatus(paths.Policy)
	printDoctorLine(stdout, paths.Policy, policyStatus, policyDetail)
	if policyErr != nil {
		hasErrors = true
	}

	traceStatus, traceDetail, traceErr := doctorTraceStatus(paths.Traces)
	printDoctorLine(stdout, paths.Traces, traceStatus, traceDetail)
	if traceErr != nil {
		hasErrors = true
	}

	approvalStatus, approvalDetail, approvalErr := doctorApprovalStatus(paths.Approvals)
	printDoctorLine(stdout, paths.Approvals, approvalStatus, approvalDetail)
	if approvalErr != nil {
		hasErrors = true
	}

	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Agents:")

	agentChecks := []struct {
		agent attach.Agent
		mode  string
	}{
		{agent: attach.AgentClaudeCode, mode: "hook"},
		{agent: attach.AgentCodex, mode: "mcp"},
	}

	for _, check := range agentChecks {
		result, err := attach.Status(check.agent)
		if err != nil {
			hasErrors = true
			_, _ = fmt.Fprintf(stdout, "  %-12s %-6s %-32s error (%v)\n", check.agent, check.mode, "-", err)
			continue
		}
		_, _ = fmt.Fprintf(stdout, "  %-12s %-6s %-32s %s\n", result.Agent, check.mode, result.ConfigPath, result.Action)
	}

	if hasErrors {
		return 1
	}
	return 0
}

func doctorPolicyStatus(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "warn", `run "agentctl attach" to set up`, nil
	}
	if err != nil {
		return "error", err.Error(), err
	}

	var doc policyFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "error", fmt.Sprintf("parse error: %v", err), err
	}

	return "ok", fmt.Sprintf("(%d actions configured)", len(doc.Actions)), nil
}

func doctorTraceStatus(path string) (string, string, error) {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "ok", "(0 traces)", nil
	}
	if err != nil {
		return "error", err.Error(), err
	}

	traces, err := trace.ReadTraces(path, trace.TraceFilter{})
	if err != nil {
		return "error", err.Error(), err
	}
	return "ok", fmt.Sprintf("(%d traces)", len(traces)), nil
}

func doctorApprovalStatus(path string) (string, string, error) {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "ok", "(0 pending)", nil
	}
	if err != nil {
		return "error", err.Error(), err
	}

	records, err := readApprovals(path, approvalStatusPending)
	if err != nil {
		return "error", err.Error(), err
	}
	return "ok", fmt.Sprintf("(%d pending)", len(records)), nil
}

func printDoctorLine(w io.Writer, label, status, detail string) {
	if detail != "" && !strings.HasPrefix(detail, "(") {
		_, _ = fmt.Fprintf(w, "%-52s %-5s %s\n", label, status, detail)
		return
	}
	_, _ = fmt.Fprintf(w, "%-52s %-5s %s\n", label, status, detail)
}
