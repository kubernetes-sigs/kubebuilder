/*
Copyright 2019 Joe Lanford.

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

	"github.com/joelanford/go-apidiff/pkg/diff"
)

func main() {
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func newRootCmd() *cobra.Command {
	opts := diff.Options{}
	var printCompatible bool

	checkArgs := func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 || len(args) > 2 {
			return fmt.Errorf("accepts 1 or 2 args, received %d", len(args))
		}
		if args[0] == "" {
			return fmt.Errorf("oldCommit should not be empty")
		}
		if len(args) == 2 && args[1] == "" {
			return fmt.Errorf("if provided, newCommit should not be empty")
		}
		return nil
	}

	cmd := &cobra.Command{
		Use:   "go-apidiff <oldCommit> [newCommit]",
		Short: "Compare API compatibility of a go module",
		Long: `go-apidiff compares API compatibility of different commits of a Go repository.

By default, it compares just the module itself and prints only incompatible
changes. However it can be configured to print compatible changes and to search
for API incompatibilities in the dependency changes of the repository.

When used with just one argument, the passed argument is used for oldCommit,
and HEAD is used for newCommit."`,
		Args: checkArgs,
		Run: func(cmd *cobra.Command, args []string) {
			opts.OldCommit = args[0]
			opts.NewCommit = "HEAD"
			if len(args) == 2 {
				opts.NewCommit = args[1]
			}

			diff, err := diff.Run(opts)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}

			diff.PrintReports(printCompatible)

			if !diff.IsCompatible() {
				os.Exit(1)
			}

			os.Exit(0)
		},
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo-path", cwd, "Path to root of git repository to compare")
	cmd.Flags().BoolVar(&opts.CompareImports, "compare-imports", false, "Compare exported API differences of the imports in the repo. ")
	cmd.Flags().BoolVar(&printCompatible, "print-compatible", false, "Print compatible API changes")

	return cmd
}
