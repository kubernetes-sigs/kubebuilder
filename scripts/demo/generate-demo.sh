#!/usr/bin/env bash

# Copyright 2025 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_ROOT}/../.." && pwd)"

# Source the e2e test infrastructure for kind cluster management
source "${PROJECT_ROOT}/test/common.sh"
source "${PROJECT_ROOT}/test/e2e/setup.sh"

# Default demo name
DEMO_NAME="${1:-kb-demo}"
DEMO_SCRIPT="${2:-${SCRIPT_ROOT}/run.sh}"

# Use color output from common.sh or define our own
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

usage() {
    cat << EOF
Usage: $0 [DEMO_NAME] [DEMO_SCRIPT]

Generate an asciinema demo recording and convert it to SVG.

Arguments:
    DEMO_NAME     Name of the demo (default: kb-demo)
    DEMO_SCRIPT   Path to the demo script to run (default: ${SCRIPT_ROOT}/run.sh)

Examples:
    $0                              # Generate default kb-demo
    $0 my-custom-demo               # Generate custom demo with default script
    $0 advanced-demo ./my-demo.sh   # Generate custom demo with custom script

EOF
    exit 0
}

check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check for help flag
    if [[ "$1" == "-h" || "$1" == "--help" ]]; then
        usage
    fi
    
    # Validate demo script exists
    if [[ ! -f "$DEMO_SCRIPT" ]]; then
        error "Demo script not found: $DEMO_SCRIPT"
    fi
    
    local missing_tools=()
    
    if ! command -v asciinema &> /dev/null; then
        missing_tools+=("asciinema")
    fi
    
    if ! command -v svg-term &> /dev/null; then
        missing_tools+=("svg-term")
    fi
    
    if ! command -v kind &> /dev/null; then
        missing_tools+=("kind")
    fi
    
    if ! command -v kubectl &> /dev/null; then
        missing_tools+=("kubectl")
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        error "Missing required tools:\n$(printf '  - %s\n' "${missing_tools[@]}")"
    fi
    
    log "All prerequisites are installed ✓"
}

setup_cluster() {
    log "Setting up Kind cluster for demo..."
    
    # Use the e2e infrastructure to set up the cluster
    export KIND_CLUSTER="kubebuilder-demo"
    
    # Build kubebuilder, fetch tools, and install kind if needed
    build_kb
    fetch_tools
    install_kind
    
    # Create the cluster using the e2e setup function
    create_cluster ${KIND_K8S_VERSION}
    
    log "Cluster connection verified ✓"
}

record_demo() {
    local recording_dir="/tmp/kb-demo-recording"
    
    log "Cleaning up any previous recording files..."
    rm -rf "$recording_dir"
    mkdir -p "$recording_dir"
    
    log "Starting demo recording for '${DEMO_NAME}' in 3 seconds..."
    sleep 3
    
    cd "$recording_dir"
    asciinema rec \
        --command "$DEMO_SCRIPT" \
        --env "DEMO_AUTO_RUN=1" \
        --title "Kubebuilder Demo: ${DEMO_NAME}" \
        --idle-time-limit 2 \
        "${DEMO_NAME}.cast"
}

convert_to_svg() {
    local recording_dir="/tmp/kb-demo-recording"
    local version="$1"
    local svg_file="${PROJECT_ROOT}/docs/gif/${DEMO_NAME}.${version}.svg"
    
    log "Converting recording to SVG..."
    svg-term \
        --in="${recording_dir}/${DEMO_NAME}.cast" \
        --out="$svg_file" \
        --window \
        --width=120 \
        --height=30
    
    log "Demo updated! New file: docs/gif/${DEMO_NAME}.${version}.svg"
    return 0
}

update_readme() {
    local version="$1"
    
    # Only update README for the default kb-demo
    if [[ "$DEMO_NAME" != "kb-demo" ]]; then
        log "Skipping README update for custom demo '${DEMO_NAME}'"
        return 0
    fi
    
    log "Updating README.md with new demo..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s|docs/gif/kb-demo\.v[^)]*\.svg|docs/gif/kb-demo.${version}.svg|g" "${PROJECT_ROOT}/README.md"
    else
        # Linux
        sed -i "s|docs/gif/kb-demo\.v[^)]*\.svg|docs/gif/kb-demo.${version}.svg|g" "${PROJECT_ROOT}/README.md"
    fi
    
    log "README.md updated with new demo file ✓"
}

cleanup() {
    log "Cleaning up temporary files..."
    rm -rf /tmp/kb-demo-recording
    log "To clean up the demo cluster, run: make clean-demo"
}

cleanup_cluster() {
    log "Cleaning up demo cluster..."
    export KIND_CLUSTER="kubebuilder-demo"
    delete_cluster
    log "Demo cluster removed ✓"
}

main() {
    # Check for help flag first
    if [[ "$1" == "-h" || "$1" == "--help" ]]; then
        usage
    fi
    
    log "Starting Kubebuilder demo generation for '${DEMO_NAME}'..."
    log "Using demo script: ${DEMO_SCRIPT}"
    
    # Extract version once to avoid duplication
    local version
    version=$(git -C "$PROJECT_ROOT" describe --tags --abbrev=0 2>/dev/null || echo "v4.0.0")
    
    check_prerequisites
    setup_cluster
    record_demo
    convert_to_svg "$version"
    update_readme "$version"
    cleanup
    
    log "Demo generation completed successfully! 🎉"
    log "Generated: docs/gif/${DEMO_NAME}.${version}.svg"
}

main "$@"
