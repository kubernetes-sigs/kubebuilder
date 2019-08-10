# Groups and Versions and Kinds, oh my!

Actually, before we get started with our API, we should talk terminology
a bit.

When we talk about APIs in Kubernetes, we often use 4 terms: *groups*,
*versions*, *kinds*, and *resources*.

## Groups and Versions

An *API Group* in Kubernetes is simply a collection of related
functionality.  Each group has one or more *versions*, which, as the name
suggests, allow us to change how an API works over time.

## Kinds and Resources

Each API group-version contains one or more API types, which we call
*Kinds*.  While a Kind may change forms between versions, each form must
be able to store all the data of the other forms, somehow (we can store
the data in fields, or in annotations).  This means that using an older
API version won't cause newer data to be lost or corrupted.  See the
[Kubernetes API
guidelines](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md)
for more information.

You'll also hear mention of *resources* on occasion.  A resource is simply
a use of a Kind in the API.  Often, there's a one-to-one mapping between
Kinds and resources.  For instance, the `pods` resource corresponds to the
`Pod` Kind.  However, sometimes, the same Kind may be returned by multiple
resources.  For instance, the `Scale` Kind is returned by all scale
subresources, like `deployments/scale` or `replicasets/scale`.  This is
what allows the Kubernetes HorizontalPodAutoscaler to interact with
different resources.  With CRDs, however, each Kind will correspond to
a single resource.

Notice that resources are always lowercase, and by convention are the
lowercase form of the Kind.

## So, how does that correspond to Go?

When we refer to a kind in a particular group-version, we'll call it
a *GroupVersionKind*, or GVK for short.  Same with resources and GVR. As
we'll see shortly, each GVK corresponds to a given root Go type in
a package.

Now that we have our terminology straight, we can *actually* create our
API!

## Err, but what's that Scheme thing?

The `Scheme` we saw before is simply a way to keep track of what Go type
corresponds to a given GVK (don't be overwhelmed by its
[godocs](https://godoc.org/k8s.io/apimachinery/pkg/runtime#Scheme)).

For instance, suppose we mark that the
`"tutorial.kubebuilder.io/api/v1".CronJob{}` type as being in the
`batch.tutorial.kubebuilder.io/v1` API group (implicitly saying it has the
Kind `CronJob`).

Then, we can later construct a new `&CronJob{}` given some JSON from the
API server that says

```json
{
    "kind": "CronJob",
    "apiVersion": "batch.tutorial.kubebuilder.io/v1",
    ...
}
```

or properly look up the group-version when we go to submit a `&CronJob{}`
in an update.
