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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

var GenerateForBuild = true
var goos = "linux"
var goarch = "amd64"
var outputdir = "bin"

var createBuildExecutablesCmd = &cobra.Command{
	Use:   "executables",
	Short: "Builds the source into executables to run on the local machine",
	Long:  `Builds the source into executables to run on the local machine`,
	Example: `# Generate code and build the apiserver and controller
# binaries in the bin directory so they can be run locally.
kubebuilder build executables

# Build binaries into the linux/ directory using the cross compiler for linux:amd64
kubebuilder build executables --goos linux --goarch amd64 --output linux/

# Regenerate Bazel BUILD files, and then build with bazel
# Must first install bazel and gazelle !!!
kubebuilder build executables --bazel --gazelle

# RunInformersAndControllers Bazel without generating BUILD files
kubebuilder build executables --bazel

# RunInformersAndControllers Bazel without generating BUILD files or generated code
kubebuilder build executables --bazel --generated=false
`,
	Run: RunBuildExecutables,
}

func AddBuildExecutables(cmd *cobra.Command) {
	cmd.AddCommand(createBuildExecutablesCmd)

	createBuildExecutablesCmd.Flags().StringVar(&vendorDir, "vendor-dir", "", "Location of directory containing vendor files.")
	createBuildExecutablesCmd.Flags().BoolVar(&GenerateForBuild, "generate", true, "if true, generate code before building")
	createBuildExecutablesCmd.Flags().StringVar(&goos, "goos", "", "if specified, set this GOOS")
	createBuildExecutablesCmd.Flags().StringVar(&goarch, "goarch", "", "if specified, set this GOARCH")
	createBuildExecutablesCmd.Flags().StringVar(&outputdir, "output", "bin", "if set, write the binaries to this directory")
}

func RunBuildExecutables(cmd *cobra.Command, args []string) {
	GoBuild(cmd, args)
}

func GoBuild(cmd *cobra.Command, args []string) {
	if GenerateForBuild {
		RunGenerate(cmd, args)
	}

	os.RemoveAll(filepath.Join("bin", "controller-manager"))

	// Build the controller manager
	path := filepath.Join("cmd", "controller-manager", "main.go")
	c := exec.Command("go", "build", "-o", filepath.Join(outputdir, "controller-manager"), path)
	c.Env = append(os.Environ(), "CGO_ENABLED=0")
	if len(goos) > 0 {
		c.Env = append(c.Env, fmt.Sprintf("GOOS=%s", goos))
	}
	if len(goarch) > 0 {
		c.Env = append(c.Env, fmt.Sprintf("GOARCH=%s", goarch))
	}

	glog.V(4).Infof("%s\n", strings.Join(c.Args, " "))
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	err := c.Run()
	if err != nil {
		log.Fatal(err)
	}
}
