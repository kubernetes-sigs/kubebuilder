{% panel style="danger", title="STAGING" %}
Staging Environment - Not Official Documentation!

This book contains APIs, libraries and tools that are proposals only and have not been ratified!
{% endpanel %}


# What is Kubebuilder

Kubebuilder is an SDK for rapidly building and publishing Kubernetes APIs in the go language using the
canonical techniques that power Kubernetes.

In the spirit of modern web development frameworks such as *Ruby on Rails* and *SpringBoot*,
Kubebuilder provides a set of tools and libraries intended to simplify API development, and to
delight and empower developers.

Kubebuilder accomplishes this through providing:

* Tools to initialize *go* projects with the canonical set of libraries and their transitive dependencies
  necessary to build Kubernetes APIs.
* Tools to bootstrap new API definitions through writing scaffolding code, tests, and documentation.
* Simple, clean, high level libraries for invoking the Kubernetes APIs from go.
* Seamless integration of standard production logging and monitoring into API implementations.
* Tools to build and publish APIs as cluster addons or installable yaml declarations.
* Tools to build and publish API reference documentation with examples.
* Step by step guidance on how to use kubebuilder to develop your APIs.
