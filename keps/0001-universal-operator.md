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
