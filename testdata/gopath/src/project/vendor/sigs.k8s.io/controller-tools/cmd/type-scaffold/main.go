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
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-tools/pkg/typescaffold"
)

func main() {
	opts := &typescaffold.ScaffoldOptions{
		Resource: typescaffold.Resource{
			Namespaced: true,
		},
	}
	scaffoldCmd := &cobra.Command{
		Use:   "type-scaffold",
		Short: "Quickly scaffold out basic bits of a Kubernetes type.",
		Long: `Quickly scaffold out the structure of a type for a Kubernetes kind and associated types.
Produces:

- a root type with approparite metadata fields
- Spec and Status types
- a list type

Also applies the appropriate comments to generate the code required to conform to runtime.Object.`,
		Example: `	# Generate types for a Kind called Foo with a resource called foos
		type-scaffold --kind Foo

	# Generate types for a Kind called Bar with a resource of foobars
	type-scaffold --kind Bar --resource foobars`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			return opts.Scaffold(os.Stdout)
		},
	}

	scaffoldCmd.Flags().StringVar(&opts.Resource.Kind, "kind", opts.Resource.Kind, "The kind of the typescaffold being scaffolded.")
	scaffoldCmd.Flags().StringVar(&opts.Resource.Resource, "resource", opts.Resource.Resource, "The resource of the typescaffold being scaffolded (defaults to a lower-case, plural version of kind).")
	scaffoldCmd.Flags().BoolVar(&opts.Resource.Namespaced, "namespaced", opts.Resource.Namespaced, "Whether or not the given resource is namespaced.")

	if err := cobra.MarkFlagRequired(scaffoldCmd.Flags(), "kind"); err != nil {
		panic("unable to mark --kind as required")
	}

	if err := scaffoldCmd.Execute(); err != nil {
		if _, err := fmt.Fprintln(os.Stderr, err); err != nil {
			// this would be exceedingly bizarre if we ever got here
			panic("unable to write to error details to standard error")
		}
		os.Exit(1)
	}
}
