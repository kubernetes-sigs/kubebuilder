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

package yaml

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	cfgv2 "sigs.k8s.io/kubebuilder/v3/pkg/config/v2"
)

var _ = Describe("New", func() {
	It("should return a new empty store", func() {
		s := New(afero.NewMemMapFs())
		Expect(s.Config()).To(BeNil())

		ys, ok := s.(*yamlStore)
		Expect(ok).To(BeTrue())
		Expect(ys.fs).NotTo(BeNil())
	})
})

var _ = Describe("yamlStore", func() {
	var (
		s *yamlStore
	)

	BeforeEach(func() {
		s = New(afero.NewMemMapFs()).(*yamlStore)
	})

	Context("New", func() {
		It("should initialize a new Config backend for the provided version", func() {
			Expect(s.New(cfgv2.Version)).To(Succeed())
			Expect(s.fs).NotTo(BeNil())
			Expect(s.mustNotExist).To(BeTrue())
			Expect(s.Config()).NotTo(BeNil())
			Expect(s.Config().GetVersion().Compare(cfgv2.Version)).To(Equal(0))
		})
	})

	Context("Load", func() {
		It("should load the Config from an existing file at the default path", func() {
			configStr := `version: "2"`

			Expect(afero.WriteFile(s.fs, DefaultPath, []byte(configStr), os.ModePerm)).To(Succeed())

			Expect(s.Load()).To(Succeed())
			Expect(s.fs).NotTo(BeNil())
			Expect(s.mustNotExist).To(BeFalse())
			Expect(s.Config()).NotTo(BeNil())
			Expect(s.Config().GetVersion().Compare(cfgv2.Version)).To(Equal(0))
		})
	})

	Context("LoadFrom", func() {
		It("should load the Config from an existing file from the specified path", func() {
			configStr := `version: "2"`
			path := DefaultPath + "2"

			Expect(afero.WriteFile(s.fs, path, []byte(configStr), os.ModePerm)).To(Succeed())

			Expect(s.LoadFrom(path)).To(Succeed())
			Expect(s.fs).NotTo(BeNil())
			Expect(s.mustNotExist).To(BeFalse())
			Expect(s.Config()).NotTo(BeNil())
			Expect(s.Config().GetVersion().Compare(cfgv2.Version)).To(Equal(0))
		})
	})

	Context("Save", func() {
		It("should success for valid configs", func() {
			s.cfg = cfgv2.New()
			Expect(s.Save()).To(Succeed())

			cfgBytes, err := afero.ReadFile(s.fs, DefaultPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(cfgBytes)).To(Equal(`version: "2"
`))
		})

		It("should fail for an empty config", func() {
			Expect(s.Save()).NotTo(Succeed())
		})
	})

	Context("SaveTo", func() {
		path := DefaultPath + "2"

		It("should success for valid configs", func() {
			s.cfg = cfgv2.New()
			Expect(s.SaveTo(path)).To(Succeed())

			cfgBytes, err := afero.ReadFile(s.fs, path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(cfgBytes)).To(Equal(`version: "2"
`))
		})

		It("should fail for an empty config", func() {
			Expect(s.SaveTo(path)).NotTo(Succeed())
		})
	})
})
