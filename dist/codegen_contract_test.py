"""Compare the exported code-generation payload with the legacy contract."""
from pathlib import Path, PurePosixPath
import sys
import tarfile


def main():
    archive, bindings, models, manifest, *docs = [Path(arg) for arg in sys.argv[1:]]
    # The old install destinations share this source-tree prefix. TAR users
    # consume these as build inputs without installing anything on the worker.
    prefix = "sonic/src/sonic-mgmt-common/"
    expected = {}
    by_source = {"translib/ocbinds/ocbinds.go": bindings, "build/yang": models}
    by_source.update({"models/yang/" + doc.name: doc for doc in docs})
    for line in manifest.read_text().splitlines():
        source, destination = line.split()
        assert destination.startswith(prefix), line
        actual = by_source[source]
        target = PurePosixPath(destination[len(prefix):]) / PurePosixPath(source).name
        if actual.is_dir():
            expected.update({str(target / p.relative_to(actual)): p.read_bytes()
                             for p in actual.rglob("*") if p.is_file()})
        else:
            expected[str(target)] = actual.read_bytes()
    with tarfile.open(archive) as tar:
        members = tar.getmembers()
        actual = {m.name.removeprefix("./"): tar.extractfile(m).read() for m in members if m.isfile()}
        assert actual == expected, "Generated-code TAR differs from the legacy install contract"
        assert len({m.name for m in members}) == len(members), "Duplicate archive paths"
        for member in members:
            assert member.uid == member.gid == 0
            assert member.mtime == 1672560000
            assert member.mode == (0o755 if member.isdir() else 0o644)
            assert member.isdir() or member.isfile()
    assert b"/home/" not in bindings.read_bytes() and b"/sandbox/" not in bindings.read_bytes()
    print(f"Verified {len(expected)} source-owned code-generation files")


if __name__ == "__main__":
    main()
