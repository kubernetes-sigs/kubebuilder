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
# - apiserver (fetch)
# - kubectl (fetch)
# - etcd (fetch)

# no need for anything fancy
FROM alpine:3.8
# Install tools
ENV DEST /usr/local/kubebuilder/bin/

RUN apk add --no-cache curl && \
    mkdir -p $DEST || echo ""
 
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.14.1/bin/linux/amd64/kubectl && \
    chmod +x kubectl && \
    cp kubectl $DEST

RUN curl -LO https://dl.k8s.io/v1.14.1/kubernetes-server-linux-amd64.tar.gz && \
    tar xzf kubernetes-server-linux-amd64.tar.gz && \
    cp kubernetes/server/bin/kube-apiserver $DEST

ENV ETCD_VERSION="3.3.11"
ENV ETCD_DOWNLOAD_FILE="etcd-v${ETCD_VERSION}-linux-amd64.tar.gz"
RUN curl -LO https://github.com/coreos/etcd/releases/download/v${ETCD_VERSION}/etcd-v${ETCD_VERSION}-linux-amd64.tar.gz -o ${ETCD_DOWNLOAD_FILE} && \
    tar xzf ${ETCD_DOWNLOAD_FILE} && \
    cp etcd-v${ETCD_VERSION}-linux-amd64/etcd $DEST

WORKDIR /usr/local
RUN tar -czvf /kubebuilder_linux_amd64.tar.gz kubebuilder/

FROM scratch
COPY --from=linux /kubebuilder_linux_amd64.tar.gz /kubebuilder_linux_amd64.tar.gz
