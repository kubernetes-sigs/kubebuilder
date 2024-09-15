
# Using External Resources

In some cases, your project may need to work with resources that aren't defined by your own APIs. These external resources fall into two main categories:

- **Core Types**: API types defined by Kubernetes itself, such as `Pods`, `Services`, and `Deployments`.
- **External Types**: API types defined in other projects, such as CRDs defined by another solution.

## Managing External Types

### Creating a Controller for External Types

To create a controller for an external type without scaffolding a resource, use the `create api` command with the `--resource=false` option and specify the path to the external API type using the `--external-api-path` option. This generates a controller for types defined outside your project, such as CRDs managed by other operators.

The command looks like this:

```shell
kubebuilder create api --group <theirgroup> --version v1alpha1 --kind <ExternalTypeKind> --controller --resource=false --external-api-path=<Golang Import Path>
```

- `--external-api-path`: Provide the Go import path where the external types are defined.

For example, if you're managing Certificates from Cert Manager:

```shell
kubebuilder create api --group certmanager --version v1 --kind Certificate --controller=true --resource=false --make=false --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1
```

This scaffolds a controller for the external type but skips creating new resource definitions since the type is defined in an external project.

### Creating a Webhook to Manage an External Type

<aside>
<H1> Support </H1>

Webhook support for external types is not currently automated by the tool. However, you can still use the tool to scaffold the webhook setup and make manual adjustments as needed. For guidance, you can follow a similar approach as described in the section [Webhooks for Core Types][webhook-for-core-types].

</aside>

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

This scaffolds a controller for the Core type `corev1.Pod` but skips creating new resource
definitions since the type is already defined in the Kubernetes API.


<aside class="note">
<h1>Scheme Registry</h1>

The CLI will not automatically register or add schemes for Core Types because these types are already included by default in Kubernetes. Therefore, you don't need to manually register the scheme for core types, as they are inherently supported.

For External Types, the schemes will be added during scaffolding when you create a controller for an external type. The import path provided during scaffolding will be used to register the external type's schema in your project’s `main.go` file and in the `suite_test.go` file.

</aside>

### Creating a Webhook to Manage a Core Type

<aside>
<H1> Support </H1>

Webhook support for Core Types is not currently automated by the tool. However, you can still use the tool to scaffold the webhook setup and make manual adjustments as needed. For guidance, you can follow [Webhooks for Core Types][webhook-for-core-types].

</aside>

[webhook-for-core-types]: ./webhook-for-core-types.md
