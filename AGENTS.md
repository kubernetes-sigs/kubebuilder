# Kubebuilder AI Agent Guide

**Kubebuilder** is a **framework** and **command-line interface (CLI)** for building **Kubernetes APIs** using **Custom Resource Definitions (CRDs)**.
It provides scaffolding and abstractions that accelerate the development of **controllers**, **webhooks**, and **APIs** written in **Go**.

## Quick Reference

| Item       | Value                                                     |
|------------|-----------------------------------------------------------|
| Language   | Defined in the go.mod                                     |
| Module     | `sigs.k8s.io/kubebuilder/v4`                              |
| Binary     | `./bin/kubebuilder`                                       |
| Core deps  | `controller-runtime`, `controller-tools`, Helm, Kustomize |
| Docs       | https://book.kubebuilder.io                               |


## Directory Map

```
pkg/
  cli/              CLI command implementations
    alpha/          Alpha/experimental commands (generate, update, etc.)
    init.go         'init' command + default PluginBundle definition
    api.go          'create api' command
    webhook.go      'create webhook' command
    edit.go         'edit' command
    root.go         Root command setup
  machinery/        Scaffolding engine (templates, markers, injectors)
    template.go     Base template interface
    inserter.go     Code injection engine
    marker.go       Marker detection and processing
    filesystem.go   Filesystem abstraction (uses afero)
  model/
    resource/       Resource model (GVK, API, Controller, Webhook)
    stage/          Plugin execution stages
  plugin/           Plugin interfaces and utilities
    interface.go    Core plugin interfaces (Plugin, Init, CreateAPI, etc.)
    bundle.go       Plugin composition
    util/           Helper functions for plugin authors
  plugins/          Plugin implementations (ADD NEW PLUGINS HERE)
    golang/v4/      Main Go scaffolding (default for go projects)
      scaffolds/    Scaffolding for init, api, webhook
        internal/templates/  Template implementations
    golang/deployimage/  Deploy-image pattern plugin
    common/kustomize/v2/  Kustomize manifest generation (default)
    optional/       Optional plugins (enabled via --plugins flag)
      helm/         Helm chart generation (v1alpha deprecated, v2alpha current)
      grafana/      Grafana dashboard generation
      autoupdate/   Auto-update GitHub workflow
    external/       External plugin support (exec-based plugins)
docs/book/          mdBook documentation (https://book.kubebuilder.io)
  src/              Markdown source files
    **/testdata/    Sample projects used in docs (regenerated)
test/
  e2e/              E2E tests requiring Kubernetes cluster
    v4/             Tests for v4 plugin
    helm/           Tests for Helm plugin
    deployimage/    Tests for deploy-image plugin
    utils/          Test helpers (TestContext, etc.)
  testdata/         Scripts to generate testdata projects
    generate.sh     Main generation script
    test.sh         Tests all testdata projects
testdata/           Generated complete sample projects (DO NOT EDIT)
  project-v4/                    Basic v4 project
  project-v4-multigroup/         Multigroup project
  project-v4-with-plugins/       Project with optional plugins
hack/docs/          Documentation generation
  generate.sh       Regenerate docs samples + marker docs
  generate_samples.go  Sample generation logic
cmd/                CLI entry point
  version.go        Version info (updated by make update-k8s-version)
main.go             Application entry point
```

**Key Locations for Common Tasks:**
- Add new plugin → `pkg/plugins/<category>/<name>/`
- Add new template → `pkg/plugins/<plugin>/scaffolds/internal/templates/`
- Modify CLI commands → `pkg/cli/`
- Add scaffolding machinery → `pkg/machinery/`
- Add tests → `test/e2e/<plugin>/` or `pkg/<package>/*_test.go`

## Critical Rules

### Do Not Manually Edit Generated Files
- `testdata/` - regenerated via `make generate-testdata`
- `docs/book/**/testdata/` - regenerated via `make generate-docs`
- `*/dist/chart/` - regenerated via `make generate-charts`

### File-Specific Requirements

After making changes, run the appropriate commands based on what you modified:

**Generate Commands (rebuild artifacts):**
- **If you modify files in `hack/docs/internal/`** → run `make install && make generate-docs`
- **If you modify files in `pkg/plugins/optional/helm/`** → run `make install && make generate-charts`
- **If you modify any boilerplate/template files** → run `make install && make generate`

**Formatting Commands:**
- After editing `*.go` → `make lint-fix`
- After editing `*.md` → `make remove-spaces`

**Always Run Before PR:**
```bash
make lint-fix    # Auto-fix Go code style
make test-unit   # Verify unit tests pass
```

**Note:** Boilerplate/template files are Go files that define scaffolding templates, typically located in `pkg/plugins/**/scaffolds/internal/templates/` or files that generate code/configs for scaffolded projects.

## Development Workflow

