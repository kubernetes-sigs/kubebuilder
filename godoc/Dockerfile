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

# Create
# export V=10 && \
# docker build . -f godoc/Dockerfile -t <user>/<pr-number>:v${V} && \
# docker push <user>/<pr-number>:v${V} && \
# kubectl run <pr-number> --image <user>/<pr-number>:v${V} && \
# kubectl expose deployment <pr-number> --type LoadBalancer --port=80 --target-port=6060

# Update
# export V=10 && \
# docker build . -f godoc/Dockerfile -t <user>/<pr-number>:v${V} && \
# docker push <user>/<pr-number>:v${V} && \
# kubectl set image deployment <pr-number> *=<user>/<pr-number>:v${V}

FROM golang:stretch

RUN groupadd -g 999 godocuser && \
    useradd -r -u 999 -g godocuser godocuser
RUN chown godocuser src
USER godocuser

RUN mkdir -p src/github.com/kubernetes-sigs/kubebuilder
COPY pkg/ src/github.com/kubernetes-sigs/kubebuilder/pkg/

WORKDIR src/github.com/kubernetes-sigs/kubebuilder
CMD godoc -http=:6060