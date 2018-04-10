/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package initproject

import (
	"path/filepath"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
)

func doDockerfile() bool {
	args := templateArgs{
		Repo: util.Repo,
	}
	//install := util.WriteIfNotFound(filepath.Join("Dockerfile.install"), "install-docker-template", installDockerfileTemplate, args)
	//docs := util.WriteIfNotFound(filepath.Join(dir, "Dockerfile.docs"), "docs-docker-template", docsDockerfileTemplate, args)
	controller := util.WriteIfNotFound(filepath.Join("Dockerfile.controller"), "controller-docker-template", controllerDockerfileTemplate, args)
	//apiserver := util.WriteIfNotFound(filepath.Join(dir, "Dockerfile.apiserver"), "apiserver-docker-template", apiserverDockerfileTemplate, args)
	//pod := util.WriteIfNotFound(filepath.Join("hack", "install.yaml"), "install-template", installPodTemplate, args)
	return controller
}

var installDockerfileTemplate = `# Instructions to install API using the installer
# Create a serviceaccount with the cluster-admin role
# $ kubectl create serviceaccount installer
# $ kubectl create clusterrolebinding installer-cluster-admin-binding --clusterrole=cluster-admin --serviceaccount=default:installer
# RunInformersAndControllers the installer image in the cluster as the cluster-admin
# $ kubectl run <name> --serviceaccount=installer --image=<install-image> --restart=OnFailure -- ./installer --controller-image=<controller-image> --docs-image=<docs-image> --name=<installation-name>

# To run the install outside of the cluster, you must give your account the cluster-admin role
# kubectl create clusterrolebinding <user>-cluster-admin-binding --clusterrole=cluster-admin --user=<user>

# Build and test the controller-manager
FROM golang:1.9.3 as builder
WORKDIR /go/src/{{ .Repo }}
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/
RUN go build -a -o installer ./cmd/installer/main.go

# Copy the controller-manager into a thin image
FROM ubuntu:latest  
RUN apt update && apt install openssl -y && apt clean && rm -rf /var/lib/apt/lists/*
WORKDIR /root/
COPY --from=builder /go/src/{{ .Repo }}/installer .
CMD ["./installer"]  
`

var controllerDockerfileTemplate = `# Instructions to install API using the installer
# Build and test the controller-manager
FROM golang:1.9.3 as builder

ENV TEST_ASSET_DIR /usr/local/bin
ENV TEST_ASSET_KUBECTL $TEST_ASSET_DIR/kubectl
ENV TEST_ASSET_KUBE_APISERVER $TEST_ASSET_DIR/kube-apiserver
ENV TEST_ASSET_ETCD $TEST_ASSET_DIR/etcd

# Download test framework binaries
ENV TEST_ASSET_URL https://storage.googleapis.com/k8s-c10s-test-binaries
RUN curl ${TEST_ASSET_URL}/etcd-Linux-x86_64 --output $TEST_ASSET_ETCD
RUN curl ${TEST_ASSET_URL}/kube-apiserver-Linux-x86_64 --output $TEST_ASSET_KUBE_APISERVER
RUN curl https://storage.googleapis.com/kubernetes-release/release/v1.9.2/bin/linux/amd64/kubectl --output $TEST_ASSET_KUBECTL
RUN chmod +x $TEST_ASSET_ETCD
RUN chmod +x $TEST_ASSET_KUBE_APISERVER
RUN chmod +x $TEST_ASSET_KUBECTL

# Copy in the go src
WORKDIR /go/src/{{ .Repo }}
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

# Build and test the API code
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o controller-manager ./cmd/controller-manager/main.go
RUN go test ./pkg/... ./cmd/...

# Copy the controller-manager into a thin image
FROM scratch
# RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/{{ .Repo }}/controller-manager .
ENTRYPOINT ["./controller-manager"]
CMD ["--install-crds=false"]
`

//var apiserverDockerfileTemplate = `# Instructions to install API using the installer
//# IGNORE THIS FILE IF YOU ARE USING CRDS.
//# THIS IS ONLY FOR APISERVER AGGREGATION.
//# Build the apiserver
//FROM golang:1.9.3 as builder
//
//# Copy in the go src
//WORKDIR /go/src/{{ .Repo }}
//COPY pkg/    pkg/
//COPY cmd/    cmd/
//COPY vendor/ vendor/
//
//# Build and test the API code
//RUN go build -a -o apiserver ./cmd/apiserver/main.go
//
//# Copy the apiserver into a thin image
//FROM ubuntu:latest
//WORKDIR /root/
//COPY --from=builder /go/src/{{ .Repo }}/apiserver .
//CMD ["./apiserver"]
//`
//
//var docsDockerfileTemplate = `
//# Builds a container to host the reference documentation for the APIs
//# To access documentation in the cluster on http://localhost:8989 run
//# kubectl port-forward  $(kubectl get pods --namespace=foo-system -l="app=docs" -o="jsonpath={.items[0].metadata.name}") 8989:80 --namespace=<install-name>-system
//FROM golang:1.9.3 as builder
//
//WORKDIR /go/src/{{ .Repo }}
//RUN go get github.com/kubernetes-incubator/reference-docs/gen-apidocs
//
//COPY pkg/    pkg/
//COPY cmd/    cmd/
//COPY vendor/ vendor/
//COPY docs/ docs/
//
//
//RUN mkdir docs/openapi-spec || echo "openapi-spec dir exists"
//RUN mkdir docs/static_includes || echo "static_includes dir exists"
//RUN go run ./cmd/apiserver/main.go --etcd-servers=http://localhost:2379 --secure-port=9443 --print-openapi --delegated-auth=false > docs/openapi-spec/swagger.json
//RUN gen-apidocs --build-operations=false --use-tags=true --allow-errors=true --config-dir=docs
//
//# RunInformersAndControllers brodocs against docs set
//FROM pwittrock/brodocs:latest as brodocs
//COPY --from=builder /go/src/{{ .Repo }}/docs docs/
//RUN mkdir /manifest
//RUN mkdir /build
//RUN cp docs/manifest.json /manifest/manifest.json
//RUN mv docs/includes /source
//RUN ./runbrodocs.sh
//
//# Publish docs in a container
//FROM nginx
//COPY --from=brodocs build/ /usr/share/nginx/html
//`

var installPodTemplate = `
# EDIT ME by replacing "<project>/<apis-name>" with your image
# Steps to install
# kubectl create serviceaccount installer
# kubectl create clusterrolebinding installer-cluster-admin-binding --clusterrole=cluster-admin --serviceaccount=default:installer
# kubectl create -f install.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: <apis-name>
  labels:
    run: <apis-name>
spec:
  template:
    metadata:
      labels:
        run: <apis-name>
    spec:
      restartPolicy: OnFailure
      serviceAccountName: installer
      containers:
      - args:
        - ./installer
        - --controller-image=gcr.io/<project>/<apis-name>-controller:v1
        - --docs-image=gcr.io/<project>/<apis-name>-docs:v1
        - --name=<apis-name>
        image: gcr.io/<project>/<apis-name>-install:v1
        imagePullPolicy: Always
        name: <apis-name>
`
