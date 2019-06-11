# Using Finalizers

`Finalizers` allow controllers to implement asynchronous pre-delete hooks. Let's
say you create an external resource (such as a storage bucket) for each object of
your API type, and you want to delete the associated external resource
on object's deletion from Kubernetes, you can use a finalizer to do that.

You can read more about the finalizers in the [Kubernetes reference docs](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers). The section below demonstrates how to register and trigger pre-delete hooks
in the `Reconcile` method of a controller.

The key point to note is that a finalizer causes "delete" on the object to become 
an "update" to set deletion timestamp. Presence of deletion timestamp on the object
indicates that it is being deleted. Otherwise, without finalizers, a delete
shows up as a reconcile where the object is missing from the cache.

Highlights:
- If the object is not being deleted and does not have the finalizer registered,
  then add the finalizer and update the object in Kubernetes.
- If object is being deleted and the finalizer is still present in finalizers list,
  then execute the pre-delete logic and remove the finalizer and update the
  object.
- Ensure that the pre-delete logic is idempotent.

{{#literatego ../cronjob-tutorial/testdata/finalizer_example.go}}

