# Status Subresource 
By convention, the Kubernetes API makes a distinction between the specification
of the desired state of an object (a nested object field called "spec") and the 
status of the object at the current time (a nested object field called "status"). 
The specification is a complete description of the desired state, including 
configuration settings provided by the user, default values expanded by the 
system, and properties initialized or otherwise changed after creation by other 
ecosystem components (e.g., schedulers, auto-scalers), and is persisted in Etcd 
with the API object. The status summarizes the current state of the object in 
the system, and is usually persisted with the object by an automated processes 
but may be generated on the fly. At some cost and perhaps some temporary 
degradation in behavior, the status could be reconstructed by observation if it 
were lost.

The PUT and POST verbs on objects MUST ignore the "status" values, to avoid 
accidentally overwriting the status in read-modify-write scenarios. A /status 
subresource MUST be provided to enable system components to update statuses of 
resources they manage.

You can read more about the API convention in [Kubernetes API Convention
doc](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status).

{% panel style="info", title="Status subresource support in Kubernetes" %}
Subresource support for CRD is enabled by default in 1.10+ releases
, so ensure that you are running kubernetes with the minimum version. You can
use `kubectl version` command to check the Kubernetes version.
{% endpanel %}

{% method %}
## Enabling Status subresource in CRD definition
First step is to enable status subresource in the CRD definition. This can be
achieved by adding a comment `// +kubebuilder:subresource:status` just above the
Go type definition as shown in example below.


{% sample lang="go" %}
```Go
// MySQL is the Schema for the mysqls API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type MySQL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MySQLSpec   `json:"spec,omitempty"`
	Status MySQLStatus `json:"status,omitempty"`
}
```

CRD generation tool will use the `+kubebuilder:subresource:status` annotation to
enable status subresource in the CRD definition. So, if you run, `make manifests`,
it will regenerate the CRD manifests under `config/crds/<kind_types.yaml`. Here
is an example manifests with status subresource enabled. Note the `subresources`
section with an empty `status` field.

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    controller-tools.k8s.io: "1.0"
  name: mysqls.myapps.examples.org
spec:
  group: myapps.examples.org
  names:
    kind: MySQL
    plural: mysqls
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          type: object
        status:
          type: object
  version: v1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
```

Ensure you have updated the CRD definition in your cluster by running `kubectl
apply -f config/crds`

{% endmethod %}

{% method %}
## Getting and Updating status in Reconciler code

In order to get the status subresource, you don't have do anything new. The
`Get` client method returns the entire object which contains the status field.

For updating the status subresource, compute the new status value and update it 
in the object and then issue `client.Status().Update(context.Background(), &obj)` to update the
status.

{% sample lang="go" %}
```go
	instance := &myappsv1.MySQL{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		// handle err
	}

	// updating the status
	instance.Status.SomeField = "new-value"
	err = r.Status().Update(context.Background(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}
```

{% endmethod %}
