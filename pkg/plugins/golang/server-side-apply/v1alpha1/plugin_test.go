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

	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var _ = Describe("Plugin", func() {
	Context("Name", func() {
		It("should return the correct name", func() {
			p := Plugin{}
			Expect(p.Name()).To(Equal("server-side-apply.go.kubebuilder.io"))
		})
	})

	Context("Version", func() {
		It("should return the correct version", func() {
			p := Plugin{}
			Expect(p.Version()).To(Equal(plugin.Version{Number: 1, Stage: stage.Alpha}))
		})
	})

	Context("SupportedProjectVersions", func() {
		It("should return the supported project versions", func() {
			p := Plugin{}
			versions := p.SupportedProjectVersions()
			Expect(versions).ToNot(BeEmpty())
		})
	})

	Context("Description", func() {
		It("should return a description", func() {
			p := Plugin{}
			Expect(p.Description()).NotTo(BeEmpty())
		})
	})

	Context("DeprecationWarning", func() {
		It("should return empty string (not deprecated)", func() {
			p := Plugin{}
			Expect(p.DeprecationWarning()).To(BeEmpty())
		})
	})
})
