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

package main

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	completionCmd := &cobra.Command{
		Use: "completion",
		Long: `Output shell completion code for the specified shell (bash or zsh). 
The shell code must be evaluated to provide interactive completion of kubebuilder commands.
This can be done by sourcing ~/.bash_profile or ~/.bashrc.
Detailed instructions on how to do this are available at docs/book/src/reference/completion.md
`,
		Example: `To load all completions run:
$ . <(kubebuilder completion)
To configure your shell to load completions for each session add to your .bashrc:
$ echo -e "\n. <(kubebuilder completion)" >> ~/.bashrc
`,
	}
	completionCmd.AddCommand(newZshCmd())
	completionCmd.AddCommand(newBashCmd())
	return completionCmd
}

func newBashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Generate bash completions",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return cmd.Root().GenBashCompletion(os.Stdout)
		},
		Example: `To load completion run:
$ . <(kubebuilder completion bash)
To configure your bash shell to load completions for each session add to your bashrc:
$ echo -e "\n. <(kubebuilder completion bash)" >> ~/.bashrc
`,
	}
}

func newZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completions",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
		Example: `To load completion run:
$ . <(kubebuilder completion zsh)
To configure your zsh shell to load completions for each session add to your bashrc:
$ echo -e "\n. <(kubebuilder completion zsh)" >> ~/.bashrc
`,
	}
}
