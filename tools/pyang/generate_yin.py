#! /usr/bin/env python3
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

import pyang
if pyang.__version__ > '2.4':
    from pyang.repository import FileRepository
    from pyang.context import Context
else:
    from pyang import FileRepository
    from pyang import Context
from io import StringIO 
from xml.sax.saxutils import quoteattr
from xml.sax.saxutils import escape

import re, pathlib, sys, io, os
import argparse, glob
from pyang import util
from pyang import syntax
from pyang import statements
from pyang import error

new_line = '' #replace with '\n' for adding new line
indent_space = '' #replace with ' ' for indentation
ns_indent_space = '' #replace with ' ' for indentation
yin_namespace = "urn:ietf:params:xml:ns:yang:yin:1"
err_prefix = '[Error]'

revision_added = False
mod_dict = dict()

class ContainerToListPlugin():
    def convert_singleton_to_list(self, ctx, modules):
        self.refs = dict()
        for module in modules:
            self.process_children(modules[module], level=0)
        for module in modules:
            self.replace_refs(modules[module])

    def replace_refs(self, stmt):
        stmts = []
        self.collect_statements(stmt, stmts)
        for c_stmt in stmts:
            for imp in c_stmt.i_module.i_prefixes:
                mod_name = c_stmt.i_module.i_prefixes[imp][0]
                if mod_name in self.refs:
                    refs = self.refs[mod_name]
                    for ref in refs:
                        old_ref_key = ref.replace(f"{mod_name}:",f"{imp}:")
                        new_ref_key = refs[ref].replace(f"{mod_name}:",f"{imp}:")
                        if old_ref_key in c_stmt.arg:
                            c_stmt.arg = self.replace_path_substring(c_stmt.arg, old_ref_key, new_ref_key)

    def process_children(self, stmt, level):
        if level == 3 and stmt.keyword == "container":
            self.convert_container_to_list(stmt)
        else:
            for child in stmt.substmts:
                self.process_children(child, level + 1)

    def convert_container_to_list(self, stmt):
        key_name = stmt.arg
        parent_stmt = stmt.parent
        list_name = parent_stmt.arg + "_LIST"
        old_xpath = statements.mk_path_str(stmt, with_prefixes=True, prefix_to_module=True)
        print(f"====> Transforming container {old_xpath} to list")
        old_container_path = statements.mk_path_str(stmt, with_prefixes=False)

        stmt.keyword = "list"
        stmt.raw_keyword = "list"
        stmt.arg = list_name

        key_id = statements.Statement(stmt.top, stmt, stmt.pos, "leaf", "key_id")
        key_id.substmts.append(
            statements.Statement(
                stmt.top,
                key_id,
                stmt.pos,
                "type",
                "enumeration",
            )
        )
        key_id.substmts[-1].substmts.append(
            statements.Statement(
                stmt.top,
                key_id.substmts[-1],
                stmt.pos,
                "enum",
                key_name,
            )
        )
        stmt.substmts.insert(0, key_id)
        stmt.substmts.insert(
            1,
            statements.Statement(stmt.top, stmt, stmt.pos, "key", "key_id"),
        )
        new_list_path = statements.mk_path_str(stmt, with_prefixes=False)
        new_list_path = f"{new_list_path}[key_id='{key_name}']"
        new_xpath = statements.mk_path_str(stmt, with_prefixes=True, prefix_to_module=True)
        new_xpath = f"{new_xpath}[key_id='{key_name}']"
        if stmt.i_module.arg not in self.refs:
            self.refs[stmt.i_module.arg] = dict()
        self.refs[stmt.i_module.arg][old_xpath] = new_xpath
        self.update_local_references(stmt, parent_stmt.arg, key_name, old_container_path, new_list_path)

    def collect_statements(self, stmt, stmts):
        if stmt.keyword in ["must", "when", "leafref"]:
            stmts.append(stmt)
        for child in stmt.substmts:
            self.collect_statements(child, stmts)

    def replace_path_substring(self, path, old_substring, new_substring):
        normalized_path = re.sub(r'/+', '/', path)  # Replace multiple '/' with a single '/'
        return normalized_path.replace(old_substring, new_substring)

    def update_local_references(self, stmt, table_name, key_name, old_container_path, new_list_path):
            refs = dict()
            refs[f"../../{table_name}/{key_name}"] = f"../../{table_name}/{table_name}_LIST[key_id='{key_name}']"
            refs[f"../../../{table_name}/{key_name}"] = f"../../../{table_name}/{table_name}_LIST[key_id='{key_name}']"
            refs[f"../../../../{table_name}/{key_name}"] = f"../../../../{table_name}/{table_name}_LIST[key_id='{key_name}']"
            stmts = []
            self.collect_statements(stmt.top, stmts)
            # Replace relative References
            for c_stmt in stmts:
                if "../.." in c_stmt.arg:
                    for ref in refs:
                        if ref in c_stmt.arg:
                            c_stmt.arg = self.replace_path_substring(c_stmt.arg, ref, refs[ref])
                # Replace Absolute references
                c_stmt.arg = self.replace_path_substring(c_stmt.arg, old_container_path, new_list_path)

