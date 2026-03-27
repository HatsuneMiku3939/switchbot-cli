package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	switchbotcli "github.com/hatsunemiku3939/switchbot-cli/internal/switchbot"
	"github.com/hatsunemiku3939/switchbot-cli/version"
	"gopkg.in/yaml.v3"
)

const (
	outputJSON    = "json"
	outputPretty  = "pretty"
	appName       = "switchbot-cli"
	legacyAppName = "switch-bot-cli"
)

type commandConfig struct {
	token   string
	secret  string
	output  string
	baseURL string
}

type fileConfig struct {
	Token                     string            `yaml:"SWITCH_BOT_TOKEN"`
	Secret                    string            `yaml:"SWITCH_BOT_SECRET"`
	InfraredRemoteTypeAliases map[string]string `yaml:"IR_TYPE_ALIASES"`
}

type runtimeConfig struct {
	token                     string
	secret                    string
	infraredRemoteTypeAliases map[string]string
}

func Run(args []string, stdout io.Writer, stderr io.Writer, environ []string) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "devices":
		return runDevices(args[1:], stdout, stderr, environ)
	case "status":
		return runStatus(args[1:], stdout, stderr, environ)
	case "command":
		return runCommand(args[1:], stdout, stderr, environ)
	case "version":
		_, _ = fmt.Fprintln(stdout, version.Version)
		return 0
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runDevices(args []string, stdout io.Writer, stderr io.Writer, environ []string) int {
	fs := newFlagSet("devices", stderr)
	cfg := registerCommonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	client, err := newClient(cfg, environ)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 2
	}

	result, err := client.ListDevices()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 1
	}

	if err := writeOutput(stdout, cfg.output, result); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 1
	}

	return 0
}

func runStatus(args []string, stdout io.Writer, stderr io.Writer, environ []string) int {
	fs := newFlagSet("status", stderr)
	cfg := registerCommonFlags(fs)
	deviceID := fs.String("device-id", "", "SwitchBot device ID")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *deviceID == "" {
		_, _ = fmt.Fprintln(stderr, "device-id is required")
		return 2
	}

	client, err := newClient(cfg, environ)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 2
	}

	result, err := client.GetStatus(*deviceID)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		if errors.Is(err, switchbotcli.ErrDeviceNotFound) {
			return 1
		}
		return 1
	}

	if err := writeOutput(stdout, cfg.output, result); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 1
	}

	return 0
}

func runCommand(args []string, stdout io.Writer, stderr io.Writer, environ []string) int {
	fs := newFlagSet("command", stderr)
	cfg := registerCommonFlags(fs)
	deviceID := fs.String("device-id", "", "SwitchBot device ID")
	commandParameterJSON := fs.String("command-parameter-json", "", "Command parameter JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *deviceID == "" {
		_, _ = fmt.Fprintln(stderr, "device-id is required")
		return 2
	}
	if *commandParameterJSON == "" {
		_, _ = fmt.Fprintln(stderr, "command-parameter-json is required")
		return 2
	}
	if !json.Valid([]byte(*commandParameterJSON)) {
		_, _ = fmt.Fprintln(stderr, "command-parameter-json must be valid JSON")
		return 2
	}

	client, err := newClient(cfg, environ)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 2
	}

	result, err := client.ExecuteCommand(*deviceID, *commandParameterJSON)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		if errors.Is(err, switchbotcli.ErrDeviceNotFound) {
			return 1
		}
		return 1
	}

	if err := writeOutput(stdout, cfg.output, result); err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 1
	}

	return 0
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func registerCommonFlags(fs *flag.FlagSet) *commandConfig {
	cfg := &commandConfig{}
	fs.StringVar(&cfg.token, "token", "", "SwitchBot token")
	fs.StringVar(&cfg.secret, "secret", "", "SwitchBot secret")
	fs.StringVar(&cfg.output, "output", outputJSON, "Output format: json or pretty")
	fs.StringVar(&cfg.baseURL, "base-url", "", "Override SwitchBot API base URL")
	return cfg
}

