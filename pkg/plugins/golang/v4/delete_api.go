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
	"slices"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

var _ plugin.DeleteAPISubcommand = &deleteAPISubcommand{}

type deleteAPISubcommand struct {
	config config.Config
	// For help text.
	commandName string

	options *goPlugin.Options

	resource *resource.Resource

	// skipConfirmation skips the confirmation prompt
	skipConfirmation bool

	// pluginChain stores the current plugin chain for cross-plugin coordination
	pluginChain []string
}

func (p *deleteAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	p.commandName = cliMeta.CommandName

	subcmdMeta.Description = `Delete a Kubernetes API and its associated files.

Automatically removes:
- API type definitions (api/<version>/<kind>_types.go)
- Controller files (internal/controller/<kind>_controller.go)
- Test files
- Kustomize manifests (samples, RBAC)
- PROJECT file entries

Requires manual cleanup (instructions provided after deletion):
- Imports in cmd/main.go
- Scheme registration calls
- Controller setup code

Note: Cannot delete an API while webhooks exist. Delete webhooks first.
`
	subcmdMeta.Examples = fmt.Sprintf(
		`  # Delete the API for the Memcached kind
  %[1]s delete api --group cache --version v1alpha1 --kind Memcached

  # Delete with automatic confirmation (use with caution)
  %[1]s delete api --group cache --version v1alpha1 --kind Memcached --skip-confirmation
`, cliMeta.CommandName)
}

func (p *deleteAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	if p.options == nil {
		p.options = &goPlugin.Options{}
	}

	fs.BoolVar(&p.skipConfirmation, "skip-confirmation", false,
		"skip confirmation prompt before deleting files")
}

func (p *deleteAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *deleteAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	// For core/external types, we need to match by GVK only, not domain
	// Try to find the resource in config
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

	// Check if the resource has webhooks - cannot delete API if webhooks exist
	if configRes.Webhooks != nil && !configRes.Webhooks.IsEmpty() {
		return fmt.Errorf("cannot delete API %q: webhooks are configured for this resource. "+
			"Please delete the webhooks first using 'kubebuilder delete webhook'", p.resource.GVK)
	}

	// Check if resource was created with additional plugins (only if those plugins aren't in the current chain)
	if err := p.checkAdditionalPlugins(); err != nil {
		return err
	}

	// Copy relevant fields from config resource
	p.resource = &configRes

	return nil
}

// SetPluginChain sets the plugin chain for cross-plugin coordination
func (p *deleteAPISubcommand) SetPluginChain(chain []string) {
	// Store for checking if required plugins are present
	p.pluginChain = chain
}

// checkAdditionalPlugins detects if this resource was created with plugins beyond the default layout
// (e.g., deploy-image) and ensures those plugins are included in the deletion command.
// This guarantees complete cleanup of both files and plugin-specific metadata.
func (p *deleteAPISubcommand) checkAdditionalPlugins() error {
	// Generic structure to check if any plugin stores metadata for this resource
	type pluginResourceConfig struct {
		Resources []struct {
			Group   string `json:"group,omitempty"`
			Version string `json:"version"`
			Kind    string `json:"kind"`
		} `json:"resources,omitempty"`
	}

	// Known plugins that store resource-specific metadata in PROJECT file
	// New plugins should be added here if they track resources
	pluginsToCheck := []string{
		"deploy-image.go.kubebuilder.io/v1-alpha",
	}

	// Find which plugins have metadata for this resource
	requiredPlugins := []string{}
	for _, pluginKey := range pluginsToCheck {
		cfg := pluginResourceConfig{}
		if err := p.config.DecodePluginConfig(pluginKey, &cfg); err != nil {
			continue // Plugin not used in this project
		}

		for _, res := range cfg.Resources {
			if res.Group == p.resource.Group && res.Version == p.resource.Version && res.Kind == p.resource.Kind {
				requiredPlugins = append(requiredPlugins, pluginKey)
				break
			}
		}
	}

	if len(requiredPlugins) == 0 {
		return nil // No additional plugins used
	}

	// Check if required plugins are in the current command's plugin chain
	for _, reqPlugin := range requiredPlugins {
		if !slices.Contains(p.pluginChain, reqPlugin) {
			// Plugin missing from chain - show user the correct command
			shortNames := convertPluginKeysToShortNames(requiredPlugins)
			fullChain := "go/v4," + strings.Join(shortNames, ",")

			return fmt.Errorf("resource %q was created with additional plugin(s): %s\n\n"+
				"To properly clean up all files and metadata, use:\n"+
				"  kubebuilder delete api --group %s --version %s --kind %s --plugins=%s\n\n"+
				"Both go/v4 (for files) and %s (for metadata) will run in sequence",
				p.resource.GVK, strings.Join(shortNames, ", "),
				p.resource.Group, p.resource.Version, p.resource.Kind,
				fullChain, strings.Join(shortNames, ", "))
		}
	}

	return nil // All required plugins are in the chain
}

