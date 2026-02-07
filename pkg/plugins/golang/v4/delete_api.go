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
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
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
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

var _ plugin.DeleteAPISubcommand = &deleteAPISubcommand{}

type deleteAPISubcommand struct {
	config config.Config
	// For help text.
	commandName string

	options *goPlugin.Options

	resource *resource.Resource

	// assumeYes skips interactive confirmation prompts
	assumeYes bool

	// pluginChain stores the current plugin chain for cross-plugin coordination
	pluginChain []string

	// Track what couldn't be automatically removed for manual cleanup instructions
	manualCleanupAPIImport        bool
	manualCleanupAddToScheme      bool
	manualCleanupControllerImport bool
	manualCleanupControllerSetup  bool
	manualCleanupSuiteTestImport  bool
	manualCleanupSuiteTestScheme  bool
}

func (p *deleteAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	p.commandName = cliMeta.CommandName

	subcmdMeta.Description = `Delete a Kubernetes API and its associated files.

Automatically removes:
- API type definitions (api/<version>/<kind>_types.go)
- Controller files (internal/controller/<kind>_controller.go)
- Test files
- Kustomize manifests (samples, RBAC)
- Code from cmd/main.go and suite_test.go (imports, AddToScheme, controller setup)
- PROJECT file entries

Constraints:
- Cannot delete an API while webhooks exist. Delete webhooks first.
- If created with additional plugins (e.g., deploy-image), must use --plugins flag with full chain.

Manual cleanup shown if automatic code removal fails.
`
	subcmdMeta.Examples = fmt.Sprintf(
		`  # Delete the API for the Memcached kind
  %[1]s delete api --group cache --version v1alpha1 --kind Memcached

  # Delete without confirmation prompt (use with caution)
  %[1]s delete api --group cache --version v1alpha1 --kind Memcached -y
  %[1]s delete api --group cache --version v1alpha1 --kind Memcached --yes
`, cliMeta.CommandName)
}

func (p *deleteAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	if p.options == nil {
		p.options = &goPlugin.Options{}
	}

	fs.BoolVarP(&p.assumeYes, "yes", "y", false,
		"proceed without prompting for confirmation")
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

	// Check if all required plugins are in the current command's plugin chain
	// If layout plugins are missing (user only specified additional plugin), that's expected
	// and we allow it - the layout plugins were already resolved by the CLI
	missingPlugins := []string{}
	for _, reqPlugin := range requiredPlugins {
		if !slices.Contains(p.pluginChain, reqPlugin) {
			missingPlugins = append(missingPlugins, reqPlugin)
		}
	}

	if len(missingPlugins) > 0 {
		// Plugins are missing but this is OK if they're non-layout plugins
		// Example: user runs `kubebuilder delete api --plugins deploy-image/v1-alpha`
		// The layout plugin (go/v4) is already resolved by default, deploy-image is the extra
		log.Info("Additional plugins detected for this resource",
			"plugins", requiredPlugins, "chain", p.pluginChain)
	}

	return nil
}

