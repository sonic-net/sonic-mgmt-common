"""Check the produced TAR against the existing package install contract."""

from pathlib import Path
import sys
import tarfile
import tempfile


def main():
    archive, models, schema, manifest, model_list, version, config = map(Path, sys.argv[1:])
    with tempfile.TemporaryDirectory() as temporary:
        prepared_list = Path(temporary) / "models_list"
        prepared_list.write_text(model_list.read_text() + "".join(
            "-" + path.name + "\n" for path in sorted((models / "sonic/common").glob("*.yang"))))
        roots = {
            "build/yang": models,
            "build/cvl/schema": schema,
            "build/transformer/models_list": prepared_list,
            "models/yang/version.xml": version,
            "cvl/conf/cvl_cfg.json": config,
        }
        expected = {}
        for line in manifest.read_text().splitlines():
            if not line.strip() or line.lstrip().startswith("#"):
                continue
            source, destination = line.split()
            matches = None
            for old, real in roots.items():
                if source == old:
                    matches = [real]
                    break
                if source.startswith(old + "/"):
                    matches = sorted(real.glob(source[len(old) + 1:]))
                    break
            assert matches, f"Unresolved legacy install input: {source}"
            for path in matches:
                files = sorted(path.rglob("*")) if path.is_dir() else [path]
                for file in files:
                    if file.is_file():
                        name = str(Path(destination) / file.name)
                        if path.is_dir():
                            name = str(Path(destination) / path.name / file.relative_to(path))
                        expected[name] = file.read_bytes()
        with tarfile.open(archive) as tar:
            members = tar.getmembers()
            assert len(members) == len({m.name for m in members}), "Duplicate archive entries"
            actual = {m.name: tar.extractfile(m).read() for m in members if m.isfile()}
            assert actual.keys() == expected.keys(), (actual.keys() - expected.keys(), expected.keys() - actual.keys())
            for name, data in expected.items():
                assert actual[name] == data, name
            for member in members:
                assert member.uid == member.gid == 0, member.name
                assert member.mode == (0o755 if member.isdir() else 0o644), member.name
                assert member.mtime == 1672560000, member.name
                assert not member.issym() and not member.islnk(), member.name
                assert not member.name.startswith("/") and ".." not in Path(member.name).parts
            assert any(name.endswith(".yin") for name in actual)
            assert any(name.endswith(".yang") for name in actual)
        print(f"Verified {len(expected)} runtime files against the legacy install manifest")


if __name__ == "__main__":
    main()