### Build & Install
```bash
make build    # Build to ./bin/kubebuilder
make install  # Copy to $(go env GOBIN)
```

### Lint & Format
```bash
make lint       # Check only (golangci-lint + yamllint)
make lint-fix   # Auto-fix Go code
```

### Testing
```bash
make test-unit         # Fast unit tests (./pkg/..., ./test/e2e/utils/...)
make test-integration  # Integration tests (may create temp dirs, download binaries)
make test-testdata     # Test all testdata projects
make test-e2e-local    # Full e2e (creates kind cluster)
make test              # CI aggregate (all of above + license)
```

## PR Submission

### PR Title Format (MANDATORY)

PR titles use **emojis** (appear in release notes).

Format: `:emoji: [(plugin/version)]: Description`

The `(plugin/version)` scope is optional; omit it for repo-wide or documentation-only changes.

**Emojis:**
- ⚠️ (`:warning:`) - Breaking change
- ✨ (`:sparkles:`) - New feature
- 🐛 (`:bug:`) - Bug fix
- 📖 (`:book:`) - Documentation
- 🌱 (`:seedling:`) - Infrastructure/tests/refactor

**Examples:**
```
🐛 Resolve nil pointer panic in scaffold generator
✨ (helm/v2-alpha): Add cluster-scoped resource support
📖 (go/v4): Update deployment documentation
✨ Update dependencies to latest versions
```

### Commit Message Format

