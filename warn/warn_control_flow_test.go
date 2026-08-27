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

func TestMissingReturnValueWarning(t *testing.T) {
	// empty return
	checkFindings(t, "return-value", `
def foo():
  if x:
    return x
  else:
    return
`, []string{
		`:5: Some but not all execution paths of "foo" return a value.`,
	}, scopeEverywhere)

	// empty return; implicit return in the end
	checkFindings(t, "return-value", `
def bar():
  if x:
    pass
  elif y:
    return y
  else:
    for z in t:
      return
`, []string{
		`:1: Some but not all execution paths of "bar" return a value.
The function may terminate by an implicit return in the end.`,
		`:8: Some but not all execution paths of "bar" return a value.`,
	}, scopeEverywhere)

	// implicit return in the end
	checkFindings(t, "return-value", `
def foo():
  if x:
    return x
  else:
    bar()
`, []string{
		`:1: Some but not all execution paths of "foo" return a value.
The function may terminate by an implicit return in the end.`,
	}, scopeEverywhere)

	// implicit return in the end
	checkFindings(t, "return-value", `
def bar():
  if x:
    return x
  elif y:
    return y
  else:
    foo
    if z:
      return z

  if foo:
     return not foo
`, []string{
		`:1: Some but not all execution paths of "bar" return a value.
The function may terminate by an implicit return in the end.`,
	}, scopeEverywhere)

	// only returned values and fail() statements, ok
	checkFindings(t, "return-value", `
def bar():
  if x:
    return x
  elif y:
    return y
  else:
    foo
    if z:
      return z

  if foo:
     return not foo
  else:
     fail("unreachable")
`, []string{}, scopeEverywhere)

	// implicit return in the end because fail() is not a statement
	checkFindings(t, "return-value", `
def bar():
  if x:
    return x
  elif y:
    return y
  else:
    foo
    if z:
      return z

  if foo:
     return not foo
  else:
     foo() or fail("unreachable")
`, []string{
		`:1: Some but not all execution paths of "bar" return a value.
The function may terminate by an implicit return in the end.`,
	}, scopeEverywhere)

	// only empty returns, ok
	checkFindings(t, "return-value", `
def bar():
  if x:
    x()
  elif y:
    return
  else:
    foo
    if z:
      fail()

  if foo:
     return
`, []string{}, scopeEverywhere)

	// no returns, ok
	checkFindings(t, "return-value", `
def foobar():
  pass
`, []string{}, scopeEverywhere)

	// only fails, ok
	checkFindings(t, "return-value", `
def foobar():
  if foo:
    fail()
`, []string{}, scopeEverywhere)

	// nested functions, no errors
	checkFindings(t, "return-value", `
def foo():
  def bar():
    pass

  return bar
`, []string{}, scopeEverywhere)

	// nested functions, missing return value in the outer function
	checkFindings(t, "return-value", `
def foo():
  def bar():
    pass

  if bar():
    return 42
`, []string{
		`:1: Some but not all execution paths of "foo" return a value.
The function may terminate by an implicit return in the end.`,
	}, scopeEverywhere)

	// nested functions, missing return value in the inner function
	checkFindings(t, "return-value", `
def foo():
  def bar():
    if something:
      return something

  if bar():
    return 42
  return 43
`, []string{
		`:2: Some but not all execution paths of "bar" return a value.
The function may terminate by an implicit return in the end.`,
	}, scopeEverywhere)

	// nested functions, missing return value in both
	checkFindings(t, "return-value", `
def foo():
  def bar():
    if something:
      return something

  if bar():
    return 42
`, []string{
		`:1: Some but not all execution paths of "foo" return a value.
The function may terminate by an implicit return in the end.`,
		`:2: Some but not all execution paths of "bar" return a value.
The function may terminate by an implicit return in the end.`,
	}, scopeEverywhere)
}

