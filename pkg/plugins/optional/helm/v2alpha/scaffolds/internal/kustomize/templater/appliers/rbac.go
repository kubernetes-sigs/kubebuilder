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
	roleRefBlockPattern := regexp.MustCompile(
		`(?s)(roleRef:\s*\n(?:\s+\w+:.*\n)*?)(\s+)name:\s+` +
			regexp.QuoteMeta(detectedPrefix) + `-manager-role`)
	yamlContent = roleRefBlockPattern.ReplaceAllString(
		yamlContent, `${1}${2}name: `+ResourceNameTemplate(chartName, "manager-role"))

	roleRefBlockPatternSimple := regexp.MustCompile(
		`(?s)(roleRef:\s*\n(?:\s+\w+:.*\n)*?)(\s+)name:\s+manager-role`)
	yamlContent = roleRefBlockPatternSimple.ReplaceAllString(
		yamlContent, `${1}${2}name: `+ResourceNameTemplate(chartName, "manager-role"))

	yamlContent = TemplateServiceAccountNameInBindings(detectedPrefix, chartName, yamlContent)

	return yamlContent
}

// TemplateServiceAccountNameInBindings templates SA name in RoleBinding/ClusterRoleBinding subjects.
func TemplateServiceAccountNameInBindings(detectedPrefix, chartName, yamlContent string) string {
	replacement := `{{ include "` + chartName + `.serviceAccountName" . }}`

	// Handle already-templated resourceName (from substituteResourceNamesWithPrefix)
	templatedPattern := regexp.MustCompile(
		`(?m)(subjects:\s*\n\s*-\s*kind:\s*ServiceAccount\s*\n\s+name:\s+)` +
			regexp.QuoteMeta(`{{ include "`+chartName+`.resourceName" (dict "suffix" "controller-manager" "context" $) }}`))
	yamlContent = templatedPattern.ReplaceAllString(yamlContent, `${1}`+replacement)

	// Handle literal names with prefix
	subjectPattern := regexp.MustCompile(
		`(?m)(subjects:\s*\n\s*-\s*kind:\s*ServiceAccount\s*\n\s+name:\s+)` +
			regexp.QuoteMeta(detectedPrefix) + `-controller-manager`)
	yamlContent = subjectPattern.ReplaceAllString(yamlContent, `${1}`+replacement)

	// Handle literal names without prefix
	subjectPatternSimple := regexp.MustCompile(
		`(?m)(subjects:\s*\n\s*-\s*kind:\s*ServiceAccount\s*\n\s+name:\s+)controller-manager`)
	yamlContent = subjectPatternSimple.ReplaceAllString(yamlContent, `${1}`+replacement)

	return yamlContent
}

// TemplateServiceAccountNameInDeployment templates serviceAccountName in Deployment spec.
func TemplateServiceAccountNameInDeployment(detectedPrefix, chartName, yamlContent string) string {
	replacement := `serviceAccountName: {{ include "` + chartName + `.serviceAccountName" . }}`

	// Handle already-templated resourceName (from substituteResourceNamesWithPrefix)
	templatedPattern := regexp.MustCompile(
		`(?m)^(\s*)serviceAccountName:\s+` +
			regexp.QuoteMeta(`{{ include "`+chartName+`.resourceName" (dict "suffix" "controller-manager" "context" $) }}`))
	yamlContent = templatedPattern.ReplaceAllString(yamlContent, `${1}`+replacement)

	// Handle literal names with prefix
	serviceAccountPattern := regexp.MustCompile(
		`(?m)^(\s*)serviceAccountName:\s+` + regexp.QuoteMeta(detectedPrefix) + `-controller-manager\s*$`)
	yamlContent = serviceAccountPattern.ReplaceAllString(yamlContent, `${1}`+replacement)

	// Handle literal names without prefix
	serviceAccountPatternSimple := regexp.MustCompile(
		`(?m)^(\s*)serviceAccountName:\s+controller-manager\s*$`)
	yamlContent = serviceAccountPatternSimple.ReplaceAllString(yamlContent, `${1}`+replacement)

	return yamlContent
}

// TemplateServiceAccount applies all ServiceAccount-specific transformations.
func TemplateServiceAccount(detectedPrefix, chartName, yamlContent string) string {
	yamlContent = AddServiceAccountLabelsAndAnnotations(yamlContent)
	yamlContent = TemplateServiceAccountName(detectedPrefix, chartName, yamlContent)
	yamlContent = WrapServiceAccountWithEnabledConditional(yamlContent)
	return yamlContent
}

// TemplateServiceAccountName replaces SA name with serviceAccountName helper.
func TemplateServiceAccountName(detectedPrefix, chartName, yamlContent string) string {
	replacement := `${1}name: {{ include "` + chartName + `.serviceAccountName" . }}`

	// Handle name with prefix
	namePattern := regexp.MustCompile(
		`(?m)^(\s*)name:\s+` + regexp.QuoteMeta(detectedPrefix) + `-controller-manager\s*$`)
	yamlContent = namePattern.ReplaceAllString(yamlContent, replacement)

	// Handle name without prefix
	namePatternSimple := regexp.MustCompile(`(?m)^(\s*)name:\s+controller-manager\s*$`)
	yamlContent = namePatternSimple.ReplaceAllString(yamlContent, replacement)

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
