# Rescaffold Command

## Overview

The Kubebuilder CLI provides a new experimental helper `alpha generate` command to re-scaffold an existing project from the scratch using the current version of KubeBuilder binary available based on PROJECT config file. 

## When to use it ?

This command is useful when you want to upgrade an existing project to the latest version of the Kubebuilder project layout. It makes it easier for the users to migrate their operator projects to the new scaffolding.

## How to use it ?

Currently, it supports two optional params, `input-dir` and `output-dir`. 

`input-dir` is the path to the existing project that you want to re-scaffold. Default is the current working directory.

`output-dir` is the path to the directory where you want to generate the new project. Default is a subdirectory in the current working directory.

```sh
kubebuilder alpha generate --input-dir=/path/to/existing/project --output-dir=/path/to/new/project
```