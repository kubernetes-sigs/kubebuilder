**Running End-to-end Tests on Remote Clusters**

**This document is for kubebuilder v1 only**

This article outlines steps to run e2e tests on remote clusters for controllers created using `kubebuilder`. For example, after developing a database controller, the developer may want to run some e2e tests on a GKE cluster to verify the controller is working as expected. Currently, `kubebuilder` does not provide a template for running the e2e tests. This article serves to address this deficit.

The steps are as follow:
1.  Create a test file named `<some-file-name>_test.go` populated with template below (referring [this](https://github.com/foxish/application/blob/master/e2e/main_test.go)):
```
import (
    "k8s.io/client-go/tools/clientcmd"
    clientset "k8s.io/redis-operator/pkg/client/clientset/versioned/typed/<some-group>/<some-version>"
    ......
)

// Specify kubeconfig file
func getClientConfig() (*rest.Config, error) {
    return clientcmd.BuildConfigFromFlags("", path.Join(os.Getenv("HOME"), "<file-path>"))
}

// Set up test environment
var _ = Describe("<some-controller-name> should work", func() {
    config, err := getClientConfig()
    if err != nil {
        ......
    }

    // Construct kubernetes client
    k8sClient, err := kubernetes.NewForConfig(config)
    if err != nil {
        ......
    }

    // Construct controller client
    client, err := clientset.NewForConfig(config)
    if err != nil {
        ......
    }

    BeforeEach(func() {
        // Create environment-specific resources such as controller image StatefulSet,
        // CRDs etc. Note: refer "install.yaml" created via "kubebuilder create config"
        // command to have an idea of what resources to be created.
        ......
    })

    AfterEach(func() {
        // Delete all test-specific resources
        ......

        // Delete all environment-specific resources
        ......
    })

    // Declare a list of testing specifications with corresponding test functions
    // Note: test-specific resources are normally created within the test functions
    It("should do something", func() {
        testDoSomething(k8sClient, roClient)
    })

    ......
```
2.  Write some controller-specific e2e tests
3.  Build controller image and upload it to an image storage website such as [gcr.io](https://cloud.google.com/container-registry/)
4.  `go test <path-to-test-file>`
