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
	"path/filepath"

	"gopkg.in/yaml.v3"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &HelmValuesBasic{}

// HelmValuesBasic scaffolds a basic values.yaml based on detected features
type HelmValuesBasic struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// DeploymentConfig stores extracted deployment configuration (env, resources, security contexts)
	DeploymentConfig map[string]interface{}
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

	// Extract values from kustomize output
	imageRepo := "controller"    // default
	imageTag := "latest"         // default
	imageDigest := ""            // empty by default
	pullPolicy := "IfNotPresent" // default

	if f.DeploymentConfig != nil {
		// Use extracted image values from kustomize
		if img, ok := f.DeploymentConfig["image"].(map[string]interface{}); ok {
			if repo, ok := img["repository"].(string); ok {
				imageRepo = repo
			}
			// Digest takes precedence over tag
			if dig, ok := img["digest"].(string); ok && dig != "" {
				imageDigest = dig
				imageTag = "" // clear tag when using digest
			} else if tag, ok := img["tag"].(string); ok && tag != "" {
				imageTag = tag
			}
		}
		if pp, ok := f.DeploymentConfig["imagePullPolicy"].(string); ok && pp != "" {
			pullPolicy = pp
		}
	}

	// Controller Manager configuration
	buf.WriteString(`# Configure the controller manager deployment
controllerManager:
  replicas: 1
  
  image:
    repository: ` + imageRepo + "\n")

	// Only include tag or digest, not both
	if imageDigest != "" {
		buf.WriteString(`    # Using digest from kustomize
    digest: ` + imageDigest + "\n")
	} else {
		buf.WriteString(`    tag: ` + imageTag + "\n")
	}

	buf.WriteString(`    pullPolicy: ` + pullPolicy + "\n\n")

	// Add extracted deployment configuration
	f.addDeploymentConfig(&buf)

	// RBAC configuration
	buf.WriteString(`# Essential RBAC permissions (required for controller operation)
# These include ServiceAccount, controller permissions, leader election, and metrics access
# Note: Essential RBAC is always enabled as it's required for the controller to function

# Helper RBAC roles for managing custom resources
# These provide convenient admin/editor/viewer roles for each CRD type
# Useful for giving users different levels of access to your custom resources
rbacHelpers:
  enable: false  # Install convenience admin/editor/viewer roles for CRDs

`)

	// CRD configuration
	buf.WriteString(`# Custom Resource Definitions
crd:
  enable: true  # Install CRDs with the chart
  keep: true    # Keep CRDs when uninstalling

`)

	// Metrics configuration (enable if metrics artifacts detected in kustomize output)
	if f.HasMetrics {
		buf.WriteString(`# Controller metrics endpoint.
# Enable to expose /metrics endpoint with RBAC protection.
metrics:
  enable: true

`)
	} else {
		buf.WriteString(`# Controller metrics endpoint.
# Enable to expose /metrics endpoint with RBAC protection.
metrics:
  enable: false

`)
	}

	// Cert-manager configuration - only if certificates/webhooks are present
	if f.HasWebhooks {
		buf.WriteString(`# Cert-manager integration for TLS certificates.
# Required for webhook certificates and metrics endpoint certificates.
certManager:
  enable: true

`)
	}

	// Prometheus configuration
	buf.WriteString(`# Prometheus ServiceMonitor for metrics scraping.
# Requires prometheus-operator to be installed in the cluster.
prometheus:
  enable: false
`)

	buf.WriteString("\n")
	return buf.String()
}

