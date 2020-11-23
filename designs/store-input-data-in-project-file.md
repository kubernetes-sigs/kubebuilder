---
title: Project file with all user input data
authors:
  - "@camilamacedo86"
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2020-11-23
last-updated: 2020-11-23
status: implementable
---

# Project file with all user input data

## Summary

The goal of this proposal is to ensure that we have all input data used to scaffold the projects properly stored in the PROJECT file. 

## Motivation

Keep the data used to scaffold the project to allow us to improve the tooling features as create in the future which would be able to help users to migrate their project to the upper plugin versions.  

## Proposal

Starts to store the missing information in the PROJECT file which are:
- if a resource and/or a controller was scaffold
- the webhook info used to scaffold the projects

## Use cases

I am as a maintainer would like to have stored all input data used to scaffold the projects by the user, so that it could help to develop the tooling features as to propose a subcommand/plugin to help users migrate from a plugin version X to X+1 since would be possible to scaffold the project from the scratch.

### Implementation Details/Notes/Constraints 

**To store the resource and controller data**

Currently, the GVK is only persistent on the PROJECT file if the resource is true. So, to allow all combinations shows that the best way is we store both booleans as use the api one to check if the api was or not scaffolded:

```yaml
resources:
...
- api: true
  controller: true
...
```

**NOTE** It is addressed in the Pull Request: https://github.com/kubernetes-sigs/kubebuilder/pull/1849

**To store the webhook data**

More than one webkook type can be scaffolded for the same GKV:

```yaml
...
- crdVersion: v1
  group: crew
  kind: FirstMate
  version: v1
  webhookVersion: v1
  webhooks:
    conversion: true
...
```

**NOTE** It is addressed in the Pull Request: https://github.com/kubernetes-sigs/kubebuilder/pull/1696

**Re-design and cleanups**

- Add a new attribute GKV to store the group, kind and version as a follow up of the above proposed changes, as suggested: 

```yaml 
- crdVersion: v1
  gkv:
    group: crew
    kind: FirstMate
    version: v1
  webhookVersion: v1
  webhooks:
    conversion: true
```

- Move the `webhookVersion` attribute to the webhooks type as suggested: 

```yaml 
- crdVersion: v1
  gkv:
    group: crew
    kind: FirstMate
    version: v1
  webhooks:
    conversion: true
    webhookVersion: v1
```
