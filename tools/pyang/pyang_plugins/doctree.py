################################################################################
#                                                                              #
#  Copyright 2019 Broadcom. The term Broadcom refers to Broadcom Inc. and/or   #
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
import io
import optparse
import re
import sys

import pyang
from pyang import plugin, statements, syntax

if pyang.__version__ > '2.4':
    from pyang.repository import FileRepository
    from pyang.context import Context
else:
    from pyang import FileRepository
    from pyang import Context

# globals
mods_dict = dict()
extended_mods_dict = dict()

def pyang_plugin_init():
    plugin.register_plugin(docApiPlugin())

def search_child(children, identifier):
    for child in children:
        if child.arg == identifier:
            return True
    return False

def build_mods_dict(ctx):
    """Build dictionaries of non-patched and patched modules"""
    global mods_dict
    global extended_mods_dict

    def load_modules(path, target_dict):
        """Helper function to load modules"""
        if path is None:
            return

        repo = FileRepository(path, use_env=False, no_path_recurse=True)
        new_ctx = Context(repo)
        new_ctx.opts = ctx.opts
        new_ctx.lax_xpath_checks = ctx.lax_xpath_checks
        new_ctx.lax_quote_checks = ctx.lax_quote_checks

        for entry in new_ctx.repository.modules:
            mod_name = entry[0]
            mod = entry[2][1]
            try:
                with io.open(mod, "r", encoding="utf-8") as fd:
                    text = fd.read()
            except IOError as ex:
                sys.stderr.write("error %s: %s\n" % (mod_name, str(ex)))
                sys.exit(1)

            mod_obj = new_ctx.add_module(mod_name, text)
            if mod_name not in target_dict:
                target_dict[mod_name] = mod_obj

        new_ctx.validate()

    if not ctx.opts.basepaths or not ctx.opts.path[0]:
        sys.stderr.write("Invalid paths provided. basepaths: %s, path: %s\n" %(ctx.opts.basepaths, ctx.opts.path[0]))
        sys.exit(1)

    # Load base modules (non-patched)
    load_modules(ctx.opts.basepaths, mods_dict)

    # Load patched modules (with extensions)
    load_modules(ctx.opts.path[0], extended_mods_dict)

def find_node_in_module(target_module, path_parts):
    """Find a node in a module using its path parts

    Args:
        target_module: The module to search in
        path_parts: List of (module_name, identifier) tuples

    Returns:
        The node if found, None otherwise
    """
    if not path_parts:
        return None

    # Start with the first path part
    module_name, identifier = path_parts[0]
    node = statements.search_child(target_module.i_children, module_name, identifier)

    if node is None:
        return None

    # Process the rest of the path
    for module_name, identifier in path_parts[1:]:
        if not hasattr(node, 'i_children'):
            return None

        child = statements.search_child(node.i_children, module_name, identifier)
        if child is None:
            return None

        node = child

    return node

def get_node_path_parts(node):
    """Get the path parts of a node as a list of (module_name, identifier) tuples"""
    path_parts = []

    def collect_path(node, parts):
        if node.parent.keyword in ['module', 'submodule']:
            parts.append((node.i_module.i_modulename, node.arg))
            return

        collect_path(node.parent, parts)
        parts.append((node.i_module.i_modulename, node.arg))

    collect_path(node, path_parts)
    return path_parts

def check_node_existence(node, non_patched_mod):
    """Check if a node exists in a non-patched module

    Args:
        node: The node to check
        non_patched_mod: The non-patched module to check against

    Returns:
        True if the node exists, False otherwise
    """
    path_parts = get_node_path_parts(node)
    found_node = find_node_in_module(non_patched_mod, path_parts)
    return found_node is not None

def mark_node_with_diff(node, exists):
    """Mark a node with its diff status

    Args:
        node: The node to mark
        exists: Whether the node exists in the non-patched modules
    """
    if exists:
        # The node exists in both - no special marking
        node.diff_status = None
    else:
        # The node is new (added) in the patched modules
        node.diff_status = "added"

def mark_children_recursively(node, status):
    """Mark all children of a node with the same status

    Args:
        node: The node whose children to mark
        status: The status to apply ("added" or "removed")
    """
    if hasattr(node, 'i_children'):
        for child in node.i_children:
            child.diff_status = status
            mark_children_recursively(child, status)

