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

package v1alpha

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
)

var _ = Describe("Plugin", func() {
	var p Plugin

	It("should have the correct plugin name", func() {
		Expect(p.Name()).To(Equal("multicluster-runtime.kubebuilder.io"))
	})

	It("should be version 1 alpha", func() {
		Expect(p.Version().Number).To(Equal(1))
		Expect(p.Version().Stage).To(Equal(stage.Alpha))
	})

	It("should support project version v3", func() {
		Expect(p.SupportedProjectVersions()).To(ContainElement(cfgv3.Version))
	})

	It("should not be deprecated", func() {
		Expect(p.DeprecationWarning()).To(BeEmpty())
	})

	It("should return non-nil subcommands", func() {
		Expect(p.GetInitSubcommand()).NotTo(BeNil())
		Expect(p.GetCreateAPISubcommand()).NotTo(BeNil())
		Expect(p.GetEditSubcommand()).NotTo(BeNil())
	})
})
