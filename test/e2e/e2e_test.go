/*
Copyright 2018 The Kubernetes Authors.

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

package e2e

import (
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kubebuilder/test/e2e/framework"
	"github.com/kubernetes-sigs/kubebuilder/test/e2e/framework/ginkgowrapper"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

func init() {
	framework.RegisterFlags()
	pflag.Parse()
}

var kubebuilderBinDir string

/*
2 steps to do before running the e2e test suite
1) go build a new kubebuilder binary
2) install all required binaries, such as etcd and kube-apiserver
If you have run testv0.sh or testv1.sh before running this e2e suite,
it will build the new kubebuilder.
*/
// TODO: ensure the required binaries are installed when integrate with Prow.
var _ = BeforeSuite(func(done Done) {
	// Uncomment the following line to set the image name before runing the e2e test
	//os.Setenv("IMG", "gcr.io/<my-project-name>/<iamge-name:tag>")
	// If you want to run the test against a GKE cluster, run the following command first
	// $ kubectl create clusterrolebinding myname-cluster-admin-binding --clusterrole=cluster-admin --user=myname@mycompany.com
	framework.TestContext.BinariesDir = "/tmp/kubebuilder/bin/"
	// build a kubebuilder
	targets := []string{"kubebuilder", "kubebuilder-gen"}
	for _, target := range targets {
		buildOptions := []string{
			"build", "-o", path.Join(framework.TestContext.BinariesDir, target), path.Join("github.com/kubernetes-sigs/kubebuilder/cmd", target)}
		cmd := exec.Command("go", buildOptions...)
		cmd.Env = os.Environ()
		command := strings.Join(cmd.Args, " ")
		log.Printf("running %v", command)
		output, err := cmd.CombinedOutput()
		log.Printf("output when running:\n%s", output)
		Expect(err).NotTo(HaveOccurred())
	}

	close(done)
}, 60)

var _ = AfterSuite(func() {
	os.RemoveAll(kubebuilderBinDir)
})

// RunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
func RunE2ETests(t *testing.T) {
	RegisterFailHandler(ginkgowrapper.Fail)
	glog.Infof("Starting kubebuilder suite")
	RunSpecs(t, "Kubebuilder e2e suite")
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
