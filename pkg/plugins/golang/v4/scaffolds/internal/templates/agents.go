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

package templates

import (
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Agents{}

// Agents scaffolds an AGENTS.md file
type Agents struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// CommandName stores the name of the bin used
	CommandName string
	// IsKubebuilderCLI indicates if kubebuilder CLI is being used (vs operator-sdk, etc)
	IsKubebuilderCLI bool
}

// SetTemplateDefaults implements machinery.Template
func (f *Agents) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "AGENTS.md"
	}

	// Check if using Kubebuilder CLI
	if f.CommandName != "" {
		f.IsKubebuilderCLI = strings.Contains(f.CommandName, "kubebuilder")
	}

	f.TemplateBody = agentsFileTemplate

	return nil
}

//nolint:lll
const agentsFileTemplate = `# {{ .ProjectName }} - AI Agent Guide

## Project Structure

**Single-group layout (default):**
` + "```" + `
cmd/main.go                    Manager entry (registers controllers/webhooks)
api/<version>/*_types.go       CRD schemas (+kubebuilder markers)
api/<version>/zz_generated.*   Auto-generated (DO NOT EDIT)
internal/controller/*          Reconciliation logic
internal/webhook/*             Validation/defaulting (if present)
config/crd/bases/*             Generated CRDs (DO NOT EDIT)
config/rbac/role.yaml          Generated RBAC (DO NOT EDIT)
config/samples/*               Example CRs (edit these)
Makefile                       Build/test/deploy commands
PROJECT                        Kubebuilder metadata Auto-generated (DO NOT EDIT)
` + "```" + `

**Multi-group layout** (for projects with multiple API groups):
` + "```" + `
api/<group>/<version>/*_types.go       CRD schemas by group
internal/controller/<group>/*          Controllers by group
internal/webhook/<group>/<version>/*   Webhooks by group and version (if present)
` + "```" + `

Multi-group layout organizes APIs by group name (e.g., ` + "`batch`" + `, ` + "`apps`" + `). Check the ` + "`PROJECT`" + ` file for ` + "`multigroup: true`" + `.

**To convert to multi-group layout:**
1. Run: ` + "`{{ .CommandName }} edit --multigroup=true`" + `
2. Move APIs: ` + "`mkdir -p api/<group> && mv api/<version> api/<group>/`" + `
3. Move controllers: ` + "`mkdir -p internal/controller/<group> && mv internal/controller/*.go internal/controller/<group>/`" + `
4. Move webhooks (if present): ` + "`mkdir -p internal/webhook/<group> && mv internal/webhook/<version> internal/webhook/<group>/`" + `
5. Update import paths in all files
6. Fix ` + "`path`" + ` in ` + "`PROJECT`" + ` file for each resource
7. Update test suite CRD paths (add one more ` + "`..`" + ` to relative paths)

## Critical Rules

### Never Edit These (Auto-Generated)
- ` + "`config/crd/bases/*.yaml`" + ` - from ` + "`make manifests`" + `
- ` + "`config/rbac/role.yaml`" + ` - from ` + "`make manifests`" + `
- ` + "`config/webhook/manifests.yaml`" + ` - from ` + "`make manifests`" + `
- ` + "`**/zz_generated.*.go`" + ` - from ` + "`make generate`" + `
- ` + "`PROJECT`" + ` - from ` + "`{{ .CommandName }} [OPTIONS]`" + `

### Never Remove Scaffold Markers
Do NOT delete ` + "`// +kubebuilder:scaffold:*`" + ` comments. CLI injects code at these markers.

### Keep Project Structure
Do not move files around. The CLI expects files in specific locations.

### Always Use CLI Commands
Always use ` + "`{{ .CommandName }} create api`" + ` and ` + "`{{ .CommandName }} create webhook`" + ` to scaffold. Do NOT create files manually.

### E2E Tests Require an Isolated Kind Cluster
The e2e tests are designed to validate the solution in an isolated environment (similar to GitHub Actions CI).
Ensure you run them against a dedicated [Kind](https://kind.sigs.k8s.io/) cluster (not your “real” dev/prod cluster).

## After Making Changes

**After editing ` + "`*_types.go`" + ` or markers:**
` + "```" + `
make manifests  # Regenerate CRDs/RBAC from markers
make generate   # Regenerate DeepCopy methods
` + "```" + `

**After editing ` + "`*.go`" + ` files:**
` + "```" + `
make lint-fix   # Auto-fix code style
make test       # Run unit tests
` + "```" + `

## CLI Commands Cheat Sheet

### Create API (your own types)
` + "```bash" + `
{{ .CommandName }} create api --group <group> --version <version> --kind <Kind>
` + "```" + `{{ if .IsKubebuilderCLI }}

### Deploy Image Plugin (scaffold to deploy/manage ANY container image)

Generate a controller that deploys and manages a container image (nginx, redis, memcached, your app, etc.):

` + "```bash" + `
# Example: deploying memcached
{{ .CommandName }} create api --group example.com --version v1alpha1 --kind Memcached \
  --image=memcached:alpine \
  --plugins=deploy-image.go.kubebuilder.io/v1-alpha
` + "```" + `

Scaffolds good-practice code: reconciliation logic, status conditions, finalizers, RBAC. Use as a reference implementation.
{{ end }}

### Create Webhooks
` + "```bash" + `
# Validation + defaulting
{{ .CommandName }} create webhook --group <group> --version <version> --kind <Kind> \
  --defaulting --programmatic-validation

# Conversion webhook (for multi-version APIs)
{{ .CommandName }} create webhook --group <group> --version v1 --kind <Kind> \
  --conversion --spoke v2
` + "```" + `

### Controller for Core Kubernetes Types
` + "```bash" + `
# Watch Pods
{{ .CommandName }} create api --group core --version v1 --kind Pod \
  --controller=true --resource=false

# Watch Deployments
{{ .CommandName }} create api --group apps --version v1 --kind Deployment \
  --controller=true --resource=false
` + "```" + `

### Controller for External Types (e.g., from other operators)

Watch resources from external APIs (cert-manager, Argo CD, Istio, etc.):

` + "```bash" + `
# Example: watching cert-manager Certificate resources
{{ .CommandName }} create api \
  --group cert-manager --version v1 --kind Certificate \
  --controller=true --resource=false \
  --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 \
  --external-api-domain=io \
  --external-api-module=github.com/cert-manager/cert-manager
` + "```" + `

**Note:** Use ` + "`--external-api-module=<module>@<version>`" + ` only if you need a specific version. Otherwise, omit ` + "`@<version>`" + ` to use what's in go.mod.

### Webhook for External Types

` + "```bash" + `
# Example: validating external resources
{{ .CommandName }} create webhook \
  --group cert-manager --version v1 --kind Issuer \
  --defaulting \
  --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 \
  --external-api-domain=io \
  --external-api-module=github.com/cert-manager/cert-manager
` + "```" + `

## Testing & Development

` + "```bash" + `
make test              # Run unit tests (uses envtest: real K8s API + etcd)
make run               # Run locally (uses current kubeconfig context)
` + "```" + `

Tests use **Ginkgo + Gomega** (BDD style). Check ` + "`suite_test.go`" + ` for setup.

## Deployment Workflow

` + "```bash" + `
# 1. Regenerate manifests
make manifests generate

# 2. Build & deploy
export IMG=<registry>/<project>:tag
make docker-build docker-push IMG=$IMG  # Or: kind load docker-image $IMG --name <cluster>
make deploy IMG=$IMG

# 3. Test
kubectl apply -k config/samples/

# 4. Debug
kubectl logs -n <project>-system deployment/<project>-controller-manager -c manager -f
` + "```" + `

### API Design

**Key markers for** ` + "`api/<version>/*_types.go`" + `:

` + "```go" + `
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=".status.conditions[?(@.type=='Ready')].status"

// On fields:
// +kubebuilder:validation:Required
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:MaxLength=100
// +kubebuilder:validation:Pattern="^[a-z]+$"
// +kubebuilder:default="value"
` + "```" + `

- **Use** ` + "`metav1.Condition`" + ` for status (not custom string fields)
- **Use predefined types**: ` + "`metav1.Time`" + ` instead of ` + "`string`" + ` for dates
- **Follow K8s API conventions**: Standard field names (` + "`spec`" + `, ` + "`status`" + `, ` + "`metadata`" + `)

### Controller Design

**RBAC markers in** ` + "`internal/controller/*_controller.go`" + `:

` + "```go" + `
// +kubebuilder:rbac:groups=mygroup.example.com,resources=mykinds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mygroup.example.com,resources=mykinds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mygroup.example.com,resources=mykinds/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
` + "```" + `

**Implementation rules:**
- **Idempotent reconciliation**: Safe to run multiple times
- **Re-fetch before updates**: ` + "`r.Get(ctx, req.NamespacedName, obj)`" + ` before ` + "`r.Update`" + ` to avoid conflicts
- **Structured logging**: ` + "`log := log.FromContext(ctx); log.Info(\"msg\", \"key\", val)`" + `
- **Owner references**: Enable automatic garbage collection (` + "`SetControllerReference`" + `)
- **Watch secondary resources**: Use ` + "`.Owns()`" + ` or ` + "`.Watches()`" + `, not just ` + "`RequeueAfter`" + `
- **Finalizers**: Clean up external resources (buckets, VMs, DNS entries)

### Webhooks
- **Create all types together**: ` + "`--defaulting --programmatic-validation --conversion`" + `
- **Avoid re-scaffolding**: Adding types later requires ` + "`--force`" + ` (overwrites file)
- **If using** ` + "`--force`" + `: Backup custom logic first, then restore after scaffolding
- **For multi-version APIs**: Use hub-and-spoke pattern (` + "`--conversion --spoke v2`" + `)
  - Hub version: Usually oldest stable version (v1)
  - Spoke versions: Newer versions that convert to/from hub (v2, v3)
  - Example: ` + "`--group crew --version v1 --kind Captain --conversion --spoke v2`" + ` (v1 is hub, v2 is spoke){{ if .IsKubebuilderCLI }}

### Learning from Examples

The **deploy-image plugin** scaffolds a complete controller following good practices. Use it as a reference implementation:

` + "```bash" + `
{{ .CommandName }} create api --group example --version v1alpha1 --kind MyApp \
  --image=<your-image> --plugins=deploy-image.go.kubebuilder.io/v1-alpha
` + "```" + `

Generated code includes: status conditions (` + "`metav1.Condition`" + `), finalizers, owner references, events, idempotent reconciliation.

## Distribution Options

### Option 1: YAML Bundle (Kustomize)

` + "```bash" + `
# Generate dist/install.yaml from Kustomize manifests
make build-installer IMG=<registry>/<project>:tag
` + "```" + `

**Key points:**
- The ` + "`dist/install.yaml`" + ` is generated from Kustomize manifests (CRDs, RBAC, Deployment)
- Commit this file to your repository for easy distribution
- Users only need ` + "`kubectl`" + ` to install (no additional tools required)

**Example:** Users install with a single command:
` + "```bash" + `
kubectl apply -f https://raw.githubusercontent.com/<org>/<repo>/<tag>/dist/install.yaml
` + "```" + `

### Option 2: Helm Chart

` + "```bash" + `
{{ .CommandName }} edit --plugins=helm/v2-alpha  # One-time, generates dist/chart/
# Users install: helm install my-release ./dist/chart/ --namespace <ns> --create-namespace
` + "```" + `

**Important:** If you add webhooks or modify manifests after initial chart generation:
1. Backup any customizations in ` + "`dist/chart/values.yaml`" + ` and ` + "`dist/chart/manager/manager.yaml`" + `
2. Re-run: ` + "`{{ .CommandName }} edit --plugins=helm/v2-alpha --force`" + `
3. Manually restore your custom values from the backup

### Publish Container Image

` + "```bash" + `
export IMG=<registry>/<project>:<version>
make docker-build docker-push IMG=$IMG
` + "```" + `{{ end }}

## References

### Essential Reading
- **Kubebuilder Book**: https://book.kubebuilder.io (comprehensive guide)
- **controller-runtime FAQ**: https://github.com/kubernetes-sigs/controller-runtime/blob/main/FAQ.md (common patterns and questions)
- **Good Practices**: https://book.kubebuilder.io/reference/good-practices.html (why reconciliation is idempotent, status conditions, etc.)

### API Design & Implementation
- **API Conventions**: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md
- **Operator Pattern**: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
- **Markers Reference**: https://book.kubebuilder.io/reference/markers.html

### Tools & Libraries
- **controller-runtime**: https://github.com/kubernetes-sigs/controller-runtime
- **controller-tools**: https://github.com/kubernetes-sigs/controller-tools
- **Kubebuilder Repo**: https://github.com/kubernetes-sigs/kubebuilder
`
