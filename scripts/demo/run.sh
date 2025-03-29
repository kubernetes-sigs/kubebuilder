#!/bin/bash

# Determine the path to the Kubebuilder root directory
kubebuilder_root=$(git rev-parse --show-toplevel)

# Define the temporary directory
tmpdir="$kubebuilder_root/tmp-demo"

# Remove the temporary directory if it exists
rm -rf -f "$tmpdir"

# Create the temporary directory
mkdir -p "$tmpdir"
echo "Creating temporary directory: $tmpdir"
cd "$tmpdir"

# Set GOPATH to the Kubebuilder root directory
export GOPATH="$kubebuilder_root"
export PATH="$PATH:$kubebuilder_root/bin"

# Source util.sh *before* starting Asciinema
# . "$kubebuilder_root/scripts/demo/util.sh"

# Start recording
asciinema rec --overwrite demo.cast

# Run the commands directly
echo "Initialize Go modules"
go mod init demo.kubebuilder.io

echo "Let's initialize the project"
kubebuilder init --domain tutorial.kubebuilder.io
clear

echo "Examine scaffolded files..."
tree .
clear

echo "Create our custom cronjob api"
kubebuilder create api --group batch --version v1 --kind CronJob
clear

echo "Let's take a look at the API and Controller files"
tree ./api ./internal/controller
clear

# Stop recording
echo "Recording finished. File saved as demo.cast"

# Copy the recording to the current directory
cp demo.cast "$kubebuilder_root/scripts/demo/demo.cast"

# Optionally upload to asciinema.org
# asciinema upload demo.cast

# Clean up the temporary directory
echo "Cleaning up temporary directory: $tmpdir"
cd "$kubebuilder_root/scripts/demo"  # Go back to the original directory
rm -rf -f "$tmpdir"

echo "Asciinema recording saved to: scripts/demo/demo.cast"

