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

import optparse
import os
import sys
import time

try:
    from pyang import Context, FileRepository
except ImportError:
    from pyang.context import Context
    from pyang.repository import FileRepository
from pyang.plugin import PyangPlugin

# Write logs to stderr..
log = sys.stderr.write
_repodir = None


def profile(func):
    def wrapper(*args, **kwars):
        try:
            t0 = time.monotonic()
            return func(*args, **kwars)
        finally:
            tt = time.monotonic() - t0
            log(f"{func.__name__} took {round(tt, 3)}s\n")
    return wrapper


def getpath(repo_rel: str) -> str:
    """Returns the file path for a repo relative path. Uses a relative path (to PWD)
    if we are inside the repo or its parent tree; otherwise returns an absolute path.
    """
    global _repodir
    if _repodir is None:
        d = os.path.abspath(os.path.dirname(__file__))
        while d != "/" and not os.path.exists(d+"/.git"):
            d = os.path.dirname(d)
        if d == "/":
            raise RuntimeError("Could not resolve repo directory")
        if os.path.commonpath([d, os.getcwd()]) != "/":
            d = os.path.relpath(d)
        _repodir = d
    p = os.path.join(_repodir, repo_rel)
    return os.path.relpath(p) if p[0] != "/" else p


def is_sonic_yangdir(path: str) -> bool:
    """Returns True if the path is a sonic yang directory (i.e, models/yang/sonic)"""
    f = os.path.join(path, "common", "sonic-extension.yang")
    return os.path.isfile(f)


def parse_argv(op: optparse.OptionParser, *plugins: PyangPlugin):
    """Adds few common options (like yang search path, logging) to the OptionParser
    and parses sys.argv. Returns the parsed values and positional arguments.
    """
    yang_sel = op.add_option_group("Yang path selection")
    yang_sel.add_option(
        "-p", "--path", dest="yangdir",
        help="Yang directory (defaults to {top}/build/yang)")
    yang_sel.add_option(
        "-c", "--commonpath", dest="yangcommondir",
        help="Not used if empty")
    yang_sel.add_option(
        "--basepath", dest="basedir",
        help="Yang directory with base yangs (defaults to {top}/models/yang)")
    yang_sel.add_option(
        "--annotpath", dest="annotdir",
        help="Yang directory with annotation (defaults to {top}/build/yang/annotations)")
    op.add_option(
        "-V", "--verbose", action="store_true",
        help="Enable verbose logs")
    for p in plugins:
        p.add_opts(op)
    opts, args = op.parse_args()
    if opts.yangdir and not os.path.isdir(opts.yangdir):
        op.error(f"{repr(opts.yangdir)} does not exists")
    if opts.basedir and not os.path.isdir(opts.basedir):
        op.error(f"{repr(opts.basedir)} does not exists")
    if opts.annotdir and not os.path.isdir(opts.annotdir):
        op.error(f"{repr(opts.annotdir)} does not exists")
    return opts, args


def create_context(opts) -> Context:
    """Returns a new pyang context object to work with yangs
    at opts.yangdir directory.
    """
    path = opts.yangdir if opts.yangdir != "." else os.getcwd()
    if opts.yangcommondir:
        path = f"{path}:{opts.yangcommondir}"
    repo = FileRepository(path, use_env=False, verbose=opts.verbose)
    ctx = Context(repo)
    ctx.strict = True
    ctx.opts = opts
    return ctx


def ensure_plugin_options(ctx: Context, *plugins: PyangPlugin):
    """Fills default values of plugin's options into ctx.opts
    if they are not already present.
    """
    temp = optparse.OptionParser()
    for p in plugins:
        p.add_opts(temp)
    for k, v in temp.get_default_values().__dict__.items():
        ctx.opts.ensure_value(k, v)


def list_api_yangs(yangdir, prefix=""):
    """Returns list of API yang names from a yang root directory.
    Yangs present directly under {yangdir} and {yangdir}/extensions
    directories are treated as API yangs.
    """
    from glob import glob
    pattern = prefix + "*.yang"
    std_yangs = glob(os.path.join(yangdir, pattern))
    ext_yangs = glob(os.path.join(yangdir, "extensions", pattern))
    return std_yangs + ext_yangs

def list_annot_yangs(annotdir, prefix=""):
    """Returns list of API yang names from a yang root directory.
    Yangs present directly under {yangdir} and {yangdir}/extensions
    directories are treated as API yangs.
    """
    from glob import glob
    pattern = prefix + "*.yang"
    annot_yangs = glob(os.path.join(annotdir, pattern))
    annot_yangs += glob(os.path.join(annotdir, "sonic-extensions.yang"))
    return annot_yangs

def load_yangs(ctx: Context, yangfiles: list) -> list:
    """Parse specified yang modules and load into the pyang context.
    Returns the list of loaded modules. Parser errors, if any, will
    be recorded in the context.
    """
    modules = []
    err_count = 0
    for y in yangfiles:
        if "*" in y:
            continue
        if not y.endswith(".yang"):
            continue
        if ctx.opts.verbose:
            log(f"Loading {y} ...\n")
        with open(y, encoding="utf-8") as f:
            text = f.read()
        mod = ctx.add_module(y, text)
        if mod:
            modules.append(mod)
        else:
            err_count += 1
    log(f"Loaded {len(modules)} yang modules, {err_count} failed\n")
    return modules

def load_annot_yangs(ctx: Context, yangfiles: list) -> list:
    """Parse annot-yang modules and load into the pyang context.
    Returns the list of loaded modules. Parser errors, if any, will
    be recorded in the context.
    """
    modules = []
    err_count = 0
    for y in yangfiles:
        if not y.endswith("annot.yang"):
            continue
        if ctx.opts.verbose:
            log(f"Loading {y} ...\n")
        with open(y, encoding="utf-8") as f:
            text = f.read()
        mod = ctx.add_module(y, text)
        if mod:
            ctx.deviation_modules.append(mod)
            modules.append(mod)
        else:
            err_count += 1
    log(f"Loaded {len(modules)} yang modules, {err_count} failed\n")
    return modules
