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

import (
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"
)

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

func TestLabelCase(t *testing.T) {
	defer setUpFileReader(map[string]string{
		"lib/BUILD":              "",
		"lib/defs.bzl":           "",
		"foo/bar/BUILD":          "",
		"foo/bar/baz/file.txt":   "",
		"foo/bar/baz/deep/x.txt": "",
		"Upper/BUILD":            "",
		"config/BUILD":           "",
		"sub/file.txt":           "",
	})()

	checkFindingsAndFix(t, "label-case", `
load("//Lib:defs.bzl", "x")
load("//lib:defs.bzl", "y")

package_group(
    name = "group",
    packages = [
        "//Lib",
        "-//Foo/bar/...",
    ],
)

cc_library(
    name = "a",
    srcs = [
        "//Foo/Bar:Baz/file.txt",
        "//foo/bar:Baz/Deep/x.txt",
        "//:Sub/file.txt",
    ],
    deps = [
        "//Lib:hello-lib",
        "//lib:hello-lib",
        "@//FOO/bar",
        "@@//foo/Bar:Baz",
        "//upper",
        "//Upper",
    ] + select({
        "//Config:opt": ["//LIB/..."],
        "//conditions:default": ["-//Foo/Bar/...:all"],
    }),
)
`, `
load("//lib:defs.bzl", "x")
load("//lib:defs.bzl", "y")

package_group(
    name = "group",
    packages = [
        "//lib",
        "-//foo/bar/...",
    ],
)

cc_library(
    name = "a",
    srcs = [
        "//foo/bar:baz/file.txt",
        "//foo/bar:baz/deep/x.txt",
        "//:sub/file.txt",
    ],
    deps = [
        "//lib:hello-lib",
        "//lib:hello-lib",
        "@//foo/bar",
        "@@//foo/bar:Baz",
        "//Upper:upper",
        "//Upper",
    ] + select({
        "//config:opt": ["//lib/..."],
        "//conditions:default": ["-//foo/bar/...:all"],
    }),
)
`, []string{
		`:1: "//Lib:defs.bzl" doesn't match the spelling of the directories on disk. Use "//lib:defs.bzl" instead.`,
		`:7: "//Lib" doesn't match the spelling of the directories on disk. Use "//lib" instead.`,
		`:8: "-//Foo/bar/..." doesn't match the spelling of the directories on disk. Use "-//foo/bar/..." instead.`,
		`:15: "//Foo/Bar:Baz/file.txt" doesn't match the spelling of the directories on disk. Use "//foo/bar:baz/file.txt" instead.`,
		`:16: "//foo/bar:Baz/Deep/x.txt" doesn't match the spelling of the directories on disk. Use "//foo/bar:baz/deep/x.txt" instead.`,
		`:17: "//:Sub/file.txt" doesn't match the spelling of the directories on disk. Use "//:sub/file.txt" instead.`,
		`:20: "//Lib:hello-lib" doesn't match the spelling of the directories on disk. Use "//lib:hello-lib" instead.`,
		`:22: "@//FOO/bar" doesn't match the spelling of the directories on disk. Use "@//foo/bar" instead.`,
		`:23: "@@//foo/Bar:Baz" doesn't match the spelling of the directories on disk. Use "@@//foo/bar:Baz" instead.`,
		`:24: "//upper" doesn't match the spelling of the directories on disk. Use "//Upper:upper" instead.`,
		`:27: "//Config:opt" doesn't match the spelling of the directories on disk. Use "//config:opt" instead.`,
		`:27: "//LIB/..." doesn't match the spelling of the directories on disk. Use "//lib/..." instead.`,
		`:28: "-//Foo/Bar/...:all" doesn't match the spelling of the directories on disk. Use "-//foo/bar/...:all" instead.`,
	}, scopeEverywhere)
}

func TestLabelCaseNoFindings(t *testing.T) {
	defer setUpFileReader(map[string]string{
		"lib/BUILD":            "",
		"lib/file.txt":         "",
		"file.txt":             "",
		"foo/bar/baz/file.txt": "",
		"amb/BUILD":            "",
		"AMB/BUILD":            "",
	})()

	checkFindings(t, "label-case", `
x = [
    "//lib:defs.bzl",
    "//lib",
    "//:Lib",
    "//",
    "//...",
    ":Lib",
    "Lib",
    "Lib/file.txt",
    "@lib//Lib:x",
    "@Lib//lib",
    "//Missing:x",
    "//Lib/missing:x",
    "//Lib//missing:x",
    "//./Lib:x",
    "//Lib/../Lib:x",
    "//File.txt:x",
    "//lib:File.txt",
    "//foo/bar:Baz/missing.txt",
    "//foo/bar:Baz/",
    "//foo/bar:Baz/../x",
    "//foo/bar:baz/File.txt",
    "//visibility:public",
    "//conditions:default",
    "//Amb:x",
    "//AMB:x",
]
`, []string{}, scopeEverywhere)
}

