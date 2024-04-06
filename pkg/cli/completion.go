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

	"github.com/spf13/cobra"
)

func (c CLI) newBashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Load bash completions",
		Example: fmt.Sprintf(`# To load completion for this session, execute:
$ source <(%[1]s completion bash)

# To load completions for each session, execute once:
Linux:
  $ %[1]s completion bash > /etc/bash_completion.d/%[1]s
MacOS:
  $ %[1]s completion bash > /usr/local/etc/bash_completion.d/%[1]s
`, c.commandName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenBashCompletion(os.Stdout)
		},
	}
}

func (c CLI) newZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Load zsh completions",
		Example: fmt.Sprintf(`# If shell completion is not already enabled in your environment you will need
# to enable it. You can execute the following once:
$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ %[1]s completion zsh > "${fpath[1]}/_%[1]s"

# You will need to start a new shell for this setup to take effect.
`, c.commandName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
	}
}

func (c CLI) newFishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fish",
		Short: "Load fish completions",
		Example: fmt.Sprintf(`# To load completion for this session, execute:
$ %[1]s completion fish | source

# To load completions for each session, execute once:
$ %[1]s completion fish > ~/.config/fish/completions/%[1]s.fish
`, c.commandName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		},
	}
}

func (CLI) newPowerShellCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "powershell",
		Short: "Load powershell completions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenPowerShellCompletion(os.Stdout)
		},
	}
}

func (c CLI) newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Load completions for the specified shell",
		Long: fmt.Sprintf(`Output shell completion code for the specified shell.
The shell code must be evaluated to provide interactive completion of %[1]s commands.
Detailed instructions on how to do this for each shell are provided in their own commands.
`, c.commandName),
	}
	cmd.AddCommand(c.newBashCmd())
	cmd.AddCommand(c.newZshCmd())
	cmd.AddCommand(c.newFishCmd())
	cmd.AddCommand(c.newPowerShellCmd())
	return cmd
}
