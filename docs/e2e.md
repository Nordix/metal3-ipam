# E2E Testing

This document describes how to run the IPAM end-to-end tests.

## Prerequisites

- Go (see `hack/e2e/ensure_go.sh` for minimum version)
- kubectl (see `hack/e2e/ensure_kubectl.sh` for minimum version)
- Docker (for Kind cluster provisioning)

## Running E2E Tests

### All tests

```sh
make test-e2e
```

### Run only basic tests

```sh
make test-e2e GINKGO_FOCUS_LABELS=basic
```

### Run only feature tests

```sh
make test-e2e GINKGO_FOCUS_LABELS=features
```

### Skip specific test labels

```sh
make test-e2e GINKGO_SKIP_LABELS=features
```

## Test Labels

Tests are organized using Ginkgo labels:

| Label | Description |
|-------|-------------|
| `basic` | Core IPAM operations: IPPool CRUD, IP allocation via Metal3 and CAPI claims, garbage collection |
| `features` | Advanced functionality: preallocations, pool exhaustion/recovery, multi-pool, status tracking, mixed claims |

## Configuration

### Make variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GINKGO_NODES` | `2` | Number of parallel Ginkgo nodes |
| `GINKGO_TIMEOUT` | `3h` | Overall test timeout |
| `GINKGO_FOCUS` | (empty) | Regex to focus on specific test names |
| `GINKGO_FOCUS_LABELS` | (empty) | Run only tests with these labels |
| `GINKGO_SKIP_LABELS` | (empty) | Skip tests with these labels |
| `USE_EXISTING_CLUSTER` | `false` | Use current kubeconfig instead of creating a Kind cluster |
| `SKIP_RESOURCE_CLEANUP` | `false` | Keep test resources after completion (for debugging) |
| `E2E_CONF_FILE` | `test/e2e/config/e2e_conf.yaml` | Path to E2E config file |

### Using an existing cluster

To run against an already-running cluster (useful for debugging):

```sh
make test-e2e USE_EXISTING_CLUSTER=true
```

### Skipping cleanup

To keep resources around after tests finish (for post-mortem debugging):

```sh
make test-e2e SKIP_RESOURCE_CLEANUP=true
```

## CI Script

The CI entry point is `hack/e2e/ci-script.sh`. It:

1. Ensures Go and kubectl meet minimum versions
1. Selects test labels via `E2E_TESTS` env var (`basic`, `features`, or `all`)
1. Runs `make test-e2e`
1. Collects artifacts into a tarball

```sh
# Run basic tests only in CI
E2E_TESTS=basic ./hack/e2e/ci-script.sh
```

## Test Structure

```text
test/e2e/
├── config/
│   └── e2e_conf.yaml       # Provider and image configuration
├── data/                    # Shared test data (metadata files)
├── e2e_suite_test.go        # Suite setup: bootstrap cluster, install providers
├── e2e_config.go            # Config parsing helpers
├── basic_test.go            # Basic label tests
├── feature_test.go          # Features label tests
└── common.go                # Shared helpers (logging, namespace, cleanup)
```

## Artifacts

Test artifacts are written to `test/e2e/_artifacts/`:

- JUnit XML report (`junit.e2e_suite.1.xml`)
- Controller logs
- Cluster resource dumps
- Kind cluster logs (when not using an existing cluster)

## Parallel Execution

Tests run with 2 parallel Ginkgo nodes by default. Each node uses its own
namespace (`ipam-e2e-1`, `ipam-e2e-2`) to avoid resource conflicts.
