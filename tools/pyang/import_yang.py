#! /usr/bin/env python3
################################################################################
#                                                                              #
#  Copyright 2023 Broadcom. The term Broadcom refers to Broadcom Inc. and/or   #
#  its subsidiaries.                                                           #
#                                                                              #
#  Licensed under the Apache License, Version 2.0 (the "License");             #
#  you may not use this file except in compliance with the License.            #
#  You may obtain a copy of the License at                                     #
#                                                                              #
#     http://www.apache.org/licenses/LICENSE-2.0                               #
#                                                                              #
#  Unless required by applicable law or agreed to in writing, software         #
#  distributed under the License is distributed on an "AS IS" BASIS,           #
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.    #
#  See the License for the specific language governing permissions and         #
#  limitations under the License.                                              #
#                                                                              #
################################################################################

import argparse
import pathlib
import pyang

if pyang.__version__ > '2.4':
    from pyang.repository import FileRepository
    from pyang.context import Context
else:
    from pyang import FileRepository
    from pyang import Context
from pyang.error import err_level, is_error, err_to_str


def process(args):
    files = list()
    for f in [f for f in args.from_dir if not f.is_dir()]:
        print(f"error: {f} is not a directory")
        exit(1)
    for fname in args.files:
        src = get_file(fname, args.from_dir)
        if src is None:
            print(f"error: {fname} not found")
            exit(1)
        files.append(src)

    if len(files) == 0:
        return

    if not args.to_dir.exists():
        args.to_dir.mkdir()
    elif not args.to_dir.is_dir():
        print(f"error: {args.to_dir} is not a directory")
        exit(1)

    try:
        p = Preprocessor(files, args)
        p.process_all()
        p.write_all(args.to_dir)
    except Exception as e:
        print(f"error: {e}")
        exit(1)


def get_file(name, dirs) -> pathlib.Path:
    for d in dirs:
        f = d.joinpath(name)
        if f.exists():
            return f
    return None


class Preprocessor(object):
    def __init__(self, files, options):
        search_path = ":".join([str(f.absolute()) for f in options.from_dir])
        repo = FileRepository(search_path, use_env=False)
        ctx = Context(repo)
        ctx.keep_comments = options.keep_comments

        excludes = [f.stem for f in options.to_dir.glob("*.yang")]
        if options.exclude:
            excludes += [f.stem for f in options.exclude]

        modules = {}
        for f in files:
            if not f.exists() or f.stem in excludes:
                continue
            m = ctx.add_module(f.stem, f.read_text())
            modules[f.stem] = m

        ctx.validate()
        errors = list()
        for pos, tag, args in ctx.errors:
            if is_error(err_level(tag)):
                errors.append(f"{pos}: {err_to_str(tag, args)}")
        if errors:
            raise RuntimeError("\n".join(errors))

        self.options = options
        self.context = ctx
        self.cur_modules = excludes
        self.new_modules = modules
        self.dep_modules = {}
        self.yang_plugin = None

    def process_all(self):
        for mod in self.new_modules.values():
            self._process_module(mod)

    def _process_module(self, mod):
        self._subst_port_refs(mod)
        self._process_imports(mod)

        for i in mod.search("import"):
            if self._is_known_module(i.arg):
                continue
            imp = self.context.get_module(i.arg)
            if imp is None:
                continue
            self.dep_modules[i.arg] = imp
            self._process_module(imp)

    def _subst_port_refs(self, mod):
        if not self.options.subst_port_ref:
            return
        imp = mod.search_one("import", arg="sonic-port")
        if imp is None:
            return
        pfx = imp.search_one("prefix")
        q = pfx.arg if pfx else "sonic-port"
        p = f"/{q}:sonic-port/{q}:PORT/{q}:PORT_LIST/{q}:"
        old_leaf, new_leaf = self.options.subst_port_ref
        self._subst_yangpaths(mod, p+old_leaf, p+new_leaf)

    def _subst_yangpaths(self, stmt, old, new):
        if stmt.keyword in ["path", "must", "when"]:
            stmt.arg = stmt.arg.replace(old, new)
            return
        for ss in stmt.substmts:
            self._subst_yangpaths(ss, old, new)

    def _process_imports(self, mod):
        # remove unused imports
        for unused in mod.i_unused_prefixes.values():
            mod.substmts.remove(unused)
        # remove revision-date from imports
        for i in mod.search("import"):
            rev = i.search_one("revision-date")
            if rev:
                i.substmts.remove(rev)

    def _is_known_module(self, name):
        return name.startswith("ietf-") \
            or name in self.cur_modules \
            or name in self.new_modules \
            or name in self.dep_modules

    def write_all(self, to_dir):
        for mod in self.new_modules.values():
            self._write_one(to_dir, mod)
        if self.dep_modules:
            dep_dir = to_dir.joinpath("common")
            if not dep_dir.exists():
                dep_dir.mkdir()
        for mod in self.dep_modules.values():
            self._write_one(dep_dir, mod)

    def _write_one(self, to_dir, mod):
        if self.yang_plugin is None:
            self._init_yang_plugin()
        f = to_dir.joinpath(mod.arg+".yang")
        with f.open(mode="w") as fd:
            print(f"Writing {f.relative_to(self.options.to_dir)}")
            self.yang_plugin.emit(self.context, [mod], fd)

    def _init_yang_plugin(self):
        import optparse
        from pyang.translators.yang import YANGPlugin

        op = optparse.OptionParser()
        yp = YANGPlugin()
        yp.add_opts(op)
        self.context.opts, _ = op.parse_args(args=[])
        yp.setup_fmt(self.context)
        self.yang_plugin = yp


def nameMap(s):
    pair = s.split(":")
    if len(pair) != 2 or len(list(filter(None, pair))) != 2:
        raise ValueError("not in OLD:NEW format")
    return (*pair,)


if __name__ == "__main__":
    ap = argparse.ArgumentParser(description="Copy selected sonic yangs from FROM_DIR to TO_DIR")
    ap.add_argument("--from", dest="from_dir", required=True, type=pathlib.Path, action="append",
                    help="Directory from where to copy the sonic yangs")
    ap.add_argument("--to", dest="to_dir", required=True, type=pathlib.Path,
                    help="Destination directory for the copied sonic yangs")
    ap.add_argument("--exclude", type=pathlib.Path, action="append",
                    help="Yang file/module names to be excluded")
    ap.add_argument("--subst-port-ref", metavar="OLD:NEW", type=nameMap,
                    help="Substitute port name leaf in the leafrefs and must expressions")
    ap.add_argument("--keep-comments", action="store_true",
                    help="Do not strip comments while copying the yangs")
    ap.add_argument("files", nargs="+", metavar="FILENAME",
                    help="Yang file names")
    args = ap.parse_args()
    process(args)
