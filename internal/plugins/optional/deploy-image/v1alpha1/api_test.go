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

package v1alpha1

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

var _ = Describe("createAPISubcommand", func() {
	var (
		subCmd *createAPISubcommand
		cfg    config.Config
		res    *resource.Resource
		fs     machinery.Filesystem
	)

	BeforeEach(func() {
		subCmd = &createAPISubcommand{}
		cfg = cfgv3.New()
		_ = cfg.SetRepository("github.com/example/test")

		subCmd.options = &goPlugin.Options{}
		res = &resource.Resource{
			GVK: resource.GVK{
				Group:   "example.com",
				Domain:  "test.io",
				Version: "v1alpha1",
				Kind:    "Memcached",
			},
			Plural:   "memcacheds",
			API:      &resource.API{},
			Webhooks: &resource.Webhooks{},
		}

		fs = machinery.Filesystem{FS: afero.NewMemMapFs()}
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
	})

	Context("PreScaffold validation", func() {
		It("should require image flag to be set", func() {
			subCmd.image = ""

			err := subCmd.PreScaffold(fs)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("you MUST inform the image"))
		})

		It("should succeed when image is provided", func() {
			subCmd.image = "memcached:1.6.15-alpine"

			tmpDir, err := os.MkdirTemp("", "deploy-image-test")
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.RemoveAll(tmpDir) }()

			originalDir, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.Chdir(originalDir) }()

			err = os.Chdir(tmpDir)
			Expect(err).NotTo(HaveOccurred())

			err = os.MkdirAll("cmd", 0o755)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(filepath.Join("cmd", "main.go"), []byte("package main"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = subCmd.PreScaffold(fs)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should check for cmd/main.go in go/v4 projects", func() {
			subCmd.image = "busybox:1.36.1"
			_ = cfg.SetPluginChain([]string{"go.kubebuilder.io/v4"})

			tmpDir, err := os.MkdirTemp("", "deploy-image-test")
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.RemoveAll(tmpDir) }()

			originalDir, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.Chdir(originalDir) }()

			err = os.Chdir(tmpDir)
			Expect(err).NotTo(HaveOccurred())

			err = subCmd.PreScaffold(fs)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("main.go file should be present in cmd/main.go"))
		})

		It("should check for main.go in go/v3 projects", func() {
			subCmd.image = "busybox:1.36.1"
			_ = cfg.SetPluginChain([]string{"go.kubebuilder.io/v3"})

			tmpDir, err := os.MkdirTemp("", "deploy-image-test")
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.RemoveAll(tmpDir) }()

			originalDir, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.Chdir(originalDir) }()

			err = os.Chdir(tmpDir)
			Expect(err).NotTo(HaveOccurred())

			err = subCmd.PreScaffold(fs)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("main.go file should be present in main.go"))
		})
	})

	Context("InjectResource validation", func() {
		It("should set API and controller flags automatically", func() {
			err := subCmd.InjectResource(res)

			Expect(err).NotTo(HaveOccurred())
			Expect(subCmd.options.DoAPI).To(BeTrue())
			Expect(subCmd.options.DoController).To(BeTrue())
			Expect(subCmd.options.Namespaced).To(BeTrue())
		})

		It("should prevent multiple groups in single-group project", func() {
			firstRes := resource.Resource{
				GVK: resource.GVK{
					Group:   "ship",
					Domain:  "test.io",
					Version: "v1",
					Kind:    "Frigate",
				},
				Plural: "frigates",
				API:    &resource.API{CRDVersion: "v1"},
			}
			Expect(cfg.AddResource(firstRes)).To(Succeed())

			res.Group = "example.com"
			res.Plural = "memcacheds"

			err := subCmd.InjectResource(res)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("multiple groups are not allowed"))
		})

		It("should allow multiple groups when multigroup is enabled", func() {
			Expect(cfg.SetMultiGroup()).To(Succeed())

			firstRes := resource.Resource{
				GVK: resource.GVK{
					Group:   "ship",
					Domain:  "test.io",
					Version: "v1",
					Kind:    "Frigate",
				},
				Plural: "frigates",
				API:    &resource.API{CRDVersion: "v1"},
			}
			Expect(cfg.AddResource(firstRes)).To(Succeed())

			res.Group = "example.com"

			Expect(subCmd.InjectResource(res)).To(Succeed())
		})
	})
})
