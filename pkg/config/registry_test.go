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

package config

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("registry", func() {
	var (
		version = Version{}
		f       = func() Config { return nil }
	)

	AfterEach(func() {
		registry = make(map[Version]func() Config)
	})

	Context("Register", func() {
		It("should register new constructors", func() {
			Register(version, f)
			Expect(registry).To(HaveKey(version))
			Expect(registry[version]()).To(BeNil())
		})
	})

	Context("IsRegistered", func() {
		It("should return true for registered constructors", func() {
			Register(version, f)
			Expect(IsRegistered(version)).To(BeTrue())
		})
		It("should fail for unregistered constructors", func() {
			Expect(IsRegistered(version)).To(BeFalse())
		})
	})

	Context("New", func() {
		It("should use the registered constructors", func() {
			registry[version] = f
			result, err := New(version)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should fail for unregistered constructors", func() {
			_, err := New(version)
			Expect(err).To(HaveOccurred())
		})
	})
})
