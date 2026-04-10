# Helm Plugin `(helm/v2-alpha)`

The Helm plugin **v2-alpha** provides a way to package your project as a Helm chart, enabling distribution in Helm’s native format.
Instead of using static templates, this plugin dynamically generates Helm charts from your project’s **kustomize output** (via `make build-installer`).
It keeps your custom settings such as environment variables, labels, annotations, and security contexts.

This lets you deliver your Kubebuilder project in two ways:
- As a **bundle** (`dist/install.yaml`) generated with kustomize
- As a **Helm chart** that matches the same output

## Why Helm?

By default, you can create a bundle of manifests with:

```shell
make build-installer IMG=<registry>/<project-name:tag>
```

Users can install it directly:

```shell
kubectl apply -f https://raw.githubusercontent.com/<org>/project-v4/<tag-or-branch>/dist/install.yaml
```
But many people prefer Helm for packaging, upgrades, and distribution.
The **helm/v2-alpha** plugin converts the bundle (`dist/install.yaml`) into a Helm chart that mirrors your project.

## Key Features

- **Dynamic Generation**: Charts are built from real kustomize output, not boilerplate.
- **Preserves Customizations**: Keeps env vars, labels, annotations, and patches.
- **Structured Output**: Templates follow your `config/` directory layout.
- **Smart Values**: `values.yaml` includes only actual configurable parameters.
- **File Preservation**: `Chart.yaml` is never overwritten. Without `--force`, `values.yaml`, `NOTES.txt`, `_helpers.tpl`, `.helmignore` and `.github/workflows/test-chart.yml` are preserved.
- **Handles Custom Resources**: Resources not matching standard layout (custom Services, ConfigMaps, etc.) are placed in `templates/extras/` with proper templating.

## When to Use It

Use the **helm/v2-alpha** plugin if:
- You want Helm charts that stay true to your kustomize setup
- You need charts that update with your project automatically
- You want a clean template layout similar to `config/`
- You want to distribute your solution using either this format

## Usage

### Basic Workflow

```shell
# Create a new project
kubebuilder init

# Build the installer bundle
make build-installer IMG=<registry>/<project:tag>

# Create Helm chart from kustomize output
kubebuilder edit --plugins=helm/v2-alpha

# Regenerate preserved files (Chart.yaml never overwritten)
kubebuilder edit --plugins=helm/v2-alpha --force
```

### Advanced Options

```shell
# Use a custom manifests file
kubebuilder edit --plugins=helm/v2-alpha --manifests=manifests/custom-install.yaml

# Write chart to a custom output directory
kubebuilder edit --plugins=helm/v2-alpha --output-dir=charts

# Combine manifests and output
kubebuilder edit --plugins=helm/v2-alpha \
  --manifests=manifests/install.yaml \
  --output-dir=helm-charts
```

## Chart Structure

The plugin creates a chart layout that matches your `config/`:

