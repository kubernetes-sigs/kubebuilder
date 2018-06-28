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

package resource

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &TypesTest{}

// VersionSuiteTest scaffolds the version_suite_test.go file to setup the versions test
type VersionSuiteTest struct {
	input.Input

	// Resource is the resource to scaffold the types_test.go file for
	Resource *Resource
}

// GetInput implements input.File
func (v *VersionSuiteTest) GetInput() (input.Input, error) {
	if v.Path == "" {
		v.Path = filepath.Join("pkg", "apis", v.Resource.Group, v.Resource.Version,
			fmt.Sprintf("%s_suite_test.go", v.Resource.Version))
	}
	v.TemplateBody = versionSuiteTestTemplate
	return v.Input, nil
}

var versionSuiteTestTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var cfg *rest.Config
var c client.Client

func TestMain(m *testing.M) {
	t := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "..", "config", "crds")},
	}

	err := SchemeBuilder.AddToScheme(scheme.Scheme)
	if err != nil {
		log.Fatal(err)
	}

	if cfg, err = t.Start(); err != nil {
		log.Fatal(err)
	}

	if c, err = client.New(cfg, client.Options{Scheme: scheme.Scheme}); err != nil {
		log.Fatal(err)
	}

	code := m.Run()
	t.Stop()
	os.Exit(code)
}
`
