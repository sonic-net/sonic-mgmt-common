"""YANG module update check tool
This plugin checks if an updated version of a module follows
the rules defined in Section 10 of RFC 6020 and Section 11 of RFC 7950.
Developed by Broadcom Inc.
"""

import optparse, sys, copy

import pyang
if pyang.__version__ > '2.4':
    from pyang.repository import FileRepository
    from pyang.context import Context
else:
    from pyang import FileRepository
    from pyang import Context
from pyang import plugin
from pyang import statements
from pyang import error
from pyang import util
from pyang import types

# Globals
log = sys.stderr.write
use_lint_ignore = True
current_data_node = None

def pyang_plugin_init():
    plugin.register_plugin(CheckDeviationPlugin())

class CheckDeviationPlugin(plugin.PyangPlugin):
    
    def add_output_format(self, fmts):
        self.multiple_modules = True
        fmts['upcheck'] = self

    def add_opts(self, optparser):
        optlist = [
            optparse.make_option("--basepath",
                                 dest="basedir",
                                 help="Directory path for non-extension modules"),
            optparse.make_option("--disable-lint-ignore",
                                 dest="disable_lint",
                                 action="store_true",
                                 help="Treat lint-ignore as error"), 
            ]
        optparser.add_options(optlist)

    def setup_ctx(self, ctx):
        # register our error codes
        error.add_error_code(
            'CHK_INVALID_MODULENAME', 1,
            "the module's name MUST NOT be changed"
            + " (RFC 6020: 10, p3)")
        error.add_error_code(
            'CHK_INVALID_NAMESPACE', 1,
            "the module's namespace MUST NOT be changed"
            + " (RFC 6020: 10, p3)")
        error.add_error_code(
            'CHK_NO_REVISION', 1,
            "a revision statement MUST be present"
            + " (RFC 6020: 10, p2)")
        error.add_error_code(
            'CHK_BAD_REVISION', 1,
            "new revision %s is not newer than old revision %s"
            + " (RFC 6020: 10, p2)")
        error.add_error_code(
            'CHK_DEF_REMOVED', 1,
            "the %s '%s', defined at %s is illegally removed")
        error.add_error_code(
            'CHK_DEF_ADDED', 1,
            "the %s '%s' is illegally added")
        error.add_error_code(
            'CHK_DEF_CHANGED', 4,
            "the %s '%s' is illegally changed from '%s'")
        error.add_error_code(
            'CHK_INVALID_STATUS', 1,
            "new status %s is not valid since the old status was %s")
        error.add_error_code(
            'CHK_CHILD_KEYWORD_CHANGED', 1,
            "the %s '%s' is illegally changed to a %s")
        error.add_error_code(
            'CHK_MANDATORY_CONFIG', 1,
            "the node %s is changed to config true, but it is mandatory")
        error.add_error_code(
            'CHK_NEW_MANDATORY', 1,
            "the mandatory node %s is illegally added")
        error.add_error_code(
            'CHK_BAD_CONFIG', 1,
            "the node %s is changed to config false")
        error.add_error_code(
            'CHK_NEW_MUST', 1,
            "a new must expression cannot be added")
        error.add_error_code(
            'CHK_UNDECIDED_MUST', 4,
            "this must expression may be more constrained than before")
        error.add_error_code(
            'CHK_NEW_WHEN', 1,
            "a new when expression cannot be added")
        error.add_error_code(
            'CHK_UNDECIDED_WHEN', 4,
            "this when expression may be different than before")
        error.add_error_code(
            'CHK_UNDECIDED_PRESENCE', 4,
            "this presence expression may be different than before")
        error.add_error_code(
            'CHK_IMPLICIT_DEFAULT', 1,
            "the leaf had an implicit default")
        error.add_error_code(
            'CHK_BASE_TYPE_CHANGED', 1,
            "the base type has illegally changed from %s to %s")
        error.add_error_code(
            'CHK_LEAFREF_PATH_CHANGED', 1,
            "the leafref's path has illegally changed")
        error.add_error_code(
            'CHK_ENUM_VALUE_CHANGED', 1,
            "the value for enum '%s', has changed from %s to %s"
            + " (RFC 6020: 10, p5, bullet 1)")
        error.add_error_code(
            'CHK_BIT_POSITION_CHANGED', 1,
            "the position for bit '%s', has changed from %s to %s"
            + " (RFC 6020: 10, p5, bullet 2)")
        error.add_error_code(
            'CHK_RESTRICTION_CHANGED', 1,
            "the %s has been illegally restricted"
            + " (RFC 6020: 10, p5, bullet 3)")
        error.add_error_code(
            'CHK_UNION_TYPES', 1,
            "the member types in the union have changed")
        error.add_error_code(
            'DEVIATION_ERROR', 1,
            "caused by deviation stmt, please check deviation in this file")            

    def setup_fmt(self, ctx):
        ctx.implicit_errors = False

    def emit(self, ctx, modules, fd):
        global use_lint_ignore
        use_lint_ignore = not(ctx.opts.disable_lint)
        ctx.errors = []

        self.validate(ctx, modules)
        
        error_seen = False
        if ctx.opts.outfile is not None:
            fd = open(ctx.opts.outfile, "w")        
        for (epos, etag, eargs, ignore_lint) in ctx.errors:
            elevel = error.err_level(etag)
            
            if error.is_warning(elevel):
                kind = "warning"
            else:
                if ignore_lint:
                    kind = "ignored"
                else:
                    kind = "error"
                    error_seen = True
            
            fd.write(str(epos) + ': %s: ' % kind + \
                error.err_to_str(etag, eargs) + '\n')                

        if ctx.opts.outfile is not None:
            fd.close()
        if error_seen:
            sys.exit(1)
        else:
            sys.exit(0)

    def validate(self, ctx, modules):
        v_modules = []
        def debug(s): log(s) if ctx.opts.verbose else None
        # Identify & tag extensions and standard yang modules
        for mod in modules:
            if is_extension_yang(mod.pos):
                debug(f"Collecting deviation info from {mod.pos.ref}\n")
                mark_deviations(mod)
            else:
                v_modules.append(ModPair(new=mod))
        # Load standard yang modules without any extension modules
        oldctx = create_base_context(ctx)
        for m in v_modules:
            mod_name = m.new.i_modulename
            debug(f"Loading base version of {mod_name}\n")
            m.old = oldctx.search_module(None, mod_name)
        # Collect parser errors if any...
        for pos, tag, args in oldctx.errors:
            if error.is_error(error.err_level(tag)):
                pyang_err_add(ctx.errors, pos, tag, args, False)
        # Validate and compare original and updated modules
        debug(f"Validating base versions...")
        oldctx.validate()
        for m in v_modules:
            check_update(ctx, m.old, m.new)

