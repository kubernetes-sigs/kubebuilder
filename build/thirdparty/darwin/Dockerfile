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
# - kube-apiserver (build)
# - kubectl (fetch)
# - etcd (fetch)

FROM golang:1.19 as builder

# Version and platform args.
ARG KUBERNETES_VERSION
ARG ETCD_VERSION=v3.5.4
ARG OS=darwin
ARG ARCH

# Tools path.
ENV DEST=/usr/local/kubebuilder/bin/

# Install dependencies.
RUN apt update && \
    apt install unzip rsync -y && \
    mkdir -p $DEST

# kube-apiserver
WORKDIR /kubernetes
RUN git clone https://github.com/kubernetes/kubernetes . --depth=1 -b ${KUBERNETES_VERSION}
ENV CGO_ENABLED=0
ENV KUBE_BUILD_PLATFORMS=${OS}/${ARCH}
RUN make WHAT=cmd/kube-apiserver && \
    cp _output/local/bin/${KUBE_BUILD_PLATFORMS}/kube-apiserver $DEST

# kubectl
RUN /bin/bash -x -c ' \
    { curl -sLO https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}/bin/${OS}/${ARCH}/kubectl && \
      chmod +x kubectl && \
      cp kubectl $DEST; } || \
    { make WHAT=cmd/kubectl && \
      cp _output/local/bin/${KUBE_BUILD_PLATFORMS}/kubectl $DEST; }'

# etcd
WORKDIR /etcd
ENV ETCD_BASE_NAME=etcd-${ETCD_VERSION}-${OS}-${ARCH}
RUN /bin/bash -x -c ' \
    { curl -sLO https://github.com/coreos/etcd/releases/download/${ETCD_VERSION}/${ETCD_BASE_NAME}.zip && \
      unzip -o ${ETCD_BASE_NAME}.zip && \
      cp ${ETCD_BASE_NAME}/etcd $DEST; } || \
    { rm -fr * && \
      git clone https://github.com/etcd-io/etcd . --depth=1 -b ${ETCD_VERSION} && \
      GOOS=${OS} GOARCH=${ARCH} ./build.sh && \
      cp ./bin/etcd $DEST; }'

# Package into tarball.
WORKDIR /usr/local
RUN tar -czvf /kubebuilder_${OS}_${ARCH}.tar.gz kubebuilder/

# Copy tarball to final image.
FROM alpine:3.16

# Platform args.
ARG OS=darwin
ARG ARCH

COPY --from=builder /kubebuilder_${OS}_${ARCH}.tar.gz /
