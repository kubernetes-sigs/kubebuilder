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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/hack"
)

var _ = Describe("Edit scaffolding of the license header", func() {
	Describe("setMakefileHeaderFile", func() {
		//nolint:lll
		const (
			manifestsWithHeader    = "\t\"$(CONTROLLER_GEN)\" rbac:roleName=manager-role crd webhook applyconfiguration:headerFile=\"hack/boilerplate.go.txt\" paths=\"./...\" output:crd:artifacts:config=config/crd/bases\n"
			manifestsWithoutHeader = "\t\"$(CONTROLLER_GEN)\" rbac:roleName=manager-role crd webhook applyconfiguration paths=\"./...\" output:crd:artifacts:config=config/crd/bases\n"
			generateWithHeader     = "\t\"$(CONTROLLER_GEN)\" object:headerFile=\"hack/boilerplate.go.txt\",year=$(YEAR) paths=\"./...\"\n"
			generateWithoutHeader  = "\t\"$(CONTROLLER_GEN)\" object paths=\"./...\"\n"
		)

		It("should remove headerFile from the generate and manifests targets when there is no boilerplate", func() {
			content := "manifests: controller-gen\n" + manifestsWithHeader +
				"generate: controller-gen\n" + generateWithHeader

			Expect(setMakefileHeaderFile(content, "")).To(Equal(
				"manifests: controller-gen\n" + manifestsWithoutHeader +
					"generate: controller-gen\n" + generateWithoutHeader))
		})

		It("should add headerFile to the generate and manifests targets when there is a boilerplate", func() {
			content := "manifests: controller-gen\n" + manifestsWithoutHeader +
				"generate: controller-gen\n" + generateWithoutHeader

			Expect(setMakefileHeaderFile(content, hack.DefaultBoilerplatePath)).To(Equal(
				"manifests: controller-gen\n" + manifestsWithHeader +
					"generate: controller-gen\n" + generateWithHeader))
		})

		It("should match the generators of the scaffolded Makefile", func() {
			Expect(generateWithHeader).To(ContainSubstring(
				generatorWithHeaderFile("object", hack.DefaultBoilerplatePath)))
			Expect(manifestsWithHeader).To(ContainSubstring(
				makefileManifestsGeneratorsWithSSA(hack.DefaultBoilerplatePath)))
			Expect(manifestsWithoutHeader).To(ContainSubstring(makefileManifestsGeneratorsWithSSA("")))
		})

		It("should not change a Makefile that already matches", func() {
			withHeader := "generate: controller-gen\n" + generateWithHeader + manifestsWithHeader
			withoutHeader := "generate: controller-gen\n" + generateWithoutHeader + manifestsWithoutHeader

			Expect(setMakefileHeaderFile(withHeader, hack.DefaultBoilerplatePath)).To(Equal(withHeader))
			Expect(setMakefileHeaderFile(withoutHeader, "")).To(Equal(withoutHeader))
		})

		It("should remove headerFile written by older releases without the year", func() {
			content := "\t$(CONTROLLER_GEN) object:headerFile=\"hack/boilerplate.go.txt\" paths=\"./...\"\n"

			Expect(setMakefileHeaderFile(content, "")).To(Equal("\t$(CONTROLLER_GEN) object paths=\"./...\"\n"))
		})

		It("should keep a headerFile that points to another file", func() {
			content := "\t\"$(CONTROLLER_GEN)\" object:headerFile=\"hack/custom.txt\" paths=\"./...\"\n"

			Expect(setMakefileHeaderFile(content, "")).To(Equal(content))
		})

		It("should only change controller-gen commands", func() {
			content := "# \"$(CONTROLLER_GEN)\" object:headerFile=\"hack/boilerplate.go.txt\",year=$(YEAR)\n" +
				"\t@echo object applyconfiguration\n" +
				"\tgo run ./tools object:headerFile=\"hack/boilerplate.go.txt\"\n"

			Expect(setMakefileHeaderFile(content, "")).To(Equal(content))
			Expect(setMakefileHeaderFile(content, hack.DefaultBoilerplatePath)).To(Equal(content))
		})
	})
})
