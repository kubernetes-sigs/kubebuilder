# Platforms Supported

Kubebuilder produces solutions that by default can work on multiple platforms or specific ones, depending on how you
build and configure your workloads. This guide aims to help you properly configure your projects according to your needs.

## Overview

To provide support on specific or multiple platforms, you must ensure that all images used in workloads are built to
support the desired platforms. Note that they may not be the same as the platform where you develop your solutions and use KubeBuilder, but instead the platform(s) where your solution should run and be distributed.
It is recommended to build solutions that work on multiple platforms so that your project works
on any Kubernetes cluster regardless of the underlying operating system and architecture.

## How to define which platforms are supported

The following covers what you need to do to provide the support for one or more platforms or architectures.

### 1) Build workload images to provide the support for other platform(s)

The images used in workloads such as in your Pods/Deployments will need to provide the support for this other platform.
You can inspect the images using a ManifestList of supported platforms using the command
[docker manifest inspect <image>][docker-manifest], i.e.:

```shell
$ docker manifest inspect myresgystry/example/myimage:v0.0.1
{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
   "manifests": [
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 739,
         "digest": "sha256:a274a1a2af811a1daf3fd6b48ff3d08feb757c2c3f3e98c59c7f85e550a99a32",
         "platform": {
            "architecture": "arm64",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 739,
         "digest": "sha256:d801c41875f12ffd8211fffef2b3a3d1a301d99f149488d31f245676fa8bc5d9",
         "platform": {
            "architecture": "amd64",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 739,
         "digest": "sha256:f4423c8667edb5372fb0eafb6ec599bae8212e75b87f67da3286f0291b4c8732",
         "platform": {
            "architecture": "s390x",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 739,
         "digest": "sha256:621288f6573c012d7cf6642f6d9ab20dbaa35de3be6ac2c7a718257ec3aff333",
         "platform": {
            "architecture": "ppc64le",
            "os": "linux"
         }
      },
   ]
}
```

### 2) (Recommended as a Best Practice) Ensure that node affinity expressions are set to match the supported platforms

Kubernetes provides a mechanism called [nodeAffinity][node-affinity] which can be used to limit the possible node
targets where a pod can be scheduled. This is especially important to ensure correct scheduling behavior in clusters
with nodes that span across multiple platforms (i.e. heterogeneous clusters).

**Kubernetes manifest example**

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/arch
          operator: In
          values:
          - amd64
          - arm64
          - ppc64le
          - s390x
        - key: kubernetes.io/os
            operator: In
            values:
              - linux
```

**Golang Example**

```go
Template: corev1.PodTemplateSpec{
    ...
    Spec: corev1.PodSpec{
        Affinity: &corev1.Affinity{
            NodeAffinity: &corev1.NodeAffinity{
                RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
                    NodeSelectorTerms: []corev1.NodeSelectorTerm{
                        {
                            MatchExpressions: []corev1.NodeSelectorRequirement{
                                {
                                    Key:      "kubernetes.io/arch",
                                    Operator: "In",
                                    Values:   []string{"amd64"},
                                },
                                {
                                    Key:      "kubernetes.io/os",
                                    Operator: "In",
                                    Values:   []string{"linux"},
                                },
                            },
                        },
                    },
                },
            },
        },
        SecurityContext: &corev1.PodSecurityContext{
            ...
        },
        Containers: []corev1.Container{{
            ...
        }},
    },
```

<aside class="note">
<h1> Example(s) </h1>

You can look for some code examples by checking the code which is generated via the Deploy
Image plugin. ([More info](../plugins/deploy-image-plugin-v1-alpha.md))

</aside>

## Producing projects that support multiple platforms

You can use [`docker buildx`][buildx] to cross-compile via emulation ([QEMU](https://www.qemu.org/)) to build the manager image.
See that projects scaffold with the latest versions of Kubebuilder have the Makefile target `docker-buildx`.

**Example of Usage**

```shell
$ make docker-buildx IMG=myregistry/myoperator:v0.0.1
```

Note that you need to ensure that all images and workloads required and used by your project will provide the same
support as recommended above, and that you properly configure the [nodeAffinity][node-affinity] for all your workloads.
Therefore, ensure that you uncomment the following code in the `config/manager/manager.yaml` file

```yaml
      # TODO(user): Uncomment the following code to configure the nodeAffinity expression
      # according to the platforms which are supported by your solution.
      # It is considered best practice to support multiple architectures. You can
      # build your manager image using the makefile target docker-buildx.
      # affinity:
      #   nodeAffinity:
      #     requiredDuringSchedulingIgnoredDuringExecution:
      #       nodeSelectorTerms:
      #         - matchExpressions:
      #           - key: kubernetes.io/arch
      #             operator: In
      #             values:
      #               - amd64
      #               - arm64
      #               - ppc64le
      #               - s390x
      #           - key: kubernetes.io/os
      #             operator: In
      #             values:
      #               - linux
```

<aside class="note">
<h1>Building images for releases</h1>


You will probably want to automate the releases of your projects to ensure that the images are always built for the
same platforms. Note that Goreleaser also supports [docker buildx][buildx]. See its [documentation][goreleaser-buildx] for more detail.

Also, you may want to configure GitHub Actions, Prow jobs, or any other solution that you use to build images to
provide multi-platform support. Note that you can also use other options like `docker manifest create` to customize
your solutions to achieve the same goals with other tools.

By using Docker and the target provided by default you should NOT change the Dockerfile to use
any specific GOOS and GOARCH to build the manager binary. However, if you are looking for to
customize the default scaffold and create your own implementations you might want to give
a look in the Golang [doc](https://go.dev/doc/install/source#environment) to knows
its available options.

</aside>

## Which (workload) images are created by default?

Projects created with the Kubebuilder CLI have two workloads which are:

### Manager

The container to run the manager implementation is configured in the `config/manager/manager.yaml` file.
This image is built with the Dockerfile file scaffolded by default and contains the binary of the project \
which will be built via the command `go build -a -o manager main.go`.

Note that when you run `make docker-build` OR `make docker-build IMG=myregistry/myprojectname:<tag>`
an image will be built from the client host (local environment) and produce an image for
the client os/arch, which is commonly linux/amd64 or linux/arm64.

<aside class="note">
<h1>Mac Os</h1>

If you are running from an Mac Os environment then, Docker also will consider it as linux/$arch. Be aware that
when, for example, is running Kind on a Mac OS operational system the nodes will
end up labeled with ` kubernetes.io/os=linux`

</aside>

[node-affinity]: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity
[docker-manifest]: https://docs.docker.com/engine/reference/commandline/manifest/
[buildx]: https://docs.docker.com/build/buildx/
[goreleaser-buildx]: https://goreleaser.com/customization/docker/#use-a-specific-builder-with-docker-buildx