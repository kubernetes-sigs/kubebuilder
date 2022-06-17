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

package v1alpha1

import (
	"fmt"
	"os"
	goPlugin "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang"

	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds"
)

const (
	// defaultCRDVersion is the default CRD API version to scaffold.
	defaultCRDVersion = "v1"
)

const deprecateMsg = "The v1beta1 API version for CRDs and Webhooks are deprecated and are no longer supported since " +
	"the Kubernetes release 1.22. This flag no longer required to exist in future releases. Also, we would like to " +
	"recommend you no longer use these API versions." +
	"More info: https://kubernetes.io/docs/reference/using-api/deprecation-guide/#v1-22"

// DefaultMainPath is default file path of main.go
const DefaultMainPath = "main.go"

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	config config.Config

	options *goPlugin.Options

	resource *resource.Resource

	// image indicates the image that will be used to scaffold the deployment
	image string

	// runMake indicates whether to run make or not after scaffolding APIs
	runMake bool

	// runManifests indicates whether to run manifests or not after scaffolding APIs
	runManifests bool
}

func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	//nolint: lll
	subcmdMeta.Description = `Scaffold the code implementation to deploy and manage your Operand which is represented by the API informed and will be reconciled by its controller. This plugin will generate the code implementation to help you out.
	
	Note: In general, it’s recommended to have one controller responsible for managing each API created for the project to properly follow the design goals set by Controller Runtime(https://github.com/kubernetes-sigs/controller-runtime).

	This plugin will work as the common behaviour of the flag --force and will scaffold the API and controller always. Use core types or external APIs is not officially support by default with.
`
	//nolint: lll
	subcmdMeta.Examples = fmt.Sprintf(`  # Create a frigates API with Group: ship, Version: v1beta1, Kind: Frigate to represent the 
	Image: example.com/frigate:v0.0.1 and its controller with a code to deploy and manage this Operand.
  %[1]s create api --group ship --version v1beta1 --kind Frigate --image=example.com/frigate:v0.0.1 --plugins=deploy-image/v1-alpha

  # Generate the manifests
  make manifests

  # Install CRDs into the Kubernetes cluster using kubectl apply
  make install

  # Regenerate code and run against the Kubernetes cluster configured by ~/.kube/config
  make run
`, cliMeta.CommandName)
}

func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&p.image, "image", "", "inform the Operand image. "+
		"The controller will be scaffolded with an example code to deploy and manage this image.")

	fs.BoolVar(&p.runMake, "make", true, "if true, run `make generate` after generating files")

	fs.BoolVar(&p.runManifests, "manifests", true, "if true, run `make manifests` after generating files")

	p.options = &goPlugin.Options{}

	fs.StringVar(&p.options.CRDVersion, "crd-version", defaultCRDVersion,
		"version of CustomResourceDefinition to scaffold. Options: [v1, v1beta1]")

	fs.StringVar(&p.options.Plural, "plural", "", "resource irregular plural form")
	fs.BoolVar(&p.options.Namespaced, "namespaced", true, "resource is namespaced")

	// (not required raise an error in this case)
	// nolint:errcheck,gosec
	fs.MarkDeprecated("crd-version", deprecateMsg)
}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	p.options.DoAPI = true
	p.options.DoController = true
	p.options.UpdateResource(p.resource, p.config)

	if err := p.resource.Validate(); err != nil {
		return err
	}

	// Check that the provided group can be added to the project
	if !p.config.IsMultiGroup() && p.config.ResourcesLength() != 0 && !p.config.HasGroup(p.resource.Group) {
		return fmt.Errorf("multiple groups are not allowed by default, " +
			"to enable multi-group visit https://kubebuilder.io/migration/multi-group.html")
	}

	// Check CRDVersion against all other CRDVersions in p.config for compatibility.
	if util.HasDifferentCRDVersion(p.config, p.resource.API.CRDVersion) {
		return fmt.Errorf("only one CRD version can be used for all resources, cannot add %q",
			p.resource.API.CRDVersion)
	}

	// Check CRDVersion against all other CRDVersions in p.config for compatibility.
	if util.HasDifferentCRDVersion(p.config, p.resource.API.CRDVersion) {
		return fmt.Errorf("only one CRD version can be used for all resources, cannot add %q",
			p.resource.API.CRDVersion)
	}

	return nil
}

func (p *createAPISubcommand) PreScaffold(machinery.Filesystem) error {
	if len(p.image) == 0 {
		return fmt.Errorf("you MUST inform the image that will be used in the reconciliation")
	}

	// check if main.go is present in the root directory
	if _, err := os.Stat(DefaultMainPath); os.IsNotExist(err) {
		return fmt.Errorf("%s file should present in the root directory", DefaultMainPath)
	}

	return nil
}

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	fmt.Println("updating scaffold with deploy-image/v1alpha1 plugin...")

	scaffolder := scaffolds.NewDeployImageScaffolder(p.config, *p.resource, p.image)
	scaffolder.InjectFS(fs)
	return scaffolder.Scaffold()
}

func (p *createAPISubcommand) PostScaffold() error {
	err := util.RunCmd("Update dependencies", "go", "mod", "tidy")
	if err != nil {
		return err
	}
	if p.runMake {
		err = util.RunCmd("Running make", "make", "generate")
		if err != nil {
			return err
		}
	}

	if p.runManifests {
		err = util.RunCmd("Running make", "make", "manifests")
		if err != nil {
			return err
		}
	}

	fmt.Print("Next: check the implementation of your new API and controller. " +
		"If you do changes in the API run the manifests with:\n$ make manifests\n")

	return nil
}
