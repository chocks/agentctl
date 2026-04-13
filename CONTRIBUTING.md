# Contributing to agentctl

## Prerequisites

- Go 1.23+
- `golangci-lint` ([install](https://golangci-lint.run/welcome/install/))

## Setup

```bash
git clone https://github.com/chocks/agentctl.git
cd agentctl
make build
make test
```

## Development Workflow

1. Create a branch from `main`.
2. Make your changes.
3. Run the checks:

```bash
make fmt
make lint
make test
```

4. Commit with a clear message. We use [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` new functionality
   - `fix:` bug fix
   - `docs:` documentation only
   - `build:` build/CI changes
   - `refactor:` no behavior change
   - `test:` test-only changes

5. Open a PR against `main`.

## What Belongs Here

agentctl governs five high-risk actions: `install_package`, `run_code`, `access_secret`, `write_file`, `call_external_api`. Contributions should stay within this scope.

Good contributions:
- Bug fixes in policy evaluation or trace recording
- New policy rules for the existing five actions
- Improvements to the CLI, TUI, hook/MCP adapters, or replay workflow
- Test coverage for edge cases in `pkg/policy` and `pkg/trace`

Probably out of scope:
- New action types (discuss in an issue first)
- External infrastructure dependencies (databases, message queues)
- Full compliance platform features

When in doubt, open an issue before writing code.

## Code Guidelines

- **Style**: follow the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md).
- **Formatting**: `gofmt` is non-negotiable. `make fmt` handles it.
- **Tests**: table-driven tests with `t.Run` subtests. Policy tests must cover every rule branch — this is a security path, not just a style preference.
- **Errors**: wrap with `fmt.Errorf("context: %w", err)` so callers can use `errors.Is`.
- **Struct literals**: always use named fields.
- **Paths**: use `path/filepath`, never manual string concatenation.
- **Dependencies**: prefer the standard library. A new dependency needs to clearly pay for itself.

## Schema Changes

`pkg/schema` is a sensitive file. If you change it:

1. Update policy evaluation, trace queries, and tests in the same PR.
2. Update hook and MCP request mapping if the request shape changed.
3. Update docs if the user-facing contract changed.

## Traces Are Sacred

Trace recording must be reliable. Do not optimize trace writes in a way that risks dropping decisions. If your change touches `pkg/trace`, test the failure paths.

## PR Checklist

- [ ] `make fmt` produces no diff
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] Schema changes are reflected in policy evaluation, trace queries, and tests
- [ ] Docs are updated when user-facing behavior changes
- [ ] CLI output changes remain script-friendly (parseable, no gratuitous formatting)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
