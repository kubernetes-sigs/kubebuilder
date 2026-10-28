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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("a controller recorded in a legacy PROJECT file", func() {
	var kbc *utils.TestContext

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())
	})

	AfterEach(func() {
		By("removing working directory")
		kbc.Destroy()
	})

	DescribeTable("should be migrated on save, refuse a colliding controller and build with a new one",
		func(multiGroup bool) {
			kind := strings.ToLower(kbc.Kind)
			gvk := resource.GVK{Group: kbc.Group, Domain: kbc.Domain, Version: kbc.Version, Kind: kbc.Kind}
			controllerDir := filepath.Join(kbc.Dir, "internal", "controller")
			apiDir := filepath.Join(kbc.Dir, "api", kbc.Version)
			if multiGroup {
				controllerDir = filepath.Join(controllerDir, kbc.Group)
				apiDir = filepath.Join(kbc.Dir, "api", kbc.Group, kbc.Version)
			}

			Expect(kbc.Init("--plugins", "go/v4", "--domain", kbc.Domain)).To(Succeed())
			if multiGroup {
				Expect(kbc.Edit("--multigroup=true")).To(Succeed())
			}
			Expect(kbc.CreateAPI("--group", kbc.Group, "--version", kbc.Version, "--kind", kbc.Kind,
				"--resource", "--controller", "--make=false")).To(Succeed())

			By("rewriting the controller in the legacy form")
			projectPath := filepath.Join(kbc.Dir, yaml.DefaultPath)
			content, err := os.ReadFile(projectPath)
			Expect(err).NotTo(HaveOccurred())
			named := "  controllers:\n  - name: " + kind + "\n"
			Expect(string(content)).To(ContainSubstring(named))
			Expect(os.WriteFile(projectPath,
				[]byte(strings.Replace(string(content), named, "  controller: true\n", 1)), 0o600)).To(Succeed())

			By("saving the PROJECT file through an unrelated command")
			Expect(kbc.CreateWebhook("--group", kbc.Group, "--version", kbc.Version, "--kind", kbc.Kind,
				"--defaulting", "--make=false")).To(Succeed())
			store := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
			Expect(store.LoadFrom(projectPath)).To(Succeed())
			migrated, err := store.Config().GetResource(gvk)
			Expect(err).NotTo(HaveOccurred())
			Expect(migrated.Controller).To(BeFalse()) //nolint:staticcheck // checking the migrated field
			Expect(migrated.GetControllerNames()).To(Equal([]string{kind}))

			By("refusing a new API whose controller collides with the existing one")
			controllerPath := filepath.Join(controllerDir, kind+"_controller.go")
			controllerBefore, err := os.ReadFile(controllerPath)
			Expect(err).NotTo(HaveOccurred())
			err = kbc.CreateAPI("--group", kbc.Group, "--version", kbc.Version, "--kind", "Sailor",
				"--resource", "--controller", "--controller-name", kind, "--make=false")
			Expect(err).To(MatchError(ContainSubstring("conflicts with controller")))
			Expect(os.ReadFile(controllerPath)).To(Equal(controllerBefore))
			Expect(filepath.Join(apiDir, "sailor_types.go")).NotTo(BeAnExistingFile())
			samples, err := filepath.Glob(filepath.Join(kbc.Dir, "config", "samples", "*sailor*"))
			Expect(err).NotTo(HaveOccurred())
			Expect(samples).To(BeEmpty())

			By("adding a controller with a free name and building the project")
			Expect(kbc.CreateAPI("--group", kbc.Group, "--version", kbc.Version, "--kind", kbc.Kind,
				"--resource=false", "--controller", "--controller-name", kind+"-backup", "--make=false")).To(Succeed())
			Expect(filepath.Join(controllerDir, kind+"_backup_controller.go")).To(BeAnExistingFile())
			Expect(kbc.Make("build")).To(Succeed())
		},
		Entry("in a single-group project", false),
		Entry("in a multigroup project", true),
	)
})
