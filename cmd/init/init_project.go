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

package init

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"golang.org/x/tools/go/packages"

	"sigs.k8s.io/kubebuilder/cmd/util"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/project"
	sutil "sigs.k8s.io/kubebuilder/pkg/scaffold/util"
)

func NewInitProjectCmd() *cobra.Command {
	o := projectOptions{}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new project",
		Long: `Initialize a new project including vendor/ directory and Go package directories.

Writes the following files:
- a boilerplate license file
- a PROJECT file with the domain and repo
- a Makefile to build the project
- a go.mod with project dependencies
- a Kustomization.yaml for customizating manifests
- a Patch file for customizing image for manager manifests
- a Patch file for enabling prometheus metrics
- a cmd/manager/main.go to run

project will prompt the user to run 'dep ensure' after writing the project files.
`,
		Example: `# Scaffold a project using the apache2 license with "The Kubernetes authors" as owners
kubebuilder init --domain example.org --license apache2 --owner "The Kubernetes authors"
`,
		Run: func(cmd *cobra.Command, args []string) {
			o.initializeProject()
		},
	}

	o.bindCmdlineFlags(initCmd)

	return initCmd
}

// module and goMod arg just enough of the output of `go mod edit -json` for our purposes
type goMod struct {
	Module module
}
type module struct {
	Path string
}

type projectOptions struct {

	// flags
	fetchDeps          bool
	skipGoVersionCheck bool

	boilerplate project.Boilerplate
	project     project.Project

	// deprecated flags
	dep     bool
	depFlag *flag.Flag
	depArgs []string

	// final result
	scaffolder scaffold.ProjectScaffolder
}

func (o *projectOptions) bindCmdlineFlags(cmd *cobra.Command) {

	cmd.Flags().BoolVar(&o.skipGoVersionCheck, "skip-go-version-check", false, "if specified, skip checking the Go version")

	// dependency args
	cmd.Flags().BoolVar(&o.fetchDeps, "fetch-deps", true, "ensure dependencies are downloaded")

	// deprecated dependency args
	cmd.Flags().BoolVar(&o.dep, "dep", true, "if specified, determines whether dep will be used.")
	o.depFlag = cmd.Flag("dep")
	cmd.Flags().StringArrayVar(&o.depArgs, "depArgs", nil, "Additional arguments for dep")
	cmd.Flags().MarkDeprecated("dep", "use the fetch-deps flag instead")
	cmd.Flags().MarkDeprecated("depArgs", "will be removed with version 1 scaffolding")

	// boilerplate args
	cmd.Flags().StringVar(&o.boilerplate.Path, "path", "", "path for boilerplate")
	cmd.Flags().StringVar(&o.boilerplate.License, "license", "apache2", "license to use to boilerplate.  May be one of apache2,none")
	cmd.Flags().StringVar(&o.boilerplate.Owner, "owner", "", "Owner to add to the copyright")

	// project args
	cmd.Flags().StringVar(&o.project.Repo, "repo", "", "name to use for go module, e.g. github.com/user/repo.  "+
		"defaults to the go package of the current working directory.")
	cmd.Flags().StringVar(&o.project.Domain, "domain", "my.domain", "domain for groups")
	cmd.Flags().StringVar(&o.project.Version, "project-version", project.Version2, "project version")
}

func (o *projectOptions) initializeProject() {
	if err := o.validate(); err != nil {
		log.Fatal(err)
	}

	if err := o.scaffolder.Scaffold(); err != nil {
		log.Fatalf("error scaffolding project: %v", err)
	}

	if err := o.postScaffold(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Next: Define a resource with:\n" +
		"$ kubebuilder create api\n")
}

func (o *projectOptions) validate() error {
	if !o.skipGoVersionCheck {
		if err := validateGoVersion(); err != nil {
			return err
		}
	}
	if o.project.Repo == "" {
		repoPath, err := findCurrentRepo()
		if err != nil {
			return fmt.Errorf("error finding current repository: %v", err)
		}
		o.project.Repo = repoPath
	}

	switch o.project.Version {
	case project.Version1:
		var defEnsure *bool
		if o.depFlag.Changed {
			defEnsure = &o.dep
		}
		o.scaffolder = &scaffold.V1Project{
			Project:     o.project,
			Boilerplate: o.boilerplate,

			DepArgs:          o.depArgs,
			DefinitelyEnsure: defEnsure,
		}
	case project.Version2:
		o.scaffolder = &scaffold.V2Project{
			Project:     o.project,
			Boilerplate: o.boilerplate,
		}
	default:
		return fmt.Errorf("unknown project version %v", o.project.Version)
	}

	if err := o.scaffolder.Validate(); err != nil {
		return err
	}

	if util.ProjectExist() {
		return fmt.Errorf("Failed to initialize project because project is already initialized")
	}

	return nil
}

func validateGoVersion() error {
	err := fetchAndCheckGoVersion()
	if err != nil {
		return fmt.Errorf("%s. You can skip this check using the --skip-go-version-check flag", err)
	}
	return nil
}

func fetchAndCheckGoVersion() error {
	cmd := exec.Command("go", "version")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Failed to retrieve 'go version': %v", string(out))
	}

	split := strings.Split(string(out), " ")
	if len(split) < 3 {
		return fmt.Errorf("Found invalid Go version: %q", string(out))
	}
	goVer := split[2]
	if err := checkGoVersion(goVer); err != nil {
		return fmt.Errorf("Go version '%s' is incompatible because '%s'", goVer, err)
	}
	return nil
}

func checkGoVersion(verStr string) error {
	goVerRegex := `^go?([0-9]+)\.([0-9]+)([\.0-9A-Za-z\-]+)?$`
	m := regexp.MustCompile(goVerRegex).FindStringSubmatch(verStr)
	if m == nil {
		return fmt.Errorf("invalid version string")
	}

	major, err := strconv.Atoi(m[1])
	if err != nil {
		return fmt.Errorf("error parsing major version '%s': %s", m[1], err)
	}

	minor, err := strconv.Atoi(m[2])
	if err != nil {
		return fmt.Errorf("error parsing minor version '%s': %s", m[2], err)
	}

	if major < 1 || minor < 11 {
		return fmt.Errorf("requires version >= 1.11")
	}

	return nil
}

// findCurrentRepo attempts to determine the current repository
// though a combination of go/packages and `go mod` commands/tricks.
func findCurrentRepo() (string, error) {
	// easiest case: project file already exists
	projFile, err := sutil.LoadProjectFile("PROJECT")
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

func (o *projectOptions) postScaffold() error {
	// preserve old "ask if not explicitly set" behavior for the `--dep` flag
	// (asking is handled by the v1 scaffolder)
	if (o.depFlag.Changed && !o.dep) || !o.fetchDeps {
		fmt.Println("Skipping fetching dependencies.")
		return nil
	}

	ensured, err := o.scaffolder.EnsureDependencies()
	if err != nil {
		return err
	}

	if !ensured {
		return nil
	}

	fmt.Println("Running make...")
	c := exec.Command("make") // #nosec
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	fmt.Println(strings.Join(c.Args, " "))
	return c.Run()
}
