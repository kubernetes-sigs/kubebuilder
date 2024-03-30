# Reference

  - [Generating CRDs](generating-crd.md)
  - [Using Finalizers](using-finalizers.md)
    Finalizers are a mechanism to
    execute any custom logic related to a resource before it gets deleted from
    Kubernetes cluster.
  - [Watching Resources](watching-resources.md)
    Watch resources in the Kubernetes cluster to be informed and take actions on changes.
      - [Resources Managed by the Operator](watching-resources/operator-managed.md)
      - [Externally Managed Resources](watching-resources/externally-managed.md)
        Controller Runtime provides the ability to watch additional resources relevant to the controlled ones.
  - [Kind cluster](kind.md)
  - [What's a webhook?](webhook-overview.md)
    Webhooks are HTTP callbacks, there are 3
    types of webhooks in k8s: 1) admission webhook 2) CRD conversion webhook 3)
    authorization webhook
    - [Admission webhook](admission-webhook.md)
      Admission webhooks are HTTP
      callbacks for mutating or validating resources before the API server admit
      them.
  - [Markers for Config/Code Generation](markers.md)

      - [CRD Generation](markers/crd.md)
      - [CRD Validation](markers/crd-validation.md)
      - [Webhook](markers/webhook.md)
      - [Object/DeepCopy](markers/object.md)
      - [RBAC](markers/rbac.md)

  - [controller-gen CLI](controller-gen.md)
  - [completion](completion.md)
  - [Artifacts](artifacts.md)
  - [Platform Support](platform.md)

  - [Sub-Module Layouts](submodule-layouts.md)
  - [Using an external Type / API](using_an_external_type.md)

  - [Metrics](metrics.md)
      - [Reference](metrics-reference.md)

  - [Makefile Helpers](makefile-helpers.md)
  - [CLI plugins](../plugins/plugins.md)
