---
name: documentation-reference
description: Reference materials for Kubebuilder documentation standards
license: Apache-2.0
metadata:
  author: The Kubernetes Authors
---

# Documentation Reference

Technical references and detailed examples for Kubebuilder documentation standards.

**Documentation location**: `docs/book/src/` (built with mdBook)

This reference provides templates, examples, and patterns for contributing high-quality documentation to Kubebuilder.

## Official Style Guides

### Kubernetes Documentation Style Guide

Primary reference for all Kubebuilder documentation.

URL: https://kubernetes.io/docs/contribute/style/style-guide/

Key topics:
- Language and grammar
- Formatting standards
- Content organization
- Code examples
- Terminology

## Related Projects

### controller-runtime

Core library for building Kubernetes controllers.

- **Repository**: https://github.com/kubernetes-sigs/controller-runtime
- **Documentation**: https://pkg.go.dev/sigs.k8s.io/controller-runtime
- **When documenting**: Verify features by checking source code (see Verifying Dependency Information below)

### controller-tools

Tools for generating CRDs, webhooks, and RBAC from Go code.

- **Repository**: https://github.com/kubernetes-sigs/controller-tools
- **Documentation**: https://pkg.go.dev/sigs.k8s.io/controller-tools
- **Markers reference**: https://book.kubebuilder.io/reference/markers.html
- **When documenting**: Verify marker behavior by checking source code (see Verifying Dependency Information below)

## Documentation Build System

### mdBook

