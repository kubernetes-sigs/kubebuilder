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
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// This file contains RBAC and ServiceAccount name/enable transformations:
//  - SubstituteRBACValues: Role and RoleBinding name templating
//  - TemplateServiceAccountNameInBindings: SA name in RoleBinding/ClusterRoleBinding subjects
//  - TemplateServiceAccountNameInDeployment: SA name in Deployment spec
//  - TemplateServiceAccount: ServiceAccount orchestration (labels+annotations, name, conditional)
//
// ServiceAccount label/annotation merging lives in labels.go with the rest of the metadata
// helpers.

// SubstituteRBACValues applies RBAC-specific template substitutions.
func SubstituteRBACValues(detectedPrefix, chartName, yamlContent string) string {
	return substituteRBACValues(detectedPrefix, chartName, yamlContent, nil)
}

// SubstituteRBACValuesWithManagerServiceAccount applies RBAC substitutions using the resolved manager SA.
func SubstituteRBACValuesWithManagerServiceAccount(
	detectedPrefix, chartName, yamlContent string, managerServiceAccount *unstructured.Unstructured,
) string {
	return substituteRBACValues(detectedPrefix, chartName, yamlContent, managerServiceAccount)
}

func substituteRBACValues(
	detectedPrefix, chartName, yamlContent string, managerServiceAccount *unstructured.Unstructured,
) string {
	roleRefBlockPattern := regexp.MustCompile(
		`(?s)(roleRef:\s*\n(?:\s+\w+:.*\n)*?)(\s+)name:\s+` +
			regexp.QuoteMeta(detectedPrefix) + `-manager-role`)
	yamlContent = roleRefBlockPattern.ReplaceAllString(
		yamlContent, `${1}${2}name: `+ResourceNameTemplate(chartName, "manager-role"))

	roleRefBlockPatternSimple := regexp.MustCompile(
		`(?s)(roleRef:\s*\n(?:\s+\w+:.*\n)*?)(\s+)name:\s+manager-role`)
	yamlContent = roleRefBlockPatternSimple.ReplaceAllString(
		yamlContent, `${1}${2}name: `+ResourceNameTemplate(chartName, "manager-role"))

	yamlContent = templateServiceAccountNameInBindings(detectedPrefix, chartName, yamlContent, managerServiceAccount)

	return yamlContent
}

// managerServiceAccountNames returns the names the manager ServiceAccount can have when templating
// runs: the default name with and without the prefix, its resourceName helper and, when the
// manager uses its own ServiceAccount, that name and its resourceName helper.
func managerServiceAccountNames(
	detectedPrefix, chartName string, managerServiceAccount *unstructured.Unstructured,
) []string {
	names := []string{
		ResourceNameTemplate(chartName, controllerManagerName),
		detectedPrefix + "-" + controllerManagerName,
		controllerManagerName,
	}
	if managerServiceAccount != nil {
		names = append(names, managerServiceAccount.GetName())
		if suffix := managerServiceAccountResourceSuffix(detectedPrefix, managerServiceAccount); suffix != "" {
			names = append(names, ResourceNameTemplate(chartName, suffix))
		}
	}
	return names
}

// managerServiceAccountResourceSuffix returns the resourceName dict suffix for a prefixed manager SA.
func managerServiceAccountResourceSuffix(
	detectedPrefix string, managerServiceAccount *unstructured.Unstructured,
) string {
	if managerServiceAccount == nil {
		return ""
	}
	if after, ok := strings.CutPrefix(managerServiceAccount.GetName(), detectedPrefix+"-"); ok && after != "" {
		return after
	}
	return ""
}

// TemplateServiceAccountNameInBindings templates SA name in RoleBinding/ClusterRoleBinding subjects.
func TemplateServiceAccountNameInBindings(detectedPrefix, chartName, yamlContent string) string {
	return templateServiceAccountNameInBindings(detectedPrefix, chartName, yamlContent, nil)
}

