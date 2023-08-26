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

package templates

import (
	"fmt"
	"strings"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Readme{}

// Readme scaffolds a README.md file
type Readme struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.ProjectNameMixin

	License string
}

// SetTemplateDefaults implements file.Template
func (f *Readme) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "README.md"
	}

	f.License = strings.Replace(
		strings.Replace(f.Boilerplate, "/*", "", 1),
		"*/", "", 1)

	f.TemplateBody = fmt.Sprintf(readmeFileTemplate,
		codeFence("kubectl apply -k config/samples/"),
		codeFence("make docker-build docker-push IMG=<some-registry>/{{ .ProjectName }}:tag"),
		codeFence("make deploy IMG=<some-registry>/{{ .ProjectName }}:tag"),
		codeFence("make uninstall"),
		codeFence("make undeploy"),
		codeFence("make install"),
		codeFence("make run"),
		codeFence("make manifests"))

	return nil
}

//nolint:lll
const readmeFileTemplate = `# {{ .ProjectName }}
// TODO(user): Add simple overview of use/purpose

## Description
// TODO(user): An in-depth paragraph about your project and overview of use

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster ` + "`kubectl cluster-info`" + ` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

%s

2. Build and push your image to the location specified by ` + "`IMG`" + `:

%s

3. Deploy the controller to the cluster with the image specified by ` + "`IMG`" + `:

%s

### Uninstall CRDs
To delete the CRDs from the cluster:

%s

### Undeploy controller
UnDeploy the controller from the cluster:

%s

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

%s

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

%s

**NOTE:** You can also run this in one step by running: ` + "`make install run`" + `

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

%s

**NOTE:** Run ` + "`make --help`" + ` for more information on all potential ` + "`make`" + ` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License
{{ .License }}
`

func codeFence(code string) string {
	return "```sh" + "\n" + code + "\n" + "```"
}
