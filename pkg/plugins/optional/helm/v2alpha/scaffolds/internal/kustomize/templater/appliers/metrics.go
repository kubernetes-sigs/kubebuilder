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
	"regexp"
	"strings"
)

// TemplateServiceMonitor applies all ServiceMonitor-specific transformations.
func TemplateServiceMonitor(yamlContent string) string {
	yamlContent = regexp.MustCompile(`(\s*)port:\s*https`).
		ReplaceAllString(yamlContent, `${1}port: {{ if .Values.metrics.secure }}https{{ else }}http{{ end }}`)

	yamlContent = regexp.MustCompile(`(\s*)scheme:\s*https`).
		ReplaceAllString(yamlContent, `${1}scheme: {{ if .Values.metrics.secure }}https{{ else }}http{{ end }}`)

	// Make bearer token and TLS config conditional on metrics.secure
	yamlContent = MakeServiceMonitorBearerTokenConditional(yamlContent)
	// IMPORTANT: Cert-manager conditional must run BEFORE TLS wrapping
	// so it can process the raw kustomize output
	yamlContent = MakeServiceMonitorCertManagerConditional(yamlContent)
	yamlContent = MakeServiceMonitorTLSConditional(yamlContent)

	return yamlContent
}

// MakeServiceMonitorTLSConditional wraps ServiceMonitor tlsConfig fields with appropriate conditionals.
// Adds metrics.secure wrapper and cert-manager conditionals around cert fields when found.
func MakeServiceMonitorTLSConditional(yamlContent string) string {
	if !strings.Contains(yamlContent, "tlsConfig:") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	var result []string
	inTLSConfig := false
	tlsConfigIndent := 0
	var tlsBlock []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))

		if strings.HasPrefix(trimmed, "tlsConfig:") {
			inTLSConfig = true
			tlsConfigIndent = currentIndent
			tlsBlock = []string{line}
			continue
		}

		if inTLSConfig {
			// Stop when we hit a line with same or less indentation (not empty/comment)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") && currentIndent <= tlsConfigIndent {
				// Process and wrap the tlsConfig block
				result = append(result, wrapTLSConfigBlock(tlsBlock, tlsConfigIndent)...)
				inTLSConfig = false
				tlsBlock = nil
				result = append(result, line)
			} else {
				tlsBlock = append(tlsBlock, line)
			}
		} else {
			result = append(result, line)
		}

		if inTLSConfig && i == len(lines)-1 {
			// End of file - process remaining tlsConfig
			result = append(result, wrapTLSConfigBlock(tlsBlock, tlsConfigIndent)...)
		}
	}

	return strings.Join(result, "\n")
}

// wrapTLSConfigBlock wraps tlsConfig content with metrics.secure conditional.
// If kustomize output contains cert fields (ca/cert/keySecret), they are included as-is.
// If not, only insecureSkipVerify is included.
func wrapTLSConfigBlock(tlsBlock []string, baseIndent int) []string {
	if len(tlsBlock) == 0 {
		return tlsBlock
	}

	indentStr := strings.Repeat(" ", baseIndent)

	var wrapped []string
	wrapped = append(wrapped, indentStr+"{{- if .Values.metrics.secure }}")
	wrapped = append(wrapped, tlsBlock[0]) // "tlsConfig:" line

	// Add all tlsConfig content as-is (preserves certs if present in kustomize output)
	for i := 1; i < len(tlsBlock); i++ {
		wrapped = append(wrapped, tlsBlock[i])
	}

	wrapped = append(wrapped, indentStr+"{{- end }}")
	return wrapped
}

// MakeServiceMonitorBearerTokenConditional makes bearer token conditional on metrics.secure.
func MakeServiceMonitorBearerTokenConditional(yamlContent string) string {
	// Keep the dash outside the conditional - the list item always exists (with path/port/scheme)
	// Only the bearerTokenFile field itself is conditional on secure mode
	listItemPattern := regexp.MustCompile(`(?m)^(\s*)-\s+bearerTokenFile:\s*([^\n]+)`)
	yamlContent = listItemPattern.ReplaceAllString(yamlContent,
		`$1- {{- if .Values.metrics.secure }}`+"\n"+`$1  bearerTokenFile: $2`+"\n"+`$1  {{- end }}`)

	return yamlContent
}

