# AGENTS.md

## Architecture Notes

- This repository is a small Go CLI for SwitchBot devices.
- Entry point: `cmd/switchbot-cli/main.go`. It forwards `os.Args`, stdio, and environment variables to `cli.Run` and exits with its return code.
- CLI layer: `internal/cli/cli.go`.
- `internal/cli` owns top-level command routing.
- `internal/cli` owns flag parsing and argument validation.
- `internal/cli` owns config file and environment resolution.
- `internal/cli` owns output formatting and exit code behavior.
- SwitchBot adapter layer: `internal/switchbot/client.go`.
- `internal/switchbot` wraps `github.com/yasu89/switch-bot-api-go`.
- `internal/switchbot` owns device listing, status lookup, command execution, infrared alias normalization, and CLI-facing output enrichment.
- Version source: `version/version.go`. Release builds overwrite `version.Version` via GoReleaser `ldflags`.
- Release config tests live in `internal/releaseconfig/release_config_test.go` and protect `.goreleaser.yaml` expectations.

## Development Environment

- Required Go version: `1.24.2` (see `go.mod`).
- Main dependency: `github.com/yasu89/switch-bot-api-go`.
- Test dependency: `github.com/stretchr/testify`.
- Config parsing dependency: `gopkg.in/yaml.v3`.
- Common command: `make build`.
- Common command: `make test`.
- Common command: `make lint`.
- Common command: `make release-check`.
- Common command: `make release-snapshot`.
- Direct build command: `go build ./cmd/switchbot-cli`.
- Direct test command: `go test ./...`.
- Direct lint command: `go vet ./...`.
- Credential lookup order is flags, then environment variables, then config file.
- Supported config file path: `~/.config/switchbot-cli/config.yaml`.
- Supported credential key: `SWITCH_BOT_TOKEN`.
- Supported credential key: `SWITCH_BOT_SECRET`.
- Optional config key: `IR_TYPE_ALIASES`.

## Testing Strategy

- Run `go test ./...` after every code change.
- Run `go vet ./...` before committing changes. CI runs both.
- Most behavior is covered by package tests using `httptest` servers instead of live SwitchBot calls.
- Prefer table-driven tests when adding new validation or branching behavior. Existing tests already follow this style in several places.
- When changing CLI behavior, update `internal/cli/cli_test.go`.
- When changing API adapter behavior, update `internal/switchbot/client_test.go`.
- When changing release packaging or version injection, update `internal/releaseconfig/release_config_test.go`.
- There are no dedicated end-to-end tests in the repository today.

## Code Style & Conventions

- Keep documentation and code comments in English.
- Keep the CLI contract stable unless the task explicitly changes it.
- Supported commands are `devices`, `status`, `command`, `version`, and `help`.
- Supported output formats are `json` and `pretty`.
- Preserve exit code `0` for success.
- Preserve exit code `1` for runtime or API failures.
- Preserve exit code `2` for usage, validation, or configuration errors.
- Preserve credential precedence: flags override environment variables, which override config file values.
- Output is JSON only. The default is compact JSON; `pretty` uses indented JSON.
- Do not expose implementation-detail fields in CLI output. `Client` is intentionally removed from serialized device output.
- Infrared remote aliases are case-sensitive and map custom `remoteType` values to supported SwitchBot infrared device kinds.
- Keep package boundaries simple.
- `internal/cli` should not absorb SwitchBot API details.
- `internal/switchbot` should not own CLI parsing or stdout/stderr formatting.

## Agent Guardrails

- Do not bypass or silently change the config resolution rules in `internal/cli`.
- Do not change JSON field shape casually. Downstream scripts may rely on current output.
- If you touch infrared remote handling, verify both normalization and command-schema exposure.
- If you change commands, flags, config keys, install instructions, or release behavior, update `README.md`.
- If you edit `.goreleaser.yaml`, run the release config tests and keep packaging expectations aligned.
- Release automation depends on `.github/workflows/release.yml`.
- Release automation depends on `.goreleaser.yaml`.
- Release automation depends on `.github/release.yml`.
- The repository includes an MIT `LICENSE` file.
- Keep `LICENSE*` included in release archives unless the release policy changes.
- Use `--base-url` for tests and local API stubs instead of hardcoding endpoint changes into production defaults.
- Prefer extending existing tests over adding one-off manual verification steps.
- > TODO: No repository-specific branching, PR, or review workflow is documented inside this repository.
