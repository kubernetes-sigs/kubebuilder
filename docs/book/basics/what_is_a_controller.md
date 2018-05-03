{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# What is a Controller

Controllers implement APIs defined by *Resources*.  Controllers are
routines running in a Kubernetes cluster that watch both the resource API they implement as well
as related resource APIs to form a whole view of the cluster state.  Controllers reconcile each object's
(resource instance) desired state as declared in the Spec (e.g. 10 replicas of Pod running nginx)
with the state observed read from the APIs (e.g. 0 replicas of Pods running nginx).  Reconciliation is
done both in response to changes in cluster state, and periodically for each observed object.

**Kubernetes APIs and controllers have *level* based implementations to facilitate self-
healing and periodic reconciliation.**

## What is a Level Based API

The term *level-based* comes from interrupts hardware, where interrupts may be either *level-based* or *edge-based*.
This book does not go into the details of the hardware definitions of these terms.

Kubernetes defines a level-based API as implemented by reading the observed state of the system,
comparing it to the desired state declared in the object *Spec*, and moving directly toward the
current desired state.
 
This has a number of notable properties:

- reconciliation works directly towards the current desired state without having to complete
  obsolete desired states
- when many events quickly occur that trigger a reconciliation for the same object, reconciliation will
  process many of the events at once by comparing observed and desired states,
  not handling the individual events.
- the system may trigger reconciliation periodically for objects without a specific event occurring.

Consider the following examples of level based API implementations.

**Example 1**: Batching Events

A user creates a Deployment with 1000 replicas.  The Deployment creates 1000 Pods and maintains a
Status field with the number of healthy Pods.  In a level based system, the controller doesn't
update the Status for each Pod (1000 writes), but instead batches updates together with
the number of observed healthy Pods during reconciliation.  In an edge based system, the
controller would respond to each individual Pod event with a Status update.

**Example 2**: Skipping Obsolete States

A user creates a rollout for a new container image.  Shortly after starting the rollout, the user realizes
the containers are crash looping because they need to increase memory thresholds for the new image to
run.  The user updates the PodTemplate with the new memory limit and a new rollout is started.  In a
level based system, cluster will immediately start working towards the new target instead of trying
to complete the old rollout, whereas in an edge based system it might finish responding to the first
event and rollingout the old image before starting the correct one.

## Watching Events and Periodic Reconcile

The controller reconciliation between the declared desired state in the object and
the observed state of the cluster is triggered both by cluster events and periodically
for each object.

##### Watching Resource Events

Controllers must watch for events on the resource whose API they implement.  The ReplicaSet-controller
watches for changes to ReplicaSets.  If a ReplicaSet is created, modified or deleted then the
controller calls the Reconcile method with the key of the ReplicaSet.  Reconcile will read the ReplicaSet
state and the state of all of its Pods.
 
###### ReplicaSet Creation

The following diagram shows a creation event triggering a reconcile.

{% sequence %}
participant API as A
participant ReplicaSetController as C

Note right of C: User creates ReplicaSet
A-->C: ReplicaSet Create Event
Note right of C: Reconcile ReplicaSet
{% endsequence %}

##### Watching Generated Resource Events

Controllers should watch for events on the resources they generate.  The ReplicaSet-controller watches
for changes to Pods.  If a Pod is deleted (e.g. machine fails), the ReplicaSet-controller will
see the Pod event and Reconcile the owning ReplicaSet.

###### ReplicaSet Creation And Self-Healing

The following diagram shows a series of events after creating a new ReplicaSet and then a Pod getting deleted.

{% sequence %}
participant API as A
participant ReplicaSetController as C

Note right of C: User creates ReplicaSet
A-->C: ReplicaSet Create Event
Note right of C: Reconcile ReplicaSet
C->A: Read ReplicaSet
C->A: List Pods
C->>A: Create Pod 1
C->>A: Create Pod 2
C->>A: Create Pod 3
C->>A: Update ReplicaSet Status
Note right of C: Pods started on Nodes
A-->C: Pod 1 Running Event
A-->C: Pod 2 Running Event
A-->C: Pod 3 Running Event
Note right of C: Reconcile ReplicaSet
C->A: Read ReplicaSet
C->A: List Pods
C->>A: Update ReplicaSet Status
Note right of C: User deletes Pod 1
A-->C: Pod 1 Deleted Event
Note right of C: Reconcile ReplicaSet
C->A: Read ReplicaSet
C->A: List Pods
C->>A: Create Pod 4
C->A: Update ReplicaSet Status
{% endsequence %}

<!---
##### Watching Transitively Generated Resource Events

Controllers may watch for events on resources generated by resources they generated.  The Deployment-controller
watches for changes to Pods even though it generates ReplicaSets.  During a rolling update, a Deployment
will gradually scale up a new ReplicaSet and gradually scale down an old ReplicaSet as Pods in the new
ReplicaSet become healthy.  In order to perform the rollout, the Deployment must respond to Pod events
to continue with the rollout.

###### Deployment Creation And Rolling Updates

The following diagram shows a series of events after creating a new
Deployment and then updating the ContainerImage to trigger a rolling update.

{% sequence %}
participant API as A
participant DeploymentController as C

Note right of C: User creates Deployment
A-->C: Deployment Create Event
Note right of C: Reconcile Deployment
C->A: Read Deployment
C->A: List ReplicaSets
C->>A: Create ReplicaSet 1 with 3 replicas
Note right of C: User Updates Deployment ContainerImage
A-->C: Deployment Update Event
Note right of C: Reconcile Deployment
C->A: Read Deployment
C->A: List ReplicaSets
C->A: ListPods ReplicaSet
C->>A: Create ReplicaSet 2 with 1 replica
Note right of C: End Reconcile
A-->C: Pod 1 Running Event
Note right of C: Reconcile Deployment
C->A: Read Deployment
C->A: List ReplicaSets
C->A: ListPods ReplicaSet
C->>A: Scale down ReplicaSet 1 by 1
C->>A: Scale up ReplicaSet 2 by 1
Note right of C: End Reconcile
A-->C: Pod 1 Running Event
Note right of C: Reconcile Deployment
C->A: Read Deployment
C->A: List ReplicaSets
C->A: ListPods ReplicaSet
C->>A: Scale down ReplicaSet 1 by 1
C->>A: Scale up ReplicaSet 2 by 1
Note right of C: End Reconcile
A-->C: Pod 1 Running Event
Note right of C: Reconcile Deployment
C->A: Read Deployment
C->A: List ReplicaSets
C->A: ListPods ReplicaSet
C->>A: Scale down ReplicaSet 1 by 1
Note right of C: End Reconcile
{% endsequence %}
-->

##### Watching Related Resource Events

Controllers may watch for events on resources that are related, but they do not own.  The
DaemonSet-controller watches for changes to Nodes.  If a new Node is created, the controller
will create a new Pod scheduled that that Node.  In this case, *all* DaemonSet objects are reconciled
each time a Node is created.

Example workflow:

1. Node is added
2. Controller gets Node create event and lists all DaemonSets
3. For each DaemonSet, controller calls Reconcile
...

##### Handling Non-Resource Events

Controllers may handle non-resoure events using user defined mechanisms such as Webhooks or polling.
This is necessary if the controller manages things outside the cluster, such as cloud provider
resources (e.g. NetworkStorage).

##### Periodic Reconcile

Each object is periodically reconciled even if no events are observed.

## Generating Objects During Reconciliation

Many controllers generated new Kubernetes objects as part of a reconcile.  For example the
Deployment controller generates ReplicaSets, and the ReplicaSet controller generates Pods.
The controller ownership relationship between the generating and generated objects is
recorded both in an *OwnersReference* in the ObjectMeta of the generated objects and through
labels (on the generated object) + selectors (on the generating object).  The labels + selectors
allow the generating controller to find all of the objects it has generated, by looking them up
based on their label and the *OwnersReference* confirms the relationship to address cases where
labels have been modified or overlap.

## Writing Status Back to Objects

Controllers are run asynchronously, meaning that the user will not get a status update
in response to `kubectl applying` (or creating, updating, patching) an object, since the controller
will not have reconciled the state.  In order to communicate status back to the user,
controllers write to the object *Status* field and the user must read this field with `kubectl get`.

{% panel style="info", title="Status" %}
The controller will keep Status up-to-date both in response to user initiated events, but also
in response to non-user initiated events, such as Node failures.
{% endpanel %}

## Walkthrough: a Deployment Rollout across Deployments, ReplicaSets, Pods

Following is a walkthrough of a Deployment Rolling update.

##### Kubectl commands

Using kubectl, it is possible to call the same watch API used by controllers to trigger
reconciles.  The following example watches Deployments, ReplicaSets and Pods; creates a Deployment;
and updates the Deployment with a new container image (triggering a rolling update).

```bash
# watch deployments in terminal 1
kubectl get -w deployments

# watch replicasets in terminal 2
kubectl get -w replicasets

# watch pods in terminal 3
kubectl get -w pods 

# create deployment
kubectl run nginx --image nginx:1.12 --replicas 3

# rollout new image
kubectl set image deployments nginx *=nginx:1.13
```

##### Flow Diagram

{% sequence width=1000 %}
participant API as A
participant DeploymentController as DC
participant ReplicaSetController as RC
participant Scheduler as S
participant Node (Kubelet) as N

Note right of A: User creates Deployment
A-->DC: Deployment Create Event
Note right of DC: Reconcile Deployment
DC->A: Create ReplicaSet
A-->RC: ReplicaSet Create Event
Note right of RC: Reconcile ReplicaSet
RC->A: Create Pods
A-->S: Pod Create Events
Note right of S: Reconcile Pods
S->A: Schedule Pods To Nodes
A-->N: Pod Update Events
Note right of N: Reconcile Pods
Note right of N: Start Container
N->A: Update Pod Status
A-->RC: Pod Update Events
Note right of RC: Reconcile ReplicaSet
RC->A: Update ReplicaSet Status
A-->DC: Pod Update Events
A-->DC: ReplicaSet Update Events
Note right of DC: Reconcile Deployment
DC->A: Update Deployment Status
{% endsequence %}

## Controllers vs Operators

Some controllers are referred to as *Operators*.  Operators are a specific type of controller
that manage running a specific application such as Redis or Cassandra.