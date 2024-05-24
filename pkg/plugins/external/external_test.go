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

package external

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

func TestExternalPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scaffold")
}

type mockValidOutputGetter struct{}

type mockInValidOutputGetter struct{}

var _ ExecOutputGetter = &mockValidOutputGetter{}

func (m *mockValidOutputGetter) GetExecOutput(_ []byte, _ string) ([]byte, error) {
	return []byte(`{
		"command": "init", 
		"error": false, 
		"error_msg": "none", 
		"universe": {"LICENSE": "Apache 2.0 License\n"}
		}`), nil
}

var _ ExecOutputGetter = &mockInValidOutputGetter{}

func (m *mockInValidOutputGetter) GetExecOutput(_ []byte, _ string) ([]byte, error) {
	return nil, fmt.Errorf("error getting exec command output")
}

type mockValidOsWdGetter struct{}

var _ OsWdGetter = &mockValidOsWdGetter{}

func (m *mockValidOsWdGetter) GetCurrentDir() (string, error) {
	return "tmp/externalPlugin", nil
}

type mockInValidOsWdGetter struct{}

var _ OsWdGetter = &mockInValidOsWdGetter{}

func (m *mockInValidOsWdGetter) GetCurrentDir() (string, error) {
	return "", fmt.Errorf("error getting current directory")
}

type mockValidFlagOutputGetter struct{}

func (m *mockValidFlagOutputGetter) GetExecOutput(_ []byte, _ string) ([]byte, error) {
	response := external.PluginResponse{
		Command:  "flag",
		Error:    false,
		Universe: nil,
		Flags:    getFlags(),
	}
	return json.Marshal(response)
}

type mockValidMEOutputGetter struct{}

func (m *mockValidMEOutputGetter) GetExecOutput(_ []byte, _ string) ([]byte, error) {
	response := external.PluginResponse{
		Command:  "metadata",
		Error:    false,
		Universe: nil,
		Metadata: getMetadata(),
	}

	return json.Marshal(response)
}

const (
	externalPlugin = "myexternalplugin.sh"
	floatVal       = "float"
)

