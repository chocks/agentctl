// Package attach configures agentctl into supported coding agents'
// integration points by reading and modifying their config files.
package attach

import (
	"fmt"
	"os"
	"path/filepath"
)

// Agent identifies a supported coding agent.
type Agent string

const (
	AgentClaudeCode Agent = "claude-code"
	AgentCodex      Agent = "codex"
)

// Result describes what happened during an attach/detach/status operation.
type Result struct {
	Agent      Agent
	ConfigPath string
	Action     string // "attached", "detached", "already attached", "not attached"
}

// Attach configures agentctl into the given agent's integration point.
func Attach(agent Agent) (Result, error) {
	switch agent {
	case AgentClaudeCode:
		claudeDir, err := claudeConfigDir()
		if err != nil {
			return Result{}, err
		}
		return attachClaudeCode(claudeDir)
	case AgentCodex:
		codexDir, err := codexConfigDir()
		if err != nil {
			return Result{}, err
		}
		return attachCodex(codexDir)
	default:
		return Result{}, fmt.Errorf("unknown agent: %s (supported: claude-code, codex)", agent)
	}
}

// Detach removes agentctl from the given agent's integration point.
func Detach(agent Agent) (Result, error) {
	switch agent {
	case AgentClaudeCode:
		claudeDir, err := claudeConfigDir()
		if err != nil {
			return Result{}, err
		}
		return detachClaudeCode(claudeDir)
	case AgentCodex:
		codexDir, err := codexConfigDir()
		if err != nil {
			return Result{}, err
		}
		return detachCodex(codexDir)
	default:
		return Result{}, fmt.Errorf("unknown agent: %s (supported: claude-code, codex)", agent)
	}
}

// Status checks whether agentctl is configured in the given agent.
func Status(agent Agent) (Result, error) {
	switch agent {
	case AgentClaudeCode:
		claudeDir, err := claudeConfigDir()
		if err != nil {
			return Result{}, err
		}
		return statusClaudeCode(claudeDir)
	case AgentCodex:
		codexDir, err := codexConfigDir()
		if err != nil {
			return Result{}, err
		}
		return statusCodex(codexDir)
	default:
		return Result{}, fmt.Errorf("unknown agent: %s", agent)
	}
}

func claudeConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

func codexConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".codex"), nil
}
