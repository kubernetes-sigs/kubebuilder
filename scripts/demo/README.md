# Kubebuilder Demo

This directory contains scripts to run a comprehensive demo of Kubebuilder features with a local Kind cluster.

## Quick Demo (Manual)

To run the demo manually:

```sh
mkdir /tmp/kb-demo
cd /tmp/kb-demo
DEMO_AUTO_RUN=1 /path/to/kubebuilder/scripts/demo/run.sh
```

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

### Generate Custom Demos

The script supports generating multiple demo variations:

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

Custom demos will be saved to `docs/gif/${DEMO_NAME}.${VERSION}.svg` and won't automatically update the README (you'll need to reference them manually).

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

## Prerequisites for Recording

- `kind`: For creating local Kubernetes clusters
- `kubectl`: For interacting with Kubernetes
- `asciinema`: For recording terminal sessions
- `svg-term`: For converting recordings to SVG (requires Node.js/npm)

## What the Demo Shows

The current demo showcases:

1. **Cluster Setup**: Creates a local Kind cluster for testing
2. **Installation**: Installing Kubebuilder from scratch
3. **Project Initialization**: Creating a new operator project
4. **API Creation**: Creating APIs with validation markers
5. **Plugin System**: Using the deploy-image plugin
6. **Modern Features**:
   - Validation markers (`+kubebuilder:validation`)
   - Multiple APIs in one project
   - Generated CRDs with OpenAPI schemas
   - Sample resource management
7. **Development Workflow**: Install CRDs, apply samples, run controller
8. **Cluster Integration**: Full integration with Kubernetes cluster

## Manual Recording Instructions (Legacy)

If you prefer to record manually:

```sh
# Set up Kind cluster first
./scripts/demo/setup-kind.sh

# Create temporary directory
mkdir /tmp/kb-demo
cd /tmp/kb-demo

# Start recording
asciinema rec --title "Kubebuilder Demo" kb-demo.cast

# Run the demo script
DEMO_AUTO_RUN=1 /path/to/kubebuilder/scripts/demo/run.sh

# Stop recording with Ctrl+C when done
# Convert to SVG
svg-term --in=kb-demo.cast --out=kb-demo.svg --window --width=120 --height=30

# Clean up when done
kind delete cluster --name kubebuilder-demo
```