class ModPair:
    def __init__(self, old=None, new=None):
        self.new = new
        self.old = old

def is_extension_yang(pos):
    return pos.ref.startswith("extensions/") or "/extensions/" in pos.ref

def create_base_context(ctx):
    repo = ctx.repository
    if ctx.opts.basedir:
        repo = FileRepository(ctx.opts.basedir, use_env=False,
                              verbose=ctx.opts.verbose)
    newctx = Context(repo)
    newctx.opts = ctx.opts
    newctx.strict = ctx.strict
    newctx.lax_xpath_checks = ctx.lax_xpath_checks
    newctx.lax_quote_checks = ctx.lax_quote_checks
    return newctx

def get_lint_ignore(node):
    name = None
    for substmt in node.substmts: 
        if substmt.keyword.__class__.__name__ == 'tuple':
            if substmt.keyword[0] == 'sonic-codegen':
                if substmt.keyword[1] == 'lint-ignore':
                    name = substmt.arg
    return name

def mark_deviations(module):
    for deviation in module.search('deviation'):
        if not hasattr(deviation.i_target_node, "deviation_list"):
            deviation.i_target_node.deviation_list = []
        deviation.i_target_node.deviation_list.append(deviation)
        if get_lint_ignore(deviation) is not None:
            deviation.i_target_node.lint_ignore = True

