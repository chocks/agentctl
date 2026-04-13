package attach

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const agentctlHookCommand = "agentctl hook claude-code"
const agentctlHookMatcher = "Bash|Write|Edit|MultiEdit|WebFetch"

func attachClaudeCode(claudeDir string) (Result, error) {
	settingsPath := filepath.Join(claudeDir, "settings.json")
	result := Result{
		Agent:      AgentClaudeCode,
		ConfigPath: settingsPath,
	}

	settings, err := readJSONFile(settingsPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, fmt.Errorf("reading %s: %w", settingsPath, err)
	}
	if settings == nil {
		settings = map[string]any{}
	}

	if hasAgentctlHook(settings) {
		result.Action = "already attached"
		return result, nil
	}

	addAgentctlHook(settings)

	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return result, fmt.Errorf("creating %s: %w", claudeDir, err)
	}
	if err := writeJSONFile(settingsPath, settings); err != nil {
		return result, err
	}

	result.Action = "attached"
	return result, nil
}

func detachClaudeCode(claudeDir string) (Result, error) {
	settingsPath := filepath.Join(claudeDir, "settings.json")
	result := Result{
		Agent:      AgentClaudeCode,
		ConfigPath: settingsPath,
	}

	settings, err := readJSONFile(settingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result.Action = "not attached"
			return result, nil
		}
		return result, fmt.Errorf("reading %s: %w", settingsPath, err)
	}

	if !hasAgentctlHook(settings) {
		result.Action = "not attached"
		return result, nil
	}

	removeAgentctlHook(settings)

	if err := writeJSONFile(settingsPath, settings); err != nil {
		return result, err
	}

	result.Action = "detached"
	return result, nil
}

func statusClaudeCode(claudeDir string) (Result, error) {
	settingsPath := filepath.Join(claudeDir, "settings.json")
	result := Result{
		Agent:      AgentClaudeCode,
		ConfigPath: settingsPath,
	}

	settings, err := readJSONFile(settingsPath)
	if err != nil {
		result.Action = "not attached"
		return result, nil
	}

	if hasAgentctlHook(settings) {
		result.Action = "attached"
	} else {
		result.Action = "not attached"
	}
	return result, nil
}

func hasAgentctlHook(settings map[string]any) bool {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return false
	}
	preToolUse, ok := hooks["PreToolUse"].([]any)
	if !ok {
		return false
	}
	for _, entry := range preToolUse {
		if isAgentctlHookEntry(entry) {
			return true
		}
	}
	return false
}

func isAgentctlHookEntry(entry any) bool {
	m, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	hooksList, ok := m["hooks"].([]any)
	if !ok {
		return false
	}
	for _, h := range hooksList {
		hm, ok := h.(map[string]any)
		if !ok {
			continue
		}
		if cmd, _ := hm["command"].(string); cmd == agentctlHookCommand {
			return true
		}
	}
	return false
}

func addAgentctlHook(settings map[string]any) {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		hooks = map[string]any{}
		settings["hooks"] = hooks
	}
	preToolUse, ok := hooks["PreToolUse"].([]any)
	if !ok {
		preToolUse = []any{}
	}

	entry := map[string]any{
		"matcher": agentctlHookMatcher,
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": agentctlHookCommand,
			},
		},
	}
	hooks["PreToolUse"] = append(preToolUse, entry)
}

func removeAgentctlHook(settings map[string]any) {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return
	}
	preToolUse, ok := hooks["PreToolUse"].([]any)
	if !ok {
		return
	}

	filtered := make([]any, 0, len(preToolUse))
	for _, entry := range preToolUse {
		if !isAgentctlHookEntry(entry) {
			filtered = append(filtered, entry)
		}
	}
	hooks["PreToolUse"] = filtered
}

func readJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing JSON in %s: %w", path, err)
	}
	return result, nil
}

func writeJSONFile(path string, data map[string]any) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
