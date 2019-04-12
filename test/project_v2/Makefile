
# Image URL to use all building/pushing image targets
IMG ?= controller:latest

all: test manager

# Run tests
test: generate fmt vet manifests
	go test ./api/... ./controllers/... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crds
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
# TODO(droot): controller-gen will require fix to take new api-path as input, so disabling this for now
# 
#	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all

# Run go fmt against code
fmt:
	go fmt ./api/... ./controllers/...

# Run go vet against code
vet:
	go vet ./api/... ./controllers/...

# Generate code
generate:
ifndef GOPATH
	$(error GOPATH not defined, please define GOPATH. Run "go help gopath" to learn more about GOPATH)
endif
	go run ./vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go -O zz_generated.deepcopy --go-header-file ./hack/boilerplate.go.txt -i ./api/...
	

# Build the docker image
docker-build: test
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

# Push the docker image
docker-push:
	docker push ${IMG}
