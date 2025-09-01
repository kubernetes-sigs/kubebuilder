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
  ```sh
  # macOS
  brew install kind
  # Linux
  curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
  chmod +x ./kind && sudo mv ./kind /usr/local/bin/kind
  ```

- `kubectl`: For interacting with Kubernetes
  ```sh
  # macOS
  brew install kubectl
  # Other platforms: https://kubernetes.io/docs/tasks/tools/install-kubectl/
  ```

- `asciinema`: For recording terminal sessions
  ```sh
  # macOS
  brew install asciinema
  # Ubuntu/Debian
  sudo apt-get install asciinema
  ```

- `svg-term`: For converting recordings to SVG
  ```sh
  npm install -g svg-term-cli
  ```

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
