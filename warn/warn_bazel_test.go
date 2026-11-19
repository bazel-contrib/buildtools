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

func TestPathMapping(t *testing.T) {
	checkFindingsAndFix(t, "path-mapping", `
load("@bazel_skylib//lib:paths.bzl", "paths")

def _dirname(file):
    return file.dirname

def _add_srcs(args, srcs):
    args.add_all([src.path for src in srcs])

def _impl(ctx):
    args = ctx.actions.args()
    actions = ctx.actions
    out_dir = ctx.actions.declare_directory("out")

    # Compatible with path mapping
    args.add(src)
    args.add("--out", out)
    args.add_all(dirs, expand_directories = False)
    args.add_all([src], map_each = _dirname)
    args.add_all(srcs, format_each = "--src=%s")
    args.add(src.short_path)
    args.add(src.basename)
    args.add(paths.dirname(src.short_path))
    ctx.actions.run(
        outputs = [out],
        executable = ctx.executable._tool,
        arguments = [args],
    )
    ctx.actions.run_shell(
        outputs = [out],
        command = "cp $1 $2",
        arguments = [args],
    )

    # Incompatible with path mapping, fixed automatically
    args.add(src.path)
    args.add("--out", out.path)
    args.add(src.path, format = "--src=%s")
    args.add("--src=%s" % src.path)
    args.add("--src=" + src.path)
    args.add(src.path + ".d")
    args.add("-I" + src.path + "/include")
    args.add("--src={}".format(src.path))
    args.add(out_dir.path)
    args.add("--out", out_dir.path)
    args.add_all([src.path, out.path])
    args.add_all([src.path for src in srcs])
    args.add_all([src.path for src in srcs], format_each = "-I%s")
    args.add_all(["-I" + inc.path for inc in incs])
    args.add_joined([src.path for src in srcs], join_with = ",")
    args.add_all([src.path, out_dir.path])
    ctx.actions.run(
        outputs = [out],
        executable = tool.path,
        arguments = [args],
    )

    # Incompatible with path mapping, not fixed automatically
    args.add(src.dirname)
    args.add("--bin=%s" % ctx.bin_dir.path)
    args.add("--gen={}".format(ctx.genfiles_dir.path))
    args.add("--root=" + src.root.path)
    args.add(paths.join(ctx.bin_dir.path, "include"))
    args.add(src.path if src else out.path)
    args.add("--src=%s" % src.path, format = "%s")
    args.add("%s=%s" % (name, src.path))
    args.add(out_dir.path, format = "--out=%s")
    args.add_all([src.path for src in srcs if src])
    args.add_all([src.path for src in srcs], map_each = _dirname)
    args.add_all(["-I" + inc.path for inc in incs], format_each = "%s")
    ctx.actions.run(
        outputs = [out],
        executable = tool.path,
        arguments = [src.path, "--out", out.path],
    )
    ctx.actions.run_shell(
        outputs = [out],
        command = "cp %s %s" % (src.path, out.path),
    )
    actions.run_shell(
        outputs = [out],
        command = "cp $1 " + out.dirname,
        arguments = [src.path],
    )

    # Unrelated
    other.add(src.path)
    ctx.actions.other(arguments = [src.path])
    ctx.actions.run(
        outputs = [out],
        executable = ctx.executable._tool,
        arguments = [args],
        progress_message = "Building %s" % out.path,
    )
`, `
load("@bazel_skylib//lib:paths.bzl", "paths")

def _dirname(file):
    return file.dirname

def _add_srcs(args, srcs):
    args.add_all(srcs)

def _impl(ctx):
    args = ctx.actions.args()
    actions = ctx.actions
    out_dir = ctx.actions.declare_directory("out")

    # Compatible with path mapping
    args.add(src)
    args.add("--out", out)
    args.add_all(dirs, expand_directories = False)
    args.add_all([src], map_each = _dirname)
    args.add_all(srcs, format_each = "--src=%s")
    args.add(src.short_path)
    args.add(src.basename)
    args.add(paths.dirname(src.short_path))
    ctx.actions.run(
        outputs = [out],
        executable = ctx.executable._tool,
        arguments = [args],
    )
    ctx.actions.run_shell(
        outputs = [out],
        command = "cp $1 $2",
        arguments = [args],
    )

    # Incompatible with path mapping, fixed automatically
    args.add(src)
    args.add("--out", out)
    args.add(src, format = "--src=%s")
    args.add(src, format = "--src=%s")
    args.add(src, format = "--src=%s")
    args.add(src, format = "%s.d")
    args.add(src, format = "-I%s/include")
    args.add(src, format = "--src=%s")
    args.add_all([out_dir], expand_directories = False)
    args.add_all(["--out", out_dir], expand_directories = False)
    args.add_all([src, out])
    args.add_all(srcs)
    args.add_all(srcs, format_each = "-I%s")
    args.add_all(incs, format_each = "-I%s")
    args.add_joined(srcs, join_with = ",")
    args.add_all([src, out_dir], expand_directories = False)
    ctx.actions.run(
        outputs = [out],
        executable = tool,
        arguments = [args],
    )

    # Incompatible with path mapping, not fixed automatically
    args.add(src.dirname)
    args.add("--bin=%s" % ctx.bin_dir.path)
    args.add("--gen={}".format(ctx.genfiles_dir.path))
    args.add("--root=" + src.root.path)
    args.add(paths.join(ctx.bin_dir.path, "include"))
    args.add(src.path if src else out.path)
    args.add("--src=%s" % src.path, format = "%s")
    args.add("%s=%s" % (name, src.path))
    args.add(out_dir.path, format = "--out=%s")
    args.add_all([src.path for src in srcs if src])
    args.add_all([src.path for src in srcs], map_each = _dirname)
    args.add_all(["-I" + inc.path for inc in incs], format_each = "%s")
    ctx.actions.run(
        outputs = [out],
        executable = tool,
        arguments = [src.path, "--out", out.path],
    )
    ctx.actions.run_shell(
        outputs = [out],
        command = "cp %s %s" % (src.path, out.path),
    )
    actions.run_shell(
        outputs = [out],
        command = "cp $1 " + out.dirname,
        arguments = [src.path],
    )

    # Unrelated
    other.add(src.path)
    ctx.actions.other(arguments = [src.path])
    ctx.actions.run(
        outputs = [out],
        executable = ctx.executable._tool,
        arguments = [args],
        progress_message = "Building %s" % out.path,
    )
`,
		[]string{
			`:7: "src.path" is not rewritten by path mapping. Pass "src" to "args.add_all()" instead (with "expand_directories = False" if it is a directory).`,
			`:35: "src.path" is not rewritten by path mapping. Pass "src" to "args.add()" instead, or use "args.add_all([src], expand_directories = False)" if it is a directory.`,
			`:36: "out.path" is not rewritten by path mapping. Pass "out" to "args.add()" instead, or use "args.add_all([out], expand_directories = False)" if it is a directory.`,
			`:37: "src.path" is not rewritten by path mapping. Pass "src" to "args.add()" instead, or use "args.add_all([src], expand_directories = False)" if it is a directory.`,
			`:38: "src.path" is not rewritten by path mapping. Pass "src" to "args.add()" instead, or use "args.add_all([src], expand_directories = False)" if it is a directory.`,
			`:39: "src.path" is not rewritten by path mapping. Pass "src" to "args.add()" instead, or use "args.add_all([src], expand_directories = False)" if it is a directory.`,
			`:40: "src.path" is not rewritten by path mapping. Pass "src" to "args.add()" instead, or use "args.add_all([src], expand_directories = False)" if it is a directory.`,
			`:41: "src.path" is not rewritten by path mapping. Pass "src" to "args.add()" instead, or use "args.add_all([src], expand_directories = False)" if it is a directory.`,
			`:42: "src.path" is not rewritten by path mapping. Pass "src" to "args.add()" instead, or use "args.add_all([src], expand_directories = False)" if it is a directory.`,
			`:43: "out_dir.path" is not rewritten by path mapping. Pass "out_dir" to "args.add()" instead, or use "args.add_all([out_dir], expand_directories = False)" if it is a directory.`,
			`:44: "out_dir.path" is not rewritten by path mapping. Pass "out_dir" to "args.add()" instead, or use "args.add_all([out_dir], expand_directories = False)" if it is a directory.`,
			`:45: "src.path" is not rewritten by path mapping. Pass "src" to "args.add_all()" instead (with "expand_directories = False" if it is a directory).`,
			`:45: "out.path" is not rewritten by path mapping. Pass "out" to "args.add_all()" instead (with "expand_directories = False" if it is a directory).`,
			`:46: "src.path" is not rewritten by path mapping. Pass "src" to "args.add_all()" instead (with "expand_directories = False" if it is a directory).`,
			`:47: "src.path" is not rewritten by path mapping. Pass "src" to "args.add_all()" instead (with "expand_directories = False" if it is a directory).`,
			`:48: "inc.path" is not rewritten by path mapping. Pass "inc" to "args.add_all()" instead (with "expand_directories = False" if it is a directory).`,
			`:49: "src.path" is not rewritten by path mapping. Pass "src" to "args.add_joined()" instead (with "expand_directories = False" if it is a directory).`,
			`:50: "src.path" is not rewritten by path mapping. Pass "src" to "args.add_all()" instead (with "expand_directories = False" if it is a directory).`,
			`:50: "out_dir.path" is not rewritten by path mapping. Pass "out_dir" to "args.add_all()" instead (with "expand_directories = False" if it is a directory).`,
			`:53: "tool.path" is not rewritten by path mapping. Pass "tool" as "executable" instead.`,
			`:58: "src.dirname" is not rewritten by path mapping. Add "src" to "ctx.actions.args()" with a "map_each" callback that returns "file.dirname" instead.`,
			`:59: "ctx.bin_dir.path" is not rewritten by path mapping. Compute it from an output "File" in a "map_each" callback via "file.root.path" instead.`,
			`:60: "ctx.genfiles_dir.path" is not rewritten by path mapping. Compute it from an output "File" in a "map_each" callback via "file.root.path" instead.`,
			`:61: "src.root.path" is not rewritten by path mapping. Add "src" to "ctx.actions.args()" with a "map_each" callback that returns "file.root.path" instead.`,
			`:62: "ctx.bin_dir.path" is not rewritten by path mapping. Compute it from an output "File" in a "map_each" callback via "file.root.path" instead.`,
			`:63: "src.path" is not rewritten by path mapping. Pass "src" to "args.add()" instead, or use "args.add_all([src], expand_directories = False)" if it is a directory.`,
			`:63: "out.path" is not rewritten by path mapping. Pass "out" to "args.add()" instead, or use "args.add_all([out], expand_directories = False)" if it is a directory.`,
			`:64: "src.path" is not rewritten by path mapping. Pass "src" to "args.add()" instead, or use "args.add_all([src], expand_directories = False)" if it is a directory.`,
			`:65: "src.path" is not rewritten by path mapping. Pass "src" to "args.add()" instead, or use "args.add_all([src], expand_directories = False)" if it is a directory.`,
			`:66: "out_dir.path" is not rewritten by path mapping. Pass "out_dir" to "args.add()" instead, or use "args.add_all([out_dir], expand_directories = False)" if it is a directory.`,
			`:67: "src.path" is not rewritten by path mapping. Pass "src" to "args.add_all()" instead (with "expand_directories = False" if it is a directory).`,
			`:68: "src.path" is not rewritten by path mapping. Pass "src" to "args.add_all()" instead (with "expand_directories = False" if it is a directory).`,
			`:69: "inc.path" is not rewritten by path mapping. Pass "inc" to "args.add_all()" instead (with "expand_directories = False" if it is a directory).`,
			`:72: "tool.path" is not rewritten by path mapping. Pass "tool" as "executable" instead.`,
			`:73: "src.path" is not rewritten by path mapping. Add "src" to "ctx.actions.args()" and pass the "Args" object in "arguments" instead.`,
			`:73: "out.path" is not rewritten by path mapping. Add "out" to "ctx.actions.args()" and pass the "Args" object in "arguments" instead.`,
			`:77: "src.path" is not rewritten by path mapping. Add "src" to "ctx.actions.args()", pass the "Args" object in "arguments" and refer to it as "$1", "$2", ... in "command" instead.`,
			`:77: "out.path" is not rewritten by path mapping. Add "out" to "ctx.actions.args()", pass the "Args" object in "arguments" and refer to it as "$1", "$2", ... in "command" instead.`,
			`:81: "out.dirname" is not rewritten by path mapping. Add "out" to "ctx.actions.args()" with a "map_each" callback that returns "file.dirname" instead.`,
			`:82: "src.path" is not rewritten by path mapping. Add "src" to "ctx.actions.args()" and pass the "Args" object in "arguments" instead.`,
		},
		scopeBzl)
}
