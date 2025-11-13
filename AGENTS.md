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
  cli/          CLI commands (init, create api, create webhook, edit, alpha)
  machinery/    Scaffolding engine (templates, markers, injectors, filesystem)
  model/        Resource and stage models
  plugin/       Plugin interfaces and utilities
  plugins/      Plugin implementations
    golang/v4/           Main Go operator scaffolding (used by default combined with kustomize/v2; see PluginBundle in cli/init.go)
    golang/deployimage/  Implements create api interface to generate code to deploy and manage container images with controller
    common/kustomize/v2/ Kustomize manifests (used by default combined with go/v4; see PluginBundle in cli/init.go)
    optional/helm/       Helm chart generation to distribute the projects (v1alpha; deprecated, v2alpha)
    optional/grafana/    Grafana dashboards
    optional/autoupdate/ Auto-update workflow
    external/            External plugin support
docs/book/      mdBook sources + tutorial samples
test/
  e2e/          End-to-end tests (v4, helm, deployimage, alpha*)
  testdata/     Testdata generation scripts
testdata/       Generated sample projects (DO NOT EDIT)
hack/docs/      Documentation generation scripts
```

## Critical Rules

### NEVER Manually Edit
- `testdata/` - regenerated via `make generate-testdata`
- `docs/book/**/testdata/` - regenerated via `make generate-docs`
- `*/dist/chart/` - regenerated via `make generate-charts`

### Always Run Before PR
```bash
make generate    # Regenerate all (testdata + docs + k8s version + tidy)
make lint-fix    # Auto-fix Go code style
make test-unit   # Verify unit tests pass
```

### File-Specific Requirements
- After editing `*.go` → `make lint-fix`
- After editing `*.md` → `make remove-spaces`
- After modifying scaffolding/templates → `make generate`

## Development Workflow

### Build & Install
```bash
make build    # Build to ./bin/kubebuilder
make install  # Copy to $(go env GOBIN)
```

### Generate Everything
```bash
make generate              # Master command (runs all below + tidy + remove-spaces)
make generate-testdata     # Recreate testdata/project-*
make generate-docs         # Regenerate docs samples & marker docs
make generate-charts       # Rebuild Helm charts
```

### Lint & Format
```bash
make lint       # Check only (golangci-lint + yamllint)
make lint-fix   # Auto-fix Go code
```

### Testing
```bash
make test-unit         # Fast unit tests (./pkg/..., ./test/e2e/utils/...)
make test-integration  # Integration tests
make test-features     # Feature tests
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
- `Init` - provides `init` subcommand
- `CreateAPI` - provides `create api` subcommand
- `CreateWebhook` - provides `create webhook` subcommand
- `Edit` - provides `edit` subcommand
- `Bundle` - groups multiple plugins

### Scaffolding Machinery
From `pkg/machinery/`:
- `Template` - file generation via Go templates
- `Inserter` - code injection at markers
- `Marker` - special comments (e.g., `// +kubebuilder:scaffold:imports`)
- `Filesystem` - abstraction over afero for testability

### Scaffolded Project Structure
`kubebuilder init` creates:
- `cmd/main.go` - Entry point (manager setup)
- `api/v1/*_types.go` - API definitions with `+kubebuilder` markers
- `internal/controller/*_controller.go` - Reconcile logic
- `config/` - Kustomize manifests (CRDs, RBAC, manager, webhooks)
- `Dockerfile`, `Makefile` - Build and deployment automation

### Reconciliation Pattern
Controllers implement `Reconcile(ctx, req) (ctrl.Result, error)`:
- **Idempotent** - Safe to run multiple times
- **Level-triggered** - React to current state, not events
- **Requeue on pending work** - Return `ctrl.Result{Requeue: true}`

### Testing Pattern
E2E tests use `test/e2e/utils/test_context.go`:
```go
ctx := utils.NewTestContext("kubebuilder", "GO111MODULE=on")
ctx.Init()                    // Run kubebuilder init
ctx.CreateAPI(...)            // Run create api
ctx.Make("build")             // Run make targets
ctx.LoadImageToKindCluster()  // Load image to kind
```

## Tool Commands

### CLI Commands
```bash
kubebuilder init --domain example.com --repo github.com/example/myproject
kubebuilder create api --group batch --version v1 --kind CronJob
kubebuilder create webhook --group batch --version v1 --kind CronJob
kubebuilder edit --plugins=helm/v2-alpha
```

### Alpha Commands (Experimental)
```bash
kubebuilder alpha generate  # Generate from existing PROJECT file
kubebuilder alpha update    # Update to latest plugin versions
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

### Scaffolding
- Use library helpers from `pkg/plugin/util/`
- Use markers for extensibility
- Follow existing template patterns in `pkg/machinery`

## Search Tips

```bash
# Use rg (ripgrep) for searching
rg "pattern" --type go
rg "\\+kubebuilder:scaffold" --type go  # Find markers to inject code via Machinery
rg "\\+kubebuilder" --type go  # Find all markers
rg "type.*Plugin struct" pkg/plugins/   # Find plugins
```

## Design Philosophy

- **Libraries over code generation** - Use libraries when possible; generated code is hard to maintain
- **Common cases easy, uncommon cases possible** - 80-90% use cases should be simple
- **Batteries included** - Projects should be deployable/testable out-of-box
- **No copy-paste** - Refactor into libraries or remote Kustomize bases

## References

- `Makefile` - All automation targets (source of truth for commands)
- `CONTRIBUTING.md` - CLA, pre-submit checklist, PR emoji policy
- `VERSIONING.md` - Release workflow and PR tagging
- `docs/book/` - User documentation (https://book.kubebuilder.io)
- `test/e2e/utils/test_context.go` - E2E test helpers
