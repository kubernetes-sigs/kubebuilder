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

package appliers

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

// AddConditionalWrappers wraps resources with appropriate {{- if .Values.* }} conditionals.
// Each resource type gets wrapped based on its purpose and dependencies.
func AddConditionalWrappers(yamlContent string, resource *unstructured.Unstructured) string {
	kind := resource.GetKind()
	apiVersion := resource.GetAPIVersion()
	name := resource.GetName()

	switch {
	case kind == common.KindNamespace:
		return ""
	case kind == common.KindCRD:
		// Add resource-policy annotation to prevent deletion on helm uninstall
		yamlContent = InjectCRDResourcePolicyAnnotation(yamlContent)
		return fmt.Sprintf("{{- if .Values.crd.enable }}\n%s{{- end }}\n", yamlContent)
	case kind == common.KindCertificate && apiVersion == common.APIVersionCertManager:
		return HandleCertificateConditionalWrappers(yamlContent, name)
	case kind == common.KindIssuer && apiVersion == common.APIVersionCertManager:
		return fmt.Sprintf("{{- if .Values.certManager.enable }}\n%s\n{{- end }}", yamlContent)
	case kind == common.KindServiceMonitor && apiVersion == common.APIVersionMonitoring:
		// CRITICAL: newline before {{- end }} prevents whitespace chomping from eating content
		return fmt.Sprintf("{{- if .Values.prometheus.enable }}\n%s\n{{- end }}", yamlContent)
	case kind == common.KindServiceAccount, kind == common.KindRole, kind == common.KindClusterRole,
		kind == common.KindRoleBinding, kind == common.KindClusterRoleBinding:
		return HandleRBACConditionalWrappers(yamlContent, kind, name)
	case kind == common.KindValidatingWebhook || kind == common.KindMutatingWebhook:
		yamlContent = MakeWebhookAnnotationsConditional(yamlContent)
		return fmt.Sprintf("{{- if .Values.webhook.enable }}\n%s{{- end }}\n", yamlContent)
	case kind == common.KindService:
		return HandleServiceConditionalWrappers(yamlContent, name)
	case kind == common.KindDeployment:
		// Manager deployment conditional allows backward compatibility:
		// enabled when manager.enabled key is absent OR when explicitly true
		if IsManagerDeployment(resource) {
			return fmt.Sprintf(
				"{{- if or (not (hasKey .Values.manager \"enabled\")) (.Values.manager.enabled) }}\n%s\n{{- end }}\n",
				yamlContent,
			)
		}
		return yamlContent
	default:
		return yamlContent
	}
}

// HandleCertificateConditionalWrappers handles conditional logic for Certificate resources.
// Uses suffix matching to avoid false positives when project name contains "metrics".
func HandleCertificateConditionalWrappers(yamlContent, name string) string {
	isMetricsCert := strings.HasSuffix(name, "-metrics-certs") || strings.HasSuffix(name, "-metrics-cert")
	if isMetricsCert {
		// Metrics certificates require certManager AND metrics.secure=true (TLS enabled)
		return fmt.Sprintf(
			"{{- if and .Values.certManager.enable .Values.metrics.enable .Values.metrics.secure }}\n%s{{- end }}\n",
			yamlContent)
	}
	// Webhook serving certificates only need certManager
	return fmt.Sprintf("{{- if .Values.certManager.enable }}\n%s{{- end }}", yamlContent)
}

// HandleServiceConditionalWrappers handles conditional logic for Service resources.
// Uses suffix matching to avoid false positives when project name contains service types.
func HandleServiceConditionalWrappers(yamlContent, name string) string {
	if strings.HasSuffix(name, "-metrics-service") || strings.HasSuffix(name, "-controller-manager-metrics-service") {
		return fmt.Sprintf("{{- if .Values.metrics.enable }}\n%s{{- end }}\n", yamlContent)
	}
	if strings.HasSuffix(name, "-webhook-service") {
		return fmt.Sprintf("{{- if .Values.webhook.enable }}\n%s{{- end }}\n", yamlContent)
	}
	return yamlContent
}

