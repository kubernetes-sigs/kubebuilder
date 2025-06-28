# How to Distribute Projects Built with Kubebuilder

## Overview

This section would help users understand how to package and deliver their Kubebuilder-based Operators to end users.

## By providing raw YAML manifests
Allowing users to install the Operator directly with `kubectl apply -f`.

### Step 1: Generate the Operator Bundle

#### Step 1:  operator-sdk generate kustomize manifests

Example output:
```
    operator-sdk generate kustomize manifests
    Generating kustomize files in config/manifests

    Display name for the operator (required):
    > my-example-operator

    Description for the operator (required):
    > To verify how to generate the bundle

    Provider's name for the operator (required):
    > My Org. Inc

    Any relevant URL for the provider name (optional):
    > https://github.com/my-example

    Comma-separated list of keywords for your operator (required):
    > security,storage,app

    Comma-separated list of maintainers and their emails (e.g. 'name1:email1, name2:email2') (required):
    > rayowang:rayo.wangzl@gmail.com
    Kustomize files generated successfully
```
This generates the file: config/manifests/bases/kubebuilderdemo.clusterserviceversion.yaml.

#### Step 2: Add the config/manifests/kustomization.yaml

These resources constitute the fully configured set of manifests, used to generate the 'manifests/' directory in a bundle.

```
resources:
- bases/kubebuilderdemo.clusterserviceversion.yaml
- ../default
- ../samples
```

#### Step 3: Generate the Operator Bundle

Operator bundles are the standard format for distributing Operators. These bundles include the necessary Kubernetes manifests, CRDs, and metadata required by platforms like OperatorHub.io and OLM.

Run the bundle generation command:

```
kustomize build config/manifests | \
operator-sdk generate bundle \
--version 0.1.0 \
--package my-operator \
--channels alpha \
--default-channel alpha

```

Now you will have a structure such as:

```
bundle
├── manifests
│   ├── app.mydomain.com_myapps.yaml
│   ├── kubebuilderdemo-controller-manager-metrics-service_v1_service.yaml
│   ├── kubebuilderdemo-metrics-reader_rbac.authorization.k8s.io_v1_clusterrole.yaml
│   ├── kubebuilderdemo-myapp-admin-role_rbac.authorization.k8s.io_v1_clusterrole.yaml
│   ├── kubebuilderdemo-myapp-editor-role_rbac.authorization.k8s.io_v1_clusterrole.yaml
│   ├── kubebuilderdemo-myapp-viewer-role_rbac.authorization.k8s.io_v1_clusterrole.yaml
│   └── my-example-operator.clusterserviceversion.yaml
└── metadata
    └── annotations.yaml
```

### Step 2: Validate Deploying the Operator with OLM

#### Step 1: Install OLM (if not installed)

If OLM is not yet installed in your Kubernetes cluster, you can install it with the following command:

```
operator-sdk olm install
```

#### Step 2: Build and Push the Bundle Image

Once OLM is installed, build and push the bundle image:

```
docker build -t docker.io/raysail/kubebuilderdemo-operator-bundle:v1 -f bundle.Dockerfile
docker push docker.io/raysail/kubebuilderdemo-operator-bundle:v1
```

#### Step 3: Run the Bundle

Run the following command to deploy the bundle:

```
operator-sdk run bundle docker.io/raysail/kubebuilderdemo-operator-bundle:v1
```

### Step3: Publishing the Operator to operatorhub.io
To make your Operator available to the public and other users, you can publish it to OperatorHub.io.
For more details, refer to https://operatorhub.io/contribute.