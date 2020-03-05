/*
Copyright 2019 The Kubernetes Authors.

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

package controller

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &SuiteTest{}
var _ file.Inserter = &SuiteTest{}

// SuiteTest scaffolds the suite_test.go file to setup the controller test
type SuiteTest struct {
	file.TemplateMixin
	file.RepositoryMixin
	file.MultiGroupMixin
	file.BoilerplateMixin
	file.ResourceMixin
}

// SetTemplateDefaults implements file.Template
func (f *SuiteTest) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup {
			f.Path = filepath.Join("controllers", "%[group]", "suite_test.go")
		} else {
			f.Path = filepath.Join("controllers", "suite_test.go")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = fmt.Sprintf(controllerSuiteTestTemplate,
		file.NewMarkerFor(f.Path, importMarker),
		file.NewMarkerFor(f.Path, addSchemeMarker),
	)

	return nil
}

const (
	importMarker    = "imports"
	addSchemeMarker = "scheme"
)

// GetMarkers implements file.Inserter
func (f *SuiteTest) GetMarkers() []file.Marker {
	return []file.Marker{
		file.NewMarkerFor(f.Path, importMarker),
		file.NewMarkerFor(f.Path, addSchemeMarker),
	}
}

const (
	apiImportCodeFragment = `%s "%s"
`
	addschemeCodeFragment = `err = %s.AddToScheme(scheme.Scheme)
Expect(err).NotTo(HaveOccurred())

`
)

// GetCodeFragments implements file.Inserter
func (f *SuiteTest) GetCodeFragments() file.CodeFragmentsMap {
	fragments := make(file.CodeFragmentsMap, 2)

	// Generate import code fragments
	imports := make([]string, 0)
	imports = append(imports, fmt.Sprintf(apiImportCodeFragment, f.Resource.ImportAlias, f.Resource.Package))

	// Generate add scheme code fragments
	addScheme := make([]string, 0)
	addScheme = append(addScheme, fmt.Sprintf(addschemeCodeFragment, f.Resource.ImportAlias))

	// Only store code fragments in the map if the slices are non-empty
	if len(imports) != 0 {
		fragments[file.NewMarkerFor(f.Path, importMarker)] = imports
	}
	if len(addScheme) != 0 {
		fragments[file.NewMarkerFor(f.Path, addSchemeMarker)] = addScheme
	}

	return fragments
}

const controllerSuiteTestTemplate = `{{ .Boilerplate }}

package controllers

import (
	"path/filepath"
	"testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	%s
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
	"Controller Suite",
	[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	%s

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
`
