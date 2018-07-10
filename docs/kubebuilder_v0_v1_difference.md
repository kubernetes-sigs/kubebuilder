# Kubebuilder v0 v.s. v1

## command difference
  - kubebuilder v0 has `init`, `create controller`, `create resouece`, `create config`, `generate` commands and the workflow is

```
  kubebuilder init --domain example.com
  kubebuilder create resource --group <group> --version <version> --kind <Kind>
  GOBIN=${PWD}/bin go install ${PWD#$GOPATH/src/}/cmd/controller-manager
  bin/controller-manager --kubeconfig ~/.kube/config

  kubectl apply -f hack/sample/<resource>.yaml
  docker build -f Dockerfile.controller . -t <image:tag>
  docker push <image:tag>
  kubebuilder create config --controller-image <image:tag> --name <project-name>
  kubectl apply -f hack/install.yaml
```
  Everytime the resource or controller is updated, users need to run `kubebuilder generate` to regenerate the project.
  - kubebuilder v1 has `init`, `create api` commands and the workflow is
  
```
  kubebuilder init --domain example.com --license apache2 --owner "The Kubernetes authors"
  kubebuilder create api --group ship --version v1beta1 --kind Frigate
  make install
  make run
```
  In v1 project, there is no generate command. When the resource or controller is updated, users don't need to regenerate the project.
  
## scaffolding difference
  Init a project with example.com and add a resource/api with group apps, version v1, kind Hello in both v0 and v1. The scaffolded projects are as follows.
  - v0 project
```
.
├── cmd
│   └── controller-manager
│       └── main.go
├── Dockerfile.controller
├── Gopkg.lock
├── Gopkg.toml
├── hack
│   ├── boilerplate.go.txt
│   ├── doc.go
│   ├── imports.go
│   └── sample
│       └── hello.yaml
├── pkg
│   ├── apis
│   │   ├── apps
│   │   │   ├── doc.go
│   │   │   └── v1
│   │   └── doc.go
│   ├── client
│   │   ├── clientset
│   │   │   └── versioned
│   │   ├── informers
│   │   │   └── externalversions
│   │   └── listers
│   │       └── apps
│   ├── controller
│   │   ├── doc.go
│   │   └── hello
│   │       ├── controller.go
│   │       ├── controller_test.go
│   │       └── hello_suite_test.go
│   ├── doc.go
│   └── inject
│       ├── args
│       │   └── args.go
│       ├── doc.go
│       ├── inject.go
│       └── zz_generated.kubebuilder.go
└── vendor

```
  - v1 project
```
├── cmd
│   └── manager
│       └── main.go
├── config
│   ├── crds
│   │   └── apps_v1_hello.yaml
│   └── manager
│       ├── apps_rolebinding_rbac.yaml
│       ├── apps_role_rbac.yaml
│       └── manager.yaml
├── Dockerfile
├── Gopkg.toml
├── hack
│   └── boilerplate.go.txt
├── Makefile
├── pkg
│   ├── apis
│   │   ├── addtoscheme_apps_v1.go
│   │   ├── apis.go
│   │   └── apps
│   │       ├── group.go
│   │       └── v1
│   │           ├── doc.go
│   │           ├── hello_types.go
│   │           ├── hello_types_test.go
│   │           ├── register.go
│   │           └── v1_suite_test.go
│   └── controller
│       ├── add_hello.go
│       ├── controller.go
│       └── hello
│           ├── hello_controller.go
│           ├── hello_controller_suite_test.go
│           └── hello_controller_test.go
├── PROJECT
└── vendor
```
Compared with v0 project, there is no `client`, `inject` folders.
  
## library difference
  
  - v0 projects import the libraries from kubebuilder, for example kubebuilder/pkg/controller. It provides a `GenericController` type with a list of functions. Note that for created resources, the corresponding client library is generated under the project folder `pkg/client`
  
  - v1 projects import the libraries from controller-runtime, for example controller-runtime/pkg/controller, controller-runtime/pkg/client, controller-runtime/pkg/reconcile. Note that for created resources or core types, the client library is provided by controller-runtime.
  
## wiring difference
  - v0 projects has a `inject` package and it provides functions for adding the controller to controller-manager as well as registering CRDs.
  - v1 projects doesn't have a `inject` package, the controller is added to controller-manager by a `init` function inside add_<type>.go file inside the controller directory. The types is registered by a `init` function inside <type>_types.go file inside the apis directory.