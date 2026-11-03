"""Stage the existing Debian install manifest as a filesystem tree for TAR."""

import argparse
from pathlib import Path
import shutil


def main():
    parser = argparse.ArgumentParser()
    for name in ["models", "schema", "models-list", "version", "config", "output"]:
        parser.add_argument("--" + name, type=Path, required=True)
    parser.add_argument("--test-model", type=Path, action="append", default=[])
    args = parser.parse_args()
    yang = args.output / "usr/models/yang"
    yang.mkdir(parents=True, exist_ok=True)
    # Match debian/sonic-mgmt-common.install's directory order. Equal duplicate
    # payloads are safe; conflicting model basenames would break the flat ABI.
    for relative in [".", "common", "extensions", "sonic", "sonic/common", "annotations"]:
        for source in sorted((args.models / relative).glob("*.yang")):
            destination = yang / source.name
            if destination.exists() and destination.read_bytes() != source.read_bytes():
                raise ValueError(f"Conflicting installed model: {source.name}")
            shutil.copyfile(source, destination)
    extra = "".join("-" + path.name + "\n" for path in sorted((args.models / "sonic/common").glob("*.yang")))
    extra += "".join(path.name + "\n" for path in sorted(args.test_model))
    (yang / "models_list").write_text(args.models_list.read_text() + extra)
    shutil.copyfile(args.version, yang / "version.xml")
    sbin = args.output / "usr/sbin"
    sbin.mkdir(parents=True, exist_ok=True)
    shutil.copytree(args.schema, sbin / "schema", dirs_exist_ok=True)
    shutil.copyfile(args.config, sbin / "cvl_cfg.json")
    for path in args.output.rglob("*"):
        path.chmod(0o755 if path.is_dir() else 0o644)


if __name__ == "__main__":
    main()