func TestUnreachableStatementWarning(t *testing.T) {
	// after return
	checkFindings(t, "unreachable", `
def foo():
  return
  bar()
  baz()
`, []string{
		`:3: The statement is unreachable.`,
	}, scopeEverywhere)

	// two returns
	checkFindings(t, "unreachable", `
def foo():
  return 1
  return 2
`, []string{
		`:3: The statement is unreachable.`,
	}, scopeEverywhere)

	// after fail()
	checkFindings(t, "unreachable", `
def foo():
  fail("die")
  bar()
  baz()
`, []string{
		`:3: The statement is unreachable.`,
	}, scopeEverywhere)

	// after break and continue
	checkFindings(t, "unreachable", `
def foo():
  for x in y:
    if x:
      break
      bar()  # unreachable
    if y:
      continue
      bar()  # unreachable

def bar():
  for x in y:
    if x:
      break
    elif y:
      continue
    else:
      return x

    foo()  # unreachable
  foobar()  # potentially reachable
`, []string{
		`:5: The statement is unreachable.`,
		`:8: The statement is unreachable.`,
		`:19: The statement is unreachable.`,
	}, scopeEverywhere)

	// ok
	checkFindings(t, "unreachable", `
def foo():
  if x:
    return
  bar()
`, []string{}, scopeEverywhere)

	// ok
	checkFindings(t, "unreachable", `
def foo():
  x() or fail("maybe")
  bar()
`, []string{}, scopeEverywhere)

	// unreacheable statement inside a nested function
	checkFindings(t, "unreachable", `
def foo():
  def bar():
    fail("die")
    baz()
`, []string{
		`:4: The statement is unreachable.`,
	}, scopeEverywhere)

	// unreachable comment is ok
	checkFindings(t, "unreachable", `
def foo():
  fail("die")

  # comment
`, []string{}, scopeEverywhere)
}

