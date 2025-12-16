# project-v4 - AI Agent Guide

## Project Structure

**Single-group layout (default):**
```
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
```

**Multi-group layout** (for projects with multiple API groups):
```
api/<group>/<version>/*_types.go       CRD schemas by group
internal/controller/<group>/*          Controllers by group
internal/webhook/<group>/*             Webhooks by group (if present)
```

Multi-group layout organizes APIs by group name (e.g., `batch`, `apps`). Check the `PROJECT` file for `multigroup: true`.

**To convert to multi-group layout:**
1. Run: `kubebuilder edit --multigroup=true`
2. Move existing APIs: `mkdir api/<group> && mv api/<version> api/<group>/`
3. Move controllers: `mkdir internal/controller/<group> && mv internal/controller/*.go internal/controller/<group>/`
4. Move webhooks (if any): `mkdir internal/webhook/<group> && mv internal/webhook/*.go internal/webhook/<group>/`
5. Update import paths in all files
6. Fix `path` in `PROJECT` file for each resource

## Critical Rules

### Never Edit These (Auto-Generated)
- `config/crd/bases/*.yaml` - from `make manifests`
- `config/rbac/role.yaml` - from `make manifests`
- `config/webhook/manifests.yaml` - from `make manifests`
- `**/zz_generated.*.go` - from `make generate`
- `PROJECT` - from `kubebuilder [OPTIONS]`

### Never Remove Scaffold Markers
Do NOT delete `// +kubebuilder:scaffold:*` comments. CLI injects code at these markers.

### Keep Project Structure
Do not move files around. The CLI expects files in specific locations.

### Always Use CLI Commands
Always use `kubebuilder create api` and `kubebuilder create webhook` to scaffold. Do NOT create files manually.

