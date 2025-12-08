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

package kustomize

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	kindNamespace          = "Namespace"
	kindCertificate        = "Certificate"
	kindService            = "Service"
	kindServiceAccount     = "ServiceAccount"
	kindRole               = "Role"
	kindClusterRole        = "ClusterRole"
	kindRoleBinding        = "RoleBinding"
	kindClusterRoleBinding = "ClusterRoleBinding"
	kindServiceMonitor     = "ServiceMonitor"
	kindIssuer             = "Issuer"
	kindValidatingWebhook  = "ValidatingWebhookConfiguration"
	kindMutatingWebhook    = "MutatingWebhookConfiguration"
	kindDeployment         = "Deployment"

	// API versions
	apiVersionCertManager = "cert-manager.io/v1"
	apiVersionMonitoring  = "monitoring.coreos.com/v1"

	chartNameTemplate = "chart.name"
)

// HelmTemplater handles converting YAML content to Helm templates
type HelmTemplater struct {
	projectName string
}

// NewHelmTemplater creates a new Helm templater
func NewHelmTemplater(projectName string) *HelmTemplater {
	return &HelmTemplater{
		projectName: projectName,
	}
}

// ApplyHelmSubstitutions converts YAML content to use Helm template syntax
func (t *HelmTemplater) ApplyHelmSubstitutions(yamlContent string, resource *unstructured.Unstructured) string {
	// Apply conditional wrappers first
	yamlContent = t.addConditionalWrappers(yamlContent, resource)

	// Apply general project name substitutions
	yamlContent = t.substituteProjectNames(yamlContent, resource)

	// Apply namespace substitutions
	yamlContent = t.substituteNamespace(yamlContent, resource)

	// Apply cert-manager and webhook-specific templating AFTER other substitutions
	yamlContent = t.substituteCertManagerReferences(yamlContent, resource)

	// Apply labels and annotations from Helm chart
	yamlContent = t.addHelmLabelsAndAnnotations(yamlContent, resource)

	// Apply resource-specific substitutions
	yamlContent = t.substituteRBACValues(yamlContent)

	// Apply deployment-specific templating
	if resource.GetKind() == kindDeployment {
		yamlContent = t.templateDeploymentFields(yamlContent)

		// Apply conditional logic for cert-manager related fields in deployments
		yamlContent = t.makeContainerArgsConditional(yamlContent)
		yamlContent = t.makeWebhookVolumeMountsConditional(yamlContent)
		yamlContent = t.makeWebhookVolumesConditional(yamlContent)
		yamlContent = t.makeMetricsVolumeMountsConditional(yamlContent)
		yamlContent = t.makeMetricsVolumesConditional(yamlContent)
	}

	// Apply port templating for Services and Deployments
	if resource.GetKind() == kindService || resource.GetKind() == kindDeployment {
		yamlContent = t.templatePorts(yamlContent, resource)
	}

	// Final tidy-up: avoid accidental blank lines after Helm if-block starts
	// Some replacements may introduce an empty line between a `{{- if ... }}`
	// and the following content; collapse that to ensure consistent formatting.
	yamlContent = t.collapseBlankLineAfterIf(yamlContent)

	return yamlContent
}

// substituteProjectNames keeps original YAML as much as possible - only add Helm templating
func (t *HelmTemplater) substituteProjectNames(yamlContent string, _ *unstructured.Unstructured) string {
	return yamlContent
}

// substituteNamespace replaces hardcoded namespace references with Release.Namespace
func (t *HelmTemplater) substituteNamespace(yamlContent string, resource *unstructured.Unstructured) string {
	hardcodedNamespace := t.projectName + "-system"
	namespaceTemplate := "{{ .Release.Namespace }}"

	// Replace hardcoded namespace references everywhere, including in the Namespace resource
	// so that metadata.name becomes the Helm release namespace.
	yamlContent = strings.ReplaceAll(yamlContent, hardcodedNamespace, namespaceTemplate)

	// Replace service DNS name placeholders in certificates
	if resource.GetKind() == kindCertificate {
		yamlContent = t.substituteCertificateDNSNames(yamlContent, resource)
	}

	return yamlContent
}

