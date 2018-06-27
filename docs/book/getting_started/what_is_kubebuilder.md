# What is Kubebuilder

Kubebuilder is an SDK for rapidly building and publishing Kubernetes APIs in Go.
It builds on top of the canonical techniques used to build the core Kubernetes APIs
to provide simple abstractions that reduce boilerplate and toil.

Similar to web development frameworks such as *Ruby on Rails* and *SpringBoot*,
Kubebuilder increases velocity and reduces the complexity managed by
developers.

Included in Kubebuilder:

* Initializing projects with a base structure including
  * Go package dependencies at canonical versions.
  * main program entry point
  * Makefile for formatting, generating, testing and building go
  * Dockerfile for building container images
* Scaffolding APIs with
  * Resource (Model) definition
  * Controller implementation
  * Integration tests for Resource and Controller
  * CRD definition
* Simple abstractions for implementing APIs
  * Controllers
  * Resource Schema Validation
  * Validating Webhooks
* Artifacts for publishing APIs for installation into clusters
  * Namespace
  * CRDs
  * RBAC Roles and RoleBindings
  * Controller StatefulSet + Service
* API reference documentation with examples

Kubebuilder is developed on top of the controller-runtime and controller-tools libraries.
