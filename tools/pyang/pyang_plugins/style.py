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

import optparse
import sys
import os
import io
import subprocess
import re

import pyang
from pyang import plugin
from pyang import statements
from pyang import error
from pyang import util
from pyang import types
from pyang.error import err_add

try:
    from StringIO import StringIO ## for Python 2
except ImportError:
    from io import StringIO ## for Python 3

issues = []
warnings = []
extensionModulesList = []

def pyang_plugin_init():
    plugin.register_plugin(CheckOcStylePlugin())

class CheckOcStylePlugin(plugin.PyangPlugin):
    
    def add_output_format(self, fmts):
        self.multiple_modules = True
        fmts['stcheck'] = self

    def add_opts(self, optparser):
        optlist = [
            optparse.make_option("--extensiondir",
                                 type="string",
                                 dest="extensiondir",
                                 help="Extension yangs's directory"),                                 
        ]
        g = optparser.add_option_group("CheckOcStylePlugin options")
        g.add_options(optlist)

    def setup_fmt(self, ctx):
        ctx.implicit_errors = False

    def emit(self, ctx, modules, fd):
        
        global extensionModulesList

        if ctx.opts.extensiondir is None:
            print("Extension YANG's directory not mentioned")
            sys.exit(2)

        extensionModulesList = list(map(lambda yn: os.path.splitext(yn)[0], os.listdir(ctx.opts.extensiondir)))

        for module in modules:
            if module.arg.startswith("openconfig-"):
                walk_child(ctx,module)
            if module.arg.startswith("openconfig-") and \
                module.arg in extensionModulesList:
                chk_version_prefix_namespace(ctx, module)
                chk_naming_for_grouping(ctx,module)
                chk_naming_for_enum_identity(ctx,module)  
                chk_choice_case_feature(ctx,module)            
                
        if ctx.opts.outfile is not None:
            fd = open(ctx.opts.outfile, "w")

        error_seen = False
        for issue in issues:
            if "/extensions/" not in str(issue):
                continue
            error_seen = True
            issue = "Error: " + issue
            fd.write(issue +"\n")
        
        for warning in warnings:
            if "/extensions/" not in str(warning):
                continue
            warning = "Warning: " + warning
            fd.write(warning +"\n")

        if ctx.opts.outfile is not None:
            fd.close()
        if error_seen:
            sys.exit(1)
        else:
            sys.exit(0)

def chk_choice_case_feature(ctx, module):
    choiceList = []
    find_node(module,"choice", choiceList)
    caseList = []
    find_node(module,"case", caseList)
    featureList = []
    find_node(module,"feature", featureList)   
    if_featureList = []
    find_node(module,"if-feature", if_featureList)    
    for choice in choiceList:
        issues.append(str(choice.pos) + " Avoid using choice statements in OC-YANG")
    for case in caseList:
        issues.append(str(case.pos) + " Avoid using case statements in OC-YANG")        
    for feature in featureList:
        issues.append(str(feature.pos) + " Avoid using feature statements in OC-YANG")        
    for if_feature in if_featureList:
        issues.append(str(if_feature.pos) + " Avoid using if-feature statements in OC-YANG")        

def chk_naming_for_enum_identity(ctx, module):
    reg = '^([A-Z0-9])+([A-Z0-9\\.\\_])*$'
    enumsList = []
    find_node(module,"enum", enumsList)
    for enum in enumsList:
        if re.match(reg, enum.arg) is None:
            issues.append(str(enum.pos) + " %s Enum not in OC-style"%(enum.arg))
    identityList = []
    find_node(module,"identity", identityList)  
    for identity in identityList:
        if re.match(reg, identity.arg) is None:
            issues.append(str(identity.pos) + " %s Identity not in OC-style"%(identity.arg))    

def chk_version_prefix_namespace(ctx, module):
    global issues
    version_present = False

    for stmt in module.substmts:
        if type(stmt.keyword) == tuple:
            if stmt.keyword[0] == "openconfig-extensions":
                if stmt.keyword[1] == "openconfig-version":
                    version_present = True
                    break
    
    if not version_present:
        issues.append(module.arg + ".yang does not have openconfig-version")
    
    if not hasattr (module, 'i_prefix'):
        issues.append(module.arg + ".yang does not have prefix")
        sys.exit(2)

    namespace = module.search_one('namespace', None, module.substmts)
    
    if module.keyword != "submodule":
        if namespace is None:
            issues.append(module.arg + ".yang does not have namespace")
        else:
            if not namespace.arg.endswith('/extension'):
                issues.append(str(namespace.pos) + " should end with /extension")

    if not module.i_prefix.startswith('oc-'):
        issues.append(module.arg + ".yang does not have prefix which begins with oc-")

def walk_child(ctx,child):

    if child.keyword in statements.data_keywords:   
        chk_naming(ctx, child)
        if child.keyword == "container":
            chk_container(ctx, child)
        elif child.keyword == "list":
            chk_list(ctx, child)
        else:
            pass

    if hasattr(child, 'i_children'):
        for ch in child.i_children:
            walk_child(ctx,ch)

def chk_naming(ctx, child):
    ''' Returns True if name is not hyphenated '''
    global issues
    name = child.arg
    if name == None:
        return False
    # Check for upper-case and underscore
    result = (name != name.lower() or "_" in name)
    if result:
        issues.append(str(child.pos) + " %s should not contain caps or underscore" %(name))
        