// substituteCertificateDNSNames replaces hardcoded DNS names in certificates with proper service templates
func (t *HelmTemplater) substituteCertificateDNSNames(yamlContent string, resource *unstructured.Unstructured) string {
	name := resource.GetName()

	// Replace service names with templated ones based on certificate type
	if strings.Contains(name, "metrics-cert") || strings.Contains(name, "metrics") {
		// Metrics certificates should point to metrics service
		// Use chart.name based service naming for consistency
		metricsServiceTemplate := "{{ include \"chart.serviceName\" " +
			"(dict \"suffix\" \"controller-manager-metrics-service\" \"context\" .) }}"
		metricsServiceFQDN := metricsServiceTemplate + ".{{ include \"chart.namespaceName\" . }}.svc"
		metricsServiceFQDNCluster := metricsServiceTemplate +
			".{{ include \"chart.namespaceName\" . }}.svc.cluster.local"

		// Replace placeholders
		yamlContent = strings.ReplaceAll(yamlContent, "SERVICE_NAME.SERVICE_NAMESPACE.svc", metricsServiceFQDN)
		yamlContent = strings.ReplaceAll(yamlContent,
			"SERVICE_NAME.SERVICE_NAMESPACE.svc.cluster.local", metricsServiceFQDNCluster)

		// Also replace hardcoded service names
		hardcodedMetricsService := t.projectName + "-controller-manager-metrics-service"
		yamlContent = strings.ReplaceAll(yamlContent, hardcodedMetricsService, metricsServiceTemplate)
	}

	// Replace hardcoded issuer reference with templated one
	hardcodedIssuer := t.projectName + "-selfsigned-issuer"
	templatedIssuer := "{{ include \"" + chartNameTemplate + "\" . }}-selfsigned-issuer"
	yamlContent = strings.ReplaceAll(yamlContent, hardcodedIssuer, templatedIssuer)

	return yamlContent
}

// substituteCertManagerReferences applies cert-manager and webhook-specific template substitutions
func (t *HelmTemplater) substituteCertManagerReferences(yamlContent string, _ *unstructured.Unstructured) string {
	return yamlContent
}

// addHelmLabelsAndAnnotations replaces kustomize managed-by labels with Helm equivalents
func (t *HelmTemplater) addHelmLabelsAndAnnotations(yamlContent string, _ *unstructured.Unstructured) string {
	// Replace app.kubernetes.io/managed-by: kustomize with Helm template
	// Use regex to handle different whitespace patterns
	managedByRegex := regexp.MustCompile(`(\s*)app\.kubernetes\.io/managed-by:\s+kustomize`)
	yamlContent = managedByRegex.ReplaceAllString(yamlContent, "${1}app.kubernetes.io/managed-by: {{ .Release.Service }}")

	return yamlContent
}

// substituteRBACValues applies RBAC-specific template substitutions
func (t *HelmTemplater) substituteRBACValues(yamlContent string) string {
	return yamlContent
}

// templateDeploymentFields converts deployment-specific fields to Helm templates
func (t *HelmTemplater) templateDeploymentFields(yamlContent string) string {
	// Template configuration fields
	yamlContent = t.templateImageReference(yamlContent)
	yamlContent = t.templateEnvironmentVariables(yamlContent)
	yamlContent = t.templateImagePullSecrets(yamlContent)
	yamlContent = t.templatePodSecurityContext(yamlContent)
	yamlContent = t.templateContainerSecurityContext(yamlContent)
	yamlContent = t.templateResources(yamlContent)
	yamlContent = t.templateSecurityContexts(yamlContent)
	yamlContent = t.templateVolumeMounts(yamlContent)
	yamlContent = t.templateVolumes(yamlContent)
	yamlContent = t.templateControllerManagerArgs(yamlContent)
	yamlContent = t.templateBasicWithStatement(
		yamlContent,
		"nodeSelector",
		"spec.template.spec",
		".Values.manager.nodeSelector",
	)
	yamlContent = t.templateBasicWithStatement(
		yamlContent,
		"affinity",
		"spec.template.spec",
		".Values.manager.affinity",
	)
	yamlContent = t.templateBasicWithStatement(
		yamlContent,
		"tolerations",
		"spec.template.spec",
		".Values.manager.tolerations",
	)

	return yamlContent
}