Commit messages follow the [Conventional Commits](https://www.conventionalcommits.org/) standard.

Format: `<type>[optional scope]: <description>`

The `[optional scope]` is typically the plugin/version (e.g., `helm/v2-alpha`, `go/v4`); omit it for repo-wide or non-plugin changes.

**Types:**

- **feat**: A new feature for the user or a plugin
- **fix**: A bug fix for the user or a plugin
- **docs**: Documentation changes only
- **test**: Adding or updating tests
- **refactor**: Code change that neither fixes a bug nor adds a feature
- **chore**: Changes to build process, dependencies, or maintenance tasks
- **breaking**: A breaking change (can be combined with other types)

**Examples:**
```
fix: Resolve nil pointer panic in scaffold generator
feat(helm/v2-alpha): Add cluster-scoped resource support
docs(go/v4): Update deployment documentation
chore: Update dependencies to latest versions
```

### Pre-PR Checklist
- [ ] One commit per PR (squash all)
- [ ] Add/update tests for new behavior
- [ ] Add/update docs for new behavior
- [ ] Run `make lint-fix`
- [ ] Run `make install`
- [ ] Run `make generate`
- [ ] Run `make test-unit`

## Core Concepts

### Plugin Architecture

Plugins implement interfaces from `pkg/plugin/`:
- `Plugin` - base interface (Name, Version, SupportedProjectVersions)
- `Init` - project initialization (`kubebuilder init`)
- `CreateAPI` - API creation (`kubebuilder create api`)
- `CreateWebhook` - webhook creation (`kubebuilder create webhook`)
- `Edit` - post-init modifications (`kubebuilder edit`)
- `Bundle` - groups multiple plugins

**Plugin Bundles:**

Default bundle (`pkg/cli/init.go`): `go.kubebuilder.io/v4` + `kustomize.common.kubebuilder.io/v2`

Plugins resolve via `pkg/plugin` registry and execute in order.

**External Plugins:**

Executable binaries in `pkg/plugins/external/` that communicate via JSON over stdin/stdout.

### Scaffolding Machinery

From `pkg/machinery/`:
- `Template` - file generation via Go templates
- `Inserter` - code injection at markers
- `Marker` - special comments (e.g., `// +kubebuilder:scaffold:imports`)
- `Filesystem` - abstraction over afero for testability

### Scaffolded Project Structure

Projects generated by the Kubebuilder CLI use the default plugin bundle (`go/v4` + `kustomize/v2`). Each plugin scaffolds different files:

**`go/v4` plugin scaffolds Go code:**
- `cmd/main.go` - Entry point (manager setup)
- `api/v1/*_types.go` - API definitions with `+kubebuilder` markers (via `create api`)
- `internal/controller/*_controller.go` - Reconcile logic (via `create api`)
- `Dockerfile`, `Makefile` - Build and deployment automation

**`kustomize/v2` plugin scaffolds manifests:**
- `config/` - Kustomize base manifests (CRDs, RBAC, manager, webhooks)
- `config/crd/` - Custom Resource Definitions (via `create api`)
- `config/samples/` - Example CR manifests (via `create api`)

**`PROJECT` file:**
- Project configuration tracking plugins, resources, domain, and layout

**Note:** These are files in projects generated BY Kubebuilder, not the Kubebuilder source code itself.

### Reconciliation Pattern

Controllers implement `Reconcile(ctx, req) (ctrl.Result, error)`:

- **Idempotent** - Safe to run multiple times
- **Level-triggered** - React to current state, not events
- **Requeue on pending work** - Return `ctrl.Result{Requeue: true}`

### Testing Pattern
E2E tests use `utils.TestContext` from `test/e2e/utils/test_context.go`:

```go
ctx := utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
ctx.Init("--domain", "example.com", "--repo", "example.com/project")
ctx.CreateAPI("--group", "crew", "--version", "v1", "--kind", "Captain")
ctx.Make("build", "test")
ctx.LoadImageToKindCluster()
```

## CLI Reference

After `make install`:

```bash
kubebuilder init --domain example.com --repo github.com/example/myproject
kubebuilder create api --group batch --version v1 --kind CronJob
kubebuilder create webhook --group batch --version v1 --kind CronJob
kubebuilder edit --plugins=helm/v2-alpha
kubebuilder alpha generate    # Experimental: generate from PROJECT file
kubebuilder alpha update      # Experimental: update to latest plugin versions
```

## Common Patterns

### Code Style
- Avoid abbreviations: `context` not `ctx` (except receivers)
- Descriptive names: `projectConfig` not `pc`
- Single/double-letter receivers OK: `(c CLI)` or `(p Plugin)`

### Testing Philosophy
- Test behaviors, not implementations
- Use real components over mocks
- Test cases as specifications (Ginkgo: `Describe`, `It`, `Context`, `By`)
- Use **Ginkgo v2** + **Gomega** for BDD-style tests.
- Tests depending on the Kubebuilder binary should use: `utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")`

### Test Organization

- **Unit tests** (`*_test.go` in `pkg/`) - Test individual packages in isolation, fast
- **Integration tests** (`*_integration_test.go` in `pkg/`) - Test multiple components together without cluster
  - Must have `//go:build integration` tag at the top
  - May create temp dirs, download binaries, or scaffold files
  - Examples: alpha update, grafana scaffolding, helm chart generation
- **E2E tests** (`test/e2e/`) - **ONLY** for tests requiring a Kubernetes cluster (KIND)
  - `v4/plugin_cluster_test.go` - Test v4 plugin deployment
  - `helm/plugin_cluster_test.go` - Test Helm chart deployment
  - `deployimage/plugin_cluster_test.go` - Test deploy-image plugin

### Scaffolding
- Use library helpers from `pkg/plugin/util/`
- Use markers for extensibility
- Follow existing template patterns in `pkg/machinery`

## Search Tips

```bash
rg "\\+kubebuilder:scaffold" --type go  # Find markers
rg "type.*Plugin struct" pkg/plugins/   # Plugin implementations
rg "PluginBundle" pkg/cli/              # Plugin registration
rg "func.*SetTemplateDefaults"          # Template definitions
rg "func new.*Command" pkg/cli/         # CLI commands
rg "NewTestContext" test/e2e/           # E2E test setup
```

## Design Philosophy

- **Libraries over code generation** - Use libraries when possible; generated code is hard to maintain
- **Common cases easy, uncommon cases possible** - 80-90% use cases should be simple
- **Batteries included** - Projects should be deployable/testable out-of-box
- **No copy-paste** - Refactor into libraries or remote Kustomize bases

## References

### Essential Files
- **`Makefile`** - All automation targets (source of truth for build/test commands)
- **`CONTRIBUTING.md`** - CLA, pre-submit checklist, PR requirements
- **`VERSIONING.md`** - Release workflow, versioning policy, PR tagging
- **`go.mod`** - Go version and dependencies

### Key Directories
- **`pkg/`** - Core Kubebuilder code (CLI, plugins, machinery)
- **`test/e2e/`** - End-to-end tests with Kubernetes cluster
- **`testdata/`** - Generated sample projects (regenerated automatically)
- **`docs/book/`** - User documentation source (https://book.kubebuilder.io)

### Important Code Files
- **`pkg/cli/init.go`** - Default plugin bundle definition
- **`pkg/plugin/interface.go`** - Plugin interface definitions
- **`pkg/machinery/scaffold.go`** - Scaffolding engine
- **`test/e2e/utils/test_context.go`** - E2E test helpers
- **`cmd/version.go`** - Version info (includes K8S version)

### Scripts
- **`test/testdata/generate.sh`** - Regenerate all testdata projects
- **`hack/docs/generate.sh`** - Regenerate documentation samples
- **`test/e2e/local.sh`** - Run e2e tests locally with Kind

### External Resources
- **Kubebuilder Book**: https://book.kubebuilder.io
- **Kubebuilder Repo**: https://github.com/kubernetes-sigs/kubebuilder
- **controller-runtime**: https://github.com/kubernetes-sigs/controller-runtime
- **controller-tools**: https://github.com/kubernetes-sigs/controller-tools
- **API Conventions**: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md
- **Operator Pattern**: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
