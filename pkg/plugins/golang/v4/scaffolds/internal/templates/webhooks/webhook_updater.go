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

package webhooks

import (
	"bytes"
	"fmt"
	log "log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

const coreGroup = "core"

var _ machinery.Template = &WebhookUpdater{}

// WebhookUpdater updates an existing webhook file to add additional webhook types
type WebhookUpdater struct {
	machinery.TemplateMixin
	machinery.RepositoryMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	// QualifiedGroupWithDash is the Group domain for the Resource replacing '.' with '-'
	QualifiedGroupWithDash string

	// AdmissionReviewVersions defines value for AdmissionReviewVersions marker
	AdmissionReviewVersions string
}

// GetPath implements file.Builder
func (f *WebhookUpdater) GetPath() string {
	baseDir := filepath.Join("internal", "webhook")

	var path string
	if f.MultiGroup && f.Resource.Group != "" {
		path = filepath.Join(baseDir, "%[group]", "%[version]", "%[kind]_webhook.go")
	} else {
		path = filepath.Join(baseDir, "%[version]", "%[kind]_webhook.go")
	}

	return f.Resource.Replacer().Replace(path)
}

// GetIfExistsAction implements file.Builder
func (*WebhookUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile
}

// SetTemplateDefaults implements file.Template
func (f *WebhookUpdater) SetTemplateDefaults() error {
	filePath := f.GetPath()

	// Read the existing file
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Error("Unable to read webhook file", "file", filePath, "error", err)
		return fmt.Errorf("failed to read webhook file: %w", err)
	}

	f.QualifiedGroupWithDash = strings.ReplaceAll(f.Resource.QualifiedGroup(), ".", "-")
	f.AdmissionReviewVersions = "v1"

	fileContent := string(content)
	var newCode strings.Builder

	// Add defaulting webhook if requested and not already present
	defaulterType := fmt.Sprintf("%sCustomDefaulter", f.Resource.Kind)
	if f.Resource.HasDefaultingWebhook() {
		typeDefPattern := regexp.MustCompile(fmt.Sprintf(`type\s+%s\s+struct`, defaulterType))
		if typeDefPattern.MatchString(string(content)) {
			log.Info("Defaulting webhook already exists, skipping", "kind", f.Resource.Kind)
		} else {
			defaultingCode := f.generateDefaultingWebhookCode()
			if defaultingCode != "" {
				newCode.WriteString(defaultingCode)
			}

			setupCode := f.generateDefaulterSetupCode()
			if !strings.Contains(fileContent, fmt.Sprintf("WithDefaulter(&%s{})", defaulterType)) {
				fileContent = f.injectBeforeComplete(fileContent, setupCode)
			}
		}
	}

	// Add validation webhook if requested and not already present
	validatorType := fmt.Sprintf("%sCustomValidator", f.Resource.Kind)
	if f.Resource.HasValidationWebhook() {
		typeDefPattern := regexp.MustCompile(fmt.Sprintf(`type\s+%s\s+struct`, validatorType))
		if typeDefPattern.MatchString(string(content)) {
			log.Info("Validation webhook already exists, skipping", "kind", f.Resource.Kind)
		} else {
			if !bytes.Contains(content, []byte("sigs.k8s.io/controller-runtime/pkg/webhook/admission")) {
				fileContent = f.addAdmissionImport(fileContent)
			}

			validationCode := f.generateValidationWebhookCode()
			if validationCode != "" {
				newCode.WriteString(validationCode)
			}

			setupCode := f.generateValidatorSetupCode()
			if !strings.Contains(fileContent, fmt.Sprintf("WithValidator(&%s{})", validatorType)) {
				fileContent = f.injectBeforeComplete(fileContent, setupCode)
			}
		}
	}

	// Append new webhook code at the end of the file
	if newCode.Len() > 0 {
		fileContent = strings.TrimRight(fileContent, "\n") + "\n" + newCode.String()
	}

	f.TemplateBody = fileContent
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

// injectBeforeComplete injects webhook setup code before the Complete() call
func (f *WebhookUpdater) injectBeforeComplete(content, code string) string {
	completePattern := regexp.MustCompile(`(?m)^(\s*)(?:\.)?\s*Complete\(\s*\)`)

	if match := completePattern.FindStringSubmatch(content); len(match) > 1 {
		completeCall := match[0]
		baseIndent := match[1]

		beforeComplete := content[:strings.Index(content, completeCall)]
		indent := f.detectIndentationBeforeComplete(beforeComplete, baseIndent)
		adjustedCode := f.adjustCodeIndentation(code, indent)

		insertPos := strings.Index(content, completeCall)
		return content[:insertPos] + adjustedCode + content[insertPos:]
	}

	log.Warn("Could not find Complete() call in setup function",
		"kind", f.Resource.Kind,
		"suggestion", "Manually wire webhook in SetupWebhookWithManager")
	return content
}

// detectIndentationBeforeComplete extracts indentation from method chain lines
func (f *WebhookUpdater) detectIndentationBeforeComplete(beforeComplete, baseIndent string) string {
	lines := strings.Split(beforeComplete, "\n")

	for i := len(lines) - 1; i >= 0 && i >= len(lines)-5; i-- {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, ".") && (strings.Contains(trimmed, "For(") ||
			strings.Contains(trimmed, "With") || strings.Contains(trimmed, "Owns(")) {
			leadingSpace := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			if leadingSpace != "" {
				return leadingSpace
			}
		}
	}

	if strings.Contains(baseIndent, "\t") {
		return baseIndent + "\t\t"
	}
	return baseIndent + "        "
}

