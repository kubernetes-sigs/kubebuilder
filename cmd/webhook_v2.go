/*
Copyright 2019 The Kubernetes Authors.

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
	"path/filepath"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/project"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/resource"
	resourcev2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v2/webhook"
)

func newWebhookV2Cmd() *cobra.Command {
	o := webhookV2Options{}

	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Scaffold a webhook for an API resource.",
		Long:  `Scaffold a webhook for an API resource. You can choose to scaffold defaulting, validating and (or) conversion webhooks.`,
		Example: `	# Create defaulting and validating webhooks for CRD of group crew, version v1 and kind FirstMate.
	kubebuilder create webhook --group crew --version v1 --kind FirstMate --defaulting --programmatic-validation

	# Create conversion webhook for CRD of group crew, version v1 and kind FirstMate.
	kubebuilder create webhook --group crew --version v1 --kind FirstMate --conversion
`,
		Run: func(cmd *cobra.Command, args []string) {
			dieIfNoProject()

			projectInfo, err := scaffold.LoadProjectFile("PROJECT")
			if err != nil {
				log.Fatalf("failed to read the PROJECT file: %v", err)
			}

			if projectInfo.Version != project.Version2 {
				fmt.Printf("kubebuilder webhook is for project version: 2, the version of this project is: %s \n", projectInfo.Version)
				os.Exit(1)
			}

			if !o.defaulting && !o.validation && !o.conversion {
				fmt.Printf("kubebuilder webhook requires at least one of --defaulting, --programmatic-validation and --conversion to be true")
				os.Exit(1)
			}

			if len(o.res.Resource) == 0 {
				o.res.Resource = flect.Pluralize(strings.ToLower(o.res.Kind))
			}

			fmt.Println("Writing scaffold for you to edit...")
			fmt.Println(filepath.Join("api", o.res.Version,
				fmt.Sprintf("%s_webhook.go", strings.ToLower(o.res.Kind))))
			if o.conversion {
				fmt.Println(`Webhook server has been set up for you.
You need to implement the conversion.Hub and conversion.Convertible interfaces for your CRD types.`)
			}
			webhookScaffolder := &webhook.Webhook{
				Resource:   o.res,
				Defaulting: o.defaulting,
				Validating: o.validation,
			}
			err = (&scaffold.Scaffold{}).Execute(
				input.Options{},
				webhookScaffolder,
			)
			if err != nil {
				fmt.Printf("error scaffolding webhook: %v", err)
				os.Exit(1)
			}

			err = (&resourcev2.Main{}).Update(
				&resourcev2.MainUpdateOptions{
					Project:        &projectInfo,
					WireResource:   false,
					WireController: false,
					WireWebhook:    true,
					Resource:       o.res,
				})
			if err != nil {
				fmt.Printf("error updating main.go: %v", err)
				os.Exit(1)
			}

		},
	}
	o.res = gvkForFlags(cmd.Flags())
	cmd.Flags().BoolVar(&o.defaulting, "defaulting", false,
		"if set, scaffold the defaulting webhook")
	cmd.Flags().BoolVar(&o.validation, "programmatic-validation", false,
		"if set, scaffold the validating webhook")
	cmd.Flags().BoolVar(&o.validation, "conversion", false,
		"if set, scaffold the conversion webhook")

	return cmd
}

// webhookOptions represents commandline options for scaffolding a webhook.
type webhookV2Options struct {
	res        *resource.Resource
	defaulting bool
	validation bool
	conversion bool
}
