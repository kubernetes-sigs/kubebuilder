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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

func TestExternalPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scaffold")
}

type mockValidOutputGetter struct{}

type mockInValidOutputGetter struct{}

var _ ExecOutputGetter = &mockValidOutputGetter{}

func (m *mockValidOutputGetter) GetExecOutput(request []byte, path string) ([]byte, error) {
	return []byte(`{
		"command": "init", 
		"error": false, 
		"error_msg": "none", 
		"universe": {"LICENSE": "Apache 2.0 License\n"}
		}`), nil
}

var _ ExecOutputGetter = &mockInValidOutputGetter{}

func (m *mockInValidOutputGetter) GetExecOutput(request []byte, path string) ([]byte, error) {
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
			Expect(err).To(BeNil())

			f, err = fs.FS.Create(pluginFilePath)
			Expect(err).To(BeNil())
			Expect(f).ToNot(BeNil())

			_, err = fs.FS.Stat(pluginFilePath)
			Expect(err).To(BeNil())

			args = []string{"--domain", "example.com"}

		})

		AfterEach(func() {
			filename := filepath.Join("tmp", "externalPlugin", "LICENSE")
			fileInfo, err := fs.FS.Stat(filename)
			Expect(err).To(BeNil())
			Expect(fileInfo).NotTo(BeNil())
		})

		It("should successfully run init subcommand on the external plugin", func() {
			i := initSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = i.Scaffold(fs)
			Expect(err).To(BeNil())
		})

		It("should successfully run edit subcommand on the external plugin", func() {
			e := editSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = e.Scaffold(fs)
			Expect(err).To(BeNil())
		})

		It("should successfully run create api subcommand on the external plugin", func() {
			c := createAPISubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = c.Scaffold(fs)
			Expect(err).To(BeNil())
		})

		It("should successfully run create webhook subcommand on the external plugin", func() {
			c := createWebhookSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = c.Scaffold(fs)
			Expect(err).To(BeNil())
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

			pluginFileName = "myexternalplugin.sh"
			args = []string{"--domain", "example.com"}

		})

		It("should return error upon running init subcommand on the external plugin", func() {
			i := initSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = i.Scaffold(fs)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("error getting exec command output"))

			outputGetter = &mockValidOutputGetter{}
			currentDirGetter = &mockInValidOsWdGetter{}

			err = i.Scaffold(fs)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("error getting current directory"))
		})

		It("should return error upon running edit subcommand on the external plugin", func() {
			e := editSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = e.Scaffold(fs)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("error getting exec command output"))

			outputGetter = &mockValidOutputGetter{}
			currentDirGetter = &mockInValidOsWdGetter{}

			err = e.Scaffold(fs)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("error getting current directory"))

		})

		It("should return error upon running create api subcommand on the external plugin", func() {
			c := createAPISubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = c.Scaffold(fs)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("error getting exec command output"))

			outputGetter = &mockValidOutputGetter{}
			currentDirGetter = &mockInValidOsWdGetter{}

			err = c.Scaffold(fs)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("error getting current directory"))
		})

		It("should return error upon running create webhook subcommand on the external plugin", func() {
			c := createWebhookSubcommand{
				Path: pluginFileName,
				Args: args,
			}

			err = c.Scaffold(fs)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("error getting exec command output"))

			outputGetter = &mockValidOutputGetter{}
			currentDirGetter = &mockInValidOsWdGetter{}

			err = c.Scaffold(fs)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("error getting current directory"))

		})
	})
})
