/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package warn

import "testing"

func TestConstantGlob(t *testing.T) {
	checkFindings(t, "constant-glob", `
cc_library(srcs = glob(["foo.cc"]))
cc_library(srcs = glob(include = ["foo.cc"]))
cc_library(srcs = glob(include = ["foo.cc"], exclude = ["bar.cc"]))
cc_library(srcs = glob(exclude = ["bar.cc"], include = ["foo.cc"]))
cc_library(srcs =
	["constant"] + glob([
		"*.cc",
		"test.cpp",
		])
	)
cc_library(srcs = glob(["*.cc"]))
cc_library(srcs = glob(["*.cc"], exclude = ["bar.cc"]))
cc_library(srcs = glob(include = ["*.cc"], exclude = ["bar.cc"]))
cc_library(srcs = glob(exclude = ["bar.cc"], include = ["*.cc"]))`,
		[]string{`:1: Glob pattern "foo.cc" has no wildcard`,
			`:2: Glob pattern "foo.cc" has no wildcard`,
			`:3: Glob pattern "foo.cc" has no wildcard`,
			`:4: Glob pattern "foo.cc" has no wildcard`,
			`:8: Glob pattern "test.cpp" has no wildcard`},
		scopeBuild|scopeBzl|scopeWorkspace)
}

func TestNativeInBuildFiles(t *testing.T) {
	checkFindingsAndFix(t, "native-build", `
native.package("foo")

native.cc_library(name = "lib")
`, `
package("foo")

cc_library(name = "lib")
`, []string{
		`:1: The "native" module shouldn't be used in BUILD files, its members are available as global symbols.`,
		`:3: The "native" module shouldn't be used in BUILD files, its members are available as global symbols.`,
	}, scopeBuild)
}

func TestNativePackage(t *testing.T) {
	checkFindings(t, "native-package", `
native.package("foo")

native.cc_library(name = "lib")
`, []string{
		`:1: "native.package()" shouldn't be used in .bzl files.`,
	}, scopeBzl)
}

func TestDuplicatedName(t *testing.T) {
	checkFindings(t, "duplicated-name", `
cc_library(name = "x")
cc_library(name = "y")
py_library(name = "x")
py_library(name = "z")
php_library(name = "x")`,
		[]string{
			`:3: A rule with name "x" was already found on line 1`,
			`:5: A rule with name "x" was already found on line 1`,
		}, scopeBuild|scopeWorkspace)

	checkFindings(t, "duplicated-name", `
exports_files(["foo.txt"])
[macro(name = "bar_%s" % i) for i in ii]
`,
		[]string{},
		scopeBuild|scopeWorkspace)
}

func TestPositionalArgumentsDoesNotWarnForNamedArguments(t *testing.T) {
	checkFindings(t, "positional-args", `
my_macro = macro()
my_rule = rule()
def my_function(foo):
  pass

my_macro(foo = "bar")
my_rule(foo = "bar")
my_function(foo = "bar")
`,
		[]string{},
		scopeBuild)
}

func TestPositionalArgumentsDoesNotWarnForAllowlistedFunctions(t *testing.T) {
	checkFindings(t, "positional-args", `
register_toolchains(
	"//foo",
	"//bar",
)`,
		[]string{},
		scopeBuild)
}

func TestPositionalArgumentsWarnsForPositionalMacrosOrRuleCalls(t *testing.T) {
	checkFindings(t, "positional-args", `
my_macro = macro()
my_rule = rule()
def my_function(foo):
  pass

my_macro("foo", "bar")
my_rule("foo", "bar")
my_function("foo", "bar")
`,
		[]string{
			`6: All calls to rules or macros should pass arguments by keyword (arg_name=value) syntax.
Found call to rule or macro "my_macro" with positional arguments.`,
			`7: All calls to rules or macros should pass arguments by keyword (arg_name=value) syntax.
Found call to rule or macro "my_rule" with positional arguments.`,
		},
		scopeBuild)
}

