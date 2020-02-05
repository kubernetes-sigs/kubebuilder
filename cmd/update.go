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

type updateError struct {
	err error
}

func (e updateError) Error() string {
	return fmt.Sprintf("failed to update vendor dependencies: %v", e.err)
}

func newUpdateCmd() *cobra.Command {
	options := &updateOptions{}

	return &cobra.Command{
		Use:   "update",
		Short: "Update vendor dependencies",
		Long:  `Update vendor dependencies`,
		Example: fmt.Sprintf(`	# Update the vendor dependencies:
	kubebuiler update vendor`),
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmdutil.Run(options); err != nil {
				log.Fatal(updateError{err})
			}
		},
	}
}

var _ cmdutil.RunOptions = &updateOptions{}

type updateOptions struct{}

func (o *updateOptions) LoadConfig() (*config.Config, error) {
	return config.LoadInitialized()
}

func (o *updateOptions) Validate(c *config.Config) error {
	if !c.IsV1() {
		return fmt.Errorf("vendor was only used in project version 1, this project is %s", c.Version)
	}

	return nil
}

func (o *updateOptions) GetScaffolder(c *config.Config) (scaffold.Scaffolder, error) { // nolint:unparam
	return scaffold.NewUpdateScaffolder(&c.Config), nil
}

func (o *updateOptions) PostScaffold(_ *config.Config) error {
	return nil
}