```shell
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
    │   ├── memcached-admin-role.yaml
    │   ├── memcached-editor-role.yaml
    │   ├── memcached-viewer-role.yaml
    │   ├── busybox-admin-role.yaml
    │   ├── busybox-editor-role.yaml
    │   ├── busybox-viewer-role.yaml
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
<p class="note-title">Chart Structure</p>

The chart structure mirrors your project's resources:

- Standard resources (RBAC, manager, webhooks, CRDs) go into dedicated template directories
- Other resources (Services, ConfigMaps, Secrets) go into `templates/extras/` with Helm templating
- **Custom Resource instances** from `config/samples/` are **not included in the chart**

By default, `make build-installer` does not include samples in `dist/install.yaml`. If you manually add CR instances to your kustomize output, the Helm plugin will ignore them.

</aside>

<aside class="note" role="note">
<p class="note-title"> Why CRDs are added under templates? </p>

Although [Helm best practices](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#method-1-let-helm-do-it-for-you) recommend placing CRDs under a top-level `crds/` directory, the Kubebuilder Helm plugin intentionally places them under `templates/crd`.

The rationale is tied to how Helm itself handles CRDs.
By default, Helm will install CRDs once during the initial release,
but it will **ignore CRD changes** on subsequent upgrades.

This can lead to surprising behavior where chart upgrades silently
skip CRD updates, leaving clusters out of sync.

To avoid endorsing this behavior, the Kubebuilder plugin follows the approach of packaging
CRDs inside `templates/`. In this mode, Helm treats CRDs like
any other resource, ensuring they are applied and upgraded as expected.
While this prevents mixing CRDs and CRs of the same type in a single chart (since Helm cannot wait between creation steps), it ensures predictable and explicit lifecycle management of CRDs.

In short:
- **Helm `crds/` directory**: one-time install only, no upgrades.
- **Kubebuilder `templates/crd`**: CRDs managed like other manifests, upgrades included.

This design choice prioritizes correctness and maintainability over Helm's default convention,
while leaving room for future improvements (such as scaffolding separate charts for APIs and controllers).
</aside>

## Post-Install Notes

The plugin generates a `NOTES.txt` template that displays helpful information after `helm install` or `helm upgrade`:

- Installation confirmation with release name and namespace
- Commands to verify the deployment (kubectl get pods, CRDs)
- How to get more information using helm commands

The `NOTES.txt` file is preserved on subsequent runs (unless `--force` is used), allowing you to customize the post-install message for your users.

## Values Configuration

The generated `values.yaml` provides configuration options extracted from your actual deployment.
Namespace creation is not managed by the chart; use Helm's `--namespace` and `--create-namespace` flags when installing.

### How Fields Are Exposed

The plugin intelligently decides which fields to include in `values.yaml` based on their purpose:

**Operator-specific runtime configuration (only if found in kustomize):**

These fields are part of your operator's runtime contract and only appear when present in your deployment:
- `manager.args` - Controller manager arguments
- `manager.env` - Environment variables
- `manager.envOverrides` - Environment variable overrides (CLI --set)
- `manager.extraVolumes` / `manager.extraVolumeMounts` - Additional volumes

**Optional Kubernetes features (commented unless found in kustomize):**

These are valid deployment options shown as commented examples when not used. If found in kustomize, they appear uncommented with actual values:
- `manager.imagePullSecrets` - Registry credentials
- `manager.podSecurityContext` - Pod-level security settings
- `manager.securityContext` - Container-level security settings
- `manager.resources` - Resource limits and requests
- `manager.strategy` - Deployment strategy (RollingUpdate/Recreate)
- `manager.priorityClassName` - Pod scheduling priority
- `manager.topologySpreadConstraints` - High availability scheduling
- `manager.terminationGracePeriodSeconds` - Graceful shutdown period

**Standard Helm configuration (always exposed):**

These are standard Helm fields always present for user customization:
- `nameOverride` / `fullnameOverride` - Chart naming (commented by default)
- `manager.replicas` - Pod replica count
- `manager.image.*` - Container image configuration
- `manager.affinity` - Pod affinity rules
- `manager.nodeSelector` - Node selection
- `manager.tolerations` - Node tolerations

**Important:** If a value is defined in kustomize, it will appear uncommented in `values.yaml` with the actual value. This ensures the Helm chart mirrors your operator's configuration exactly.

**Example**

```yaml
## String to partially override chart.fullname template (will maintain the release name)
##
# nameOverride: ""

## String to fully override chart.fullname template
##
# fullnameOverride: ""

