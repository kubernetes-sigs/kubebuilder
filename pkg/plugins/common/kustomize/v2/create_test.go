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

package v2

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ = Describe("createSubcommand", func() {
	var (
		subCmd *createSubcommand
		cfg    config.Config
		res    *resource.Resource
	)

	BeforeEach(func() {
		subCmd = &createSubcommand{}
		cfg = cfgv3.New()
		res = &resource.Resource{
			GVK: resource.GVK{
				Group:   "crew",
				Domain:  "test.io",
				Version: "v1",
				Kind:    "Captain",
			},
		}
	})

	It("should parse force flag correctly", func() {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		fs.Bool("force", false, "force flag")
		subCmd.BindFlags(fs)

		err := fs.Set("force", "true")
		Expect(err).NotTo(HaveOccurred())

		err = subCmd.configure()
		Expect(err).NotTo(HaveOccurred())
		Expect(subCmd.force).To(BeTrue())
	})

	It("should return error for invalid force flag value", func() {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		fs.String("force", "invalid", "force flag")
		subCmd.BindFlags(fs)

		err := subCmd.configure()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid value for --force"))
	})

	It("should inject config and resource successfully", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		Expect(subCmd.config).To(Equal(cfg))

		Expect(subCmd.InjectResource(res)).To(Succeed())
		Expect(subCmd.resource).To(Equal(res))
	})
})
