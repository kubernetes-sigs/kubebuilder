# Admission Webhooks

[Admission webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#what-are-admission-webhooks) are HTTP callbacks that receive admission requests, process
them and return admission responses.

Kubernetes provides the following types of admission webhooks:

- **Mutating Admission Webhook**:
These can mutate the object while it's being created or updated, before it gets
stored. It can be used to default fields in a resource requests, e.g. fields in
Deployment that are not specified by the user. It can be used to inject sidecar
containers.

- **Validating Admission Webhook**:
These can validate the object while it's being created or updated, before it gets
stored. It allows more complex validation than pure schema-based validation.
e.g. cross-field validation and pod image whitelisting.

The apiserver by default doesn't authenticate itself to the webhooks. However,
if you want to authenticate the clients, you can configure the apiserver to use
basic auth, bearer token, or a cert to authenticate itself to the webhooks.
You can find detailed steps
[here](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#authenticate-apiservers).

<aside class="note">
<H1>Execution Order</H1>

**Validating webhooks run after all mutating webhooks**, so you don't need to worry about another webhook changing an 
object after your validation has accepted it.

</aside>

## Handling Resource Status in Admission Webhooks

<aside class="warning">
<H1>Modify status</H1>

**You cannot modify or default the status of a resource using a mutating admission webhook**. 
Set initial status in your controller when you first see a new object.

</aside>

### Understanding Why:

#### Mutating Admission Webhooks

Mutating Admission Webhooks are primarily designed to intercept and modify requests concerning the creation, 
modification, or deletion of objects. Though they possess the capability to modify an object's specification, 
directly altering its status isn't deemed a standard practice, 
often leading to unintended results.

```go
// MutatingWebhookConfiguration allows for modification of objects.
// However, direct modification of the status might result in unexpected behavior.
type MutatingWebhookConfiguration struct {
    ...
}
```

#### Setting Initial Status

For those diving into custom controllers for custom resources, it's imperative to grasp the concept of setting an 
initial status. This initialization typically takes place within the controller itself. The moment the controller 
identifies a new instance of its managed resource, primarily through a watch mechanism, it holds the authority 
to assign an initial status to that resource.

```go
// Custom controller's reconcile function might look something like this:
func (r *ReconcileMyResource) Reconcile(request reconcile.Request) (reconcile.Result, error) {
    // ... 
    // Upon discovering a new instance, set the initial status
    instance.Status = SomeInitialStatus
    // ...
}
```

#### Status Subresource

Delving into Kubernetes custom resources, a clear demarcation exists between the spec (depicting the desired state) 
and the status (illustrating the observed state). Activating the /status subresource for a custom resource definition 
(CRD) bifurcates the `status` and `spec`, each assigned to its respective API endpoint. 
This separation ensures that changes introduced by users, such as modifying the spec, and system-driven updates, 
like status alterations, remain distinct. Leveraging a mutating webhook to tweak the status during a spec-modifying 
operation might not pan out as expected, courtesy of this isolation.

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: myresources.mygroup.mydomain
spec:
  ...
  subresources:
    status: {} # Enables the /status subresource
```

#### Conclusion

While certain edge scenarios might allow a mutating webhook to seamlessly modify the status, treading this path isn't a 
universally acclaimed or recommended strategy. Entrusting the controller logic with status updates remains the 
most advocated approach.