// convertPluginKeysToShortNames converts full plugin keys to user-friendly short names
// Example: "deploy-image.go.kubebuilder.io/v1-alpha" -> "deploy-image/v1-alpha"
func convertPluginKeysToShortNames(pluginKeys []string) []string {
	shortNames := make([]string, len(pluginKeys))
	for i, fullKey := range pluginKeys {
		if strings.HasPrefix(fullKey, "deploy-image") {
			shortNames[i] = "deploy-image/v1-alpha"
		} else {
			// Generic conversion for other plugins
			shortNames[i] = strings.ReplaceAll(fullKey, ".kubebuilder.io/", "/")
			shortNames[i] = strings.ReplaceAll(shortNames[i], ".go.kubebuilder.io/", "/")
		}
	}
	return shortNames
}

func (p *deleteAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	// Prompt for confirmation unless skipped
	if !p.skipConfirmation {
		if !p.confirmDeletion() {
			return fmt.Errorf("deletion cancelled by user")
		}
	}

	log.Info("Deleting API files...")

	multigroup := p.config.IsMultiGroup()

	// Build list of files to delete
	filesToDelete := []string{}

	// API types file
	kindLower := strings.ToLower(p.resource.Kind)
	var apiTypesPath string
	if multigroup && p.resource.Group != "" {
		apiTypesPath = filepath.Join("api", p.resource.Group, p.resource.Version,
			fmt.Sprintf("%s_types.go", kindLower))
	} else {
		apiTypesPath = filepath.Join("api", p.resource.Version,
			fmt.Sprintf("%s_types.go", kindLower))
	}
	filesToDelete = append(filesToDelete, apiTypesPath)

	// Controller files
	var controllerPath, controllerTestPath, controllerSuiteTestPath string
	if multigroup && p.resource.Group != "" {
		controllerPath = filepath.Join("internal", "controller", p.resource.Group,
			fmt.Sprintf("%s_controller.go", kindLower))
		controllerTestPath = filepath.Join("internal", "controller", p.resource.Group,
			fmt.Sprintf("%s_controller_test.go", kindLower))
		controllerSuiteTestPath = filepath.Join("internal", "controller", p.resource.Group,
			"suite_test.go")
	} else {
		controllerPath = filepath.Join("internal", "controller",
			fmt.Sprintf("%s_controller.go", kindLower))
		controllerTestPath = filepath.Join("internal", "controller",
			fmt.Sprintf("%s_controller_test.go", kindLower))
		controllerSuiteTestPath = filepath.Join("internal", "controller",
			"suite_test.go")
	}

	if p.resource.HasController() {
		filesToDelete = append(filesToDelete, controllerPath, controllerTestPath)

		// Delete suite_test.go if this is the last controller in this group/version
		if p.isLastControllerInGroup() {
			filesToDelete = append(filesToDelete, controllerSuiteTestPath)
		}
	}

	// Delete the files
	deletedFiles := []string{}
	failedFiles := []string{}

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

	// Remove the resource from the PROJECT file
	if err := p.config.RemoveResource(p.resource.GVK); err != nil {
		return fmt.Errorf("failed to remove resource from PROJECT file: %w", err)
	}

	// Clean up shared golang files (kustomize files handled by kustomize plugin)
	additionalDeleted := p.cleanupSharedAPIFiles(fs)
	deletedFiles = append(deletedFiles, additionalDeleted...)

	// Report results
	if len(failedFiles) > 0 {
		return fmt.Errorf("failed to delete some files: %v", failedFiles)
	}

	fmt.Printf("\nSuccessfully deleted API %s/%s (%s)\n",
		p.resource.Group, p.resource.Version, p.resource.Kind)
	fmt.Printf("Deleted %d file(s)\n", len(deletedFiles))

	return nil
}