// templateEnvironmentVariables exposes environment variables via values.yaml
func (t *HelmTemplater) templateEnvironmentVariables(yamlContent string) string {
	if !strings.Contains(yamlContent, "name: manager") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) != "env:" {
			continue
		}

		indentStr, indentLen := leadingWhitespace(lines[i])
		end := i + 1
		for ; end < len(lines); end++ {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed == "" {
				break
			}
			lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
			if lineIndent < indentLen {
				break
			}
			if lineIndent == indentLen && !strings.HasPrefix(trimmed, "-") {
				break
			}
		}

		if i+1 < len(lines) && strings.Contains(lines[i+1], ".Values.manager.env") {
			return yamlContent
		}

		childIndent := indentStr + "  "
		childIndentWidth := strconv.Itoa(len(childIndent))

		block := []string{
			indentStr + "env:",
			childIndent + "{{- if .Values.manager.env }}",
			childIndent + "{{- toYaml .Values.manager.env | nindent " + childIndentWidth + " }}",
			childIndent + "{{- else }}",
			childIndent + "[]",
			childIndent + "{{- end }}",
		}

		newLines := append([]string{}, lines[:i]...)
		newLines = append(newLines, block...)
		newLines = append(newLines, lines[end:]...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

// templateResources converts resource sections to Helm templates
func (t *HelmTemplater) templateResources(yamlContent string) string {
	if !strings.Contains(yamlContent, "name: manager") || !strings.Contains(yamlContent, "resources:") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) != "resources:" {
			continue
		}

		indentStr, indentLen := leadingWhitespace(lines[i])
		end := i + 1
		for ; end < len(lines); end++ {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed == "" {
				break
			}
			lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
			if lineIndent < indentLen {
				break
			}
			// stop at same-level keys that are not part of the resources mapping
			if lineIndent == indentLen && !strings.Contains(trimmed, ":") {
				break
			}
			if lineIndent == indentLen && strings.HasSuffix(trimmed, ":") {
				break
			}
		}

		if i+1 < len(lines) && strings.Contains(lines[i+1], ".Values.manager.resources") {
			return yamlContent
		}

		childIndent := indentStr + "  "
		childIndentWidth := strconv.Itoa(len(childIndent))

		block := []string{
			indentStr + "resources:",
			childIndent + "{{- if .Values.manager.resources }}",
			childIndent + "{{- toYaml .Values.manager.resources | nindent " + childIndentWidth + " }}",
			childIndent + "{{- else }}",
			childIndent + "{}",
			childIndent + "{{- end }}",
		}

		newLines := append([]string{}, lines[:i]...)
		newLines = append(newLines, block...)
		newLines = append(newLines, lines[end:]...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

// templateSecurityContexts preserves security contexts from kustomize output
func (t *HelmTemplater) templateSecurityContexts(yamlContent string) string {
	// Security contexts are preserved as-is from the kustomize output to maintain
	// the exact security configuration without interfering with other container fields
	return yamlContent
}

// templateVolumeMounts converts volumeMounts sections to keep them as-is since they're webhook-specific
func (t *HelmTemplater) templateVolumeMounts(yamlContent string) string {
	// For webhook volumeMounts, we keep them as-is since they're required for webhook functionality
	// They will be conditionally included based on webhook configuration
	return yamlContent
}

// templateVolumes converts volumes sections to keep them as-is since they're webhook-specific
func (t *HelmTemplater) templateVolumes(yamlContent string) string {
	// For webhook volumes, we keep them as-is since they're required for webhook functionality
	// They will be conditionally included based on webhook configuration
	return yamlContent
}

// templateImagePullSecrets exposes imagePullSecrets via values.yaml
func (t *HelmTemplater) templateImagePullSecrets(yamlContent string) string {
	if !strings.Contains(yamlContent, "imagePullSecrets:") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	for i := range lines {
		// Use prefix to allow `imagePullSecrets: []` to be preserved
		if !strings.HasPrefix(strings.TrimSpace(lines[i]), "imagePullSecrets:") {
			continue
		}
		indentStr, indentLen := leadingWhitespace(lines[i])
		end := i + 1
		for ; end < len(lines); end++ {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed == "" {
				break
			}
			lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
			if lineIndent < indentLen {
				break
			}
			if lineIndent == indentLen && !strings.HasPrefix(trimmed, "-") {
				break
			}
		}

		if i+1 < len(lines) && strings.Contains(lines[i+1], ".Values.manager.imagePullSecrets") {
			return yamlContent
		}

		childIndent := indentStr + "  "
		childIndentWidth := strconv.Itoa(len(childIndent))

		block := []string{
			indentStr + "{{- if .Values.manager.imagePullSecrets }}",
			indentStr + "imagePullSecrets:",
			childIndent + "{{- toYaml .Values.manager.imagePullSecrets | nindent " + childIndentWidth + " }}",
			indentStr + "{{- end }}",
		}

		newLines := append([]string{}, lines[:i]...)
		newLines = append(newLines, block...)
		newLines = append(newLines, lines[end:]...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

// templatePodSecurityContext exposes podSecurityContext via values.yaml
func (t *HelmTemplater) templatePodSecurityContext(yamlContent string) string {
	if !strings.Contains(yamlContent, "securityContext:") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) != "securityContext:" {
			continue
		}

		indentStr, indentLen := leadingWhitespace(lines[i])
		end := i + 1
		for ; end < len(lines); end++ {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed == "" {
				break
			}
			lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
			if lineIndent <= indentLen {
				break
			}
		}

		if end >= len(lines) {
			break
		}

		if !strings.HasPrefix(strings.TrimSpace(lines[end]), "serviceAccountName:") {
			continue
		}

		if i+1 < len(lines) && strings.Contains(lines[i+1], ".Values.manager.podSecurityContext") {
			return yamlContent
		}

		childIndent := indentStr + "  "
		childIndentWidth := strconv.Itoa(len(childIndent))

		block := []string{
			indentStr + "securityContext:",
			childIndent + "{{- if .Values.manager.podSecurityContext }}",
			childIndent + "{{- toYaml .Values.manager.podSecurityContext | nindent " + childIndentWidth + " }}",
			childIndent + "{{- else }}",
			childIndent + "{}",
			childIndent + "{{- end }}",
		}

		newLines := append([]string{}, lines[:i]...)
		newLines = append(newLines, block...)
		newLines = append(newLines, lines[end:]...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

// templateContainerSecurityContext exposes container securityContext via values.yaml
func (t *HelmTemplater) templateContainerSecurityContext(yamlContent string) string {
	if !strings.Contains(yamlContent, "name: manager") || !strings.Contains(yamlContent, "securityContext:") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) != "securityContext:" {
			continue
		}

		indentStr, indentLen := leadingWhitespace(lines[i])
		end := i + 1
		for ; end < len(lines); end++ {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed == "" {
				break
			}
			lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
			if lineIndent <= indentLen {
				break
			}
		}

		if end >= len(lines) {
			break
		}

		if strings.HasPrefix(strings.TrimSpace(lines[end]), "serviceAccountName:") {
			continue
		}

		lookAheadEnd := min(end+5, len(lines))
		joined := strings.Join(lines[i:lookAheadEnd], "\n")
		if strings.Contains(joined, ".Values.manager.securityContext") {
			return yamlContent
		}

		childIndent := indentStr + "  "
		childIndentWidth := strconv.Itoa(len(childIndent))

		block := []string{
			indentStr + "securityContext:",
			childIndent + "{{- if .Values.manager.securityContext }}",
			childIndent + "{{- toYaml .Values.manager.securityContext | nindent " + childIndentWidth + " }}",
			childIndent + "{{- else }}",
			childIndent + "{}",
			childIndent + "{{- end }}",
		}

		newLines := append([]string{}, lines[:i]...)
		newLines = append(newLines, block...)
		newLines = append(newLines, lines[end:]...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

func leadingWhitespace(line string) (string, int) {
	trimmed := strings.TrimLeft(line, " \t")
	indentLen := len(line) - len(trimmed)
	return line[:indentLen], indentLen
}

// templateControllerManagerArgs exposes controller manager args via values.yaml while keeping core defaults
func (t *HelmTemplater) templateControllerManagerArgs(yamlContent string) string {
	if !strings.Contains(yamlContent, "name: manager") {
		return yamlContent
	}

	argsPattern := regexp.MustCompile(`(?m)([ \t]+)args:\n((?:[ \t]+-.*\n)+)`)
	loc := argsPattern.FindStringSubmatchIndex(yamlContent)
	if loc == nil {
		return yamlContent
	}

	match := yamlContent[loc[0]:loc[1]]
	if strings.Contains(match, ".Values.manager.args") {
		return yamlContent
	}

	indent := yamlContent[loc[2]:loc[3]]
	itemsBlock := yamlContent[loc[4]:loc[5]]

	itemIndent := indent + "  "
	lines := strings.Split(itemsBlock, "\n")
	var (
		metricsLine    string
		metricsIndent  string
		healthLine     string
		preservedLines []string
	)

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if itemIndent == indent+"  " {
			if idx := strings.Index(line, "-"); idx > 0 {
				itemIndent = line[:idx]
			}
		}

		switch {
		case strings.Contains(trimmed, "--metrics-bind-address"):
			metricsLine = line
			if idx := strings.Index(line, "-"); idx > 0 {
				metricsIndent = line[:idx]
			}
		case strings.Contains(trimmed, "--health-probe-bind-address"):
			healthLine = line
		case strings.Contains(trimmed, "--webhook-cert-path"),
			strings.Contains(trimmed, "--metrics-cert-path"):
			preservedLines = append(preservedLines, line)
		default:
			// Remaining args will be handled through values.yaml
		}
	}

	var builder strings.Builder
	builder.WriteString(indent)
	builder.WriteString("args:\n")

	if metricsLine != "" {
		if metricsIndent == "" {
			metricsIndent = itemIndent
		}
		builder.WriteString(metricsIndent)
		builder.WriteString("{{- if .Values.metrics.enable }}\n")
		builder.WriteString(metricsLine)
		builder.WriteString("\n")
		builder.WriteString(metricsIndent)
		builder.WriteString("{{- else }}\n")
		builder.WriteString(metricsIndent)
		builder.WriteString("# Bind to :0 to disable the controller-runtime managed metrics server\n")
		builder.WriteString(metricsIndent)
		builder.WriteString("- --metrics-bind-address=0\n")
		builder.WriteString(metricsIndent)
		builder.WriteString("{{- end }}\n")
	}
	if healthLine != "" {
		builder.WriteString(healthLine)
		builder.WriteString("\n")
	}

	builder.WriteString(itemIndent)
	builder.WriteString("{{- range .Values.manager.args }}\n")
	builder.WriteString(itemIndent)
	builder.WriteString("- {{ . }}\n")
	builder.WriteString(itemIndent)
	builder.WriteString("{{- end }}\n")

	for _, line := range preservedLines {
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	newBlock := strings.TrimRight(builder.String(), "\n") + "\n"

	return yamlContent[:loc[0]] + newBlock + yamlContent[loc[1]:]
}

// templateImageReference converts hardcoded image references to Helm templates
func (t *HelmTemplater) templateImageReference(yamlContent string) string {
	if !strings.Contains(yamlContent, "name: manager") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "image:") {
			continue
		}

		if strings.Contains(lines[i], ".Values.manager.image.repository") {
			return yamlContent
		}

		indentStr, indentLen := leadingWhitespace(lines[i])

		end := i + 1
		for ; end < len(lines); end++ {
			nextTrimmed := strings.TrimSpace(lines[end])
			if nextTrimmed == "" {
				break
			}
			lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
			if lineIndent <= indentLen {
				break
			}
			// Stop when we reach a sibling key like env:, args:, etc.
			if lineIndent == indentLen+2 && strings.HasSuffix(nextTrimmed, ":") {
				if strings.Contains(nextTrimmed, "imagePullPolicy") {
					continue
				}
				break
			}
		}

		// Remove any existing imagePullPolicy line inside the block
		blockLines := lines[i+1 : end]
		filtered := make([]string, 0, len(blockLines))
		for _, line := range blockLines {
			if strings.Contains(strings.TrimSpace(line), "imagePullPolicy") {
				continue
			}
			filtered = append(filtered, line)
		}
		lines = append(lines[:i+1], append(filtered, lines[end:]...)...)
		end = i + 1 + len(filtered)

		imageLine := indentStr + "image: \"{{ .Values.manager.image.repository }}:{{ .Values.manager.image.tag }}\""
		pullPolicyLine := indentStr + "imagePullPolicy: {{ .Values.manager.image.pullPolicy }}"

		remainder := lines[end:]
		if len(remainder) > 0 && strings.HasPrefix(strings.TrimSpace(remainder[0]), "imagePullPolicy:") {
			remainder = remainder[1:]
		}

		newLines := append([]string{}, lines[:i]...)
		newLines = append(newLines, imageLine, pullPolicyLine)
		newLines = append(newLines, remainder...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

func (t *HelmTemplater) templateBasicWithStatement(
	yamlContent string,
	key string,
	parentKey string,
	valuePath string,
) string {
	lines := strings.Split(yamlContent, "\n")
	yamlKey := fmt.Sprintf("%s:", key)

	var start, end int
	var indentLen int
	if !strings.Contains(yamlContent, yamlKey) {
		// Find parent block start if the key is missing
		pKeyParts := strings.Split(parentKey, ".")
		pKeyIdx := 0
		pKeyInit := false
		currIndent := 0
		for i := range len(lines) {
			_, lineIndent := leadingWhitespace(lines[i])
			if pKeyInit && lineIndent <= currIndent {
				return yamlContent
			}
			if !strings.HasPrefix(strings.TrimSpace(lines[i]), pKeyParts[pKeyIdx]) {
				continue
			}

			// Parent key part found
			pKeyIdx++
			pKeyInit = true
			if pKeyIdx >= len(pKeyParts) {
				start = i + 1
				end = start
				break
			}
		}
		_, indentLen = leadingWhitespace(lines[start])
	} else {
		// Find the existing block
		for i := range len(lines) {
			if !strings.HasPrefix(strings.TrimSpace(lines[i]), key) {
				continue
			}
			start = i
			end = i + 1
			trimmed := strings.TrimSpace(lines[i])
			if len(trimmed) == len(yamlKey) {
				_, indentLenSearch := leadingWhitespace(lines[i])
				for j := end; j < len(lines); j++ {
					_, indentLenLine := leadingWhitespace(lines[j])
					if indentLenLine <= indentLenSearch {
						end = j
						break
					}
				}
			}
		}
		_, indentLen = leadingWhitespace(lines[start])
	}

	indentStr := strings.Repeat(" ", indentLen)

	var builder strings.Builder
	builder.WriteString(indentStr)
	builder.WriteString("{{- with ")
	builder.WriteString(valuePath)
	builder.WriteString(" }}\n")
	builder.WriteString(indentStr)
	builder.WriteString(yamlKey)
	builder.WriteString(" {{ toYaml . | nindent ")
	builder.WriteString(strconv.Itoa(indentLen + 4))
	builder.WriteString(" }}\n")
	builder.WriteString(indentStr)
	builder.WriteString("{{- end }}\n")

	newBlock := strings.TrimRight(builder.String(), "\n")

	newLines := append([]string{}, lines[:start]...)
	newLines = append(newLines, strings.Split(newBlock, "\n")...)
	newLines = append(newLines, lines[end:]...)
	return strings.Join(newLines, "\n")
}

// makeWebhookAnnotationsConditional makes only cert-manager annotations conditional, not the entire webhook
func (t *HelmTemplater) makeWebhookAnnotationsConditional(yamlContent string) string {
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

// makeContainerArgsConditional makes webhook-cert-path and metrics-cert-path args conditional on certManager.enable
func (t *HelmTemplater) makeContainerArgsConditional(yamlContent string) string {
	// Make webhook-cert-path arg conditional on certManager.enable
	if strings.Contains(yamlContent, "--webhook-cert-path") {
		// Match only spaces/tabs for indent to avoid consuming the newline
		webhookArgPattern := regexp.MustCompile(`([ \t]+)-\s*--webhook-cert-path=[^\n]*`)
		yamlContent = webhookArgPattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
			indentMatch := regexp.MustCompile(`^(\s+)`).FindStringSubmatch(match)
			indent := ""
			if len(indentMatch) > 1 {
				indent = indentMatch[1]
			}

			argLine := strings.TrimSpace(match)
			return fmt.Sprintf("%s{{- if .Values.certManager.enable }}\n%s%s\n%s{{- end }}",
				indent, indent, argLine, indent)
		})
	}

	// Make metrics-cert-path arg conditional on certManager.enable AND metrics.enable
	if strings.Contains(yamlContent, "--metrics-cert-path") {
		// Match only spaces/tabs for indent to avoid consuming the newline
		metricsArgPattern := regexp.MustCompile(`([ \t]+)-\s*--metrics-cert-path=[^\n]*`)
		yamlContent = metricsArgPattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
			indentMatch := regexp.MustCompile(`^(\s+)`).FindStringSubmatch(match)
			indent := ""
			if len(indentMatch) > 1 {
				indent = indentMatch[1]
			}

			argLine := strings.TrimSpace(match)
			return fmt.Sprintf("%s{{- if and .Values.certManager.enable .Values.metrics.enable }}\n%s%s\n%s{{- end }}",
				indent, indent, argLine, indent)
		})
	}

	return yamlContent
}

func makeYamlContent(match string) string {
	lines := strings.Split(match, "\n")
	if len(lines) > 0 {
		indent := ""
		if len(lines[0]) > 0 && lines[0][0] == ' ' {
			// Count leading spaces
			for _, char := range lines[0] {
				if char == ' ' {
					indent += " "
				} else {
					break
				}
			}
		}

		// Reconstruct the block with conditional wrapper
		var result strings.Builder
		result.WriteString(fmt.Sprintf("%s{{- if .Values.certManager.enable }}\n", indent))
		for _, line := range lines {
			result.WriteString(line + "\n")
		}
		result.WriteString(fmt.Sprintf("%s{{- end }}", indent))
		return result.String()
	}
	return match
}

// makeWebhookVolumesConditional makes webhook volumes conditional on certManager.enable
func (t *HelmTemplater) makeWebhookVolumesConditional(yamlContent string) string {
	// Make webhook volumes conditional on certManager.enable
	if strings.Contains(yamlContent, "webhook-certs") && strings.Contains(yamlContent, "secretName: webhook-server-cert") {
		// Match only spaces/tabs for indent to avoid consuming the newline
		volumePattern := regexp.MustCompile(`([ \t]+)-\s*name:\s*webhook-certs[\s\S]*?secretName:\s*webhook-server-cert`)
		yamlContent = volumePattern.ReplaceAllStringFunc(yamlContent, makeYamlContent)
	}

	return yamlContent
}

// makeWebhookVolumeMountsConditional makes webhook volumeMounts conditional on certManager.enable
func (t *HelmTemplater) makeWebhookVolumeMountsConditional(yamlContent string) string {
	// Make webhook volumeMounts conditional on certManager.enable
	webhookCertsPath := "/tmp/k8s-webhook-server/serving-certs"
	if strings.Contains(yamlContent, "webhook-certs") && strings.Contains(yamlContent, webhookCertsPath) {
		// Match only spaces/tabs for indent to avoid consuming the newline
		mountPattern := regexp.MustCompile(
			`([ \t]+)-\s*mountPath:\s*/tmp/k8s-webhook-server/serving-certs[\s\S]*?readOnly:\s*true`)
		yamlContent = mountPattern.ReplaceAllStringFunc(yamlContent, makeYamlContent)
	}

	return yamlContent
}

// makeMetricsVolumesConditional makes metrics volumes conditional on certManager.enable AND metrics.enable
func (t *HelmTemplater) makeMetricsVolumesConditional(yamlContent string) string {
	// Make metrics volumes conditional on certManager.enable AND metrics.enable
	if strings.Contains(yamlContent, "metrics-certs") && strings.Contains(yamlContent, "secretName: metrics-server-cert") {
		// Match only spaces/tabs for indent to avoid consuming the newline
		volumePattern := regexp.MustCompile(`([ \t]+)-\s*name:\s*metrics-certs[\s\S]*?secretName:\s*metrics-server-cert`)
		yamlContent = volumePattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
			lines := strings.Split(match, "\n")
			if len(lines) > 0 {
				indent := ""
				if len(lines[0]) > 0 && lines[0][0] == ' ' {
					// Count leading spaces
					for _, char := range lines[0] {
						if char == ' ' {
							indent += " "
						} else {
							break
						}
					}
				}

				// Reconstruct the block with conditional wrapper
				var result strings.Builder
				result.WriteString(fmt.Sprintf("%s{{- if and .Values.certManager.enable .Values.metrics.enable }}\n", indent))
				for _, line := range lines {
					result.WriteString(line + "\n")
				}
				result.WriteString(fmt.Sprintf("%s{{- end }}", indent))
				return result.String()
			}
			return match
		})
	}

	return yamlContent
}

// injectCommonLabels adds a Helm template snippet to append user-provided common labels
// (.Values.commonLabels) to every metadata.labels block while preserving indentation.
// It avoids duplicate insertion by checking for an existing snippet nearby.
// no common labels injection; labels come from kustomize manifests

// makeMetricsVolumeMountsConditional makes metrics volumeMounts conditional on certManager.enable AND metrics.enable
func (t *HelmTemplater) makeMetricsVolumeMountsConditional(yamlContent string) string {
	// Make metrics volumeMounts conditional on certManager.enable AND metrics.enable
	metricsCertsPath := "/tmp/k8s-metrics-server/metrics-certs"
	if strings.Contains(yamlContent, "metrics-certs") && strings.Contains(yamlContent, metricsCertsPath) {
		// Match only spaces/tabs for indent to avoid consuming the newline
		mountPattern := regexp.MustCompile(
			`([ \t]+)-\s*mountPath:\s*/tmp/k8s-metrics-server/metrics-certs[\s\S]*?readOnly:\s*true`)
		yamlContent = mountPattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
			lines := strings.Split(match, "\n")
			if len(lines) > 0 {
				indent := ""
				if len(lines[0]) > 0 && lines[0][0] == ' ' {
					// Count leading spaces
					for _, char := range lines[0] {
						if char == ' ' {
							indent += " "
						} else {
							break
						}
					}
				}

				// Reconstruct the block with conditional wrapper
				var result strings.Builder
				result.WriteString(fmt.Sprintf("%s{{- if and .Values.certManager.enable .Values.metrics.enable }}\n", indent))
				for _, line := range lines {
					result.WriteString(line + "\n")
				}
				result.WriteString(fmt.Sprintf("%s{{- end }}", indent))
				return result.String()
			}
			return match
		})
	}

	return yamlContent
}

// addConditionalWrappers adds conditional Helm logic based on resource type
func (t *HelmTemplater) addConditionalWrappers(yamlContent string, resource *unstructured.Unstructured) string {
	kind := resource.GetKind()
	apiVersion := resource.GetAPIVersion()
	name := resource.GetName()

	switch {
	case kind == kindNamespace:
		return ""
	case kind == "CustomResourceDefinition":
		// CRDs need crd.enable condition
		return fmt.Sprintf("{{- if .Values.crd.enable }}\n%s{{- end }}\n", yamlContent)
	case kind == kindCertificate && apiVersion == apiVersionCertManager:
		// Handle different certificate types
		if strings.Contains(name, "metrics-cert") || strings.Contains(name, "metrics") {
			// Metrics certificates need both certManager and metrics enabled
			return fmt.Sprintf("{{- if and .Values.certManager.enable .Values.metrics.enable }}\n%s{{- end }}\n",
				yamlContent)
		}
		// Other certificates (webhook serving certs) only need certManager enabled
		return fmt.Sprintf("{{- if .Values.certManager.enable }}\n%s{{- end }}", yamlContent)
	case kind == kindIssuer && apiVersion == apiVersionCertManager:
		// All cert-manager issuers need certManager enabled
		return fmt.Sprintf("{{- if .Values.certManager.enable }}\n%s{{- end }}", yamlContent)
	case kind == kindServiceMonitor && apiVersion == apiVersionMonitoring:
		// ServiceMonitors need prometheus enabled
		return fmt.Sprintf("{{- if .Values.prometheus.enable }}\n%s{{- end }}", yamlContent)
	case kind == kindServiceAccount || kind == kindRole || kind == kindClusterRole ||
		kind == kindRoleBinding || kind == kindClusterRoleBinding:
		// Distinguish between essential RBAC and helper RBAC
		if strings.Contains(name, "admin-role") || strings.Contains(name, "editor-role") ||
			strings.Contains(name, "viewer-role") {
			// Helper RBAC roles (admin/editor/viewer) - convenience roles for CRD management
			return fmt.Sprintf("{{- if .Values.rbacHelpers.enable }}\n%s{{- end }}\n", yamlContent)
		}
		if strings.Contains(name, "metrics") {
			// Metrics RBAC depends on metrics being enabled
			return fmt.Sprintf("{{- if .Values.metrics.enable }}\n%s{{- end }}\n", yamlContent)
		}
		// Essential RBAC (controller-manager, leader-election, manager roles) - always enabled
		// These are required for the controller to function properly
		return yamlContent
	case kind == kindValidatingWebhook || kind == kindMutatingWebhook:
		// Webhook configurations should always exist if project has webhooks
		// Only the cert-manager annotations should be conditional
		return t.makeWebhookAnnotationsConditional(yamlContent)
	case kind == kindService:
		// Services need conditional logic based on their purpose
		if strings.Contains(name, "metrics") {
			// Metrics services need metrics enabled
			return fmt.Sprintf("{{- if .Values.metrics.enable }}\n%s{{- end }}\n", yamlContent)
		}
		// Other services (webhook service, etc.) don't need conditionals - they're essential
		return yamlContent
	default:
		// No conditional wrapper needed for other resources (Deployment, Namespace)
		return yamlContent
	}
}

// collapseBlankLineAfterIf removes a single empty line that may appear
// immediately after a Helm if directive line, e.g. `{{- if ... }}`.
// This keeps templates compact and matches expected formatting in tests.
func (t *HelmTemplater) collapseBlankLineAfterIf(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	if len(lines) == 0 {
		return yamlContent
	}
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		// If current line is an if, and next line is blank, skip the blank
		if strings.Contains(line, "{{- if ") {
			out = append(out, line)
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
				i++ // skip one blank line after if
			}
			continue
		}
		// If current line is blank, and next line is an end, skip the blank
		if strings.TrimSpace(line) == "" && i+1 < len(lines) && strings.Contains(lines[i+1], "{{- end }}") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// templatePorts replaces hardcoded port values with Helm template references
// This makes ports configurable via values.yaml under webhook.port and metrics.port
func (t *HelmTemplater) templatePorts(yamlContent string, resource *unstructured.Unstructured) string {
	resourceName := resource.GetName()

	// Determine if this is a webhook-related resource
	isWebhook := strings.Contains(resourceName, "webhook")

	// Determine if this is a metrics-related resource
	isMetrics := strings.Contains(resourceName, "metrics")

	// For Deployments, check for webhook ports in the content
	if resource.GetKind() == kindDeployment {
		// Check if this deployment has webhook-server ports
		if strings.Contains(yamlContent, "webhook-server") || strings.Contains(yamlContent, "name: webhook") {
			isWebhook = true
		}
	}

	// Template webhook ports (9443 by default)
	if isWebhook {
		// Replace containerPort: 9443 (or any value) for webhook-server with template
		if strings.Contains(yamlContent, "webhook-server") {
			yamlContent = regexp.MustCompile(`(?m)(\s*- )?containerPort:\s*\d+(\s*\n\s*name:\s*webhook-server)`).
				ReplaceAllString(yamlContent, "${1}containerPort: {{ .Values.webhook.port }}${2}")
		}

		// Replace targetPort: 9443 with webhook.port template
		yamlContent = regexp.MustCompile(`(\s*)targetPort:\s*9443`).
			ReplaceAllString(yamlContent, "${1}targetPort: {{ .Values.webhook.port }}")
	}

	// Template metrics ports (8443 by default)
	if isMetrics {
		// Replace port: 8443 with metrics.port template
		yamlContent = regexp.MustCompile(`(\s*)port:\s*8443`).
			ReplaceAllString(yamlContent, "${1}port: {{ .Values.metrics.port }}")

		// Replace targetPort: 8443 with metrics.port template
		yamlContent = regexp.MustCompile(`(\s*)targetPort:\s*8443`).
			ReplaceAllString(yamlContent, "${1}targetPort: {{ .Values.metrics.port }}")
	}

	// Template port-related arguments in Deployment
	if resource.GetKind() == kindDeployment {
		// Replace --metrics-bind-address=:8443 with templated version
		yamlContent = regexp.MustCompile(`--metrics-bind-address=:[0-9]+`).
			ReplaceAllString(yamlContent, "--metrics-bind-address=:{{ .Values.metrics.port }}")

		// Replace --webhook-port=9443 with templated version (if present)
		yamlContent = regexp.MustCompile(`--webhook-port=[0-9]+`).
			ReplaceAllString(yamlContent, "--webhook-port={{ .Values.webhook.port }}")
	}

	return yamlContent
}
