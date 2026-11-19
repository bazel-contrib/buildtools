"""
The module defines buildifier as a Bazel rule.
"""

load(
    "//buildifier/internal:factory.bzl",
    "buildifier_attr_factory",
    "buildifier_impl_factory",
)

def _buildifier_impl(ctx):
    return [buildifier_impl_factory(ctx)]

_buildifier = rule(
    implementation = _buildifier_impl,
    attrs = buildifier_attr_factory(),
    executable = True,
)

def buildifier(name = None, tags = [], **kwargs):
    """
    Wrapper for the _buildifier rule. Adds 'manual' to the tags and ensures windows compatibility.

    Args:
      name: The name to be used for the rule
      tags: The tags to be used for the rule
      **kwargs: all parameters for _buildifier
    """

    if "manual" not in tags:
        tags = tags + ["manual"]

    _buildifier(name = "{}.sh".format(name), tags = tags, **kwargs)

    native.sh_binary(
        name = name,
        srcs = ["{}.sh".format(name)],
        data = [":{}.sh".format(name)],
        tags = tags,
    )

def _buildifier_test_impl(ctx):
    return [buildifier_impl_factory(ctx, test_rule = True)]

_buildifier_test = rule(
    implementation = _buildifier_test_impl,
    attrs = buildifier_attr_factory(True),
    test = True,
)

def buildifier_test(**kwargs):
    """
    Wrapper for the _buildifier_test rule. Optionally disables sandboxing and caching.

    Args:
      **kwargs: all parameters for _buildifier_test
    """
    if kwargs.get("no_sandbox", False):
        tags = kwargs.get("tags", [])

        # Note: the "external" tag is a workaround for
        # https://github.com/bazelbuild/bazel/issues/15516.
        for t in ["no-sandbox", "no-cache", "external"]:
            if t not in tags:
                tags.append(t)
        kwargs["tags"] = tags
    _buildifier_test(**kwargs)