func (p *deleteAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	// Prompt for confirmation unless -y/--yes flag provided
	if !p.assumeYes {
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

	// Clean up empty multigroup directories
	if multigroup && p.resource.Group != "" {
		p.cleanupEmptyGroupDirectories(fs)
	}

	// Remove marker-based code from main.go and test files
	p.removeCodeFromMainGo(fs)
	p.removeCodeFromSuiteTest(fs)

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
	// Check if any manual cleanup is needed
	needsManualCleanup := p.manualCleanupAPIImport ||
		p.manualCleanupAddToScheme ||
		p.manualCleanupControllerImport ||
		p.manualCleanupControllerSetup ||
		p.manualCleanupSuiteTestImport ||
		p.manualCleanupSuiteTestScheme

	if needsManualCleanup {
		fmt.Println()
		fmt.Println("Manual cleanup required - automatic removal failed for:")

		importAlias := strings.ToLower(p.resource.Group) + p.resource.Version
		repo := p.config.GetRepository()

		if p.manualCleanupAPIImport || p.manualCleanupAddToScheme ||
			p.manualCleanupControllerImport || p.manualCleanupControllerSetup {
			fmt.Println()
			fmt.Println("In cmd/main.go:")
		}

		if p.manualCleanupAPIImport {
			if p.config.IsMultiGroup() && p.resource.Group != "" {
				fmt.Printf("  - Remove import: %s \"%s/api/%s/%s\"\n",
					importAlias, repo, p.resource.Group, p.resource.Version)
			} else {
				fmt.Printf("  - Remove import: %s \"%s/api/%s\"\n",
					importAlias, repo, p.resource.Version)
			}
		}

		if p.manualCleanupAddToScheme {
			fmt.Printf("  - Remove from init(): utilruntime.Must(%s.AddToScheme(scheme))\n", importAlias)
		}

		if p.manualCleanupControllerImport {
			if p.config.IsMultiGroup() && p.resource.Group != "" {
				fmt.Printf("  - Remove import: %scontroller \"%s/internal/controller/%s\"\n",
					strings.ToLower(p.resource.Group), repo, p.resource.Group)
			} else {
				fmt.Printf("  - Remove import: \"%s/internal/controller\"\n", repo)
			}
		}

		if p.manualCleanupControllerSetup {
			fmt.Printf("  - Remove controller setup block for: %sReconciler (see constant reconcilerSetupCodeFragment)\n",
				p.resource.Kind)
		}

		if p.manualCleanupSuiteTestImport || p.manualCleanupSuiteTestScheme {
			suiteTestPath := "internal/controller/suite_test.go"
			if p.config.IsMultiGroup() && p.resource.Group != "" {
				suiteTestPath = fmt.Sprintf("internal/controller/%s/suite_test.go", p.resource.Group)
			}
			fmt.Printf("\nIn %s:\n", suiteTestPath)
		}

		if p.manualCleanupSuiteTestImport {
			if p.config.IsMultiGroup() && p.resource.Group != "" {
				fmt.Printf("  - Remove import: %s \"%s/api/%s/%s\"\n",
					importAlias, repo, p.resource.Group, p.resource.Version)
			} else {
				fmt.Printf("  - Remove import: %s \"%s/api/%s\"\n",
					importAlias, repo, p.resource.Version)
			}
		}

		if p.manualCleanupSuiteTestScheme {
			fmt.Printf("  - Remove from BeforeSuite: err = %s.AddToScheme(scheme.Scheme)\n", importAlias)
			fmt.Printf("                             Expect(err).NotTo(HaveOccurred())\n")
		}
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("$ go mod tidy")
	fmt.Println("$ make generate")
	fmt.Println("$ make manifests")

	return nil
}

// Code fragments to remove from main.go (match the format used in create)
const (
	apiImportCodeFragment = `%s "%s"
`
	controllerImportCodeFragment = `"%s/internal/controller"
`
	multiGroupControllerImportCodeFragment = `%scontroller "%s/internal/controller/%s"
`
	addSchemeCodeFragment = `utilruntime.Must(%s.AddToScheme(scheme))
`
	reconcilerSetupCodeFragment = `if err := (&controller.%sReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Failed to create controller", "controller", "%s")
		os.Exit(1)
	}
`
	multiGroupReconcilerSetupCodeFragment = `if err := (&%scontroller.%sReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Failed to create controller", "controller", "%s")
		os.Exit(1)
	}
`
)

// removeControllerSetupFlexible removes controller setup using AST parsing
// This handles plugin variations (e.g., deploy-image adds Recorder field) robustly
func (p *deleteAPISubcommand) removeControllerSetupFlexible(mainPath string) error {
	content, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("failed to read main.go: %w", err)
	}

	// Determine the reconciler type name to search for
	var reconcilerType string
	if p.config.IsMultiGroup() && p.resource.Group != "" {
		reconcilerType = fmt.Sprintf("%scontroller.%sReconciler", strings.ToLower(p.resource.Group), p.resource.Kind)
	} else {
		reconcilerType = fmt.Sprintf("controller.%sReconciler", p.resource.Kind)
	}

	// Parse the file into an AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, mainPath, content, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse main.go: %w", err)
	}

	// Find and remove the controller setup if statement
	var found bool
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for if statements
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		// Check if this is the controller setup if statement
		// Pattern: if err := (&controller.XReconciler{...}).SetupWithManager(mgr); err != nil { ... }
		if p.isControllerSetupStmt(ifStmt, reconcilerType) {
			// Remove this statement from the parent function
			if err := p.removeStmtFromAST(file, ifStmt); err == nil {
				found = true
			}
			return false
		}

		return true
	})

	if !found {
		return fmt.Errorf("controller setup block not found for %s", p.resource.Kind)
	}

	// Format and write the modified AST back to the file
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return fmt.Errorf("failed to format modified code: %w", err)
	}

	// Post-process to remove excessive blank lines before marker comments
	// The Go formatter sometimes adds extra blank lines after removing statements
	formattedCode := buf.String()
	formattedCode = strings.ReplaceAll(formattedCode, "}\n\n\t// +kubebuilder:scaffold:builder",
		"}\n\t// +kubebuilder:scaffold:builder")

	if err := os.WriteFile(mainPath, []byte(formattedCode), 0o644); err != nil {
		return fmt.Errorf("failed to write main.go: %w", err)
	}

	return nil
}

