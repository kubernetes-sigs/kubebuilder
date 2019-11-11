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

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"

	"sigs.k8s.io/kubebuilder/cmd/version"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/project"
)

const (
	NoticeColor = "\033[1;36m%s\033[0m"
)

// module and goMod arg just enough of the output of `go mod edit -json` for our purposes
type goMod struct {
	Module module
}
type module struct {
	Path string
}

// findGoModulePath finds the path of the current module, if present.
func findGoModulePath(forceModules bool) (string, error) {
	cmd := exec.Command("go", "mod", "edit", "-json")
	cmd.Env = append(cmd.Env, os.Environ()...)
	if forceModules {
		cmd.Env = append(cmd.Env, "GO111MODULE=on" /* turn on modules just for these commands */)
	}
	out, err := cmd.Output()
	if err != nil {
		if exitErr, isExitErr := err.(*exec.ExitError); isExitErr {
			err = fmt.Errorf("%s", string(exitErr.Stderr))
		}
		return "", err
	}
	mod := goMod{}
	if err := json.Unmarshal(out, &mod); err != nil {
		return "", err
	}
	return mod.Module.Path, nil
}

// findCurrentRepo attempts to determine the current repository
// though a combination of go/packages and `go mod` commands/tricks.
func findCurrentRepo() (string, error) {
	// easiest case: project file already exists
	projFile, err := scaffold.LoadProjectFile("PROJECT")
	if err == nil {
		return projFile.Repo, nil
	}

	// next easy case: existing go module
	path, err := findGoModulePath(false)
	if err == nil {
		return path, nil
	}

	// next, check if we've got a package in the current directory
	pkgCfg := &packages.Config{
		Mode: packages.NeedName, // name gives us path as well
	}
	pkgs, err := packages.Load(pkgCfg, ".")
	// NB(directxman12): when go modules are off and we're outside GOPATH and
	// we don't otherwise have a good guess packages.Load will fabricate a path
	// that consists of `_/absolute/path/to/current/directory`.  We shouldn't
	// use that when it happens.
	if err == nil && len(pkgs) > 0 && len(pkgs[0].PkgPath) > 0 && pkgs[0].PkgPath[0] != '_' {
		return pkgs[0].PkgPath, nil
	}

	// otherwise, try to get `go mod init` to guess for us -- it's pretty good
	cmd := exec.Command("go", "mod", "init")
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "GO111MODULE=on" /* turn on modules just for these commands */)
	if _, err := cmd.Output(); err != nil {
		if exitErr, isExitErr := err.(*exec.ExitError); isExitErr {
			err = fmt.Errorf("%s", string(exitErr.Stderr))
		}
		// give up, let the user figure it out
		return "", fmt.Errorf("could not determine repository path from module data, package data, or by initializing a module: %v", err)
	}
	defer os.Remove("go.mod") // clean up after ourselves
	return findGoModulePath(true)
}

func main() {
	rootCmd := defaultCommand()

	rootCmd.AddCommand(
		newInitProjectCmd(),
		newCreateCmd(),
		version.NewVersionCmd(),
	)

	foundProject, projectVersion := getProjectVersion()
	if foundProject && projectVersion == project.Version1 {
		printV1DeprecationWarning()

		rootCmd.AddCommand(
			newAlphaCommand(),
			newVendorUpdateCmd(),
		)
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func defaultCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "kubebuilder",
		Short: "Development kit for building Kubernetes extensions and tools.",
		Long: `
Development kit for building Kubernetes extensions and tools.

Provides libraries and tools to create new projects, APIs and controllers.
Includes tools for packaging artifacts into an installer container.

Typical project lifecycle:

- initialize a project:

  kubebuilder init --domain example.com --license apache2 --owner "The Kubernetes authors"

- create one or more a new resource APIs and add your code to them:

  kubebuilder create api --group <group> --version <version> --kind <Kind>

Create resource will prompt the user for if it should scaffold the Resource and / or Controller. To only
scaffold a Controller for an existing Resource, select "n" for Resource. To only define
the schema for a Resource without writing a Controller, select "n" for Controller.

After the scaffold is written, api will run make on the project.
`,
		Example: `
	# Initialize your project
	kubebuilder init --domain example.com --license apache2 --owner "The Kubernetes authors"

	# Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
	kubebuilder create api --group ship --version v1beta1 --kind Frigate

	# Edit the API Scheme
	nano api/v1beta1/frigate_types.go

	# Edit the Controller
	nano controllers/frigate_controller.go

	# Install CRDs into the Kubernetes cluster using kubectl apply
	make install

	# Regenerate code and run against the Kubernetes cluster configured by ~/.kube/config
	make run
`,

		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
}

// getProjectVersion tries to load PROJECT file and returns if the file exist
// and the version string
func getProjectVersion() (bool, string) {
	if _, err := os.Stat("PROJECT"); os.IsNotExist(err) {
		return false, ""
	}
	projectInfo, err := scaffold.LoadProjectFile("PROJECT")
	if err != nil {
		log.Fatalf("failed to read the PROJECT file: %v", err)
	}
	return true, projectInfo.Version
}

func printV1DeprecationWarning() {
	fmt.Printf(NoticeColor, "[Deprecation Notice] The v1 projects are deprecated and will not be supported beyond Feb 1, 2020.\nSee how to upgrade your project to v2: https://book.kubebuilder.io/migration/guide.html\n")
}
