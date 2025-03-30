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

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Readme{}

// Readme scaffolds a README.md file
type Readme struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.ProjectNameMixin

	License string

	// CommandName stores the name of the bin used
	CommandName string
}

// SetTemplateDefaults implements machinery.Template
func (f *Readme) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "README.md"
	}

	f.License = strings.Replace(
		strings.Replace(f.Boilerplate, "/*", "", 1),
		"*/", "", 1)

	f.TemplateBody = fmt.Sprintf(readmeFileTemplate,
		codeFence("make docker-build docker-push IMG=<some-registry>/{{ .ProjectName }}:tag"),
		codeFence("make install"),
		codeFence("make deploy IMG=<some-registry>/{{ .ProjectName }}:tag"),
		codeFence("kubectl apply -k config/samples/"),
		codeFence("kubectl delete -k config/samples/"),
		codeFence("make uninstall"),
		codeFence("make undeploy"),
		codeFence("make build-installer IMG=<some-registry>/{{ .ProjectName }}:tag"),
		codeFence("kubectl apply -f https://raw.githubusercontent.com/<org>/{{ .ProjectName }}/"+
			"<tag or branch>/dist/install.yaml"),
		codeFence(fmt.Sprintf("%s edit --plugins=helm/v1-alpha", f.CommandName)),
	)

	return nil
}

const readmeFileTemplate = `# {{ .ProjectName }}
// TODO(user): Add simple overview of use/purpose

## Description
// TODO(user): An in-depth paragraph about your project and overview of use

## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by ` + "`IMG`" + `:**

%s

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

%s

**Deploy the Manager to the cluster with the image specified by ` + "`IMG`" + `:**

%s

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

%s

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

%s

**Delete the APIs(CRDs) from the cluster:**

%s

**UnDeploy the controller from the cluster:**

%s

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

%s

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

%s

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

%s

2. See that a chart was generated under 'dist/chart', and users
can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes. Furthermore,
if you create webhooks, you need to use the above command with
the '--force' flag and manually ensure that any custom configuration
previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
is manually re-applied afterwards.

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run ` + "`make help`" + ` for more information on all potential ` + "`make`" + ` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License
{{ .License }}
`

func codeFence(code string) string {
	return "```sh" + "\n" + code + "\n" + "```"
}