func (p *deleteAPISubcommand) PostScaffold() error {
	// Build the list of manual cleanup tasks
	importAlias := strings.ToLower(p.resource.Group) + p.resource.Version

	fmt.Println()
	fmt.Println("Manual cleanup required in cmd/main.go:")
	if p.resource.HasAPI() {
		if p.config.IsMultiGroup() && p.resource.Group != "" {
			fmt.Printf("  - Remove import: %s \"<repo>/api/%s/%s\"\n",
				importAlias, p.resource.Group, p.resource.Version)
		} else {
			fmt.Printf("  - Remove import: %s \"<repo>/api/%s\"\n", importAlias, p.resource.Version)
		}
		fmt.Printf("  - Remove: utilruntime.Must(%s.AddToScheme(scheme))\n", importAlias)
	}

	if p.resource.HasController() {
		if !p.config.IsMultiGroup() {
			fmt.Println("  - Remove import: \"<repo>/internal/controller\"")
			fmt.Printf("  - Remove controller setup for: %s\n", p.resource.Kind)
		} else if p.resource.Group != "" {
			fmt.Printf("  - Remove import: %scontroller \"<repo>/internal/controller/%s\"\n",
				strings.ToLower(p.resource.Group), p.resource.Group)
			fmt.Printf("  - Remove controller setup for: %s\n", p.resource.Kind)
		}
	}

	fmt.Println()
	fmt.Println("Next: after removing the code above, run:")
	fmt.Println("$ go mod tidy")
	fmt.Println("$ make generate")
	fmt.Println("$ make manifests")

	return nil
}

func (p *deleteAPISubcommand) confirmDeletion() bool {
	fmt.Printf("\nWARNING: You are about to delete the API for %s/%s (%s)\n",
		p.resource.Group, p.resource.Version, p.resource.Kind)
	fmt.Println("This will remove:")
	fmt.Println("  - API type definitions")
	if p.resource.HasController() {
		fmt.Println("  - Controller and test files")
	}
	fmt.Println("  - Resource entry from PROJECT file")
	fmt.Println("  - Sample files and CRD kustomization (best effort)")
	fmt.Println("\nThis operation cannot be undone!")
	fmt.Print("\nAre you sure you want to continue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Error("Failed to read user input", "error", err)
		return false
	}

	response = response[:len(response)-1] // Remove newline
	return response == "y" || response == "Y" || response == "yes" || response == "YES"
}

// cleanupSharedAPIFiles removes shared Go files created for this API version (best effort)
// Note: Kustomize files (samples, RBAC, kustomization.yaml) are cleaned by kustomize/v2 plugin
// Returns list of successfully deleted files
func (p *deleteAPISubcommand) cleanupSharedAPIFiles(fs machinery.Filesystem) []string {
	deleted := []string{}
	multigroup := p.config.IsMultiGroup()

	// Delete groupversion_info.go if this is the last API in this version
	if p.isLastAPIInVersion() {
		var groupVersionPath string
		if multigroup && p.resource.Group != "" {
			groupVersionPath = filepath.Join("api", p.resource.Group, p.resource.Version, "groupversion_info.go")
		} else {
			groupVersionPath = filepath.Join("api", p.resource.Version, "groupversion_info.go")
		}

		if exists, _ := afero.Exists(fs.FS, groupVersionPath); exists {
			if err := fs.FS.Remove(groupVersionPath); err != nil {
				log.Warn("Failed to delete groupversion_info.go", "file", groupVersionPath, "error", err)
			} else {
				log.Info("Deleted groupversion_info.go", "file", groupVersionPath)
				deleted = append(deleted, groupVersionPath)
			}
		}
	}

	return deleted
}

// isLastAPIInVersion checks if this is the last API in this specific version
func (p *deleteAPISubcommand) isLastAPIInVersion() bool {
	resources, err := p.config.GetResources()
	if err != nil {
		return false
	}

	for _, res := range resources {
		// Skip the current resource being deleted
		if res.Group == p.resource.Group && res.Version == p.resource.Version && res.Kind == p.resource.Kind {
			continue
		}
		// Check if any other resource exists in the same group/version
		if res.Group == p.resource.Group && res.Version == p.resource.Version {
			return false
		}
	}

	return true
}

// isLastControllerInGroup checks if this is the last controller in this group
// Used to determine if controller suite_test.go should be deleted
func (p *deleteAPISubcommand) isLastControllerInGroup() bool {
	if !p.resource.HasController() {
		return false
	}

	resources, err := p.config.GetResources()
	if err != nil {
		return false
	}

	for _, res := range resources {
		// Skip the current resource being deleted
		if res.Group == p.resource.Group && res.Version == p.resource.Version && res.Kind == p.resource.Kind {
			continue
		}
		// For multigroup, check same group; for single group, check any controller
		if p.config.IsMultiGroup() {
			if res.Group == p.resource.Group && res.Controller {
				return false
			}
		} else {
			if res.Controller {
				return false
			}
		}
	}

	return true
}
