package v1alpha1_test

import (
	"testing"

	"github.com/kubernetes-sigs/kubebuilder/pkg/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"

	"github.com/kubernetes-sigs/kubebuilder/samples/memcached-api-server/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/kubebuilder/samples/memcached-api-server/pkg/inject"
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
