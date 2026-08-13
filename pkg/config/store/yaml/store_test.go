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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

func TestConfigStoreYaml(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Store YAML Suite")
}

// missingConfigError is the error the store reports when nothing is at the path.
func missingConfigError(path string) error {
	return store.LoadError{Err: fmt.Errorf("failed to read %q file: %w", path, os.ErrNotExist)}
}

// occupiedConfigError is the error the store reports when the path holds no configuration because
// something else is at it.
func occupiedConfigError(path, description string) error {
	return store.LoadError{Err: fmt.Errorf("%q is %s", path, description)}
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
		s    *yamlStore
		path string
	)

	BeforeEach(func() {
		s = New(machinery.Filesystem{FS: afero.NewMemMapFs()}).(*yamlStore)
		path = DefaultPath + "2"
	})

	Context("New", func() {
		It("should fail for an unregistered config version", func() {
			Expect(s.New(config.Version{})).NotTo(Succeed())
		})

		It("should require the configuration not to exist yet when saving it", func() {
			Expect(s.New(cfgv3.Version)).To(Succeed())
			Expect(afero.WriteFile(s.fs, DefaultPath, []byte(v3File), os.ModePerm)).To(Succeed())

			Expect(s.Save()).To(MatchError(ContainSubstring("configuration already exists")))
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
			Expect(err).To(MatchError(missingConfigError(DefaultPath)))
			Expect(err).To(MatchError(os.ErrNotExist))
		})

		It("should report a directory at the default path instead of a missing file", func() {
			Expect(s.fs.MkdirAll(DefaultPath, os.ModePerm)).To(Succeed())

			err := s.Load()
			Expect(err).To(MatchError(occupiedConfigError(DefaultPath, "a directory")))
			Expect(err).NotTo(MatchError(os.ErrNotExist))
		})

		It("should ignore directories named as the config file elsewhere in the project", func() {
			Expect(s.fs.MkdirAll(filepath.Join("docs", DefaultPath), os.ModePerm)).To(Succeed())
			Expect(afero.WriteFile(s.fs, DefaultPath, []byte(commentStr+v3File), os.ModePerm)).To(Succeed())

			Expect(s.Load()).To(Succeed())
			Expect(s.Config().GetVersion().Compare(cfgv3.Version)).To(Equal(0))
		})

		It("should fail if unable to identify the version of the file at the default path", func() {
			Expect(afero.WriteFile(s.fs, DefaultPath, []byte(commentStr+unversionedFile), os.ModePerm)).To(Succeed())

			err := s.Load()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(store.LoadError{
				Err: fmt.Errorf("failed to determine config version: %w",
					fmt.Errorf("error unmarshaling JSON: %w",
						fmt.Errorf("while decoding JSON: %w",
							errors.New("project version is empty"),
						),
					),
				),
			}))
		})

		It("should fail if unable to create a Config for the version of the file at the default path", func() {
			Expect(afero.WriteFile(s.fs, DefaultPath, []byte(commentStr+nonexistentVersionFile), os.ModePerm)).To(Succeed())

			err := s.Load()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(store.LoadError{
				Err: fmt.Errorf("failed to create config for version %q: %w", "1-alpha", config.UnsupportedVersionError{
					Version: config.Version{Number: 1, Stage: 2},
				}),
			}))
		})

		It("should fail if unable to unmarshal the file at the default path", func() {
			Expect(afero.WriteFile(s.fs, DefaultPath, []byte(commentStr+wrongFile), os.ModePerm)).To(Succeed())

			err := s.Load()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(store.LoadError{
				Err: fmt.Errorf("failed to create config for version %q: %w", "2", config.UnsupportedVersionError{
					Version: config.Version{
						Number: 2,
						Stage:  0,
					},
				}),
			}))
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
			Expect(err).To(MatchError(missingConfigError(path)))
			Expect(err).To(MatchError(os.ErrNotExist))
		})

		It("should report a directory at the specified path instead of a missing file", func() {
			Expect(s.fs.MkdirAll(path, os.ModePerm)).To(Succeed())

			err := s.LoadFrom(path)
			Expect(err).To(MatchError(occupiedConfigError(path, "a directory")))
			Expect(err).NotTo(MatchError(os.ErrNotExist))
		})

		It("should never read a non-regular file at the specified path", func() {
			Expect(afero.WriteFile(s.fs, path, []byte(commentStr+v3File), os.ModePerm)).To(Succeed())
			fs := &nonRegularFs{Fs: s.fs, path: path}
			s.fs = fs

			err := s.LoadFrom(path)
			Expect(err).To(MatchError(occupiedConfigError(path, "not a regular file")))
			Expect(fs.reads).To(BeZero())
		})

		It("should load the Config through a symlink to a regular file", func() {
			skipWithoutSymlinks()
			link, target := danglingSymlink()
			Expect(os.WriteFile(target, []byte(commentStr+v3File), 0o644)).To(Succeed())
			s.fs = afero.NewOsFs()

			Expect(s.LoadFrom(link)).To(Succeed())
			Expect(s.Config().GetVersion().Compare(cfgv3.Version)).To(Equal(0))
		})

		It("should report a dangling symbolic link instead of a missing file", func() {
			skipWithoutSymlinks()
			link, _ := danglingSymlink()
			s.fs = afero.NewOsFs()

			err := s.LoadFrom(link)
			Expect(err).To(MatchError(occupiedConfigError(link, "a symbolic link")))
			Expect(err).NotTo(MatchError(os.ErrNotExist))
		})

		It("should fail if unable to identify the version of the file at the specified path", func() {
			Expect(afero.WriteFile(s.fs, path, []byte(commentStr+unversionedFile), os.ModePerm)).To(Succeed())

			err := s.LoadFrom(path)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(store.LoadError{
				Err: fmt.Errorf("failed to determine config version: %w",
					fmt.Errorf("error unmarshaling JSON: %w",
						fmt.Errorf("while decoding JSON: %w",
							errors.New("project version is empty"),
						),
					),
				),
			}))
		})

		It("should fail if unable to create a Config for the version of the file at the specified path", func() {
			Expect(afero.WriteFile(s.fs, path, []byte(commentStr+nonexistentVersionFile), os.ModePerm)).To(Succeed())

			err := s.LoadFrom(path)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(store.LoadError{
				Err: fmt.Errorf("failed to create config for version %q: %w", "1-alpha", config.UnsupportedVersionError{
					Version: config.Version{Number: 1, Stage: 2},
				}),
			}))
		})

		It("should fail if unable to unmarshal the file at the specified path", func() {
			Expect(afero.WriteFile(s.fs, path, []byte(commentStr+wrongFile), os.ModePerm)).To(Succeed())

			err := s.LoadFrom(path)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(store.LoadError{
				Err: fmt.Errorf("failed to create config for version %q: %w", "2", config.UnsupportedVersionError{
					Version: config.Version{
						Number: 2,
					},
				}),
			}))
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
			Expect(err).To(MatchError(store.SaveError{
				Err: errors.New("undefined config, use one of the initializers: New, Load, LoadFrom"),
			}))
		})

		It("should fail for a pre-existent file that must not exist", func() {
			s.cfg = cfgv3.New()
			s.mustNotExist = true
			Expect(afero.WriteFile(s.fs, DefaultPath, []byte(v3File), os.ModePerm)).To(Succeed())

			err := s.Save()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(store.SaveError{
				Err: fmt.Errorf("configuration already exists in %q", DefaultPath),
			}))
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
			Expect(err).To(MatchError(store.SaveError{
				Err: errors.New("undefined config, use one of the initializers: New, Load, LoadFrom"),
			}))
		})

		It("should fail for a pre-existent file that must not exist", func() {
			s.cfg = cfgv3.New()
			s.mustNotExist = true
			Expect(afero.WriteFile(s.fs, path, []byte(v3File), os.ModePerm)).To(Succeed())

			err := s.SaveTo(path)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(store.SaveError{
				Err: fmt.Errorf("configuration already exists in %q", path),
			}))
		})

		It("should fail if a directory exists at the target path", func() {
			s.cfg = cfgv3.New()
			s.mustNotExist = true
			Expect(s.fs.MkdirAll(path, os.ModePerm)).To(Succeed())

			Expect(s.SaveTo(path)).To(MatchError(store.SaveError{
				Err: fmt.Errorf("cannot save configuration to %q: path is a directory", path),
			}))
		})

		It("should fail if a non-regular file exists at the target path", func() {
			s.cfg = cfgv3.New()
			s.mustNotExist = true
			Expect(afero.WriteFile(s.fs, path, []byte(v3File), os.ModePerm)).To(Succeed())
			s.fs = &nonRegularFs{Fs: s.fs, path: path}

			Expect(s.SaveTo(path)).To(MatchError(store.SaveError{
				Err: fmt.Errorf("cannot save configuration to %q: path is not a regular file", path),
			}))
		})

		It("should fail without creating the target if the target path is a dangling symbolic link", func() {
			skipWithoutSymlinks()
			link, target := danglingSymlink()

			s.fs = afero.NewOsFs()
			s.cfg = cfgv3.New()
			s.mustNotExist = true

			Expect(s.SaveTo(link)).To(MatchError(store.SaveError{
				Err: fmt.Errorf("cannot save configuration to %q: path is a symbolic link", link),
			}))
			Expect(target).NotTo(BeAnExistingFile())
		})

		It("should fail without creating the target of a dangling symbolic link the filesystem "+
			"cannot tell apart", func() {
			skipWithoutSymlinks()
			link, target := danglingSymlink()

			// A filesystem that is not an afero.Lstater cannot report the link, so the write itself
			// has to refuse the path.
			s.fs = &opaqueFs{Fs: afero.NewOsFs()}
			s.cfg = cfgv3.New()
			s.mustNotExist = true

			Expect(s.SaveTo(link)).NotTo(Succeed())
			Expect(target).NotTo(BeAnExistingFile())
		})

		It("should reject a symbolic link when updating an existing configuration", func() {
			skipWithoutSymlinks()
			link, target := danglingSymlink()
			existingContent := []byte(commentStr + v3File)
			Expect(os.WriteFile(target, existingContent, 0o644)).To(Succeed())

			s.fs = afero.NewOsFs()
			s.cfg = cfgv3.New()
			s.mustNotExist = false

			Expect(s.SaveTo(link)).To(MatchError(store.SaveError{
				Err: fmt.Errorf("cannot save configuration to %q: path is a symbolic link", link),
			}))
			Expect(target).To(BeAnExistingFile())
			content, err := os.ReadFile(target)
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal(existingContent))

			info, err := os.Lstat(link)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode() & os.ModeSymlink).NotTo(BeZero())
		})
	})

	Context("when the path cannot be checked", func() {
		var statErr error

		BeforeEach(func() {
			statErr = errors.New("permission denied")
			s.fs = &failingStatFs{Fs: s.fs, path: path, err: statErr}
		})

		It("should fail to load the Config", func() {
			Expect(s.LoadFrom(path)).To(MatchError(store.LoadError{
				Err: fmt.Errorf("failed to check %q: %w", path, statErr),
			}))
		})

		It("should fail to save a new Config", func() {
			s.cfg = cfgv3.New()
			s.mustNotExist = true

			Expect(s.SaveTo(path)).To(MatchError(store.SaveError{
				Err: fmt.Errorf("failed to check %q: %w", path, statErr),
			}))
		})
	})
})

