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

package controller

import (
	"fmt"
	"path/filepath"
	"strings"

	createutil "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/util"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
)

func doControllerTest(dir string, args controllerTemplateArgs) bool {
	path := filepath.Join(dir, "pkg", "controller", strings.ToLower(createutil.KindName),
		fmt.Sprintf("%s_suite_test.go",
			strings.ToLower(createutil.KindName)))
	util.WriteIfNotFound(path, "resource-controller-suite-test-template", controllerSuiteTestTemplate, args)

	path = filepath.Join(dir, "pkg", "controller", strings.ToLower(createutil.KindName), "controller_test.go")
	fmt.Printf("\t%s\n", filepath.Join(
		"pkg", "controller", strings.ToLower(createutil.KindName), "controller_test.go"))
	return util.WriteIfNotFound(path, "controller-test-template", controllerTestTemplate, args)
}

var controllerSuiteTestTemplate = `
{{.BoilerPlate}}

package {{lower .Kind}}_test

import (
    "testing"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    "github.com/kubernetes-sigs/kubebuilder/pkg/controller"
    "github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
    "github.com/kubernetes-sigs/kubebuilder/pkg/test"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    {{if not .CoreType}}"{{ .Repo }}/pkg/client/clientset/versioned"{{end}}
    "{{ .Repo }}/pkg/inject"
    "{{ .Repo }}/pkg/inject/args"
)

var (
    testenv *test.TestEnvironment
    config *rest.Config
    {{if not .CoreType}}
    cs *versioned.Clientset
    {{end}}
    ks *kubernetes.Clientset
    shutdown chan struct{}
    ctrl *controller.GenericController
)

func TestBee(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecsWithDefaultAndCustomReporters(t, "{{ .Kind }} Suite", []Reporter{test.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
    testenv = &test.TestEnvironment{CRDs: inject.Injector.CRDs}
    var err error
    config, err = testenv.Start()
    Expect(err).NotTo(HaveOccurred())
    {{if not .CoreType}}
    cs = versioned.NewForConfigOrDie(config)
    {{end}}
    ks = kubernetes.NewForConfigOrDie(config)

    shutdown = make(chan struct{})
    arguments := args.CreateInjectArgs(config)
    go func() {
        defer GinkgoRecover()
        Expect(inject.RunAll(run.RunArguments{Stop: shutdown}, arguments)).
            To(BeNil())
    }()

    // Wait for RunAll to create the controllers and then set the reference
    defer GinkgoRecover()
    Eventually(func() interface{} { return arguments.ControllerManager.GetController("{{.Kind}}Controller") }).
        Should(Not(BeNil()))
    ctrl = arguments.ControllerManager.GetController("{{.Kind}}Controller")
})

var _ = AfterSuite(func() {
    close(shutdown)
    testenv.Stop()
})
`

var controllerTestTemplate = `
{{.BoilerPlate}}

package {{ lower .Kind }}_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    {{if .CoreType}}
    . "k8s.io/api/{{.Group}}/{{.Version}}"
    . "k8s.io/client-go/kubernetes/typed/{{.Group}}/{{.Version}}"
    {{else}}
    . "{{ .Repo }}/pkg/apis/{{ .Group }}/{{ .Version }}"
    . "{{ .Repo }}/pkg/client/clientset/versioned/typed/{{ .Group }}/{{ .Version }}"
    {{end}}
)

// EDIT THIS FILE!
// Created by "kubebuilder create resource" for you to implement controller logic tests

var _ = Describe("{{ .Kind }} controller", func() {
    var instance {{ .Kind }}
    var expectedKey types.ReconcileKey
    var client {{ .Kind }}Interface

    BeforeEach(func() {
        instance = {{ .Kind }}{}
        instance.Name = "instance-1"
        expectedKey = types.ReconcileKey{
            Namespace: "{{ if not .NonNamespacedKind }}default{{ end }}",
            Name: "instance-1",
        }
    })

    AfterEach(func() {
        client.Delete(instance.Name, &metav1.DeleteOptions{})
    })

    Describe("when creating a new object", func() {
        It("invoke the reconcile method", func() {
            after := make(chan struct{})
            ctrl.AfterReconcile = func(key types.ReconcileKey, err error) {
                defer func() {
                    // Recover in case the key is reconciled multiple times
                    defer func() { recover() }()
                    close(after)
                }()
                defer GinkgoRecover()
                Expect(key).To(Equal(expectedKey))
                Expect(err).ToNot(HaveOccurred())
            }

            // Create the instance
            {{if .CoreType}}
            client = ks.{{title .Group}}{{title .Version}}().{{ plural .Kind }}({{ if not .NonNamespacedKind }}"default"{{ end }})
            {{else}}
            client = cs.{{title .Group}}{{title .Version}}().{{ plural .Kind }}({{ if not .NonNamespacedKind }}"default"{{ end }})
            {{end}}
            _, err := client.Create(&instance)
            Expect(err).ShouldNot(HaveOccurred())

            // Wait for reconcile to happen
            Eventually(after, "10s", "100ms").Should(BeClosed())

            // INSERT YOUR CODE HERE - test conditions post reconcile
        })
    })
})
`
