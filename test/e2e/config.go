/*
Copyright 2018 The Kubernetes Authors.

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

package e2e

import (
	"path/filepath"

	"github.com/kubernetes-sigs/kubebuilder/test/e2e/framework"
)

// runtime config specified to run e2e tests
type config struct {
	domain              string
	group               string
	version             string
	kind                string
	installName         string
	namespace           string
	controllerImageName string
	workDir             string
}

// initConfig init with a random suffix for test config stuff,
// to avoid conflict when running tests synchronously.
func initConfig(testSuffix string) *config {
	testGroup := "bar" + testSuffix
	installName := "kube" + testGroup + testSuffix
	testNamespace := installName + "-system"

	return &config{
		domain:              "example.com" + testSuffix,
		group:               testGroup,
		version:             "v1alpha1",
		kind:                "Foo" + testSuffix,
		installName:         installName,
		namespace:           testNamespace,
		controllerImageName: "gcr.io/kubeships/controller-manager:" + testSuffix,
		workDir:             filepath.Join(framework.TestContext.ProjectDir, "e2e-"+testSuffix),
	}
}

type deploymentTemplateArguments struct {
	Namespace string
}

var deploymentTemplate = `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: deployment-example
  namespace: {{ .Namespace }}
  labels:
    name: deployment-example
spec:
  replicas: 3
  selector:
    matchLabels:
      name: nginx
  template:
    metadata:
      labels:
        name: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
`
