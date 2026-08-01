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

package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	yamlstore "sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	golangv4 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4"
)

const (
	createSubcommand  = "create"
	apiSubcommand     = "api"
	webhookSubcommand = "webhook"
	editSubcommand    = "edit"
	kindFlagArg       = "--kind"
	kindValue         = "Captain"
)

func makeMockPluginsFor(projectVersion config.Version, pluginKeys ...string) []plugin.Plugin {
	plugins := make([]plugin.Plugin, 0, len(pluginKeys))
	for _, key := range pluginKeys {
		n, v := plugin.SplitKey(key)
		plugins = append(plugins, newMockPlugin(n, v, projectVersion))
	}
	return plugins
}

func makeMapFor(plugins ...plugin.Plugin) map[string]plugin.Plugin {
	pluginMap := make(map[string]plugin.Plugin, len(plugins))
	for _, p := range plugins {
		pluginMap[plugin.KeyFor(p)] = p
	}
	return pluginMap
}

func setFlag(flag, value string) {
	os.Args = append(os.Args, "subcommand", "--"+flag, value)
}

func setBoolFlag(flag string) {
	os.Args = append(os.Args, "subcommand", "--"+flag)
}

func setProjectVersionFlag(value string) {
	setFlag(projectVersionFlag, value)
}

func setPluginsFlag(value string) {
	setFlag(pluginsFlag, value)
}

// parseFlagsFromArgs resolves the flags of the CLI from the arguments of the running program, the
// way New does when it builds the CLI.
func parseFlagsFromArgs(c *CLI) error {
	c.args = defaultArgs()

	return c.getInfoFromFlags(false)
}

func hasSubCommand(cmd *cobra.Command, name string) bool {
	for _, subcommand := range cmd.Commands() {
		if subcommand.Name() == name {
			return true
		}
	}
	return false
}

// filesystemWithUnloadableProject returns a filesystem holding a PROJECT file that cannot be loaded
// because its version is not supported.
func filesystemWithUnloadableProject() machinery.Filesystem {
	return filesystemWithProject(`version: "1"
`)
}

// filesystemWithInvalidProject returns a filesystem holding a PROJECT file that is not valid YAML.
func filesystemWithInvalidProject() machinery.Filesystem {
	return filesystemWithProject(`{ version: "3"
`)
}

// filesystemWithUnresolvableProject returns a filesystem holding a valid PROJECT file whose plugin
// chain cannot be resolved.
func filesystemWithUnresolvableProject() machinery.Filesystem {
	return filesystemWithProject(`domain: example.com
layout:
- gone.example.com/v1
projectName: test
version: "3"
`)
}

// filesystemWithResolvableProject returns a filesystem holding a valid PROJECT file whose plugin
// chain resolves to the go/v4 plugin.
func filesystemWithResolvableProject() machinery.Filesystem {
	return filesystemWithProject(`domain: example.com
layout:
- base.go.kubebuilder.io/v4
projectName: test
version: "3"
`)
}

func filesystemWithProject(content string) machinery.Filesystem {
	fs := filesystemWithoutProject()
	Expect(afero.WriteFile(fs.FS, yamlstore.DefaultPath, []byte(content), machinery.DefaultFilePermission)).To(Succeed())

	return fs
}

func filesystemWithoutProject() machinery.Filesystem {
	return machinery.Filesystem{FS: afero.NewBasePathFs(afero.NewOsFs(), GinkgoT().TempDir())}
}

// filesystemWithProjectDirectory returns a filesystem where PROJECT is a directory, which holds no
// configuration and must be handled as a project that was never initialized.
func filesystemWithProjectDirectory() machinery.Filesystem {
	fs := filesystemWithoutProject()
	Expect(fs.FS.Mkdir(yamlstore.DefaultPath, machinery.DefaultDirectoryPermission)).To(Succeed())

	return fs
}

// filesystemWithDanglingProjectSymlink returns a filesystem where PROJECT is a symbolic link whose
// target does not exist, along with the path that the link points to.
func filesystemWithDanglingProjectSymlink() (machinery.Filesystem, string) {
	GinkgoHelper()

	dir := GinkgoT().TempDir()
	target := filepath.Join(dir, "stolen.yaml")
	Expect(os.Symlink(target, filepath.Join(dir, yamlstore.DefaultPath))).To(Succeed())

	return machinery.Filesystem{FS: afero.NewBasePathFs(afero.NewOsFs(), dir)}, target
}

// skipWithoutSymlinks skips the test where creating a symbolic link needs extra privileges.
func skipWithoutSymlinks() {
	GinkgoHelper()

	if runtime.GOOS == "windows" {
		Skip("symlink creation requires elevated privileges on Windows")
	}
}

// expectNothingScaffolded asserts that the filesystem holds nothing besides the PROJECT path.
func expectNothingScaffolded(fs machinery.Filesystem) {
	GinkgoHelper()

	entries, err := afero.ReadDir(fs.FS, ".")
	Expect(err).NotTo(HaveOccurred())
	Expect(entries).To(HaveLen(1))
	Expect(entries[0].Name()).To(Equal(yamlstore.DefaultPath))
}

// projectReadCountingFs counts how many times the project configuration file is opened.
type projectReadCountingFs struct {
	afero.Fs
	reads int
}

func (f *projectReadCountingFs) Open(name string) (afero.File, error) {
	if name == yamlstore.DefaultPath {
		f.reads++
	}

	file, err := f.Fs.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %w", name, err)
	}

	return file, nil
}

// executeCLI builds a CLI over the given filesystem and runs it with the given arguments, as the
// binary does when invoked from a project directory.
func executeCLI(filesystem machinery.Filesystem, args ...string) error {
	_, err := runCLI(filesystem, args, args)

	return err
}

// executeEmbeddedCLI builds a CLI while the program was invoked with programArgs, and runs it with
// commandArgs, as an embedded CLI driven by Command().SetArgs does.
func executeEmbeddedCLI(
	filesystem machinery.Filesystem,
	programArgs, commandArgs []string,
	options ...Option,
) error {
	_, err := runCLI(filesystem, programArgs, commandArgs, options...)

	return err
}