## Configure the controller manager deployment
##
manager:
  ## Set to false to skip manager installation.
  ## Defaults to true when this field is missing (backward compatibility).
  ##
  enabled: true

  replicas: 1

  image:
    repository: controller
    tag: latest
    pullPolicy: IfNotPresent

  ## Arguments
  ##
  args:
    - --leader-elect

  ## Environment variables
  ##
  env:
    - name: BUSYBOX_IMAGE
      value: busybox:1.36.1
    - name: MEMCACHED_IMAGE
      value: memcached:1.6.26-alpine3.19

  ## Env overrides (--set manager.envOverrides.VAR=value)
  ## Same name in env above: this value takes precedence.
  ##
  envOverrides: {}

  ## Image pull secrets
  ##
  # imagePullSecrets:
  #   - name: myregistrykey

  ## Pod-level security settings
  ##
  podSecurityContext:
    runAsNonRoot: true
    seccompProfile:
        type: RuntimeDefault

  ## Container-level security settings
  ##
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
        drop:
            - ALL
    readOnlyRootFilesystem: true

  ## Resource limits and requests
  ##
  resources:
    limits:
        cpu: 500m
        memory: 128Mi
    requests:
        cpu: 10m
        memory: 64Mi

  ## Manager pod's affinity
  ##
  affinity: {}
  # Example:
  # affinity:
  #   nodeAffinity:
  #     requiredDuringSchedulingIgnoredDuringExecution:
  #       nodeSelectorTerms:
  #         - matchExpressions:
  #           - key: kubernetes.io/arch
  #             operator: In
  #             values:
  #               - amd64
  #               - arm64

  ## Manager pod's node selector
  ##
  nodeSelector: {}
  # Example:
  # nodeSelector:
  #   kubernetes.io/os: linux
  #   disktype: ssd

  ## Manager pod's tolerations
  ##
  tolerations: []
  # Example:
  # tolerations:
  #   - key: "node.kubernetes.io/unreachable"
  #     operator: "Exists"
  #     effect: "NoExecute"
  #     tolerationSeconds: 6000

  ## Deployment strategy
  ##
  # strategy:
  #   type: RollingUpdate
  #   rollingUpdate:
  #     maxSurge: 25%
  #     maxUnavailable: 25%

  ## Priority class name
  ##
  # priorityClassName: ""

  ## Topology spread constraints
  ##
  # topologySpreadConstraints: []
  # Example:
  # topologySpreadConstraints:
  #   - maxSkew: 1
  #     topologyKey: kubernetes.io/hostname
  #     whenUnsatisfiable: DoNotSchedule
  #     labelSelector:
  #       matchLabels:
  #         app.kubernetes.io/name: project

  ## Termination grace period seconds
  ##
  terminationGracePeriodSeconds: 10

  ## Custom Deployment labels
  ##
  # labels: {}

  ## Custom Deployment annotations
  ##
  # annotations: {}

  ## Custom Pod labels and annotations
  ##
  # pod:
  #   labels: {}
  #   annotations: {}

## RBAC configuration
##
rbac:
  ## RBAC resource scope
  ## - false (default): ClusterRole/ClusterRoleBinding (all namespaces)
  ## - true: Role/RoleBinding (release namespace only)
  ##
  namespaced: false

  ## Helper roles for CRD management (admin/editor/viewer)
  ##
  helpers:
    ## Install convenience admin/editor/viewer roles for CRDs
    ##
    enable: false

## Custom Resource Definitions
##
crd:
  # Install CRDs with the chart
  enable: true
  # Keep CRDs when uninstalling
  keep: true

## Controller metrics endpoint.
## Enable to expose /metrics endpoint with RBAC protection.
##
metrics:
  enable: true
  # Metrics server port
  port: 8443

## Cert-manager integration for TLS certificates.
## Required for webhook certificates and metrics endpoint certificates.
##
certManager:
  enable: true

## Webhook server configuration
##
webhook:
  enable: true
  # Webhook server port
  port: 9443

## Prometheus ServiceMonitor for metrics scraping.
## Requires prometheus-operator to be installed in the cluster.
##
prometheus:
  enable: false
```

### Common Installation Patterns

**CRD and RBAC only installation** (for CRD management separate from the operator):

```shell
helm install my-release ./dist/chart \
  --set manager.enabled=false \
  --set webhook.enable=false \
  --set certManager.enable=false \
  --set metrics.enable=false
```

**Manager without webhooks** (e.g., operator without admission control):

```shell
helm install my-release ./dist/chart \
  --set webhook.enable=false \
  --set certManager.enable=false