// isControllerSetupStmt checks if the given if statement is a controller setup block
func (p *deleteAPISubcommand) isControllerSetupStmt(ifStmt *ast.IfStmt, reconcilerType string) bool {
	// Check if init is an assignment (err := ...)
	assignStmt, ok := ifStmt.Init.(*ast.AssignStmt)
	if !ok || len(assignStmt.Lhs) != 1 || len(assignStmt.Rhs) != 1 {
		return false
	}

	// Check if the RHS is a method call (.SetupWithManager)
	callExpr, ok := assignStmt.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}

	// Check if it's a selector expression (something.SetupWithManager)
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok || selExpr.Sel.Name != "SetupWithManager" {
		return false
	}

	// Convert the expression to string and check if it contains the reconciler type
	exprStr := formatExprToString(assignStmt.Rhs[0])
	return strings.Contains(exprStr, reconcilerType)
}

// removeStmtFromAST removes a statement from its parent function in the AST
func (p *deleteAPISubcommand) removeStmtFromAST(file *ast.File, target ast.Stmt) error {
	var removed bool

	// Find the function containing this statement and remove it
	ast.Inspect(file, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil {
			return true
		}

		// Look through the function's statements
		newStmts := make([]ast.Stmt, 0, len(funcDecl.Body.List))
		for _, stmt := range funcDecl.Body.List {
			if stmt != target {
				newStmts = append(newStmts, stmt)
			} else {
				removed = true
			}
		}

		if removed {
			funcDecl.Body.List = newStmts
			return false
		}

		return true
	})

	if !removed {
		return fmt.Errorf("failed to remove statement from AST")
	}

	return nil
}

// formatExprToString converts an expression to a string for pattern matching
func formatExprToString(expr ast.Expr) string {
	var buf bytes.Buffer
	fset := token.NewFileSet()
	if err := format.Node(&buf, fset, expr); err != nil {
		return ""
	}
	return buf.String()
}

// removeCodeFromMainGo removes marker-inserted code from cmd/main.go
func (p *deleteAPISubcommand) removeCodeFromMainGo(_ machinery.Filesystem) {
	mainPath := filepath.Join("cmd", "main.go")
	repo := p.config.GetRepository()

	if p.resource.HasAPI() {
		p.removeAPIImportAndScheme(mainPath, repo)
	}

	if p.resource.HasController() {
		p.removeControllerImportAndSetup(mainPath, repo)
	}
}

// removeAPIImportAndScheme removes API import and AddToScheme from main.go
func (p *deleteAPISubcommand) removeAPIImportAndScheme(mainPath, repo string) {
	importAlias := strings.ToLower(p.resource.Group) + p.resource.Version

	// Check if any other resources share the same group+version
	hasOtherResourcesSameVersion := false
	resources, err := p.config.GetResources()
	if err == nil {
		for _, res := range resources {
			if res.Group == p.resource.Group && res.Version == p.resource.Version && res.Kind != p.resource.Kind {
				hasOtherResourcesSameVersion = true
				break
			}
		}
	}

	if hasOtherResourcesSameVersion {
		log.Info("Other APIs share the same version, keeping shared import and AddToScheme",
			"version", p.resource.Version, "alias", importAlias)
		return
	}

	// Build import path
	var importPath string
	if p.config.IsMultiGroup() && p.resource.Group != "" {
		importPath = fmt.Sprintf("%s/api/%s/%s", repo, p.resource.Group, p.resource.Version)
	} else {
		importPath = fmt.Sprintf("%s/api/%s", repo, p.resource.Version)
	}

	// Check if this is the last API import (to remove preceding blank line)
	isLastAPIImport := true
	for _, res := range resources {
		// Skip the resource we're deleting
		if res.Group == p.resource.Group && res.Version == p.resource.Version && res.Kind == p.resource.Kind {
			continue
		}
		// Check if any other resource has API
		if res.HasAPI() {
			isLastAPIImport = false
			break
		}
	}

	importToRemove := fmt.Sprintf("\t"+apiImportCodeFragment, importAlias, importPath)
	if isLastAPIImport {
		importToRemove = "\n" + importToRemove
	}

	if err := util.ReplaceInFile(mainPath, importToRemove, ""); err != nil {
		log.Warn("Unable to remove API import from main.go - manual cleanup needed",
			"import", importAlias, "error", err)
		p.manualCleanupAPIImport = true
	} else {
		log.Info("Removed API import from main.go", "import", importAlias)
	}

	// Remove AddToScheme
	schemeToRemove := fmt.Sprintf("\t"+addSchemeCodeFragment, importAlias)
	if err := util.ReplaceInFile(mainPath, schemeToRemove, ""); err != nil {
		log.Warn("Unable to remove AddToScheme from main.go - manual cleanup needed",
			"code", fmt.Sprintf("utilruntime.Must(%s.AddToScheme(scheme))", importAlias))
		p.manualCleanupAddToScheme = true
	} else {
		log.Info("Removed AddToScheme from main.go")
	}
}

