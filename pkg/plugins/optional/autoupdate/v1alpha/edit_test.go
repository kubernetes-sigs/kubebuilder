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
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("editSubcommand", func() {
	var (
		subCmd *editSubcommand
		cfg    config.Config
		fs     machinery.Filesystem
	)

	BeforeEach(func() {
		subCmd = &editSubcommand{}
		cfg = cfgv3.New()
		fs = machinery.Filesystem{FS: afero.NewMemMapFs()}
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
	})

	It("should require cliVersion to be set in PROJECT file", func() {
		err := subCmd.PreScaffold(fs)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("must manually upgrade your project"))
		Expect(err.Error()).To(ContainSubstring("cliVersion"))
	})

	It("should succeed when cliVersion is set", func() {
		Expect(cfg.SetCliVersion("v4.0.0")).To(Succeed())

		err := subCmd.PreScaffold(fs)

		Expect(err).NotTo(HaveOccurred())
	})
})
