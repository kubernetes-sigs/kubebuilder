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

package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var _ = Describe("CreateAPI", func() {
	var (
		cmd        createAPISubcommand
		cfg        config.Config
		res        *resource.Resource
		cliMeta    plugin.CLIMetadata
		subcmdMeta plugin.SubcommandMetadata
	)

	BeforeEach(func() {
		cfg = cfgv3.New()
		Expect(cfg.SetRepository("example.com/test")).To(Succeed())
		Expect(cfg.SetDomain("example.com")).To(Succeed())

		res = &resource.Resource{
			GVK: resource.GVK{
				Group:   "apps",
				Version: "v1",
				Kind:    "Application",
			},
			Plural: "applications",
			API:    &resource.API{},
		}

		cmd = createAPISubcommand{}
		cliMeta = plugin.CLIMetadata{CommandName: "kubebuilder"}
		subcmdMeta = plugin.SubcommandMetadata{}
	})

	Context("UpdateMetadata", func() {
		It("should set description and examples", func() {
			cmd.UpdateMetadata(cliMeta, &subcmdMeta)
			Expect(subcmdMeta.Description).NotTo(BeEmpty())
			Expect(subcmdMeta.Examples).NotTo(BeEmpty())
			Expect(subcmdMeta.Examples).To(ContainSubstring("server-side-apply"))
		})
	})

	Context("InjectConfig", func() {
		It("should inject config successfully", func() {
			err := cmd.InjectConfig(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(cmd.config).NotTo(BeNil())
		})
	})

	Context("InjectResource", func() {
		It("should inject resource and force controller=true", func() {
			err := cmd.InjectConfig(cfg)
			Expect(err).NotTo(HaveOccurred())

			err = cmd.InjectResource(res)
			Expect(err).NotTo(HaveOccurred())
			Expect(cmd.resource.Controller).To(BeTrue())
			Expect(cmd.resource.API).NotTo(BeNil())
		})
	})
})
