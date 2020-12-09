/*
Copyright 2020 The Kubernetes Authors.

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

package resource_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "sigs.k8s.io/kubebuilder/v2/pkg/model/resource"
)

var _ = Describe("GVK", func() {
	Context("IsEqualTo", func() {
		var (
			a = GVK{
				Group:   "group",
				Version: "v1",
				Kind:    "Kind",
			}
			b = GVK{
				Group:   "group",
				Version: "v1",
				Kind:    "Kind",
			}
			c = GVK{
				Group:   "group2",
				Version: "v1",
				Kind:    "Kind",
			}
			d = GVK{
				Group:   "group",
				Version: "v2",
				Kind:    "Kind",
			}
			e = GVK{
				Group:   "group",
				Version: "v1",
				Kind:    "Kind2",
			}
		)

		It("should return true for itself", func() {
			Expect(a.IsEqualTo(a)).To(BeTrue())
		})

		It("should return true for the same GVK", func() {
			Expect(a.IsEqualTo(b)).To(BeTrue())
		})

		It("should return false if the group is different", func() {
			Expect(a.IsEqualTo(c)).To(BeFalse())
		})

		It("should return false if the version is different", func() {
			Expect(a.IsEqualTo(d)).To(BeFalse())
		})

		It("should return false if the kind is different", func() {
			Expect(a.IsEqualTo(e)).To(BeFalse())
		})
	})
})
