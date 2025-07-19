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

package v1alpha_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	v3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/grafana/v1alpha"
)

var _ = Describe("Grafana Plugin", func() {
	var (
		grafanaPlugin *v1alpha.Plugin
		mockFS        machinery.Filesystem
	)

	BeforeEach(func() {
		grafanaPlugin = &v1alpha.Plugin{}
		mockFS = machinery.Filesystem{FS: afero.NewMemMapFs()}

		cfg := v3.New()
		Expect(grafanaPlugin.GetInitSubcommand().(plugin.RequiresConfig).InjectConfig(cfg)).ToNot(HaveOccurred())
		Expect(grafanaPlugin.GetEditSubcommand().(plugin.RequiresConfig).InjectConfig(cfg)).ToNot(HaveOccurred())
	})

	It("should return the correct plugin name", func() {
		Expect(grafanaPlugin.Name()).To(Equal("grafana.kubebuilder.io"))
	})

	It("should return the correct plugin version number and stage", func() {
		ver := grafanaPlugin.Version()
		Expect(ver.Number).To(Equal(1))
		Expect(ver.Stage.String()).To(Equal("alpha"))
	})

	It("should support project version 3", func() {
		versions := grafanaPlugin.SupportedProjectVersions()
		Expect(versions).ToNot(BeEmpty())
		Expect(versions[0].String()).To(Equal("3"))
	})

	It("should scaffold grafana init successfully", func() {
		plg := grafanaPlugin.GetInitSubcommand()
		err := plg.Scaffold(mockFS)
		Expect(err).NotTo(HaveOccurred())

		directory, err := mockFS.FS.Open("grafana")
		Expect(err).ToNot(HaveOccurred())
		Expect(directory).ToNot(BeNil())
		Expect(directory.Readdir(-1)).To(HaveLen(3))

		fileInfoResourceMetrics, err := mockFS.FS.Stat("grafana/controller-resources-metrics.json")
		Expect(err).ToNot(HaveOccurred())
		Expect(fileInfoResourceMetrics.IsDir()).To(BeFalse())
		Expect(fileInfoResourceMetrics.Name()).To(Equal("controller-resources-metrics.json"))
		Expect(fileInfoResourceMetrics.Size()).To(BeNumerically(">", 0))

		fileInfoRuntimeMetrics, err := mockFS.FS.Stat("grafana/controller-runtime-metrics.json")
		Expect(err).ToNot(HaveOccurred())
		Expect(fileInfoRuntimeMetrics.IsDir()).To(BeFalse())
		Expect(fileInfoRuntimeMetrics.Name()).To(Equal("controller-runtime-metrics.json"))
		Expect(fileInfoRuntimeMetrics.Size()).To(BeNumerically(">", 0))

		fileInfoConfig, err := mockFS.FS.Stat("grafana/custom-metrics/config.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(fileInfoConfig.IsDir()).To(BeFalse())
		Expect(fileInfoConfig.Name()).To(Equal("config.yaml"))
		Expect(fileInfoConfig.Size()).To(BeNumerically(">", 0))
	})

	It("should scaffold grafana edit successfully", func() {
		err := grafanaPlugin.GetEditSubcommand().Scaffold(mockFS)
		Expect(err).NotTo(HaveOccurred())

		directory, err := mockFS.FS.Open("grafana")
		Expect(err).ToNot(HaveOccurred())
		Expect(directory).ToNot(BeNil())
		Expect(directory.Readdir(-1)).To(HaveLen(3))

		fileInfoResourceMetrics, err := mockFS.FS.Stat("grafana/controller-resources-metrics.json")
		Expect(err).ToNot(HaveOccurred())
		Expect(fileInfoResourceMetrics.IsDir()).To(BeFalse())
		Expect(fileInfoResourceMetrics.Name()).To(Equal("controller-resources-metrics.json"))
		Expect(fileInfoResourceMetrics.Size()).To(BeNumerically(">", 0))

		fileInfoRuntimeMetrics, err := mockFS.FS.Stat("grafana/controller-runtime-metrics.json")
		Expect(err).ToNot(HaveOccurred())
		Expect(fileInfoRuntimeMetrics.IsDir()).To(BeFalse())
		Expect(fileInfoRuntimeMetrics.Name()).To(Equal("controller-runtime-metrics.json"))
		Expect(fileInfoRuntimeMetrics.Size()).To(BeNumerically(">", 0))

		fileInfoConfig, err := mockFS.FS.Stat("grafana/custom-metrics/config.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(fileInfoConfig.IsDir()).To(BeFalse())
		Expect(fileInfoConfig.Name()).To(Equal("config.yaml"))
		Expect(fileInfoConfig.Size()).To(BeNumerically(">", 0))
	})
})
