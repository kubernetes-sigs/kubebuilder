# AutoUpdate (`autoupdate/v1-alpha`)

Keeping your Kubebuilder project up to date with the latest improvements shouldn’t be a chore.
With a small amount of setup, you can receive **automatic Pull Request** suggestions whenever a new
Kubebuilder release is available — keeping your project **maintained, secure, and aligned with ecosystem changes**.

This automation uses the [`kubebuilder alpha update`][alpha-update-command] command with a **3-way merge strategy** to
refresh your project scaffold, and wraps it in a GitHub Actions workflow that opens an **Issue** with a **Pull Request compare link** so you can create the PR and review it.

## When to Use It

- When you don’t deviate too much from the default scaffold — ensure that you see the note about customization [here](https://book.kubebuilder.io/versions_compatibility_supportability#project-customizations).
- When you want to reduce the burden of keeping the project updated and well-maintained.
- When you want to guidance and help from AI to know what changes are needed to keep your project up to date
as to solve conflicts.

## How to Use It

- If you want to add the `autoupdate` plugin to your project:

```shell
kubebuilder edit --plugins="autoupdate.kubebuilder.io/v1-alpha"
```

- If you want to create a new project with the `autoupdate` plugin:

```shell
kubebuilder init --plugins=go/v4,autoupdate/v1-alpha
```

## How It Works

This will scaffold a GitHub Actions workflow that runs the [kubebuilder alpha update][alpha-update-command] command.
Whenever a new Kubebuilder release is available, the workflow will automatically open an **Issue** with a Pull Request compare link so you can easily create the PR and review it, such as:

<img width="638" height="482" alt="Example Issue" src="https://github.com/user-attachments/assets/589fd16b-7709-4cd5-b169-fd53d69790d4" />

Moreover, AI models are used to help you understand what changes are needed to keep your project up to date,
and to suggest resolutions if conflicts are encountered, as in the following example:

<img width="715" height="424" alt="AI Example Comment" src="https://github.com/user-attachments/assets/3f8bc35d-8ba0-4fc5-931b-56b1cb238462" />

You will also see a list of files changed, making it easier to review the updates.
And, if conflicts arise, a summary of the conflicts will be provided to help you resolve them.

### Workflow details

The workflow will check once a week for new releases, and if there are any, it will create an Issue with a Pull Request compare link so you can create the PR and review it.
The command called by the workflow is:

```shell
	# More info: https://kubebuilder.io/reference/commands/alpha_update
    - name: Run kubebuilder alpha update
      run: |
		# Executes the update command with specified flags.
		# --force: Completes the merge even if conflicts occur, leaving conflict markers.
		# --push: Automatically pushes the resulting output branch to the 'origin' remote.
		# --restore-path: Preserves specified paths (e.g., CI workflow files) when squashing.
		# --open-gh-models: Adds an AI-generated comment to the created Issue with
		#   a short overview of the scaffold changes and conflict-resolution guidance (If Any).
        kubebuilder alpha update \
          --force \
          --push \
          --restore-path .github/workflows \
          --open-gh-issue \
          --use-gh-models
```

<aside class="warning">
<h1>Protect your branches</h1>

This workflow by default **only** creates and pushes the merged files to a branch
called `kubebuilder-update-from-<from-version>-to-<to-version>`.

To keep your codebase safe, use branch protection rules to ensure that
changes aren't pushed or merged without proper review.

</aside>

[alpha-update-command]: ./../../reference/commands/alpha_update.md