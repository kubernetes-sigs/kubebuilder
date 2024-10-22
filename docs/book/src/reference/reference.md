# Reference

  - [Generating CRDs](generating-crd.md)
  - [Using Finalizers](using-finalizers.md)
    Finalizers are a mechanism to
    execute any custom logic related to a resource before it gets deleted from
    Kubernetes cluster.
  - [Watching Resources](watching-resources.md)
    Watch resources in the Kubernetes cluster to be informed and take actions on changes.
      - [Watching Secondary Resources that are `Owned` ](watching-resources/secondary-owned-resources.md)
      - [Watching Secondary Resources that are NOT `Owned`](watching-resources/secondary-resources-not-owned)
      - [Using Predicates to Refine Watches](watching-resources/predicates-with-watch.md)
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
      - [Scaffold](markers/scaffold.md)

  - [Monitoring with Pprof](pprof-tutorial.md)
  - [controller-gen CLI](controller-gen.md)
  - [completion](completion.md)
  - [Artifacts](artifacts.md)
  - [Platform Support](platform.md)

  - [Sub-Module Layouts](submodule-layouts.md)
  - [Using an external Resource / API](using_an_external_resource.md)

  - [Metrics](metrics.md)
      - [Reference](metrics-reference.md)

  - [CLI plugins](../plugins/plugins.md)
