################################################################################
#                                                                              #
#  Copyright 2022 Dell, Inc.                                                   #
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
import re

from pyang import plugin
from pyang import statements
from pyang.plugins import tree

def pyang_plugin_init():
    plugin.register_plugin(annotTreePlugin())

# inherit from TreePlugin class
class annotTreePlugin(tree.TreePlugin):
    def __init__(self):
        tree.TreePlugin.__init__(self)

    def emit(self, ctx, modules, fd):
        if ctx.opts.tree_path is not None:
            path = ctx.opts.tree_path.split('/')
            if path[0] == '':
                path = path[1:]
        else:
            path = None
        emit_annot_tree(ctx, modules, fd, ctx.opts.tree_depth,
                  ctx.opts.tree_line_length, path)
    
    def add_opts(self, optparser):
        pass                  


def emit_annot_tree(ctx, modules, fd, depth, llen, path):
    for module in modules:
        printed_header = False

        def print_header():
            bstr = ""
            b = module.search_one('belongs-to')
            if b is not None:
                bstr = " (belongs-to %s)" % b.arg
            fd.write("%s: %s%s\n" % (module.keyword, module.arg, bstr))
            printed_header = True
            
        annot_statement = [d for d in ctx.deviation_modules if d.i_modulename == module.i_modulename + '-annot']

        chs = [ch for ch in module.i_children
               if ch.keyword in statements.data_definition_keywords]
        if path is not None and len(path) > 0:
            chs = [ch for ch in chs if ch.arg == path[0]]
            chpath = path[1:]
        else:
            chpath = path

        if len(chs) > 0:
            if not printed_header:
                print_header()
                printed_header = True

            print_children(chs, module, fd, '', chpath, 'data', depth, llen,
                           annot_statement, ctx.opts.tree_no_expand_uses,
                           prefix_with_modname=ctx.opts.modname_prefix)

        mods = [module]
        for i in module.search('include'):
            subm = ctx.get_module(i.arg)
            if subm is not None:
                mods.append(subm)
        section_delimiter_printed=False
        for m in mods:
            for augment in m.search('augment'):
                if (hasattr(augment.i_target_node, 'i_module') and
                    augment.i_target_node.i_module not in modules + mods):
                    if not section_delimiter_printed:
                        fd.write('\n')
                        section_delimiter_printed = True
                    # this augment has not been printed; print it
                    if not printed_header:
                        print_header()
                        printed_header = True
                    tree.print_path("  augment", ":", augment.arg, fd, llen)
                    mode = 'augment'
                    if augment.i_target_node.keyword == 'input':
                        mode = 'input'
                    elif augment.i_target_node.keyword == 'output':
                        mode = 'output'
                    elif augment.i_target_node.keyword == 'notification':
                        mode = 'notification'
                    m_annot_statement = [d for d in ctx.deviation_modules if d.i_modulename == m.i_modulename + '-annot']
                    print_children(augment.i_children, m, fd,
                                   '  ', path, mode, depth, llen,
                                   m_annot_statement, ctx.opts.tree_no_expand_uses,
                                   prefix_with_modname=ctx.opts.modname_prefix)

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
            if not printed_header:
                print_header()
                printed_header = True
            fd.write("\n  rpcs:\n")
            print_children(rpcs, module, fd, '  ', rpath, 'rpc', depth, llen,
                           annot_statement, ctx.opts.tree_no_expand_uses,
                           prefix_with_modname=ctx.opts.modname_prefix)

        notifs = [ch for ch in module.i_children
                  if ch.keyword == 'notification']
        npath = path
        if path is not None:
            if len(path) > 0:
                notifs = [n for n in notifs if n.arg == path[0]]
                npath = path[1:]
            else:
                notifs = []
        if len(notifs) > 0:
            if not printed_header:
                print_header()
                printed_header = True
            fd.write("\n  notifications:\n")
            print_children(notifs, module, fd, '  ', npath,
                           'notification', depth, llen,
                           annot_statement, ctx.opts.tree_no_expand_uses,
                           prefix_with_modname=ctx.opts.modname_prefix)

        if ctx.opts.tree_print_groupings:
            section_delimiter_printed = False
            for m in mods:
                for g in m.search('grouping'):
                    if not printed_header:
                        print_header()
                        printed_header = True
                    if not section_delimiter_printed:
                        fd.write('\n')
                        section_delimiter_printed = True
                    fd.write("  grouping %s\n" % g.arg)
                    
                    m_annot_statement = [d for d in ctx.deviation_modules if d.i_modulename == m.i_modulename + '-annot']

                    print_children(g.i_children, m, fd,
                                   '  ', path, 'grouping', depth, llen,
                                   m_annot_statement, ctx.opts.tree_no_expand_uses,
                                   prefix_with_modname=ctx.opts.modname_prefix)

        if ctx.opts.tree_print_yang_data:
            yds = module.search(('ietf-restconf', 'yang-data'))
            if len(yds) > 0:
                if not printed_header:
                    print_header()
                    printed_header = True
                section_delimiter_printed = False
                for yd in yds:
                    if not section_delimiter_printed:
                        fd.write('\n')
                        section_delimiter_printed = True
                    fd.write("  yang-data %s:\n" % yd.arg)
                    print_children(yd.i_children, module, fd, '  ', path,
                                   'yang-data', depth, llen,
                                   annot_statement, ctx.opts.tree_no_expand_uses,
                                   prefix_with_modname=ctx.opts.modname_prefix)