// HandleRBACConditionalWrappers handles conditional logic for RBAC resources.
// Uses suffix matching to avoid false positives when project name contains role types.
func HandleRBACConditionalWrappers(yamlContent, kind, name string) string {
	// Helper roles (admin, editor, viewer) provide Kubernetes RBAC for custom resources.
	// These allow cluster admins to grant different access levels to CRs without cluster-admin.
	// Example: Grant namespace-scoped editor access to a team managing CRs in their namespace.
	isHelper := strings.HasSuffix(name, "-admin-role") || strings.HasSuffix(name, "-editor-role") ||
		strings.HasSuffix(name, "-viewer-role") ||
		strings.HasSuffix(name, "-admin-rolebinding") || strings.HasSuffix(name, "-editor-rolebinding") ||
		strings.HasSuffix(name, "-viewer-rolebinding")

	// Check for specific Kubebuilder-scaffolded metrics RBAC resources
	isMetricsAuthRole := strings.HasSuffix(name, "-metrics-auth-role")
	isMetricsAuthBinding := strings.HasSuffix(name, "-metrics-auth-rolebinding")
	isMetricsReader := strings.HasSuffix(name, "-metrics-reader")

	// Apply kind-switching for ClusterRole/ClusterRoleBinding (except metrics-auth role/binding and metrics-reader)
	isClusterRoleKind := kind == common.KindClusterRole || kind == common.KindClusterRoleBinding
	needsKindSwitching := !isMetricsAuthRole && !isMetricsAuthBinding && !isMetricsReader
	if isClusterRoleKind && needsKindSwitching {
		// Metrics-auth-role/binding/reader must stay ClusterRole/ClusterRoleBinding (use cluster-scoped APIs/nonResourceURLs)
		yamlContent = MakeRBACKindConditional(yamlContent, kind)
	}

	if isHelper {
		return fmt.Sprintf("{{- if .Values.rbac.helpers.enable }}\n%s{{- end }}\n", yamlContent)
	}
	if isMetricsAuthRole {
		// Only needed when secure metrics enabled (authn via TokenReview/SubjectAccessReview)
		return fmt.Sprintf("{{- if and .Values.metrics.enable .Values.metrics.secure }}\n%s{{- end }}\n", yamlContent)
	}
	if isMetricsReader {
		// Only needed when secure metrics enabled (uses nonResourceURLs for /metrics access)
		return fmt.Sprintf("{{- if and .Values.metrics.enable .Values.metrics.secure }}\n%s{{- end }}\n", yamlContent)
	}
	if isMetricsAuthBinding {
		// Binding for metrics-auth-role, only needed when secure metrics enabled
		return fmt.Sprintf("{{- if and .Values.metrics.enable .Values.metrics.secure }}\n%s{{- end }}\n", yamlContent)
	}
	// Essential RBAC (manager, leader-election) - always created
	return yamlContent
}

// MakeRBACKindConditional adds conditional rendering for ClusterRole/ClusterRoleBinding
// to switch between cluster-scoped and namespace-scoped based on .Values.rbac.namespaced.
func MakeRBACKindConditional(yamlContent string, kind string) string {
	var replacements []struct{ old, new string }

	if kind == common.KindClusterRole {
		// Replace kind: ClusterRole with conditional
		replacements = append(replacements, struct{ old, new string }{
			old: "kind: ClusterRole",
			new: "{{- if .Values.rbac.namespaced }}\nkind: Role\n{{- else }}\nkind: ClusterRole\n{{- end }}",
		})
		// Add namespace after metadata for Role variant
		replacements = append(replacements, struct{ old, new string }{
			old: "metadata:",
			new: "metadata:\n{{- if .Values.rbac.namespaced }}\n  namespace: {{ .Release.Namespace }}\n{{- end }}",
		})
	}

	if kind == common.KindClusterRoleBinding {
		// Replace kind: ClusterRoleBinding with conditional
		replacements = append(replacements, struct{ old, new string }{
			old: "kind: ClusterRoleBinding",
			new: "{{- if .Values.rbac.namespaced }}\nkind: RoleBinding\n{{- else }}\nkind: ClusterRoleBinding\n{{- end }}",
		})
		// Add namespace after metadata for RoleBinding variant
		replacements = append(replacements, struct{ old, new string }{
			old: "metadata:",
			new: "metadata:\n{{- if .Values.rbac.namespaced }}\n  namespace: {{ .Release.Namespace }}\n{{- end }}",
		})
		// Replace roleRef kind
		replacements = append(replacements, struct{ old, new string }{
			old: "  kind: ClusterRole",
			new: "  {{- if .Values.rbac.namespaced }}\n  kind: Role\n  {{- else }}\n  kind: ClusterRole\n  {{- end }}",
		})
	}

	result := yamlContent
	for _, r := range replacements {
		result = strings.Replace(result, r.old, r.new, 1)
	}

	return result
}

