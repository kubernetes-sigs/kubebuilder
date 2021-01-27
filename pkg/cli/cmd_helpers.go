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
	"os"

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

// preRunECmdFunc returns a cobra PreRunE function that loads the configuration file
// and injects it into the subcommand
func preRunECmdFunc(subcmd plugin.Subcommand, cfg *config.Config, msg string) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		err := cfg.Load()
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: unable to find configuration file, project must be initialized", msg)
		} else if err != nil {
			return fmt.Errorf("%s: unable to load configuration file: %w", msg, err)
		}

		subcmd.InjectConfig(cfg.Config)
		return nil
	}
}

// runECmdFunc returns a cobra RunE function that runs subcommand
func runECmdFunc(fs afero.Fs, subcommand plugin.Subcommand, msg string) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		if err := subcommand.Run(fs); err != nil {
			return fmt.Errorf("%s: %v", msg, err)
		}
		return nil
	}
}

// postRunECmdFunc returns a cobra PostRunE function that saves the configuration file
func postRunECmdFunc(cfg *config.Config, msg string) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		err := cfg.Save()
		if err != nil {
			return fmt.Errorf("%s: unable to save configuration file: %w", msg, err)
		}
		return nil
	}
}