// adjustCodeIndentation replaces existing indentation with target indentation
func (f *WebhookUpdater) adjustCodeIndentation(code, targetIndent string) string {
	lines := strings.Split(code, "\n")
	adjusted := make([]string, len(lines))

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			adjusted[i] = line
			continue
		}

		trimmed := strings.TrimLeft(line, " \t")
		if len(trimmed) < len(line) {
			adjusted[i] = targetIndent + trimmed
		} else {
			adjusted[i] = line
		}
	}

	return strings.Join(adjusted, "\n")
}

// addAdmissionImport adds the admission package import after the webhook import
func (f *WebhookUpdater) addAdmissionImport(content string) string {
	admissionImport := "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	if strings.Contains(content, admissionImport) {
		return content
	}

	// Add after webhook import
	webhookPattern := regexp.MustCompile(`(?m)^(\s*)"sigs\.k8s\.io/controller-runtime/pkg/webhook"`)

	if match := webhookPattern.FindStringSubmatch(content); len(match) > 1 {
		indent := match[1]
		webhookLine := match[0]
		replacement := webhookLine + "\n" + indent + `"` + admissionImport + `"`
		return strings.Replace(content, webhookLine, replacement, 1)
	}

	// Fallback: add to end of import block
	importBlockPattern := regexp.MustCompile(`(?s)(import\s*\([^)]+)(\s*\))`)

	if match := importBlockPattern.FindStringSubmatch(content); len(match) > 2 {
		lastImportPattern := regexp.MustCompile(`(?m)^(\s*)"[^"]+"\s*$`)
		imports := lastImportPattern.FindAllStringSubmatch(match[1], -1)

		indent := "\t"
		if len(imports) > 0 && len(imports[len(imports)-1]) > 1 {
			indent = imports[len(imports)-1][1]
		}

		newImport := "\n" + indent + `"` + admissionImport + `"`
		replacement := match[1] + newImport + match[2]
		return strings.Replace(content, match[0], replacement, 1)
	}

	log.Warn("Could not add admission import",
		"kind", f.Resource.Kind,
		"suggestion", "Manually add: "+admissionImport)
	return content
}

// generateDefaulterSetupCode generates the setup code for defaulting webhook
func (f *WebhookUpdater) generateDefaulterSetupCode() string {
	code := fmt.Sprintf("\t\tWithDefaulter(&%sCustomDefaulter{}).", f.Resource.Kind)
	if f.Resource.Webhooks.DefaultingPath != "" {
		code += fmt.Sprintf("\n\t\tWithDefaulterCustomPath(\"%s\").", f.Resource.Webhooks.DefaultingPath)
	}
	return code + "\n"
}

// generateValidatorSetupCode generates the setup code for validation webhook
func (f *WebhookUpdater) generateValidatorSetupCode() string {
	code := fmt.Sprintf("\t\tWithValidator(&%sCustomValidator{}).", f.Resource.Kind)
	if f.Resource.Webhooks.ValidationPath != "" {
		code += fmt.Sprintf("\n\t\tWithValidatorCustomPath(\"%s\").", f.Resource.Webhooks.ValidationPath)
	}
	return code + "\n"
}

