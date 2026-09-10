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
})
