/*
Copyright 2021 The Kubernetes Authors.

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

package util

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Util Suite")
}

var _ = Describe("hasDifferentAPIVersion", func() {
	DescribeTable("should return false",
		func(versions []string) { Expect(hasDifferentAPIVersion(versions, "v1")).To(BeFalse()) },
		Entry("for an empty list of versions", []string{}),
		Entry("for a list of only that version", []string{"v1"}),
	)

	DescribeTable("should return true",
		func(versions []string) { Expect(hasDifferentAPIVersion(versions, "v1")).To(BeTrue()) },
		Entry("for a list of only a different version", []string{"v2"}),
		Entry("for a list of several different versions", []string{"v2", "v3"}),
		Entry("for a list of several versions containing that version", []string{"v1", "v2"}),
	)
})

var _ = Describe("CategorizeHubAndSpokes", func() {

	It("check if the right hub and spoke verisons are restured", func() {
		res := []resource.Resource{{
			GVK: resource.GVK{
				Group:   "group",
				Version: "v1",
				Kind:    "kind",
			},
			Webhooks: &resource.Webhooks{
				Spokes: []string{"v2", "v3"},
			},
		}}
		hub, spoke, err := categorizeHubAndSpokes(res)
		Expect(err).NotTo(HaveOccurred())
		Expect(hub).To(BeEquivalentTo("v1"))
		Expect(len(spoke)).To(BeEquivalentTo(2))
	})

	It("check if error is spoke and hub are nil when not present", func() {
		res := []resource.Resource{{
			GVK: resource.GVK{
				Group:   "group",
				Version: "v1",
				Kind:    "kind",
			},
		}}
		hub, spoke, err := categorizeHubAndSpokes(res)
		Expect(err).NotTo(HaveOccurred())
		Expect(hub).To(BeEquivalentTo(""))
		Expect(len(spoke)).To(BeEquivalentTo(0))
	})

	It("check if error occurs when multiple hubs are found", func() {
		res := []resource.Resource{{
			GVK: resource.GVK{
				Group:   "group",
				Version: "v1",
				Kind:    "kind",
			},
			Webhooks: &resource.Webhooks{
				Spokes: []string{"v2", "v3"},
			},
		},
			{
				GVK: resource.GVK{
					Group:   "group",
					Version: "v6",
					Kind:    "kind",
				},
				Webhooks: &resource.Webhooks{
					Spokes: []string{"v4", "v5"},
				},
			},
		}
		hub, spoke, err := categorizeHubAndSpokes(res)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("multiples hubs are not allowed"))
		Expect(hub).To(BeEquivalentTo(""))
		Expect(len(spoke)).To(BeEquivalentTo(0))
	})
})
