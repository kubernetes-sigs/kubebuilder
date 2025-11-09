# Using External Resources

In some cases, your project may need to work with resources that aren't defined by your own APIs.
These external resources fall into two main categories:

- **Core Types**: API types defined by Kubernetes itself, such as `Pods`, `Services`, and `Deployments`.
- **External Types**: API types defined in other projects, such as CRDs defined by another solution.

## Managing External Types

### Creating a Controller for External Types

To create a controller for an external type without scaffolding a resource,
use the `create api` command with the `--resource=false` option and specify the path to the
external API type using the `--external-api-path` and `--external-api-domain` flag options.
This generates a controller for types defined outside your project,
such as CRDs managed by other Operators.

The command looks like this:

```shell
kubebuilder create api --group <theirgroup> --version <theirversion> --kind <theirKind> --controller --resource=false --external-api-path=<their Golang path import> --external-api-domain=<theirdomain>
```

- `--external-api-path`: Provide the Go import path where the external types are defined.
- `--external-api-domain`:  Provide the domain for the external types. This value will be used to generate RBAC permissions and create the QualifiedGroup, such as - `apiGroups: <group>.<domain>`

For example, if you're managing Certificates from Cert Manager:

```shell
kubebuilder create api --group certmanager --version v1 --kind Certificate --controller=true --resource=false --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 --external-api-domain=io
```

<aside class="note">
<h1>Pinning External API Versions</h1>

You can pin a specific version of the external API dependency using the `--external-api-module` flag:

```shell
kubebuilder create api --group certmanager --version v1 --kind Certificate \
  --controller=true --resource=false \
  --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 \
  --external-api-domain=io \
  --external-api-module=github.com/cert-manager/cert-manager@v1.18.2
```

The flag accepts the module path with optional version (e.g., `github.com/cert-manager/cert-manager@v1.18.2`).
The module is stored in the PROJECT file and added to `go.mod` using `go get`,
which cleanly adds it as a direct dependency without polluting go.mod with unnecessary indirect dependencies.

</aside>

See the RBAC [markers][markers-rbac] generated for this:

```go
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates/finalizers,verbs=update
```

Also, the RBAC role:

```ymal
- apiGroups:
  - cert-manager.io
  resources:
  - certificates
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - cert-manager.io
  resources:
  - certificates/finalizers
  verbs:
  - update
- apiGroups:
  - cert-manager.io
  resources:
  - certificates/status
  verbs:
  - get
  - patch
  - update
```

This scaffolds a controller for the external type but skips creating new resource
definitions since the type is defined in an external project.

### Creating a Webhook to Manage an External Type

You can create webhooks for external types by providing the external API path, domain, and optionally the module:

```shell
kubebuilder create webhook --group certmanager --version v1 --kind Issuer \
  --defaulting --programmatic-validation \
  --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 \
  --external-api-domain=cert-manager.io
```

You can also pin the version using the `--external-api-module` flag:

```shell
kubebuilder create webhook --group certmanager --version v1 --kind Issuer \
  --defaulting --programmatic-validation \
  --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 \
  --external-api-domain=cert-manager.io \
  --external-api-module=github.com/cert-manager/cert-manager@v1.18.2
```

## Managing Core Types

Core Kubernetes API types, such as `Pods`, `Services`, and `Deployments`, are predefined by Kubernetes.
To create a controller for these core types without scaffolding the resource,
use the Kubernetes group name described in the following
table and specify the version and kind.

| Group                    | K8s API Group            |
|---------------------------|------------------------------------|
| admission                 | k8s.io/admission                  |
| admissionregistration      | k8s.io/admissionregistration      |
| apps                      | apps                              |
| auditregistration          | k8s.io/auditregistration          |
| apiextensions              | k8s.io/apiextensions              |
| authentication             | k8s.io/authentication             |
| authorization              | k8s.io/authorization              |
| autoscaling                | autoscaling                       |
| batch                     | batch                             |
| certificates               | k8s.io/certificates               |
| coordination               | k8s.io/coordination               |
| core                      | core                              |
| events                    | k8s.io/events                     |
| extensions                | extensions                        |
| imagepolicy               | k8s.io/imagepolicy                |
| networking                | k8s.io/networking                 |
| node                      | k8s.io/node                       |
| metrics                   | k8s.io/metrics                    |
| policy                    | policy                            |
| rbac.authorization        | k8s.io/rbac.authorization         |
| scheduling                | k8s.io/scheduling                 |
| setting                   | k8s.io/setting                    |
| storage                   | k8s.io/storage                    |

The command to create a controller to manage `Pods` looks like this:

```shell
kubebuilder create api --group core --version v1 --kind Pod --controller=true --resource=false
```

For instance, to create a controller to manage Deployment the command would be like:

```sh
create api --group apps --version v1 --kind Deployment --controller=true --resource=false
```

See the RBAC [markers][markers-rbac] generated for this:

```go
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update
```

Also, the RBAC for the above [markers][markers-rbac]:

```yaml
- apiGroups:
  - apps
  resources:
  - deployments
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - apps
  resources:
  - deployments/finalizers
  verbs:
  - update
- apiGroups:
  - apps
  resources:
  - deployments/status
  verbs:
  - get
  - patch
  - update
```

This scaffolds a controller for the Core type `corev1.Pod` but skips creating new resource
definitions since the type is already defined in the Kubernetes API.

### Creating a Webhook to Manage a Core Type

You will run the command with the Core Type data, just as you would for controllers.
See an example:

```go
kubebuilder create webhook --group core --version v1 --kind Pod --programmatic-validation
```
[markers-rbac]: ./markers/rbac.md