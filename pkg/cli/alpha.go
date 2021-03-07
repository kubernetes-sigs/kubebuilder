/*
Copyright 2021 The Kubernetes Authors.

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
	"strings"

	"github.com/spf13/cobra"
	configgen "sigs.k8s.io/kubebuilder/v3/pkg/cli/alpha/config-gen"
)

var alphaCommands = []*cobra.Command{
	configgen.NewCommand(),
}

func (c *CLI) newAlphaCmd() *cobra.Command {
	alpha := &cobra.Command{
		Use:        "alpha",
		SuggestFor: []string{"experimental"},
		Short:      "Alpha kubebuilder subcommands",
		Long: strings.TrimSpace(`
Alpha kubebuilder commands are for unstable features.

- Alpha commands are exploratory and may be removed without warning.
- No backwards compatibility is provided for any alpha commands.
 		`),
	}
	for i := range alphaCommands {
		alpha.AddCommand(alphaCommands[i])
	}
	return alpha
}

func (c *CLI) addAlphaCmd() {
	if len(alphaCommands) > 0 {
		c.cmd.AddCommand(c.newAlphaCmd())
	}
}
