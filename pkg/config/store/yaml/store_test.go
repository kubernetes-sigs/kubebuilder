/*
Copyright 2022 The Kubernetes Authors.

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
	"errors"
	"os"
	"testing"

	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

func TestConfigStoreYaml(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Store YAML Suite")
}

var _ = Describe("New", func() {
	It("should return a new empty store", func() {
		s := New(machinery.Filesystem{FS: afero.NewMemMapFs()})
		Expect(s.Config()).To(BeNil())

		ys, ok := s.(*yamlStore)
		Expect(ok).To(BeTrue())
		Expect(ys.fs).NotTo(BeNil())
	})
})

var _ = Describe("yamlStore", func() {
	const (
		v3File = `version: "3"
`
		unversionedFile = `version:
`
		nonexistentVersionFile = `version: 1-alpha
` // v1-alpha never existed
		wrongFile = `version: "2"
layout: ""
` // layout field does not exist in v2
	)

	var (
		s *yamlStore

		path = DefaultPath + "2"
	)

	BeforeEach(func() {
		s = New(machinery.Filesystem{FS: afero.NewMemMapFs()}).(*yamlStore)
	})

	Context("New", func() {
		It("should fail for an unregistered config version", func() {
			Expect(s.New(config.Version{})).NotTo(Succeed())
		})
	})

	Context("Load", func() {
		It("should load the Config from an existing file at the default path", func() {
			Expect(afero.WriteFile(s.fs, DefaultPath, []byte(commentStr+v3File), os.ModePerm)).To(Succeed())

			Expect(s.Load()).To(Succeed())
			Expect(s.fs).NotTo(BeNil())
			Expect(s.mustNotExist).To(BeFalse())
			Expect(s.Config()).NotTo(BeNil())
			Expect(s.Config().GetVersion().Compare(cfgv3.Version)).To(Equal(0))
		})

		It("should fail if no file exists at the default path", func() {
			err := s.Load()
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &store.LoadError{})).To(BeTrue())
		})

		It("should fail if unable to identify the version of the file at the default path", func() {
			Expect(afero.WriteFile(s.fs, DefaultPath, []byte(commentStr+unversionedFile), os.ModePerm)).To(Succeed())

			err := s.Load()
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &store.LoadError{})).To(BeTrue())
		})

		It("should fail if unable to create a Config for the version of the file at the default path", func() {
			Expect(afero.WriteFile(s.fs, DefaultPath, []byte(commentStr+nonexistentVersionFile), os.ModePerm)).To(Succeed())

			err := s.Load()
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &store.LoadError{})).To(BeTrue())
		})

		It("should fail if unable to unmarshal the file at the default path", func() {
			Expect(afero.WriteFile(s.fs, DefaultPath, []byte(commentStr+wrongFile), os.ModePerm)).To(Succeed())

			err := s.Load()
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &store.LoadError{})).To(BeTrue())
		})
	})

	Context("LoadFrom", func() {
		It("should load the Config from an existing file from the specified path", func() {
			Expect(afero.WriteFile(s.fs, path, []byte(commentStr+v3File), os.ModePerm)).To(Succeed())

			Expect(s.LoadFrom(path)).To(Succeed())
			Expect(s.fs).NotTo(BeNil())
			Expect(s.mustNotExist).To(BeFalse())
			Expect(s.Config()).NotTo(BeNil())
			Expect(s.Config().GetVersion().Compare(cfgv3.Version)).To(Equal(0))
		})

		It("should fail if no file exists at the specified path", func() {
			err := s.LoadFrom(path)
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &store.LoadError{})).To(BeTrue())
		})

		It("should fail if unable to identify the version of the file at the specified path", func() {
			Expect(afero.WriteFile(s.fs, path, []byte(commentStr+unversionedFile), os.ModePerm)).To(Succeed())

			err := s.LoadFrom(path)
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &store.LoadError{})).To(BeTrue())
		})

		It("should fail if unable to create a Config for the version of the file at the specified path", func() {
			Expect(afero.WriteFile(s.fs, path, []byte(commentStr+nonexistentVersionFile), os.ModePerm)).To(Succeed())

			err := s.LoadFrom(path)
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &store.LoadError{})).To(BeTrue())
		})

		It("should fail if unable to unmarshal the file at the specified path", func() {
			Expect(afero.WriteFile(s.fs, path, []byte(commentStr+wrongFile), os.ModePerm)).To(Succeed())

			err := s.LoadFrom(path)
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &store.LoadError{})).To(BeTrue())
		})
	})

	Context("Save", func() {
		It("should succeed for a valid config", func() {
			s.cfg = cfgv3.New()
			Expect(s.Save()).To(Succeed())

			cfgBytes, err := afero.ReadFile(s.fs, DefaultPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(cfgBytes)).To(Equal(commentStr + v3File))
		})

		It("should succeed for a valid config that must not exist", func() {
			s.cfg = cfgv3.New()
			s.mustNotExist = true
			Expect(s.Save()).To(Succeed())

			cfgBytes, err := afero.ReadFile(s.fs, DefaultPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(cfgBytes)).To(Equal(commentStr + v3File))
		})

		It("should fail for an empty config", func() {
			err := s.Save()
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &store.SaveError{})).To(BeTrue())
		})

		It("should fail for a pre-existent file that must not exist", func() {
			s.cfg = cfgv3.New()
			s.mustNotExist = true
			Expect(afero.WriteFile(s.fs, DefaultPath, []byte(v3File), os.ModePerm)).To(Succeed())

			err := s.Save()
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &store.SaveError{})).To(BeTrue())
		})
	})

	Context("SaveTo", func() {
		It("should success for valid configs", func() {
			s.cfg = cfgv3.New()
			Expect(s.SaveTo(path)).To(Succeed())

			cfgBytes, err := afero.ReadFile(s.fs, path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(cfgBytes)).To(Equal(commentStr + v3File))
		})

		It("should succeed for a valid config that must not exist", func() {
			s.cfg = cfgv3.New()
			s.mustNotExist = true
			Expect(s.SaveTo(path)).To(Succeed())

			cfgBytes, err := afero.ReadFile(s.fs, path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(cfgBytes)).To(Equal(commentStr + v3File))
		})

		It("should fail for an empty config", func() {
			err := s.SaveTo(path)
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &store.SaveError{})).To(BeTrue())
		})

		It("should fail for a pre-existent file that must not exist", func() {
			s.cfg = cfgv3.New()
			s.mustNotExist = true
			Expect(afero.WriteFile(s.fs, path, []byte(v3File), os.ModePerm)).To(Succeed())

			err := s.SaveTo(path)
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &store.SaveError{})).To(BeTrue())
		})
	})
})
