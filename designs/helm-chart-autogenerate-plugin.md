| Authors       | Creation Date | Status      | Extra |
|---------------|---------------|-------------|---|
| @name | date | Implementeble | - |

# Auto-generate Helm Chart Plugin
===================

The proposal is to add a new alpha command to auto-generate a helm chart for a given kustomize directory created by kubebuilder.

## Example

By running the following command, the plugin will generate a helm chart from the specific kustomize directory and output it to the directory specified by the `--output` flag.

```shell
kubebuilder alpha generate-helm-chart --from=<path> --output=<path>
```

## Open Questions [optional]

TBF

## Summary

TBF

## Motivation

Currently, kubebuilder only supports the distribution of the operator by the kustomize. However, many well-known operators contains the distribution of helm charts:

- [mongodb](https://artifacthub.io/packages/helm/mongodb-helm-charts/community-operator
)
- [cert-manager](https://cert-manager.io/v1.6-docs/installation/helm/#1-add-the-jetstack-helm-repository)
- [prometheus](https://bitnami.com/stack/prometheus-operator/helm)

The proposed command will auto-generate a helm chart from a kustomize directory created by kubebuilder with minimizing the manual effort. The generated helm chart will cover the frequently-used configurations.

Context:

- See the discussion [Is there a good solution/tool to generate a new helm chart from the manifests?](https://github.com/kubernetes-sigs/kubebuilder/discussions/3074)

### Goals

- Help developers to generate a helm chart from a kustomize directory.
- Generate a helm chart without human interactions once the kustomize directory updated.
- Reserve the ability to customize the helm chart by the developer.

### Non-Goals

- Generate the helm chart from a kustomize directory not created by kubebuilder.
- Generate the fine-grained configurations.

## Proposal

To reduce the user adaptation, the command
leverage the existing kustomize default directory and create the similar layout of the helm chart from it. The generated `values.yaml` is as follows:

```yaml
# global configurations
nameOverride: ""

fullnameOverride: ""

crd:
  create: true

## More info: https://kubernetes.io/docs/admin/authorization/rbac/
rbac:
  create: true
  serviceAccountName: ""


## Configure the manager
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

     ## The environment variables of the controller container.
     ## More info: https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/
  	env: []

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

     ## Configure the scheduling rules.
     ## More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
     affinity: {}

     ## More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector
     nodeSelector: {}

     ## More info: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration
     tolerations: {}

     ## Configure the rbac roles.
     serviceAccountName: ""

## Configure the webhook
webhook:
  enabled: true
  ## More info: https://kubernetes.io/docs/concepts/services-networking/service/
  service:
    type: ClusterIP
    ports:
    - port: 443
      protocol: TCP
      targetPort: 9443

metrics:
  enabled: true
  service:
    type: ClusterIP
    ports:
    - port: 8443
      protocol: TCP

## Configure the cert-manager
certmanager:
  enabled: true
  installCRDs: true
  ## Add the owner reference to the certificate.
  extraArgs:
  - --enable-certificate-owner-ref=true

prometheus:
  ## Enable the Prometheus Monitor
  enabled: false

## The domain name of Kubernetes cluster 
## More info: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/
clusterDomain: cluster.local
```

### User Stories

- As a developer, I want to be able to generate a helm chart from a kustomize directory so that I can distribute the helm chart to my users. Also, I want the 
generation to be as simple as possible without the need to write any additional duplicate files.

- As a user, I want the helm chart can cover all potential configurations when I deploy it on the Kubernetes cluster.

### Implementation Details/Notes/Constraints [optional]

TBF

### Risks and Mitigations

TBF

### Proof of Concept [optional]

Refer to the open source tool
[helmify](https://github.com/arttor/helmify).


## Drawbacks

TBF

## Alternatives

TBF
