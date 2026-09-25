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

package v1alpha

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

const controllerPath = "internal/controller/bar_controller.go"

var _ = Describe("createAPISubcommand", func() {
	var (
		subCmd *createAPISubcommand
		cfg    config.Config
	)

	BeforeEach(func() {
		subCmd = &createAPISubcommand{}
		cfg = cfgv3.New()
		_ = cfg.SetRepository("github.com/example/myop")
		_ = cfg.SetDomain("example.com")
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
	})

	Context("InjectConfig", func() {
		It("should store the config", func() {
			Expect(subCmd.config).To(Equal(cfg))
		})
	})

	Context("InjectResource", func() {
		It("should store the resource", func() {
			res := &resource.Resource{
				GVK: resource.GVK{Group: "foo", Version: "v1", Kind: "Bar"},
			}
			Expect(subCmd.InjectResource(res)).To(Succeed())
			Expect(subCmd.resource).To(Equal(res))
		})
	})

	Context("Scaffold", func() {
		var memFS machinery.Filesystem

		BeforeEach(func() {
			memFS = machinery.Filesystem{FS: afero.NewMemMapFs()}
		})

		It("should be a no-op when resource is nil", func() {
			subCmd.resource = nil
			Expect(subCmd.Scaffold(memFS)).To(Succeed())
		})

		It("should be a no-op when resource has no controller", func() {
			subCmd.resource = &resource.Resource{
				GVK:        resource.GVK{Group: "foo", Version: "v1", Kind: "Bar"},
				Controller: false,
			}
			Expect(subCmd.Scaffold(memFS)).To(Succeed())
		})

		It("should write a controller when resource has a controller", func() {
			subCmd.resource = &resource.Resource{
				GVK:        resource.GVK{Group: "foo", Version: "v1", Kind: "Bar"},
				Plural:     "bars",
				Path:       "github.com/example/myop/api/v1",
				Controller: true,
			}
			Expect(subCmd.Scaffold(memFS)).To(Succeed())

			exists, err := afero.Exists(memFS.FS, controllerPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			content, err := afero.ReadFile(memFS.FS, controllerPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("mcreconcile.Request"))
			Expect(string(content)).To(ContainSubstring("mcbuilder.ControllerManagedBy"))
			Expect(string(content)).To(ContainSubstring("mcmanager.Manager"))
			Expect(string(content)).To(ContainSubstring(`"sigs.k8s.io/multicluster-runtime/pkg/reconcile"`))
		})

		It("should write a controller under a group subdirectory in multi-group mode", func() {
			Expect(cfg.SetMultiGroup()).To(Succeed())
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			subCmd.resource = &resource.Resource{
				GVK:        resource.GVK{Group: "foo", Version: "v1", Kind: "Bar"},
				Plural:     "bars",
				Path:       "github.com/example/myop/api/foo/v1",
				Controller: true,
			}
			Expect(subCmd.Scaffold(memFS)).To(Succeed())

			multiGroupPath := "internal/controller/foo/bar_controller.go"
			exists, err := afero.Exists(memFS.FS, multiGroupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			content, err := afero.ReadFile(memFS.FS, multiGroupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("package foo"))
			Expect(string(content)).To(ContainSubstring("mcreconcile.Request"))
		})

		// go/v4's MainUpdater injects mgr.GetClient() / mgr.GetScheme() at the
		// +kubebuilder:scaffold:builder marker. mcmanager.Manager does not expose
		// those methods; they must go through GetLocalManager(). Scaffold() must
		// rewrite any such occurrences that appear in cmd/main.go.
		It("should rewrite mgr.GetClient() and mgr.GetScheme() in a pre-existing cmd/main.go", func() {
			subCmd.resource = &resource.Resource{
				GVK:        resource.GVK{Group: "foo", Version: "v1", Kind: "Bar"},
				Plural:     "bars",
				Path:       "github.com/example/myop/api/v1",
				Controller: true,
			}

			// Simulate what go/v4's MainUpdater injects at +kubebuilder:scaffold:builder.
			goV4Injection := `package main

func main() {
	if err := (&controller.BarReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder
}
`
			Expect(afero.WriteFile(memFS.FS, "cmd/main.go", []byte(goV4Injection), 0600)).To(Succeed())

			Expect(subCmd.Scaffold(memFS)).To(Succeed())

			mainBytes, err := afero.ReadFile(memFS.FS, "cmd/main.go")
			Expect(err).NotTo(HaveOccurred())
			main := string(mainBytes)

			Expect(main).NotTo(ContainSubstring("mgr.GetClient()"))
			Expect(main).NotTo(ContainSubstring("mgr.GetScheme()"))
			Expect(main).To(ContainSubstring("mgr.GetLocalManager().GetClient()"))
			Expect(main).To(ContainSubstring("mgr.GetLocalManager().GetScheme()"))
		})

		It("should not introduce mgr.GetClient() when cmd/main.go already uses GetLocalManager()", func() {
			subCmd.resource = &resource.Resource{
				GVK:        resource.GVK{Group: "foo", Version: "v1", Kind: "Bar"},
				Plural:     "bars",
				Path:       "github.com/example/myop/api/v1",
				Controller: true,
			}

			// Simulate a cmd/main.go that was already correctly scaffolded by the
			// mc plugin (uses GetLocalManager). The fixup must not corrupt it.
			alreadyCorrect := `package main

func main() {
	if err := (&controller.BarReconciler{
		Client: mgr.GetLocalManager().GetClient(),
		Scheme: mgr.GetLocalManager().GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		os.Exit(1)
	}
	// +kubebuilder:scaffold:multicluster-builder
	// +kubebuilder:scaffold:builder
}
`
			Expect(afero.WriteFile(memFS.FS, "cmd/main.go", []byte(alreadyCorrect), 0600)).To(Succeed())

			Expect(subCmd.Scaffold(memFS)).To(Succeed())

			mainBytes, err := afero.ReadFile(memFS.FS, "cmd/main.go")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(mainBytes)).NotTo(ContainSubstring("mgr.GetClient()"))
			Expect(string(mainBytes)).NotTo(ContainSubstring("mgr.GetScheme()"))
		})
	})
})