def check_update(ctx, oldmod, newmod):
    if oldmod is None or newmod is None:
        return
    if ctx.opts.verbose:
        log(f"Upgrade check for {newmod.i_modulename}\n")

    chk_modulename(oldmod, newmod, ctx)

    chk_namespace(oldmod, newmod, ctx)

    #chk_revision(oldmod, newmod, ctx)

    for olds in oldmod.search('feature'):
        chk_feature(olds, newmod, ctx)

    for olds in oldmod.search('identity'):
        chk_identity(olds, newmod, ctx)

    for olds in oldmod.search('typedef'):
        chk_typedef(olds, newmod, ctx)

    for olds in oldmod.search('grouping'):
        chk_grouping(olds, newmod, ctx)

    for olds in oldmod.search('rpc'):
        chk_rpc(olds, newmod, ctx)

    for olds in oldmod.search('notification'):
        chk_notification(olds, newmod, ctx)

    for olds in oldmod.search('extension'):
        chk_extension(olds, newmod, ctx)

    for olds in oldmod.search('augment'):
        chk_augment(olds, newmod, ctx)

    chk_i_children(oldmod, newmod, ctx)

def chk_modulename(oldmod, newmod, ctx):
    if oldmod.i_modulename != newmod.i_modulename:
        err_add(ctx.errors, newmod.pos, 'CHK_INVALID_MODULENAME', ())

def chk_namespace(oldmod, newmod, ctx):
    oldns = oldmod.search_one('namespace')
    newns = newmod.search_one('namespace')
    if oldns is not None and newns is not None and oldns.arg != newns.arg:
        err_add(ctx.errors, newmod.pos, 'CHK_INVALID_NAMESPACE', ())

def chk_revision(oldmod, newmod, ctx):
    oldrev = get_latest_revision(oldmod)
    newrev = get_latest_revision(newmod)
    if newrev is None:
        err_add(ctx.errors, newmod.pos, 'CHK_NO_REVISION', ())
    elif (oldrev is not None) and (oldrev > newrev):
        err_add(ctx.errors, newmod.pos, 'CHK_BAD_REVISION', (newrev, oldrev))

def get_latest_revision(m):
    revs = [r.arg for r in m.search('revision')]
    revs.sort()
    if len(revs) > 0:
        return revs[-1]
    else:
        return None

def chk_feature(olds, newmod, ctx):
    chk_stmt(olds, newmod, ctx)

def chk_identity(olds, newmod, ctx):
    news = chk_stmt(olds, newmod, ctx)
    if news is None:
        return
    # make sure the base isn't changed (other than syntactically)
    oldbases = olds.search('base')
    newbases = news.search('base')
    if newmod.i_version == '1.1':
        old_ids = [oldbase.i_identity.arg for oldbase in oldbases]
        new_ids = [newbase.i_identity.arg for newbase in newbases]
        for old_id in set(old_ids) - set(new_ids):
            err_def_removed(oldbases[old_ids.index(old_id)], news, ctx)
        for old_id in set(old_ids) & set(new_ids):
            oldbase = oldbases[old_ids.index(old_id)]
            newbase = newbases[new_ids.index(old_id)]
            if oldbase.i_identity.i_module.i_modulename != \
               newbase.i_identity.i_module.i_modulename:
                err_def_changed(oldbase, newbase, ctx)
    else:
        oldbase = next(iter(oldbases), None)
        newbase = next(iter(newbases), None)
        if oldbase is None and newbase is not None:
            err_def_added(newbase, ctx)
        elif newbase is None and oldbase is not None:
            err_def_removed(oldbase, news, ctx)
        elif oldbase is None and newbase is None:
            pass
        elif ((oldbase.i_identity.i_module.i_modulename !=
               newbase.i_identity.i_module.i_modulename)
              or (oldbase.i_identity.arg != newbase.i_identity.arg)):
            err_def_changed(oldbase, newbase, ctx)

def chk_typedef(olds, newmod, ctx):
    news = chk_stmt(olds, newmod, ctx)
    if news is None:
        return
    chk_type(olds.search_one('type'), news.search_one('type'), ctx)

def chk_grouping(olds, newmod, ctx):
    news = chk_stmt(olds, newmod, ctx)
    if news is None:
        return
    chk_i_children(olds, news, ctx)

def chk_rpc(olds, newmod, ctx):
    news = chk_stmt(olds, newmod, ctx)
    if news is None:
        return
    chk_i_children(olds, news, ctx)

