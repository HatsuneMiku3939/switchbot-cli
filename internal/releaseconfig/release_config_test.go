package releaseconfig

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

type goreleaserConfig struct {
	Builds []struct {
		ID      string   `yaml:"id"`
		Goos    []string `yaml:"goos"`
		Ldflags []string `yaml:"ldflags"`
	} `yaml:"builds"`
	Archives []struct {
		ID           string `yaml:"id"`
		NameTemplate string `yaml:"name_template"`
	} `yaml:"archives"`
	NFPMS []struct {
		ID               string   `yaml:"id"`
		PackageName      string   `yaml:"package_name"`
		IDs              []string `yaml:"ids"`
		Formats          []string `yaml:"formats"`
		FileNameTemplate string   `yaml:"file_name_template"`
	} `yaml:"nfpms"`
	HomebrewCasks []struct {
		Name string   `yaml:"name"`
		IDs  []string `yaml:"ids"`
	} `yaml:"homebrew_casks"`
}

func TestGoReleaserConfigEmbedsReleaseVersion(t *testing.T) {
	t.Parallel()

	config := loadGoReleaserConfig(t)

	for _, build := range config.Builds {
		if build.ID != "switchbot-cli" {
			continue
		}

		found := false
		for _, ldflag := range build.Ldflags {
			if ldflag == "-s -w -X github.com/hatsunemiku3939/switchbot-cli/version.Version={{ .Version }}" {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("build %q is missing the release version ldflags", build.ID)
		}

		return
	}

	t.Fatal("switchbot-cli build not found in .goreleaser.yaml")
}

func TestGoReleaserConfigProducesLinuxPackages(t *testing.T) {
	t.Parallel()

	config := loadGoReleaserConfig(t)

	for _, pkg := range config.NFPMS {
		if pkg.ID != "linux-packages" {
			continue
		}

		if pkg.PackageName != "switchbot-cli" {
			t.Fatalf("unexpected package name: %s", pkg.PackageName)
		}

		assertContainsAll(t, pkg.IDs, []string{"switchbot-cli"})
		assertContainsAll(t, pkg.Formats, []string{"deb", "rpm"})

		if pkg.FileNameTemplate != "{{ .ConventionalFileName }}" {
			t.Fatalf("unexpected file name template: %s", pkg.FileNameTemplate)
		}

		return
	}

	t.Fatal("linux-packages nfpms entry not found in .goreleaser.yaml")
}

func TestGoReleaserConfigUsesStableArchiveNameTemplate(t *testing.T) {
	t.Parallel()

	config := loadGoReleaserConfig(t)
	if len(config.Archives) == 0 {
		t.Fatal("archives section not found in .goreleaser.yaml")
	}

	const want = "{{ .ProjectName }}_{{ .Version }}_{{ title .Os }}_{{ if eq .Arch \"amd64\" }}x86_64{{ else }}{{ .Arch }}{{ end }}"
	if config.Archives[0].NameTemplate != want {
		t.Fatalf("unexpected archive name template: %s", config.Archives[0].NameTemplate)
	}

	if config.Archives[0].ID != "release-archives" {
		t.Fatalf("unexpected archive id: %s", config.Archives[0].ID)
	}
}

func TestGoReleaserConfigHomebrewCaskUsesArchiveIDs(t *testing.T) {
	t.Parallel()

	config := loadGoReleaserConfig(t)

	archiveIDs := make(map[string]struct{}, len(config.Archives))
	for _, archive := range config.Archives {
		if archive.ID == "" {
			continue
		}

		archiveIDs[archive.ID] = struct{}{}
	}

	if len(archiveIDs) == 0 {
		t.Fatal("expected at least one archive id in .goreleaser.yaml")
	}

	for _, cask := range config.HomebrewCasks {
		if cask.Name != "switchbot-cli" {
			continue
		}

		if len(cask.IDs) == 0 {
			t.Fatal("homebrew cask must reference at least one archive id")
		}

		for _, id := range cask.IDs {
			if _, ok := archiveIDs[id]; !ok {
				t.Fatalf("homebrew cask references unknown archive id %q", id)
			}
		}

		return
	}

	t.Fatal("switchbot-cli homebrew cask not found in .goreleaser.yaml")
}

func loadGoReleaserConfig(t *testing.T) goreleaserConfig {
	t.Helper()

	configPath := filepath.Join("..", "..", ".goreleaser.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	var config goreleaserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal %s: %v", configPath, err)
	}

	return config
}

func assertContainsAll(t *testing.T, got []string, want []string) {
	t.Helper()

	gotSet := make(map[string]struct{}, len(got))
	for _, value := range got {
		gotSet[value] = struct{}{}
	}

	for _, value := range want {
		if _, ok := gotSet[value]; !ok {
			t.Fatalf("expected %q in %v", value, got)
		}
	}
}
