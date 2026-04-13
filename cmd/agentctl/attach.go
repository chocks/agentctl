package main

import (
	"fmt"
	"io"
	"os"

	"github.com/chocks/agentctl/pkg/attach"
	"github.com/chocks/agentctl/pkg/config"
)

func cmdAttach(paths config.Paths) {
	os.Exit(runAttach(paths, os.Args[2:], os.Stdout, os.Stderr))
}

func cmdDetach() {
	os.Exit(runDetach(os.Args[2:], os.Stdout, os.Stderr))
}

func runAttach(paths config.Paths, args []string, stdout, stderr io.Writer) int {
	agent, ok := parseAgentArg(args)
	if !ok {
		_, _ = fmt.Fprintln(stderr, "usage: agentctl attach <claude-code|codex>")
		return 1
	}

	if err := paths.EnsureHome(); err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if err := paths.WriteDefaultPolicy(); err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	result, err := attach.Attach(agent)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "%s %s\n", result.Agent, result.Action)
	_, _ = fmt.Fprintf(stdout, "config: %s\n", result.ConfigPath)
	_, _ = fmt.Fprintf(stdout, "policy: %s\n", paths.Policy)
	return 0
}

func runDetach(args []string, stdout, stderr io.Writer) int {
	agent, ok := parseAgentArg(args)
	if !ok {
		_, _ = fmt.Fprintln(stderr, "usage: agentctl detach <claude-code|codex>")
		return 1
	}

	result, err := attach.Detach(agent)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "%s %s\n", result.Agent, result.Action)
	_, _ = fmt.Fprintf(stdout, "config: %s\n", result.ConfigPath)
	return 0
}

func parseAgentArg(args []string) (attach.Agent, bool) {
	if len(args) != 1 {
		return "", false
	}

	agent := attach.Agent(args[0])
	switch agent {
	case attach.AgentClaudeCode, attach.AgentCodex:
		return agent, true
	default:
		return "", false
	}
}
