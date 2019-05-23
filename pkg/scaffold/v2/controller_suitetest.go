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

package v2

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/markbates/inflect"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v2/internal"
)

var _ input.File = &ControllerSuiteTest{}

// ControllerSuiteTest scaffolds the suite_test.go file to setup the controller test
type ControllerSuiteTest struct {
	input.Input

	// Resource is the resource to scaffold the controller_kind_test.go file for
	Resource *resource.Resource

	// ResourcePackage is the package of the Resource
	ResourcePackage string

	// Plural is the plural lowercase of kind
	Plural string

	// Is the Group + "." + Domain for the Resource
	GroupDomain string
}

// GetInput implements input.File
func (v *ControllerSuiteTest) GetInput() (input.Input, error) {
	if v.Path == "" {
		v.Path = filepath.Join("controllers", "suite_test.go")
	}
	v.TemplateBody = controllerSuiteTestTemplate
	return v.Input, nil
}

// Validate validates the values
func (v *ControllerSuiteTest) Validate() error {
	return v.Resource.Validate()
}

var controllerSuiteTestTemplate = `{{ .Boilerplate }}

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
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	// +kubebuilder:scaffold:imports
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
	[]Reporter{envtest.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}
	
	cfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	// +kubebuilder:scaffold:scheme


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

// UpdateTestSuite updates given file (suite_test.go) with code fragments required for
// adding import paths and code setup for new types.
func (a *ControllerSuiteTest) UpdateTestSuite() error {

	a.ResourcePackage, a.GroupDomain = getResourceInfo(a.Resource, a.Input)
	if a.Plural == "" {
		rs := inflect.NewDefaultRuleset()
		a.Plural = rs.Pluralize(strings.ToLower(a.Resource.Kind))
	}

	ctrlImportCodeFragment := fmt.Sprintf(`"%s/controllers"
`, a.Repo)
	apiImportCodeFragment := fmt.Sprintf(`%s%s "%s/%s"
`, a.Resource.Group, a.Resource.Version, a.ResourcePackage, a.Resource.Version)

	addschemeCodeFragment := fmt.Sprintf(`err = %s%s.AddToScheme(scheme.Scheme)
Expect(err).NotTo(HaveOccurred())

`, a.Resource.Group, a.Resource.Version)

	err := internal.InsertStringsInFile(a.Path,
		apiPkgImportScaffoldMarker, ctrlImportCodeFragment,
		apiPkgImportScaffoldMarker, apiImportCodeFragment,
		apiSchemeScaffoldMarker, addschemeCodeFragment)
	if err != nil {
		return err
	}

	return nil
}
