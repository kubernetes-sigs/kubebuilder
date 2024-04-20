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

//go:deprecated This package has been deprecated
package v2

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

func TestConfigV2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config V2 Suite")
}

var _ = Describe("cfg", func() {
	const (
		domain = "my.domain"
		repo   = "myrepo"

		otherDomain = "other.domain"
		otherRepo   = "otherrepo"
	)

	var c cfg

	BeforeEach(func() {
		c = cfg{
			Version:    Version,
			Domain:     domain,
			Repository: repo,
		}
	})

	Context("Version", func() {
		It("GetVersion should return version 2", func() {
			Expect(c.GetVersion().Compare(Version)).To(Equal(0))
		})
	})

	Context("Domain", func() {
		It("GetDomain should return the domain", func() {
			Expect(c.GetDomain()).To(Equal(domain))
		})

		It("SetDomain should set the domain", func() {
			Expect(c.SetDomain(otherDomain)).To(Succeed())
			Expect(c.Domain).To(Equal(otherDomain))
		})
	})

	Context("Repository", func() {
		It("GetRepository should return the repository", func() {
			Expect(c.GetRepository()).To(Equal(repo))
		})

		It("SetRepository should set the repository", func() {
			Expect(c.SetRepository(otherRepo)).To(Succeed())
			Expect(c.Repository).To(Equal(otherRepo))
		})
	})

	Context("Project name", func() {
		It("GetProjectName should return an empty name", func() {
			Expect(c.GetProjectName()).To(Equal(""))
		})

		It("SetProjectName should fail to set the name", func() {
			Expect(c.SetProjectName("name")).NotTo(Succeed())
		})
	})

	Context("Plugin chain", func() {
		It("GetPluginChain should return the only supported plugin", func() {
			Expect(c.GetPluginChain()).To(Equal([]string{"go.kubebuilder.io/v2"}))
		})

		It("SetPluginChain should fail to set the plugin chain", func() {
			Expect(c.SetPluginChain([]string{})).NotTo(Succeed())
		})
	})

	Context("Multi group", func() {
		It("IsMultiGroup should return false if not set", func() {
			Expect(c.IsMultiGroup()).To(BeFalse())
		})

		It("IsMultiGroup should return true if set", func() {
			c.MultiGroup = true
			Expect(c.IsMultiGroup()).To(BeTrue())
		})

		It("SetMultiGroup should enable multi-group support", func() {
			Expect(c.SetMultiGroup()).To(Succeed())
			Expect(c.MultiGroup).To(BeTrue())
		})

		It("ClearMultiGroup should disable multi-group support", func() {
			c.MultiGroup = true
			Expect(c.ClearMultiGroup()).To(Succeed())
			Expect(c.MultiGroup).To(BeFalse())
		})
	})

	Context("Component config", func() {
		It("IsComponentConfig should return false", func() {
			Expect(c.IsComponentConfig()).To(BeFalse())
		})

		It("SetComponentConfig should fail to enable component config support", func() {
			Expect(c.SetComponentConfig()).NotTo(Succeed())
		})

		It("ClearComponentConfig should fail to disable component config support", func() {
			Expect(c.ClearComponentConfig()).NotTo(Succeed())
		})
	})

	Context("Resources", func() {
		res := resource.Resource{
			GVK: resource.GVK{
				Group:   "group",
				Version: "v1",
				Kind:    "Kind",
			},
		}

		DescribeTable("ResourcesLength should return the number of resources",
			func(n int) {
				for i := 0; i < n; i++ {
					c.Gvks = append(c.Gvks, res.GVK)
				}
				Expect(c.ResourcesLength()).To(Equal(n))
			},
			Entry("for no resources", 0),
			Entry("for one resource", 1),
			Entry("for several resources", 3),
		)

		It("HasResource should return false for a non-existent resource", func() {
			Expect(c.HasResource(res.GVK)).To(BeFalse())
		})

		It("HasResource should return true for an existent resource", func() {
			c.Gvks = append(c.Gvks, res.GVK)
			Expect(c.HasResource(res.GVK)).To(BeTrue())
		})

		It("GetResource should fail for a non-existent resource", func() {
			_, err := c.GetResource(res.GVK)
			Expect(err).To(HaveOccurred())
		})

		It("GetResource should return an existent resource", func() {
			c.Gvks = append(c.Gvks, res.GVK)
			r, err := c.GetResource(res.GVK)
			Expect(err).NotTo(HaveOccurred())
			Expect(r.GVK.IsEqualTo(res.GVK)).To(BeTrue())
		})

		It("GetResources should return a slice of the tracked resources", func() {
			c.Gvks = append(c.Gvks, res.GVK, res.GVK, res.GVK)
			resources, err := c.GetResources()
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(Equal([]resource.Resource{res, res, res}))
		})

		It("AddResource should add the provided resource if non-existent", func() {
			l := len(c.Gvks)
			Expect(c.AddResource(res)).To(Succeed())
			Expect(len(c.Gvks)).To(Equal(l + 1))
			Expect(c.Gvks[0].IsEqualTo(res.GVK)).To(BeTrue())
		})

		It("AddResource should do nothing if the resource already exists", func() {
			c.Gvks = append(c.Gvks, res.GVK)
			l := len(c.Gvks)
			Expect(c.AddResource(res)).To(Succeed())
			Expect(len(c.Gvks)).To(Equal(l))
		})

		It("UpdateResource should add the provided resource if non-existent", func() {
			l := len(c.Gvks)
			Expect(c.UpdateResource(res)).To(Succeed())
			Expect(len(c.Gvks)).To(Equal(l + 1))
			Expect(c.Gvks[0].IsEqualTo(res.GVK)).To(BeTrue())
		})

		It("UpdateResource should do nothing if the resource already exists", func() {
			c.Gvks = append(c.Gvks, res.GVK)
			l := len(c.Gvks)
			Expect(c.UpdateResource(res)).To(Succeed())
			Expect(len(c.Gvks)).To(Equal(l))
		})

		It("HasGroup should return false with no tracked resources", func() {
			Expect(c.HasGroup(res.Group)).To(BeFalse())
		})

		It("HasGroup should return true with tracked resources in the same group", func() {
			c.Gvks = append(c.Gvks, res.GVK)
			Expect(c.HasGroup(res.Group)).To(BeTrue())
		})

		It("HasGroup should return false with tracked resources in other group", func() {
			c.Gvks = append(c.Gvks, res.GVK)
			Expect(c.HasGroup("other-group")).To(BeFalse())
		})

		It("ListCRDVersions should return an empty list", func() {
			Expect(c.ListCRDVersions()).To(BeEmpty())
		})

		It("ListWebhookVersions should return an empty list", func() {
			Expect(c.ListWebhookVersions()).To(BeEmpty())
		})
	})

	Context("Plugins", func() {
		It("DecodePluginConfig should fail", func() {
			Expect(c.DecodePluginConfig("", nil)).NotTo(Succeed())
		})

		It("EncodePluginConfig should fail", func() {
			Expect(c.EncodePluginConfig("", nil)).NotTo(Succeed())
		})
	})

	Context("Persistence", func() {
		var (
			// BeforeEach is called after the entries are evaluated, and therefore, c is not available
			c1 = cfg{
				Version:    Version,
				Domain:     domain,
				Repository: repo,
			}
			c2 = cfg{
				Version:    Version,
				Domain:     otherDomain,
				Repository: otherRepo,
				MultiGroup: true,
				Gvks: []resource.GVK{
					{Group: "group", Version: "v1", Kind: "Kind"},
					{Group: "group", Version: "v1", Kind: "Kind2"},
					{Group: "group", Version: "v1-beta", Kind: "Kind"},
					{Group: "group2", Version: "v1", Kind: "Kind"},
				},
			}
			s1 = `domain: my.domain
repo: myrepo
version: "2"
`
			s2 = `domain: other.domain
multigroup: true
repo: otherrepo
resources:
- group: group
  kind: Kind
  version: v1
- group: group
  kind: Kind2
  version: v1
- group: group
  kind: Kind
  version: v1-beta
- group: group2
  kind: Kind
  version: v1
version: "2"
`
		)

		DescribeTable("MarshalYAML should succeed",
			func(c cfg, content string) {
				b, err := c.MarshalYAML()
				Expect(err).NotTo(HaveOccurred())
				Expect(string(b)).To(Equal(content))
			},
			Entry("for a basic configuration", c1, s1),
			Entry("for a full configuration", c2, s2),
		)
		DescribeTable("UnmarshalYAML should succeed",
			func(content string, c cfg) {
				var unmarshalled cfg
				Expect(unmarshalled.UnmarshalYAML([]byte(content))).To(Succeed())
				Expect(unmarshalled.Version.Compare(c.Version)).To(Equal(0))
				Expect(unmarshalled.Domain).To(Equal(c.Domain))
				Expect(unmarshalled.Repository).To(Equal(c.Repository))
				Expect(unmarshalled.MultiGroup).To(Equal(c.MultiGroup))
				Expect(unmarshalled.Gvks).To(Equal(c.Gvks))
			},
			Entry("basic", s1, c1),
			Entry("full", s2, c2),
		)

		DescribeTable("UnmarshalYAML should fail",
			func(content string) {
				var c cfg
				Expect(c.UnmarshalYAML([]byte(content))).NotTo(Succeed())
			},
			Entry("for unknown fields", `field: 1
version: "2"`),
		)
	})
})

var _ = Describe("New", func() {
	It("should return a new config for project configuration 2", func() {
		Expect(New().GetVersion().Compare(Version)).To(Equal(0))
	})
})
