/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2alpha

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var _ plugin.DeleteSubcommand = &deleteSubcommand{}

type deleteSubcommand struct {
	config config.Config
}

func (p *deleteSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = "Remove Helm chart scaffolding added by this plugin"
	subcmdMeta.Examples = fmt.Sprintf(`  # Remove Helm chart from the project
  %[1]s delete --plugins=%[2]s
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *deleteSubcommand) BindFlags(_ *pflag.FlagSet) {}

func (p *deleteSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *deleteSubcommand) Scaffold(fs machinery.Filesystem) error {
	return deleteHelmChart(p.config, fs)
}

// deleteHelmChart removes the chart directory and CI workflow added by this plugin,
// then clears the plugin's entry in the PROJECT file.
func deleteHelmChart(cfg config.Config, fs machinery.Filesystem) error {
	// Resolve the output directory from saved plugin config.
	stored := pluginConfig{}
	key := plugin.GetPluginKeyForConfig(cfg.GetPluginChain(), Plugin{})
	if err := cfg.DecodePluginConfig(key, &stored); err != nil {
		if errors.As(err, &config.PluginKeyNotFoundError{}) {
			canonicalKey := plugin.KeyFor(Plugin{})
			if key != canonicalKey {
				_ = cfg.DecodePluginConfig(canonicalKey, &stored)
			}
		}
	}

	outputDir := stored.OutputDir
	if outputDir == "" {
		outputDir = DefaultOutputDir
	}

	deleted, warned := 0, 0

	chartDir := filepath.Join(outputDir, "chart")
	if exists, _ := afero.DirExists(fs.FS, chartDir); exists {
		if err := fs.FS.RemoveAll(chartDir); err != nil {
			slog.Warn("Failed to delete Helm chart directory", "path", chartDir, "error", err)
			warned++
		} else {
			deleted++
		}
	} else {
		slog.Warn("Helm chart directory not found", "path", chartDir)
		warned++
	}

	testChart := filepath.Join(".github", "workflows", "test-chart.yml")
	if exists, _ := afero.Exists(fs.FS, testChart); exists {
		if err := fs.FS.Remove(testChart); err != nil {
			slog.Warn("Failed to delete test-chart workflow", "path", testChart, "error", err)
			warned++
		} else {
			deleted++
		}
	} else {
		slog.Warn("Test chart workflow not found", "path", testChart)
		warned++
	}

	if err := removeMakefileHelmSection(); err != nil {
		slog.Warn("Failed to remove Helm targets from Makefile", "error", err)
		warned++
	} else {
		deleted++
	}

	if err := plugin.RemovePluginConfig(cfg, Plugin{}); err != nil {
		slog.Warn("Failed to remove plugin config from PROJECT file", "error", err)
		warned++
	}

	fmt.Printf("Helm plugin deletion: %d item(s) deleted", deleted)
	if warned > 0 {
		fmt.Printf(", %d warning(s) — check logs for details", warned)
	}
	fmt.Println()

	return nil
}

// removeMakefileHelmSection removes the ##@ Helm Deployment block from the Makefile.
func removeMakefileHelmSection() error {
	const makefilePath = "Makefile"
	const helmSectionHeader = "\n##@ Helm Deployment"

	content, err := os.ReadFile(makefilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read Makefile: %w", err)
	}

	text := string(content)
	start := strings.Index(text, helmSectionHeader)
	if start == -1 {
		return nil
	}

	// Find the next ##@ section after the Helm one, or trim to EOF.
	rest := text[start+1:]
	nextSection := strings.Index(rest, "\n##@")
	var result string
	if nextSection == -1 {
		result = text[:start]
	} else {
		result = text[:start] + rest[nextSection:]
	}

	if err := os.WriteFile(makefilePath, []byte(result), 0o644); err != nil {
		return fmt.Errorf("write Makefile: %w", err)
	}
	return nil
}
