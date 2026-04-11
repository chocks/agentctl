# Changelog

All notable changes to agentctl are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Add new work under `## [Unreleased]`. On release, rename the section to the new
version and date, and the release workflow will lift it into the GitHub Release
notes automatically.

## [Unreleased]

## [0.0.2] - 2026-04-11

### Changed

- Release notes for each tagged release are now sourced from the matching
  `[<version>]` section in `CHANGELOG.md` and injected into the GitHub Release
  header.
- The auto-generated commit changelog now groups commits into **Features**,
  **Bug fixes**, and **Other** sections.

## [0.0.1] - 2026-04-11

First tagged release of agentctl — a small Go toolkit for gating, tracing, and
replaying high-risk agent actions.

### Added

- **`agentctl gate`** — deterministic policy evaluation against the action
  schema in `pkg/schema`, with YAML policies (see `agentctl.policy.yaml`).
- **`agentctl trace`** — reliable append-only trace storage for every gated
  decision, kept in the agentctl home directory.
- **`agentctl replay`** — inspect and replay recorded decisions for incident
  review and policy iteration.
- **Local HTTP server and approval workflow** — `agentctl serve` exposes a
  local API for gate/trace/replay plus an approval endpoint for escalations.
- **MCP server** (`agentctl mcp`) — expose agentctl to MCP clients.
- **Claude Code hook adapter** — gate tool calls from Claude Code via hooks.
- **VS Code extension** — approvals panel and trace viewer.
- **OpenAPI contract** (`api/openapi.yaml`) and generated JS / Python SDKs
  under `sdk/` — the OpenAPI spec is the source of truth for cross-language
  clients.
- **Release pipeline** — cross-platform binaries for linux, darwin, windows
  across amd64 and arm64, built and published via goreleaser.

[Unreleased]: https://github.com/chocks/agentctl/compare/v0.0.2...HEAD
[0.0.2]: https://github.com/chocks/agentctl/releases/tag/v0.0.2
[0.0.1]: https://github.com/chocks/agentctl/releases/tag/v0.0.1
