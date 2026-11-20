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
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

// AddConditionalWrappers wraps resources with appropriate {{- if .Values.* }} conditionals.
// Each resource type gets wrapped based on its purpose and dependencies.
func AddConditionalWrappers(
	yamlContent string, resource *unstructured.Unstructured, detectedPrefix string,
) string {
	kind := resource.GetKind()
	apiVersion := resource.GetAPIVersion()
	name := resource.GetName()

	switch {
	case kind == common.KindNamespace:
		return ""
	case kind == common.KindCRD:
		// Add resource-policy annotation to prevent deletion on helm uninstall
		yamlContent = InjectCRDResourcePolicyAnnotation(yamlContent)
		yamlContent = fmt.Sprintf("{{- if .Values.crd.enabled }}\n%s{{- end }}\n", yamlContent)
		if !hasWebhookConversion(resource) {
			return yamlContent
		}
		return fmt.Sprintf(`{{- if and .Values.crd.enabled (not .Values.webhook.enabled) }}
{{- fail "webhook.enabled must be true for CRD conversion" }}
{{- end }}
%s`, yamlContent)
	case kind == common.KindCertificate && apiVersion == common.APIVersionCertManager:
		return HandleCertificateConditionalWrappers(yamlContent, name, detectedPrefix)
	case kind == common.KindIssuer && apiVersion == common.APIVersionCertManager:
		return fmt.Sprintf("{{- if .Values.certManager.enabled }}\n%s\n{{- end }}", yamlContent)
	case kind == common.KindServiceMonitor && apiVersion == common.APIVersionMonitoring:
		// The ServiceMonitor scrapes the metrics Service, so it needs metrics enabled too.
		// CRITICAL: newline before {{- end }} prevents whitespace chomping from eating content
		return fmt.Sprintf(
			"{{- if and .Values.prometheus.enabled .Values.metrics.enabled }}\n%s\n{{- end }}",
			yamlContent,
		)
	case kind == common.KindNetworkPolicy && apiVersion == common.APIVersionNetworking:
		switch {
		case IsScaffoldedPolicyName(name, detectedPrefix, "allow-webhook-traffic"):
			return fmt.Sprintf(
				"{{- if and .Values.networkPolicy.enabled .Values.webhook.enabled }}\n%s\n{{- end }}",
				yamlContent,
			)
		case IsScaffoldedPolicyName(name, detectedPrefix, "allow-metrics-traffic"):
			return fmt.Sprintf(
				"{{- if and .Values.networkPolicy.enabled .Values.metrics.enabled }}\n%s\n{{- end }}",
				yamlContent,
			)
		default:
			return fmt.Sprintf("{{- if .Values.networkPolicy.enabled }}\n%s\n{{- end }}", yamlContent)
		}
	case kind == common.KindServiceAccount, kind == common.KindRole, kind == common.KindClusterRole,
		kind == common.KindRoleBinding, kind == common.KindClusterRoleBinding:
		return HandleRBACConditionalWrappers(yamlContent, kind, name)
	case kind == common.KindValidatingWebhook || kind == common.KindMutatingWebhook:
		yamlContent = MakeWebhookAnnotationsConditional(yamlContent)
		return fmt.Sprintf("{{- if .Values.webhook.enabled }}\n%s{{- end }}\n", yamlContent)
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

func hasWebhookConversion(resource *unstructured.Unstructured) bool {
	strategy, found, err := unstructured.NestedString(resource.Object, "spec", "conversion", "strategy")
	return err == nil && found && strategy == "Webhook"
}

// HandleCertificateConditionalWrappers handles conditional logic for Certificate resources.
// It identifies the scaffolded webhook certificate by its exact generated name.
func HandleCertificateConditionalWrappers(yamlContent, name, detectedPrefix string) string {
	isMetricsCert := strings.HasSuffix(name, "-metrics-certs") || strings.HasSuffix(name, "-metrics-cert")
	if isMetricsCert {
		// Metrics certificates require certManager AND metrics.secure=true (TLS enabled)
		return fmt.Sprintf(
			"{{- if and .Values.certManager.enabled .Values.metrics.enabled .Values.metrics.secure }}\n%s{{- end }}\n",
			yamlContent)
	}
	if name == detectedPrefix+"-serving-cert" {
		// The scaffolded webhook serving certificate requires both features.
		return fmt.Sprintf(
			"{{- if and .Values.certManager.enabled .Values.webhook.enabled }}\n%s{{- end }}", yamlContent)
	}
	return fmt.Sprintf("{{- if .Values.certManager.enabled }}\n%s{{- end }}", yamlContent)
}

// HandleServiceConditionalWrappers handles conditional logic for Service resources.
// Uses suffix matching to avoid false positives when project name contains service types.
func HandleServiceConditionalWrappers(yamlContent, name string) string {
	if strings.HasSuffix(name, "-metrics-service") || strings.HasSuffix(name, "-controller-manager-metrics-service") {
		return fmt.Sprintf("{{- if .Values.metrics.enabled }}\n%s{{- end }}\n", yamlContent)
	}
	if strings.HasSuffix(name, "-webhook-service") {
		return fmt.Sprintf("{{- if .Values.webhook.enabled }}\n%s{{- end }}\n", yamlContent)
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
		return fmt.Sprintf("{{- if .Values.rbac.helpers.enabled }}\n%s{{- end }}\n", yamlContent)
	}
	// metrics-auth-role, metrics-reader, and metrics-auth-rolebinding all require secure metrics
	if isMetricsAuthRole || isMetricsReader || isMetricsAuthBinding {
		return fmt.Sprintf("{{- if and .Values.metrics.enabled .Values.metrics.secure }}\n%s{{- end }}\n", yamlContent)
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
// Only the CRD's own (first) metadata is considered: nested "annotations:" keys, such as those in
// the OpenAPI schema of a CRD that embeds ObjectMeta, are left untouched.
func InjectCRDResourcePolicyAnnotation(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")

	metadataLine := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == common.YamlKeyMetadata {
			metadataLine = i
			break
		}
	}
	if metadataLine < 0 {
		return yamlContent
	}
	metadataIndentStr, metadataIndent := LeadingWhitespace(lines[metadataLine])

	// Look for an annotations block directly under metadata.
	insertAfter := -1
	for i := metadataLine + 1; i < len(lines) && !closesMetadataBlock(lines[i], metadataIndent); i++ {
		if isMetadataMapChildHeader(lines[i], common.YamlKeyAnnotations, metadataIndent) {
			insertAfter = i
			break
		}
	}

	// Fields under metadata and annotation values are indented 2 and 4 spaces deeper (sigs.k8s.io/yaml).
	fieldIndent := metadataIndentStr + "  "
	valueIndent := metadataIndentStr + "    "
	resourcePolicyBlock := fmt.Sprintf(
		"%s{{- if .Values.crd.keep }}\n%s\"helm.sh/resource-policy\": keep\n%s{{- end }}",
		valueIndent, valueIndent, valueIndent)

	if insertAfter < 0 {
		// No annotations section exists, so add one right after metadata:
		insertAfter = metadataLine
		resourcePolicyBlock = fieldIndent + common.YamlKeyAnnotations + "\n" + resourcePolicyBlock
	} else {
		// crd.keep is the only source of the policy: drop a copy the source CRD already sets, which
		// would otherwise render a duplicate key and keep the policy even when crd.keep is false.
		lines = removeAnnotationEntry(lines, insertAfter, len(valueIndent), crdResourcePolicyKey)
	}

	result := make([]string, 0, len(lines)+4)
	result = append(result, lines[:insertAfter+1]...)
	result = append(result, resourcePolicyBlock)
	result = append(result, lines[insertAfter+1:]...)
	return strings.Join(result, "\n")
}

const crdResourcePolicyKey = "helm.sh/resource-policy"

// removeAnnotationEntry removes the key entry (plain or quoted) from the annotations block whose
// header is at headerIndex, along with any continuation lines of its value. entryIndent is the
// indentation of the block's entries; deeper lines belong to a value and are never matched.
func removeAnnotationEntry(lines []string, headerIndex, entryIndent int, key string) []string {
	keyForms := []string{key + ":", `"` + key + `":`, "'" + key + "':"}
	result := make([]string, 0, len(lines))
	result = append(result, lines[:headerIndex+1]...)

	i := headerIndex + 1
	for ; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		_, indent := LeadingWhitespace(lines[i])
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && indent < entryIndent {
			break
		}
		isEntry := indent == entryIndent && slices.ContainsFunc(keyForms, func(form string) bool {
			return trimmed == form || strings.HasPrefix(trimmed, form+" ")
		})
		if !isEntry {
			result = append(result, lines[i])
			continue
		}
		// Skip the entry and any lines of its value, which are indented deeper than the entry.
		for i+1 < len(lines) {
			next := lines[i+1]
			_, nextIndent := LeadingWhitespace(next)
			if strings.TrimSpace(next) != "" && nextIndent <= entryIndent {
				break
			}
			i++
		}
	}
	return append(result, lines[i:]...)
}

// MakeWebhookAnnotationsConditional makes cert-manager annotations conditional on .Values.certManager.enabled.
func MakeWebhookAnnotationsConditional(yamlContent string) string {
	// Find cert-manager.io/inject-ca-from annotation and make it conditional
	if !strings.Contains(yamlContent, "cert-manager.io/inject-ca-from") {
		return yamlContent
	}
	certManagerPattern := regexp.MustCompile(`(\s+)cert-manager\.io/inject-ca-from:\s*[^\n]+`)
	// Replace the cert-manager annotation with conditional wrapper
	yamlContent = certManagerPattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
		// Extract the indentation
		indentMatch := regexp.MustCompile(`^(\s+)`).FindStringSubmatch(match)
		indent := ""
		if len(indentMatch) > 1 {
			indent = indentMatch[1]
		}
		// Extract the annotation line with proper indentation
		annotationLine := strings.TrimSpace(match)
		return fmt.Sprintf("%s{{- if .Values.certManager.enabled }}\n%s%s\n%s{{- end }}",
			indent, indent, annotationLine, indent)
	})
	return yamlContent
}
