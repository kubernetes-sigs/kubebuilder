| Authors       | Creation Date | Status      | Extra |
|---------------|---------------|-------------|---|
| @dashanji,@camilamacedo86,@LCaparelli... | Sep, 2023 | Implementeble | - |

# New Plugin to allow project distribution via helm charts

This proposal aims to introduce an optional mechanism that allows users to generate a Helm Chart from their Kubebuilder-scaffolded project. This will enable them to effectively package and distribute their solutions.

To achieve this goal, we are proposing a new native Kubebuilder [plugin](https://book.kubebuilder.io/plugins/plugins) (i.e., helm-chart/v1-alpha) which will provide the necessary scaffolds. The plugin will function similarly to the existing [Grafana Plugin](https://book.kubebuilder.io/plugins/grafana-v1-alpha), generating or regenerating HelmChart files using the init and edit sub-commands (i.e., `kubebuilder init|edit --plugins helm-chart/v1-alpha`). An alternative solution could be to implement an alpha command, similar to the [helper provided to upgrade projects](https://book.kubebuilder.io/reference/rescaffold).

## Example

**To enable the helm-chart generation when a project is initialized**

> kubebuilder init --plugins=`go/v4,helm-chart/v1-alpha`

**To enable the helm-chart generation after the project be scaffolded OR to update the helm-chart files**

> kubebuilder edit --plugins=`go/v4,helm-chart/v1-alpha`

#### Case 1: Initial Helm chart distribution

For users who have developed an Kubebuilder Project and wish to distribute it via a Helm chart for the first time:

By running `kubebuilder init|edit --plugins=helm-chart/v1alpha`, the scaffolds of Helm Chart to package the Project would be created in the project as follows:

```shell
$ tree
.
..
├── PROJECT
...
├── deploy
│   └── helm-chart // The helm chart with all values will be scaffold under a directory of the project
...
```

The initial setup will be straightforward, creating the Helm chart structure and populating it with the default values as determined by the project's configuration.

#### Case 2: Update the Helm chart with manifests changes

For users who want to update an existing Helm chart to reflect changes in the kustomize directory, such as CRDs, Webhooks, or RBAC roles:

The plugin will automates this process. When you run the plugin, it will generate the necessary manifests and synchronize them with the Helm chart. This ensures that all changes in your kubebuilder project files, such as CRDs, RBAC roles, and webhooks, are accurately reflected in the Helm chart.

#### Case 3: Updating the Helm Chart with Customizations

For users who have customized their Helm chart (e.g., added new configurations in `values.yaml`) and want to update it while preserving these customizations:

Just like the other plugins in Kubebuilder, the Helm Chart plugin will not re-write the `values.yaml` after the initial generation. This means that any customizations made by the user will be preserved.

## Open Questions [optional]

- How to effectively manage the lifecycle of CRDs within the generated Helm chart to support upgrades and changes to CRDs while minimizing potential impacts on existing resources and user deployment processes?
- How to deal with CertManager and Prometheus? Should we add them as dependencies to the Helm Chart?

## Summary

The Helm Chart created by the project should accurately reflect the default values utilized in Kubebuilder. Furthermore, any newly generated CRDs or Webhooks should be seamlessly integrated into the Helm Chart. To accomplish this, the new plugin needs to incorporate all the subCommands available to the [kustomize plugin](https://book.kubebuilder.io/plugins/kustomize-v2). Additionally, the edit feature, similar to what is available in the optional [Grafana plugin](https://book.kubebuilder.io/plugins/grafana-v1-alpha), should be integrated to enable modifications in projects that were previously created.

Thus, the init subCommand should take charge of creating the Helm Chart directory and scaffolding the base of the Helm Chart, based on the default scaffold provided by Kustomize. This means that the same default values, scaffolded to the manager and proxy, along with the options to enable metrics via Prometheus, cert-manager, and webhooks, should be disabled (commented out) by default, as seen in the [config/default/kustomization.yaml](https://github.com/kubernetes-sigs/kubebuilder/blob/f4744670e6fc8ed29f87161d39a8f2f3838c27f4/testdata/project-v4/config/default/kustomization.yaml#L21-L27) file.

Subsequently, if a user decides to scaffold webhooks, we should uncomment the relevant sections. Please refer to PRs https://github.com/kubernetes-sigs/kubebuilder/pull/3627 and https://github.com/kubernetes-sigs/kubebuilder/pull/3629 for examples of this implementation.

Lastly, when new API(s) (CRDs) are introduced, they must be added to the Helm Chart as well. As a result, the Makefile targets `make generate` and `make manifest` should be modified when the proposed plugin is in use to ensure that CRDs are accurately copied and pasted into the Helm Chart's respective directory.

To allow users to customize their Helm Charts, we should avoid overwriting files within the Helm Chart, with the exception of CRDs/webhooks, as changes made to these must be mirrored in the Helm Chart. However, what is scaffolded by the init/edit should remain primarily unchanged. We would only uncomment the options as described above.

## Motivation

Kubebuilder users currently face the challenges of lacking an officially supported distribution mechanism for Helm Chart. They seek to:

- Harness the power of Helm Chart as a package manager for the Project, enabling seamless adaptation to diverse deployment environments.
- Take advantage of Helm's dependency management capabilities to simplify the installation process of Project dependencies, such as cert-manager.
- Seamlessly integrate with Helm's ecosystem, including FluxCD, to efficiently manage the Project.

However, Kubebuilder does not offer any utilities to facilitate the distribution of projects currently. One could clone the entire project and utilize the Makefile targets to apply the solution on clusters. Consequently, this proposal aims to introduce a method that allows Kubebuilder users to easily distribute their projects through Helm Charts, a strategy that many similar many well-known projects contains the distribution of helm charts:

- [mongodb](https://artifacthub.io/packages/helm/mongodb-helm-charts/community-operator
)
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
- Support the deprecated plugins. This option should be supported from go/v4 and kustomize/v2
- Introduce support for Helm in addition to Kustomize, or replace Kustomize with Helm entirely, similar to the approach taken by Operator-SDK, thereby allowing users to utilize Helm Charts to build their Project.
- Attend standard practices that deviate from Helm Chart layout, definition, or conventions to workaround its limitations.
- Provide values and options to allow users to perform customizations on the Helm Chart of features that are not used by default in the Project layout.

## Proposal

The new optional plugin will create a new directory, `deploy/helm-chart`, where the scaffolding of the project should be constructed as follows:
//TODO: We need a better definition 
// We must create a Helm Chart with the default Kubebuilder scaffold to add here
```shell
❯ tree deploy/helm-chart
helm-chart
    ├── Chart.yaml
    ├── templates
    │   ├── <CRDs YAML files generate under config/crds/>
    │   ├── deployment.yaml 
    │   ├── _helpers.tpl
    │   ├── role_binding.yaml
    │   ├── role.yaml 
    │   ├── <all under config/rbac less the edit and view roles which are helpers for admins>
    └── values.yaml
```

To reduce the user adaptation, the command
leverage the existing kustomize default directory and create the similar layout of the helm chart from it. The generated `values.yaml` is as follows:

```yaml
# global configurations
nameOverride: ""

fullnameOverride: ""

## CRD configuration under the `config/crd` directory
crd:
  ## Whether the `resources` field contains `crd` in the `config/default/kustomization.yaml` file
  create: false

## RBAC configuration under the `config/rbac` directory
rbac:
  ## Whether the `resources` field contains `rbac` in the `config/default/kustomization.yaml` file
  create: true
  serviceAccountName: "controller-manager"


## Manager configuration referent to `config/manager/manager.yaml`
manager:
  ## The Security Context of the manager Pod.
  ## More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod
  podSecurityContext:
    runAsNonRoot: true

  ## Configure the kubeRbacProxy container.
  kubeRbacProxy:
    image:
     registry: gcr.io
     repository: "kubebuilder/kube-rbac-proxy"
     tag: v0.13.0
     pullPolicy: IfNotPresent

  ## Configure the resources.
  ## More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ 
  resources:
    limits:
      cpu: 500m
      memory: 128Mi
    requests:
      cpu: 5m
      memory: 64Mi

  ## Configure the controller container.
  controller:
    image:
      registry: docker.io
      repository: ""
      tag: latest
      pullPolicy: IfNotPresent

    ## Configure extra options for liveness probe.
    ## More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#configure-probes
    livenessProbe:
      enabled: true
      httpGet:
        path: /healthz
        port: 8081
      initialDelaySeconds: 15
      periodSeconds: 20

    readinessProbe:
      enabled: true
      httpGet:
        path: /readyz
        port: 8081
      initialDelaySeconds: 5
      periodSeconds: 10

    ## Configure the resources.
    ## More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ 
    resources:
      limits:
        cpu: 500m
        memory: 128Mi
      requests:
        cpu: 10m
        memory: 64Mi

     ## Configure the rbac roles.
     serviceAccountName: ""

## Webhook configuration under the `config/webhook` directory
webhook:
  ## Whether the `resources` field contains `webhook` in the `config/default/kustomization.yaml` file
  enabled: true
  ## More info: https://kubernetes.io/docs/concepts/services-networking/service/
  service:
    ports:
    - name: webhook-server
      protocol: TCP
      containerPort: 9443

metrics:
  enabled: true
  service:
    type: ClusterIP
    ports:
    - port: 8443
      protocol: TCP

## Configure the cert-manager dependency.
certmanager:
  enabled: true
  installCRDs: true
  ## Add the owner reference to the certificate.
  extraArgs:
  - --enable-certificate-owner-ref=true

## Prometheus configuration under the `config/prometheus` directory
prometheus:
  ## Whether the `resources` field contains `prometheus` in the `config/default/kustomization.yaml` file
  enabled: false

## The domain name of Kubernetes cluster 
## More info: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/
clusterDomain: cluster.local
```

### User Stories

- As a developer, I want to be able to generate a helm chart from a kustomize directory so that I can distribute the helm chart to my users. Also, I want the 
generation to be as simple as possible without the need to write any additional duplicate files.

- As a user, I want the helm chart can cover all potential configurations when I deploy it on the Kubernetes cluster.

- As a platform engineer, I want to be able to manage different versions and configurations of a project across multiple clusters and environments based on the same distribution artifact (Helm Chart), with versioning and dependency locking for supply chain security.

### Implementation Details/Notes/Constraints [optional]

TBF

### Risks and Mitigations

**Difficulty in Maintaining the Solution**

Maintaining the solution may prove challenging in the long term, particularly if it does not gain community adoption and, consequently, collaboration. To mitigate this risk, the proposal aims to introduce an optional alpha plugin or to implement it through an alpha command. This approach provides us with greater flexibility to make adjustments or, if necessary, to deprecate the feature without definitively compromising support.

Additionally, it is crucial to cover the solution with end-to-end tests to ensure that, despite any changes, the Helm Chart remains deployable and continues to function well, as is the practice with our other plugins and solutions.

### Proof of Concept [optional]

Refer to the open source tool
[helmify](https://github.com/arttor/helmify).


## Drawbacks

**Generation of Deployable Helm Chart:**

There might be challenges in generating a deployable Helm Chart without human intervention, similar to the current process for Kubebuilder projects.

## Alternatives

**Via a new command (Alternative Option)**
By running the following command, the plugin will generate a helm chart from the specific kustomize directory and output it to the directory specified by the `--output` flag.

```shell
kubebuilder alpha generate-helm-chart --from=<path> --output=<path>
```

pros: Always generates the Helm Chart from the project, ensuring that the output is synchronized.

cons: Does not preserve customizations made by the users.
