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

func TestPackageOnTop(t *testing.T) {
	checkFindingsAndFix(t,
		"package-on-top",
		`
my_macro(name = "foo")
package()`,
		`
package()
my_macro(name = "foo")`,
		[]string{":2: Package declaration should be at the top of the file, after the load() statements, but before any call to a rule or a macro. package_group() and licenses() may be called before package()."},
		scopeDefault|scopeBzl|scopeBuild)

	checkFindingsAndFix(t,
		"package-on-top",
		`
# Some comments

"""This is a docstring"""

load(":foo.bzl", "foo")
load(":bar.bzl", baz = "bar")

package()

foo(baz)`,
		`
# Some comments

"""This is a docstring"""

load(":foo.bzl", "foo")
load(":bar.bzl", baz = "bar")

package()

foo(baz)`,
		[]string{},
		scopeDefault|scopeBzl|scopeBuild)

	checkFindingsAndFix(t,
		"package-on-top",
		`
# Some comments

"""This is a docstring"""

load(":foo.bzl", "foo")
load(":bar.bzl", baz = "bar")

package_group(name = "my_group")
licenses(["my_license"])
foo(baz)
package()`,
		`
# Some comments

"""This is a docstring"""

load(":foo.bzl", "foo")
load(":bar.bzl", baz = "bar")

package_group(name = "my_group")
licenses(["my_license"])

package()
foo(baz)`,
		[]string{":11: Package declaration should be at the top of the file, after the load() statements, but before any call to a rule or a macro. package_group() and licenses() may be called before package()."},
		scopeDefault|scopeBzl|scopeBuild)

	checkFindingsAndFix(t,
		"package-on-top",
		`
"""This is a docstring"""

load(":foo.bzl", "foo")
load(":bar.bzl", baz = "bar")

VISIBILITY = baz

foo()

package(default_visibility = VISIBILITY)`,
		`
"""This is a docstring"""

load(":foo.bzl", "foo")
load(":bar.bzl", baz = "bar")

VISIBILITY = baz

foo()

package(default_visibility = VISIBILITY)`,
		[]string{},
		scopeDefault|scopeBzl|scopeBuild)

	checkFindingsAndFix(t,
		"package-on-top",
		`
"""This is a docstring"""

load(":foo.bzl", "foo")
load(":bar.bzl", baz = "bar")

irrelevant = baz

foo()

package(default_visibility = ["//visibility:public"])`,
		`
"""This is a docstring"""

load(":foo.bzl", "foo")
load(":bar.bzl", baz = "bar")

package(default_visibility = ["//visibility:public"])
irrelevant = baz

foo()`,
		[]string{":10: Package declaration should be at the top of the file, after the load() statements, but before any call to a rule or a macro. package_group() and licenses() may be called before package()."},
		scopeDefault|scopeBzl|scopeBuild)

	// Similar test case from issue #1420: package() after a list assignment
	checkFindingsAndFix(t,
		"package-on-top",
		`package_group(name = "my_group")

some_target(name = "foo")

some_list = []

package(default_visibility = ["//visibility:public"])`,
		`package_group(name = "my_group")

package(default_visibility = ["//visibility:public"])
some_target(name = "foo")

some_list = []`,
		[]string{":7: Package declaration should be at the top of the file, after the load() statements, but before any call to a rule or a macro. package_group() and licenses() may be called before package()."},
		scopeDefault|scopeBzl|scopeBuild)
}

