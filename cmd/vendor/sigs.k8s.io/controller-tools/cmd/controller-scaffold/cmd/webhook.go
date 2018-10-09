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

package cmd

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

var res *resource.Resource
var operations []string
var server string
var webhookType string

// WebhookCmd represents the webhook command
var WebhookCmd = &cobra.Command{
	Use:     "webhook",
	Short:   "Scaffold a webhook server",
	Long:    `Scaffold a webhook server`,
	Example: `webhook example: TBD`,
	Run: func(cmd *cobra.Command, args []string) {
		DieIfNoProject()

		fmt.Println("Writing scaffold for you to edit...")

		if len(res.Resource) == 0 {
			gvr, _ := meta.UnsafeGuessKindToResource(schema.GroupVersionKind{
				Group: res.Group, Version: res.Version, Kind: res.Kind})
			res.Resource = gvr.Resource
		}

		err := (&scaffold.Scaffold{}).Execute(input.Options{},
			&webhook.AdmissionHandler{Resource: res, Config: webhook.Config{Server: server, Type: webhookType, Operations: operations}},
			&webhook.AdmissionWebhookBuilder{Resource: res, Config: webhook.Config{Server: server, Type: webhookType, Operations: operations}},
			&webhook.AdmissionWebhooks{Resource: res, Config: webhook.Config{Server: server, Type: webhookType, Operations: operations}},
			&webhook.AddAdmissionWebhookBuilderHandler{Resource: res, Config: webhook.Config{Server: server, Type: webhookType, Operations: operations}},
			&webhook.Server{Resource: res, Config: webhook.Config{Server: server, Type: webhookType, Operations: operations}},
			&webhook.AddServer{Resource: res, Config: webhook.Config{Server: server, Type: webhookType, Operations: operations}},
		)
		if err != nil {
			log.Fatal(err)
		}

		if doMake {
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

func init() {
	rootCmd.AddCommand(WebhookCmd)
	WebhookCmd.Flags().StringVar(&server, "server", "default",
		"name of the server")
	WebhookCmd.Flags().StringVar(&webhookType, "type", "",
		"webhook type, e.g. mutating or validating")
	WebhookCmd.Flags().StringSliceVar(&operations, "operations", []string{"create"},
		"the operations that the webhook will intercept, e.g. create, update, delete and connect")
	WebhookCmd.Flags().BoolVar(&doMake, "make", true,
		"if true, run make after generating files")
	res = gvkForFlags(WebhookCmd.Flags())
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