var _ = Describe("pathClass", func() {
	DescribeTable("should describe what occupies the path",
		func(class pathClass, description string) {
			Expect(class.String()).To(Equal(description))
		},
		Entry("for a regular file", pathRegularFile, "a regular file"),
		Entry("for a directory", pathDirectory, "a directory"),
		Entry("for a symbolic link", pathSymbolicLink, "a symbolic link"),
		Entry("for any other file", pathIrregularFile, "not a regular file"),
		Entry("for a path where nothing is", pathMissing, ""),
		Entry("for an unknown class", pathClass(-1), ""),
	)
})

// skipWithoutSymlinks skips the test where creating a symbolic link needs extra privileges.
func skipWithoutSymlinks() {
	GinkgoHelper()

	if runtime.GOOS == "windows" {
		Skip("symlink creation requires elevated privileges on Windows")
	}
}

// danglingSymlink creates a symbolic link whose target does not exist, and returns both paths.
func danglingSymlink() (link, target string) {
	GinkgoHelper()

	dir := GinkgoT().TempDir()
	link = filepath.Join(dir, DefaultPath)
	target = filepath.Join(dir, "stolen.yaml")
	Expect(os.Symlink(target, link)).To(Succeed())

	return link, target
}

// opaqueFs hides the optional interfaces of the wrapped filesystem, such as afero.Lstater.
type opaqueFs struct {
	afero.Fs
}

