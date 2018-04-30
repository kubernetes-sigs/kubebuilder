{% panel style="danger", title="Staging" %}
Staging documentation under review.
{% endpanel %}

# What is a Resource

A Kubernetes resource is a declarative API with a well defined Schema structure
and Endpoints.  Because the structure of the Schema and Endpoints are both well
understood, many Kubernetes tools work with all Kubernetes resources.

{% method %}

#### What is a Declarative API

A declarative API expresses a fixed state that the cluster must continually
work towards.  Declarative APIs define the *what*, but not the *how*.
Example: `$ replicas 3`

An imperative API expresses an operation that may change state, but does not
define an absolute state that must be maintained.  Imperative APIs express the
*how*, but not *what*.  Example: `$ add-replicas 2`.

In the declarative case if replica is lost the cluster has a clear directive
to create another one, whereas in the latter case this is not necessarily true.

Constraints on the declarative *how* may be defined,
such as ensuring new replicas are healthy before scaling down
old ones when performing a rollout.

{% sample lang="yaml" %}
*Declarative API usage example by invoking `apply`.*

```bash
# object.yaml contains an object declaration
$ kubectl apply -f object.yaml
```
{% endmethod %}

#### Resource Schema

{% method %}

##### Group, Version, Kind

Every Kubernetes resource has a *Group*, *Version* and *Kind* that uniquely identifies it.

* The resource *Kind* is the name of the API - such as Deployment or Service.
* The resource *Version* defines the stability of the API and backward compatibility guarantees -
  such as v1beta1 or v1.
* The resource *Group* is similar to package in a language.  It disambiguate logically different APIs
  that may happen to have identically named *Kind*s.  Groups often contain a domain name, such as k8s.io.

{% sample lang="yaml" %}
*Deployment yaml config Group Version Kind*

```yaml
apiVersion: apps/v1
kind: Deployment
```
{% endmethod %}

{% panel style="info", title="Versions" %}
Resources with different *Version*s but the same *Group* and *Kind* differ in the following ways:

* Unspecified fields may have different defaults
* The same logical fields may have different names or representations

However resources with different versions share the same features and controller.

**Alpha** APIs may break backwards compatibility by changing field names, defaults or behavior.  They may
also be removed without being deprecated.

**Beta** APIs maintain backwards compatibility on field names, defaults and behavior.  They may be
missing features required for GA.  However once the API goes GA, the features will be available
in the Beta version.

**GA** APIs have been available and running in production for sufficient time to have developed
a stable set of field names and defaults as well as complete feature set.

{% endpanel %}

{% method %}

##### Spec, Status, Metadata

Most Kubernetes resources Schema contain 3 components: Spec, Status and Metadata

**Spec**: the resource Spec defines the desired state of the cluster as specified by the user.

**Status**: the resource Status publishes the observed state of the cluster as observed by the controller.

**Metadata**: the resource Metadata contains metadata common to most resources about the object
including as the object name, annotations, labels and more.

{% sample lang="yaml" %}

**Note**: this config have been abbreviated for the purposes of display

*Deployment yaml config with Spec Status and Metadata*

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: nginx
  namespace: default
spec:
  replicas: 1
  template:
    spec:
      containers:
      - image: nginx
        name: nginx
status:
  replicas: 1
  unavailableReplicas: 1
  updatedReplicas: 1
```
{% endmethod %}

{% panel style="warning", title="Spec vs Status" %}
The resource *Status* should not contain the source of truth for any information and should be
able to be generated from controllers by looking at the cluster state.  Values assigned by
controllers, such as the Service `spec.clusterIp`, should still be set on the *Spec*, even if they are
never intended to be set by end users.
{% endpanel %}
{% method %}
#### Resource Endpoints

Kubernetes resources have well defined endpoints as described below.

##### Create, Update, Patch, Delete

The create, update, patch and delete endpoints may be used to modify objects.  The update endpoint
replaces the object with what is provided, whereas the patch endpoint selectively updates
fields.

##### Get, List, Watch

The get, list and watch endpoints may be used to get a specific resource by name, list all
resources matching a labels, or continually watch for updates.

{% sample lang="yaml" %}

*Deployment Endpoints under `/apis/apps/v1`*

```yaml
name: "deployments"
kind: "Deployment"
verbs:
- "create"
- "delete"
- "deletecollection"
- "get"
- "list"
- "patch"
- "update"
- "watch"
```

{% endmethod %}

{% panel style="warning", title="Warning on Updates" %}
The update API should only be used to read-then-write an object, and never used to
update an object directly from declarative config.  This is because the object state
may be partially managed by controllers running in the cluster and this state would
be lost when the update replaces the current object with the declarative config.
For example updating a Service from declarative config rather than a read-then-write
would clear the Service `spec.clusterIp` field set by the controller.
{% endpanel %}

{% panel style="info", title="Watch Timeouts" %}
If used directly, a watch API call will timeout and need to be re-established.  The kubebuilder
libraries hide the details behind watches from users and automatically re-establish connections.
{% endpanel %}

{% method %}

#### Subresources

While most operations can be represented declaratively, some may not, such as
*logs*, *attach* or *exec*.  These operations may be implemented as *subresources*.

Subresources are functions attached to resources, but that have their
own Schema and Endpoints.  By having different resources each implement
the same subresource API, resources can implemented shared interfaces.

For example Deployment, ReplicaSet and StatefulSet each implement the
*scale* subresource API, making it easy to build tools which scale any of them
as well as scale any other resources that implement the *scale* subresource.

{% sample lang="yaml" %}

*Deployment **Scale Subresource** Endpoints under `/apis/apps/v1`*

```yaml
name: "deployments/scale"
group: "autoscaling"
version: "v1"
kind: "Scale"
verbs:
- "get"
- "patch"
- "update"
```

{% endmethod %}

#### Labels, Selectors and Annotations

Labels in ObjectMeta data are key-value pairs that may be queried to find objects matching a query.
Labels are used to form logical connections between objects in a Kubernetes cluster.  For instance
Services use labels to determine which Pods to direct traffic to, and Deployments use labels
(along with OwnersReferences) to identify Pods they own.

Annotations allow arbitrary data to be written to resources that may not fit within the
Schema of the resource.

{% panel style="info", title="Extending Built In Types" %}
Annotations may be used to define new extension fields on resources without modifying the
Schema of the object.  This allows users to define their own controller extensions for
existing core Kubernetes resources without modifying upstream Kubernetes.
{% endpanel %}

#### Namespaces

While most resources are Namespaced, that is the objects are scoped to a Namespace, some resources
are non-namespaces and scoped to the cluster.  Examples of non-namespaced resources include
Nodes, Namespaces and ClusterRole.
