/*
Copyright 2022 The Kubernetes Authors.

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
)

var _ = Describe("validateNamespace", func() {
	Context("with valid namespaces", func() {
		It("should accept simple lowercase names", func() {
			Expect(validateNamespace("monitoring")).To(Succeed())
			Expect(validateNamespace("system")).To(Succeed())
		})

		It("should accept names with hyphens", func() {
			Expect(validateNamespace("monitoring-system")).To(Succeed())
			Expect(validateNamespace("my-namespace")).To(Succeed())
		})

		It("should accept names with numbers", func() {
			Expect(validateNamespace("namespace1")).To(Succeed())
			Expect(validateNamespace("123namespace")).To(Succeed())
			Expect(validateNamespace("my-ns-123")).To(Succeed())
		})

		It("should accept maximum length (63 chars)", func() {
			longName := "a123456789012345678901234567890123456789012345678901234567890bc"
			Expect(len(longName)).To(Equal(63))
			Expect(validateNamespace(longName)).To(Succeed())
		})
	})

	Context("with invalid namespaces", func() {
		It("should reject empty namespace", func() {
			err := validateNamespace("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be empty"))
		})

		It("should reject names longer than 63 characters", func() {
			tooLong := "a123456789012345678901234567890123456789012345678901234567890123"
			Expect(len(tooLong)).To(Equal(64))
			err := validateNamespace(tooLong)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("63 characters or less"))
		})

		It("should reject names with uppercase letters", func() {
			err := validateNamespace("Monitoring")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("lowercase alphanumeric"))
		})

		It("should reject names with special characters", func() {
			Expect(validateNamespace("monitoring_system")).To(MatchError(ContainSubstring("lowercase alphanumeric")))
			Expect(validateNamespace("monitoring.system")).To(MatchError(ContainSubstring("lowercase alphanumeric")))
			Expect(validateNamespace("monitoring@system")).To(MatchError(ContainSubstring("lowercase alphanumeric")))
		})

		It("should reject names starting with hyphen", func() {
			err := validateNamespace("-monitoring")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("start and end with an alphanumeric"))
		})

		It("should reject names ending with hyphen", func() {
			err := validateNamespace("monitoring-")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("start and end with an alphanumeric"))
		})
	})
})
