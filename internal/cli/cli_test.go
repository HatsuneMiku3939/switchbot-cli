package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hatsunemiku3939/switchbot-cli/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunTopLevelCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		wantExitCode   int
		wantStdoutPart string
		wantStderrPart string
	}{
		{
			name:           "prints usage when no command is given",
			args:           nil,
			wantExitCode:   2,
			wantStderrPart: "Usage:",
		},
		{
			name:           "prints help",
			args:           []string{"help"},
			wantExitCode:   0,
			wantStdoutPart: "Usage:",
		},
		{
			name:           "prints version",
			args:           []string{"version"},
			wantExitCode:   0,
			wantStdoutPart: version.Version,
		},
		{
			name:           "rejects unknown command",
			args:           []string{"wat"},
			wantExitCode:   2,
			wantStderrPart: "unknown command: wat",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := Run(tt.args, &stdout, &stderr, nil)

			assert.Equal(t, tt.wantExitCode, exitCode)
			if tt.wantStdoutPart != "" {
				assert.Contains(t, stdout.String(), tt.wantStdoutPart)
			}
			if tt.wantStderrPart != "" {
				assert.Contains(t, stderr.String(), tt.wantStderrPart)
			}
		})
	}
}

func TestRunDevicesUsesConfigCredentialsAndAliases(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/devices", r.URL.Path)
		assert.Equal(t, "config-token", r.Header.Get("Authorization"))

		writeJSONResponse(t, w, map[string]any{
			"statusCode": 100,
			"body": map[string]any{
				"deviceList": []any{},
				"infraredRemoteList": []any{
					map[string]any{
						"deviceId":    "ir-light-1",
						"deviceName":  "Living Room Light",
						"remoteType":  "DIY Light",
						"hubDeviceId": "hub-1",
					},
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	configRoot := t.TempDir()
	writeTestConfig(t, configRoot, `
SWITCH_BOT_TOKEN: config-token
SWITCH_BOT_SECRET: config-secret
IR_TYPE_ALIASES:
  DIY Light: Light
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(
		[]string{"devices", "--base-url", server.URL},
		&stdout,
		&stderr,
		[]string{"XDG_CONFIG_HOME=" + configRoot},
	)

	require.Equal(t, 0, exitCode, stderr.String())
	assert.Empty(t, stderr.String())

	var output map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))

	infraredList := output["infraredRemoteList"].([]any)
	require.Len(t, infraredList, 1)

	device := infraredList[0].(map[string]any)
	assert.Equal(t, "DIY Light", device["remoteType"])
	assert.Contains(t, device, "commandParameterJSONSchema")
	assert.NotContains(t, device, "Client")
}

func TestRunStatusFetchesDeviceStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/devices":
			writeJSONResponse(t, w, map[string]any{
				"statusCode": 100,
				"body": map[string]any{
					"deviceList": []any{
						map[string]any{
							"deviceId":           "bot-1",
							"deviceType":         "Bot",
							"hubDeviceId":        "hub-1",
							"deviceName":         "Desk Switch",
							"enableCloudService": true,
						},
					},
					"infraredRemoteList": []any{},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/devices/bot-1/status":
			writeJSONResponse(t, w, map[string]any{
				"statusCode": 100,
				"body": map[string]any{
					"deviceId":    "bot-1",
					"deviceType":  "Bot",
					"hubDeviceId": "hub-1",
					"power":       "on",
					"battery":     100,
					"version":     "V1.0",
					"deviceMode":  "pressMode",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(
		[]string{
			"status",
			"--device-id", "bot-1",
			"--token", "flag-token",
			"--secret", "flag-secret",
			"--base-url", server.URL,
		},
		&stdout,
		&stderr,
		isolatedEnviron(t),
	)

	require.Equal(t, 0, exitCode, stderr.String())

	var output map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	assert.Equal(t, "on", output["power"])
	assert.Equal(t, "pressMode", output["deviceMode"])
}

func TestRunCommandUsesInfraredAliases(t *testing.T) {
	t.Parallel()

	var commandRequest map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/devices":
			writeJSONResponse(t, w, map[string]any{
				"statusCode": 100,
				"body": map[string]any{
					"deviceList": []any{},
					"infraredRemoteList": []any{
						map[string]any{
							"deviceId":    "ir-light-1",
							"deviceName":  "Living Room Light",
							"remoteType":  "DIY Light",
							"hubDeviceId": "hub-1",
						},
					},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/devices/ir-light-1/commands":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&commandRequest))
			writeJSONResponse(t, w, map[string]any{
				"statusCode": 100,
				"message":    "success",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	configRoot := t.TempDir()
	writeTestConfig(t, configRoot, `
SWITCH_BOT_TOKEN: config-token
SWITCH_BOT_SECRET: config-secret
IR_TYPE_ALIASES:
  DIY Light: Light
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(
		[]string{
			"command",
			"--device-id", "ir-light-1",
			"--command-parameter-json", `{"command":"TurnOn"}`,
			"--base-url", server.URL,
		},
		&stdout,
		&stderr,
		[]string{"XDG_CONFIG_HOME=" + configRoot},
	)

	require.Equal(t, 0, exitCode, stderr.String())
	assert.Equal(t, map[string]any{
		"command":     "turnOn",
		"commandType": "command",
		"parameter":   "default",
	}, commandRequest)

	var output map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	assert.Equal(t, float64(100), output["statusCode"])
}

func TestRunValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantExitCode int
		wantStderr   string
	}{
		{
			name:         "devices requires token",
			args:         []string{"devices"},
			wantExitCode: 2,
			wantStderr:   "SWITCH_BOT_TOKEN is required",
		},
		{
			name:         "status requires device id",
			args:         []string{"status"},
			wantExitCode: 2,
			wantStderr:   "device-id is required",
		},
		{
			name:         "command requires device id",
			args:         []string{"command"},
			wantExitCode: 2,
			wantStderr:   "device-id is required",
		},
		{
			name: "command requires parameter json",
			args: []string{
				"command",
				"--device-id", "device-1",
			},
			wantExitCode: 2,
			wantStderr:   "command-parameter-json is required",
		},
		{
			name: "command validates json",
			args: []string{
				"command",
				"--device-id", "device-1",
				"--command-parameter-json", "{",
			},
			wantExitCode: 2,
			wantStderr:   "command-parameter-json must be valid JSON",
		},
		{
			name: "devices validates output format",
			args: []string{
				"devices",
				"--output", "yaml",
				"--token", "token",
				"--secret", "secret",
			},
			wantExitCode: 2,
			wantStderr:   "unsupported output format: yaml",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := Run(tt.args, &stdout, &stderr, isolatedEnviron(t))

			assert.Equal(t, tt.wantExitCode, exitCode)
			assert.Contains(t, stderr.String(), tt.wantStderr)
			assert.Empty(t, stdout.String())
		})
	}
}

func TestResolveRuntimeConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		configYAML  string
		environ     []string
		flagToken   string
		flagSecret  string
		wantToken   string
		wantSecret  string
		wantAliases map[string]string
	}{
		{
			name: "uses config values when no overrides are set",
			configYAML: `
SWITCH_BOT_TOKEN: config-token
SWITCH_BOT_SECRET: config-secret
IR_TYPE_ALIASES:
  DIY Light: Light
  Custom TV: TV
`,
			wantToken:  "config-token",
			wantSecret: "config-secret",
			wantAliases: map[string]string{
				"DIY Light": "Light",
				"Custom TV": "TV",
			},
		},
		{
			name: "environment overrides credentials but keeps aliases",
			configYAML: `
SWITCH_BOT_TOKEN: config-token
SWITCH_BOT_SECRET: config-secret
IR_TYPE_ALIASES:
  DIY Light: Light
`,
			environ: []string{
				"SWITCH_BOT_TOKEN=env-token",
				"SWITCH_BOT_SECRET=env-secret",
			},
			wantToken:  "env-token",
			wantSecret: "env-secret",
			wantAliases: map[string]string{
				"DIY Light": "Light",
			},
		},
		{
			name: "flags override environment credentials",
			configYAML: `
SWITCH_BOT_TOKEN: config-token
SWITCH_BOT_SECRET: config-secret
IR_TYPE_ALIASES: {}
`,
			environ: []string{
				"SWITCH_BOT_TOKEN=env-token",
				"SWITCH_BOT_SECRET=env-secret",
			},
			flagToken:   "flag-token",
			flagSecret:  "flag-secret",
			wantToken:   "flag-token",
			wantSecret:  "flag-secret",
			wantAliases: map[string]string{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configRoot := t.TempDir()
			writeTestConfig(t, configRoot, tt.configYAML)

			environ := append([]string{"XDG_CONFIG_HOME=" + configRoot}, tt.environ...)

			config, err := resolveRuntimeConfig(tt.flagToken, tt.flagSecret, environ)
			require.NoError(t, err)

			assert.Equal(t, tt.wantToken, config.token)
			assert.Equal(t, tt.wantSecret, config.secret)
			assert.Equal(t, tt.wantAliases, config.infraredRemoteTypeAliases)
		})
	}
}