### E2E Tests Require an Isolated Kind Cluster
The e2e tests are designed to validate the solution in an isolated environment (similar to GitHub Actions CI).
Ensure you run them against a dedicated [Kind](https://kind.sigs.k8s.io/) cluster (not your “real” dev/prod cluster).

## After Making Changes

**After editing `*_types.go` or markers:**
```
make manifests  # Regenerate CRDs/RBAC from markers
make generate   # Regenerate DeepCopy methods
```

**After editing `*.go` files:**
```
make lint-fix   # Auto-fix code style
make test       # Run unit tests
```

## CLI Commands Cheat Sheet

### Create API (your own types)
```bash
kubebuilder create api --group <group> --version <version> --kind <Kind>
```

### Deploy Image Plugin (scaffold to deploy/manage a container image)
Use this plugin to generate a complete controller that deploys and manages a container image on the cluster. It scaffolds best-practice code including reconciliation logic, status conditions, finalizers, and RBAC. Check the scaffolded code as an example of best practices.

```bash
# Basic usage
kubebuilder create api --group example.com --version v1alpha1 --kind Memcached \\
  --image=memcached:1.6.26-alpine3.19 \\
  --plugins=deploy-image.go.kubebuilder.io/v1-alpha
```

### Create Webhooks
```bash
# Validation + defaulting
kubebuilder create webhook --group <group> --version <version> --kind <Kind> \\
  --defaulting --programmatic-validation

# Conversion webhook (for multi-version APIs)
kubebuilder create webhook --group <group> --version v1 --kind <Kind> \\
  --conversion --spoke v2
```

### Controller for Core Kubernetes Types
```bash
# Watch Pods
kubebuilder create api --group core --version v1 --kind Pod \\
  --controller=true --resource=false

# Watch Deployments
kubebuilder create api --group apps --version v1 --kind Deployment \\
  --controller=true --resource=false
```

### Controller for External Types (e.g. cert-manager)
```bash
kubebuilder create api \\
  --group cert-manager --version v1 --kind Certificate \\
  --controller=true --resource=false \\
  --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 \\
  --external-api-domain=io \\
  --external-api-module=github.com/cert-manager/cert-manager@v1.18.2
```

### Webhook for External Types
```bash
kubebuilder create webhook \\
  --group cert-manager --version v1 --kind Issuer \\
  --defaulting \\
  --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 \\
  --external-api-domain=io \\
  --external-api-module=github.com/cert-manager/cert-manager@v1.18.2
```

## API Design (`api/<version>/*_types.go`)

Use `+kubebuilder` markers to control CRD generation:

```go
// On types:
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status        // RECOMMENDED
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=".status.phase"

// On fields:
// +kubebuilder:validation:Required
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:MaxLength=100
// +kubebuilder:validation:Pattern="^[a-z]+$"
// +kubebuilder:default="value"
```

## RBAC Markers (`internal/controller/*_controller.go`)
```go
// +kubebuilder:rbac:groups=mygroup.example.com,resources=mykinds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mygroup.example.com,resources=mykinds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mygroup.example.com,resources=mykinds/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
```

## Testing

### Unit Tests (`*_controller_test.go`)
- Use **envtest** (real Kubernetes API server + etcd for testing)
- BDD style with **Ginkgo + Gomega**
- Check `suite_test.go` for setup

### Run Tests
```bash
make test              # All tests
make test ARGS="-v"    # Verbose
```

## Deploy & Test (Quick Checklist)

When you need to validate behavior end-to-end against a real cluster:

```bash
# Regenerate code and manifests (ensure CRDs/RBAC reflect latest markers)
make generate
make manifests

# Install CRDs
make install

# Choose your image name once and reuse it
export IMG=<some-registry>/<project-name>:tag

# Build image
make docker-build IMG=$IMG

# If using Kind for local development, load the image into the cluster (no registry push required):
# (Pick the Kind cluster name from: kind get clusters)
kind load docker-image $IMG --name <kind-cluster-name>

# Otherwise (non-Kind/remote cluster), push to a registry that your cluster can pull from:
make docker-push IMG=$IMG

# Deploy controller with that image
make deploy IMG=$IMG

# Apply sample CRs (edit config/samples/* first as needed)
kubectl apply -k config/samples/

# Inspect what will be applied (single rendered manifest: CRDs/RBAC/manager, etc.)
make build-installer IMG=$IMG
less dist/install.yaml

# Verify the controller is running (namespace may vary by manifests)
kubectl get pods -n system

# Debug: controller logs (namespace/name may vary by manifests)
kubectl logs -n system deployment/<project-name>-controller-manager -c manager --tail=200
```

## Makefile Commands

See `Makefile` for all targets. Common ones:

| Command | Purpose |
| --- | --- |
| `make manifests` | Generate CRDs/RBAC from markers |
| `make generate` | Generate DeepCopy code |
| `make lint` | Check code style |
| `make lint-fix` | Auto-fix code style |
| `make test` | Run unit tests |
| `make build` | Build manager binary |
| `make docker-build` | Build container image |
| `make deploy` | Deploy to cluster |
| `make install` | Install CRDs only |
| `make build-installer` | Build `dist/install.yaml` bundle |
| `make help` | Show all targets |

## Recommendations

### API Design
- Enable status subresource: `+kubebuilder:subresource:status`
- Use `metav1.Condition` for status tracking
- Add validation markers for all fields
- Set sensible defaults
- Follow Kubernetes API conventions

### Controller Design
- Set owner references for garbage collection. Use `SetControllerReference` from `controller-runtime`
- Re-fetch resources before updates to avoid conflicts
- Use structured logging: `log.FromContext(ctx)`

### Webhooks
- Add all webhook types in one command when possible: `--defaulting --programmatic-validation --conversion`
- Avoid re-scaffolding webhooks - adding types later requires `--force` which overwrites the file
- If `--force` is needed: backup webhook code first, scaffold, then restore custom logic

## References

- **Kubebuilder Book**: https://book.kubebuilder.io
- **Kubebuilder Repo**: https://github.com/kubernetes-sigs/kubebuilder
- **controller-runtime**: https://github.com/kubernetes-sigs/controller-runtime
- **controller-tools**: https://github.com/kubernetes-sigs/controller-tools
- **API Conventions**: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md
- **Operator Pattern**: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
- **Markers Reference**: https://book.kubebuilder.io/reference/markers.html
