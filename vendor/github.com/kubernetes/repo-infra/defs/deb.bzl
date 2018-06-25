# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load("//defs:pkg.bzl", "pkg_tar")
load("@bazel_tools//tools/build_defs/pkg:pkg.bzl", "pkg_deb")

KUBERNETES_AUTHORS = "Kubernetes Authors <kubernetes-dev+release@googlegroups.com>"

KUBERNETES_HOMEPAGE = "http://kubernetes.io"

def k8s_deb(name, **kwargs):
    pkg_deb(
        name = name,
        architecture = "amd64",
        data = name + "-data",
        homepage = KUBERNETES_HOMEPAGE,
        maintainer = KUBERNETES_AUTHORS,
        package = name,
        **kwargs
    )

def deb_data(name, data = []):
    deps = []
    for i, info in enumerate(data):
        dname = "%s-deb-data-%s" % (name, i)
        deps += [dname]
        pkg_tar(
            name = dname,
            srcs = info["files"],
            mode = info["mode"],
            package_dir = info["dir"],
        )
    pkg_tar(
        name = name + "-data",
        deps = deps,
    )
