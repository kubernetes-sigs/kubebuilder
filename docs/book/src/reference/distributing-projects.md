# Distributing Your Kubebuilder Project

## Overview

This guide outlines how to distribute your Kubebuilder-based Operator by generating a **bundle** and publishing it either to **OperatorHub.io** or using **OLM (Operator Lifecycle Manager)** for Kubernetes deployments. This process is vital for making your Operator available to a broader audience and ensuring it is easily deployed and managed in Kubernetes clusters.

### Prerequisites

Before proceeding, ensure you have the following installed:
- **Kubebuilder** (for creating the Operator)
- **Operator SDK** (for generating and managing bundles)
- **Kustomize** (for managing Kubernetes configurations)
- **Docker** (for building container images)

### Default Kubebuilder Assumptions for Distribution

Kubebuilder follows a standard project structure and assumes the following when distributing an Operator:
- **Directory Structure**: Your Kubebuilder project should follow this basic structure:

```bash
  /config
    ├── crds/
    ├── manifests/
    └── samples/
  /Makefile
  /PROJECT
  /Dockerfile
  ...
  ```

- **Kustomize**: Kubebuilder uses Kustomize to manage Kubernetes manifests, which simplifies the customization and deployment process.
- **Operator Versions**: Each Operator release is versioned (e.g., 0.1.0), and you can use release channels such as alpha and stable to manage versions.
- **Channels**: By default, the alpha channel is used for experimental or preview versions. You can configure this to use other channels based on your needs.

### Step-by-Step Guide: Publishing Kubebuilder Projects to OperatorHub.io

## Step 1: Prepare Your Kubebuilder Project

Ensure that your Kubebuilder project is properly created and contains the necessary Kubernetes manifests and CRD files.

### Step 1: To initialize a basic Kubebuilder project

```
kubebuilder init --domain mydomain.com --repo github.com/myorg/my-operator
```

### Step 2: Create your API resources

```
kubebuilder create api --group app --version v1 --kind MyApp
```
This will create the necessary directories and files under the config/ directory.

### Step 3: Compltete your business and create your crds

```
make manifests
```

## Step 2: Generate the Operator Bundle

### Step 1:  operator-sdk generate kustomize manifests

Example output:
```
    operator-sdk generate kustomize manifests
    Generating kustomize files in config/manifests

    Display name for the operator (required):
    > my-operator

    Description for the operator (required):
    > To verify how to generate the bundle

    Provider's name for the operator (required):
    > zhilongwang

    Any relevant URL for the provider name (optional):
    > https://github.com/rayowang

    Comma-separated list of keywords for your operator (required):
    > crd,custom-resource,controller

    Comma-separated list of maintainers and their emails (e.g. 'name1:email1, name2:email2') (required):
    > rayowang:rayo.wangzl@gmail.com
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

Note: Due to compatibility issues between the latest Kubebuilder v4 and Operator SDK, we need to disable the cliVersion field before running the command. To do so, execute:

```
# On macOS
sed -i '' '/^cliVersion:/s/^/#/' PROJECT
# On Linux
sed -i '/^cliVersion:/s/^/#/' PROJECT
```

Then, run the bundle generation command:

```
kustomize build config/manifests | \
operator-sdk generate bundle \
--version 0.1.0 \
--package my-operator \
--channels alpha \
--default-channel alpha

```

Breakdown of the Command:

•	--version: Specifies the version of the bundle (e.g., 0.1.0).
•	--kustomize-dir: Points to the directory containing the Kubernetes manifests.
•	--output-dir: Specifies the directory to save the generated bundle.
•	--package: The package name, which is usually the same as the project name.
•	--channels and --default-channel: The release channel(s) for the bundle (e.g., alpha for preview releases).

## Step 3: Validate the Generated Bundle Files

The generated files must adhere to the expected structure and format. Here’s an overview of the important files:
•	annotations.yaml: Contains metadata about the Operator, such as the author’s name, the operator’s source, and description.
•	ClusterServiceVersion.yaml (CSV): Defines the installation properties of the Operator, including deployment information and required resources.
•	bundle.Dockerfile: A Dockerfile used to build the image containing the Operator.

Example structure:

```
bundle
├── manifests
│   ├── app.mydomain.com_myapps.yaml
│   ├── kubebuilderdemo-controller-manager-metrics-service_v1_service.yaml
│   ├── kubebuilderdemo-metrics-reader_rbac.authorization.k8s.io_v1_clusterrole.yaml
│   ├── kubebuilderdemo-myapp-admin-role_rbac.authorization.k8s.io_v1_clusterrole.yaml
│   ├── kubebuilderdemo-myapp-editor-role_rbac.authorization.k8s.io_v1_clusterrole.yaml
│   ├── kubebuilderdemo-myapp-viewer-role_rbac.authorization.k8s.io_v1_clusterrole.yaml
│   └── my-operator.clusterserviceversion.yaml
└── metadata
    └── annotations.yaml
