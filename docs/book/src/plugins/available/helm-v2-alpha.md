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
- **File Preservation**: `Chart.yaml` is never overwritten. Without `--force`, `values.yaml`, `NOTES.txt`, `_helpers.tpl`, `.helmignore`, and `.github/workflows/test-chart.yml` are preserved.
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

### Removing Helm Charts

To remove the Helm chart from your project:

```shell
kubebuilder delete --plugins helm/v2-alpha
```

This removes:
- `dist/chart/` directory
- `.github/workflows/test-chart.yml`
- Plugin configuration from PROJECT file

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

<aside class="note">
<H1>Chart Structure</H1>

The chart structure mirrors your project's resources:

- Standard resources (RBAC, manager, webhooks, CRDs) go into dedicated template directories
- Other resources (Services, ConfigMaps, Secrets) go into `templates/extras/` with Helm templating
- **Custom Resource instances** from `config/samples/` are **not included in the chart**

By default, `make build-installer` does not include samples in `dist/install.yaml`. If you manually add CR instances to your kustomize output, the Helm plugin will ignore them.

</aside>

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

## Post-Install Notes

The plugin generates a `NOTES.txt` template that displays helpful information after `helm install` or `helm upgrade`:

- Installation confirmation with release name and namespace
- Commands to verify the deployment (kubectl get pods, CRDs)
- How to get more information using helm commands

The `NOTES.txt` file is preserved on subsequent runs (unless `--force` is used), allowing you to customize the post-install message for your users.

## Values Configuration

The generated `values.yaml` provides configuration options extracted from your actual deployment.
Namespace creation is not managed by the chart; use Helm's `--namespace` and `--create-namespace` flags when installing.

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

  ## Image pull secrets
  ##
  imagePullSecrets: []
  # Example:
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

## Helper RBAC roles for managing custom resources
##
rbacHelpers:
  # Install convenience admin/editor/viewer roles for CRDs
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

### Installation

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
| **--force**         | Regenerates preserved files except `Chart.yaml` (`values.yaml`, `NOTES.txt`, `_helpers.tpl`, `.helmignore`, `test-chart.yml`) |

<aside class="note">
<H1> Examples </H1>

You can find example projects in [testdata/project-v4-with-plugins](https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/project-v4-with-plugins).

</aside>
