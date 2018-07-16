// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"sigs.k8s.io/controller-tools/pkg/scaffold"
	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
	"sigs.k8s.io/controller-tools/pkg/scaffold/manager"
	"sigs.k8s.io/controller-tools/pkg/scaffold/project"
)

var prj *project.Project
var bp *project.Boilerplate
var gopkg *project.GopkgToml
var mrg *manager.Cmd
var dkr *manager.Dockerfile
var dep bool
var depFlag *flag.Flag
var mkFile *project.Makefile

// default controller manager image name
var imgName = "controller:latest"

// ProjectCmd represents the project command
var ProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Scaffold a new project.",
	Long: `Scaffold a project.

Writes the following files:
- a boilerplate license file
- a PROJECT file with the domain and repo
- a Makefile to build the project
- a Gopkg.toml with project dependencies
- a Kustomization.yaml for customizating manifests
- a Patch file for customizing image for manager manifests
- a cmd/manager/main.go to run

project will prompt the user to run 'dep ensure' after writing the project files.
`,
	Example: `# Scaffold a project using the apache2 license with "The Kubernetes authors" as owners
controller-scaffold project --domain k8s.io --license apache2 --owner "The Kubernetes authors"
`,
	Run: func(cmd *cobra.Command, args []string) {
		// project and boilerplate must come before main so the boilerplate exists
		s := &scaffold.Scaffold{
			BoilerplateOptional: true,
			ProjectOptional:     true,
		}

		p, err := prj.GetInput()
		if err != nil {
			log.Fatal(err)
		}

		b, err := bp.GetInput()
		if err != nil {
			log.Fatal(err)
		}

		err = s.Execute(input.Options{ProjectPath: p.Path, BoilerplatePath: b.Path}, prj, bp)
		if err != nil {
			log.Fatal(err)
		}

		s = &scaffold.Scaffold{}
		err = s.Execute(input.Options{ProjectPath: p.Path, BoilerplatePath: b.Path},
			gopkg,
			mrg,
			mkFile,
			dkr,
			&manager.APIs{},
			&manager.Controller{},
			&manager.Config{Image: imgName},
			&project.GitIgnore{},
			&project.Kustomize{},
			&project.KustomizeImagePatch{})
		if err != nil {
			log.Fatal(err)
		}

		if !depFlag.Changed {
			fmt.Println("Run `dep ensure` to fetch dependencies (Recommended) [y/n]?")
			dep = yesno()
		}
		if dep {
			c := exec.Command("dep", "ensure") // #nosec
			c.Stderr = os.Stderr
			c.Stdout = os.Stdout
			fmt.Println(strings.Join(c.Args, " "))
			if err := c.Run(); err != nil {
				log.Fatal(err)
			}

			fmt.Println("Running make...")
			c = exec.Command("make") // #nosec
			c.Stderr = os.Stderr
			c.Stdout = os.Stdout
			fmt.Println(strings.Join(c.Args, " "))
			if err := c.Run(); err != nil {
				log.Fatal(err)
			}
		} else {
			fmt.Println("Skipping `dep ensure`.  Dependencies will not be fetched.")
		}
	},
}

func init() {
	rootCmd.AddCommand(ProjectCmd)

	ProjectCmd.Flags().BoolVar(
		&dep, "dep", true, "if specified, determines whether dep will be used.")
	depFlag = ProjectCmd.Flag("dep")
	prj = ProjectForFlags(ProjectCmd.Flags())
	bp = BoilerplateForFlags(ProjectCmd.Flags())
	gopkg = &project.GopkgToml{}
	mrg = &manager.Cmd{}
	dkr = &manager.Dockerfile{}
	mkFile = MakefileForFlags(ProjectCmd.Flags())
}

// ProjectForFlags registers flags for Project fields and returns the Project
func ProjectForFlags(f *flag.FlagSet) *project.Project {
	p := &project.Project{}
	f.StringVar(&p.Domain, "domain", "k8s.io", "domain for groups")
	f.StringVar(&p.Version, "project-version", "2", "project version")
	f.StringVar(&p.Repo, "repo", "", "name of the github repo.  "+
		"defaults to the go package of the current working directory.")
	return p
}

// BoilerplateForFlags registers flags for Boilerplate fields and returns the Boilerplate
func BoilerplateForFlags(f *flag.FlagSet) *project.Boilerplate {
	b := &project.Boilerplate{}
	f.StringVar(&b.Path, "path", "", "domain for groups")
	f.StringVar(&b.License, "license", "apache2",
		"license to use to boilerplate.  Maybe one of apache2,none")
	f.StringVar(&b.Owner, "owner", "",
		"Owner to add to the copyright")
	return b
}

// MakefileForFlags registers flags for Makefile fields and returns the Makefile
func MakefileForFlags(f *flag.FlagSet) *project.Makefile {
	m := &project.Makefile{Image: imgName}
	f.StringVar(&m.ControllerToolsPath, "controller-tools-path", "", "path to controller tools repo")
	return m
}
