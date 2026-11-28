"""Apply models/yang/Makefile's import policy to declared source inputs."""

import argparse
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

from tools.pyang import import_yang


def preprocess(sources, destination, names, *, excluded=(), keep_comments=False, subst_port_ref=None):
    import_yang.process(argparse.Namespace(
        from_dir=sources,
        to_dir=destination,
        files=names,
        exclude=[Path(name) for name in sorted(excluded)],
        subst_port_ref=subst_port_ref,
        keep_comments=keep_comments,
    ))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--imports", type=Path, required=True)
    parser.add_argument("--owner-wheel", type=Path, required=True)
    parser.add_argument("--tar", type=Path, required=True)
    parser.add_argument("--source", type=Path, action="append", required=True)
    parser.add_argument("--test-source", type=Path, action="append", default=[])
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    source_root = args.imports.absolute().parent.parent
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    patterns = []
    for line in args.imports.read_text().splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        match = re.fullmatch(r"SONICYANG_IMPORTS\s*\+=\s*(\S+)", line.strip())
        if not match:
            raise ValueError(f"Unsupported model import declaration: {line!r}")
        patterns.append(match.group(1))
    with tempfile.TemporaryDirectory(dir=output.parent) as temporary:
        unpacked = Path(temporary)
        # The source owner has already rendered both py and CVL variants. BSD tar
        # extracts the exact generated wheel; no second model producer is run.
        subprocess.run([str(args.tar.resolve()), "-xf", str(args.owner_wheel.resolve()), "-C", temporary], check=True)
        candidates = list(unpacked.glob("*.data/data/cvlyang-models"))
        if len(candidates) != 1:
            raise ValueError("Owner wheel must contain exactly one CVL model directory")
        imported = candidates[0]
        names = sorted({p.name for pattern in patterns for p in imported.glob(pattern)})
        for pattern in patterns:
            if not list(imported.glob(pattern)):
                raise ValueError(f"Required owner model is missing: {pattern}")
        local_names = []
        for source in sorted(args.source):
            relative = source.absolute().relative_to(source_root)
            if relative.parent == Path("sonic"):
                local_names.append(relative.name)
                continue
            destination = output / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(source, destination)
        sonic_output = output / "sonic"
        sonic_output.mkdir(exist_ok=True)
        preprocess([source_root / "sonic", source_root / "common"], sonic_output,
                   local_names, excluded=set(names) & set(local_names), keep_comments=True,
                   subst_port_ref=("ifname", "name") if "sonic-port.yang" in names else None)
        preprocess([imported, source_root / "common"], sonic_output, names,
                   subst_port_ref=None if "sonic-port.yang" in names else ("name", "ifname"))
        for source in sorted(args.test_source):
            directory = "annotations" if source.name.endswith("-annot.yang") else (
                "sonic" if source.name.startswith("sonic-") else ".")
            shutil.copyfile(source, output / directory / source.name)


if __name__ == "__main__":
    main()
