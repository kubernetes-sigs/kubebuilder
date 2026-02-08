/*
Copyright 2022 The Kubernetes Authors.

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

package all

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Test specs for deploy-image plugin
var _ = Describe("kubebuilder", func() {
	Context("deploy image plugin", func() {
		var kbc *utils.TestContext

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			By("clean up API objects created during the test")
			Expect(kbc.Make("undeploy")).To(Succeed())

			By("removing controller image and working dir")
			kbc.Destroy()
		})

		It("should generate a runnable project with deploy-image/v1-alpha options ", func() {
			generateDeployImageWithOptions(kbc)
			runDeployImageTests(kbc)
		})

		It("should generate a runnable project with deploy-image/v1-alpha without options ", func() {
			generateDeployImage(kbc)
			runDeployImageTests(kbc)
		})
	})
})

// generateDeployImageWithOptions implements a go/v4 plugin project and scaffold an API using the image options
func generateDeployImageWithOptions(kbc *utils.TestContext) {
	initDeployImageProject(kbc)

	By("creating API definition with deploy-image/v1-alpha plugin with options")
	err := kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--plugins", "deploy-image/v1-alpha",
		"--image", "memcached:1.6.26-alpine3.19",
		"--image-container-port", "11211",
		"--image-container-command", "memcached,--memory-limit=64,-o,modern,-v",
		"--run-as-user", "1001",
		"--make=false",
		"--manifests=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to create API definition with deploy-image/v1-alpha")
}

// generateDeployImage implements a go/v4 plugin project and scaffold an API using the deploy image plugin
func generateDeployImage(kbc *utils.TestContext) {
	initDeployImageProject(kbc)

	By("creating API definition with deploy-image/v1-alpha plugin")
	err := kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--plugins", "deploy-image/v1-alpha",
		"--image", "busybox:1.36.1",
		"--run-as-user", "1001",
		"--make=false",
		"--manifests=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to create API definition")
}

func initDeployImageProject(kbc *utils.TestContext) {
	By("initializing a project")
	err := kbc.Init(
		"--plugins", "go/v4",
		"--project-version", "3",
		"--domain", kbc.Domain,
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to initialize project")
}

// runDeployImageTests runs a set of e2e tests for a scaffolded deploy-image project.
func runDeployImageTests(kbc *utils.TestContext) {
	var controllerPodName string
	var err error

	SetDefaultEventuallyPollingInterval(time.Second)
	SetDefaultEventuallyTimeout(time.Minute)

	By("updating the go.mod")
	Expect(kbc.Tidy()).To(Succeed())

	By("run make manifests")
	Expect(kbc.Make("manifests")).To(Succeed())

	By("run make generate")
	Expect(kbc.Make("generate")).To(Succeed())

	By("run make all")
	Expect(kbc.Make("all")).To(Succeed())

	By("run make install")
	Expect(kbc.Make("install")).To(Succeed())

	By("building the controller image")
	Expect(kbc.Make("docker-build", "IMG="+kbc.ImageName)).To(Succeed())

	By("loading the controller docker image into the kind cluster")
	Expect(kbc.LoadImageToKindCluster()).To(Succeed())

	By("deploying the controller-manager")
	cmd := exec.Command("make", "deploy", "IMG="+kbc.ImageName)
	out, err := kbc.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(out)).NotTo(ContainSubstring("Warning: would violate PodSecurity"))

	By("validating that the controller-manager pod is running as expected")
	verifyControllerUp := func(g Gomega) {
		// Get pod name
		var podOutput string
		podOutput, err = kbc.Kubectl.Get(
			true,
			"pods", "-l", "control-plane=controller-manager",
			"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
				"{{ \"\\n\" }}{{ end }}{{ end }}")
		g.Expect(err).NotTo(HaveOccurred())
		podNames := util.GetNonEmptyLines(podOutput)
		g.Expect(podNames).To(HaveLen(1), "wrong number of controller-manager pods")
		controllerPodName = podNames[0]
		g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

		// Validate pod status
		g.Expect(kbc.Kubectl.Get(true, "pods", controllerPodName, "-o", "jsonpath={.status.phase}")).
			To(Equal("Running"), "incorrect controller pod status")
	}
	defer func() {
		out, errDescribe := kbc.Kubectl.CommandInNamespace("describe", "all")
		Expect(errDescribe).NotTo(HaveOccurred())
		_, _ = fmt.Fprintln(GinkgoWriter, out)
	}()
	Eventually(verifyControllerUp).Should(Succeed())

	By("creating an instance of the CR")
	sampleFile := filepath.Join("config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", kbc.Group, kbc.Version, strings.ToLower(kbc.Kind)))

	sampleFilePath, err := filepath.Abs(filepath.Join(fmt.Sprintf("e2e-%s", kbc.TestSuffix), sampleFile))
	Expect(err).To(Not(HaveOccurred()))

	Eventually(func(g Gomega) {
		g.Expect(kbc.Kubectl.Apply(true, "-f", sampleFilePath)).Error().NotTo(HaveOccurred())
	}).Should(Succeed())

	By("validating that pod(s) status.phase=Running")
	verifyMemcachedPodStatus := func(g Gomega) {
		g.Expect(kbc.Kubectl.Get(true, "pods", "-l",
			fmt.Sprintf("app.kubernetes.io/name=e2e-%s", kbc.TestSuffix),
			"-o", "jsonpath={.items[*].status}",
		)).To(ContainSubstring("\"phase\":\"Running\""))
	}
	Eventually(verifyMemcachedPodStatus).Should(Succeed())

	By("validating that the status of the custom resource created is updated or not")
	verifyAvailableStatus := func(g Gomega) {
		g.Expect(kbc.Kubectl.Get(true, strings.ToLower(kbc.Kind),
			strings.ToLower(kbc.Kind)+"-sample",
			"-o", "jsonpath={.status.conditions}")).To(ContainSubstring("Available"),
			`status condition with type "Available" should be set`)
	}
	Eventually(verifyAvailableStatus).Should(Succeed())

	By("validating the finalizer")
	Eventually(func(g Gomega) {
		g.Expect(kbc.Kubectl.Delete(true, "-f", sampleFilePath)).Error().NotTo(HaveOccurred())
	}).Should(Succeed())

	Eventually(func(g Gomega) {
		g.Expect(kbc.Kubectl.Get(true, "events", "--field-selector=type=Warning",
			"-o", "jsonpath={.items[*].message}",
		)).To(ContainSubstring("is being deleted from the namespace"))
	}).Should(Succeed())
}
