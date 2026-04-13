package attach

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func attachCodex(codexDir string) (Result, error) {
	configPath := filepath.Join(codexDir, "config.toml")
	result := Result{
		Agent:      AgentCodex,
		ConfigPath: configPath,
	}

	cfg, err := readTOMLFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, fmt.Errorf("reading %s: %w", configPath, err)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}

	if hasAgentctlMCP(cfg) {
		result.Action = "already attached"
		return result, nil
	}

	addAgentctlMCP(cfg)

	if err := os.MkdirAll(codexDir, 0755); err != nil {
		return result, fmt.Errorf("creating %s: %w", codexDir, err)
	}
	if err := writeTOMLFile(configPath, cfg); err != nil {
		return result, err
	}

	result.Action = "attached"
	return result, nil
}

func detachCodex(codexDir string) (Result, error) {
	configPath := filepath.Join(codexDir, "config.toml")
	result := Result{
		Agent:      AgentCodex,
		ConfigPath: configPath,
	}

	cfg, err := readTOMLFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result.Action = "not attached"
			return result, nil
		}
		return result, fmt.Errorf("reading %s: %w", configPath, err)
	}

	if !hasAgentctlMCP(cfg) {
		result.Action = "not attached"
		return result, nil
	}

	removeAgentctlMCP(cfg)

	if err := writeTOMLFile(configPath, cfg); err != nil {
		return result, err
	}

	result.Action = "detached"
	return result, nil
}

func statusCodex(codexDir string) (Result, error) {
	configPath := filepath.Join(codexDir, "config.toml")
	result := Result{
		Agent:      AgentCodex,
		ConfigPath: configPath,
	}

	cfg, err := readTOMLFile(configPath)
	if err != nil {
		result.Action = "not attached"
		return result, nil
	}

	if hasAgentctlMCP(cfg) {
		result.Action = "attached"
	} else {
		result.Action = "not attached"
	}
	return result, nil
}

func hasAgentctlMCP(cfg map[string]any) bool {
	servers, ok := cfg["mcp_servers"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = servers["agentctl"]
	return ok
}

func addAgentctlMCP(cfg map[string]any) {
	servers, ok := cfg["mcp_servers"].(map[string]any)
	if !ok {
		servers = map[string]any{}
		cfg["mcp_servers"] = servers
	}
	servers["agentctl"] = map[string]any{
		"command": "agentctl",
		"args":    []any{"mcp"},
	}
}

func removeAgentctlMCP(cfg map[string]any) {
	servers, ok := cfg["mcp_servers"].(map[string]any)
	if !ok {
		return
	}
	delete(servers, "agentctl")
	if len(servers) == 0 {
		delete(cfg, "mcp_servers")
	}
}

func readTOMLFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if _, err := toml.Decode(string(data), &result); err != nil {
		return nil, fmt.Errorf("parsing TOML in %s: %w", path, err)
	}
	return result, nil
}

func writeTOMLFile(path string, data map[string]any) error {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encoding TOML: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
