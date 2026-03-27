# switch-bot-cli

`switch-bot-cli` is a standalone CLI for SwitchBot devices built on top of [switch-bot-api-go](https://github.com/yasu89/switch-bot-api-go).

## Commands

```bash
switch-bot-cli devices
switch-bot-cli status --device-id <device-id>
switch-bot-cli command --device-id <device-id> --command-parameter-json '{"command":"TurnOn"}'
```

## Authentication

The CLI reads credentials from the following environment variables by default.

- `SWITCH_BOT_TOKEN`
- `SWITCH_BOT_SECRET`

You can also override them per command.

If `~/.config/switch-bot-cli/config.yaml` contains the same keys, the CLI uses it as the fallback credential source. The precedence is `flags > environment variables > config.yaml`.

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
switch-bot-cli devices --token "$SWITCH_BOT_TOKEN" --secret "$SWITCH_BOT_SECRET"
```

## Output

The default output format is compact JSON.

```bash
switch-bot-cli devices --output pretty
```

`pretty` renders indented JSON for easier inspection.

The CLI omits implementation-detail fields such as the internal `Client` pointer from command output.

## Development

```bash
go test ./...
go build ./cmd/switch-bot-cli
```

Unit tests cover CLI command routing, config resolution, output formatting, and the SwitchBot client behavior against local HTTP test servers.

## Release

Pushing a tag that starts with `v` to the remote repository triggers the GitHub Actions release workflow.

```bash
git tag v0.1.0
git push origin v0.1.0
```
