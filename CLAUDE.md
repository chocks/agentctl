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
- Follow the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md). Key rules enforced here:
  - **Use `path/filepath`** — never reimplement `filepath.Join` / `filepath.Dir` manually.
  - **Wrap errors with `%w`** — `fmt.Errorf("context: %w", err)` so callers can use `errors.Is`.
  - **Always use field names in struct literals** — positional init is fragile as fields grow.
  - **Table-driven tests** — group related cases under a single `t.Run` loop; name each subcase.
  - **No package-level mutable state** — prefer dependency injection; globals make tests order-dependent.
  - **Verify interface compliance at compile time** — `var _ http.Handler = (*apiServer)(nil)` where it matters.

## Cross-Language Contract

- Treat `api/openapi.yaml` as the source of truth for generated JS and Python clients.
- Keep OpenAPI and `pkg/schema` synchronized.
- Prefer contract-first additions over language-specific SDK drift.
- If you add or rename request fields, update the OpenAPI spec in the same change.

## Product Guardrails

- Do not expand scope into a full compliance platform in this repo without clear product pull.
- Govern only high-risk actions.
- Minimize runtime assumptions so future SDK wrappers stay low-friction.

## License

All code and docs in this repository are released under the MIT License. See `LICENSE`.
