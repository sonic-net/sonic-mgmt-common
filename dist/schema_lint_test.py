"""Run the owner's OpenConfig and IETF schema checks with pinned plugins."""
from pathlib import Path
import subprocess
import sys
import tempfile


def main():
    tool, models, oc_ignore, ietf_ignore = [Path(arg).absolute() for arg in sys.argv[1:]]
    with tempfile.TemporaryDirectory() as work:
        for family, ignore in [("oc", oc_ignore), ("ietf", ietf_ignore)]:
            subprocess.run([
                str(tool), "--" + family + "-linter", "-p", str(models),
                "--annotpath", str(models / "annotations"),
                "--ignorefile", str(ignore), "-o", str(Path(work) / (family + ".log")),
            ], check=True)


if __name__ == "__main__":
    main()
