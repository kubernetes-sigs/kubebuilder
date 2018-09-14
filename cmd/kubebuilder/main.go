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
	gobuild "go/build"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/initproject"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/v0"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/v1"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/version"
	"github.com/spf13/cobra"
)

func main() {
	util.CheckInstall()
	gopath := os.Getenv("GOPATH")
	if len(gopath) == 0 {
		gopath = gobuild.Default.GOPATH
	}
	util.GoSrc = filepath.Join(gopath, "src")

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if !strings.HasPrefix(filepath.Dir(wd), util.GoSrc) {
		log.Fatalf("kubebuilder must be run from the project root under $GOPATH/src/<package>. "+
			"\nCurrent GOPATH=%s.  \nCurrent directory=%s", gopath, wd)
	}
	util.Repo = strings.Replace(wd, util.GoSrc+string(filepath.Separator), "", 1)
	initproject.AddInit(cmd)
	version.AddVersion(cmd)

	if util.IsNewVersion() || util.IsProjectNotInitialized() {
		v1.AddCmds(cmd)
	} else {
		v0.AddCmds(cmd)
	}

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var cmd = &cobra.Command{
	Use:   "kubebuilder",
	Short: "Development kit for building Kubernetes extensions and tools.",
	Run:   RunMain,
}

func RunMain(cmd *cobra.Command, args []string) {
	cmd.Help()
}
