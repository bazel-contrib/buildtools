# Buildozer

Buildozer is a command line tool to rewrite multiple
[Bazel](https://github.com/bazelbuild/bazel) BUILD files using
standard commands.

## Installation

1. Build a binary and put it into your $GOPATH/bin:

```bash
go install github.com/bazelbuild/buildtools/buildozer@latest
```

## Usage

```shell
buildozer [OPTIONS] ['command arg...'...|-f FILE] [label]...
```

Here, `label...` is a (space-separated, possibly empty) list of Bazel labels,
for example `//path/to/pkg1:rule1 relative/path/to/pkg2:rule2`. In addition to
the Bazel label syntax for specifying a package, Buildozer also allows the
package part to refer to a BUILD-like file, for example `//WORKSPACE:all` or
`toolchains/BUILD.tpl:host_toolchain`.

Buildozer commands are passed as single positional arguments, and thus have to
be quoted (or otherwise escaped). Multiple commands and multiple labels can be
passed. Buildozer will execute all commands on all targets. (So if you do not
specify at least one command and one target, nothing will happen.) Commands are
executed in order, files are processed in parallel.

When `-f FILE` is used instead of literal commands, buildozer reads commands
from `FILE`. `FILE` can be `-`, in which case commands are read from the
standard input.

The format of the command file is as follows: Empty lines and lines beginning
with `#` are ignored (including leading whitespace). Non-ignored lines consist
of `|`-separated sets of commands and labels:

```shell
command arg arg...|command arg arg...|...|label|label|...
```

(In fact, commands and labels can appear interleaved in arbitrary order.) `|`
characters in commands can be escaped like `\|`, but double null bytes
(`\x00\x00`) are not valid in command files. See below for special handling of
labels to allow reading from the standard input. When a line in a command file
uses the single label '*', then the command(s) will be applied to all elements
of the list `label...` from the command line.

### Targets

Targets look like Bazel labels, but there can be some differences in presence of
macros.

  * Use the label notation to refer to a rule: `//buildtools/buildozer:edit`
  * Use the `__pkg__` suffix to refer to the package declaration:
   `//buildtools/buildozer:__pkg__`
  * Use an asterisk to refer to all rules in a file: `//pkg:*`
  * Use `...` to refer to all descendant BUILD files in a directory: `//pkg/...:*`
  * Use percent to refer to all rules of a certain kind: `//pkg:%java_library`
  * Use percent-and-number to refer to a rule that begins at a certain line:
   `//pkg:%123`.
  * Use the special package name `-` to read the BUILD file from the standard
    input instead of from a local file in the package directory: `-:all_tests`.
    (It is presumably not useful to both use a `-` package name and use the `-f
    -` flag to read commands from the standard input.)

### Options

OPTIONS include the following options:

  * `-stdout` : write changed BUILD file to stdout
  * `-buildifier` : format output using a specific buildifier binary. If empty, use built-in formatter.
  * `-k` : apply all commands, even if there are failures
  * `-quiet` : suppress informational messages
  * `-shorten_labels` : convert added labels to short form, e.g. //foo:bar => :bar
  * `-types`: Filter the targets, keeping only those of the given types, e.g.
    `buildozer -types go_library,go_binary 'print rule' '//buildtools/buildozer:*'`
  * `-eol-comments=false`: When adding new comments, put them on a separate line.
  * `-disable_symlink_safety`: By default Buildozer will not write to symlinks pointing outside of the Bazel workspace. Setting this to true will disable this behavior.

See `buildozer -help` for the full list.

### Edit commands

Buildozer supports the following commands(`'command args'`):

  * `add <attr>[:<type>] <value(s)>`: Adds value(s) to a list attribute of a rule.
    If a value is already present in the list, it is not added. `type` specifies
    the type of values being added. See [supported types](#supported-types) for
    the available types.
  * `new_load <path> <[to=]from(s)>`: Add a load statement for the given path,
    importing the symbols. Afterwards, consider running `buildozer 'fix unusedLoads'`.
  * `replace_load <path> <[to=]from(s)>`: Similar to `new_load`, but removes
    existing load statements for the requested symbols before adding new loads.
  * `substitute_load <old_regexp> <new_template>` Replaces modules of loads which
    match `old_regexp` according to `new_template`. The regular expression must
    follow [RE2 syntax](https://github.com/google/re2/wiki/Syntax).
    `new_template` may be a simple replacement string, but it may also expand
    numbered or named groups using `$0` or `$x`.
  * `comment <attr>? <value>? <comment>`: Add a comment to a rule, an attribute,
    or a specific value in a list. Spaces in the comment should be escaped with
    backslashes.
  * `print_comment <attr>? <value>?`
  * `delete`: Delete a rule.
  * `fix <fix(es)>?`: Apply a fix.
  * `move <old_attr> <new_attr> <value(s)>`: Moves `value(s)` from the list `old_attr`
    to the list `new_attr`. The wildcard `*` matches all values.
  * `new <rule_kind> <rule_name> [(before|after) <relative_rule_name>]`: Add a
    new rule at the end of the BUILD file (before/after `<relative_rule>`). The
    identifier `__pkg__` can be used to position rules relative to package().
  * `new_if_absent <rule_kind> <rule_name> [(before|after) <relative_rule_name>]`:
    Same as `new`, but does nothing if a rule with the same name already exists.
  * `print <attr(s)>`
  * `remove <attr>`: Removes attribute `attr`. The wildcard `*` matches all
    attributes except `name`.
  * `remove <attr> <value(s)>`: Removes `value(s)` from the list `attr`. The
    wildcard `*` matches all attributes. Lists containing none of the `value(s)` are
    not modified.
  * `remove_comment <attr>? <value>?`: Removes the comment attached to the rule,
    an attribute, or a specific value in a list.
  * `remove_if_equal <attr> <value>`: Removes the attribute `attr` if its value
    is equal to `value`.
  * `rename <old_attr> <new_attr>`: Rename the `old_attr` to `new_attr` which must
    not yet exist.
  * `replace <attr> <old_value> <new_value>`: Replaces `old_value` with `new_value`
    in the list `attr`. Wildcard `*` matches all attributes. Lists not containing
    `old_value` are not modified.
  * `substitute <attr> <old_regexp> <new_template>`: Replaces strings which
    match `old_regexp` in the list `attr` according to `new_template`. Wildcard
    `*` matches all attributes. The regular expression must follow
    [RE2 syntax](https://github.com/google/re2/wiki/Syntax). `new_template` may
    be a simple replacement string, but it may also expand numbered or named
    groups using `$0` or `$x`. Lists without strings that match `old_regexp`
    are not modified.
  * `set <attr>[:<type>] <value(s)>`: Sets the value of an attribute. If the
   attribute was already present, its old value is replaced. `type` specifies
   the type of values being added. See
    [supported types](#supported-types) for the available types.
  * `set_if_absent <attr>[:<type>] <value(s)>`: Sets the value of an attribute.
    If the attribute was already present, no action is taken. `type` specifies
    the type of values being added. See [supported types](#supported-types) for
  * the available types.
  * `set kind <value>`: Set the target type to value.
  * `set_select <attr> <key_1> <value_1> <key_n> <value_n>`
  * `copy <attr> <from_rule>`: Copies the value of `attr` between rules. If it
    exists in the `to_rule`, it will be overwritten.
  * `copy_no_overwrite <attr> <from_rule>`:  Copies the value of `attr` between
    rules. If it exists in the `to_rule`, no action is taken.
  * `dict_add <attr> <(key:value)(s)>`:  Sets the value of a key for the dict
    attribute `attr`. If the key was already present, it will _not_ be overwritten.
    Colon characters can be included in key and value escaped as "\\:".
  * `dict_set <attr> <(key:value)(s)>`:  Sets the value of a key for the dict
    attribute `attr`. If the key was already present, its old value is replaced.
    Colon characters can be included in key and value escaped as "\\:".
  * `dict_remove <attr> <key(s)>`:  Deletes the key for the dict attribute `attr`.
  * `dict_replace_if_equal <attr> <key> <old_value> <new_value>`: Replaces
    `old_value` with `new_value` for key `key` in dictionary attribute `attr`.
    If the key is not present in the dictionary, or does not have value
    `old_value`, it will _not_ be updated. Wildcard `*` matches all keys.
  * `dict_list_add <attr> <key> <value(s)>`:  Adds value(s) to the list in the
    dict attribute `attr`.
  * `format`: Force formatting of all files, even if they were not changed by
    other commands.

Here, `<attr>` represents an attribute (being `add`ed/`rename`d/`delete`d etc.),
e.g.: `srcs`, `<value(s)>` represents values of the attribute and so on.
A '?' indicates that the preceding argument is optional.

The fix command without a fix specified applied to all eligible fixes.
Use `//path/to/pkg:__pkg__` as label for file level changes like `new_load` and
`new`.
A transformation can be applied to all rules of a particular kind by using
`%rule_kind` at the end of the label(see examples below).

The following commands only apply to `MODULE.bazel` files (e.g. the target
`//MODULE.bazel:all`):

  * `use_repo_add <use_extension variable name> <repo(s)>`:
    Ensures that the given repositories are imported via `use_repo` for the
    extension for which the given top-level variable contains the return value
    of a `use_extension` call.
  * `use_repo_remove <use_extension variable name> <repo(s)>`:
    Ensures that the given repositories are *not* imported via `use_repo` for
    the extension for which the given top-level variable contains the return
    value of a `use_extension` call.
  * `use_repo_add [dev] <extension .bzl file> <extension name> <repo(s)>`:
    Ensures that the given repositories generated by the given extension are
    imported via `use_repo`. If the `dev` argument is given, extension usages
    with `dev_dependency = True` will be considered instead. Extension usages
    with `isolated = True` are ignored.
  * `use_repo_remove [dev] <extension .bzl file> <extension name> <repo(s)>`:
    Ensures that the given repositories generated by the given extension are
    *not* imported via `use_repo`. If the `dev` argument is given, extension
    usages with `dev_dependency = True` will be considered instead. Extension
    usages with `isolated = True` are ignored.

#### Supported types

For the commands that support explicitly providing argument types, the following
types are available:

  * `expr`: arbitrary expression, won't be quoted as a string unless already
    quoted. Buildozer doesn't check that the expression is syntactically
    correct, however the subsequent formatting step will fail if it's not.

#### Examples

```bash
# Edit //pkg:rule and //pkg:rule2, and add a dependency on //base
buildozer 'add deps //base' //pkg:rule //pkg:rule2

# A load for a skylark file in //pkg
buildozer 'new_load //tools/build_rules:build_test.bzl build_test' //pkg:__pkg__

# Replaces existing loads for build_test in //pkg
buildozer 'replace_load @rules_build//build:defs.bzl build_test' //pkg:__pkg__

# Replaces modules of loads using regular expressions.
#
# In this example
#     load("@rules_foo//foo:defs.bzl", "foo_library", "foo_test")
# will be replaced with
#     load("//third_party/build_defs/rules_foo/foo:defs.bzl", "foo_library", "foo_test")
buildozer 'substitute_load ^@([^/]*)//([^:].*)$ //third_party/build_defs/${1}/${2}' //pkg:__pkg__

# Change the default_visibility to public for the package //pkg
buildozer 'set default_visibility //visibility:public' //pkg:__pkg__

# Change all gwt_module targets to java_library in the package //pkg
buildozer 'set kind java_library' //pkg:%gwt_module

# Replace the dependency on pkg_v1 with a dependency on pkg_v2
buildozer 'replace deps //pkg_v1 //pkg_v2' //pkg:rule

# Replace all dependencies using regular expressions.
buildozer 'substitute deps //old/(.*) //new/${1}' //pkg:rule

# Delete the dependency on foo in every cc_library in the package
buildozer 'remove deps foo' //pkg:%cc_library

# Delete the testonly attribute in every rule in the package
buildozer 'remove testonly' '//pkg:*'

# Add a comment to the timeout attribute of //pkg:rule_test
buildozer 'comment timeout Delete\ this\ after\ 2015-12-31.' //pkg:rule_test

# Add a new rule at the end of the file
buildozer 'new java_library foo' //pkg:__pkg__

# Add a cc_binary rule named new_bin before the rule named tests
buildozer 'new cc_binary new_bin before tests' //:__pkg__

# Copy an attribute from `protolib` to `py_protolib`.
buildozer 'copy testonly protolib' //pkg:py_protolib

# Set two attributes in the same rule
buildozer 'set compile 1' 'set srcmap 1' //pkg:rule

# Make a default explicit in all soy_js rules in a package
buildozer 'set_if_absent allowv1syntax 1' //pkg:%soy_js

# Add an attribute new_attr with value "def_val" to all cc_binary rules
# Note that special characters will automatically be escaped in the string
buildozer 'add new_attr def_val' //:%cc_binary

# Specifying an argument type to prevent automatic quoting
buildozer 'add tags:expr FOO' //pkg:rule  # adds tags = [FOO]
```

### Print commands

These commands are not modifying files, Buildifier returns 0 after a successful
execution.

  * `print <attribute(s)>`: For each target, prints the value of the attributes
   (see below).
  * `print_comment <attr>? <value>?`: Prints a comment associated with a rule,
    an attribute or a specific value in a list.

The print command prints the value of the attributes. If a target doesn't have
the attribute, a warning is printed on stderr.

There are some special attributes in the `print` command:

  * `kind`: displays the name of the function
  * `attrs`: displays the attribute names of the rule
  * `label`: the fully qualified label
  * `rule`: the entire rule definition
  * `startline`: the line number on which the rule begins in the BUILD file
  * `endline`: the line number on which the rule ends in the BUILD file
  * `path`: the absolute path to the BUILD file that contains the rules

#### Examples

```shell
# Print the kind of a target
buildozer 'print kind' base  # output: cc_library

# Print the name of all cc_library in //base
buildozer 'print name' base:%cc_library

# Get the default visibility of the //base package
buildozer 'print default_visibility' base:%package

# Print labels of cc_library targets in //base that have a deps attribute
buildozer 'print label deps' base:%cc_library 2>/dev/null | cut -d' ' -f1

# Print the list of labels in //base that explicitly set the testonly attribute:
buildozer 'print label testonly' 'base:*' 2>/dev/null

# Print the entire definition (including comments) of the //base:heapcheck rule:
buildozer 'print rule' //base:heapcheck
```

## Converting labels

Buildozer works at the syntax-level. It doesn't evaluate the BUILD files. If you
need to query the information Bazel has, please use `bazel query`. If you have a
list of Bazel labels, chances are that some of them are generated by BUILD
extensions. Labels in Buildozer are slightly different from labels in Bazel.
Bazel cares about the generated code, while Buildozer looks at the BUILD file
before macro expansion.

To see the expanded BUILD files, try:

```shell
bazel query --output=build //path/to/BUILD
```

## Do multiple changes at once

Use `buildozer -f <file>` to load a list of commands and labels from a file (see
[Usage](#usage) above).

```shell
$ cat /tmp/cmds
# a comment
new cc_library foo|//buildtools/buildozer/BUILD
add deps //base //strings|add srcs foo.cc|//buildtools/buildozer:foo
add deps :foo|//buildtools/buildozer

$ buildozer -f /tmp/cmds
fixed //buildtools/buildozer/BUILD
```

The list of commands will typically be generated and can be large. This is
efficient: Commands are grouped so that each file is modified once. Files are
processed in parallel.

Alternatively, BUILD files can be read from the standard input and written to
the standard output, by using the `-` package name:

```shell
$ cat /tmp/cmds
add deps //base //strings|-:foo|-:bar

$ cat some/path/BUILD | buildozer -f /tmp/cmds
```

This writes the result of updating the `:foo` and `:bar` targets in the input
BUILD file to the standard output.

Buildozer commands can be made executable by means of a shebang line, too:

```shell
#!/usr/bin/env -S buildozer -f
#
# Adds //base and //string dependencies to :foo and :bar.

add deps //base //strings|-:foo|-:bar
```

## Using Buildozer in-memory

Some clients of Buildozer have the need to execute buildozer actions in memory
(due to a service environment which does not have access to their file system).
This can be done using, `edit.ExecuteCommandsOnInlineFile`, which accepts
commands and BUILD file content as bytes, applies the changes and returns the
raw file content.
For more details and implementation, see [`/edit/buildozer.go`](../edit/buildozer.go)

Some caveats of running Buildozer in-memory:

* The function assumes (and validates to some extent) that all commands apply to
  the same file and will return errors if there are commands affecting different
  paths.
* When referencing targets, the function will not reliably determine if targets
  are local or remote. Hence redundant path references may be included in
  output. (e.g. `add dep //package/path:bar|//package/path:foo` would add the dep
  `//package/path:bar` instead of just `:bar`).

## Error code

The return code is:

  * `0` on success, if changes were made or only readonly commands were executed
  * `1` when there is a usage error
  * `2` when at least one command has failed
  * `3` on success, when no changes were made

## Source Structure

  * `buildozer/main.go` : Entry point for the buildozer binary
  * `edit/buildozer.go` : Implementation of functions for the buildozer commands
  * `edit/edit.go`: Library functions to perform various operations on ASTs. These
  * functions are called by the impl functions in buildozer.go
  * `edit/fix.go`:  Functions for various fixes for the `buildozer 'fix <fix(es)>'`
   command, like cleaning unused loads, changing labels to canonical notation, etc.
  * `edit/types.go`: Type information for attributes

