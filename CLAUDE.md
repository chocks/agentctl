# CLAUDE.md

## Project

`agentctl` is a small Go codebase for gating and tracing high-risk agent actions. Keep the product focused on the three v1 primitives: `gate`, `trace`, and `replay`.

## Engineering Rules

- Prefer small, composable packages under `pkg/`.
- Keep CLI behavior in `cmd/agentctl`; avoid leaking flag parsing into lower layers.
- Preserve the action schema in `pkg/schema` carefully. Changes there affect policy, tracing, replay, and future SDKs.
- Favor standard library implementations unless a dependency clearly pays for itself.
- Record traces reliably. Do not optimize trace writes in a way that risks dropping decisions.
- Keep policy evaluation deterministic and side-effect free.

## Go Practices

- Run `gofmt -w` on changed Go files.
- Keep functions straightforward; avoid clever abstractions in v1.
- Add tests for policy and trace edge cases when behavior changes.
- Use `golangci-lint` defaults plus the repo config before merging.

## Product Guardrails

- Do not expand scope into a full compliance platform in this repo without clear product pull.
- Govern only high-risk actions.
- Minimize runtime assumptions so future SDK wrappers stay low-friction.

## License

All code and docs in this repository are released under the MIT License. See `LICENSE`.