// InjectCRDResourcePolicyAnnotation adds the helm.sh/resource-policy: keep annotation to CRDs.
// This prevents Helm from deleting CRDs when the chart is uninstalled.
func InjectCRDResourcePolicyAnnotation(yamlContent string) string {
	// Check if metadata section exists
	if !strings.Contains(yamlContent, "metadata:") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")

	// Check if annotations: already exists
	if strings.Contains(yamlContent, common.YamlKeyAnnotations) {
		// Find the annotations: line and determine its indentation
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == common.YamlKeyAnnotations || strings.HasPrefix(trimmed, common.YamlKeyAnnotations) {
				annotationsIndent, _ := LeadingWhitespace(line)
				// Annotation values need one more level of indentation (2 spaces for sigs.k8s.io/yaml)
				valueIndent := annotationsIndent + "  "

				// Build the conditional annotation block
				resourcePolicyBlock := fmt.Sprintf(
					"%s{{- if .Values.crd.keep }}\n%s\"helm.sh/resource-policy\": keep\n%s{{- end }}",
					valueIndent, valueIndent, valueIndent)

				// Insert after the annotations: line
				result := make([]string, 0, len(lines)+3)
				result = append(result, lines[:i+1]...)
				result = append(result, resourcePolicyBlock)
				result = append(result, lines[i+1:]...)
				return strings.Join(result, "\n")
			}
		}
	} else {
		// No annotations section exists, need to add it after metadata:
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == common.YamlKeyMetadata || strings.HasPrefix(trimmed, common.YamlKeyMetadata) {
				metadataIndent, _ := LeadingWhitespace(line)
				// Fields under metadata need one more level of indentation (2 spaces for sigs.k8s.io/yaml)
				fieldIndent := metadataIndent + "  "
				// Annotation values need two more levels (4 spaces total)
				valueIndent := metadataIndent + "    "

				// Build annotations section with conditional resource-policy
				annotationsSection := fmt.Sprintf(
					"%sannotations:\n%s{{- if .Values.crd.keep }}\n%s\"helm.sh/resource-policy\": keep\n%s{{- end }}",
					fieldIndent, valueIndent, valueIndent, valueIndent)

				// Insert after the metadata: line
				result := make([]string, 0, len(lines)+4)
				result = append(result, lines[:i+1]...)
				result = append(result, annotationsSection)
				result = append(result, lines[i+1:]...)
				return strings.Join(result, "\n")
			}
		}
	}

	return yamlContent
}

// MakeWebhookAnnotationsConditional makes cert-manager annotations conditional on .Values.certManager.enable.
func MakeWebhookAnnotationsConditional(yamlContent string) string {
	// Find cert-manager.io/inject-ca-from annotation and make it conditional
	if strings.Contains(yamlContent, "cert-manager.io/inject-ca-from") {
		// Replace the cert-manager annotation with conditional wrapper
		certManagerPattern := regexp.MustCompile(`(\s+)cert-manager\.io/inject-ca-from:\s*[^\n]+`)
		yamlContent = certManagerPattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
			// Extract the indentation
			indentMatch := regexp.MustCompile(`^(\s+)`).FindStringSubmatch(match)
			indent := ""
			if len(indentMatch) > 1 {
				indent = indentMatch[1]
			}

			// Extract the annotation line with proper indentation
			annotationLine := strings.TrimSpace(match)

			return fmt.Sprintf("%s{{- if .Values.certManager.enable }}\n%s%s\n%s{{- end }}",
				indent, indent, annotationLine, indent)
		})
	}

	return yamlContent
}
