# Contributing to buildtools

Want to contribute? Great! First, read this page.

## Before you contribute

Before you start working on a larger contribution, you should get in touch with
us first through the issue tracker with your idea so that we can help out and
possibly guide you. Coordinating up front makes it much easier to avoid
frustration later on.

## Building and testing

The repository is built with Bazel. Use [Bazelisk](https://github.com/bazelbuild/bazelisk)
to get the version pinned in `.bazelversion`. The Go toolchain is managed by
Bazel, so you don't need a local Go installation.

Run the full test suite, which is what CI runs on Linux, macOS and Windows:

```sh
bazel test //...
```

Other commands you may need:

* After adding, removing or renaming Go files or imports, regenerate the
  `BUILD.bazel` files:

  ```sh
  bazel run //:gazelle
  ```

* After changing Go dependencies, tidy `go.mod`, `go.sum` and `MODULE.bazel`.
  CI fails if they are out of date:

  ```sh
  bazel run @io_bazel_rules_go//go -- mod tidy
  ```

* Go code must be formatted with `gofmt`. Starlark files in this repository
  must be formatted with buildifier itself, either via
  [pre-commit](https://pre-commit.com/) (`pre-commit install`) or by running:

  ```sh
  bazel run //:buildifier
  ```

### Generated files

Some generated files are checked in, and tests fail if they are stale:

* Protobuf bindings (`*.gen.pb.go`) and `lang/tables.gen.go`:

  ```sh
  ./update_generated.sh
  ```

* `WARNINGS.md`, after editing `warn/docs/warnings.textproto`:

  ```sh
  bazel build //warn/docs:warnings_docs && cp bazel-bin/warn/docs/WARNINGS.md .
  ```

* `build/parse.y.go`, after editing the grammar in `build/parse.y`:

  ```sh
  bazel build //build:parse.y.go_yacc && tail -n +3 bazel-bin/build/parse.y.baz.go > build/parse.y.go
  ```

### Adding a linter warning

Register the warning in one of the warning maps in `warn/warn.go`, add tests
next to the implementation, document it in `warn/docs/warnings.textproto` and
regenerate `WARNINGS.md` as described above. A test checks that every warning
is documented.

## Pull requests

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Pull requests are squash-merged, so
the pull request title and description become the commit message.

Before you open a pull request, please make sure that:

* The change is small and focused. Read Google's engineering practices on
  [small changes](https://google.github.io/eng-practices/review/developer/small-cls.html);
  if your change can't follow them, explain why in the description.
* The description explains what the change does and why, following Google's
  guidance on [writing good descriptions](https://google.github.io/eng-practices/review/developer/cl-descriptions.html).
* The code is covered by unit or integration tests.
* If you tested the change in ways that aren't captured by automated tests,
  the description includes the steps to reproduce that testing.
* `bazel test //...` passes and all generated files are up to date.
