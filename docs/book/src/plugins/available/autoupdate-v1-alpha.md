# AutoUpdate (`autoupdate/v1-alpha`)

Keeping your Kubebuilder project up to date with the latest improvements shouldn’t be a chore.
With a small amount of setup, you can receive **automatic Pull Request** suggestions whenever a new
Kubebuilder release is available — keeping your project **maintained, secure, and aligned with ecosystem changes**.

This automation uses the [`kubebuilder alpha update`][alpha-update-command] command with a **3-way merge strategy** to
refresh your project scaffold, and wraps it in a GitHub Actions workflow that opens an **Issue** with a **Pull Request compare link** so you can create the PR and review it.

<aside class="warning">
<h1>Protect your branches</h1>

This workflow by default **only** creates and pushes the merged files to a branch
called `kubebuilder-update-from-<from-version>-to-<to-version>`.

To keep your codebase safe, use branch protection rules to ensure that
changes aren't pushed or merged without proper review.

</aside>

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

By default, the workflow scaffolded uses `--use-gh-model` the flag to leverage in [AI models][ai-models] to help you understand
what changes are needed. You'll get a concise list of changed files to streamline the review, for example:

<img width="582" height="646" alt="Screenshot 2025-08-26 at 13 40 53" src="https://github.com/user-attachments/assets/d460a5af-5ca4-4dd5-afb8-7330dd6de148" />

If conflicts arise, AI-generated comments call them out and provide next steps, such as:

<img width="600" height="188" alt="Conflicts" src="https://github.com/user-attachments/assets/2142887a-730c-499a-94df-c717f09ab600" />

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

[alpha-update-command]: ./../../reference/commands/alpha_update.md
[ai-models]: https://docs.github.com/en/github-models/about-github-models
