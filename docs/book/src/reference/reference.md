# Reference

  - [Generating CRDs](generating-crd.md)
  - [Using Finalizers](using-finalizers.md)
    Finalizers are a mechanism to
    execute any custom logic related to a resource before it gets deleted from
    Kubernetes cluster.
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
  - [Artifacts](artifacts.md)
  - [Writing controller tests](writing-tests.md)
  - [Metrics](metrics.md)