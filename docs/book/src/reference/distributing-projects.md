# How to Distribute Projects Built with Kubebuilder

## Overview

This section would help users understand how to package and deliver their Kubebuilder-based Operators to end users.

### Step 1:  operator-sdk generate kustomize manifests

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
    > my-example.com
    Kustomize files generated successfully
```
This generates the file: config/manifests/bases/kubebuilderdemo.clusterserviceversion.yaml.

### Step 2: Add the config/manifests/kustomization.yaml

These resources constitute the fully configured set of manifests, used to generate the 'manifests/' directory in a bundle.

```
resources:
- bases/kubebuilderdemo.clusterserviceversion.yaml
- ../default
- ../samples
```

### Step 3: Generate the Operator Bundle

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