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

package cli

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

func (c *cli) newInitCmd() *cobra.Command {
	ctx := c.newInitContext()
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialize a new project",
		Long:    ctx.Description,
		Example: ctx.Examples,
		Run:     func(cmd *cobra.Command, args []string) {},
	}

	// Register --project-version on the dynamically created command
	// so that it shows up in help and does not cause a parse error.
	cmd.Flags().String("project-version", c.defaultProjectVersion,
		fmt.Sprintf("project version, possible values: (%s)", strings.Join(c.getAvailableProjectVersions(), ", ")))

	// Pre-parse the project version and help flags so that we can
	// dynamically bind to a plugin's init implementation (or not).
	var isHelpOnly bool
	c.projectVersion, isHelpOnly = c.getBaseFlags()

	// If only the help flag was set, return the command as is.
	if isHelpOnly {
		return cmd
	}

	// Lookup the plugin for projectVersion and bind it to the command.
	c.bindInit(ctx, cmd)
	return cmd
}

func (c cli) newInitContext() plugin.Context {
	return plugin.Context{
		CommandName: c.commandName,
		Description: `Initialize a new project.

For further help about a specific project version, set --project-version.
`,
		Examples: c.getInitHelpExamples(),
	}
}

func (c cli) getInitHelpExamples() string {
	var sb strings.Builder
	for _, version := range c.getAvailableProjectVersions() {
		rendered := fmt.Sprintf(`  # Help for initializing a project with version %s
  %s init --project-version=%s -h

`,
			version, c.commandName, version)
		sb.WriteString(rendered)
	}
	return strings.TrimSuffix(sb.String(), "\n\n")
}

func (c cli) getAvailableProjectVersions() (projectVersions []string) {
	for version, versionedPlugins := range c.plugins {
		for _, p := range versionedPlugins {
			// There will only be one init plugin per version.
			if _, isInit := p.(plugin.Init); !isInit {
				// Only return project versions from non-deprecated plugins.
				if _, isDeprecated := p.(plugin.Deprecated); !isDeprecated {
					projectVersions = append(projectVersions, strconv.Quote(version))
					break
				}
			}
		}
	}
	return projectVersions
}

// getBaseFlags parses the command line arguments, looking for --project-version
// and help. If an error occurs or only --help is set, getBaseFlags returns an
// empty string and true. Otherwise, getBaseFlags returns the project version
// and false.
func (c cli) getBaseFlags() (string, bool) {
	fs := pflag.NewFlagSet("base", pflag.ExitOnError)
	fs.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}

	var (
		projectVersion string
		help           bool
	)
	fs.StringVar(&projectVersion, "project-version", c.defaultProjectVersion, "project version")
	fs.BoolVarP(&help, "help", "h", false, "print help")

	err := fs.Parse(os.Args[1:])
	doHelp := err != nil || help && !fs.Lookup("project-version").Changed
	if doHelp {
		return "", true
	}
	return projectVersion, false
}

func (c cli) bindInit(ctx plugin.Context, cmd *cobra.Command) {
	versionedPlugins, err := c.getVersionedPlugins()
	if err != nil {
		log.Fatal(err)
	}
	var getter plugin.InitPluginGetter
	var hasGetter bool
	for _, p := range versionedPlugins {
		tmpGetter, isGetter := p.(plugin.InitPluginGetter)
		if isGetter {
			if hasGetter {
				log.Fatalf("duplicate initialization plugins for project version %q: %s, %s",
					c.projectVersion, getter.Name(), p.Name())
			}
			hasGetter = true
			getter = tmpGetter
		}
	}
	if !hasGetter {
		log.Fatalf("project version %q does not support a project initialization plugin",
			c.projectVersion)
	}

	init := getter.GetInitPlugin()
	init.BindFlags(cmd.Flags())
	init.SetVersion(c.projectVersion)
	init.UpdateContext(&ctx)
	cmd.Long = ctx.Description
	cmd.Example = ctx.Examples
	cmd.RunE = runECmdFunc(init, fmt.Sprintf("failed to initialize project with version %q", c.projectVersion))
}
