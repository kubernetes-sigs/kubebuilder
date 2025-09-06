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

package v2alpha

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
)

var _ = Describe("Plugin", func() {
	var p Plugin

	BeforeEach(func() {
		p = Plugin{}
	})

	Context("Name", func() {
		It("should return the correct plugin name", func() {
			Expect(p.Name()).To(Equal("helm.kubebuilder.io"))
		})
	})

	Context("Version", func() {
		It("should return version 2-alpha", func() {
			version := p.Version()
			Expect(version.Number).To(Equal(2))
			Expect(version.Stage.String()).To(Equal("alpha"))
		})
	})

	Context("SupportedProjectVersions", func() {
		It("should support project version 3", func() {
			versions := p.SupportedProjectVersions()
			expectedVersion := config.Version{Number: 3}
			Expect(versions).To(ContainElement(expectedVersion))
		})
	})

	Context("GetEditSubcommand", func() {
		It("should return an edit subcommand", func() {
			subcommand := p.GetEditSubcommand()
			Expect(subcommand).NotTo(BeNil())
		})
	})

	Context("DeprecationWarning", func() {
		It("should return empty string since v2-alpha is not deprecated", func() {
			warning := p.DeprecationWarning()
			Expect(warning).To(BeEmpty())
		})
	})
})
