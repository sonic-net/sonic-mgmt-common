"""Declare management model preparation and the existing owner generators."""

_TAR_TOOLCHAIN = "@tar.bzl//tar/toolchain:type"

def _prepared_models_impl(ctx):
    output = ctx.actions.declare_directory(ctx.label.name)
    tar = ctx.toolchains[_TAR_TOOLCHAIN]
    args = ctx.actions.args()
    args.add("--imports", ctx.file.imports)
    args.add("--owner-wheel", ctx.file.owner_wheel)
    args.add("--tar", tar.tarinfo.binary)
    args.add("--output", output.path)
    args.add_all(ctx.files.sources, before_each = "--source")
    args.add_all(ctx.files.test_sources, before_each = "--test-source")
    ctx.actions.run(
        executable = ctx.executable._preparer,
        arguments = [args],
        inputs = depset(
            [ctx.file.imports, ctx.file.owner_wheel] + ctx.files.sources + ctx.files.test_sources,
            transitive = [tar.default.files],
        ),
        tools = [ctx.attr._preparer[DefaultInfo].files_to_run],
        outputs = [output],
        env = {"PYTHONHASHSEED": "0", "PYTHONNOUSERSITE": "1"},
        execution_requirements = {"block-network": "1"},
        mnemonic = "MgmtYangModels",
    )
    return [DefaultInfo(files = depset([output]))]

prepared_models = rule(
    implementation = _prepared_models_impl,
    attrs = {
        "sources": attr.label_list(allow_files = [".yang"], mandatory = True),
        "test_sources": attr.label_list(allow_files = [".yang"]),
        "imports": attr.label(allow_single_file = True, mandatory = True),
        "owner_wheel": attr.label(allow_single_file = True, mandatory = True),
        "_preparer": attr.label(default = "//tools/bazel:prepare_models", executable = True, cfg = "exec"),
    },
    toolchains = [_TAR_TOOLCHAIN],
)

def _schema_impl(ctx):
    output = ctx.actions.declare_directory(ctx.label.name)
    args = ctx.actions.args()
    if ctx.files.test_sources:
        args.add("--path=" + ctx.files.test_sources[0].dirname)

        # CVL's fixture models use the local common definitions, exactly as
        # cvl/Makefile does; production imports contain a different schema set.
        args.add("--path=" + ctx.files._test_common[0].dirname)
    else:
        args.add("--path=" + ctx.file.models.path + "/sonic")
    args.add("--path=" + ctx.file.models.path + "/common")
    args.add("--out-dir", output.path)
    ctx.actions.run(
        executable = ctx.executable._generator,
        arguments = [args],
        inputs = [ctx.file.models] + ctx.files.test_sources + ctx.files._test_common,
        outputs = [output],
        tools = [ctx.attr._generator[DefaultInfo].files_to_run],
        env = {"PYTHONHASHSEED": "0", "PYTHONNOUSERSITE": "1"},
        execution_requirements = {"block-network": "1"},
        mnemonic = "MgmtCvlSchema",
    )
    return [DefaultInfo(files = depset([output]))]

cvl_schema = rule(
    implementation = _schema_impl,
    attrs = {
        "models": attr.label(allow_single_file = True, mandatory = True),
        "test_sources": attr.label_list(allow_files = [".yang"]),
        "_test_common": attr.label_list(default = ["//models/yang:local_sonic_common"], allow_files = [".yang"]),
        "_generator": attr.label(default = "//cvl/tools:generate_yin", executable = True, cfg = "exec"),
    },
)

def _ocbinds_impl(ctx):
    output = ctx.actions.declare_file("ocbinds.go")
    args = ctx.actions.args()
    args.add("--generator", ctx.executable.generator)
    args.add("--models", ctx.file.models.path)
    args.add("--output", output.path)
    ctx.actions.run(
        executable = ctx.executable._driver,
        arguments = [args],
        inputs = [ctx.file.models],
        outputs = [output],
        tools = [ctx.attr._driver[DefaultInfo].files_to_run, ctx.attr.generator[DefaultInfo].files_to_run],
        env = {"PYTHONHASHSEED": "0"},
        execution_requirements = {"block-network": "1"},
        mnemonic = "MgmtGoBindings",
    )
    return [DefaultInfo(files = depset([output]))]

