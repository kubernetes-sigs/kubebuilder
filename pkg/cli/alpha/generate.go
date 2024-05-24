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
	"sigs.k8s.io/kubebuilder/v4/pkg/rescaffold"
)

// NewScaffoldCommand return a new scaffold command
func NewScaffoldCommand() *cobra.Command {
	opts := rescaffold.MigrateOptions{}
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
			if err := opts.Rescaffold(); err != nil {
				log.Fatalf("Failed to rescaffold %s", err)
			}
		},
	}
	scaffoldCmd.Flags().StringVar(&opts.InputDir, "input-dir", "",
		"path to a Kubebuilder project file if not in the current working directory")
	scaffoldCmd.Flags().StringVar(&opts.OutputDir, "output-dir", "",
		"path to output the scaffolding. defaults a directory in the current working directory")

	return scaffoldCmd
}
