# AMD64 management-common with Bazel

This build compiles this checkout's CVL and translib Go code and generates its
YANG bindings, validation schemas, and install trees. It targets native AMD64
Debian Trixie with Bazel 8.5.1, Go 1.25.9, the shared SONiC GCC toolchain and
Python 3.13. It does not generate Debian packages.

| Consumer interface | Output |
| --- | --- |
| `//cvl`, `//translib`, `//translib/ocbinds` | Source-owned Go library providers, including generated bindings |
| `//dist:runtime` | Reproducible TAR: `/usr/models/yang`, `/usr/sbin/schema`, `/usr/sbin/cvl_cfg.json` |
| `//dist:codegen` | Reproducible TAR of prepared YANG, generated Go bindings, and tree/HTML documentation for build consumers |
| `//translib/ocbinds:generated_bindings` | Generated `ocbinds.go`, outside the source tree |

The runtime and code-generation TAR tests compare all installed files against
the existing `debian/*.install` contracts, including their bytes and archive
metadata. All payloads are root-owned with fixed timestamps and normalized
permissions. Downstream management-framework and gNMI should depend on these
owner targets and use the runtime TAR in their container composition.

## Build and test

The maintained registry configured in `.bazelrc` supplies every module,
including the merged OpenConfig linter registration from
[registry PR #163](https://github.com/securely1g/sonic-bazel-registry/pull/163):

```sh
bazel build //cvl //translib //dist:runtime //dist:codegen
bazel test //cvl:cvl_test //translib:translib_test //translib/db:db_test \
  //translib/cs:cs_test //translib/path:path_test //translib/platform:platform_test \
  //translib/internal/apis:apis_test //translib/ocbinds:ocbinds_test \
  //translib/transformer:transformer_test \
  //dist:runtime_test //dist:codegen_contract_test //dist:schema_lint_test \
  //dist:bindings_repro_test
bazel test --config=xfmr //translib/transformer:transformer_model_test
```

Local builds and CI use the same maintained `main` registry and BCR endpoints.
CI copies the committed configuration together with its configured cache paths;
it does not replace the registry with a review snapshot.

`--config=xfmr` selects the existing transformer's `xfmrtest` Go tags and
`INCLUDE_TEST_MODELS=y` schema inputs together. Default builds do not contain
these test models. The default transformer suite uses the existing `testapp`
tests. Experimental prune tag variants are outside this initial configuration.

Tests start private Redis instances with isolated sockets, ports, data and
configuration. The declared Redis and libyang runtime uses its matching Trixie
loader; tests never stop or modify a worker's Redis service. Existing CVL
assertions run against the same generated YIN and JSON fixtures used by Make.
The other Redis suites reuse `tools/test/dbinit.py` and `platform.json`.

## Sources and locks

Model import policy comes from `models/yang/sonic/import.mk`. The shared
`sonic-yang-models` owner builds its wheel; management-common consumes its CVL
models and applies the existing `import_yang.py` transformations. The existing
patched ygot generator produces Go bindings, and the existing owner pyang tools
produce lint results and documentation. No action clones dependencies or
mutates `vendor`, `go.mod`, or a generated source file in the checkout.

Go inputs come from `go.mod`/`go.sum`, with the existing vendor patch set applied
inside external repositories in the shared Go dependency graph. Bazel's shared Go dependency graph also
contains the rules_go tool dependencies; it selects `golang.org/x/text` 0.40.0
over the source module's 0.39.0 minimum. Source `go.mod` is unchanged. The generated
module resolution and test artifacts record the actual selected graph.

Python tool dependencies are pinned with hashes in `requirements_lock.txt`.
`tools/test/apt.lock.json` freezes the AMD64 Redis 8.0.2 test runtime and its
40-package dependency closure by source URL and SHA256. These external Debian
packages are imported, not produced. To refresh the test lock, resolve only
`mgmt_test_deps` through the selected rules_distroless `apt.install` using the
same immutable Trixie snapshots, export that set's generated lock content, and
rerun all private-Redis suites. Do not replace it with live host packages.

`MODULE.bazel.lock` is generated, ignored, and retained as CI evidence. Bazel's
local action and repository caches can be reused; no clean is needed between
edits. CI starts from a source checkout without generated files and records
the selected inputs, exact TAR hashes, native architecture, commit parents,
test logs, and schema-edit timings in its downloadable artifact.

Run `python3 tools/bazel/check_incremental.py` to measure a normal build, a
no-op, a schema-node edit, and restoration. It verifies that the runtime TAR,
code-generation TAR and Go bindings change after the edit and return to their
original SHA256 values after restoration. Source bytes are restored even when
validation fails. `BAZEL_COMMAND` and `BAZEL_BUILD_FLAGS` accept local cache
configuration; their use does not change the checked-in build contract.

## Consuming the public Go libraries

The consuming root selects Go 1.25.9, Python 3.13 and the same SONiC dependency
patches. Gazelle deliberately permits dependency overrides only in the root
module. This repository applies them through a development-only extension use
when it is the root; a downstream root applies the exported patch labels to its
single shared `go_deps` extension. See [the checked consumer module](tests/consumer/MODULE.bazel)
and its [Go replacements](tests/consumer/go.mod). It compiles against the public
CVL, translib and generated bindings without copying their source. In particular,
retain the pinned glog replacement required by the existing glog patch.

CI also builds this separate consumer workspace with `--override_module` pointing
to the tested source checkout. Registry validation replaces that local override
with the immutable source registration. The consumer-root configuration is not
an isolated second copy of the owner's Go dependency graph.

The inherited patched YANG generator used map iteration for ordered child lists,
identity inheritance, cross-module augments, and union enum reflection candidates. The additional goyang patch keeps
YANG declaration order and makes the unordered traversals stable. The ygot repair follows its stored union subtype indexes, preserving the declared hardware-before-software precedence. The binding
reproducibility test runs the actual generator in 20 independent processes and
compares all output bytes, so restoring an action-cache entry cannot mask a
nondeterministic generator.

CI copies the setup action's disk and repository cache settings alongside the
committed registry configuration. The owner and consumer keep separate output
bases, so validating the consumer preserves the owner's retained test results.