func newClient(cfg *commandConfig, environ []string) (*switchbotcli.Client, error) {
	if cfg.output != outputJSON && cfg.output != outputPretty {
		return nil, fmt.Errorf("unsupported output format: %s", cfg.output)
	}

	resolvedConfig, err := resolveRuntimeConfig(cfg.token, cfg.secret, environ)
	if err != nil {
		return nil, err
	}
	if resolvedConfig.token == "" {
		return nil, fmt.Errorf("SWITCH_BOT_TOKEN is required")
	}
	if resolvedConfig.secret == "" {
		return nil, fmt.Errorf("SWITCH_BOT_SECRET is required")
	}

	return switchbotcli.NewClient(
		resolvedConfig.token,
		resolvedConfig.secret,
		cfg.baseURL,
		resolvedConfig.infraredRemoteTypeAliases,
	), nil
}

func resolveRuntimeConfig(flagToken string, flagSecret string, environ []string) (*runtimeConfig, error) {
	envMap := parseEnvironment(environ)
	config, err := loadFileConfig(envMap)
	if err != nil {
		return nil, err
	}

	token := config.Token
	secret := config.Secret

	if envMap["SWITCH_BOT_TOKEN"] != "" {
		token = envMap["SWITCH_BOT_TOKEN"]
	}
	if envMap["SWITCH_BOT_SECRET"] != "" {
		secret = envMap["SWITCH_BOT_SECRET"]
	}

	if flagToken != "" {
		token = flagToken
	}
	if flagSecret != "" {
		secret = flagSecret
	}

	return &runtimeConfig{
		token:                     token,
		secret:                    secret,
		infraredRemoteTypeAliases: cloneStringMap(config.InfraredRemoteTypeAliases),
	}, nil
}

func parseEnvironment(environ []string) map[string]string {
	envMap := map[string]string{}
	for _, item := range environ {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		envMap[key] = value
	}

	return envMap
}

func loadFileConfig(envMap map[string]string) (*fileConfig, error) {
	configPaths, err := resolveConfigPaths(envMap)
	if err != nil {
		return nil, err
	}
	if len(configPaths) == 0 {
		return &fileConfig{}, nil
	}

	var data []byte
	for _, configPath := range configPaths {
		data, err = os.ReadFile(configPath)
		if err == nil {
			break
		}
		if os.IsNotExist(err) {
			continue
		}
		return nil, err
	}
	if err != nil {
		return &fileConfig{}, nil
	}

	config := &fileConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

func resolveConfigPaths(envMap map[string]string) ([]string, error) {
	configRoot, err := resolveConfigRoot(envMap)
	if err != nil {
		return nil, err
	}
	if configRoot == "" {
		return nil, nil
	}

	return []string{
		filepath.Join(configRoot, appName, "config.yaml"),
		filepath.Join(configRoot, legacyAppName, "config.yaml"),
	}, nil
}

func resolveConfigRoot(envMap map[string]string) (string, error) {
	configRoot := envMap["XDG_CONFIG_HOME"]
	if configRoot == "" {
		homeDir := envMap["HOME"]
		if homeDir == "" {
			homeDir, _ = os.UserHomeDir()
		}
		if homeDir == "" {
			return "", nil
		}
		configRoot = filepath.Join(homeDir, ".config")
	}

	return configRoot, nil
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}

	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}

	return cloned
}

func writeOutput(stdout io.Writer, format string, value interface{}) error {
	var (
		data []byte
		err  error
	)

	if format == outputPretty {
		data, err = json.MarshalIndent(value, "", "  ")
	} else {
		data, err = json.Marshal(value)
	}
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout, string(data))
	return err
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintf(w, "  %s devices [--token TOKEN] [--secret SECRET] [--output json|pretty] [--base-url URL]\n", appName)
	_, _ = fmt.Fprintf(w, "  %s status --device-id ID [--token TOKEN] [--secret SECRET] [--output json|pretty] [--base-url URL]\n", appName)
	_, _ = fmt.Fprintf(w, "  %s command --device-id ID --command-parameter-json JSON [--token TOKEN] [--secret SECRET] [--output json|pretty] [--base-url URL]\n", appName)
	_, _ = fmt.Fprintf(w, "  %s version\n", appName)
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintf(w, "Credentials precedence: flags > environment > ~/.config/%s/config.yaml\n", appName)
}

func Main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr, os.Environ()))
}
