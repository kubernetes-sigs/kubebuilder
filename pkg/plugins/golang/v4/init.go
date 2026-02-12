/*
Copyright 2022 The Kubernetes Authors.

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
	"fmt"
	log "log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds"
)

// Variables and function to check Go version requirements.
var (
	goVerMin = golang.MustParse("go1.23.0")
	goVerMax = golang.MustParse("go2.0alpha1")
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config config.Config
	// For help text.
	commandName string

	// boilerplate options
	license     string
	owner       string
	licenseFile string

	// go config options
	repo string

	// flags
	fetchDeps          bool
	skipGoVersionCheck bool
	namespaced         bool
}

func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	p.commandName = cliMeta.CommandName

	subcmdMeta.Description = `Initialize a new project including the following files:
  - a "go.mod" with project dependencies
  - a "PROJECT" file that stores project configuration
  - a "Makefile" with several useful make targets for the project
  - several YAML files for project deployment under the "config" directory
  - a "cmd/main.go" file that creates the manager that will run the project controllers

Namespaced layout (--namespaced):
  Scaffolds the project for namespace-scoped deployment instead of cluster-scoped.
  - Creates Role/RoleBinding instead of ClusterRole/ClusterRoleBinding
  - Adds WATCH_NAMESPACE environment variable to manager deployment
  - New controllers will include namespace= in RBAC markers automatically
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Initialize a new project
  %[1]s init --plugins go/v4 --domain example.org --owner "Your name"

  # Initialize with namespaced layout (namespace-scoped)
  %[1]s init --plugins go/v4 --domain example.org --namespaced

  # Initialize with specific project version
  %[1]s init --plugins go/v4 --project-version 3

  # Initialize with custom license header from file
  %[1]s init --plugins go/v4 --domain example.org --license-file ./my-header.txt

  # Initialize with built-in license (apache2, none)
  %[1]s init --plugins go/v4 --domain example.org --license apache2
`, cliMeta.CommandName)
}

func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.skipGoVersionCheck, "skip-go-version-check",
		false, "if specified, skip checking the Go version")

	// dependency args
	fs.BoolVar(&p.fetchDeps, "fetch-deps", true, "ensure dependencies are downloaded")

	// boilerplate args
	fs.StringVar(&p.license, "license", "apache2",
		"license to use to boilerplate, may be one of 'apache2', 'none'"+
			" (see: https://book.kubebuilder.io/reference/license-header)")
	fs.StringVar(&p.owner, "owner", "", "owner to add to the copyright")
	fs.StringVar(&p.licenseFile, "license-file", "",
		"path to custom license file; content copied to hack/boilerplate.go.txt (overrides --license)")

	// project args
	fs.StringVar(&p.repo, "repo", "", "name to use for go module (e.g., github.com/user/repo), "+
		"defaults to the go package of the current working directory.")
	fs.BoolVar(&p.namespaced, "namespaced", false,
		"if specified, scaffold the project with namespaced layout (default: cluster-scoped)")
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	// Try to guess repository if flag is not set.
	if p.repo == "" {
		repoPath, err := golang.FindCurrentRepo()
		if err != nil {
			return fmt.Errorf("error finding current repository: %w", err)
		}
		p.repo = repoPath
	}

	if err := p.config.SetRepository(p.repo); err != nil {
		return fmt.Errorf("error setting repository: %w", err)
	}

	if p.namespaced {
		if err := p.config.SetNamespaced(); err != nil {
			return fmt.Errorf("error setting namespaced: %w", err)
		}
	}

	return nil
}

func (p *initSubcommand) PreScaffold(machinery.Filesystem) error {
	// Ensure Go version is in the allowed range if check not turned off.
	if !p.skipGoVersionCheck {
		if err := golang.ValidateGoVersion(goVerMin, goVerMax); err != nil {
			return fmt.Errorf("error validating go version: %w", err)
		}
	}

	// Trim whitespace from license file path and treat empty/whitespace-only as not provided
	p.licenseFile = strings.TrimSpace(p.licenseFile)

	// Validate license file exists and has proper format before scaffolding begins to prevent broken state
	if p.licenseFile != "" {
		if _, err := os.Stat(p.licenseFile); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("license file %q does not exist", p.licenseFile)
			}
			return fmt.Errorf("failed to access license file %q: %w", p.licenseFile, err)
		}

		// Validate that the license file is a valid Go comment block
		content, err := os.ReadFile(p.licenseFile)
		if err != nil {
			return fmt.Errorf("failed to read license file %q: %w", p.licenseFile, err)
		}

		// Only validate format if file is not empty (empty files are allowed)
		if len(content) > 0 {
			contentStr := strings.TrimSpace(string(content))
			if !strings.HasPrefix(contentStr, "/*") || !strings.HasSuffix(contentStr, "*/") {
				return fmt.Errorf("license file %q must be a valid Go comment block (start with /* and end with */)", p.licenseFile)
			}
		}
	}

	// Check if the current directory has no files or directories which does not allow to init the project
	return checkDir()
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewInitScaffolder(p.config, p.license, p.owner, p.licenseFile, p.commandName)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding init plugin: %w", err)
	}

	if !p.fetchDeps {
		log.Info("skipping fetching dependencies")
		return nil
	}

	// Ensure that we are pinning controller-runtime version
	// xref: https://github.com/kubernetes-sigs/kubebuilder/issues/997
	err := util.RunCmd("Get controller runtime", "go", "get",
		"sigs.k8s.io/controller-runtime@"+scaffolds.ControllerRuntimeVersion)
	if err != nil {
		return fmt.Errorf("error getting controller-runtime version: %w", err)
	}

	return nil
}

