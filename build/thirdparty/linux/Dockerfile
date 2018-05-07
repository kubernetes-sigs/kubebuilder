#  Copyright 2018 The Kubernetes Authors.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

# Build the following into binaries for linux and then host them in a tar.gz file in an alpine image
# - apiserver
# - kubectl
# - kube-controller-manager
# - etcd
# - *-gen code generators
# - reference-docs

FROM golang:1.10-stretch as kubernetes-linux
# Install tools
RUN apt update
RUN apt install rsync -y
RUN go get github.com/jteeuwen/go-bindata/go-bindata
ENV CGO 0
ENV DEST /usr/local/kubebuilder/bin/
RUN mkdir -p $DEST || echo ""
RUN git clone https://github.com/kubernetes/kubernetes $GOPATH/src/k8s.io/kubernetes --depth=1 -b release-1.10
WORKDIR /go/src/k8s.io/kubernetes
ENV KUBE_BUILD_PLATFORMS linux/amd64
RUN make WHAT=cmd/kube-apiserver
RUN make WHAT=cmd/kube-controller-manager
RUN make WHAT=cmd/kubectl
RUN cp _output/local/bin/$KUBE_BUILD_PLATFORMS/kube-apiserver $DEST
RUN cp _output/local/bin/$KUBE_BUILD_PLATFORMS/kube-controller-manager $DEST
RUN cp _output/local/bin/$KUBE_BUILD_PLATFORMS/kubectl $DEST

FROM golang:1.10-stretch as etcd-linux
ENV CGO 0
ENV GOOS linux
ENV GOARCH amd64
ENV DEST=/usr/local/kubebuilder/bin/
RUN mkdir -p $DEST || echo ""
RUN git clone https://github.com/coreos/etcd $GOPATH/src/github.com/coreos/etcd --depth=1
RUN go build -o $DEST/etcd github.com/coreos/etcd

FROM golang:1.10-stretch as code-generator-linux
ENV CGO 0
ENV GOOS linux
ENV GOARCH amd64
ENV DEST /usr/local/kubebuilder/bin/
RUN mkdir -p $DEST || echo ""
RUN git clone https://github.com/kubernetes/code-generator $GOPATH/src/k8s.io/code-generator --depth=1 -b release-1.10
RUN go build -o $DEST/client-gen k8s.io/code-generator/cmd/client-gen
RUN go build -o $DEST/conversion-gen k8s.io/code-generator/cmd/conversion-gen
RUN go build -o $DEST/deepcopy-gen k8s.io/code-generator/cmd/deepcopy-gen
RUN go build -o $DEST/defaulter-gen k8s.io/code-generator/cmd/defaulter-gen
RUN go build -o $DEST/informer-gen k8s.io/code-generator/cmd/informer-gen
RUN go build -o $DEST/lister-gen k8s.io/code-generator/cmd/lister-gen
RUN go build -o $DEST/openapi-gen k8s.io/code-generator/cmd/openapi-gen

FROM golang:1.10-stretch as reference-docs-linux
ENV CGO 0
ENV GOOS linux
ENV GOARCH amd64
ENV DEST /usr/local/kubebuilder/bin/
RUN mkdir -p $DEST || echo ""
RUN git clone https://github.com/kubernetes-incubator/reference-docs $GOPATH/src/github.com/kubernetes-incubator/reference-docs --branch kubebuilder --depth=1
RUN go build -o $DEST/gen-apidocs github.com/kubernetes-incubator/reference-docs/gen-apidocs

FROM golang:1.10-stretch as linux
RUN mkdir -p /usr/local/kubebuilder/bin/
COPY --from=etcd-linux /usr/local/kubebuilder/bin/* /usr/local/kubebuilder/bin/
COPY --from=kubernetes-linux /usr/local/kubebuilder/bin/* /usr/local/kubebuilder/bin/
COPY --from=code-generator-linux /usr/local/kubebuilder/bin/* /usr/local/kubebuilder/bin/
COPY --from=reference-docs-linux /usr/local/kubebuilder/bin/* /usr/local/kubebuilder/bin/
WORKDIR /usr/local
RUN tar -czvf /kubebuilder_linux_amd64.tar.gz kubebuilder/

FROM alpine:3.7
COPY --from=linux /kubebuilder_linux_amd64.tar.gz /kubebuilder_linux_amd64.tar.gz