def chk_notification(olds, newmod, ctx):
    news = chk_stmt(olds, newmod, ctx)
    if news is None:
        return
    chk_i_children(olds, news, ctx)

def chk_extension(olds, newmod, ctx):
    news = chk_stmt(olds, newmod, ctx)
    if news is None:
        return
    oldarg = olds.search_one('argument')
    newarg = news.search_one('argument')
    if oldarg is None and newarg is not None:
        err_def_added(newarg, ctx)
    elif oldarg is not None and newarg is None:
        err_def_removed(oldarg, newmod, ctx)
    elif oldarg is not None and newarg is not None:
        oldyin = oldarg.search_one('yin-element')
        newyin = newarg.search_one('yin-element')
        if oldyin is None and newyin is not None and newyin.arg != 'false':
            err_def_added(newyin, ctx)
        elif oldyin is not None and newyin is None and oldyin.arg != 'false':
            err_def_removed(oldyin, newarg, ctx)
        elif (oldyin is not None and newyin is not None and
              newyin.arg != oldyin.arg):
            err_def_changed(oldyin, newyin, ctx)

def chk_augment(olds, newmod, ctx):
    ## this is not quite correct; it should be ok to change the
    ## prefix, so augmenting /x:a in the old module, but /y:a in the
    ## new module, if x and y are prefixes to the same module, should
    ## be ok.
    news = chk_stmt(olds, newmod, ctx)
    if news is None:
        return
    chk_i_children(olds, news, ctx)

def chk_stmt(olds, newp, ctx):
    news = newp.search_one(olds.keyword, arg = olds.arg)
    if news is None:
        err_def_removed(olds, newp, ctx)
        return None
    chk_status(olds, news, ctx)
    chk_if_feature(olds, news, ctx)
    return news

def chk_i_children(old, new, ctx):
    for oldch in old.i_children:
        chk_child(oldch, new, ctx)
    # chk_child removes all old children
    for newch in new.i_children:
        if statements.is_mandatory_node(newch):
            err_add(ctx.errors, newch.pos, 'CHK_NEW_MANDATORY', newch.arg, newch)

def chk_child(oldch, newp, ctx):

    global current_data_node

    newch = None
    for ch in newp.i_children:
        if ch.arg == oldch.arg:
            newch = ch
            break
    if newch is None:
        err_def_removed(oldch, newp, ctx)
        return
    newp.i_children.remove(newch)
    if newch.keyword != oldch.keyword:
        err_add(ctx.errors, newch.pos, 'CHK_CHILD_KEYWORD_CHANGED',
                (oldch.keyword, newch.arg, newch.keyword), newch)
        return
    current_data_node = newch
    chk_status(oldch, newch, ctx)
    chk_if_feature(oldch, newch, ctx)
    chk_config(oldch, newch, ctx)
    chk_must(oldch, newch, ctx)
    chk_when(oldch, newch, ctx)

    if newch.keyword == 'leaf':
        chk_leaf(oldch, newch, ctx)
    elif newch.keyword == 'leaf-list':
        chk_leaf_list(oldch, newch, ctx)
    elif newch.keyword == 'container':
        chk_container(oldch, newch, ctx)
    elif newch.keyword == 'list':
        chk_list(oldch, newch, ctx)
    elif newch.keyword == 'choice':
        chk_choice(oldch, newch, ctx)
    elif newch.keyword == 'case':
        chk_case(oldch, newch, ctx)
    elif newch.keyword == 'input':
        chk_input_output(oldch, newch, ctx)
    elif newch.keyword == 'output':
        chk_input_output(oldch, newch, ctx)

    current_data_node = None #reset

def chk_status(old, new, ctx):
    oldstatus = old.search_one('status')
    newstatus = new.search_one('status')
    if oldstatus is None or oldstatus.arg == 'current':
        # any new status is ok
        return
    if newstatus is None:
        err_add(ctx.errors, new.pos, 'CHK_INVALID_STATUS',
                ("(implicit) current", oldstatus.arg), new)
    elif ((newstatus.arg == 'current') or
          (oldstatus.arg == 'obsolete' and newstatus.arg != 'obsolete')):
        err_add(ctx.errors, newstatus.pos, 'CHK_INVALID_STATUS',
                (newstatus.arg, oldstatus.arg), new)

