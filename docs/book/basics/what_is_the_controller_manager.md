{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# What is the Controller-Manager

The Controller-Manager is an executable that wraps one or more Controllers.  It may
either be built and run locally against a remote cluster, or run as a container
in the cluster.

When run as a container, it should be installed into its own Namespace with a
ServiceAccount and RBAC permissions on the appropriate resources.

The Controller-Manager may optionally install the CRDs when it is started, or
the CRDs may be installed by the yaml config that creates the Namespace,
Controller-Manager StatefulSet, and RBAC rules.

Note that the Controller-Manager is run as a StatefulSet and not a Deployment.  This
is to ensure that only 1 instance of the Controller-Manager is run at a time (a Deployment
may sometimes run multiple instances even with replicas set to 1).

#### Building and Running Locally

Build and run locally against the cluster defined in ~/.kube/config.

> GOBIN=${PWD}/bin go install ${PWD#$GOPATH/src/}/cmd/controller-manager
> ./bin/controller-manager --kubeconfig ~/.kube/config

In another terminal, create the sample resource.

> kubectl apply -f hack/sample/containerset.yaml

{% panel style="info", title="Building and Running a Controller-Manager Container" %}
The image for the Controller-Manager maybe built using the `Dockerfile.controller` Dockerfile.

> docker build . -t gcr.io/containerset/controller -f Dockerfile.controller
> docker push gcr.io/containerset/controller

The yaml configuration for the Controller-Manager maybe created with
`kubebuilder create config --image gcr.io/containerset/controller --name containerset`
This will create the `hack/install.yaml` file to install the Controller-Manager.

> kubectl apply -f hack/install.yaml

{% endpanel %}
