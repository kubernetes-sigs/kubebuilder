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
	"errors"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds"
)

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

	// imageCommand indicates the command that we should use to init the deployment
	imageContainerCommand string

	// imageContainerPort indicates the port that we should use in the scaffold
	imageContainerPort string

	// runAsUser indicates the user-id used for running the container
	runAsUser string
}

func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	//nolint:lll
	subcmdMeta.Description = `Scaffold the code implementation to deploy and manage your Operand which is represented by the API informed and will be reconciled by its controller. This plugin will generate the code implementation to help you out.

	Note: In general, it’s recommended to have one controller responsible for managing each API created for the project to properly follow the design goals set by Controller Runtime(https://github.com/kubernetes-sigs/controller-runtime).

	This plugin will work as the common behaviour of the flag --force and will scaffold the API and controller always. Use core types or external APIs is not officially support by default with.
`
	//nolint:lll
	subcmdMeta.Examples = fmt.Sprintf(`  # Create a frigates API with Group: ship, Version: v1beta1, Kind: Frigate to represent the
	Image: example.com/frigate:v0.0.1 and its controller with a code to deploy and manage this Operand.

	Note that in the following example we are also adding the optional options to let you inform the command which should be used to create the container and initialize itvia the flag --image-container-command as the Port that should be used

	- By informing the command (--image-container-command="memcached,--memory-limit=64,-o,modern,-v") your deployment will be scaffold with, i.e.:

		Command: []string{"memcached","--memory-limit=64","-o","modern","-v"},

	- By informing the Port (--image-container-port) will deployment will be scaffold with, i.e:

		Ports: []corev1.ContainerPort{
			ContainerPort: Memcached.Spec.ContainerPort,
			Name:          "Memcached",
		},

	Therefore, the default values informed will be used to scaffold specs for the API.

  %[1]s create api --group example.com --version v1alpha1 --kind Memcached --image=memcached:1.6.15-alpine --image-container-command="memcached --memory-limit=64 modern -v" --image-container-port="11211" --plugins="%[2]s" --make=false --namespaced=false

  # Generate the manifests
  make manifests

  # Install CRDs into the Kubernetes cluster using kubectl apply
  make install

  # Regenerate code and run against the Kubernetes cluster configured by ~/.kube/config
  make run
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&p.image, "image", "", "inform the Operand image. "+
		"The controller will be scaffolded with an example code to deploy and manage this image.")

	fs.StringVar(&p.imageContainerCommand, "image-container-command", "", "[Optional] if informed, "+
		"will be used to scaffold the container command that should be used to init a container to run the image in "+
		"the controller and its spec in the API (CRD/CR). (i.e. "+
		"--image-container-command=\"memcached,--memory-limit=64,modern,-o,-v\")")
	fs.StringVar(&p.imageContainerPort, "image-container-port", "", "[Optional] if informed, "+
		"will be used to scaffold the container port that should be used by container image in "+
		"the controller and its spec in the API (CRD/CR). (i.e --image-container-port=\"11211\") ")
	fs.StringVar(&p.runAsUser, "run-as-user", "", "User-Id for the container formed will be set to this value")

	fs.BoolVar(&p.runMake, "make", true, "if true, run `make generate` after generating files")
	fs.BoolVar(&p.runManifests, "manifests", true, "if true, run `make manifests` after generating files")

	p.options = &goPlugin.Options{}

	fs.StringVar(&p.options.Plural, "plural", "", "resource irregular plural form")
}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res
	p.options.DoAPI = true
	p.options.DoController = true
	p.options.Namespaced = true

	p.options.UpdateResource(p.resource, p.config)

	if err := p.resource.Validate(); err != nil {
		return err
	}

	// Check that the provided group can be added to the project
	if !p.config.IsMultiGroup() && p.config.ResourcesLength() != 0 && !p.config.HasGroup(p.resource.Group) {
		return fmt.Errorf("multiple groups are not allowed by default, " +
			"to enable multi-group visit https://kubebuilder.io/migration/multi-group.html")
	}

	return nil
}

func (p *createAPISubcommand) PreScaffold(machinery.Filesystem) error {
	if len(p.image) == 0 {
		return fmt.Errorf("you MUST inform the image that will be used in the reconciliation")
	}

	isGoV3 := false
	for _, pluginKey := range p.config.GetPluginChain() {
		if strings.Contains(pluginKey, "go.kubebuilder.io/v3") {
			isGoV3 = true
		}
	}

	defaultMainPath := "cmd/main.go"
	if isGoV3 {
		defaultMainPath = "main.go"
	}
	// check if main.go is present in the cmd/ directory
	if _, err := os.Stat(defaultMainPath); os.IsNotExist(err) {
		return fmt.Errorf("main.go file should be present in %s", defaultMainPath)
	}

	return nil
}

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	log.Println("updating scaffold with deploy-image/v1alpha1 plugin...")

	scaffolder := scaffolds.NewDeployImageScaffolder(p.config,
		*p.resource,
		p.image,
		p.imageContainerCommand,
		p.imageContainerPort,
		p.runAsUser)
	scaffolder.InjectFS(fs)
	err := scaffolder.Scaffold()
	if err != nil {
		return err
	}

	// Track the resources following a declarative approach
	cfg := PluginConfig{}
	if err := p.config.DecodePluginConfig(pluginKey, &cfg); errors.As(err, &config.UnsupportedFieldError{}) {
		// Skip tracking as the config doesn't support per-plugin configuration
		return nil
	} else if err != nil && !errors.As(err, &config.PluginKeyNotFoundError{}) {
		// Fail unless the key wasn't found, which just means it is the first resource tracked
		return err
	}

	configDataOptions := options{
		Image:            p.image,
		ContainerCommand: p.imageContainerCommand,
		ContainerPort:    p.imageContainerPort,
		RunAsUser:        p.runAsUser,
	}
	cfg.Resources = append(cfg.Resources, ResourceData{
		Group:   p.resource.GVK.Group,
		Domain:  p.resource.GVK.Domain,
		Version: p.resource.GVK.Version,
		Kind:    p.resource.GVK.Kind,
		Options: configDataOptions,
	})
	return p.config.EncodePluginConfig(pluginKey, cfg)
}

func (p *createAPISubcommand) PostScaffold() error {
	err := util.RunCmd("Update dependencies", "go", "mod", "tidy")
	if err != nil {
		return err
	}
	if p.runMake && p.resource.HasAPI() {
		err = util.RunCmd("Running make", "make", "generate")
		if err != nil {
			return err
		}
	}

	if p.runManifests && p.resource.HasAPI() {
		err = util.RunCmd("Running make", "make", "manifests")
		if err != nil {
			return err
		}
	}

	fmt.Print("Next: check the implementation of your new API and controller. " +
		"If you do changes in the API run the manifests with:\n$ make manifests\n")

	return nil
}
