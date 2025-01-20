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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha/internal"
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
		Short: "Re-scaffold an existing Kuberbuilder project",
		Long: `It's an experimental feature that has the purpose of re-scaffolding the whole project from the scratch 
using the current version of KubeBuilder binary available.
# make sure the PROJECT file is in the 'input-dir' argument, the default is the current directory.
$ kubebuilder alpha generate --input-dir="./test" --output-dir="./my-output"
Then we will re-scaffold the project by Kubebuilder in the directory specified by 'output-dir'.
		`,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return opts.Validate()
		},
		Run: func(_ *cobra.Command, _ []string) {
			if err := opts.Generate(); err != nil {
				log.Fatalf("Failed to command %s", err)
			}
		},
	}
	scaffoldCmd.Flags().StringVar(&opts.InputDir, "input-dir", "",
		"Specifies the full path to a Kubebuilder project file. If not provided, "+
			"the current working directory is used.")
	scaffoldCmd.Flags().StringVar(&opts.OutputDir, "output-dir", "",
		"Specifies the full path where the scaffolded files will be output. "+
			"Defaults to a directory within the current working directory.")

	return scaffoldCmd
}