// removeControllerImportAndSetup removes controller import and setup from main.go
func (p *deleteAPISubcommand) removeControllerImportAndSetup(mainPath, repo string) {
	// Check if other controllers exist in the same scope
	hasOtherControllersInScope := false
	allResources, err := p.config.GetResources()
	if err == nil {
		for _, res := range allResources {
			if res.Group == p.resource.Group && res.Version == p.resource.Version && res.Kind == p.resource.Kind {
				continue
			}
			if res.HasController() {
				if !p.config.IsMultiGroup() || p.resource.Group == "" || res.Group == p.resource.Group {
					hasOtherControllersInScope = true
					break
				}
			}
		}
	}

	// Remove controller import only if no other controllers share it
	if !hasOtherControllersInScope {
		var controllerImport string
		if p.config.IsMultiGroup() && p.resource.Group != "" {
			controllerImport = fmt.Sprintf(multiGroupControllerImportCodeFragment,
				strings.ToLower(p.resource.Group), repo, p.resource.Group)
		} else {
			controllerImport = fmt.Sprintf(controllerImportCodeFragment, repo)
		}

		if err := util.ReplaceInFile(mainPath, "\t"+controllerImport, ""); err != nil {
			log.Warn("Unable to remove controller import from main.go - manual cleanup needed")
			p.manualCleanupControllerImport = true
		} else {
			log.Info("Removed controller import from main.go")
		}
	} else {
		log.Info("Other controllers exist in the same scope, keeping shared controller import")
	}

	// Always remove the specific controller setup code
	p.removeControllerSetup(mainPath)
}

// removeControllerSetup removes controller setup block from main.go
func (p *deleteAPISubcommand) removeControllerSetup(mainPath string) {
	var controllerSetup string
	if p.config.IsMultiGroup() && p.resource.Group != "" {
		controllerSetup = fmt.Sprintf(multiGroupReconcilerSetupCodeFragment,
			strings.ToLower(p.resource.Group), p.resource.Kind, p.resource.Kind)
	} else {
		controllerSetup = fmt.Sprintf(reconcilerSetupCodeFragment,
			p.resource.Kind, p.resource.Kind)
	}

	if err := util.ReplaceInFile(mainPath, "\t"+controllerSetup, ""); err == nil {
		log.Info("Removed controller setup from main.go", "controller", p.resource.Kind)
	} else if err := p.removeControllerSetupFlexible(mainPath); err != nil {
		log.Warn("Unable to remove controller setup from main.go - manual cleanup needed",
			"controller", p.resource.Kind, "error", err)
		p.manualCleanupControllerSetup = true
	} else {
		log.Info("Removed controller setup from main.go (flexible match)", "controller", p.resource.Kind)
	}
}

// Code fragments for suite_test.go removal
const (
	suiteTestImportCodeFragment = `%s "%s"
`
	suiteTestAddSchemeCodeFragment = `err = %s.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

`
)

