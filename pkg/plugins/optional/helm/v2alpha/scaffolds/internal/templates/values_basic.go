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
	"slices"
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
	// HasClusterScopedRBAC is true when ClusterRole resources were found (excluding metrics-auth-role)
	HasClusterScopedRBAC bool
	// RoleNamespaces maps Role/RoleBinding resource name suffixes (without project prefix) to their target namespaces
	// for multi-namespace RBAC. Keys are suffixes for stability across different release names.
	// Example: {"manager-role-infrastructure": "infrastructure", "manager-role-users": "users"}
	// NOT: {"project-manager-role-infrastructure": ...} - the project prefix is stripped.
	RoleNamespaces map[string]string
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
	imageTag := "" // Defaults to Chart.appVersion in template
	imagePullPolicy := "IfNotPresent"
	replicas := 1
	if f.DeploymentConfig != nil {
		if imgCfg, ok := f.DeploymentConfig["image"].(map[string]any); ok {
			if repo, ok := imgCfg["repository"].(string); ok && repo != "" {
				imageRepo = repo
			}
			if tag, ok := imgCfg["tag"].(string); ok && tag != "" && tag != "latest" {
				imageTag = tag
			}
			if policy, ok := imgCfg["pullPolicy"].(string); ok && policy != "" {
				imagePullPolicy = policy
			}
		}
		// Accept replicas >= 0 (scale-to-zero is valid)
		// Handle both int and int64 (YAML unmarshaling often produces int64)
		if repValue, exists := f.DeploymentConfig["replicas"]; exists {
			if rep, ok := repValue.(int); ok && rep >= 0 {
				replicas = rep
			} else if rep64, ok := repValue.(int64); ok && rep64 >= 0 {
				replicas = int(rep64)
			}
		}
	}

	var imageSection string
	if imageTag == "" {
		imageSection = fmt.Sprintf(`  image:
    repository: %s
    ## Defaults to Chart.appVersion
    ##
    # tag: ""
    pullPolicy: %s`, imageRepo, imagePullPolicy)
	} else {
		imageSection = fmt.Sprintf(`  image:
    repository: %s
    tag: %s
    pullPolicy: %s`, imageRepo, imageTag, imagePullPolicy)
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
  ## Set to false to skip manager installation
  ##
  enabled: true

  replicas: %d

%s

`, replicas, imageSection))

	// Add extracted deployment configuration
	f.addDeploymentConfig(&buf)

	// RBAC configuration
	buf.WriteString(`## RBAC configuration
##
rbac:
`)
	// Only add namespaced option if the project has cluster-scoped RBAC
	if f.HasClusterScopedRBAC {
		buf.WriteString(`  ## RBAC resource scope
  ## - false (default): ClusterRole/ClusterRoleBinding (all namespaces)
  ## - true: Role/RoleBinding (release namespace only)
  ##
  namespaced: false

`)
	}

	// Add role-specific namespace configuration if detected
	if len(f.RoleNamespaces) > 0 {
		buf.WriteString(`  ## Namespace configuration for Roles deployed to namespaces different from the manager namespace
  ## Keys are resource name suffixes (without project prefix)
  ##
  roleNamespaces:
`)
		// Sort role names for consistent output
		roleNames := make([]string, 0, len(f.RoleNamespaces))
		for roleName := range f.RoleNamespaces {
			roleNames = append(roleNames, roleName)
		}
		slices.Sort(roleNames)

		for _, roleName := range roleNames {
			ns := f.RoleNamespaces[roleName]
			buf.WriteString(fmt.Sprintf("    # RBAC resource %s deploys to namespace %s\n", roleName, ns))
			buf.WriteString(fmt.Sprintf("    %q: %q\n", roleName, ns))
		}
		buf.WriteString("\n")
	}

	// Add helper roles section
	buf.WriteString(`  ## Helper roles for CRD management (admin/editor/viewer)
  ##
`)
	buf.WriteString(`  helpers:
    ## Install convenience admin/editor/viewer roles for CRDs
    ##
    enable: false

`)

	// ServiceAccount configuration
	buf.WriteString(`## ServiceAccount configuration
##
serviceAccount:
  # Install default ServiceAccount provided
  enable: true

  ## Existing ServiceAccount name (only when enable=false)
  ## Note: When enable=true, respects nameOverride/fullnameOverride
  ##
  # name: ""

  ## Custom ServiceAccount annotations
  ##
  # annotations: {}

  ## Custom ServiceAccount labels
  ##
  # labels: {}

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
## Enable to expose /metrics endpoint
##
metrics:
  enable: true
  # Metrics server port
  port: %d
  # Enable secure metrics: HTTPS with certs/auth (true) or HTTP (false).
  # Note: Metrics authn/authz needs ClusterRole access.
  secure: true

`, metricsPort))
	} else {
		buf.WriteString(fmt.Sprintf(`## Controller metrics endpoint.
## Enable to expose /metrics endpoint
##
metrics:
  enable: false
  # Metrics server port
  port: %d
  # Enable secure metrics: HTTPS with certs/auth (true) or HTTP (false).
  # Note: Metrics authn/authz needs ClusterRole access.
  secure: true

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

	f.addStrategySection(buf)
	f.addPriorityClassNameSection(buf)
	f.addTopologySpreadConstraintsSection(buf)
	f.addTerminationGracePeriodSecondsSection(buf)
	f.addCustomLabelsAnnotationsSections(buf)
	f.addExtraVolumesFromConfig(buf)
}

