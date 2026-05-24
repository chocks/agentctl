// Package summary renders one-line, human-readable descriptions of the
// concrete operation behind an action request — the shell command for
// run_code, the target path for write_file, and so on. The output is shared
// by the CLI trace list and the terminal UI so both describe an action the
// same way.
package summary

import (
	"encoding/json"

	"github.com/chocks/agentctl/pkg/schema"
)

// Request renders the concrete operation behind an action request. It is
// display-only and must never fail: on any unmarshal error it falls back to an
// empty string so the caller simply renders blank.
//
// The result is always plain text — never lipgloss/ANSI-styled — since styled
// content corrupts fixed-width table cells (truncation counts escape bytes as
// visible width).
func Request(req schema.ActionRequest) string {
	switch req.Action {
	case schema.ActionRunCode:
		var p schema.RunCodeParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return ""
		}
		return formatRunCode(p)

	case schema.ActionWriteFile:
		var p schema.WriteFileParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return ""
		}
		return formatWriteFile(p)

	case schema.ActionAccessSecret:
		var p schema.AccessSecretParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return ""
		}
		return formatAccessSecret(p)

	case schema.ActionCallExternalAPI:
		var p schema.CallExternalAPIParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return ""
		}
		return formatCallExternalAPI(p)

	case schema.ActionInstallPackage:
		var p schema.InstallPackageParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return ""
		}
		return formatInstallPackage(p)

	default:
		return ""
	}
}

func formatRunCode(p schema.RunCodeParams) string {
	if p.Language == "" {
		return p.Command
	}
	return p.Language + ": " + p.Command
}

func formatWriteFile(p schema.WriteFileParams) string {
	if p.Operation == "" {
		return p.Path
	}
	return p.Operation + " " + p.Path
}

func formatAccessSecret(p schema.AccessSecretParams) string {
	if p.Scope == "" {
		return p.Name
	}
	return p.Name + " (" + p.Scope + ")"
}

func formatCallExternalAPI(p schema.CallExternalAPIParams) string {
	if p.Method == "" {
		return p.URL
	}
	return p.Method + " " + p.URL
}

func formatInstallPackage(p schema.InstallPackageParams) string {
	s := p.Manager + " " + p.Package
	if p.Version != "" {
		s += "@" + p.Version
	}
	return s
}
