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

func TestDeprecatedFunction(t *testing.T) {
	defer setUpFileReader(map[string]string{
		"test/package/foo.bzl": `
def foo():
  """
  This is a function foo.

  Please use it in favor of the
  deprecated function bar
  """
  pass

def bar():
  """
  This is a function bar.

  Deprecated:
    please use foo instead.
  """
  pass
`,
		"test/package/invalid.bzl": `
This is not a valid Starlark file
`,
	})()

	checkFindings(t, "deprecated-function", `
load(":foo.bzl", "foo", "bar", "baz")
load("//test/package:foo.bzl", "foo", "bar", "baz")
load(":invalid.bzl", "foo", "bar", "baz")
load(":nonexistent.bzl", "foo", "bar", "baz")
`,
		[]string{
			`1: The function "bar" defined in "//test/package/foo.bzl" is deprecated.`,
			`2: The function "bar" defined in "//test/package/foo.bzl" is deprecated.`,
		},
		scopeEverywhere)
}

func TestDeprecatedFunctionAnotherRepo(t *testing.T) {
	defer setUpFileReader(map[string]string{
		"test/package/foo.bzl": `
def foo():
  """
  This is a function foo.

  Deprecated:
    do not use.
  """
  pass
`,
	})()

	checkFindings(t, "deprecated-function", `
load("@another_repo//test/package:foo.bzl", "foo")
`,
		[]string{},
		scopeEverywhere)
}

func TestDeprecatedFunctionNoReader(t *testing.T) {
	checkFindings(t, "deprecated-function", `
load(":foo.bzl", "foo", "bar", "baz")
load("//test/package:foo.bzl", "foo", "bar", "baz")
load(":invalid.bzl", "foo", "bar", "baz")
load(":nonexistent.bzl", "foo", "bar", "baz")
`,
		[]string{},
		scopeEverywhere)
}