def chk_list(ctx,list_node):
    global issues
    for child in list_node.i_children:
        if child.keyword != "container":
            if child in list_node.i_key:
                continue
            issues.append(str(child.pos) + " not allowed directly under list")
        
    if list_node.parent.keyword != "container":
        issues.append(str(list_node.pos) + " should be wrapped by a container")
    if list_node.i_config:
        state_container = list_node.parent.search_one('container', 'state', list_node.i_children)
        config_container = list_node.parent.search_one('container', 'config', list_node.i_children)
        if state_container is None or config_container is None:
            issues.append(str(list_node.pos) + " config:true list must have both config and state containers")
    for key in list_node.i_key:
        key_type = key.search_one('type').arg
        if key_type != "leafref":
            if list_node.i_config and config_container is not None:
                if len(config_container.i_children) > 0:
                    issues.append(str(key.pos) + " should be a leafref")
        else:
            try:
                pointed_node = key.search_one('type').i_type_spec.i_target_node.parent.arg
            except:
                issues.append(str(key.pos) + " leafref path is invalid")
            if pointed_node != "config" and pointed_node != "state":
                issues.append(str(key.pos) + " should point to config/state attributes")
            path_ = key.search_one('type').i_type_spec.path_.arg
            if not path_.startswith('../config') and not path_.startswith('../state'):
                issues.append(str(key.pos) + " should point to ../config/<leaf> ../state/<leaf>")
            else:
                if len(key.search_one('type').i_type_spec.path_.arg.split('/')) != 3:
                    issues.append(str(key.pos) + " should point to ../config/<leaf> ../state/<leaf>")


def chk_container(ctx,container):
    global issues
    if container.arg == "config" and not container.i_config:
        issues.append(str(container.pos) + " container with name config must be config:true")
    if container.arg == "state" and container.i_config:
        issues.append(str(container.pos) + " container with name state must be config:false")

    list_node = container.search('list', container.i_children)
    if len(list_node) > 0 and len(container.i_children) > 1:
        issues.append(str(container.pos) + " Lists should have an enclosing container with no other data nodes inside it")

    if container.arg != "config" and container.i_config:
        for child in container.i_children:
            if child.keyword == "leaf" or child.keyword == "leaf-list":
                issues.append(str(child.pos) + " non-config named container cannot contain leaf/leaf-list")

    if container.arg == "config" and container.i_config:
        
        for child in container.i_children:
            if child.keyword != "leaf" and child.keyword != "leaf-list":
                issues.append(str(child.pos) + " not allowed, config container should only contain leaf/leaf-list")

        state_container = container.parent.search_one('container', 'state', container.parent.i_children)
        if state_container is None:
            issues.append(str(container.parent.pos) + " does not have state container node")
        else:
            state_childs = []                          
            for child in state_container.search('leaf', state_container.i_children):
                state_childs.append(child.arg)
            for child in state_container.search('leaf-list', state_container.i_children):
                state_childs.append(child.arg)

            for child in container.search('leaf', container.i_children):
                if child.arg not in state_childs:
                    issues.append(str(child.pos) + " must have state counterpart (../state)")
            for child in container.search('leaf-list', container.i_children):
                if child.arg not in state_childs:
                    issues.append(str(child.pos) + " must have state counterpart (../state)")

def isLeavesOnlyGrouping(grouping):
    for child in grouping.substmts:
        if child.keyword == "container" or \
            child.keyword == "list" or \
                child.keyword == "choice" or \
                    child.keyword == "anyxml" or \
                        child.keyword == "grouping" or \
                            child.keyword == "uses": 
                            return False
    return True

def chk_naming_for_grouping(ctx, module):
    global warnings
    groupingList = []
    find_node(module,"grouping", groupingList)
    for grouping in groupingList:
        name = grouping.arg
        if '-' not in name:
            warnings.append(str(grouping.pos) + " Grouping %s's name does not follow valid naming convention" %(name))
        if not name.startswith(grouping.i_module.i_modulename[11:]):
            warnings.append(str(grouping.pos) + " Grouping %s's name should begin with module name" %(name))

        last_part = name.split('-')[-1]                
        if isLeavesOnlyGrouping(grouping) and last_part not in ["config", "state", "statistics", "counters", "stats"]:
            warnings.append(str(grouping.pos) + " Grouping %s's name must end with config|state|counters|stats|statistics" %(name))        
        if not isLeavesOnlyGrouping(grouping) and last_part not in ["config", "state", "counters","stats","statistics","common","attributes","refs","structural","top"]:
            warnings.append(str(grouping.pos) + " Grouping %s's name must end with config|state|counters|stats|statistics|common|attributes|refs|structural|top" %(name))
    
    for stmt in module.substmts:
        if stmt.keyword == 'uses':
            last_part = stmt.arg.split('-')[-1]
            if last_part != "top":
                warnings.append(str(stmt.pos) + " Grouping %s's name must end with top" %(stmt.arg))

def find_node(stmt, arg, nodelist=[]):
    if stmt.keyword == arg:
        nodelist.append(stmt)
    for child in stmt.substmts:
        find_node(child,arg,nodelist)


