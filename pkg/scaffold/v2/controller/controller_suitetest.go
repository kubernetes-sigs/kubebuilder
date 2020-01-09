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

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/util"
	scaffoldv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v2/internal"
)

var _ input.File = &SuiteTest{}

// SuiteTest scaffolds the suite_test.go file to setup the controller test
type SuiteTest struct {
	input.Input

	// Resource is the Resource to make the Controller for
	Resource *resource.Resource
}

// GetInput implements input.File
func (f *SuiteTest) GetInput() (input.Input, error) {

	if f.Path == "" {
		if f.MultiGroup {
			f.Path = filepath.Join("controllers", f.Resource.Group, "suite_test.go")
		} else {
			f.Path = filepath.Join("controllers", "suite_test.go")
		}
	}

	f.TemplateBody = controllerSuiteTestTemplate
	return f.Input, nil
}

// Validate validates the values
func (f *SuiteTest) Validate() error {
	return f.Resource.Validate()
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
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
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

// Update updates given file (suite_test.go) with code fragments required for
// adding import paths and code setup for new types.
func (f *SuiteTest) Update() error {

	resourcePackage, _ := util.GetResourceInfo(f.Resource, f.Repo, f.Domain, f.MultiGroup)

	ctrlImportCodeFragment := fmt.Sprintf(`"%s/controllers"
`, f.Repo)
	apiImportCodeFragment := fmt.Sprintf(`%s%s "%s/%s"
`, f.Resource.GroupImportSafe, f.Resource.Version, resourcePackage, f.Resource.Version)

	addschemeCodeFragment := fmt.Sprintf(`err = %s%s.AddToScheme(scheme.Scheme)
Expect(err).NotTo(HaveOccurred())

`, f.Resource.GroupImportSafe, f.Resource.Version)

	err := internal.InsertStringsInFile(f.Path,
		map[string][]string{
			scaffoldv2.APIPkgImportScaffoldMarker: {ctrlImportCodeFragment, apiImportCodeFragment},
			scaffoldv2.APISchemeScaffoldMarker:    {addschemeCodeFragment},
		})
	if err != nil {
		return err
	}

	return nil
}
