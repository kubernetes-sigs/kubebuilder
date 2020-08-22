# Makefile Helpers

By default, the projects are scaffolded with a `Makefile`. You can customize and update this file as please you. Here, you will find some helpers that can be useful. 

## To debug with go-delve

The projects are built with Go and you have a lot of ways to do that. One of the options would be use [go-delve](https://github.com/go-delve/delve) for it:

```sh
# Run with Delve for development purposes against the configured Kubernetes cluster in ~/.kube/config
# Delve is a debugger for the Go programming language. More info: https://github.com/go-delve/delve
run-delve: generate fmt vet manifests
    go build -gcflags "all=-trimpath=$(shell go env GOPATH)" -o bin/manager main.go
    dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./bin/manager
```

## To change the version of CRDs 

The tool generate the CRDs by using [controller-tools](https://github.com/kubernetes-sigs/controller-tools), see in the manifests target:

```sh
# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
```

In this way, update `CRD_OPTIONS` to define the version of the CRDs manifests which will be generated in the `config/crd/bases` directory:

```sh
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
```

|   CRD_OPTIONS	|   API version	|  
|---	|---	|
| `"crd:trivialVersions=true"` |  `apiextensions.k8s.io/v1beta1` |
| `"crd:crdVersions=v1"` | `apiextensions.k8s.io/v1`	|  