/*
Copyright 2025 The Kubernetes Authors.

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

package v2alpha

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("deleteSubcommand", func() {
	var (
		sub *deleteSubcommand
		fs  machinery.Filesystem
	)

	BeforeEach(func() {
		sub = &deleteSubcommand{}
		fs = machinery.Filesystem{FS: afero.NewMemMapFs()}
		cfg := cfgv3.New()
		Expect(sub.InjectConfig(cfg)).To(Succeed())
	})

	It("should succeed when no Helm files exist (best-effort)", func() {
		// Missing files produce warnings, not errors
		Expect(sub.Scaffold(fs)).To(Succeed())
	})

	It("should delete the chart directory and test-chart workflow", func() {
		chartDir := filepath.Join(DefaultOutputDir, "chart")
		chartFile := filepath.Join(chartDir, "Chart.yaml")
		testChart := filepath.Join(".github", "workflows", "test-chart.yml")

		Expect(fs.FS.MkdirAll(chartDir, 0o755)).To(Succeed())
		Expect(afero.WriteFile(fs.FS, chartFile, []byte("apiVersion: v2"), 0o644)).To(Succeed())
		Expect(fs.FS.MkdirAll(filepath.Dir(testChart), 0o755)).To(Succeed())
		Expect(afero.WriteFile(fs.FS, testChart, []byte("on: push"), 0o644)).To(Succeed())

		Expect(sub.Scaffold(fs)).To(Succeed())

		exists, err := afero.DirExists(fs.FS, chartDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeFalse(), "chart directory should be deleted")

		exists, err = afero.Exists(fs.FS, testChart)
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeFalse(), "test-chart workflow should be deleted")
	})

	It("should use the stored output-dir from plugin config", func() {
		cfg := cfgv3.New()
		customDir := "custom-charts"
		Expect(cfg.EncodePluginConfig("helm.kubebuilder.io/v2-alpha", pluginConfig{OutputDir: customDir})).To(Succeed())
		Expect(sub.InjectConfig(cfg)).To(Succeed())

		chartDir := filepath.Join(customDir, "chart")
		chartFile := filepath.Join(chartDir, "Chart.yaml")
		Expect(fs.FS.MkdirAll(chartDir, 0o755)).To(Succeed())
		Expect(afero.WriteFile(fs.FS, chartFile, []byte("apiVersion: v2"), 0o644)).To(Succeed())

		Expect(sub.Scaffold(fs)).To(Succeed())

		exists, err := afero.DirExists(fs.FS, chartDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeFalse(), "custom chart directory should be deleted")
	})
})
