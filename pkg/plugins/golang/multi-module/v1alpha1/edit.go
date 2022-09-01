package v1alpha1

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config config.Config

	multimodule     bool
	canUseAPIModule bool
	pluginConfig
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

	// Track the config and ensure it exists and can be parsed
	cfg := pluginConfig{}
	if err := p.config.DecodePluginConfig(pluginKey, &cfg); errors.As(err, &config.UnsupportedFieldError{}) {
		// Config doesn't support per-plugin configuration, so we can't track them
	} else {
		// Fail unless they key wasn't found, which just means it is the first resource tracked
		if err != nil && !errors.As(err, &config.PluginKeyNotFoundError{}) {
			return err
		}
	}
	p.pluginConfig = cfg

	if res, err := p.config.GetResources(); err != nil {
		return err
	} else if len(res) == 0 {
		p.canUseAPIModule = false
	} else {
		foundAtLeastOneAPI := false
		for i := range res {
			if res[i].HasAPI() {
				foundAtLeastOneAPI = true
				break
			}
		}
		p.canUseAPIModule = foundAtLeastOneAPI
	}

	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	if !p.canUseAPIModule {
		return nil
	}

	if p.multimodule {
		if p.pluginConfig.ApiGoModCreated {
			return nil
		}

		if err := CreateGoModForAPI(fs, p.config); err != nil {
			return err
		}

		if err := TidyGoModForAPI(p.config.IsMultiGroup()); err != nil {
			return err
		}

		p.pluginConfig.ApiGoModCreated = true
	} else {
		if !p.pluginConfig.ApiGoModCreated {
			return nil
		}

		if err := CleanUpGoModForAPI(fs, p.config); err != nil {
			return err
		}

		p.pluginConfig.ApiGoModCreated = false
	}

	return p.config.EncodePluginConfig(pluginKey, p.pluginConfig)
}
