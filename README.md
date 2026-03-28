# switchbot-cli

`switchbot-cli` is a standalone CLI for SwitchBot devices built on top of [switch-bot-api-go](https://github.com/yasu89/switch-bot-api-go).

## Commands

```bash
switchbot-cli devices
switchbot-cli status --device-id <device-id>
switchbot-cli command --device-id <device-id> --command-parameter-json '{"command":"TurnOn"}'
```

## Authentication

The CLI reads credentials from the following environment variables by default.

- `SWITCH_BOT_TOKEN`
- `SWITCH_BOT_SECRET`

You can also override them per command.

If `~/.config/switchbot-cli/config.yaml` contains the same keys, the CLI uses it as the fallback credential source. The precedence is `flags > environment variables > config.yaml`.

For a smoother rename, the CLI also reads the legacy path `~/.config/switch-bot-cli/config.yaml` when the new path does not exist.

```yaml
SWITCH_BOT_TOKEN: your-token
SWITCH_BOT_SECRET: your-secret
IR_TYPE_ALIASES:
  DIY Light: Light
```

`IR_TYPE_ALIASES` lets you map custom infrared remote types to a known SwitchBot infrared device kind. This is useful when SwitchBot returns a custom `remoteType` value but the CLI should still expose the command schema and command handling for a supported kind such as `Light` or `TV`.

If the key is omitted, the CLI does not apply any infrared remote type aliases.

Supported alias target values are:

- `Air Conditioner`
- `TV`
- `Light`
- `Streamer`
- `Set Top Box`
- `DVD Player`
- `Fan`
- `Projector`
- `Camera`
- `Air Purifier`
- `Speaker`
- `Water Heater`
- `Robot Vacuum Cleaner`
- `Others`

These values are case-sensitive and must match exactly.

```bash
switchbot-cli devices --token "$SWITCH_BOT_TOKEN" --secret "$SWITCH_BOT_SECRET"
```

## Output

The default output format is compact JSON.

```bash
switchbot-cli devices --output pretty
```

`pretty` renders indented JSON for easier inspection.

The CLI omits implementation-detail fields such as the internal `Client` pointer from command output.

## Development

```bash
go test ./...
go build ./cmd/switchbot-cli
```

Unit tests cover CLI command routing, config resolution, output formatting, and the SwitchBot client behavior against local HTTP test servers.

GitHub Actions runs `go test ./...` and `go vet ./...` on pushes to `master` and on pull requests.

## Homebrew

Install the macOS cask from the custom tap.

```bash
brew tap hatsunemiku3939/tap
brew install --cask switchbot-cli
```

You can also install it without a separate tap step.

```bash
brew install --cask hatsunemiku3939/tap/switchbot-cli
```

The cask installs an unsigned binary. If macOS blocks execution, inspect the binary first and then remove the quarantine attribute manually.

```bash
xattr -dr com.apple.quarantine "$(brew --prefix)/Caskroom/switchbot-cli/<version>/switchbot-cli"
```

## Release

Pushing a tag that starts with `v` to the remote repository triggers the GitHub Actions release workflow.

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow also updates the Homebrew cask in the `HatsuneMiku3939/homebrew-tap` repository. Set the `HOMEBREW_TAP_GITHUB_TOKEN` GitHub Actions secret to a personal access token with write access to that repository before publishing a release.
