---
name: plugin-helm
description: Guidelines for maintaining the Kubebuilder Helm plugin (helm/v2-alpha) in pkg/plugins/optional/helm/v2alpha/. Use when working on Helm plugin code, templates, or chart generation.
license: Apache-2.0
metadata:
  author: The Kubernetes Authors
---

## Plugin Purpose

Transform kustomize YAML → Helm chart for operator distribution.

**Default:** Reads `dist/install.yaml`, writes to `dist/chart/`
**Flags:** `--manifests=<input>` and/or `--output-dir=<output>`

## Development Workflow

After code changes in `pkg/plugins/optional/helm/v2alpha/`:

1. `make install` - Build and install updated binary
2. `make generate-charts` - Regenerate all sample charts in testdata
3. `make verify-helm` - Run yamllint + helm lint + kube-linter
4. Test installation in real cluster if changing generation logic

**Never commit without running `make verify-helm`.**

## Values.yaml Rules

### Comment Syntax (Strict)

- `##` = Documentation/description
- `#` = Commented-out spec value
- Always blank `##` line between description and spec

```yaml
## Description of the field
##
# fieldName: value
```

### Uncomment Policy

**Uncomment a value only if:**
1. **Extracted from kustomize** - The value exists in the source manifests
2. **Standard Helm convention** - Common fields users expect uncommented (replicas, image, resource names)

**Keep commented if:**
1. **Optional Kubernetes feature** - Not currently used in the operator (imagePullSecrets, priorityClassName)
2. **Advanced configuration** - Not needed for basic usage (topology spread, pod disruption budget)
3. **User customization fields** - Name overrides, custom labels that users should explicitly set

### Template Conditional Pattern (Critical)

**All values.yaml fields MUST be conditional in templates.** If a user removes a field from values.yaml, the chart must still work.

**Required patterns:**

Use `{{- with }}` for optional blocks:
```yaml
{{- with .Values.manager.affinity }}
affinity: {{ toYaml . | nindent 10 }}
{{- end }}
```

Use `{{- if }}` with fallback for required fields:
```yaml
resources:
  {{- if .Values.manager.resources }}
  {{- toYaml .Values.manager.resources | nindent 10 }}
  {{- else }}
  {}
  {{- end }}
```

**Never:** Direct value reference without conditionals
```yaml
# WRONG - breaks if removed from values.yaml
affinity: {{ toYaml .Values.manager.affinity | nindent 10 }}
```

**When to use `| default` vs conditionals:**

- **Do NOT use `| default` for fields extracted from kustomize** - they are always present in values.yaml
- **NEVER use `| default` for fields that can be legitimately set to 0** (e.g., replicas for scale-to-zero). Helm treats 0 as falsy and will use the default instead, breaking the intended behavior.
- For optional K8s fields that have built-in defaults (e.g., `imagePullPolicy`), use `{{- with }}` conditionals to let K8s apply its own defaults when not specified:
  ```yaml
  {{- with .Values.manager.image.pullPolicy }}
  imagePullPolicy: {{ . }}
  {{- end }}
  ```
- Core fields like `replicas`, `image.repository` are always extracted from kustomize - use direct references:
  ```yaml
  replicas: {{ .Values.manager.replicas }}
  image: "{{ .Values.manager.image.repository }}..."
  ```

When adding new values.yaml fields, always add conditionals in templates.

### Comment Content Rules (values.yaml)

