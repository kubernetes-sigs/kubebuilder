/*
Copyright 2025 The Kubernetes Authors.

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

package templates

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &HelmValuesBasic{}

// HelmValuesBasic scaffolds a basic values.yaml based on detected features
type HelmValuesBasic struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// DeploymentConfig stores extracted deployment configuration (env, resources, security contexts)
	DeploymentConfig map[string]any
	// OutputDir specifies the output directory for the chart
	OutputDir string
	// Force if true allows overwriting the scaffolded file
	Force bool
	// HasWebhooks is true when webhooks were found in the config
	HasWebhooks bool
	// HasMetrics is true when metrics service/monitor were found in the config
	HasMetrics bool
}

// SetTemplateDefaults implements machinery.Template
func (f *HelmValuesBasic) SetTemplateDefaults() error {
	if f.Path == "" {
		outputDir := f.OutputDir
		if outputDir == "" {
			outputDir = "dist"
		}
		f.Path = filepath.Join(outputDir, "chart", "values.yaml")
	}

	f.TemplateBody = f.generateBasicValues()

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

// generateBasicValues creates a basic values.yaml based on detected features
func (f *HelmValuesBasic) generateBasicValues() string {
	var buf bytes.Buffer

	// Controller Manager configuration
	imageRepo := "controller"
	imageTag := "latest"
	imagePullPolicy := "IfNotPresent"
	if f.DeploymentConfig != nil {
		if imgCfg, ok := f.DeploymentConfig["image"].(map[string]any); ok {
			if repo, ok := imgCfg["repository"].(string); ok && repo != "" {
				imageRepo = repo
			}
			if tag, ok := imgCfg["tag"].(string); ok && tag != "" {
				imageTag = tag
			}
			if policy, ok := imgCfg["pullPolicy"].(string); ok && policy != "" {
				imagePullPolicy = policy
			}
		}
	}

	buf.WriteString(fmt.Sprintf(`## String to partially override chart.fullname template (will maintain the release name)
##
# nameOverride: ""

## String to fully override chart.fullname template
##
# fullnameOverride: ""

## Configure the controller manager deployment
##
manager:
  replicas: 1

  image:
    repository: %s
    tag: %s
    pullPolicy: %s

`, imageRepo, imageTag, imagePullPolicy))

	// Add extracted deployment configuration
	f.addDeploymentConfig(&buf)

	// RBAC configuration
	buf.WriteString(`## Helper RBAC roles for managing custom resources
##
rbacHelpers:
  # Install convenience admin/editor/viewer roles for CRDs
  enable: false

`)

	// CRD configuration
	buf.WriteString(`## Custom Resource Definitions
##
crd:
  # Install CRDs with the chart
  enable: true
  # Keep CRDs when uninstalling
  keep: true

`)

	// Metrics configuration (enable if metrics artifacts detected in kustomize output)
	metricsPort := 8443
	if f.DeploymentConfig != nil {
		if mp, ok := f.DeploymentConfig["metricsPort"].(int); ok && mp > 0 {
			metricsPort = mp
		}
	}

	if f.HasMetrics {
		buf.WriteString(fmt.Sprintf(`## Controller metrics endpoint.
## Enable to expose /metrics endpoint with RBAC protection.
##
metrics:
  enable: true
  # Metrics server port
  port: %d

`, metricsPort))
	} else {
		buf.WriteString(fmt.Sprintf(`## Controller metrics endpoint.
## Enable to expose /metrics endpoint with RBAC protection.
##
metrics:
  enable: false
  # Metrics server port
  port: %d

`, metricsPort))
	}

	// Cert-manager configuration (always present, enabled based on webhooks)
	if f.HasWebhooks {
		buf.WriteString(`## Cert-manager integration for TLS certificates.
## Required for webhook certificates and metrics endpoint certificates.
##
certManager:
  enable: true

`)
	} else {
		buf.WriteString(`## Cert-manager integration for TLS certificates.
## Required for webhook certificates and metrics endpoint certificates.
##
certManager:
  enable: false

`)
	}

	// Webhook configuration - only if webhooks are present
	if f.HasWebhooks {
		webhookPort := 9443
		if f.DeploymentConfig != nil {
			if wp, ok := f.DeploymentConfig["webhookPort"].(int); ok && wp > 0 {
				webhookPort = wp
			}
		}

		buf.WriteString(fmt.Sprintf(`## Webhook server configuration
##
webhook:
  enable: true
  # Webhook server port
  port: %d

`, webhookPort))
	}

	// Prometheus configuration
	buf.WriteString(`## Prometheus ServiceMonitor for metrics scraping.
## Requires prometheus-operator to be installed in the cluster.
##
prometheus:
  enable: false
`)

	buf.WriteString("\n")
	return buf.String()
}

// addDeploymentConfig adds extracted deployment configuration to the values
func (f *HelmValuesBasic) addDeploymentConfig(buf *bytes.Buffer) {
	f.addArgsSection(buf)

	if f.DeploymentConfig == nil {
		// Add default sections with examples
		f.addDefaultDeploymentSections(buf)
		return
	}

	f.addEnvSection(buf)

	// Add image pull secrets
	if imagePullSecrets, exists := f.DeploymentConfig["imagePullSecrets"]; exists && imagePullSecrets != nil {
		buf.WriteString("  ## Image pull secrets\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  imagePullSecrets:\n")
		if imagePullSecretsYaml, err := yaml.Marshal(imagePullSecrets); err == nil {
			lines := bytes.SplitSeq(imagePullSecretsYaml, []byte("\n"))
			for line := range lines {
				if len(line) > 0 {
					buf.WriteString("    ")
					buf.Write(line)
					buf.WriteString("\n")
				}
			}
		}
		buf.WriteString("\n")
	} else {
		f.addDefaultImagePullSecrets(buf)
	}

	// Add podSecurityContext
	if podSecCtx, exists := f.DeploymentConfig["podSecurityContext"]; exists && podSecCtx != nil {
		buf.WriteString("  ## Pod-level security settings\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  podSecurityContext:\n")
		if secYaml, err := yaml.Marshal(podSecCtx); err == nil {
			f.IndentYamlProperly(buf, secYaml)
		}
		buf.WriteString("\n")
	} else {
		f.addDefaultPodSecurityContext(buf)
	}

	// Add securityContext
	if secCtx, exists := f.DeploymentConfig["securityContext"]; exists && secCtx != nil {
		buf.WriteString("  ## Container-level security settings\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  securityContext:\n")
		if secYaml, err := yaml.Marshal(secCtx); err == nil {
			f.IndentYamlProperly(buf, secYaml)
		}
		buf.WriteString("\n")
	} else {
		f.addDefaultSecurityContext(buf)
	}

	// Add resources
	if resources, exists := f.DeploymentConfig["resources"]; exists && resources != nil {
		buf.WriteString("  ## Resource limits and requests\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  resources:\n")
		if resYaml, err := yaml.Marshal(resources); err == nil {
			f.IndentYamlProperly(buf, resYaml)
		}
		buf.WriteString("\n")
	} else {
		f.addDefaultResources(buf)
	}

	buf.WriteString("  ## Manager pod's affinity\n")
	buf.WriteString("  ##\n")
	if affinity, exists := f.DeploymentConfig["podAffinity"]; exists && affinity != nil {
		buf.WriteString("  affinity:\n")
		if affYaml, err := yaml.Marshal(affinity); err == nil {
			f.IndentYamlProperly(buf, affYaml)
		}
		buf.WriteString("\n")
	} else {
		buf.WriteString("  affinity: {}\n")
		buf.WriteString("\n")
	}

	buf.WriteString("  ## Manager pod's node selector\n")
	buf.WriteString("  ##\n")
	if nodeSelector, exists := f.DeploymentConfig["podNodeSelector"]; exists && nodeSelector != nil {
		buf.WriteString("  nodeSelector:\n")
		if nodYaml, err := yaml.Marshal(nodeSelector); err == nil {
			f.IndentYamlProperly(buf, nodYaml)
		}
		buf.WriteString("\n")
	} else {
		buf.WriteString("  nodeSelector: {}\n")
		buf.WriteString("\n")
	}

	buf.WriteString("  ## Manager pod's tolerations\n")
	buf.WriteString("  ##\n")
	if tolerations, exists := f.DeploymentConfig["podTolerations"]; exists && tolerations != nil {
		buf.WriteString("  tolerations:\n")
		if tolYaml, err := yaml.Marshal(tolerations); err == nil {
			f.IndentYamlProperly(buf, tolYaml)
		}
		buf.WriteString("\n")
	} else {
		buf.WriteString("  tolerations: []\n")
		buf.WriteString("\n")
	}
}

func (f *HelmValuesBasic) IndentYamlProperly(buf *bytes.Buffer, envYaml []byte) {
	lines := bytes.SplitSeq(envYaml, []byte("\n"))
	for line := range lines {
		if len(line) > 0 {
			buf.WriteString("    ")
			buf.Write(line)
			buf.WriteString("\n")
		}
	}
}

// addEnvSection writes env (list, same as master) and only adds envOverrides for CLI.
func (f *HelmValuesBasic) addEnvSection(buf *bytes.Buffer) {
	buf.WriteString("  ## Environment variables\n")
	buf.WriteString("  ##\n")
	if env, exists := f.DeploymentConfig["env"]; exists && env != nil {
		if list, ok := env.([]any); ok && len(list) > 0 {
			buf.WriteString("  env:\n")
			if envYaml, err := yaml.Marshal(list); err == nil {
				f.IndentYamlProperly(buf, envYaml)
			}
		} else {
			buf.WriteString("  env: []\n")
		}
	} else {
		buf.WriteString("  env: []\n")
	}
	buf.WriteString("\n")
	buf.WriteString("  ## Env overrides (--set manager.envOverrides.VAR=value)\n")
	buf.WriteString("  ## Same name in env above: this value takes precedence.\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  envOverrides: {}\n")
	buf.WriteString("\n")
}

// addDefaultDeploymentSections adds default sections when no deployment config is available
func (f *HelmValuesBasic) addDefaultDeploymentSections(buf *bytes.Buffer) {
	buf.WriteString("  ## Environment variables\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  env: []\n")
	buf.WriteString("\n")
	buf.WriteString("  ## Env overrides (--set manager.envOverrides.VAR=value)\n")
	buf.WriteString("  ## Same name in env above: this value takes precedence.\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  envOverrides: {}\n")
	buf.WriteString("\n")

	f.addDefaultImagePullSecrets(buf)
	f.addDefaultPodSecurityContext(buf)
	f.addDefaultSecurityContext(buf)
	f.addDefaultResources(buf)
}

// addArgsSection adds controller manager args section to the values file
func (f *HelmValuesBasic) addArgsSection(buf *bytes.Buffer) {
	buf.WriteString("  ## Arguments\n  ##\n")

	if f.DeploymentConfig != nil {
		if args, exists := f.DeploymentConfig["args"]; exists && args != nil {
			if argsYaml, err := yaml.Marshal(args); err == nil {
				if trimmed := strings.TrimSpace(string(argsYaml)); trimmed != "" && trimmed != "[]" {
					lines := bytes.Split(argsYaml, []byte("\n"))
					buf.WriteString("  args:\n")
					for _, line := range lines {
						if len(line) > 0 {
							buf.WriteString("    ")
							buf.Write(line)
							buf.WriteString("\n")
						}
					}
					buf.WriteString("\n")
					return
				}
			}
		}
	}

	buf.WriteString("  args: []\n\n")
}

// addDefaultImagePullSecrets adds default imagePullSecrets section
func (f *HelmValuesBasic) addDefaultImagePullSecrets(buf *bytes.Buffer) {
	buf.WriteString("  ## Image pull secrets\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  imagePullSecrets: []\n\n")
}

// addDefaultPodSecurityContext adds default podSecurityContext section
func (f *HelmValuesBasic) addDefaultPodSecurityContext(buf *bytes.Buffer) {
	buf.WriteString("  ## Pod-level security settings\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  podSecurityContext: {}\n")
	buf.WriteString("    # fsGroup: 2000\n\n")
}

// addDefaultSecurityContext adds default securityContext section
func (f *HelmValuesBasic) addDefaultSecurityContext(buf *bytes.Buffer) {
	buf.WriteString("  ## Container-level security settings\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  securityContext: {}\n")
	buf.WriteString("    # capabilities:\n")
	buf.WriteString("    #   drop:\n")
	buf.WriteString("    #   - ALL\n")
	buf.WriteString("    # readOnlyRootFilesystem: true\n")
	buf.WriteString("    # runAsNonRoot: true\n")
	buf.WriteString("    # runAsUser: 1000\n\n")
}

// addDefaultResources adds default resources section
func (f *HelmValuesBasic) addDefaultResources(buf *bytes.Buffer) {
	buf.WriteString("  ## Resource limits and requests\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  resources: {}\n")
	buf.WriteString("    # limits:\n")
	buf.WriteString("    #   cpu: 100m\n")
	buf.WriteString("    #   memory: 128Mi\n")
	buf.WriteString("    # requests:\n")
	buf.WriteString("    #   cpu: 100m\n")
	buf.WriteString("    #   memory: 128Mi\n\n")
}
