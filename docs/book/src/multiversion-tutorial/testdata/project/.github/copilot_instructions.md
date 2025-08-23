# Repository Instructions

These are repository-wide instructions for a **Kubebuilder-style Kubernetes Operator** written in Go.

---

## Project Overview
- **Type:** Kubernetes Operator (Kubebuilder scaffold)
- **Language:** Go (version pinned in `go.mod`)
- **Key libs:** `sigs.k8s.io/controller-runtime`, `k8s.io/*`
- **Tools:** `sigs.k8s.io/controller-tools` (`controller-gen`), kustomize, envtest, golangci-lint, kind (for e2e)

---

## Project Layout
- `/api` — CRD types & deepcopy files
- `internal/controllers` — Reconciliation logic
- `internal/webhooks` — Admission webhooks: defaulting (mutating) & validating
- `/cmd` — Main entrypoint
- `/config` — Kustomize manifests and operator configuration:
  - `certmanager/` — Certificates & issuers
  - `crd/` — CustomResourceDefinitions
  - `default/` — Default overlays
  - `manager/` — Manager deployment manifests
  - `network-policy/` — Network policies
  - `prometheus/` — Monitoring resources
  - `rbac/` — Roles and bindings
  - `samples/` — Example custom resources
  - `webhook/` — Webhook configurations
- `/test` — E2E tests and helpers
- `/dist` — Distribution artifacts:
- `Makefile` — Build automation
- `PROJECT` — Project definition file (**auto-generated**; records inputs for scaffolding — avoid manual edits.)

**Generated files include:**
- `api/**/zz_generated.*.go`
- `config/crd/bases/*.yaml`
- `config/rbac/*.yaml`
All other files are considered **source**.

---

## Code Review Rules

:robot: AI suggestions (may be incorrect) Always run `make vet` and `make fix-lint`.

### For [GENERATED]
- DO NOT recommend manual edits.
- For `api/**/zz_generated.*.go`, never edit by hand; regenerate with:
  - `make generate`

- For changes in `api/*_types.go`, `internal/controllers`, or `internal/webhooks`, rerun:
  - `make generate`
  - `make manifests`

- For `dist/install.yaml` artifacts, never edit by hand; regenerate with:
  - `make build-installer`