1. **One to two lines max** - Be concise
2. **Focus on user action** - What they can configure, not how it works internally
3. **Include examples** - Show correct YAML structure for commented values
4. **Use plain English** - Follow [Kubernetes Documentation Style Guide](https://kubernetes.io/docs/contribute/style/style-guide/)
5. **End with blank `##` line**

**Good:**
```yaml
## Container image pull secrets for private registries
##
# imagePullSecrets:
#   - name: regcred
```

**Bad:**
```yaml
# This field is used by the template engine to render imagePullSecrets
# into the deployment spec when the value is provided by the user
# imagePullSecrets: []
```

## Code Style

### Go Code Comments

- **Be concise** - Explain WHY, not WHAT (code should be self-documenting)
- **No unnecessary comments** - Well-named functions/variables > comments
- **Plain text only** - No arrows, emojis, or decorative symbols
- **Follow Go conventions** - See [Effective Go](https://go.dev/doc/effective_go#commentary)

**Good:**
```go
// TemplatePorts replaces port numbers with Helm template references.
// Uses suffix matching to avoid false positives when project name contains "webhook".
func TemplatePorts(yamlContent string, resource *unstructured.Unstructured) string {
```

**Bad:**
```go
// This function processes the yaml content and templates the ports
// by checking if it's a webhook or metrics service and then using
// regular expressions to replace the port numbers with values
func TemplatePorts(yamlContent string, resource *unstructured.Unstructured) string {
```

## Code Generation Logic

The plugin uses two different Machinery approaches:

### Dynamic Templates (from kustomize)

**Location:** `pkg/plugins/optional/helm/v2alpha/scaffolds/internal/kustomize/`

All Kubernetes resources from `dist/install.yaml` are converted using the **templater**:
- Parses kustomize YAML → Applies Helm templating → Outputs to `templates/`
- Uses `DynamicTemplate` wrapper (pre-rendered content, not Go templates)
- **Always overwrites** (`IfExistsAction = OverwriteFile`)
- Examples: `manager/manager.yaml`, `rbac/*.yaml`, `webhook/*.yaml`

**Key file:** `dynamic_template.go` - Wraps pre-rendered templates for Machinery

### Boilerplate Templates (fixed files)

**Location:** `pkg/plugins/optional/helm/v2alpha/scaffolds/internal/templates/`

Static chart files use standard Machinery Go templates:
- `Chart.yaml` - **Never overwrites** (`IfExistsAction = SkipFile`)
- `values.yaml`, `NOTES.txt`, `_helpers.tpl`, `.helmignore` - Respect `--force` flag
- Generated from Go templates, not kustomize output

**Important:** Never delete the `templates/` directory. Let Machinery overwrite files in place.

### Values.yaml Generation

When modifying what appears in values.yaml:
1. Check `Extraction` struct for available extracted data
2. Use conditionals to only render if value exists
3. Maintain comment/uncomment logic based on rules above
4. Test with multiple project types (with/without webhooks, metrics, etc.)
5. Follow Kubernetes Documentation Style Guide for all user-facing comments

## Critical Design Decisions

### CRD Placement: `templates/crd/` not `crds/`

**Decision:** CRDs are placed in `templates/crd/` instead of the standard `crds/` directory.

**Rationale:** Helm's `crds/` directory never upgrades CRDs on `helm upgrade`. For Kubernetes operators, CRD updates are critical (new API fields, validation changes). Placing CRDs in `templates/crd/` ensures they upgrade with the chart.

**Trade-off:** Cannot mix CRDs and Custom Resources in the same chart (Helm installation order limitation).

**Action:** Preserve this placement. Do not move CRDs to `crds/` directory.

### File Preservation

The plugin uses Kubebuilder's Machinery framework for file management:

**Never overwrite:** `Chart.yaml` (user-managed version info)

**Overwrite only with `--force`:** `values.yaml`, `NOTES.txt`, `_helpers.tpl`, `.helmignore`, `.github/workflows/test-chart.yml`

**Always overwrite:** All `templates/` resources (manager, rbac, webhook, etc.)

**Important:** Use Machinery's `IfExistsAction` to control overwrite behavior. Never delete the `templates/` directory or individual files - let Machinery overwrite them in place. This preserves file timestamps and avoids breaking user workflows (e.g., open editors, file watchers).

### Documentation Updates

**Plugin documentation must stay synchronized with code changes.**

When modifying plugin behavior:
1. Update `docs/book/src/plugins/available/helm-v2-alpha.md` to reflect changes
2. Follow Kubebuilder tutorial-based documentation style:
   - Be concise
   - Example-driven with code snippets showing actual usage
   - Clear section structure: What - Why/When - Usage - Examples (if required)
3. Follow [Kubernetes Documentation Style Guide](https://kubernetes.io/docs/contribute/style/style-guide/)
4. Add specific error scenarios only for known, reproducible errors with fixes

## Common Tasks

### Adding a New values.yaml Field

1. Check if value exists in `Extraction` struct
2. Add extraction logic in the extractor
3. Add rendering logic in values generation
4. Follow comment rules above
5. **Add conditional usage in template** (use `{{- with }}` or `{{- if }}` - see Template Conditional Pattern)
6. Run `make install && make generate-charts && make verify-helm`

### Changing Chart Structure

1. Modify chart scaffolder orchestration
2. Update template generators if needed
3. Regenerate all samples with `make generate-charts`
4. Update plugin documentation if user-visible

### Fixing Template Generation

1. Locate template in the templates directory
2. Modify template logic
3. Test with `make install` and run plugin on test project
4. Run `make generate-charts` to update all samples
5. Check diffs carefully - ensure no unintended changes
6. Run `make verify-helm`

## Testing

### Test Structure (Ginkgo/Gomega BDD)

All tests use Ginkgo v2 + Gomega for behavior-driven development:

```go
var _ = Describe("Feature Name", func() {
    Context("Scenario description", func() {
        It("should do specific behavior", func() {
            // Arrange, Act, Assert
            Expect(result).To(Equal(expected))
        })
    })
})
```

- **Unit tests** (`*_test.go`): Fast, isolated component tests
- **Integration tests** (`//go:build integration`): End-to-end chart generation

### Running Tests

```bash
# Unit tests only (fast)
make test-unit

# Integration tests (requires kubebuilder binary in PATH)
make test-integration

# Helm chart validation (yamllint, helm lint, kube-linter)
make verify-helm
```

### Test Coverage Requirements

When adding features, ensure tests cover:

1. **Extraction scenarios**: Does it extract the new field from kustomize YAML?
2. **Generation scenarios**: Does it render correctly in values.yaml?
3. **Template scenarios**: Does the template handle missing/present values?
4. **Edge cases**:
   - Field present vs absent in kustomize
   - Field removed from values.yaml by user
   - Different data types (string, int, bool, map, array)
   - Empty values vs nil vs not present

**BDD pattern example:**
```go
Describe("Webhook port extraction", func() {
    Context("when webhook is present in kustomize", func() {
        It("should extract port and add to values.yaml", func() { ... })
        It("should use port conditionally in template", func() { ... })
    })

    Context("when webhook is not present", func() {
        It("should not add webhook section to values.yaml", func() { ... })
        It("should not render webhook port in template", func() { ... })
    })

    Context("when user removes webhook from values.yaml", func() {
        It("should not break template rendering", func() { ... })
    })
})
```

### Testing Checklist

Before submitting PR:

- [ ] `make test-unit` passes
- [ ] `make test-integration` passes
- [ ] `make install` succeeds
- [ ] `make generate-charts` succeeds
- [ ] `make verify-helm` passes
- [ ] **Test chart installation** (at minimum):
  - `cd testdata/project-v4-with-plugins && make helm-deploy IMG=test:latest`
- [ ] **Test scenarios not covered by default testdata:**
  - Create mock test projects for edge cases your change affects
  - Examples: projects without webhooks, without metrics, without cert-manager, cluster-scoped RBAC, multi-namespace RBAC
  - Run plugin on mock projects and verify chart generation
  - Install generated charts on test cluster
- [ ] Check generated values.yaml follows comment rules (syntax, uncomment/comment logic)
- [ ] Verify templates use conditionals (`{{- with }}` or `{{- if }}`) for all values
- [ ] Test with custom `--manifests` and `--output-dir` flags
- [ ] New features have unit + integration tests with BDD scenarios
- [ ] Tests cover extraction, generation, and template rendering
- [ ] Tests cover edge cases (field present/absent/removed)

## References

See `references/REFERENCE.md` for:
- Documentation links
- Helm best practices
- Kubernetes operator resources
- Sample chart locations
