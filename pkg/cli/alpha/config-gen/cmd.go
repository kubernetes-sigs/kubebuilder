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

package configgen

import (
	// required to make sure the controller-tools is initialized fully
	_ "sigs.k8s.io/controller-runtime/pkg/scheme"

	"embed"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/parser"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TemplateFS contains the templates used by config-gen
//go:embed templates/resources/* templates/patches/*
var TemplateFS embed.FS

func buildProcessor(value *KubebuilderConfigGen) framework.ResourceListProcessor {
	return framework.TemplateProcessor{
		MergeResources: true,

		PreProcessFilters: []kio.Filter{
			// run controller-gen libraries to generate configuration from code
			ControllerGenFilter{KubebuilderConfigGen: value},
			// inject generated certificates
			CertFilter{KubebuilderConfigGen: value},
		},

		ResourceTemplates: []framework.ResourceTemplate{{
			Templates: parser.TemplateFiles(filepath.Join("templates", "resources")).FromFS(TemplateFS),
		}},
		PatchTemplates: []framework.PatchTemplate{
			&framework.ResourcePatchTemplate{
				Selector: &framework.Selector{
					Kinds: []string{"CustomResourceDefinition"},
					ResourceMatcher: func(m *yaml.RNode) bool {
						meta, _ := m.GetMeta()
						return value.Spec.Webhooks.Conversions[meta.Name]
					},
				},
				Templates: parser.TemplateFiles(filepath.Join("templates", "patches", "crd")).FromFS(TemplateFS),
			},
			&framework.ResourcePatchTemplate{
				Selector: &framework.Selector{
					TemplateData: value,
					Kinds:        []string{"Deployment"},
					Names:        []string{"controller-manager"},
					Namespaces:   []string{"{{ .Namespace }}"},
					Labels:       map[string]string{"control-plane": "controller-manager"},
				},
				Templates: parser.TemplateFiles(filepath.Join("templates", "patches", "controller-manager")).FromFS(TemplateFS),
			},
			&framework.ResourcePatchTemplate{
				Selector: &framework.Selector{
					Kinds: []string{
						"CustomResourceDefinition",
						"ValidatingWebhookConfiguration",
						"MutatingWebhookConfiguration",
					},
				},
				Templates: parser.TemplateFiles(filepath.Join("templates", "patches", "cert-manager")).FromFS(TemplateFS),
			},
		},

		PostProcessFilters: []kio.Filter{
			ComponentFilter{KubebuilderConfigGen: value},
			SortFilter{KubebuilderConfigGen: value},
		},
		TemplateData: value,
	}
}

func buildCmd() *cobra.Command {
	kp := &KubebuilderConfigGen{}

	// legacy kustomize function support
	legacyPlugin := os.Getenv("KUSTOMIZE_PLUGIN_CONFIG_STRING")
	err := yaml.Unmarshal([]byte(legacyPlugin), kp)
	if err != nil {
		log.Fatal(err)
	}

	cmd := command.Build(buildProcessor(kp), command.StandaloneEnabled, false)
	return cmd
}

// NewCommand returns a new cobra command
func NewCommand() *cobra.Command {
	c := buildCmd()

	if os.Getenv("KUSTOMIZE_FUNCTION") == "true" {
		// run as part of kustomize -- read from stdin
		c.Args = cobra.MinimumNArgs(0)
	} else {
		c.Args = cobra.MinimumNArgs(1)
	}
	c.RemoveCommand(c.Commands()...)
	c.Use = "config-gen PROJECT_FILE [RESOURCE_PATCHES...]"
	c.Version = `v0.1.0`
	c.Short = `Generate configuration for controller-runtime based projects`
	c.Long = strings.TrimSpace(`
config-gen programatically generates configuration for a controller-runtime based
project using the project source code (golang) and a KubebuilderConfigGen resource file.

This is an alternative to expressing configuration as a static set of kustomize patches
in the "config" directory.

config-gen may be used as a standalone command run against a file, as a kustomize
transformer plugin, or as a configuration function (e.g. kpt).

config-gen uses the controller-tools generators to generate CRDs from the go source
and then generates additional resources such as the namespace, controller-manager,
webhooks, etc.

Following is an example KubebuilderConfigGen resource used by config-gen:

  # kubebuilderconfiggen.yaml
  # this resource describes how to generate configuration for a controller-runtime
  # based project
  apiVersion: kubebuilder.sigs.k8s.io/v1alpha1
  kind: KubebuilderConfigGen
  metadata:
    name: my-project-name
  spec:
    controllerManager:
      image: my-org-name/my-project-name:v0.1.0

If this file was at the project source root, config-gen could be used to emit
configuration using:

  kubebuilder alpha config-gen ./kubebuilderconfiggen.yaml

The KubebuilderConfigGen resource has the following fields:

  apiVersion: kubebuilder.sigs.k8s.io/v1alpha1
  kind: KubebuilderConfigGen

  metadata:
    # name of the project.  used in various resource names.
    # required
    name: project-name

    # namespace for the project
    # optional -- defaults to "${metadata.name}-system"
    namespace: project-namespace

  spec:
    # configure how CRDs are generated
    crds:
      # path to go module source directory provided to controller-gen libraries
      # optional -- defaults to '.'
      sourceDirectory: ./relative/path

    # configure how the controller-manager is generated
    controllerManager:
      # image to run
      image: my-org/my-project:v0.1.0

      # if set, use component config for the controller-manager
      # optional
      componentConfig:
        # use component config
        enable: true

        # path to component config to put into a ConfigMap
        configFilepath: ./path/to/componentconfig.yaml

      # configure how metrics are exposed
      metrics:
        # disable the auth proxy required for scraping metrics
        # disable: false

        # generate prometheus ServiceMonitor resource
        enableServiceMonitor: true

    # configure how webhooks are generated
    # optional -- defaults to not generating webhook configuration
    webhooks:
      # enable will cause webhook config to be generated
      enable: true

      # configures crds which use conversion webhooks
      enableConversion:
        # key is the name of the CRD
        "bars.example.my.domain": true

      # configures where to get the certificate used for webhooks
      # discriminated union
      certificateSource:
        # type of certificate source
        # one of ["certManager", "dev", "manual"] -- defaults to "manual"
        # certManager: certmanager is used to manage certificates -- requires CertManager to be installed
        # dev: certificate is generated and wired into resources
        # manual: no certificate is generated or wired into resources
        type: "dev"

        # options for a dev certificate -- requires "dev" as the type
        devCertificate:
          duration: 1h
`)
	c.Example = strings.TrimSpace(`
#
# As command
#
# create the kubebuilderconfiggen.yaml at project root
cat > kubebuilderconfiggen.yaml <<EOF
apiVersion: kubebuilder.sigs.k8s.io/v1alpha1
  kind: KubebuilderConfigGen
  metadata:
    name: project
  spec:
    controllerManager
      image: org/project:v0.1.0
EOF

# run the config generator
kubebuilder alpha config-gen kubebuilderconfiggen.yaml

# run the config generator and apply
kubebuilder alpha config-gen kubebuilderconfiggen.yaml | kubectl apply -f -

# generate configuration from a file with patches
kubebuilder alpha config-gen kubebuilderconfiggen.yaml patch1.yaml patch2.yaml

#
# As Kustomize plugin
# this allows using config-gen with kustomize features such as patches, commonLabels,
# commonAnnotations, resources, configMapGenerator and other transformer plugins.
#

# install the latest kustomize
GO111MODULE=on go get sigs.k8s.io/kustomize/kustomize/v4

# install the command as a kustomize plugin
kubebuilder alpha config-gen install-as-plugin

# create the kustomization.yaml containing the KubebuilderConfigGen resource
cat > kustomization.yaml <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
transformers:
- |-
  apiVersion: kubebuilder.sigs.k8s.io/v1alpha1
  kind: KubebuilderConfigGen
  metadata:
    name: my-project
  spec:
    controllerManager:
      image: my-org/my-project:v0.1.0
EOF

# generate configuration from kustomize > v4.0.0
kustomize build --enable-alpha-plugins .

# generate configuration from kustomize <= v4.0.0
kustomize build --enable_alpha_plugins .
`)

	// command for installing the plugin
	install := &cobra.Command{
		Use:   "install-as-plugin",
		Short: "Install config-gen as a kustomize plugin",
		Long: fmt.Sprintf(`Write a script to %s for kustomize to locate as a plugin.
This path will be written to $XDG_CONFIG_HOME if set, otherwise $HOME.
`, pluginScriptPath),
		Example: `
kubebuilder alpha config-gen install-as-plugin
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			hd, err := getPluginHomeDir()
			if err != nil {
				log.Fatal(err)
			}
			fullScriptPath := filepath.Join(hd, pluginScriptPath)

			// Given the script perms, this command will not be able to overwrite the plugin script file.
			// That's ok, let the user handle removal to maintain security.
			if info, err := os.Stat(fullScriptPath); err == nil && !info.IsDir() {
				fmt.Fprintf(cmd.OutOrStdout(), "kustomize plugin configured at %s\n", fullScriptPath)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "writing kustomize plugin file at %s\n", fullScriptPath)

			dir, _ := filepath.Split(fullScriptPath)
			if err = os.MkdirAll(dir, 0700); err != nil {
				return err
			}

			// r-x perms to prevent overwrite vulnerability since the script will be executed out-of-tree.
			return ioutil.WriteFile(fullScriptPath, []byte(pluginScript), 0500)
		},
	}
	c.AddCommand(install)

	return c
}

// Kustomize plugin execution script.
const pluginScript = `#!/bin/bash
KUSTOMIZE_FUNCTION=true kubebuilder alpha config-gen
`

// Qualified directory containing the config-gen plugin script. Child of plugin home dir.
var pluginScriptPath = filepath.Join("kustomize", "plugin",
	"kubebuilder.sigs.k8s.io", "v1alpha1", "kubebuilderconfiggen", "KubebuilderConfigGen")

// getPluginHomeDir returns $XDG_CONFIG_HOME if set, otherwise $HOME.
func getPluginHomeDir() (string, error) {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		dir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdg = filepath.Join(dir, ".config")
	}
	return xdg, nil
}
