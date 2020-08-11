# Kubebuilder v2 vs v3

This document cover all breaking changes when migrating from v2 to v3.

The details of all changes (breaking or otherwise) can be found in
[kubebuilder](https://github.com/kubernetes-sigs/kubebuilder/releases) release notes.

## controller-tools

#### CustomResourceDefinition-related

The default `CustomResourceDefinition` version is now `v1`, bumped from `v1beta1` which is
[deprecated][crds-deprecated-doc].

#### Webhook-related

The default `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` versions are now `v1`,
bumped from `v1beta1` which is [deprecated][webhook-deprecated-pr].

## Kubebuilder

- Kubebuilder v3 introduces [plugins][plugins-design] that determine what scaffolds your project
should have when running `kubebuilder` commands.

- A few fields have been added to the `PROJECT` file:
  - `projectName`: the name of your project, typically the base of your `repo` value.
  - `layout`: the key of the plugin used to scaffold your project.

[crds-deprecated-doc]:https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#create-a-customresourcedefinition
[webhook-deprecated-pr]:https://github.com/kubernetes-sigs/controller-runtime/issues/1123
[plugins-design]:../../../../designs/extensible-cli-and-scaffolding-plugins-phase-1.md
