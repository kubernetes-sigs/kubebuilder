# What is the Manager

The Manager is an executable that wraps one or more Controllers.  It may
either be built and run locally against a remote cluster, or run as a container
in the cluster.

When run as a container, it should be installed into its own Namespace with a
ServiceAccount and RBAC permissions on the appropriate resources.  The configs
to do this are automatically generated for the user by running `make`.

Note that the Manager is run as a StatefulSet and not a Deployment.  This
is to ensure that only 1 instance of the Manager is run at a time (a Deployment
may sometimes run multiple instances even with replicas set to 1).

#### Building and Running Locally

Build and run locally against the cluster defined in ~/.kube/config.  Note
this requires a running Kubernetes cluster to be accessible with the
~/.kube/config.

```bash
make run
```

In another terminal, create an instance of your resource.

```bash
kubectl apply -f config/samples/yourinstance.yaml
```
