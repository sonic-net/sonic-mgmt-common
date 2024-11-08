################################################################################
#                                                                              #
#  Copyright 2019 Broadcom. The term Broadcom refers to Broadcom Inc. and/or   #
#  its subsidiaries.                                                            #
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
import sys

from pyang import plugin
from pyang import error
from pyang import statements


def prepare_ignore_list(ctx,ignore_file_dict):
    ignore_file = ctx.opts.ignore_file
    if ignore_file is None:
        return None
    with open(ignore_file, "r") as ignore_fh:
        for entry in ignore_fh:
            entry = entry.strip()
            if entry.startswith('#'):
                continue
            entry_list = list(filter(None,entry.split(' ')))
            if len(entry_list) == 0:
                continue
            mod_name = entry_list[0]
            if mod_name not in ignore_file_dict:
                ignore_file_dict[mod_name] = set()
            for line_num in entry_list[1:]:
                ignore_file_dict[mod_name].add(int(line_num.strip()))
    return ignore_file_dict

def prepare_patched_mods_list(ctx,patched_mods):
    patch_listfile = ctx.opts.patch_listfile
    if patch_listfile is None:
        return None
    with open(patch_listfile) as fp:
        lines = fp.readlines()
        for patch_file in lines:
            patch_file = patch_file.strip()
            if patch_file.endswith('.yang'):
                mod_name = patch_file.replace('.yang','')
                patched_mods.add(mod_name)
    return patched_mods

def pyang_plugin_init():
    plugin.register_plugin(CheckStrictLintPlugin())
    plugin.register_plugin(OpenconfigExtraChecksPlugin())


class OpenconfigExtraChecksPlugin(plugin.PyangPlugin):
    """
    This plugin will include checks missed (or not handled correctly) by oc-linter
    """
    def check_leaf_mirroring_for_deviate_not_supported(self, ctx, modules):
        """
         oc-linter is performing leaf mirroring check at reference_2 stage, but not-supported deviation is only applied
         post validation. Any deviation removing mirror elements are not error reported
        """

        def handle_leaf(deviation):
            # Handle if the target is the leaf/leaf-list, provided they are part of state container
            target_node = deviation.i_target_node
            target_node_parent = target_node.parent
            if target_node.keyword in [u"leaf", u"leaf-list"] and target_node_parent.arg == "state":
                config_container = target_node_parent.parent.search_one("container", "config")
                if config_container and config_container.search_one(target_node.keyword, target_node.arg,
                                                                    children=config_container.i_children):
                    error.err_add(ctx.errors, deviation.pos, "OC_OPSTATE_APPLIED_CONFIG",
                                  (target_node.arg, statements.mk_path_str(config_container, False)))

        def handle_state_container(deviation):
            # Handle if the target is the state container
            target_node = deviation.i_target_node
            if target_node.keyword == "container" and target_node.arg == "state":
                config_container = target_node.parent.search_one("container", "config")
                if config_container:
                    for child in config_container.i_children:
                        if child.arg != "config" and child.keyword in [u"leaf", u"leaf-list", u"container", u"list",
                                                                       u"choice"]:
                            # Ideally we should only be caring leaf and leaf-list, but mimic oc-linter behavior here
                            error.err_add(ctx.errors, deviation.pos, "OC_OPSTATE_APPLIED_CONFIG",
                                          (child.arg, statements.mk_path_str(config_container, False)))

        def process(module):
            for deviation in module.search("deviation"):
                if deviation.search_one("deviate", "not-supported"):
                    handle_leaf(deviation)
                    handle_state_container(deviation)

        for module in modules:
            # Act only for deviations belonging to extension yang models
            if "/extensions/" in str(module.pos):
                process(module)

    def post_validate(self, ctx, modules):
        self.check_leaf_mirroring_for_deviate_not_supported(ctx, modules)


class CheckStrictLintPlugin(plugin.PyangPlugin):
    def __init__(self):
        self.errors = [] # 4-tuple error info (pos, tag, args, kind)
        self.ignore_info = None
        self.patched_mods = None
    
    def add_output_format(self, fmts):
        self.multiple_modules = True
        fmts['strictlint'] = self

    def add_opts(self, optparser):
        optlist = [
            optparse.make_option("--patchlistfile",
                                 type="string",
                                 dest="patch_listfile",
                                 help="File path containing list of patched yangs"),
            optparse.make_option("--ignorefile",
                                 type="string",
                                 dest="ignore_file",
                                 help="File path containing ignore list of modules"), 
            optparse.make_option("--errorsonly",
                                 action="store_true",
                                 help="Print errors only (hides warnings and ignored errors)"),
        ]
        g = optparser.add_option_group("CheckStrictLintPlugin options")
        g.add_options(optlist)

    def setup_fmt(self, ctx):
        ctx.implicit_errors = False

    def pre_validate(self, ctx, modules):
        # Collect yang load errors without applying ignore rules
        self.collect_errors(ctx)

    def post_validate(self, ctx, modules):
        # Collect linter errors after applying ignore rules
        self.ignore_info = prepare_ignore_list(ctx, dict())
        self.patched_mods = prepare_patched_mods_list(ctx, set())
        self.collect_errors(ctx)

    def emit(self, ctx, modules, fd):
        error_seen = False
        self.errors.sort(key=lambda x: (x[0].ref, x[0].line))
        if ctx.opts.outfile is not None:
            fd = open(ctx.opts.outfile, "w")        
            fd = TeeWriter(sys.stdout, fd)
        for epos, etag, eargs, kind in self.errors:
            if kind == "error":
                error_seen = True
            elif ctx.opts.errorsonly:
                continue
            fd.write(str(epos) + ': %s: ' % kind + \
                                    error.err_to_str(etag, eargs) + '\n')
        
        if ctx.opts.outfile is not None:
            fd.close()
        if error_seen:
            sys.exit(1)
        else:
            sys.exit(0)

    def collect_errors(self, ctx):
        for err in ctx.errors:
            kind = self.classify_error(ctx, err)
            if kind:
                self.errors.append((err[0], err[1], err[2], kind))
        ctx.errors = []

    def has_error(self):
        for err in self.errors:
            if err[3] == "error":
                return True
        return False

    def classify_error(self, ctx, err):
        pos, tag = err[0], err[1]
        lvl = error.err_level(tag)
        mod = pos.ref.split('/')[-1].split('.')[0]
        if "/extensions/" not in str(pos):
            if not self.patched_mods or mod not in self.patched_mods:
                return "ignored" if ctx.opts.verbose else None
        if error.is_warning(lvl):
            return "warning"
        if len(err) > 3 and err[3]:
            return "ignored"
        if self.ignore_info is not None and mod in self.ignore_info:
            ignore_entry = self.ignore_info[mod]
            if len(ignore_entry) == 0 or pos.line in ignore_entry:
                return "ignored"
        return "error"


class TeeWriter(object):
    def __init__(self, *fd_list):
        self.fd_list = fd_list

    def write(self, s):
        for fd in self.fd_list:
            fd.write(s)

    def close(self):
        sys_fds = [sys.stdout, sys.stderr]
        for fd in self.fd_list:
            if fd not in sys_fds: fd.close()
