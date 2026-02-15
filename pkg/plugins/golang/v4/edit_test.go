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

package v4

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("editSubcommand", func() {
	var (
		subCmd *editSubcommand
		cfg    config.Config
	)

	BeforeEach(func() {
		subCmd = &editSubcommand{}
		cfg = cfgv3.New()
	})

	It("should inject config successfully", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		Expect(subCmd.config).To(Equal(cfg))
	})

	Context("PreScaffold", func() {
		var (
			fs     *pflag.FlagSet
			mockFS machinery.Filesystem
		)

		BeforeEach(func() {
			fs = pflag.NewFlagSet("test", pflag.ContinueOnError)
			subCmd.BindFlags(fs)
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			mockFS = machinery.Filesystem{FS: afero.NewMemMapFs()}
		})

		It("should preserve existing multigroup setting when only namespaced flag is set", func() {
			// Set multigroup in PROJECT file
			Expect(cfg.SetMultiGroup()).To(Succeed())
			Expect(cfg.IsMultiGroup()).To(BeTrue())

			// Only set namespaced flag (multigroup not set, so it defaults to false)
			Expect(fs.Set("namespaced", "true")).To(Succeed())

			// PreScaffold should preserve the existing multigroup value
			Expect(subCmd.PreScaffold(mockFS)).To(Succeed())

			// Both should be true
			Expect(subCmd.multigroup).To(BeTrue(), "multigroup should be preserved from PROJECT file")
			Expect(subCmd.namespaced).To(BeTrue(), "namespaced should be set from flag")
		})

		It("should preserve existing namespaced setting when only multigroup flag is set", func() {
			// Set namespaced in PROJECT file
			Expect(cfg.SetNamespaced()).To(Succeed())
			Expect(cfg.IsNamespaced()).To(BeTrue())

			// Only set multigroup flag (namespaced not set, so it defaults to false)
			Expect(fs.Set("multigroup", "true")).To(Succeed())

			// PreScaffold should preserve the existing namespaced value
			Expect(subCmd.PreScaffold(mockFS)).To(Succeed())

			// Both should be true
			Expect(subCmd.multigroup).To(BeTrue(), "multigroup should be set from flag")
			Expect(subCmd.namespaced).To(BeTrue(), "namespaced should be preserved from PROJECT file")
		})

		It("should allow explicitly disabling a flag", func() {
			// Set both in PROJECT file
			Expect(cfg.SetMultiGroup()).To(Succeed())
			Expect(cfg.SetNamespaced()).To(Succeed())

			// Explicitly disable multigroup
			Expect(fs.Set("multigroup", "false")).To(Succeed())

			// PreScaffold should respect the explicit false
			Expect(subCmd.PreScaffold(mockFS)).To(Succeed())

			// multigroup should be false (explicitly set), namespaced should be true (from PROJECT)
			Expect(subCmd.multigroup).To(BeFalse(), "multigroup should be explicitly disabled")
			Expect(subCmd.namespaced).To(BeTrue(), "namespaced should be preserved from PROJECT file")
		})

		It("should use flag values when both flags are explicitly set", func() {
			// Set different values in PROJECT file
			Expect(cfg.SetMultiGroup()).To(Succeed())
			Expect(cfg.ClearNamespaced()).To(Succeed())

			// Explicitly set both flags to opposite values
			Expect(fs.Set("multigroup", "false")).To(Succeed())
			Expect(fs.Set("namespaced", "true")).To(Succeed())

			// PreScaffold should use the explicit flag values
			Expect(subCmd.PreScaffold(mockFS)).To(Succeed())

			Expect(subCmd.multigroup).To(BeFalse(), "multigroup should use explicit flag value")
			Expect(subCmd.namespaced).To(BeTrue(), "namespaced should use explicit flag value")
		})

		It("should preserve PROJECT file values when no flags are set", func() {
			// Set values in PROJECT file
			Expect(cfg.SetMultiGroup()).To(Succeed())
			Expect(cfg.SetNamespaced()).To(Succeed())

			// Don't set any flags

			// PreScaffold should preserve both values from PROJECT file
			Expect(subCmd.PreScaffold(mockFS)).To(Succeed())

			Expect(subCmd.multigroup).To(BeTrue(), "multigroup should be preserved from PROJECT file")
			Expect(subCmd.namespaced).To(BeTrue(), "namespaced should be preserved from PROJECT file")
		})
	})
})