// MakeServiceMonitorCertManagerConditional handles cert-manager conditional logic for ServiceMonitor.
// It wraps cert-manager specific fields with {{- if .Values.certManager.enable }} when default
// cert-manager secrets are detected. Custom certificate secrets are preserved as-is.
// If no certificate fields are found, it converts insecureSkipVerify: false to true.
func MakeServiceMonitorCertManagerConditional(yamlContent string) string {
	if !strings.Contains(yamlContent, "tlsConfig:") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	var result []string
	inTLSConfig := false
	tlsConfigIndent := 0
	var certFields []string
	var nonCertLines []string
	hasCertManagerFields := false
	hasDefaultCertManagerSecret := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))

		if strings.HasPrefix(trimmed, "tlsConfig:") {
			inTLSConfig = true
			tlsConfigIndent = currentIndent
			result = append(result, line)
			continue
		}

		if inTLSConfig {
			// Check if we've exited the tlsConfig block (line at same/less indent)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") && currentIndent <= tlsConfigIndent {
				// Process collected cert fields
				if hasCertManagerFields && hasDefaultCertManagerSecret {
					result = append(result, wrapCertManagerFields(certFields, nonCertLines, tlsConfigIndent)...)
				} else if len(certFields) == 0 && len(nonCertLines) > 0 {
					// No cert fields, just insecureSkipVerify: false
					result = append(result, convertInsecureSkipVerify(nonCertLines)...)
				} else {
					// Custom certs or other fields - preserve as-is
					result = append(result, nonCertLines...)
					result = append(result, certFields...)
				}
				inTLSConfig = false
				certFields = nil
				nonCertLines = nil
				hasCertManagerFields = false
				hasDefaultCertManagerSecret = false
				result = append(result, line)
				continue
			}

			// Check for cert-manager fields
			if strings.HasPrefix(trimmed, "ca:") || strings.HasPrefix(trimmed, "cert:") ||
				strings.HasPrefix(trimmed, "keySecret:") || strings.HasPrefix(trimmed, "serverName:") {
				hasCertManagerFields = true
			}

			// Collect cert-related fields (ca, cert, keySecret and their nested content)
			if strings.HasPrefix(trimmed, "ca:") || strings.HasPrefix(trimmed, "cert:") ||
				strings.HasPrefix(trimmed, "keySecret:") {
				// Start collecting this cert field block
				certFieldIndent := currentIndent
				certFields = append(certFields, line)
				// Collect all nested content
				for i+1 < len(lines) {
					i++
					nextLine := lines[i]
					nextTrimmed := strings.TrimSpace(nextLine)
					nextIndent := len(nextLine) - len(strings.TrimLeft(nextLine, " \t"))

					// Stop if we hit a line at same or less indentation (or a conditional)
					if nextTrimmed != "" && !strings.HasPrefix(nextTrimmed, "#") &&
						!strings.HasPrefix(nextTrimmed, "{{") && nextIndent <= certFieldIndent {
						i-- // Back up so the outer loop processes this line
						break
					}

					// Check for default cert-manager secret name
					if strings.Contains(nextTrimmed, "metrics-server-cert") {
						hasDefaultCertManagerSecret = true
					}

					certFields = append(certFields, nextLine)
				}
				continue
			}

			// Everything else goes to nonCertLines
			nonCertLines = append(nonCertLines, line)
		} else {
			result = append(result, line)
		}
	}

	// Handle end of file - if we're still in tlsConfig, process remaining fields
	if inTLSConfig {
		if hasCertManagerFields && hasDefaultCertManagerSecret {
			result = append(result, wrapCertManagerFields(certFields, nonCertLines, tlsConfigIndent)...)
		} else if len(certFields) == 0 && len(nonCertLines) > 0 {
			// No cert fields, just insecureSkipVerify: false
			result = append(result, convertInsecureSkipVerify(nonCertLines)...)
		} else {
			// Custom certs or other fields - preserve as-is
			result = append(result, nonCertLines...)
			result = append(result, certFields...)
		}
	}

	return strings.Join(result, "\n")
}

// wrapCertManagerFields wraps cert-manager certificate fields with the certManager.enable conditional.
func wrapCertManagerFields(certFields, nonCertLines []string, baseIndent int) []string {
	indentStr := strings.Repeat(" ", baseIndent+2)
	result := make([]string, 0, len(nonCertLines)+len(certFields)+4)

	// Add non-cert lines first (like serverName) but exclude insecureSkipVerify to avoid duplication
	for _, line := range nonCertLines {
		if !strings.Contains(line, "insecureSkipVerify:") {
			result = append(result, line)
		}
	}

	// Wrap cert fields with cert-manager conditional and template insecureSkipVerify
	result = append(result, indentStr+"{{- if .Values.certManager.enable }}")
	result = append(result, certFields...)
	result = append(result, indentStr+"insecureSkipVerify: false")
	result = append(result, indentStr+"{{- else }}")
	result = append(result, indentStr+"insecureSkipVerify: true")
	result = append(result, indentStr+"{{- end }}")

	return result
}

// convertInsecureSkipVerify converts insecureSkipVerify: false to true when no certs are present.
func convertInsecureSkipVerify(lines []string) []string {
	var result []string
	for _, line := range lines {
		if strings.Contains(line, "insecureSkipVerify:") && strings.Contains(line, "false") {
			result = append(result, strings.Replace(line, "false", "true", 1))
		} else {
			result = append(result, line)
		}
	}
	return result
}
