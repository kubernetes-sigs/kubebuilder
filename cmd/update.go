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
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v2/internal/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/scaffold"
)

type updateError struct {
	err error
}

func (e updateError) Error() string {
	return fmt.Sprintf("failed to update vendor dependencies: %v", e.err)
}

func newUpdateCmd() *cobra.Command {
	options := &updateOptions{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update vendor dependencies",
		Long:  `Update vendor dependencies`,
		Example: `Update the vendor dependencies:
kubebuilder update
`,
		Run: func(_ *cobra.Command, _ []string) {
			if err := run(options); err != nil {
				log.Fatal(updateError{err})
			}
		},
	}

	options.bindFlags(cmd)

	return cmd
}

var _ commandOptions = &updateOptions{}

type updateOptions struct{}

func (o *updateOptions) bindFlags(_ *cobra.Command) {}

func (o *updateOptions) loadConfig() (*config.Config, error) {
	projectConfig, err := config.Load()
	if os.IsNotExist(err) {
		return nil, errors.New("unable to find configuration file, project must be initialized")
	}

	return projectConfig, err
}

func (o *updateOptions) validate(c *config.Config) error {
	if !c.IsV1() {
		return fmt.Errorf("vendor was only used for v1, this project is %s", c.Version)
	}

	return nil
}

func (o *updateOptions) scaffolder(c *config.Config) (scaffold.Scaffolder, error) { //nolint:unparam
	return scaffold.NewUpdateScaffolder(&c.Config), nil
}

func (o *updateOptions) postScaffold(_ *config.Config) error {
	return nil
}
