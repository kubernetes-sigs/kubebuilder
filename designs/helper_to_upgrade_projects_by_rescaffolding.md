| Authors                            | Creation Date | Status      | Extra |
|------------------------------------|---------------|-------------|---|
| @camilamacedo86,@Kavinjsir,@varshaprasad96 | Feb, 2023     | Implementeble | - |

Experimental Helper to upgrade projects by re-scaffolding
===================

This proposal amis to provide a new alpha command with a helper which
would be able to re-scaffold the project from the scratch based on
the [PROJECT config][project-config].

## Example

By running a command like following users would be able to re-scaffold the whole project from the scratch using the
current version of KubeBuilder binary available.

```shell
kubebuilder alpha generate [OPTIONS]
```

## Open Questions

N/A

## Summary

The [PROJECT config][project-config] tracks all configuration and inputs used to create the
project with the CLI. Therefore, a command could be able to read this information and re-run
the same commands with the same data used previously to re-generate the project.

## Motivation

The recommended straightforward steps to upgrade the projects are re-scaffold all from scract using the upper
version/plugin and then, re-add all code source on top again. Therefore, this command would help users in
order to minimize the required effort by automating which shows technically possible of this process.

The main motivation of this proposal is to provide a helper for upgrades and
make less painful this process. Examples:

- See the discussion [How to regenerate scaffolding?](https://github.com/kubernetes-sigs/kubebuilder/discussions/2864)
- From [slack channel By Paul Laffitte](https://kubernetes.slack.com/archives/CAR30FCJZ/p1675166014762669)

### Goals

- Help users upgrade their project with the latest changes
- Help users to re-scaffold the projects from the scratch based on what was done previously with the tool
- Make less painful the process to upgrade

### Non-Goals

- Change the default layout or how the KubeBuilder CLI works
- Deal with customizations or deviations from the proposed layout
- Be able to perform the project upgrade to the latest changes without human bean interactions
- Deal and support external plugins
- Provides support to [declarative](https://book.kubebuilder.io/plugins/declarative-v1.html) plugin
  since it is desired and planned to decouple this solution and donate this plugin to its own authors [More info](https://github.com/kubernetes-sigs/kubebuilder/issues/3186)

## Proposal

The proposed solution to achieve this goal is to create an alpha command as described
in the example section above, see:

```shell
kubebuilder alpha generate \
    --from-project-file=<path> 
    --to=<path where the project should be re-scaffold> 
    --backup=<path-where the current version of the project should be copied as backup>
    --init-plugins=<chain of plugins key that can be used to create the layout with init sub-command>
```

Where:

- project-file: If not informed then, the command would check it in the current directory
- output: If not informed then, it should be the current repository
- backup: if not informed then, it would be the current path -1 level with the name backup
- init-plugins: if not informed then, it is the same plugin chain available in the layout field

This command would mainly perform the following operations:

- 1. Make a backup of the current project
- 2. Ensure that the output path is clean
- 3. Read the [PROJECT config][project-config]
- 4. Re-run all commands using the KubeBuilder binary to recreate the project in the output directory

### User Stories

- As a developer I can regenerate my project from the scratch based on all commands that I used the tool to build
  my project previously, so that I can easily upgrade my current project to new CLI/plugin versions and get the
  latest changes, bug fixes and features
- As a developer I can regenerate my project from the scratch based on all commands that I used the tool to build
  my project previously but informing a new init plugin chain, so that I could upgrade my current project to new
  layout versions and experiment alpha ones.
- As a KubeBuilder maintainer, I can leverage on this helper to encourage its users to migrate to upper versions more often, making it easier to maintain the project.

### Implementation Details/Notes/Constraints

Note that in the [e2e tests](https://github.com/kubernetes-sigs/kubebuilder/tree/master/test/e2e) the binary is used to do the scaffolds.
Also, very similar to the implementation that exist in the integration test KubeBuilder has
a code implementation to re-generate the samples used in the docs and add customizations on top,
for further information check the [hack/docs](https://github.com/kubernetes-sigs/kubebuilder/tree/master/hack/docs).

This subcommand could have a similar implementation that could be used by the tests and this plugin.
Note that to run the commands using the binaries we are mainly using the following golang implementation:

```go
cmd := exec.Command(t.BinaryName, Options)
_, err := t.Run(cmd)
```

### Risks and Mitigations

**Hard to keep the command maintained**

I risk to consider would be we identify that would be hard to keep this command maintained
because we need to develop specific code operations for each plugin. The mitigation for
this problem could be developing a design more generic that could work with all plugins.

However, initially a more generic design implementation does not appear to be achievable and
would be considered out of the scope of this proposal (no goal). It should to be considered
as a second phase of this implementation.

Therefore, the current achievable mitigation in place is that KubeBuilder has a policy of
does not provide official support and distributed many plugins within for the same reasons.

### Proof of Concept

All input data is tracked. Also, as described above we have examples of code implementation
that uses the binary to scaffold the projects. Therefore, the goal of this project seems
very reasonable and achievable. An initial work to try to address this requirement can
be checked in this [pull request](https://github.com/kubernetes-sigs/kubebuilder/pull/3022)

## Drawbacks

- If the value that feature provides does not pay off the effort to keep it
  maintained then we would need to drawback.
- If a better suggestion solution to address the need be proposed.

## Alternatives

N/A

## Implementation History

The idea of automate the re-scaffold of the project is what motivates
us track all input data in to the [project config][project-config]
in the past. We also tracked the [issue](https://github.com/kubernetes-sigs/kubebuilder/issues/2068)
based on discussion that we have to indeed try to add further
specific implementations to do operations per major bumps. For example:

To upgrade from go/v3 to go/v4 we know exactly what are the changes in the layout
then, we could automate these specific operations as well. However, this first idea is harder yet
to be addressed and maintained.

[project-config]: https://book.kubebuilder.io/reference/project-config.html