func TestLabelCaseConcatenation(t *testing.T) {
	defer setUpFileReader(map[string]string{
		"lib/BUILD": "",
	})()

	checkFindingsAndFix(t, "label-case", `
cc_library(
    name = "a",
    deps = [
        "//Lib" + ":x",
        "@repo" + "//Lib:y",
        "//Lib:" + NAME,
    ],
)
`, `
cc_library(
    name = "a",
    deps = [
        "//lib" + ":x",
        "@repo" + "//Lib:y",
        "//lib:" + NAME,
    ],
)
`, []string{
		`:4: "//Lib" doesn't match the spelling of the directories on disk. Use "//lib" instead.`,
		`:6: "//Lib:" doesn't match the spelling of the directories on disk. Use "//lib:" instead.`,
	}, scopeEverywhere)
}

func TestLabelCaseSpecialLabels(t *testing.T) {
	defer setUpFileReader(map[string]string{
		"Conditions/BUILD": "",
		"Visibility/BUILD": "",
	})()

	// Labels with a special meaning are left alone, even if a directory with a
	// differently spelled name exists.
	checkFindingsAndFix(t, "label-case", `
x = [
    "//conditions:default",
    "@//conditions:default",
    "//visibility:public",
    "@@//visibility:private",
    "//conditions:other",
]
`, `
x = [
    "//conditions:default",
    "@//conditions:default",
    "//visibility:public",
    "@@//visibility:private",
    "//Conditions:other",
]
`, []string{
		`:6: "//conditions:other" doesn't match the spelling of the directories on disk. Use "//Conditions:other" instead.`,
	}, scopeEverywhere)
}

func TestLabelCaseNoSpecialLabelsIntroduced(t *testing.T) {
	defer setUpFileReader(map[string]string{
		"conditions/BUILD": "",
		"visibility/BUILD": "",
	})()

	// Fixing these labels would give them a special meaning.
	checkFindings(t, "label-case", `
x = [
    "//Conditions:default",
    "//Visibility:public",
]
`, []string{}, scopeEverywhere)
}

func TestLabelCaseVisibilityFunction(t *testing.T) {
	defer setUpFileReader(map[string]string{
		"lib/BUILD": "",
	})()

	// visibility() in .bzl files takes package specifications rather than labels.
	checkFindingsAndFix(t, "label-case", `
visibility(["//Lib", "//Lib/..."])

visibility("//Lib")
`, `
visibility(["//lib", "//lib/..."])

visibility("//lib")
`, []string{
		`:1: "//Lib" doesn't match the spelling of the directories on disk. Use "//lib" instead.`,
		`:1: "//Lib/..." doesn't match the spelling of the directories on disk. Use "//lib/..." instead.`,
		`:3: "//Lib" doesn't match the spelling of the directories on disk. Use "//lib" instead.`,
	}, scopeEverywhere)
}

func TestLabelCaseSymlinks(t *testing.T) {
	defer setUpFileReaderFS(fstest.MapFS{
		"lib/BUILD": {},
		"linked":    {Mode: fs.ModeSymlink, Data: []byte("lib")},
		"broken":    {Mode: fs.ModeSymlink, Data: []byte("missing")},
		"filelink":  {Mode: fs.ModeSymlink, Data: []byte("lib/BUILD")},
	})()

	checkFindingsAndFix(t, "label-case", `
x = [
    "//Linked:x",
    "//Broken:x",
    "//FileLink:x",
]
`, `
x = [
    "//linked:x",
    "//Broken:x",
    "//FileLink:x",
]
`, []string{
		`:2: "//Linked:x" doesn't match the spelling of the directories on disk. Use "//linked:x" instead.`,
	}, scopeEverywhere)
}

func TestLabelCaseWithoutLookup(t *testing.T) {
	// No FileReader at all.
	checkFindings(t, "label-case", `
x = "//Lib:x"
`, []string{}, scopeEverywhere)

	// A FileReader that can read files, but not look up directory entries.
	testFileReader = NewFileReader(func(string) ([]byte, error) {
		return nil, fmt.Errorf("file not found")
	})
	defer func() { testFileReader = nil }()
	checkFindings(t, "label-case", `
x = "//Lib:x"
`, []string{}, scopeEverywhere)
}
