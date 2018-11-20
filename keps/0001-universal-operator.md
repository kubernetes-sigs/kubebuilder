# Maestro: A Universal Operator

## Summary

We propose a universal operator that generates and runs operators based on a standard data format. This universal operator manages the lifecycle of stateful workloads.

## Motivation

While Kubebuilder provides an abstraction around the plumbing for building operators, there is still a lot of work to implement operators in code. To combat this problem, Mesosphere built an SDK for DC/OS, as well as a package data format, that handles a lot of the common high level tasks in running stateful workloads and provides lifecycle hooks. The purpose of Maestro is to take this work and enable the same standard packaging format and lifecycle hooks to be applied to Kubernetes operators.

## Goals

* Make it easy to deploy stateful operators on Kubernetes using a standard data format
* Enable re-use between DC/OS and Kubernetes SDK packages

## Development

Development will be located at [https://github.com/kubernetes-sigs/maestro](https://github.com/kubernetes-sigs/maestro).

## Design

The core of the Maestro Universal Operator is the "Framework" CRD. This CRD describes the implementation details of a framework, such as the deployment plan, components required, and other information needed to start a package. Initially, this Framework CRD's spec will be based on the [DC/OS Service SDK](https://mesosphere.github.io/dcos-commons/).

Creating a Framework triggers the operator to create package-specific CRDs. Users will create these CRDs, which are watched by Maestro to then create the service based on the Framework CRD's specification. Initially, only a few lifecycle hooks will be supported. The roadmap includes implementation of arbitrary plan specs, as well as the ability to create plan overriders with custom code. This will enable support for more complicated deployment lifecycles for managing software such as ZooKeeper or etcd clusters.

## Packaging Format

The packaging format will ultimately be created as the Framework CRD, but is initially defined by the DC/OS Service SDK specification. An example can be found [here](https://github.com/mesosphere/dcos-commons/tree/master/frameworks/kafka/universe).
