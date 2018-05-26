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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/kubernetes-sigs/kubebuilder/test/e2e/framework"
	"github.com/kubernetes-sigs/kubebuilder/test/e2e/framework/ginkgowrapper"
	e2einternal "github.com/kubernetes-sigs/kubebuilder/test/internal/e2e"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/wait"
)

// RunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
func RunE2ETests(t *testing.T) {
	RegisterFailHandler(ginkgowrapper.Fail)
	glog.Infof("Starting kubebuilder suite")
	RunSpecs(t, "Kubebuilder e2e suite")
}

var _ = Describe("main workflow", func() {
	It("should perform main kubebuilder workflow successfully", func() {
		testSuffix := framework.RandomSuffix()
		c := initConfig(testSuffix)
		kubebuilderTest := e2einternal.NewKubebuilderTest(c.workDir, framework.TestContext.BinariesDir)

		prepare(c.workDir)
		defer cleanup(kubebuilderTest, c.workDir, c.controllerImageName)

		var controllerPodName string
		var e error

		By("init project")
		initOptions := []string{"--domain", c.domain}
		err := kubebuilderTest.Init(initOptions)
		Expect(err).NotTo(HaveOccurred())

		By("creating resource definition")
		resourceOptions := []string{"--group", c.group, "--version", c.version, "--kind", c.kind}
		err = kubebuilderTest.CreateResource(resourceOptions)
		Expect(err).NotTo(HaveOccurred())

		By("creating core-type resource controller")
		controllerOptions := []string{"--group", "apps", "--version", "v1beta2", "--kind", "Deployment", "--core-type"}
		err = kubebuilderTest.CreateController(controllerOptions)
		Expect(err).NotTo(HaveOccurred())

		By("building image")
		// The scaffold test cases generated for core types cannot work without manually modification.
		// See https://github.com/kubernetes-sigs/kubebuilder/pull/193 for more details
		// Skip the test part in build process as we don't care about it here.
		err = framework.ReplaceFileConent(`RUN go test(.*)\n`, "", filepath.Join(c.workDir, "Dockerfile.controller"))
		Expect(err).NotTo(HaveOccurred())

		imageOptions := []string{"-t", c.controllerImageName}
		err = kubebuilderTest.BuildImage(imageOptions)
		Expect(err).NotTo(HaveOccurred())

		By("creating config")
		configOptions := []string{"--controller-image", c.controllerImageName, "--name", c.installName}
		err = kubebuilderTest.CreateConfig(configOptions)
		Expect(err).NotTo(HaveOccurred())

		By("installing controller-manager in cluster")
		inputFile := filepath.Join(kubebuilderTest.Dir, "hack", "install.yaml")
		installOptions := []string{"apply", "-f", inputFile}
		_, err = kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(installOptions))
		Expect(err).NotTo(HaveOccurred())

		By("validate the controller-manager pod running as expected")
		verifyContollerUp := func() (bool, error) {
			e = nil
			// Get pod name
			getOptions := []string{"get", "pods", "-n", c.namespace, "-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}{{ \"\\n\" }}{{ end }}{{ end }}"}
			podOutput, err := kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(getOptions))
			Expect(err).NotTo(HaveOccurred())
			// TODO: validate pod replicas if not default to 1
			podNames := framework.ParseCmdOutput(podOutput)
			if len(podNames) != 1 {
				e = fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				return false, nil
			}
			controllerPodName = podNames[0]

			// Validate pod status
			getOptions = []string{"get", "pods", controllerPodName, "-n", c.namespace, "-o", "jsonpath={.status.phase}"}
			status, err := kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(getOptions))
			Expect(err).NotTo(HaveOccurred())
			if status != "Running" {
				e = fmt.Errorf("controller pod in %s status", status)
				return false, nil
			}

			return true, nil
		}
		varifyErr := wait.PollImmediate(500*time.Millisecond, 1*time.Minute, verifyContollerUp)
		if varifyErr != nil {
			framework.Failf(e.Error())
		}

		By("creating resource object")
		inputFile = filepath.Join(kubebuilderTest.Dir, "hack", "sample", strings.ToLower(c.kind)+".yaml")
		createOptions := []string{"create", "-f", inputFile}
		_, err = kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(createOptions))
		Expect(err).NotTo(HaveOccurred())

		By("validate the created resource object gets reconciled in controller")
		verifyResourceReconciled := func() (bool, error) {
			// Check container log to validate that the created resource object gets reconciled in controller
			logOptions := []string{"logs", controllerPodName, "-n", c.namespace}
			logOutput, err := kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(logOptions))
			Expect(err).NotTo(HaveOccurred())

			if !strings.Contains(logOutput, fmt.Sprintf("to reconcile %s-example", strings.ToLower(c.kind))) {
				e = fmt.Errorf("created resource object %s-example not reconciled yet", strings.ToLower(c.kind))
				return false, nil
			}
			return true, nil
		}
		varifyErr = wait.PollImmediate(500*time.Millisecond, 1*time.Minute, verifyResourceReconciled)
		if varifyErr != nil {
			framework.Failf(e.Error())
		}

		By("creating other kind of resource object")
		inputFile = filepath.Join(kubebuilderTest.Dir, "hack", "sample", "deployment.yaml")
		util.WriteIfNotFound(inputFile,
			"deployment-template",
			deploymentTemplate,
			deploymentTemplateArguments{Namespace: c.namespace},
		)
		createOptions = []string{"create", "-f", inputFile}
		_, err = kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(createOptions))
		Expect(err).NotTo(HaveOccurred())

		By("validate other kind of object gets reconciled in controller")
		verifyResourceReconciled = func() (bool, error) {
			// Check container log to validate that the created resource object gets reconciled in controller
			logOptions := []string{"logs", controllerPodName, "-n", c.namespace}
			logOutput, err := kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(logOptions))
			Expect(err).NotTo(HaveOccurred())

			if !strings.Contains(logOutput, "to reconcile deployment-example") {
				e = fmt.Errorf("created resource object deployment-example not reconciled yet")
				return false, nil
			}
			return true, nil
		}
		varifyErr = wait.PollImmediate(500*time.Millisecond, 1*time.Minute, verifyResourceReconciled)
		if varifyErr != nil {
			framework.Failf(e.Error())
		}
	})
})

func prepare(workDir string) {
	By("create a path under given project dir, as the test work dir")
	err := os.MkdirAll(workDir, 0755)
	Expect(err).NotTo(HaveOccurred())
}

func cleanup(builderTest *e2einternal.KubebuilderTest, workDir string, imageName string) {
	By("clean up created API objects during test process")
	inputFile := filepath.Join(workDir, "hack", "install.yaml")
	createOptions := []string{"delete", "-f", inputFile}
	builderTest.RunKubectlCommand(framework.GetKubectlArgs(createOptions))

	By("remove container image created during test")
	builderTest.CleanupImage([]string{imageName})

	By("remove test work dir")
	os.RemoveAll(workDir)
}
