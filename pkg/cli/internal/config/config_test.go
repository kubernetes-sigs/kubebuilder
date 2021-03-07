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

package config

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	cfgv2 "sigs.k8s.io/kubebuilder/v3/pkg/config/v2"
)

var _ = Describe("Config", func() {
	Context("Save", func() {
		It("should success for valid configs", func() {
			cfg := Config{
				Config: cfgv2.New(),
				fs:     afero.NewMemMapFs(),
				path:   DefaultPath,
			}
			Expect(cfg.Save()).To(Succeed())

			cfgBytes, err := afero.ReadFile(cfg.fs, DefaultPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(cfgBytes)).To(Equal(`version: "2"
`))
		})

		It("should fail if path is not provided", func() {
			cfg := Config{
				Config: cfgv2.New(),
				fs:     afero.NewMemMapFs(),
			}
			Expect(cfg.Save()).NotTo(Succeed())
		})
	})

	Context("readFrom", func() {
		It("should success for valid configs", func() {
			configStr := `domain: example.com
repo: github.com/example/project
version: "2"`
			expectedConfig := cfgv2.New()
			_ = expectedConfig.SetDomain("example.com")
			_ = expectedConfig.SetRepository("github.com/example/project")

			fs := afero.NewMemMapFs()
			Expect(afero.WriteFile(fs, DefaultPath, []byte(configStr), os.ModePerm)).To(Succeed())

			cfg, err := readFrom(fs, DefaultPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).To(Equal(expectedConfig))
		})
	})
})
