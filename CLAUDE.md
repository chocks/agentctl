# CLAUDE.md

## Project

`agentctl` is a small Go codebase for gating and tracing high-risk agent actions. Keep the product focused on the v1 control plane: attach/detach, gate, trace, replay, approvals, and the terminal UI.

## Engineering Rules

- Prefer small, composable packages under `pkg/`.
- Keep CLI behavior in `cmd/agentctl`; avoid leaking flag parsing into lower layers.
- Preserve the action schema in `pkg/schema` carefully. Changes there affect policy, tracing, replay, and the hook/MCP adapters.
- Favor standard library implementations unless a dependency clearly pays for itself.
- Record traces reliably. Do not optimize trace writes in a way that risks dropping decisions.
- Keep policy evaluation deterministic and side-effect free.
- Keep the user-facing configuration model simple: one global policy at `~/.agentctl/policy.yaml`.

## Go Practices

- Run `gofmt -w` on changed Go files.
- Keep functions straightforward; avoid clever abstractions in v1.
- Add tests for policy and trace edge cases when behavior changes.
- Use `golangci-lint` defaults plus the repo config before merging.
- Follow the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md). Key rules enforced here:
  - **Use `path/filepath`** — never reimplement `filepath.Join` / `filepath.Dir` manually.
  - **Wrap errors with `%w`** — `fmt.Errorf("context: %w", err)` so callers can use `errors.Is`.
  - **Always use field names in struct literals** — positional init is fragile as fields grow.
  - **Table-driven tests** — group related cases under a single `t.Run` loop; name each subcase.
  - **No package-level mutable state** — prefer dependency injection; globals make tests order-dependent.
  - **Verify interface compliance at compile time** where it materially protects a boundary.

## Product Guardrails

- Do not expand scope into a full compliance platform in this repo without clear product pull.
- Govern only high-risk actions.
- Prefer local-first workflows over background daemons or extra infrastructure.

## License

All code and docs in this repository are released under the MIT License. See `LICENSE`.
