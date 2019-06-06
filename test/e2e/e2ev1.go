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
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("kubebuilder", func() {
	Context("with v1 scaffolding", func() {
		imageName := "controller:v0.0.1"
		var testSuffix string
		var c *config
		var kbTest *kubebuilderTest

		BeforeEach(func() {
			var err error
			testSuffix, err = randomSuffix()
			Expect(err).NotTo(HaveOccurred())
			c, err = configWithSuffix(testSuffix)
			Expect(err).NotTo(HaveOccurred())
			kbTest = &kubebuilderTest{
				Dir: c.workDir,
				Env: []string{"GO111MODULE=off"},
			}
			prepare(c.workDir)
		})

		AfterEach(func() {
			By("clean up created API objects during test process")
			resources, err := kbTest.RunKustomizeCommand("build", filepath.Join("config", "default"))
			if err != nil {
				fmt.Fprintf(GinkgoWriter, "error when running kustomize build during cleaning up: %v\n", err)
			}
			if _, err = kbTest.RunKubectlCommandWithInput(resources, "delete", "--recursive", "-f", "-"); err != nil {
				fmt.Fprintf(GinkgoWriter, "error when running kubectl delete during cleaning up: %v\n", err)
			}
			if _, err = kbTest.RunKubectlCommand(
				"delete", "--recursive",
				"-f", filepath.Join("config", "crds"),
			); err != nil {
				fmt.Fprintf(GinkgoWriter, "error when running kubectl delete during cleaning up crd: %v\n", err)
			}

			By("remove container image created during test")
			kbTest.CleanupImage(c.controllerImageName)

			By("remove test work dir")
			os.RemoveAll(c.workDir)
		})

		It("should generate a runnable project", func() {
			// prepare v1 vendor
			By("untar the vendor tarball")
			cmd := exec.Command("tar", "-zxf", "../../../testdata/vendor.v1.tgz")
			cmd.Dir = c.workDir
			err := cmd.Run()
			Expect(err).Should(Succeed())

			var controllerPodName string

			By("init v1 project")
			err = kbTest.Init(
				"--project-version", "1",
				"--domain", c.domain,
				"--dep=false")
			Expect(err).Should(Succeed())

			By("creating api definition")
			err = kbTest.CreateAPI(
				"--group", c.group,
				"--version", c.version,
				"--kind", c.kind,
				"--namespaced",
				"--resource",
				"--controller",
				"--make=false")
			Expect(err).Should(Succeed())

			By("creating core-type resource controller")
			err = kbTest.CreateAPI(
				"--group", "apps",
				"--version", "v1",
				"--kind", "Deployment",
				"--namespaced",
				"--resource=false",
				"--controller",
				"--make=false")
			Expect(err).Should(Succeed())

			By("building image")
			err = kbTest.Make("docker-build", "IMG="+imageName)
			Expect(err).Should(Succeed())

			By("loading docker image into kind cluster")
			err = kbTest.LoadImageToKindCluster(imageName)
			Expect(err).Should(Succeed())

			// NOTE: If you want to run the test against a GKE cluster, you will need to grant yourself permission.
			// Otherwise, you may see "... is forbidden: attempt to grant extra privileges"
			// $ kubectl create clusterrolebinding myname-cluster-admin-binding --clusterrole=cluster-admin --user=myname@mycompany.com
			// https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control
			By("deploying controller manager")
			err = kbTest.Make("deploy")
			Expect(err).Should(Succeed())

			By("validate the controller-manager pod running as expected")
			verifyControllerUp := func() error {
				// Get pod name
				podOutput, err := kbTest.RunKubectlGetPodsInNamespace(
					testSuffix,
					"-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}{{ \"\\n\" }}{{ end }}{{ end }}",
				)
				Expect(err).NotTo(HaveOccurred())
				podNames := getNonEmptyLines(podOutput)
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				Expect(controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				status, err := kbTest.RunKubectlGetPodsInNamespace(
					testSuffix,
					controllerPodName, "-o", "jsonpath={.status.phase}",
				)
				Expect(err).NotTo(HaveOccurred())
				if status != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}

				return nil
			}
			Eventually(verifyControllerUp, 2*time.Minute, time.Second).Should(Succeed())

			By("creating an instance of CR")
			inputFile := filepath.Join("config", "samples", fmt.Sprintf("%s_%s_%s.yaml", c.group, c.version, strings.ToLower(c.kind)))
			_, err = kbTest.RunKubectlCommand("apply", "-f", inputFile)
			Expect(err).NotTo(HaveOccurred())

			By("validate the created resource object gets reconciled in controller")
			controllerContainerLogs := func() string {
				// Check container log to validate that the created resource object gets reconciled in controller
				logOutput, err := kbTest.RunKubectlCommand(
					"logs", controllerPodName,
					"-c", "manager",
					"-n", fmt.Sprintf("e2e-%s-system", testSuffix),
				)
				Expect(err).NotTo(HaveOccurred())

				return logOutput
			}
			Eventually(controllerContainerLogs, 2*time.Minute, time.Second).Should(ContainSubstring("Updating"))
		})
	})
})
