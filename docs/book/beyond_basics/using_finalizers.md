# Using Finalizers

`Finalizers` allow controllers to implement asynchronous pre-delete hooks. Let
say you create an external resource such as a storage bucket for each object of
the API type you are implementing and you would like to clean up the external resource
when the corresponding object is deleted from Kubernetes, you can use a
finalizer to delete the external resource.

You can read more about the finalizers in the [kubernetes reference docs](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers). Section below
demonstrates how to register and trigger pre-delete hooks in the `Reconcile`
method of a controller.

{% method %}

Highlights:
- If object is not being deleted and does not have the finalizer registered,
  then add the finalizer and update the object in kubernetes.
- If object is being deleted and the finalizer is still present in finalizers list,
  then execute the pre-delete logic and remove the finalizer and update the
  object.
- You should implement the pre-delete logic in such a way that it is safe to 
 invoke it multiple times for the same object.

{% sample lang="go" %}
```go

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		// handle error
	}

	// name of your custom finalizer
	myFinalizerName := "storage.finalizers.example.com"

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object.
		if !containsString(instance.ObjectMeta.Finalizers, myFinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{Requeue: true}, nil
			}
		}
	} else {
		// The object is being deleted
		if containsString(instance.ObjectMeta.Finalizers, myFinalizerName) {
			// our finalizer is present, so lets handle our external dependency
			if err := r.deleteExternalDependency(instance); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return reconcile.Result{}, err
			}

			// remove our finalizer from the list and update it.
			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{Requeue: true}, nil
			}
		}

        // Our finalizer has finished, so the reconciler can do nothing.
        return reconcile.Result{}, nil
	}
	....
	....
}

func (r *Reconciler) deleteExternalDependency(instance *MyType) error {
	log.Printf("deleting the external dependencies")
	//
	// delete the external dependency here
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple types for same object.
}

//
// Helper functions to check and remove string from a slice of strings.
//
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

```
{% endmethod %}
