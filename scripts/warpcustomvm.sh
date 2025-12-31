#!/usr/bin/env bash

set -euo pipefail

# WarpCustomVM build script
# Builds the warpcustomvm binary and sets up plugin symlinks

if ! [[ "$0" =~ scripts/warpcustomvm.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

source ./scripts/constants.sh

echo "Building warpcustomvm plugin..."
go build -ldflags="$static_ld_flags" -o ./build/warpcustomvm ./vms/example/warpcustomvm/cmd/warpcustomvm/

# The VM ID is deterministic based on the VM's name and genesis
# For warpcustomvm, we'll use a generated ID
# You can generate a specific ID using: echo -n "warpcustomvm" | shasum -a 256
# For now, using a placeholder ID - replace with your actual VM ID
VM_ID="v3m4wPxaHpvGr8qfMeyK6PRW3idZrPHmYcMTt7oXdK47yurVH"

# Symlink to both global and local plugin directories to simplify
# usage for testing. The local directory should be preferred but the
# global directory remains supported for backwards compatibility.
LOCAL_PLUGIN_PATH="${PWD}/build/plugins"
GLOBAL_PLUGIN_PATH="${HOME}/.avalanchego/plugins"

echo ""
echo "Copying plugin to plugin directories..."
for plugin_dir in "${GLOBAL_PLUGIN_PATH}" "${LOCAL_PLUGIN_PATH}"; do
  PLUGIN_PATH="${plugin_dir}/${VM_ID}"
  echo "Copying ./build/warpcustomvm to ${PLUGIN_PATH}"
  mkdir -p "${plugin_dir}"
  cp -f "${PWD}/build/warpcustomvm" "${PLUGIN_PATH}"
done

echo ""
echo "✅ Build complete!"
echo "Binary location: ./build/warpcustomvm"
echo "Plugin ID: ${VM_ID}"
echo ""
echo "To generate a proper VM ID, run:"
echo "  avalanchego-genesis-generator create-vm-id --name warpcustomvm"
echo ""
echo "Plugin installed to:"
echo "  - ${LOCAL_PLUGIN_PATH}/${VM_ID}"
echo "  - ${GLOBAL_PLUGIN_PATH}/${VM_ID}"
