# Reduce the Burn to Keep Your Project Maintained: How to Automate Kubebuilder Updates and Stay in Sync with the Ecosystem

Keeping your Kubebuilder project up to date with the latest improvements shouldn’t be a chore.
With a small amount of setup, you can receive **automatic Pull Requests** whenever a new
Kubebuilder release is available — keeping your project **maintained, secure, and aligned with ecosystem changes**.

This automation uses the [kubebuilder alpha update][alpha-update] command with a **3-way merge strategy** to
refresh your project scaffold, and wraps it in a GitHub Actions workflow that opens a PR for you to review.

Follow these steps to set up automation.

### 1. Add the Workflow

Create a new file in your repository:
`/.github/workflows/update.yaml`

```yaml
name: Workflow Auto-Update

permissions:
  contents: write
  pull-requests: write

on:
  workflow_dispatch:
  schedule:
    - cron: "0 0 * * 1"    # Every Monday 00:00 UTC

jobs:
  alpha-update:
    runs-on: ubuntu-latest

    steps:
      # 1) Checkout the repository with full history
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          fetch-depth: 0

      # 2) Install the latest stable Go toolchain
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: 'stable'

      # 3) Install Kubebuilder CLI
      - name: Install Kubebuilder
        run: |
          curl -L -o kubebuilder "https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)"
          chmod +x kubebuilder
          sudo mv kubebuilder /usr/local/bin/

      # 4) Extract Kubebuilder version (e.g., v4.6.0) for branch/title/body
      - name: Get Kubebuilder version
        id: kb
        shell: bash
        run: |
          RAW="$(kubebuilder version 2>/dev/null || true)"
          VERSION="$(printf "%s" "$RAW" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
          echo "version=${VERSION:-vunknown}" >> "$GITHUB_OUTPUT"

      # 5) Run kubebuilder alpha update
      - name: Run kubebuilder alpha update
        run: |
          kubebuilder alpha update --force

      # 6) Restore workflow files so the update doesn't overwrite CI config
      - name: Restore workflows directory
        run: |
          git restore --source=main --staged --worktree .github/workflows
          git add .github/workflows
          git commit --amend --no-edit || true

      # 7) Push to a versioned branch; create PR if missing, otherwise it just updates
      - name: Push branch and create/update PR
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        shell: bash
        run: |
          set -euo pipefail
          VERSION="${{ steps.kb.outputs.version }}"
          PR_BRANCH="kubebuilder-update-to-${VERSION}"

          # Create or update the branch and push
          git checkout -B "$PR_BRANCH"
          git push -u origin "$PR_BRANCH" --force

          PR_TITLE="chore: update scaffolding to Kubebuilder ${VERSION}"
          PR_BODY=$'Automated update of Kubebuilder project scaffolding to '"${VERSION}"$'.\n\nMore info: https://github.com/kubernetes-sigs/kubebuilder/releases\n\n :warning: If conflicts arise, resolve them and run:\n```bash\nmake manifests generate fmt vet lint-fix\n```'

          # Try to create the PR; ignore error only if it already exists
          if ! gh pr create \
            --title "${PR_TITLE}" \
            --body "${PR_BODY}" \
            --base main \
            --head "$PR_BRANCH"
          then
            EXISTING="$(gh pr list --state open --head "$PR_BRANCH" --json number --jq '.[0].number' || true)"
            if [ -n "${EXISTING}" ]; then
              echo "PR #${EXISTING} already exists for ${PR_BRANCH}, branch updated."
            else
              echo "Failed to create PR for ${PR_BRANCH} and no open PR found."
              exit 1
            fi
          fi
```

### 2. Configure Your Repository

**Enable workflow write permissions**

1. Go to **Settings → Actions → General → Workflow permissions**
2. Select **Read and write permissions**
3. Check **Allow GitHub Actions to create and approve pull requests**
4. Click **Save**

**Ensure your `main` (or default) branch is protected**

1. Go to **Settings → Branches → Branch protection rules**
2. Add a rule for your default branch
3. Enable **Require pull requests before merging** — ensures all updates arrive via PR for review

## How It Works

- **Checkout** — Gets your full repo history so `git restore` works correctly.
- **Install Kubebuilder** — Downloads the latest CLI for your OS/arch from the official endpoint.
- **Run `kubebuilder alpha update`** — Uses the 3-way merge strategy to update your scaffold.
- **Preserve workflows** — Restores `.github/workflows` so the workflow itself is not overwritten.
- **Open PR** — Pushes a branch and creates a PR describing the automated update.

## Demonstration

Video coming soon — stay tuned!

## References

- [GitHub Actions: `GITHUB_TOKEN` permissions](https://docs.github.com/en/actions/security-guides/automatic-token-authentication#modifying-the-permissions-for-the-github_token)
- [GitHub Actions: protected branches](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/defining-the-mergeability-of-pull-requests/about-protected-branches)
- [GitHub CLI: `gh pr create`](https://cli.github.com/manual/gh_pr_create)
- [Workflow syntax / `schedule` (cron)](https://docs.github.com/en/actions/writing-workflows/workflow-syntax-for-github-actions#onschedule)
- [Kubebuilder releases](https://github.com/kubernetes-sigs/kubebuilder/releases)
- [Alpha update Command](./alpha_update.md)

[alpha-update]: ./alpha_update.md