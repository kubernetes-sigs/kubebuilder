/*
Copyright 2020 The Kubernetes Authors.

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

package v1

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	"sigs.k8s.io/kubebuilder/pkg/plugin/internal"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
)

type createWebhookPlugin struct {
	resource    *resource.Options
	server      string
	webhookType string
	operations  []string
	doMake      bool
}

var (
	_ plugin.CreateWebhook = &createWebhookPlugin{}
	_ cmdutil.RunOptions   = &createAPIPlugin{}
)

func (p createWebhookPlugin) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Scaffold a webhook server if there is no existing server.
Scaffolds webhook handlers based on group, version, kind and other user inputs.
This command is only available for v1 scaffolding project.
`
	ctx.Examples = fmt.Sprintf(`  # Create webhook for CRD of group crew, version v1 and kind FirstMate.
  # Set type to be mutating and operations to be create and update.
  %s alpha webhook --group crew --version v1 --kind FirstMate --type=mutating --operations=create,update
`,
		ctx.CommandName)
}

func (p *createWebhookPlugin) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&p.server, "server", "default", "name of the server")
	fs.StringVar(&p.webhookType, "type", "", "webhook type, e.g. mutating or validating")
	fs.StringSliceVar(&p.operations, "operations", []string{"create"},
		"the operations that the webhook will intercept, e.g. create, update, delete and connect")

	fs.BoolVar(&p.doMake, "make", true, "if true, run make after generating files")

	p.resource = &resource.Options{}
	fs.StringVar(&p.resource.Group, "group", "", "resource Group")
	fs.StringVar(&p.resource.Version, "version", "", "resource Version")
	fs.StringVar(&p.resource.Kind, "kind", "", "resource Kind")
	fs.StringVar(&p.resource.Plural, "resource", "", "resource Resource")
}

func (p *createWebhookPlugin) Run() error {
	return cmdutil.Run(p)
}

func (p *createWebhookPlugin) LoadConfig() (*config.Config, error) {
	return config.LoadInitialized()
}

func (p *createWebhookPlugin) Validate(_ *config.Config) error {
	if err := p.resource.Validate(); err != nil {
		return err
	}
	return nil
}

func (p *createWebhookPlugin) GetScaffolder(c *config.Config) (scaffold.Scaffolder, error) { // nolint:unparam
	// Load the boilerplate
	bp, err := ioutil.ReadFile(filepath.Join("hack", "boilerplate.go.txt")) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("unable to load boilerplate: %v", err)
	}

	// Create the actual resource from the resource options
	res := p.resource.NewV1Resource(&c.Config, false)

	return scaffold.NewV1WebhookScaffolder(&c.Config, string(bp), res, p.server, p.webhookType, p.operations), nil
}

func (p *createWebhookPlugin) PostScaffold(_ *config.Config) error {
	if p.doMake {
		err := internal.RunCmd("Running make", "make")
		if err != nil {
			return err
		}
	}
	return nil
}
