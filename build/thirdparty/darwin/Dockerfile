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

# Build or fetch the following binaries for darwin and then host them in a tar.gz file in an alpine image
# - apiserver (build)
# - kubectl (fetch)
# - etcd (fetch)

FROM golang:1.11.2-stretch as darwin
# Install tools
RUN apt update
RUN apt install rsync -y
RUN apt-get install unzip
RUN go get github.com/jteeuwen/go-bindata/go-bindata
ENV CGO 0
ENV DEST /usr/local/kubebuilder/bin/
RUN mkdir -p $DEST || echo ""
RUN git clone https://github.com/kubernetes/kubernetes $GOPATH/src/k8s.io/kubernetes --depth=1 -b v1.13.5
WORKDIR /go/src/k8s.io/kubernetes

# Build for linux first otherwise it won't work for darwin - :(
ENV KUBE_BUILD_PLATFORMS linux/amd64
RUN make WHAT=cmd/kube-apiserver
ENV KUBE_BUILD_PLATFORMS darwin/amd64
RUN make WHAT=cmd/kube-apiserver
RUN cp _output/local/bin/$KUBE_BUILD_PLATFORMS/kube-apiserver $DEST

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.13.5/bin/darwin/amd64/kubectl
RUN chmod +x kubectl
RUN cp kubectl $DEST

ENV ETCD_VERSION="3.3.11"
ENV ETCD_DOWNLOAD_FILE="etcd-v${ETCD_VERSION}-darwin-amd64.zip"
RUN curl -LO https://github.com/coreos/etcd/releases/download/v${ETCD_VERSION}/etcd-v${ETCD_VERSION}-darwin-amd64.zip -o ${ETCD_DOWNLOAD_FILE}
RUN unzip -o ${ETCD_DOWNLOAD_FILE}
RUN cp etcd-v${ETCD_VERSION}-darwin-amd64/etcd $DEST

WORKDIR /usr/local
RUN tar -czvf /kubebuilder_darwin_amd64.tar.gz kubebuilder/

# Host the tar.gz file in a thin image
FROM alpine:3.7
COPY --from=darwin /kubebuilder_darwin_amd64.tar.gz /kubebuilder_darwin_amd64.tar.gz