func TestPositionalArgumentsWarnsWhenCalledInNestedContexts(t *testing.T) {
	checkFindings(t, "positional-args", `
my_macro = macro()
my_rule = rule()
def my_function(foo):
  pass

other_function(foo = my_macro(x))
other_function(foo = my_rule(x))
other_function(foo = my_function(x))

[my_macro(foo) for foo in bar]
[my_rule(foo) for foo in bar]
[my_function(foo) for foo in bar]
`,
		[]string{
			`6: All calls to rules or macros should pass arguments by keyword (arg_name=value) syntax.
Found call to rule or macro "my_macro" with positional arguments.`,
			`7: All calls to rules or macros should pass arguments by keyword (arg_name=value) syntax.
Found call to rule or macro "my_rule" with positional arguments.`,
			`10: All calls to rules or macros should pass arguments by keyword (arg_name=value) syntax.
Found call to rule or macro "my_macro" with positional arguments.`,
			`11: All calls to rules or macros should pass arguments by keyword (arg_name=value) syntax.
Found call to rule or macro "my_rule" with positional arguments.`,
		},
		scopeBuild)

}

func TestKwargsInBuildFilesWarning(t *testing.T) {
	checkFindings(t, "build-args-kwargs", `
cc_library(
  name = "foo",
  *args,
  **kwargs,
)

foo(*bar(**kgs))`,
		[]string{
			":3: *args are not allowed in BUILD files.",
			":4: **kwargs are not allowed in BUILD files.",
			":7: *args are not allowed in BUILD files.",
			":7: **kwargs are not allowed in BUILD files.",
		},
		scopeBuild)

	checkFindings(t, "build-args-kwargs", `
cc_library(
  name = "foo",
  -args,
)

foo(not bar(-kgs))`,
		[]string{},
		scopeBuild)
}

func TestPrintWarning(t *testing.T) {
	checkFindings(t, "print", `
foo()

print("foo")

def f(x):
  print(x)

  g(x) or print("not g")
`,
		[]string{
			`:3: "print()" is a debug function and shouldn't be submitted.`,
			`:6: "print()" is a debug function and shouldn't be submitted.`,
			`:8: "print()" is a debug function and shouldn't be submitted.`,
		},
		scopeBazel)
}

func TestMakeLocationVariable(t *testing.T) {
	checkFindings(t, "make-location", `
genrule(
    name = "a",
    srcs = [":foo"],
    outs = ["out"],
    cmd = "cp $(location :foo) $@",
)

genrule(
    name = "b",
    srcs = [":a", ":b"],
    outs = ["out"],
    cmd = "cat $(locations :a :b) > $@",
)

genrule(
    name = "c",
    srcs = [":foo"],
    outs = ["out"],
    cmd = "cp $$(location :foo) $$@",
)

cc_test(
    name = "d",
    args = ["--config=$(execpath :cfg)"],
)

load("location", "symbol")
`,
		[]string{
			`:5: The "$(location)" and "$(locations)" make variables are deprecated. Use "$(execpath ...)" or "$(rootpath ...)" instead.`,
			`:12: The "$(location)" and "$(locations)" make variables are deprecated. Use "$(execpath ...)" or "$(rootpath ...)" instead.`,
			`:19: The "$(location)" and "$(locations)" make variables are deprecated. Use "$(execpath ...)" or "$(rootpath ...)" instead.`,
		},
		scopeBuild | scopeBzl)
	checkFindings(t, "make-location", `
load("location", "symbol")
def generate_targets(name, src1, src2):
    native.genrule(
        name = name + "a",
        srcs = [src1],
        outs = ["out"],
        cmd = "cp $(location " + src1 + ") $@",
    )

    native.genrule(
        name = name + "b",
        srcs = [src1, src2],
        outs = ["out"],
        cmd = "cat $(locations " + src1 + " " + src2 + ") > $@",
    )

    native.genrule(
        name = name + "c",
        srcs = [src1],
        outs = ["out"],
        cmd = "cp $$(location " + src1 + ") $$@",
    )
`,
		[]string{
			`:7: The "$(location)" and "$(locations)" make variables are deprecated. Use "$(execpath ...)" or "$(rootpath ...)" instead.`,
			`:14: The "$(location)" and "$(locations)" make variables are deprecated. Use "$(execpath ...)" or "$(rootpath ...)" instead.`,
			`:21: The "$(location)" and "$(locations)" make variables are deprecated. Use "$(execpath ...)" or "$(rootpath ...)" instead.`,
		},
		scopeBuild | scopeBzl)
}

