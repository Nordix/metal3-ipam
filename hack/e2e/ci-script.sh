#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# Description: This script sets up the environment and runs E2E tests for the
#              IPAM project. It uses ushy-tools as bmc protocol emulator.
#              Supported protocols are: redfish and redfish-virtualmedia.
# Usage:       From the root of the repo, run:
#              ./test/e2e/ci-e2e.sh
# -----------------------------------------------------------------------------

set -eux

REPO_ROOT=$(realpath "$(dirname "${BASH_SOURCE[0]}")/../..")
cd "${REPO_ROOT}" || exit 1

# Set environment variables for the e2e tests
export IPAMPATH="${REPO_ROOT}"
FORCE_REPO_UPDATE="${FORCE_REPO_UPDATE:-false}"

export IPAMRELEASEBRANCH="${IPAMRELEASEBRANCH:-main}"

# Extract release version from release-branch name
if [[ "${IPAMRELEASEBRANCH}" == release-* ]]; then
  export IPAMRELEASE="v${IPAM_RELEASE_PREFIX}.99"
  export CAPI_RELEASE_PREFIX="v${IPAM_RELEASE_PREFIX}."
else
  export IPAMRELEASE="v1.14.99"
  export CAPI_RELEASE_PREFIX="v1.13."
fi

# Default CAPI_CONFIG_FOLDER to $HOME/.config folder if XDG_CONFIG_HOME not set
CONFIG_FOLDER="${XDG_CONFIG_HOME:-$HOME/.config}"
export CAPI_CONFIG_FOLDER="${CONFIG_FOLDER}/cluster-api"

# CAPI test framework uses kubectl in the background
"${REPO_ROOT}/hack/e2e/ensure_kubectl.sh"

# Verify they are available and have correct versions.
PATH=$PATH:/usr/local/go/bin
PATH=$PATH:$(go env GOPATH)/bin

case "${GINKGO_FOCUS:-}" in
  *upgrade*)
    ;;
  *)
  ;;
esac

# Run the e2e tests
make test-e2e
test_status="$?"

LOGS_DIR="${REPO_ROOT}/test/e2e/_artifacts/logs"
# Collect all artifacts
tar --directory ${REPO_ROOT}/test/e2e/_artifacts/ -czf "artifacts-e2e-ipam.tar.gz" ${REPO_ROOT}/test/e2e/_artifacts/

exit "${test_status}"
