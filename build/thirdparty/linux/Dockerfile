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

# Fetch the following into binaries for linux and then host them in a tar.gz file in an alpine image
# - kube-apiserver (fetch)
# - kubectl (fetch)
# - etcd (fetch)

# no need for anything fancy
FROM alpine:3.8 as builder

# Version and platform args.
ARG KUBERNETES_VERSION
ARG ETCD_VERSION=v3.4.16
ARG OS=linux
ARG ARCH

# Tools path.
ENV DEST=/usr/local/kubebuilder/bin/

# Install dependencies.
RUN apk add --no-cache curl && \
    mkdir -p $DEST

# kube-apiserver
ENV K8S_SERVER_BASE_NAME=kubernetes-server-${OS}-${ARCH}
RUN curl -sLO https://dl.k8s.io/${KUBERNETES_VERSION}/${K8S_SERVER_BASE_NAME}.tar.gz && \
    tar xzf ${K8S_SERVER_BASE_NAME}.tar.gz && \
    cp kubernetes/server/bin/kube-apiserver $DEST

# kubectl
RUN curl -sLO https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}/bin/${OS}/${ARCH}/kubectl && \
    chmod +x kubectl && \
    cp kubectl $DEST

# etcd
ENV ETCD_BASE_NAME=etcd-${ETCD_VERSION}-${OS}-${ARCH}
RUN curl -sLO https://github.com/coreos/etcd/releases/download/${ETCD_VERSION}/${ETCD_BASE_NAME}.tar.gz && \
    tar xzf ${ETCD_BASE_NAME}.tar.gz && \
    cp ${ETCD_BASE_NAME}/etcd $DEST

# Package into tarball.
WORKDIR /usr/local
RUN tar -czvf /kubebuilder_${OS}_${ARCH}.tar.gz kubebuilder/

# Copy tarball to final image.
FROM alpine:3.8

# Platform args.
ARG OS=linux
ARG ARCH

COPY --from=builder /kubebuilder_${OS}_${ARCH}.tar.gz /
