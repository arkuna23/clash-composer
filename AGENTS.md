# Repository Guidelines

## Project Structure & Module Organization
- `main.go` contains the CLI entrypoint. Current commands are `merge` and `download`.
- `composer/` holds merge and download workflow logic. Keep orchestration here, not in `main.go`.
- `config/` defines Mihomo-compatible raw config types plus YAML marshal/unmarshal helpers.
- `example/` contains sample templates and proxy configs used by tests and local verification.
- `scripts/` contains helper shell scripts such as `get_rulesets.sh`.
- Tests live next to the code: `composer/merge_test.go`, `config/raw_test.go`.

## Build, Test, and Development Commands
- `make build`: builds `build/clash-composer` with a local `build/.gocache`.
- `make clean`: removes `build/` and its local build cache.
- `go test ./...`: runs all unit tests.
- `go run . merge example/merge.json`: merges example inputs and writes `merged.yaml`.
- `go run . download <subscription-url>`: downloads a subscription and writes the raw response to `stdout`.

## Coding Style & Naming Conventions
- Follow standard Go style and always run `gofmt` on edited files.
- Use tabs as produced by `gofmt`; do not hand-align with spaces.
- Package names stay lowercase (`composer`, `config`); exported names use `CamelCase`.
- Keep functions narrowly scoped: download/loading logic belongs in `composer/download.go`, merge logic in `composer/merge.go`.
- Prefer explicit logging on main workflow steps; keep low-level helpers quiet unless failures need context.

## Testing Guidelines
- Use Go’s built-in `testing` package.
- Name tests `TestXxx` and keep them close to the code they cover.
- Prefer deterministic tests with stubbed HTTP clients instead of live network calls.
- When changing examples or merge behavior, run `go test ./...` and re-check `go run . merge example/merge.json`.

## Commit & Pull Request Guidelines
- Match the existing history: short, imperative subjects such as `Refactor config download flow` or `Add logging for merge workflow`.
- Keep each commit focused on one behavior change.
- After completing code or documentation changes and passing the relevant tests, create a git commit for the completed work.
- PRs should describe the user-facing effect, list validation steps, and call out any example or CLI changes.

## Security & Configuration Tips
- Do not commit real subscription URLs, credentials, or generated personal configs.
- Treat `merged.yaml` and downloaded subscription content as local output, not source files.
