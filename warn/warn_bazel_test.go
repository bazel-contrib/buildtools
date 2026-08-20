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

func TestExternalPathWarning(t *testing.T) {
	checkFindings(t, "external-path", `
cc_library(
    name = "foo",
    srcs = ["//external/com_google_protobuf:src/google/protobuf/message.h"],
)

py_binary(
    name = "tool",
    srcs = ["tool.py"],
    data = ["/external/some_repo/data.txt"],
)

java_library(
    name = "lib",
    srcs = glob(["*.java"]),
    deps = ["@maven//:org_apache_commons_commons_lang3"],
)

filegroup(
    name = "configs",
    srcs = ["config.txt"],
)

some_rule(
    arg1 = "normal/path/file.txt",
    arg2 = "/external/repo/file.py",
    arg3 = ["file1.txt", "/external/another/file.cc"],
)

def foo():
    pass
    """unused triple-quote string with /external/"""
`,
		[]string{
			`:9: String contains "/external/" which may indicate a dependency on external repositories that could be fragile.`,
			`:25: String contains "/external/" which may indicate a dependency on external repositories that could be fragile.`,
			`:26: String contains "/external/" which may indicate a dependency on external repositories that could be fragile.`,
			`31: String contains "/external/" which may indicate a dependency on external repositories that could be fragile.`,
		},
		scopeBazel)

	// Test cases that should NOT warn (main repository paths with // prefix)
	checkFindings(t, "external-path", `
cc_library(
    name = "foo",
    srcs = ["//external/repo/file.h"],
    hdrs = ["//external/another_repo/header.h"],
)

py_binary(
    name = "tool",
    srcs = ["//external/tools/tool.py"],
    data = ["//some/path/external/nested/file.txt"],
    args = ["//different/external/location/config.json"],
)`,
		[]string{},
		scopeBazel)

	// Test cases that should NOT warn (doc strings)
	checkFindings(t, "external-path", `
def foo():
    """/external/"""
    pass
`,
		[]string{},
		scopeBazel)
}

func TestCanonicalRepositoryWarning(t *testing.T) {
	checkFindings(t, "canonical-repository", `
load("@@rules_go//go:def.bzl", "go_library")
load("@repo//file.bzl", "symbol")  # Should NOT warn (single @)
load("@@protobuf~5.27.0//src:defs.bzl", "proto_library")

cc_library(
    name = "test",
    deps = ["@@rules_go//cc:toolchain"],
    srcs = ["@@protobuf~5.27.0//src/google/protobuf:message_lite_h"],
)

py_binary(
    name = "tool",
    srcs = ["tool.py"],
    data = ["@repo//file.txt"],  # Should NOT warn (single @)
    args = ["@@some_canonical_repo//path:target"],
)

def foo():
    pass
    """unused triple-quote string with @@"""
`,
		[]string{
			`:1: String contains "@@" which indicates a canonical repository name reference that should be avoided.`,
			`:3: String contains "@@" which indicates a canonical repository name reference that should be avoided.`,
			`:7: String contains "@@" which indicates a canonical repository name reference that should be avoided.`,
			`:8: String contains "@@" which indicates a canonical repository name reference that should be avoided.`,
			`:15: String contains "@@" which indicates a canonical repository name reference that should be avoided.`,
			`20: String contains "@@" which indicates a canonical repository name reference that should be avoided.`,
		},
		scopeBazel)

	// Test cases that should NOT warn (doc strings)
	checkFindings(t, "canonical-repository", `
def foo():
    """@@foo"""
    pass
`,
		[]string{},
		scopeBazel)
}
