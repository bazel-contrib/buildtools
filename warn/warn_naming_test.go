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

func TestAmbiguousNames(t *testing.T) {
	checkFindings(t, "confusing-name", `
i = 0
I, x = 1, 2  # here
l = 3  # here
L = []
L[l] = 4
O = 5  # here

x, l = a, b  # here
a, b = x, l
`,
		[]string{
			":2: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
			":3: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
			":6: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
			":8: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
		}, scopeEverywhere)

	checkFindings(t, "confusing-name", `
def l():  # here
  if True:
    i = 1
    I = 2  # here
  else:
    l = 3  # here
    L = 4
  
  for O in Os:  # here
    pass

  x, l = a, b  # here
  a, b = x, l
`,
		[]string{
			":1: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
			":4: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
			":6: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
			":9: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
			":12: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
		}, scopeEverywhere)

	checkFindings(t, "confusing-name", `
[l for l in s]

cc_library(
  name = "name-conventions",
  tags = [I for I in tags if I],
)

def f(x):
  return [O for O in x]
`,
		[]string{
			":1: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
			":5: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
			":9: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
		}, scopeEverywhere)

	checkFindings(t, "confusing-name", `
[
  foo(l, I, O)
  for l in ls  # here
  if l < m
  for I, (x, O) in bar # here 2 times
  for L in Ls
  if I > l and O == 0
  if x == L
]
`,
		[]string{
			":3: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
			":5: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
			":5: Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').",
		}, scopeEverywhere)
}

func TestVariableNames(t *testing.T) {
	checkFindings(t, "name-conventions", `
_ = 0
foo = 1
FOO = 2
Foo = 3
foo, Bar, BazInfo = 4, 5, 6
foo, (BAR, _) = 7, (8, 9)
fOO = 10
FooInfo = 11
_FooInfo = 12
_BAR = 13
_baz = 14
_Foo_bar, foo_Baz = 15, 16
`,
		[]string{
			`:7: Variable name "fOO" should be lower_snake_case (for variables), UPPER_SNAKE_CASE (for constants), or UpperCamelCase (for providers).`,
			`:12: Variable name "_Foo_bar" should be lower_snake_case (for variables), UPPER_SNAKE_CASE (for constants), or UpperCamelCase (for providers).`,
			`:12: Variable name "foo_Baz" should be lower_snake_case (for variables), UPPER_SNAKE_CASE (for constants), or UpperCamelCase (for providers).`,
		}, scopeEverywhere)

	checkFindings(t, "name-conventions", `
def f(x, _, Arg = None):
  _ = 0
  foo = 1
  FOO = 2
  Foo = 3
  foo, Bar, BazInfo = 4, 5, 6
  foo, (BAR, _) = 7, (8, 9)
  fOO = 10
  FooInfo = 11
  _FooInfo = 12
  _BAR = 13
  _baz = 14
  _Foo_bar, foo_Baz = 15, 16
`,
		[]string{
			`:8: Variable name "fOO" should be lower_snake_case (for variables), UPPER_SNAKE_CASE (for constants), or UpperCamelCase (for providers).`,
			`:13: Variable name "_Foo_bar" should be lower_snake_case (for variables), UPPER_SNAKE_CASE (for constants), or UpperCamelCase (for providers).`,
			`:13: Variable name "foo_Baz" should be lower_snake_case (for variables), UPPER_SNAKE_CASE (for constants), or UpperCamelCase (for providers).`,
		}, scopeEverywhere)

	checkFindings(t, "name-conventions", `
foo(
  Bar = 1,
  ___ = 2,
  BazInfo = 3,
	baz = 4,
  FOO = 5,
)
`,
		[]string{}, scopeEverywhere)
}