def chk_if_feature(old, new, ctx):
    # make sure no if-features are removed if node is mandatory
    for s in old.search('if-feature'):
        if new.search_one('if-feature', arg=s.arg) is None:
            if statements.is_mandatory_node(new):
                err_def_removed(s, new, ctx)

    # make sure no if-features are added
    for s in new.search('if-feature'):
        if old.search_one('if-feature', arg=s.arg) is None:
            err_def_added(s, ctx)

def chk_config(old, new, ctx):
    if old.i_config == False and new.i_config == True:
        if statements.is_mandatory_node(new):
            err_add(ctx.errors, new.pos, 'CHK_MANDATORY_CONFIG', new.arg)
    elif old.i_config == True and new.i_config == False:
        err_add(ctx.errors, new.pos, 'CHK_BAD_CONFIG', new.arg, new)

def chk_must(old, new, ctx):
    oldmust = old.search('must')
    newmust = new.search('must')
    # remove all common musts
    for oldm in old.search('must'):
        newm = new.search_one('must', arg = oldm.arg)
        if newm is not None:
            newmust.remove(newm)
            oldmust.remove(oldm)
    if len(newmust) == 0:
        # this is good; maybe some old musts were removed
        pass
    elif len(oldmust) == 0:
        for newm in newmust:
            err_add(ctx.errors, newm.pos, 'CHK_NEW_MUST', (), new)
    else:
        for newm in newmust:
            err_add(ctx.errors, newm.pos, 'CHK_UNDECIDED_MUST', (), new)

def chk_when(old, new, ctx):
    oldwhen = old.search('when')
    newwhen = new.search('when')
    # remove all common whens
    for oldw in old.search('when'):
        neww = new.search_one('when', arg = oldw.arg)
        if neww is not None:
            newwhen.remove(neww)
            oldwhen.remove(oldw)
    if new.i_module.i_version == '1.1':
        if len(newwhen) == 0:
            # this is good; maybe some old whens were removed
            return
    elif len(oldwhen) == 0:
        for neww in newwhen:
            err_add(ctx.errors, neww.pos, 'CHK_NEW_WHEN', ())
    else:
        for neww in newwhen:
            err_add(ctx.errors, neww.pos, 'CHK_UNDECIDED_WHEN', ())

def chk_units(old, new, ctx):
    oldunits = old.search_one('units')
    if oldunits is None:
        return
    newunits = new.search_one('units')
    if newunits is None:
        err_def_removed(oldunits, new, ctx, new)
    elif newunits.arg != oldunits.arg:
        err_def_changed(oldunits, newunits, ctx, new)

def chk_default(old, new, ctx):
    newdefault = new.search_one('default')
    olddefault = old.search_one('default')
    if olddefault is None and newdefault is None:
        return
    if olddefault is not None and newdefault is None:
        err_def_removed(olddefault, new, ctx, new)
    elif olddefault is None and newdefault is not None:
        # default added, check old implicit default
        oldtype = old.search_one('type')
        if (oldtype.i_typedef is not None and
            hasattr(oldtype.i_typedef, 'i_default_str') and
            oldtype.i_typedef.i_default_str is not None and
            oldtype.i_typedef.i_default_str != newdefault.arg):
            err_add(ctx.errors, newdefault.pos, 'CHK_IMPLICIT_DEFAULT', (), new)
    elif olddefault.arg != newdefault.arg:
        err_def_changed(olddefault, newdefault, ctx, new)

def chk_mandatory(old, new, ctx):
    oldmandatory = old.search_one('mandatory')
    newmandatory = new.search_one('mandatory')
    if newmandatory is not None and newmandatory.arg == 'true':
        if oldmandatory is None:
            err_def_added(newmandatory, ctx, new)
        elif oldmandatory.arg == 'false':
            err_def_changed(oldmandatory, newmandatory, ctx, new)

def chk_min_max(old, new, ctx):
    oldmin = old.search_one('min-elements')
    newmin = new.search_one('min-elements')
    if newmin is None:
        pass
    elif oldmin is None:
        err_def_added(newmin, ctx, new)
    elif int(newmin.arg) > int(oldmin.arg):
        err_def_changed(oldmin, newmin, ctx, new)
    oldmax = old.search_one('max-elements')
    newmax = new.search_one('max-elements')
    if oldmax is None:
        pass
    elif newmax is None:
        pass
    elif int(newmax.arg) < int(oldmax.arg):
        err_def_changed(oldmax, newmax, ctx, new)