def print_children(i_children, module, fd, prefix, path, mode, depth,
                   llen, annot_statement, no_expand_uses, width=0, prefix_with_modname=False):
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

    if no_expand_uses:
        i_children = tree.unexpand_uses(i_children)

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
            print_node(ch, module, fd, newprefix, path, mode, depth, llen,
                       annot_statement, no_expand_uses, width,
                       prefix_with_modname=prefix_with_modname)

def print_node(s, module, fd, prefix, path, mode, depth, llen, annot_statement,
               no_expand_uses, width, prefix_with_modname=False):

    line = "%s%s--" % (prefix[0:-1], tree.get_status_str(s))

    brcol = len(line) + 4

    if s.i_module.i_modulename == module.i_modulename:
        name = s.arg
    else:
        if prefix_with_modname:
            name = s.i_module.i_modulename + ':' + s.arg
        else:
            name = s.i_module.i_prefix + ':' + s.arg
    flags = tree.get_flags_str(s, mode)
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
        t = tree.get_typename(s, prefix_with_modname)
        if t == '':
            line += "%s %s" % (flags, name)
        elif (llen is not None and
              len(line) + len(flags) + width+1 + len(t) + 4 > llen):
            # there's no room for the type name
            if (tree.get_leafref_path(s) is not None and
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
            keystr = " [%s]" % re.sub(r'\s+', ' ', s.search_one('key').arg)
            if (llen is not None and
                len(line) + len(keystr) > llen):
                fd.write(line + '\n')
                line = prefix + ' ' * (brcol - len(prefix))
            line += keystr
        else:
            line += " []"

    features = s.search('if-feature')
    featurenames = [f.arg for f in features]
    if hasattr(s, 'i_augment'):
        afeatures = s.i_augment.search('if-feature')
        featurenames.extend([f.arg for f in afeatures
                             if f.arg not in featurenames])

    if len(featurenames) > 0:
        fstr = " {%s}?" % ",".join(featurenames)
        if (llen is not None and len(line) + len(fstr) > llen):
            fd.write(line + '\n')
            line = prefix + ' ' * (brcol - len(prefix))
        line += fstr

    def find_target_deviate(d, s):
        e = [e for e in d if e.i_target_node.arg == s.arg]
        for dev in e:
            dp = dev.i_target_node.parent
            sp = s.parent
            while dp is not None and sp is not None:
                if dp.arg == sp.arg:
                    dp = dp.parent
                    sp = sp.parent
                    continue
                else:
                    break
            if dp is not None or sp is not None:
                continue 
            if s.keyword == 'leaf' or s.keyword == 'leaf-list':
                return dev
            elif dev.i_target_node.i_children == s.i_children:
                return dev
        return None

    if len(annot_statement) > 0:
        d = [annot for annot in annot_statement[0].substmts if annot.keyword == 'deviation']
        a = find_target_deviate(d,s)
        if a is not None:
            for dev in a.substmts:
                if dev.arg == 'add':
                    line += ' #annot#'
                    for e in dev.substmts:
                        line += ' ' + e.i_extension.arg + ':' + e.arg
                break
    
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
                           llen, annot_statement, no_expand_uses, width - 3,
                           prefix_with_modname=prefix_with_modname)
        else:
            print_children(chs, module, fd, prefix, path, mode, depth, llen,
                           annot_statement, no_expand_uses,
                           prefix_with_modname=prefix_with_modname)

