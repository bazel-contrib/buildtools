load("@buildozer//:buildozer.bzl", "BUILDOZER_LABEL")

def _repo_impl(repository_ctx):
    buildozer = repository_ctx.path(BUILDOZER_LABEL)

    # Verify that the stable buildozer label results in the same path.
    buildozer2 = repository_ctx.path(Label("@buildozer_binary//:buildozer.exe"))
    if buildozer != buildozer2:
        fail("buildozer != buildozer2: {} != {}".format(buildozer, buildozer2))
    repository_ctx.file("WORKSPACE")
    repository_ctx.file("BUILD.bazel", """
load("@rules_shell//shell:sh_test.bzl", "sh_test")
sh_test(
    name = "repo_rule_test",
    srcs = ["repo_rule_test.sh"],
    env = {"SHOULD_PASS": "0"},
    visibility = ["//visibility:public"],
)
""")
    repository_ctx.file(
        "repo_rule_test.sh",
        """
#!/usr/bin/env bash
[[ $SHOULD_PASS == "1" ]]
""",
        executable = True,
    )
    repository_ctx.execute(
        [buildozer, "dict_set env SHOULD_PASS:1", "//:repo_rule_test"],
    )

_repo = repository_rule(_repo_impl)

def _repo_ext_impl(_):
    _repo(name = "repo_rule_repo")

repo_ext = module_extension(_repo_ext_impl)