ocbinds = rule(
    implementation = _ocbinds_impl,
    attrs = {
        "models": attr.label(allow_single_file = True, mandatory = True),
        "generator": attr.label(executable = True, cfg = "exec", mandatory = True),
        "_driver": attr.label(default = "//tools/bazel:generate_ocbinds", executable = True, cfg = "exec"),
    },
)

def _runtime_tree_impl(ctx):
    output = ctx.actions.declare_directory(ctx.label.name)
    args = ctx.actions.args()
    args.add("--models", ctx.file.models.path)
    args.add("--schema", ctx.file.schema.path)
    args.add("--models-list", ctx.file.models_list)
    args.add("--version", ctx.file.version)
    args.add("--config", ctx.file.config)
    args.add("--output", output.path)
    args.add_all(ctx.files.test_sources, before_each = "--test-model")
    ctx.actions.run(
        executable = ctx.executable._stager,
        arguments = [args],
        inputs = [ctx.file.models, ctx.file.schema, ctx.file.models_list, ctx.file.version, ctx.file.config] + ctx.files.test_sources,
        outputs = [output],
        tools = [ctx.attr._stager[DefaultInfo].files_to_run],
        mnemonic = "MgmtRuntimeLayout",
        execution_requirements = {"block-network": "1"},
    )
    return [DefaultInfo(files = depset([output]))]

runtime_tree = rule(
    implementation = _runtime_tree_impl,
    attrs = {
        "models": attr.label(allow_single_file = True, mandatory = True),
        "schema": attr.label(allow_single_file = True, mandatory = True),
        "models_list": attr.label(allow_single_file = True, mandatory = True),
        "test_sources": attr.label_list(allow_files = [".yang"]),
        "version": attr.label(allow_single_file = True, mandatory = True),
        "config": attr.label(allow_single_file = True, mandatory = True),
        "_stager": attr.label(default = "//tools/bazel:stage_runtime", executable = True, cfg = "exec"),
    },
)

def _model_docs_impl(ctx):
    outputs = []
    for prefix in ["", "sonic_"]:
        stem = prefix + "allyangs"
        files = [ctx.actions.declare_file(ctx.label.name + "/" + stem + suffix) for suffix in [".tree", "_tree.html"]]
        args = ctx.actions.args()
        args.add("--text", files[0])
        args.add("--html", files[1])
        args.add("-p", ctx.file.models.path + ("/sonic" if prefix else ""))
        args.add("-c", ctx.file.models.path + "/common")
        args.add("--annotpath", ctx.file.models.path + "/annotations")
        ctx.actions.run(
            executable = ctx.executable._tree,
            arguments = [args],
            inputs = [ctx.file.models],
            tools = [ctx.attr._tree[DefaultInfo].files_to_run],
            outputs = files,
            env = {"PYTHONHASHSEED": "0", "PYTHONNOUSERSITE": "1"},
            execution_requirements = {"block-network": "1"},
            mnemonic = "MgmtYangDocs",
        )
        outputs.extend(files)
    return [DefaultInfo(files = depset(outputs))]

model_docs = rule(
    implementation = _model_docs_impl,
    attrs = {
        "models": attr.label(allow_single_file = True, mandatory = True),
        "_tree": attr.label(default = "//tools/pyang:tree_tool", executable = True, cfg = "exec"),
    },
)

def _codegen_tree_impl(ctx):
    output = ctx.actions.declare_directory(ctx.label.name)
    args = ctx.actions.args()
    args.add("--models", ctx.file.models.path)
    args.add("--bindings", ctx.file.bindings)
    args.add_all(ctx.files.docs, before_each = "--doc")
    args.add("--output", output.path)
    ctx.actions.run(
        executable = ctx.executable._stager,
        arguments = [args],
        inputs = [ctx.file.models, ctx.file.bindings] + ctx.files.docs,
        tools = [ctx.attr._stager[DefaultInfo].files_to_run],
        outputs = [output],
        execution_requirements = {"block-network": "1"},
        mnemonic = "MgmtCodegenLayout",
    )
    return [DefaultInfo(files = depset([output])), OutputGroupInfo(generated_code = depset([ctx.file.bindings, ctx.file.models]))]

codegen_tree = rule(
    implementation = _codegen_tree_impl,
    attrs = {
        "models": attr.label(allow_single_file = True, mandatory = True),
        "bindings": attr.label(allow_single_file = True, mandatory = True),
        "docs": attr.label_list(allow_files = True),
        "_stager": attr.label(default = "//tools/bazel:stage_codegen", executable = True, cfg = "exec"),
    },
)
