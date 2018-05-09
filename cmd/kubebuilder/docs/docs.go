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

	generatecmd "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/generate"
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
var cleanup, verbose bool
var outputDir string
var copyright string
var brodocs bool

func AddDocs(cmd *cobra.Command) {
	docsCmd.Flags().BoolVar(&cleanup, "cleanup", true, "If true, cleanup intermediary files")
	docsCmd.Flags().BoolVar(&verbose, "verbose", true, "If true, use verbose output")
	docsCmd.Flags().BoolVar(&generateConfig, "generate-config", true, "If true, generate the docs/reference/config.yaml.")
	docsCmd.Flags().StringVar(&outputDir, "output-dir", filepath.Join("docs", "reference"), "Build docs into this directory")
	docsCmd.Flags().StringVar(&copyright, "copyright", filepath.Join("hack", "boilerplate.go.txt"), "Location of copyright boilerplate file.")
	docsCmd.Flags().BoolVar(&brodocs, "brodocs", true, "Run brodocs to generate html.")
	docsCmd.Flags().StringVar(&generatecmd.Docscopyright, "docs-copyright", "<a href=\"https://github.com/kubernetes/kubernetes\">Copyright 2018 The Kubernetes Authors.</a>", "html for the copyright text on the docs")
	docsCmd.Flags().StringVar(&generatecmd.Docstitle, "title", "API Reference", "title of the docs page")
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

var openapipkg = filepath.Join("pkg", "generated", "openapi")

func RunDocs(cmd *cobra.Command, args []string) {
	// Delete old build artifacts
	os.RemoveAll(filepath.Join(outputDir, "includes"))
	os.RemoveAll(filepath.Join(outputDir, "build"))
	os.Remove(filepath.Join(outputDir, "manifest.json"))

	os.MkdirAll(filepath.Join(outputDir, "openapi-spec"), 0700)
	os.MkdirAll(filepath.Join(outputDir, "static_includes"), 0700)
	os.MkdirAll(filepath.Join(outputDir, "examples"), 0700)

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("error: %v\n", err)
	}

	if generateConfig {
		// Regenerate the config.yaml with the table of contents
		os.Remove(filepath.Join(outputDir, "config.yaml"))
		CodeGenerator{}.Execute(wd)
	}

	// Make sure to generate the openapi
	generatecmd.Codegenerators = []string{"openapi"}
	generatecmd.RunGenerate(cmd, args)

	// Create the go program to create the swagger.json by serializing the openapi go struct
	cr := util.GetCopyright(copyright)
	doSwaggerGen(wd, swaggerGenTemplateArgs{
		cr,
		util.Repo,
	})
	defer func() {
		if cleanup {
			os.RemoveAll(filepath.Join(wd, filepath.Join("pkg", "generated")))
			os.RemoveAll(filepath.Join(wd, outputDir, "includes"))
			os.RemoveAll(filepath.Join(wd, outputDir, "manifest.json"))
			os.RemoveAll(filepath.Join(wd, outputDir, "build", "documents"))
			os.RemoveAll(filepath.Join(wd, outputDir, "build", "documents"))
			os.RemoveAll(filepath.Join(wd, outputDir, "build", "runbrodocs.sh"))
			os.RemoveAll(filepath.Join(wd, outputDir, "build", "node_modules", "marked", "Makefile"))
		}
	}()

	// Run the go program to write the swagger.json output to a file
	c := exec.Command("go", "run", filepath.Join(openapipkg, "cmd", "main.go"))
	if verbose {
		log.Println(strings.Join(c.Args, " "))
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout
	}
	err = c.Run()
	if err != nil {
		log.Fatalf("error: %v\n", err)
	}

	// Call the apidocs code generator to create the markdown files that will be converted into
	// html
	generatecmd.Codegenerators = []string{"apidocs"}
	generatecmd.RunGenerate(cmd, args)

	if brodocs {
		// Run the docker command to build the docs
		c = exec.Command("docker", "run",
			"-v", fmt.Sprintf("%s:%s", filepath.Join(wd, outputDir, "includes"), "/source"),
			"-v", fmt.Sprintf("%s:%s", filepath.Join(wd, outputDir, "build"), "/build"),
			"-v", fmt.Sprintf("%s:%s", filepath.Join(wd, outputDir, "build"), "/build"),
			"-v", fmt.Sprintf("%s:%s", filepath.Join(wd, outputDir), "/manifest"),
			"gcr.io/kubebuilder/brodocs",
		)
		if verbose {
			log.Println(strings.Join(c.Args, " "))
			c.Stderr = os.Stderr
			c.Stdout = os.Stdout
		}
		err = c.Run()
		if err != nil {
			log.Fatalf("error: %v\n", err)
		}
		fmt.Printf("Reference docs written to %s\n", filepath.Join(outputDir, "build", "index.html"))
	}
}

// Scaffolding file for writing the openapi generated structs to a swagger.json file
type swaggerGenTemplateArgs struct {
	BoilerPlate string
	Repo        string
}

// Create a go file that will take the generated openapi.go file and serialize the go structs into a swagger.json.
func doSwaggerGen(dir string, args swaggerGenTemplateArgs) bool {
	path := filepath.Join(dir, openapipkg, "cmd", "main.go")
	return util.WriteIfNotFound(path, "swagger-template", swaggerGenTemplate, args)
}

var swaggerGenTemplate = `
{{.BoilerPlate}}

package main

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/docs"
	"{{ .Repo }}/pkg/generated/openapi"
)

func main() {
	docs.WriteOpenAPI(openapi.GetOpenAPIDefinitions)
}
`
