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

const (
	// testClosingLine is the closing line of a Describe block in test files
	testClosingLine = "\n})\n"
)

var _ machinery.Template = &WebhookTestUpdater{}

// WebhookTestUpdater updates an existing webhook test file to add validator/defaulter variables
type WebhookTestUpdater struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.ResourceMixin
}

// GetPath implements file.Builder
func (f *WebhookTestUpdater) GetPath() string {
	baseDir := filepath.Join("internal", "webhook")

	var path string
	if f.MultiGroup && f.Resource.Group != "" {
		path = filepath.Join(baseDir, "%[group]", "%[version]", "%[kind]_webhook_test.go")
	} else {
		path = filepath.Join(baseDir, "%[version]", "%[kind]_webhook_test.go")
	}

	return f.Resource.Replacer().Replace(path)
}

// GetIfExistsAction implements file.Builder
func (*WebhookTestUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile
}

// SetTemplateDefaults implements file.Template
func (f *WebhookTestUpdater) SetTemplateDefaults() error {
	filePath := f.GetPath()

	// Read the existing file
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Warn("Unable to read webhook test file, skipping update", "file", filePath, "error", err)
		// Return nil to continue - file might not exist yet
		return nil
	}

	fileContent := string(content)
	modified := false

	// Check if we need to add validator variable and tests
	validatorVar := fmt.Sprintf("validator %sCustomValidator", f.Resource.Kind)
	if f.Resource.HasValidationWebhook() && !bytes.Contains(content, []byte(validatorVar)) {
		fileContent = f.addValidatorVariable(fileContent)
		fileContent = f.addValidatorInit(fileContent)
		fileContent = f.addValidationTestContext(fileContent)
		modified = true
	}

	// Check if we need to add defaulter variable and tests
	defaulterVar := fmt.Sprintf("defaulter %sCustomDefaulter", f.Resource.Kind)
	if f.Resource.HasDefaultingWebhook() && !bytes.Contains(content, []byte(defaulterVar)) {
		fileContent = f.addDefaulterVariable(fileContent)
		fileContent = f.addDefaulterInit(fileContent)
		fileContent = f.addDefaultingTestContext(fileContent)
		modified = true
	}

	// Check if we need to add conversion test context
	if f.Resource.HasConversionWebhook() && !bytes.Contains(content, []byte("Conversion Webhook")) {
		fileContent = f.addConversionTestContext(fileContent)
		modified = true
	}

	if !modified {
		// No updates needed, skip writing
		return nil
	}

	f.TemplateBody = fileContent
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

// addValidatorVariable adds the validator variable to the var block
func (f *WebhookTestUpdater) addValidatorVariable(content string) string {
	varName := "validator"
	typeName := f.Resource.Kind + "CustomValidator"
	return f.addVariableToBlock(content, varName, typeName)
}

// addDefaulterVariable adds the defaulter variable to the var block
func (f *WebhookTestUpdater) addDefaulterVariable(content string) string {
	varName := "defaulter"
	typeName := f.Resource.Kind + "CustomDefaulter"
	return f.addVariableToBlock(content, varName, typeName)
}

// addVariableToBlock adds a variable declaration to the var block before the closing paren
func (f *WebhookTestUpdater) addVariableToBlock(content, varName, typeName string) string {
	varBlockPattern := regexp.MustCompile(`(?s)(var\s*\(\s*)([^)]*?)(\s*\))`)

	if match := varBlockPattern.FindStringSubmatch(content); len(match) > 3 {
		opening := match[1]
		declarations := match[2]
		closing := match[3]

		indent := f.detectIndentationInBlock(declarations)
		if indent == "" {
			indent = "\t\t"
		}

		varDecl := fmt.Sprintf("\n%s%s %s", indent, varName, typeName)
		newVarBlock := opening + declarations + varDecl + closing

		return strings.Replace(content, match[0], newVarBlock, 1)
	}

	log.Warn("Could not find var block in test file",
		"kind", f.Resource.Kind,
		"variable", varName,
		"suggestion", fmt.Sprintf("Manually add '%s %s' to the var block", varName, typeName))
	return content
}

// detectIndentationInBlock extracts indentation from existing code
func (f *WebhookTestUpdater) detectIndentationInBlock(blockContent string) string {
	lines := strings.Split(blockContent, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed != "var" && trimmed != "(" {
			trimmedLeft := strings.TrimLeft(line, " \t")
			if len(line) > len(trimmedLeft) {
				return line[:len(line)-len(trimmedLeft)]
			}
		}
	}
	return "\t\t"
}

// addValidatorInit adds validator initialization in BeforeEach
func (f *WebhookTestUpdater) addValidatorInit(content string) string {
	varName := "validator"
	typeName := f.Resource.Kind + "CustomValidator"
	return f.addWebhookInit(content, varName, typeName)
}

// addDefaulterInit adds defaulter initialization in BeforeEach
func (f *WebhookTestUpdater) addDefaulterInit(content string) string {
	varName := "defaulter"
	typeName := f.Resource.Kind + "CustomDefaulter"
	return f.addWebhookInit(content, varName, typeName)
}

