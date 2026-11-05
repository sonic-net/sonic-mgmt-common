"""Exercise deterministic discovery through the original pyang tree command."""

import os
import shutil
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import yangtools


TREE = Path(sys.argv.pop(1)).resolve()
MODELS = Path(sys.argv.pop(1)).resolve()


class DiscoveryTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)

    def models(self, name, reverse=False):
        root = self.root / name
        (root / "extensions").mkdir(parents=True)
        (root / "annotations").mkdir()
        (root / "common").mkdir()
        names = ["api-alpha", "api-zulu", "extensions/ext-alpha", "extensions/ext-zulu"]
        for relative in reversed(names) if reverse else names:
            module = Path(relative).name
            (root / (relative + ".yang")).write_text(
                'module %s { yang-version 1.1; namespace "urn:%s"; prefix x; '
                'container %s { leaf preserved { type string; } } }\n'
                % (module, module, module)
            )
        return root

    def generate(self, root, explicit=(), reverse_discovery=False, shared=None):
        shared = shared or root
        text = root / "tree.txt"
        html = root / "tree.html"
        env = dict(os.environ, PYTHONHASHSEED="0", PYTHONNOUSERSITE="1")
        env["PYTHONPATH"] = os.pathsep.join(sys.path)
        # Directory enumeration order is filesystem-specific. Reorder only the
        # real glob results so both permitted orders are exercised everywhere;
        # the original command still parses and renders every actual YANG file.
        bootstrap = (
            "import glob,runpy,sys; original=glob.glob; "
            "reverse=bool(int(sys.argv.pop(1))); "
            "glob.glob=lambda *a,**k: sorted(original(*a,**k),reverse=reverse); "
            "sys.argv=sys.argv[1:]; runpy.run_path(sys.argv[0],run_name='__main__')"
        )
        subprocess.run(
            [sys.executable, "-c", bootstrap, str(int(reverse_discovery)), str(TREE),
             "-p", str(root), "-c", str(shared / "common"),
             "--annotpath", str(shared / "annotations"), "--text", str(text),
             "--html", str(html), *map(str, explicit)],
            env=env, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        )
        return text.read_bytes(), html.read_bytes()

    def test_creation_order_does_not_change_text_or_html(self):
        forward = self.models("forward")
        reverse = self.models("reverse", reverse=True)
        first = self.generate(forward)
        second = self.generate(reverse, reverse_discovery=True)
        self.assertEqual(first, second)
        for module in ("api-alpha", "api-zulu", "ext-alpha", "ext-zulu"):
            self.assertIn(("module: " + module).encode(), first[0])
            self.assertIn(module.encode(), first[1])
        self.assertEqual(first[0].count(b"preserved"), 4)

    def test_actual_sonic_models_match_explicit_canonical_generation(self):
        roots = []
        files = [p for p in MODELS.rglob("*") if p.is_file()]
        self.assertTrue(files)
        for reverse in (False, True):
            root = self.root / ("reverse" if reverse else "forward")
            for source in sorted(files, reverse=reverse):
                target = root / source.relative_to(MODELS)
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(source, target)
            roots.append(root)
        for family in ("", "sonic"):
            with self.subTest(family=family or "openconfig"):
                first = self.generate(roots[0] / family, shared=roots[0])
                second = self.generate(roots[1] / family, reverse_discovery=True, shared=roots[1])
                self.assertEqual(first, second)
                api = roots[0] / family
                explicit = sorted(api.glob("*.yang")) + sorted((api / "extensions").glob("*.yang"))
                self.assertTrue(explicit)
                canonical = self.generate(api, explicit, shared=roots[0])
                self.assertEqual(first, canonical)

    def test_root_then_extension_order_and_prefix_filter(self):
        root = self.models("models", reverse=True)
        actual = [Path(p).relative_to(root).as_posix() for p in yangtools.list_api_yangs(str(root))]
        self.assertEqual(actual, ["api-alpha.yang", "api-zulu.yang",
                                  "extensions/ext-alpha.yang", "extensions/ext-zulu.yang"])
        self.assertEqual([Path(p).name for p in yangtools.list_api_yangs(str(root), "api-")],
                         ["api-alpha.yang", "api-zulu.yang"])

    def test_annotation_order_keeps_sonic_extensions_last(self):
        for name in ("openconfig-zulu-annot.yang", "sonic-extensions.yang", "openconfig-alpha-annot.yang"):
            (self.root / name).write_text(name)
        self.assertEqual([Path(p).name for p in yangtools.list_annot_yangs(str(self.root), "openconfig")],
                         ["openconfig-alpha-annot.yang", "openconfig-zulu-annot.yang", "sonic-extensions.yang"])

    def test_explicit_file_order_remains_callers_choice(self):
        root = self.models("explicit")
        text, _ = self.generate(root, [root / "api-zulu.yang", root / "api-alpha.yang"])
        self.assertLess(text.index(b"module: api-zulu"), text.index(b"module: api-alpha"))
        self.assertNotIn(b"module: ext-", text)


if __name__ == "__main__":
    unittest.main()