def process_modules_diff(ctx):
    """Process the differences between patched and non-patched modules"""
    # For each patched module
    for mod_name, patched_mod in extended_mods_dict.items():
        # Check if the module exists in non-patched modules
        if mod_name not in mods_dict:
            # If not, mark the entire module as added
            for child in patched_mod.i_children:
                if child.keyword in statements.data_definition_keywords + ['rpc', 'notification']:
                    child.diff_status = "added"
                    mark_children_recursively(child, "added")
        else:
            # If it exists, check each child node
            non_patched_mod = mods_dict[mod_name]
            process_node_diff(patched_mod, non_patched_mod)

    # Process nodes that exist in non-patched but not in patched
    for mod_name, non_patched_mod in mods_dict.items():
        if mod_name in extended_mods_dict:
            # We've already checked the common modules above
            continue

        # This module only exists in non-patched, mark it all as removed
        non_patched_mod.diff_status = "removed"
        for child in non_patched_mod.i_children:
            if child.keyword in statements.data_definition_keywords + ['rpc', 'notification']:
                child.diff_status = "removed"
                mark_children_recursively(child, "removed")

def process_node_diff(patched_node, non_patched_node):
    """Process differences between corresponding nodes

    Args:
        patched_node: The node from patched modules
        non_patched_node: The corresponding node from non-patched modules
    """
    if hasattr(patched_node, 'i_children'):
        patched_children = {
            (child.i_module.i_modulename, child.arg): child
            for child in patched_node.i_children
            if child.keyword in statements.data_definition_keywords + ['rpc', 'notification', 'input', 'output']
        }

        non_patched_children = {
            (child.i_module.i_modulename, child.arg): child
            for child in non_patched_node.i_children
            if child.keyword in statements.data_definition_keywords + ['rpc', 'notification', 'input', 'output']
        }

        # Check for added nodes
        for key, child in patched_children.items():
            if key not in non_patched_children:
                child.diff_status = "added"
                mark_children_recursively(child, "added")
            else:
                # Node exists in both, check its children
                process_node_diff(child, non_patched_children[key])

        # Check for removed nodes - add them to patched with 'removed' status
        for key, non_patched_child in non_patched_children.items():
            if key not in patched_children:
                # Create a copy of the node to represent the removed node
                removed_child = non_patched_child
                removed_child.diff_status = "removed"
                patched_node.i_children.append(removed_child)
                mark_children_recursively(removed_child, "removed")

class docApiPlugin(plugin.PyangPlugin):
    def add_output_format(self, fmts):
        self.multiple_modules = True
        fmts['doctree'] = self

    def add_opts(self, optparser):
        optlist = [
            optparse.make_option("--basepaths",
                                 type="string",
                                 dest="basepaths",
                                 help="Base YANG Paths"),
        ]
        g = optparser.add_option_group("docApiPlugin options")
        g.add_options(optlist)

    def setup_fmt(self, ctx):
        ctx.implicit_errors = False

    def emit(self, ctx, modules, fd):
        # Build module dictionaries
        build_mods_dict(ctx)

        # Process diffs between patched and non-patched modules
        process_modules_diff(ctx)

        # Generate markdown output
        fd.write("# List of Yang Modules\n")
        for mod in sorted(extended_mods_dict):
            fd.write("* [%s](#%s)\n" % (mod, mod))
        # Output the tree representation with diffs
        emit_tree(ctx, fd, None, None, None)

def get_sub_mods(ctx, module, submods=[]):
    """Get submodules of a module"""
    local_mods = []
    for i in module.search('include'):
        subm = ctx.get_module(i.arg)
        if subm is not None:
            local_mods.append(subm)
            submods.append(subm.arg)
    for local_mod in local_mods:
        get_sub_mods(ctx, local_mod, submods)

def emit_tree(ctx, fd, depth, llen, path):
    """Output the tree representation with diff marks"""
    for mod in extended_mods_dict:
        module = extended_mods_dict[mod]
        fd.write("## %s\n" % (module.arg))
        fd.write('```diff\n')

        chs = [ch for ch in module.i_children
               if ch.keyword in statements.data_definition_keywords]
        if path is not None and len(path) > 0:
            chs = [ch for ch in chs if ch.arg == path[0]]
            chpath = path[1:]
        else:
            chpath = path

        if len(chs) > 0:
            print_children(chs, module, fd, '', chpath, 'data', depth, llen,
                           None, 0, False, False,
                           prefix_with_modname=False)

        rpcs = [ch for ch in module.i_children
                if ch.keyword == 'rpc']
        rpath = path
        if path is not None:
            if len(path) > 0:
                rpcs = [rpc for rpc in rpcs if rpc.arg == path[0]]
                rpath = path[1:]
            else:
                rpcs = []
        if len(rpcs) > 0:
            fd.write("\n  rpcs:\n")
            print_children(rpcs, module, fd, '  ', rpath, 'rpc', depth, llen,
                           None, 0, False, False,
                           prefix_with_modname=False)
        fd.write('\n```\n')

        # Output module information
        if len(module.i_prefixes) > 0:
            fd.write("| Prefix |     Module Name    |\n")
            fd.write("|:---:|:-----------:|\n")
            for i_prefix in module.i_prefixes:
                fd.write("| %s | %s  |\n" % (i_prefix, module.i_prefixes[i_prefix][0]))

        submods = []
        get_sub_mods(ctx, module, submods)
        if len(submods) > 0:
            fd.write("\n| Submodules |\n")
            fd.write("|:---:|\n")
            for submod in submods:
                fd.write("| %s |\n" % (submod))

        fd.write('\n')

