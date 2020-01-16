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
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/cmd/internal"
	"sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/manager"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/webhook"
)

func newWebhookCmd() *cobra.Command {
	o := webhookOptions{}

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
		Run: func(cmd *cobra.Command, args []string) {
			internal.DieIfNotConfigured()

			projectConfig, err := config.Read()
			if err != nil {
				log.Fatalf("failed to read the configuration file: %v", err)
			}

			if !projectConfig.IsV1() {
				log.Fatalf("webhook scaffolding is not supported for this project version: %s \n", projectConfig.Version)
			}

			if err := o.res.Validate(); err != nil {
				log.Fatal(err)
			}

			fmt.Println("Writing scaffold for you to edit...")

			universe, err := model.NewUniverse(
				model.WithConfig(projectConfig),
				// TODO: missing model.WithBoilerplate[From], needs boilerplate or path
				model.WithResource(o.res, projectConfig),
			)
			if err != nil {
				log.Fatalf("error scaffolding webhook: %v", err)
			}

			webhookConfig := webhook.Config{Server: o.server, Type: o.webhookType, Operations: o.operations}

			err = (&scaffold.Scaffold{}).Execute(
				universe,
				input.Options{},
				&manager.Webhook{},
				&webhook.AdmissionHandler{Resource: o.res, Config: webhookConfig},
				&webhook.AdmissionWebhookBuilder{Resource: o.res, Config: webhookConfig},
				&webhook.AdmissionWebhooks{Resource: o.res, Config: webhookConfig},
				&webhook.AddAdmissionWebhookBuilderHandler{Resource: o.res, Config: webhookConfig},
				&webhook.Server{Config: webhookConfig},
				&webhook.AddServer{Config: webhookConfig},
			)
			if err != nil {
				log.Fatalf("error scaffolding webhook: %v", err)
			}

			if o.doMake {
				fmt.Println("Running make...")
				cm := exec.Command("make") // #nosec
				cm.Stderr = os.Stderr
				cm.Stdout = os.Stdout
				if err := cm.Run(); err != nil {
					log.Fatal(err)
				}
			}
		},
	}
	cmd.Flags().StringVar(&o.server, "server", "default",
		"name of the server")
	cmd.Flags().StringVar(&o.webhookType, "type", "",
		"webhook type, e.g. mutating or validating")
	cmd.Flags().StringSliceVar(&o.operations, "operations", []string{"create"},
		"the operations that the webhook will intercept, e.g. create, update, delete and connect")
	cmd.Flags().BoolVar(&o.doMake, "make", true,
		"if true, run make after generating files")
	o.res = gvkForFlags(cmd.Flags())
	return cmd
}

// webhookOptions represents commandline options for scaffolding a webhook.
type webhookOptions struct {
	res         *resource.Resource
	operations  []string
	server      string
	webhookType string
	doMake      bool
}

// gvkForFlags registers flags for Resource fields and returns the Resource
func gvkForFlags(f *flag.FlagSet) *resource.Resource {
	r := &resource.Resource{}
	f.StringVar(&r.Group, "group", "", "resource Group")
	f.StringVar(&r.Version, "version", "", "resource Version")
	f.StringVar(&r.Kind, "kind", "", "resource Kind")
	f.StringVar(&r.Resource, "resource", "", "resource Resource")
	return r
}
