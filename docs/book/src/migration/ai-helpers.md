# Using AI to Migrate Projects from Any Version to the Latest

AI can assist manual migrations by reducing repetitive work and helping resolve breaking changes. It won't replace the [Manual Migration Process](./manual-process.md), but it can help reduce effort and accomplish the goal.

<aside class="warning">

<h1>Important</h1>

These AI instructions are provided as examples to help guide your migration. Always validate AI output carefully - you remain responsible for ensuring correctness.

</aside>

## Workflow and AI-Assisted Steps

**Step 1: Reorganize to New Layout** (required only for legacy layouts)

AI helps ensure the project is structured with the new layout (main.go under cmd/, controllers and webhooks inside internal/). Review and verify the reorganization, then run `make build` to ensure it still compiles.

See [Step 1: Reorganize to New Layout](./reorganize-layout.md)

**Step 2: Discovery CLI Commands to Re-scaffold**

AI analyzes your project and generates all Kubebuilder CLI commands to fully re-scaffold with the latest release. Create a backup (`mkdir ../migration-backup && cp -r . ../migration-backup/`), then execute the generated commands to scaffold a fresh project.

See [Step 2: Discovery CLI Commands](./discovery-commands.md)

**Step 3: Port Custom Code**

AI helps port your custom code from backup to the new scaffolded project. Review all changes carefully and ensure business logic is correctly transferred.

See [Step 3: Port Custom Code](./port-code.md)

**Step 4: Validate**

Run `make generate && make manifests && make build`, then `make test` to verify all tests pass. Deploy to a test cluster and verify your solution still does the same thing.

See the [Manual Migration Process](./manual-process.md) for complete details.
