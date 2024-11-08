"""
This pyang plugin validates SONiC Yangs as per guidelines at
https://github.com/Azure/SONiC/blob/master/doc/mgmt/SONiC_YANG_Model_Guidelines.md
"""
import optparse

from pyang import plugin
from pyang import statements
from pyang import error
from pyang.plugins import lint
from pyang.error import err_add


# utils

def get_statement_depth(statement):
    depth = 0
    while statement.parent:
        depth += 1
        statement = statement.parent
    return depth


def get_keys(stmt):
    """Gets the key names for the node if present.
    Returns a list of key name strings.
    """
    key_obj = stmt.search_one('key')
    key_names = []
    keys = getattr(key_obj, 'arg', None)
    if keys:
        key_names = keys.split()
    return key_names


class SonicValidationRules(object):
    """Definitions of the validation rules specific for SONiC Yangs"""
    required_substmts = {
        'module': (('contact', 'organization', 'description', 'revision'), "RFC 8407: 4.8"),
        'extension': (('description',), "RFC 8407: 4.14"),
        'notification': (('description',), "RFC 8407: 4.14,4.16"),
    }

    recommended_substmts = {
        'container': (('description',), "RFC 8407: 4.14"),
        'list': (('description',), "RFC 8407: 4.14"),
        'grouping': (('description',), "RFC 8407: 4.14"),
        'enum': (('description',), "RFC 8407: 4.11.3,4.14"),
        'bit': (('description',), "RFC 8407: 4.11.3,4.14"),
        'typedef': (('description',), "RFC 8407: 4.13,4.14"),
        'leaf': (('description',), "RFC 8407: 4.14"),
        'leaf-list': (('description',), "RFC 8407: 4.14"),
        'choice': (('description',), "RFC 8407: 4.14"),
        'rpc': (('description',), "RFC 8407: 4.14"),
        'pattern': (('error-message', 'error-app-tag'), "SONiC"),
        'range': (('error-message', 'error-app-tag'), "SONiC"),
        'length': (('error-message', 'error-app-tag'), "SONiC"),
        'must': (('error-message', 'error-app-tag'), "SONiC"),
    }

    extensions_stmts = [
        'db-name',
        'key-delim',
        'key-pattern',
        'map-list',
        'map-leaf',
        'custom-validation',
        'dependent-on',
    ]

    db_names = {
        'APPL_DB': ':',
        'ASIC_DB': ':',
        'CONFIG_DB': '|',
        'COUNTERS_DB': ':',
        'STATE_DB': '|',
        'ERROR_DB': ':',
        'FLEX_COUNTER_DB': ':',
        'LOGLEVEL_DB': ':',
        'SNMP_OVERLAY_DB': '|',
        'EVENT_DB': '|',
    }


def pyang_plugin_init():
    plugin.register_plugin(SonicYangPlugin())


