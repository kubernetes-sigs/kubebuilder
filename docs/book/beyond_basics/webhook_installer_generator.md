# Webhook Configuration Installation

Installing webhook configurations requires higher privileges which manager's service account might not have.
In that case, a separate process with higher privileges would like to install the webhook configurations.

There are 2 options to install webhook configurations into a cluster:

- Configure the Webhook Server to automatically install the webhook configurations when it starts.
- Use the webhook manifests generator to generate webhook configurations.
Then install the webhook configurations and deploy the webhook servers.

## Webhook Installer

Webhook installer is a feature provided by the [controller-runtime](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/webhook) library.
For a [Webhook Server](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/webhook#Server),
you can choose to enable the webhook installer.
Depending on your [ServerOptions](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/webhook#ServerOptions),
the installer may install [mutatingWebhookConfigurations](https://github.com/kubernetes/api/blob/9fcf73cc980bd64f38a4f721a7371b0ebb72e1ff/admissionregistration/v1beta1/types.go#L113-L124),
[validatingWebhookConfigurations](https://github.com/kubernetes/api/blob/9fcf73cc980bd64f38a4f721a7371b0ebb72e1ff/admissionregistration/v1beta1/types.go#L83-L94)
and service; it may also update the secret if needed.

To make the webhook installer work correctly, please ensure the manager's service account has
the right permissions. For example. it may need permissions:
- create and update MutatingWebhookConfigurations and ValidatingWebhookConfigurations
- create and update the Service in the same namespace of the manager
- update the Secret in the same namespace of the manager

The service fronts the webhook server.
So please ensure the service's selectors select your webhook server pods.

The secret contains
- the serving certificate and its corresponding private key
- the signing CA certificate and its corresponding private key

Webhook installer can be very helpful for
- faster iteration during development
- easier deployment in production if policy allows

Webhook installer is on by default. Set
[`DisableWebhookConfigInstaller`](https://github.com/kubernetes-sigs/controller-runtime/blob/a8ea2056444a5d74d7408e4b8798cbe63b14066b/pkg/webhook/server.go#L57-L60)
to true to turn it off.

## Webhook Manifests Generator

From cluster administrators perspective, they may want to have the webhook configurations
installation to be a separate process from running the webhook server,
since permissions to create and update webhook configurations are considered as high privileges.

Similar to other generators in [controller-tools](https://github.com/kubernetes-sigs/controller-tools),
webhook manifest generator is annotation-driven.

How the webhook manifest generator works
1) It parses the annotations to configure
[Webhook](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/webhook/admission#Webhook).
1) It parses the annotations to configure
[ServerOptions](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/webhook#ServerOptions).
1) It uses the library in controller-runtime, which is the same machinery as
the installer, to generate the webhook configuration manifests.

#### Comment Group

A comment group represents a sequence of comments with no empty lines between.
Comment group is a concept that is important for writing and parsing annotations correctly.

For example, the following comments are in one comment group

```
// +kubebuilder:webhook:groups=apps
// +kubebuilder:webhook:resources=deployments
```

The following comments are in 2 comments groups

```
// +kubebuilder:webhook:groups=apps

// +kubebuilder:webhook:resources=deployments
```

#### Annotations

Each comment line that starts with `+kubebuilder:webhook:` will be processed to extract annotations.

The annotations can be grouped in 2 categories based on what struct they configure.

The _first_ category is for each individual webhook.
They are used to set the fields in [`Webhook`](https://github.com/kubernetes-sigs/controller-runtime/blob/a8ea2056444a5d74d7408e4b8798cbe63b14066b/pkg/webhook/admission/webhook.go#L55-L80) struct.
The annotations for the same webhook are allowed to span across multiple lines as long as they are prefixed with
`+kubebuilder:webhook:` and in the __same__ comment group.
It is suggested to put this category of annotations in the same file as its corresponding webhook.

For example, the following is for one single webhook

```
// +kubebuilder:webhook:groups=apps,versions=v1,resources=deployments,verbs=create,update
// +kubebuilder:webhook:name=mutating-create-update-deployment.testproject.org
// +kubebuilder:webhook:path=/mutating-create-update-deployment
// +kubebuilder:webhook:type=mutating,failure-policy=fail
```

`groups`, `versions` and `resources` have the same semantic as the ones used for generating RBAC manifests.
They can reference a core type or a CRD type.

`verbs` are used to set [`Operations`](https://github.com/kubernetes/api/blob/9fcf73cc980bd64f38a4f721a7371b0ebb72e1ff/admissionregistration/v1beta1/types.go#L234-L243).
It supports `create`, `update`, `delete`, `connect` and `*` case-insensitively.

`name` is the name of the webhook and
is used to set [`Name`](https://github.com/kubernetes/api/blob/9fcf73cc980bd64f38a4f721a7371b0ebb72e1ff/admissionregistration/v1beta1/types.go#L146).
`path` is the endpoint that this webhook serves and
is used to set [`Path`](https://github.com/kubernetes-sigs/controller-runtime/blob/a8ea2056444a5d74d7408e4b8798cbe63b14066b/pkg/webhook/admission/webhook.go#L61-L62).
Both `name` and `path` do NOT allow `,` and `;`.

`type` indicates the webhook type which can be either `mutating` or `validating`.

`failure-policy` is used to set [`FailurePolicy`](https://github.com/kubernetes/api/blob/9fcf73cc980bd64f38a4f721a7371b0ebb72e1ff/admissionregistration/v1beta1/types.go#L54-L61).
It supports `fail` and `ignore` case-insensitively.

The _second_ category is for the webhook server.
All of them are used to configuration [`ServerOptions`](https://github.com/kubernetes-sigs/controller-runtime/blob/a8ea2056444a5d74d7408e4b8798cbe63b14066b/pkg/webhook/server.go#L39-L98) struct.
Each annotation should only be used once.
They don't have to be in the same comment group.
It is suggested to put this category of annotations in the same file as the webhook server.

The following is an example using webhook server annotations.

```
// +kubebuilder:webhook:port=7890,cert-dir=/path/to/cert
// +kubebuilder:webhook:service=test-system:webhook-service,selector=app:webhook-server
// +kubebuilder:webhook:secret=test-system:webhook-secret
// +kubebuilder:webhook:mutating-webhook-config-name=test-mutating-webhook-cfg
// +kubebuilder:webhook:validating-webhook-config-name=test-validating-webhook-cfg
```

`port` is the port that the webhook server serves. It is used to set [`Port`](https://github.com/kubernetes-sigs/controller-runtime/blob/a8ea2056444a5d74d7408e4b8798cbe63b14066b/pkg/webhook/server.go#L43).

`service` should be formatted as `<namespace>:<name>`. It is used to set
[the name and namespace of the service](https://github.com/kubernetes-sigs/controller-runtime/blob/a8ea2056444a5d74d7408e4b8798cbe63b14066b/pkg/webhook/server.go#L102-L105).

`selector` should be formatted as `key1:value1;key2:value2` and it has 2 usages:
- use as [selectors in the service](https://github.com/kubernetes-sigs/controller-runtime/blob/a8ea2056444a5d74d7408e4b8798cbe63b14066b/pkg/webhook/server.go#L106-L108)
- use as additional labels that will be added to field `.spec.template.metadata.labels` of
the StatefulSet through [`kustomize`](https://github.com/kubernetes-sigs/kustomize).

`host` is used to set [`Host`](https://github.com/kubernetes-sigs/controller-runtime/blob/a8ea2056444a5d74d7408e4b8798cbe63b14066b/pkg/webhook/server.go#L87-L91).

`secret` should be formatted as `<namespace>:<name>`. It is used to set [`Secret`](https://github.com/kubernetes-sigs/controller-runtime/blob/a8ea2056444a5d74d7408e4b8798cbe63b14066b/pkg/webhook/server.go#L73-L77)

`mutating-webhook-config-name` is used to set [`MutatingWebhookConfigName`](https://github.com/kubernetes-sigs/controller-runtime/blob/a8ea2056444a5d74d7408e4b8798cbe63b14066b/pkg/webhook/server.go#L68-L69).

`validating-webhook-config-name` is used to set [`ValidatingWebhookConfigName`](https://github.com/kubernetes-sigs/controller-runtime/blob/a8ea2056444a5d74d7408e4b8798cbe63b14066b/pkg/webhook/server.go#L70-L71).