```

**Full installation** (default - all features enabled):

```shell
helm install my-release ./dist/chart
```

### RBAC Configuration

#### `rbac.namespaced`

Controls the scope of RBAC permissions:

- **`false` (default)**: ClusterRole/ClusterRoleBinding (all namespaces)
- **`true`**: Role/RoleBinding (release namespace only)

<aside class="note" role="note">
<p class="note-title">Important Notes</p>

- This controls RBAC permissions only. To control watch scope, use `WATCH_NAMESPACE` ([Manager Scope](../../reference/manager-scope.md)).
- `metrics-auth-role` is always a `ClusterRole` ([Metrics Configuration](#metrics-configuration)).
- `leader-election-role` is always a `Role`.

</aside>

#### `rbac.roleNamespaces`

When your controller requires RBAC permissions in specific namespaces, the Helm plugin automatically detects Roles and RoleBindings deployed to non-manager namespaces and creates separate `roleNamespaces` entries for each resource (both the Role and its corresponding RoleBinding).

**Example scenario:**

Your controller needs to manage deployments in an `infrastructure` namespace and secrets in a `users` namespace:

```go
// +kubebuilder:rbac:groups=apps,namespace=infrastructure,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups="",namespace=users,resources=secrets,verbs=get;list;watch
```

The plugin detects the RBAC resource-to-namespace mappings and generates:

**`values.yaml`:**
```yaml
rbac:
  ## RBAC resource scope
  ## - false (default): ClusterRole/ClusterRoleBinding (all namespaces)
  ## - true: Role/RoleBinding (release namespace only)
  ##
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

  ## Helper roles for CRD management (admin/editor/viewer)
  ##
  helpers:
    ## Install convenience admin/editor/viewer roles for CRDs
    ##
    enable: false
```

**Generated templates:**
```yaml
# Role template
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ include "my-operator.resourceName" (dict "suffix" "manager-role-infrastructure" "context" $) }}
  namespace: {{ index .Values.rbac.roleNamespaces "manager-role-infrastructure" | default "infrastructure" }}
rules:
- apiGroups: [apps]
  resources: [deployments]
  verbs: [get, list, watch]