func (p *initSubcommand) PostScaffold() error {
	err := util.RunCmd("Update dependencies", "go", "mod", "tidy")
	if err != nil {
		return fmt.Errorf("error updating go dependencies: %w", err)
	}

	fmt.Printf("Next: define a resource with:\n$ %s create api\n", p.commandName)
	return nil
}

// checkDir checks the target directory before scaffolding:
// 1. Returns error if key kubebuilder files already exist (prevents re-initialization)
// 2. Warns if directory is not empty (but allows scaffolding to continue)
func checkDir() error {
	// Files scaffolded by 'kubebuilder init' that indicate the directory is already initialized.
	// Blocking these prevents accidental re-initialization and file conflicts.
	// Note: go.mod and go.sum are NOT blocked because:
	//   - They may exist in pre-existing Go projects
	//   - Kubebuilder will overwrite them (machinery.OverwriteFile)
	//   - Testdata generation creates go.mod before running init
	scaffoldedFiles := []string{
		"PROJECT",                       // Kubebuilder project config (key indicator)
		"Makefile",                      // Build automation
		filepath.Join("cmd", "main.go"), // Controller manager entry point
	}

	// Check for existing scaffolded files
	for _, file := range scaffoldedFiles {
		if _, err := os.Stat(file); err == nil {
			return fmt.Errorf("target directory is already initialized. "+
				"Found existing kubebuilder file %q. "+
				"Please run this command in a new directory or remove existing scaffolded files", file)
		}
	}

	// Check if directory has any other files (warn only)
	// Note: We ignore certain files that are expected or safely overwritten:
	//   - go.mod and go.sum: Users may run `go mod init` before `kubebuilder init`
	//   - .gitignore and .dockerignore: Safely overwritten by kubebuilder
	//   - Other dot directories (.git, .vscode, .idea): Not scaffolded by kubebuilder
	// However, we DO check .github directory since kubebuilder scaffolds workflows there
	var hasFiles bool
	err := filepath.Walk(".",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("error walking path %q: %w", path, err)
			}
			// Skip the current directory itself
			if path == "." {
				return nil
			}
			// Skip dot directories EXCEPT .github (which contains scaffolded workflows)
			if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
				if info.Name() != ".github" {
					return filepath.SkipDir
				}
			}
			// Skip files that are expected or safely overwritten
			ignoredFiles := []string{"go.mod", "go.sum"}
			if slices.Contains(ignoredFiles, info.Name()) {
				return nil
			}
			// Track if any other files/directories exist
			hasFiles = true
			return nil
		})
	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	// Warn if directory is not empty (but don't block)
	if hasFiles {
		log.Warn("The target directory is not empty. " +
			"Scaffolding may overwrite existing files or cause conflicts. " +
			"It is recommended to initialize in an empty directory.")
	}

	return nil
}
