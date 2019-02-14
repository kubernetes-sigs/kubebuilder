# What is a Controller

Controllers implement APIs defined by *Resources*.  Unlike Controllers in the ModelViewController
pattern, Kubernetes Controllers are run *asynchronously* after the Resources (Models) have
been written to storage.  This model is highly flexible and allows new Controllers to be
added for Models through extension instead of modification.
  
A Kubernetes Controller is a routine running in a Kubernetes cluster that watches for create /
update / delete events on Resources, and triggers a Reconcile function in response.  Reconcile
is a function that may be called at any time with the Namespace and Name of an object (Resource
instance), and it will make the cluster state match the state declared in the object Spec.
Upon completion, Reconcile updates the object Status to the new actual state.

It is common for Controllers to watch for changes to the Resource type that they Reconcile
*and* Resource types of objects they create.  e.g. a ReplicaSet Controller watches for
changes to ReplicaSets *and* Pods.  The Controller will trigger a Reconcile for a ReplicaSet
in response to either an event for that ReplicaSet *or* in response to an event for a
Pod created by that ReplicaSet.

In some cases Reconcile may only update the Status without updating any cluster state.  

Illustrative example:

- A ReplicaSet object is created with 10 replicas specified in the Spec
- ReplicaSetController Reconcile reads the Spec and lists the Pods owned by the ReplicaSet
- No Pods are found, ReplicaSetController creates 10 Pods and updates the Status with 0/10 Pods running
- ReplicaSetController Reconcile is triggered as the Pods start running, and updates Status in the
  ReplicaSet object.


**Kubernetes APIs and Controllers have *level* based implementations to facilitate self-
healing and periodic reconciliation.  This means no state is provided to the Reconcile
when it is called.**

## What is a Level Based API

The term *level-based* comes from interrupts hardware, where interrupts may be either *level-based* or *edge-based*.

Kubernetes defines a level-based API as implemented by reading the observed (actual) state of the system,
comparing it to what is declared in the object *Spec*, and making changes to the system state so
it matches the state of the Spec **at the time Reconcile is called**.
 
This has a number of notable properties:

- Reconcile skips intermediate or obsolete values declared in the Spec and
  works directly toward the *current* Spec.
- Reconcile may batch multiple events together before processing them instead
  of handling each individually

Consider the following examples of level based API implementations.

**Example 1**: Batching Events

A user creates a ReplicaSet with 1000 replicas.  The ReplicaSet creates 1000 Pods and maintains a
Status field with the number of healthy Pods.  In a level based system, the Controller batches
the Pod updates together (the Reconcile only gets the ReplicaSet Namespace and Name) before triggering
the Reconcile.  In an edge based system, the Controller responds to each individual Pod event, potentially
performing 1000 sequential updates to the Status instead of 1.

**Example 2**: Skipping Obsolete States

A user creates a rollout for a Deployment containing a new container image.  Shortly after
starting the rollout, the user realizes the containers are crash looping because they need
to increase memory thresholds when running the new image.
The user updates the Deployment with the new memory limit to start a new rollout.  In a
level based system, the Controller will immediately stop rolling out the old values and start
the rollout for the new values.  In an edge based system the Controller may complete the first
rollout before starting the next.

## Watching Events

The Controller Reconcile is triggered by cluster events.

##### Watching Resources

Controllers must watch for events for the Resource they Reconcile. The ReplicaSetController
watches for changes to ReplicaSets and triggers a Reconcile in response.
 
###### ReplicaSet Creation

The following diagram shows a creation event triggering a reconcile.

{% sequence %}
participant API as A
participant ReplicaSetController as C

Note right of C: User creates ReplicaSet
A-->C: ReplicaSet Create Event
Note right of C: Reconcile ReplicaSet
{% endsequence %}

##### Watching Created Resources

Controllers should watch for events on the Resources they create.  The ReplicaSetController watches
for Pod events.  If a Pod is deleted, the ReplicaSetController will see the Pod event and
Reconcile the ReplicaSet that created the Pod so it can create a new one.

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

##### Watching Related Resource Events

Controllers may watch for events on Resources that are related, but they did not create.  The
DaemonSetController watches for changes to Nodes.  If a new Node is created, the Controller
will create a new Pod scheduled on that Node.  In this case, *all* DaemonSet objects are reconciled
each time a Node is created.

## Create Objects During Reconciliation

Many Controllers create new Kubernetes objects as part of a reconcile.  These objects
are *owned* by the object responsible for their creation.
This relationship is recorded both in an *OwnersReference* in the ObjectMeta of the created
objects and through labels (on the created object) + selectors (on the created object).

The labels + selectors allow the creating controller to find all of the objects it has created,
by listing them using their label.  The *OwnersReference* maps the created object to its
owner when there is an event for the created object.

## Writing Status Back to Objects

Controllers are run asynchronously, meaning that the user operation will return a success to
the user before the Controller is run.  If there are issues when the Controller is run,
such as the container image being invalid, the user will not be notified.

Instead the Controller must write back the Status of the object at each Reconcile and
users must check the object Status.

{% panel style="info", title="Status" %}
The controller will keep Status up-to-date not only in response to user initiated events, but also
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

Controllers that implement an API for a specific application, such as Etcd, Spark or Cassandra are
often referred to as *Operators*.
