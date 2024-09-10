# Project Upgrade Assistant

## Overview

Please note that all input utilized via the Kubebuilder tool is tracked in the PROJECT file ([example][example]).
This file is responsible for storing essential information, representing various facets of the Project such as its layout,
plugins, APIs, and more. ([More info][more-info]).

With the release of new plugin versions/layouts or even a new Kubebuilder CLI version with scaffold changes,
an easy way to upgrade your project is by re-scaffolding. This process allows users to employ tools like IDEs to compare
changes, enabling them to overlay their code implementation on the new scaffold or integrate these changes into their existing projects.

## When to use it ?

This command is useful when you want to upgrade an existing project to the latest version of the Kubebuilder project layout.
It makes it easier for the users to migrate their operator projects to the new scaffolding.

## How to use it ?

**To upgrade the scaffold of your project to use a new plugin version:**

```sh
kubebuilder alpha generate --plugins="pluginkey/version"
```

**To upgrade the scaffold of your project to get the latest changes:**

Currently, it supports two optional params, `input-dir` and `output-dir`.

`input-dir` is the path to the existing project that you want to re-scaffold. Default is the current working directory.

`output-dir` is the path to the directory where you want to generate the new project. Default is a subdirectory in the current working directory.

```sh
kubebuilder alpha generate --input-dir=/path/to/existing/project --output-dir=/path/to/new/project
```

<aside class="note warning">
<h1>Regarding `input-dir` and `output-dir`:</h1>

If neither `input-dir` nor `output-dir` are specified, the project will be regenerated in the current directory.
This approach facilitates comparison between your current local branch and the version stored upstream (e.g., GitHub main branch).
This way, you can easily overlay your project's code changes atop the new scaffold.

</aside>

## Further Resources:

- Check out [video to show how it works](https://youtu.be/7997RIbx8kw?si=ODYMud5lLycz7osp)
- See the [desing proposal documentation](../../../../designs/helper_to_upgrade_projects_by_rescaffolding.md)

[example]: ./../../../../testdata/project-v4-with-plugins/PROJECT
[more-info]: ./../reference/project-config.md