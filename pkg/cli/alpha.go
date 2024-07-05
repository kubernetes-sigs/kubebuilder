/*
Copyright 2022 The Kubernetes Authors.

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
	"strings"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha"
)

const (
	alphaCommand = "alpha"
)

var alphaCommands = []*cobra.Command{
	newAlphaCommand(),
	alpha.NewScaffoldCommand(),
}

func newAlphaCommand() *cobra.Command {
	cmd := &cobra.Command{
		// TODO: If we need to create alpha commands please add a new file for each command
	}
	return cmd
}

func (c *CLI) newAlphaCmd() *cobra.Command {
	alpha := &cobra.Command{
		Use:        alphaCommand,
		SuggestFor: []string{"experimental"},
		Short:      "Alpha-stage subcommands",
		Long: strings.TrimSpace(`
Alpha subcommands are for unstable features.

- Alpha subcommands are exploratory and may be removed without warning.
- No backwards compatibility is provided for any alpha subcommands.
`),
	}
	// TODO: Add alpha commands here if we need to have them
	for i := range alphaCommands {
		alpha.AddCommand(alphaCommands[i])
	}
	return alpha
}

func (c *CLI) addAlphaCmd() {
	if (len(alphaCommands) + len(c.extraAlphaCommands)) > 0 {
		c.cmd.AddCommand(c.newAlphaCmd())
	}
}

func (c *CLI) addExtraAlphaCommands() error {
	// Search for the alpha subcommand
	var alpha *cobra.Command
	for _, subCmd := range c.cmd.Commands() {
		if subCmd.Name() == alphaCommand {
			alpha = subCmd
			break
		}
	}
	if alpha == nil {
		return fmt.Errorf("no %q command found", alphaCommand)
	}

	for _, cmd := range c.extraAlphaCommands {
		for _, subCmd := range alpha.Commands() {
			if cmd.Name() == subCmd.Name() {
				return fmt.Errorf("command %q already exists", fmt.Sprintf("%s %s", alphaCommand, cmd.Name()))
			}
		}
		alpha.AddCommand(cmd)
	}
	return nil
}
