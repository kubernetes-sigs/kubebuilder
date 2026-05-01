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
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/internal/templates/api"
)

var _ = Describe("Types template", func() {
	scaffoldTypes := func(skipApplyConfig bool) string {
		res := resource.Resource{
			GVK: resource.GVK{
				Group:   "example.com",
				Domain:  "test.io",
				Version: "v1alpha1",
				Kind:    "Memcached",
			},
			Plural: "memcacheds",
			API:    &resource.API{CRDVersion: "v1", Namespaced: true},
		}
		cfg := cfgv3.New()
		Expect(cfg.SetRepository("sigs.k8s.io/kubebuilder/test")).To(Succeed())

		fs := machinery.Filesystem{FS: afero.NewMemMapFs()}
		scaffold := machinery.NewScaffold(fs,
			machinery.WithConfig(cfg),
			machinery.WithBoilerplate("/* boilerplate */"),
			machinery.WithResource(&res),
		)
		Expect(scaffold.Execute(&api.Types{Port: "11211", SkipApplyConfig: skipApplyConfig})).To(Succeed())

		typesPath := filepath.Join("api", res.Version, strings.ToLower(res.Kind)+"_types.go")
		content, err := afero.ReadFile(fs.FS, typesPath)
		Expect(err).NotTo(HaveOccurred())
		return string(content)
	}

	It("should scaffold the opt-out marker when another kind in the package has SSA enabled", func() {
		Expect(scaffoldTypes(true)).To(ContainSubstring("// +kubebuilder:ac:generate=false"))
	})

	It("should scaffold no SSA markers when the project does not use SSA", func() {
		Expect(scaffoldTypes(false)).NotTo(ContainSubstring("+kubebuilder:ac:generate"))
	})
})

var _ = Describe("hasSSAInPackage", func() {
	newResource := func(kind string, ssa bool) resource.Resource {
		return resource.Resource{
			GVK: resource.GVK{
				Group:   "example.com",
				Domain:  "test.io",
				Version: "v1alpha1",
				Kind:    kind,
			},
			Plural: resource.RegularPlural(kind),
			API:    &resource.API{CRDVersion: "v1", Namespaced: true, SSA: ssa},
		}
	}

	newConfig := func(resources ...resource.Resource) config.Config {
		cfg := cfgv3.New()
		Expect(cfg.SetRepository("sigs.k8s.io/kubebuilder/test")).To(Succeed())
		for _, res := range resources {
			Expect(cfg.AddResource(res)).To(Succeed())
		}
		return cfg
	}

	It("should return true when another kind in the same group/version has SSA enabled", func() {
		navigator := newResource("Navigator", true)
		memcached := newResource("Memcached", false)
		s := &apiScaffolder{
			config:   newConfig(navigator, memcached),
			resource: memcached,
		}

		Expect(s.hasSSAInPackage()).To(BeTrue())
	})

	It("should return false when the project does not use SSA", func() {
		busybox := newResource("Busybox", false)
		memcached := newResource("Memcached", false)
		s := &apiScaffolder{
			config:   newConfig(busybox, memcached),
			resource: memcached,
		}

		Expect(s.hasSSAInPackage()).To(BeFalse())
	})
})
