#!/usr/bin/env bash

# Copyright 2023 The Kubernetes Authors.
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

# Colors for output
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

check_prerequisites() {
    log "Checking prerequisites..."
    
    local missing_tools=()
    
    if ! command -v asciinema &> /dev/null; then
        missing_tools+=("asciinema (install with: brew install asciinema)")
    fi
    
    if ! command -v svg-term &> /dev/null; then
        missing_tools+=("svg-term (install with: npm install -g svg-term-cli)")
    fi
    
    if ! command -v kind &> /dev/null; then
        missing_tools+=("kind (install with: brew install kind)")
    fi
    
    if ! command -v kubectl &> /dev/null; then
        missing_tools+=("kubectl (install from: https://kubernetes.io/docs/tasks/tools/install-kubectl/)")
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        error "Missing required tools:\n$(printf '  - %s\n' "${missing_tools[@]}")"
    fi
    
    log "All prerequisites are installed ✓"
}

setup_cluster() {
    log "Setting up Kind cluster for demo..."
    "${SCRIPT_ROOT}/setup-kind.sh"
    
    log "Verifying cluster connection..."
    kubectl cluster-info --context kind-kubebuilder-demo > /dev/null
    log "Cluster connection verified ✓"
}

record_demo() {
    local recording_dir="/tmp/kb-demo-recording"
    
    log "Cleaning up any previous recording files..."
    rm -rf "$recording_dir"
    mkdir -p "$recording_dir"
    
    log "Starting demo recording in 3 seconds..."
    sleep 3
    
    cd "$recording_dir"
    asciinema rec \
        --command "${SCRIPT_ROOT}/run.sh" \
        --env "DEMO_AUTO_RUN=1" \
        --title "Kubebuilder Demo" \
        --idle-time-limit 2 \
        kb-demo.cast
}

convert_to_svg() {
    local recording_dir="/tmp/kb-demo-recording"
    local version
    version=$(git -C "$PROJECT_ROOT" describe --tags --abbrev=0 2>/dev/null || echo "v4.0.0")
    local svg_file="${PROJECT_ROOT}/docs/gif/kb-demo.${version}.svg"
    
    log "Converting recording to SVG..."
    svg-term \
        --in="${recording_dir}/kb-demo.cast" \
        --out="$svg_file" \
        --window \
        --width=120 \
        --height=30
    
    log "Demo updated! New file: docs/gif/kb-demo.${version}.svg"
    return 0
}

update_readme() {
    local version
    version=$(git -C "$PROJECT_ROOT" describe --tags --abbrev=0 2>/dev/null || echo "v4.0.0")
    
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

main() {
    log "Starting Kubebuilder demo generation..."
    
    check_prerequisites
    setup_cluster
    record_demo
    convert_to_svg
    update_readme
    cleanup
    
    log "Demo generation completed successfully! 🎉"
}

main "$@"
