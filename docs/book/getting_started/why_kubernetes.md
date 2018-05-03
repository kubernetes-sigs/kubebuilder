{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}


# Why Kubernetes APIs

Kubenernetes APIs allow users to specify the desired state of a Kubernetes cluster in an
object by writing declarative *yaml* or *json* config for a Resource.  This
approach provides a number of advantages including:

* Facilitating self-healing APIs that continuously watch the state of the system.
* Leveraging tools that work with any Kubernetes config such as `kubectl` and `kustomize`.
* Integrating with Kubernetes Authz and Authn.
* API versioning.
* Providing a native Kubernetes experience.

Developers can build and publish their own Kubernetes APIs which may be installed
into running clusters by cluster admins.
