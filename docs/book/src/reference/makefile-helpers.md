# Makefile Helpers

By default, the projects are scaffolded with a `Makefile`. You can customize and update this file as please you. Here, you will find some helpers that can be useful.

## To debug with go-delve

The projects are built with Go and you have a lot of ways to do that. One of the options would be use [go-delve](https://github.com/go-delve/delve) for it:

```makefile
# Run with Delve for development purposes against the configured Kubernetes cluster in ~/.kube/config
# Delve is a debugger for the Go programming language. More info: https://github.com/go-delve/delve
run-delve: generate fmt vet manifests
    go build -gcflags "all=-trimpath=$(shell go env GOPATH)" -o bin/manager main.go
    dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./bin/manager
```

## To change the version of CRDs

The `controller-gen` program (from [controller-tools](https://github.com/kubernetes-sigs/controller-tools))
generates CRDs for kubebuilder projects, wrapped in the following `make` rule:

```makefile
manifests: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
```

`controller-gen` lets you specify what CRD API version to generate (either "v1", the default, or "v1beta1").
You can direct it to generate a specific version by adding `crd:crdVersions={<version>}` to your `CRD_OPTIONS`,
found at the top of your Makefile:

```makefile
CRD_OPTIONS ?= "crd:crdVersions={v1beta1},preserveUnknownFields=false"

manifests: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=manager-role $(CRD_OPTIONS) webhook paths="./..." output:crd:artifacts:config=config/crd/bases
```

## To get all the manifests without deploying

By adding `make dry-run` you can get the patched manifests in the dry-run folder, unlike `make depĺoy` which runs `kustomize` and `kubectl apply`.

To accomplish this, add the following lines to the Makefile:

```makefile
dry-run: manifests
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	mkdir -p dry-run
	$(KUSTOMIZE) build config/default > dry-run/manifests.yaml
```
