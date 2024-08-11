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

package controllers

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &SuiteTest{}
var _ machinery.Inserter = &SuiteTest{}

// SuiteTest scaffolds the file that sets up the controller tests
// nolint:maligned
type SuiteTest struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	// K8SVersion define the k8s version used to do the scaffold
	// so that is possible retrieve the binaries
	K8SVersion string

	// CRDDirectoryRelativePath define the Path for the CRD
	CRDDirectoryRelativePath string

	Force bool
}

// SetTemplateDefaults implements file.Template
func (f *SuiteTest) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("internal", "controller", "%[group]", "suite_test.go")
		} else {
			f.Path = filepath.Join("internal", "controller", "suite_test.go")
		}
	}

	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Println(f.Path)

	f.TemplateBody = fmt.Sprintf(controllerSuiteTestTemplate,
		machinery.NewMarkerFor(f.Path, importMarker),
		machinery.NewMarkerFor(f.Path, addSchemeMarker),
	)

	// If is multigroup the path needs to be ../../ since it has
	// the group dir.
	f.CRDDirectoryRelativePath = `"..",".."`
	if f.MultiGroup && f.Resource.Group != "" {
		f.CRDDirectoryRelativePath = `"..", "..",".."`
	}

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	}

	return nil
}

const (
	importMarker    = "imports"
	addSchemeMarker = "scheme"
)

// GetMarkers implements file.Inserter
func (f *SuiteTest) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(f.Path, importMarker),
		machinery.NewMarkerFor(f.Path, addSchemeMarker),
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
func (f *SuiteTest) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap, 2)

	// Generate import code fragments
	imports := make([]string, 0)
	if f.Resource.Path != "" {
		imports = append(imports, fmt.Sprintf(apiImportCodeFragment, f.Resource.ImportAlias(), f.Resource.Path))
	}

	// Generate add scheme code fragments
	addScheme := make([]string, 0)
	if f.Resource.Path != "" {
		addScheme = append(addScheme, fmt.Sprintf(addschemeCodeFragment, f.Resource.ImportAlias()))
	}

	// Only store code fragments in the map if the slices are non-empty
	if len(imports) != 0 {
		fragments[machinery.NewMarkerFor(f.Path, importMarker)] = imports
	}
	if len(addScheme) != 0 {
		fragments[machinery.NewMarkerFor(f.Path, addSchemeMarker)] = addScheme
	}

	return fragments
}

const controllerSuiteTestTemplate = `{{ .Boilerplate }}

{{if and .MultiGroup .Resource.Group }}
package {{ .Resource.PackageName }}
{{else}}
package controller
{{end}}

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	%s
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join({{ .CRDDirectoryRelativePath }}, "config", "crd", "bases")},
		ErrorIfCRDPathMissing: {{ .Resource.HasAPI }},

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join({{ .CRDDirectoryRelativePath }}, "bin", "k8s",
			fmt.Sprintf("{{ .K8SVersion }}-%%s-%%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	%s

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
`
