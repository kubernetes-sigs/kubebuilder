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

var _ = Describe("Open GitHub Issue Helpers", func() {
	Describe("excludedFromDiff", func() {
		It("should exclude unimportant files", func() {
			Expect(excludedFromDiff("PROJECT")).To(BeTrue())
			Expect(excludedFromDiff("README.md")).To(BeTrue())
			Expect(excludedFromDiff("hack/boilerplate.go.txt")).To(BeTrue())
			Expect(excludedFromDiff("go.sum")).To(BeTrue())
		})

		It("should include important files", func() {
			Expect(excludedFromDiff("go.mod")).To(BeFalse())
			Expect(excludedFromDiff("Makefile")).To(BeFalse())
			Expect(excludedFromDiff("Dockerfile")).To(BeFalse())
		})
	})

	Describe("importantFile", func() {
		It("should identify important core files", func() {
			Expect(importantFile("go.mod")).To(BeTrue())
			Expect(importantFile("Makefile")).To(BeTrue())
			Expect(importantFile("cmd/main.go")).To(BeTrue())
			Expect(importantFile("api/v1/captain_types.go")).To(BeTrue())
		})

		It("should exclude generated and unimportant files", func() {
			Expect(importantFile("PROJECT")).To(BeFalse())
			Expect(importantFile("hack/boilerplate.go.txt")).To(BeFalse())
			Expect(importantFile("config/crd/bases/captain.yaml")).To(BeFalse())
		})
	})

	Describe("filePriority", func() {
		It("should prioritize files correctly", func() {
			Expect(filePriority("go.mod")).To(Equal(0))
			Expect(filePriority("Makefile")).To(Equal(1))
			Expect(filePriority("Dockerfile")).To(Equal(2))
			Expect(filePriority("cmd/main.go")).To(Equal(3))
			Expect(filePriority("config/default/kustomization.yaml")).To(Equal(4))
		})
	})

	Describe("FirstURL", func() {
		It("should extract URLs from text", func() {
			Expect(FirstURL("https://github.com/user/repo")).To(Equal("https://github.com/user/repo"))
			Expect(FirstURL("Check https://example.com here")).To(Equal("https://example.com"))
			Expect(FirstURL("no links here")).To(Equal(""))
		})
	})

	Describe("IssueNumberFromURL", func() {
		It("should extract issue numbers", func() {
			Expect(IssueNumberFromURL("https://github.com/user/repo/issues/123")).To(Equal("123"))
			Expect(IssueNumberFromURL("https://github.com/user/repo/pull/456")).To(Equal("456"))
		})
	})

	Describe("bulletList", func() {
		It("should format bullet lists", func() {
			Expect(bulletList([]string{})).To(Equal("<none>"))
			Expect(bulletList([]string{"item1"})).To(Equal("- item1"))
			Expect(bulletList([]string{"item1", "item2"})).To(Equal("- item1\n- item2"))
		})
	})

	Describe("keepGoModLine", func() {
		It("should keep important go.mod lines", func() {
			Expect(keepGoModLine("+module github.com/user/repo")).To(BeTrue())
			Expect(keepGoModLine("+go 1.21")).To(BeTrue())
			Expect(keepGoModLine("+require example.com/pkg v1.0.0")).To(BeTrue())
		})

		It("should skip indirect dependencies", func() {
			Expect(keepGoModLine("+require example.com/pkg v1.0.0 // indirect")).To(BeFalse())
		})
	})

	Describe("interestingLine", func() {
		It("should detect interesting Go lines", func() {
			Expect(interestingLine("main.go", "+import \"context\"")).To(BeTrue())
			Expect(interestingLine("controller.go", "+//+kubebuilder:rbac:groups=apps")).To(BeTrue())
		})

		It("should detect interesting YAML lines", func() {
			Expect(interestingLine("manager.yaml", "+apiVersion: apps/v1")).To(BeTrue())
			Expect(interestingLine("config.yaml", "+image: controller:latest")).To(BeTrue())
		})

		It("should skip uninteresting lines", func() {
			Expect(interestingLine("main.go", "+x := 1")).To(BeFalse())
			Expect(interestingLine("config.yaml", "+# comment")).To(BeFalse())
		})
	})
})