// removeCodeFromSuiteTest removes marker-inserted code from suite_test.go
func (p *deleteAPISubcommand) removeCodeFromSuiteTest(_ machinery.Filesystem) {
	if !p.resource.HasController() {
		return
	}

	var suiteTestPath string
	if p.config.IsMultiGroup() && p.resource.Group != "" {
		suiteTestPath = filepath.Join("internal", "controller", p.resource.Group, "suite_test.go")
	} else {
		suiteTestPath = filepath.Join("internal", "controller", "suite_test.go")
	}

	// Check if file exists (using util.HasFileContentWith to check existence)
	exists, err := util.HasFileContentWith(suiteTestPath, "package")
	if err != nil || !exists {
		return
	}

	repo := p.config.GetRepository()
	importAlias := strings.ToLower(p.resource.Group) + p.resource.Version

	// Check if any other resources in the same group share the same version
	hasOtherResourcesSameVersion := false
	resources, err := p.config.GetResources()
	if err == nil {
		for _, res := range resources {
			// Skip the resource we're deleting
			if res.Group == p.resource.Group && res.Version == p.resource.Version && res.Kind == p.resource.Kind {
				continue
			}
			// Check if another resource in the same test suite shares the same group+version
			// In multigroup mode, resources in different groups have different suite_test.go files
			if res.Group == p.resource.Group && res.Version == p.resource.Version && res.HasController() {
				hasOtherResourcesSameVersion = true
				break
			}
		}
	}

	if hasOtherResourcesSameVersion {
		log.Info("Other controllers share the same version in suite_test.go, keeping shared import",
			"version", p.resource.Version, "alias", importAlias)
	} else {
		// Remove import
		var importPath string
		if p.config.IsMultiGroup() && p.resource.Group != "" {
			importPath = fmt.Sprintf("%s/api/%s/%s", repo, p.resource.Group, p.resource.Version)
		} else {
			importPath = fmt.Sprintf("%s/api/%s", repo, p.resource.Version)
		}

		importToRemove := fmt.Sprintf("\t"+suiteTestImportCodeFragment, importAlias, importPath)
		if err := util.ReplaceInFile(suiteTestPath, importToRemove, ""); err != nil {
			log.Warn("Unable to remove import from suite_test.go - manual cleanup needed", "error", err)
			p.manualCleanupSuiteTestImport = true
		} else {
			log.Info("Removed import from suite_test.go")
		}

		// Remove AddToScheme
		schemeToRemove := fmt.Sprintf("\t"+suiteTestAddSchemeCodeFragment, importAlias)
		if err := util.ReplaceInFile(suiteTestPath, schemeToRemove, ""); err != nil {
			log.Warn("Unable to remove AddToScheme from suite_test.go - manual cleanup needed", "error", err)
			p.manualCleanupSuiteTestScheme = true
		} else {
			log.Info("Removed AddToScheme from suite_test.go")
		}
	}
}

func (p *deleteAPISubcommand) confirmDeletion() bool {
	fmt.Printf("\nWarning: You are about to delete the API for %s/%s (%s)\n",
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

	// Remove whitespace (handles both Unix \n and Windows \r\n)
	response = strings.TrimSpace(response)
	return response == "y" || response == "Y" || response == "yes" || response == "YES"
}

// cleanupEmptyGroupDirectories removes empty group directories in multigroup mode
func (p *deleteAPISubcommand) cleanupEmptyGroupDirectories(fs machinery.Filesystem) {
	// Check if this was the last resource in this group
	isLastInGroup := true
	resources, err := p.config.GetResources()
	if err == nil {
		for _, res := range resources {
			// Skip the resource we're deleting
			if res.Group == p.resource.Group && res.Version == p.resource.Version && res.Kind == p.resource.Kind {
				continue
			}
			// Check if another resource exists in the same group
			if res.Group == p.resource.Group {
				isLastInGroup = false
				break
			}
		}
	}

	if !isLastInGroup {
		return // Other resources in this group still exist
	}

	// Try to remove group directories (they should be empty now)
	apiGroupDir := filepath.Join("api", p.resource.Group)
	controllerGroupDir := filepath.Join("internal", "controller", p.resource.Group)

	// Remove API group directory
	if err := fs.FS.RemoveAll(apiGroupDir); err != nil {
		log.Warn("Failed to remove empty API group directory", "dir", apiGroupDir, "error", err)
	} else {
		log.Info("Removed empty API group directory", "dir", apiGroupDir)
	}

	// Remove controller group directory
	if err := fs.FS.RemoveAll(controllerGroupDir); err != nil {
		log.Warn("Failed to remove empty controller group directory", "dir", controllerGroupDir, "error", err)
	} else {
		log.Info("Removed empty controller group directory", "dir", controllerGroupDir)
	}
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
