# Why Kubernetes APIs

Kubernetes APIs provide consistent and well defined endpoints for
objects adhering to a consistent and rich structure.

This approach has fostered a rich ecosystem of tools and libraries for working
with Kubernetes APIs.

Users work with the APIs through declaring objects as *yaml* or *json* config, and using
common tooling to manage the objects.

Building services as Kubernetes APIs provides many advantages to plain old REST, including:

* Hosted API endpoints, storage, and validation.
* Rich tooling and clis such as `kubectl` and `kustomize`.
* Support for Authn and granular Authz.
* Support for API evolution through API versioning and conversion.
* Facilitation of adaptive / self-healing APIs that continuously respond to changes
  in the system state without user intervention.
* Kubernetes as a hosting environment

Developers may build and publish their own Kubernetes APIs for installation into
running Kubernetes clusters.
