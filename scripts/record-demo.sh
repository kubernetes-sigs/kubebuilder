#!/bin/bash
set -e

# Ensure asciinema is installed
if ! command -v asciinema &> /dev/null; then
    echo "Error: asciinema is not installed. Please install it first."
    echo "Visit https://asciinema.org/docs/installation for instructions."
    exit 1
fi

# Start recording
echo "Starting demo recording..."
asciinema rec -t "Kubebuilder Demo" demo.cast << EOF

# Show basic kubebuilder commands
echo "# Initialize a new project with Kubebuilder"
kubebuilder init --domain example.com --repo example.com/demo

echo "# Create a new API"
kubebuilder create api --group webapp --version v1 --kind Guestbook

echo "# Build and run the controller"
make install
make run

# Add small delays to make the demo more readable
sleep 2

EOF

echo "Recording completed! The demo has been saved as 'demo.cast'" 
