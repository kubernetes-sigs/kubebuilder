{% panel style="danger", title="STAGING" %}
Staging Environment - Not Official Documentation!

This book contains APIs, libraries and tools that are proposals only and have not been ratified!
{% endpanel %}

# Why Kubernetes APIs

Kubenernetes APIs allow users to declaratively specify the desired state of a Kubernetes cluster in the
form of *yaml* or *json* text, and integrate with standard tools such as `kubectl` and `kustomize`.  The
declarative API approach provides a number of advantage as outlined below.

Developers may build and publish new solutions as Kubernetes APIs which may be dynamically installed
into running clusters by admins. Benefits to building a solution as a Kubernetes API
instead of a generic REST service or RPC service include:

* Facilitating self-healing by watching the state of the system and responding to cluster events
  without user intervention.
* Integrating with the Kubernetes ecosystem of tools such as `kubectl` and `kustomize` and looking
  the same as core Kubernetes APIs to tools.
* Integrating with Kubernetes Authz and Authn, allowing policies such as RBAC to user control access.
* Natively supporting multiple versions of the same API, thereby allowing developers to evolve APIs
  by changing field names and defaults without breaking backwards compatibility.
* Integrating with Kubernetes cluster components such as autoscalers.
* Providing an experience consistent with the core APIs.