func templateServiceAccountNameInBindings(
	detectedPrefix, chartName, yamlContent string, managerServiceAccount *unstructured.Unstructured,
) string {
	lines := strings.Split(yamlContent, "\n")
	subjectsLine := slices.Index(lines, "subjects:")
	if subjectsLine < 0 {
		return yamlContent
	}

	managerNames := managerServiceAccountNames(detectedPrefix, chartName, managerServiceAccount)
	managerNamespace := ""
	if managerServiceAccount != nil {
		managerNamespace = managerServiceAccount.GetNamespace()
	}

	for start := subjectsLine + 1; start < len(lines); {
		trimmed := strings.TrimSpace(lines[start])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			start++
			continue
		}
		_, indent := LeadingWhitespace(lines[start])
		if !strings.HasPrefix(trimmed, "- ") {
			break
		}

		end := start + 1
		for end < len(lines) {
			trimmed = strings.TrimSpace(lines[end])
			_, nextIndent := LeadingWhitespace(lines[end])
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") && nextIndent <= indent {
				break
			}
			end++
		}
		templateServiceAccountSubject(lines[start:end], chartName, managerNames, managerNamespace)
		start = end
	}

	return strings.Join(lines, "\n")
}

// Keep substitutions within one subject because Helm expressions prevent YAML decoding here.
func templateServiceAccountSubject(lines []string, chartName string, managerNames []string, managerNamespace string) {
	var kind, name, namespace string
	nameLine := -1
	_, subjectIndent := LeadingWhitespace(lines[0])
	for i, line := range lines {
		field := strings.TrimSpace(line)
		if i == 0 {
			field = strings.TrimPrefix(field, "- ")
		} else if _, indent := LeadingWhitespace(line); indent != subjectIndent+2 {
			continue
		}
		key, value, found := strings.Cut(field, ":")
		if !found {
			continue
		}
		switch key {
		case "kind":
			kind = strings.TrimSpace(value)
		case "name":
			name = strings.TrimSpace(value)
			nameLine = i
		case "namespace":
			namespace = strings.TrimSpace(value)
		}
	}
	// Namespace substitution has already mapped the manager namespace to the release namespace.
	// The resolved manager ServiceAccount's own namespace is accepted as well.
	if kind != "ServiceAccount" || nameLine < 0 ||
		(namespace != "" && namespace != `""` && namespace != "null" && namespace != "{{ .Release.Namespace }}" &&
			(managerNamespace == "" || namespace != managerNamespace)) {
		return
	}
	if slices.Contains(managerNames, name) {
		colon := strings.IndexByte(lines[nameLine], ':')
		lines[nameLine] = lines[nameLine][:colon+1] + ` {{ include "` + chartName + `.serviceAccountName" . }}`
	}
}

// TemplateServiceAccountNameInDeployment templates spec.template.spec.serviceAccountName in a Deployment.
// A label or selector with the same key is left unchanged.
func TemplateServiceAccountNameInDeployment(detectedPrefix, chartName, yamlContent string) string {
	return templateServiceAccountNameInDeployment(detectedPrefix, chartName, yamlContent, nil)
}

func templateServiceAccountNameInDeployment(
	detectedPrefix, chartName, yamlContent string, managerServiceAccount *unstructured.Unstructured,
) string {
	lines := strings.Split(yamlContent, "\n")
	index := podSpecFieldIndex(lines, "serviceAccountName")
	if index < 0 {
		return yamlContent
	}
	_, value, _ := strings.Cut(strings.TrimSpace(lines[index]), ":")
	if !slices.Contains(managerServiceAccountNames(detectedPrefix, chartName, managerServiceAccount),
		strings.TrimSpace(value)) {
		return yamlContent
	}
	indent, _ := LeadingWhitespace(lines[index])
	lines[index] = indent + `serviceAccountName: {{ include "` + chartName + `.serviceAccountName" . }}`
	return strings.Join(lines, "\n")
}

// podSpecFieldIndex returns the index of the line that sets spec.template.spec.<field>, or -1.
func podSpecFieldIndex(lines []string, field string) int {
	type pathKey struct {
		indent int
		name   string
	}
	var path []pathKey
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "{{") {
			continue
		}
		_, indent := LeadingWhitespace(line)
		for len(path) > 0 && path[len(path)-1].indent >= indent {
			path = path[:len(path)-1]
		}
		name, _, found := strings.Cut(trimmed, ":")
		if !found || strings.HasPrefix(trimmed, "-") {
			// A list entry is never on the path to a pod spec field.
			path = append(path, pathKey{indent: indent, name: "-"})
			continue
		}
		if name == field && len(path) == 3 &&
			path[0].name == "spec" && path[1].name == "template" && path[2].name == "spec" {
			return i
		}
		path = append(path, pathKey{indent: indent, name: name})
	}
	return -1
}