// completeCLI runs the hidden command that Cobra uses to complete a command line, and returns what
// it offers.
func completeCLI(filesystem machinery.Filesystem, args ...string) (string, error) {
	completionArgs := append([]string{cobra.ShellCompRequestCmd}, args...)

	return runCLI(filesystem, completionArgs, completionArgs)
}

// runCLI builds a CLI while the program was invoked with programArgs, runs it with commandArgs, and
// returns what the command wrote.
func runCLI(
	filesystem machinery.Filesystem,
	programArgs, commandArgs []string,
	options ...Option,
) (string, error) {
	GinkgoHelper()

	originalArgs := os.Args
	DeferCleanup(func() { os.Args = originalArgs })
	os.Args = append([]string{kubebuilderCommandName}, programArgs...)

	testProjectVersion := config.Version{Number: 3}
	c, err := New(append([]Option{
		WithPlugins(&golangv4.Plugin{}),
		WithDefaultPlugins(testProjectVersion, &golangv4.Plugin{}),
		WithDefaultProjectVersion(testProjectVersion),
		WithVersion("version string"),
		WithCompletion(),
		WithFilesystem(filesystem),
	}, options...)...)
	Expect(err).NotTo(HaveOccurred())

	var out bytes.Buffer
	c.cmd.SetOut(&out)
	// Cobra falls back to the arguments of the running program when they are set to nil.
	if commandArgs == nil {
		commandArgs = []string{}
	}
	c.cmd.SetArgs(commandArgs)

	if err := c.cmd.Execute(); err != nil {
		return out.String(), fmt.Errorf("failed to execute the command: %w", err)
	}

	return out.String(), nil
}

type pluginChainCapturingSubcommand struct {
	pluginChain []string
}

func (s *pluginChainCapturingSubcommand) Scaffold(machinery.Filesystem) error {
	return nil
}

func (s *pluginChainCapturingSubcommand) SetPluginChain(chain []string) {
	s.pluginChain = append([]string(nil), chain...)
}

type testCreateAPIPlugin struct {
	name        string
	version     plugin.Version
	subcommand  *testCreateAPISubcommand
	projectVers []config.Version
}

func newTestCreateAPIPlugin(name string, version plugin.Version) testCreateAPIPlugin {
	return testCreateAPIPlugin{
		name:        name,
		version:     version,
		subcommand:  &testCreateAPISubcommand{},
		projectVers: []config.Version{{Number: 3}},
	}
}

func (p testCreateAPIPlugin) Name() string                               { return p.name }
func (p testCreateAPIPlugin) Version() plugin.Version                    { return p.version }
func (p testCreateAPIPlugin) SupportedProjectVersions() []config.Version { return p.projectVers }
func (p testCreateAPIPlugin) GetCreateAPISubcommand() plugin.CreateAPISubcommand {
	return p.subcommand
}

type testCreateAPISubcommand struct{}

func (s *testCreateAPISubcommand) InjectResource(*resource.Resource) error {
	return nil
}

func (s *testCreateAPISubcommand) Scaffold(machinery.Filesystem) error {
	return nil
}

type fakeStore struct {
	cfg config.Config
}

func (f *fakeStore) New(config.Version) error { return nil }
func (f *fakeStore) Load() error              { return nil }
func (f *fakeStore) LoadFrom(string) error    { return nil }
func (f *fakeStore) Save() error              { return nil }
func (f *fakeStore) SaveTo(string) error      { return nil }
func (f *fakeStore) Config() config.Config    { return f.cfg }

type captureSubcommand struct {
	lastChain []string
}

func (c *captureSubcommand) Scaffold(machinery.Filesystem) error { return nil }