def process(args):
    if args.no_err_prefix:
        global err_prefix
        err_prefix = ""
    if args.pretty:
        global new_line, indent_space, ns_indent_space
        new_line, indent_space, ns_indent_space = "\n", "  ", " "

    if not args.out_dir.exists():
        args.out_dir.mkdir(parents=True, exist_ok=True)
    yang_search_paths = ":".join([str(p.absolute()) for p in args.path])
    repo = FileRepository(yang_search_paths, use_env=False)
    ctx = Context(repo)
    for entry in ctx.repository.modules:
        mod_name = entry[0]
        if not mod_name.startswith('sonic-'):
            continue
        mod = entry[2][1]
        try:
            fd = io.open(mod, "r", encoding="utf-8")
            text = fd.read()
        except IOError as ex:
            sys.stderr.write("error %s: %s\n" % (mod_name, str(ex)))
            sys.exit(3)
        mod_obj = ctx.add_module(mod_name, text)
        if mod_name not in mod_dict:
            mod_dict[mod_name] = mod_obj
    ctx.validate()
    conv = ContainerToListPlugin()
    conv.convert_singleton_to_list(ctx,mod_dict)
    error_seen = False
    for (epos, etag, eargs) in ctx.errors:
        elevel = error.err_level(etag)

        if error.is_warning(elevel):
            kind = "warning"
        else:
            kind = "error"
            error_seen = True
            sys.stdout.write(str(epos) + ': %s: ' % kind + \
                error.err_to_str(etag, eargs) + '\n')
    yin_sets = set()
    yin_sets_old = set()
    for yin in glob.glob(str(args.out_dir.absolute())+'/*.yin'):
        yin_sets_old.add(os.path.basename(yin))
    schema_mods = find_schema_modules(mod_dict)
    for mod in mod_dict:
        if mod not in schema_mods:
            print(f"{mod}.yin has no schema elements") if args.verbose else None
            continue
        fd = StringIO()
        emit_yin(ctx, mod_dict[mod], fd)
        yin_content = fd.getvalue()
        yin_name = mod + ".yin"
        yin_sets.add(yin_name)
        yin_file = args.out_dir.joinpath(yin_name)
        if yin_file.exists():
            if yin_file.read_text() == yin_content:
                print(f"{mod}.yin not changed") if args.verbose else None
                continue
        print("Writing " + str(yin_file))
        yin_file.write_text(yin_content)
    for entry in yin_sets_old - yin_sets:
        stale_yin = args.out_dir.joinpath(entry)
        if stale_yin.exists():
            print("Removing {}".format(stale_yin))
            stale_yin.unlink()
    if error_seen:
        sys.exit(2)

def find_schema_modules(mod_dict) -> set:
    schema_mods = set()
    for modname, mod in mod_dict.items():
        if not has_schema(mod):
            continue
        imports = [ss.arg for ss in mod.substmts if ss.raw_keyword == "import"]
        schema_mods.add(modname)
        schema_mods.update(imports)
    return schema_mods

def has_schema(stmt) -> bool:
    if stmt.raw_keyword in ["leaf", "leaf-list", "list"]:
        return not is_config_false(stmt)
    if stmt.raw_keyword in ["module", "container", "choice"]:
        return not is_config_false(stmt) and [True for ss in stmt.substmts if has_schema(ss)]
    return False

def is_config_false(stmt) -> bool:
    return bool([ss for ss in stmt.substmts if ss.raw_keyword == "config" and ss.arg == "false"])

