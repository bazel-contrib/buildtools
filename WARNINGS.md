# Buildifier warnings

Warning categories supported by buildifier's linter:

  * [`allowed-symbol-load-locations`](#allowed-symbol-load-locations)
  * [`attr-applicable_licenses`](#attr-applicable_licenses)
  * [`attr-cfg`](#attr-cfg)
  * [`attr-license`](#attr-license)
  * [`attr-licenses`](#attr-licenses)
  * [`attr-non-empty`](#attr-non-empty)
  * [`attr-output-default`](#attr-output-default)
  * [`attr-package-metadata`](#attr-package-metadata)
  * [`attr-single-file`](#attr-single-file)
  * [`build-args-kwargs`](#build-args-kwargs)
  * [`bzl-visibility`](#bzl-visibility)
  * [`canonical-repository`](#canonical-repository)
  * [`confusing-name`](#confusing-name)
  * [`constant-glob`](#constant-glob)
  * [`ctx-actions`](#ctx-actions)
  * [`ctx-args`](#ctx-args)
  * [`deprecated-function`](#deprecated-function)
  * [`depset-items`](#depset-items)
  * [`depset-iteration`](#depset-iteration)
  * [`depset-union`](#depset-union)
  * [`dict-concatenation`](#dict-concatenation)
  * [`dict-method-named-arg`](#dict-method-named-arg)
  * [`duplicated-name`](#duplicated-name)
  * [`external-path`](#external-path)
  * [`filetype`](#filetype)
  * [`function-docstring`](#function-docstring)
  * [`function-docstring-args`](#function-docstring-args)
  * [`function-docstring-header`](#function-docstring-header)
  * [`function-docstring-return`](#function-docstring-return)
  * [`git-repository`](#git-repository)
  * [`http-archive`](#http-archive)
  * [`integer-division`](#integer-division)
  * [`keyword-positional-params`](#keyword-positional-params)
  * [`list-append`](#list-append)
  * [`load`](#load)
  * [`load-on-top`](#load-on-top)
  * [`module-docstring`](#module-docstring)
  * [`name-conventions`](#name-conventions)
  * [`native-android`](#native-android)
  * [`native-build`](#native-build)
  * [`native-cc-binary`](#native-cc-binary)
  * [`native-cc-common`](#native-cc-common)
  * [`native-cc-debug-package-info`](#native-cc-debug-package-info)
  * [`native-cc-fdo-prefetch-hints`](#native-cc-fdo-prefetch-hints)
  * [`native-cc-fdo-profile`](#native-cc-fdo-profile)
  * [`native-cc-import`](#native-cc-import)
  * [`native-cc-info`](#native-cc-info)
  * [`native-cc-library`](#native-cc-library)
  * [`native-cc-memprof-profile`](#native-cc-memprof-profile)
  * [`native-cc-objc-import`](#native-cc-objc-import)
  * [`native-cc-objc-library`](#native-cc-objc-library)
  * [`native-cc-propeller-optimize`](#native-cc-propeller-optimize)
  * [`native-cc-proto`](#native-cc-proto)
  * [`native-cc-shared-library`](#native-cc-shared-library)
  * [`native-cc-shared-library-hint-info`](#native-cc-shared-library-hint-info)
  * [`native-cc-shared-library-info`](#native-cc-shared-library-info)
  * [`native-cc-test`](#native-cc-test)
  * [`native-cc-toolchain`](#native-cc-toolchain)
  * [`native-cc-toolchain-suite`](#native-cc-toolchain-suite)
  * [`native-java-binary`](#native-java-binary)
  * [`native-java-common`](#native-java-common)
  * [`native-java-import`](#native-java-import)
  * [`native-java-info`](#native-java-info)
  * [`native-java-library`](#native-java-library)
  * [`native-java-lite-proto`](#native-java-lite-proto)
  * [`native-java-package-config`](#native-java-package-config)
  * [`native-java-plugin`](#native-java-plugin)
  * [`native-java-plugin-info`](#native-java-plugin-info)
  * [`native-java-proto`](#native-java-proto)
  * [`native-java-runtime`](#native-java-runtime)
  * [`native-java-test`](#native-java-test)
  * [`native-java-toolchain`](#native-java-toolchain)
  * [`native-package`](#native-package)
  * [`native-proto`](#native-proto)
  * [`native-proto-common`](#native-proto-common)
  * [`native-proto-info`](#native-proto-info)
  * [`native-proto-lang-toolchain`](#native-proto-lang-toolchain)
  * [`native-proto-lang-toolchain-info`](#native-proto-lang-toolchain-info)
  * [`native-py`](#native-py)
  * [`native-sh-binary`](#native-sh-binary)
  * [`native-sh-library`](#native-sh-library)
  * [`native-sh-test`](#native-sh-test)
  * [`no-effect`](#no-effect)
  * [`out-of-order-load`](#out-of-order-load)
  * [`output-group`](#output-group)
  * [`overly-nested-depset`](#overly-nested-depset)
  * [`package-name`](#package-name)
  * [`package-on-top`](#package-on-top)
  * [`positional-args`](#positional-args)
  * [`print`](#print)
  * [`provider-params`](#provider-params)
  * [`redefined-variable`](#redefined-variable)
  * [`repository-name`](#repository-name)
  * [`return-value`](#return-value)
  * [`rule-impl-return`](#rule-impl-return)
  * [`same-origin-load`](#same-origin-load)
  * [`skylark-comment`](#skylark-comment)
  * [`skylark-docstring`](#skylark-docstring)
  * [`string-iteration`](#string-iteration)
  * [`uninitialized`](#uninitialized)
  * [`unnamed-macro`](#unnamed-macro)
  * [`unreachable`](#unreachable)
  * [`unsorted-dict-items`](#unsorted-dict-items)
  * [`unused-variable`](#unused-variable)

### <a name="suppress"></a>How to disable warnings

All warnings can be disabled / suppressed / ignored by adding a special comment `# buildifier: disable=<category_name>` to
the expression that causes the warning. Historically comments with `buildozer` instead of
`buildifier` are also supported, they are equivalent.

To disable multiple warnings, you can specify them as a comma-separated list on a single line:

```python
# buildifier: disable=warning1,warning2
```

Or place them on separate lines:

```python
# buildifier: disable=warning1
# buildifier: disable=warning2
```

#### Examples

```python
# buildifier: disable=no-effect
"""
A multiline comment as a string literal.

Docstrings don't trigger the warning if they are first statements of a file or a function.
"""

if debug:
    print("Debug information:", foo)  # buildifier: disable=print
```

--------------------------------------------------------------------------------

## <a name="allowed-symbol-load-locations"></a>Symbol must be loaded from a specific location

  * Category name: `allowed-symbol-load-locations`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=allowed-symbol-load-locations`

Warns when a symbol is loaded from a location other than the expected ones.
Expected locations are specified in the tables file:

```json
{
  "AllowedSymbolLoadLocations": {
    "genrule": [
      "//tools/bazel:genrule.bzl"
    ]
  }
}
```

--------------------------------------------------------------------------------

## <a name="attr-applicable_licenses"></a>Do not use `applicable_licenses` as an attribute name.

  * Category name: `attr-applicable_licenses`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=attr-applicable_licenses`

Using `applicable_licenses` as an attribute name may cause unexpected behavior. Its use may be prohibited in future Bazel releases.

--------------------------------------------------------------------------------

## <a name="attr-cfg"></a>`cfg = "data"` for attr definitions has no effect

  * Category name: `attr-cfg`
  * Flag in Bazel: [`--incompatible_disallow_data_transition`](https://github.com/bazelbuild/bazel/issues/6153)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=attr-cfg`

The [Configuration](https://docs.bazel.build/versions/main/skylark/rules.html#configurations)
`cfg = "data"` is deprecated and has no effect. Consider removing it.
The [Configuration](https://docs.bazel.build/versions/main/skylark/rules.html#configurations)
`cfg = "host"` is deprecated. Consider replacing it with `cfg = "exec"`.

--------------------------------------------------------------------------------

## <a name="attr-license"></a>`attr.license()` is deprecated and shouldn't be used

  * Category name: `attr-license`
  * Flag in Bazel: `--incompatible_no_attr_license`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=attr-license`

The `attr.license()` method is almost never used and being deprecated.

--------------------------------------------------------------------------------

## <a name="attr-licenses"></a>Do not use `licenses` as an attribute name.

  * Category name: `attr-licenses`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=attr-licenses`

Using licenses as an attribute name may cause unexpected behavior.

--------------------------------------------------------------------------------

## <a name="attr-non-empty"></a>`non_empty` attribute for attr definitions is deprecated

  * Category name: `attr-non-empty`
  * Flag in Bazel: `--incompatible_disable_deprecated_attr_params`
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=attr-non-empty`

The `non_empty` [attribute](https://docs.bazel.build/versions/main/skylark/lib/attr.html)
for attr definitions is deprecated, please use `allow_empty` with an opposite value instead.

--------------------------------------------------------------------------------

## <a name="attr-output-default"></a>The `default` parameter for `attr.output()`is deprecated

  * Category name: `attr-output-default`
  * Flag in Bazel: [`--incompatible_no_output_attr_default`](https://github.com/bazelbuild/bazel/issues/7950)
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=attr-output-default`

The `default` parameter of `attr.output()` is bug-prone, as two targets of the same rule would be
unable to exist in the same package under default behavior. Use Starlark macros to specify defaults
for these attributes instead.

--------------------------------------------------------------------------------

## <a name="attr-package-metadata"></a>Do not use `package_metadata` as an attribute name.

  * Category name: `attr-package-metadata`
  * Automatic fix: no
  * Not supported by the latest version of Buildifier
  * [Suppress the warning](#suppress): `# buildifier: disable=attr-package-metadata`

Using `package_metadata` as an attribute name may cause unexpected behavior. Its use may be prohibited in future Bazel releases.

--------------------------------------------------------------------------------

## <a name="attr-single-file"></a>`single_file` is deprecated

  * Category name: `attr-single-file`
  * Flag in Bazel: `--incompatible_disable_deprecated_attr_params`
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=attr-single-file`

The `single_file` [attribute](https://docs.bazel.build/versions/main/skylark/lib/attr.html)
is deprecated, please use `allow_single_file` instead.

--------------------------------------------------------------------------------

## <a name="build-args-kwargs"></a>`*args` and `**kwargs` are not allowed in BUILD files

  * Category name: `build-args-kwargs`
  * Flag in Bazel: `--incompatible_no_kwargs_in_build_files`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=build-args-kwargs`

Having `*args` or `**kwargs` makes BUILD files hard to read and manipulate. The list of
arguments should be explicit.

--------------------------------------------------------------------------------

## <a name="bzl-visibility"></a>Module shouldn't be used directly

  * Category name: `bzl-visibility`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=bzl-visibility`

If a directory `foo` contains a subdirectory `internal` or `private`, only files located under `foo`
can access it.

For example, `dir/rules_mockascript/private/foo.bzl` can be loaded from
`dir/rules_mockascript/private/bar.bzl` or `dir/rules_mockascript/sub/public.bzl`,
but not from `dir/other_rule/file.bzl`.

--------------------------------------------------------------------------------

## <a name="canonical-repository"></a>String contains `@@` which indicates a canonical repository name reference that should be avoided

  * Category name: `canonical-repository`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=canonical-repository`

Using canonical repository names (with `@@` prefix) makes BUILD files fragile
to repository mapping changes and external dependency updates. Canonical names
are internal implementation details that can change between Bazel versions
or when external dependencies are updated.

Instead of using canonical names like:

```python
load("@@rules_go//go:def.bzl", "go_library")
deps = ["@@protobuf~5.27.0//src:message"]
```

Use apparent names with single `@`:

```python
load("@rules_go//go:def.bzl", "go_library")
deps = ["@protobuf//src:message"]
```

This makes your BUILD files more maintainable and resilient to changes.

--------------------------------------------------------------------------------

## <a name="confusing-name"></a>Never use `l`, `I`, or `O` as names

  * Category name: `confusing-name`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=confusing-name`

The names `l`, `I`, or `O` can be easily confused with `I`, `l`, or `0` correspondingly.

--------------------------------------------------------------------------------

## <a name="constant-glob"></a>Glob pattern has no wildcard ('*')

  * Category name: `constant-glob`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=constant-glob`

[Glob function](https://docs.bazel.build/versions/main/be/functions.html#glob)
is used to get a list of files from the depot. The patterns (the first argument)
typically include a wildcard (* character). A pattern without a wildcard is
often useless and sometimes harmful.

To fix the warning, move the string out of the glob:

```diff
- glob(["*.cc", "test.cpp"])
+ glob(["*.cc"]) + ["test.cpp"]
```

**There’s one important difference**: before the change, Bazel would silently
ignore test.cpp if file is missing; after the change, Bazel will throw an error
if file is missing.

If `test.cpp` doesn’t exist, the fix becomes:

```diff
- glob(["*.cc", "test.cpp"])
+ glob(["*.cc"])
```

which improves maintenance and readability.

If no pattern has a wildcard, just remove the glob. It will also improve build
performance (glob can be relatively slow):

```diff
- glob(["test.cpp"])
+ ["test.cpp"]
```

--------------------------------------------------------------------------------

## <a name="ctx-actions"></a>`ctx.{action_name}` is deprecated

  * Category name: `ctx-actions`
  * Flag in Bazel: [`--incompatible_new_actions_api`](https://github.com/bazelbuild/bazel/issues/5825)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=ctx-actions`

The following [actions](https://docs.bazel.build/versions/main/skylark/lib/actions.html)
are deprecated, please use the new API:

  * [`ctx.new_file`](https://docs.bazel.build/versions/main/skylark/lib/ctx.html#new_file) → [`ctx.actions.declare_file`](https://docs.bazel.build/versions/main/skylark/lib/actions.html#declare_file)
  * `ctx.experimental_new_directory` → [`ctx.actions.declare_directory`](https://docs.bazel.build/versions/main/skylark/lib/actions.html#declare_directory)
  * [`ctx.file_action`](https://docs.bazel.build/versions/main/skylark/lib/ctx.html#file_action) → [`ctx.actions.write`](https://docs.bazel.build/versions/main/skylark/lib/actions.html#write)
  * [`ctx.action(command = "...")`](https://docs.bazel.build/versions/main/skylark/lib/ctx.html#action) → [`ctx.actions.run_shell`](https://docs.bazel.build/versions/main/skylark/lib/actions.html#run_shell)
  * [`ctx.action(executable = "...")`](https://docs.bazel.build/versions/main/skylark/lib/ctx.html#action) → [`ctx.actions.run`](https://docs.bazel.build/versions/main/skylark/lib/actions.html#run)
  * [`ctx.empty_action`](https://docs.bazel.build/versions/main/skylark/lib/ctx.html#empty_action) → [`ctx.actions.do_nothing`](https://docs.bazel.build/versions/main/skylark/lib/actions.html#do_nothing)
  * [`ctx.template_action`](https://docs.bazel.build/versions/main/skylark/lib/ctx.html#template_action) → [`ctx.actions.expand_template`](https://docs.bazel.build/versions/main/skylark/lib/actions.html#expand_template)

--------------------------------------------------------------------------------

## <a name="ctx-args"></a>`ctx.actions.args().add()` for multiple arguments is deprecated

  * Category name: `ctx-args`
  * Flag in Bazel: [`--incompatible_disallow_old_style_args_add`](https://github.com/bazelbuild/bazel/issues/5822)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=ctx-args`

It's deprecated to use the [`add`](https://docs.bazel.build/versions/main/skylark/lib/Args.html#add)
method of `ctx.actions.args()` to add a list (or a depset) of variables. Please use either
[`add_all`](https://docs.bazel.build/versions/main/skylark/lib/Args.html#add_all) or
[`add_joined`](https://docs.bazel.build/versions/main/skylark/lib/Args.html#add_joined),
depending on the desired behavior.

--------------------------------------------------------------------------------

## <a name="deprecated-function"></a>The function is deprecated

  * Category name: `deprecated-function`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=deprecated-function`

The function defined in another .bzl file has a docstring stating that it's deprecated, i.e. it
contains a `Deprecated:` section. The convention for function docstrings is described by
the [`function-docstring`](#function-docstring) warning.

--------------------------------------------------------------------------------

## <a name="depset-items"></a>Depset's "items" parameter is deprecated

  * Category name: `depset-items`
  * Flag in Bazel: [`--incompatible_disable_depset_items`](https://github.com/bazelbuild/bazel/issues/9017)
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=depset-items`

The `items` parameter for [`depset`](https://docs.bazel.build/versions/main/skylark/lib/globals.html#depset)
is deprecated. In its old form it's either a list of direct elements to be
added (use the `direct` or unnamed first parameter instead) or a depset that
becomes a transitive element of the new depset (use the `transitive` parameter
instead).

--------------------------------------------------------------------------------

## <a name="depset-iteration"></a>Depset iteration is deprecated

  * Category name: `depset-iteration`
  * Flag in Bazel: [`--incompatible_depset_is_not_iterable`](https://github.com/bazelbuild/bazel/issues/5816)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=depset-iteration`

Depsets are complex structures, iterations over them and lookups require flattening them to
a list which may be a heavy operation. To make it more obvious it's now required to call
the `.to_list()` method on them in order to be able to iterate their items:

```python
deps = depset()
[x.path for x in deps]  # deprecated
[x.path for x in deps.to_list()]  # recommended
```

--------------------------------------------------------------------------------

## <a name="depset-union"></a>Depsets should be joined using the depset constructor

  * Category name: `depset-union`
  * Flag in Bazel: [`--incompatible_depset_union`](https://github.com/bazelbuild/bazel/issues/5817)
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=depset-union`

The following ways to merge two depsets are deprecated:

```python
depset1 + depset2
depset1 | depset2
depset1.union(depset2)
```

Please use the [depset](https://docs.bazel.build/versions/main/skylark/lib/depset.html) constructor
instead:

```python
depset(transitive = [depset1, depset2])
```

When fixing this issue, make sure you
[understand depsets](https://docs.bazel.build/versions/main/skylark/depsets.html)
and try to
[reduce the number of calls to depset](https://docs.bazel.build/versions/main/skylark/performance.html#reduce-the-number-of-calls-to-depset).
See this [explanation](https://github.com/bazelbuild/bazel/issues/5817#issuecomment-496910826) for more detail.

--------------------------------------------------------------------------------

## <a name="dict-concatenation"></a>Dictionary concatenation is deprecated

  * Category name: `dict-concatenation`
  * Flag in Bazel: [`--incompatible_disallow_dict_plus`](https://github.com/bazelbuild/bazel/issues/6461)
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=dict-concatenation`

The `+` operator to concatenate dicts is deprecated. The operator used to create a new dict and
copy the data to it. There are several ways to avoid it, for example, instead of `d = d1 + d2 + d3`
you can use one of the following:

  * Use [Skylib](https://github.com/bazelbuild/bazel-skylib):

```python
load("@bazel_skylib//lib:dicts.bzl", "dicts")

d = dicts.add(d1, d2, d3)
```

  * The same if you don't want to use Skylib:

```python
d = dict(d1.items() + d2.items() + d3.items())
```

  * The same in several steps:

```python
d = dict(d1)  # If you don't want `d1` to be mutated
d.update(d2)
d.update(d3)
```

--------------------------------------------------------------------------------

## <a name="dict-method-named-arg"></a>Dict methods do not have a named argument `default`

  * Category name: `dict-method-named-arg`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=dict-method-named-arg`

Dict methods `get`, `pop` and `setdefault` do not accept a named argument
called `default`. Due to a bug, Bazel currently accepts that named argument.
It is better to use a positional argument instead:

```diff
- mydict.get(5, default = 0)
+ mydict.get(5, 0)
```

--------------------------------------------------------------------------------

## <a name="duplicated-name"></a>A rule with name `foo` was already found on line

  * Category name: `duplicated-name`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=duplicated-name`

Each label in Bazel has a unique name, and Bazel doesn’t allow two rules to have
the same name. With macros, this may be accepted by Bazel (if each macro
generates different rules):

```python
my_first_macro(name = "foo")
my_other_macro(name = "foo")
```

Although the build may work, this code can be very confusing. It can confuse
users reading a BUILD file (if they look for the rule “foo”, they may read see
only one of the macros). It will also confuse tools that edit BUILD files.

To fix the issue just change the name attribute of one rule/macro.

--------------------------------------------------------------------------------

## <a name="external-path"></a>String contains `/external/` which may indicate a dependency on external repositories that could be fragile

  * Category name: `external-path`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=external-path`

Using `/external/` paths may indicate a hard-coded dependency on external
repository locations, which can be fragile and break when external dependencies
are updated or reorganized.

Paths containing `/external/` (without a leading `//`) may indicate:
- Direct file system paths to external repositories
- Dependencies that bypass Bazel's dependency management
- Code that relies on Bazel's internal directory structure

Examples that trigger this warning:

```python
copts = ["-iquote $(BINDIR)/external/some_repo/include"]
args = ["./external/repo/path/data.txt"]
```

Instead, use proper Bazel labels:

```python
copts = ["-iquote $(execpath @some_repo//include:foo.h)/.."]
args = ["$(execpath @repo//path:data.txt)"]
```

Note: This warning does not apply to main repository paths like `//external/...`
which are legitimate Bazel labels.

--------------------------------------------------------------------------------

## <a name="filetype"></a>The `FileType` function is deprecated

  * Category name: `filetype`
  * Flag in Bazel: [`--incompatible_disallow_filetype`](https://github.com/bazelbuild/bazel/issues/5831)
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=filetype`

The function `FileType` is deprecated. Instead of using it as an argument to the
[`rule` function](https://docs.bazel.build/versions/main/skylark/lib/globals.html#rule)
just use a list of strings.

--------------------------------------------------------------------------------

## <a name="function-docstring"></a><a name="function-docstring-header"></a><a name="function-docstring-args"></a><a name="function-docstring-return"></a>Function docstring

  * Category names:
    * `function-docstring`
    * `function-docstring-header`
    * `function-docstring-args`
    * `function-docstring-return`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=function-docstring`, `# buildifier: disable=function-docstring-header`, `# buildifier: disable=function-docstring-args`, `# buildifier: disable=function-docstring-return`

Public functions should have docstrings describing functions and their signatures.
A docstring is a string literal (not a comment) which should be the first statement
of a function (it may follow comment lines). Function docstrings are expected to be
formatted in the following way:

```python
"""One-line summary: must be followed and may be preceded by a blank line.

Optional additional description like this.

If it's a function docstring and the function has more than one argument, the docstring has
to document these parameters as follows:

Args:
  parameter1: description of the first parameter. Each parameter line
    should be indented by one, preferably two, spaces (as here).
  parameter2: description of the second
    parameter that spans two lines. Each additional line should have a
    hanging indentation of at least one, preferably two, additional spaces (as here).
  another_parameter (unused, mutable): a parameter may be followed
    by additional attributes in parentheses

Returns:
  Description of the return value.
  Should be indented by at least one, preferably two spaces (as here)
  Can span multiple lines.

Deprecated:
  Optional, description of why the function is deprecated and what should be used instead.
"""
```

Docstrings are required for all public functions with at least 5 statements. If a docstring exists
it should start with a one-line summary line followed by an empty line. If a docstring is required
or it describes some arguments, it should describe all of them. If a docstring is required and
the function returns a value, it should be described.

--------------------------------------------------------------------------------

## <a name="git-repository"></a>Function `git_repository` is not global anymore

  * Category name: `git-repository`
  * Flag in Bazel: [`--incompatible_remove_native_git_repository`](https://github.com/bazelbuild/bazel/issues/6569)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=git-repository`

Native `git_repository` and `new_git_repository` functions are removed.
Please use the Starlark version instead:

```python
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
```

--------------------------------------------------------------------------------

## <a name="http-archive"></a>Function `http_archive` is not global anymore

  * Category name: `http-archive`
  * Flag in Bazel: [`--incompatible_remove_native_http_archive`](https://github.com/bazelbuild/bazel/issues/6570)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=http-archive`

Native `http_archive` function is removed.
Please use the Starlark version instead:

```python
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
```

--------------------------------------------------------------------------------

## <a name="integer-division"></a>The `/` operator for integer division is deprecated

  * Category name: `integer-division`
  * Flag in Bazel: [`--incompatible_disallow_slash_operator`](https://github.com/bazelbuild/bazel/issues/5823)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=integer-division`

The `/` operator is deprecated in favor of `//`, please use the latter for
integer division:

```python
a = b // c
d //= e
```

--------------------------------------------------------------------------------

## <a name="keyword-positional-params"></a>Keyword parameter should be positional

  * Category name: `keyword-positional-params`
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=keyword-positional-params`

Some parameters for builtin functions in Starlark are keyword for legacy reasons;
their names are not meaningful (e.g. `x`). Making them positional-only will improve
the readability.

--------------------------------------------------------------------------------

## <a name="list-append"></a>Prefer using `.append()` to adding a single element list

  * Category name: `list-append`
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=list-append`

Transforming `x += [expr]` to `x.append(expr)` avoids a list allocation.
NOTE: .append() does not work if the target is a select, hence this warning can safely
be ignored or suppressed.

--------------------------------------------------------------------------------

## <a name="load"></a>Loaded symbol is unused

  * Category name: `load`
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=load`

### Background

[load](https://docs.bazel.build/versions/main/skylark/concepts.html#loading-an-extension)
is used to import definitions in a BUILD file. If the definition is not used in
the file, the load can be safely removed. If a symbol is loaded two times, you
will get a warning on the second occurrence.

### How to fix it

Delete the line. When load is used to import multiple symbols, you can remove
the unused symbols from the list. To fix your BUILD files automatically, try
this command:

```bash
$ buildozer 'fix unusedLoads' path/to/BUILD
```

If you want to keep the load, you can disable the warning by adding a comment
`# @unused`.

--------------------------------------------------------------------------------

## <a name="load-on-top"></a>Load statements should be at the top of the file

  * Category name: `load-on-top`
  * Flag in Bazel: [`--incompatible_bzl_disallow_load_after_statement`](https://github.com/bazelbuild/bazel/issues/5815)
  * Automatic fix: yes
  * Not supported by the latest version of Buildifier
  * [Suppress the warning](#suppress): `# buildifier: disable=load-on-top`

Obsolete; the warning has been implemented in the formatter and the fix is now automatically applied to all files except `WORKSPACE` files (unless suppressed).

Load statements should be first statements (with the exception of `WORKSPACE` files),
they can follow only comments and docstrings.

--------------------------------------------------------------------------------

## <a name="module-docstring"></a>The file has no module docstring

  * Category name: `module-docstring`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=module-docstring`

`.bzl` files should have docstrings on top of them. A docstring is a string literal
(not a comment) which should be the first statement of the file (it may follow
comment lines). For example:

```python
"""
This module contains build rules for my project.
"""

...
```

--------------------------------------------------------------------------------

## <a name="name-conventions"></a>Name conventions

  * Category name: `name-conventions`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=name-conventions`

By convention, all variables should be lower_snake_case, constant should be
UPPER_SNAKE_CASE, and providers should be UpperCamelCase ending with `Info`.

--------------------------------------------------------------------------------

## <a name="native-android"></a>All Android build rules should be loaded from Starlark

  * Category name: `native-android`
  * Flag in Bazel: [`--incompatible_disable_native_android_rules`](https://github.com/bazelbuild/bazel/issues/8391)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-android`

The Android build rules should be loaded from Starlark.

Update: the plans for disabling native rules
[have been postponed](https://groups.google.com/g/bazel-discuss/c/XNvpWcge4AE/m/aJ-aQzszAwAJ),
at the moment it's not required to load Starlark rules.

--------------------------------------------------------------------------------

## <a name="native-build"></a>The `native` module shouldn't be used in BUILD files

  * Category name: `native-build`
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-build`

There's no need in using `native.` in BUILD files, its members are available
as global symbols there.

--------------------------------------------------------------------------------

## <a name="native-cc-binary"></a><a name="native-cc-common"></a><a name="native-cc-debug-package-info"></a><a name="native-cc-fdo-prefetch-hints"></a><a name="native-cc-fdo-profile"></a><a name="native-cc-import"></a><a name="native-cc-info"></a><a name="native-cc-library"></a><a name="native-cc-memprof-profile"></a><a name="native-cc-objc-import"></a><a name="native-cc-objc-library"></a><a name="native-cc-propeller-optimize"></a><a name="native-cc-shared-library"></a><a name="native-cc-shared-library-hint-info"></a><a name="native-cc-shared-library-info"></a><a name="native-cc-test"></a><a name="native-cc-toolchain"></a><a name="native-cc-toolchain-suite"></a>All C++ build rules should be loaded from Starlark

  * Category names:
    * `native-cc-binary`
    * `native-cc-common`
    * `native-cc-debug-package-info`
    * `native-cc-fdo-prefetch-hints`
    * `native-cc-fdo-profile`
    * `native-cc-import`
    * `native-cc-info`
    * `native-cc-library`
    * `native-cc-memprof-profile`
    * `native-cc-objc-import`
    * `native-cc-objc-library`
    * `native-cc-propeller-optimize`
    * `native-cc-shared-library`
    * `native-cc-shared-library-hint-info`
    * `native-cc-shared-library-info`
    * `native-cc-test`
    * `native-cc-toolchain`
    * `native-cc-toolchain-suite`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-cc-binary`, `# buildifier: disable=native-cc-common`, `# buildifier: disable=native-cc-debug-package-info`, `# buildifier: disable=native-cc-fdo-prefetch-hints`, `# buildifier: disable=native-cc-fdo-profile`, `# buildifier: disable=native-cc-import`, `# buildifier: disable=native-cc-info`, `# buildifier: disable=native-cc-library`, `# buildifier: disable=native-cc-memprof-profile`, `# buildifier: disable=native-cc-objc-import`, `# buildifier: disable=native-cc-objc-library`, `# buildifier: disable=native-cc-propeller-optimize`, `# buildifier: disable=native-cc-shared-library`, `# buildifier: disable=native-cc-shared-library-hint-info`, `# buildifier: disable=native-cc-shared-library-info`, `# buildifier: disable=native-cc-test`, `# buildifier: disable=native-cc-toolchain`, `# buildifier: disable=native-cc-toolchain-suite`

The C++ build rules should be loaded from @rules_cc.

--------------------------------------------------------------------------------

## <a name="native-cc-proto"></a>cc_proto_library rule should be loaded from Starlark

  * Category name: `native-cc-proto`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-cc-proto`

The cc_proto_library rule should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-binary"></a>All Java build rules should be loaded from Starlark

  * Category name: `native-java-binary`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-binary`

The Java build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-common"></a>All Java build rules should be loaded from Starlark

  * Category name: `native-java-common`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-common`

The Java build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-import"></a>All Java build rules should be loaded from Starlark

  * Category name: `native-java-import`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-import`

The Java build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-info"></a>All Java build rules should be loaded from Starlark

  * Category name: `native-java-info`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-info`

The Java build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-library"></a>All Java build rules should be loaded from Starlark

  * Category name: `native-java-library`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-library`

The Java build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-lite-proto"></a>java_lite_proto_library rule should be loaded from Starlark

  * Category name: `native-java-lite-proto`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-lite-proto`

The java_lite_proto_library rule should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-package-config"></a>All Java build rules should be loaded from Starlark

  * Category name: `native-java-package-config`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-package-config`

The Java build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-plugin"></a>All Java build rules should be loaded from Starlark

  * Category name: `native-java-plugin`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-plugin`

The Java build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-plugin-info"></a>All Java build rules should be loaded from Starlark

  * Category name: `native-java-plugin-info`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-plugin-info`

The Java build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-proto"></a>java_proto_library rule should be loaded from Starlark

  * Category name: `native-java-proto`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-proto`

The java_proto_library rule should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-runtime"></a>All Java build rules should be loaded from Starlark

  * Category name: `native-java-runtime`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-runtime`

The Java build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-test"></a>All Java build rules should be loaded from Starlark

  * Category name: `native-java-test`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-test`

The Java build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-java-toolchain"></a>All Java build rules should be loaded from Starlark

  * Category name: `native-java-toolchain`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-java-toolchain`

The Java build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-package"></a>`native.package()` shouldn't be used in .bzl files

  * Category name: `native-package`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=native-package`

It's discouraged and will be disallowed to use `native.package()` in .bzl files.
It can silently modify the semantics of a BUILD file and makes it hard to maintain.

--------------------------------------------------------------------------------

## <a name="native-proto"></a>proto_library rule should be loaded from Starlark

  * Category name: `native-proto`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-proto`

The proto_library rule should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-proto-common"></a>proto_common module should be loaded from Starlark

  * Category name: `native-proto-common`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-proto-common`

The proto_common module should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-proto-info"></a>ProtoInfo provider should be loaded from Starlark

  * Category name: `native-proto-info`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-proto-info`

The ProtoInfo provider should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-proto-lang-toolchain"></a>proto_lang_toolchain rule should be loaded from Starlark

  * Category name: `native-proto-lang-toolchain`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-proto-lang-toolchain`

The proto_lang_toolchain rule should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-proto-lang-toolchain-info"></a>ProtoLangToolchainInfo provider should be loaded from Starlark

  * Category name: `native-proto-lang-toolchain-info`
  * Flag in Bazel: [`--incompatible_autoload_externally`](https://github.com/bazelbuild/bazel/issues/23043)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-proto-lang-toolchain-info`

The ProtoLangToolchainInfo provider should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-py"></a>All Python build rules should be loaded from Starlark

  * Category name: `native-py`
  * Flag in Bazel: [`--incompatible_load_python_rules_from_bzl`](https://github.com/bazelbuild/bazel/issues/9006)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-py`

The Python build rules should be loaded from Starlark.

Update: the plans for disabling native rules
[have been postponed](https://groups.google.com/g/bazel-discuss/c/XNvpWcge4AE/m/aJ-aQzszAwAJ),
at the moment it's not required to load Starlark rules.

--------------------------------------------------------------------------------

## <a name="native-sh-binary"></a>sh_binary build rules should be loaded from Starlark

  * Category name: `native-sh-binary`
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-sh-binary`

The sh_binary build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-sh-library"></a>sh_library build rules should be loaded from Starlark

  * Category name: `native-sh-library`
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-sh-library`

The sh_library build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="native-sh-test"></a>sh_test build rules should be loaded from Starlark

  * Category name: `native-sh-test`
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=native-sh-test`

The sh_test build rules should be loaded from Starlark.

--------------------------------------------------------------------------------

## <a name="no-effect"></a>Expression result is not used

  * Category name: `no-effect`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=no-effect`

The statement has no effect. Consider removing it or storing its result in a variable.

--------------------------------------------------------------------------------

## <a name="out-of-order-load"></a>Load statements should be ordered by their labels

  * Category name: `out-of-order-load`
  * Automatic fix: yes
  * Not supported by the latest version of Buildifier
  * [Suppress the warning](#suppress): `# buildifier: disable=out-of-order-load`

Obsolete; the warning has been implemented in the formatter and the fix is now automatically applied to all files (unless suppressed).

Load statements should be ordered by their first argument - extension file label.
This makes it easier to developers to locate loads of interest and reduces chances
for conflicts when performing large-scale automated refactoring.

When applying automated fixes, it's highly recommended to also use
[`load-on-top`](#load-on-top) fixes, since otherwise the relative order
of a symbol load and its usage can change resulting in runtime error.

--------------------------------------------------------------------------------

## <a name="output-group"></a>`ctx.attr.dep.output_group` is deprecated

  * Category name: `output-group`
  * Flag in Bazel: [`--incompatible_no_target_output_group`](https://github.com/bazelbuild/bazel/issues/7949)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=output-group`

The `output_group` field of a target is deprecated in favor of the
[`OutputGroupInfo` provider](https://docs.bazel.build/versions/main/skylark/lib/OutputGroupInfo.html).

--------------------------------------------------------------------------------

## <a name="overly-nested-depset"></a>The depset is potentially overly nested

  * Category name: `overly-nested-depset`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=overly-nested-depset`

If a depset is iteratively chained in a for loop, e.g. the following pattern is used:

```python
for ...:
    x = depset(..., transitive = [..., x, ...])
```

this can result in an overly nested depset with a long chain of transitive elements. Such patterns
can lead to performance problems, consider refactoring the code to create a flat list of transitive
elements and call the depset constructor just once:

```python
transitive = []

for ...:
    transitive += ...

x = depset(..., transitive = transitive)
```

Or in simple cases you can use list comprehensions instead:

```python
x = depset(..., transitive = [y.deps for y in ...])
```

For more information, read Bazel documentation about
[depsets](https://docs.bazel.build/versions/main/skylark/depsets.html)
and
[reducing the number of calls to depset](https://docs.bazel.build/versions/main/skylark/performance.html#reduce-the-number-of-calls-to-depset).

--------------------------------------------------------------------------------

## <a name="package-name"></a>Global variable `PACKAGE_NAME` is deprecated

  * Category name: `package-name`
  * Flag in Bazel: [`--incompatible_package_name_is_a_function`](https://github.com/bazelbuild/bazel/issues/5827)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=package-name`

The global variable `PACKAGE_NAME` is deprecated, please use
[`native.package_name()`](https://docs.bazel.build/versions/main/skylark/lib/native.html#package_name)
instead.

--------------------------------------------------------------------------------

## <a name="package-on-top"></a>Package declaration should be at the top of the file

  * Category name: `package-on-top`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=package-on-top`

Here is a typical structure of a BUILD file:

  * `load()` statements
  * `package()`
  * calls to rules, macros

Instantiating a rule and setting the package defaults later can be very
confusing, and has been a source of bugs (tools and humans sometimes believe
package applies to everything in a BUILD file). This might become an error in
the future.

### What can be used before package()?

The linter allows the following to be before `package()`:

  * comments
  * `load()`
  * variable declarations
  * `package_group()`
  * `licenses()`

--------------------------------------------------------------------------------

## <a name="positional-args"></a>Keyword arguments should be used over positional arguments

  * Category name: `positional-args`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=positional-args`

All macro and rule calls should use keyword args over positional arguments.
Positional arguments can cause subtle errors if the order is switched or if
an argument is removed. Keyword args also greatly improve readability.
Additionally, positional arguments prevent migration from [Legacy Macros](https://bazel.build/extending/legacy-macros)
to [Symbolic Macros](https://bazel.build/extending/macros).

```diff
- my_macro("foo", "bar")
+ my_macro(name = "foo", env = "bar")
```

The linter allows the following functions to be called with positional arguments:

  * `load()`
  * `vardef()`
  * `export_files()`
  * `licenses()`
  * `print()`

--------------------------------------------------------------------------------

## <a name="print"></a>`print()` is a debug function and shouldn't be submitted

  * Category name: `print`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=print`

Using the `print()` function for warnings is discouraged: they are often spammy and
non actionable, the people who see the warning are usually not the people who can
fix the code to make the warning disappear, and the actual maintainers of the code
may never see the warning.

--------------------------------------------------------------------------------

## <a name="provider-params"></a>Calls to `provider` should specify a list of fields and a documentation

  * Category name: `provider-params`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=provider-params`

Calls to `provider` should specify a documentation string and a list of fields:

```python
ServerAddressInfo = provider(
    "The address of an HTTP server. Fields are host (string) and port (int).",
    fields = ["host", "port"]
)
```

Fields should also be documented when needed:

```python
ServerAddressInfo = provider(
    "The address of an HTTP server.",
    fields = {
        "host": "string, e.g. 'example.com'",
        "port": "int, a TCP port number",
    }
)
```

Note that specifying a list of fields is a breaking change. It is an error if a
call to the provider uses undeclared fields. If you cannot declare the list of
fields, you may explicitly set it to None (and explain why in a comment).

```python
AllInfo = provider("This provider accepts any field.", fields = None)

NoneInfo = provider("This provider cannot have fields.", fields = [])
```

See the [documentation for providers](https://docs.bazel.build/versions/main/skylark/lib/globals.html#provider).

--------------------------------------------------------------------------------

## <a name="redefined-variable"></a>Variable has already been defined

  * Category name: `redefined-variable`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=redefined-variable`

### Background

In .bzl files, redefining a global variable is already forbidden. This helps
both humans and tools reason about the code. For consistency, we want to bring
this restriction also to BUILD files.

### How to fix it

Rename one of the variables.

Note that the content of lists and dictionaries can still be modified. We will
forbid reassignment, but not every side-effect.

--------------------------------------------------------------------------------

## <a name="repository-name"></a>Global variable `REPOSITORY_NAME` is deprecated

  * Category name: `repository-name`
  * Flag in Bazel: [`--incompatible_package_name_is_a_function`](https://github.com/bazelbuild/bazel/issues/5827)
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=repository-name`

The global variable `REPOSITORY_NAME` is deprecated, please use
[`native.repository_name()`](https://docs.bazel.build/versions/main/skylark/lib/native.html#repository_name)
instead.

--------------------------------------------------------------------------------

## <a name="return-value"></a>Some but not all execution paths of a function return a value

  * Category name: `return-value`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=return-value`

Some but not all execution paths of a function return a value. Either there's
an explicit empty `return` statement, or an implicit return in the end of a
function. If it is intentional, make it explicit using `return None`. If you
know certain parts of the code cannot be reached, add the statement
`fail("unreachable")` to them.

--------------------------------------------------------------------------------

## <a name="rule-impl-return"></a>Avoid using the legacy provider syntax

  * Category name: `rule-impl-return`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=rule-impl-return`

Returning structs from rule implementation functions is
[deprecated](https://docs.bazel.build/versions/main/skylark/rules.html#migrating-from-legacy-providers),
consider using
[providers](https://docs.bazel.build/versions/main/skylark/rules.html#providers)
or lists of providers instead.

--------------------------------------------------------------------------------

## <a name="same-origin-load"></a>Same label is used for multiple loads

  * Category name: `same-origin-load`
  * Automatic fix: yes
  * Not supported by the latest version of Buildifier
  * [Suppress the warning](#suppress): `# buildifier: disable=same-origin-load`

Obsolete; the warning has been implemented in the formatter and the fix is now automatically applied to all files except `WORKSPACE` files (unless suppressed).

### Background

[load](https://docs.bazel.build/versions/main/skylark/concepts.html#loading-an-extension)
is used to import definitions in a BUILD file. If the same label is used for loading
symbols more the ones, all such loads can be merged into a single one.

### How to fix it

Merge all loads into a single one. For example,

```python
load(":f.bzl", "s1")
load(":f.bzl", "s2")
```

can be written more compactly as

```python
load(":f.bzl", "s1", "s2")
```

--------------------------------------------------------------------------------

## <a name="skylark-comment"></a><a name="skylark-docstring"></a>"Skylark" is an outdated name of the language, please use "starlark" instead

  * Category names:
    * `skylark-comment`
    * `skylark-docstring`
  * Automatic fix: yes
  * [Suppress the warning](#suppress): `# buildifier: disable=skylark-comment`, `# buildifier: disable=skylark-docstring`

The configuration language for Bazel is called "Starlark" now, the name "Skylark" is
outdated and shouldn't be used.

--------------------------------------------------------------------------------

## <a name="string-iteration"></a>String iteration is deprecated

  * Category name: `string-iteration`
  * Flag in Bazel: [`--incompatible_string_is_not_iterable`](https://github.com/bazelbuild/bazel/issues/5830)
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=string-iteration`

Iteration over strings often leads to confusion with iteration over a sequence of strings,
therefore strings won't be recognized as sequences of 1-element strings (like in Python).
Use string indexing and `len` instead:

```python
my_string = "hello world"
for i in range(len(my_string)):
    char = my_string[i]
    # do something with char
```

--------------------------------------------------------------------------------

## <a name="uninitialized"></a>Variable may not have been initialized

  * Category name: `uninitialized`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=uninitialized`

The local value can be not initialized at the time of execution. It may happen if it's
initialized in one of the if-else clauses but not in all of them, or in a for-loop which
can potentially be empty.

--------------------------------------------------------------------------------

## <a name="unnamed-macro"></a>The macro should have a keyword argument called "name"

  * Category name: `unnamed-macro`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=unnamed-macro`

By convention all macro functions should have a keyword argument called `name`
(even if they don't use it). This is important for tooling and automation.

A macro is a function that calls a rule (either directly or indirectly by calling other
macros).

If this function is a helper function that's not supposed to be used outside of its file,
please make it private (rename it so that the name starts with `_`), this will
prevent loading the function from BUILD files and suppress the warning.

--------------------------------------------------------------------------------

## <a name="unreachable"></a>The statement is unreachable

  * Category name: `unreachable`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=unreachable`

The statement is unreachable because it follows a `return`, `break`, `continue`,
or `fail()` statement.

--------------------------------------------------------------------------------

## <a name="unsorted-dict-items"></a>Dictionary items should be ordered by their keys

  * Category name: `unsorted-dict-items`
  * Automatic fix: yes
  * [Disabled by default](buildifier/README.md#linter)
  * [Suppress the warning](#suppress): `# buildifier: disable=unsorted-dict-items`

Dictionary items should be sorted lexicographically by their keys. This makes
it easier to find the item of interest and reduces chances of conflicts when
performing large-scale automated refactoring.

The order is affected by `NamePriority` dictionary passed using `-tables` or
`-add_tables` flags.

If you want to preserve the original dictionary items order, you can disable
the warning by adding a comment `# @unsorted-dict-items` to the dictionary
expression or any of its enclosing expressions (binary, if etc). For example,

```python
# @unsorted-dict-items
d = {
    "b": "bvalue",
    "a": "avalue",
}
```

will not be reported as an issue because the assignment operation that uses
the dictionary with unsorted items has a comment disabling this warning.

--------------------------------------------------------------------------------

## <a name="unused-variable"></a>Variable is unused

  * Category name: `unused-variable`
  * Automatic fix: no
  * [Suppress the warning](#suppress): `# buildifier: disable=unused-variable`

This happens when a variable or function is set but not used in the file, e.g.

```python
x = [1, 2]
```

The line can often be safely removed.

If you want to keep the variable, you can disable the warning by adding a
comment `# @unused`.

```python
x = [1, 2] # @unused

# @unused
def f(
        x,
        y,  # @unused
):
    pass
```

If an unused variable is used for partially unpacking tuples, just prefix
its name with an underscore to suppress the warning:

```python
x, _y = foo()
for _, (a, _b) in iterable:
    print(a + x)
```

The same applies for function arguments that are not used by design:

```python
def foo(a, _b, *_args):
    return bar(a)
```

If a tuple is unpacked not in a for-loop and all variables are unused,
it'll still trigger a warning, even if all variables are underscored:

```python
_a, _b = pair
_unused = 3
```