func TestParseEnvironment(t *testing.T) {
	t.Parallel()

	envMap := parseEnvironment([]string{
		"SWITCH_BOT_TOKEN=token=value",
		"SWITCH_BOT_SECRET=secret",
		"INVALID",
	})

	assert.Equal(t, map[string]string{
		"SWITCH_BOT_TOKEN":  "token=value",
		"SWITCH_BOT_SECRET": "secret",
	}, envMap)
}

func TestLoadFileConfig(t *testing.T) {
	t.Parallel()

	t.Run("returns empty config when missing", func(t *testing.T) {
		t.Parallel()

		config, err := loadFileConfig(map[string]string{
			"XDG_CONFIG_HOME": t.TempDir(),
		})
		require.NoError(t, err)
		assert.Equal(t, &fileConfig{}, config)
	})

	t.Run("returns parse errors", func(t *testing.T) {
		t.Parallel()

		configRoot := t.TempDir()
		writeTestConfig(t, configRoot, "IR_TYPE_ALIASES: [")

		_, err := loadFileConfig(map[string]string{
			"XDG_CONFIG_HOME": configRoot,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})

	t.Run("falls back to legacy config path", func(t *testing.T) {
		t.Parallel()

		configRoot := t.TempDir()
		legacyConfigPath := filepath.Join(configRoot, legacyAppName, "config.yaml")
		writeTestFile(t, legacyConfigPath, `
SWITCH_BOT_TOKEN: legacy-token
SWITCH_BOT_SECRET: legacy-secret
`)

		config, err := loadFileConfig(map[string]string{
			"XDG_CONFIG_HOME": configRoot,
		})
		require.NoError(t, err)
		assert.Equal(t, "legacy-token", config.Token)
		assert.Equal(t, "legacy-secret", config.Secret)
	})
}

func TestResolveConfigPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		envMap map[string]string
		want   []string
	}{
		{
			name: "uses xdg config home when available",
			envMap: map[string]string{
				"XDG_CONFIG_HOME": "/tmp/xdg",
			},
			want: []string{
				"/tmp/xdg/switchbot-cli/config.yaml",
				"/tmp/xdg/switch-bot-cli/config.yaml",
			},
		},
		{
			name: "falls back to home config directory",
			envMap: map[string]string{
				"HOME": "/tmp/home",
			},
			want: []string{
				"/tmp/home/.config/switchbot-cli/config.yaml",
				"/tmp/home/.config/switch-bot-cli/config.yaml",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveConfigPaths(tt.envMap)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWriteOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		format     string
		wantOutput string
	}{
		{
			name:       "writes compact json",
			format:     outputJSON,
			wantOutput: "{\"hello\":\"world\"}\n",
		},
		{
			name:       "writes pretty json",
			format:     outputPretty,
			wantOutput: "{\n  \"hello\": \"world\"\n}\n",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer

			err := writeOutput(&stdout, tt.format, map[string]string{"hello": "world"})
			require.NoError(t, err)

			assert.Equal(t, tt.wantOutput, stdout.String())
		})
	}
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, payload map[string]any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}

func writeTestConfig(t *testing.T, configRoot string, content string) {
	t.Helper()

	configPath := filepath.Join(configRoot, appName, "config.yaml")
	writeTestFile(t, configPath, content)
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err)

	err = os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
}

func isolatedEnviron(t *testing.T) []string {
	t.Helper()

	configRoot := t.TempDir()
	homeRoot := t.TempDir()

	return []string{
		"XDG_CONFIG_HOME=" + configRoot,
		"HOME=" + homeRoot,
	}
}
