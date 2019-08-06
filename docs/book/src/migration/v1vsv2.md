# Kubebuilder v1 vs v2

This document cover all breaking changes when migrating from v1 to v2.

The details of all changes (breaking or otherwise) can be found in
[controller-runtime](https://github.com/kubernetes-sigs/controller-runtime/releases),
[controller-tools](https://github.com/kubernetes-sigs/controller-tools/releases)
and [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder/releases)
release notes.

## Common changes

V2 project uses go modules. But kubebuilder will continue to support `dep` until
go 1.13 is out.

## controller-runtime

- `Client.List` now uses functional options (`List(ctx, list, ...option)`) instead
of `List(ctx, ListOptions, list)`.
- `Client.DeleteAllOf` was added to the `Client` interface.

- Metrics are on by default now.

- A number of packages under `pkg/runtime` have been moved, with their old
locations deprecated. The old locations will be removed before
controller-runtime v1.0.0. See the [godocs][pkg-runtime-godoc] for more
information.

#### Webhook-related

- Automatic certificate generation for webhooks has been removed, and webhooks
will no longer self-register. Use controller-tools to generate a webhook
configuration. If you need certificate generation, we recommend using
[cert-manager](https://github.com/jetstack/cert-manager). Kubebuilder v2 will
scaffold out cert manager configs for you to use -- see the
[Webhook Tutorial](/cronjob-tutorial/webhook-implementation.md) for more details.

- The `builder` package now has separate builders for controllers and webhooks,
which facilitates choosing which to run.

## controller-tools

The generator framework has been rewritten in v2. It still works the same as
before in many cases, but be aware that there are some breaking changes.
Please check [marker documentation](/reference/markers.md) for more details.

## Kubebuilder

- Kubebuilder v2 introduces a simplified project layout. You can find the design
doc [here](https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/simplified-scaffolding.md).

- In v1, the manager is deployed as a `StatefulSet`, while it's deployed as a
`Deployment` in v2.

- The `kubebuilder create webhook` command was added to scaffold
mutating/validating/conversion webhooks. It replaces the
`kubebuilder alpha webhook` command.

- v2 uses `distroless/static` instead of Ubuntu as base image. This reduces
image size and attack surface.

- v2 requires kustomize v3.1.0+.

[LeaderElectionRunable]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/manager#LeaderElectionRunnable
[pkg-runtime-godoc]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/runtime
