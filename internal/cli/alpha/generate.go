/*
Copyright 2023 The Kubernetes Authors.
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

package alpha

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/internal/cli/alpha/internal"
)

// NewScaffoldCommand returns a new scaffold command, providing the `kubebuilder alpha generate`
// feature to re-scaffold projects and assist users with updates.
//
// IMPORTANT: This command is intended solely for Kubebuilder's use, as it is designed to work
// specifically within Kubebuilder's project configuration, key mappings, and plugin initialization.
// Its implementation includes fixed values and logic tailored to Kubebuilder’s unique setup, which may
// not apply to other projects. Consequently, importing and using this command directly in other projects
// will likely result in unexpected behavior, as external projects may have different supported plugin
// structures, configurations, and requirements.
//
// For other projects using Kubebuilder as a library, replicating similar functionality would require
// a custom implementation to ensure compatibility with the specific configurations and plugins of that project.
//
// Technically, implementing functions that allow re-scaffolding with the exact plugins and project-specific
// code of external projects is not feasible within Kubebuilder’s current design.
func NewScaffoldCommand() *cobra.Command {
	opts := internal.Generate{}

	scaffoldCmd := &cobra.Command{
		Use:   "generate",
		Short: "Re-scaffold a Kubebuilder project from its PROJECT file",
		Long: `The 'generate' command re-creates a Kubebuilder project scaffold based on the configuration 
defined in the PROJECT file, using the latest installed Kubebuilder version and plugins.

This is helpful for migrating projects to a newer Kubebuilder layout or plugin version (e.g., v3 to v4)
as update your project from any previous version to the current one.

If no output directory is provided, the current working directory will be cleaned (except .git and PROJECT).`,
		Example: `
  # **WARNING**(will delete all files to allow the re-scaffold except .git and PROJECT)
  # Re-scaffold the project in-place 
  kubebuilder alpha generate

  # Re-scaffold the project from ./test into ./my-output
  kubebuilder alpha generate --input-dir="./path/to/project" --output-dir="./my-output"
`,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return opts.Validate()
		},
		Run: func(_ *cobra.Command, _ []string) {
			if err := opts.Generate(); err != nil {
				slog.Error("failed to generate project", "error", err)
				os.Exit(1)
			}
		},
	}

	scaffoldCmd.Flags().StringVar(&opts.InputDir, "input-dir", "",
		"Path to the directory containing the PROJECT file. "+
			"Defaults to the current working directory. WARNING: delete existing files (except .git and PROJECT).")

	scaffoldCmd.Flags().StringVar(&opts.OutputDir, "output-dir", "",
		"Directory where the new project scaffold will be written. "+
			"If unset, re-scaffolding occurs in-place "+
			"and will delete existing files (except .git and PROJECT).")

	return scaffoldCmd
}
