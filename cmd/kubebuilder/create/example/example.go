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

package example

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"strings"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "example",
	Short: "Create the docs example scaffolding for an API",
	Long: `Create the docs example scaffoling for an API.

Example is written to docs/reference/examples/<lower kind>/<lower kind>.yaml
`,
	Example: `# Create a new documentation example under docs/reference/examples/mykind/mykind.yaml
kubebuilder create example --kind MyKind --version v1beta1 --group mygroup.my.domain
`,
	Run: func(cmd *cobra.Command, args []string) {
		if kind == "" {
			fmt.Printf("Must specify --kind\n")
			return
		}
		if version == "" {
			fmt.Printf("Must specify --version\n")
			return
		}
		if group == "" {
			fmt.Printf("Must specify --group\n")
			return
		}
		wd, err := os.Getwd()
		if err != nil {
			log.Fatalf("error: %v\n", err)
		}
		path := filepath.Join(wd, outputDir, "examples", strings.ToLower(kind), strings.ToLower(kind)+".yaml")
		CodeGenerator{}.Execute(path)
		fmt.Printf("Edit your controller function...\n")
		fmt.Printf("\t%s\n", path)
	},
}

var kind, version, group, outputDir string

func AddCreateExample(cmd *cobra.Command) {
	cmd.AddCommand(configCmd)
	configCmd.Flags().StringVar(&kind, "kind", "", "api Kind.")
	configCmd.Flags().StringVar(&group, "group", "", "api group.")
	configCmd.Flags().StringVar(&version, "version", "", "api version.")
	configCmd.Flags().StringVar(&outputDir, "output-dir", filepath.Join("docs", "reference"), "reference docs location")

}

// CodeGenerator generates code for Kubernetes resources and controllers
type CodeGenerator struct{}

// Execute parses packages and executes the code generators against the resource and controller packages
func (g CodeGenerator) Execute(path string) error {
	os.Remove(path)
	args := ConfigArgs{
		Group:   group,
		Version: version,
		Kind:    strings.ToLower(kind),
	}
	util.WriteIfNotFound(path, "example-config-template", exampleConfigTemplate, args)
	return nil
}

type ConfigArgs struct {
	Group, Version, Kind string
}

var exampleConfigTemplate = `note: {{ .Kind }} example
sample: |
  apiVersion: {{ .Version }}
  kind: {{ .Kind }}
  metadata:
    name: {{ lower .Kind }}
  spec:
    todo: "write me"
`