// addCustomLabelsAnnotationsSections adds custom labels and annotations sections for both
// Deployment and Pod template metadata to values.yaml.
func (f *HelmValuesBasic) addCustomLabelsAnnotationsSections(buf *bytes.Buffer) {
	buf.WriteString("  ## Custom Deployment labels\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # labels: {}\n")
	buf.WriteString("\n")

	buf.WriteString("  ## Custom Deployment annotations\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # annotations: {}\n")
	buf.WriteString("\n")

	buf.WriteString("  ## Custom Pod labels and annotations\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # pod:\n")
	buf.WriteString("  #   labels: {}\n")
	buf.WriteString("  #   annotations: {}\n")
	buf.WriteString("\n")
}

// addExtraVolumesFromConfig adds manager.extraVolumeMounts and manager.extraVolumes to values
// only when the deployment config has extra volumes (not webhook/metrics). Config volumes
// are in the chart template; use these keys to add more without re-running edit.
func (f *HelmValuesBasic) addExtraVolumesFromConfig(buf *bytes.Buffer) {
	if f.DeploymentConfig == nil {
		return
	}
	_, hasMounts := f.DeploymentConfig["extraVolumeMounts"]
	_, hasVols := f.DeploymentConfig["extraVolumes"]
	if !hasMounts && !hasVols {
		return
	}
	buf.WriteString("  ## Additional volume mounts\n")
	buf.WriteString("  extraVolumeMounts: []\n")
	buf.WriteString("  extraVolumes: []\n")
	buf.WriteString("\n")
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

// addEnvSection writes env and envOverrides ONLY if env exists in kustomize input.
// These are operator-specific runtime configurations that are part of the operator's contract.
// We only add them to values.yaml when found in the kustomize-generated deployment,
// preventing users from being misled into thinking unsupported env vars are configurable.
func (f *HelmValuesBasic) addEnvSection(buf *bytes.Buffer) {
	if env, exists := f.DeploymentConfig["env"]; exists && env != nil {
		buf.WriteString("  ## Environment variables\n")
		buf.WriteString("  ##\n")
		if list, ok := env.([]any); ok && len(list) > 0 {
			buf.WriteString("  env:\n")
			if envYaml, err := yaml.Marshal(list); err == nil {
				f.IndentYamlProperly(buf, envYaml)
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
	// If env doesn't exist in kustomize, don't add env or envOverrides at all
}

// addDefaultDeploymentSections adds optional Kubernetes deployment fields as commented examples.
// These are shown as commented to guide users on available configuration options.
// Operator-specific fields (env, args) are handled separately and only appear when found in kustomize.
func (f *HelmValuesBasic) addDefaultDeploymentSections(buf *bytes.Buffer) {
	f.addDefaultImagePullSecrets(buf)
	f.addDefaultPodSecurityContext(buf)
	f.addDefaultSecurityContext(buf)
	f.addDefaultResources(buf)

	// Add standard scheduling fields
	buf.WriteString("  ## Manager pod's affinity\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  affinity: {}\n")
	buf.WriteString("\n")

	buf.WriteString("  ## Manager pod's node selector\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  nodeSelector: {}\n")
	buf.WriteString("\n")

	buf.WriteString("  ## Manager pod's tolerations\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  tolerations: []\n")
	buf.WriteString("\n")

	// Add new optional Kubernetes fields (always shown for discoverability)
	f.addStrategySection(buf)
	f.addPriorityClassNameSection(buf)
	f.addTopologySpreadConstraintsSection(buf)
	f.addTerminationGracePeriodSecondsSection(buf)
	f.addCustomLabelsAnnotationsSections(buf)
}

// addArgsSection adds controller manager args ONLY if args exist in kustomize input.
// Args are operator-specific runtime configuration that are part of the operator's contract.
// We only add them to values.yaml when found in the kustomize-generated deployment,
// preventing users from being misled into thinking arbitrary args are supported.
func (f *HelmValuesBasic) addArgsSection(buf *bytes.Buffer) {
	if f.DeploymentConfig != nil {
		if args, exists := f.DeploymentConfig["args"]; exists && args != nil {
			buf.WriteString("  ## Arguments\n  ##\n")
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
			// If args exists but is empty/invalid, still show []
			buf.WriteString("  args: []\n\n")
		}
	}
	// If args doesn't exist in kustomize, don't add it at all
}

// addDefaultImagePullSecrets adds imagePullSecrets as a commented example.
// This is an optional Kubernetes feature shown to guide users.
func (f *HelmValuesBasic) addDefaultImagePullSecrets(buf *bytes.Buffer) {
	buf.WriteString("  ## Image pull secrets\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # imagePullSecrets:\n")
	buf.WriteString("  #   - name: myregistrykey\n\n")
}

// addDefaultPodSecurityContext adds podSecurityContext as a commented example.
// This is an optional Kubernetes feature shown to guide users.
func (f *HelmValuesBasic) addDefaultPodSecurityContext(buf *bytes.Buffer) {
	buf.WriteString("  ## Pod-level security settings\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # podSecurityContext:\n")
	buf.WriteString("  #   fsGroup: 2000\n\n")
}

// addDefaultSecurityContext adds securityContext as a commented example.
// This is an optional Kubernetes feature shown to guide users.
func (f *HelmValuesBasic) addDefaultSecurityContext(buf *bytes.Buffer) {
	buf.WriteString("  ## Container-level security settings\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # securityContext:\n")
	buf.WriteString("  #   capabilities:\n")
	buf.WriteString("  #     drop:\n")
	buf.WriteString("  #     - ALL\n")
	buf.WriteString("  #   readOnlyRootFilesystem: true\n")
	buf.WriteString("  #   runAsNonRoot: true\n")
	buf.WriteString("  #   runAsUser: 1000\n\n")
}

// addDefaultResources adds resources as a commented example.
// This is an optional Kubernetes feature shown to guide users.
func (f *HelmValuesBasic) addDefaultResources(buf *bytes.Buffer) {
	buf.WriteString("  ## Resource limits and requests\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # resources:\n")
	buf.WriteString("  #   limits:\n")
	buf.WriteString("  #     cpu: 100m\n")
	buf.WriteString("  #     memory: 128Mi\n")
	buf.WriteString("  #   requests:\n")
	buf.WriteString("  #     cpu: 100m\n")
	buf.WriteString("  #     memory: 128Mi\n\n")
}

// addStrategySection adds deployment strategy section.
// Shows actual value from kustomize if found, otherwise provides a commented example.
func (f *HelmValuesBasic) addStrategySection(buf *bytes.Buffer) {
	if f.DeploymentConfig != nil {
		if strategy, exists := f.DeploymentConfig["strategy"]; exists && strategy != nil {
			buf.WriteString("  ## Deployment strategy\n")
			buf.WriteString("  ##\n")
			buf.WriteString("  strategy:\n")
			if stratYaml, err := yaml.Marshal(strategy); err == nil {
				f.IndentYamlProperly(buf, stratYaml)
			}
			buf.WriteString("\n")
			return
		}
	}
	buf.WriteString("  ## Deployment strategy\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # strategy:\n")
	buf.WriteString("  #   type: RollingUpdate\n")
	buf.WriteString("  #   rollingUpdate:\n")
	buf.WriteString("  #     maxSurge: 25%\n")
	buf.WriteString("  #     maxUnavailable: 25%\n")
	buf.WriteString("\n")
}

// addPriorityClassNameSection adds priorityClassName for pod scheduling priority.
// Shows actual value from kustomize if found, otherwise provides a commented example.
func (f *HelmValuesBasic) addPriorityClassNameSection(buf *bytes.Buffer) {
	if f.DeploymentConfig != nil {
		if priorityClassName, exists := f.DeploymentConfig["priorityClassName"].(string); exists && priorityClassName != "" {
			buf.WriteString("  ## Priority class name\n")
			buf.WriteString("  ##\n")
			// Use YAML marshaling to ensure proper quoting (handles values like "true", "null", etc.)
			if priorityClassNameYAML, err := yaml.Marshal(priorityClassName); err == nil {
				fmt.Fprintf(buf, "  priorityClassName: %s\n\n", strings.TrimSpace(string(priorityClassNameYAML)))
				return
			}
		}
	}
	buf.WriteString("  ## Priority class name\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # priorityClassName: \"\"\n\n")
}

// addTopologySpreadConstraintsSection adds topology spread constraints for high availability.
// Shows actual value from kustomize if found, otherwise provides a commented example.
func (f *HelmValuesBasic) addTopologySpreadConstraintsSection(buf *bytes.Buffer) {
	if f.DeploymentConfig != nil {
		topologySpreadConstraints, hasTopologySpreadConstraints := f.DeploymentConfig["topologySpreadConstraints"]
		if hasTopologySpreadConstraints && topologySpreadConstraints != nil {
			buf.WriteString("  ## Topology spread constraints\n")
			buf.WriteString("  ##\n")
			buf.WriteString("  topologySpreadConstraints:\n")
			if tscYaml, err := yaml.Marshal(topologySpreadConstraints); err == nil {
				f.IndentYamlProperly(buf, tscYaml)
			}
			buf.WriteString("\n")
			return
		}
	}
	buf.WriteString("  ## Topology spread constraints\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # topologySpreadConstraints: []\n\n")
}

// addTerminationGracePeriodSecondsSection adds graceful shutdown period.
// Shows actual value from kustomize if found, otherwise provides a commented example.
func (f *HelmValuesBasic) addTerminationGracePeriodSecondsSection(buf *bytes.Buffer) {
	if f.DeploymentConfig != nil {
		// Accept terminationGracePeriodSeconds >= 0 (0 means immediate termination, which is valid)
		if tgpsValue, exists := f.DeploymentConfig["terminationGracePeriodSeconds"]; exists && tgpsValue != nil {
			buf.WriteString("  ## Termination grace period seconds\n")
			buf.WriteString("  ##\n")
			if tgpsYaml, err := yaml.Marshal(tgpsValue); err == nil {
				fmt.Fprintf(buf, "  terminationGracePeriodSeconds: %s\n\n", strings.TrimSpace(string(tgpsYaml)))
				return
			}
		}
	}
	buf.WriteString("  ## Termination grace period seconds\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # terminationGracePeriodSeconds: 10\n\n")
}
