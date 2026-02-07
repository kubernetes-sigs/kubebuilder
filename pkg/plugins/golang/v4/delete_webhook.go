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

package v4

import (
	"bufio"
	"fmt"
	log "log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

var _ plugin.DeleteWebhookSubcommand = &deleteWebhookSubcommand{}

type deleteWebhookSubcommand struct {
	config config.Config
	// For help text.
	commandName string

	options *goPlugin.Options

	resource *resource.Resource

	// assumeYes skips interactive confirmation prompts
	assumeYes bool

	// Webhook type flags - which webhook types to delete
	doDefaulting bool
	doValidation bool
	doConversion bool

	// Deprecated - TODO: remove it for go/v5
	// isLegacyPath indicates that the webhook is in the legacy path under the api
	isLegacyPath bool

	// Track what couldn't be automatically removed for manual cleanup instructions
	manualCleanupWebhookImport bool
	manualCleanupWebhookSetup  bool
}

func (p *deleteWebhookSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	p.commandName = cliMeta.CommandName

	subcmdMeta.Description = `Delete webhook(s) for an API resource.

Supports granular deletion: use flags to delete specific webhook types 
(--defaulting, --programmatic-validation, --conversion). Without flags, 
deletes all webhooks for the resource.

Automatically removes:
- Webhook files (when all types deleted)
- Test files (when all types deleted)
- Kustomize manifests (certmanager, webhook service)
- PROJECT file webhook entries

Requires manual cleanup (instructions provided after deletion):
- Webhook imports in cmd/main.go
- Webhook setup calls in cmd/main.go

Note: Webhook files remain if other types still exist for the resource.
`
	subcmdMeta.Examples = fmt.Sprintf(
		`  # Delete all webhooks for the Memcached kind
  %[1]s delete webhook --group cache --version v1alpha1 --kind Memcached

  # Delete only the defaulting webhook
  %[1]s delete webhook --group cache --version v1alpha1 --kind Memcached --defaulting

  # Delete validation and defaulting webhooks, keep conversion
  %[1]s delete webhook --group cache --version v1alpha1 --kind Memcached \
    --defaulting --programmatic-validation

  # Delete without confirmation prompt (use with caution)
  %[1]s delete webhook --group cache --version v1alpha1 --kind Memcached -y
  %[1]s delete webhook --group cache --version v1alpha1 --kind Memcached --yes
`, cliMeta.CommandName)
}

func (p *deleteWebhookSubcommand) BindFlags(fs *pflag.FlagSet) {
	if p.options == nil {
		p.options = &goPlugin.Options{}
	}

	fs.BoolVarP(&p.assumeYes, "yes", "y", false,
		"proceed without prompting for confirmation")

	// Webhook type flags - same as create webhook for consistency
	fs.BoolVar(&p.doDefaulting, "defaulting", false,
		"delete defaulting webhook")
	fs.BoolVar(&p.doValidation, "programmatic-validation", false,
		"delete validation webhook")
	fs.BoolVar(&p.doConversion, "conversion", false,
		"delete conversion webhook")

	// Deprecated flag for backwards compatibility
	fs.BoolVar(&p.isLegacyPath, "legacy-path", false,
		"(Deprecated) indicates webhook is in legacy path under api/")
}

func (p *deleteWebhookSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *deleteWebhookSubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	// For core/external types, match by GVK only, not domain
	var configRes resource.Resource
	var found bool

	resources, err := p.config.GetResources()
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	for _, r := range resources {
		if r.Group == p.resource.Group && r.Version == p.resource.Version && r.Kind == p.resource.Kind {
			configRes = r
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("resource {%q %q %q} does not exist in the project",
			p.resource.Group, p.resource.Version, p.resource.Kind)
	}

	// Check if the resource has webhooks
	if configRes.Webhooks == nil || configRes.Webhooks.IsEmpty() {
		return fmt.Errorf("resource %q does not have any webhooks configured", p.resource.GVK)
	}

	// If no specific webhook type flags are set, default to deleting all webhooks
	if !p.doDefaulting && !p.doValidation && !p.doConversion {
		p.doDefaulting = configRes.Webhooks.Defaulting
		p.doValidation = configRes.Webhooks.Validation
		p.doConversion = configRes.Webhooks.Conversion
		log.Info("No specific webhook type specified, will delete all configured webhooks",
			"defaulting", p.doDefaulting, "validation", p.doValidation, "conversion", p.doConversion)
	}

	// Validate that the specified webhook types actually exist
	if p.doDefaulting && !configRes.Webhooks.Defaulting {
		return fmt.Errorf("resource %q does not have a defaulting webhook configured", p.resource.GVK)
	}
	if p.doValidation && !configRes.Webhooks.Validation {
		return fmt.Errorf("resource %q does not have a validation webhook configured", p.resource.GVK)
	}
	if p.doConversion && !configRes.Webhooks.Conversion {
		return fmt.Errorf("resource %q does not have a conversion webhook configured", p.resource.GVK)
	}

	// Copy relevant fields from config resource
	p.resource = &configRes

	return nil
}

func (p *deleteWebhookSubcommand) Scaffold(fs machinery.Filesystem) error {
	// Prompt for confirmation unless -y/--yes flag provided
	if !p.assumeYes {
		if !p.confirmDeletion() {
			return fmt.Errorf("deletion cancelled by user")
		}
	}

	log.Info("Deleting webhook(s)...",
		"defaulting", p.doDefaulting,
		"validation", p.doValidation,
		"conversion", p.doConversion)

	// Determine if webhook files should be deleted
	shouldDeleteFiles := p.shouldDeleteWebhookFiles()

	deletedFiles, failedFiles := p.deleteWebhookFilesIfNeeded(fs, shouldDeleteFiles)

	// Also delete webhook suite test if appropriate
	if shouldDeleteFiles && p.shouldDeleteWebhookSuiteTest() {
		suiteDeleted := p.deleteWebhookSuiteTest(fs)
		deletedFiles = append(deletedFiles, suiteDeleted...)
	}

	// Build the new webhook configuration
	newWebhooks := p.resource.Webhooks.Copy()

	if p.doDefaulting {
		newWebhooks.Defaulting = false
		newWebhooks.DefaultingPath = ""
	}
	if p.doValidation {
		newWebhooks.Validation = false
		newWebhooks.ValidationPath = ""
	}
	if p.doConversion {
		newWebhooks.Conversion = false
		newWebhooks.Spoke = nil
	}

	// If all webhook types are now disabled, clear the entire webhooks struct
	var webhooksToSet *resource.Webhooks
	if !newWebhooks.Defaulting && !newWebhooks.Validation && !newWebhooks.Conversion {
		webhooksToSet = nil
	} else {
		webhooksToSet = &newWebhooks
	}

	// Use SetResourceWebhooks to properly replace (not merge) the webhook configuration
	if err := p.config.SetResourceWebhooks(p.resource.GVK, webhooksToSet); err != nil {
		return fmt.Errorf("failed to update resource webhooks in PROJECT file: %w", err)
	}

	// Report results (kustomize cleanup is handled by kustomize plugin in the chain)
	if len(failedFiles) > 0 {
		return fmt.Errorf("failed to delete some files: %v", failedFiles)
	}

	webhookTypes := []string{}
	if p.doDefaulting {
		webhookTypes = append(webhookTypes, "defaulting")
	}
	if p.doValidation {
		webhookTypes = append(webhookTypes, "validation")
	}
	if p.doConversion {
		webhookTypes = append(webhookTypes, "conversion")
	}

	fmt.Printf("\nSuccessfully deleted %s webhook(s) for %s/%s (%s)\n",
		strings.Join(webhookTypes, ", "),
		p.resource.Group, p.resource.Version, p.resource.Kind)
	if shouldDeleteFiles {
		if len(deletedFiles) > 0 {
			fmt.Printf("Deleted %d file(s)\n", len(deletedFiles))
		}
		// Remove marker-based code from main.go when all webhooks are deleted
		p.removeCodeFromMainGo(fs)
	} else {
		fmt.Println("Updated PROJECT file to remove webhook configuration")
		fmt.Println("Note: Webhook implementation files were not deleted as other webhook types remain")
	}

	return nil
}

func (p *deleteWebhookSubcommand) PostScaffold() error {
	shouldDeleteFiles := p.shouldDeleteWebhookFiles()

	if shouldDeleteFiles {
		// Check if manual cleanup is needed
		if p.manualCleanupWebhookImport || p.manualCleanupWebhookSetup {
			fmt.Println()
			fmt.Println("Manual cleanup required - automatic removal failed for:")
			fmt.Println()
			fmt.Println("In cmd/main.go:")

			repo := p.config.GetRepository()
			webhookImportAlias := "webhook" + strings.ToLower(p.resource.Group) + p.resource.Version
			if !p.config.IsMultiGroup() || p.resource.Group == "" {
				webhookImportAlias = "webhook" + p.resource.Version
			}

			if p.manualCleanupWebhookImport {
				if p.config.IsMultiGroup() && p.resource.Group != "" {
					fmt.Printf("  - Remove import: %s \"%s/internal/webhook/%s/%s\"\n",
						webhookImportAlias, repo, p.resource.Group, p.resource.Version)
				} else {
					fmt.Printf("  - Remove import: %s \"%s/internal/webhook/%s\"\n",
						webhookImportAlias, repo, p.resource.Version)
				}
			}

			if p.manualCleanupWebhookSetup {
				fmt.Printf("  - Remove webhook setup block for: %s (see constant webhookSetupCodeFragment)\n",
					p.resource.Kind)
			}
		}

		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("$ go mod tidy")
		fmt.Println("$ make generate")
		fmt.Println("$ make manifests")
	} else {
		webhooksRemaining := []string{}
		if p.resource.Webhooks.Defaulting && !p.doDefaulting {
			webhooksRemaining = append(webhooksRemaining, "defaulting")
		}
		if p.resource.Webhooks.Validation && !p.doValidation {
			webhooksRemaining = append(webhooksRemaining, "validation")
		}
		if p.resource.Webhooks.Conversion && !p.doConversion {
			webhooksRemaining = append(webhooksRemaining, "conversion")
		}

		// Build list of deleted webhook types for manual cleanup
		deletedTypes := []string{}
		if p.doDefaulting {
			deletedTypes = append(deletedTypes, ".WithDefaulter()")
		}
		if p.doValidation {
			deletedTypes = append(deletedTypes, ".WithValidator()")
		}
		if p.doConversion {
			deletedTypes = append(deletedTypes, "conversion webhook setup")
		}

		fmt.Printf("\nWebhook files kept (remaining types: %s).\n", strings.Join(webhooksRemaining, ", "))
		if len(deletedTypes) > 0 {
			versionPath := p.resource.Version
			if p.config.IsMultiGroup() && p.resource.Group != "" {
				versionPath = p.resource.Group + "/" + p.resource.Version
			}
			webhookFile := fmt.Sprintf("internal/webhook/%s/%s_webhook.go",
				versionPath, strings.ToLower(p.resource.Kind))
			fmt.Printf("Manual cleanup required in %s:\n", webhookFile)
			fmt.Printf("  - Remove deleted webhook setup: %s\n", strings.Join(deletedTypes, ", "))
		}
		fmt.Println()
		fmt.Println("Next: update your webhook implementation and run:")
		fmt.Println("$ make generate")
		fmt.Println("$ make manifests")
	}

	return nil
}

func (p *deleteWebhookSubcommand) confirmDeletion() bool {
	webhookTypes := []string{}
	if p.doDefaulting {
		webhookTypes = append(webhookTypes, "defaulting")
	}
	if p.doValidation {
		webhookTypes = append(webhookTypes, "validation")
	}
	if p.doConversion {
		webhookTypes = append(webhookTypes, "conversion")
	}

	fmt.Printf("\nWarning: You are about to delete %s webhook(s) for %s/%s (%s)\n",
		strings.Join(webhookTypes, ", "),
		p.resource.Group, p.resource.Version, p.resource.Kind)

	fmt.Println("This will remove:")

	// Check if ALL webhook types will be removed
	willHaveDefaulting := p.resource.Webhooks.Defaulting && !p.doDefaulting
	willHaveValidation := p.resource.Webhooks.Validation && !p.doValidation
	willHaveConversion := p.resource.Webhooks.Conversion && !p.doConversion
	willRemoveAllWebhooks := !willHaveDefaulting && !willHaveValidation && !willHaveConversion

	if willRemoveAllWebhooks {
		fmt.Println("  - Webhook implementation files")
		fmt.Println("  - Webhook test files")
	}
	fmt.Printf("  - %s webhook configuration from PROJECT file\n", strings.Join(webhookTypes, ", "))

	if !willRemoveAllWebhooks {
		fmt.Println("\nNote: Webhook implementation files will NOT be deleted as other webhook types remain")
	}

	fmt.Println("\nThis operation cannot be undone!")
	fmt.Print("\nAre you sure you want to continue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Error("Failed to read user input", "error", err)
		return false
	}

	// Remove whitespace (handles both Unix \n and Windows \r\n)
	response = strings.TrimSpace(response)
	return response == "y" || response == "Y" || response == "yes" || response == "YES"
}

// shouldDeleteWebhookFiles determines if webhook files should be deleted
// Files are deleted only if NO webhook types will remain for this resource
func (p *deleteWebhookSubcommand) shouldDeleteWebhookFiles() bool {
	willHaveDefaulting := p.resource.Webhooks.Defaulting && !p.doDefaulting
	willHaveValidation := p.resource.Webhooks.Validation && !p.doValidation
	willHaveConversion := p.resource.Webhooks.Conversion && !p.doConversion
	return !willHaveDefaulting && !willHaveValidation && !willHaveConversion
}

// deleteWebhookFilesIfNeeded deletes webhook implementation files if shouldDelete is true
func (p *deleteWebhookSubcommand) deleteWebhookFilesIfNeeded(
	fs machinery.Filesystem, shouldDelete bool,
) (deletedFiles, failedFiles []string) {
	if !shouldDelete {
		return nil, nil
	}

	multigroup := p.config.IsMultiGroup()
	kindLower := strings.ToLower(p.resource.Kind)

	// Get webhook file paths
	webhookPath, webhookTestPath := p.getWebhookFilePaths(multigroup, kindLower)
	filesToDelete := []string{webhookPath, webhookTestPath}

	// Delete each file
	for _, file := range filesToDelete {
		exists, err := afero.Exists(fs.FS, file)
		if err != nil {
			log.Warn("Error checking file existence", "file", file, "error", err)
			continue
		}

		if !exists {
			log.Warn("File does not exist, skipping", "file", file)
			continue
		}

		if err := fs.FS.Remove(file); err != nil {
			log.Error("Failed to delete file", "file", file, "error", err)
			failedFiles = append(failedFiles, file)
		} else {
			log.Info("Deleted file", "file", file)
			deletedFiles = append(deletedFiles, file)
		}
	}

	return deletedFiles, failedFiles
}

// getWebhookFilePaths returns the paths to webhook implementation and test files
func (p *deleteWebhookSubcommand) getWebhookFilePaths(multigroup bool, kindLower string) (string, string) {
	var webhookPath, webhookTestPath string

	if p.isLegacyPath {
		// Legacy path: api/<version>/<kind>_webhook.go
		if multigroup && p.resource.Group != "" {
			webhookPath = filepath.Join("api", p.resource.Group, p.resource.Version,
				fmt.Sprintf("%s_webhook.go", kindLower))
			webhookTestPath = filepath.Join("api", p.resource.Group, p.resource.Version,
				fmt.Sprintf("%s_webhook_test.go", kindLower))
		} else {
			webhookPath = filepath.Join("api", p.resource.Version,
				fmt.Sprintf("%s_webhook.go", kindLower))
			webhookTestPath = filepath.Join("api", p.resource.Version,
				fmt.Sprintf("%s_webhook_test.go", kindLower))
		}
	} else {
		// Standard path: internal/webhook/<version>/<kind>_webhook.go
		if multigroup && p.resource.Group != "" {
			webhookPath = filepath.Join("internal", "webhook", p.resource.Group, p.resource.Version,
				fmt.Sprintf("%s_webhook.go", kindLower))
			webhookTestPath = filepath.Join("internal", "webhook", p.resource.Group, p.resource.Version,
				fmt.Sprintf("%s_webhook_test.go", kindLower))
		} else {
			webhookPath = filepath.Join("internal", "webhook", p.resource.Version,
				fmt.Sprintf("%s_webhook.go", kindLower))
			webhookTestPath = filepath.Join("internal", "webhook", p.resource.Version,
				fmt.Sprintf("%s_webhook_test.go", kindLower))
		}
	}

	return webhookPath, webhookTestPath
}

// willBeLastWebhookAfterDeletion checks if there will be any webhooks remaining after this deletion
// It checks:
// 1. Will the current resource have any webhooks left after deletion?
// 2. Do any other resources have webhooks?
func (p *deleteWebhookSubcommand) willBeLastWebhookAfterDeletion() (bool, error) {
	resources, err := p.config.GetResources()
	if err != nil {
		return false, fmt.Errorf("failed to get resources from config: %w", err)
	}

	// Check if current resource will have webhooks remaining after deletion
	currentResourceWillHaveWebhooks := false
	if p.resource.Webhooks != nil {
		willHaveDefaulting := p.resource.Webhooks.Defaulting && !p.doDefaulting
		willHaveValidation := p.resource.Webhooks.Validation && !p.doValidation
		willHaveConversion := p.resource.Webhooks.Conversion && !p.doConversion
		currentResourceWillHaveWebhooks = willHaveDefaulting || willHaveValidation || willHaveConversion
	}

	if currentResourceWillHaveWebhooks {
		// Current resource will still have webhooks, so definitely not the last
		return false, nil
	}

	// Check if any other resources have webhooks
	for _, res := range resources {
		// Skip the current resource (we already know it won't have webhooks after deletion)
		if res.IsEqualTo(p.resource.GVK) {
			continue
		}

		// Check if this other resource has webhooks
		if res.Webhooks != nil && !res.Webhooks.IsEmpty() {
			return false, nil
		}
	}

	// No webhooks will remain anywhere
	return true, nil
}

// deleteWebhookSuiteTest deletes the webhook_suite_test.go file for this version
func (p *deleteWebhookSubcommand) deleteWebhookSuiteTest(fs machinery.Filesystem) []string {
	deleted := []string{}

	var suiteTestPath string
	if p.config.IsMultiGroup() && p.resource.Group != "" {
		suiteTestPath = filepath.Join("internal", "webhook", p.resource.Group,
			p.resource.Version, "webhook_suite_test.go")
	} else {
		suiteTestPath = filepath.Join("internal", "webhook", p.resource.Version,
			"webhook_suite_test.go")
	}

	if exists, _ := afero.Exists(fs.FS, suiteTestPath); exists {
		if err := fs.FS.Remove(suiteTestPath); err != nil {
			log.Warn("Failed to delete webhook suite test", "file", suiteTestPath, "error", err)
		} else {
			log.Info("Deleted webhook suite test", "file", suiteTestPath)
			deleted = append(deleted, suiteTestPath)
		}
	}

	return deleted
}

// shouldDeleteWebhookSuiteTest checks if the webhook_suite_test.go should be deleted
// It should be deleted if no other resources in this version have webhooks
func (p *deleteWebhookSubcommand) shouldDeleteWebhookSuiteTest() bool {
	resources, err := p.config.GetResources()
	if err != nil {
		return false
	}

	for _, res := range resources {
		// Skip current resource
		if res.Group == p.resource.Group && res.Version == p.resource.Version && res.Kind == p.resource.Kind {
			continue
		}

		// Check if resource is in same version and has webhooks
		if res.Version == p.resource.Version && res.Webhooks != nil && !res.Webhooks.IsEmpty() {
			return false
		}
	}

	return true
}

// Webhook code fragments to remove from main.go (match the format used in create)
const (
	webhookImportCodeFragment = `%s "%s/internal/webhook/%s"
`
	multiGroupWebhookImportCodeFragment = `%s "%s/internal/webhook/%s/%s"
`
	webhookSetupCodeFragment = `// nolint:goconst
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err := %s.Setup%sWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "Failed to create webhook", "webhook", "%s")
			os.Exit(1)
		}
	}
`
)

// removeCodeFromMainGo removes marker-inserted webhook code from cmd/main.go
func (p *deleteWebhookSubcommand) removeCodeFromMainGo(_ machinery.Filesystem) {
	mainPath := filepath.Join("cmd", "main.go")

	repo := p.config.GetRepository()

	webhookImportAlias := "webhook" + strings.ToLower(p.resource.Group) + p.resource.Version
	if !p.config.IsMultiGroup() || p.resource.Group == "" {
		webhookImportAlias = "webhook" + p.resource.Version
	}

	// Remove webhook import
	var webhookImport string
	if p.config.IsMultiGroup() && p.resource.Group != "" {
		webhookImport = fmt.Sprintf(multiGroupWebhookImportCodeFragment,
			webhookImportAlias, repo, p.resource.Group, p.resource.Version)
	} else {
		webhookImport = fmt.Sprintf(webhookImportCodeFragment,
			webhookImportAlias, repo, p.resource.Version)
	}

	if err := util.ReplaceInFile(mainPath, "\t"+webhookImport, ""); err != nil {
		log.Warn("Unable to remove webhook import from main.go - manual cleanup needed",
			"import", webhookImportAlias, "error", err)
		p.manualCleanupWebhookImport = true
	} else {
		log.Info("Removed webhook import from main.go", "import", webhookImportAlias)
	}

	// Remove webhook setup code
	webhookSetup := fmt.Sprintf("\t"+webhookSetupCodeFragment,
		webhookImportAlias, p.resource.Kind, p.resource.Kind)

	if err := util.ReplaceInFile(mainPath, webhookSetup, ""); err != nil {
		log.Warn("Unable to remove webhook setup from main.go - manual cleanup needed",
			"webhook", p.resource.Kind, "error", err)
		p.manualCleanupWebhookSetup = true
	} else {
		log.Info("Removed webhook setup from main.go", "webhook", p.resource.Kind)
	}
}
