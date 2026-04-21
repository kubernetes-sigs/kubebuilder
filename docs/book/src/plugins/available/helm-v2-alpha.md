# Helm Plugin `(helm/v2-alpha)`

The `helm/v2-alpha` plugin generates Helm charts from your project’s kustomize output, letting you distribute your operator as either a bundle or a Helm chart.

The plugin dynamically builds charts from `make build-installer` output and preserves your customizations like environment variables, labels, annotations, and security contexts.

## Why use Helm

By default, Kubebuilder creates a bundle of manifests:

```bash
make build-installer IMG=<registry>/<project-name:tag>
```

Users install it with:

```bash
kubectl apply -f https://raw.githubusercontent.com/<org>/project-v4/<tag-or-branch>/dist/install.yaml
```

Many users prefer Helm for packaging and upgrades. This plugin converts `dist/install.yaml` into a Helm chart that mirrors your project.

## Features

- Generates charts from kustomize output, not boilerplate
- Preserves environment variables, labels, annotations, and patches
- Organizes templates to match your `config/` directory layout
- Includes only configurable parameters in `values.yaml`
- Never overwrites `Chart.yaml`; preserves `values.yaml`, `NOTES.txt`, `_helpers.tpl`, `.helmignore`, and `test-chart.yml` unless you use `--force`
- Places custom resources in `templates/extras/` with Helm templating

## Usage

### Basic workflow

Create a project and build the installer bundle:

```bash
kubebuilder init
make build-installer IMG=<registry>/<project:tag>
```

Generate the Helm chart from kustomize output:

```bash
kubebuilder edit --plugins=helm/v2-alpha
```

To regenerate preserved files (except `Chart.yaml`), use `--force`:

```bash
kubebuilder edit --plugins=helm/v2-alpha --force
```

### Advanced options

Use a custom manifests file:

```bash
kubebuilder edit --plugins=helm/v2-alpha --manifests=manifests/custom-install.yaml
```

Write chart to a custom output directory:

```bash
kubebuilder edit --plugins=helm/v2-alpha --output-dir=charts
```

Combine custom manifests and output directory:

```bash
kubebuilder edit --plugins=helm/v2-alpha \
  --manifests=manifests/install.yaml \
  --output-dir=helm-charts
```

## Chart structure

The plugin generates a chart layout that mirrors your `config/` directory:

```text
<output-dir>/chart/
├── Chart.yaml
├── values.yaml
├── .helmignore
└── templates/
    ├── NOTES.txt
    ├── _helpers.tpl
    ├── rbac/                    # Individual RBAC files (examples)
    │   ├── controller-manager.yaml
    │   ├── leader-election-role.yaml
    │   ├── leader-election-rolebinding.yaml
    │   ├── manager-role.yaml
    │   ├── manager-rolebinding.yaml
    │   ├── metrics-auth-role.yaml
    │   ├── metrics-auth-rolebinding.yaml
    │   ├── metrics-reader.yaml
    │   └── ...
    ├── crd/                     # Individual CRD files (examples)
    │   ├── busyboxes.example.com.testproject.org.yaml
    │   └── ...
    ├── cert-manager/
    │   ├── metrics-certs.yaml
    │   ├── selfsigned-issuer.yaml
    │   └── serving-cert.yaml
    ├── manager/
    │   └── manager.yaml
    ├── metrics/
    │   └── controller-manager-metrics-service.yaml
    ├── webhook/
    │   ├── validating-webhook-configuration.yaml
    │   └── webhook-service.yaml
    ├── monitoring/
    │   └── servicemonitor.yaml
    └── extras/                  # Custom resources (if any)
        ├── my-service.yaml
        └── my-config.yaml
```

<aside class="note" role="note">
<p class="note-title">What is templates/extras/</p>

Standard resources (RBAC, manager, webhooks, CRDs) use dedicated template directories. Other resources go in `templates/extras/`.