def chk_presence(old, new, ctx):
    oldpresence = old.search_one('presence')
    newpresence = new.search_one('presence')
    if oldpresence is None and newpresence is None:
        pass
    elif oldpresence is None and newpresence is not None:
        err_def_added(newpresence, ctx)
    elif oldpresence is not None and newpresence is None:
        err_def_removed(oldpresence, new, ctx)
    elif oldpresence.arg != newpresence.arg:
        err_add(ctx.errors, newpresence.pos, 'CHK_UNDECIDED_PRESENCE', ())

def chk_key(old, new, ctx):
    oldkey = old.search_one('key')
    newkey = new.search_one('key')
    if oldkey is None and newkey is None:
        pass
    elif oldkey is None and newkey is not None:
        err_def_added(newkey, ctx)
    elif oldkey is not None and newkey is None:
        err_def_removed(oldkey, new, ctx)
    else:
        # check the key argument string; i_key is not set in groupings
        oldks = [k for k in oldkey.arg.split() if k != '']
        newks = [k for k in newkey.arg.split() if k != '']
        if len(oldks) != len(newks):
            err_def_changed(oldkey, newkey, ctx)
        else:
            def name(x):
                if x.find(":") == -1:
                    return x
                else:
                    [prefix, name] = x.split(':', 1)
                    return name
            for (ok, nk) in zip(oldks, newks):
                if name(ok) != name(nk):
                    err_def_changed(oldkey, newkey, ctx)
                    return

def chk_unique(old, new, ctx):
    # do not check the unique argument string; check the parsed unique instead
    # i_unique is not set in groupings; ignore
    if not hasattr(old, 'i_unique') or not hasattr(new, 'i_unique'):
        return
    oldunique = []
    for (u, l) in old.i_unique:
        oldunique.append((u, [s.arg for s in l]))
    for (u, l) in new.i_unique:
        # check if this unique was present before
        o = util.keysearch([s.arg for s in l], 1, oldunique)
        if o is not None:
            oldunique.remove(o)
        else:
            err_def_added(u, ctx)

def chk_leaf(old, new, ctx):
    old_t = old.search_one('type')
    new_t = new.search_one('type')
    chk_type(old_t, new_t, ctx)
    chk_units(old, new, ctx)
    chk_default(old, new, ctx)
    chk_mandatory(old, new, ctx)

def chk_leaf_list(old, new, ctx):
    chk_type(old.search_one('type'), new.search_one('type'), ctx)
    chk_units(old, new, ctx)
    chk_min_max(old, new, ctx)

def chk_container(old, new, ctx):
    chk_presence(old, new, ctx)
    chk_i_children(old, new, ctx)

def chk_list(old, new, ctx):
    chk_min_max(old, new, ctx)
    chk_key(old, new, ctx)
    chk_unique(old, new, ctx)
    chk_i_children(old, new, ctx)

def chk_choice(old, new, ctx):
    chk_mandatory(old, new, ctx)
    chk_i_children(old, new, ctx)

def chk_case(old, new, ctx):
    chk_i_children(old, new, ctx)

def chk_input_output(old, new, ctx):
    chk_i_children(old, new, ctx)

def chk_type(old, new, ctx):
    oldts = old.i_type_spec
    newts = new.i_type_spec
    if oldts is None or newts is None:
        return
    # verify that the base type is the same
    if oldts.name != newts.name:
        err_add(ctx.errors, new.pos, 'CHK_BASE_TYPE_CHANGED',
                (oldts.name, newts.name), new)
        return

    # check the allowed restriction changes
    if oldts.name in chk_type_func:
        chk_type_func[oldts.name](old, new, oldts, newts, ctx)

def chk_integer(old, new, oldts, newts, ctx):
    chk_range(old, new, oldts, newts, ctx)

