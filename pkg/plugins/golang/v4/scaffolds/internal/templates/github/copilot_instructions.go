package github

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &CopilotInstructions{}

// CopilotInstructions scaffolds a repository-wide Markdown file with AI/assistant guidance
type CopilotInstructions struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *CopilotInstructions) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join(".github", "copilot_instructions.md")
	}

	f.TemplateBody = copilotInstructionsTemplate
	f.IfExistsAction = machinery.SkipFile
	return nil
}

//nolint:lll
const copilotInstructionsTemplate = "# Repository Instructions\n\n" +
	"These are repository-wide instructions for a **Kubebuilder-style Kubernetes Operator** written in Go.\n\n" +
	"---\n\n" +
	"## Project Overview\n" +
	"- **Type:** Kubernetes Operator (Kubebuilder scaffold)\n" +
	"- **Language:** Go (version pinned in `go.mod`)\n" +
	"- **Key libs:** `sigs.k8s.io/controller-runtime`, `k8s.io/*`\n" +
	"- **Tools:** `sigs.k8s.io/controller-tools` (`controller-gen`), kustomize, envtest, golangci-lint, kind (for e2e)\n\n" +
	"---\n\n" +
	"## Project Layout\n" +
	"- `/api` — CRD types & deepcopy files\n" +
	"- `internal/controllers` — Reconciliation logic\n" +
	"- `internal/webhooks` — Admission webhooks: defaulting (mutating) & validating\n" +
	"- `/cmd` — Main entrypoint\n" +
	"- `/config` — Kustomize manifests and operator configuration:\n" +
	"  - `certmanager/` — Certificates & issuers\n" +
	"  - `crd/` — CustomResourceDefinitions\n" +
	"  - `default/` — Default overlays\n" +
	"  - `manager/` — Manager deployment manifests\n" +
	"  - `network-policy/` — Network policies\n" +
	"  - `prometheus/` — Monitoring resources\n" +
	"  - `rbac/` — Roles and bindings\n" +
	"  - `samples/` — Example custom resources\n" +
	"  - `webhook/` — Webhook configurations\n" +
	"- `/test` — E2E tests and helpers\n" +
	"- `/dist` — Distribution artifacts:\n" +
	"- `Makefile` — Build automation\n" +
	"- `PROJECT` — Project definition file (**auto-generated**; records inputs for scaffolding — avoid manual edits.)\n\n" +
	"**Generated files include:**\n" +
	"- `api/**/zz_generated.*.go`\n" +
	"- `config/crd/bases/*.yaml`\n" +
	"- `config/rbac/*.yaml`\n" +
	"All other files are considered **source**.\n\n" +
	"---\n\n" +
	"## Code Review Rules\n\n" +
	":robot: AI suggestions (may be incorrect) Always run `make vet` and `make fix-lint`.\n\n" +
	"### For [GENERATED]\n" +
	"- DO NOT recommend manual edits.\n" +
	"- For `api/**/zz_generated.*.go`, never edit by hand; regenerate with:\n" +
	"  - `make generate`\n\n" +
	"- For changes in `api/*_types.go`, `internal/controllers`, or `internal/webhooks`, rerun:\n" +
	"  - `make generate`\n" +
	"  - `make manifests`\n\n" +
	"- For `dist/install.yaml` artifacts, never edit by hand; regenerate with:\n" +
	"  - `make build-installer`\n"
