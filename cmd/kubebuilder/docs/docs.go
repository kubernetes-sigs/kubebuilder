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

package docs

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate API reference docs.",
	Long: `Generate API reference docs.

Documentation will be written to docs/reference/build/index.html

For creating documentation examples see "kubebuilder create example"
`,
	Example: `# Build docs/build/index.html
kubebuilder docs

# Add examples in the right-most column then rebuild the docs
kubebuilder create example --group group --version version --kind kind
nano -w docs/reference/examples/kind/kind.yaml
kubebuilder docs


# Add manual documentation to the generated reference docs by updating the header .md files
# Edit docs/reference/static_includes/*.md
# e.g. docs/reference/static_include/_overview.md

	# <strong>API OVERVIEW</strong>
	Add your markdown here
`,
	Run: RunDocs,
}

var generateConfig bool
var cleanup bool
var outputDir string

func AddDocs(cmd *cobra.Command) {
	docsCmd.Flags().BoolVar(&cleanup, "cleanup", true, "If true, cleanup intermediary files")
	docsCmd.Flags().BoolVar(&generateConfig, "generate-config", true, "If true, generate the docs/reference/config.yaml.")
	docsCmd.Flags().StringVar(&outputDir, "output-dir", filepath.Join("docs", "reference"), "Build docs into this directory")
	cmd.AddCommand(docsCmd)
	docsCmd.AddCommand(docsCleanCmd)
}

func GetDocs() *cobra.Command {
	return docsCmd
}

var docsCleanCmd = &cobra.Command{
	Use:     "clean",
	Short:   "Removes generated docs",
	Long:    `Removes generated docs`,
	Example: ``,
	Run:     RunCleanDocs,
}

func RunCleanDocs(cmd *cobra.Command, args []string) {
	os.RemoveAll(filepath.Join(outputDir, "build"))
	os.RemoveAll(filepath.Join(outputDir, "includes"))
	os.Remove(filepath.Join(outputDir, "manifest.json"))
}

func RunDocs(cmd *cobra.Command, args []string) {
	os.RemoveAll(filepath.Join(outputDir, "includes"))
	os.MkdirAll(filepath.Join(outputDir, "openapi-spec"), 0700)
	os.MkdirAll(filepath.Join(outputDir, "static_includes"), 0700)
	os.MkdirAll(filepath.Join(outputDir, "examples"), 0700)

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("error: %v\n", err)
	}

	if generateConfig {
		CodeGenerator{}.Execute(wd)
	}

	// Run the docker command to build the docs
	c := exec.Command("docker", "run",
		"-v", fmt.Sprintf("%s:%s", filepath.Join(wd), "/host/repo"),
		"-e", "DOMAIN="+util.GetDomain(),
		"-e", "DIR="+filepath.Join("src", util.Repo),
		"-e", "OUTPUT="+outputDir,
		"gcr.io/kubebuilder/gendocs",
	)
	log.Println(strings.Join(c.Args, " "))
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	err = c.Run()
	if err != nil {
		log.Fatalf("error: %v\n", err)
	}

	// Run the docker command to build the docs
	c = exec.Command("docker", "run",
		"-v", fmt.Sprintf("%s:%s", filepath.Join(wd, outputDir, "includes"), "/source"),
		"-v", fmt.Sprintf("%s:%s", filepath.Join(wd, outputDir, "build"), "/build"),
		"-v", fmt.Sprintf("%s:%s", filepath.Join(wd, outputDir, "build"), "/build"),
		"-v", fmt.Sprintf("%s:%s", filepath.Join(wd, outputDir), "/manifest"),
		"gcr.io/kubebuilder/brodocs",
	)
	log.Println(strings.Join(c.Args, " "))
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	err = c.Run()
	if err != nil {
		log.Fatalf("error: %v\n", err)
	}

	// Cleanup intermediate files
	if cleanup {
		os.RemoveAll(filepath.Join(wd, outputDir, "includes"))
		os.RemoveAll(filepath.Join(wd, outputDir, "manifest.json"))
		os.RemoveAll(filepath.Join(wd, outputDir, "openapi-spec"))
		os.RemoveAll(filepath.Join(wd, outputDir, "build", "documents"))
		os.RemoveAll(filepath.Join(wd, outputDir, "build", "documents"))
		os.RemoveAll(filepath.Join(wd, outputDir, "build", "runbrodocs.sh"))
		os.RemoveAll(filepath.Join(wd, outputDir, "build", "node_modules", "marked", "Makefile"))
	}
}
