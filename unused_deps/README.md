# Unused Deps

unused_deps is a command line tool to determine any unused dependencies
in [java_library](https://docs.bazel.build/versions/main/be/java.html#java_library)
rules. targets.  It outputs `buildozer` commands to apply the suggested
prunings.

## Installation

Build a binary and put it into your $GOPATH/bin:

```bash
go install github.com/bazelbuild/buildtools/unused_deps@latest
```

## Usage

```shell
unused_deps TARGET...
```

Here, `TARGET` is a space-separated list of Bazel labels, with support for `:all` and `...`

## Options

* `-disable_symlink_safety`: By default unused_deps will not write to symlinks pointing outside of the Bazel workspace. Setting this to true will disable this behavior.

