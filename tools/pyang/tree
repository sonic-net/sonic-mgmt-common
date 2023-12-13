#!/usr/bin/env python3
################################################################################
#                                                                              #
#  Copyright 2022 Broadcom. The term Broadcom refers to Broadcom Inc. and/or   #
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

try:
    from pyang import Context
except ImportError:
    from pyang.context import Context
from pyang.error import err_level, is_error, err_to_str

import optparse
import os
import sys
import yangtools
from yangtools import log

plugins = {}
modules = []

def main():
    init_plugins()
    opts, files = parse_args()
    ctx = create_pyang_context(opts)
    load_yangs(ctx, files)
    validate_context(ctx)
    if ctx.opts.text:
        write_tree(ctx, "text", ctx.opts.text)
    if ctx.opts.html:
        write_tree(ctx, "html", ctx.opts.html)
    if ctx.opts.markdown:
        prepare_markdown_tree_ctx(ctx)
        write_tree(ctx, "markdown", ctx.opts.markdown)
    if ctx.opts.annot:
        load_annot_yangs(ctx, files)
        validate_context(ctx)
        write_tree(ctx, "annot", ctx.opts.annot)

def init_plugins():
    from pyang.plugins import tree, jstree
    from pyang_plugins.annot_tree import annotTreePlugin
    from pyang_plugins.doctree import docApiPlugin

    global plugins
    plugins["text"] = tree.TreePlugin()
    plugins["html"] = jstree.JSTreePlugin()
    plugins["markdown"] = docApiPlugin()
    plugins["annot"] = annotTreePlugin()


def parse_args():
    op = optparse.OptionParser(
        usage="%prog [options]",
        description="Generates yang trees for the API yangs in the root directory specified " +
                    "by the -p/--path option. Can generate the standard pyang's text tree " +
                    "html tree and a markdown version of the text tree. Markdown tree also " +
                    "shows the deviations from the base models if --basepath option is given." +
                    "Can also generate the text tree with annotations",
    )
    tree_sel = op.add_option_group("Tree format and output file selection")
    tree_sel.add_option(
        "--text", metavar="FILE",
        help="Write pyang text tree to FILE")
    tree_sel.add_option(
        "--html", metavar="FILE",
        help="Write pyang html tree to FILE")
    tree_sel.add_option(
        "--markdown", metavar="FILE",
        help="Write markdown yang tree to FILE")
    tree_sel.add_option(
        "--annot", metavar="FILE",
        help="Write pyang text tree with annotations to FILE")

    opts, filenames = yangtools.parse_argv(op)
    if not any([opts.text, opts.html, opts.markdown, opts.annot]):
        opts.text = "-"
    if not opts.yangdir:
        opts.yangdir = yangtools.getpath("build/yang")
    if not opts.annotdir:
        opts.annotdir = yangtools.getpath("build/yang/annotations")
    return opts, filenames


def create_pyang_context(opts) -> Context:
    ctx = yangtools.create_context(opts)
    yangtools.ensure_plugin_options(ctx, *plugins.values())
    setattr(ctx.opts, "tree_print_yang_data", False)
    setattr(ctx.opts, "tree_print_structures", False)
    return ctx


@ yangtools.profile
def load_yangs(ctx: Context, yangfiles: list):
    if not yangfiles:
        yangfiles = yangtools.list_api_yangs(ctx.opts.yangdir)
        log(f"Discovered {len(yangfiles)} yangs from {ctx.opts.yangdir}\n")
    global modules
    modules = yangtools.load_yangs(ctx, yangfiles)

@ yangtools.profile
def load_annot_yangs(ctx: Context, yangfiles: list):
    if not yangfiles:
        yangfiles = yangtools.list_annot_yangs(ctx.opts.annotdir, "openconfig")
        log(f"Discovered {len(yangfiles)} yangs from {ctx.opts.annotdir}\n")
    global modules
    modules += yangtools.load_annot_yangs(ctx, yangfiles)

@ yangtools.profile
def validate_context(ctx: Context):
    ctx.validate()
    for m in modules:
        m.prune()
    # Print errors and exit if there are any.. Ignore warnings
    num_errs = 0
    for pos, tag, args in ctx.errors:
        if is_error(err_level(tag)):
            kind = "error"
            num_errs += 1
        else:
            kind = "warning"
        if kind == "error" or ctx.opts.verbose:
            msg = err_to_str(tag, args)
            sys.stderr.write(f"{pos}: {kind}: {msg}\n")
    if num_errs != 0:
        sys.exit(1)


def prepare_markdown_tree_ctx(ctx: Context):
    # Guess base & extensions directory
    is_sonic_yangdir = yangtools.is_sonic_yangdir(ctx.opts.yangdir)
    basedir = ctx.opts.yangdir
    extndir = None
    if not basedir:
        if is_sonic_yangdir:
            basedir = yangtools.getpath("models/yang/sonic")
        else:
            basedir = yangtools.getpath("models/yang")
    if not is_sonic_yangdir:
        extndir = os.path.join(ctx.opts.yangdir,"extensions")
    # Set ctx.opts as pyang would have set
    def f(x): return f"{x}/common:{x}:{x}/extensions" if x else None
    def f_without_ext(x): return f"{x}/common:{x}" if x else None
    setattr(ctx.opts, "path", [f(ctx.opts.yangdir)])
    setattr(ctx.opts, "basepaths", f_without_ext(basedir))
    setattr(ctx.opts, "extdir", extndir)
    if ctx.opts.verbose:
        log(f"markdown_ctx.path      = {ctx.opts.path}\n")
        log(f"markdown_ctx.basepaths = {ctx.opts.basepaths}\n")
        log(f"markdown_ctx.extdir    = {ctx.opts.extdir}\n")

@ yangtools.profile
def write_tree(ctx, kind, outfile):
    tmpfile = None
    if outfile == "-":
        outfile = "<stdout>"
        fd = sys.stdout
    else:
        fd = open(outfile+".tmp", mode="w+")
        tmpfile = fd.name
    try:
        log(f"Writing {kind} tree to {outfile} ...\n")
        plugin = plugins[kind]
        plugin.emit(ctx, modules, fd)
        if tmpfile:
            fd.close()
            os.rename(tmpfile, outfile)
    except:
        if tmpfile:
            fd.close()
            os.remove(tmpfile)
        raise


if __name__ == "__main__":
    main()
