/*
Copyright 2017 The Kubernetes Authors.

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
	"strings"

	createutil "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/util"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
)

func doResourceTest(dir string, args resourceTemplateArgs) bool {
	typesFileName := fmt.Sprintf("%s_suite_test.go", strings.ToLower(createutil.VersionName))
	path := filepath.Join(dir, "pkg", "apis", createutil.GroupName, createutil.VersionName, typesFileName)
	util.WriteIfNotFound(path, "version-suite-test-template", resourceSuiteTestTemplate, args)

	typesFileName = fmt.Sprintf("%s_types_test.go", strings.ToLower(createutil.KindName))
	path = filepath.Join(dir, "pkg", "apis", createutil.GroupName, createutil.VersionName, typesFileName)
	fmt.Printf("\t%s\n", filepath.Join(
		"pkg", "apis", createutil.GroupName, createutil.VersionName, typesFileName,
	))
	return util.WriteIfNotFound(path, "resource-test-template", resourceTestTemplate, args)
}

var resourceSuiteTestTemplate = `
{{.BoilerPlate}}

package {{.Version}}_test

import (
    "testing"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    "github.com/kubernetes-sigs/kubebuilder/pkg/test"
    "k8s.io/client-go/rest"

    "{{ .Repo }}/pkg/inject"
    "{{ .Repo }}/pkg/client/clientset/versioned"
)

var testenv *test.TestEnvironment
var config *rest.Config
var cs *versioned.Clientset

func Test{{title .Version}}(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecsWithDefaultAndCustomReporters(t, "v1 Suite", []Reporter{test.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
    testenv = &test.TestEnvironment{CRDs: inject.Injector.CRDs}

    var err error
    config, err = testenv.Start()
    Expect(err).NotTo(HaveOccurred())

    cs = versioned.NewForConfigOrDie(config)
})

var _ = AfterSuite(func() {
    testenv.Stop()
})
`

var resourceTestTemplate = `
{{.BoilerPlate}}

package {{.Version}}_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

    . "{{.Repo}}/pkg/apis/{{.Group}}/{{.Version}}"
    . "{{.Repo}}/pkg/client/clientset/versioned/typed/{{.Group}}/{{.Version}}"
)

// EDIT THIS FILE!
// Created by "kubebuilder create resource" for you to implement the {{.Kind}} resource tests

var _ = Describe("{{.Kind}}", func() {
    var instance {{ .Kind}}
    var expected {{ .Kind}}
    var client {{ .Kind}}Interface

    BeforeEach(func() {
        instance = {{ .Kind}}{}
        instance.Name = "instance-1"

        expected = instance
    })

    AfterEach(func() {
        client.Delete(instance.Name, &metav1.DeleteOptions{})
    })

    // INSERT YOUR CODE HERE - add more "Describe" tests

    // Automatically created storage tests
    Describe("when sending a storage request", func() {
        Context("for a valid config", func() {
            It("should provide CRUD access to the object", func() {
                client = cs.{{ title .Group}}{{title .Version}}().{{plural .Kind}}({{ if not .NonNamespacedKind }}"default"{{ end }})

                By("returning success from the create request")
                actual, err := client.Create(&instance)
                Expect(err).ShouldNot(HaveOccurred())

                By("defaulting the expected fields")
                Expect(actual.Spec).To(Equal(expected.Spec))

                By("returning the item for list requests")
                result, err := client.List(metav1.ListOptions{})
                Expect(err).ShouldNot(HaveOccurred())
                Expect(result.Items).To(HaveLen(1))
                Expect(result.Items[0].Spec).To(Equal(expected.Spec))

                By("returning the item for get requests")
                actual, err = client.Get(instance.Name, metav1.GetOptions{})
                Expect(err).ShouldNot(HaveOccurred())
                Expect(actual.Spec).To(Equal(expected.Spec))

                By("deleting the item for delete requests")
                err = client.Delete(instance.Name, &metav1.DeleteOptions{})
                Expect(err).ShouldNot(HaveOccurred())
                result, err = client.List(metav1.ListOptions{})
                Expect(err).ShouldNot(HaveOccurred())
                Expect(result.Items).To(HaveLen(0))
            })
        })
    })
})
`
