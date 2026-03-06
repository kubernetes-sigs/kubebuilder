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
	license string
	owner   string

	// go config options
	repo string

	// flags
	fetchDeps          bool
	skipGoVersionCheck bool
	multigroup         bool
	namespaced         bool
}

func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	p.commandName = cliMeta.CommandName

	subcmdMeta.Description = `Initialize a new project in the current directory.
 
Following files will be generated automatically:
  - go.mod: Go module with project dependencies
  - PROJECT: file that stores project configuration
  - Makefile: provides useful make targets for the project
  - config/: Kubernetes manifests for deployment
  - cmd/main.go: controller manager entry point
  - Dockerfile: build controller manager container image
  - test/: unit tests for the project
  - hack/: contains licensing boilerplate.

Note:
	
    Below are some useful flags, see the Flags section for detailed descriptions.
	Required flags:      --domain
	Configuration flags: --repo, --owner, --license
	Plugin flags:        --plugins
	Layout flags:        --multigroup	
	
	Run 'kubebuilder init --plugins --help' to see available plugins.

	Layout settings can be changed later with 'kubebuilder edit'.

`
	subcmdMeta.Examples = fmt.Sprintf(`  # Initialize a new project
  %[1]s init --domain example.org

  # Initialize with multigroup layout
  %[1]s init --domain example.org --multigroup

  # Initialize with namespace-scoped deployment
  %[1]s init --domain example.org --namespaced

  # Initialize with optional plugins
  %[1]s init --plugins go/v4,autoupdate/v1-alpha --domain example.org
  %[1]s init --plugins go/v4,helm/v2-alpha --domain example.org

  # Initialize with custom settings
  %[1]s init --domain example.org --owner "Your Name" --license apache2

  # Initialize with all options combined
  %[1]s init --plugins go/v4,autoupdate/v1-alpha --domain example.org --multigroup --namespaced
`, cliMeta.CommandName)
}

func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.skipGoVersionCheck, "skip-go-version-check",
		false, "skip Go version check")

	// dependency args
	fs.BoolVar(&p.fetchDeps, "fetch-deps", true, "download dependencies after scaffolding")

	// boilerplate args
	fs.StringVar(&p.license, "license", "apache2",
		"[Configuration] license to use (apache2 or none, default: apache2)")
	fs.StringVar(&p.owner, "owner", "", "[Configuration] copyright owner for license headers")

	// project args
	fs.StringVar(&p.repo, "repo", "", "[Configuration] Go module path (e.g., github.com/user/repo); "+
		"auto-detected if not provided and project is initialized within $GOPATH")
	fs.BoolVar(&p.multigroup, "multigroup", false,
		"[Layout] Enable multigroup layout to organize APIs by group. "+
			"Scaffolds APIs in api/<group>/<version>/ instead of api/<version>/ "+
			"Useful when managing multiple API groups (e.g., batch, apps, crew). ")
	fs.BoolVar(&p.namespaced, "namespaced", false,
		"[Layout] Enable namespace-scoped deployment instead of cluster-scoped (default: cluster-scoped). "+
			"Manager watches one or more specific namespaces instead of all namespaces. "+
			"Namespaces to watch are configured via WATCH_NAMESPACE environment variable. "+
			"Uses Role/RoleBinding instead of ClusterRole/ClusterRoleBinding. "+
			"Suitable for multi-tenant environments or limited scope deployments.")
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

	if p.multigroup {
		if err := p.config.SetMultiGroup(); err != nil {
			return fmt.Errorf("error setting multigroup: %w", err)
		}
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

	// Check if the current directory has no files or directories which does not allow to init the project
	return checkDir()
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewInitScaffolder(p.config, p.license, p.owner, p.commandName)
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
