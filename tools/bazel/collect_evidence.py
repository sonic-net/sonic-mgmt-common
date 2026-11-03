"""Retain source identity, exact TARs, generated resolution and real test logs."""

import hashlib
import json
import os
from pathlib import Path
import platform
import shutil
import subprocess
import xml.etree.ElementTree as ET


def main():
    root = Path("bazel-evidence")
    root.mkdir(exist_ok=True)
    if os.environ.get("CI_INCREMENTAL_RESULT") == "success":
        for name in ["runtime.tar", "codegen.tar", "incremental.json"]:
            if not (root / name).is_file():
                raise FileNotFoundError("Missing verified production evidence: " + name)
    for name in ["MODULE.bazel.lock", "tools/test/apt.lock.json", "requirements_lock.txt", "go.mod", "go.sum"]:
        source = Path(name)
        if source.is_file():
            destination = root / "inputs" / name
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)
    # check_incremental.py already retained the byte-verified production TARs.
    # bazel-bin may now name an optional test-model configuration, particularly
    # on a failed xfmr run, so never infer artifact provenance from that link.
    logs = list(Path("bazel-testlogs").rglob("test.*"))
    failed_bindings = list(Path("bazel-testlogs").glob("**/test.outputs/bindings-*.go"))
    for source in logs + failed_bindings:
        if (source not in failed_bindings and source.name not in ("test.log", "test.xml")) or not source.is_file():
            continue
        destination = root / "testlogs" / source.relative_to("bazel-testlogs")
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, destination)
    # These direct rules_go suites need verbose output for per-case XML.
    # A successful process with an empty report is insufficient evidence.
    if os.environ.get("CI_DEFAULT_TEST_RESULT") == "success":
        for suite in ("ocbinds", "path", "platform"):
            report = root / "testlogs/translib" / suite / (suite + "_test") / "test.xml"
            cases = ET.parse(report).getroot().findall(".//testcase")
            if not cases or any(case.find("failure") is not None or case.find("error") is not None for case in cases):
                raise ValueError("Missing or failed Go test cases: " + str(report))
    def git(*args):
        return subprocess.check_output(["git", "-c", "safe.directory=*", *args], text=True).strip()
    identity = {
        "checkout": git("rev-parse", "HEAD"),
        "commit_headers": git("cat-file", "-p", "HEAD").split("\n\n", 1)[0],
        "pull_request_head": os.environ.get("CI_HEAD_SHA", ""),
        "machine": platform.machine(),
        "os_release": Path("/etc/os-release").read_text(),
        "target": "@sonic_build_infra//platforms:x86_64_trixie",
        "bazel": Path(".bazelversion").read_text().strip(),
        "go_sdk": "1.25.9",
        "test_profiles": {
            "production/testapp": os.environ.get("CI_DEFAULT_TEST_RESULT", "unknown"),
            "xfmrtest with test models": os.environ.get("CI_XFMR_TEST_RESULT", "unknown"),
        },
        "external_consumer_result": os.environ.get("CI_CONSUMER_RESULT", "unknown"),
        "incremental_result": os.environ.get("CI_INCREMENTAL_RESULT", "unknown"),
        "artifacts": {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in root.glob("*.tar")},
    }
    (root / "provenance.json").write_text(json.dumps(identity, indent=2) + "\n")


if __name__ == "__main__":
    main()
