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
- **File Preservation**: Manual edits in `values.yaml`, `Chart.yaml`, `_helpers.tpl` are kept unless `--force` is used.

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

# Overwrite preserved files if needed
kubebuilder edit --plugins=helm/v2-alpha --force
```

### Advanced Options

```shell
# Use a custom manifests file
kubebuilder edit --plugins=helm/v2-alpha --manifests=manifests/custom-install.yaml

# Write chart to a custom output directory
kubebuilder edit --plugins=helm/v2-alpha --output=charts

# Combine manifests and output
kubebuilder edit --plugins=helm/v2-alpha \
  --manifests=manifests/install.yaml \
  --output=helm-charts
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
    ├── namespace.yaml
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
    └── prometheus/
        └── servicemonitor.yaml
```

## Values Configuration

The generated `values.yaml` provides configuration options extracted from your actual deployment, such as:

```yaml
# Default values for project-v4-with-plugins.
# This chart is generated from your kustomize manifests.
# Control namespace creation and naming
namespace:
  create: true  # Create the namespace (if false, assumes namespace exists)
  name: ""      # Override namespace name (defaults to "<release-name>-system")

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

# Essential RBAC permissions (required for controller operation)
# These include ServiceAccount, controller permissions, leader election, and metrics access
# Note: Essential RBAC is always enabled as it's required for the controller to function

# Helper RBAC roles for managing custom resources
# These provide convenient admin/editor/viewer roles for each CRD type
# Useful for giving users different levels of access to your custom resources
rbacHelpers:
  enable: false  # Install convenience admin/editor/viewer roles for CRDs

# Custom Resource Definitions
crd:
  enable: true  # Install CRDs with the chart
  keep: true    # Keep CRDs when uninstalling

# Controller metrics endpoint is disabled by default (--metrics-bind-address=0).
# Enable to expose /metrics endpoint with RBAC protection.
metrics:
  enable: false

# Cert-manager integration for TLS certificates.
# Required for webhook certificates and metrics endpoint certificates.
certManager:
  enable: true

# Prometheus ServiceMonitor for metrics scraping.
# Requires prometheus-operator to be installed in the cluster.
prometheus:
  enable: false
```

## Flags

| Flag                | Description                                                                 |
|---------------------|-----------------------------------------------------------------------------|
| **--manifests**     | Path to YAML file containing Kubernetes manifests (default: `dist/install.yaml`) |
| **--output** string | Output directory for chart (default: `dist`)                                |
| **--force**         | Overwrites preserved files (`values.yaml`, `Chart.yaml`, `_helpers.tpl`)    |

<aside class="note">
<H1> Examples </H1>

You can find example projects in [testdata/project-v4-with-plugins](https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/project-v4-with-plugins).

</aside>