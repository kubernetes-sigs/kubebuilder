//go:build integration

/*
Copyright 2026 The Kubernetes Authors.

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

package scaffolds

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("Server-Side Apply (--ssa) Scaffolding", func() {
	var kbc *utils.TestContext

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())

		By("initializing a project")
		Expect(kbc.Init(
			"--domain", "test.io",
			"--repo", "test.io/ssatest",
		)).To(Succeed())
	})

	AfterEach(func() {
		By("removing working directory")
		kbc.Destroy()
	})

	It("should scaffold a webhook for an SSA resource and run 'make manifests' successfully", func() {
		By("creating an API with Server-Side Apply enabled")
		Expect(kbc.CreateAPI(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Captain",
			"--resource", "--controller",
			"--ssa",
			"--make=false",
		)).To(Succeed())

		By("verifying the SSA applyconfiguration generator was wired into the Makefile")
		makefile, err := os.ReadFile(filepath.Join(kbc.Dir, "Makefile"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(makefile)).To(ContainSubstring("applyconfiguration:headerFile"))

		By("creating defaulting and validation webhooks for the SSA resource")
		Expect(kbc.CreateWebhook(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Captain",
			"--defaulting",
			"--programmatic-validation",
			"--make=false",
		)).To(Succeed())

		By("running 'make manifests'")
		Expect(kbc.Make("manifests")).To(Succeed())

		By("verifying webhook manifests were generated for the SSA resource")
		webhookManifest, err := os.ReadFile(filepath.Join(kbc.Dir, "config", "webhook", "manifests.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(webhookManifest)).To(ContainSubstring("captains"))

		By("verifying applyconfiguration code was generated for the SSA resource")
		acFile := filepath.Join(kbc.Dir, "api", "v1", "applyconfiguration", "api", "v1", "captain.go")
		_, err = os.Stat(acFile)
		Expect(err).NotTo(HaveOccurred(), "applyconfiguration should be generated for the SSA resource")
	})

	It("should support splitting the SSA API and its controller across two commands", func() {
		typesFile := filepath.Join(kbc.Dir, "api", "v1", "admiral_types.go")
		controllerFile := filepath.Join(kbc.Dir, "internal", "controller", "admiral_controller.go")

		By("creating the SSA API resource WITHOUT a controller")
		Expect(kbc.CreateAPI(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Admiral",
			"--resource=true",
			"--controller=false",
			"--ssa",
			"--make=false",
		)).To(Succeed())

		By("verifying the SSA marker was scaffolded on the resource")
		typesContent, err := os.ReadFile(typesFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(typesContent)).To(ContainSubstring("+genclient"))

		By("verifying no controller was scaffolded yet")
		_, err = os.Stat(controllerFile)
		Expect(os.IsNotExist(err)).To(BeTrue(), "controller should not exist before it is requested")

		By("adding the controller for the existing SSA resource WITHOUT recreating the API")
		Expect(kbc.CreateAPI(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Admiral",
			"--resource=false",
			"--controller=true",
			"--make=false",
		)).To(Succeed())

		By("verifying the controller was scaffolded and wired to the SSA resource")
		controllerContent, err := os.ReadFile(controllerFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(controllerContent)).To(ContainSubstring("type AdmiralReconciler struct"))
		Expect(string(controllerContent)).To(ContainSubstring("crewv1.Admiral"))

		By("verifying the controller is registered with the manager in main.go")
		mainContent, err := os.ReadFile(filepath.Join(kbc.Dir, "cmd", "main.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainContent)).To(ContainSubstring("AdmiralReconciler{"))

		By("verifying the SSA marker on the resource was preserved after adding the controller")
		typesContent, err = os.ReadFile(typesFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(typesContent)).To(ContainSubstring("+genclient"))

		By("verifying the PROJECT file tracks both ssa and the controller for the resource")
		projectContent, err := os.ReadFile(filepath.Join(kbc.Dir, "PROJECT"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectContent)).To(ContainSubstring("ssa: true"),
			"adding the controller must not drop the ssa flag from the PROJECT file")
		Expect(string(projectContent)).To(ContainSubstring("controller: true"))
	})

	It("should keep the SSA scaffold when the API is recreated with --force and without --ssa", func() {
		typesFile := filepath.Join(kbc.Dir, "api", "v1", "captain_types.go")

		By("creating an API with Server-Side Apply enabled")
		Expect(kbc.CreateAPI(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Captain",
			"--resource", "--controller",
			"--ssa",
			"--make=false",
		)).To(Succeed())

		By("recreating the same API with --force and without --ssa")
		Expect(kbc.CreateAPI(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Captain",
			"--resource", "--controller",
			"--force",
			"--make=false",
		)).To(Succeed())

		By("verifying the SSA markers were kept in the regenerated types file")
		typesContent, err := os.ReadFile(typesFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(typesContent)).To(ContainSubstring("+genclient"),
			"the regenerated API must keep the SSA markers tracked in the PROJECT file")
		Expect(string(typesContent)).To(ContainSubstring("+kubebuilder:resource:path=captains"))
		Expect(string(typesContent)).NotTo(ContainSubstring("+kubebuilder:ac:generate=false"))

		By("verifying the PROJECT file still tracks ssa")
		projectContent, err := os.ReadFile(filepath.Join(kbc.Dir, "PROJECT"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectContent)).To(ContainSubstring("ssa: true"))

		By("verifying 'make manifests' still generates the applyconfiguration code")
		Expect(kbc.Make("manifests")).To(Succeed())
		_, err = os.Stat(filepath.Join(kbc.Dir, "api", "v1", "applyconfiguration", "api", "v1", "captain.go"))
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("should keep cluster-scoped SSA markers when the API is recreated with --force",
		func(includeSSA bool) {
			typesFile := filepath.Join(kbc.Dir, "api", "v1", "admiral_types.go")

			By("creating a cluster-scoped API with Server-Side Apply enabled")
			Expect(kbc.CreateAPI(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Admiral",
				"--resource", "--controller",
				"--namespaced=false",
				"--ssa",
				"--make=false",
			)).To(Succeed())

			By("recreating the same cluster-scoped API with --force")
			forceOptions := []string{
				"--group", "crew",
				"--version", "v1",
				"--kind", "Admiral",
				"--resource", "--controller",
				"--namespaced=false",
				"--force",
				"--make=false",
			}
			if includeSSA {
				forceOptions = append(forceOptions, "--ssa")
			}
			Expect(kbc.CreateAPI(forceOptions...)).To(Succeed())

			By("verifying the cluster-scoped SSA markers were kept in the regenerated types file")
			typesContent, err := os.ReadFile(typesFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(typesContent)).To(ContainSubstring("// +genclient\n// +genclient:nonNamespaced"),
				"the regenerated cluster-scoped API must keep the SSA client markers")
			Expect(string(typesContent)).To(ContainSubstring(
				"// +kubebuilder:resource:path=admirals,scope=Cluster"))
		},
		Entry("when --ssa is provided again", true),
		Entry("when --ssa is omitted", false),
	)

	It("should complete 'create api --ssa' when groupversion_info.go was customized", func() {
		By("creating a first SSA API to scaffold the group/version package")
		Expect(kbc.CreateAPI(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Captain",
			"--resource", "--controller",
			"--ssa",
			"--make=false",
		)).To(Succeed())

		By("removing the generate markers from groupversion_info.go, as a user customization")
		gvPath := filepath.Join(kbc.Dir, "api", "v1", "groupversion_info.go")
		customized := `// Package v1 contains API Schema definitions for the crew v1 API group.
// +groupName=crew.test.io
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	// This name is used by applyconfiguration generators (e.g. controller-gen).
	SchemeGroupVersion = schema.GroupVersion{Group: "crew.test.io", Version: "v1"}

	// GroupVersion is an alias for SchemeGroupVersion, for backward compatibility.
	GroupVersion = SchemeGroupVersion

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(func(scheme *runtime.Scheme) error {
		metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
		return nil
	})

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
`
		Expect(os.WriteFile(gvPath, []byte(customized), 0o644)).To(Succeed())

		By("creating a second SSA API in the same group/version")
		Expect(kbc.CreateAPI(
			"--group", "crew",
			"--version", "v1",
			"--kind", "FirstMate",
			"--resource", "--controller",
			"--ssa",
			"--make=false",
		)).To(Succeed(), "scaffolding must warn and continue when the marker cannot be injected")

		By("verifying the customized file was left untouched")
		content, err := os.ReadFile(gvPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal(customized))

		By("verifying the new API was fully scaffolded")
		_, err = os.Stat(filepath.Join(kbc.Dir, "api", "v1", "firstmate_types.go"))
		Expect(err).NotTo(HaveOccurred())
		_, err = os.Stat(filepath.Join(kbc.Dir, "internal", "controller", "firstmate_controller.go"))
		Expect(err).NotTo(HaveOccurred())
	})
})