def chk_range(old, new, oldts, newts, ctx):
    ots = old.i_type_spec
    nts = new.i_type_spec
    if (type(ots) == types.RangeTypeSpec and
        type(nts) == types.RangeTypeSpec):
        tmperrors = []
        types.validate_ranges(tmperrors, new.pos, ots.ranges, new)
        if tmperrors != []:
            err_add(ctx.errors, new.pos, 'CHK_RESTRICTION_CHANGED',
                    'range', new)

def chk_decimal64(old, new, oldts, newts, ctx):
    oldbasets = get_base_type(oldts)
    newbasets = get_base_type(newts)
    if newbasets.fraction_digits != oldbasets.fraction_digits:
        err_add(ctx.errors, new.pos, 'CHK_DEF_CHANGED',
                ('fraction-digits', newts.fraction_digits,
                 oldts.fraction_digits), new)
    # a decimal64 can only be restricted with range
    chk_range(old, new, oldts, newts, ctx)

def get_base_type(ts):
    if ts.base is None:
        return ts
    else:
        return get_base_type(ts.base)

def chk_length(old, new, oldts, newts, ctx):
    ots = old.i_type_spec
    nts = new.i_type_spec
    if (type(ots) == types.LengthTypeSpec and
        type(nts) == types.LengthTypeSpec):
        tmperrors = []
        validate_lengths(tmperrors, new.pos, ots.lengths, new)
        if tmperrors != []:
            err_add(ctx.errors, new.pos, 'CHK_RESTRICTION_CHANGED',
                    'length', new)

def chk_string(old, new, oldts, newts, ctx):
    # FIXME: see types.py; we can't check the length
    #chk_length(old, new, oldts, newts, ctx)
    return

def chk_enumeration(old, new, oldts, newts, ctx):
    # verify that all old enums are still in new, with the same values
    for (name, val) in oldts.enums:
        n = util.keysearch(name, 0, newts.enums)
        if n is None:
            err_add(ctx.errors, new.pos, 'CHK_DEF_REMOVED',
                    ('enum', name, old.pos), new)
        elif n[1] != val:
            err_add(ctx.errors, new.pos, 'CHK_ENUM_VALUE_CHANGED',
                    (name, val, n[1]), new)

def chk_bits(old, new, oldts, newts, ctx):
    # verify that all old bits are still in new, with the same positions
    for (name, pos) in oldts.bits:
        n = util.keysearch(name, 0, newts.bits)
        if n is None:
            err_add(ctx.errors, new.pos, 'CHK_DEF_REMOVED',
                    ('bit', name, old.pos), new)
        elif n[1] != pos:
            err_add(ctx.errors, new.pos, 'CHK_BIT_POSITION_CHANGED',
                    (name, pos, n[1]), new)

def chk_binary(old, new, oldts, newts, ctx):
    # FIXME: see types.py; we can't check the length
    return

def chk_leafref(old, new, oldts, newts, ctx):
    # verify that the path refers to the same leaf
    if (not hasattr(old.parent, 'i_leafref_ptr') or
        not hasattr(new.parent, 'i_leafref_ptr')):
        return
    if (old.parent.i_leafref_ptr is None or
        new.parent.i_leafref_ptr is None):
        return
    def cmp_node(optr, nptr):
        if optr.parent is None:
            return
        if (optr.i_module.i_modulename == nptr.i_module.i_modulename and
            optr.arg == nptr.arg):
            return cmp_node(optr.parent, nptr.parent)
        else:
            err_add(ctx.errors, new.pos, 'CHK_LEAFREF_PATH_CHANGED', (), new)
    cmp_node(old.parent.i_leafref_ptr[0], new.parent.i_leafref_ptr[0])

def chk_identityref(old, new, oldts, newts, ctx):
    # verify that the bases are the same
    extra = [n for n in newts.idbases]

    for oidbase in oldts.idbases:
        for nidbase in newts.idbases:
            if (nidbase.i_module.i_modulename ==
                    oidbase.i_module.i_modulename and
                    nidbase.arg.split(':')[-1] == oidbase.arg.split(':')[-1]):
                extra.remove(nidbase)
            if (nidbase.i_identity.i_module.i_modulename ==
                    oidbase.i_identity.i_module.i_modulename and
                    nidbase.arg.split(':')[-1] == oidbase.arg.split(':')[-1]):
                if nidbase in extra:
                    extra.remove(nidbase)
    for n in extra:
        err_add(ctx.errors, n.pos, 'CHK_DEF_ADDED',
                ('base', n.arg), new)

