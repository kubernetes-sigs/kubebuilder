package v1alpha1

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/multi-module/v1alpha1/scaffolds"
)

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config config.Config

	multimodule bool
}

func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = subcmdMeta.Description + `
  - Toggle between single or multi module projects.
`
	subcmdMeta.Examples = fmt.Sprintf(subcmdMeta.Examples+`
  # Enable the multimodule layout
  %[1]s edit --multimodule

  # Disable the multimodule layout
  %[1]s edit --multimodule=false
`, cliMeta.CommandName)
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.multimodule, "multimodule", false, "enable or disable multimodule layout")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *editSubcommand) PreScaffold(fs machinery.Filesystem) error {
	// Track the config and ensure it exists and can be parsed
	cfg := pluginConfig{}
	if err := p.config.DecodePluginConfig(pluginKey, &cfg); errors.As(err, &config.UnsupportedFieldError{}) {
		// Config doesn't support per-plugin configuration, so we can't track them
	} else {
		// Fail unless they key wasn't found, which just means it is the first resource tracked
		if err != nil && !errors.As(err, &config.PluginKeyNotFoundError{}) {
			return err
		}

		if err := p.config.EncodePluginConfig(pluginKey, cfg); err != nil {
			return err
		}
	}
	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	if p.multimodule {
		fmt.Println("updating scaffold with multi-module support...")
		res, err := p.config.GetResources()
		if err != nil {
			return err
		}
		for i := range res {
			resource := &res[i]
			fmt.Println("using gvk:", resource.Group, resource.Domain, resource.Version)
			apiPath := GetAPIPath(p.config.IsMultiGroup(), resource)
			goModPath := filepath.Join(apiPath, "go.mod")
			fmt.Println("using go.mod path: " + goModPath)

			scaffolder := scaffolds.NewAPIScaffolder(p.config, *resource, goModPath)
			scaffolder.InjectFS(fs)
			err := scaffolder.Scaffold()
			if err != nil {
				return err
			}

			if err := util.RunInDir(apiPath, func() error {
				err = util.RunCmd("Update dependencies in "+apiPath, "go", "mod", "tidy")
				if err != nil {
					return err
				}
				return nil
			}); err != nil {
				return err
			}

			if err := util.RunCmd("Add require directive of API module", "go", "mod", "edit", "-require",
				resource.Path+"@v0.0.0-v1alpha1"); err != nil {
				return err
			}

			if err := util.RunCmd("Update dependencies", "go", "mod", "edit", "-replace",
				resource.Path+"="+"."+string(filepath.Separator)+apiPath); err != nil {
				return err
			}
			// Update Dockerfile
			err = insertModUpdatesInDockerfile(apiPath)
			if err != nil {
				return err
			}
		}
	} else {
		fmt.Println("disabling multi-module support...")
		res, err := p.config.GetResources()
		if err != nil {
			return err
		}

		for i := range res {
			resource := &res[i]
			fmt.Println("using gvk:", resource.Group, resource.Domain, resource.Version)
			apiPath := GetAPIPath(p.config.IsMultiGroup(), resource)
			goModPath := filepath.Join(apiPath, "go.mod")
			fmt.Println("using go.mod path: " + goModPath)
			goSumPath := filepath.Join(apiPath, "go.sum")
			fmt.Println("using go.sum path: " + goSumPath)

			if err := os.Remove(goModPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			if err := os.Remove(goSumPath); err != nil && !os.IsNotExist(err) {
				return err
			}

			if err := util.RunCmd("Remove require directive of API module", "go", "mod", "edit", "-droprequire",
				resource.Path); err != nil {
				return err
			}

			if err := util.RunCmd("Update dependencies", "go", "mod", "edit", "-dropreplace",
				resource.Path); err != nil {
				return err
			}
			// Update Dockerfile
			if err := removeModUpdatesInDockerfile(apiPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *editSubcommand) PostScaffold() error {
	err := util.RunCmd("Update dependencies", "go", "mod", "tidy")
	if err != nil {
		return err
	}

	return nil
}
