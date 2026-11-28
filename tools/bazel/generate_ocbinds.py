"""Pass the prepared model inventory to the pinned, patched upstream ygot tool."""

import argparse
from pathlib import Path
import subprocess


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--generator", type=Path, required=True)
    parser.add_argument("--models", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    # ygot records its input arguments in the generated header. Keep paths
    # relative to the execution root so sandbox/cache locations never leak.
    models = args.models
    selected = sorted(path for path in models.rglob("*.yang")
                      if "annotations" not in path.relative_to(models).parts)
    if not selected:
        raise ValueError("The owner model set is empty")
    subprocess.run([
        str(args.generator.resolve()),
        "--logtostderr",
        "--output_file=" + str(args.output),
        "--package_name=ocbinds",
        "--generate_fakeroot",
        "--fakeroot_name=device",
        "--compress_paths=false",
        "--path=" + str(models),
        "--exclude_modules=ietf-interfaces,sonic-types,sonic-common,",
        *map(str, selected),
    ], check=True)


if __name__ == "__main__":
    main()
