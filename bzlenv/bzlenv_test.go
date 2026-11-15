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

package bzlenv

import (
	"strconv"
	"strings"
	"testing"

	"github.com/bazel-contrib/buildtools/v10/build"
)

func TestWalkEnvironment(t *testing.T) {
	input := `
a, b = 2, 3

def bar(x, y = a):
    b = 4
    c = a
    [a for a in [b, c]]
    if True:
        return foo()

def foo():
    pass

type T[U] = list[U]

def baz[U](x: T) -> U:
    y: int
    y = 0
    return x[y]
`

	expected := `
a0, b1 = 2, 3

def bar2(x6, y7 = a0):
    b8 = 4
    c9 = a0
    [a10 for a10 in [b8, c9]]
    if True:
        return foo3()

def foo3():
    pass

type T4[U11] = list[U11]

def baz5[U12](x13: T4) -> U12:
    y15: int
    y15 = 0
    return x13[y15]
`

	var buildFile build.Expr
	buildFile, err := build.Parse("test_file.bzl", []byte(input))
	if err != nil {
		t.Fatalf("Bad test input: %v", err)
	}

	var walk func(e *build.Expr, env *Environment)
	walk = func(e *build.Expr, env *Environment) {
		switch e := (*e).(type) {
		case *build.DefStmt:
			binding := env.Get(e.Name)
			if binding != nil {
				e.Name += strconv.Itoa(binding.ID)
			}
		case *build.Ident:
			binding := env.Get(e.Name)
			if binding != nil {
				e.Name += strconv.Itoa(binding.ID)
			}
		}
		WalkOnceWithEnvironment(*e, env, walk)
	}
	walk(&buildFile, NewEnvironment())

	output := strings.Trim(build.FormatString(buildFile), "\n")
	expected = strings.Trim(expected, "\n")
	if output != expected {
		t.Errorf("\nexpected:\n%s\ngot:\n%s", expected, output)
	}
}
