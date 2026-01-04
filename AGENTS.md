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

### Title Format (MANDATORY)
```
:emoji: (optional/scope): User-facing description
```

**Emojis:**
- ⚠️ - Breaking change
- ✨ - New feature
- 🐛 - Bug fix
- 📖 - Documentation
- 🌱 - Infrastructure/tests/non-user-facing/refactor

**Examples:**
```
✨ (helm/v2-alpha): Add chart generation for cluster-scoped resources
🐛: Fix project creation failure when GOBIN is unset
📖: Update migration guide for Go 1.25 compatibility
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
- `DeleteAPI` - API deletion (`kubebuilder delete api`)
- `DeleteWebhook` - webhook deletion (`kubebuilder delete webhook`)
- `Edit` - post-init modifications (`kubebuilder edit`)
- `Bundle` - groups multiple plugins

**Delete = Undo of Create:**

Each plugin's delete implementation MUST undo exactly what its create implementation did:
- `go/v4`: Removes Go code (API types, controllers, main.go imports/setup, suite_test.go)
- `kustomize/v2`: Removes manifests (samples, RBAC, CRD kustomization entries)
- `deploy-image/v1-alpha`: Removes plugin metadata from PROJECT file
- When plugins run in chain (e.g., `--plugins deploy-image/v1-alpha`), both layout and additional plugins execute

**Integration tests MUST verify**: `state_before_create == state_after_delete`

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

**Integration Tests** use `utils.TestContext` from `test/e2e/utils/test_context.go`:

```go
ctx := utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
ctx.Init("--domain", "example.com", "--repo", "example.com/project")
ctx.CreateAPI("--group", "crew", "--version", "v1", "--kind", "Captain")
ctx.Make("build", "test")
ctx.LoadImageToKindCluster()
```

**Baseline Testing (Required for Delete):**

Delete integration tests MUST verify exact state restoration:

```go
It("should restore exact state after delete", func() {
    mainBefore, _ := os.ReadFile("cmd/main.go")
    ctx.CreateAPI(...)
    ctx.DeleteAPI(..., "-y")
    mainAfter, _ := os.ReadFile("cmd/main.go")
    Expect(mainAfter).To(Equal(mainBefore))  // Exact match required
})
```

## CLI Reference

After `make install`:

```bash
# Initialize project
kubebuilder init --domain example.com --repo github.com/example/myproject

# Create resources
kubebuilder create api --group batch --version v1 --kind CronJob
kubebuilder create webhook --group batch --version v1 --kind CronJob

# Delete resources (complete undo of create)
kubebuilder delete api --group batch --version v1 --kind CronJob
kubebuilder delete webhook --group batch --version v1 --kind CronJob --defaulting

# Delete with plugin chain
kubebuilder delete api --group app --version v1 --kind Cache --plugins deploy-image/v1-alpha

# Delete optional plugin features
kubebuilder delete --plugins helm/v2-alpha
kubebuilder delete --plugins grafana/v1-alpha

# Edit project
kubebuilder edit --plugins helm/v2-alpha

# Alpha commands
kubebuilder alpha generate    # Experimental: generate from PROJECT file
kubebuilder alpha update      # Experimental: update to latest plugin versions
```

## Implementing Delete

**Rule**: If you add a `create` command, you MUST add the corresponding `delete` command.

**Key Principle**: Each plugin undoes ONLY what it created. When plugins run in chain (default: `go/v4` + `kustomize/v2`), each cleans its own artifacts:
- `go/v4` → removes Go code (types, controllers, main.go, suite_test.go)
- `kustomize/v2` → removes manifests (samples, RBAC, CRD entries)
- Additional plugins → remove their metadata from PROJECT file

**Shared Resources**: Imports/code used by multiple resources are preserved until the last one is deleted (e.g., `appv1` import kept while any app/v1 API exists).

**Integration Test**: Add `delete_integration_test.go` with baseline verification:
```go
baseline := captureState()
createResource()
deleteResource("-y")
Expect(currentState()).To(Equal(baseline))  // Exact match required
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
  - **Delete tests**: MUST use baseline pattern (verify before_create == after_delete)
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
