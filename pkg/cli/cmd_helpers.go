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

package cli

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v3/pkg/cli/internal/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

// cmdErr updates a cobra command to output error information when executed
// or used with the help flag.
func cmdErr(cmd *cobra.Command, err error) {
	cmd.Long = fmt.Sprintf("%s\nNote: %v", cmd.Long, err)
	cmd.RunE = errCmdFunc(err)
}

// cmdErrNoHelp calls cmdErr(cmd, err) then turns cmd's usage off.
func cmdErrNoHelp(cmd *cobra.Command, err error) {
	cmdErr(cmd, err)
	cmd.SilenceUsage = true
}

// errCmdFunc returns a cobra RunE function that returns the provided error
func errCmdFunc(err error) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		return err
	}
}

// runECmdFunc returns a cobra RunE function that runs subcommand and saves the
// config, which may have been modified by subcommand.
func runECmdFunc(
	fs afero.Fs,
	c *config.Config,
	subcommand plugin.Subcommand,
	msg string,
) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		if err := subcommand.Run(fs); err != nil {
			return fmt.Errorf("%s: %v", msg, err)
		}
		return c.Save()
	}
}
