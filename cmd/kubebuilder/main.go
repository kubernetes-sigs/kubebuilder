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
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/build"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/initproject"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/update"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/version"
)

func main() {
	util.CheckInstall()
	gopath := os.Getenv("GOPATH")
	if len(gopath) == 0 {
		log.Fatal("GOPATH not defined")
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
	create.AddCreate(cmd)
	build.AddBuild(cmd)
	// run.AddRun(cmd)
	update.AddUpdate(cmd)
	version.AddVersion(cmd)

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var cmd = &cobra.Command{
	Use:   "kubebuilder",
	Short: "Development kit for building Kubernetes extensions and tools.",
	Long: `Development kit for building Kubernetes extensions and tools.

Provides libraries and tools to create new projects, APIs and controllers.
Includes tools for packaging artifacts into an installer container.

Typical project lifecycle:

- initialize a project:

  kubebuilder init --domain example.com

- create one or more a new resource APIs and add your code to them:

  kubebuilder create resource --group <group> --version <version> --kind <Kind>

- run the controller locally, installing APIs into the cluster if they are missing:

  GOBIN=$(pwd)/bin go install <your-project-go-package>/cmd/controller-manager
  bin/controller-manager --kubeconfig ~/.kube/config

- build a docker container to install the API and controller into a namespace with RBAC configured:

  Note: You may need to give yourself admin privs in order to install the RBAC rules
  kubectl create clusterrolebinding <your-binding-name> --clusterrole=cluster-admin --user=<your-user>

  docker build -f Dockerfile.controller . -t <image:tag>
  docker push <image:tag>
  kubebuilder create config --controller-image <image:tag> --name <project-name>
  kubectl apply -f hack/install.yaml

More options:

- run tests
  kubebuilder build generated
  export TEST_ASSET_KUBECTL=/usr/local/kubebuilder/bin/kubectl
  export TEST_ASSET_KUBE_APISERVER=/usr/local/kubebuilder/bin/kube-apiserver
  export TEST_ASSET_ETCD=/usr/local/kubebuilder/bin/etcd
  go test ./pkg/...
`,
	Example: `# Initialize your project
kubebuilder init --domain example.com

# Initialize your project adding a go-header file to all generated files
touch hack/boilerplate.go.txt
kubebuilder init --domain example.com`,
	Run: RunMain,
}

func RunMain(cmd *cobra.Command, args []string) {
	cmd.Help()
}
