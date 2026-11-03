"""Stage the generated-code contract as a build input, without installing it."""
import argparse
from pathlib import Path
import shutil

p = argparse.ArgumentParser()
p.add_argument("--models", type=Path, required=True)
p.add_argument("--bindings", type=Path, required=True)
p.add_argument("--doc", type=Path, action="append", required=True)
p.add_argument("--output", type=Path, required=True)
a = p.parse_args()
shutil.copytree(a.models, a.output / "build/yang")
for source, relative in [(a.bindings, "translib/ocbinds/ocbinds.go")] + [
    (doc, "models/yang/" + doc.name) for doc in a.doc
]:
    output = a.output / relative
    output.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, output)
