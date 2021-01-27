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
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	cfgv2 "sigs.k8s.io/kubebuilder/v3/pkg/config/v2"
)

var _ = Describe("readFrom", func() {
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

var _ = Describe("New", func() {
	It("should return a new Config backend without a stored Config", func() {
		cfg := New(afero.NewMemMapFs())
		Expect(cfg).NotTo(BeNil())
		Expect(cfg.fs).NotTo(BeNil())
		// path and mustNotExist will be checked per initializer
		Expect(cfg.Config).To(BeNil())
	})
})

var _ = Describe("Config", func() {
	var cfg *Config

	BeforeEach(func() {
		cfg = New(afero.NewMemMapFs())
	})

	Context("Init", func() {
		It("should initialize a new Config backend for the provided version", func() {
			Expect(cfg.Init(cfgv2.Version)).To(Succeed())
			Expect(cfg.fs).NotTo(BeNil())
			Expect(cfg.path).To(Equal(DefaultPath))
			Expect(cfg.mustNotExist).To(BeTrue())
			Expect(cfg.Config).NotTo(BeNil())
			Expect(cfg.Config.GetVersion().Compare(cfgv2.Version)).To(Equal(0))
		})
	})

	Context("InitTo", func() {
		It("should initialize a new Config backend for the provided version", func() {
			path := DefaultPath + "2"
			Expect(cfg.InitTo(path, cfgv2.Version)).To(Succeed())
			Expect(cfg.fs).NotTo(BeNil())
			Expect(cfg.path).To(Equal(path))
			Expect(cfg.mustNotExist).To(BeTrue())
			Expect(cfg.Config).NotTo(BeNil())
			Expect(cfg.Config.GetVersion().Compare(cfgv2.Version)).To(Equal(0))
		})
	})

	Context("Load", func() {
		It("should load a Config backend from an existing file", func() {
			configStr := `version: "2"`

			Expect(afero.WriteFile(cfg.fs, DefaultPath, []byte(configStr), os.ModePerm)).To(Succeed())

			Expect(cfg.Load()).To(Succeed())
			Expect(cfg.fs).NotTo(BeNil())
			Expect(cfg.path).To(Equal(DefaultPath))
			Expect(cfg.mustNotExist).To(BeFalse())
			Expect(cfg.Config).NotTo(BeNil())
			Expect(cfg.Config.GetVersion().Compare(cfgv2.Version)).To(Equal(0))
		})
	})

	Context("LoadFrom", func() {
		It("should load a Config backend from an existing file", func() {
			configStr := `version: "2"`
			path := DefaultPath + "2"

			Expect(afero.WriteFile(cfg.fs, path, []byte(configStr), os.ModePerm)).To(Succeed())

			Expect(cfg.LoadFrom(path)).To(Succeed())
			Expect(cfg.fs).NotTo(BeNil())
			Expect(cfg.path).To(Equal(path))
			Expect(cfg.mustNotExist).To(BeFalse())
			Expect(cfg.Config).NotTo(BeNil())
			Expect(cfg.Config.GetVersion().Compare(cfgv2.Version)).To(Equal(0))
		})
	})

	Context("Save", func() {
		It("should success for valid configs", func() {
			cfg.Config = cfgv2.New()
			Expect(cfg.Save()).To(Succeed())

			cfgBytes, err := afero.ReadFile(cfg.fs, DefaultPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(cfgBytes)).To(Equal(`version: "2"
`))
		})

		DescribeTable("should fail for invalid configs",
			func(cfg *Config) { Expect(cfg.Save()).NotTo(Succeed()) },
			Entry("constructor was not used", &Config{Config: cfgv2.New()}), // Rest of the fields are unexported
			Entry("no initializer called", New(afero.NewMemMapFs())),
		)
	})
})