def print_children(i_children, module, fd, prefix, path, mode, depth,
                   llen, no_expand_uses, width=0, isExtended=False, not_supported=False,
                   prefix_with_modname=False,
                   ):
    if depth == 0:
        if i_children: fd.write(prefix + '     ...\n')
        return

    def get_width(w, chs):
        for ch in chs:
            if ch.keyword in ['choice', 'case']:
                nlen = 3 + get_width(0, ch.i_children)
            else:
                if ch.i_module.i_modulename == module.i_modulename:
                    nlen = len(ch.arg)
                else:
                    nlen = len(ch.i_module.i_prefix) + 1 + len(ch.arg)
            if nlen > w:
                w = nlen
        return w

    if width == 0:
        width = get_width(0, i_children)

    for ch in i_children:
        if ((ch.keyword == 'input' or ch.keyword == 'output') and
            len(ch.i_children) == 0):
            pass
        else:
            if (ch == i_children[-1] or
                (i_children[-1].keyword == 'output' and
                 len(i_children[-1].i_children) == 0)):
                # the last test is to detect if we print input, and the
                # next node is an empty output node; then don't add the |
                newprefix = prefix + '   '
            else:
                newprefix = prefix + '  |'
            if ch.keyword == 'input':
                mode = 'input'
            elif ch.keyword == 'output':
                mode = 'output'

            # Check diff status from our new marking system
            is_added = hasattr(ch, 'diff_status') and ch.diff_status == "added"
            is_removed = hasattr(ch, 'diff_status') and ch.diff_status == "removed"

            print_node(ch, module, fd, newprefix, path, mode, depth, llen,
                       no_expand_uses, width, is_added, is_removed,
                       prefix_with_modname=prefix_with_modname)

def print_node(s, module, fd, prefix, path, mode, depth, llen,
               no_expand_uses, width, is_added=False, is_removed=False,
               prefix_with_modname=False
               ):
    line = "%s%s--" % (prefix[0:-1], get_status_str(s))

    # Apply diff status prefixes
    if is_added:
        line = '+' + line[1:]

    if is_removed:
        line = '-' + line[1:]

    brcol = len(line) + 4

    if s.i_module.i_modulename == module.i_modulename:
        name = s.arg
    else:
        if prefix_with_modname:
            name = s.i_module.i_modulename + ':' + s.arg
        else:
            name = s.i_module.i_prefix + ':' + s.arg
    flags = get_flags_str(s, mode)
    if s.keyword == 'list':
        name += '*'
        line += flags + " " + name
    elif s.keyword == 'container':
        p = s.search_one('presence')
        if p is not None:
            name += '!'
        line += flags + " " + name
    elif s.keyword  == 'choice':
        m = s.search_one('mandatory')
        if m is None or m.arg == 'false':
            line += flags + ' (' + name + ')?'
        else:
            line += flags + ' (' + name + ')'
    elif s.keyword == 'case':
        line += ':(' + name + ')'
        brcol += 1
    else:
        if s.keyword == 'leaf-list':
            name += '*'
        elif (s.keyword == 'leaf' and not hasattr(s, 'i_is_key')
              or s.keyword == 'anydata' or s.keyword == 'anyxml'):
            m = s.search_one('mandatory')
            if m is None or m.arg == 'false':
                name += '?'
        t = get_typename(s, prefix_with_modname)
        if t == '':
            line += "%s %s" % (flags, name)
        elif (llen is not None and
              len(line) + len(flags) + width+1 + len(t) + 4 > llen):
            # there's no room for the type name
            if (get_leafref_path(s) is not None and
                len(t) + brcol > llen):
                # there's not even room for the leafref path; skip it
                line += "%s %-*s   leafref" % (flags, width+1, name)
            else:
                line += "%s %s" % (flags, name)
                fd.write(line + '\n')
                line = prefix + ' ' * (brcol - len(prefix)) + ' ' + t
        else:
            line += "%s %-*s   %s" % (flags, width+1, name, t)

    if s.keyword == 'list':
        if s.search_one('key') is not None:
            keystr = " [%s]" % re.sub('\s+', ' ', s.search_one('key').arg)
            if (llen is not None and
                len(line) + len(keystr) > llen):
                fd.write(line + '\n')
                line = prefix + ' ' * (brcol - len(prefix))
            line += keystr
        else:
            line += " []"

    fd.write(line + '\n')
    if hasattr(s, 'i_children') and s.keyword != 'uses':
        if depth is not None:
            depth = depth - 1
        chs = s.i_children
        if path is not None and len(path) > 0:
            chs = [ch for ch in chs
                   if ch.arg == path[0]]
            path = path[1:]
        if s.keyword in ['choice', 'case']:
            print_children(chs, module, fd, prefix, path, mode, depth,
                           llen, no_expand_uses, width - 3, is_added, is_removed,
                           prefix_with_modname=prefix_with_modname)
        else:
            print_children(chs, module, fd, prefix, path, mode, depth, llen,
                           no_expand_uses, 0, is_added, is_removed,
                           prefix_with_modname=prefix_with_modname)