class SonicYangPlugin(lint.LintPlugin):
    def __init__(self):
        lint.LintPlugin.__init__(self)
        self.modulename_prefixes = ['sonic']
        self.namespace_prefixes = ['http://github.com/Azure/']
        self.modulename = None

    def add_opts(self, optparser):
        optlist = [
            optparse.make_option("--sonic",
                                 dest="sonic",
                                 action="store_true",
                                 help="Validate the module(s) according to SONiC Yang Guidelines."),
        ]
        optparser.add_options(optlist)

    def setup_ctx(self, ctx):
        if not ctx.opts.sonic:
            return

        statements.add_validation_var("$chk_default", lambda keyword: keyword in lint._keyword_with_default)
        statements.add_validation_var("$chk_required",
                                      lambda keyword: keyword in SonicValidationRules.required_substmts)
        statements.add_validation_var("$chk_recommended",
                                      lambda keyword: keyword in SonicValidationRules.recommended_substmts)

        statements.add_validation_fun('init', ['$extension'], lambda ctx, s: self.chk_extensions_stmts(ctx, s))
        statements.add_validation_fun('init', ['module'], lambda ctx, s: self.get_module_name(s))

        statements.add_validation_fun("grammar", ["$chk_default"], lambda ctx, s: lint.v_chk_default(ctx, s))
        statements.add_validation_fun("grammar", ["$chk_required"], lambda ctx, s: self.chk_required_substmt(ctx, s))
        statements.add_validation_fun("grammar", ["$chk_recommended"],
                                      lambda ctx, s: self.chk_recommended_substmt(ctx, s))

        statements.add_validation_fun("grammar", ["module"], lambda ctx, s: self.chk_module_name(ctx, s))
        statements.add_validation_fun("grammar", ["namespace"], lambda ctx, s: self.chk_namespace(ctx, s))
        statements.add_validation_fun("grammar", ["container"], lambda ctx, s: self.chk_top_container_name(ctx, s))
        statements.add_validation_fun("grammar", ["list"], lambda ctx, s: self.chk_list_name(ctx, s))

        # register error codes
        error.add_error_code('LINT_SONIC_BAD_TOP_CONTAINER_NAME', 3,
                             'Top container name "%s" must be same as module name "%s"')
        error.add_error_code('LINT_SONIC_BAD_LIST_NAME_SUFFIX', 3, 'List name "%s" must be suffixed with "_LIST"')
        error.add_error_code('LINT_SONIC_BAD_LIST_NAME_PREFIX', 3, 'List name "%s" must starts with "%s_"')
        error.add_error_code('LINT_SONIC_BAD_DYNAMIC_FIELD_KEY', 3, 'Inner List "%s" for dynamic field'
                                                                    ' must have single key leaf')
        error.add_error_code('LINT_SONIC_BAD_DYNAMIC_FIELD_VALUE', 3, 'Inner List "%s" for dynamic field'
                                                                      ' must have single non-key leaf')
        error.add_error_code('LINT_SONIC_MISSING_REQUIRED_SUBSTMT', 3, '%s: Statement "%s" must have "%s" substatement')
        error.add_error_code('LINT_SONIC_MISSING_RECOMMENDED_SUBSTMT', 4,
                             '%s: Statement "%s" should have "%s" substatement')
        error.add_error_code('LINT_SONIC_BAD_EXTENSION_STMT', 3, 'The SONiC extension "%s" is not allowed')
        error.add_error_code('LINT_SONIC_BAD_CUSTOM_VALIDATE_FUNC_NAME', 3,
                             'Custom validation extension has invalid argument "%s"')
        error.add_error_code('LINT_SONIC_BAD_DB_NAME', 3, 'db-name extension has invalid argument "%s"')
        error.add_error_code('LINT_SONIC_DEPENDENT_ON_WRONG_PLACE', 3,
                             'dependent-on extension must be defined under List only')
        error.add_error_code('LINT_SONIC_BAD_DEPENDENT_ON_ARGUMENT', 3,
                             'dependent-on extension value "%s" must be suffixed with "_LIST"')
        error.add_error_code('LINT_SONIC_BAD_DB_KEY_SEPARATOR', 3, 'key-delim extension value for DB "%s" must be "%s"')

        # ovveride these error codes denfined in linter to bump up error level
        error.add_error_code('LINT_BAD_NAMESPACE_VALUE', 3, 'Namespace value must be "%s"')
        error.add_error_code('LINT_BAD_MODULENAME_PREFIX_1', 3, 'The module name value must starts with the string %s')

    def chk_required_substmt(self, ctx, stmt):
        if not is_sonic_validatable_module(self.modulename):
            return

        if stmt.keyword in SonicValidationRules.required_substmts:
            (required, s) = SonicValidationRules.required_substmts[stmt.keyword]
            for r in required:
                if stmt.search_one(r) is None:
                    err_add(ctx.errors, stmt.pos, 'LINT_SONIC_MISSING_REQUIRED_SUBSTMT', (s, stmt.keyword, r))

    def chk_recommended_substmt(self, ctx, stmt):
        if not is_sonic_validatable_module(self.modulename):
            return

        if stmt.keyword in SonicValidationRules.recommended_substmts:
            (recommended, s) = SonicValidationRules.recommended_substmts[stmt.keyword]
            for r in recommended:
                if stmt.search_one(r) is None:
                    err_add(ctx.errors, stmt.pos, 'LINT_SONIC_MISSING_RECOMMENDED_SUBSTMT', (s, stmt.keyword, r))

    def chk_module_name(self, ctx, stmt):
        if not is_sonic_validatable_module(self.modulename):
            return

        lint.v_chk_module_name(ctx, stmt, self.modulename_prefixes)

    def chk_namespace(self, ctx, stmt):
        if not is_sonic_validatable_module(self.modulename):
            return

        lint.v_chk_namespace(ctx, stmt, self.namespace_prefixes)

    def chk_top_container_name(self, ctx, stmt):
        if not is_sonic_validatable_module(self.modulename):
            return

        if stmt.keyword == 'container' and stmt.parent.keyword == 'module':
            if stmt.arg == stmt.i_module.arg:
                return
            err_add(ctx.errors, stmt.pos, 'LINT_SONIC_BAD_TOP_CONTAINER_NAME', (stmt.arg, stmt.i_module.arg))

    def chk_list_name(self, ctx, stmt):
        if not is_sonic_validatable_module(self.modulename):
            return

        if stmt.keyword == 'list':
            # if list is inside rpc or notification, then sonic naming convention for List
            # not applicable
            parent_stmt = stmt.parent
            while parent_stmt.keyword != 'module':
                if parent_stmt.keyword in ['rpc', 'notification']:
                    return
                else:
                    parent_stmt = parent_stmt.parent

            if stmt.parent.keyword == 'container':
                if not stmt.arg.endswith('_LIST'):
                    err_add(ctx.errors, stmt.pos, 'LINT_SONIC_BAD_LIST_NAME_SUFFIX', stmt.arg)
                    return

                if not stmt.arg.startswith(stmt.parent.arg + "_"):
                    err_add(ctx.errors, stmt.pos, 'LINT_SONIC_BAD_LIST_NAME_PREFIX', (stmt.arg, stmt.parent.arg))
                    return

            # Dynamic fields - Making sure inner LIST has exactly one key leaf and one non-key leaf
            if get_statement_depth(stmt) == 4:
                key_leaves = get_keys(stmt)
                if len(key_leaves) != 1:
                    err_add(ctx.errors, stmt.pos, 'LINT_SONIC_BAD_DYNAMIC_FIELD_KEY', stmt.arg)
                    return
                non_key_leaves = [leaf for leaf in stmt.substmts if leaf.keyword == 'leaf' and leaf.arg not in key_leaves]
                if len(non_key_leaves) != 1:
                    err_add(ctx.errors, stmt.pos, 'LINT_SONIC_BAD_DYNAMIC_FIELD_VALUE', stmt.arg)
                    return

    def get_module_name(self, stmt):
        self.modulename = stmt.i_modulename

    def chk_extensions_stmts(self, ctx, stmt):
        if not is_sonic_validatable_module(self.modulename):
            return

        if stmt.keyword[0] == 'sonic-extension' and stmt.keyword[1] not in SonicValidationRules.extensions_stmts:
            err_add(ctx.errors, stmt.pos, 'LINT_SONIC_BAD_EXTENSION_STMT', (stmt.keyword[1]))

        if stmt.keyword[1] == 'custom-validation' and not stmt.arg.isidentifier():
            err_add(ctx.errors, stmt.pos, 'LINT_SONIC_BAD_CUSTOM_VALIDATE_FUNC_NAME', (stmt.arg))

        if stmt.keyword[1] == 'dependent-on':
            if stmt.parent.keyword != 'list':
                err_add(ctx.errors, stmt.pos, 'LINT_SONIC_DEPENDENT_ON_WRONG_PLACE', ())
            elif not stmt.arg.endswith('_LIST'):
                err_add(ctx.errors, stmt.pos, 'LINT_SONIC_BAD_DEPENDENT_ON_ARGUMENT', (stmt.arg))

        if stmt.keyword[1] == 'db-name':
            if stmt.arg not in SonicValidationRules.db_names:
                err_add(ctx.errors, stmt.pos, 'LINT_SONIC_BAD_DB_NAME', (stmt.arg))
            else:
                separator = SonicValidationRules.db_names[stmt.arg]
                keydelim_stmt = search_child_stmt(stmt.parent, 'key-delim')
                if keydelim_stmt is not None and keydelim_stmt.arg != separator:
                    err_add(ctx.errors, stmt.pos, 'LINT_SONIC_BAD_DB_KEY_SEPARATOR', (stmt.arg, separator))


def is_sonic_validatable_module(mod):
    """ Skip modules like sonic-extension and sonic-common and IETF"""
    if mod[0:mod.index('-')] in ['ietf', 'iana']:
        return False
    elif mod in ['sonic-extension', 'sonic-common', 'sonic-interface-common']:
        return False

    return True


def search_child_stmt(stmt, keyword):
    children = stmt.substmts
    for ch in children:
        if isinstance(ch.keyword, str) and ch.keyword == keyword:
            return ch
        if isinstance(ch.keyword, tuple) and len(ch.keyword) >= 2 and ch.keyword[1] == keyword:
            return ch

    return None
