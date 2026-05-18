#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# Description: This script sets up the environment and runs E2E tests for the
#              IPAM project.
# Usage:       From the root of the repo, run:
#              ./test/e2e/ci-e2e.sh
# -----------------------------------------------------------------------------

set -eux

REPO_ROOT=$(realpath "$(dirname "${BASH_SOURCE[0]}")/../..")
cd "${REPO_ROOT}" || exit 1

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
"${REPO_ROOT}/hack/e2e/ensure_go.sh"

# Verify they are available and have correct versions.
PATH=$PATH:/usr/local/go/bin
PATH=$PATH:$(go env GOPATH)/bin

case "${GINKGO_FOCUS:-}" in
  *upgrade*)
    ;;
  *)
  ;;
esac

# Support label-based test filtering via E2E_TESTS env var.
# Valid values: "basic", "features", "all" (default).
# Examples:
#   E2E_TESTS=basic ./hack/e2e/ci-script.sh    # Run only basic tests
#   E2E_TESTS=features ./hack/e2e/ci-script.sh # Run only feature tests
#   E2E_TESTS=all ./hack/e2e/ci-script.sh      # Run all tests (default)
E2E_TESTS="${E2E_TESTS:-all}"
case "${E2E_TESTS}" in
  basic)
    export GINKGO_FOCUS_LABELS="basic"
    ;;
  features)
    export GINKGO_FOCUS_LABELS="features"
    ;;
  all)
    ;;
  *)
    echo "ERROR: Invalid E2E_TESTS value '${E2E_TESTS}'. Valid values: basic, features, all"
    exit 1
    ;;
esac

# Run the e2e tests
make test-e2e
test_status="$?"

LOGS_DIR="${REPO_ROOT}/test/e2e/_artifacts/logs"
# Collect all artifacts
tar --directory ${REPO_ROOT}/test/e2e/_artifacts/ -czf "artifacts-e2e-ipam.tar.gz" ${REPO_ROOT}/test/e2e/_artifacts/

exit "${test_status}"