// TemplateServiceAccount applies manager ServiceAccount-specific transformations.
func TemplateServiceAccount(detectedPrefix, chartName, yamlContent string) string {
	return templateServiceAccount(detectedPrefix, chartName, yamlContent, nil, nil)
}

// TemplateServiceAccountForResource applies manager ServiceAccount transforms for the resolved manager SA.
func TemplateServiceAccountForResource(
	detectedPrefix, chartName, yamlContent string,
	resource, managerServiceAccount *unstructured.Unstructured,
) string {
	return templateServiceAccount(detectedPrefix, chartName, yamlContent, resource, managerServiceAccount)
}

func templateServiceAccount(
	detectedPrefix, chartName, yamlContent string,
	resource, managerServiceAccount *unstructured.Unstructured,
) string {
	if !IsResolvedManagerServiceAccount(resource, managerServiceAccount) {
		return yamlContent
	}
	yamlContent = AddServiceAccountLabelsAndAnnotations(yamlContent)
	yamlContent = templateServiceAccountName(detectedPrefix, chartName, yamlContent, managerServiceAccount)
	yamlContent = WrapServiceAccountWithEnabledConditional(yamlContent)
	return yamlContent
}

// IsResolvedManagerServiceAccount reports whether resource is the manager ServiceAccount
// resolved during parsing. Falls back to the controller-manager name suffix when unset.
func IsResolvedManagerServiceAccount(resource, managerServiceAccount *unstructured.Unstructured) bool {
	if resource == nil {
		return false
	}
	if managerServiceAccount != nil {
		return resource.GetName() == managerServiceAccount.GetName() &&
			resource.GetNamespace() == managerServiceAccount.GetNamespace()
	}
	return IsManagerServiceAccount(resource)
}

// TemplateServiceAccountName replaces SA name with serviceAccountName helper.
func TemplateServiceAccountName(detectedPrefix, chartName, yamlContent string) string {
	return templateServiceAccountName(detectedPrefix, chartName, yamlContent, nil)
}

func templateServiceAccountName(
	detectedPrefix, chartName, yamlContent string, managerServiceAccount *unstructured.Unstructured,
) string {
	replacement := `${1}name: {{ include "` + chartName + `.serviceAccountName" . }}`

	// Handle name with prefix
	namePattern := regexp.MustCompile(
		`(?m)^(\s*)name:\s+` + regexp.QuoteMeta(detectedPrefix) + `-controller-manager\s*$`)
	yamlContent = namePattern.ReplaceAllString(yamlContent, replacement)

	// Handle name without prefix
	namePatternSimple := regexp.MustCompile(`(?m)^(\s*)name:\s+controller-manager\s*$`)
	yamlContent = namePatternSimple.ReplaceAllString(yamlContent, replacement)

	if managerServiceAccount != nil {
		customNamePattern := regexp.MustCompile(
			`(?m)^([ \t]{2})name: ` + regexp.QuoteMeta(managerServiceAccount.GetName()) + `[ \t]*$`)
		yamlContent = customNamePattern.ReplaceAllString(yamlContent, replacement)
	}

	return yamlContent
}

// WrapServiceAccountWithEnabledConditional wraps SA in serviceAccount.enabled conditional.
func WrapServiceAccountWithEnabledConditional(yamlContent string) string {
	// Ensure yamlContent ends with newline so {{- end }} is on its own line
	if !strings.HasSuffix(yamlContent, "\n") {
		yamlContent += "\n"
	}
	// Create the ServiceAccount only when serviceAccount.enabled is truthy. Plain truthiness matches
	// the serviceAccountName helper and every other section toggle in the chart.
	return "{{- if .Values.serviceAccount.enabled }}\n" + yamlContent + "{{- end }}\n"
}