var _ = Describe("CLI", func() {
	const (
		pluginGoV1           = "go/v1"
		pluginExampleV2      = "example/v2"
		pluginTestV1         = "test/v1"
		deployImageFooPlugin = "deploy-image.foo.example.com/v1-alpha"
		deployImageBarPlugin = "deploy-image.bar.example.com/v1-alpha"
		subcommandExtra      = "extra"
	)

	var (
		c              *CLI
		projectVersion config.Version
	)

	BeforeEach(func() {
		c = &CLI{
			fs: machinery.Filesystem{FS: afero.NewMemMapFs()},
		}

		projectVersion = config.Version{Number: 3}
	})

	Describe("filterSubcommands", func() {
		It("propagates bundle keys to wrapped subcommands", func() {
			bundleVersion := plugin.Version{Number: 1, Stage: stage.Alpha}

			fooPlugin := newTestCreateAPIPlugin("deploy-image.go.kubebuilder.io", plugin.Version{Number: 1, Stage: stage.Alpha})
			barPlugin := newTestCreateAPIPlugin("deploy-image.go.kubebuilder.io", plugin.Version{Number: 1, Stage: stage.Alpha})

			fooBundle, err := plugin.NewBundleWithOptions(
				plugin.WithName("deploy-image.foo.example.com"),
				plugin.WithVersion(bundleVersion),
				plugin.WithPlugins(fooPlugin),
			)
			Expect(err).NotTo(HaveOccurred())

			barBundle, err := plugin.NewBundleWithOptions(
				plugin.WithName("deploy-image.bar.example.com"),
				plugin.WithVersion(bundleVersion),
				plugin.WithPlugins(barPlugin),
			)
			Expect(err).NotTo(HaveOccurred())

			c.resolvedPlugins = []plugin.Plugin{fooBundle, barBundle}

			tuples := c.filterSubcommands(
				func(p plugin.Plugin) bool {
					_, isCreateAPI := p.(plugin.CreateAPI)
					return isCreateAPI
				},
				func(p plugin.Plugin) plugin.Subcommand {
					return p.(plugin.CreateAPI).GetCreateAPISubcommand()
				},
			)

			Expect(tuples).To(HaveLen(2))
			Expect(tuples[0].key).To(Equal("deploy-image.go.kubebuilder.io/v1-alpha"))
			Expect(tuples[0].configKey).To(Equal(deployImageFooPlugin))
			Expect(tuples[1].key).To(Equal("deploy-image.go.kubebuilder.io/v1-alpha"))
			Expect(tuples[1].configKey).To(Equal(deployImageBarPlugin))
		})
	})

	Describe("executionHooksFactory", func() {
		It("temporarily reorders the plugin chain while invoking bundled subcommands", func() {
			cfg := cfgv3.New()
			Expect(cfg.SetPluginChain([]string{
				deployImageFooPlugin,
				deployImageBarPlugin,
			})).To(Succeed())

			store := &fakeStore{cfg: cfg}
			first := &captureSubcommand{}
			second := &captureSubcommand{}

			factory := executionHooksFactory{
				store: store,
				subcommands: []keySubcommandTuple{
					{configKey: deployImageFooPlugin, subcommand: first},
					{configKey: deployImageBarPlugin, subcommand: second},
				},
				errorMessage: "test",
			}

			callErr := factory.forEach(func(sub plugin.Subcommand) error {
				cs := sub.(*captureSubcommand)
				cs.lastChain = append([]string(nil), store.Config().GetPluginChain()...)
				return nil
			}, "scaffold")
			Expect(callErr).NotTo(HaveOccurred())
			Expect(first.lastChain[0]).To(Equal(deployImageFooPlugin))
			Expect(second.lastChain[0]).To(Equal(deployImageBarPlugin))
			Expect(store.Config().GetPluginChain()).To(Equal([]string{
				deployImageFooPlugin,
				deployImageBarPlugin,
			}))
		})
	})

	Context("buildCmd", func() {
		var projectFile string

		BeforeEach(func() {
			projectFile = `domain: zeusville.com
layout: go.kubebuilder.io/v3
projectName: demo-zeus-operator
repo: github.com/jmrodri/demo-zeus-operator
resources:
- crdVersion: v1
  group: test
  kind: Test
  version: v1
version: 3-alpha
plugins:
  manifests.sdk.operatorframework.io/v2: {}
`
			f, err := c.fs.FS.Create("PROJECT")
			Expect(err).To(Not(HaveOccurred()))

			_, err = f.WriteString(projectFile)
			Expect(err).To(Not(HaveOccurred()))

			// A subcommand that consumes the configuration, so that the file is read.
			c.args = []string{kubebuilderSubcommandInit}
		})

		When("reading a 3-alpha config", func() {
			It("should succeed and set the projectVersion", func() {
				c.buildCmd()
				Expect(c.configErr).NotTo(HaveOccurred())
				Expect(c.projectVersion.Compare(
					config.Version{
						Number: 3,
						Stage:  stage.Stable,
					})).To(Equal(0))
			})
			It("should record the failure when stable is not registered", func() {
				// overwrite project file with fake 4-alpha
				f, err := c.fs.FS.OpenFile("PROJECT", os.O_WRONLY, 0)
				Expect(err).To(Not(HaveOccurred()))
				_, err = f.WriteString(strings.ReplaceAll(projectFile, "3-alpha", "4-alpha"))
				Expect(err).To(Not(HaveOccurred()))

				c.buildCmd()
				Expect(c.configErr).To(HaveOccurred())
				Expect(c.cmd.Commands()).NotTo(BeEmpty())
			})
		})
	})

	// TODO: test CLI.getInfoFromConfigFile using a mock filesystem

	Context("getInfoFromConfig", func() {
		When("having a single plugin in the layout field", func() {
			It("should succeed", func() {
				pluginChain := []string{pluginGoKubebuilderV4}
				projectConfig := cfgv3.New()
				Expect(projectConfig.SetPluginChain(pluginChain)).To(Succeed())

				Expect(c.getInfoFromConfig(projectConfig)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginChain))
				Expect(c.projectVersion.Compare(projectConfig.GetVersion())).To(Equal(0))
			})
		})

		When("having multiple plugins in the layout field", func() {
			It("should succeed", func() {
				pluginChain := []string{pluginGoKubebuilderV2, "deploy-image.go.kubebuilder.io/v1-alpha"}

				projectConfig := cfgv3.New()
				Expect(projectConfig.SetPluginChain(pluginChain)).To(Succeed())

				Expect(c.getInfoFromConfig(projectConfig)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginChain))
				Expect(c.projectVersion.Compare(projectConfig.GetVersion())).To(Equal(0))
			})
		})

		When("having invalid plugin keys in the layout field", func() {
			It("should fail", func() {
				pluginChain := []string{"_/v1"}

				projectConfig := cfgv3.New()
				Expect(projectConfig.SetPluginChain(pluginChain)).To(Succeed())

				Expect(c.getInfoFromConfig(projectConfig)).NotTo(Succeed())
			})
		})
	})

	Context("getInfoFromFlags", func() {
		// Save os.Args and restore it for every test
		var args []string
		BeforeEach(func() {
			c.cmd = c.newRootCmd()

			args = os.Args
		})
		AfterEach(func() {
			os.Args = args
		})

		When("no flag is set", func() {
			It("should succeed", func() {
				Expect(parseFlagsFromArgs(c)).To(Succeed())
				Expect(c.pluginKeys).To(BeEmpty())
				Expect(c.projectVersion.Compare(config.Version{})).To(Equal(0))
			})
		})

		When(fmt.Sprintf("--%s flag is set", pluginsFlag), func() {
			It("should succeed using one plugin key", func() {
				pluginKeys := []string{pluginGoV1}
				setPluginsFlag(strings.Join(pluginKeys, ","))

				Expect(parseFlagsFromArgs(c)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginKeys))
				Expect(c.projectVersion.Compare(config.Version{})).To(Equal(0))
			})

			It("should succeed using more than one plugin key", func() {
				pluginKeys := []string{pluginGoV1, pluginExampleV2, pluginTestV1}
				setPluginsFlag(strings.Join(pluginKeys, ","))

				Expect(parseFlagsFromArgs(c)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginKeys))
				Expect(c.projectVersion.Compare(config.Version{})).To(Equal(0))
			})

			It("should succeed using more than one plugin key with spaces", func() {
				pluginKeys := []string{pluginGoV1, pluginExampleV2, pluginTestV1}
				setPluginsFlag(strings.Join(pluginKeys, ", "))

				Expect(parseFlagsFromArgs(c)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginKeys))
				Expect(c.projectVersion.Compare(config.Version{})).To(Equal(0))
			})

			It("should fail for an invalid plugin key", func() {
				setPluginsFlag("_/v1")

				Expect(parseFlagsFromArgs(c)).NotTo(Succeed())
			})
		})

		When(fmt.Sprintf("--%s flag is set", projectVersionFlag), func() {
			It("should succeed", func() {
				setProjectVersionFlag(projectVersion.String())

				Expect(parseFlagsFromArgs(c)).To(Succeed())
				Expect(c.pluginKeys).To(BeEmpty())
				Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
			})

			It("should fail for an invalid project version", func() {
				setProjectVersionFlag("v_1")

				Expect(parseFlagsFromArgs(c)).NotTo(Succeed())
			})
		})

		When(fmt.Sprintf("--%s and --%s flags are set", pluginsFlag, projectVersionFlag), func() {
			It("should succeed using one plugin key", func() {
				pluginKeys := []string{pluginGoV1}
				setPluginsFlag(strings.Join(pluginKeys, ","))
				setProjectVersionFlag(projectVersion.String())

				Expect(parseFlagsFromArgs(c)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginKeys))
				Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
			})

			It("should succeed using more than one plugin key", func() {
				pluginKeys := []string{pluginGoV1, pluginExampleV2, pluginTestV1}
				setPluginsFlag(strings.Join(pluginKeys, ","))
				setProjectVersionFlag(projectVersion.String())

				Expect(parseFlagsFromArgs(c)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginKeys))
				Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
			})

			It("should succeed using more than one plugin key with spaces", func() {
				pluginKeys := []string{pluginGoV1, pluginExampleV2, pluginTestV1}
				setPluginsFlag(strings.Join(pluginKeys, ", "))
				setProjectVersionFlag(projectVersion.String())

				Expect(parseFlagsFromArgs(c)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginKeys))
				Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
			})
		})

		When("additional flags are set", func() {
			It("should succeed", func() {
				setFlag("extra-flag", "extra-value")

				Expect(parseFlagsFromArgs(c)).To(Succeed())
			})

			// `--help` is not captured by the allowlist, so we need to special case it
			It("should not fail for `--help`", func() {
				setBoolFlag("help")

				Expect(parseFlagsFromArgs(c)).To(Succeed())
			})

			// When --plugins is followed by --help, --help is consumed as plugin value
			// This should not trigger plugin validation errors
			It("should not fail when `--plugins --help` is used together", func() {
				os.Args = append(os.Args, "edit", pluginsFlagArg, "--help")

				Expect(parseFlagsFromArgs(c)).To(Succeed())
				Expect(c.pluginKeys).To(BeEmpty())
			})

			// Same test for short help flag
			It("should not fail when `--plugins -h` is used together", func() {
				os.Args = append(os.Args, "edit", pluginsFlagArg, "-h")

				Expect(parseFlagsFromArgs(c)).To(Succeed())
				Expect(c.pluginKeys).To(BeEmpty())
			})

			It("should keep the plugin key when `--plugins=<value> --help` is used together", func() {
				os.Args = append(os.Args, "edit", pluginsFlagArg+"="+pluginGoV1, "--help")

				Expect(parseFlagsFromArgs(c)).To(Succeed())
				Expect(c.pluginKeys).To(Equal([]string{pluginGoV1}))
			})

			It("should keep the plugin key when `--plugins=<value> -h` is used together", func() {
				os.Args = append(os.Args, "edit", pluginsFlagArg+"="+pluginGoV1, "-h")

				Expect(parseFlagsFromArgs(c)).To(Succeed())
				Expect(c.pluginKeys).To(Equal([]string{pluginGoV1}))
			})
		})
	})

	Context("getInfoFromDefaults", func() {
		var pluginKeys []string

		BeforeEach(func() {
			pluginKeys = []string{pluginGoKubebuilderV2}
		})

		It("should be a no-op if already have plugin keys", func() {
			c.pluginKeys = pluginKeys

			c.getInfoFromDefaults()
			Expect(c.pluginKeys).To(Equal(pluginKeys))
			Expect(c.projectVersion.Compare(config.Version{})).To(Equal(0))
		})

		It("should succeed if default plugins for project version are set", func() {
			c.projectVersion = projectVersion
			c.defaultPlugins = map[config.Version][]string{projectVersion: pluginKeys}

			c.getInfoFromDefaults()
			Expect(c.pluginKeys).To(Equal(pluginKeys))
			Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
		})

		It("should succeed if default plugins for default project version are set", func() {
			c.defaultPlugins = map[config.Version][]string{projectVersion: pluginKeys}
			c.defaultProjectVersion = projectVersion

			c.getInfoFromDefaults()
			Expect(c.pluginKeys).To(Equal(pluginKeys))
			Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
		})

		It("should succeed if default plugins for only a single project version are set", func() {
			c.defaultPlugins = map[config.Version][]string{projectVersion: pluginKeys}

			c.getInfoFromDefaults()
			Expect(c.pluginKeys).To(Equal(pluginKeys))
			Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
		})
	})

	Context("resolvePlugins", func() {
		BeforeEach(func() {
			pluginKeys := []string{
				"foo.example.com/v1",
				"bar.example.com/v1",
				"baz.example.com/v1",
				"foo.kubebuilder.io/v1",
				"foo.kubebuilder.io/v2",
				"bar.kubebuilder.io/v1",
				"bar.kubebuilder.io/v2",
			}

			plugins := makeMockPluginsFor(projectVersion, pluginKeys...)
			plugins = append(plugins,
				newMockPlugin("invalid.kubebuilder.io", "v1"),
				newMockPlugin("only1.kubebuilder.io", "v1",
					config.Version{Number: 1}),
				newMockPlugin("only2.kubebuilder.io", "v1",
					config.Version{Number: 2}),
				newMockPlugin("1and2.kubebuilder.io", "v1",
					config.Version{Number: 1}, config.Version{Number: 2}),
				newMockPlugin("2and3.kubebuilder.io", "v1",
					config.Version{Number: 2}, config.Version{Number: 3}),
				newMockPlugin("1-2and3.kubebuilder.io", "v1",
					config.Version{Number: 1}, config.Version{Number: 2}, config.Version{Number: 3}),
			)
			pluginMap := makeMapFor(plugins...)

			c.plugins = pluginMap
		})

		DescribeTable("should resolve",
			func(key, qualified string) {
				c.pluginKeys = []string{key}
				c.projectVersion = projectVersion

				Expect(c.resolvePlugins()).To(Succeed())
				Expect(c.resolvedPlugins).To(HaveLen(1))
				Expect(plugin.KeyFor(c.resolvedPlugins[0])).To(Equal(qualified))
			},
			Entry("fully qualified plugin", "foo.example.com/v1", "foo.example.com/v1"),
			Entry("plugin without version", "foo.example.com", "foo.example.com/v1"),
			Entry("shortname without version", "baz", "baz.example.com/v1"),
			Entry("shortname with version", "foo/v2", "foo.kubebuilder.io/v2"),
		)

		DescribeTable("should not resolve",
			func(key string) {
				c.pluginKeys = []string{key}
				c.projectVersion = projectVersion

				Expect(c.resolvePlugins()).NotTo(Succeed())
			},
			Entry("for an ambiguous version", "foo.kubebuilder.io"),
			Entry("for an ambiguous name", "foo/v1"),
			Entry("for an ambiguous name and version", "foo"),
			Entry("for a non-existent name", "blah"),
			Entry("for a non-existent version", "foo.example.com/v2"),
			Entry("for a non-existent version", "foo/v3"),
			Entry("for a non-existent version", "foo.example.com/v3"),
			Entry("for a plugin that doesn't support the project version", "invalid.kubebuilder.io/v1"),
		)

		It("should succeed if only one common project version is found", func() {
			c.pluginKeys = []string{"1and2", "2and3"}

			Expect(c.resolvePlugins()).To(Succeed())
			Expect(c.projectVersion.Compare(config.Version{Number: 2})).To(Equal(0))
		})

		It("should fail if no common project version is found", func() {
			c.pluginKeys = []string{"only1", "only2"}

			Expect(c.resolvePlugins()).NotTo(Succeed())
		})

		It("should fail if more than one common project versions are found", func() {
			c.pluginKeys = []string{"1and2", "1-2and3"}

			Expect(c.resolvePlugins()).NotTo(Succeed())
		})

		It("should succeed if more than one common project versions are found and one is the default", func() {
			c.pluginKeys = []string{"2and3", "1-2and3"}
			c.defaultProjectVersion = projectVersion

			Expect(c.resolvePlugins()).To(Succeed())
			Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
		})
	})

	Context("applySubcommandHooks", func() {
		var (
			cmd        *cobra.Command
			sub1, sub2 *pluginChainCapturingSubcommand
			tuples     []keySubcommandTuple
			chainKeys  []string
		)

		BeforeEach(func() {
			cmd = &cobra.Command{}
			sub1 = &pluginChainCapturingSubcommand{}
			sub2 = &pluginChainCapturingSubcommand{}
			tuples = []keySubcommandTuple{
				{key: "alpha.kubebuilder.io/v1", subcommand: sub1},
				{key: "beta.kubebuilder.io/v1", subcommand: sub2},
			}
			chainKeys = []string{"alpha.kubebuilder.io/v1", "beta.kubebuilder.io/v1"}
		})

		It("sets the plugin chain on subcommands", func() {
			c.applySubcommandHooks(cmd, tuples, "test", false)

			Expect(sub1.pluginChain).To(Equal(chainKeys))
			Expect(sub2.pluginChain).To(Equal(chainKeys))
		})

		It("sets the plugin chain when creating a new configuration", func() {
			c.resolvedPlugins = makeMockPluginsFor(projectVersion, chainKeys...)

			c.applySubcommandHooks(cmd, tuples, "test", true)

			Expect(sub1.pluginChain).To(Equal(chainKeys))
			Expect(sub2.pluginChain).To(Equal(chainKeys))
		})
	})

	Context("New", func() {
		var c *CLI
		var err error

		When("no option is provided", func() {
			It("should create a valid CLI", func() {
				_, err = New()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		// NOTE: Options are extensively tested in their own tests.
		//       The ones tested here ensure better coverage.

		When("providing a version string", func() {
			It("should create a valid CLI", func() {
				const version = "version string"
				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithVersion(version),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(hasSubCommand(c.cmd, "version")).To(BeTrue())

				// Test the version command
				c.cmd.SetArgs([]string{kubebuilderSubcommandVersion})
				// Overwrite stdout to read the output and reset it afterwards
				r, w, _ := os.Pipe()
				temp := os.Stdout
				defer func() {
					os.Stdout = temp
				}()
				os.Stdout = w
				Expect(c.cmd.Execute()).Should(Succeed())

				_ = w.Close()

				Expect(err).NotTo(HaveOccurred())
				printed, _ := io.ReadAll(r)
				Expect(string(printed)).To(Equal(
					fmt.Sprintf("%s\n", version)))
			})
		})

		When("enabling completion", func() {
			It("should create a valid CLI", func() {
				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithCompletion(),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(hasSubCommand(c.cmd, "completion")).To(BeTrue())
			})
		})

		When("requesting help", func() {
			It("should ignore an invalid PROJECT path when a flag without a value precedes help", func() {
				err = executeCLI(filesystemWithUnloadableProject(),
					kubebuilderSubcommandInit, pluginsFlagArg, helpFlagArg)
				Expect(err).To(MatchError(errHelpDisplayed))
			})

			It("should read a loadable PROJECT file so help reflects the project's plugin chain", func() {
				fs := &projectReadCountingFs{Fs: filesystemWithResolvableProject().FS}

				Expect(executeCLI(machinery.Filesystem{FS: fs},
					createSubcommand, apiSubcommand, helpFlagArg)).To(Succeed())
				Expect(fs.reads).NotTo(BeZero())
			})

			It("should return a configuration error for commands requiring a project", func() {
				err = executeCLI(filesystemWithUnloadableProject(),
					createSubcommand, apiSubcommand, kindFlagArg, kubebuilderSubcommandHelp)
				Expect(err).To(MatchError(ContainSubstring("version 1 is not supported")))
			})
		})

		When("a flag value is spelled as a help flag", func() {
			// A flag value is data, so it must not become a help request that hides the plugin resolution error.
			DescribeTable("should report the plugin resolution error",
				func(flagArg string) {
					err = executeCLI(filesystemWithUnresolvableProject(),
						createSubcommand, apiSubcommand, flagArg, kubebuilderSubcommandHelp)
					Expect(err).To(MatchError(ContainSubstring(
						`no plugin could be resolved with key "gone.example.com/v1"`)))
				},
				Entry("for --kind", kindFlagArg),
				Entry("for --group", "--group"),
				Entry("for --version", "--version"),
			)
		})

		When("PROJECT is a directory", func() {
			It("should report what occupies the path instead of reading the configuration", func() {
				err = executeCLI(filesystemWithProjectDirectory(),
					createSubcommand, apiSubcommand, kindFlagArg, kindValue)
				Expect(err).To(MatchError(ContainSubstring(`"PROJECT" is a directory`)))
			})

			It("should refuse to initialize the project without scaffolding anything", func() {
				fs := filesystemWithProjectDirectory()

				err = executeCLI(fs, kubebuilderSubcommandInit)
				Expect(err).To(MatchError(ContainSubstring(`"PROJECT" is a directory`)))
				Expect(err).NotTo(MatchError(ContainSubstring("already initialized")))
				expectNothingScaffolded(fs)
			})
		})

		When("PROJECT is a symbolic link with a missing target", func() {
			It("should refuse to initialize without creating the target or scaffolding anything", func() {
				skipWithoutSymlinks()
				fs, target := filesystemWithDanglingProjectSymlink()

				err = executeCLI(fs, kubebuilderSubcommandInit)
				Expect(err).To(MatchError(ContainSubstring(`"PROJECT" is a symbolic link`)))
				Expect(target).NotTo(BeAnExistingFile())
				expectNothingScaffolded(fs)
			})

			It("should report the link instead of reading the configuration", func() {
				skipWithoutSymlinks()
				fs, target := filesystemWithDanglingProjectSymlink()

				err = executeCLI(fs, createSubcommand, apiSubcommand, kindFlagArg, kindValue)
				Expect(err).To(MatchError(ContainSubstring(`"PROJECT" is a symbolic link`)))
				Expect(target).NotTo(BeAnExistingFile())
			})
		})

		When("completing a command line", func() {
			It("should offer the subcommands of the project", func() {
				completions, completeErr := completeCLI(filesystemWithResolvableProject(), createSubcommand, "")
				Expect(completeErr).NotTo(HaveOccurred())
				Expect(completions).To(ContainSubstring(apiSubcommand))
				Expect(completions).To(ContainSubstring(webhookSubcommand))
			})

			It("should offer the flags of a subcommand", func() {
				completions, completeErr := completeCLI(filesystemWithResolvableProject(),
					createSubcommand, apiSubcommand, "--")
				Expect(completeErr).NotTo(HaveOccurred())
				Expect(completions).To(ContainSubstring(kindFlagArg))
			})

			It("should still offer completions when the PROJECT file cannot be read", func() {
				completions, completeErr := completeCLI(filesystemWithProjectDirectory(), createSubcommand, "")
				Expect(completeErr).NotTo(HaveOccurred())
				Expect(completions).To(ContainSubstring(apiSubcommand))
			})
		})

		When("no PROJECT file exists", func() {
			DescribeTable("should ask for an initialized project without scaffolding anything",
				func(args ...string) {
					fs := filesystemWithoutProject()

					err = executeCLI(fs, args...)
					Expect(err).To(MatchError(ContainSubstring("project must be initialized")))

					entries, readErr := afero.ReadDir(fs.FS, ".")
					Expect(readErr).NotTo(HaveOccurred())
					Expect(entries).To(BeEmpty())
				},
				Entry("for create api", createSubcommand, apiSubcommand, kindFlagArg, kindValue),
				Entry("for create webhook", createSubcommand, webhookSubcommand, kindFlagArg, kindValue, "--defaulting"),
				Entry("for edit", editSubcommand),
			)
		})

		When("initializing a project", func() {
			It("should refuse as already initialized when a valid PROJECT file exists", func() {
				fs := filesystemWithResolvableProject()

				err = executeCLI(fs, kubebuilderSubcommandInit)
				Expect(err).To(MatchError(ContainSubstring("already initialized")))
				expectNothingScaffolded(fs)
			})

			It("should stop at startup without scaffolding when the PROJECT file cannot be loaded", func() {
				fs := filesystemWithUnloadableProject()

				err = executeCLI(fs, kubebuilderSubcommandInit)
				Expect(err).To(MatchError(ContainSubstring("version 1 is not supported")))
				expectNothingScaffolded(fs)
			})
		})

		When("providing an invalid option", func() {
			It("should return an error", func() {
				// An empty project version is not valid
				_, err = New(WithDefaultProjectVersion(config.Version{}))
				Expect(err).To(HaveOccurred())
			})
		})

		When("being unable to resolve plugins", func() {
			// Save os.Args and restore it for every test
			var args []string
			BeforeEach(func() { args = os.Args })
			AfterEach(func() { os.Args = args })

			It("should return a CLI that returns an error", func() {
				setPluginsFlag("foo")

				c, err = New()
				Expect(err).NotTo(HaveOccurred())

				// Overwrite stderr to read the output and reset it afterwards
				_, w, _ := os.Pipe()
				temp := os.Stderr
				defer func() {
					os.Stderr = temp
					_ = w.Close()
				}()
				os.Stderr = w

				Expect(c.Run()).NotTo(Succeed())
			})
		})

		When("providing extra commands", func() {
			It("should create a valid CLI for non-conflicting ones", func() {
				extraCommand := &cobra.Command{Use: subcommandExtra}
				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithExtraCommands(extraCommand),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(hasSubCommand(c.cmd, extraCommand.Use)).To(BeTrue())
			})

			It("should return an error for conflicting ones", func() {
				extraCommand := &cobra.Command{Use: kubebuilderSubcommandInit}
				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithExtraCommands(extraCommand),
				)
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing extra alpha commands", func() {
			It("should create a valid CLI for non-conflicting ones", func() {
				extraAlphaCommand := &cobra.Command{Use: subcommandExtra}
				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithExtraAlphaCommands(extraAlphaCommand),
				)
				Expect(err).NotTo(HaveOccurred())
				var alpha *cobra.Command
				for _, subcmd := range c.cmd.Commands() {
					if subcmd.Name() == alphaCommand {
						alpha = subcmd
						break
					}
				}
				Expect(alpha).NotTo(BeNil())
				Expect(hasSubCommand(alpha, extraAlphaCommand.Use)).To(BeTrue())
			})

			It("should return an error for conflicting ones", func() {
				extraAlphaCommand := &cobra.Command{Use: subcommandExtra}
				_, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithExtraAlphaCommands(extraAlphaCommand, extraAlphaCommand),
				)
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing deprecated plugins", func() {
			It("should succeed and print the deprecation notice", func() {
				const (
					deprecationWarning = "DEPRECATED"
				)
				deprecatedPlugin := newMockDeprecatedPlugin("deprecated", "v1", deprecationWarning, projectVersion)

				// Overwrite stderr to read the deprecation output and reset it afterwards
				r, w, _ := os.Pipe()
				temp := os.Stderr
				defer func() {
					os.Stderr = temp
				}()
				os.Stderr = w

				c, err = New(
					WithPlugins(deprecatedPlugin),
					WithDefaultPlugins(projectVersion, deprecatedPlugin),
					WithDefaultProjectVersion(projectVersion),
				)

				_ = w.Close()

				Expect(err).NotTo(HaveOccurred())
				printed, _ := io.ReadAll(r)
				Expect(string(printed)).To(Equal(
					fmt.Sprintf(noticeColor, fmt.Sprintf(deprecationFmt, deprecationWarning))))
			})
		})

		When("new succeeds", func() {
			It("should return the underlying command", func() {
				c, err = New()
				Expect(err).NotTo(HaveOccurred())
				Expect(c.Command()).NotTo(BeNil())
				Expect(c.Command()).To(Equal(c.cmd))
			})
		})
	})
})

var _ = Describe("isSubcommandWithoutConfig", func() {
	var c *CLI

	BeforeEach(func() {
		var err error
		c, err = newCLI()
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("should not read the project configuration",
		func(args ...string) {
			Expect(c.isSubcommandWithoutConfig(args)).To(BeTrue())
		},
		Entry("without arguments"),
		Entry("with root flags only", pluginsFlagArg, pluginGoKubebuilderV4),
		Entry("for the help subcommand", kubebuilderSubcommandHelp, kubebuilderSubcommandInit),
		Entry("for the version subcommand", kubebuilderSubcommandVersion),
		Entry("for the completion subcommand", kubebuilderSubcommandCompletion, shellZsh),
	)

	DescribeTable("should read the project configuration",
		func(args ...string) {
			Expect(c.isSubcommandWithoutConfig(args)).To(BeFalse())
		},
		Entry("with help as a flag value", "create", "api", "--kind", kubebuilderSubcommandHelp),
		Entry("with help as a flag value in the equals form", "create", "api", "--kind="+kubebuilderSubcommandHelp),
		Entry("with help as an argument of another subcommand", "create", "api", kubebuilderSubcommandHelp),
		// Help output describes the plugin chain of the project, so the configuration is read.
		Entry("with the help flag of a subcommand", kubebuilderSubcommandInit, helpFlagArg),
		Entry("with a dangling plugins flag", kubebuilderSubcommandInit, pluginsFlagArg),
		Entry("for a subcommand that requires a project", kubebuilderSubcommandInit),
	)

	It("should not read the project configuration for subcommands registered by the CLI", func() {
		const docsSubcommand = "docs"

		var err error
		c, err = newCLI(WithSubcommandsWithoutConfig(docsSubcommand, "alpha generate"))
		Expect(err).NotTo(HaveOccurred())

		Expect(c.isSubcommandWithoutConfig([]string{docsSubcommand})).To(BeTrue())
		Expect(c.isSubcommandWithoutConfig([]string{"alpha", "generate"})).To(BeTrue())
		Expect(c.isSubcommandWithoutConfig([]string{"alpha", "update"})).To(BeFalse())
		Expect(c.isSubcommandWithoutConfig([]string{kubebuilderSubcommandVersion})).To(BeTrue())
	})
})

var _ = Describe("isHelpFlag", func() {
	DescribeTable("should recognize a request for help",
		func(arg string) {
			Expect(isHelpFlag(arg)).To(BeTrue())
		},
		Entry("for the help flag", helpFlagArg),
		Entry("for the help shorthand", helpShorthandArg),
		Entry("for the help flag set to a value", helpFlagArg+"=true"),
		Entry("for the help shorthand set to a value", helpShorthandArg+"=true"),
		Entry("for the bare help word", kubebuilderSubcommandHelp),
	)

	DescribeTable("should not recognize a request for help",
		func(arg string) {
			Expect(isHelpFlag(arg)).To(BeFalse())
		},
		Entry("for the help flag explicitly set to false", helpFlagArg+"=false"),
		Entry("for the help shorthand explicitly set to false", helpShorthandArg+"=false"),
		Entry("for a non boolean value", helpFlagArg+"=please"),
		Entry("for a plugin key", pluginGoKubebuilderV4),
		Entry("for another flag", pluginsFlagArg),
	)
})

type describedFilesystem struct {
	description string
	build       func() machinery.Filesystem
}

type describedCommand struct {
	description string
	args        []string
}

// brokenProjectStates are the project states that must not stop a command that does not consume the
// project configuration.
var brokenProjectStates = []describedFilesystem{
	{"the PROJECT file is missing", filesystemWithoutProject},
	{"the PROJECT file is not valid YAML", filesystemWithInvalidProject},
	{"the PROJECT project version is not supported", filesystemWithUnloadableProject},
	{"the PROJECT plugin chain cannot be resolved", filesystemWithUnresolvableProject},
	{"PROJECT is a directory", filesystemWithProjectDirectory},
}

// configFreeCommands are the commands that do not consume the project configuration.
var configFreeCommands = []describedCommand{
	{"the root command", nil},
	{"the version subcommand", []string{kubebuilderSubcommandVersion}},
	{"the help subcommand", []string{kubebuilderSubcommandHelp}},
	{"the completion subcommand", []string{kubebuilderSubcommandCompletion, shellZsh}},
	{"the help flag of a subcommand", []string{kubebuilderSubcommandInit, helpFlagArg}},
	{"root flags without a subcommand", []string{pluginsFlagArg, pluginGoKubebuilderV4}},
	{"a completion request", []string{cobra.ShellCompRequestCmd, createSubcommand, ""}},
}

func configFreeCommandEntries() []TableEntry {
	entries := make([]TableEntry, 0, len(brokenProjectStates)*len(configFreeCommands))
	for _, state := range brokenProjectStates {
		for _, command := range configFreeCommands {
			entries = append(entries,
				Entry(fmt.Sprintf("%s when %s", command.description, state.description), state.build, command.args))
		}
	}

	return entries
}

var _ = Describe("commands that do not consume the project configuration", func() {
	DescribeTable("should run",
		func(filesystem func() machinery.Filesystem, args []string) {
			Expect(executeCLI(filesystem(), args...)).To(Succeed())
		},
		configFreeCommandEntries(),
	)

	It("should not read the PROJECT file", func() {
		fs := &projectReadCountingFs{Fs: filesystemWithUnresolvableProject().FS}

		Expect(executeCLI(machinery.Filesystem{FS: fs}, kubebuilderSubcommandVersion)).To(Succeed())
		Expect(fs.reads).To(BeZero())
	})
})

// An embedded CLI is built before the caller chooses the command with Command().SetArgs, so the
// arguments of the running program say nothing about the command that will run.
var _ = Describe("commands driven by Command().SetArgs", func() {
	// The arguments of the running program, which name a command that consumes the configuration.
	var programArgs []string

	BeforeEach(func() {
		programArgs = []string{createSubcommand, apiSubcommand, kindFlagArg, kindValue}
	})

	DescribeTable("should run",
		func(filesystem func() machinery.Filesystem, args []string) {
			Expect(executeEmbeddedCLI(filesystem(), programArgs, args)).To(Succeed())
		},
		configFreeCommandEntries(),
	)

	It("should run a command registered as not requiring the configuration", func() {
		const docsSubcommand = "docs"
		docs := &cobra.Command{Use: docsSubcommand, RunE: func(*cobra.Command, []string) error { return nil }}

		Expect(executeEmbeddedCLI(filesystemWithInvalidProject(), programArgs, []string{docsSubcommand},
			WithExtraCommands(docs),
			WithSubcommandsWithoutConfig(docsSubcommand),
		)).To(Succeed())
	})

	DescribeTable("should report the failure for a command that consumes the configuration",
		func(filesystem func() machinery.Filesystem, expected string) {
			err := executeEmbeddedCLI(filesystem(), []string{kubebuilderSubcommandVersion},
				[]string{createSubcommand, apiSubcommand, kindFlagArg, kindValue})
			Expect(err).To(MatchError(ContainSubstring(expected)))
		},
		Entry("when the PROJECT file is not valid YAML",
			filesystemWithInvalidProject, "failed to determine config version"),
		Entry("when the PROJECT project version is not supported",
			filesystemWithUnloadableProject, "version 1 is not supported"),
		Entry("when the PROJECT plugin chain cannot be resolved",
			filesystemWithUnresolvableProject, `no plugin could be resolved with key "gone.example.com/v1"`),
	)

	It("should refuse to scaffold with a plugin chain the command tree was not built for", func() {
		// The command tree was built for the CLI defaults, so the plugins the project asks for
		// cannot be honored anymore.
		extraPlugin := newTestCreateAPIPlugin("mock.example.com", plugin.Version{Number: 1})

		err := executeEmbeddedCLI(filesystemWithResolvableProject(),
			[]string{kubebuilderSubcommandVersion},
			[]string{createSubcommand, apiSubcommand, kindFlagArg, kindValue},
			WithPlugins(extraPlugin),
			WithDefaultPlugins(config.Version{Number: 3}, extraPlugin),
		)
		Expect(err).To(MatchError(ContainSubstring("the command tree was built for the plugin chain")))
		Expect(err).To(MatchError(ContainSubstring("WithArgs")))
	})

	It("should read the project configuration when the caller passes the same arguments", func() {
		fs := &projectReadCountingFs{Fs: filesystemWithResolvableProject().FS}
		args := []string{createSubcommand, apiSubcommand, helpFlagArg}

		Expect(executeEmbeddedCLI(machinery.Filesystem{FS: fs},
			[]string{kubebuilderSubcommandVersion}, args, WithArgs(args))).To(Succeed())
		Expect(fs.reads).NotTo(BeZero())
	})
})