Kubebuilder documentation uses [mdBook](https://rust-lang.github.io/mdBook/) with custom preprocessors:

- **literatego** (`./litgo.sh`): Processes `{{#literatego path/to/file.go}}` includes
- **markerdocs** (`./markerdocs.sh`): Generates marker documentation

Configuration: `docs/book/book.toml`

### Building Locally

```bash
# Install mdBook (one time)
cd docs/book && ./install-and-build.sh

# Build docs
cd docs/book && mdbook build

# Serve with live reload
cd docs/book && mdbook serve
```

## Sample Generation

### Two Sources of Generated Files

Testdata projects are generated from **two different sources**:

#### 1. Tutorial-Specific Generators

**Location**: `hack/docs/internal/`

**Key files**:
- `generate.sh`: Main generation script called by `make generate-docs`
- `internal/cronjob-tutorial/`: CronJob tutorial generator
- `internal/getting-started/`: Getting Started generator
- `internal/multiversion-tutorial/`: Multiversion generator

**Generates**: Tutorial-specific customizations and examples

#### 2. Plugin Default Scaffold

**Location**: `pkg/plugins/`

**Key templates**:
- `pkg/plugins/golang/v4/scaffolds/internal/templates/agents.go` - Generates `AGENTS.md`
- `pkg/plugins/golang/v4/scaffolds/internal/templates/` - Other default boilerplate
- `pkg/plugins/optional/*/scaffolds/internal/templates/` - Optional plugin templates
- `pkg/plugins/common/kustomize/*/scaffolds/internal/templates/` - Kustomize config files

**Generates**: Default project structure files (Makefile, main.go, AGENTS.md, config/ kustomize files, etc.)

### Critical Workflow for Fixing Testdata

**If you find a documentation issue in testdata files**:

1. **Identify the source**:
   - Tutorial-specific content (custom examples, tutorial text) → Edit `hack/docs/internal/`
   - Default scaffold files (`AGENTS.md`, `config/` kustomize files, standard boilerplate) → Edit `pkg/plugins/`

2. **Fix the template source**:
   - Never edit generated files directly
   - Always edit the template that generates them

3. **Rebuild and regenerate**:
   ```bash
   make install         # Rebuild kubebuilder binary with template changes
   make generate-docs   # Regenerate all testdata using new binary
   ```

4. **Verify the fix**:
   ```bash
   git diff docs/book/src/*/testdata/project/  # Check generated files updated correctly
   ```

### Generated Locations

Auto-generated (DO NOT EDIT DIRECTLY):
- `docs/book/src/cronjob-tutorial/testdata/project/`
- `docs/book/src/getting-started/testdata/project/`
- `docs/book/src/multiversion-tutorial/testdata/project/`

### Commands

```bash
# Regenerate all samples (after template changes)
make install         # Required after changing pkg/plugins/ templates
make generate-docs   # Regenerates all testdata

# Fix accessibility and trailing spaces
make fix-docs

# Test tutorial code compiles
make test-book
```

## Link Examples

### Internal Links with Relative Paths

Good:
```markdown
See [Creating a controller](../cronjob-tutorial/controller-implementation.md) for details.
See [API markers](./markers.md) in the same directory.
```

Avoid:
```markdown
See [guide](/docs/book/src/getting-started.md) for details.
See [guide](https://kubebuilder.io/getting-started.md) for internal content.
```

### Link Aliases at Bottom

```markdown
Content with links to [Kubernetes API][k8s-api] and [contributing guide][contributing].

[k8s-api]: https://kubernetes.io/docs/reference/
[contributing]: ../CONTRIBUTING.md
```

### Descriptive Link Text

Good:
```markdown
See the [Kubernetes API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md).
```

Avoid:
```markdown
Click [here](https://github.com/...) for API conventions.
```

## Code Example Patterns

### Prefer Testdata Includes Over Inline Code

**Problem**: Inline code becomes outdated when the codebase changes.
**Solution**: Use `{{#include}}` or `{{#literatego}}` to reference testdata projects.

**Available testdata** (all tested by `make test-book`):
- `docs/book/src/cronjob-tutorial/testdata/project/`
- `docs/book/src/getting-started/testdata/project/`
- `docs/book/src/multiversion-tutorial/testdata/project/`
- `testdata/project-v4-with-plugins/`

### Include Shortcode Syntax

**For Go code** (adds syntax highlighting, handles imports):
```markdown
{{#literatego ./testdata/project/path/to/file.go}}
```

**For YAML, JSON, Makefile, shell** (includes raw):
```markdown
{{#include ./testdata/project/config/manager/manager.yaml}}
```

**For specific sections** (using anchors in source files):
```markdown
{{#include ./testdata/project/config/default/kustomization.yaml:webhook-resources}}
```

### Using Anchors in Tutorial Testdata

Anchors mark sections in testdata files for selective inclusion:

```go
// +kubebuilder:docs-gen:collapse=Apache License

// ANCHOR: imports
import (
    "context"
    ctrl "sigs.k8s.io/controller-runtime"
)
// ANCHOR_END: imports
```

Include the anchored section:
```markdown
{{#literatego ./testdata/project/main.go:imports}}
```

### Documentation Around Includes

**Pattern** (always follow this):
1. **Context before**: Explain what the reader will see
2. **Include shortcode**: Reference testdata
3. **Explanation after**: Summarize what they saw

**Good example**:
```markdown
## Implementing the controller

The basic logic of our CronJob controller is this:

1. Load the named CronJob
2. List all active jobs, and update the status
3. Clean up old jobs according to the history limits

{{#literatego ./testdata/project/internal/controller/cronjob_controller.go}}

The reconciler returns successfully when the CronJob is deleted (using `client.IgnoreNotFound`).
For existing CronJobs, it ensures child Jobs match the schedule.
```

**Why this pattern works**:
- Reader knows what to look for (context)
- Sees actual, tested code (include)
- Understands the key points (explanation)

### When to Use Inline Code

Use inline code when:
- Code is not available in testdata
- Documentation is not a tutorial AND snippet is short (non-tutorial conceptual/reference docs with brief examples)

For tutorials: Always use testdata includes. Add code to testdata first or create issue to track.

### Inline Code Comments

```go
// Good: Comments explain WHY, not WHAT
// Reconcile implements the control loop logic.
// It ensures the actual state matches the desired state.
func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Fetch the instance
    obj := &myv1.MyKind{}
    if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Your logic here
    return ctrl.Result{}, nil
}
```

### Shell Examples

Use `$` for commands:
```bash
$ kubebuilder init --domain example.com
$ make install
```

No `$` for output:
```bash
$ kubectl get pods
NAME                     READY   STATUS    RESTARTS   AGE
my-pod-abc123           1/1     Running   0          5m
```

## Admonitions (Aside Blocks)

### Note Blocks

Use for important information:
```markdown
<aside class="note" role="note">
<p class="note-title">Title Here</p>

Content goes here. Can include markdown, code blocks, links, etc.

</aside>
```

### Warning Blocks

Use for critical information or caveats:
```markdown
<aside class="warning" role="note">
<p class="note-title">Title Here</p>

Content goes here. Can include markdown, code blocks, links, etc.

</aside>
```

### General Guidelines

- Always include `role="note"` attribute
- Use descriptive titles with `<p class="note-title">`
- Leave a blank line after the title
- Can contain multiple paragraphs, code blocks, lists
- Always close with blank line before `</aside>`
- Do NOT use "TIP!" or similar callouts in regular text - use aside blocks only

## Accessibility Guidelines

Requirements:
- Semantic HTML headings (h1, h2, h3)
- Alt text for images
- Descriptive link text
- Proper heading hierarchy
- Sufficient color contrast
- No trailing spaces
- Proper aside block format (not deprecated shortcodes)

Run `make fix-docs` to automatically fix accessibility issues and remove trailing spaces.

## Common External References

Centralize frequently used URLs:

- Kubernetes documentation: https://kubernetes.io/docs/
- Kubernetes API conventions: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md
- Kubernetes glossary: https://kubernetes.io/docs/reference/glossary/
- mdBook documentation: https://rust-lang.github.io/mdBook/
- Go documentation: https://go.dev/doc/

**Kubebuilder ecosystem**:
- controller-runtime docs: https://pkg.go.dev/sigs.k8s.io/controller-runtime
- controller-runtime repo: https://github.com/kubernetes-sigs/controller-runtime
- controller-tools docs: https://pkg.go.dev/sigs.k8s.io/controller-tools
- controller-tools repo: https://github.com/kubernetes-sigs/controller-tools

## Verifying Dependency Information

When documenting features from controller-runtime, controller-tools, or other dependencies, verify accuracy by checking the source code.

### Why This Matters

**Problem**: Documentation about dependencies can become outdated or incorrect:
- API methods change signatures
- Marker behavior changes
- Default values change
- New options are added

**Solution**: Verify by checking the actual source code.

### How to Verify

Use `go mod vendor` to download and inspect dependency source code:

```bash
# Create temp directory for verification
mkdir -p /tmp/verify-deps
cd /tmp/verify-deps

# Initialize a Go module
go mod init example.com/verify

# Add the dependency you want to check
go get sigs.k8s.io/controller-runtime@latest
# or
go get sigs.k8s.io/controller-tools@latest

# Vendor the dependencies
go mod vendor

# Now inspect the source code
cd vendor/sigs.k8s.io/controller-runtime
# or
cd vendor/sigs.k8s.io/controller-tools
```

### What to Verify

**For controller-runtime**:
- Manager options and defaults: `vendor/sigs.k8s.io/controller-runtime/pkg/manager/manager.go`
- Client behavior: `vendor/sigs.k8s.io/controller-runtime/pkg/client/client.go`
- Reconciler interface: `vendor/sigs.k8s.io/controller-runtime/pkg/reconcile/reconcile.go`
- Builder options: `vendor/sigs.k8s.io/controller-runtime/pkg/builder/controller.go`

**For controller-tools**:
- Marker definitions: `vendor/sigs.k8s.io/controller-tools/pkg/markers/`
- CRD generation: `vendor/sigs.k8s.io/controller-tools/pkg/crd/`
- Webhook generation: `vendor/sigs.k8s.io/controller-tools/pkg/webhook/`
- RBAC generation: `vendor/sigs.k8s.io/controller-tools/pkg/rbac/`

### Example Verification Workflow

Documenting a controller-runtime feature:

```bash
# Setup
mkdir -p /tmp/verify-controller-runtime
cd /tmp/verify-controller-runtime
go mod init example.com/verify
go get sigs.k8s.io/controller-runtime@latest
go mod vendor

# Check Manager.Start behavior
cat vendor/sigs.k8s.io/controller-runtime/pkg/manager/manager.go | grep -A 10 "func.*Start"

# Check default reconcile options
grep -r "DefaultRecoverPanic" vendor/sigs.k8s.io/controller-runtime/

# Verify the exact version being documented
go list -m sigs.k8s.io/controller-runtime
```

### When to Verify

Always verify when:
- Documenting specific API behavior
- Describing default values or configurations
- Explaining marker syntax or options
- Updating documentation for new versions
- User reports documentation is incorrect

Do NOT document from memory or assumptions - verify the source code.

## Testing Documentation

Before submitting:

1. **Test all command examples manually** in a clean environment:
   ```bash
   # Create temp directory for testing
   mkdir -p /tmp/kb-doc-test
   cd /tmp/kb-doc-test

   # Run commands exactly as documented
   kubebuilder init --domain example.com
   # ... follow all documented steps

   # Verify output matches documentation
   ```

2. **Verify code examples compile and run**:
   - Go code: Ensure it compiles without errors
   - YAML/JSON: Validate syntax
   - Shell scripts: Test in clean shell

3. **Check all links are valid**:
   - Internal links resolve correctly
   - External links are accessible
   - No broken anchors

**Rule**: If you cannot test a command or step yourself, do not include it in documentation.
