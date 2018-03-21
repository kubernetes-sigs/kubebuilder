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

# Build an image containing apiserver-builder for generating docs

FROM golang:1.10-stretch

ENV URL https://github.com/kubernetes-incubator/apiserver-builder/releases/download/v1.9-alpha.2
ENV BIN apiserver-builder-v1.9-alpha.2-linux-amd64.tar.gz
ENV DEST /usr/local/apiserver-builder/bin/
RUN curl -L $URL/$BIN -o /tmp/$BIN
RUN mkdir -p /usr/local/apiserver-builder
RUN tar -xzvf /tmp/$BIN -C /usr/local/apiserver-builder/

ENV PATH /usr/local/apiserver-builder/bin/:$PATH

RUN apt-get update
RUN apt-get install less -y
RUN apt-get install nano -yu

COPY docs.sh /usr/local/bin/docs.sh

CMD docs.sh