/*
Copyright 2020 The Kubernetes Authors.

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

	"sigs.k8s.io/kubebuilder/pkg/model/config"
)

var _ = Describe("Config", func() {
	It("should save correctly", func() {
		var (
			cfg               Config
			expectedConfigStr string
		)

		By("saving empty config")
		Expect(cfg.Save()).To(HaveOccurred())

		By("saving empty config with path")
		cfg = Config{
			fs:   afero.NewMemMapFs(),
			path: DefaultPath,
		}
		Expect(cfg.Save()).ToNot(HaveOccurred())
		cfgBytes, err := afero.ReadFile(cfg.fs, DefaultPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(cfgBytes)).To(Equal(expectedConfigStr))

		By("saving config version 2")
		cfg = Config{
			Config: config.Config{
				Version: config.Version2,
				Repo:    "github.com/example/project",
				Domain:  "example.com",
			},
			fs:   afero.NewMemMapFs(),
			path: DefaultPath,
		}
		expectedConfigStr = `domain: example.com
repo: github.com/example/project
version: "2"
`
		Expect(cfg.Save()).ToNot(HaveOccurred())
		cfgBytes, err = afero.ReadFile(cfg.fs, DefaultPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(cfgBytes)).To(Equal(expectedConfigStr))

		By("saving config version 2 with plugin config")
		cfg = Config{
			Config: config.Config{
				Version: config.Version2,
				Repo:    "github.com/example/project",
				Domain:  "example.com",
				Plugins: map[string]interface{}{
					"plugin-x": map[string]interface{}{
						"data-1": "single plugin datum",
					},
					"plugin-y/v1": map[string]interface{}{
						"data-1": "plugin value 1",
						"data-2": "plugin value 2",
						"data-3": []string{"plugin value 3", "plugin value 4"},
					},
				},
			},
			fs:   afero.NewMemMapFs(),
			path: DefaultPath,
		}
		expectedConfigStr = `domain: example.com
repo: github.com/example/project
version: "2"
plugins:
  plugin-x:
    data-1: single plugin datum
  plugin-y/v1:
    data-1: plugin value 1
    data-2: plugin value 2
    data-3:
    - plugin value 3
    - plugin value 4
`
		Expect(cfg.Save()).ToNot(HaveOccurred())
		cfgBytes, err = afero.ReadFile(cfg.fs, DefaultPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(cfgBytes)).To(Equal(expectedConfigStr))
	})

	It("should load correctly", func() {
		var (
			fs             = afero.NewMemMapFs()
			configStr      string
			expectedConfig config.Config
		)

		By("loading config version 2")
		configStr = `domain: example.com
repo: github.com/example/project
version: "2"`
		expectedConfig = config.Config{
			Version: config.Version2,
			Repo:    "github.com/example/project",
			Domain:  "example.com",
		}
		err := afero.WriteFile(fs, DefaultPath, []byte(configStr), os.ModePerm)
		Expect(err).ToNot(HaveOccurred())
		cfg, err := readFrom(fs, DefaultPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).To(BeEquivalentTo(expectedConfig))

		By("loading config version 2 with plugin config")
		fs = afero.NewMemMapFs()
		configStr = `domain: example.com
repo: github.com/example/project
version: "2"
plugins:
  plugin-x:
    data-1: single plugin datum
  plugin-y/v1:
    data-1: plugin value 1
    data-2: plugin value 2
    data-3:
    - "plugin value 3"
    - "plugin value 4"`
		expectedConfig = config.Config{
			Version: config.Version2,
			Repo:    "github.com/example/project",
			Domain:  "example.com",
			Plugins: map[string]interface{}{
				"plugin-x": map[string]interface{}{
					"data-1": "single plugin datum",
				},
				"plugin-y/v1": map[string]interface{}{
					"data-1": "plugin value 1",
					"data-2": "plugin value 2",
					"data-3": []interface{}{"plugin value 3", "plugin value 4"},
				},
			},
		}
		err = afero.WriteFile(fs, DefaultPath, []byte(configStr), os.ModePerm)
		Expect(err).ToNot(HaveOccurred())
		cfg, err = readFrom(fs, DefaultPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).To(Equal(expectedConfig))
	})
})