---
# RoleBinding template
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ include "my-operator.resourceName" (dict "suffix" "manager-rolebinding-infrastructure" "context" $) }}
  namespace: {{ index .Values.rbac.roleNamespaces "manager-rolebinding-infrastructure" | default "infrastructure" }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: {{ include "my-operator.resourceName" (dict "suffix" "manager-role-infrastructure" "context" $) }}
subjects:
- kind: ServiceAccount
  name: {{ include "my-operator.resourceName" (dict "suffix" "controller-manager" "context" $) }}
  namespace: {{ .Release.Namespace }}
```

**Override namespace names at deployment:**

You can override namespaces using a values file:

```yaml
# custom-values.yaml
rbac:
  roleNamespaces:
    "manager-role-infrastructure": "prod-infra"
    "manager-rolebinding-infrastructure": "prod-infra"
    "manager-role-users": "prod-users"
    "manager-rolebinding-users": "prod-users"
```

```shell
helm install my-operator ./dist/chart -f custom-values.yaml
```

Alternatively, use bracket notation with `--set`:
```shell
helm install my-operator ./dist/chart \
  --set 'rbac.roleNamespaces[manager-role-infrastructure]=prod-infra' \
  --set 'rbac.roleNamespaces[manager-rolebinding-infrastructure]=prod-infra' \
  --set 'rbac.roleNamespaces[manager-role-users]=prod-users' \
  --set 'rbac.roleNamespaces[manager-rolebinding-users]=prod-users'
```

<aside class="note" role="note">
<p class="note-title">Namespace Configuration</p>

- **Keys use suffix-based naming** (without project prefix) for stability across different release names and overrides
- Templates use the `index` function with the suffix to access namespace values (e.g., `{{ index .Values.rbac.roleNamespaces "manager-role-infrastructure" | default "infrastructure" }}`), which safely handles names with hyphens
- If a `roleNamespaces` entry is missing, the template falls back to the original namespace from kustomize, preventing `<no value>` errors
- This setting controls RBAC permissions only. To control which namespaces the operator watches, configure the `WATCH_NAMESPACE` environment variable (see [Manager Scope](../../reference/manager-scope.md))

</aside>

#### `rbac.helpers.enable`

Controls whether to create convenience RBAC roles (admin/editor/viewer) for your Custom Resources. Default: `false`.

These helper roles follow Kubernetes conventions and can be bound to users or groups to grant different levels of access to your CRs:
- **admin**: Full CRUD access to the custom resource
- **editor**: Create, update, and delete access (no special permissions)
- **viewer**: Read-only access

### Metrics Configuration

#### `metrics.secure`

Controls transport security and authentication for the metrics endpoint. Default: `true`.

- **`true` (HTTPS with RBAC)**:
  - Uses HTTPS transport (`--metrics-secure=true`, default in controller-runtime)
  - Creates TLS certificates (when `certManager.enable=true`)
  - Creates `metrics-auth-role` ClusterRole for TokenReview/SubjectAccessReview
  - ServiceMonitor uses `scheme: https` with TLS config

- **`false` (HTTP without RBAC)**:
  - Uses HTTP transport (`--metrics-secure=false`)
  - No TLS certificates created
  - No RBAC authentication (metrics endpoint is open)
  - ServiceMonitor uses `scheme: http` without TLS config

<aside class="note" role="note">
<p class="note-title">Metrics RBAC is independent from manager RBAC</p>

The `metrics-auth-role` and `metrics-reader` are **always ClusterRoles**, even when `rbac.namespaced=true`.

This is because metrics authentication uses cluster-scoped APIs ([`TokenReview`/`SubjectAccessReview`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/metrics/filters#WithAuthenticationAndAuthorization)) for authenticating metric scrapers (like Prometheus), which is separate from the manager's operational RBAC for managing resources.

**You can use `rbac.namespaced=true` with `metrics.secure=true`** - the manager will use namespace-scoped Roles while metrics authentication uses cluster-scoped ClusterRoles.

</aside>

### Extra volumes

The chart supports additional volumes and volume mounts for the manager (e.g. secrets, config files), alongside the built-in webhook and metrics cert volumes.

- **Config volumes**: Volumes in the manager deployment (e.g. `config/manager/manager.yaml` or kustomize patches) are written into the chart template. Re-running `kubebuilder edit --plugins=helm/v2-alpha` updates the template from config; `values.yaml` is not overwritten.
- **Values**: When the manager deployment has extra volumes (other than webhook/metrics), `values.yaml` gets `manager.extraVolumes` and `manager.extraVolumeMounts`. Use them to add more entries; the template appends them after the config volumes. Same structure as in a Pod spec; mount names must match volume names.

Webhook and metrics (`webhook-certs`, `metrics-certs`) are not in `extraVolumes`. They are conditional on `certManager.enable` and `metrics.enable`, like the rest of the chart.

### Template Conditionals

Operator-specific and optional fields use Helm conditionals in templates matching the patterns in `templates/manager/manager.yaml`:

```yaml
# Operator-specific - env always renders the key, with [] when unset
env:
{{- if .Values.manager.env }}
  {{- toYaml .Values.manager.env | nindent 2 }}
{{- else }}
  []
{{- end }}

# Optional feature - strategy renders only when defined (uses 'with')
{{- with .Values.manager.strategy }}
strategy:
  {{- toYaml . | nindent 6 }}
{{- end }}

# terminationGracePeriodSeconds renders when the key exists, even if set to 0
{{- if and (hasKey .Values.manager "terminationGracePeriodSeconds") (ne .Values.manager.terminationGracePeriodSeconds nil) }}
terminationGracePeriodSeconds: {{ .Values.manager.terminationGracePeriodSeconds }}
{{- end }}
```

This means:
- If you comment out an optional field in `values.yaml`, it won't appear in the deployed manifests
- You can safely uncomment optional fields to enable Kubernetes features
- Operator-specific fields (env, args) always render when present in templates
- Zero values like `terminationGracePeriodSeconds: 0` work correctly with `hasKey`

### Custom Labels and Annotations

Add custom labels and annotations to the manager Deployment and Pod template:

- **`manager.labels`**: Custom labels for the Deployment (e.g., team, environment)
- **`manager.annotations`**: Custom annotations for the Deployment
- **`manager.pod.labels`**: Custom labels for the Pod template
- **`manager.pod.annotations`**: Custom annotations for the Pod template (e.g., Prometheus metrics)

<aside class="note" role="note">
<p class="note-title">Duplicate Key Filtering</p>

Duplicate keys are automatically filtered - existing keys from the kustomize output (such as `control-plane`) are detected and excluded from your custom values to prevent conflicts.

</aside>

### ServiceAccount Configuration

The chart provides flexible ServiceAccount management through the `serviceAccount` configuration block:

- **`serviceAccount.enable`**: Controls whether the chart creates a ServiceAccount (default: `true`)
- **`serviceAccount.name`**: Specifies an existing ServiceAccount name (only used when `enable=false`)
- **`serviceAccount.labels`**: Custom labels for the ServiceAccount (only applied when `enable=true`)
- **`serviceAccount.annotations`**: Custom annotations for the ServiceAccount (only applied when `enable=true`)

**Example values.yaml:**

```yaml
serviceAccount:
  # Install default ServiceAccount provided
  enable: true

  ## Existing ServiceAccount name (only when enable=false)
  ## Note: When enable=true, respects nameOverride/fullnameOverride
  ##
  # name: ""

  ## Custom ServiceAccount annotations
  ##
  # annotations: {}

  ## Custom ServiceAccount labels
  ##
  # labels: {}
```

#### Default ServiceAccount (enable=true)

When `enable: true` (default), the chart creates a ServiceAccount with a name that respects `nameOverride` and `fullnameOverride`:

```shell
# Default: <release-name>-<project-name>-controller-manager
helm install my-release ./dist/chart

# With nameOverride: <release-name>-myname-controller-manager
helm install my-release ./dist/chart --set nameOverride=myname

# With fullnameOverride: myfullname-controller-manager
helm install my-release ./dist/chart --set fullnameOverride=myfullname
```

Add custom labels and annotations for cloud provider integrations (e.g., workload identity):

```yaml
serviceAccount:
  enable: true
  annotations:
    iam.gke.io/gcp-service-account: my-operator@project.iam.gserviceaccount.com
  labels:
    team: platform
    environment: production
```

#### External ServiceAccount (enable=false)

When `enable: false`, the chart skips creating a ServiceAccount and uses the name specified in `serviceAccount.name`. This is useful when:
- The ServiceAccount is managed externally (e.g., by a security team)
- You need to use a pre-existing ServiceAccount with specific permissions

```yaml
serviceAccount:
  enable: false
  name: external-service-account
```

<aside class="note" role="note">
<p class="note-title">External ServiceAccount Naming</p>

When using an external ServiceAccount (`enable=false`), the `serviceAccount.name` is used as-is and **does not** respect `nameOverride` or `fullnameOverride`. This ensures the chart references exactly the ServiceAccount you specify.

Custom `labels` and `annotations` are ignored when `enable=false` since the ServiceAccount is not created by the chart.

</aside>

### Installation

The first time you run the plugin, it adds convenient Helm deployment targets to your `Makefile`:

```shell
make helm-deploy IMG=<registry>/<project:tag>  # Deploy/upgrade the chart
make helm-status                                # Check release status
make helm-history                               # View release history
make helm-rollback                              # Rollback to previous version
make helm-uninstall                             # Remove the release
```

You can also install manually using Helm commands:

```shell
helm install my-release ./dist/chart \
  --namespace my-project-system \
  --create-namespace
```

The Makefile targets use sensible defaults extracted from your project configuration (namespace from manifests, release name from project name, chart directory from `--output-dir` flag).

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
