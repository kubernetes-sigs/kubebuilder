# Project Config

## Overview

The Project Config represents the configuration of a Kubebuilder project. All projects that are scaffolded with the CLI will generate the `PROJECT` file in the projects' root directory.

The PROJECT file was introduced since Kubebuilder [release 3.0.0](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v3.0.0), where the Plugins design was introduced. 
It tracks all data used to do the scaffolds. It stores more information about what resources and plugins are in use, to better enable plugins to make useful decisions when scaffolding ([see an example PROJECT file](https://github.com/kubernetes-sigs/kubebuilder/blob/6f1f8c43cfbf260c971037ff039cca86a0980006/testdata/project-v3-with-deploy-image/PROJECT#L2-L21)). 

## Versioning

The Project config is versioned according to its layout. For further information see [Versioning][versioning].

## Layout Definition

The `PROJECT` version `3` layout looks like:

```yaml
domain: testproject.org
layout:
- go.kubebuilder.io/v3
plugins:
  declarative.go.kubebuilder.io/v1:
    resources:
    - domain: testproject.org
      group: crew
      kind: FirstMate
      version: v1
projectName: example
repo: sigs.k8s.io/kubebuilder/example
resources:
- api:
    crdVersion: v1
    namespaced: true
  controller: true
  domain: testproject.org
  group: crew
  kind: Captain
  path: sigs.k8s.io/kubebuilder/example/api/v1
  version: v1
  webhooks:
    defaulting: true
    validation: true
    webhookVersion: v1
```

Now let's check its layout fields definition:

| Field | Description | 
|----------|-------------|
| `layout` | Defines the global plugins, e.g. a project `init` with `--plugins="go/v3,declarative"` means that any sub-command used will always call its implementation for both plugins in a chain. |  
| `domain` | Store the domain of the project. This information can be provided by the user when the project is generate with the `init` sub-command and the `domain` flag. |
| `plugins` | Defines the plugins used to do custom scaffolding, e.g. to use the optional `declarative` plugin to do scaffolding for just a specific api via the command `kubebuider create api [options] --plugins=declarative/v1`. |
| `projectName` | The name of the project. This will be used to scaffold the manager data. By default it is the name of the project directory, however, it can be provided by the user in the `init` sub-command via the `--project-name` flag. |
| `repo` |  The project repository which is the Golang module, e.g `github.com/example/myproject-operator`.  |
| `resources` |  An array of all resources which were scaffolded in the project. | 
| `resources.api` | The API scaffolded in the project via the sub-command `create api`. |
| `resources.api.crdVersion` | The Kubernetes API version (`apiVersion`) used to do the scaffolding for the CRD resource. |
| `resources.api.namespaced` | The API RBAC permissions which can be namespaced or cluster scoped. | 
| `resources.controller` | Indicates whether a controller was scaffolded for the API.  |
| `resources.domain` | The domain of the resource which is provided by the `--domain` flag when the sub-command `create api` is used. | 
| `resources.group` | The GKV group of the resource which is provided by the `--group` flag when the sub-command `create api` is used. |
| `resources.version` | The GKV version of the resource which is provided by the `--version` flag when the sub-command `create api` is used. |
| `resources.kind` | Store GKV Kind of the resource which is provided by the `--kind` flag when the sub-command `create api` is used. |
| `resources.path` | The import path for the API resource. It will be `<repo>/api/<kind>` unless the API added to the project is an external or core-type. For the core-types scenarios, the paths used are mapped [here][core-types]. |
| `resources.webhooks`| Store the webhooks data when the sub-command `create webhook` is used. |
| `resources.webhooks.webhookVersion` | The Kubernetes API version (`apiVersion`) used to scaffold the webhook resource. |
| `resources.webhooks.conversion` | It is `true` when the webhook was scaffold with the `--conversion` flag which means that is a conversion webhook. |
| `resources.webhooks.defaulting` | It is `true` when the webhook was scaffold with the `--defaulting` flag which means that is a defaulting webhook. |
| `resources.webhooks.validation` | It is `true` when the webhook was scaffold with the `--programmatic-validation` flag which means that is a validation webhook. |

[project]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/testdata/project-v3/PROJECT
[versioning]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/VERSIONING.md#Versioning
[core-types]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/plugins/golang/options.go