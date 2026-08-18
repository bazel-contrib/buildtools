"""
This module contains factory methods for simple rule and implementation generation
"""

load("@bazel_skylib//lib:shell.bzl", "shell")

# buildifier: disable=print
def _value_deprecation(ctx, attr, value):
    """
    Prints a deprecation message related to a specific value for an attr.

    Args:
      ctx:      The execution context
      attr:     A String representing the attribute name
      value:    The deprecated value
    """
    print("DEPRECATION NOTICE: value '%s' for attribute '%s' will be removed in the future. Migrate '%s' to buildifier_test." % (value, attr, ctx.label))

# buildifier: disable=print
def _attr_deprecation(ctx, attr):
    """
    Prints an attribute deprecation message.

    Args:
      ctx:      The execution context
      attr:     A String representing the deprecated attribute name
    """
    print("DEPRECATION NOTICE: attribute '%s' will be removed in the future. Migrate '%s' to buildifier_test." % (attr, ctx.label))

def buildifier_attr_factory(test_rule = False):
    """
    Helper macro to generate a struct of attrs for use in a rule() definition.

    Args:
      test_rule: Whether or not to generate attrs for a test rule.

    Returns:
      A dictionary of attributes relevant to the rule
    """
    attrs = {
        "buildifier": attr.label(
            default = "//buildifier",
            cfg = "exec",
            executable = True,
        ),
        "verbose": attr.bool(
            doc = "Print verbose information on standard error",
        ),
        "exclude_patterns": attr.string_list(
            allow_empty = True,
            doc = "A list of Go-style path patterns excluded from recursive file discovery. E.g. './vendor/*' to exclude the Go vendor directory. In test rules, this attribute requires the use of the no_sandbox attribute.",
        ),
        "mode": attr.string(
            default = "fix" if not test_rule else "diff",
            doc = "Formatting mode",
            values = ["check", "diff", "print_if_changed"] + ["fix"] if not test_rule else [],
        ),
        "lint_mode": attr.string(
            doc = "Linting mode",
            values = ["", "warn"] + ["fix"] if not test_rule else [],
        ),
        "lint_warnings": attr.string_list(
            allow_empty = True,
            doc = "all prefixed with +/- if you want to include in or exclude from the default set of warnings, or none prefixed with +/- if you want to override the default set, or 'all' for all available warnings",
        ),
        "diff_command": attr.string(
            doc = "Command to use to show diff, with mode=diff. E.g. 'diff -u'",
        ),
        "multi_diff": attr.bool(
            default = False,
            doc = "Set to True if the diff command specified by the 'diff_command' can diff multiple files in the style of 'tkdiff'",
        ),
        "add_tables": attr.label(
            mandatory = False,
            doc = "path to JSON file with custom table definitions which will be merged with the built-in tables",
            allow_single_file = True,
        ),
        "_runner": attr.label(
            default = "//buildifier:runner.bash.template",
            allow_single_file = True,
        ),
        "_windows_runner": attr.label(
            default = "//buildifier:runner.bat.template",
            allow_single_file = True,
        ),
    }

    if test_rule:
        attrs.update({
            "srcs": attr.label_list(
                allow_files = [
                    ".bazel",
                    ".bzl",
                    ".oss",
                    ".sky",
                    "BUILD",
                    "WORKSPACE",
                    "WORKSPACE.bzlmod",
                ],
                doc = "A list of labels representing the starlark files to include in the test",
            ),
            "no_sandbox": attr.bool(
                default = False,
                doc = "Set to True to enable running buildifier on all files in the workspace",
            ),
            "workspace": attr.label(
                allow_single_file = True,
                doc = "Label of the WORKSPACE file; required when the no-sandbox attribute is True",
            ),
        })

    return attrs

def buildifier_impl_factory(ctx, test_rule = False):
    """
    Helper macro to generate a buildifier or buildifier_test rule.

    This macro does not depend on defaults encoded in the binary, instead
    preferring to set explicit values for each flag.

    Args:
      ctx:          The execution context.
      test_rule:    Whether or not to generate a test rule.

    Returns:
      A DefaultInfo provider
    """

    if not test_rule and ctx.attr.mode in ["check", "diff", "print_if_changed"]:
        _value_deprecation(ctx, "mode", ctx.attr.mode)

    args = [
        "-mode=%s" % ctx.attr.mode,
        "-v=%s" % str(ctx.attr.verbose).lower(),
    ]

    if ctx.attr.lint_mode:
        args.append("-lint=%s" % ctx.attr.lint_mode)

    if ctx.attr.lint_warnings:
        if not ctx.attr.lint_mode:
            fail("Cannot pass 'lint_warnings' without a 'lint_mode'")
        args.append("--warnings={}".format(",".join(ctx.attr.lint_warnings)))

    if ctx.attr.multi_diff:
        args.append("-multi_diff")
        if not test_rule:
            _attr_deprecation(ctx, "multi_diff")

    if ctx.attr.diff_command:
        args.append("-diff_command=%s" % ctx.attr.diff_command)
        if not test_rule:
            _attr_deprecation(ctx, "diff_command")

    if ctx.attr.add_tables:
        args.append("-add_tables=%s" % ctx.file.add_tables.path)

    if ctx.attr.exclude_patterns:
        if test_rule and not ctx.attr.no_sandbox:
            fail("Cannot use 'exclude_patterns' in a test rule without 'no_sandbox'")
        args.extend(["--exclude=%s" % pattern for pattern in ctx.attr.exclude_patterns])

    workspace = ""
    if test_rule and ctx.attr.no_sandbox:
        if not ctx.file.workspace:
            fail("Cannot use 'no_sandbox' without a 'workspace'")
        workspace = ctx.file.workspace.path

    # buildifier: disable=canonical-repository
    substitutions = {
        "@@ARGS@@": shell.array_literal(args),
        "@@BUILDIFIER_SHORT_PATH@@": shell.quote(ctx.executable.buildifier.short_path),
        "@@WORKSPACE@@": workspace,
    }

    if ctx.executable.buildifier.extension.lower() == "exe":
        out_file = ctx.actions.declare_file(ctx.label.name + ".bat")
        runner_template = ctx.file._windows_runner
    else:
        out_file = ctx.actions.declare_file(ctx.label.name + ".bash")
        runner_template = ctx.file._runner

    ctx.actions.expand_template(
        template = runner_template,
        output = out_file,
        substitutions = substitutions,
        is_executable = True,
    )

    runfiles = [ctx.executable.buildifier]
    if ctx.attr.add_tables:
        runfiles.append(ctx.file.add_tables)
    if test_rule:
        runfiles.extend(ctx.files.srcs)
        if ctx.attr.no_sandbox:
            runfiles.append(ctx.file.workspace)

    return DefaultInfo(
        files = depset([out_file]),
        runfiles = ctx.runfiles(files = runfiles),
        executable = out_file,
    )
