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
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("initSubcommand", func() {
	var (
		subCmd *initSubcommand
		cfg    config.Config
		fs     *pflag.FlagSet
	)

	BeforeEach(func() {
		subCmd = &initSubcommand{}
		cfg = cfgv3.New()
		_ = cfg.SetRepository("github.com/example/myop")
		_ = cfg.SetDomain("example.com")
		fs = pflag.NewFlagSet("test", pflag.ContinueOnError)
		subCmd.BindFlags(fs)
	})

	Context("InjectConfig", func() {
		It("should store the config", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			Expect(subCmd.config).To(Equal(cfg))
		})
	})

	Context("BindFlags defaults", func() {
		It("should default provider to kubeconfig", func() {
			Expect(subCmd.provider).To(Equal("kubeconfig"))
		})

		It("should default kubeconfig-dir to /etc/kubeconfig", func() {
			Expect(subCmd.kubeconfigDir).To(Equal("/etc/kubeconfig"))
		})
	})

	Context("Scaffold with valid providers", func() {
		var memFS machinery.Filesystem

		BeforeEach(func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			memFS = machinery.Filesystem{FS: afero.NewMemMapFs()}
		})

		for _, provider := range []string{"kubeconfig", "namespace", "cluster-api", "file"} {
			It("should succeed with provider "+provider, func() {
				subCmd.provider = provider
				Expect(subCmd.Scaffold(memFS)).To(Succeed())
			})
		}
	})

	Context("Scaffold with invalid provider", func() {
		It("should return an error for unknown provider", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			subCmd.provider = "bogus"
			err := subCmd.Scaffold(machinery.Filesystem{FS: afero.NewMemMapFs()})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown provider"))
			Expect(err.Error()).To(ContainSubstring("bogus"))
		})
	})
})

var _ = Describe("validateProvider", func() {
	for _, v := range []string{"kubeconfig", "namespace", "cluster-api", "file"} {
		It("should accept "+v, func() {
			Expect(validateProvider(v)).To(Succeed())
		})
	}

	It("should reject an unknown provider", func() {
		err := validateProvider("unknown")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unknown provider"))
		Expect(err.Error()).To(ContainSubstring("kubeconfig, namespace, cluster-api, file"))
	})
})
