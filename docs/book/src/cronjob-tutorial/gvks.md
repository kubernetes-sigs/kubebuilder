# Groups and versions and kinds, oh my!

Before getting started with the API, talk about terminology
a bit.

When discussing APIs in Kubernetes, 4 terms are often used: *groups*,
*versions*, *kinds*, and *resources*.

## Groups and versions

An *API Group* in Kubernetes is simply a collection of related
functionality.  Each group has one or more *versions*, which, as the name
suggests, allow you to change how an API works over time.

## Kinds and resources

Each API group-version contains one or more API types, called
*Kinds*.  While a Kind may change forms between versions, each form must
be able to store all the data of the other forms, somehow (you can store
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

When referring to a kind in a particular group version, it's called
a *GroupVersionKind*, or GVK for short.  Same with resources and GVR. As
you'll see shortly, each GVK corresponds to a given root Go type in
a package.

Now that the terminology is clear, *actually* create the
API!

## So, how can you create an API?

In the next section, [Adding a new API](../cronjob-tutorial/new-api.html), check how the tool helps you
create your APIs with the command `kubebuilder create api`.

The goal of this command is to create a Custom Resource (CR) and Custom Resource Definition (CRD) for your Kind(s). To check it further see; [Extend the Kubernetes API with CustomResourceDefinitions][kubernetes-extend-api].

## But, why create APIs at all?

New APIs are how you teach Kubernetes about your custom objects. Controller-gen uses the Go structs to generate a CRD which includes the schema for your data as well as tracking data like what the new type is called. You can then create instances of your custom objects which your [controllers][controllers] manage.

Your APIs and resources represent your solutions on the clusters. Basically, the CRDs are a definition of your customized Objects, and the CRs are an instance of it.

## Ah, do you have an example?

Think about the classic scenario where the goal is to have an application and its database running on the platform with Kubernetes. Then, one CRD could represent the App, and another one could represent the DB. By having one CRD to describe the App and another one for the DB, you avoid hurting concepts such as encapsulation, the single responsibility principle, and cohesion. Damaging these concepts could cause unexpected side effects, such as difficulty in extending, reuse, or maintenance, just to mention a few.

In this way, you can create the App CRD which has its controller and which would be responsible for things like creating Deployments that contain the App and creating Services to access it and etc. Similarly, you could create a CRD to represent the DB, and deploy a controller that would manage DB instances.

## Err, but what's that Scheme thing?

The `Scheme` you saw before is simply a way to keep track of what Go type
corresponds to a given GVK (don't be overwhelmed by its
[godocs](https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime?tab=doc#Scheme)).

For instance, suppose you mark the
`"tutorial.kubebuilder.io/api/v1".CronJob{}` type as being in the
`batch.tutorial.kubebuilder.io/v1` API group (implicitly saying it has the
Kind `CronJob`).

Then, you can later construct a new `&CronJob{}` given some JSON from the
API server that says

```json
{
    "kind": "CronJob",
    "apiVersion": "batch.tutorial.kubebuilder.io/v1",
    ...
}
```

or properly look up the group version when you go to submit a `&CronJob{}`
in an update.

[kubernetes-extend-api]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/
[controllers]: ../cronjob-tutorial/controller-overview.md