// nonRegularFileInfo reports a device file mode on top of the wrapped FileInfo.
type nonRegularFileInfo struct{ os.FileInfo }

// Mode adds a device-file mode to the wrapped file information.
func (i nonRegularFileInfo) Mode() os.FileMode { return i.FileInfo.Mode() | os.ModeDevice }

// nonRegularFs makes path look like a non-regular file and counts how many times it is opened.
type nonRegularFs struct {
	afero.Fs
	path  string
	reads int
}

// Stat reports the configured path as a non-regular file.
func (f *nonRegularFs) Stat(name string) (os.FileInfo, error) {
	info, err := f.Fs.Stat(name)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %q: %w", name, err)
	}
	if name == f.path {
		info = nonRegularFileInfo{info}
	}

	return info, nil
}

// failingStatFs fails to stat the given path.
type failingStatFs struct {
	afero.Fs
	path string
	err  error
}

// Stat returns the configured error for the target path.
func (f *failingStatFs) Stat(name string) (os.FileInfo, error) {
	if name == f.path {
		return nil, f.err
	}

	info, err := f.Fs.Stat(name)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %q: %w", name, err)
	}

	return info, nil
}

// Open counts reads of the configured path before delegating to the wrapped filesystem.
func (f *nonRegularFs) Open(name string) (afero.File, error) {
	if name == f.path {
		f.reads++
	}

	file, err := f.Fs.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %w", name, err)
	}

	return file, nil
}
