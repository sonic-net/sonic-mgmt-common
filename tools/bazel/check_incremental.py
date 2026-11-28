"""Measure a source schema edit, a no-op build, and byte-identical restoration."""

import hashlib
import json
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import time
import tempfile


TARGETS = ["//dist:runtime", "//dist:codegen", "//translib", "//cvl",
           "//translib/ocbinds:generated_bindings"]
OUTPUTS = ["dist/runtime.tar", "dist/codegen.tar", "translib/ocbinds/ocbinds.go"]


def main():
    command = shlex.split(os.environ.get("BAZEL_COMMAND", "bazel"))
    flags = shlex.split(os.environ.get("BAZEL_BUILD_FLAGS", ""))
    evidence = Path(os.environ.get("BAZEL_EVIDENCE", "bazel-evidence"))
    evidence.mkdir(parents=True, exist_ok=True)
    records = {}

    def build(name):
        started = time.monotonic()
        with (evidence / (name + ".log")).open("w") as log:
            subprocess.run(command + ["build", *flags, *TARGETS],
                           stdout=log, stderr=subprocess.STDOUT, check=True)
        records[name] = {"seconds": round(time.monotonic() - started, 3), "sha256": {
            p: hashlib.sha256((Path("bazel-bin") / p).read_bytes()).hexdigest() for p in OUTPUTS
        }}
        return records[name]["sha256"]

    baseline = build("baseline")
    assert build("noop") == baseline
    source = Path("models/yang/openconfig-system.yang")
    original = source.read_bytes()
    marker = original.rfind(b"}")
    assert marker > 0
    changed = False
    try:
        # A new real schema node changes generated Go bindings as well as the
        # installed models and documentation. It is removed in all outcomes.
        source.write_bytes(original[:marker] + b"\n  container bazel-incremental-probe {\n"
                           b"    leaf value { type string; }\n  }\n" + original[marker:])
        edited = build("schema_edit")
        assert all(edited[p] != baseline[p] for p in OUTPUTS), records
        changed = True
    finally:
        source.write_bytes(original)
        restored = build("schema_restore")
        assert restored == baseline, records
        (evidence / "incremental.json").write_text(json.dumps(records, indent=2) + "\n")
    assert changed
    # Capture only the verified production configuration, before another
    # invocation can select the optional test-model output tree.
    for relative in OUTPUTS[:2]:
        # Bazel outputs are read-only. Publish a fresh evidence file atomically
        # so reusing this evidence directory never rewrites a read-only copy.
        with tempfile.NamedTemporaryFile(dir=evidence, delete=False) as temporary:
            with (Path("bazel-bin") / relative).open("rb") as source_file:
                shutil.copyfileobj(source_file, temporary)
        os.replace(temporary.name, evidence / Path(relative).name)
    print(json.dumps(records, indent=2))


if __name__ == "__main__":
    main()