func TestNoEffect(t *testing.T) {
	checkFindings(t, "no-effect", `
"""Docstring."""
def bar():
    """Other Docstring"""
    fct()
    pass
    return 2

[f() for i in rang(3)] # top-level comprehension is okay
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "no-effect", `
def foo():
    [fct() for i in range(3)]
	`,
		[]string{":2: Expression result is not used. Use a for-loop instead"},
		scopeEverywhere)

	checkFindings(t, "no-effect", `None`,
		[]string{":1: Expression result is not used."},
		scopeEverywhere)

	checkFindings(t, "no-effect", `
foo             # 1
foo()

def bar():
    [1, 2]      # 5
    if True:
      "string"  # 7
`,
		[]string{":1:", ":5:", ":7:"},
		scopeEverywhere)

	checkFindings(t, "no-effect", `
# A comment

"""A docstring"""

# Another comment

"""Not a docstring"""

def bar():
    """A docstring"""
    foo
    """ Not a docstring"""
    return foo
`,
		[]string{
			":7: Expression result is not used. Docstrings should be the first statements of a file or a function (they may follow comment lines).",
			":11: Expression result is not used.",
			":12: Expression result is not used. Docstrings should be the first statements of a file or a function (they may follow comment lines).",
		}, scopeEverywhere)

	checkFindings(t, "no-effect", `
foo == bar
foo = bar
a + b
c // d
-e
foo != bar

foo += bar
bar -= bar
bar *= bar
bar /= bar
bar //= bar
bar %= bar
bar &= bar
bar |= bar
bar ^= bar
bar <<= bar
bar >>= bar
`,
		[]string{":1:", ":3:", ":4:", ":5:", ":6:"},
		scopeEverywhere)

	checkFindings(t, "no-effect", `
def foo():
  """Doc."""
  def bar():
    """Doc."""
    foo == bar
`,
		[]string{":5:"},
		scopeEverywhere)
}

func TestWarnUnusedVariable(t *testing.T) {
	checkFindings(t, "unused-variable", `
load(":f.bzl", "x")
x = "unused"
y = "also unused"
z = "name"
t = "unused by design"  # @unused
_foo, _bar = pair  #@unused
cc_library(name = z)

def f():
  pass

def g():
  pass

g() + 3
`,
		[]string{":2: Variable \"x\" is unused.",
			":3: Variable \"y\" is unused.",
			":9: Function \"f\" is unused."},
		scopeDeclarative)

	checkFindings(t, "unused-variable", `
a = 1
b = 2
c = 3
d = (a if b else c)  # only d is unused
`,
		[]string{":4: Variable \"d\" is unused."},
		scopeDeclarative)

	checkFindings(t, "unused-variable", `
_a = 1
_a += 2
_b = 3
print(_b)

def _f(): pass
def _g(): pass
_g()
`,
		[]string{
			":1: Variable \"_a\" is unused.",
			":6: Function \"_f\" is unused.",
		},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
a = 1

def foo(
    x,
    y = 0,
    z = 1):
  b = 2
  c = 3
  d = (a if b else c)  # only d is unused
  e = 7
  f = 8  # @unused
  # @unused
  g = 9

  return e + z

foo()
`,
		[]string{
			":4: Variable \"x\" is unused.",
			":5: Variable \"y\" is unused.",
			":9: Variable \"d\" is unused.",
		},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
a = 1

def foo(a):
  b = 2
  return a

def foo():
  pass

def bar(c, cc):
  d = 3
  print(c)

  def baz():
    foo()
    d = 4
    return a

bar()
`,
		[]string{
			":4: Variable \"b\" is unused.",
			":10: Variable \"cc\" is unused.",
			":11: Variable \"d\" is unused.",
			":14: Function \"baz\" is unused.",
			":16: Variable \"d\" is unused.",
		},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
def foo():
  a = 1
  b = 2
  c = 3

  def bar(
      x = a + baz(c = 4),
      y = b):
    pass

foo()
`,
		[]string{
			":4: Variable \"c\" is unused.",
			":6: Function \"bar\" is unused.",
			":7: Variable \"x\" is unused.",
			":8: Variable \"y\" is unused.",
		},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
def foo():
  a = 1
  b = 2
  c = [x for a in aa if a % b for x in a]
  return c

foo()
`,
		[]string{
			":2: Variable \"a\" is unused.",
		},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
def foo():
  a = 1
  b = 2
  c = [
    a + b
    for a in [b for b in bb if b]
    if a
  ]
  return c

foo()
`,
		[]string{
			":2: Variable \"a\" is unused.",
		},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
def foo():
  a = 1
  b = 2

  def bar(*args):
    def baz(**kwargs):
      def foobar(*a,
                 **kw):
        return b
      return foobar(**kwargs)
    return baz(*args)
  return bar()

foo()
`,
		[]string{
			":2: Variable \"a\" is unused.",
			":7: Variable \"a\" is unused.",
			":8: Variable \"kw\" is unused.",
		},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
def foo():
  a = 1
  b = 2
  c = 3
  d = 4
  e, f = 5, 6

  for x, yy in xx:
    for (y, z, _, _t) in yy:
      print(a + y)

  if bar:
    print(c)
  elif baz:
    print(d)
  else:
    print(e)

foo()
`,
		[]string{
			":3: Variable \"b\" is unused.",
			":6: Variable \"f\" is unused.",
			":8: Variable \"x\" is unused.",
			":9: Variable \"z\" is unused.",
		},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
def foo():

  # @unused
  def bar():
    pass

  def baz():
    pass

foo()
`,
		[]string{
			":7: Function \"baz\" is unused.",
		},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
def foo(my_iterable, arg, _some_unused_argument, _also_unused = None, *_args, **_kwargs):

  a, b, _c = 1, 2, 3  # ok to not use _c
  print(a)

  _d, _e = 4, 5  # all are underscored
  print(_d)

  for f, g, _h, _ in my_iterable:  # ok to not use any underscored
    print(f)

  for _i, (_j, _k) in another_iterable:  # ok to not use any of them
    pass

  [1 for (_y, _z) in bar]

foo()
`,
		[]string{
			":1: Variable \"arg\" is unused.",
			":3: Variable \"b\" is unused.",
			":6: Variable \"_e\" is unused.",
			":9: Variable \"g\" is unused.",
		},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
def foo(
    x,
    _y,
    z,  # @unused
    t = 42,  #@unused
    *args,  # @unused
    **kwargs,  ### also @unused
):
  pass

foo()
`,
		[]string{
			":2: Variable \"x\" is unused.",
		},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
def foo(
    name,
    x):
  pass


def bar(
    name = "",
    y = 3):
  pass


foo()
bar()
`,
		[]string{
			":3: Variable \"x\" is unused.",
			":9: Variable \"y\" is unused.",
		},
		scopeEverywhere)

	// foo is unused in the outer macro, since the symbol is overloaded, and only
	// the inner_def "foo" is actually used.
	checkFindings(t, "unused-variable", `
def sample_macro_with_unused_foo(name, foo = "foo"):
  def inner_def(foo = "bar"):
    print(foo)

  inner_def()

sample_macro_with_unused_foo()
`,
		[]string{
			":1: Variable \"foo\" is unused.",
		},
		scopeEverywhere)

	// Foo is referenced as the default value for foo in inner_def, hence it is not unused.
	checkFindings(t, "unused-variable", `
def sample_macro_with_used_foo(name, foo = "foo"):
  def inner_def(bar = foo):
    print(bar)

  inner_def()

sample_macro_with_used_foo()
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
def foo(items):
  items.append(1)

foo([])
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
def foo(passed_in_list_of_lists):
  for mutatable_list in passed_in_list_of_lists:
    mutatable_list.append(1)

foo([[], []])
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "unused-variable", `
def framework_import_impl():
  used_list = []
  unused_list = []
  unused_dictionary = {}

  used_list.extend([1])
  unused_dictionary.update(_ensure_swiftmodule_is_embedded(swiftmodule))
  unused_list.extend([
			x
      for x in used_list
      if x > 5
  ])

  return used_list

framework_import_impl()
`,
		[]string{
			":3: Variable \"unused_list\" is unused.",
			":4: Variable \"unused_dictionary\" is unused.",
		},
		scopeEverywhere)
}

func TestRedefinedVariable(t *testing.T) {
	checkFindings(t, "redefined-variable", `
x = "old_value"
x = "new_value"
x[1] = "new"
cc_library(name = x)`,
		[]string{":2: Variable \"x\" has already been defined."},
		scopeEverywhere)

	checkFindings(t, "redefined-variable", `
x = "a"

def foo():
  x = "b"
  y = "c"
  y = "d"

def bar():
  x = "e"
  y = "f"
  y = "g"`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "redefined-variable", `
x = [1, 2, 3]
y = [a for a in b]
z = list()
n = 43

x += something()
y += something()
z += something()
n += something()
x -= something()`,
		[]string{
			":9: Variable \"n\" has already been defined.",
			":10: Variable \"x\" has already been defined.",
		},
		scopeEverywhere)

	checkFindings(t, "redefined-variable", `
x = [1, 2, 3]
y = [a for a in b]
z = list()

a = something()
b = something()
c = something()
d = something()
e = something()

a += x
b += y
c += z
d += [42]
e += foo`,
		[]string{
			":15: Variable \"e\" has already been defined.",
		},
		scopeEverywhere)
}

func TestWarnUnusedLoad(t *testing.T) {
	checkFindingsAndFix(t, "load", `
load(":f.bzl", "s1", "s2")
load(":bar.bzl", "s1")
foo(name = s1)`, `
load(":f.bzl", "s1")
load(":bar.bzl", "s1")
foo(name = s1)`,
		[]string{
			":1: Loaded symbol \"s2\" is unused.",
			":2: A different symbol \"s1\" has already been loaded on line 1.",
		},
		scopeEverywhere)

	checkFindingsAndFix(t, "load", `
load("foo", "b", "a", "c")
load("foo", "a", "d", "e")

z = a + b + d`, `
load("foo", "a", "b")
load("foo", "d")

z = a + b + d`,
		[]string{
			":1: Loaded symbol \"c\" is unused.",
			":2: Symbol \"a\" has already been loaded on line 1.",
			":2: Loaded symbol \"e\" is unused.",
		},
		scopeEverywhere)

	checkFindingsAndFix(t, "load", `
load("foo", "a")
a(1)
load("bar", "a")
a(2)
load("bar", a = "a")
a(3)
load("bar", a = "b")
a(4)
load("foo", "a")
a(5)
load("foo", "a")
a(6)
load("foo", a = "a")
a(7)`, `
load("foo", "a")
a(1)
load("bar", "a")
a(2)

a(3)
load("bar", a = "b")
a(4)
load("foo", "a")
a(5)

a(6)

a(7)`,
		[]string{
			":3: A different symbol \"a\" has already been loaded on line 1.",
			":5: Symbol \"a\" has already been loaded on line 3.",
			":7: A different symbol \"a\" has already been loaded on line 5.",
			":9: A different symbol \"a\" has already been loaded on line 7.",
			":11: Symbol \"a\" has already been loaded on line 9.",
			":13: Symbol \"a\" has already been loaded on line 11.",
		},
		scopeEverywhere)

	checkFindingsAndFix(t, "load", `
load(
  ":f.bzl",
   "s1",
   "s2",  # @unused (s2)
)

# @unused - both s3 and s4
load(
  ":f.bzl",
   "s3",
   "s4",
)`, `
load(
  ":f.bzl",
   "s2",  # @unused (s2)
)

# @unused - both s3 and s4
load(
  ":f.bzl",
   "s3",
   "s4",
)`,
		[]string{":3: Loaded symbol \"s1\" is unused."},
		scopeEverywhere)

	checkFindingsAndFix(t, "load", `
load(":f.bzl", "x")
x = "unused"`, `
x = "unused"`,
		[]string{":1: Loaded symbol \"x\" is unused."},
		scopeEverywhere)

	checkFindings(t, "load", `
load(
  ":f.bzl",
   "s1",
)

def test(x: s1):
  pass
`,
		[]string{},
		scopeEverywhere)
	checkFindings(t, "load", `
load(
  ":f.bzl",
  "s1",
  "s2",
)

def test(x: s1) -> List[s2]:
  pass
`,
		[]string{},
		scopeEverywhere)
	checkFindingsAndFix(t, "load", `
load(
  ":f.bzl",
  "s1",
  "s2",
)

load(
  ":s.bzl",
  "s3",
)

def test(x: s1) -> List[s2]:
  pass
`, `
load(
  ":f.bzl",
  "s1",
  "s2",
)

def test(x: s1) -> List[s2]:
  pass
`,
		[]string{
			":9: Loaded symbol \"s3\" is unused.",
		},
		scopeEverywhere)
}

func TestUninitializedVariable(t *testing.T) {
	checkFindings(t, "uninitialized", `
def foo(x):
  if bar:
    x = 1
    y = 2

  bar = True
  print(x + y)
`,
		[]string{
			":2: Variable \"bar\" may not have been initialized.",
			":7: Variable \"y\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(x):
  for t in s:
    x = 1
    y = 2

  print(x + y)
`,
		[]string{
			":6: Variable \"y\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo():
  if bar:
    x = 1
    y = 2
  else:
    if foobar:
      x = 3
      y = 4
    else:
      x = 5

  print(x + y)
`,
		[]string{
			":12: Variable \"y\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(x):
  if bar:
    t = 1
  else:
    for t in maybe_empty:  
      pass

  print(t)
`,
		[]string{
			":8: Variable \"t\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(x):
  if bar:
    t = 1
  else:
    for y in maybe_empty:  
      return

  print(t)
`,
		[]string{
			":8: Variable \"t\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(x):
  if bar:
    for t in [2, 3]:
      pass

  print(t)
`,
		[]string{
			":6: Variable \"t\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(x):
  print(t)  # maybe global or loaded
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(x):
  if bar:
    y = 1

  print(y)
  x, y = y, x
  print(y)
`,
		[]string{
			":5: Variable \"y\" may not have been initialized.",
			":6: Variable \"y\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo():
  if a:
    x = 1
    y = 1
    z = 1
  elif b:
    x = 2
    z = 2
    t = 2
  else:
    x = 3
    y = 3
    t = 3

  print(x + y + z + t)
`,
		[]string{
			":15: Variable \"y\" may not have been initialized.",
			":15: Variable \"z\" may not have been initialized.",
			":15: Variable \"t\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(y):
  if y < 0:
    x = -1
  elif y > 0:
    x = 1
  else:
    fail()

  print(x)
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(y):
  if y < 0:
    x = -1
  elif y > 0:
    x = 1
  else:
    if z:
      fail("z")
    else:
      fail("not z")

  print(x)
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(y):
  if y < 0:
    return
  elif y > 0:
    x = 1
  else:
    return x  # not initialized

  print(x)
`,
		[]string{
			":7: Variable \"x\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(y):
  if y < 0:
    return
  elif y > 0:
    x = 1
  else:
    pass

  print(x)  # not initialized
`,
		[]string{
			":9: Variable \"x\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo():
  for x in y:
    print(x)
    print(x.attr)

  print(x)
  print(x.attr)
`,
		[]string{
			":6: Variable \"x\" may not have been initialized.",
			":7: Variable \"x\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo():
  for x in y:
    a = x
    print(a)

  print(a)
`,
		[]string{
			":6: Variable \"a\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def f():
    if foo:
        x = foo

    f(x = y)
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def f():
    if foo:
        x = foo

    f(x + y)
`,
		[]string{
			":5: Variable \"x\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def f():
    if foo:
        x = foo

    x, y = 1, x
`,
		[]string{
			":5: Variable \"x\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def f():
    if foo:
        x = foo

    x, y = 1, 2
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def f(y):
    return [x for x in y if x]

    x = 1
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def f():
    if foo:
        f(x = foo)
    else:
        x = 3

    print(x)
`,
		[]string{
			":7: Variable \"x\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(x):
  for y in x:
    if foo:
      break
    elif bar:
      continue
    elif baz:
      return
    else:
      z = 3
    print(z)
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(x: int, y: int = 2):
  if bar:
    x = 1
    y = 2
    z = 3

  print(x + y + z)
`,
		[]string{
			":7: Variable \"z\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(x: int, y: int = 2):
  def bar(y=x):
    if baz:
      x = 1
      y = 2
      z = 3

    print(x + y + z)

  if something:
    x = bar()

  return x
`,
		[]string{
			":8: Variable \"x\" may not have been initialized.",
			":8: Variable \"z\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo(x: int, y: int = 2):
  if bar:
    y, z, t = 2, 3, 4
    w, s = 5, 6
    r = 7

  [t for t in range(5)]
  [a for a in range(z + y)]
  {b: c + s for b, c in [
    d * 2 for d in range(t)
    if d != baz(r=w)
  ]}
`,
		[]string{
			":8: Variable \"z\" may not have been initialized.",
			":9: Variable \"s\" may not have been initialized.",
			":10: Variable \"t\" may not have been initialized.",
			":11: Variable \"w\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo():
  x = 1
  for y, z in t:
      print(y, z)

  def bar(x, y, s = z):
    pass
`,
		[]string{
			":6: Variable \"z\" may not have been initialized.",
		},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo():
  for bar in baz:
    pass

  y = [baz.get(bar) for bar in bars]
  x = lambda bar: baz.get(bar)
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo():
  def bar(x):
    print(x)

  for x, y in z:
    bar(x)
`,
		[]string{},
		scopeEverywhere)

	checkFindings(t, "uninitialized", `
def foo():
  [x, y] = [1, 2]
  x = 3
  print(x)
`,
		[]string{},
		scopeEverywhere)
}
