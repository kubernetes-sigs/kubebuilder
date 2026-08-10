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

package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	_ "sigs.k8s.io/kubebuilder/v4/pkg/config/v3" // registers project version 3 so the store can decode PROJECT
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Integration test for `alpha generate`. The command re-scaffolds a project on
// disk from its PROJECT file, so it needs the kubebuilder binary in PATH (the
// test-integration target installs it) but no cluster.
var _ = Describe("alpha generate", func() {
	var (
		kbc         *utils.TestContext
		snapshotDir string
	)

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())

		snapshotDir, err = os.MkdirTemp("", "kb-integration-alpha-generate")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(snapshotDir)).To(Succeed())
		By("removing working dir")
		kbc.Destroy()
	})

	It("should re-scaffold a project with an unsupported layout in place, preserving its configuration", func() {
		By("validating the help output")
		//nolint:gosec
		output, err := kbc.Run(exec.Command(kbc.BinaryName, "alpha", "generate", "--help"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(output)).To(ContainSubstring("kubebuilder alpha generate [flags]"))

		By("scaffolding a project with one API and a webhook")
		err = kbc.Init(
			"--plugins", "go/v4",
			"--project-version", "3",
			"--domain", kbc.Domain,
		)
		Expect(err).NotTo(HaveOccurred(), "Failed to initialize project")
		err = kbc.CreateAPI(
			"--group", kbc.Group,
			"--version", kbc.Version,
			"--kind", kbc.Kind,
			"--controller=true",
			"--resource=true",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred(), "Failed to create API")
		err = kbc.CreateWebhook(
			"--group", kbc.Group,
			"--version", kbc.Version,
			"--kind", kbc.Kind,
			"--defaulting",
			"--programmatic-validation",
		)
		Expect(err).NotTo(HaveOccurred(), "Failed to create webhook")

		// The snapshot is taken before the layout is rewritten below, so the
		// comparison at the end checks against the configuration the project
		// really had, layout included.
		By("keeping the PROJECT file aside for the comparison after the run")
		projectFile := filepath.Join(kbc.Dir, "PROJECT")
		originalProject, err := os.ReadFile(projectFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(filepath.Join(snapshotDir, "PROJECT"), originalProject, 0o644)).To(Succeed())

		// The command exists to bring forward projects that the installed CLI
		// can no longer scaffold, so the test must exercise reading a layout
		// it no longer supports.
		By("marking the project with a layout that is no longer supported")
		Expect(util.ReplaceInFile(projectFile, "go.kubebuilder.io/v4", "go.kubebuilder.io/v3")).To(Succeed())

		By("running alpha generate in place")
		Expect(kbc.Regenerate()).To(Succeed())

		// Both PROJECT files are decoded through the config store, so
		// formatting and key order cannot mask a real change.
		By("checking the PROJECT configuration was preserved")
		originalCfg, err := loadProjectConfig(snapshotDir)
		Expect(err).NotTo(HaveOccurred())
		regeneratedCfg, err := loadProjectConfig(kbc.Dir)
		Expect(err).NotTo(HaveOccurred())
		diffs, err := compareProjectConfigs(originalCfg, regeneratedCfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(diffs).To(BeEmpty(), "alpha generate did not preserve the project configuration:\n  %s",
			strings.Join(diffs, "\n  "))

		By("checking the layout was upgraded")
		regeneratedProject, err := os.ReadFile(projectFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(regeneratedProject)).To(ContainSubstring("go.kubebuilder.io/v4"))
		Expect(string(regeneratedProject)).NotTo(ContainSubstring("go.kubebuilder.io/v3"))

		// A command that exits 0 without writing the scaffold would otherwise
		// pass.
		By("checking the scaffold was written")
		for _, path := range []string{
			"Makefile",
			"Dockerfile",
			filepath.Join("cmd", "main.go"),
			filepath.Join("api", kbc.Version),
			filepath.Join("internal", "controller"),
			filepath.Join("internal", "webhook", kbc.Version),
			filepath.Join("config", "webhook"),
			filepath.Join("config", "default"),
		} {
			_, err := os.Stat(filepath.Join(kbc.Dir, path))
			Expect(err).NotTo(HaveOccurred(), "missing from the regenerated project: %s", path)
		}
	})
})

func loadProjectConfig(dir string) (config.Config, error) {
	store := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := store.LoadFrom(filepath.Join(dir, yaml.DefaultPath)); err != nil {
		return nil, fmt.Errorf("failed to load the PROJECT file under %q: %w", dir, err)
	}

	return store.Config(), nil
}

// compareProjectConfigs checks that `alpha generate` rebuilt the project from
// the same configuration it started with. Resources are compared as a set: the
// scaffold replays them in the order the config lists them, which need not
// match how the original project was built.
func compareProjectConfigs(original, regenerated config.Config) ([]string, error) {
	var diffs []string

	scalar := func(name, want, got string) {
		if want != got {
			diffs = append(diffs, fmt.Sprintf("%s: expected %q, got %q", name, want, got))
		}
	}

	scalar("domain", original.GetDomain(), regenerated.GetDomain())
	scalar("repository", original.GetRepository(), regenerated.GetRepository())
	scalar("projectName", original.GetProjectName(), regenerated.GetProjectName())
	scalar("version", original.GetVersion().String(), regenerated.GetVersion().String())
	scalar("layout", strings.Join(original.GetPluginChain(), ","),
		strings.Join(regenerated.GetPluginChain(), ","))

	if original.IsMultiGroup() != regenerated.IsMultiGroup() {
		diffs = append(diffs, fmt.Sprintf("multigroup: expected %t, got %t",
			original.IsMultiGroup(), regenerated.IsMultiGroup()))
	}

	originalResources, err := original.GetResources()
	if err != nil {
		return nil, fmt.Errorf("read resources from the original PROJECT file: %w", err)
	}

	regeneratedResources, err := regenerated.GetResources()
	if err != nil {
		return nil, fmt.Errorf("read resources from the regenerated PROJECT file: %w", err)
	}

	return append(diffs, compareResourceSets(originalResources, regeneratedResources)...), nil
}

func compareResourceSets(original, regenerated []resource.Resource) []string {
	want := resourceKeys(original)
	got := resourceKeys(regenerated)

	var diffs []string
	for key := range want {
		if _, ok := got[key]; !ok {
			diffs = append(diffs, "resource missing after regeneration: "+key)
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			diffs = append(diffs, "unexpected resource after regeneration: "+key)
		}
	}
	slices.Sort(diffs)

	return diffs
}

// resourceKeys renders each resource as a single comparable string so resources
// can be matched regardless of the order they appear in the config.
func resourceKeys(resources []resource.Resource) map[string]struct{} {
	keys := make(map[string]struct{}, len(resources))
	for _, res := range resources {
		key := fmt.Sprintf("%s/%s/%s controller=%t webhooks=%s",
			res.Group, res.Version, res.Kind, res.Controller, enabledWebhooks(res))
		keys[key] = struct{}{}
	}

	return keys
}

func enabledWebhooks(res resource.Resource) string {
	if res.Webhooks == nil || res.Webhooks.IsEmpty() {
		return "none"
	}

	var enabled []string
	if res.Webhooks.Defaulting {
		enabled = append(enabled, "defaulting")
	}
	if res.Webhooks.Validation {
		enabled = append(enabled, "validation")
	}
	if res.Webhooks.Conversion {
		enabled = append(enabled, "conversion")
	}
	if len(enabled) == 0 {
		return "none"
	}
	slices.Sort(enabled)

	return strings.Join(enabled, "+")
}