def emit_yin(ctx, module, fd):
    fd.write('<?xml version="1.0" encoding="UTF-8"?>' + new_line)
    fd.write(('<%s name="%s"' + new_line) % (module.keyword, module.arg))
    fd.write(ns_indent_space * len(module.keyword) + ns_indent_space + ' xmlns="%s"' % yin_namespace)

    prefix = module.search_one('prefix')
    if prefix is not None:
        namespace = module.search_one('namespace')
        fd.write('' + new_line)
        fd.write(ns_indent_space * len(module.keyword))
        fd.write(ns_indent_space + ' xmlns:' + prefix.arg + '=' +
                 quoteattr(namespace.arg))
    else:
        belongs_to = module.search_one('belongs-to')
        if belongs_to is not None:
            prefix = belongs_to.search_one('prefix')
            if prefix is not None:
                # read the parent module in order to find the namespace uri
                res = ctx.read_module(belongs_to.arg, extra={'no_include':True})
                if res is not None:
                    namespace = res.search_one('namespace')
                    if namespace is None or namespace.arg is None:
                        pass
                    else:
                        # success - namespace found
                        fd.write('' + new_line)
                        fd.write(sonic-acl.yin * len(module.keyword))
                        fd.write(sonic-acl.yin + ' xmlns:' + prefix.arg + '=' +
                                 quoteattr(namespace.arg))
            
    for imp in module.search('import'):
        prefix = imp.search_one('prefix')
        if prefix is not None:
            rev = None
            r = imp.search_one('revision-date')
            if r is not None:
                rev = r.arg
            mod = statements.modulename_to_module(module, imp.arg, rev)
            if mod is not None:
                ns = mod.search_one('namespace')
                if ns is not None:
                    fd.write('' + new_line)
                    fd.write(ns_indent_space * len(module.keyword))
                    fd.write(ns_indent_space + ' xmlns:' + prefix.arg + '=' +
                             quoteattr(ns.arg))
    fd.write('>' + new_line)

    global revision_added
    revision_added = False

    substmts = module.substmts
    for s in substmts:
        emit_stmt(ctx, module, s, fd, indent_space, indent_space)
    fd.write(('</%s>' + new_line) % module.keyword)
    
def emit_stmt(ctx, module, stmt, fd, indent, indentstep):
    global revision_added

    if stmt.raw_keyword == "revision" and revision_added == False:
        revision_added = True
    elif stmt.raw_keyword == "revision" and revision_added == True:
        #Only add the latest revision
        return

    #Don't keep the following keywords as they are not used in CVL
    # stmt.raw_keyword == "revision" or
    if ((stmt.raw_keyword == "organization" or 
            stmt.raw_keyword == "contact" or 
            stmt.raw_keyword == "rpc" or
            stmt.raw_keyword == "notification" or
            stmt.raw_keyword == "description")):
        return

    #Check for "config false" statement and skip the node containing the same
    for s in stmt.substmts:
        if (s.raw_keyword  == "config" and s.arg == "false"):
            return

    if util.is_prefixed(stmt.raw_keyword):
        # this is an extension.  need to find its definition
        (prefix, identifier) = stmt.raw_keyword
        tag = prefix + ':' + identifier
        if stmt.i_extension is not None:
            ext_arg = stmt.i_extension.search_one('argument')
            if ext_arg is not None:
                yin_element = ext_arg.search_one('yin-element')
                if yin_element is not None and yin_element.arg == 'true':
                    argname = prefix + ':' + ext_arg.arg
                    argiselem = True
                else:
                    # explicit false or no yin-element given
                    argname = ext_arg.arg
                    argiselem = False
            else:
                argiselem = False
                argname = None
        else:
            argiselem = False
            argname = None
    else:
        (argname, argiselem) = syntax.yin_map[stmt.raw_keyword]
        tag = stmt.raw_keyword
    if argiselem == False or argname is None:
        if argname is None:
            attr = ''
        else:
            attr = ' ' + argname + '=' + quoteattr(stmt.arg)
        if len(stmt.substmts) == 0:
            fd.write(indent + '<' + tag + attr + '/>' + new_line)
        else:
            fd.write(indent + '<' + tag + attr + '>' + new_line)
            for s in stmt.substmts:
                emit_stmt(ctx, module, s, fd, indent + indentstep,
                          indentstep)
            fd.write(indent + '</' + tag + '>' + new_line)
    else:
        value = escape(stmt.arg)
        if tag == "error-message" and err_prefix:
            value = err_prefix + value
        fd.write(indent + '<' + tag + '>' + new_line)
        fd.write(indent + indentstep + '<' + argname + '>' + \
                   value + \
                   '</' + argname + '>' + new_line)
        substmts = stmt.substmts

        for s in substmts:
            emit_stmt(ctx, module, s, fd, indent + indentstep, indentstep)

        fd.write(indent + '</' + tag + '>' + new_line)


if __name__== "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--path', action='append', help='Sonic yang lookup directory', required=True, type=pathlib.Path)
    parser.add_argument('--out-dir', dest='out_dir', help='Output directory for YIN files', required=True, type=pathlib.Path)
    parser.add_argument('--pretty', help='Pretty print output', action='store_true')
    parser.add_argument('--no-err-prefix', help=f'Do not prefix "{err_prefix}" to error-message strings', action='store_true')
    parser.add_argument('--verbose', '-v', help='Print verbose logs', action='store_true')
    args = parser.parse_args()
    process(args)
