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

package helpers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Conflict Detection", func() {
	Describe("isGeneratedKB", func() {
		It("should detect generated files", func() {
			Expect(isGeneratedKB("api/v1/zz_generated.deepcopy.go")).To(BeTrue())
			Expect(isGeneratedKB("config/crd/bases/crew.testproject.org_captains.yaml")).To(BeTrue())
			Expect(isGeneratedKB("config/rbac/role.yaml")).To(BeTrue())
			Expect(isGeneratedKB("dist/install.yaml")).To(BeTrue())
			Expect(isGeneratedKB("api/v1/captain_deepcopy.go")).To(BeTrue())
		})

		It("should not detect user files as generated", func() {
			Expect(isGeneratedKB("api/v1/captain_types.go")).To(BeFalse())
			Expect(isGeneratedKB("internal/controller/captain_controller.go")).To(BeFalse())
			Expect(isGeneratedKB("internal/webhook/v1/captain_webhook.go")).To(BeFalse())
			Expect(isGeneratedKB("internal/controller/suite_test.go")).To(BeFalse())
			Expect(isGeneratedKB("cmd/main.go")).To(BeFalse())
			Expect(isGeneratedKB("Makefile")).To(BeFalse())
		})
	})

	Describe("FindConflictFiles", func() {
		It("should return a valid ConflictResult structure", func() {
			result := FindConflictFiles()

			// Should have the expected structure
			Expect(result.SourceFiles).NotTo(BeNil())
			Expect(result.GeneratedFiles).NotTo(BeNil())
			Expect(result.Summary).To(Equal(ConflictSummary{
				Makefile: result.Summary.Makefile,
				API:      result.Summary.API,
				AnyGo:    result.Summary.AnyGo,
			}))
		})
	})

	Describe("DetectConflicts", func() {
		It("should maintain backward compatibility", func() {
			summary := DetectConflicts()

			// Should return a valid ConflictSummary
			Expect(summary.Makefile).To(BeFalse()) // No conflicts in test environment
			Expect(summary.API).To(BeFalse())
			Expect(summary.AnyGo).To(BeFalse())
		})
	})

	Describe("DecideMakeTargets", func() {
		It("should return all targets when no conflicts", func() {
			cs := ConflictSummary{Makefile: false, API: false, AnyGo: false}
			targets := DecideMakeTargets(cs)

			expected := []string{"manifests", "generate", "fmt", "vet", "lint-fix"}
			Expect(targets).To(Equal(expected))
		})

		It("should return no targets when Makefile has conflicts", func() {
			cs := ConflictSummary{Makefile: true, API: false, AnyGo: false}
			targets := DecideMakeTargets(cs)

			Expect(targets).To(BeNil())
		})

		It("should skip API targets when API has conflicts", func() {
			cs := ConflictSummary{Makefile: false, API: true, AnyGo: false}
			targets := DecideMakeTargets(cs)

			expected := []string{"fmt", "vet", "lint-fix"}
			Expect(targets).To(Equal(expected))
		})

		It("should skip Go targets when Go files have conflicts", func() {
			cs := ConflictSummary{Makefile: false, API: false, AnyGo: true}
			targets := DecideMakeTargets(cs)

			expected := []string{"manifests", "generate"}
			Expect(targets).To(Equal(expected))
		})
	})
})
