# AGENTS.md

## Purpose

This repository implements a narrow agent control layer:

- intercept risky actions
- gate them with simple policy
- trace every decision
- replay prior sessions under different policy

## Contribution Guidance

- Keep the control plane narrow and high-signal.
- Treat the schema and trace format as product contracts.
- Prefer additive changes over broad refactors during early iterations.
- When adding a new action type, update schema validation, policy evaluation, trace queries, tests, and docs together.
- If CLI output changes, keep it script-friendly.

## Go Workflow

- Format with `gofmt`.
- Lint with `golangci-lint run`.
- Test with `go test ./...`.
- Build with `go build ./...`.

## Release Bias

- Reliability beats feature count.
- Local-first developer workflows are preferred for v1.
- Avoid introducing infra requirements unless they unlock a clear next-stage product need.

## License

This project is MIT licensed. See `LICENSE`.
