# buildozer

This Bazel module provides a pinned, prebuilt version of
[buildozer](https://github.com/bazel-contrib/buildtools/tree/main/buildozer),
a tool for manipulating Bazel BUILD files.

## Requirements

* Bazel 6.2.0 or later

## Usage

1. Add the following line to your `MODULE.bazel` file:

```starlark
bazel_dep(name = "buildozer", version = "10.0.1", dev_dependency = True)
```

2. Run buildozer via `bazel run`:

```shell
bazel run @buildozer -- ...
```

The `--` is optional if you don't need to pass arguments to buildozer that
start with a dash. You can also create an `alias` for the `@buildozer` target
in your main repository.

## Using buildozer in repository rules and module extensions

You can use buildozer during the loading phase from a repository rule or
module extension:

1. Add the following line to your `MODULE.bazel` file:

```starlark
bazel_dep(name = "buildozer", version = "10.0.1")
```

2. Resolve the buildozer binary in your implementation function:

```starlark
load("@buildozer//:buildozer.bzl", "BUILDOZER_LABEL")

def my_impl(repository_or_module_ctx):
    buildozer = repository_or_module_ctx.path(BUILDOZER_LABEL)
    repository_or_module_ctx.execute(
        [buildozer, "set foo bar", "//path/to/pkg:target"],
    )
```

Keep the `path` call at the top of the implementation function because it may
cause a [repository rule restart](https://bazel.build/extending/repo#restarting_the_implementation_function).

### Alternative usage

If you don't want to or can't load from `@buildozer`, use the module extension
directly:

```starlark
bazel_dep(name = "buildozer", version = "10.0.1")

buildozer_binary = use_extension("@buildozer//:buildozer_binary.bzl", "buildozer_binary")
use_repo(buildozer_binary, "buildozer_binary")
```

The binary is then available at the platform-independent label
`@buildozer_binary//:buildozer.exe`. The `.exe` suffix is present on every
platform so the same label also works on Windows.
