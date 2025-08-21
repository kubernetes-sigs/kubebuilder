# AI integration for `alpha update`

> Status: Experimental, optional, opt‑in workflow. Not enabled by default.

## Design decisions

We experimented with different AI integrations for the `alpha update` workflow and chose a single, reliable path based on the following non‑negotiable rules:

- No Personal Access Tokens (PATs), no GitHub App tokens
- No repository setting changes beyond standard Actions permissions
- Must work with the default `GITHUB_TOKEN`
- Must require as fewer permissions as possible

Outcome:

After exploring different integrations, we found out that:

- The only feasible integration is using **GitHub Models** in **GitHub Actions** with the additional permission `models: read`.
- Assigning a Copilot Agent to issues was discarded (requires extra permissions and is not available in all orgs).
- Auto‑resolving conflicts with models was discarded (non‑deterministic, inconsistent, and risky).
- Using GitHub Models to produce **summaries** has proven to be reliable and helpful.

This page documents that single supported recipe.

---

## GitHub Models summary on Issues

Best for: users who want a dependable, automated summary on Issues created by `kubebuilder alpha update`.

You get:

- Concise summary of scaffold changes
- File list with conflict markers (if any)
- Room for prompt engineering
- Room for model parameters tuning.

You do not get:

- Automatic conflict resolution (intentionally out of scope)

Required workflow permissions:

- Default `GITHUB_TOKEN` with `permissions: models: read`.

---

### Workflow (`.github/workflows/update-ai-summary.yml`)

The following workflow sets a template that can be optmized by users via prompt engineering and model parameters tuning.

```yaml
name: Alpha Update (with AI summary)

permissions:
  contents: write
  issues: write
  pull-requests: write
  models: read  # Necessary permission

on:
  workflow_dispatch:
  schedule:
    - cron: "0 0 * * 2" # Every Tuesday at 00:00 UTC

jobs:
  alpha-update:
    runs-on: ubuntu-latest
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}

    steps:
    - name: Checkout repository
      uses: actions/checkout@v4
      with:
        token: ${{ secrets.GITHUB_TOKEN }}
        fetch-depth: 0
    
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: stable

    - name: Configure Git
      run: |
        git config --local user.name "github-actions[bot]"
        git config --local user.email "github-actions[bot]@users.noreply.github.com"

    - name: Install Kubebuilder
      run: |
        git clone --depth 1 --branch master \
          https://github.com/kubernetes-sigs/kubebuilder.git /tmp/kubebuilder
        cd /tmp/kubebuilder
        make build
        sudo cp bin/kubebuilder /usr/local/bin/
        kubebuilder version

    - name: Run kubebuilder alpha update
      run: |
        kubebuilder alpha update \
          --force \
          --restore-path .github/workflows \
          --push \
          --open-gh-issue

    - name: Create patch of last commit # Users may want to extract other info
      run: |
        git show HEAD > update.patch

    - name: Prepare model prompt # Room for prompt engineering
      run: |
        cat > prompt.txt <<'EOF'
        Start by stating that this summary is AI generated and can contain errors.
        You are a concise update summarizer for an automated Kubebuilder scaffold update.
        Analyze the patch and write a GitHub Issue comment that explains what changed.
        Scan the patch for Git conflict markers (<<<<<<<, =======, >>>>>>>). For each conflict:
        Explain what caused the conflict.
        How to properly resolve them.
        Inform the user about the recommended next steps.
        Constraints: Markdown only, no raw diff snippets.

        ---- DIFF BELOW ----
        EOF
        cat update.patch >> prompt.txt

    - name: Generate summary with GitHub Models # Room for parameters tuning
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      run: |
        set -o pipefail
        jq -Rs '
        {
          model: "openai/gpt-4o",
          messages: [
            { "role": "user", "content": . }
          ],
          max_tokens: 4000,
          temperature: 0
        }' < prompt.txt \
        | curl -sS https://models.github.ai/inference/chat/completions \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $GITHUB_TOKEN" \
            --data @- \
        | jq -r '.choices[0].message.content' > COMMENT.md

    - name: Determine update issue number
      id: find-issue
      env:
        GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      run: |
        out=$(gh issue list --state open --limit 100 --json number,title,createdAt)
        issue_number=$(echo "$out" | jq -r '
          sort_by(.createdAt)
          | map(select(.title|test("^\\[Action Required\\] Upgrade the Scaffold: v[0-9.]+ -> v[0-9.]+$")))
          | last.number // empty
        ')
        echo "issue_number=${issue_number}" >> "$GITHUB_OUTPUT"

    - name: Post comment to issue
      env:
        GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      run: |
        gh issue comment "${{ steps.find-issue.outputs.issue_number }}" --body-file COMMENT.md

```

Notes:

- You may want to truncate or chunk diffs to fit prompt limits.
- You may want to pipe into the prompt only a group of files as input for more targeted summaries.
- The prompts can be customized for better fit results.
- The model parameters can also be tuned (max tokens, temperature, etc).
- Model IDs/endpoints may change; verify in GitHub Models docs.

---

## Guardrails and validation

- Keep humans in the loop; treat AI output as advisory.
- Prefer summaries/checklists over automatic conflict resolution.
- After edits, always run:

```bash
make manifests generate fmt vet lint-fix
# and your test suite
```

---

## References

- [Using AI Models with GitHub Actions](https://docs.github.com/en/github-models/use-github-models/integrating-ai-models-into-your-development-workflow#using-ai-models-with-github-actions)
- [GitHub Models rate limits](https://docs.github.com/en/github-models/use-github-models/prototyping-with-ai-models#rate-limits)
- [GitHub Models Quickstart Guide](https://docs.github.com/en/github-models/quickstart)