// generateDefaultingWebhookCode generates the defaulting webhook code
func (f *WebhookUpdater) generateDefaultingWebhookCode() string {
	var code strings.Builder

	// Webhook marker
	defaultingPath := f.Resource.Webhooks.DefaultingPath
	if defaultingPath == "" {
		if f.Resource.Core && f.Resource.QualifiedGroup() == coreGroup {
			defaultingPath = fmt.Sprintf("/mutate--%s-%s",
				f.Resource.Version, strings.ToLower(f.Resource.Kind))
		} else {
			defaultingPath = fmt.Sprintf("/mutate-%s-%s-%s",
				f.QualifiedGroupWithDash, f.Resource.Version, strings.ToLower(f.Resource.Kind))
		}
	}

	//nolint:lll
	code.WriteString(fmt.Sprintf(
		`
// +kubebuilder:webhook:path=%s,mutating=true,failurePolicy=fail,sideEffects=None,groups=%s,resources=%s,verbs=create;update,versions=%s,name=m%s-%s.kb.io,admissionReviewVersions=%s

// %sCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind %s when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type %sCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

`,
		defaultingPath, f.getGroupValue(), f.Resource.Plural, f.Resource.Version,
		strings.ToLower(f.Resource.Kind), f.Resource.Version,
		f.AdmissionReviewVersions,
		f.Resource.Kind, f.Resource.Kind, f.Resource.Kind))

	// Default method
	objType := f.Resource.ImportAlias() + "." + f.Resource.Kind

	code.WriteString(fmt.Sprintf(
		`// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind %s.
func (d *%sCustomDefaulter) Default(_ context.Context, obj *%s) error {
	%slog.Info("Defaulting for %s", "name", obj.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}
`,
		f.Resource.Kind, f.Resource.Kind, objType,
		strings.ToLower(f.Resource.Kind), f.Resource.Kind))

	return code.String()
}

// generateValidationWebhookCode generates the validation webhook code
func (f *WebhookUpdater) generateValidationWebhookCode() string {
	var code strings.Builder

	// Webhook marker
	validationPath := f.Resource.Webhooks.ValidationPath
	if validationPath == "" {
		if f.Resource.Core && f.Resource.QualifiedGroup() == coreGroup {
			validationPath = fmt.Sprintf("/validate--%s-%s",
				f.Resource.Version, strings.ToLower(f.Resource.Kind))
		} else {
			validationPath = fmt.Sprintf("/validate-%s-%s-%s",
				f.QualifiedGroupWithDash, f.Resource.Version, strings.ToLower(f.Resource.Kind))
		}
	}

	code.WriteString(
		`// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
`)
	//nolint:lll
	code.WriteString(fmt.Sprintf(
		`// +kubebuilder:webhook:path=%s,mutating=false,failurePolicy=fail,sideEffects=None,groups=%s,resources=%s,verbs=create;update,versions=%s,name=v%s-%s.kb.io,admissionReviewVersions=%s

// %sCustomValidator struct is responsible for validating the %s resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type %sCustomValidator struct{
	// TODO(user): Add more fields as needed for validation
}

`,
		validationPath, f.getGroupValue(), f.Resource.Plural, f.Resource.Version,
		strings.ToLower(f.Resource.Kind), f.Resource.Version, f.AdmissionReviewVersions,
		f.Resource.Kind, f.Resource.Kind, f.Resource.Kind))

	// Validation methods
	objType := f.Resource.ImportAlias() + "." + f.Resource.Kind

	code.WriteString(fmt.Sprintf(
		`// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type %s.
func (v *%sCustomValidator) ValidateCreate(_ context.Context, obj *%s) (admission.Warnings, error) {
	%slog.Info("Validation for %s upon creation", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type %s.
func (v *%sCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *%s) (admission.Warnings, error) {
	%slog.Info("Validation for %s upon update", "name", newObj.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type %s.
func (v *%sCustomValidator) ValidateDelete(_ context.Context, obj *%s) (admission.Warnings, error) {
	%slog.Info("Validation for %s upon deletion", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
`,
		f.Resource.Kind, f.Resource.Kind, objType,
		strings.ToLower(f.Resource.Kind), f.Resource.Kind,
		f.Resource.Kind, f.Resource.Kind, objType,
		strings.ToLower(f.Resource.Kind), f.Resource.Kind,
		f.Resource.Kind, f.Resource.Kind, objType,
		strings.ToLower(f.Resource.Kind), f.Resource.Kind))

	return code.String()
}

// getGroupValue returns the group value for webhook markers
func (f *WebhookUpdater) getGroupValue() string {
	if f.Resource.Core && f.Resource.QualifiedGroup() == coreGroup {
		return `""`
	}
	return f.Resource.QualifiedGroup()
}