// addWebhookInit adds webhook variable initialization at the end of BeforeEach block
func (f *WebhookTestUpdater) addWebhookInit(content, varName, typeName string) string {
	checkPattern := fmt.Sprintf("%s = %s", varName, typeName)
	if strings.Contains(content, checkPattern) {
		return content
	}

	// Add at the END of BeforeEach block (before closing brace)
	beforeEachPattern := regexp.MustCompile(`(?s)(BeforeEach\s*\(\s*func\s*\(\s*\)\s*\{)(.*?)(\n\s*\}\s*\))`)

	if match := beforeEachPattern.FindStringSubmatch(content); len(match) > 3 {
		opening := match[1]
		blockContent := match[2]
		closing := match[3]

		closingLines := strings.Split(closing, "\n")
		indent := "\t\t"
		if len(closingLines) > 1 {
			closingLine := closingLines[1]
			baseIndent := closingLine[:len(closingLine)-len(strings.TrimLeft(closingLine, " \t"))]
			indent = baseIndent + "\t"
		}

		init := fmt.Sprintf("\n%s%s = %s{}\n%sExpect(%s).NotTo(BeNil(), \"Expected %s to be initialized\")",
			indent, varName, typeName, indent, varName, varName)
		replacement := opening + blockContent + init + closing
		return strings.Replace(content, match[0], replacement, 1)
	}

	log.Warn("Could not find BeforeEach block",
		"kind", f.Resource.Kind,
		"variable", varName,
		"suggestion", fmt.Sprintf("Manually add '%s = %s{}' to BeforeEach", varName, typeName))
	return content
}

// addValidationTestContext adds the validation test context
func (f *WebhookTestUpdater) addValidationTestContext(content string) string {
	testContext := fmt.Sprintf(`
	Context("When creating or updating %s under Validating Webhook", func() {
		// TODO (user): Add logic for validating webhooks
		// Example:
		// It("Should deny creation if a required field is missing", func() {
		//     By("simulating an invalid creation scenario")
		//     obj.SomeRequiredField = ""
		//     Expect(validator.ValidateCreate(ctx, obj)).Error().To(HaveOccurred())
		// })
		//
		// It("Should admit creation if all required fields are present", func() {
		//     By("simulating an invalid creation scenario")
		//     obj.SomeRequiredField = "valid_value"
		//     Expect(validator.ValidateCreate(ctx, obj)).To(BeNil())
		// })
		//
		// It("Should validate updates correctly", func() {
		//     By("simulating a valid update scenario")
		//     oldObj.SomeRequiredField = "updated_value"
		//     obj.SomeRequiredField = "updated_value"
		//     Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil())
		// })
	})
`, f.Resource.Kind)

	return f.addContextToEnd(content, testContext)
}

// addDefaultingTestContext adds the defaulting test context
func (f *WebhookTestUpdater) addDefaultingTestContext(content string) string {
	testContext := fmt.Sprintf(`
	Context("When creating %s under Defaulting Webhook", func() {
		// TODO (user): Add logic for defaulting webhooks
		// Example:
		// It("Should apply defaults when a required field is empty", func() {
		//     By("simulating a scenario where defaults should be applied")
		//     obj.SomeFieldWithDefault = ""
		//     By("calling the Default method to apply defaults")
		//     defaulter.Default(ctx, obj)
		//     By("checking that the default values are set")
		//     Expect(obj.SomeFieldWithDefault).To(Equal("default_value"))
		// })
	})
`, f.Resource.Kind)

	return f.addContextToEnd(content, testContext)
}

// addConversionTestContext adds the conversion test context
func (f *WebhookTestUpdater) addConversionTestContext(content string) string {
	testContext := fmt.Sprintf(`
	Context("When creating %s under Conversion Webhook", func() {
		// TODO (user): Add logic to convert the object to the desired version and verify the conversion
		// Example:
		// It("Should convert the object correctly", func() {
		//     convertedObj := &%s.%s{}
		//     Expect(obj.ConvertTo(convertedObj)).To(Succeed())
		//     Expect(convertedObj).ToNot(BeNil())
		// })
	})
`, f.Resource.Kind, f.Resource.ImportAlias(), f.Resource.Kind)

	return f.addContextToEnd(content, testContext)
}

// addContextToEnd adds a test context before the Describe block's closing })
func (f *WebhookTestUpdater) addContextToEnd(content, testContext string) string {
	// Find the Describe block's closing })
	describePattern := regexp.MustCompile(`(?s)var\s*_\s*=\s*Describe\([^}]+`)
	if match := describePattern.FindStringIndex(content); match != nil {
		lastClosing := regexp.MustCompile(`\n\}\)\s*$`)
		if closingMatch := lastClosing.FindStringIndex(content); closingMatch != nil {
			return content[:closingMatch[0]] + testContext + content[closingMatch[0]:]
		}
	}

	// Fallback: find last }) in file (test files typically end with })
	if idx := strings.LastIndex(content, testClosingLine); idx != -1 {
		return content[:idx] + testContext + testClosingLine
	}

	// Last resort: append at end of file
	content = strings.TrimRight(content, "\n")
	return content + testContext + "\n"
}
