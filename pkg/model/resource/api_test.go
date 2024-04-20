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

package resource

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//nolint:dupl
var _ = Describe("API", func() {
	Context("Validate", func() {
		It("should succeed for a valid API", func() {
			Expect(API{CRDVersion: v1}.Validate()).To(Succeed())
		})

		DescribeTable("should fail for invalid APIs",
			func(api API) { Expect(api.Validate()).NotTo(Succeed()) },
			// Ensure that the rest of the fields are valid to check each part
			Entry("empty CRD version", API{}),
			Entry("invalid CRD version", API{CRDVersion: "1"}),
		)
	})

	Context("Update", func() {
		var api, other API

		It("should do nothing if provided a nil pointer", func() {
			api = API{}
			Expect(api.Update(nil)).To(Succeed())
			Expect(api.CRDVersion).To(Equal(""))
			Expect(api.Namespaced).To(BeFalse())

			api = API{
				CRDVersion: v1,
				Namespaced: true,
			}
			Expect(api.Update(nil)).To(Succeed())
			Expect(api.CRDVersion).To(Equal(v1))
			Expect(api.Namespaced).To(BeTrue())
		})

		Context("CRD version", func() {
			It("should modify the CRD version if provided and not previously set", func() {
				api = API{}
				other = API{CRDVersion: v1}
				Expect(api.Update(&other)).To(Succeed())
				Expect(api.CRDVersion).To(Equal(v1))
			})

			It("should keep the CRD version if not provided", func() {
				api = API{CRDVersion: v1}
				other = API{}
				Expect(api.Update(&other)).To(Succeed())
				Expect(api.CRDVersion).To(Equal(v1))
			})

			It("should keep the CRD version if provided the same as previously set", func() {
				api = API{CRDVersion: v1}
				other = API{CRDVersion: v1}
				Expect(api.Update(&other)).To(Succeed())
				Expect(api.CRDVersion).To(Equal(v1))
			})

			It("should fail if previously set and provided CRD versions do not match", func() {
				api = API{CRDVersion: v1}
				other = API{CRDVersion: "v1beta1"}
				Expect(api.Update(&other)).NotTo(Succeed())
			})
		})

		Context("Namespaced", func() {
			It("should set the namespace scope if provided and not previously set", func() {
				api = API{}
				other = API{Namespaced: true}
				Expect(api.Update(&other)).To(Succeed())
				Expect(api.Namespaced).To(BeTrue())
			})

			It("should keep the namespace scope if previously set", func() {
				api = API{Namespaced: true}

				By("not providing it")
				other = API{}
				Expect(api.Update(&other)).To(Succeed())
				Expect(api.Namespaced).To(BeTrue())

				By("providing it")
				other = API{Namespaced: true}
				Expect(api.Update(&other)).To(Succeed())
				Expect(api.Namespaced).To(BeTrue())
			})

			It("should not set the namespace scope if not provided and not previously set", func() {
				api = API{}
				other = API{}
				Expect(api.Update(&other)).To(Succeed())
				Expect(api.Namespaced).To(BeFalse())
			})
		})
	})

	Context("IsEmpty", func() {
		var (
			none    = API{}
			cluster = API{
				CRDVersion: v1,
			}
			namespaced = API{
				CRDVersion: v1,
				Namespaced: true,
			}
		)

		It("should return true fo an empty object", func() {
			Expect(none.IsEmpty()).To(BeTrue())
		})

		DescribeTable("should return false for non-empty objects",
			func(api API) { Expect(api.IsEmpty()).To(BeFalse()) },
			Entry("cluster-scope", cluster),
			Entry("namespace-scope", namespaced),
		)
	})
})