// addDeploymentConfig adds extracted deployment configuration to the values
func (f *HelmValuesBasic) addDeploymentConfig(buf *bytes.Buffer) {
	if f.DeploymentConfig == nil {
		// Add default sections with examples
		f.addDefaultDeploymentSections(buf)
		return
	}

	// Add environment variables if they exist
	if env, exists := f.DeploymentConfig["env"]; exists && env != nil {
		buf.WriteString("  # Environment variables\n")
		buf.WriteString("  env:\n")
		if envYaml, err := yaml.Marshal(env); err == nil {
			// Indent the YAML properly
			lines := bytes.Split(envYaml, []byte("\n"))
			for _, line := range lines {
				if len(line) > 0 {
					buf.WriteString("    ")
					buf.Write(line)
					buf.WriteString("\n")
				}
			}
		} else {
			buf.WriteString("    []\n")
		}
		buf.WriteString("\n")
	} else {
		buf.WriteString("  # Environment variables\n")
		buf.WriteString("  env: []\n\n")
	}

	// Add podSecurityContext
	if podSecCtx, exists := f.DeploymentConfig["podSecurityContext"]; exists && podSecCtx != nil {
		buf.WriteString("  # Pod-level security settings\n")
		buf.WriteString("  podSecurityContext:\n")
		if secYaml, err := yaml.Marshal(podSecCtx); err == nil {
			lines := bytes.Split(secYaml, []byte("\n"))
			for _, line := range lines {
				if len(line) > 0 {
					buf.WriteString("    ")
					buf.Write(line)
					buf.WriteString("\n")
				}
			}
		}
		buf.WriteString("\n")
	} else {
		f.addDefaultPodSecurityContext(buf)
	}

	// Add securityContext
	if secCtx, exists := f.DeploymentConfig["securityContext"]; exists && secCtx != nil {
		buf.WriteString("  # Container-level security settings\n")
		buf.WriteString("  securityContext:\n")
		if secYaml, err := yaml.Marshal(secCtx); err == nil {
			lines := bytes.Split(secYaml, []byte("\n"))
			for _, line := range lines {
				if len(line) > 0 {
					buf.WriteString("    ")
					buf.Write(line)
					buf.WriteString("\n")
				}
			}
		}
		buf.WriteString("\n")
	} else {
		f.addDefaultSecurityContext(buf)
	}

	// Add resources
	if resources, exists := f.DeploymentConfig["resources"]; exists && resources != nil {
		buf.WriteString("  # Resource limits and requests\n")
		buf.WriteString("  resources:\n")
		if resYaml, err := yaml.Marshal(resources); err == nil {
			lines := bytes.Split(resYaml, []byte("\n"))
			for _, line := range lines {
				if len(line) > 0 {
					buf.WriteString("    ")
					buf.Write(line)
					buf.WriteString("\n")
				}
			}
		}
		buf.WriteString("\n")
	} else {
		f.addDefaultResources(buf)
	}
}

// addDefaultDeploymentSections adds default sections when no deployment config is available
func (f *HelmValuesBasic) addDefaultDeploymentSections(buf *bytes.Buffer) {
	buf.WriteString("  # Environment variables\n")
	buf.WriteString("  env: []\n\n")

	f.addDefaultPodSecurityContext(buf)
	f.addDefaultSecurityContext(buf)
	f.addDefaultResources(buf)
}

// addDefaultPodSecurityContext adds default podSecurityContext section
func (f *HelmValuesBasic) addDefaultPodSecurityContext(buf *bytes.Buffer) {
	buf.WriteString("  # Pod-level security settings\n")
	buf.WriteString("  podSecurityContext: {}\n")
	buf.WriteString("    # fsGroup: 2000\n\n")
}

// addDefaultSecurityContext adds default securityContext section
func (f *HelmValuesBasic) addDefaultSecurityContext(buf *bytes.Buffer) {
	buf.WriteString("  # Container-level security settings\n")
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
	buf.WriteString("  # Resource limits and requests\n")
	buf.WriteString("  resources: {}\n")
	buf.WriteString("    # limits:\n")
	buf.WriteString("    #   cpu: 100m\n")
	buf.WriteString("    #   memory: 128Mi\n")
	buf.WriteString("    # requests:\n")
	buf.WriteString("    #   cpu: 100m\n")
	buf.WriteString("    #   memory: 128Mi\n\n")
}
