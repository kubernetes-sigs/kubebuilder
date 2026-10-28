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
	"errors"
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
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	yamlstore "sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	golangv4 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4"
	helmv2alpha "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha"
)

const (
	createSubcommand  = "create"
	apiSubcommand     = "api"
	webhookSubcommand = "webhook"
	editSubcommand    = "edit"
	docsSubcommand    = "docs"
	groupFlagArg      = "--group"
	versionFlagArg    = "--version"
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
	return findSubCommand(cmd, name) != nil
}

// findSubCommand returns the direct subcommand with the given name, or nil when there is none.
func findSubCommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, subcommand := range cmd.Commands() {
		if subcommand.Name() == name {
			return subcommand
		}
	}
	return nil
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

// filesystemWithProjectChainProject returns a filesystem holding a valid PROJECT file whose plugin
// chain names a registered plugin that is not one of the CLI defaults.
func filesystemWithProjectChainProject() machinery.Filesystem {
	return filesystemWithProject(`domain: example.com
layout:
- ` + projectChainPluginKey + `
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

// filesystemWithProject returns a temporary filesystem containing the given PROJECT content.
func filesystemWithProject(content string) machinery.Filesystem {
	GinkgoHelper()

	fs := filesystemWithoutProject()
	Expect(afero.WriteFile(fs.FS, yamlstore.DefaultPath, []byte(content), machinery.DefaultFilePermission)).To(Succeed())

	return fs
}

// filesystemWithoutProject returns an empty temporary filesystem.
func filesystemWithoutProject() machinery.Filesystem {
	return machinery.Filesystem{FS: afero.NewBasePathFs(afero.NewOsFs(), GinkgoT().TempDir())}
}

func filesystemWithInvalidProjectPath() machinery.Filesystem {
	fs := filesystemWithoutProject()
	Expect(fs.FS.Mkdir(yamlstore.DefaultPath, machinery.DefaultDirectoryPermission)).To(Succeed())

	return fs
}

// filesystemWithNonRegularProject returns a filesystem where PROJECT is not a regular file, which
// must never be opened.
func filesystemWithNonRegularProject() machinery.Filesystem {
	return machinery.Filesystem{FS: &nonRegularProjectFs{Fs: filesystemWithResolvableProject().FS}}
}

// filesystemWithPopulatedProjectDirectory returns a filesystem where PROJECT is a directory that
// holds a file.
func filesystemWithPopulatedProjectDirectory() machinery.Filesystem {
	GinkgoHelper()

	fs := filesystemWithProjectDirectory()
	Expect(afero.WriteFile(fs.FS, filepath.Join(yamlstore.DefaultPath, "inner.yaml"),
		[]byte(`version: "3"`), machinery.DefaultFilePermission)).To(Succeed())

	return fs
}

// filesystemWithProjectDirectory returns a filesystem where PROJECT is a directory, which holds no
// configuration and must be handled as a project that was never initialized.
func filesystemWithProjectDirectory() machinery.Filesystem {
	GinkgoHelper()

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

// createAPIArgs returns a command line that creates an API with every required resource flag.
func createAPIArgs() []string {
	return []string{createSubcommand, apiSubcommand, groupFlagArg, "crew", versionFlagArg, "v1", kindFlagArg, kindValue}
}

// newExtraCommand returns a command that does nothing when run. Cobra runs no hooks for a command
// without a run function.
func newExtraCommand(name string) *cobra.Command {
	return &cobra.Command{Use: name, RunE: func(*cobra.Command, []string) error { return nil }}
}

// newExtraCommandWithHook returns an extra command whose pre-run hook, set by setHook, records in
// the returned flag that it ran.
func newExtraCommandWithHook(name string, setHook func(*cobra.Command, *bool)) (*cobra.Command, *bool) {
	cmd := newExtraCommand(name)
	hookRan := false
	setHook(cmd, &hookRan)

	return cmd, &hookRan
}

// executeCLIWith builds a CLI with one more option and runs it with args, as executeCLI does.
func executeCLIWith(filesystem machinery.Filesystem, option Option, args ...string) error {
	GinkgoHelper()

	_, err := runCLI(filesystem, args, args, option)

	return err
}

// preRunHookEntries returns one entry per pre-run hook a command can set. Each hook records that
// it ran.
func preRunHookEntries() []TableEntry {
	return []TableEntry{
		Entry("for PersistentPreRunE", setPersistentPreRunE),
		Entry("for PersistentPreRun", setPersistentPreRun),
		Entry("for PreRunE", setPreRunE),
		Entry("for PreRun", setPreRun),
	}
}

func setPersistentPreRunE(cmd *cobra.Command, ran *bool) {
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		*ran = true
		return nil
	}
}

func setPersistentPreRun(cmd *cobra.Command, ran *bool) {
	cmd.PersistentPreRun = func(*cobra.Command, []string) { *ran = true }
}

func setPreRunE(cmd *cobra.Command, ran *bool) {
	cmd.PreRunE = func(*cobra.Command, []string) error {
		*ran = true
		return nil
	}
}

func setPreRun(cmd *cobra.Command, ran *bool) {
	cmd.PreRun = func(*cobra.Command, []string) { *ran = true }
}

// expectStoppedBeforeHook asserts that the command stopped on the directory at PROJECT before its
// own hook ran.
func expectStoppedBeforeHook(err error, hookRan *bool) {
	GinkgoHelper()

	Expect(err).To(MatchError(ContainSubstring(`"PROJECT" is a directory`)))
	Expect(*hookRan).To(BeFalse())
}

// expectHookRan asserts that the command succeeded and its own hook ran.
func expectHookRan(err error, hookRan *bool) {
	GinkgoHelper()

	Expect(err).NotTo(HaveOccurred())
	Expect(*hookRan).To(BeTrue())
}

// projectReadCountingFs counts how many times the project configuration file is opened.
type projectReadCountingFs struct {
	afero.Fs
	reads int
}

// Open counts PROJECT reads before delegating to the wrapped filesystem.
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

// nonRegularProjectFileInfo reports a named pipe mode on top of the wrapped FileInfo.
type nonRegularProjectFileInfo struct{ os.FileInfo }

// Mode adds a named-pipe mode to the wrapped file information.
func (i nonRegularProjectFileInfo) Mode() os.FileMode { return i.FileInfo.Mode() | os.ModeNamedPipe }

// nonRegularProjectFs makes the project configuration file look like a named pipe.
type nonRegularProjectFs struct {
	afero.Fs
}

// Stat reports PROJECT as a named pipe while preserving the wrapped filesystem behavior.
func (f *nonRegularProjectFs) Stat(name string) (os.FileInfo, error) {
	info, err := f.Fs.Stat(name)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %q: %w", name, err)
	}
	if name == yamlstore.DefaultPath {
		info = nonRegularProjectFileInfo{info}
	}

	return info, nil
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

	return runBuiltCLI(buildCLI(filesystem, programArgs, options...), commandArgs)
}

// setProgramArgs sets the arguments of the running program until the test ends.
func setProgramArgs(programArgs ...string) {
	GinkgoHelper()

	originalArgs := os.Args
	DeferCleanup(func() { os.Args = originalArgs })
	os.Args = append([]string{kubebuilderCommandName}, programArgs...)
}

// buildCLI builds a CLI over the given filesystem while the program was invoked with programArgs.
func buildCLI(filesystem machinery.Filesystem, programArgs []string, options ...Option) *CLI {
	GinkgoHelper()

	setProgramArgs(programArgs...)

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

	return c
}

// runBuiltCLI runs the CLI with commandArgs and returns what the command wrote.
func runBuiltCLI(c *CLI, commandArgs []string) (string, error) {
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

const (
	projectChainPluginName  = "mock.example.com"
	projectChainPluginKey   = projectChainPluginName + "/v1"
	projectChainFlag        = "mock-only"
	projectChainDescription = "Scaffold with the plugin the project asks for"
)

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

// newDescribedCreateAPIPlugin returns a plugin whose create api subcommand brings a description and
// a flag of its own, so that the help output tells which plugin chain built it.
func newDescribedCreateAPIPlugin() testCreateAPIPlugin {
	p := newTestCreateAPIPlugin(projectChainPluginName, plugin.Version{Number: 1})
	p.subcommand.description = projectChainDescription
	p.subcommand.flagName = projectChainFlag

	return p
}

func (p testCreateAPIPlugin) Name() string                               { return p.name }
func (p testCreateAPIPlugin) Version() plugin.Version                    { return p.version }
func (p testCreateAPIPlugin) SupportedProjectVersions() []config.Version { return p.projectVers }
func (p testCreateAPIPlugin) GetCreateAPISubcommand() plugin.CreateAPISubcommand {
	return p.subcommand
}

type testCreateAPISubcommand struct {
	description string
	flagName    string
	scaffolded  bool
}

func (s *testCreateAPISubcommand) UpdateMetadata(_ plugin.CLIMetadata, meta *plugin.SubcommandMetadata) {
	if s.description != "" {
		meta.Description = s.description
	}
}

func (s *testCreateAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	if s.flagName != "" {
		fs.Bool(s.flagName, false, "flag of the plugin the project asks for")
	}
}

func (s *testCreateAPISubcommand) InjectResource(*resource.Resource) error {
	return nil
}

func (s *testCreateAPISubcommand) Scaffold(machinery.Filesystem) error {
	s.scaffolded = true

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
			It("should fail when stable is not registered", func() {
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

	Context("getInfoFromConfigFile", func() {
		When("the project configuration file does not exist", func() {
			It("should fail with error loading configuration", func() {
				c.fs = filesystemWithoutProject()
				err := c.getInfoFromConfigFile()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error loading configuration"))
			})
		})

		When("the project configuration file is invalid YAML", func() {
			It("should fail with error loading configuration", func() {
				c.fs = filesystemWithInvalidProject()
				err := c.getInfoFromConfigFile()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error loading configuration"))
			})
		})

		When("the project configuration file version is not supported", func() {
			It("should fail with error loading configuration", func() {
				c.fs = filesystemWithUnloadableProject()
				err := c.getInfoFromConfigFile()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error loading configuration"))
			})
		})

		When("the project configuration file is valid", func() {
			It("should succeed and populate pluginKeys and projectVersion", func() {
				c.fs = filesystemWithResolvableProject()
				Expect(c.getInfoFromConfigFile()).To(Succeed())
				Expect(c.pluginKeys).To(Equal([]string{"base.go.kubebuilder.io/v4"}))
				Expect(c.projectVersion.Compare(config.Version{Number: 3})).To(Equal(0))
			})
		})

		When("the project configuration file has multiple plugins in layout", func() {
			It("should succeed and populate all pluginKeys", func() {
				c.fs = filesystemWithProject(`domain: example.com
layout:
- ` + pluginGoKubebuilderV2 + `
- deploy-image.go.kubebuilder.io/v1-alpha
projectName: test
version: "3"
`)
				Expect(c.getInfoFromConfigFile()).To(Succeed())
				Expect(c.pluginKeys).To(Equal([]string{pluginGoKubebuilderV2, "deploy-image.go.kubebuilder.io/v1-alpha"}))
				Expect(c.projectVersion.Compare(config.Version{Number: 3})).To(Equal(0))
			})
		})

		When("the project configuration file contains invalid plugin keys", func() {
			It("should fail to parse the plugin key", func() {
				c.fs = filesystemWithProject(`domain: example.com
layout:
- _/v1
projectName: test
version: "3"
`)
				Expect(c.getInfoFromConfigFile()).NotTo(Succeed())
			})
		})

		When("running alpha generate on an older go/v3 project", func() {
			It("should patch the project in memory to go/v4 and succeed", func() {
				c.args = []string{alphaCommand, generateSubcommand}
				c.fs = filesystemWithProject(`domain: example.com
layout:
- go.kubebuilder.io/v3
projectName: test
version: "3"
`)
				Expect(c.getInfoFromConfigFile()).To(Succeed())
				Expect(c.pluginKeys).To(Equal([]string{pluginGoKubebuilderV4}))
			})
		})

		When("running alpha generate on a non-regular project file", func() {
			It("should skip in-memory patching and report an error", func() {
				c.args = []string{alphaCommand, generateSubcommand}
				c.fs = filesystemWithNonRegularProject()
				err := c.getInfoFromConfigFile()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("getInfoFromConfig", func() {
		DescribeTable("resolves legacy Helm only for alpha generate and preserves the source configuration",
			func(args []string, expectedHelm string) {
				c.args = args
				projectConfig := cfgv3.New()
				legacyHelm := "helm.kubebuilder.io/v1-alpha"
				pluginChain := []string{pluginGoKubebuilderV4, legacyHelm}
				Expect(projectConfig.SetPluginChain(pluginChain)).To(Succeed())
				Expect(projectConfig.EncodePluginConfig(legacyHelm, map[string]any{})).To(Succeed())
				Expect(c.getInfoFromConfig(projectConfig)).To(Succeed())
				Expect(c.pluginKeys).To(Equal([]string{pluginGoKubebuilderV4, expectedHelm}))
				Expect(projectConfig.GetPluginChain()).To(Equal(pluginChain))
				var helmConfig map[string]any
				Expect(projectConfig.DecodePluginConfig(legacyHelm, &helmConfig)).To(Succeed())
			},
			Entry("alpha generate", []string{alphaCommand, generateSubcommand}, "helm.kubebuilder.io/v2-alpha"),
			Entry("create webhook", []string{"create", "webhook"}, "helm.kubebuilder.io/v1-alpha"),
		)

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

			It("should ignore an invalid PROJECT path for version", func() {
				const version = "version string"
				args := os.Args
				defer func() {
					os.Args = args
				}()
				os.Args = []string{kubebuilderCommandName, kubebuilderSubcommandVersion}

				fs := filesystemWithInvalidProjectPath()

				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithVersion(version),
					WithFilesystem(fs),
				)
				Expect(err).NotTo(HaveOccurred())

				c.cmd.SetArgs([]string{kubebuilderSubcommandVersion})
				Expect(c.cmd.Execute()).To(Succeed())
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

			It("should ignore an invalid PROJECT path for completion", func() {
				args := os.Args
				defer func() {
					os.Args = args
				}()
				os.Args = []string{kubebuilderCommandName, kubebuilderSubcommandCompletion, shellZsh}

				fs := filesystemWithInvalidProjectPath()

				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithVersion("version string"),
					WithCompletion(),
					WithFilesystem(fs),
				)
				Expect(err).NotTo(HaveOccurred())

				c.cmd.SetArgs([]string{kubebuilderSubcommandCompletion, shellZsh})
				Expect(c.cmd.Execute()).To(Succeed())
			})
		})

		When("requesting help", func() {
			It("should ignore an invalid PROJECT path when a flag without a value precedes help", func() {
				args := os.Args
				defer func() {
					os.Args = args
				}()
				os.Args = []string{kubebuilderCommandName, kubebuilderSubcommandInit, pluginsFlagArg, helpFlagArg}

				fs := filesystemWithInvalidProjectPath()

				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithVersion("version string"),
					WithCompletion(),
					WithFilesystem(fs),
				)
				Expect(err).NotTo(HaveOccurred())

				c.cmd.SetArgs([]string{kubebuilderSubcommandInit, pluginsFlagArg, helpFlagArg})
				Expect(c.cmd.Execute()).To(MatchError("help displayed"))
			})

			It("should read a loadable PROJECT file so help reflects the project's plugin chain", func() {
				fs := &projectReadCountingFs{Fs: filesystemWithResolvableProject().FS}

				Expect(executeCLI(machinery.Filesystem{FS: fs},
					createSubcommand, apiSubcommand, helpFlagArg)).To(Succeed())
				Expect(fs.reads).NotTo(BeZero())
			})

			DescribeTable("should display help when it is passed as a plugins value",
				func(filesystem func() machinery.Filesystem) {
					err = executeCLI(filesystem(), createSubcommand, apiSubcommand,
						pluginsFlagArg+"="+kubebuilderSubcommandHelp)
					Expect(err).To(MatchError(errHelpDisplayed))
				},
				Entry("when the PROJECT file is missing", filesystemWithoutProject),
				Entry("when PROJECT is a directory", filesystemWithProjectDirectory),
				Entry("when the PROJECT file cannot be loaded", filesystemWithUnloadableProject),
			)

			It("should return a configuration error for commands requiring a project", func() {
				err = executeCLI(filesystemWithUnloadableProject(),
					createSubcommand, apiSubcommand, kindFlagArg, kubebuilderSubcommandHelp)
				Expect(err).To(MatchError(ContainSubstring("version 1 is not supported")))
			})

			It("should ignore an invalid PROJECT path for the help subcommand", func() {
				args := os.Args
				defer func() {
					os.Args = args
				}()
				os.Args = []string{kubebuilderCommandName, kubebuilderSubcommandHelp}

				fs := filesystemWithInvalidProjectPath()

				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithVersion("version string"),
					WithCompletion(),
					WithFilesystem(fs),
				)
				Expect(err).NotTo(HaveOccurred())

				c.cmd.SetArgs([]string{kubebuilderSubcommandHelp})
				Expect(c.cmd.Execute()).To(Succeed())
			})

			It("should ignore an invalid PROJECT path for the root command", func() {
				args := os.Args
				defer func() {
					os.Args = args
				}()
				os.Args = []string{kubebuilderCommandName}

				fs := filesystemWithInvalidProjectPath()

				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithVersion("version string"),
					WithCompletion(),
					WithFilesystem(fs),
				)
				Expect(err).NotTo(HaveOccurred())

				Expect(c.cmd.Execute()).To(Succeed())
			})

			It("should ignore an invalid PROJECT path for root flags without a subcommand", func() {
				args := os.Args
				defer func() {
					os.Args = args
				}()
				os.Args = []string{kubebuilderCommandName, pluginsFlagArg, pluginGoKubebuilderV4}

				fs := filesystemWithInvalidProjectPath()

				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithVersion("version string"),
					WithCompletion(),
					WithFilesystem(fs),
				)
				Expect(err).NotTo(HaveOccurred())

				c.cmd.SetArgs([]string{pluginsFlagArg, pluginGoKubebuilderV4})
				Expect(c.cmd.Execute()).To(Succeed())
			})

			It("should ignore an invalid PROJECT path when help is requested after a subcommand", func() {
				args := os.Args
				defer func() {
					os.Args = args
				}()
				os.Args = []string{kubebuilderCommandName, kubebuilderSubcommandInit, helpFlagArg}

				fs := filesystemWithInvalidProjectPath()

				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithVersion("version string"),
					WithCompletion(),
					WithFilesystem(fs),
				)
				Expect(err).NotTo(HaveOccurred())

				c.cmd.SetArgs([]string{kubebuilderSubcommandInit, helpFlagArg})
				Expect(c.cmd.Execute()).To(Succeed())
			})

			It("should report the current PROJECT path error for commands requiring a project", func() {
				args := os.Args
				defer func() {
					os.Args = args
				}()
				os.Args = []string{
					kubebuilderCommandName, createSubcommand, apiSubcommand,
					kindFlagArg, kubebuilderSubcommandHelp,
				}

				fs := filesystemWithInvalidProjectPath()

				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithFilesystem(fs),
				)
				Expect(err).NotTo(HaveOccurred())

				c.cmd.SetArgs([]string{createSubcommand, apiSubcommand, kindFlagArg, kubebuilderSubcommandHelp})
				Expect(c.cmd.Execute()).To(MatchError(ContainSubstring(`"PROJECT" is a directory`)))
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

		When("the PROJECT file names a registered plugin chain that is not the default", func() {
			var projectPlugin testCreateAPIPlugin

			BeforeEach(func() {
				projectPlugin = newDescribedCreateAPIPlugin()
			})

			It("should build the help of a subcommand from that chain", func() {
				args := []string{createSubcommand, apiSubcommand, helpFlagArg}

				out, helpErr := runCLI(filesystemWithProjectChainProject(), args, args, WithPlugins(projectPlugin))
				Expect(helpErr).NotTo(HaveOccurred())
				Expect(out).To(ContainSubstring(projectChainDescription))
				Expect(out).To(ContainSubstring("--" + projectChainFlag))
				// A flag of the default plugin, which the project does not ask for.
				Expect(out).NotTo(ContainSubstring("--controller"))
			})

			It("should build help requested through the help subcommand from that chain", func() {
				args := []string{kubebuilderSubcommandHelp, createSubcommand, apiSubcommand}

				out, helpErr := runCLI(filesystemWithProjectChainProject(), args, args, WithPlugins(projectPlugin))
				Expect(helpErr).NotTo(HaveOccurred())
				Expect(out).To(ContainSubstring(projectChainDescription))
				Expect(out).To(ContainSubstring("--" + projectChainFlag))
				Expect(out).NotTo(ContainSubstring("--controller"))
			})

			It("should complete the flags of a subcommand from that chain", func() {
				args := []string{cobra.ShellCompRequestCmd, createSubcommand, apiSubcommand, "--mock"}

				completions, completeErr := runCLI(filesystemWithProjectChainProject(), args, args,
					WithPlugins(projectPlugin))
				Expect(completeErr).NotTo(HaveOccurred())
				Expect(completions).To(ContainSubstring("--" + projectChainFlag))
			})

			It("should scaffold with that chain", func() {
				_, runErr := runCLI(filesystemWithProjectChainProject(), createAPIArgs(), createAPIArgs(),
					WithPlugins(projectPlugin))
				Expect(runErr).NotTo(HaveOccurred())
				Expect(projectPlugin.subcommand.scaffolded).To(BeTrue())
			})

			It("should refuse a flag of that plugin when the PROJECT file names only the default chain", func() {
				for _, args := range [][]string{
					append(createAPIArgs(), "--"+projectChainFlag),
					append(createAPIArgs(), pluginsFlagArg, "go/v4", "--"+projectChainFlag),
					append(createAPIArgs(), "--"+projectChainFlag, pluginsFlagArg, "go/v4"),
				} {
					projectPlugin = newDescribedCreateAPIPlugin()
					_, runErr := runCLI(filesystemWithResolvableProject(), args, args, WithPlugins(projectPlugin))
					Expect(runErr).To(MatchError(ContainSubstring("unknown flag: --"+projectChainFlag)), "%v", args)
					Expect(projectPlugin.subcommand.scaffolded).To(BeFalse(), "%v", args)
				}
			})

			It("should fall back to the default plugins only when the configuration cannot be loaded", func() {
				args := []string{createSubcommand, apiSubcommand, helpFlagArg}

				out, helpErr := runCLI(filesystemWithInvalidProject(), args, args, WithPlugins(projectPlugin))
				Expect(helpErr).NotTo(HaveOccurred())
				Expect(out).To(ContainSubstring("--controller"))
				Expect(out).NotTo(ContainSubstring("--" + projectChainFlag))
			})
		})

		When("the PROJECT file names a plugin layout that is no longer supported", func() {
			var filesystem machinery.Filesystem

			BeforeEach(func() {
				filesystem = filesystemWithProject(`domain: example.com
layout:
- go.kubebuilder.io/v3
projectName: test
version: "3"
`)
			})

			It("should patch the layout so that alpha generate can read it", func() {
				// The command scaffolds from the working directory, so only the read matters here.
				_ = executeCLI(filesystem, alphaCommand, generateSubcommand)

				content, readErr := afero.ReadFile(filesystem.FS, yamlstore.DefaultPath)
				Expect(readErr).NotTo(HaveOccurred())
				Expect(string(content)).To(ContainSubstring(pluginGoKubebuilderV4))
				Expect(string(content)).NotTo(ContainSubstring(pluginGoKubebuilderV3))
			})

			It("should leave the layout alone for any other command", func() {
				err = executeCLI(filesystem, createSubcommand, apiSubcommand, kindFlagArg, kindValue)
				Expect(err).To(MatchError(ContainSubstring(pluginGoKubebuilderV3)))

				content, readErr := afero.ReadFile(filesystem.FS, yamlstore.DefaultPath)
				Expect(readErr).NotTo(HaveOccurred())
				Expect(string(content)).To(ContainSubstring(pluginGoKubebuilderV3))
			})

			It("should not patch through a symbolic link", func() {
				skipWithoutSymlinks()
				dir := GinkgoT().TempDir()
				target := filepath.Join(dir, "project.yaml")
				link := filepath.Join(dir, yamlstore.DefaultPath)
				oldProject := []byte(`domain: example.com
layout:
- go.kubebuilder.io/v3
projectName: test
version: "3"
`)
				Expect(os.WriteFile(target, oldProject, 0o644)).To(Succeed())
				Expect(os.Symlink(target, link)).To(Succeed())

				filesystem := machinery.Filesystem{FS: afero.NewBasePathFs(afero.NewOsFs(), dir)}
				err = executeCLI(filesystem, alphaCommand, generateSubcommand)
				Expect(err).To(MatchError(ContainSubstring(pluginGoKubebuilderV3)))

				content, readErr := os.ReadFile(target)
				Expect(readErr).NotTo(HaveOccurred())
				Expect(content).To(Equal(oldProject))
			})
		})

		When("PROJECT is not a regular file", func() {
			It("should not read it while patching the plugin layout for alpha generate", func() {
				// Patching reads raw bytes, and opening a named pipe blocks until something writes.
				fs := &projectReadCountingFs{Fs: &nonRegularProjectFs{Fs: filesystemWithResolvableProject().FS}}

				err = executeCLI(machinery.Filesystem{FS: fs}, alphaCommand, "generate")
				Expect(err).To(MatchError(ContainSubstring(`"PROJECT" is not a regular file`)))
				Expect(fs.reads).To(BeZero())
			})
		})

		When("the command line is malformed", func() {
			DescribeTable("should report an invalid project version flag without a subcommand",
				func(filesystem func() machinery.Filesystem) {
					err = executeCLI(filesystem(), "--"+projectVersionFlag, "not-a-version")
					Expect(err).To(MatchError(ContainSubstring("invalid project version flag")))
				},
				Entry("when the PROJECT file is missing", filesystemWithoutProject),
				Entry("when PROJECT is a directory", filesystemWithProjectDirectory),
				Entry("when the PROJECT file is not valid YAML", filesystemWithInvalidProject),
			)

			It("should report an invalid plugin key without a subcommand", func() {
				err = executeCLI(filesystemWithoutProject(), pluginsFlagArg, "//")
				Expect(err).To(MatchError(ContainSubstring("invalid plugin")))
			})

			DescribeTable("should report it for a subcommand that runs without the configuration",
				func(command ...string) {
					err = executeCLI(filesystemWithProjectDirectory(), append(command, pluginsFlagArg, "//")...)
					Expect(err).To(MatchError(ContainSubstring("invalid plugin")))
				},
				Entry("for the version subcommand", kubebuilderSubcommandVersion),
				Entry("for the help subcommand", kubebuilderSubcommandHelp),
				Entry("for the completion subcommand", kubebuilderSubcommandCompletion, shellZsh),
			)

			DescribeTable("should report it for a command group, whatever the project looks like",
				func(filesystem func() machinery.Filesystem, group string) {
					err = executeCLI(filesystem(), group, pluginsFlagArg, "//")
					Expect(err).To(MatchError(ContainSubstring(`invalid plugin "//"`)))
					// The user made a typo in the flag, so the error is about the flag alone.
					Expect(err).NotTo(MatchError(ContainSubstring(yamlstore.DefaultPath)))
					Expect(err).NotTo(MatchError(ContainSubstring("configuration")))
				},
				commandGroupEntries(func(group commandGroup, state describedFilesystem) TableEntry {
					return Entry(fmt.Sprintf("for %s when %s", group.name, state.description), state.build, group.name)
				}, usableAndBrokenProjectStates),
			)

			DescribeTable("should still display the help of a command group asked for next to it",
				func(group commandGroup, helpArg string) {
					args := []string{group.name, pluginsFlagArg, "//", helpArg}
					out, helpErr := runCLI(filesystemWithoutProject(), args, args)
					Expect(helpErr).NotTo(HaveOccurred())
					Expect(out).To(ContainSubstring(group.help))
				},
				Entry("for create", createGroup, helpFlagArg),
				Entry("for alpha", alphaGroup, helpFlagArg),
				Entry("for create with the help shorthand", createGroup, helpShorthandArg),
				Entry("for alpha with the help shorthand", alphaGroup, helpShorthandArg),
			)
		})

		When("a command group runs on its own", func() {
			DescribeTable("should display its help",
				func(filesystem func() machinery.Filesystem, group commandGroup) {
					out, runErr := runCLI(filesystem(), []string{group.name}, []string{group.name})
					Expect(runErr).NotTo(HaveOccurred())
					Expect(out).To(ContainSubstring(group.help))
				},
				commandGroupEntries(func(group commandGroup, state describedFilesystem) TableEntry {
					return Entry(fmt.Sprintf("for %s when %s", group.name, state.description), state.build, group)
				}, brokenProjectStates),
			)
		})

		When("a tool adds a command group with nothing to run", func() {
			// runToolGroup runs a CLI with a `tools` group that a tool adds with nothing to run, and
			// returns what it wrote and whether the argument check or the persistent hook of the group
			// ran.
			const toolsCommand = "tools"
			runToolGroup := func(filesystem machinery.Filesystem, args ...string) (
				out string, groupRan bool, runErr error,
			) {
				tools := &cobra.Command{Use: toolsCommand}
				tools.Args = func(*cobra.Command, []string) error {
					groupRan = true
					return nil
				}
				tools.PersistentPreRun = func(*cobra.Command, []string) { groupRan = true }
				tools.AddCommand(newExtraCommand("check"))

				out, runErr = runCLI(filesystem, args, args, WithExtraCommands(tools))

				return out, groupRan, runErr
			}

			DescribeTable("should report it for a command group a tool adds",
				func(filesystem func() machinery.Filesystem) {
					_, groupRan, runErr := runToolGroup(filesystem(), toolsCommand, pluginsFlagArg, "//")
					Expect(runErr).To(MatchError(ContainSubstring(`invalid plugin "//"`)))
					Expect(runErr).NotTo(MatchError(ContainSubstring("PROJECT")))
					Expect(groupRan).To(BeFalse())
				},
				Entry("when the PROJECT file is missing", filesystemWithoutProject),
				Entry("when the PROJECT file is usable", filesystemWithResolvableProject),
				Entry("when PROJECT is a directory", filesystemWithProjectDirectory),
			)

			DescribeTable("should still print the help of a command group a tool adds",
				func(filesystem func() machinery.Filesystem, args ...string) {
					out, groupRan, runErr := runToolGroup(filesystem(), args...)
					Expect(runErr).NotTo(HaveOccurred())
					Expect(out).To(ContainSubstring("Usage:"))
					Expect(groupRan).To(BeFalse())
				},
				Entry("on its own when PROJECT is a directory", filesystemWithProjectDirectory, toolsCommand),
				Entry("with a bad plugin key and --help", filesystemWithProjectDirectory,
					toolsCommand, pluginsFlagArg, "//", helpFlagArg),
			)
		})

		When("a tool sets the persistent hook of its command after the CLI is built", func() {
			// buildReportCLI returns a CLI with a `report` command whose persistent hook the test
			// replaces afterwards, and the flags that record which hook ran.
			const reportCommand = "report"
			buildReportCLI := func(filesystem machinery.Filesystem) (*CLI, *cobra.Command) {
				report := newExtraCommand(reportCommand)
				report.PersistentPreRun = func(*cobra.Command, []string) {}

				return buildCLI(filesystem, []string{reportCommand}, WithExtraCommands(report)), report
			}

			It("should run the new hook when there is no PROJECT file", func() {
				reportCLI, report := buildReportCLI(filesystemWithoutProject())
				newHookRan := false
				report.PersistentPreRun = func(*cobra.Command, []string) { newHookRan = true }

				_, runErr := runBuiltCLI(reportCLI, []string{reportCommand})
				Expect(runErr).NotTo(HaveOccurred())
				Expect(newHookRan).To(BeTrue())
			})

			It("should still stop the command before the new hook when PROJECT is a directory", func() {
				reportCLI, report := buildReportCLI(filesystemWithProjectDirectory())
				newHookRan := false
				report.PersistentPreRun = func(*cobra.Command, []string) { newHookRan = true }

				_, runErr := runBuiltCLI(reportCLI, []string{reportCommand})
				Expect(runErr).To(HaveOccurred())
				Expect(newHookRan).To(BeFalse())
			})

			// runSummary builds a CLI with a `report` command whose persistent hook serves its
			// `summary` child, clears the hook of the child after the CLI is built, runs the child,
			// and returns which parts ran.
			const summaryCommand = "summary"
			runSummary := func(filesystem machinery.Filesystem) (
				parentHookRan, clearedHookRan, summaryRan bool, runErr error,
			) {
				report := &cobra.Command{Use: reportCommand}
				report.PersistentPreRun = func(*cobra.Command, []string) { parentHookRan = true }
				summary := &cobra.Command{Use: summaryCommand, Run: func(*cobra.Command, []string) { summaryRan = true }}
				summary.PersistentPreRun = func(*cobra.Command, []string) { clearedHookRan = true }
				report.AddCommand(summary)
				args := []string{reportCommand, summaryCommand}
				reportCLI := buildCLI(filesystem, args, WithExtraCommands(report))
				summary.PersistentPreRun = nil

				_, runErr = runBuiltCLI(reportCLI, args)

				return parentHookRan, clearedHookRan, summaryRan, runErr
			}

			DescribeTable("should run the hook of the parent when the tool clears the hook of the child",
				func(filesystem func() machinery.Filesystem) {
					parentHookRan, clearedHookRan, summaryRan, runErr := runSummary(filesystem())
					Expect(runErr).NotTo(HaveOccurred())
					Expect(parentHookRan).To(BeTrue())
					Expect(clearedHookRan).To(BeFalse())
					Expect(summaryRan).To(BeTrue())
				},
				Entry("when there is no PROJECT file", filesystemWithoutProject),
				Entry("when the PROJECT file is usable", filesystemWithResolvableProject),
			)

			It("should still stop the child before the hook of the parent when PROJECT is a directory", func() {
				parentHookRan, _, summaryRan, runErr := runSummary(filesystemWithProjectDirectory())
				Expect(runErr).To(MatchError(ContainSubstring(`"PROJECT" is a directory`)))
				Expect(parentHookRan).To(BeFalse())
				Expect(summaryRan).To(BeFalse())
			})

			It("should run the new hook of the parent for a child that has none when the tool replaces it", func() {
				oldHookRan, newHookRan, summaryRan := false, false, false
				report := &cobra.Command{Use: reportCommand}
				report.PersistentPreRun = func(*cobra.Command, []string) { oldHookRan = true }
				report.AddCommand(&cobra.Command{
					Use: summaryCommand,
					Run: func(*cobra.Command, []string) { summaryRan = true },
				})
				args := []string{reportCommand, summaryCommand}
				reportCLI := buildCLI(filesystemWithoutProject(), args, WithExtraCommands(report))
				report.PersistentPreRun = func(*cobra.Command, []string) { newHookRan = true }

				_, runErr := runBuiltCLI(reportCLI, args)
				Expect(runErr).NotTo(HaveOccurred())
				Expect(newHookRan).To(BeTrue())
				Expect(oldHookRan).To(BeFalse())
				Expect(summaryRan).To(BeTrue())
			})
		})

		When("a tool replaces the hooks and body of init after the CLI is built", func() {
			// runInit builds the CLI, replaces the pre-run hook, body and post-run hook of init with ones
			// that record their names, runs init and returns what ran.
			runInit := func(filesystem machinery.Filesystem) ([]string, error) {
				args := []string{kubebuilderSubcommandInit}
				initCLI := buildCLI(filesystem, args)
				initCmd, _, findErr := initCLI.cmd.Find(args)
				Expect(findErr).NotTo(HaveOccurred())
				Expect(initCmd.Name()).To(Equal(kubebuilderSubcommandInit))

				var ran []string
				initCmd.PreRunE = func(*cobra.Command, []string) error { ran = append(ran, "pre-run"); return nil }
				initCmd.RunE = func(*cobra.Command, []string) error { ran = append(ran, "run"); return nil }
				initCmd.PostRunE = func(*cobra.Command, []string) error { ran = append(ran, "post-run"); return nil }

				_, runErr := runBuiltCLI(initCLI, args)

				return ran, runErr
			}

			It("should run them when there is no PROJECT file", func() {
				ran, runErr := runInit(filesystemWithoutProject())
				Expect(runErr).NotTo(HaveOccurred())
				Expect(ran).To(Equal([]string{"pre-run", "run", "post-run"}))
			})

			DescribeTable("should run none of them when PROJECT cannot be used",
				func(filesystem func() machinery.Filesystem) {
					ran, runErr := runInit(filesystem())
					Expect(runErr).To(HaveOccurred())
					Expect(ran).To(BeEmpty())
				},
				Entry("when PROJECT is a directory", filesystemWithProjectDirectory),
				Entry("when the PROJECT file is not valid YAML", filesystemWithInvalidProject),
				Entry("when the version of the PROJECT file is not supported", filesystemWithUnloadableProject),
				Entry("when the layout names a plugin the CLI does not have", filesystemWithUnresolvableProject),
			)
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

			// Completing the flags of create api reads PROJECT, since they come from the plugin chain it names.
			DescribeTable("should offer the flags of the default plugins when PROJECT cannot be used",
				func(filesystem func() machinery.Filesystem) {
					completions, completeErr := completeCLI(filesystem(), createSubcommand, apiSubcommand, "--")
					Expect(completeErr).NotTo(HaveOccurred())
					Expect(completions).To(ContainSubstring(kindFlagArg))
				},
				Entry("when the PROJECT file is not valid YAML", filesystemWithInvalidProject),
				Entry("when the PROJECT project version is not supported", filesystemWithUnloadableProject),
				Entry("when the PROJECT plugin chain cannot be resolved", filesystemWithUnresolvableProject),
				Entry("when PROJECT is a directory", filesystemWithProjectDirectory),
				Entry("when PROJECT is not a regular file", filesystemWithNonRegularProject),
			)

			It("should not open PROJECT when it is not a regular file", func() {
				// Opening a named pipe blocks until something writes to it, so completion would hang.
				fs := &projectReadCountingFs{Fs: &nonRegularProjectFs{Fs: filesystemWithResolvableProject().FS}}

				completions, completeErr := completeCLI(machinery.Filesystem{FS: fs}, createSubcommand, apiSubcommand, "--")
				Expect(completeErr).NotTo(HaveOccurred())
				Expect(completions).To(ContainSubstring(kindFlagArg))
				Expect(fs.reads).To(BeZero())
			})
		})

		DescribeTable("should report what occupies the configuration path",
			func(filesystem func() machinery.Filesystem, occupant string, command []string) {
				fs := filesystem()

				err = executeCLI(fs, command...)
				Expect(err).To(MatchError(ContainSubstring(occupant)))
				Expect(err).NotTo(MatchError(ContainSubstring("initialized")))
				expectNothingScaffolded(fs)
			},
			Entry("for init when PROJECT is a directory", filesystemWithProjectDirectory,
				`"PROJECT" is a directory`, []string{kubebuilderSubcommandInit}),
			Entry("for edit when PROJECT is a directory", filesystemWithProjectDirectory,
				`"PROJECT" is a directory`, []string{editSubcommand}),
			Entry("for create api when PROJECT is a directory", filesystemWithProjectDirectory,
				`"PROJECT" is a directory`, []string{createSubcommand, apiSubcommand, kindFlagArg, kindValue}),
			Entry("for create webhook when PROJECT is a directory", filesystemWithProjectDirectory,
				`"PROJECT" is a directory`,
				[]string{createSubcommand, webhookSubcommand, kindFlagArg, kindValue, "--defaulting"}),
			Entry("for alpha generate when PROJECT is a directory", filesystemWithProjectDirectory,
				`"PROJECT" is a directory`, []string{alphaCommand, generateSubcommand}),
			Entry("for init when PROJECT is not a regular file", filesystemWithNonRegularProject,
				`"PROJECT" is not a regular file`, []string{kubebuilderSubcommandInit}),
			Entry("for edit when PROJECT is not a regular file", filesystemWithNonRegularProject,
				`"PROJECT" is not a regular file`, []string{editSubcommand}),
			Entry("for alpha generate when PROJECT is not a regular file", filesystemWithNonRegularProject,
				`"PROJECT" is not a regular file`, []string{alphaCommand, generateSubcommand}),
		)

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

		When("the plugin chain comes from a flag", func() {
			It("should keep it as it was given however often the arguments are read", func() {
				// Resolving the chain again must not grow it: the CLI reads the arguments once
				// while it builds the command tree and again when it falls back to the defaults.
				fs := filesystemWithProjectDirectory()

				pluginKey := plugin.KeyFor(&golangv4.Plugin{})

				originalArgs := os.Args
				DeferCleanup(func() { os.Args = originalArgs })
				os.Args = []string{kubebuilderCommandName, kubebuilderSubcommandInit, pluginsFlagArg, pluginKey}

				c, err = New(
					WithPlugins(&golangv4.Plugin{}),
					WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
					WithDefaultProjectVersion(projectVersion),
					WithFilesystem(fs),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c.pluginKeys).To(ConsistOf(pluginKey))
				Expect(c.resolvedPlugins).To(HaveLen(1))
			})
		})

		When("help was displayed", func() {
			It("should exit without an error", func() {
				originalArgs := os.Args
				DeferCleanup(func() { os.Args = originalArgs })
				os.Args = []string{kubebuilderCommandName, kubebuilderSubcommandInit, pluginsFlagArg, helpFlagArg}

				c, err = New(WithPlugins(&golangv4.Plugin{}), WithDefaultPlugins(projectVersion, &golangv4.Plugin{}))
				Expect(err).NotTo(HaveOccurred())
				c.cmd.SetOut(io.Discard)
				c.cmd.SetArgs([]string{kubebuilderSubcommandInit, pluginsFlagArg, helpFlagArg})

				Expect(c.Run()).To(Succeed())
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

var _ = Describe("isCommandGroup", func() {
	It("should hold only the commands that display their help as command groups", func() {
		var groups []string
		var visit func(cmd *cobra.Command)
		visit = func(cmd *cobra.Command) {
			if isCommandGroup(cmd) {
				groups = append(groups, cmd.CommandPath())
			}
			for _, subcommand := range cmd.Commands() {
				visit(subcommand)
			}
		}
		visit(buildCLI(filesystemWithoutProject(), nil).cmd)

		Expect(groups).To(ConsistOf(kubebuilderCommandName+" "+createSubcommand,
			kubebuilderCommandName+" "+alphaCommand))
	})
})

var _ = Describe("isSubcommandWithoutConfig", func() {
	DescribeTable("should not read the project configuration",
		func(args ...string) {
			Expect(isSubcommandWithoutConfig(args)).To(BeTrue())
		},
		Entry("with root flags only", pluginsFlagArg, pluginGoKubebuilderV4),
		Entry("for help without a target", kubebuilderSubcommandHelp),
		Entry("for the version subcommand", kubebuilderSubcommandVersion),
		Entry("for the completion subcommand", kubebuilderSubcommandCompletion, shellZsh),
		Entry("for help on the version subcommand", kubebuilderSubcommandHelp, kubebuilderSubcommandVersion),
		Entry("for help on the completion subcommand", kubebuilderSubcommandHelp, kubebuilderSubcommandCompletion),
		Entry("for completing a command group", cobra.ShellCompRequestCmd, createSubcommand, ""),
		Entry("for completing a subcommand name", cobra.ShellCompRequestCmd, createSubcommand, "a"),
		Entry("for completing the flags of alpha generate", cobra.ShellCompRequestCmd, alphaCommand,
			generateSubcommand, "--"),
	)

	DescribeTable("should read the project configuration",
		func(args ...string) {
			Expect(isSubcommandWithoutConfig(args)).To(BeFalse())
		},
		Entry("with help as a flag value", createSubcommand, apiSubcommand, kindFlagArg, kubebuilderSubcommandHelp),
		Entry("with help as a flag value in the equals form", createSubcommand, apiSubcommand,
			kindFlagArg+"="+kubebuilderSubcommandHelp),
		Entry("with help as an argument of another subcommand", createSubcommand, apiSubcommand, kubebuilderSubcommandHelp),
		// Help output describes the plugin chain of the project, so the configuration is read.
		Entry("with the help flag of a subcommand", kubebuilderSubcommandInit, helpFlagArg),
		Entry("with a dangling plugins flag", kubebuilderSubcommandInit, pluginsFlagArg),
		Entry("for a subcommand that requires a project", kubebuilderSubcommandInit),
		Entry("for a subcommand added to the CLI", "docs"),
		// The flags of these commands come from the plugin chain the project names.
		Entry("for completing the flags of create api", cobra.ShellCompRequestCmd, createSubcommand, apiSubcommand, "--"),
		Entry("for completing the flags of create webhook without descriptions", cobra.ShellCompNoDescRequestCmd,
			createSubcommand, webhookSubcommand, "--"),
		Entry("for completing the flags of edit", cobra.ShellCompRequestCmd, editSubcommand, ""),
		Entry("for completing the flags of init", cobra.ShellCompRequestCmd, kubebuilderSubcommandInit, "--"),
		// A tool may complete the name of a command, then run that command on the same CLI.
		Entry("for completing the name of create api", cobra.ShellCompRequestCmd, createSubcommand, apiSubcommand),
		Entry("for completing the name of edit without descriptions", cobra.ShellCompNoDescRequestCmd, editSubcommand),
		// A tool may build the CLI before it sets the arguments of the command it runs.
		Entry("without arguments"),
	)
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
	{"PROJECT is a directory holding a file", filesystemWithPopulatedProjectDirectory},
	{"PROJECT is not a regular file", filesystemWithNonRegularProject},
}

// usableAndBrokenProjectStates adds a usable PROJECT file to brokenProjectStates.
var usableAndBrokenProjectStates = append([]describedFilesystem{
	{"the PROJECT file can be used", filesystemWithResolvableProject},
}, brokenProjectStates...)

// commandGroup is a command that only holds other subcommands, with a line of the help it displays.
type commandGroup struct {
	name string
	help string
}

var (
	createGroup   = commandGroup{createSubcommand, "Scaffold a Kubernetes API or webhook."}
	alphaGroup    = commandGroup{alphaCommand, "Alpha subcommands are for unstable features."}
	commandGroups = []commandGroup{createGroup, alphaGroup}
)

// commandGroupEntries returns one table entry for each command group in each of the project states.
func commandGroupEntries(
	entry func(commandGroup, describedFilesystem) TableEntry,
	states []describedFilesystem,
) []TableEntry {
	entries := make([]TableEntry, 0, len(commandGroups)*len(states))
	for _, group := range commandGroups {
		for _, state := range states {
			entries = append(entries, entry(group, state))
		}
	}

	return entries
}

// configFreeCommands are the commands that do not consume the project configuration.
var configFreeCommands = []describedCommand{
	{"the root command", nil},
	{"the version subcommand", []string{kubebuilderSubcommandVersion}},
	{"the help subcommand", []string{kubebuilderSubcommandHelp}},
	{"help on the help subcommand", []string{kubebuilderSubcommandHelp, kubebuilderSubcommandHelp}},
	{"the completion subcommand", []string{kubebuilderSubcommandCompletion, shellZsh}},
	{"the bash completion subcommand", []string{kubebuilderSubcommandCompletion, "bash"}},
	{"the fish completion subcommand", []string{kubebuilderSubcommandCompletion, "fish"}},
	{"the powershell completion subcommand", []string{kubebuilderSubcommandCompletion, "powershell"}},
	{"the bare completion subcommand", []string{kubebuilderSubcommandCompletion}},
	{"help on the completion subcommand", []string{kubebuilderSubcommandHelp, kubebuilderSubcommandCompletion}},
	{"version help", []string{kubebuilderSubcommandVersion, helpFlagArg}},
	{"version help shorthand", []string{kubebuilderSubcommandVersion, helpShorthandArg}},
	{"version help set to true", []string{kubebuilderSubcommandVersion, helpFlagArg + "=true"}},
	{"version help shorthand set to true", []string{kubebuilderSubcommandVersion, helpShorthandArg + "=true"}},
	{"completion help", []string{kubebuilderSubcommandCompletion, helpFlagArg}},
	{"completion help shorthand", []string{kubebuilderSubcommandCompletion, helpShorthandArg}},
	{"completion help set to true", []string{kubebuilderSubcommandCompletion, helpFlagArg + "=true"}},
	{"completion help shorthand set to true", []string{kubebuilderSubcommandCompletion, helpShorthandArg + "=true"}},
	{"the help flag of a subcommand", []string{kubebuilderSubcommandInit, helpFlagArg}},
	{"the help shorthand of a subcommand", []string{kubebuilderSubcommandInit, helpShorthandArg}},
	{"the help flag of a subcommand set to a value", []string{kubebuilderSubcommandInit, helpFlagArg + "=true"}},
	{"the help shorthand of a subcommand set to a value", []string{kubebuilderSubcommandInit, helpShorthandArg + "=true"}},
	{"help for the version subcommand", []string{kubebuilderSubcommandHelp, kubebuilderSubcommandVersion}},
	{"the help flag of the root command", []string{helpFlagArg}},
	{"the help shorthand of the root command", []string{helpShorthandArg}},
	{"the help flag of the root command set to a value", []string{helpFlagArg + "=true"}},
	{"the help shorthand of the root command set to a value", []string{helpShorthandArg + "=true"}},
	{"root flags without a subcommand", []string{pluginsFlagArg, pluginGoKubebuilderV4}},
	{"root flags without a subcommand in the equals form", []string{pluginsFlagArg + "=" + pluginGoKubebuilderV4}},
	{"the bare create command", []string{createSubcommand}},
	{"the bare alpha command", []string{alphaCommand}},
	{"a completion request", []string{cobra.ShellCompRequestCmd, createSubcommand, ""}},
	{"a completion request for a command group", []string{cobra.ShellCompRequestCmd, alphaCommand, ""}},
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

	DescribeTable("should not read the PROJECT file for root help or completion",
		func(args ...string) {
			fs := &projectReadCountingFs{Fs: filesystemWithUnresolvableProject().FS}

			Expect(executeCLI(machinery.Filesystem{FS: fs}, args...)).To(Succeed())
			Expect(fs.reads).To(BeZero())
		},
		Entry("for root help", helpFlagArg),
		Entry("for root help shorthand", helpShorthandArg),
		Entry("for a shell completion request", cobra.ShellCompRequestCmd, createSubcommand, ""),
		Entry("for a shell completion request without descriptions", cobra.ShellCompNoDescRequestCmd,
			createSubcommand, ""),
	)
})

// A completion request for a command the plugin chain builds reads a usable PROJECT file, so the
// layout's plugins add their flags.
var _ = Describe("a shell completion request", func() {
	var projectPlugin testCreateAPIPlugin

	BeforeEach(func() {
		projectPlugin = newDescribedCreateAPIPlugin()
	})

	// Cobra offers only the required flags for a bare "--", so the request types the start of the flag.
	complete := func(filesystem machinery.Filesystem, request, typed string) string {
		GinkgoHelper()

		args := []string{request, createSubcommand, apiSubcommand, typed}
		out, err := runCLI(filesystem, args, args, WithPlugins(projectPlugin))
		Expect(err).NotTo(HaveOccurred())

		return out
	}

	It("should offer only the default plugins with no PROJECT file", func() {
		out := complete(filesystemWithoutProject(), cobra.ShellCompRequestCmd, "--mo")
		Expect(out).NotTo(ContainSubstring("--" + projectChainFlag))
	})

	// A tool may set one of its own flags in code and answer a request before it runs a command on
	// the same CLI. Answering the request must leave the value the tool set.
	It("should keep the value of a flag the tool set when the request is answered through Run", func() {
		const debugFlag = "debug"

		c := buildCLI(filesystemWithoutProject(), nil)
		c.cmd.PersistentFlags().Bool(debugFlag, false, "print debug output")
		Expect(c.cmd.PersistentFlags().Set(debugFlag, "true")).To(Succeed())
		c.cmd.SetOut(io.Discard)
		c.cmd.SetArgs([]string{cobra.ShellCompRequestCmd, kubebuilderSubcommandVersion, ""})

		Expect(c.Run()).To(Succeed())
		Expect(c.cmd.PersistentFlags().Lookup(debugFlag).Value.String()).To(Equal("true"))
	})
})

// A tool may build the CLI while the program was started with no arguments, then set the arguments
// of the command it runs and call Run, the way a sample generator drives the CLI. The command tree
// is built before the command is known, so it comes from a usable PROJECT file.
var _ = Describe("a CLI built before the program names a command", func() {
	var projectPlugin testCreateAPIPlugin

	BeforeEach(func() {
		projectPlugin = newDescribedCreateAPIPlugin()
	})

	// runLater builds the CLI with no program arguments, then sets them to args and calls Run.
	runLater := func(filesystem machinery.Filesystem, args ...string) (string, error) {
		GinkgoHelper()

		c := buildCLI(filesystem, nil, WithPlugins(projectPlugin))
		var out bytes.Buffer
		c.cmd.SetOut(&out)
		setProgramArgs(args...)
		err := c.Run()

		return out.String(), err
	}

	It("should offer the flags of the plugins the layout names", func() {
		out, err := runLater(filesystemWithProjectChainProject(), createSubcommand, apiSubcommand, helpFlagArg)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("--" + projectChainFlag))
	})

	DescribeTable("should still answer the root help when the configuration cannot be used",
		func(filesystem func() machinery.Filesystem) {
			out, err := runLater(filesystem())
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(ContainSubstring("Usage:"))
		},
		Entry("when PROJECT is a directory", filesystemWithProjectDirectory),
		Entry("when PROJECT cannot be parsed", filesystemWithInvalidProject),
		Entry("when the PROJECT plugin chain cannot be resolved", filesystemWithUnresolvableProject),
	)
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

	It("should report the failure for a command added to the CLI", func() {
		docs := newExtraCommand(docsSubcommand)

		err := executeEmbeddedCLI(filesystemWithInvalidProject(), programArgs, []string{docsSubcommand},
			WithExtraCommands(docs))
		Expect(err).To(MatchError(ContainSubstring("failed to determine config version")))
	})

	It("should report the failure for an added command when the program needs no configuration", func() {
		// The program was started with a command that reads no configuration, so the startup left
		// it unread. The command the caller runs does consume it, so the failure still surfaces
		// and that command never runs.
		docsRan := false
		docs := &cobra.Command{Use: docsSubcommand, RunE: func(*cobra.Command, []string) error {
			docsRan = true

			return nil
		}}

		err := executeEmbeddedCLI(filesystemWithInvalidProject(), []string{kubebuilderSubcommandVersion},
			[]string{docsSubcommand}, WithExtraCommands(docs))
		Expect(err).To(MatchError(ContainSubstring("failed to determine config version")))
		Expect(docsRan).To(BeFalse())
	})

	It("should report the failure for an added command before its own argument check runs", func() {
		// Cobra validates the arguments of the command it executes before it runs any hook, so a
		// check that sits only in the hooks would let the tool's own argument check run first.
		argsChecked := false
		docs := &cobra.Command{
			Use: docsSubcommand,
			Args: func(*cobra.Command, []string) error {
				argsChecked = true

				return nil
			},
			RunE: func(*cobra.Command, []string) error { return nil },
		}

		err := executeEmbeddedCLI(filesystemWithInvalidProject(), programArgs, []string{docsSubcommand},
			WithExtraCommands(docs))
		Expect(err).To(MatchError(ContainSubstring("failed to determine config version")))
		Expect(argsChecked).To(BeFalse())
	})

	It("should hand an added command's own argument check the arguments with no configuration", func() {
		var checked []string
		docs := &cobra.Command{
			Use: docsSubcommand,
			Args: func(_ *cobra.Command, args []string) error {
				checked = args

				return nil
			},
			RunE: func(*cobra.Command, []string) error { return nil },
		}

		Expect(executeEmbeddedCLI(filesystemWithoutProject(), programArgs, []string{docsSubcommand, "changelog"},
			WithExtraCommands(docs))).To(Succeed())
		Expect(checked).To(Equal([]string{"changelog"}))
	})

	It("should keep the answer of an added command's own argument check", func() {
		docs := &cobra.Command{
			Use:  docsSubcommand,
			Args: cobra.ExactArgs(1),
			RunE: func(*cobra.Command, []string) error { return nil },
		}

		err := executeEmbeddedCLI(filesystemWithoutProject(), programArgs, []string{docsSubcommand},
			WithExtraCommands(docs))
		Expect(err).To(MatchError(ContainSubstring("accepts 1 arg(s), received 0")))
	})

	DescribeTable("should report the failure for a command that consumes the configuration",
		func(filesystem func() machinery.Filesystem, expected string) {
			err := executeEmbeddedCLI(filesystem(), []string{kubebuilderSubcommandVersion}, programArgs)
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
		treePlugin := newTestCreateAPIPlugin("tree.example.com", plugin.Version{Number: 1})
		projectPlugin := newTestCreateAPIPlugin(projectChainPluginName, plugin.Version{Number: 1})

		err := executeEmbeddedCLI(filesystemWithProjectChainProject(),
			[]string{kubebuilderSubcommandVersion},
			createAPIArgs(),
			WithPlugins(treePlugin, projectPlugin),
			WithDefaultPlugins(config.Version{Number: 3}, treePlugin),
		)
		Expect(err).To(MatchError(ContainSubstring("the command tree was built for the plugin chain")))
		Expect(treePlugin.subcommand.scaffolded).To(BeFalse())
		Expect(projectPlugin.subcommand.scaffolded).To(BeFalse())
	})

	It("should scaffold with the plugin chain the project names when the command tree was built for it", func() {
		projectPlugin := newTestCreateAPIPlugin(projectChainPluginName, plugin.Version{Number: 1})
		projectVersion := config.Version{Number: 3}

		setProgramArgs(kubebuilderSubcommandVersion)
		c, err := New(
			WithPlugins(projectPlugin),
			WithDefaultPlugins(projectVersion, projectPlugin),
			WithDefaultProjectVersion(projectVersion),
			WithVersion("version string"),
			WithFilesystem(filesystemWithProjectChainProject()),
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = runBuiltCLI(c, createAPIArgs())
		Expect(err).NotTo(HaveOccurred())
		Expect(projectPlugin.subcommand.scaffolded).To(BeTrue())
	})

	It("should keep refusing that plugin chain when the caller runs a command again", func() {
		// A tool keeps one CLI and runs more than one command through it. Reading the project
		// configuration for the first command must not make the chain of the next one look like
		// the chain the command tree was built for.
		extraPlugin := newTestCreateAPIPlugin(projectChainPluginName, plugin.Version{Number: 1})

		originalArgs := os.Args
		DeferCleanup(func() { os.Args = originalArgs })
		os.Args = []string{kubebuilderCommandName, kubebuilderSubcommandVersion}

		testProjectVersion := config.Version{Number: 3}
		c, err := New(
			WithPlugins(&golangv4.Plugin{}, extraPlugin),
			WithDefaultPlugins(testProjectVersion, extraPlugin),
			WithDefaultProjectVersion(testProjectVersion),
			WithVersion("version string"),
			WithFilesystem(filesystemWithResolvableProject()),
		)
		Expect(err).NotTo(HaveOccurred())

		var out bytes.Buffer
		c.cmd.SetOut(&out)
		for range 2 {
			c.cmd.SetArgs([]string{createSubcommand, apiSubcommand, kindFlagArg, kindValue})
			Expect(c.cmd.Execute()).To(MatchError(
				ContainSubstring("the command tree was built for the plugin chain")))
		}
		Expect(extraPlugin.subcommand.scaffolded).To(BeFalse())
	})

	It("should read the skipped configuration when the command tree was built for its plugin chain", func() {
		fs := &projectReadCountingFs{Fs: filesystemWithResolvableProject().FS}

		// The command reaches the plugins of the project, so it stops at the missing scaffold and
		// not at the plugin chain.
		err := executeEmbeddedCLI(machinery.Filesystem{FS: fs},
			[]string{kubebuilderSubcommandVersion}, []string{editSubcommand})
		Expect(err).To(MatchError(ContainSubstring("failed to edit scaffold")))
		Expect(err).NotTo(MatchError(ContainSubstring("the command tree was built for the plugin chain")))
		Expect(fs.reads).NotTo(BeZero())
	})

	// A tool may also hand the command it picked to Run instead of executing the tree itself.
	Context("through Run", func() {
		run := func(c *CLI, args ...string) error {
			c.cmd.SetOut(io.Discard)
			c.cmd.SetErr(io.Discard)
			c.cmd.SetArgs(args)

			return c.Run()
		}

		DescribeTable("should run version when the program was started with a command that consumes the configuration",
			func(filesystem func() machinery.Filesystem) {
				Expect(run(buildCLI(filesystem(), programArgs), kubebuilderSubcommandVersion)).To(Succeed())
			},
			Entry("when PROJECT is a directory", filesystemWithProjectDirectory),
			Entry("when PROJECT cannot be parsed", filesystemWithInvalidProject),
			Entry("when the PROJECT plugin chain cannot be resolved", filesystemWithUnresolvableProject),
		)

		It("should report the failure for an added command when the program was started with version", func() {
			docsRan := false
			docs := &cobra.Command{Use: docsSubcommand, RunE: func(*cobra.Command, []string) error {
				docsRan = true

				return nil
			}}
			c := buildCLI(filesystemWithInvalidProject(), []string{kubebuilderSubcommandVersion}, WithExtraCommands(docs))

			Expect(run(c, docsSubcommand)).To(MatchError(ContainSubstring("failed to determine config version")))
			Expect(docsRan).To(BeFalse())
		})
	})
})

// A tool adds extra commands with WithExtraCommands and WithExtraAlphaCommands. The CLI cannot know
// whether such a command reads the project, so a configuration it cannot use has to stop the
// command before its own hooks run.
var _ = Describe("extra commands", func() {
	// newDocsWithGenerate returns a docs command whose generate subcommand has a persistent hook of
	// its own.
	newDocsWithGenerate := func() (*cobra.Command, *bool) {
		docs := &cobra.Command{Use: docsSubcommand}
		generate, hookRan := newExtraCommandWithHook(generateSubcommand, setPersistentPreRunE)
		docs.AddCommand(generate)

		return docs, hookRan
	}

	DescribeTable("should stop the command before its own hook when PROJECT is a directory",
		func(setHook func(*cobra.Command, *bool)) {
			docs, hookRan := newExtraCommandWithHook(docsSubcommand, setHook)

			err := executeCLIWith(filesystemWithProjectDirectory(), WithExtraCommands(docs), docsSubcommand)
			expectStoppedBeforeHook(err, hookRan)
		},
		preRunHookEntries(),
	)

	DescribeTable("should run the hook of the command when PROJECT can be used",
		func(setHook func(*cobra.Command, *bool)) {
			docs, hookRan := newExtraCommandWithHook(docsSubcommand, setHook)

			err := executeCLIWith(filesystemWithResolvableProject(), WithExtraCommands(docs), docsSubcommand)
			expectHookRan(err, hookRan)
		},
		preRunHookEntries(),
	)

	It("should stop a subcommand before its own persistent hook when PROJECT is a directory", func() {
		docs, hookRan := newDocsWithGenerate()

		err := executeCLIWith(filesystemWithProjectDirectory(), WithExtraCommands(docs),
			docsSubcommand, generateSubcommand)
		expectStoppedBeforeHook(err, hookRan)
	})

	It("should run the persistent hook of a subcommand when PROJECT can be used", func() {
		docs, hookRan := newDocsWithGenerate()

		err := executeCLIWith(filesystemWithResolvableProject(), WithExtraCommands(docs),
			docsSubcommand, generateSubcommand)
		expectHookRan(err, hookRan)
	})

	It("should run the persistent hook of a subcommand when there is no PROJECT file", func() {
		docs, hookRan := newDocsWithGenerate()

		err := executeCLIWith(filesystemWithoutProject(), WithExtraCommands(docs),
			docsSubcommand, generateSubcommand)
		expectHookRan(err, hookRan)
	})

	It("should stop an alpha command before its own hook when PROJECT is a directory", func() {
		docs, hookRan := newExtraCommandWithHook(docsSubcommand, setPersistentPreRunE)

		err := executeCLIWith(filesystemWithProjectDirectory(), WithExtraAlphaCommands(docs),
			alphaCommand, docsSubcommand)
		expectStoppedBeforeHook(err, hookRan)
	})

	It("should run the hook of an alpha command when PROJECT can be used", func() {
		docs, hookRan := newExtraCommandWithHook(docsSubcommand, setPersistentPreRunE)

		err := executeCLIWith(filesystemWithResolvableProject(), WithExtraAlphaCommands(docs),
			alphaCommand, docsSubcommand)
		expectHookRan(err, hookRan)
	})

	It("should run the hook of an alpha command when there is no PROJECT file", func() {
		docs, hookRan := newExtraCommandWithHook(docsSubcommand, setPersistentPreRunE)

		err := executeCLIWith(filesystemWithoutProject(), WithExtraAlphaCommands(docs),
			alphaCommand, docsSubcommand)
		expectHookRan(err, hookRan)
	})

	It("should report the error of the hook", func() {
		hookErr := errors.New("hook failed")
		docs := newExtraCommand(docsSubcommand)
		docs.PersistentPreRunE = func(*cobra.Command, []string) error { return hookErr }

		err := executeCLIWith(filesystemWithResolvableProject(), WithExtraCommands(docs), docsSubcommand)
		Expect(err).To(MatchError(hookErr))
	})

	It("should report a plugin chain the command tree was not built for", func() {
		docs, hookRan := newExtraCommandWithHook(docsSubcommand, setPersistentPreRunE)
		projectPlugin := newTestCreateAPIPlugin(projectChainPluginName, plugin.Version{Number: 1})

		err := executeEmbeddedCLI(filesystemWithProjectChainProject(),
			[]string{kubebuilderSubcommandVersion}, []string{docsSubcommand},
			WithExtraCommands(docs), WithPlugins(projectPlugin))
		Expect(err).To(MatchError(ContainSubstring("the command tree was built for the plugin chain")))
		Expect(*hookRan).To(BeFalse())
	})

	It("should stop the command when the tool sets the flag-name mapping again after the CLI is built", func() {
		docs, hookRan := newExtraCommandWithHook(docsSubcommand, setPersistentPreRunE)
		c := buildCLI(filesystemWithProjectDirectory(), []string{docsSubcommand}, WithExtraCommands(docs))
		c.Command().SetGlobalNormalizationFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
			return pflag.NormalizedName(strings.ReplaceAll(name, "_", "-"))
		})

		_, err := runBuiltCLI(c, []string{docsSubcommand})
		expectStoppedBeforeHook(err, hookRan)
	})
})

// A tool builds several CLIs in one process, as its own tests do. Each CLI has to answer for the
// project it was built over, whatever the build order.
var _ = Describe("several CLIs in one process", func() {
	var generateArgs []string

	BeforeEach(func() {
		// The input directory does not exist, so a CLI that gets to the command stops at the
		// validation of the command instead of scaffolding.
		missingDir := filepath.Join(GinkgoT().TempDir(), "missing")
		generateArgs = []string{alphaCommand, generateSubcommand, "--input-dir", missingDir}
	})

	// answerAlone returns what a CLI built over the filesystem answers when it is the only one.
	answerAlone := func(filesystem func() machinery.Filesystem) error {
		GinkgoHelper()

		err := executeCLI(filesystem(), generateArgs...)
		Expect(err).To(HaveOccurred())

		return err
	}

	expectEachToAnswerForItsProject := func(unusable, usable *CLI) {
		GinkgoHelper()

		_, err := runBuiltCLI(usable, generateArgs)
		Expect(err).To(MatchError(answerAlone(filesystemWithResolvableProject).Error()))

		_, err = runBuiltCLI(unusable, generateArgs)
		Expect(err).To(MatchError(answerAlone(filesystemWithUnresolvableProject).Error()))
	}

	It("should answer for its own project when the CLI over the unusable project is built first", func() {
		unusable := buildCLI(filesystemWithUnresolvableProject(), generateArgs)
		usable := buildCLI(filesystemWithResolvableProject(), generateArgs)

		expectEachToAnswerForItsProject(unusable, usable)
	})

	It("should answer for its own project when the CLI over the usable project is built first", func() {
		usable := buildCLI(filesystemWithResolvableProject(), generateArgs)
		unusable := buildCLI(filesystemWithUnresolvableProject(), generateArgs)

		expectEachToAnswerForItsProject(unusable, usable)
	})
})

var _ = Describe("help requested through the plugins flag", func() {
	DescribeTable("should exit successfully through the binary workflow",
		func(args []string) {
			setProgramArgs(args...)

			c, err := New(
				WithPlugins(&golangv4.Plugin{}),
				WithDefaultPlugins(config.Version{Number: 3}, &golangv4.Plugin{}),
				WithVersion("version string"),
				WithFilesystem(filesystemWithInvalidProjectPath()),
			)
			Expect(err).NotTo(HaveOccurred())

			c.cmd.SetOut(io.Discard)
			c.cmd.SetArgs(args)
			Expect(c.Run()).To(Succeed())
		},
		Entry("for --plugins --help", []string{kubebuilderSubcommandInit, pluginsFlagArg, helpFlagArg}),
		Entry("for --plugins -h", []string{kubebuilderSubcommandInit, pluginsFlagArg, helpShorthandArg}),
		Entry("for --plugins help", []string{kubebuilderSubcommandInit, pluginsFlagArg, kubebuilderSubcommandHelp}),
		Entry("for --plugins=<plugin> --help", []string{
			kubebuilderSubcommandInit, pluginsFlagArg + "=" + pluginGoKubebuilderV4, helpFlagArg,
		}),
		Entry("for --plugins=<plugin> -h", []string{
			kubebuilderSubcommandInit, pluginsFlagArg + "=" + pluginGoKubebuilderV4, helpShorthandArg,
		}),
	)
})

var _ = Describe("CLI argument and path helpers", func() {
	Context("positionalArgs", func() {
		It("should return empty slice for empty args", func() {
			Expect(positionalArgs([]string{})).To(BeEmpty())
		})

		It("should return only positional arguments", func() {
			args := []string{alphaCommand, generateSubcommand, "--flag=val", "subarg"}
			Expect(positionalArgs(args)).To(Equal([]string{alphaCommand, generateSubcommand, "subarg"}))
		})

		It("should skip flag arguments and their values", func() {
			args := []string{pluginsFlagArg, pluginGoKubebuilderV4, kubebuilderSubcommandInit, "--custom", "value"}
			Expect(positionalArgs(args)).To(Equal([]string{kubebuilderSubcommandInit}))
		})

		It("should handle consecutive flags properly", func() {
			args := []string{"--flag1", "--flag2", "val"}
			Expect(positionalArgs(args)).To(BeEmpty())
		})
	})

	Context("hasOnlyRootFlags", func() {
		It("should return true for empty args", func() {
			Expect(hasOnlyRootFlags([]string{})).To(BeTrue())
		})

		It("should return true for root flags with separate values", func() {
			Expect(hasOnlyRootFlags([]string{pluginsFlagArg, pluginGoKubebuilderV4})).To(BeTrue())
			Expect(hasOnlyRootFlags([]string{"--" + projectVersionFlag, "3"})).To(BeTrue())
			Expect(hasOnlyRootFlags([]string{helpShorthandArg})).To(BeTrue())
			Expect(hasOnlyRootFlags([]string{helpFlagArg})).To(BeTrue())
		})

		It("should return true for root flags with = syntax", func() {
			Expect(hasOnlyRootFlags([]string{pluginsFlagArg + "=" + pluginGoKubebuilderV4})).To(BeTrue())
			Expect(hasOnlyRootFlags([]string{"--" + projectVersionFlag + "=3"})).To(BeTrue())
			Expect(hasOnlyRootFlags([]string{helpFlagArg + "=true"})).To(BeTrue())
		})

		It("should return false when non-root flags or commands are present", func() {
			Expect(hasOnlyRootFlags([]string{"--unknown"})).To(BeFalse())
			cmdWithFlag := []string{kubebuilderSubcommandInit, pluginsFlagArg + "=" + pluginGoKubebuilderV4}
			Expect(hasOnlyRootFlags(cmdWithFlag)).To(BeFalse())
			Expect(hasOnlyRootFlags([]string{"--domain", "test.domain"})).To(BeFalse())
		})
	})

	Context("isAlphaGenerateCommand", func() {
		It("should return true when alpha generate is invoked", func() {
			Expect(isAlphaGenerateCommand([]string{alphaCommand, generateSubcommand})).To(BeTrue())
			Expect(isAlphaGenerateCommand([]string{kubebuilderCommandName, alphaCommand, generateSubcommand})).To(BeTrue())
			pluginAndAlpha := []string{pluginsFlagArg + "=" + pluginGoKubebuilderV4, alphaCommand, generateSubcommand}
			Expect(isAlphaGenerateCommand(pluginAndAlpha)).To(BeTrue())
			Expect(isAlphaGenerateCommand([]string{alphaCommand, generateSubcommand, "--force"})).To(BeTrue())
		})

		It("should return false for other commands", func() {
			Expect(isAlphaGenerateCommand([]string{})).To(BeFalse())
			Expect(isAlphaGenerateCommand([]string{kubebuilderSubcommandInit})).To(BeFalse())
			Expect(isAlphaGenerateCommand([]string{createSubcommand, apiSubcommand})).To(BeFalse())
			Expect(isAlphaGenerateCommand([]string{alphaCommand, "other"})).To(BeFalse())
			Expect(isAlphaGenerateCommand([]string{generateSubcommand, alphaCommand})).To(BeFalse())
		})
	})

	Context("isRegularPathNoFollow", func() {
		It("should return true for a regular file", func() {
			fs := afero.NewMemMapFs()
			Expect(afero.WriteFile(fs, "test.txt", []byte("hello"), 0o644)).To(Succeed())
			Expect(isRegularPathNoFollow(fs, "test.txt")).To(BeTrue())
		})

		It("should return false for a non-existent file", func() {
			fs := afero.NewMemMapFs()
			Expect(isRegularPathNoFollow(fs, "nonexistent.txt")).To(BeFalse())
		})

		It("should return false for a directory", func() {
			fs := afero.NewMemMapFs()
			Expect(fs.Mkdir("testdir", 0o755)).To(Succeed())
			Expect(isRegularPathNoFollow(fs, "testdir")).To(BeFalse())
		})
	})

	Context("patchProjectFileInMemoryIfNeeded", func() {
		It("should patch go/v2 to go/v4 in memory", func() {
			fs := afero.NewMemMapFs()
			content := `domain: example.com
layout:
- go.kubebuilder.io/v2
version: "3"
`
			Expect(afero.WriteFile(fs, yamlstore.DefaultPath, []byte(content), machinery.DefaultFilePermission)).To(Succeed())
			Expect(patchProjectFileInMemoryIfNeeded(fs, yamlstore.DefaultPath)).To(Succeed())

			updated, err := afero.ReadFile(fs, yamlstore.DefaultPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(updated)).To(ContainSubstring(pluginGoKubebuilderV4))
			Expect(string(updated)).NotTo(ContainSubstring(pluginGoKubebuilderV2))
		})

		It("should patch go/v3 to go/v4 in memory", func() {
			fs := afero.NewMemMapFs()
			content := `domain: example.com
layout:
- go.kubebuilder.io/v3
version: "3"
`
			Expect(afero.WriteFile(fs, yamlstore.DefaultPath, []byte(content), machinery.DefaultFilePermission)).To(Succeed())
			Expect(patchProjectFileInMemoryIfNeeded(fs, yamlstore.DefaultPath)).To(Succeed())

			updated, err := afero.ReadFile(fs, yamlstore.DefaultPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(updated)).To(ContainSubstring(pluginGoKubebuilderV4))
			Expect(string(updated)).NotTo(ContainSubstring(pluginGoKubebuilderV3))
		})

		DescribeTable("should patch a retired Go plugin key to go/v4",
			func(layout, expected string) {
				fs := afero.NewMemMapFs()
				project := func(key string) string {
					return "domain: example.com\nlayout:\n- " + key + "\nversion: \"3\"\n"
				}
				content := project(layout)
				Expect(afero.WriteFile(fs, yamlstore.DefaultPath,
					[]byte(content), machinery.DefaultFilePermission)).To(Succeed())
				Expect(patchProjectFileInMemoryIfNeeded(fs, yamlstore.DefaultPath)).To(Succeed())

				updated, err := afero.ReadFile(fs, yamlstore.DefaultPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(updated)).To(Equal(project(expected)))
			},
			Entry("go/v3-alpha", pluginGoKubebuilderV3Alpha, pluginGoKubebuilderV4),
			Entry("go/v4-alpha", pluginGoKubebuilderV4Alpha, pluginGoKubebuilderV4),
			Entry("base go/v3 still migrates", "base.go.kubebuilder.io/v3", "base.go.kubebuilder.io/v4"),
		)

		It("should not modify files that do not need patching", func() {
			fs := afero.NewMemMapFs()
			content := `domain: example.com
layout:
- go.kubebuilder.io/v4
version: "3"
`
			Expect(afero.WriteFile(fs, yamlstore.DefaultPath, []byte(content), machinery.DefaultFilePermission)).To(Succeed())
			Expect(patchProjectFileInMemoryIfNeeded(fs, yamlstore.DefaultPath)).To(Succeed())

			updated, err := afero.ReadFile(fs, yamlstore.DefaultPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(updated)).To(Equal(content))
		})

		It("should return nil if file does not exist", func() {
			fs := afero.NewMemMapFs()
			Expect(patchProjectFileInMemoryIfNeeded(fs, "nonexistent.yaml")).To(Succeed())
		})
	})
})

// Cobra takes any true value of a boolean flag, so `-h=true` and `--help=1` ask for help the same
// way `--help` does.
var _ = Describe("help asked with another true value of the help flag", func() {
	DescribeTable("should print the help of create api when the configuration cannot be used",
		func(filesystem func() machinery.Filesystem, spelling string) {
			args := []string{createSubcommand, apiSubcommand, spelling}
			out, err := runCLI(filesystem(), args, args)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(ContainSubstring(createSubcommand + " " + apiSubcommand + " [flags]"))
			Expect(out).To(ContainSubstring(kindFlagArg))
		},
		Entry("for -h=true when PROJECT is a directory", filesystemWithProjectDirectory, helpShorthandArg+"=true"),
		Entry("for --help=1 when PROJECT is a directory", filesystemWithProjectDirectory, helpFlagArg+"=1"),
		Entry("for -h=true when PROJECT cannot be parsed", filesystemWithInvalidProject, helpShorthandArg+"=true"),
		Entry("for --help=1 when PROJECT cannot be parsed", filesystemWithInvalidProject, helpFlagArg+"=1"),
		Entry("for -h=true when the PROJECT plugin chain cannot be resolved",
			filesystemWithUnresolvableProject, helpShorthandArg+"=true"),
		Entry("for --help=1 when the PROJECT plugin chain cannot be resolved",
			filesystemWithUnresolvableProject, helpFlagArg+"=1"),
	)
})

// --plugins replaces the plugins the layout names, so the help shows only the flags of the chain the
// flag names.
var _ = Describe("a plugin chain named with --plugins next to a usable PROJECT", func() {
	var (
		projectPlugin testCreateAPIPlugin
		filesystem    machinery.Filesystem
	)

	BeforeEach(func() {
		projectPlugin = newDescribedCreateAPIPlugin()
		filesystem = filesystemWithProject(`domain: example.com
layout:
- ` + plugin.KeyFor(golangv4.Plugin{}) + `
- ` + projectChainPluginKey + `
projectName: test
version: "3"
`)
	})

	It("should offer the flags of every plugin the layout names without --plugins", func() {
		args := []string{createSubcommand, apiSubcommand, helpFlagArg}
		out, err := runCLI(filesystem, args, args, WithPlugins(projectPlugin))
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("--" + projectChainFlag))
	})

	It("should leave out the flags of the layout's other plugins", func() {
		args := []string{createSubcommand, apiSubcommand, pluginsFlagArg, plugin.KeyFor(golangv4.Plugin{}), helpFlagArg}
		out, err := runCLI(filesystem, args, args, WithPlugins(projectPlugin))
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring(kindFlagArg))
		Expect(out).NotTo(ContainSubstring("--" + projectChainFlag))
	})
})

// A tool's code reads the command line as cobra parsed it, and its hooks run as cobra runs them.
var _ = Describe("what the code of an extra command sees", func() {
	It("should give every step the --plugins value the user typed", func() {
		seen := map[string]string{}
		record := func(step string) func(*cobra.Command, []string) error {
			return func(cmd *cobra.Command, _ []string) error {
				flag := cmd.Flag(pluginsFlag)
				Expect(flag).NotTo(BeNil())
				seen[step] = fmt.Sprintf("%s changed=%t", flag.Value.String(), flag.Changed)

				return nil
			}
		}
		docs := &cobra.Command{
			Use:               docsSubcommand,
			Args:              record("argument check"),
			PersistentPreRunE: record("persistent pre-run"),
			PreRunE:           record("pre-run"),
			RunE:              record("run"),
		}
		key := plugin.KeyFor(golangv4.Plugin{})

		args := []string{docsSubcommand, pluginsFlagArg, key}
		_, err := runCLI(filesystemWithoutProject(), args, args, WithExtraCommands(docs))
		Expect(err).NotTo(HaveOccurred())
		for _, step := range []string{"argument check", "persistent pre-run", "pre-run", "run"} {
			Expect(seen).To(HaveKeyWithValue(step, "["+key+"] changed=true"))
		}
	})

	It("should run every persistent hook once, in cobra's order, when hook traversal is on", func() {
		original := cobra.EnableTraverseRunHooks
		DeferCleanup(func() { cobra.EnableTraverseRunHooks = original })
		cobra.EnableTraverseRunHooks = true

		var steps []string
		trace := func(step string) func(*cobra.Command, []string) {
			return func(*cobra.Command, []string) { steps = append(steps, step) }
		}
		docs := &cobra.Command{
			Use:               docsSubcommand,
			PersistentPreRun:  trace("docs persistent pre-run"),
			PersistentPostRun: trace("docs persistent post-run"),
		}
		generate := &cobra.Command{
			Use:               generateSubcommand,
			PersistentPreRun:  trace("generate persistent pre-run"),
			Run:               trace("generate run"),
			PersistentPostRun: trace("generate persistent post-run"),
		}
		docs.AddCommand(generate)

		args := []string{docsSubcommand, generateSubcommand}
		_, err := runCLI(filesystemWithoutProject(), args, args, WithExtraCommands(docs))
		Expect(err).NotTo(HaveOccurred())
		Expect(steps).To(Equal([]string{
			"docs persistent pre-run",
			"generate persistent pre-run",
			"generate run",
			"generate persistent post-run",
			"docs persistent post-run",
		}))
	})

	It("should leave a cobra tree outside the CLI running only its nearest persistent hook", func() {
		_ = buildCLI(filesystemWithoutProject(), []string{docsSubcommand})

		var steps []string
		trace := func(step string) func(*cobra.Command, []string) {
			return func(*cobra.Command, []string) { steps = append(steps, step) }
		}
		solo := &cobra.Command{Use: "solo", PersistentPreRun: trace("solo persistent pre-run")}
		solo.AddCommand(&cobra.Command{
			Use:              "child",
			PersistentPreRun: trace("child persistent pre-run"),
			Run:              trace("child run"),
		})
		solo.SetArgs([]string{"child"})

		Expect(solo.Execute()).To(Succeed())
		Expect(steps).To(Equal([]string{"child persistent pre-run", "child run"}))
	})
})

// A mistyped --plugins value is the end user's mistake, so a command that consumes the project
// configuration reports that value, not the state of PROJECT.
var _ = Describe("a mistyped plugins value on a command that consumes the configuration", func() {
	DescribeTable("should report the value, not PROJECT",
		func(filesystem func() machinery.Filesystem, command ...string) {
			err := executeCLI(filesystem(), append(command, pluginsFlagArg, "//")...)
			Expect(err).To(MatchError(ContainSubstring(`invalid plugin "//"`)))
			Expect(err).NotTo(MatchError(ContainSubstring(yamlstore.DefaultPath)))
		},
		Entry("for init when PROJECT is a directory", filesystemWithProjectDirectory, kubebuilderSubcommandInit),
		Entry("for create api when the PROJECT file is not valid YAML", filesystemWithInvalidProject,
			createSubcommand, apiSubcommand),
		Entry("for edit when the PROJECT project version is not supported", filesystemWithUnloadableProject,
			editSubcommand),
	)
})

// A tool can declare --help as a persistent flag on a group of its own, so the group's commands
// inherit it. The commands under it still stop before their own steps when PROJECT cannot be used.
var _ = Describe("a command under a help flag the tool declares as persistent", func() {
	newDocsWithPersistentHelp := func() (*cobra.Command, *bool, *bool) {
		docs := &cobra.Command{Use: docsSubcommand}
		docs.PersistentFlags().BoolP("help", "h", false, "help for the tool's docs")
		generate, hookRan := newExtraCommandWithHook(generateSubcommand, setPersistentPreRunE)
		argsRan := false
		generate.Args = func(*cobra.Command, []string) error {
			argsRan = true
			return nil
		}
		docs.AddCommand(generate)

		return docs, hookRan, &argsRan
	}

	It("should stop the command before its argument check and hook when PROJECT is a directory", func() {
		docs, hookRan, argsRan := newDocsWithPersistentHelp()

		err := executeCLIWith(filesystemWithProjectDirectory(), WithExtraCommands(docs),
			docsSubcommand, generateSubcommand)
		expectStoppedBeforeHook(err, hookRan)
		Expect(*argsRan).To(BeFalse())
	})

	It("should run the command and its hook when there is no PROJECT file", func() {
		docs, hookRan, argsRan := newDocsWithPersistentHelp()

		err := executeCLIWith(filesystemWithoutProject(), WithExtraCommands(docs),
			docsSubcommand, generateSubcommand)
		expectHookRan(err, hookRan)
		Expect(*argsRan).To(BeTrue())
	})
})

// A tool can add a flag of its own to a command Kubebuilder gives it, after the CLI is built. Naming
// the plugin chain with --plugins must not take that flag away.
var _ = Describe("a flag a tool adds to create api after the CLI is built", func() {
	addTeamFlag := func(c *CLI) *string {
		GinkgoHelper()

		createAPI, _, err := c.Command().Find([]string{createSubcommand, apiSubcommand})
		Expect(err).NotTo(HaveOccurred())

		return createAPI.Flags().String("team", "", "team that owns what this command scaffolds")
	}

	It("should be listed in the help when --plugins names the default chain", func() {
		args := []string{createSubcommand, apiSubcommand, pluginsFlagArg, plugin.KeyFor(golangv4.Plugin{}), helpFlagArg}
		c := buildCLI(filesystemWithoutProject(), args)
		_ = addTeamFlag(c)

		out, err := runBuiltCLI(c, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("--team"))
	})

	It("should be parsed in a run when --plugins names the default chain", func() {
		args := append(createAPIArgs(), pluginsFlagArg, plugin.KeyFor(golangv4.Plugin{}), "--team", "platform")
		c := buildCLI(filesystemWithoutProject(), args)
		team := addTeamFlag(c)

		_, err := runBuiltCLI(c, args)
		Expect(err).To(MatchError(ContainSubstring("project must be initialized")))
		Expect(*team).To(Equal("platform"))
	})
})

// --plugins help asks for help on any command, a tool's own top-level version included.
var _ = Describe("help requested through the plugins flag on a tool's own version", func() {
	DescribeTable("should print the help of that version and run nothing of it",
		func(filesystem func() machinery.Filesystem, pluginsArgs ...string) {
			ran := false
			record := func(*cobra.Command, []string) error {
				ran = true
				return nil
			}
			version := &cobra.Command{
				Use:               kubebuilderSubcommandVersion,
				Short:             "the tool's own version",
				PersistentPreRunE: record,
				RunE:              record,
			}

			args := append([]string{kubebuilderSubcommandVersion}, pluginsArgs...)
			out, err := runBuiltCLI(newToolCLI(filesystem(), args, version), args)
			Expect(err).To(MatchError(errHelpDisplayed))
			Expect(out).To(ContainSubstring(kubebuilderSubcommandVersion + " [flags]"))
			Expect(out).To(ContainSubstring("the tool's own version"))
			Expect(ran).To(BeFalse())
		},
		Entry("for --plugins help when PROJECT is a directory", filesystemWithProjectDirectory,
			pluginsFlagArg, kubebuilderSubcommandHelp),
		Entry("for --plugins=help when there is no PROJECT file", filesystemWithoutProject,
			pluginsFlagArg+"="+kubebuilderSubcommandHelp),
	)
})

// --plugins help asks for help on a command a tool adds, so the command's help is printed and none of
// its steps runs, even when PROJECT cannot be used.
var _ = Describe("help requested through the plugins flag on a command a tool adds", func() {
	DescribeTable("should print the help of the command and run nothing of it when PROJECT is a directory",
		func(pluginsArgs ...string) {
			docs, hookRan := newExtraCommandWithHook(docsSubcommand, setPersistentPreRunE)
			docs.Short = "the tool's own docs"

			args := append([]string{docsSubcommand}, pluginsArgs...)
			out, err := runCLI(filesystemWithProjectDirectory(), args, args, WithExtraCommands(docs))
			Expect(err).To(MatchError(errHelpDisplayed))
			Expect(out).To(ContainSubstring(docsSubcommand + " [flags]"))
			Expect(*hookRan).To(BeFalse())
		},
		Entry("for --plugins help", pluginsFlagArg, kubebuilderSubcommandHelp),
		Entry("for --plugins=help", pluginsFlagArg+"="+kubebuilderSubcommandHelp),
	)
})

// A --plugins flag replaces the chain the project names, but a project whose own chain cannot be
// resolved is broken whatever the command line asks for, so a command that consumes it stops.
var _ = Describe("a PROJECT file whose plugin chain cannot be resolved, with --plugins", func() {
	const unresolvableChain = `no plugin could be resolved with key "gone.example.com/v1"`

	var (
		dir        string
		filesystem machinery.Filesystem
		project    []byte
	)

	BeforeEach(func() {
		// The filesystem and the working directory are the same directory, so a plugin step that
		// runs a tool in the working directory is caught by the same check as a file it writes.
		dir = GinkgoT().TempDir()
		filesystem = machinery.Filesystem{FS: afero.NewBasePathFs(afero.NewOsFs(), dir)}
		project = []byte(`domain: example.com
layout:
- gone.example.com/v1
projectName: test
version: "3"
`)
		Expect(afero.WriteFile(filesystem.FS, yamlstore.DefaultPath, project,
			machinery.DefaultFilePermission)).To(Succeed())

		originalDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.Chdir(originalDir)).To(Succeed()) })
		Expect(os.Chdir(dir)).To(Succeed())
	})

	// expectUntouched checks that the run left the PROJECT file as it was and wrote nothing else.
	expectUntouched := func() {
		GinkgoHelper()

		entries, err := os.ReadDir(dir)
		Expect(err).NotTo(HaveOccurred())
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		Expect(names).To(ConsistOf(yamlstore.DefaultPath))

		content, err := afero.ReadFile(filesystem.FS, yamlstore.DefaultPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(content).To(Equal(project))
	}

	It("should refuse create api instead of scaffolding with the plugin the flag names", func() {
		scaffolder := newDescribedCreateAPIPlugin()
		args := []string{
			createSubcommand, apiSubcommand, pluginsFlagArg, projectChainPluginKey, kindFlagArg, kindValue,
		}

		err := executeEmbeddedCLI(filesystem, args, args, WithPlugins(scaffolder))
		Expect(err).To(MatchError(ContainSubstring(unresolvableChain)))
		Expect(scaffolder.subcommand.scaffolded).To(BeFalse(),
			"the plugin named on the command line must not scaffold over a broken project")
		expectUntouched()
	})

	It("should refuse edit with an optional plugin", func() {
		args := []string{editSubcommand, pluginsFlagArg + "=helm/v2-alpha"}

		err := executeEmbeddedCLI(filesystem, args, args, WithPlugins(&helmv2alpha.Plugin{}))
		Expect(err).To(MatchError(ContainSubstring(unresolvableChain)))
		expectUntouched()
	})

	It("should still answer a help request with the help of the plugin the flag names", func() {
		scaffolder := newDescribedCreateAPIPlugin()
		args := []string{createSubcommand, apiSubcommand, pluginsFlagArg, projectChainPluginKey, helpFlagArg}

		out, err := runCLI(filesystem, args, args, WithPlugins(scaffolder))
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("--" + projectChainFlag))
		expectUntouched()
	})
})

// A tool started with a help request in its process arguments runs the command it picks with
// SetArgs, so a bad value for that command's flag fails it instead of printing help.
var _ = Describe("a command a tool picks with SetArgs while the program was started with --help", func() {
	newCommandWithFlag := func() (*cobra.Command, *bool) {
		ran := false
		cmd := &cobra.Command{Use: docsSubcommand, RunE: func(*cobra.Command, []string) error {
			ran = true

			return nil
		}}
		cmd.Flags().Bool("enabled", false, "a boolean flag of the tool")

		return cmd, &ran
	}

	run := func(filesystem machinery.Filesystem, viaRun bool, extra *cobra.Command, commandArgs ...string) error {
		GinkgoHelper()

		c := buildCLI(filesystem, []string{helpFlagArg}, WithExtraCommands(extra))
		c.cmd.SetOut(io.Discard)
		c.cmd.SetErr(io.Discard)
		c.cmd.SetArgs(commandArgs)
		if viaRun {
			return c.Run()
		}

		return c.cmd.Execute()
	}

	DescribeTable("should fail on a bad value of its flag and not run",
		func(viaRun bool, filesystem func() machinery.Filesystem) {
			docs, ran := newCommandWithFlag()
			Expect(run(filesystem(), viaRun, docs, docsSubcommand, "--enabled=maybe")).NotTo(Succeed())
			Expect(*ran).To(BeFalse())
		},
		Entry("through Execute, with no PROJECT file", false, filesystemWithoutProject),
		Entry("through Execute, with a directory at the path", false, filesystemWithProjectDirectory),
		Entry("through Run, with no PROJECT file", true, filesystemWithoutProject),
		Entry("through Run, with a directory at the path", true, filesystemWithProjectDirectory),
	)

	DescribeTable("should run with a good value of its flag and no PROJECT file",
		func(viaRun bool) {
			docs, ran := newCommandWithFlag()
			Expect(run(filesystemWithoutProject(), viaRun, docs, docsSubcommand, "--enabled=true")).To(Succeed())
			Expect(*ran).To(BeTrue())
		},
		Entry("through Execute", false),
		Entry("through Run", true),
	)
})
