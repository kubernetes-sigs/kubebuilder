/*
Copyright 2016 The Kubernetes Authors.

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

package run

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/build"
	"k8s.io/client-go/util/homedir"
)

var localCmd = &cobra.Command{
	Use:   "run",
	Short: "Builds and runs the controller-manager locally against a Kubernetes cluster.",
	Long: `Builds and runs the controller-manager locally against a Kubernetes cluster.

Controller-manager will automatically install APIs in the cluster if they are missing.

To run the controller-manager remotely as a Deployment, see the instructions in the 'Dockerfile'.

To build and run the controller using Bazel, use:
  bazel run gazelle
  bazel run cmd/controller-manager:controller-manager -- --kubeconfig ~/.kube/config`,
	Example: `# Install APIs and run controller-manager against a cluster
kubebuilder run local

# RunInformersAndControllers controller-manager without rebuilding the binary
kubebuilder run local --build=false

# RunInformersAndControllers controller-manager using a specific kubeconfig
kubebuilder run local --config=path/to/kubeconfig`,
	Run: RunLocal,
}

var buildBin bool
var config string
var controllermanager string
var generate bool
var printcontrollermanager bool

func AddRun(cmd *cobra.Command) {
	localCmd.Flags().StringVar(&controllermanager, "controller-manager", "", "path to controller-manager binary to run")
	localCmd.Flags().StringVar(&config, "config", filepath.Join(homedir.HomeDir(), ".kube", "config"), "path to the kubeconfig to write for using kubectl")

	localCmd.Flags().BoolVar(&printcontrollermanager, "print-controller-manager", true, "if true, pipe the controller-manager stdout and stderr")
	localCmd.Flags().BoolVar(&buildBin, "build", true, "if true, build the binaries before running")
	localCmd.Flags().BoolVar(&generate, "generate", true, "if true, generate code before building")

	cmd.AddCommand(localCmd)
}

func RunLocal(cmd *cobra.Command, args []string) {
	if buildBin {
		fmt.Printf("Building controller executable...\n.")
		build.GenerateForBuild = generate
		build.RunBuildExecutables(cmd, args)
	}

	fmt.Printf(
		"Next: Read the 'Dockerfile.install' for instructions to build an installer for your API and controller.`\n")

	// Start controller manager
	fmt.Printf("Starting controller...\n.")
	go RunControllerManager()

	select {} // wait forever
}

func RunControllerManager() *exec.Cmd {
	if len(controllermanager) == 0 {
		controllermanager = "bin/controller-manager"
	}
	controllerManagerCmd := exec.Command(controllermanager,
		fmt.Sprintf("--kubeconfig=%s", config),
	)
	fmt.Printf("%s\n", strings.Join(controllerManagerCmd.Args, " "))
	if printcontrollermanager {
		controllerManagerCmd.Stderr = os.Stderr
		controllerManagerCmd.Stdout = os.Stdout
	}

	err := controllerManagerCmd.Run()
	if err != nil {
		defer controllerManagerCmd.Process.Kill()
		log.Fatalf("Failed to run controller-manager %v", err)
		os.Exit(-1)
	}

	return controllerManagerCmd
}
