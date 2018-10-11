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

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-tools/pkg/scaffold"
	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
	"sigs.k8s.io/controller-tools/pkg/scaffold/resource"
	"sigs.k8s.io/controller-tools/pkg/scaffold/webhook"
)

func newWebhookCmd() *cobra.Command {
	o := webhookOptions{}

	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Scaffold a webhook server",
		Long: `Scaffold a webhook server if there is no existing server.
Scaffolds webhook handlers based on group, version, kind and other user inputs.
`,
		Example: `	# Create webhook for CRD of group crew, version v1 and kind FirstMate.
	# Set type to be mutating and operations to be create and update.
	kubebuilder webhook --group crew --version v1 --kind FirstMate --type=mutating --operations=create,update
`,
		Run: func(cmd *cobra.Command, args []string) {
			dieIfNoProject()

			fmt.Println("Writing scaffold for you to edit...")

			if len(o.res.Resource) == 0 {
				gvr, _ := meta.UnsafeGuessKindToResource(schema.GroupVersionKind{
					Group: o.res.Group, Version: o.res.Version, Kind: o.res.Kind})
				o.res.Resource = gvr.Resource
			}

			err := (&scaffold.Scaffold{}).Execute(input.Options{},
				&webhook.AdmissionHandler{Resource: o.res, Config: webhook.Config{Server: o.server, Type: o.webhookType, Operations: o.operations}},
				&webhook.AdmissionWebhookBuilder{Resource: o.res, Config: webhook.Config{Server: o.server, Type: o.webhookType, Operations: o.operations}},
				&webhook.AdmissionWebhooks{Resource: o.res, Config: webhook.Config{Server: o.server, Type: o.webhookType, Operations: o.operations}},
				&webhook.AddAdmissionWebhookBuilderHandler{Resource: o.res, Config: webhook.Config{Server: o.server, Type: o.webhookType, Operations: o.operations}},
				&webhook.Server{Resource: o.res, Config: webhook.Config{Server: o.server, Type: o.webhookType, Operations: o.operations}},
				&webhook.AddServer{Resource: o.res, Config: webhook.Config{Server: o.server, Type: o.webhookType, Operations: o.operations}},
			)
			if err != nil {
				log.Fatal(err)
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
