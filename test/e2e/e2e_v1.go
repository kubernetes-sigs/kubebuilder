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
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kubernetes-sigs/kubebuilder/test/e2e/framework"
	e2einternal "github.com/kubernetes-sigs/kubebuilder/test/internal/e2e"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("v1 main workflow", func() {
	It("should perform main kubebuilder workflow successfully", func() {
		testSuffix := framework.RandomSuffix()
		c := initConfig(testSuffix)
		kubebuilderTest := e2einternal.NewKubebuilderTest(c.workDir, framework.TestContext.BinariesDir)

		prepare(c.workDir)
		defer cleanupv1(kubebuilderTest, c.workDir, c.controllerImageName)

		var controllerPodName string

		output, errrr := exec.Command("which", "kubebuilder").Output()
		fmt.Printf("output of which: %s", output)
		Expect(errrr).NotTo(HaveOccurred())

		By("init v1 project")
		initOptions := []string{
			"--project-version", "v1",
			"--domain", c.domain,
			"--dep", "true",
		}
		err := kubebuilderTest.Init(initOptions)
		Expect(err).NotTo(HaveOccurred())

		By("creating api definition")
		crdOptions := []string{
			"--group", c.group,
			"--version", c.version,
			"--kind", c.kind,
			"--namespaced", "false",
			"--resource", "true",
			"--controller", "true",
		}
		err = kubebuilderTest.CreateAPI(crdOptions)
		Expect(err).NotTo(HaveOccurred())

		// TODO: enable this test after we support gen rbac for core type controller
		By("creating core-type resource controller")
		coreControllerOptions := []string{
			"--group", "apps",
			"--version", "v1",
			"--kind", "Deployment",
			"--namespaced", "true",
			"--resource", "false",
			"--controller", "true",
		}
		err = kubebuilderTest.CreateAPI(coreControllerOptions)
		Expect(err).NotTo(HaveOccurred())

		By("building image")
		makeDockerBuildOptions := []string{"docker-build"}
		err = kubebuilderTest.Make(makeDockerBuildOptions)
		Expect(err).NotTo(HaveOccurred())

		By("pushing image")
		makeDockerPushOptions := []string{"docker-push"}
		err = kubebuilderTest.Make(makeDockerPushOptions)
		Expect(err).NotTo(HaveOccurred())

		// NOTE: If you want to run the test against a GKE cluster, you will need to grant yourself permission.
		// Otherwise, you may see "... is forbidden: attempt to grant extra privileges"
		// $ kubectl create clusterrolebinding myname-cluster-admin-binding --clusterrole=cluster-admin --user=myname@mycompany.com
		// https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control
		By("deploying controller manager")
		makeDeployOptions := []string{"deploy"}
		err = kubebuilderTest.Make(makeDeployOptions)

		By("validate the controller-manager pod running as expected")
		verifyControllerUp := func() error {
			// Get pod name
			getOptions := []string{"get", "pods", "-l", "control-plane=controller-manager", "-n", fmt.Sprintf("e2e-%s-system", testSuffix), "-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}{{ \"\\n\" }}{{ end }}{{ end }}"}
			podOutput, err := kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(getOptions))
			Expect(err).NotTo(HaveOccurred())
			podNames := framework.ParseCmdOutput(podOutput)
			if len(podNames) != 1 {
				return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
			}
			controllerPodName = podNames[0]
			Expect(controllerPodName).Should(ContainSubstring("controller-manager"))

			// Validate pod status
			getOptions = []string{"get", "pods", controllerPodName, "-o", "jsonpath={.status.phase}", "-n", fmt.Sprintf("e2e-%s-system", testSuffix)}
			status, err := kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(getOptions))
			Expect(err).NotTo(HaveOccurred())
			if status != "Running" {
				return fmt.Errorf("controller pod in %s status", status)
			}

			return nil
		}
		Eventually(verifyControllerUp, 5*time.Minute, time.Second).Should(BeNil())

		By("creating an instance of CR")
		inputFile := filepath.Join("config", "samples", fmt.Sprintf("%s_%s_%s.yaml", c.group, c.version, strings.ToLower(c.kind)))
		createOptions := []string{"apply", "-f", inputFile}
		_, err = kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(createOptions))
		Expect(err).NotTo(HaveOccurred())

		By("validate the created resource object gets reconciled in controller")
		controllerContainerLogs := func() string {
			// Check container log to validate that the created resource object gets reconciled in controller
			logOptions := []string{"logs", controllerPodName, "-n", fmt.Sprintf("e2e-%s-system", testSuffix)}
			logOutput, err := kubebuilderTest.RunKubectlCommand(framework.GetKubectlArgs(logOptions))
			Expect(err).NotTo(HaveOccurred())

			return logOutput
		}
		Eventually(controllerContainerLogs, 3*time.Minute, time.Second).Should(ContainSubstring("Updating"))
		Eventually(controllerContainerLogs, 3*time.Minute, time.Second).Should(ContainSubstring("Updating"))
	})
})
