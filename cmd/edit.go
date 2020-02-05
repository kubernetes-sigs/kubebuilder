/*
Copyright 2017 The Kubernetes Authors.

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

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
)

type editError struct {
	err error
}

func (e editError) Error() string {
	return fmt.Sprintf("failed to edit configuration: %v", e.err)
}

func newEditCmd() *cobra.Command {
	options := &editOptions{}

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "This command will edit the project configuration",
		Long:  `This command will edit the project configuration`,
		Example: `	# Enable the multigroup layout
	kubebuilder edit --multigroup

	# Disable the multigroup layout
	kubebuilder edit --multigroup=false`,
		Run: func(_ *cobra.Command, _ []string) {
			if err := cmdutil.Run(options); err != nil {
				log.Fatal(editError{err})
			}
		},
	}

	options.bindFlags(cmd)

	return cmd
}

var _ cmdutil.RunOptions = &editOptions{}

type editOptions struct {
	multigroup bool
}

func (o *editOptions) bindFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.multigroup, "multigroup", false, "enable or disable multigroup layout")
}

func (o *editOptions) LoadConfig() (*config.Config, error) {
	return config.LoadInitialized()
}

func (o *editOptions) Validate(c *config.Config) error {
	if !c.IsV2() {
		if c.MultiGroup {
			return fmt.Errorf("multiple group support can't be enabled for version %s", c.Version)
		}
	}

	return nil
}

func (o *editOptions) GetScaffolder(c *config.Config) (scaffold.Scaffolder, error) { // nolint:unparam
	return scaffold.NewEditScaffolder(c, o.multigroup), nil
}

func (o *editOptions) PostScaffold(_ *config.Config) error {
	return nil
}
