//go:build !integration

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

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/api"
)

func ssaTestResourceGV(kind, group, version string, ssa bool) resource.Resource {
	return resource.Resource{
		GVK: resource.GVK{
			Group:   group,
			Domain:  "test.io",
			Version: version,
			Kind:    kind,
		},
		Plural: resource.RegularPlural(kind),
		API:    &resource.API{CRDVersion: "v1", Namespaced: true, SSA: ssa},
	}
}

func ssaTestResource(kind string, ssa bool) resource.Resource {
	return ssaTestResourceGV(kind, "crew", "v1", ssa)
}

func newSSATestConfig(resources ...resource.Resource) config.Config {
	cfg := cfgv3.New()
	Expect(cfg.SetRepository("sigs.k8s.io/kubebuilder/test")).To(Succeed())
	for _, res := range resources {
		Expect(cfg.AddResource(res)).To(Succeed())
	}
	return cfg
}

var _ = Describe("API scaffolding with Server-Side Apply", func() {
	Describe("hasSSAInPackage", func() {
		It("should return true when another kind in the same group/version has SSA enabled", func() {
			navigator := ssaTestResource("Navigator", true)
			captain := ssaTestResource("Captain", false)
			s := &apiScaffolder{
				config:   newSSATestConfig(navigator, captain),
				resource: captain,
			}

			Expect(s.hasSSAInPackage()).To(BeTrue())
		})

		It("should return false when no other kind has SSA enabled", func() {
			sailor := ssaTestResource("Sailor", false)
			captain := ssaTestResource("Captain", false)
			s := &apiScaffolder{
				config:   newSSATestConfig(sailor, captain),
				resource: captain,
			}

			Expect(s.hasSSAInPackage()).To(BeFalse())
		})

		It("should return false when the SSA kind is in another version", func() {
			navigatorV2 := ssaTestResourceGV("Navigator", "crew", "v2", true)
			captain := ssaTestResource("Captain", false)
			s := &apiScaffolder{
				config:   newSSATestConfig(navigatorV2, captain),
				resource: captain,
			}

			Expect(s.hasSSAInPackage()).To(BeFalse())
		})

		It("should return false when the SSA kind is in another group", func() {
			prawn := ssaTestResourceGV("Prawn", "sea-creatures", "v1", true)
			captain := ssaTestResource("Captain", false)
			s := &apiScaffolder{
				config:   newSSATestConfig(prawn, captain),
				resource: captain,
			}

			Expect(s.hasSSAInPackage()).To(BeFalse())
		})

		It("should not count the resource being scaffolded itself", func() {
			navigator := ssaTestResource("Navigator", true)
			s := &apiScaffolder{
				config:   newSSATestConfig(navigator),
				resource: navigator,
			}

			Expect(s.hasSSAInPackage()).To(BeFalse())
		})
	})

	Describe("isFirstSSAAPI", func() {
		It("should return true when the project has no other API with SSA enabled", func() {
			captain := ssaTestResource("Captain", false)
			navigator := ssaTestResource("Navigator", true)
			s := &apiScaffolder{
				config:   newSSATestConfig(captain, navigator),
				resource: navigator,
			}

			Expect(s.isFirstSSAAPI()).To(BeTrue())
		})

		It("should return false when the project already has an API with SSA enabled", func() {
			navigator := ssaTestResource("Navigator", true)
			prawn := ssaTestResource("Prawn", true)
			s := &apiScaffolder{
				config:   newSSATestConfig(navigator, prawn),
				resource: prawn,
			}

			Expect(s.isFirstSSAAPI()).To(BeFalse())
		})
	})

	Describe("Types template", func() {
		scaffoldTypes := func(res resource.Resource, skipApplyConfig bool) string {
			fs := machinery.Filesystem{FS: afero.NewMemMapFs()}
			scaffold := machinery.NewScaffold(fs,
				machinery.WithConfig(newSSATestConfig()),
				machinery.WithBoilerplate("/* boilerplate */"),
				machinery.WithResource(&res),
			)
			Expect(scaffold.Execute(&api.Types{SkipApplyConfig: skipApplyConfig})).To(Succeed())

			typesPath := filepath.Join("api", res.Version, strings.ToLower(res.Kind)+"_types.go")
			content, err := afero.ReadFile(fs.FS, typesPath)
			Expect(err).NotTo(HaveOccurred())
			return string(content)
		}

		It("should scaffold the genclient and resource markers when SSA is enabled", func() {
			content := scaffoldTypes(ssaTestResource("Navigator", true), false)

			Expect(content).To(ContainSubstring("// +genclient"))
			Expect(content).To(ContainSubstring("// +kubebuilder:resource:path=navigators"))
			Expect(content).NotTo(ContainSubstring("+kubebuilder:ac:generate=false"))
		})

		It("should scaffold the nonNamespaced marker when SSA is enabled for a cluster-scoped kind", func() {
			res := ssaTestResource("Admiral", true)
			res.API.Namespaced = false
			content := scaffoldTypes(res, false)

			Expect(content).To(ContainSubstring("// +genclient\n// +genclient:nonNamespaced"))
		})

		It("should scaffold the opt-out marker when another kind in the package has SSA enabled", func() {
			content := scaffoldTypes(ssaTestResource("Captain", false), true)

			Expect(content).To(ContainSubstring("// +kubebuilder:ac:generate=false"))
			Expect(content).NotTo(ContainSubstring("+genclient"))
		})

		It("should scaffold no SSA markers by default", func() {
			content := scaffoldTypes(ssaTestResource("Captain", false), false)

			Expect(content).NotTo(ContainSubstring("+genclient"))
			Expect(content).NotTo(ContainSubstring("+kubebuilder:ac:generate"))
		})

		It("should scaffold both the scope and opt-out markers for a cluster-scoped kind in a package with SSA", func() {
			res := ssaTestResource("Admiral", false)
			res.API.Namespaced = false
			content := scaffoldTypes(res, true)

			Expect(content).To(ContainSubstring("// +kubebuilder:resource:scope=Cluster"))
			Expect(content).To(ContainSubstring("// +kubebuilder:ac:generate=false"))
			Expect(content).NotTo(ContainSubstring("+genclient"))
		})
	})

	Describe("optOutExistingKinds", func() {
		const captainTypes = `package v1

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type Captain struct{}
`

		var captainPath string

		BeforeEach(func() {
			tmpDir, err := os.MkdirTemp("", "ssa-opt-out")
			Expect(err).NotTo(HaveOccurred())
			oldWd, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Chdir(tmpDir)).To(Succeed())
			DeferCleanup(func() {
				Expect(os.Chdir(oldWd)).To(Succeed())
				_ = os.RemoveAll(tmpDir)
			})

			captainPath = filepath.Join("api", "v1", "captain_types.go")
			Expect(os.MkdirAll(filepath.Dir(captainPath), 0o755)).To(Succeed())
			Expect(os.WriteFile(captainPath, []byte(captainTypes), 0o644)).To(Succeed())
		})

		optOut := func(captain, navigator resource.Resource) {
			cfg := newSSATestConfig(captain, navigator)
			s := &apiScaffolder{config: cfg, resource: navigator}
			resources, err := cfg.GetResources()
			Expect(err).NotTo(HaveOccurred())
			s.optOutExistingKinds(resources)
		}

		It("should add the opt-out marker to kinds scaffolded without SSA", func() {
			optOut(ssaTestResource("Captain", false), ssaTestResource("Navigator", true))

			content, err := os.ReadFile(captainPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring(
				"// +kubebuilder:object:root=true\n// +kubebuilder:ac:generate=false"))
		})

		It("should keep a single opt-out marker when run more than once", func() {
			captain := ssaTestResource("Captain", false)
			navigator := ssaTestResource("Navigator", true)
			optOut(captain, navigator)
			optOut(captain, navigator)

			content, err := os.ReadFile(captainPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(string(content), "+kubebuilder:ac:generate=false")).To(Equal(1))
		})

		It("should not change kinds that have SSA enabled", func() {
			optOut(ssaTestResource("Captain", true), ssaTestResource("Navigator", true))

			content, err := os.ReadFile(captainPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(captainTypes))
		})

		It("should not change kinds that opted in manually with ac:generate=true", func() {
			optedIn := strings.Replace(captainTypes,
				"// +kubebuilder:object:root=true",
				"// +kubebuilder:ac:generate=true\n// +kubebuilder:object:root=true", 1)
			Expect(os.WriteFile(captainPath, []byte(optedIn), 0o644)).To(Succeed())

			optOut(ssaTestResource("Captain", false), ssaTestResource("Navigator", true))

			content, err := os.ReadFile(captainPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(optedIn))
		})

		It("should not change kinds in another group/version", func() {
			optOut(ssaTestResourceGV("Captain", "crew", "v2", false), ssaTestResource("Navigator", true))

			content, err := os.ReadFile(captainPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(captainTypes))
		})

		It("should warn and continue when a tracked kind has no types file", func() {
			sailor := ssaTestResource("Sailor", false)
			captain := ssaTestResource("Captain", false)
			navigator := ssaTestResource("Navigator", true)
			cfg := newSSATestConfig(sailor, captain, navigator)
			s := &apiScaffolder{config: cfg, resource: navigator}
			resources, err := cfg.GetResources()
			Expect(err).NotTo(HaveOccurred())

			s.optOutExistingKinds(resources)

			content, err := os.ReadFile(captainPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("+kubebuilder:ac:generate=false"))
		})
	})

	Describe("updateGroupVersionInfo", func() {
		const groupVersionInfo = `// Package v1 contains API Schema definitions for the crew v1 API group.
// +kubebuilder:object:generate=true
// +groupName=crew.test.io
package v1
`

		BeforeEach(func() {
			tmpDir, err := os.MkdirTemp("", "ssa-gv-info")
			Expect(err).NotTo(HaveOccurred())
			oldWd, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Chdir(tmpDir)).To(Succeed())
			DeferCleanup(func() {
				Expect(os.Chdir(oldWd)).To(Succeed())
				_ = os.RemoveAll(tmpDir)
			})
		})

		writeGroupVersionInfo := func(path string) {
			Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
			Expect(os.WriteFile(path, []byte(groupVersionInfo), 0o644)).To(Succeed())
		}

		It("should add the ac:generate marker after the object:generate marker", func() {
			gvPath := filepath.Join("api", "v1", "groupversion_info.go")
			writeGroupVersionInfo(gvPath)
			navigator := ssaTestResource("Navigator", true)
			s := &apiScaffolder{config: newSSATestConfig(navigator), resource: navigator}

			Expect(s.updateGroupVersionInfo()).To(Succeed())

			content, err := os.ReadFile(gvPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring(
				"// +kubebuilder:object:generate=true\n// +kubebuilder:ac:generate=true"))
		})

		It("should keep a single marker when run more than once", func() {
			gvPath := filepath.Join("api", "v1", "groupversion_info.go")
			writeGroupVersionInfo(gvPath)
			navigator := ssaTestResource("Navigator", true)
			s := &apiScaffolder{config: newSSATestConfig(navigator), resource: navigator}

			Expect(s.updateGroupVersionInfo()).To(Succeed())
			Expect(s.updateGroupVersionInfo()).To(Succeed())

			content, err := os.ReadFile(gvPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(string(content), "+kubebuilder:ac:generate=true")).To(Equal(1))
		})

		It("should use the group directory in multi-group projects", func() {
			gvPath := filepath.Join("api", "crew", "v1", "groupversion_info.go")
			writeGroupVersionInfo(gvPath)
			navigator := ssaTestResource("Navigator", true)
			cfg := newSSATestConfig(navigator)
			Expect(cfg.SetMultiGroup()).To(Succeed())
			s := &apiScaffolder{config: cfg, resource: navigator}

			Expect(s.updateGroupVersionInfo()).To(Succeed())

			content, err := os.ReadFile(gvPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("+kubebuilder:ac:generate=true"))
		})
	})

	Describe("replaceObjectGenInMakefile", func() {
		var makefilePath string

		BeforeEach(func() {
			tmpDir, err := os.MkdirTemp("", "ssa-makefile")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				_ = os.RemoveAll(tmpDir)
			})
			makefilePath = filepath.Join(tmpDir, "Makefile")
		})

		DescribeTable("should add applyconfiguration generation to known generate targets",
			func(oldTarget, newTarget string) {
				content := "generate: controller-gen\n\t" + oldTarget + "\n"
				Expect(os.WriteFile(makefilePath, []byte(content), 0o644)).To(Succeed())

				updated, err := addApplyConfigGenToMakefile(makefilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated).To(BeTrue())

				result, err := os.ReadFile(makefilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(result)).To(ContainSubstring(newTarget))
			},
			Entry("with boilerplate and year",
				makefileOldObjectGenWithBoilerplateAndYear, makefileNewObjectGenWithBoilerplateAndYear),
			Entry("with boilerplate",
				makefileOldObjectGenWithBoilerplate, makefileNewObjectGenWithBoilerplate),
			Entry("without boilerplate",
				makefileOldObjectGenNoBoilerplate, makefileNewObjectGenNoBoilerplate),
		)

		It("should update the generate target help text when it was not customized", func() {
			content := makefileOldGenerateHelp + "\n\t" + makefileOldObjectGenWithBoilerplate + "\n"
			Expect(os.WriteFile(makefilePath, []byte(content), 0o644)).To(Succeed())

			updated, err := addApplyConfigGenToMakefile(makefilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).To(BeTrue())

			result, err := os.ReadFile(makefilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(result)).To(ContainSubstring(makefileNewGenerateHelp))
		})

		It("should skip a Makefile that already runs applyconfiguration generation", func() {
			content := "generate: controller-gen\n\t" + makefileNewObjectGenWithBoilerplate + "\n"
			Expect(os.WriteFile(makefilePath, []byte(content), 0o644)).To(Succeed())

			updated, err := addApplyConfigGenToMakefile(makefilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).To(BeFalse())
		})

		It("should return an error when no known pattern matches", func() {
			content := "generate: controller-gen\n\tcustom-generator paths=\"./...\"\n"
			Expect(os.WriteFile(makefilePath, []byte(content), 0o644)).To(Succeed())

			_, err := addApplyConfigGenToMakefile(makefilePath)
			Expect(err).To(HaveOccurred())
		})

		It("should return an error when the Makefile does not exist", func() {
			_, err := addApplyConfigGenToMakefile(filepath.Join("does", "not", "exist", "Makefile"))
			Expect(err).To(HaveOccurred())
		})
	})
})
