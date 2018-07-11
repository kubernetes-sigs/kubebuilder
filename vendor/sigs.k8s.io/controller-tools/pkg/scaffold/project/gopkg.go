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

package project

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &GopkgToml{}

// GopkgToml writes a templatefile for Gopkg.toml
type GopkgToml struct {
	input.Input

	// ManagedHeader is the header to write after the user owned pieces and before the managed parts of the Gopkg.toml
	ManagedHeader string

	// DefaultGopkgUserContent is the default content to use for the user owned pieces
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

	// Revision will appear after 'revsion=' and does not include quotes
	Revision string
}

// GetInput implements input.File
func (g *GopkgToml) GetInput() (input.Input, error) {
	if g.Path == "" {
		g.Path = "Gopkg.toml"
	}
	if g.ManagedHeader == "" {
		g.ManagedHeader = DefaultGopkgHeader
	}

	// Set the user content to be used if the Gopkg.toml doesn't exist
	if g.DefaultUserContent == "" {
		g.DefaultUserContent = DefaultGopkgUserContent
	}

	// Set the user owned content from the last Gopkg.toml file - e.g. everything before the header
	lastBytes, err := ioutil.ReadFile(g.Path)
	if err != nil {
		g.UserContent = g.DefaultUserContent
	} else if g.UserContent, err = g.getUserContent(lastBytes); err != nil {
		return input.Input{}, err
	}

	g.Input.IfExistsAction = input.Overwrite
	g.TemplateBody = depTemplate
	return g.Input, nil
}

func (g *GopkgToml) getUserContent(b []byte) (string, error) {
	// Keep the users lines
	scanner := bufio.NewScanner(bytes.NewReader(b))
	userLines := []string{}
	found := false
	for scanner.Scan() {
		l := scanner.Text()
		if l == g.ManagedHeader {
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
    "k8s.io/code-generator/cmd/deepcopy-gen", # for go generate
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

var depTemplate = `{{ .UserContent }}
# STANZAS BELOW ARE GENERATED AND MAY BE WRITTEN - DO NOT MODIFY BELOW THIS LINE.

{{ range $element := .Stanzas -}}
[[{{ .Type }}]]
name="{{ .Name }}"
{{ if .Version }}version="{{.Version}}"{{ end }}
{{ if .Revision }}revision="{{.Revision}}"{{ end }}
{{ end -}}

[[override]]
name="cloud.google.com/go"
version="v0.21.0"

[[override]]
name="github.com/PuerkitoBio/purell"
version="v1.1.0"

[[override]]
name="github.com/PuerkitoBio/urlesc"
revision="de5bf2ad457846296e2031421a34e2568e304e35"

[[override]]
name="github.com/davecgh/go-spew"
version="v1.1.0"

[[override]]
name="github.com/emicklei/go-restful"
version="v2.7.0"

[[override]]
name="github.com/ghodss/yaml"
version="v1.0.0"

[[override]]
name="github.com/go-openapi/jsonpointer"
revision="3a0015ad55fa9873f41605d3e8f28cd279c32ab2"

[[override]]
name="github.com/go-openapi/jsonreference"
revision="3fb327e6747da3043567ee86abd02bb6376b6be2"

[[override]]
name="github.com/go-openapi/spec"
revision="bcff419492eeeb01f76e77d2ebc714dc97b607f5"

[[override]]
name="github.com/go-openapi/swag"
revision="811b1089cde9dad18d4d0c2d09fbdbf28dbd27a5"

[[override]]
name="github.com/gogo/protobuf"
version="v1.0.0"

[[override]]
name="github.com/golang/glog"
revision="23def4e6c14b4da8ac2ed8007337bc5eb5007998"

[[override]]
name="github.com/golang/groupcache"
revision="66deaeb636dff1ac7d938ce666d090556056a4b0"

[[override]]
name="github.com/golang/protobuf"
version="v1.1.0"

[[override]]
name="github.com/google/gofuzz"
revision="24818f796faf91cd76ec7bddd72458fbced7a6c1"

[[override]]
name="github.com/googleapis/gnostic"
version="v0.1.0"

[[override]]
name="github.com/hashicorp/golang-lru"
revision="0fb14efe8c47ae851c0034ed7a448854d3d34cf3"

[[override]]
name="github.com/howeyc/gopass"
revision="bf9dde6d0d2c004a008c27aaee91170c786f6db8"

[[override]]
name="github.com/imdario/mergo"
version="v0.3.4"

[[override]]
name="github.com/json-iterator/go"
version="1.1.3"

[[override]]
name="github.com/mailru/easyjson"
revision="8b799c424f57fa123fc63a99d6383bc6e4c02578"

[[override]]
name="github.com/modern-go/concurrent"
version="1.0.3"

[[override]]
name="github.com/modern-go/reflect2"
version="1.0.0"

[[override]]
name="github.com/onsi/ginkgo"
version="v1.4.0"

[[override]]
name="github.com/onsi/gomega"
version="v1.3.0"

[[override]]
name="github.com/spf13/pflag"
version="v1.0.1"

[[override]]
name="golang.org/x/crypto"
revision="4ec37c66abab2c7e02ae775328b2ff001c3f025a"

[[override]]
name="golang.org/x/net"
revision="640f4622ab692b87c2f3a94265e6f579fe38263d"

[[override]]
name="golang.org/x/oauth2"
revision="cdc340f7c179dbbfa4afd43b7614e8fcadde4269"

[[override]]
name="golang.org/x/sys"
revision="7db1c3b1a98089d0071c84f646ff5c96aad43682"

[[override]]
name="golang.org/x/text"
version="v0.3.0"

[[override]]
name="golang.org/x/time"
revision="fbb02b2291d28baffd63558aa44b4b56f178d650"

[[override]]
name="google.golang.org/appengine"
version="v1.0.0"

[[override]]
name="gopkg.in/inf.v0"
version="v0.9.1"

[[override]]
name="gopkg.in/yaml.v2"
version="v2.2.1"

[[override]]
name="k8s.io/api"
version="kubernetes-1.10.0"

[[override]]
name="k8s.io/apiextensions-apiserver"
version="kubernetes-1.10.1"

[[override]]
name="k8s.io/apimachinery"
version="kubernetes-1.10.0"

[[override]]
name="k8s.io/client-go"
version="kubernetes-1.10.1"

[[override]]
name="k8s.io/kube-aggregator"
version="kubernetes-1.10.1"

[[override]]
name="k8s.io/kube-openapi"
revision="f08db293d3ef80052d6513ece19792642a289fea"

[[override]]
name="sigs.k8s.io/testing_frameworks"
revision="f53464b8b84b4507805a0b033a8377b225163fea"

[[override]]
name = "github.com/thockin/logr"
source = "https://github.com/directxman12/logr.git"
branch = "features/structed"
`
