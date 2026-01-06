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
- **File Preservation**: `Chart.yaml` is never overwritten. Without `--force`, `values.yaml`, `_helpers.tpl`, `.helmignore`, and `.github/workflows/test-chart.yml` are preserved.
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
<output>/chart/
├── Chart.yaml
├── values.yaml
├── .helmignore
└── templates/
    ├── _helpers.tpl
    ├── rbac/                    # Individual RBAC files
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
    │   └── ...
    ├── crd/                     # Individual CRD files
    │   ├── memcacheds.example.com.testproject.org.yaml
    │   ├── busyboxes.example.com.testproject.org.yaml
    │   ├── wordpresses.example.com.testproject.org.yaml
    │   └── ...
    ├── cert-manager/
    │   ├── metrics-certs.yaml
    │   ├── serving-cert.yaml
    │   └── selfsigned-issuer.yaml
    ├── manager/
    │   └── manager.yaml
    ├── service/
    │   └── service.yaml
    ├── webhook/
    │   └── validating-webhook-configuration.yaml
    ├── prometheus/
    │   └── servicemonitor.yaml
    └── extras/                  # Custom resources (if any)
        ├── my-service.yaml
        └── my-config.yaml
```

**Note:** Resources that don't match the standard scaffold layout (custom Services, ConfigMaps, Secrets, etc.)
are automatically placed in `templates/extras/` with proper Helm templating applied (namePrefix, labels, etc.).

<aside class="note">
<H1> Why CRDs are added under templates? </H1>

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
- **Helm `crds/` directory** → one-time install only, no upgrades.
- **Kubebuilder `templates/crd`** → CRDs managed like other manifests, upgrades included.

This design choice prioritizes correctness and maintainability over Helm's default convention,
while leaving room for future improvements (such as scaffolding separate charts for APIs and controllers).
</aside>

## Values Configuration

The generated `values.yaml` provides configuration options extracted from your actual deployment.
Namespace creation is not managed by the chart; use Helm's `--namespace` and `--create-namespace` flags when installing.

**Example**

```yaml
# Configure the controller manager deployment
controllerManager:
  replicas: 1

  image:
    repository: controller
    tag: latest
    pullPolicy: IfNotPresent

  # Environment variables from your deployment
  env:
    - name: BUSYBOX_IMAGE
      value: busybox:1.36.1
    - name: MEMCACHED_IMAGE
      value: memcached:1.6.26-alpine3.19

  # Pod-level security settings
  podSecurityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault

  # Container-level security settings
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
        - ALL
    readOnlyRootFilesystem: true

  # Resource limits and requests
  resources:
    limits:
      cpu: 500m
      memory: 128Mi
    requests:
      cpu: 10m
      memory: 64Mi

  # Manager pod's affinity
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
          - matchExpressions:
            - key: kubernetes.io/arch
              operator: In
              values:
                - amd64
                - arm64
                - ppc64le
                - s390x
            - key: kubernetes.io/os
              operator: In
              values:
                - linux

  # Manager pod's node selector
  nodeSelector:
    kubernetes.io/os: linux

  # Manager pod's tolerations
  tolerations:
    - key: "node.kubernetes.io/unreachable"
      operator: "Exists"
      effect: "NoExecute"
      tolerationSeconds: 6000

# Essential RBAC permissions (required for controller operation)
# These include ServiceAccount, controller permissions, leader election, and metrics access
# Note: Essential RBAC is always enabled as it's required for the controller to function

# Extra labels applied to all rendered manifests
commonLabels: {}

# Helper RBAC roles for managing custom resources
# These provide convenient admin/editor/viewer roles for each CRD type
# Useful for giving users different levels of access to your custom resources
rbacHelpers:
  enable: false  # Install convenience admin/editor/viewer roles for CRDs

# Custom Resource Definitions
crd:
  enable: true  # Install CRDs with the chart
  keep: true    # Keep CRDs when uninstalling

# Controller metrics endpoint.
# Enable to expose /metrics endpoint with RBAC protection.
metrics:
  enable: true

# Cert-manager integration for TLS certificates.
# Required for webhook certificates and metrics endpoint certificates.
certManager:
  enable: true

# Prometheus ServiceMonitor for metrics scraping.
# Requires prometheus-operator to be installed in the cluster.
prometheus:
  enable: false
```

### Installation Tip

Install the chart into a namespace using Helm flags (the chart does not create namespaces):

```shell
helm install my-release ./dist/chart \
  --namespace my-project-system \
  --create-namespace
```

## Flags

| Flag                | Description                                                                 |
|---------------------|-----------------------------------------------------------------------------|
| **--manifests**     | Path to YAML file containing Kubernetes manifests (default: `dist/install.yaml`) |
| **--output-dir** string | Output directory for chart (default: `dist`)                                |
| **--force**         | Regenerates preserved files except `Chart.yaml` (values.yaml, _helpers.tpl, .helmignore, test-chart.yml) |

<aside class="note">
<H1> Examples </H1>

You can find example projects in [testdata/project-v4-with-plugins](https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/project-v4-with-plugins).

</aside>