# Helper functions - kept mostly unchanged
def get_status_str(s):
    status = s.search_one('status')
    if status is None or status.arg == 'current':
        return '+'
    elif status.arg == 'deprecated':
        return 'x'
    elif status.arg == 'obsolete':
        return 'o'

def get_flags_str(s, mode):
    if mode == 'input':
        return "-w"
    elif s.keyword in ('rpc', 'action', ('tailf-common', 'action')):
        return '-x'
    elif s.keyword == 'notification':
        return '-n'
    elif s.keyword == 'uses':
        return '-u'
    elif s.i_config == True:
        return 'rw'
    elif s.i_config == False or mode == 'output' or mode == 'notification':
        return 'ro'
    else:
        return ''

def get_leafref_path(s):
    t = s.search_one('type')
    if t is not None:
        if t.arg == 'leafref':
            return t.search_one('path')
    else:
        return None

def get_typename(s, prefix_with_modname=False):
    t = s.search_one('type')
    if t is not None:
        if t.arg == 'leafref':
            p = t.search_one('path')
            if p is not None:
                # Try to make the path as compact as possible.
                # Remove local prefixes, and only use prefix when
                # there is a module change in the path.
                target = []
                curprefix = s.i_module.i_prefix
                for name in p.arg.split('/'):
                    if name.find(":") == -1:
                        prefix = curprefix
                    else:
                        [prefix, name] = name.split(':', 1)
                    if prefix == curprefix:
                        target.append(name)
                    else:
                        if prefix_with_modname:
                            if prefix in s.i_module.i_prefixes:
                                # Try to map the prefix to the module name
                                (module_name, _) = s.i_module.i_prefixes[prefix]
                            else:
                                # If we can't then fall back to the prefix
                                module_name = prefix
                            target.append(module_name + ':' + name)
                        else:
                            target.append(prefix + ':' + name)
                        curprefix = prefix
                return "-> %s" % "/".join(target)
            else:
                # This should never be reached. Path MUST be present for
                # leafref type. See RFC6020 section 9.9.2
                # (https://tools.ietf.org/html/rfc6020#section-9.9.2)
                if prefix_with_modname:
                    if t.arg.find(":") == -1:
                        # No prefix specified. Leave as is
                        return t.arg
                    else:
                        # Prefix found. Replace it with the module name
                        [prefix, name] = t.arg.split(':', 1)
                        #return s.i_module.i_modulename + ':' + name
                        if prefix in s.i_module.i_prefixes:
                            # Try to map the prefix to the module name
                            (module_name, _) = s.i_module.i_prefixes[prefix]
                        else:
                            # If we can't then fall back to the prefix
                            module_name = prefix
                        return module_name + ':' + name
                else:
                    return t.arg
        else:
            if prefix_with_modname:
                if t.arg.find(":") == -1:
                    # No prefix specified. Leave as is
                    return t.arg
                else:
                    # Prefix found. Replace it with the module name
                    [prefix, name] = t.arg.split(':', 1)
                    #return s.i_module.i_modulename + ':' + name
                    if prefix in s.i_module.i_prefixes:
                        # Try to map the prefix to the module name
                        (module_name, _) = s.i_module.i_prefixes[prefix]
                    else:
                        # If we can't then fall back to the prefix
                        module_name = prefix
                    return module_name + ':' + name
            else:
                return t.arg
    elif s.keyword == 'anydata':
        return '<anydata>'
    elif s.keyword == 'anyxml':
        return '<anyxml>'
    else:
        return ''