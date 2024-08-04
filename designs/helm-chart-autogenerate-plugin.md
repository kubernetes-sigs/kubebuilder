| Authors                                | Creation Date | Status      | Extra |
|----------------------------------------|---------------|-------------|---|
| @dashanji,@camilamacedo86,@LCaparelli  | Sep, 2023 | Implementeble | - |

# New Plugin to allow project distribution via helm charts

This proposal aims to introduce an optional mechanism that allows users
to generate a Helm Chart from their Kubebuilder-scaffolded project.

This will enable them to effectively package and distribute their solutions.
To achieve this goal, we are proposing a new native Kubebuilder
[plugin](https://book.kubebuilder.io/plugins/plugins) (i.e., `helm-chart/v1-alpha`)
which will provide the necessary scaffolds.

The plugin will function similarly to the existing [Grafana Plugin](https://book.kubebuilder.io/plugins/grafana-v1-alpha),
generating or regenerating HelmChart files using the init and edit
sub-commands (i.e., `kubebuilder init|edit --plugins helm-chart/v1-alpha`).

An alternative solution could be to implement an alpha command,
similar to the [helper provided to upgrade projects](https://book.kubebuilder.io/reference/rescaffold) that would
provide the HelmChart under the `dist`directory, similar to what
is done by [helmify](https://github.com/arttor/helmify).

## Example

**To enable the helm-chart generation when a project is initialized**

> kubebuilder init --plugins=`go/v4,helm-chart/v1-alpha`

**To enable the helm-chart generation after the project be scaffolded OR to update the helm-chart files**

> kubebuilder edit --plugins=`go/v4,helm-chart/v1-alpha`

The HelmChart should be scaffold under the `dist/` directory:
```shell
├── PROJECT
...
├── dist
...
├── helm-charts
│   └── {{ projectName }}-chart
...
```

## Open Questions

### How to manage and scaffold the CRDs for the HelmChart?

According to [Helm Best Practices for Custom Resource Definitions](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#method-1-let-helm-do-it-for-you),
there are two main methods for handling CRDs:

- **Method 1:Let Helm Do It For You:**  Place CRDs in the `crds/` directory. Helm installs these CRDs during the initial
  install but does not manage upgrades or deletions.
- **Method 2:Separate Charts:**  Place the CRD definition in one chart
  and the resources using the CRD in another chart.
  This method requires separate installations for each chart.

**Raised Considerations and Concerns**
- **Use Helm crd directory** The upgraded chart versions will silently ignore CRDs even if they differ from the installed versions. This could lead to surprising and unexpected behavior.
  However, it is how [cert-manager][https://cert-manager.io/docs/installation/helm/]
- **Templates Folder**: Moving CRDs to the `templates` folder facilitates upgrades but uninstalls CRDs when the operator is uninstalled.
- **Separate Helm Chart for CRDs:** This approach allows control over both CRD and operator versions without deleting CRDs when the operator chart is deleted.
- **When Webhooks are used:** If a CRD specifies, for example, a conversion webhook, the "API chart" needs to contain the CRDs and the webhook `service/workload`.
  It would also make sense to include `validating/mutating` webhooks, requiring the scaffolding of separate main modules and image builds for
  webhooks and controllers which does not shows to be compatible with Kubebuilder Golang scaffold.

**Proposed Solution**

Follow the same approach adopted by [Cert-Manager](https://cert-manager.io/docs/installation/helm/).
Add the CRDs under the `template` directory and have a spec in the `values.yaml`
which will define if the CRDs should or not be applied

```shell
helm install|upgrade \
  myrelease \
  --namespace my-namespace \
  --set `crds.enabled=true`
```

Additionally, we might want to consider in the future also
scaffold separate charts for the APIs and support both.
An example of this approach provided as feedback
was [karpenter-provider-aws](https://github.com/aws/karpenter-provider-aws/tree/main/charts).

### How to manage dependencies such as Cert-Manager and Prometheus?

Helm charts allow maintainers to define dependencies via `requirements.yaml`.

However, we can convey that in the initial version of this plugin, we do not need to consider any dependencies.
As it stands today, consumers of the solutions built with Kubebuilder
are responsible for installing their dependencies prior
to installing the project built with Kubebuilder. For example:
- If the project is using cert-manager, or
- If managing metrics is required, ensure that the cluster has Prometheus installed and with the required permissions beforehand.

It seems out of the scope of the project itself to manage these dependencies.
However, it seems that would be fine scaffold the HelmChart file with
prometheus and cert-manager commented or disabled with a comment like `#TODO(user)`
to allow the user uncomment if that please them.

### Why should Kubebuilder maintainers maintain a new plugin to achieve this goal instead of suggest the usage of [helmify]( https://github.com/arttor/helmify)?

Maintaining a new plugin within Kubebuilder for generating Helm charts ensures direct integration into the Kubebuilder workflow, providing a consistent user experience.
It allows for extensibility, enabling other projects to add customizations.
This approach guarantees that the generated Helm charts remain consistent
with Kubebuilder project changes. Furthermore, it centralizes documentation
and support within the Kubebuilder community, providing a unified source
for help and guidance.

## Motivation

Currently, projects scaffolded with Kubebuilder can be distributed via YAML. Users can run
`make build-installer IMG=<some-registry>/<project-name>:tag`, which will generate `dist/install.yaml`.
Therefore, its consumers can install the solution by applying this YAML file, such as:
`kubectl apply -f https://raw.githubusercontent.com/<org>/<project-name>/<tag or branch>/dist/install.yaml`.

However, many adopt solutions require the Helm Chart format, such as FluxCD. Therefore,
maintainers are looking to also provide their solutions via Helm Chart. Users currently face the challenges of lacking
an officially supported distribution mechanism for Helm Charts. They seek to:

- Harness the power of Helm Chart as a package manager for the project, enabling seamless adaptation to diverse deployment environments.
- Take advantage of Helm's dependency management capabilities to simplify the installation process of project dependencies, such as cert-manager.
- Seamlessly integrate with Helm's ecosystem, including FluxCD, to efficiently manage the project.

Consequently, this proposal aims to introduce a method that allows Kubebuilder users to easily distribute their projects through Helm Charts, a strategy that many well-known projects have adopted:

- [mongodb](https://artifacthub.io/packages/helm/mongodb-helm-charts/community-operator)
- [cert-manager](https://cert-manager.io/v1.6-docs/installation/helm/#1-add-the-jetstack-helm-repository)
- [prometheus](https://bitnami.com/stack/prometheus-operator/helm)
- [aws-load-balancer-controller](https://github.com/kubernetes-sigs/aws-load-balancer-controller/tree/main/helm/aws-load-balancer-controller)

**NOTE** For further context see the [discussion topic](https://github.com/kubernetes-sigs/kubebuilder/discussions/3074)

### Goals

- Allow Kubebuilder users distribute their projects using Helm easily.
- Make the best effort to preserve any customizations made to the Helm Charts by the users, which means we will skip syncs in the `values.ymal`.
- Stick with Helm layout definitions and externalize into the relevant values-only options to distribute the default scaffold done by Kubebuilder. We should follow https://helm.sh/docs/chart_best_practices.

### Non-Goals

- Converting any Kustomize configuration to Helm Charts like [helmify](https://github.com/arttor/helmify) does.
- Support the deprecated plugins. This option should be supported from `go/v4` and `kustomize/v2`
- Introduce support for Helm in addition to Kustomize, or replace Kustomize with Helm entirely, similar to the approach
taken by Operator-SDK, thereby allowing users to utilize Helm Charts to build their Project.
- Attend standard practices that deviate from Helm Chart layout, definition, or conventions to workaround its limitations.

### User Stories

- As a developer, I want to be able to generate a helm chart from a kustomize directory so that I can distribute the helm chart to my users. Also, I want the
  generation to be as simple as possible without the need to write any additional duplicate files.
- As a user, I want the helm chart can cover all potential configurations when I deploy it on the Kubernetes cluster.
- As a platform engineer, I want to be able to manage different versions and configurations of a project across multiple clusters and environments based on the same distribution artifact (Helm Chart), with versioning and dependency locking for supply chain security.

### Implementation Details/Notes/Constraints

#### Plugin Layout

- **Location and Versioning**: The new plugin should follow Kubebuilder standards and
be implemented under `pkg/plugins/optional`. It should be introduced as an alpha version
(`v1alpha`), similar to the [Grafana plugin](https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/optional/grafana/v1alpha).

- **The data should be tracked in PROJECT File**: Usage of the plugin should be tracked in the `PROJECT`
file with the input via flags and options if required. Example entry in the `PROJECT` file:

```yaml
...
plugins:
  helm-chart.go.kubebuilder.io/v1-alpha:
    options:
      <flag/key>: <value>
```

Ensure that user-provided input is properly tracked, similar to how it's done in
other plugins [(see the code in the plugin.go)](https://github.com/kubernetes-sigs/kubebuilder/blob/c058fb95fe0ccd8d2a3147990251ca501df5eb26/pkg/plugins/golang/deploy-image/v1alpha1/plugin.go#L58-L75)
and the [(code source to track the data)](https://github.com/kubernetes-sigs/kubebuilder/blob/c058fb95fe0ccd8d2a3147990251ca501df5eb26/pkg/plugins/golang/deploy-image/v1alpha1/api.go#L191-L217)
of the deploy-image plugin for reference.

**NOTE** We might not need options/flags in the first implementation. However, we should still track the plugin as
we do for the Grafana plugin.

#### Plugin Implementation Structure

Following the structure implementation for the source code of this plugin:

```shell
.
├── helm-chart
│   └── v1alpha1
│       ├── init.go
│       ├── edit.go
│       ├── api.go -> Implement any required change in the scaffolds when we run kubebuilder create api
│       ├── webhook.go -> Implement any required change in the scaffolds when we run kubebuilder create webhook
│       ├── plugin.go
│       └── scaffolds
│           ├── init.go
│           ├── edit.go
│           ├── api.go
│           ├── webhook.go
│           └── internal
│               └── templates
```

**Example**

We might want to have only webhook relate scaffold in the HelmChart when/if
the user run `kubebuilder create webhook`. Therefore, the templates and
logic to change the scaffold in this case would be under this subCommand.

Note that we can inject values and/or overwrite then as we do in other places
using the helpers implemented in Kubebuilder. See for example, that we
only add the NetworkPolices relevant to webhooks when those are created in the
project. For further information check its code implementation [here](https://github.com/kubernetes-sigs/kubebuilder/blob/c058fb95fe0ccd8d2a3147990251ca501df5eb26/pkg/plugins/common/kustomize/v2/scaffolds/webhook.go#L93-L146)

#### HelmChart Values Scaffolded by the Plugin

- **Allow values.yaml to be fully re-generated with the flag --force**:

By default, the `values.yaml` file should not
be overwritten. However, users should have the option to overwrite it using
a flag (`--force=true`).

This can be implemented in the specific template as done for other plugins:

```go
if f.Force {
    f.IfExistsAction = machinery.OverwriteFile
} else {
    f.IfExistsAction = machinery.Error
}
```

**NOTE:** We will evaluate the cases when we implement `webhook.go` and `api.go`
for the HelmChart plugin. However, we might use the force flag to replicate
the same behavior implemented in the subCommands of the kustomize plugin.
For instance, if the flag is used when creating an API, it forces
the overwrite of the generated samples. Similarly, if the api subCommand
of the HelmChart plugin is called with `--force`, we should replace
all samples with the latest versions instead of only adding the new one.

- **Helm Chart Templates should have conditions**:

Ensure templates install resources based on
conditions defined in the `values.yaml`. Example for CRDs:

```
# To install CRDs
{{- if .Values.crd.install }}
...
{{- end }}
```

- **Customizable Values**: Set customizable values in the `values.yaml`,
  such as defining ServiceAccount names, and whether they should be created or not.
  Furthermore, we should include comments to help end-users understand the source
  of configurations. Example:

```yaml
{{- if .Values.rbac.create }}
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: project-v4
    app.kubernetes.io/managed-by: kustomize
  name: {{ .Values.rbac.serviceAccountName }}
  namespace: {{ .Release.Namespace }}
{{- end }}
```

- **Layout of the Helm-Chart**:

Following an example of the expected result of this plugin:

```shell
example-project/
  dist/
  helm-chart/
    example-project-crd/
      ├── Chart.yaml
      ├── templates/
      │   ├── _helpers.tpl
      │   ├── crds/
      │   │   └── <CRDs YAML files generated under config/crds/>
      └── values.yaml
    example-project/
      ├── Chart.yaml
      ├── templates/
      │   ├── _helpers.tpl
      │   ├── crds/
      │   └── <CRDs YAML files generated under config/crds/>
      │   ├── certmanager/
      │   │   └── certificate.yaml
      │   ├── default/
      │   │   └── metrics_service.yaml
      │   ├── manager/
      │   │   └── manager.yaml
      │   │   └── namespace.yaml
      │   ├── network-policy/
      │   │   ├── allow-metrics-traffic.yaml
      │   │   └── allow-webhook-traffic.yaml // Should be added by the plugin subCommand webhook.go
      │   ├── prometheus/
      │   │   └── monitor.yaml
      │   ├── rbac/
      │   │   ├── kind_editor_role.yaml
      │   │   ├── kind_viewer_role.yaml
      │   │   ├── leader_election_role.yaml
      │   │   ├── leader_election_role_binding.yaml
      │   │   ├── metrics_auth_role.yaml
      │   │   ├── metrics_auth_role_binding.yaml
      │   │   ├── metrics_reader_role.yaml
      │   │   ├── role.yaml
      │   │   ├── role_binding.yaml
      │   │   └── service_account.yaml
      │   ├── samples/
      │   │   └── kind_version_admiral.yaml
      │   ├── webhook/
      │   │   ├── manifests.yaml
      │   │   └── service.yaml
      └── values.yaml
```

- **Example of values.yaml**:

```yaml
namespace:
  create: true
  name: <projectName>-system

# Install CRDs under the template
crd:
  install: false

# Webhook configuration sourced from the `config/webhook`
webhook:
  enabled: true
  conversion:
    enabled: true

## RBAC configuration under the `config/rbac` directory
rbac:
  create: true
  serviceAccountName: "controller-manager"

# Cert-manager configuration
certmanager:
  enabled: false
  issuerName: "letsencrypt-prod"
  commonName: "example.com"
  dnsName: "example.com"

# Network policy configuration sourced from the `config/network_policy`
networkPolicy:
  enabled: false

# Prometheus configuration
prometheus:
  enabled: false

# Sample configuration sourced from the `config/samples`
samples:
  install: true

# Manager configuration sourced from the `config/manager`
manager:
  replicas: 1
  image:
    repository: "controller"
    tag: "latest"
  resources:
    limits:
      cpu: 100m
      memory: 128Mi
    requests:
      cpu: 100m
      memory: 64Mi

# Metrics configuration sourced from the `config/metrics`
metrics:
  enabled: true

# Leader election configuration sourced from the `config/leader_election`
leaderElection:
  enabled: true
  role: "leader-election-role"
  rolebinding: "leader-election-rolebinding"


# Controller Manager configuration sourced from the `config/manager`
controllerManager:
  manager:
    args:
    - --metrics-bind-address=:8443
    - --leader-elect
    - --health-probe-bind-address=:8081
    containerSecurityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
    image:
      repository: controller
      tag: latest
    resources:
      limits:
        cpu: 500m
        memory: 128Mi
      requests:
        cpu: 10m
        memory: 64Mi
  replicas: 1
  serviceAccount:
    annotations: {}

# Kubernetes cluster domain configuration
kubernetesClusterDomain: cluster.local

# Metrics service configuration sourced from the `config/metrics`
metricsService:
  ports:
  - name: https
    port: 8443
    protocol: TCP
    targetPort: 8443
  type: ClusterIP

# Webhook service configuration sourced from the `config/webhook`
webhookService:
  ports:
  - port: 443
    protocol: TCP
    targetPort: 9443
  type: ClusterIP
```

- **Optional configurations should be disabled by default**:

The HelmChart plugin should not scaffold optional options enabled
when those are scaffolded as disabled by the default implementation
of `kustomize/v2` and consequently the `go/v4` plugin used by default. Example:

The dependency on Cert-Manager is disabled by default.

```yaml
From config/default/kusyomization.yaml
# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER'. 'WEBHOOK' components are required.
#- ../certmanager
```

Therefore, by default the values.ymal should be scaffold with:

```
# Cert-manager configuration
certmanager:
  enabled: false
```

- **Namespace Creation**:

For Golang projects, the namespace will always be created, and the project will be deployed within it.
For the Helm chart, we should:

```
# templates/manager/namespace.yaml
{{- if .Values.namespace.create }}
apiVersion: v1
kind: Namespace
metadata:
  name: {{ .Values.namespace.name }}
{{- end }}
```

#### To Sync the changes to the HelmCharts

For all implemementations, we need to check the resources which
are scaffold for each subCommand via the [kustomize](https://kubebuilder.io/plugins/kustomize-v2) plugin
and ensure that we will implement the subCommand of the HelmChart plugin
to the respective scaffolds as well.

#### When kubebuilder create api is executed

We can check in [pkg/plugins/common/kustomize/v2/scaffolds/api.go](https://github.com/kubernetes-sigs/kubebuilder/blob/c058fb95fe0ccd8d2a3147990251ca501df5eb26/pkg/plugins/common/kustomize/v2/scaffolds/api.go#L75-L84)
the boilerplates called and the scaffolds done. Therefore, the
`pkg/plugins/optional/helm-chart/v1alpha/scaffolds/api.go`
will also need to have the relevant code implementation to do the same
scaffolds under `dist/helm-charts/`.

#### When kubebuilder create webhook is executed

In this case, we should add the webhook section to the `values.yaml`
as `enabled:true` and ensure that HelmChart has the relevant templates
for webhooks.

We will need to use the utils helpers such as [AppendCodeIfNotExist](https://github.com/kubernetes-sigs/kubebuilder/blob/c058fb95fe0ccd8d2a3147990251ca501df5eb26/pkg/plugin/util/util.go#L101C6-L112)
to achieve this goal. Also, we will need to ensure that we do the equivalent
scaffolds as described above for the create api subCommand.

#### To Sync the Manifests Created with `controller-gen`

Users will need to call the subcommand `edit` passing the plugin to
ensure that the Helm chart is properly synced.

Therefore, the `PostScaffold` of this command should:

- **Run `make manifests`**: Generate the latest CRDs and other manifests.
- **Copy the files to Helm chart templates**:
  - Copy CRDs: `cp config/crd/bases/*.yaml helm-chart/example-project-crd/templates/crds/`
  - Copy RBAC manifests: `cp config/rbac/*.yaml helm-chart/example-project/templates/rbac/`
  - Copy webhook configurations: `cp config/webhook/*.yaml helm-chart/example-project/templates/webhook/`
  - Copy the manager manifest: `cp config/default/manager.yaml helm-chart/example-project/templates/manager/manager.ymal0`
- **Replace placeholders with Helm values**: Ensure that customized fields, such as the namespace, are properly replaced accordingly.
Example: Replace `name: system` with `{{ .Values.namespace.name }}`.

This ensures the Helm chart is always up-to-date with the latest
manifests generated by Kubebuilder, maintaining consistency with the
configured namespace and other customizable fields.

We will need to use the utils helpers such as [ReplaceInFile](https://github.com/kubernetes-sigs/kubebuilder/blob/c058fb95fe0ccd8d2a3147990251ca501df5eb26/pkg/plugin/util/util.go#L303-L323)
or [EnsureExistAndReplace](https://github.com/kubernetes-sigs/kubebuilder/blob/c058fb95fe0ccd8d2a3147990251ca501df5eb26/pkg/plugin/util/util.go#L276)
to achieve this goal.

### Risks and Mitigations

**Difficulty in Maintaining the Solution**

Maintaining the solution may prove challenging in the long term,
particularly if it does not gain community adoption and, consequently, collaboration.
To mitigate this risk, the proposal aims to introduce an optional alpha plugin or to
implement it through an alpha command. This approach provides us with greater flexibility
to make adjustments or, if necessary, to deprecate the feature without definitively
compromising support.

### Proof of Concept

In order to prove that would be possible we could
refer to the open source tool
[helmify](https://github.com/arttor/helmify).

## Drawbacks

**Inability to Handle Complex Kubebuilder Scenarios**

The proposed plugin may struggle to appropriately handle complex scenarios commonly encountered
in Kubebuilder projects, such as intricate webhook configurations. Kubebuilder’s scaffolded
projects can have sophisticated webhook setups, and translating these accurately into Helm
Charts may prove challenging. This could result in Helm Charts that are not fully reflective
of the original project’s functionality or configurations.

**Incomplete Generation of Valid and Deployable Helm Charts**

The proposed solution may not be capable of generating a fully valid and deployable Helm Chart
for all use cases supported by Kubebuilder. Given the diversity and complexity of potential
configurations within Kubebuilder projects, there is a risk that the generated Helm Charts
may require significant manual intervention to be functional. This drawback undermines the
goal of simplifying distribution via Helm Charts and could lead to frustration for users who
expect a seamless and automated process.

## Alternatives

**Via a new command (Alternative Option)**

By running the following command, the plugin will generate a helm chart from the specific kustomize directory and output it to the directory specified by the `--output` flag.

```shell
kubebuilder alpha generate-helm-chart --from=<path> --output=<path>
```

The main drawback of this option is that it does not adhere to the Kubebuilder ecosystem.
Additionally, we would not take advantage of Kubebuilder library features, such as avoiding
overwriting the `values.yaml`. It might also be harder to support and maintain since we would
not have the templates as we usually do.

Lastly, another con is that it would not allow us to scaffold projects with the plugin
enabled and in the future provide further configurations and customizations for this plugin.
These configurations would be tracked in the `PROJECT` file, allowing integration with other
projects, extensions, and the re-scaffolding of the HelmChart while preserving the inputs
provided by the user via plugins flags as it is done for example for
the [Deploy Image](https://book.kubebuilder.io/plugins/deploy-image-plugin-v1-alpha) plugin.