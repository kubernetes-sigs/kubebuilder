/*
Copyright 2018 The Kubernetes Authors.

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

package v1

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &GopkgToml{}

// GopkgToml writes a templatefile for Gopkg.toml
type GopkgToml struct {
	file.TemplateMixin

	// ManagedHeader is the header to write after the user owned pieces and before the managed parts of the Gopkg.toml
	ManagedHeader string

	// DefaultUserContent is the default content to use for the user owned pieces
	DefaultUserContent string

	// UserContent is the content to use for the user owned pieces
	UserContent string

	// Stanzas are additional managed stanzas to add after the ManagedHeader
	Stanzas []Stanza
}

// Stanza is a single Gopkg.toml entry
type Stanza struct {
	// Type will be between the'[[]]' e.g. override
	Type string

	// Name will appear after 'name=' and does not include quotes e.g. k8s.io/client-go
	Name string
	// Version will appear after 'version=' and does not include quotes
	Version string

	// Revision will appear after 'revision=' and does not include quotes
	Revision string
}

// SetTemplateDefaults implements input.Template
func (f *GopkgToml) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "Gopkg.toml"
	}

	f.TemplateBody = depTemplate

	f.IfExistsAction = file.Overwrite

	if f.ManagedHeader == "" {
		f.ManagedHeader = DefaultGopkgHeader
	}

	// Set the user content to be used if the Gopkg.toml doesn't exist
	if f.DefaultUserContent == "" {
		f.DefaultUserContent = DefaultGopkgUserContent
	}

	// Set the user owned content from the last Gopkg.toml file - e.g. everything before the header
	lastBytes, err := ioutil.ReadFile(f.Path)
	if err != nil {
		f.UserContent = f.DefaultUserContent
	} else if f.UserContent, err = f.getUserContent(lastBytes); err != nil {
		return err
	}

	return nil
}

func (f *GopkgToml) getUserContent(b []byte) (string, error) {
	// Keep the users lines
	scanner := bufio.NewScanner(bytes.NewReader(b))
	userLines := []string{}
	found := false
	for scanner.Scan() {
		l := scanner.Text()
		if l == f.ManagedHeader {
			found = true
			break
		}
		userLines = append(userLines, l)
	}

	if !found {
		return "", fmt.Errorf(
			"skipping modifying Gopkg.toml - file already exists and is unmanaged")
	}
	return strings.Join(userLines, "\n"), nil
}

// DefaultGopkgHeader is the default header used to separate user managed lines and controller-manager managed lines
const DefaultGopkgHeader = "# STANZAS BELOW ARE GENERATED AND MAY BE WRITTEN - DO NOT MODIFY BELOW THIS LINE."

// DefaultGopkgUserContent is the default user managed lines to provide.
const DefaultGopkgUserContent = `required = [
    "github.com/emicklei/go-restful",
    "github.com/onsi/ginkgo", # for test framework
    "github.com/onsi/gomega", # for test matchers
    "k8s.io/client-go/plugin/pkg/client/auth/gcp", # for development against gcp
    "k8s.io/code-generator/cmd/client-gen", # for go generate
    "k8s.io/code-generator/cmd/deepcopy-gen", # for go generate
    "sigs.k8s.io/controller-tools/cmd/controller-gen", # for crd/rbac generation
    "sigs.k8s.io/controller-runtime/pkg/client/config",
    "sigs.k8s.io/controller-runtime/pkg/controller",
    "sigs.k8s.io/controller-runtime/pkg/handler",
    "sigs.k8s.io/controller-runtime/pkg/manager",
    "sigs.k8s.io/controller-runtime/pkg/runtime/signals",
    "sigs.k8s.io/controller-runtime/pkg/source",
    "sigs.k8s.io/testing_frameworks/integration", # for integration testing
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1",
    ]

[prune]
  go-tests = true

`

const depTemplate = `{{ .UserContent }}
# STANZAS BELOW ARE GENERATED AND MAY BE WRITTEN - DO NOT MODIFY BELOW THIS LINE.

{{ range $element := .Stanzas -}}
[[{{ .Type }}]]
name="{{ .Name }}"
{{ if .Version }}version="{{.Version}}"{{ end }}
{{ if .Revision }}revision="{{.Revision}}"{{ end }}
{{ end -}}

[[constraint]]
  name="sigs.k8s.io/controller-runtime"
  version="v0.1.8"

[[constraint]]
  name="sigs.k8s.io/controller-tools"
  version="v0.1.1"

# For dependency below: Refer to issue https://github.com/golang/dep/issues/1799
[[override]]
name = "gopkg.in/fsnotify.v1"
source = "https://github.com/fsnotify/fsnotify.git"
version="v1.4.7"
`
