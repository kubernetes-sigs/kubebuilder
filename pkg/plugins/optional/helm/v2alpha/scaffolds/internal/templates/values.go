/*
Copyright 2026 The Kubernetes Authors.

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
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/extractor"
)

var _ machinery.Template = &HelmValues{}

// HelmValues scaffolds values.yaml with comprehensive configuration extracted from kustomize resources.
type HelmValues struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// Extraction contains all extracted information from parsed resources
	Extraction *extractor.Extraction
	// OutputDir specifies the output directory for the chart
	OutputDir string
	// Force if true allows overwriting the scaffolded file
	Force bool
}

// SetTemplateDefaults implements machinery.Template
func (f *HelmValues) SetTemplateDefaults() error {
	if f.Path == "" {
		outputDir := f.OutputDir
		if outputDir == "" {
			outputDir = common.DefaultOutputDir
		}
		f.Path = filepath.Join(outputDir, "chart", "values.yaml")
	}

	f.TemplateBody = f.generateValues()

	f.IfExistsAction = machinery.SkipFile
	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	}

	return nil
}

// generateValues creates values.yaml using string buffer approach
func (f *HelmValues) generateValues() string {
	var buf bytes.Buffer

	// Header comments
	buf.WriteString(`## String to partially override chart.fullname template (will maintain the release name)
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

`)

	// Replicas
	replicas := 1
	if f.Extraction != nil && f.Extraction.Values.Manager.Replicas != nil {
		// Use extracted value (supports scale-to-zero: replicas can be 0)
		replicas = *f.Extraction.Values.Manager.Replicas
	}
	fmt.Fprintf(&buf, "  replicas: %d\n\n", replicas)

	// Image configuration
	f.addImageSection(&buf)

	// Deployment configuration
	f.addDeploymentConfig(&buf)

	// RBAC configuration
	f.addRBACSection(&buf)

	// ServiceAccount configuration
	f.addServiceAccountSection(&buf)

	// CRD configuration
	if f.Extraction != nil && f.Extraction.Features.HasCRDs {
		buf.WriteString(`## Custom Resource Definitions
##
crd:
  # Install CRDs with the chart
  enable: true
  # Keep CRDs when uninstalling
  keep: true

`)
	}

	// Metrics configuration (always present, enabled based on detected metrics artifacts)
	f.addMetricsSection(&buf)

	// Cert-manager configuration (always present)
	// IMPORTANT: Webhooks REQUIRE cert-manager for TLS certificates.
	// HasWebhooks = true means cert-manager MUST be enabled.
	// Also enabled when cert-manager resources exist (e.g., for metrics TLS without webhooks).
	if f.Extraction != nil && (f.Extraction.Features.HasWebhooks || f.Extraction.Features.HasCertManager) {
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

	// Webhook configuration
	if f.Extraction != nil && f.Extraction.Features.HasWebhooks {
		f.addWebhookSection(&buf)
	}

	// Prometheus configuration (always present)
	buf.WriteString(`## Prometheus ServiceMonitor for metrics scraping.
## Requires prometheus-operator to be installed in the cluster.
##
prometheus:
  enable: false

`)

	return buf.String()
}

// addImageSection adds the image configuration
func (f *HelmValues) addImageSection(buf *bytes.Buffer) {
	repo := "controller"
	tag := ""
	pullPolicy := "IfNotPresent"

	if f.Extraction != nil {
		if f.Extraction.Values.Manager.Image.Repository != "" {
			repo = f.Extraction.Values.Manager.Image.Repository
		}
		if f.Extraction.Values.Manager.Image.Tag != "" && f.Extraction.Values.Manager.Image.Tag != "latest" {
			tag = f.Extraction.Values.Manager.Image.Tag
		}
		if f.Extraction.Values.Manager.Image.PullPolicy != "" {
			pullPolicy = f.Extraction.Values.Manager.Image.PullPolicy
		}
	}

	buf.WriteString("  image:\n")
	fmt.Fprintf(buf, "    repository: %s\n", repo)
	buf.WriteString("    ## Image tag (defaults to Chart.appVersion if not set)\n")
	buf.WriteString("    ##\n")
	if tag == "" {
		buf.WriteString("    # tag: \"\"\n")
	} else {
		fmt.Fprintf(buf, "    tag: %q\n", tag)
	}
	fmt.Fprintf(buf, "    pullPolicy: %s\n\n", pullPolicy)
}

// addDeploymentConfig adds extracted deployment configuration
func (f *HelmValues) addDeploymentConfig(buf *bytes.Buffer) {
	// Args
	f.addArgsSection(buf)

	// Environment variables
	f.addEnvSection(buf)

	// Image pull secrets
	f.addImagePullSecretsSection(buf)

	// Pod security context
	f.addPodSecurityContextSection(buf)

	// Container security context
	f.addSecurityContextSection(buf)

	// Resources
	f.addResourcesSection(buf)

	// Affinity
	f.addAffinitySection(buf)

	// Node selector
	f.addNodeSelectorSection(buf)

	// Tolerations
	f.addTolerationsSection(buf)

	// Strategy
	f.addStrategySection(buf)

	// Priority class name
	f.addPriorityClassNameSection(buf)

	// Topology spread constraints
	f.addTopologySpreadConstraintsSection(buf)

	// Termination grace period
	f.addTerminationGracePeriodSection(buf)

	// Custom labels and annotations
	f.addCustomLabelsAnnotationsSection(buf)

	// Extra volumes and volume mounts
	f.addExtraVolumesSection(buf)
}

// addArgsSection adds the args configuration
func (f *HelmValues) addArgsSection(buf *bytes.Buffer) {
	if f.Extraction != nil && len(f.Extraction.Values.Manager.Args) > 0 {
		buf.WriteString("  ## Arguments\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  args:\n")
		f.marshalAndIndent(buf, f.Extraction.Values.Manager.Args, "args")
		buf.WriteString("\n")
	}
}

// addEnvSection adds the environment variables configuration
func (f *HelmValues) addEnvSection(buf *bytes.Buffer) {
	if f.Extraction != nil && len(f.Extraction.Values.Manager.Env) > 0 {
		buf.WriteString("  ## Environment variables\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  env:\n")
		f.marshalAndIndent(buf, f.Extraction.Values.Manager.Env, "env")
		buf.WriteString("\n")

		// Add envOverrides when env exists
		buf.WriteString("  ## Env overrides (--set manager.envOverrides.VAR=value)\n")
		buf.WriteString("  ## Same name in env above: this value takes precedence.\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  envOverrides: {}\n\n")
	}
}

// addImagePullSecretsSection adds image pull secrets configuration
func (f *HelmValues) addImagePullSecretsSection(buf *bytes.Buffer) {
	if f.Extraction != nil && len(f.Extraction.Values.Manager.ImagePullSecrets) > 0 {
		buf.WriteString("  ## Image pull secrets\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  imagePullSecrets:\n")
		f.marshalAndIndent(buf, f.Extraction.Values.Manager.ImagePullSecrets, "imagePullSecrets")
		buf.WriteString("\n")
	} else {
		buf.WriteString("  ## Image pull secrets\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  # imagePullSecrets:\n")
		buf.WriteString("  #   - name: myregistrykey\n\n")
	}
}

// addPodSecurityContextSection adds pod security context configuration
func (f *HelmValues) addPodSecurityContextSection(buf *bytes.Buffer) {
	buf.WriteString("  ## Pod-level security settings\n")
	buf.WriteString("  ##\n")
	if f.Extraction != nil && f.Extraction.Values.Manager.PodSecurityContext != nil {
		buf.WriteString("  podSecurityContext:\n")
		f.marshalAndIndent(buf, f.Extraction.Values.Manager.PodSecurityContext, "podSecurityContext")
		buf.WriteString("\n")
	} else {
		buf.WriteString("  # podSecurityContext:\n")
		buf.WriteString("  #   runAsNonRoot: true\n")
		buf.WriteString("  #   seccompProfile:\n")
		buf.WriteString("  #     type: RuntimeDefault\n\n")
	}
}

// addSecurityContextSection adds container security context configuration
func (f *HelmValues) addSecurityContextSection(buf *bytes.Buffer) {
	buf.WriteString("  ## Container-level security settings\n")
	buf.WriteString("  ##\n")
	if f.Extraction != nil && f.Extraction.Values.Manager.SecurityContext != nil {
		buf.WriteString("  securityContext:\n")
		f.marshalAndIndent(buf, f.Extraction.Values.Manager.SecurityContext, "securityContext")
		buf.WriteString("\n")
	} else {
		buf.WriteString("  # securityContext:\n")
		buf.WriteString("  #   allowPrivilegeEscalation: false\n")
		buf.WriteString("  #   capabilities:\n")
		buf.WriteString("  #     drop:\n")
		buf.WriteString("  #     - ALL\n\n")
	}
}

// addResourcesSection adds resources configuration
func (f *HelmValues) addResourcesSection(buf *bytes.Buffer) {
	buf.WriteString("  ## Resource limits and requests\n")
	buf.WriteString("  ##\n")
	if f.Extraction != nil && f.Extraction.Values.Manager.Resources != nil {
		buf.WriteString("  resources:\n")
		f.marshalAndIndent(buf, f.Extraction.Values.Manager.Resources, "resources")
		buf.WriteString("\n")
	} else {
		buf.WriteString("  # resources:\n")
		buf.WriteString("  #   limits:\n")
		buf.WriteString("  #     cpu: 500m\n")
		buf.WriteString("  #     memory: 128Mi\n")
		buf.WriteString("  #   requests:\n")
		buf.WriteString("  #     cpu: 10m\n")
		buf.WriteString("  #     memory: 64Mi\n\n")
	}
}

// addAffinitySection adds affinity configuration
func (f *HelmValues) addAffinitySection(buf *bytes.Buffer) {
	buf.WriteString("  ## Manager pod's affinity\n")
	buf.WriteString("  ##\n")
	if f.Extraction != nil && f.Extraction.Values.Manager.Affinity != nil {
		buf.WriteString("  affinity:\n")
		f.marshalAndIndent(buf, f.Extraction.Values.Manager.Affinity, "affinity")
		buf.WriteString("\n")
	} else {
		buf.WriteString("  affinity: {}\n\n")
	}
}

// addNodeSelectorSection adds node selector configuration
func (f *HelmValues) addNodeSelectorSection(buf *bytes.Buffer) {
	buf.WriteString("  ## Manager pod's node selector\n")
	buf.WriteString("  ##\n")
	if f.Extraction != nil && f.Extraction.Values.Manager.NodeSelector != nil {
		buf.WriteString("  nodeSelector:\n")
		f.marshalAndIndent(buf, f.Extraction.Values.Manager.NodeSelector, "nodeSelector")
		buf.WriteString("\n")
	} else {
		buf.WriteString("  nodeSelector: {}\n\n")
	}
}

// addTolerationsSection adds tolerations configuration
func (f *HelmValues) addTolerationsSection(buf *bytes.Buffer) {
	buf.WriteString("  ## Manager pod's tolerations\n")
	buf.WriteString("  ##\n")
	if f.Extraction != nil && len(f.Extraction.Values.Manager.Tolerations) > 0 {
		buf.WriteString("  tolerations:\n")
		f.marshalAndIndent(buf, f.Extraction.Values.Manager.Tolerations, "tolerations")
		buf.WriteString("\n")
	} else {
		buf.WriteString("  tolerations: []\n\n")
	}
}

// addStrategySection adds deployment strategy configuration
func (f *HelmValues) addStrategySection(buf *bytes.Buffer) {
	if f.Extraction != nil && f.Extraction.Values.Manager.Strategy != nil {
		buf.WriteString("  ## Deployment strategy\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  strategy:\n")
		f.marshalAndIndent(buf, f.Extraction.Values.Manager.Strategy, "strategy")
		buf.WriteString("\n")
	} else {
		buf.WriteString("  ## Deployment strategy\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  # strategy:\n")
		buf.WriteString("  #   type: RollingUpdate\n")
		buf.WriteString("  #   rollingUpdate:\n")
		buf.WriteString("  #     maxSurge: 25%\n")
		buf.WriteString("  #     maxUnavailable: 25%\n\n")
	}
}

// addPriorityClassNameSection adds priority class name configuration
func (f *HelmValues) addPriorityClassNameSection(buf *bytes.Buffer) {
	buf.WriteString("  ## Priority class name\n")
	buf.WriteString("  ##\n")
	if f.Extraction != nil && f.Extraction.Values.Manager.PriorityClassName != "" {
		fmt.Fprintf(buf, "  priorityClassName: %q\n\n", f.Extraction.Values.Manager.PriorityClassName)
	} else {
		buf.WriteString("  # priorityClassName: \"\"\n\n")
	}
}

// addTopologySpreadConstraintsSection adds topology spread constraints configuration
func (f *HelmValues) addTopologySpreadConstraintsSection(buf *bytes.Buffer) {
	if f.Extraction != nil && len(f.Extraction.Values.Manager.TopologySpreadConstraints) > 0 {
		buf.WriteString("  ## Topology spread constraints\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  topologySpreadConstraints:\n")
		f.marshalAndIndent(buf, f.Extraction.Values.Manager.TopologySpreadConstraints, "topologySpreadConstraints")
		buf.WriteString("\n")
	} else {
		buf.WriteString("  ## Topology spread constraints\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  # topologySpreadConstraints: []\n\n")
	}
}

// addTerminationGracePeriodSection adds termination grace period configuration
func (f *HelmValues) addTerminationGracePeriodSection(buf *bytes.Buffer) {
	if f.Extraction != nil && f.Extraction.Values.Manager.TerminationGracePeriodSeconds != nil {
		// Extracted value found - emit it (supports 0 for immediate termination)
		buf.WriteString("  ## Termination grace period seconds\n")
		buf.WriteString("  ##\n")
		fmt.Fprintf(buf, "  terminationGracePeriodSeconds: %d\n\n",
			*f.Extraction.Values.Manager.TerminationGracePeriodSeconds)
	} else {
		// No value extracted - keep as commented example
		buf.WriteString("  ## Termination grace period seconds\n")
		buf.WriteString("  ##\n")
		buf.WriteString("  # terminationGracePeriodSeconds: 10\n\n")
	}
}

// addCustomLabelsAnnotationsSection adds custom labels and annotations configuration
func (f *HelmValues) addCustomLabelsAnnotationsSection(buf *bytes.Buffer) {
	buf.WriteString("  ## Custom Deployment labels\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # labels: {}\n\n")
	buf.WriteString("  ## Custom Deployment annotations\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # annotations: {}\n\n")
	buf.WriteString("  ## Custom Pod labels and annotations\n")
	buf.WriteString("  ##\n")
	buf.WriteString("  # pod:\n")
	buf.WriteString("  #   labels: {}\n")
	buf.WriteString("  #   annotations: {}\n\n")
}

// addExtraVolumesSection adds extra volumes and volume mounts configuration
func (f *HelmValues) addExtraVolumesSection(buf *bytes.Buffer) {
	hasExtraVolumes := f.Extraction != nil && len(f.Extraction.Values.Manager.ExtraVolumes) > 0
	hasExtraVolumeMounts := f.Extraction != nil && len(f.Extraction.Values.Manager.ExtraVolumeMounts) > 0

	if hasExtraVolumes || hasExtraVolumeMounts {
		buf.WriteString("  ## Extra volumes and volume mounts\n")
		buf.WriteString("  ##\n")

		if hasExtraVolumes {
			buf.WriteString("  extraVolumes:\n")
			f.marshalAndIndent(buf, f.Extraction.Values.Manager.ExtraVolumes, "extraVolumes")
			buf.WriteString("\n")
		}

		if hasExtraVolumeMounts {
			buf.WriteString("  extraVolumeMounts:\n")
			f.marshalAndIndent(buf, f.Extraction.Values.Manager.ExtraVolumeMounts, "extraVolumeMounts")
			buf.WriteString("\n")
		}
	}
}

// addRBACSection adds RBAC configuration
func (f *HelmValues) addRBACSection(buf *bytes.Buffer) {
	buf.WriteString(`## RBAC configuration
##
rbac:
  ## RBAC resource scope
  ## - false (default): ClusterRole/ClusterRoleBinding (all namespaces)
  ## - true: Role/RoleBinding (release namespace only)
  ##
  namespaced: false

`)

	// Only add roleNamespaces if multi-namespace RBAC is detected
	if f.Extraction != nil && len(f.Extraction.Features.RoleNamespaces) > 0 {
		buf.WriteString(`  ## Multi-namespace RBAC role mappings (advanced use)
  ## Maps role suffixes to target namespaces for multi-namespace deployments
  ##
  roleNamespaces:
`)
		// Sort keys for deterministic output (avoid nondeterministic map iteration)
		keys := make([]string, 0, len(f.Extraction.Features.RoleNamespaces))
		for k := range f.Extraction.Features.RoleNamespaces {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		// Write sorted role namespace mappings with quoted keys and values to prevent YAML type coercion
		for _, k := range keys {
			fmt.Fprintf(buf, "    %q: %q\n", k, f.Extraction.Features.RoleNamespaces[k])
		}
		buf.WriteString("\n")
	}

	buf.WriteString(`  ## Helper roles for CRD management (admin/editor/viewer)
  ##
  helpers:
    ## Install convenience admin/editor/viewer roles for CRDs
    ##
    enable: false

`)
}

// addServiceAccountSection adds ServiceAccount configuration
func (f *HelmValues) addServiceAccountSection(buf *bytes.Buffer) {
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
}

// addMetricsSection adds metrics configuration
func (f *HelmValues) addMetricsSection(buf *bytes.Buffer) {
	port := 8443
	enableMetrics := false

	if f.Extraction != nil {
		enableMetrics = f.Extraction.Features.HasMetrics
		if f.Extraction.Features.MetricsPort > 0 {
			port = f.Extraction.Features.MetricsPort
		}
	}

	buf.WriteString(`## Controller metrics endpoint.
## Enable to expose /metrics endpoint
##
metrics:
`)
	fmt.Fprintf(buf, "  enable: %t\n", enableMetrics)
	buf.WriteString(`  # Metrics server port
`)
	fmt.Fprintf(buf, "  port: %d\n", port)
	buf.WriteString(`  # Enable secure metrics: HTTPS with certs/auth (true) or HTTP (false).
  # Note: Metrics authn/authz needs ClusterRole access.
  secure: true

`)
}

// addWebhookSection adds webhook configuration
func (f *HelmValues) addWebhookSection(buf *bytes.Buffer) {
	port := 9443
	if f.Extraction != nil && f.Extraction.Features.WebhookPort > 0 {
		port = f.Extraction.Features.WebhookPort
	}

	buf.WriteString(`## Webhook server configuration
##
webhook:
  enable: true
  # Webhook server port
`)
	fmt.Fprintf(buf, "  port: %d\n\n", port)
}

// indentYAML indents YAML content by 4 spaces
func (f *HelmValues) indentYAML(buf *bytes.Buffer, yamlContent []byte) {
	indent := strings.Repeat(" ", 4)
	lines := strings.SplitSeq(string(yamlContent), "\n")
	for line := range lines {
		if line != "" {
			buf.WriteString(indent)
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}
}

// marshalAndIndent marshals the value to YAML, logs errors, and indents the output
func (f *HelmValues) marshalAndIndent(buf *bytes.Buffer, value any, fieldName string) {
	yamlContent, err := yaml.Marshal(value)
	if err != nil {
		slog.Warn("Failed to marshal field for values.yaml", "field", fieldName, "error", err)
		buf.WriteString("    # Error: failed to marshal " + fieldName + "\n")
		return
	}
	f.indentYAML(buf, yamlContent)
}