var _ = Describe("Run external plugin using Scaffold", func() {
	Context("with valid mock values", func() {
		const filePerm os.FileMode = 755
		var (
			pluginFileName string
			args           []string
			f              afero.File
			fs             machinery.Filesystem

			err error
		)

		BeforeEach(func() {
			outputGetter = &mockValidOutputGetter{}
			currentDirGetter = &mockValidOsWdGetter{}
			fs = machinery.Filesystem{
				FS: afero.NewMemMapFs(),
			}

			pluginFileName = "externalPlugin.sh"
			pluginFilePath := filepath.Join("tmp", "externalPlugin", pluginFileName)

			err = fs.FS.MkdirAll(filepath.Dir(pluginFilePath), filePerm)
			Expect(err).ToNot(HaveOccurred())

			f, err = fs.FS.Create(pluginFilePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(f).ToNot(BeNil())

			_, err = fs.FS.Stat(pluginFilePath)
			Expect(err).ToNot(HaveOccurred())

			args = []string{"--domain", "example.com"}
		})

		AfterEach(func() {
			filename := filepath.Join("tmp", "externalPlugin", "LICENSE")
			fileInfo, err := fs.FS.Stat(filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileInfo).NotTo(BeNil())
		})

		It("should successfully run init subcommand on the external plugin", func() {
			i := initSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = i.Scaffold(fs)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should successfully run edit subcommand on the external plugin", func() {
			e := editSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = e.Scaffold(fs)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should successfully run create api subcommand on the external plugin", func() {
			c := createAPISubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = c.Scaffold(fs)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should successfully run create webhook subcommand on the external plugin", func() {
			c := createWebhookSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = c.Scaffold(fs)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("with invalid mock values of GetExecOutput() and GetCurrentDir()", func() {
		var (
			pluginFileName string
			args           []string
			fs             machinery.Filesystem
			err            error
		)
		BeforeEach(func() {
			outputGetter = &mockInValidOutputGetter{}
			currentDirGetter = &mockValidOsWdGetter{}
			fs = machinery.Filesystem{
				FS: afero.NewMemMapFs(),
			}

			pluginFileName = externalPlugin
			args = []string{"--domain", "example.com"}
		})

		It("should return error upon running init subcommand on the external plugin", func() {
			i := initSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = i.Scaffold(fs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting exec command output"))

			outputGetter = &mockValidOutputGetter{}
			currentDirGetter = &mockInValidOsWdGetter{}

			err = i.Scaffold(fs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting current directory"))
		})

		It("should return error upon running edit subcommand on the external plugin", func() {
			e := editSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = e.Scaffold(fs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting exec command output"))

			outputGetter = &mockValidOutputGetter{}
			currentDirGetter = &mockInValidOsWdGetter{}

			err = e.Scaffold(fs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting current directory"))
		})

		It("should return error upon running create api subcommand on the external plugin", func() {
			c := createAPISubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = c.Scaffold(fs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting exec command output"))

			outputGetter = &mockValidOutputGetter{}
			currentDirGetter = &mockInValidOsWdGetter{}

			err = c.Scaffold(fs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting current directory"))
		})

		It("should return error upon running create webhook subcommand on the external plugin", func() {
			c := createWebhookSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = c.Scaffold(fs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting exec command output"))

			outputGetter = &mockValidOutputGetter{}
			currentDirGetter = &mockInValidOsWdGetter{}

			err = c.Scaffold(fs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting current directory"))
		})
	})

	Context("with successfully getting flags from external plugin", func() {
		var (
			pluginFileName string
			args           []string
			flagset        *pflag.FlagSet

			// Make an array of flags to represent the ones that should be returned in these tests
			flags = getFlags()

			checkFlagset func()
		)
		BeforeEach(func() {
			outputGetter = &mockValidFlagOutputGetter{}
			currentDirGetter = &mockValidOsWdGetter{}

			pluginFileName = externalPlugin
			args = []string{"--captain", "black-beard", "--sail"}
			flagset = pflag.NewFlagSet("test", pflag.ContinueOnError)

			checkFlagset = func() {
				Expect(flagset.HasFlags()).To(BeTrue())

				for _, flag := range flags {
					Expect(flagset.Lookup(flag.Name)).NotTo(BeNil())
					// we parse floats as float64 Go type so this check will account for that
					if flag.Type != floatVal {
						Expect(flagset.Lookup(flag.Name).Value.Type()).To(Equal(flag.Type))
					} else {
						Expect(flagset.Lookup(flag.Name).Value.Type()).To(Equal("float64"))
					}
					Expect(flagset.Lookup(flag.Name).Usage).To(Equal(flag.Usage))
					Expect(flagset.Lookup(flag.Name).DefValue).To(Equal(flag.Default))
				}
			}
		})

		It("should successfully bind external plugin specified flags for `init` subcommand", func() {
			sc := initSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			sc.BindFlags(flagset)

			checkFlagset()
		})

		It("should successfully bind external plugin specified flags for `create api` subcommand", func() {
			sc := createAPISubcommand{
				Path: pluginFileName,
				Args: args,
			}

			sc.BindFlags(flagset)

			checkFlagset()
		})

		It("should successfully bind external plugin specified  flags for `create webhook` subcommand", func() {
			sc := createWebhookSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			sc.BindFlags(flagset)

			checkFlagset()
		})

		It("should successfully bind external plugin specified flags for `edit` subcommand", func() {
			sc := editSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			sc.BindFlags(flagset)

			checkFlagset()
		})
	})

	Context("with failure to get flags from external plugin", func() {
		var (
			pluginFileName string
			args           []string
			flagset        *pflag.FlagSet
			usage          string
			checkFlagset   func()
		)
		BeforeEach(func() {
			outputGetter = &mockInValidOutputGetter{}
			currentDirGetter = &mockValidOsWdGetter{}

			pluginFileName = externalPlugin
			args = []string{"--captain", "black-beard", "--sail"}
			flagset = pflag.NewFlagSet("test", pflag.ContinueOnError)
			usage = "Kubebuilder could not validate this flag with the external plugin. " +
				"Consult the external plugin documentation for more information."

			checkFlagset = func() {
				Expect(flagset.HasFlags()).To(BeTrue())

				Expect(flagset.Lookup("captain")).NotTo(BeNil())
				Expect(flagset.Lookup("captain").Value.Type()).To(Equal("string"))
				Expect(flagset.Lookup("captain").Usage).To(Equal(usage))

				Expect(flagset.Lookup("sail")).NotTo(BeNil())
				Expect(flagset.Lookup("sail").Value.Type()).To(Equal("bool"))
				Expect(flagset.Lookup("sail").Usage).To(Equal(usage))
			}
		})

		It("should successfully bind all user passed flags for `init` subcommand", func() {
			sc := initSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			sc.BindFlags(flagset)

			checkFlagset()
		})

		It("should successfully bind all user passed flags for `create api` subcommand", func() {
			sc := createAPISubcommand{
				Path: pluginFileName,
				Args: args,
			}

			sc.BindFlags(flagset)

			checkFlagset()
		})

		It("should successfully bind all user passed flags for `create webhook` subcommand", func() {
			sc := createWebhookSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			sc.BindFlags(flagset)

			checkFlagset()
		})

		It("should successfully bind all user passed flags for `edit` subcommand", func() {
			sc := editSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			sc.BindFlags(flagset)

			checkFlagset()
		})
	})

	Context("Flag Parsing Filter Functions", func() {
		It("gvk(Arg/Flag)Filter should filter out (--)group, (--)version, (--)kind", func() {
			for _, toBeFiltered := range []string{
				"group", "version", "kind",
			} {
				Expect(gvkArgFilter("--" + toBeFiltered)).To(BeFalse())
				Expect(gvkArgFilter(toBeFiltered)).To(BeFalse())
				Expect(gvkFlagFilter(external.Flag{Name: "--" + toBeFiltered})).To(BeFalse())
				Expect(gvkFlagFilter(external.Flag{Name: "--" + toBeFiltered})).To(BeFalse())
			}
			Expect(gvkArgFilter("somerandomflag")).To(BeTrue())
			Expect(gvkFlagFilter(external.Flag{Name: "somerandomflag"})).To(BeTrue())
		})

		It("helpArgFilter should filter out (--)help", func() {
			Expect(helpArgFilter("--help")).To(BeFalse())
			Expect(helpArgFilter("help")).To(BeFalse())
			Expect(helpArgFilter("somerandomflag")).To(BeTrue())
			Expect(helpFlagFilter(external.Flag{Name: "--help"})).To(BeFalse())
			Expect(helpFlagFilter(external.Flag{Name: "help"})).To(BeFalse())
			Expect(helpFlagFilter(external.Flag{Name: "somerandomflag"})).To(BeTrue())
		})
	})

	Context("Flag Parsing Helper Functions", func() {
		var (
			fs   *pflag.FlagSet
			args = []string{
				"--domain", "something.com",
				"--boolean",
				"--another", "flag",
				"--help",
				"--group", "somegroup",
				"--kind", "somekind",
				"--version", "someversion",
			}
			forbidden = []string{
				"help", "group", "kind", "version",
			}
			flags               []external.Flag
			argFilters          []argFilterFunc
			externalFlagFilters []externalFlagFilterFunc
		)

		BeforeEach(func() {
			fs = pflag.NewFlagSet("test", pflag.ContinueOnError)

			flagsToAppend := getFlags()

			flags = make([]external.Flag, len(flagsToAppend))
			copy(flags, flagsToAppend)

			argFilters = []argFilterFunc{
				gvkArgFilter, helpArgFilter,
			}
			externalFlagFilters = []externalFlagFilterFunc{
				gvkFlagFilter, helpFlagFilter,
			}
		})

		It("isBooleanFlag should return true if boolean flag provided at index", func() {
			Expect(isBooleanFlag(2, args)).To(BeTrue())
		})

		It("isBooleanFlag should return false if boolean flag not provided at index", func() {
			Expect(isBooleanFlag(0, args)).To(BeFalse())
		})

		It("bindAllFlags should bind all flags", func() {
			usage := "Kubebuilder could not validate this flag with the external plugin. " +
				"Consult the external plugin documentation for more information."

			bindAllFlags(fs, filterArgs(args, argFilters))
			Expect(fs.HasFlags()).To(BeTrue())
			Expect(fs.Lookup("domain")).NotTo(BeNil())
			Expect(fs.Lookup("domain").Value.Type()).To(Equal("string"))
			Expect(fs.Lookup("domain").Usage).To(Equal(usage))
			Expect(fs.Lookup("boolean")).NotTo(BeNil())
			Expect(fs.Lookup("boolean").Value.Type()).To(Equal("bool"))
			Expect(fs.Lookup("boolean").Usage).To(Equal(usage))
			Expect(fs.Lookup("another")).NotTo(BeNil())
			Expect(fs.Lookup("another").Value.Type()).To(Equal("string"))
			Expect(fs.Lookup("another").Usage).To(Equal(usage))

			By("bindAllFlags not have bound any forbidden flag after filtering")
			for i := range forbidden {
				Expect(fs.Lookup(forbidden[i])).To(BeNil())
			}
		})

		It("bindSpecificFlags should bind all flags in given []Flag", func() {
			filteredFlags := filterFlags(flags, externalFlagFilters)
			bindSpecificFlags(fs, filteredFlags)

			Expect(fs.HasFlags()).To(BeTrue())

			for _, flag := range filteredFlags {
				Expect(fs.Lookup(flag.Name)).NotTo(BeNil())
				// we parse floats as float64 Go type so this check will account for that
				if flag.Type != floatVal {
					Expect(fs.Lookup(flag.Name).Value.Type()).To(Equal(flag.Type))
				} else {
					Expect(fs.Lookup(flag.Name).Value.Type()).To(Equal("float64"))
				}
				Expect(fs.Lookup(flag.Name).Usage).To(Equal(flag.Usage))
				Expect(fs.Lookup(flag.Name).DefValue).To(Equal(flag.Default))
			}

			By("bindSpecificFlags not have bound any forbidden flag after filtering")
			for i := range forbidden {
				Expect(fs.Lookup(forbidden[i])).To(BeNil())
			}
		})
	})

	// TODO(everettraven): Add tests for an external plugin setting the Metadata and Examples
	Context("Successfully retrieving metadata and examples from external plugin", func() {
		var (
			pluginFileName string
			metadata       *plugin.SubcommandMetadata
			checkMetadata  func()
		)
		BeforeEach(func() {
			outputGetter = &mockValidMEOutputGetter{}
			currentDirGetter = &mockValidOsWdGetter{}

			pluginFileName = externalPlugin
			metadata = &plugin.SubcommandMetadata{}

			checkMetadata = func() {
				Expect(metadata.Description).Should(Equal(getMetadata().Description))
				Expect(metadata.Examples).Should(Equal(getMetadata().Examples))
			}
		})

		It("should use the external plugin's metadata and examples for `init` subcommand", func() {
			sc := initSubcommand{
				Path: pluginFileName,
				Args: nil,
			}

			sc.UpdateMetadata(plugin.CLIMetadata{}, metadata)

			checkMetadata()
		})

		It("should use the external plugin's metadata and examples for `create api` subcommand", func() {
			sc := createAPISubcommand{
				Path: pluginFileName,
				Args: nil,
			}

			sc.UpdateMetadata(plugin.CLIMetadata{}, metadata)

			checkMetadata()
		})

		It("should use the external plugin's metadata and examples for `create webhook` subcommand", func() {
			sc := createWebhookSubcommand{
				Path: pluginFileName,
				Args: nil,
			}

			sc.UpdateMetadata(plugin.CLIMetadata{}, metadata)

			checkMetadata()
		})

		It("should use the external plugin's metadata and examples for `edit` subcommand", func() {
			sc := editSubcommand{
				Path: pluginFileName,
				Args: nil,
			}

			sc.UpdateMetadata(plugin.CLIMetadata{}, metadata)

			checkMetadata()
		})
	})

	Context("Failing to retrieve metadata and examples from external plugin", func() {
		var (
			pluginFileName string
			metadata       *plugin.SubcommandMetadata
			checkMetadata  func()
		)
		BeforeEach(func() {
			outputGetter = &mockInValidOutputGetter{}
			currentDirGetter = &mockValidOsWdGetter{}

			pluginFileName = externalPlugin
			metadata = &plugin.SubcommandMetadata{}

			checkMetadata = func() {
				Expect(metadata.Description).Should(Equal(fmt.Sprintf(defaultMetadataTemplate, "myexternalplugin")))
				Expect(metadata.Examples).Should(BeEmpty())
			}
		})

		It("should use the default metadata and examples for `init` subcommand", func() {
			sc := initSubcommand{
				Path: pluginFileName,
				Args: nil,
			}

			sc.UpdateMetadata(plugin.CLIMetadata{}, metadata)

			checkMetadata()
		})

		It("should use the default metadata and examples for `create api` subcommand", func() {
			sc := createAPISubcommand{
				Path: pluginFileName,
				Args: nil,
			}

			sc.UpdateMetadata(plugin.CLIMetadata{}, metadata)

			checkMetadata()
		})

		It("should use the default metadata and examples for `create webhook` subcommand", func() {
			sc := createWebhookSubcommand{
				Path: pluginFileName,
				Args: nil,
			}

			sc.UpdateMetadata(plugin.CLIMetadata{}, metadata)

			checkMetadata()
		})

		It("should use the default metadata and examples for `edit` subcommand", func() {
			sc := editSubcommand{
				Path: pluginFileName,
				Args: nil,
			}

			sc.UpdateMetadata(plugin.CLIMetadata{}, metadata)

			checkMetadata()
		})
	})

	Context("Helper functions for Sending request to external plugin and parsing response", func() {
		It("getUniverseMap should return path to content mapping of all files in Filesystem", func() {
			fs := machinery.Filesystem{
				FS: afero.NewMemMapFs(),
			}

			files := []struct {
				path    string
				name    string
				content string
			}{
				{
					path:    "./",
					name:    "file",
					content: "level 0 file",
				},
				{
					path:    "dir/",
					name:    "file",
					content: "level 1 file",
				},
				{
					path:    "dir/subdir",
					name:    "file",
					content: "level 2 file",
				},
			}

			// create files in Filesystem
			for _, file := range files {
				err := fs.FS.MkdirAll(file.path, 0o700)
				Expect(err).ToNot(HaveOccurred())

				f, err := fs.FS.Create(filepath.Join(file.path, file.name))
				Expect(err).ToNot(HaveOccurred())

				_, err = f.Write([]byte(file.content))
				Expect(err).ToNot(HaveOccurred())

				err = f.Close()
				Expect(err).ToNot(HaveOccurred())
			}

			universe, err := getUniverseMap(fs)

			Expect(err).ToNot(HaveOccurred())
			Expect(universe).To(HaveLen(len(files)))

			for _, file := range files {
				content := universe[filepath.Join(file.path, file.name)]
				Expect(content).To(Equal(file.content))
			}
		})
	})
})

func getFlags() []external.Flag {
	return []external.Flag{
		{
			Name:    "captain",
			Type:    "string",
			Usage:   "specify the ship captain",
			Default: "jack-sparrow",
		},
		{
			Name:    "sail",
			Type:    "bool",
			Usage:   "deploy the sail",
			Default: "false",
		},
		{
			Name:    "crew-count",
			Type:    "int",
			Usage:   "number of crew members",
			Default: "123",
		},
		{
			Name:    "treasure-value",
			Type:    "float",
			Usage:   "value of treasure on board the ship",
			Default: "123.45",
		},
	}
}

func getMetadata() plugin.SubcommandMetadata {
	return plugin.SubcommandMetadata{
		Description: "Test description",
		Examples:    "Test examples",
	}
}
