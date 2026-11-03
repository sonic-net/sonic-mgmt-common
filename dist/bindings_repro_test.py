"""Regenerate actual owner bindings independently, bypassing action caches."""

from pathlib import Path
import hashlib
import os
import shutil
import subprocess
import sys
import tempfile


def main():
    driver, generator, models = [Path(p).absolute() for p in sys.argv[1:]]
    with tempfile.TemporaryDirectory() as temporary:
        outputs = [Path(temporary) / (str(i) + ".go") for i in range(2)]
        for iteration in range(20):
            output = outputs[0] if iteration == 0 else outputs[1]
            subprocess.run([str(driver), "--generator", str(generator),
                            "--models", str(models), "--output", str(output)], check=True)
            text = output.read_text()
            # openconfig-platform.yang declares hardware before software in
            # this union. Its generated reflection candidates must retain that
            # precedence; alphabetical sorting is not the schema contract.
            types = text.split('"/components/component/state/type": []reflect.Type{', 1)[1].split("}", 1)[0]
            ordered = types.index("HARDWARE_COMPONENT") < types.index("SOFTWARE_COMPONENT")
            identical = output.read_bytes() == outputs[0].read_bytes()
            if not ordered or not identical:
                evidence = Path(os.environ.get("TEST_UNDECLARED_OUTPUTS_DIR", temporary))
                for i, path in enumerate(outputs):
                    if path.exists():
                        shutil.copyfile(path, evidence / ("bindings-" + str(i) + ".go"))
                raise AssertionError("Binding regeneration %d changed bytes or YANG union precedence" % iteration)
        print("20 independent generator processes preserved bytes and declared union order; SHA256="
              + hashlib.sha256(outputs[0].read_bytes()).hexdigest())


if __name__ == "__main__":
    main()