```

Use the following command to generate the bundle:

```
operator-sdk bundle validate ./bundle
```

You should see the following log:

```
INFO[0000] All validation tests have completed successfully
```

Note: Ensure the namespace and manager image defined in bundle/manifests/my-operator.clusterserviceversion.yaml are prepared in advance or adjusted according to your actual configuration.
build -t docker.io/raysail/kubebuilderdemo-manager:v1 -f Dockerfile

## Step 4: 验证 Deploying the Operator with OLM

### Step 1: Install OLM (if not installed)

If OLM is not yet installed in your Kubernetes cluster, you can install it with the following command:

```
operator-sdk olm install
```

### Step 2: Build and Push the Bundle Image

Once OLM is installed, build and push the bundle image:

```
docker build -t docker.io/raysail/kubebuilderdemo-operator-bundle:v1 -f bundle.Dockerfile
docker push docker.io/raysail/kubebuilderdemo-operator-bundle:v1
```

### Step 3: Run the Bundle

Run the following command to deploy the bundle:

```
operator-sdk run bundle docker.io/raysail/kubebuilderdemo-operator-bundle:v1
```

You should see logs similar to the following:

```
INFO[0035] Creating a File-Based Catalog of the bundle "docker.io/raysail/kubebuilderdemo-operator-bundle:v1"
INFO[0039] Generated a valid File-Based Catalog
INFO[0042] Created registry pod: docker-io-raysail-kubebuilderdemo-operator-bundle-v1
INFO[0042] Created CatalogSource: my-operator-catalog
INFO[0042] Created Subscription: my-operator-v0-1-0-sub
INFO[0060] Approved InstallPlan install-d7dbw for the Subscription: my-operator-v0-1-0-sub
INFO[0060] Waiting for ClusterServiceVersion "default/my-operator.v0.1.0" to reach 'Succeeded' phase
INFO[0061] Waiting for ClusterServiceVersion "default/my-operator.v0.1.0" to appear
INFO[0062] Found ClusterServiceVersion "default/my-operator.v0.1.0" phase: Pending
INFO[0064] Found ClusterServiceVersion "default/my-operator.v0.1.0" phase: Installing
INFO[0095] Found ClusterServiceVersion "default/my-operator.v0.1.0" phase: Succeeded
INFO[0095] OLM has successfully installed "my-operator.v0.1.0"

```
### Step 3: 进一步 Check Installation Status

To verify if the Operator has been successfully installed, you can run:

```
kubectl get operators
```

You can also check the deployment of the CR to see if there are any errors.

## Step 5: Publishing the Operator to operatorhub.io or olm.operatorframework.io

To make your Operator available to the public and other users, you can publish it to OperatorHub.io. Follow the steps below to submit your Operator.

### Step 1: Login to operatorhub.io or olm.operatorframework.io

Go to OperatorHub.io and create a developer account. Alternatively, if you’re submitting to OLM, log in to olm.operatorframework.io.

### Step 2: Submit Your Operator

Create a new Operator entry on OperatorHub.io or OLM, and provide the necessary metadata (name, description, etc.). Upload the generated bundle files, including annotations.yaml, ClusterServiceVersion.yaml, and bundle.Dockerfile.

### Step 3: Submit for Review

After uploading the files, submit your Operator for review by the OperatorHub.io team. Once approved, your Operator will be publicly available for installation by others directly from OperatorHub.io.

## Conclusion

By following this guide, you can easily distribute your Kubebuilder-based Operator to a wider audience. Whether you choose to publish your Operator on OperatorHub.io or deploy it via OLM, Kubebuilder provides the tools necessary to create a standardized bundle. Automating this process with Makefile further streamlines the deployment process, ensuring that your Operator is ready for production use in Kubernetes environments.