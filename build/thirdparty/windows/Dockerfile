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

# Build or fetch the following binaries for windows and then host them in a tar.gz file in an alpine image
# - kube-apiserver (build)
# - kubectl (fetch)
# - etcd (fetch)

FROM golang:1.20 as builder

# Version and platform args.
ARG KUBERNETES_VERSION
ARG ETCD_VERSION=v3.5.10
ARG OS=windows
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
    cp _output/local/bin/${KUBE_BUILD_PLATFORMS}/kube-apiserver.exe $DEST

# kubectl
RUN /bin/bash -x -c ' \
    { curl -sLO https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}/bin/${OS}/${ARCH}/kubectl.exe && \
      chmod +x kubectl.exe &&  \
      cp kubectl.exe $DEST; } || \
    { make WHAT=cmd/kubectl && \
      cp _output/local/bin/${KUBE_BUILD_PLATFORMS}/kubectl.exe $DEST; }'

# etcd
WORKDIR /etcd
ENV ETCD_BASE_NAME=etcd-${ETCD_VERSION}-${OS}-${ARCH}
RUN /bin/bash -x -c ' \
    { curl -sLO https://github.com/coreos/etcd/releases/download/${ETCD_VERSION}/${ETCD_BASE_NAME}.zip && \
      unzip -o ${ETCD_BASE_NAME}.zip && \
      cp ${ETCD_BASE_NAME}/etcd.exe $DEST; } || \
    { rm -fr * && \
      git clone https://github.com/etcd-io/etcd . --depth=1 -b ${ETCD_VERSION} && \
      GOOS=${OS} GOARCH=${ARCH} ./build.sh && \
      cp ./bin/etcd.exe $DEST; }'

# Package into tarball.
WORKDIR /usr/local
RUN tar -czvf /kubebuilder_${OS}_${ARCH}.tar.gz kubebuilder/

# Copy tarball to final image.
FROM alpine:3.18

# Platform args.
ARG OS=windows
ARG ARCH

COPY --from=builder /kubebuilder_${OS}_${ARCH}.tar.gz /
