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
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
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
				Group:   exampleDomain,
				Domain:  testIODomain,
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

	Context("BindFlags", func() {
		It("should not expose the --ssa flag since Server-Side Apply is not supported", func() {
			flagSet := pflag.NewFlagSet("create-api", pflag.ContinueOnError)
			subCmd.BindFlags(flagSet)

			Expect(flagSet.Lookup("image")).NotTo(BeNil())
			Expect(flagSet.Lookup("ssa")).To(BeNil())
		})
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
			_ = cfg.SetPluginChain([]string{pluginGoKubebuilder})

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

		It("should reject a resource with Server-Side Apply set by go/v4 earlier in the plugin chain", func() {
			res.API = &resource.API{CRDVersion: "v1", Namespaced: true, SSA: true}

			err := subCmd.InjectResource(res)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"'--ssa' is not supported by the deploy-image.go.kubebuilder.io/v1-alpha plugin"))
			Expect(res.API.SSA).To(BeTrue(), "the rejected resource must be left as it was")
			Expect(subCmd.resource).To(BeNil())
		})

		It("should reject an API that the PROJECT file records with Server-Side Apply", func() {
			existing := *res
			existing.API = &resource.API{CRDVersion: "v1", Namespaced: true, SSA: true}
			Expect(cfg.AddResource(existing)).To(Succeed())

			err := subCmd.InjectResource(res)

			Expect(err).To(MatchError(ContainSubstring("it uses Server-Side Apply, which this plugin does not support")))
			Expect(subCmd.resource).To(BeNil())
			tracked, err := cfg.GetResource(res.GVK)
			Expect(err).NotTo(HaveOccurred())
			Expect(tracked.API.SSA).To(BeTrue(), "the PROJECT file must keep Server-Side Apply")
		})

		It("should prevent multiple groups in single-group project", func() {
			firstRes := resource.Resource{
				GVK: resource.GVK{
					Group:   "ship",
					Domain:  testIODomain,
					Version: "v1",
					Kind:    "Frigate",
				},
				Plural: "frigates",
				API:    &resource.API{CRDVersion: "v1"},
			}
			Expect(cfg.AddResource(firstRes)).To(Succeed())

			res.Group = exampleDomain
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
					Domain:  testIODomain,
					Version: "v1",
					Kind:    "Frigate",
				},
				Plural: "frigates",
				API:    &resource.API{CRDVersion: "v1"},
			}
			Expect(cfg.AddResource(firstRes)).To(Succeed())

			res.Group = exampleDomain

			Expect(subCmd.InjectResource(res)).To(Succeed())
		})

		Context("when the plugin already scaffolded a resource", func() {
			BeforeEach(func() {
				Expect(cfg.EncodePluginConfig(plugin.KeyFor(Plugin{}), PluginConfig{Resources: []ResourceData{{
					Group: res.Group, Domain: res.Domain, Version: res.Version, Kind: res.Kind,
				}}})).To(Succeed())
			})

			It("should refuse the same resource before writing any file", func() {
				before, err := cfg.GetResources()
				Expect(err).NotTo(HaveOccurred())

				err = subCmd.InjectResource(res)

				Expect(err).To(MatchError(ContainSubstring("already scaffolded the API and controller")))
				Expect(cfg.GetResources()).To(Equal(before))
			})

			It("should accept another kind", func() {
				res.Kind = "Busybox"
				res.Plural = "busyboxes"

				Expect(subCmd.InjectResource(res)).To(Succeed())
			})
		})

		Context("when another resource records a controller", func() {
			// userController stands for the controller file the user implemented.
			const userController = "package controller\n\n// implemented by the user\n"

			addCaptain := func(controller string) {
				Expect(cfg.AddResource(resource.Resource{
					GVK:         resource.GVK{Group: exampleDomain, Domain: testIODomain, Version: "v1alpha1", Kind: "Captain"},
					Plural:      "captains",
					API:         &resource.API{CRDVersion: "v1", Namespaced: true},
					Controllers: &resource.Controllers{{Name: controller}},
				})).To(Succeed())
			}

			It("should refuse a controller generating the same reconciler before writing any file", func() {
				controllerFile := filepath.Join("internal", "controller", "memcached_controller.go")
				addCaptain("memcached")
				Expect(afero.WriteFile(fs.FS, controllerFile, []byte(userController), 0o644)).To(Succeed())
				before, err := cfg.GetResources()
				Expect(err).NotTo(HaveOccurred())

				err = subCmd.InjectResource(res)

				Expect(err).To(MatchError(ContainSubstring("both generate MemcachedReconciler")))
				Expect(cfg.GetResources()).To(Equal(before))
				Expect(afero.ReadFile(fs.FS, controllerFile)).To(BeEquivalentTo(userController))
			})

			It("should accept a controller generating another reconciler", func() {
				addCaptain("captain-backup")

				Expect(subCmd.InjectResource(res)).To(Succeed())
			})
		})

		Context("when the same resource records a controller", func() {
			addMemcached := func(controller string) {
				Expect(cfg.AddResource(resource.Resource{
					GVK:         res.GVK,
					Plural:      res.Plural,
					API:         &resource.API{CRDVersion: "v1", Namespaced: true},
					Controllers: &resource.Controllers{{Name: controller}},
				})).To(Succeed())
			}

			It("should refuse a default controller generating the same reconciler before writing any file", func() {
				res.Kind = "MyKind"
				res.Plural = "mykinds"
				addMemcached("my-kind")
				before, err := cfg.GetResources()
				Expect(err).NotTo(HaveOccurred())

				err = subCmd.InjectResource(res)

				Expect(err).To(MatchError(ContainSubstring("both generate MyKindReconciler")))
				Expect(cfg.GetResources()).To(Equal(before))
			})

			It("should accept the recorded default controller again", func() {
				addMemcached(resource.DefaultControllerName(res.Kind))

				Expect(subCmd.InjectResource(res)).To(Succeed())
			})
		})
	})
})
