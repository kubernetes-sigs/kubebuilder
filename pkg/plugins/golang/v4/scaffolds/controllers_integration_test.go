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
	sigsyaml "sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
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
			if multiGroup {
				controllerDir = filepath.Join(controllerDir, kbc.Group)
			}

			Expect(kbc.Init("--plugins", "go/v4", "--domain", kbc.Domain)).To(Succeed())
			if multiGroup {
				Expect(kbc.Edit("--multigroup=true")).To(Succeed())
			}
			Expect(kbc.CreateAPI("--group", kbc.Group, "--version", kbc.Version, "--kind", kbc.Kind,
				"--resource", "--controller", "--make=false")).To(Succeed())

			By("rewriting the controller in the legacy form")
			recordLegacyController(kbc.Dir, gvk, kind)
			projectPath := filepath.Join(kbc.Dir, yaml.DefaultPath)

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
			before := projectFiles(kbc.Dir)
			err = kbc.CreateAPI("--group", kbc.Group, "--version", kbc.Version, "--kind", "Sailor",
				"--resource", "--controller", "--controller-name", kind, "--make=false")
			Expect(err).To(MatchError(ContainSubstring("conflicts with controller")))
			Expect(projectFiles(kbc.Dir)).To(Equal(before), "a refused create api must change no file")

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

// projectFiles returns the content of every file in the project, by path. The bin directory holds
// downloaded tools, so it is skipped.
func projectFiles(dir string) map[string]string {
	GinkgoHelper()

	files := map[string]string{}
	Expect(filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		content, readErr := os.ReadFile(path)
		Expect(readErr).NotTo(HaveOccurred())
		files[path] = string(content)

		return nil
	})).To(Succeed())

	return files
}

// recordLegacyController rewrites the PROJECT file under dir as an older release wrote it: the
// controller name of the resource gvk is recorded with `controller: true` instead of by name.
func recordLegacyController(dir string, gvk resource.GVK, name string) {
	GinkgoHelper()

	projectPath := filepath.Join(dir, yaml.DefaultPath)
	content, err := os.ReadFile(projectPath)
	Expect(err).NotTo(HaveOccurred())
	var cfg cfgv3.Cfg
	Expect(cfg.UnmarshalYAML(content)).To(Succeed())

	found := false
	for i := range cfg.Resources {
		res := &cfg.Resources[i]
		if !res.GVK.IsEqualTo(gvk) {
			continue
		}
		found = true
		Expect(res.Controllers.HasController(name)).To(BeTrue(), "the controller must be recorded by name")
		kept := resource.Controllers{}
		for _, controller := range *res.Controllers {
			if controller.Name != name {
				kept = append(kept, controller)
			}
		}
		res.Controller = true //nolint:staticcheck // writing the legacy form is the point
		res.Controllers = nil
		if len(kept) > 0 {
			res.Controllers = &kept
		}
	}
	Expect(found).To(BeTrue(), "the PROJECT file must record the resource")

	// Cfg.MarshalYAML migrates the controller back to its name, so the file is written as it is.
	legacy, err := sigsyaml.Marshal(cfg)
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(projectPath, legacy, 0o600)).To(Succeed())
}