def chk_instance_identifier(old, new, oldts, newts, ctx):
    # FIXME:
    return

def chk_union(old, new, oldts, newts, ctx):
    if len(newts.types) != len(oldts.types):
        err_add(ctx.errors, new.pos, 'CHK_UNION_TYPES', (), new)
    else:
        for (o,n) in zip(oldts.types, newts.types):
            chk_type(o, n, ctx)

def chk_dummy(old, new, oldts, newts, ctx):
    return

chk_type_func = \
  {'int8': chk_integer,
   'int16': chk_integer,
   'int32': chk_integer,
   'int64': chk_integer,
   'uint8': chk_integer,
   'uint16': chk_integer,
   'uint32': chk_integer,
   'uint64': chk_integer,
   'decimal64': chk_decimal64,
   'string': chk_string,
   'boolean': chk_dummy,
   'enumeration': chk_enumeration,
   'bits': chk_bits,
   'binary': chk_binary,
   'leafref': chk_leafref,
   'identityref': chk_identityref,
   'instance-identifier': chk_instance_identifier,
   'empty': chk_dummy,
   'union': chk_union}


def validate_lengths(errors, pos, ranges, type_):
    # make sure the range values are of correct type and increasing
    cur_lo = None
    for lo, hi in ranges:
        if isinstance(type_.i_type_spec, types.LengthTypeSpec):
            type_.i_type_spec.validate(errors, pos, (lo, hi), "")
        else:
            if lo is not None and lo != 'min' and lo != 'max':
                type_.i_type_spec.validate(errors, pos, lo, "")
            if hi is not None and hi != 'min' and hi != 'max':
                type_.i_type_spec.validate(errors, pos, hi, "")
        # check that cur_lo < lo < hi
        if not types.is_smaller(cur_lo, lo):
            err_add(errors, pos, 'RANGE_BOUNDS', (str(lo), cur_lo))
            return None
        if not types.is_smaller(lo, hi):
            err_add(errors, pos, 'RANGE_BOUNDS', (str(hi), str(lo)))
            return None
        if (lo == 'max' and cur_lo is not None
                and cur_lo >= type_.i_type_spec.max):
            err_add(errors, pos, 'RANGE_BOUNDS', (str(lo), str(cur_lo)))
            return None
        if hi is None:
            cur_lo = lo
        else:
            cur_lo = hi
    return (ranges, pos)

def err_def_added(new, ctx, data_node=None):
    err_add(ctx.errors, new.pos, 'CHK_DEF_ADDED', (new.keyword, new.arg), data_node)

def err_def_removed(old, newp, ctx, data_node=None):
    #suppressing as we support "not-supported" deviation
    return
    # err_add(ctx.errors, newp.pos, 'CHK_DEF_REMOVED',
    #         (old.keyword, old.arg, old.pos), data_node)

def err_def_changed(old, new, ctx, data_node=None):
    err_add(ctx.errors, new.pos, 'CHK_DEF_CHANGED',
            (new.keyword, new.arg, old.arg), data_node)

def pyang_err_add(errors, pos, tag, args, ignore_error=False):
    error = (copy.copy(pos), tag, args, ignore_error)
    for (i_pos, i_tag, i_args, i_ignore) in errors:
        if (i_pos.line == pos.line and i_pos.ref == pos.ref and
            i_pos.top == pos.top and i_tag == tag and i_args == args):
            return
    errors.append(error)

def is_lint_ignore(node):
    if node != None and hasattr(node, 'lint_ignore'):
        return True
    return False

def err_add(ctx, pos, err_code, extra, data_node=None):
    ignore_error = use_lint_ignore and is_lint_ignore(current_data_node)
    if data_node != None and hasattr(data_node, 'deviation_list'):
        for deviation in data_node.deviation_list:
            pyang_err_add(ctx, deviation.pos, err_code, extra, ignore_error)
    elif current_data_node != None and hasattr(current_data_node, 'deviation_list'):
        for deviation in current_data_node.deviation_list:
            pyang_err_add(ctx, deviation.pos, err_code, extra, ignore_error)
    else:
        pyang_err_add(ctx, pos, err_code, extra, ignore_error)