func TestUnsortedDictItems(t *testing.T) {
	checkFindingsAndFix(t, "unsorted-dict-items", `
d = {
	"b": "b value",
	"a": "a value",
}`, `
d = {
	"a": "a value",
	"b": "b value",
}`,
		[]string{":3: Dictionary items are out of their lexicographical order."},
		scopeEverywhere)

	checkFindings(t, "unsorted-dict-items", `
d = {
	"a": "a value",
	"a": "a value",
}`,
		[]string{},
		scopeEverywhere)

	checkFindingsAndFix(t, "unsorted-dict-items", `
d = {
	2: "two",
	"b": "b value",
	1: "one",
	"a": "a value",
	3: "three",
}`, `
d = {
	2: "two",
	"a": "a value",
	1: "one",
	"b": "b value",
	3: "three",
}`,
		[]string{":5: Dictionary items are out of their lexicographical order."},
		scopeEverywhere)

	checkFindings(t, "unsorted-dict-items", `
d = {}`,
		[]string{},
		scopeEverywhere)

	checkFindingsAndFix(t, "unsorted-dict-items", `
d = {
	# b comment
	"b": "b value",
	"a": "a value",
}`, `
d = {
	"a": "a value",
	# b comment
	"b": "b value",
}`,
		[]string{":4: Dictionary items are out of their lexicographical order."},
		scopeEverywhere)

	checkFindings(t, "unsorted-dict-items", `
# @unsorted-dict-items
d = {
	"b": "b value",
	"a": "a value",
}`,
		[]string{},
		scopeEverywhere)

	checkFindingsAndFix(t, "unsorted-dict-items", `
d = {
	"key" : {
		"b": "b value",
		"a": "a value",
	}
}`, `
d = {
	"key" : {
		"a": "a value",
		"b": "b value",
	}
}`,
		[]string{"4: Dictionary items are out of their lexicographical order."},
		scopeEverywhere)

	checkFindingsAndFix(t, "unsorted-dict-items", `
d = {
	"srcs": ["foo.go"],
	"deps": [],
}`, `
d = {
	"deps": [],
	"srcs": ["foo.go"],
}`,
		[]string{"3: Dictionary items are out of their lexicographical order."},
		scopeEverywhere)

	checkFindingsAndFix(t, "unsorted-dict-items", `
d = select({
	"//conditions:zzz": ["myrule_b.sh"],
	"//conditions:default": ["myrule_default.sh"],
})`, `
d = select({
	"//conditions:zzz": ["myrule_b.sh"],
	"//conditions:default": ["myrule_default.sh"],
})`,
		[]string{},
		scopeEverywhere)

	checkFindingsAndFix(t, "unsorted-dict-items", `
foo_binary = rule(
	implementation = _foo_binary_impl,
	attrs = {
		"_foocc": attr.label(
			default = Label("//depsets:foocc"),
		),
		"srcs": attr.label_list(allow_files = True),
		"deps": attr.label_list(),
	},
	outputs = {"out": "%{name}.out"},
)`, `
foo_binary = rule(
	implementation = _foo_binary_impl,
	attrs = {
		"deps": attr.label_list(),
		"srcs": attr.label_list(allow_files = True),
		"_foocc": attr.label(
			default = Label("//depsets:foocc"),
		),
	},
	outputs = {"out": "%{name}.out"},
)`,
		[]string{
			"7: Dictionary items are out of their lexicographical order.",
			"8: Dictionary items are out of their lexicographical order.",
		},
		scopeEverywhere)

	checkFindings(t, "unsorted-dict-items", `
# @unsorted-dict-items
d = {
	"key" : {
		"b": "b value",
		"a": "a value",
	}
}`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "unsorted-dict-items", `
d.update(
	# @unsorted-dict-items
	{
		"b": "value2",
		"a": "value1",
	},
)`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "unsorted-dict-items", `
d.update(
	{
		"b": "value2",
		"a": "value1",
	}, # @unsorted-dict-items
)`,
		[]string{},
		scopeEverywhere)
}

func TestSkylark(t *testing.T) {
	checkFindingsAndFix(t, "skylark-comment", `
# Skyline
foo()
# SkyLark

# Implemented in skylark
# Skylark
bar() # SKYLARK

# see https://docs.bazel.build/versions/main/skylark/lib/Label.html
Label()
`, `
# Skyline
foo()
# Starlark

# Implemented in starlark
# Starlark
bar() # STARLARK

# see https://docs.bazel.build/versions/main/skylark/lib/Label.html
Label()
`,
		[]string{
			`:3: "Skylark" is an outdated name of the language, please use "starlark" instead.`,
			`:7: "Skylark" is an outdated name of the language, please use "starlark" instead.`,
		},
		scopeEverywhere)

	checkFindingsAndFix(t, "skylark-comment", `
"""
Some docstring with skylark
""" # buildifier: disable=skylark-docstring

def f():
  """Some docstring with skylark"""
  # buildozer: disable=skylark-docstring
`, `
"""
Some docstring with skylark
""" # buildifier: disable=skylark-docstring

def f():
  """Some docstring with skylark"""
  # buildozer: disable=skylark-docstring
`,
		[]string{},
		scopeEverywhere)

	checkFindingsAndFix(t, "skylark-docstring", `
# Some file

"""
This is a docstring describing a skylark file
"""

def f():
  """SKYLARK"""

def l():
  """
  Returns https://docs.bazel.build/versions/main/skylark/lib/Label.html
  """
  return Label("skylark")
`, `
# Some file

"""
This is a docstring describing a starlark file
""" 

def f():
  """STARLARK"""

def l():
  """
  Returns https://docs.bazel.build/versions/main/skylark/lib/Label.html
  """
  return Label("skylark")
`,
		[]string{
			`:3: "Skylark" is an outdated name of the language, please use "starlark" instead.`,
			`:8: "Skylark" is an outdated name of the language, please use "starlark" instead.`,
		},
		scopeEverywhere)
}
