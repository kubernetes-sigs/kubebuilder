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

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/cmd/internal"
	"sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
)

type webhookError struct {
	err error
}

func (e webhookError) Error() string {
	return fmt.Sprintf("failed to create webhook: %v", e.err)
}

func newWebhookCmd() *cobra.Command {
	options := &webhookV1Options{}

	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Scaffold a webhook server",
		Long: `Scaffold a webhook server if there is no existing server.
Scaffolds webhook handlers based on group, version, kind and other user inputs.
This command is only available for v1 scaffolding project.
`,
		Example: `	# Create webhook for CRD of group crew, version v1 and kind FirstMate.
	# Set type to be mutating and operations to be create and update.
	kubebuilder alpha webhook --group crew --version v1 --kind FirstMate --type=mutating --operations=create,update
`,
		Run: func(_ *cobra.Command, _ []string) {
			if err := run(options); err != nil {
				log.Fatal(webhookError{err})
			}
		},
	}

	options.bindFlags(cmd)

	return cmd
}

var _ commandOptions = &webhookV1Options{}

type webhookV1Options struct {
	resource    *resource.Options
	server      string
	webhookType string
	operations  []string
	doMake      bool
}

func (o *webhookV1Options) bindFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.server, "server", "default", "name of the server")
	cmd.Flags().StringVar(&o.webhookType, "type", "", "webhook type, e.g. mutating or validating")
	cmd.Flags().StringSliceVar(&o.operations, "operations", []string{"create"},
		"the operations that the webhook will intercept, e.g. create, update, delete and connect")

	cmd.Flags().BoolVar(&o.doMake, "make", true, "if true, run make after generating files")

	o.resource = &resource.Options{}
	cmd.Flags().StringVar(&o.resource.Group, "group", "", "resource Group")
	cmd.Flags().StringVar(&o.resource.Version, "version", "", "resource Version")
	cmd.Flags().StringVar(&o.resource.Kind, "kind", "", "resource Kind")
	cmd.Flags().StringVar(&o.resource.Plural, "resource", "", "resource Resource")
}

func (o *webhookV1Options) loadConfig() (*config.Config, error) {
	projectConfig, err := config.Load()
	if os.IsNotExist(err) {
		return nil, errors.New("unable to find configuration file, project must be initialized")
	}

	return projectConfig, err
}

func (o *webhookV1Options) validate(c *config.Config) error {
	if !c.IsV1() {
		return fmt.Errorf("webhook scaffolding is no longer alpha for version %s", c.Version)
	}

	if err := o.resource.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *webhookV1Options) scaffolder(c *config.Config) (scaffold.Scaffolder, error) {
	// Load the boilerplate
	bp, err := ioutil.ReadFile(filepath.Join("hack", "boilerplate.go.txt")) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("unable to load boilerplate: %v", err)
	}

	// Create the actual resource from the resource options
	var res *resource.Resource
	switch {
	case c.IsV1():
		res = o.resource.NewV1Resource(&c.Config, false)
	case c.IsV2():
		res = o.resource.NewResource(&c.Config, false)
	default:
		return nil, fmt.Errorf("unknown project version %v", c.Version)
	}

	return scaffold.NewV1WebhookScaffolder(&c.Config, string(bp), res, o.server, o.webhookType, o.operations), nil
}

func (o *webhookV1Options) postScaffold(_ *config.Config) error {
	if o.doMake {
		err := internal.RunCmd("Running make", "make")
		if err != nil {
			return err
		}
	}

	return nil
}

func newWebhookV2Cmd() *cobra.Command {
	options := &webhookV2Options{}

	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Scaffold a webhook for an API resource.",
		Long: `Scaffold a webhook for an API resource. You can choose to scaffold defaulting, ` +
			`validating and (or) conversion webhooks.`,
		Example: `	# Create defaulting and validating webhooks for CRD of group crew, version v1 and kind FirstMate.
	kubebuilder create webhook --group crew --version v1 --kind FirstMate --defaulting --programmatic-validation

	# Create conversion webhook for CRD of group crew, version v1 and kind FirstMate.
	kubebuilder create webhook --group crew --version v1 --kind FirstMate --conversion
`,
		Run: func(_ *cobra.Command, _ []string) {
			if err := run(options); err != nil {
				log.Fatal(webhookError{err})
			}
		},
	}

	options.bindFlags(cmd)

	return cmd
}

var _ commandOptions = &webhookV2Options{}

type webhookV2Options struct {
	resource   *resource.Options
	defaulting bool
	validation bool
	conversion bool
}

func (o *webhookV2Options) bindFlags(cmd *cobra.Command) {
	o.resource = &resource.Options{}
	cmd.Flags().StringVar(&o.resource.Group, "group", "", "resource Group")
	cmd.Flags().StringVar(&o.resource.Version, "version", "", "resource Version")
	cmd.Flags().StringVar(&o.resource.Kind, "kind", "", "resource Kind")
	cmd.Flags().StringVar(&o.resource.Plural, "resource", "", "resource Resource")

	cmd.Flags().BoolVar(&o.defaulting, "defaulting", false,
		"if set, scaffold the defaulting webhook")
	cmd.Flags().BoolVar(&o.validation, "programmatic-validation", false,
		"if set, scaffold the validating webhook")
	cmd.Flags().BoolVar(&o.conversion, "conversion", false,
		"if set, scaffold the conversion webhook")
}

func (o *webhookV2Options) loadConfig() (*config.Config, error) {
	projectConfig, err := config.Load()
	if os.IsNotExist(err) {
		return nil, errors.New("unable to find configuration file, project must be initialized")
	}

	return projectConfig, err
}

func (o *webhookV2Options) validate(c *config.Config) error {
	if c.IsV1() {
		return fmt.Errorf("webhook scaffolding is alpha for version %s", c.Version)
	}

	if err := o.resource.Validate(); err != nil {
		return err
	}

	if !o.defaulting && !o.validation && !o.conversion {
		return errors.New("kubebuilder webhook requires at least one of" +
			" --defaulting, --programmatic-validation and --conversion to be true")
	}

	return nil
}

func (o *webhookV2Options) scaffolder(c *config.Config) (scaffold.Scaffolder, error) { //nolint:unparam
	// Load the boilerplate
	bp, err := ioutil.ReadFile(filepath.Join("hack", "boilerplate.go.txt")) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("unable to load boilerplate: %v", err)
	}

	// Create the actual resource from the resource options
	var res *resource.Resource
	switch {
	case c.IsV1():
		res = o.resource.NewV1Resource(&c.Config, false)
	case c.IsV2():
		res = o.resource.NewResource(&c.Config, false)
	default:
		return nil, fmt.Errorf("unknown project version %v", c.Version)
	}

	return scaffold.NewV2WebhookScaffolder(&c.Config, string(bp), res, o.defaulting, o.validation, o.conversion), nil
}

func (o *webhookV2Options) postScaffold(_ *config.Config) error {
	return nil
}
