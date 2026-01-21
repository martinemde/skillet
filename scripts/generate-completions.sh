#!/bin/bash
# Generate shell completion scripts for skillet
# This script is called during the build process to generate
# completion files that are included in release archives.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPLETIONS_DIR="${PROJECT_ROOT}/completions"

echo "Generating shell completion scripts..."

# Create completions directory
mkdir -p "${COMPLETIONS_DIR}"

# Build skillet first if needed
if [[ ! -x "${PROJECT_ROOT}/skillet" ]]; then
    echo "Building skillet..."
    go build -o "${PROJECT_ROOT}/skillet" "${PROJECT_ROOT}/cmd/skillet"
fi

# Generate completion scripts
echo "  - bash"
"${PROJECT_ROOT}/skillet" completion bash > "${COMPLETIONS_DIR}/skillet.bash"

echo "  - zsh"
"${PROJECT_ROOT}/skillet" completion zsh > "${COMPLETIONS_DIR}/_skillet"

echo "  - fish"
"${PROJECT_ROOT}/skillet" completion fish > "${COMPLETIONS_DIR}/skillet.fish"

echo "Done! Completion scripts generated in ${COMPLETIONS_DIR}/"
