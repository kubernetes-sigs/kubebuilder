---
title: Neat-Enhancement-Idea
authors:
  - "@parul5sahoo"
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2022-04-11
last-updated: --
status: 
---

# New Plugin (`krpomz-metrics.go.kubebuilder.io/v1beta1`) to generate code

## Summary

This proposal defines a new plugin which allow users to generate the manifests required to provide Grafana dashboards for visualizing the default [metrics exported](https://book.kubebuilder.io/reference/metrics.html). 
 
## Motivation

The major chunk of Kubebuilder users build projects collaboratively having many contributors working on the same project. Given the current scenario, almost all operators built using Kubebuilder publish the same metrics with a few exceptions arising from the usage of some labels. So having a plugin to collect the controller-runtime metrics and export them for visualization and consequently having a dedicated Grafana dashboard in association with the plugin at the [Grafana marketplace](https://grafana.com/grafana/dashboards), would help maintain uniformity among the data collected and visualized by each and every member. And the plugin will not restrict users from customizing additional metric that they wish to export. This will not only prevent teams using Grafana from each having to build their own dashboards but also to fasten the process of visualizing operator metrics.

### Goals

- Add a new plugin to generate manifests that export the desired set of controller-runtime metrics.
- Promote the best practices as give example of common implementations and reduce the learning curve
- Fasten the process of exporting and visualizing controller runtime metrics. 
- Provide a dedicated dashboard for visualizing the controller-runtime metrics exported by the plugin on Grafana.
 
## Proposal

Add the new plugin to generate manifests which will scaffold the metrics and exports them to the Prometheus Opertor such as; `kubebuilder init --domain my.domain --repo my.domain/guestbook --plugins=kpromz-metrics.go.kubebuilder.io/v1beta1` which will also include the execution of:-
`kubebuilder create clusterrolebinding metrics --clusterrole=<namePreifx>-metrics-reader --serviceaccount=<namespace>:<service-account-name>`


### User Stories

- I am a user, I would like to use a command to scaffold my manifests and export the metrics collected from controller-runtime securely to Prometheus for visualization

- I am a user and I work in with a team of few members, we want to have uniform metrics exported from the controller-runtime of the cluster we are all working on and use the same Grafana dashboard for visualization. But being a global community with members residing in different time zones, connecting regulalry to check and maintain uniformity in our metrics visualization is inconvinient.  

- I am a maintainer,I would like to fasten the process of metrics collections and visualization to add new features and keep it updated with new Go releases.
 


## Design Details

### Test Plan

To ensure this implementation a new project example should be generated in the [testdata](../testdata/) directory of the project. See the [test/testadata/generate.sh](../test/testadata/generate.sh). Also, we should use this scaffold in the [integration tests](../test/e2e/) to ensure that the data scaffolded with works on the cluster as expected.

### Graduation Criteria

- The new plugin will support `project-version=3` 
- The attribute image with the value informed should be added to the resources model in the PROJECT file to let the tool know that the Resources need to get done with the common basic code implementation: 

```yaml
plugins:
    krpomz-metrics.go.kubebuilder.io/v1beta1:
        resources:
          - domain: example.io
            group: crew
            kind: Captain
            version: v1
            image: "<some-registry>/<project-name>:<tag>
``` 