Custom Resource instances from `config/samples/` are not included. The plugin ignores CR instances even if you add them to kustomize output.

</aside>

<aside class="note" role="note">
<p class="note-title">Why CRDs are in templates/</p>

Although [Helm best practices](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#method-1-let-helm-do-it-for-you) recommend placing CRDs under a top-level `crds/` directory, the Kubebuilder Helm plugin intentionally places them under `templates/crd`.

The rationale is tied to how Helm itself handles CRDs. By default, Helm will install CRDs once during the initial release, but it will **ignore CRD changes** on subsequent upgrades.

This can lead to surprising behavior where chart upgrades silently skip CRD updates, leaving clusters out of sync.

To avoid endorsing this behavior, the Kubebuilder plugin follows the approach of packaging CRDs inside `templates/`. In this mode, Helm treats CRDs like any other resource, ensuring they are applied and upgraded as expected. While this prevents mixing CRDs and CRs of the same type in a single chart (since Helm cannot wait between creation steps), it ensures predictable and explicit lifecycle management of CRDs.

In short:
- **Helm `crds/` directory**: one-time install only, no upgrades.
- **Kubebuilder `templates/crd`**: CRDs managed like other manifests, upgrades included.

This design choice prioritizes correctness and maintainability over Helm's default convention, while leaving room for future improvements (such as scaffolding separate charts for APIs and controllers).

</aside>

## Values configuration

The generated `values.yaml` provides configuration options extracted from your actual deployment.
Namespace creation is not managed by the chart; use Helm's `--namespace` and `--create-namespace` flags when installing.

### How values are formatted

Values are uncommented when:
- Extracted from kustomize source manifests
- Standard Helm fields (replicas, image, resource names)

Values stay commented when:
- Optional Kubernetes features not in use (imagePullSecrets, priorityClassName)
- Advanced configuration not needed for basic usage (topology spread, pod disruption budget)
- User customization fields (name overrides, custom labels)

**Example:**

```yaml
{{#include ../../getting-started/testdata/project/dist/chart/values.yaml}}
```

### Installation

The plugin adds Helm targets to your `Makefile`:

```bash
make helm-deploy IMG=<registry>/<project:tag>
make helm-status
```

Install manually with all features enabled:

```bash
helm install my-release ./dist/chart --namespace my-project-system --create-namespace
```

Install only CRDs and RBAC:

```bash
helm install my-release ./dist/chart --set manager.enabled=false --set webhook.enable=false
```

Install without webhooks:

```bash
helm install my-release ./dist/chart --set webhook.enable=false --set certManager.enable=false
```

### Extra volumes

Add volumes and volume mounts to the manager deployment beyond webhook and metrics certificates.

Volumes in your kustomize configuration (`config/manager/manager.yaml` or patches) are written to the chart template. When the manager has extra volumes, `values.yaml` includes `manager.extraVolumes` and `manager.extraVolumeMounts` fields. Use these to add more volumes at install time.

Webhook and metrics certificates (`webhook-certs`, `metrics-certs`) are managed separately and controlled by `certManager.enable` and `metrics.enable`.

### Metrics configuration

#### `metrics.secure`

Control transport security and authentication for the metrics endpoint (default: `true`).

When `true`:
- Uses HTTPS with TLS certificates (when `certManager.enable=true`)
- Creates `metrics-auth-role` ClusterRole for authentication
- ServiceMonitor uses HTTPS

When `false`:
- Uses HTTP without authentication
- No TLS certificates
- ServiceMonitor uses HTTP

<aside class="note" role="note">
<p class="note-title">Metrics roles are always cluster-scoped</p>

The `metrics-auth-role` and `metrics-reader` are always ClusterRoles, even when `rbac.namespaced=true`. Metrics authentication uses cluster-scoped APIs for authenticating scrapers like Prometheus.

</aside>

### Custom labels and annotations

Add custom labels and annotations using `manager.labels`, `manager.annotations`, `manager.pod.labels`, and `manager.pod.annotations`. Duplicate keys from kustomize are filtered automatically.

### ServiceAccount configuration

Set `serviceAccount.enable: true` (default) to create a ServiceAccount. Set `serviceAccount.enable: false` to use an existing one.

Add annotations for cloud provider integrations:

```yaml
serviceAccount:
  enable: true
  annotations:
    iam.gke.io/gcp-service-account: my-operator@project.iam.gserviceaccount.com
```

External ServiceAccount names are used as-is and ignore `nameOverride` or `fullnameOverride`.

### RBAC configuration

#### `rbac.namespaced`

Set the scope of RBAC permissions:

- `false` (default): ClusterRole and ClusterRoleBinding for all namespaces
- `true`: Role and RoleBinding for release namespace only

<aside class="note" role="note">
<p class="note-title">What rbac.namespaced controls</p>

This controls RBAC permissions only. To control watch scope, use `WATCH_NAMESPACE` ([Manager Scope](../../reference/manager-scope.md)). The `metrics-auth-role` is always a ClusterRole and `leader-election-role` is always a Role.

</aside>

#### `rbac.roleNamespaces`

When your kustomize output includes Roles and RoleBindings for specific namespaces (other than the manager namespace), the plugin automatically detects them and creates `roleNamespaces` entries.

Add namespace-specific RBAC markers to your controller:

```go
// +kubebuilder:rbac:groups=apps,namespace=infrastructure,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups="",namespace=users,resources=secrets,verbs=get;list;watch
```

Run `make manifests` to generate the RBAC manifests, then run the plugin. The generated `values.yaml` includes:

```yaml
rbac:
  namespaced: false

  ## Namespace configuration for Roles deployed to namespaces different from the manager namespace
  ## Keys are resource name suffixes (without project prefix)
  ##
  roleNamespaces:
    # RBAC resource manager-role-infrastructure deploys to namespace infrastructure
    "manager-role-infrastructure": "infrastructure"
    # RBAC resource manager-rolebinding-infrastructure deploys to namespace infrastructure
    "manager-rolebinding-infrastructure": "infrastructure"
    # RBAC resource manager-role-users deploys to namespace users
    "manager-role-users": "users"
    # RBAC resource manager-rolebinding-users deploys to namespace users
    "manager-rolebinding-users": "users"

  helpers:
    enable: false
```

Override namespaces at deployment using a values file:

```yaml
# custom-values.yaml
rbac:
  roleNamespaces:
    "manager-role-infrastructure": "prod-infra"
    "manager-rolebinding-infrastructure": "prod-infra"
    "manager-role-users": "prod-users"
    "manager-rolebinding-users": "prod-users"
```

Install with custom namespaces:

```bash
helm install my-operator ./dist/chart -f custom-values.yaml
```

Or use `--set`:

```bash
helm install my-operator ./dist/chart \
  --set 'rbac.roleNamespaces[manager-role-infrastructure]=prod-infra' \
  --set 'rbac.roleNamespaces[manager-role-users]=prod-users'
```

<aside class="note" role="note">
<p class="note-title">Helper roles and optional values</p>

Set `rbac.helpers.enable: true` to create admin, editor, and viewer roles for Custom Resources.

Optional fields in `values.yaml` use Helm conditionals. Comment them out to exclude them from deployed manifests.

</aside>

## Flags

| Flag                | Description                                                                 |
|---------------------|-----------------------------------------------------------------------------|
| **--manifests**     | Path to YAML file containing Kubernetes manifests (default: `dist/install.yaml`) |
| **--output-dir** string | Output directory for chart (default: `dist`)                                |
| **--force**         | Regenerates preserved files except `Chart.yaml` (`values.yaml`, `NOTES.txt`, `_helpers.tpl`, `.helmignore`, `test-chart.yml`) |

<aside class="note" role="note">
<p class="note-title"> Examples </p>

You can find example projects in [testdata/project-v4-with-plugins](https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/project-v4-with-plugins).

</aside>