func TestNonConstantProgressMessage(t *testing.T) {
	checkFindings(t, "non-constant-progress-message", `
load(":foo.bzl", "LOADED_MSG")

PROGRESS_MSG = "Building %{input}"
COMPOSED_MSG = PROGRESS_MSG + " into %{output}"
NONE_MSG = None
DYNAMIC_GLOBAL = "Building %s" % "foo"

def _impl(ctx, param_msg):
    local_msg = "Building %{input}"
    actions = ctx.actions

    # Valid constant progress_message values
    ctx.actions.run(
        outputs = [out],
        executable = exe,
        progress_message = "Building %{input}",
    )
    ctx.actions.run_shell(
        outputs = [out],
        command = "cmd",
        progress_message = ("Building " + "%{input}"),
    )
    ctx.actions.symlink(
        output = out,
        target_file = target,
        progress_message = PROGRESS_MSG,
    )
    ctx.actions.run(
        outputs = [out],
        executable = exe,
        progress_message = COMPOSED_MSG + "!",
    )
    ctx.actions.run(
        outputs = [out],
        executable = exe,
        progress_message = None,
    )
    ctx.actions.run(
        outputs = [out],
        executable = exe,
        progress_message = NONE_MSG,
    )

    # Non-constant progress_message values (should warn)
    ctx.actions.run(
        outputs = [out],
        executable = exe,
        progress_message = "Building %s" % ctx.label,
    )
    ctx.actions.run_shell(
        outputs = [out],
        command = "cmd",
        progress_message = "Building {}".format(ctx.label),
    )
    ctx.actions.symlink(
        output = out,
        target_file = target,
        progress_message = "Linking " + ctx.label.name,
    )
    ctx.actions.run(
        outputs = [out],
        executable = exe,
        progress_message = param_msg,
    )
    ctx.actions.run(
        outputs = [out],
        executable = exe,
        progress_message = local_msg,
    )
    ctx.actions.run(
        outputs = [out],
        executable = exe,
        progress_message = LOADED_MSG,
    )
    ctx.actions.run(
        outputs = [out],
        executable = exe,
        progress_message = DYNAMIC_GLOBAL,
    )
    actions.run(
        outputs = [out],
        executable = exe,
        progress_message = get_message(),
    )

    # Unrelated calls (should not warn)
    other.run(progress_message = "Building %s" % ctx.label)
    ctx.actions.other(progress_message = "Building %s" % ctx.label)

def _shadowed(ctx, PROGRESS_MSG):
    ctx.actions.run(
        outputs = [out],
        executable = exe,
        progress_message = PROGRESS_MSG,
    )
`,
		[]string{
			`:48: "progress_message" should be a constant string. Use "%{label}", "%{input}", or "%{output}" instead of dynamic string formatting.`,
			`:53: "progress_message" should be a constant string. Use "%{label}", "%{input}", or "%{output}" instead of dynamic string formatting.`,
			`:58: "progress_message" should be a constant string. Use "%{label}", "%{input}", or "%{output}" instead of dynamic string formatting.`,
			`:63: "progress_message" should be a constant string. Use "%{label}", "%{input}", or "%{output}" instead of dynamic string formatting.`,
			`:68: "progress_message" should be a constant string. Use "%{label}", "%{input}", or "%{output}" instead of dynamic string formatting.`,
			`:73: "progress_message" should be a constant string. Use "%{label}", "%{input}", or "%{output}" instead of dynamic string formatting.`,
			`:78: "progress_message" should be a constant string. Use "%{label}", "%{input}", or "%{output}" instead of dynamic string formatting.`,
			`:83: "progress_message" should be a constant string. Use "%{label}", "%{input}", or "%{output}" instead of dynamic string formatting.`,
			`:94: "progress_message" should be a constant string. Use "%{label}", "%{input}", or "%{output}" instead of dynamic string formatting.`,
		},
		scopeBzl)
}
