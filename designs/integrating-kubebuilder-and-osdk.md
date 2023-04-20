| Authors       | Creation Date | Status      | Extra |
|---------------|---------------|-------------|-------|
| @joelanford |  Sep 6, 2019  | implemented | -     |

Integrating Kubebuilder and Operator SDK
========================================

## Goal

To unite Kubebuilder and Operator SDK around Kubebuilder’s project scaffolding, to move Operator SDK’s Go operator features upstream, where appropriate, and to join forces on maintaining Kubebuilder so that both Kubebuilder and Operator SDK support the same project structure and command line interface for Go-based operators.

## Background

Kubebuilder and [Operator SDK][operator-sdk] are similar projects meant to simplify the process of building a Kubernetes operator (or controller). Both projects make extensive use of the upstream controller-runtime and controller-tools projects, and therefore scaffold similar Go source files and package structures.

## Motivation

The Operator SDK and Kubebuilder contributors collaborate on improvements to their shared upstream dependencies, but there is significant overlap between Operator SDK and Kubebuilder related to scaffolding Go operators. Both projects have commands to initialize a new project and add boilerplate implementations of new APIs and controllers.

The motivation for integrating Kubebuilder and Operator SDK is that rather than duplicating work related to project scaffolding of Go operators, the projects could work together on one implementation, which would speed up progress and likely result in a more general solution.

## Integration Plan

The Kubebuilder and Operator SDK contributors created a [GitHub project][kb-osdk-github-project] to track the work necessary to align the projects. There are three main themes.

### Upstream code from Operator SDK

The Operator SDK project contains various features that can be used by Go operator developers regardless of whether the project is based on Kubebuilder or Operator SDK. These features will be upstreamed into `kubebuilder`, `controller-runtime`, and `controller-tools`, where appropriate. These include:
* a `DynamicRESTMapper` that enables an operator to dynamically and automatically discover new CRDs added to the cluster after the operator has started
* a `GenerationChangedPredicate` that can trigger reconciliation events when a resource's `metadata.generation` field has changed.
* flags and helpers that can be used to provide more fine-grained configuration when constructing the default `zap`-based logger.

The Operator SDK contributors plan to begin conducting all development of Go operator related code in upstream Kubebuilder (and related projects) and to spend more time helping the Kubebuilder contributors maintain these projects.

### Prototypes

To make Kubebuilder more extensible, the community has been discussing a proposal to add extension points to Kubebuilder to support different operator patterns. One example of an operator pattern is the [addon pattern][addon-pattern-pr] that uses an existing library to instantiate an opinionated API and controller.

More broadly, the idea is to add support for executable plugin-based extensions that can modify Kubebuilder’s base scaffolding before files are written to disk so that the project (e.g. Go code, kustomize templates, the project Makefile and Dockerfile) can have customized content provided by an extension.

### Documentation

Operator SDK and Kubebuilder currently maintain separate documentation even though a significant chunk of it overlaps. By combining efforts, the SDK contributors will migrate and integrate their Go-based operator documentation upstream into the Kubebuilder documentation and join the Kubebuilder contributors in keeping it up-to-date.

[operator-sdk]: https://github.com/operator-framework/operator-sdk
[kb-osdk-github-project]: https://github.com/kubernetes-sigs/kubebuilder/projects/7
[addon-pattern-pr]: https://github.com/kubernetes-sigs/kubebuilder/pull/943
