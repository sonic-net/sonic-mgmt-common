"""Run Go suites against a declared, private Redis service and writable data."""

load("@io_bazel_rules_go//go:def.bzl", "go_test")
load("@rules_python//python:py_test.bzl", "py_test")

def redis_go_test(name, schema = "//dist:schema", initialize = True, **kwargs):
    fixtures = kwargs.pop("data", [])
    tags = kwargs.pop("tags", [])
    go_test(
        name = name + "_binary",
        data = fixtures,
        tags = tags + ["manual"],
        **kwargs
    )
    py_test(
        name = name,
        srcs = ["//tools/bazel:run_go_tests.py"],
        main = "//tools/bazel:run_go_tests.py",
        size = "large",
        timeout = "long",
        tags = tags,
        args = [
            "$(location :" + name + "_binary)",
            "$(location //tools/bazel:test_runtime)",
            "$(location //dist:runtime)",
            "$(location " + schema + ")",
            "$(location //tools/test:database_config.json)",
            "$(location //cvl:test_fixture_marker)",
            "$(location //tools/test:dbinit)",
            "$(location //tools/test:platform.json)",
            "1" if initialize else "0",
        ],
        data = fixtures + [
            ":" + name + "_binary",
            "//tools/bazel:test_runtime",
            "//dist:runtime",
            schema,
            "//tools/test:database_config.json",
            "//cvl:test_fixtures",
            "//cvl:test_fixture_marker",
            "//tools/test:dbinit",
            "//tools/test:platform.json",
        ],
    )
