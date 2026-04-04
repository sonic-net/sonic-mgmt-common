# Copilot Instructions for sonic-mgmt-common

## Project Overview

sonic-mgmt-common provides the shared libraries and YANG models used by the SONiC management framework. It contains CVL (Config Validation Library) for validating Redis DB entries against YANG models, the translib framework for translating REST/gNMI requests into DB operations, and the common YANG model definitions. This is the foundation layer that `sonic-mgmt-framework` builds upon.

## Architecture

```
sonic-mgmt-common/
├── cvl/                          # Config Validation Library (Go)
│   ├── cvl.go                    # CVL main implementation
│   ├── cvl_api.go                # Public CVL API
│   ├── cvl_semantics.go          # Semantic validation logic
│   ├── cvl_syntax.go             # Syntax validation logic
│   ├── cvl_cache.go              # Caching layer
│   ├── cvl_hint.go               # Validation hints
│   ├── cvl_luascript.go          # Lua scripts for Redis operations
│   ├── common/                   # Common utilities
│   ├── custom_validation/        # Custom validation handlers
│   ├── conf/                     # CVL configuration
│   ├── *_test.go                 # Extensive test suite
│   └── Makefile
├── translib/                     # Translation library (Go)
│   ├── app_interface.go          # Application interface definitions
│   ├── app_utils.go              # Application utilities
│   ├── db/                       # Database access layer
│   ├── ocbinds/                  # OpenConfig YANG bindings
│   ├── path/                     # Path parsing utilities
│   ├── cs/                       # Common subscription support
│   ├── internal/                 # Internal implementation
│   ├── authorize.go              # Authorization logic
│   ├── subscribe.go              # gNMI subscribe support
│   ├── request_binder.go         # Request binding and validation
│   ├── acl_app.go                # ACL application handler
│   ├── lldp_app.go               # LLDP application handler
│   ├── pfm_app.go                # Platform application handler
│   ├── sys_app.go                # System application handler
│   ├── common_app.go             # Common/generic application handler
│   ├── *_test.go                 # Unit tests
│   └── Makefile
├── models/                       # YANG model definitions
│   └── yang/                     # YANG files (OpenConfig + SONiC)
├── tools/                        # Code generation and utilities
│   ├── pyang/                    # pyang plugins for YANG processing
│   ├── xfmr/                    # Transformer utilities
│   └── test/                    # Test tools
├── config/                       # Configuration files
├── patches/                      # Vendor patches for Go dependencies
├── Makefile                      # Top-level build
└── go.mod                        # Go module definition
```

### Key Concepts
- **CVL (Config Validation Library)**: Validates Redis CONFIG_DB entries against YANG models before they are applied; enforces syntax, semantics, leafref, must expressions, and custom validation
- **translib**: Translates high-level REST/gNMI requests into Redis DB read/write operations using application-specific handlers
- **App interface**: Each feature (ACL, LLDP, Platform, etc.) implements the `appInterface` to handle CRUD operations
- **YANG models**: Define the schema for SONiC configuration and OpenConfig compatibility
- **Transformer (xfmr)**: Maps between YANG paths and Redis DB table/field names

## Language & Style

- **Primary language**: Go
- **Secondary**: YANG (models), Python (tools/pyang), Lua (Redis scripts)
- **Go conventions**:
  - Follow standard `gofmt` formatting
  - Types: `PascalCase` for exported, `camelCase` for unexported
  - Functions: `PascalCase` for exported, `camelCase` for unexported
  - Variables: `camelCase`
  - Constants: `PascalCase` or `UPPER_SNAKE_CASE`
  - File names: `snake_case.go`
- **YANG conventions**: Follow OpenConfig and SONiC YANG naming patterns
- **Test files**: `*_test.go` adjacent to the code they test

## Build Instructions

```bash
# Set up Go environment
export GOPATH=/tmp/go
export GO=/usr/local/go/bin/go

# Build everything (models, CVL, translib)
make all

# Build individual components
make models
make cvl
make translib

# The build process:
# 1. Runs `go mod vendor` and applies patches
# 2. Processes YANG models with goyang
# 3. Builds CVL and translib libraries

# Build Debian package
dpkg-buildpackage -us -uc -b
```

## Testing

```bash
# Run CVL tests
cd cvl
go test -v ./...

# Run translib tests
cd translib
go test -v ./...

# Specific test suites in CVL:
#   cvl_test.go          — Core validation tests
#   cvl_error_test.go    — Error handling tests
#   cvl_leafref_test.go  — Leafref validation tests
#   cvl_must_test.go     — YANG must expression tests
#   cvl_leaflist_test.go — Leaf-list tests
#   cvl_hint_test.go     — Validation hint tests
#   cvl_optimisation_test.go — Performance tests
#   cvl_cust_validation_test.go — Custom validation tests

# Tests may require Redis running locally
redis-server &
```

## PR Guidelines

- **Signed-off-by**: REQUIRED on all commits (`git commit -s`)
- **CLA**: Sign the Linux Foundation EasyCLA
- **Single commit per PR**: Squash commits before merge
- **YANG models**: New or modified YANG models must pass `pyang` validation
- **CVL tests**: Changes to validation logic must include test cases
- **Go formatting**: Run `gofmt` before submitting
- **Reference**: Link to SONiC management framework HLD documents

## Dependencies

- **Go**: Go compiler and standard library
- **goyang**: YANG parser for Go (vendored with patches)
- **pyang**: Python YANG validator (for model processing)
- **Redis**: Required for CVL validation operations and testing
- **libyang**: YANG data modeling library (used by CVL internally)
- **sonic-buildimage**: Provides build environment context
- **OpenConfig models**: Referenced YANG models for standards compliance

## Gotchas

- **Vendor patches**: Go vendor directory has patches applied via `patches/apply.sh` — running `go mod vendor` alone is not enough
- **YANG model ordering**: Models have import dependencies; the build system handles ordering, but manual `go test` may fail without proper model setup
- **Redis required for tests**: Most CVL and translib tests require a running Redis instance
- **Go module path**: Uses `github.com/Azure/sonic-mgmt-common` (Azure path, not sonic-net) in `go.mod`
- **Custom validation**: Adding new custom validators requires registering them in `cvl/custom_validation/`
- **Leafref validation**: CVL validates leafref constraints against live Redis data — test fixtures must set up referenced data
- **Build order**: `models` must build before `cvl` and `translib`
- **GOPATH sensitivity**: Build assumes specific `GOPATH` layout; follow the Makefile's environment setup
