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
	"path/filepath"
	"strings"
	"time"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/kubernetes-sigs/kubebuilder/test/e2e/framework"
	e2einternal "github.com/kubernetes-sigs/kubebuilder/test/internal/e2e"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("v0 main workflow", func() {
	It("should perform main kubebuilder workflow successfully", func() {
		testSuffix := framework.RandomSuffix()
		c := initConfig(testSuffix)
		kubebuilderTest := e2einternal.NewKubebuilderTest(c.workDir, framework.TestContext.BinariesDir)

		prepare(c.workDir)
		defer cleanupv0(kubebuilderTest, c.workDir, c.controllerImageName)

		var controllerPodName string

		By("init project")
		initOptions := []string{"--domain", c.domain, "--project-version", "v0"}
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
		// The scaffold test cases generated for core types controller cannot work
		// without manually modification.
		// See https://github.com/kubernetes-sigs/kubebuilder/pull/193 for more details
		// Skip the test for core types controller in build process.
		testCmdWithoutCoreType := "RUN find ./ -not -path './pkg/controller/deployment/*' -name '*_test.go' -print0 | xargs -0n1 dirname | xargs go test\n"
		err = framework.ReplaceFileConent(`RUN go test(.*)\n`, testCmdWithoutCoreType, filepath.Join(c.workDir, "Dockerfile.controller"))
		Expect(err).NotTo(HaveOccurred())

		imageOptions := []string{"-t", c.controllerImageName}
		err = kubebuilderTest.BuildImage(imageOptions)
		Expect(err).NotTo(HaveOccurred())

		By("creating config")
		configOptions := []string{"--controller-image", c.controllerImageName, "--name", c.installName}
		err = kubebuilderTest.CreateConfig(configOptions)
		Expect(err).NotTo(HaveOccurred())

		By("installing controller-manager in cluster")
		inputFile := filepath.Join("hack", "install.yaml")
		installOptions := []string{"apply", "-f", inputFile}
		_, err = kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(installOptions))
		Expect(err).NotTo(HaveOccurred())

		By("validate the controller-manager pod running as expected")
		verifyControllerUp := func() error {
			// Get pod name
			// TODO: Use kubectl to format the output with a go-template
			getOptions := []string{"get", "pods", "-n", c.namespace, "-l", "control-plane=controller-manager", "-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}{{ \"\\n\" }}{{ end }}{{ end }}"}
			podOutput, err := kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(getOptions))
			Expect(err).NotTo(HaveOccurred())
			// TODO: validate pod replicas if not default to 1
			podNames := framework.ParseCmdOutput(podOutput)
			if len(podNames) != 1 {
				return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
			}
			controllerPodName = podNames[0]
			Expect(controllerPodName).Should(HavePrefix(c.installName + "-controller-manager"))

			// Validate pod status
			getOptions = []string{"get", "pods", controllerPodName, "-n", c.namespace, "-o", "jsonpath={.status.phase}"}
			status, err := kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(getOptions))
			Expect(err).NotTo(HaveOccurred())
			if status != "Running" {
				return fmt.Errorf("controller pod in %s status", status)
			}

			return nil
		}
		Eventually(verifyControllerUp, 1*time.Minute, 500*time.Millisecond).Should(BeNil())

		By("creating resource object")
		inputFile = filepath.Join(kubebuilderTest.Dir, "hack", "sample", strings.ToLower(c.kind)+".yaml")
		createOptions := []string{"create", "-f", inputFile}
		_, err = kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(createOptions))
		Expect(err).NotTo(HaveOccurred())

		By("validate the created resource object gets reconciled in controller")
		controllerContainerLogs := func() string {
			// Check container log to validate that the created resource object gets reconciled in controller
			logOptions := []string{"logs", controllerPodName, "-n", c.namespace}
			logOutput, err := kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(logOptions))
			Expect(err).NotTo(HaveOccurred())

			return logOutput
		}
		Eventually(controllerContainerLogs, 1*time.Minute, 500*time.Millisecond).Should(ContainSubstring(fmt.Sprintf("to reconcile %s-example", strings.ToLower(c.kind))))

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
		Eventually(controllerContainerLogs, 1*time.Minute, 500*time.Millisecond).Should(ContainSubstring("to reconcile deployment-example"))
	})
})
