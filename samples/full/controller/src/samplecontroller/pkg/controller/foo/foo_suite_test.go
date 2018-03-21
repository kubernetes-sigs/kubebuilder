
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


package foo_test

import (
    "testing"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    "github.com/kubernetes-sigs/kubebuilder/pkg/controller"
    "github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
    "github.com/kubernetes-sigs/kubebuilder/pkg/test"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"

    "samplecontroller/pkg/client/clientset/versioned"
    "samplecontroller/pkg/inject"
    "samplecontroller/pkg/inject/args"
)

var (
    testenv *test.TestEnvironment
    config *rest.Config
    cs *versioned.Clientset
    ks *kubernetes.Clientset
    shutdown chan struct{}
    ctrl *controller.GenericController
)

func TestBee(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecsWithDefaultAndCustomReporters(t, "Foo Suite", []Reporter{test.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
    testenv = &test.TestEnvironment{CRDs: inject.Injector.CRDs}
    var err error
    config, err = testenv.Start()
    Expect(err).NotTo(HaveOccurred())
    cs = versioned.NewForConfigOrDie(config)
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
    Eventually(func() interface{} { return arguments.ControllerManager.GetController("FooController") }).
        Should(Not(BeNil()))
    ctrl = arguments.ControllerManager.GetController("FooController")
})

var _ = AfterSuite(func() {
    close(shutdown)
    testenv.Stop()
})
