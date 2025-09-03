/*
Copyright 2025 The Kubernetes Authors.

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

package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	v3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

type fakeConfig struct {
	config.Config
	pluginChain []string
	domain      string
	repo        string
	multigroup  bool
	resources   []resource.Resource
	pluginErr   error
	getResErr   error
	plugins     map[string]any
}

func (f *fakeConfig) GetPluginChain() []string { return f.pluginChain }
func (f *fakeConfig) GetDomain() string        { return f.domain }
func (f *fakeConfig) GetRepository() string    { return f.repo }
func (f *fakeConfig) IsMultiGroup() bool       { return f.multigroup }
func (f *fakeConfig) GetResources() ([]resource.Resource, error) {
	if f.getResErr != nil {
		return nil, f.getResErr
	}
	return f.resources, nil
}

func (f *fakeConfig) DecodePluginConfig(key string, _ any) error {
	if len(f.plugins) == 0 {
		return config.PluginKeyNotFoundError{Key: key}
	}
	if f.pluginErr != nil {
		return f.pluginErr
	}
	return nil
}

type fakeStore struct {
	store.Store
	cfg *fakeConfig
}

func (f *fakeStore) Config() config.Config { return f.cfg }

func TestGenerateHelpers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Generate helpers Suite")
}

var _ = Describe("generate: validate", func() {
	var (
		kbc         *utils.TestContext
		projectFile string
	)

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext("kubebuilder", "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())
		projectFile = filepath.Join(kbc.Dir, yaml.DefaultPath)
	})

	AfterEach(func() {
		By("cleaning up test artifacts")
		kbc.Destroy()
	})

	// Validate
	Context("Validate", func() {
		Context("Success", func() {
			var oldPath, fakePath string
			BeforeEach(func() {
				Expect(os.WriteFile(projectFile, []byte(":?!"), 0o644)).To(Succeed())
				home, err := os.UserHomeDir()
				Expect(err).ToNot(HaveOccurred())

				oldPath = os.Getenv("PATH")
				fakePath = filepath.Join(home, "fake-kubebuilder")
				Expect(os.MkdirAll(fakePath, 0o755)).To(Succeed())

				fakeKubebuilder := filepath.Join(fakePath, "kubebuilder")
				content := "#!/bin/bash\necho 'Fake kubebuilder works!'\n"
				Expect(os.WriteFile(fakeKubebuilder, []byte(content), 0o755)).To(Succeed())

				// Create a sample `sh` file in the fake PATH
				fakeSh := filepath.Join(fakePath, "sh")
				shContent := "#!/bin/bash\necho 'Fake sh works!'\n"
				Expect(os.WriteFile(fakeSh, []byte(shContent), 0o755)).To(Succeed())

				Expect(os.Setenv("PATH", fakePath)).To(Succeed())
			})
			AfterEach(func() {
				Expect(os.RemoveAll(fakePath)).To(Succeed())
				Expect(os.Setenv("PATH", oldPath)).To(Succeed())
			})

			It("success", func() {
				g := &Generate{InputDir: kbc.Dir}
				Expect(g.Validate()).To(Succeed())
			})
		})

		Context("failure", func() {
			var oldPath string
			BeforeEach(func() {
				Expect(os.WriteFile(projectFile, []byte(":?!"), 0o644)).To(Succeed())
				oldPath = os.Getenv("PATH")
				Expect(os.Setenv("PATH", "")).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				Expect(os.Setenv("PATH", oldPath)).To(Succeed())
			})

			It("returns error if GetInputPath fails", func() {
				g := &Generate{InputDir: filepath.Join(kbc.Dir, "notfound")}
				Expect(g.Validate()).NotTo(Succeed())
			})

			It("returns error if kubebuilder not found", func() {
				g := &Generate{InputDir: kbc.Dir}
				Expect(g.Validate()).NotTo(Succeed())
			})
		})
	})
})

var _ = Describe("generate: directory-helpers", func() {
	// createDirectory
	Context("createDirectory", func() {
		var dir string
		BeforeEach(func() {
			tempDir := os.TempDir()
			dir = filepath.Join(tempDir, "testdir-generate-go")
		})

		AfterEach(func() {
			Expect(os.RemoveAll(dir)).To(Succeed())
		})

		It("creates directory successfully", func() {
			Expect(createDirectory(dir)).To(Succeed())
			_, err := os.Stat(dir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error for invalid path", func() {
			Expect(createDirectory("/dev/null/foo")).NotTo(Succeed())
		})
	})

	// changeWorkingDirectory
	Context("changeWorkingDirectory", func() {
		var tempDir string
		var cwd string
		BeforeEach(func() {
			var err error
			tempDir = os.TempDir()
			cwd, err = os.Getwd()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.Chdir(cwd)).To(Succeed())
		})

		It("changes directory successfully", func() {
			Expect(changeWorkingDirectory(tempDir)).To(Succeed())
		})

		It("returns error for invalid path", func() {
			Expect(changeWorkingDirectory("/dev/null/foo")).NotTo(Succeed())
		})
	})
})

var _ = Describe("generate: file-helpers", func() {
	// copyFile
	Context("copyFile", func() {
		Context("success", func() {
			var src, dst string
			BeforeEach(func() {
				tempDir := os.TempDir()
				src = filepath.Join(tempDir, "src.txt")
				dst = filepath.Join(tempDir, "dst.txt")
				Expect(os.WriteFile(src, []byte("hello"), 0o644)).To(Succeed())
			})
			AfterEach(func() {
				Expect(os.Remove(src)).To(Succeed())
				Expect(os.Remove(dst)).To(Succeed())
			})

			It("copies file successfully", func() {
				Expect(copyFile(src, dst)).To(Succeed())
				b, err := os.ReadFile(dst)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(b)).To(Equal("hello"))
			})
		})

		Context("failure", func() {
			It("returns error if src does not exist", func() {
				tempDir := os.TempDir()
				src := filepath.Join(tempDir, "notfound")
				dst := filepath.Join(tempDir, "nowhere")
				Expect(copyFile(src, dst)).NotTo(Succeed())
			})
		})
	})
})

var _ = Describe("generate: get-args-helpers", func() {
	// getInitArgs
	Describe("getInitArgs", func() {
		Context("for outdated plugins", func() {
			It("should return correct args for plugins, domain, repo", func() {
				cfg := &fakeConfig{pluginChain: []string{"go.kubebuilder.io/v3"}, domain: "foo.com", repo: "bar"}
				store := &fakeStore{cfg: cfg}
				args := getInitArgs(store)
				Expect(args).To(ContainElements("--plugins", ContainSubstring("go.kubebuilder.io/v4"),
					"--domain", "foo.com", "--repo", "bar"))
			})
			It("should return correct args for plugins, domain, repo", func() {
				cfg := &fakeConfig{pluginChain: []string{"go.kubebuilder.io/v3-alpha"}, domain: "foo.com", repo: "bar"}
				store := &fakeStore{cfg: cfg}
				args := getInitArgs(store)
				Expect(args).To(ContainElements("--plugins", ContainSubstring("go.kubebuilder.io/v4"),
					"--domain", "foo.com", "--repo", "bar"))
			})
		})
		Context("for latest plugins", func() {
			It("returns correct args for plugins, domain, repo", func() {
				cfg := &fakeConfig{pluginChain: []string{"go.kubebuilder.io/v4"}, domain: "foo.com", repo: "bar"}
				store := &fakeStore{cfg: cfg}
				args := getInitArgs(store)
				Expect(args).To(ContainElements("--plugins", ContainSubstring("go.kubebuilder.io/v4"),
					"--domain", "foo.com", "--repo", "bar"))
			})
		})
	})

	// getGVKFlags
	Context("getGVKFlags", func() {
		It("returns correct flags", func() {
			res := resource.Resource{Plural: "foos"}
			res.Group = "example.com"
			res.Version = "v1"
			res.Kind = "Foo"
			flags := getGVKFlags(res)
			Expect(flags).To(ContainElements("--plural", "foos", "--group", "example.com", "--version", "v1", "--kind", "Foo"))
		})
	})

	// getGVKFlagsFromDeployImage
	Context("getGVKFlagsFromDeployImage", func() {
		It("returns correct flags", func() {
			rd := v1alpha1.ResourceData{Group: "example.com", Version: "v1", Kind: "Foo"}
			flags := getGVKFlagsFromDeployImage(rd)
			Expect(flags).To(ContainElements("--group", "example.com", "--version", "v1", "--kind", "Foo"))
		})
	})

	// getDeployImageOptions
	Context("getDeployImageOptions", func() {
		It("returns correct options", func() {
			rd := v1alpha1.ResourceData{}
			rd.Options.Image = "test-kubebuilder"
			rd.Options.ContainerCommand = "echo 'Hello'"
			rd.Options.ContainerPort = "8000"
			rd.Options.RunAsUser = "test"
			opts := getDeployImageOptions(rd)
			Expect(opts).To(ContainElements("--image=test-kubebuilder",
				"--image-container-command=echo 'Hello'",
				"--image-container-port=8000",
				"--run-as-user=test",
				"--plugins=deploy-image.go.kubebuilder.io/v1-alpha"))
		})
	})

	// getAPIResourceFlags
	Context("getAPIResourceFlags", func() {
		var res resource.Resource
		BeforeEach(func() {
			res = resource.Resource{API: &resource.API{}}
		})
		Context("returns correct flags", func() {
			It("for nil API with Controller set", func() {
				res.Controller = true
				Expect(getAPIResourceFlags(res)).To(ContainElements("--resource=false", "--controller"))
			})
			It("for non nil API (namespaced not set) with Controller not set", func() {
				res.API.CRDVersion = "v1"
				res.API.Namespaced = true
				Expect(getAPIResourceFlags(res)).To(ContainElements("--resource", "--namespaced", "--controller=false"))
			})
			It("for non nil API (namespaced set) with Controller not set", func() {
				res.API.CRDVersion = "v1"
				res.API.Namespaced = false
				Expect(getAPIResourceFlags(res)).To(ContainElements("--resource", "--namespaced=false", "--controller=false"))
			})
		})
	})

	// getWebhookResourceFlags
	Context("getWebhookResourceFlags", func() {
		It("returns correct flags for specified resources", func() {
			res := resource.Resource{
				Path:     "external/test",
				GVK:      resource.GVK{Group: "example.com", Version: "v1", Kind: "Example", Domain: "test"},
				External: true,
				Webhooks: &resource.Webhooks{
					Validation: true,
					Defaulting: true,
					Conversion: true,
					Spoke:      []string{"v2"},
				},
			}
			flags := getWebhookResourceFlags(res)
			Expect(flags).To(ContainElements("--external-api-path", "external/test", "--external-api-domain", "test",
				"--programmatic-validation", "--defaulting", "--conversion", "--spoke", "v2"))
		})
	})
})

var _ = Describe("generate: create-helpers", func() {
	var fakePath, oldPath string

	BeforeEach(func() {
		home, err := os.UserHomeDir()
		Expect(err).ToNot(HaveOccurred())
		fakePath = filepath.Join(home, "fake-kubebuilder")
		Expect(os.MkdirAll(fakePath, 0o755)).To(Succeed())

		fakeKubebuilder := filepath.Join(fakePath, "kubebuilder")
		content := "#!/bin/bash\necho 'Fake kubebuilder create successful!'\n"
		Expect(os.WriteFile(fakeKubebuilder, []byte(content), 0o755)).To(Succeed())

		oldPath = os.Getenv("PATH")
		Expect(os.Setenv("PATH", fakePath)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(fakePath)).To(Succeed())
		Expect(os.Setenv("PATH", oldPath)).To(Succeed())
	})

	// createAPI
	Describe("createAPI", func() {
		Context("Without External flag", func() {
			It("runs kubebuilder create api successfully for a resource", func() {
				res := resource.Resource{
					GVK:        resource.GVK{Group: "example.com", Version: "v1", Kind: "Example", Domain: "test"},
					Plural:     "examples",
					API:        &resource.API{Namespaced: true},
					Controller: true,
				}

				// Run createAPI and verify no errors
				Expect(createAPI(res)).To(Succeed())
			})
		})

		Context("Without External flag set", func() {
			It("runs kubebuilder create api successfully for a resource", func() {
				res := resource.Resource{
					GVK:        resource.GVK{Group: "example.com", Version: "v1", Kind: "Example", Domain: "external"},
					Plural:     "examples",
					API:        &resource.API{Namespaced: true},
					Controller: true,
					External:   true,
					Path:       "external/path",
				}

				// Run createAPI and verify no errors
				Expect(createAPI(res)).To(Succeed())
			})
		})
	})

	// createWebhook
	Describe("createWebhook", func() {
		It("runs kubebuilder create webhook successfully for a resource", func() {
			res := resource.Resource{
				GVK:      resource.GVK{Group: "example.com", Version: "v1", Kind: "Example", Domain: "test"},
				Plural:   "examples",
				Webhooks: &resource.Webhooks{WebhookVersion: "v1"},
			}

			// Run createWebhook and verify no errors
			Expect(createWebhook(res)).To(Succeed())
		})

		It("ignores web creation if webhook resource is empty", func() {
			res := resource.Resource{
				GVK:      resource.GVK{Group: "example.com", Version: "v1", Kind: "Example", Domain: "test"},
				Plural:   "examples",
				Webhooks: &resource.Webhooks{},
			}

			// Run createWebhook and verify no errors
			Expect(createWebhook(res)).To(Succeed())
		})
	})

	Describe("createAPIWithDeployImage", func() {
		It("runs kubebuilder create api successfully with deploy image", func() {
			resourceData := v1alpha1.ResourceData{
				Group:   "example.com",
				Version: "v1",
				Kind:    "Example",
			}
			resourceData.Options.Image = "example-image"
			resourceData.Options.ContainerCommand = "run"
			resourceData.Options.ContainerPort = "8080"
			resourceData.Options.Image = "test"

			// Run createAPIWithDeployImage and verify no errors
			Expect(createAPIWithDeployImage(resourceData)).To(Succeed())
		})
	})
})

var _ = Describe("generate: kubebuilder", func() {
	var fakePath, oldPath string

	BeforeEach(func() {
		// Create a fake kubebuilder binary in the PATH
		home, err := os.UserHomeDir()
		Expect(err).ToNot(HaveOccurred())
		fakePath = filepath.Join(home, "fake-kubebuilder")
		Expect(os.MkdirAll(fakePath, 0o755)).To(Succeed())

		fakeKubebuilder := filepath.Join(fakePath, "kubebuilder")
		content := "#!/bin/bash\necho 'Fake kubebuilder edit successful!'\n"
		Expect(os.WriteFile(fakeKubebuilder, []byte(content), 0o755)).To(Succeed())

		oldPath = os.Getenv("PATH")
		Expect(os.Setenv("PATH", fakePath)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(fakePath)).To(Succeed())
		Expect(os.Setenv("PATH", oldPath)).To(Succeed())
	})

	Context("kubebuilderInit", func() {
		var tempDir string
		BeforeEach(func() {
			// Create a temporary directory for the test
			tempDir = filepath.Join(os.TempDir(), "kubebuilder-init-test")
			Expect(createDirectory(tempDir)).To(Succeed())
			Expect(changeWorkingDirectory(tempDir)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})

		It("runs kubebuilder init successfully", func() {
			cfg := &fakeConfig{
				pluginChain: []string{"go.kubebuilder.io/v4"}, domain: "example.com",
				repo: "github.com/example/repo",
			}
			store := &fakeStore{cfg: cfg}

			// Run kubebuilderInit and verify no errors
			Expect(kubebuilderInit(store)).To(Succeed())
		})
	})

	Context("kubebuilderCreate", func() {
		var tempDir string
		BeforeEach(func() {
			// Create a temporary directory for the test
			tempDir = filepath.Join(os.TempDir(), "kubebuilder-create-test")
			Expect(createDirectory(tempDir)).To(Succeed())
			Expect(changeWorkingDirectory(tempDir)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})

		It("runs kubebuilder create successfully for resources", func() {
			cfg := &fakeConfig{
				resources: []resource.Resource{
					{Plural: "foos", GVK: resource.GVK{Group: "example.com", Version: "v1", Kind: "Foo"}},
					{Plural: "bars", GVK: resource.GVK{Group: "example.com", Version: "v1", Kind: "Bar"}},
				},
			}
			store := &fakeStore{cfg: cfg}

			// Run kubebuilderCreate and verify no errors
			Expect(kubebuilderCreate(store)).To(Succeed())
		})
	})

	Context("kubebuilderEdit", func() {
		var tempDir string
		BeforeEach(func() {
			// Create a temporary directory for the test
			tempDir = filepath.Join(os.TempDir(), "kubebuilder-edit-test")
			Expect(createDirectory(tempDir)).To(Succeed())
			Expect(changeWorkingDirectory(tempDir)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})

		It("runs kubebuilder edit successfully for multigroup layout", func() {
			cfg := &fakeConfig{multigroup: true}
			store := &fakeStore{cfg: cfg}

			// Run kubebuilderEdit and verify no errors
			Expect(kubebuilderEdit(store)).To(Succeed())
		})
	})

	Context("kubebuilderGrafanaEdit", func() {
		It("runs kubebuilder edit successfully for Grafana plugin", func() {
			// Run kubebuilderGrafanaEdit and verify no errors
			Expect(kubebuilderGrafanaEdit()).To(Succeed())
		})
	})

	Context("kubebuilderHelmEdit", func() {
		It("runs kubebuilder edit successfully for Helm plugin", func() {
			// Run kubebuilderHelmEdit and verify no errors
			Expect(kubebuilderHelmEdit()).To(Succeed())
		})
	})
})

var _ = Describe("generate: hasHelmPlugin", func() {
	It("returns true if plugin present", func() {
		cfg := &fakeConfig{plugins: map[string]any{"helm.kubebuilder.io/v1-alpha": true}}
		store := &fakeStore{cfg: cfg}
		Expect(hasHelmPlugin(store)).To(BeTrue())
	})
	It("returns false if plugin not found", func() {
		cfg := &fakeConfig{pluginErr: &config.PluginKeyNotFoundError{Key: "helm.kubebuilder.io/v1-alpha"}}
		store := &fakeStore{cfg: cfg}
		Expect(hasHelmPlugin(store)).To(BeFalse())
	})
})

var _ = Describe("generate: migrate-plugins", func() {
	var fakePath, oldPath string

	BeforeEach(func() {
		// Create a fake kubebuilder binary in the PATH
		home, err := os.UserHomeDir()
		Expect(err).ToNot(HaveOccurred())
		fakePath = filepath.Join(home, "fake-kubebuilder")
		Expect(os.MkdirAll(fakePath, 0o755)).To(Succeed())

		fakeKubebuilder := filepath.Join(fakePath, "kubebuilder")
		content := "#!/bin/bash\necho 'Fake kubebuilder edit successful!'\n"
		Expect(os.WriteFile(fakeKubebuilder, []byte(content), 0o755)).To(Succeed())

		oldPath = os.Getenv("PATH")
		Expect(os.Setenv("PATH", fakePath)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(fakePath)).To(Succeed())
		Expect(os.Setenv("PATH", oldPath)).To(Succeed())
	})

	Context("migrateGrafanaPlugin", func() {
		It("skips migration as Grafana plugin not found", func() {
			cfg := &fakeConfig{pluginErr: &config.PluginKeyNotFoundError{Key: "grafana.kubebuilder.io/v1-alpha"}}
			store := &fakeStore{cfg: cfg}
			Expect(migrateGrafanaPlugin(store, "src", "dest")).To(Succeed())
		})

		It("returns error if decoding Grafana plugin config fails", func() {
			cfg := &fakeConfig{
				pluginErr: fmt.Errorf("decoding error"),
				plugins:   map[string]any{"grafana.kubebuilder.io/v1-alpha": true},
			}
			store := &fakeStore{cfg: cfg}
			Expect(migrateGrafanaPlugin(store, "src", "dest")).NotTo(Succeed())
		})

		Context("success", func() {
			var src, dest string
			BeforeEach(func() {
				// Create temporary directories for src and dest
				src = filepath.Join(os.TempDir(), "src")
				dest = filepath.Join(os.TempDir(), "dest")

				Expect(os.MkdirAll(filepath.Join(src, "grafana/custom-metrics"), 0o755)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(src, "grafana/custom-metrics/config.yaml"),
					[]byte("config"), 0o755)).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(dest, "grafana/custom-metrics"), 0o755)).To(Succeed())
			})
			AfterEach(func() {
				Expect(os.RemoveAll(src)).To(Succeed())
				Expect(os.RemoveAll(dest)).To(Succeed())
			})

			It("migrates Grafana plugin successfully", func() {
				cfg := &fakeConfig{plugins: map[string]any{"grafana.kubebuilder.io/v1-alpha": true}}
				store := &fakeStore{cfg: cfg}

				Expect(migrateGrafanaPlugin(store, src, dest)).To(Succeed())
				b, err := os.ReadFile(filepath.Join(dest, "grafana/custom-metrics/config.yaml"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(b)).To(Equal("config"))
			})
		})
	})

	Context("migrateAutoUpdatePlugin", func() {
		It("skips migration as AutoUpdate plugin not found", func() {
			cfg := &fakeConfig{pluginErr: &config.PluginKeyNotFoundError{Key: "autoupdate.kubebuilder.io/v1-alpha"}}
			store := &fakeStore{cfg: cfg}
			Expect(migrateGrafanaPlugin(store, "src", "dest")).To(Succeed())
		})

		It("returns error if failed to decode Auto Update plugin", func() {
			cfg := &fakeConfig{
				pluginErr: fmt.Errorf("decoding error"),
				plugins:   map[string]any{"autoupdate.kubebuilder.io/v1-alpha": true},
			}
			store := &fakeStore{cfg: cfg}
			Expect(migrateAutoUpdatePlugin(store)).NotTo(Succeed())
		})

		It("migrates Auto Update plugin successfully", func() {
			cfg := &fakeConfig{plugins: map[string]any{"autoupdate.kubebuilder.io/v1-alpha": true}}
			store := &fakeStore{cfg: cfg}
			Expect(migrateAutoUpdatePlugin(store)).To(Succeed())
		})
	})

	Context("migrateDeployImagePlugin", func() {
		It("returns error if failed to decode Deploy Image plugin", func() {
			cfg := &fakeConfig{pluginErr: &config.PluginKeyNotFoundError{Key: "deploy-image.kubebuilder.io/v1-alpha"}}
			store := &fakeStore{cfg: cfg}
			Expect(migrateDeployImagePlugin(store)).To(Succeed())
		})

		It("returns error if decoding Deploy Image plugin config fails", func() {
			cfg := &fakeConfig{
				pluginErr: fmt.Errorf("decoding error"),
				plugins:   map[string]any{"deploy-image.kubebuilder.io/v1-alpha": true},
			}
			store := &fakeStore{cfg: cfg}
			Expect(migrateDeployImagePlugin(store)).NotTo(Succeed())
		})

		It("migrates Deploy Image plugin successfully", func() {
			cfg := &fakeConfig{plugins: map[string]any{"deploy-image.kubebuilder.io/v1-alpha": true}}
			store := &fakeStore{cfg: cfg}

			// Mock resources for the plugin
			resources := []v1alpha1.ResourceData{
				{
					Group:   "example.com",
					Version: "v1",
					Kind:    "Example",
				},
			}
			cfg.pluginChain = []string{"deploy-image.kubebuilder.io/v1-alpha"}
			store.cfg = cfg

			// Use the mocked resources
			for _, r := range resources {
				Expect(createAPIWithDeployImage(r)).To(Succeed())
			}

			Expect(migrateDeployImagePlugin(store)).To(Succeed())
		})
	})
})

var _ = Describe("Generate", func() {
	var (
		kbc         *utils.TestContext
		projectFile string
		oldPath     string
		fakePath    string
		g           *Generate
	)

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext("kubebuilder", "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())
		projectFile = filepath.Join(kbc.Dir, yaml.DefaultPath)

		// Set up a fake kubebuilder binary
		home, err := os.UserHomeDir()
		Expect(err).ToNot(HaveOccurred())
		oldPath = os.Getenv("PATH")
		fakePath = filepath.Join(home, "fake-kubebuilder")
		Expect(os.MkdirAll(fakePath, 0o755)).To(Succeed())
		fakeKubebuilder := filepath.Join(fakePath, "kubebuilder")
		content := "#!/bin/bash\necho 'Fake kubebuilder create successful!'\n"
		Expect(os.WriteFile(fakeKubebuilder, []byte(content), 0o755)).To(Succeed())

		// Create a sample `sh` file in the fake PATH
		fakeSh := filepath.Join(fakePath, "sh")
		shContent := "#!/bin/bash\necho 'Fake sh works!'\n"
		Expect(os.WriteFile(fakeSh, []byte(shContent), 0o755)).To(Succeed())

		Expect(os.Setenv("PATH", fakePath)).To(Succeed())

		// Register Version 3 config
		config.Register(config.Version{Number: 3}, func() config.Config {
			return &v3.Cfg{Version: config.Version{Number: 3}}
		})

		// Create a project file with version 3
		const version = `version: "3"
`
		Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())

		// initialize Generate
		g = &Generate{InputDir: kbc.Dir}
	})

	AfterEach(func() {
		By("cleaning up test artifacts")
		kbc.Destroy()
		Expect(os.RemoveAll(fakePath)).To(Succeed())
		Expect(os.Setenv("PATH", oldPath)).To(Succeed())
	})

	Context("outputDir is non empty", func() {
		It("scaffolds the project in output dir", func() {
			g.OutputDir = kbc.Dir
			Expect(g.Generate()).To(Succeed())
		})
	})

	Context("outputDir is empty", func() {
		It("re-scaffolds the project in input dir", func() {
			Expect(g.Generate()).To(Succeed())
		})
	})
})
