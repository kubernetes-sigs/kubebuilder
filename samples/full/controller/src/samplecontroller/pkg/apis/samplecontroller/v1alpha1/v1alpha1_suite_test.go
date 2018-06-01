
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


package v1alpha1_test

import (
    "testing"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    "github.com/kubernetes-sigs/kubebuilder/pkg/test"
    "k8s.io/client-go/rest"

    "github.com/kubernetes-sigs/kubebuilder/samples/full/controller/src/samplecontroller/pkg/inject"
    "github.com/kubernetes-sigs/kubebuilder/samples/full/controller/src/samplecontroller/pkg/client/clientset/versioned"
)

var testenv *test.TestEnvironment
var config *rest.Config
var cs *versioned.Clientset

func TestV1alpha1(t *testing.T) {
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
