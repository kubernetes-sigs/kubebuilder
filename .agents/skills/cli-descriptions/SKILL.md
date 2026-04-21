---
name: cli-descriptions
description: Standards for CLI command and flag descriptions in Kubebuilder. Use when writing or reviewing CLI flags, commands, or help text.
license: Apache-2.0
metadata:
  author: The Kubernetes Authors
---

# CLI Description Standards for Kubebuilder

Use this skill to write, review, or normalize Kubebuilder CLI command descriptions, flag help text, and examples.

## Workflow

1. Identify the CLI element:
   - boolean flag
   - value flag
   - command short description
   - command long description
   - examples/help text
2. Apply the matching standard below.
3. Rewrite the text to match the standard.
4. If reviewing, explain the issue briefly and propose the corrected wording.

## Core principles

- Prefer clarity over brevity.
- Use consistent patterns across similar flags and commands.
- Include examples, defaults, or fallback behavior when useful.
- Tell users what happens when they use the flag.

## Boolean flags

### Default false (opt-in behavior)

Use this pattern:

```
If set, <what happens>
```

Examples:

- `If set, enable multigroup layout (organize APIs by group)`
- `If set, skip Go version check`
- `If set, attempt to create resource even if it already exists`

### Default true (opt-out behavior)

When a boolean flag defaults to `true`, describe the behavior and how to disable it:

```
<what happens by default>; use --flag=false to <opposite behavior>
```

Or for flags with more context:

```
<what happens> (enabled by default; use --flag=false to disable)
```

Examples:

- `Run 'make generate' after generating files (enabled by default; use --make=false to disable)`
- `Run 'make manifests' after generating files (enabled by default; use --manifests=false to disable)`
- `Resource is namespaced by default; use --namespaced=false to create a cluster-scoped resource`
- `Generate the resource without prompting the user (enabled by default; use --resource=false to disable)`
- `Download dependencies after scaffolding (enabled by default; use --fetch-deps=false to disable)`

### Tri-state (enable/disable/preserve)

When a boolean flag preserves existing values if not provided, and supports both enabling and disabling:

```
Enable or disable <what>; use --flag=false to disable
```

Examples:

- `Enable or disable multigroup layout (organize APIs by group); use --multigroup=false to disable`
- `Enable or disable namespace-scoped deployment (default: cluster-scoped); use --namespaced=false to disable`

### Prompting flags

When a flag controls prompting behavior (prompts by default, but accepts explicit true/false to skip the prompt):

```
Prompt whether to <action> by default; use --flag=true or --flag=false to skip the prompt
```

Examples:

- `Prompt whether to generate the controller by default; use --controller=true or --controller=false to skip the prompt`

## Value flags

Use these patterns:

With an example:
```
<what> (e.g., <example>)
```

With a default:
```
<what> (e.g., <example>). Defaults to <default> if unset
```

With auto-detection:
```
<what> (e.g., <example>); auto-detected <method> if not provided
```

Examples:

- `Go module name (e.g., github.com/user/repo); auto-detected from current directory if not provided`
- `License header to use for boilerplate (e.g., apache2, none). Defaults to apache2 if unset`
- `Project version (e.g., 3). Defaults to CLI version if unset`
- `Domain for your APIs (e.g., example.org creates crew.example.org for API groups)`

## Optional value flags

When a flag is optional and modifies behavior, use:

```
[Optional] <what>. <additional context if useful> (e.g., <example>)
```

Examples:

- `[Optional] Container command to use for image initialization (e.g., --image-container-command="memcached,--memory-limit=64,modern,-o,-v")`
- `[Optional] Container port used by the container image (e.g., --image-container-port="11211")`

## Path flags

Use:

```
Path to <what>; <behavior> (e.g., <example>)
```

Example:

- `Path to custom license file; content copied to hack/boilerplate.go.txt (overrides --license)`

## List flags

For comma-separated lists or arrays:

```
Comma-separated list of <what> (e.g., --flag value1,value2)
```

Example:

- `Comma-separated list of spoke versions to be added to the conversion webhook (e.g., --spoke v1,v2)`

## Deprecated flags

For deprecated flags that will be removed in future versions:

```
[DEPRECATED] If set, <what it does>. This option will be removed in future versions
```

Example:

- `[DEPRECATED] If set, attempts to create resource under the API directory (legacy path). This option will be removed in future versions`

## Command short descriptions

Rules:

- Start with a capital letter.
- Use a brief phrase or sentence.
- Do not end with a period.
- Prefer imperative wording.

Examples:

- `Initialize a new project`
- `Scaffold a Kubernetes API`
- `Scaffold a Kubernetes API or webhook`

## Command long descriptions

Long descriptions should:

- explain what the command does
- mention generated files or directories when relevant
- separate required vs optional/configuration flags when helpful
- use full sentences and punctuation
- include behavioral notes when useful

Use this structure when it fits:

```
Initialize a new project including the following files:
  - a "go.mod" with project dependencies
  - a "PROJECT" file that stores project configuration
  - a "Makefile" with useful make targets

Required flags:
  --domain: Domain for your APIs (e.g., example.org creates crew.example.org for API groups)

Configuration flags:
  --repo: Go module name (e.g., github.com/user/repo); auto-detected from current directory if not provided

Note: Layout settings can be changed later with 'kubebuilder edit'.
```

## Command examples

Examples should:

- be realistic and valid
- progress from simple to complex
- explain non-obvious cases
- use the command name variable when writing Go examples

Example:

```go
subcmdMeta.Examples = fmt.Sprintf(`  # Initialize a new project
  %[1]s init --domain example.org

  # Initialize with multigroup layout
  %[1]s init --domain example.org --multigroup

  # Initialize with all options combined
  %[1]s init --plugins go/v4,autoupdate/v1-alpha --domain example.org --multigroup --namespaced
`, cliMeta.CommandName)
```

## Common Kubebuilder flag wording

Use these standard descriptions when applicable:

- `--domain`: `Domain for your APIs (e.g., example.org creates crew.example.org for API groups)`
- `--repo`: `Go module name (e.g., github.com/user/repo); auto-detected from current directory if not provided`
- `--plugins`: `Comma-separated list of plugin keys to use (e.g., go/v4, helm/v2-alpha). Defaults to the built-in go/v4 bundle if unset`
- `--multigroup`:
  - In `init`: `If set, enable multigroup layout (organize APIs by group)`
  - In `edit`: `Enable or disable multigroup layout (organize APIs by group); use --multigroup=false to disable`
- `--skip-go-version-check`:
  - When default is `false`: `If set, skip Go version check`
  - When default is `true`: `Skip the Go version check (enabled by default; use --skip-go-version-check=false to enforce)`
- `--force`: `If set, attempt to create resource even if it already exists` (or `If set, regenerate all files except Chart.yaml` for helm plugins)
- `--license`:
  - In `init`: `License header to use for boilerplate (e.g., apache2, none). Defaults to apache2 if unset`
  - In `edit`: `License header to use for boilerplate (e.g., apache2, none). If unset, preserves the existing boilerplate`
- `--owner`: `Owner name for copyright license headers`
- `--namespaced`:
  - In `api`: `Resource is namespaced by default; use --namespaced=false to create a cluster-scoped resource`
  - In `edit`: `Enable or disable namespace-scoped deployment (default: cluster-scoped); use --namespaced=false to disable`
- `--fetch-deps`: `Download dependencies after scaffolding (enabled by default; use --fetch-deps=false to disable)`
- `--make`: `Run 'make generate' after generating files (enabled by default; use --make=false to disable)`
- `--manifests`: `Run 'make manifests' after generating files (enabled by default; use --manifests=false to disable)`
- `--resource`: `Generate the resource without prompting the user (enabled by default; use --resource=false to disable)`
- `--controller`: `Prompt whether to generate the controller by default; use --controller=true or --controller=false to skip the prompt`

## Review output pattern

When reviewing, use this format:

- **Issue**: one short sentence describing what is wrong
- **Suggested text**: the corrected help text
- **Reason**: one short sentence tying it back to the standard

Example:

- **Issue**: Boolean flag description does not describe the behavior when the flag is present.
- **Suggested text**: `If set, enable multigroup layout (organize APIs by group)`
- **Reason**: Boolean flags should follow the `If set, ...` pattern.

## Checklist

- [ ] Boolean flags that default to `false` use `If set, ...`
- [ ] Boolean flags that default to `true` describe the behavior and how to disable (e.g., "enabled by default; use --flag=false to disable")
- [ ] Value flags include examples with `(e.g., ...)`
- [ ] List flags use `Comma-separated list of ...`
- [ ] Deprecated flags are marked with `[DEPRECATED]`
- [ ] Defaults are explicitly stated when they exist with "if unset" (e.g., "Defaults to X if unset")
- [ ] Auto-detection or fallback behavior is documented
- [ ] Short descriptions are capitalized and do not end with a period
- [ ] Long descriptions are complete and organized
- [ ] Examples are realistic and valid
- [ ] Terminology is consistent across commands and plugins

## Notes

- Apply these standards across all Kubebuilder plugins.
- Prefer consistency across the codebase over one-off wording choices.
- When the same flag appears with equivalent semantics across plugins, use identical descriptions. When semantics differ (e.g., `--force` in `create`/`api`/`edit`, or `--license` in `init`/`edit`), document the per-command variant explicitly.
- Flag descriptions should not end with a period (matches the `--help` convention used by cobra and the rest of Kubebuilder).
- For multi-line flag descriptions in code, use string concatenation with `+` and break at natural boundaries (70-80 characters per line).
- When in doubt, choose the wording that is clearest in `--help` output.
- See [references/REFERENCE.md](references/REFERENCE.md) for technical references and industry standards that inform these guidelines.
