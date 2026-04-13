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
- Keep the configuration model global and simple: `~/.agentctl/policy.yaml`, `traces.jsonl`, and `approvals.jsonl`.

## Go Workflow

- Format with `gofmt`.
- Lint with `golangci-lint run`.
- Test with `go test ./...`.
- Build with `go build ./...`.

## Style Guide

Follow the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md). Non-negotiable rules for this codebase:

- **`path/filepath` for all path ops** — `filepath.Join`, `filepath.Dir`; never roll your own.
- **`%w` in `fmt.Errorf`** — enables `errors.Is` / `errors.As` for callers.
- **Named fields in struct literals** — `Foo{Bar: x}`, not `Foo{x}`.
- **Table-driven tests** — one test function per behaviour area, subtests via `t.Run`.
- **No global mutable state** — pass dependencies explicitly; `policy.Engine` and `trace.Store` are already correct examples.
- **Policy tests cover every rule branch** — `pkg/policy` is the security enforcement path; a missing test for a rule branch is a gap, not just a style issue.

## Release Bias

- Reliability beats feature count.
- Local-first developer workflows are preferred for v1.
- Avoid introducing infra requirements unless they unlock a clear next-stage product need.

## License

This project is MIT licensed. See `LICENSE`.
