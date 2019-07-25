## Prerequisites
CRD conversion webhook support was introduced as alpha feature in Kubernetes 1.13
release and has gone beta in Kubernetes 1.15 release. So ensure that you have a 
Kubernetes cluster that supports CRD conversion feature enabled.
Refer to [instructions](../TODO.txt) to enable CRD conversion feature in your
cluster.
Refer to [instructions](../reference/kind.md) to setup a local cluster with
Kind.

### What are we building ?
In this tutorial, we will implement a simple Disk API. Disk API has a field
called price that represents price per GB. We will go through three
iterations to evolve the price field specification.

- In v1 version of Disk API, price field is string with "AMOUNT CURRENCY" format.
  Example values could be "10 USD", "100 USD".
- In v2 version of Disk, price field is represented by a structure `Price`
  that has `amount` and `currency` as separate fields.
- In v3 version of Disk, we rename the price field to `pricePerGB` to make it
  more explicit.

Here are some sample manifests of the three versions representing same Disk
object.
```yaml

apiVersion: infra.kubebuilder.io/v1
kind: Disk
metadata:
  name: disk-sample
spec:
  price: 10 USD <--- price as string
----------------------------------------

apiVersion: infra.kubebuilder.io/v2
kind: Disk
metadata:
  name: disk-sample
spec: 
  price:  <---- price as structured object
    amount: 10
    currency: USD
----------------------------------------

apiVersion: infra.kubebuilder.io/v3
kind: Disk
metadata:
  name: disk-sample
spec:
  pricePerGB: <--- price is renamed to pricePerGB
    amount: 10
    currency: USD
```

## Tutorial
Now that we have covered the basics and the goal, we are all set to begin this
tutorial. We will go through the following steps:

- Project Setup
- Adding API with versions v1, v2, v3 of Disk API
- Setting up Webhooks
- CRD Generation
- Configuring Kustomization 
- Deploying and testing


### Project Setup
Assuming you have created a new directory and cd in to it. Let's initialize the project.
```bash

# Initialize Go module
go mod init infra.kubebuilder.io

# Initilize Kubebuilder project
kubebuilder init --domain kubebuilder.io

```


### Version v1

Let's create version `v1` of our `Disk` API.

```bash

# create v1 version with resource and controller
kubebuilder create api --group infra --kind Disk --version v1
Create Resource [y/n]
y
Create Controller [y/n]
y
```

Let's take a look at file `api/v1/disk_types.go`.

{{#literatego ./testdata/project/api/v1/disk_types.go}}

### Version v2

Let's add version `v2` to the `Disk` API. We will not add any controller this
time because we already have a controller for `Disk` API.

```bash
# create v2 version without controller
kubebuilder create api --group infra --kind Disk --version v2
Create Resource [y/n]
y
Create Controller [y/n]
n
```

Now, let's take a look at file `api/v2/disk_types.go`.

{{#literatego ./testdata/project/api/v2/disk_types.go}}

### Version v3

Let's add version `v3` to the `Disk` API and once again, we will not add any
controller since we already have controller for the `Disk` API.

```bash
# create v3 version without controller
kubebuilder create api --group infra --kind Disk --version v3
Create Resource [y/n]
y
Create Controller [y/n]
n

```

{{#literatego ./testdata/project/api/v3/disk_types.go}}

Now that we have all the API implementations in place, let's take a look at what
is required to setup conversion webhook for our `Disk` API.

### Setting up Webhooks
In `2.0.0+` release, Kubebuilder introduced new command `create webhook` to make
it easy to setup admission and conversion webhooks. Run the following command to
setup conversion webhook. Note that we can specify any version from v1, v2 or v3
in this command because there is single conversion webhook for a Kind.

```bash
kubebuilder create webhook --group infra --kind Disk --version v1 --conversion

Writing scaffold for you to edit...
api/v1/disk_webhook.go
```
Above commands does the following:
- Scaffolds a new file `api/v1/disk_webhook.go` to implement webhook setup method.
- Updates `main.go` to setup webhooks with the manager instance.

Let's take a quick look at the `api/v1/disk_webhook.go` file.

{{#literatego ./testdata/project/api/v1/disk_webhook.go}}

If you look at `main.go`, you will notice the following snippet that invokes the
SetupWebhook method.
```go
	.....

	if err = (&infrav1.Disk{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Disk")
		os.Exit(1)
	}

	....
```

### CRD Generation

The `controller-gen` tool that generates the CRD manifest takes a parameter to indicate if our API has multiple versions. We need to specify `trivialVersions=false` in CRD_OPTIONS in your project's Makefile to enable multi-version.

``` bash
...
CRD_OPTIONS ?= "crd:trivialVersions=false"
...
```

Run `make manifests` to ensure that CRD manifests gets generated under `config/crd/bases/` directory.

<details><summary>`infra.kubebuilder.io_disks.yaml`: the generated CRD YAML</summary>

```yaml
{{#include ./testdata/project/config/crd/bases/infra.kubebuilder.io_disks.yaml}}
```

</details>

### Manifests Generation

Kubebuilder generates Kubernetes manifests under 'config' directory with webhook
bits disabled. Follow the steps below to enable conversion webhook in manifests
generation.

- Ensure that `patches/webhook_in_<kind>.yaml` and `patches/cainjection_in_<kind>.yaml` are enabled in `config/crds/kustomization.yaml` file.
- Ensure that `../certmanager` and `../webhook` directories are enabled under `bases` section in `config/default/kustomization.yaml` file.
- Ensure that `manager_webhook_patch.yaml` is enabled under `patches` section in `config/default/kustomization.yaml` file.
- Enable all the vars under section `CERTMANAGER` in `config/default/kustomization.yaml` file.

