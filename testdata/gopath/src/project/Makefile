
# Image URL to use all building/pushing image targets
IMG ?= controller:latest

all: test manager

# Run tests
test: generate lint manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build manager binary
manager: generate lint
	go build -o bin/manager project/cmd/manager

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate lint
	go run ./cmd/manager/main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crds
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all

# Run golangci-lint against code
lint: $(GOPATH)/bin/golangci-lint
	golangci-lint run  -E gofmt ./pkg/... ./cmd/...

$(GOPATH)/bin/golangci-lint:
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

# Generate code
generate:
ifndef GOPATH
	$(error GOPATH not defined, please define GOPATH. Run "go help gopath" to learn more about GOPATH)
endif
	go generate ./pkg/... ./cmd/...

# Build the docker image
docker-build: test
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

# Push the docker image
docker-push:
	docker push ${IMG}
