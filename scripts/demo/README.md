# Kubebuilder Demo

This directory contains scripts to run a comprehensive demo of Kubebuilder features with a local Kind cluster.

## Quick Demo (Manual)

To run the demo manually:

```sh
mkdir /tmp/kb-demo
cd /tmp/kb-demo
DEMO_AUTO_RUN=1 /path/to/kubebuilder/scripts/demo/run.sh
```

**Keyboard shortcuts during manual run:**
- `<CTRL-C>` to terminate the script
- `<CTRL-D>` to terminate the asciinema recording
- `<CTRL-C>` to save the recording locally

## Automated Demo Recording

To automatically record and update the demo using Asciinema:

```sh
# From the root of the Kubebuilder repository
make update-demo
```

This will:
1. Check for required dependencies (asciinema, svg-term, kind, kubectl)
2. Set up a Kind cluster for the demo
3. Record the demo session automatically
4. Convert the recording to SVG format
5. Update the demo file in `docs/gif/kb-demo.${VERSION}.svg`
6. Clean up temporary files

## Using the Script Utilities

The demo generation script (`generate-demo.sh`) supports flexible usage:

```sh
# Generate the default kb-demo
./scripts/demo/generate-demo.sh

# Generate a custom demo with a different name
./scripts/demo/generate-demo.sh my-custom-demo

# Generate a custom demo using a different script
./scripts/demo/generate-demo.sh advanced-demo ./path/to/custom-script.sh

# Show help
./scripts/demo/generate-demo.sh --help
```

**Note**: Custom demos (non-default names) won't automatically update the README. You'll need to reference them manually in documentation.

## Setup Demo Cluster Only

If you just want to set up the Kind cluster for testing:

```sh
make setup-demo-cluster
```

## Clean Up Demo Cluster

To remove the demo Kind cluster when done:

```sh
make clean-demo
```

## Prerequisites

- `kind`: For creating local Kubernetes clusters
- `kubectl`: For interacting with Kubernetes
- `asciinema`: For recording terminal sessions (recording only)
- `svg-term`: For converting recordings to SVG (recording only, requires Node.js/npm)
