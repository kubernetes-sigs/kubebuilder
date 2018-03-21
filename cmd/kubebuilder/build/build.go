/*
Copyright 2017 The Kubernetes Authors.

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

package build

import (
	"github.com/spf13/cobra"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/docs"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/generate"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Command group for building source into artifacts.",
	Long:  `Command group for building source into artifacts.`,
	Example: `# Generate code and build the apiserver and controller-manager binaries into bin/
kubebuilder build docs

# Rebuild generated code
kubebuilder build generated
`,
	Run: RunBuild,
	Deprecated: "`build generated` and `build docs` have been moved to `generate` and `docs`",
}

func AddBuild(cmd *cobra.Command) {
	cmd.AddCommand(buildCmd)

	buildCmd.AddCommand(docs.GetDocs())
	buildCmd.AddCommand(generate.GetGenerate())
}

func RunBuild(cmd *cobra.Command, args []string) {
	cmd.Help()
}
