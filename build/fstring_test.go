/*
Copyright 2026 Google LLC

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

package build

import "testing"

// F-strings are opaque text: {...} fields ride through in StringExpr.Value;
// the "f" prefix round-trips via StringExpr.Prefix.

func TestFStringParsePrefixAndValue(t *testing.T) {
	cases := []struct {
		src    string // single f-string expression
		value  string // expected StringExpr.Value
		triple bool
		prefix string
	}{
		{`f"hello"`, "hello", false, "f"},
		{`f"{name}_lib"`, "{name}_lib", false, "f"},
		{`f"{prefix}_{suffix}_test"`, "{prefix}_{suffix}_test", false, "f"},
		{`f"//path/to:{target}"`, "//path/to:{target}", false, "f"},
		{`f"{ctx.label.name}.out"`, "{ctx.label.name}.out", false, "f"},
		{`f"got {value!r}, want {expected!r}"`, "got {value!r}, want {expected!r}", false, "f"},
		{`f"{count:>5d} items"`, "{count:>5d} items", false, "f"},
		{`f"{{not a field}}"`, "{{not a field}}", false, "f"},
		{`f'single {x}'`, "single {x}", false, "f"},
		{`f"""
error: {msg}
  at {loc}
"""`, "\nerror: {msg}\n  at {loc}\n", true, "f"},
		// Plain strings must not pick up a prefix.
		{`"plain {x}"`, "plain {x}", false, ""},
	}

	for _, c := range cases {
		f, err := ParseDefault("test.bzl", []byte("x = "+c.src+"\n"))
		if err != nil {
			t.Errorf("ParseDefault(%q) error: %v", c.src, err)
			continue
		}
		assign, ok := f.Stmt[0].(*AssignExpr)
		if !ok {
			t.Errorf("%q: expected AssignExpr, got %T", c.src, f.Stmt[0])
			continue
		}
		s, ok := assign.RHS.(*StringExpr)
		if !ok {
			t.Errorf("%q: expected StringExpr RHS, got %T", c.src, assign.RHS)
			continue
		}
		if s.Value != c.value {
			t.Errorf("%q: Value = %q, want %q", c.src, s.Value, c.value)
		}
		if s.TripleQuote != c.triple {
			t.Errorf("%q: TripleQuote = %v, want %v", c.src, s.TripleQuote, c.triple)
		}
		if s.Prefix != c.prefix {
			t.Errorf("%q: Prefix = %q, want %q", c.src, s.Prefix, c.prefix)
		}
	}
}

// "f" stays a valid identifier; only "f" immediately followed by a quote
// starts an f-string.
func TestFIdentifierNotPrefix(t *testing.T) {
	src := `f = 1

def f():
  pass

def g(f):
  return f + 1

g(f = 42)
obj.f
`
	if _, err := ParseDefault("test.bzl", []byte(src)); err != nil {
		t.Errorf("expected 'f' to be a valid identifier in all positions, got: %v", err)
	}
}

func TestFStringPrintRoundTrip(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// Preserve as written.
		{`f"{prefix}_{suffix}_test"`, `f"{prefix}_{suffix}_test"`},
		// No inner double quote: canonicalize to double quotes.
		{`f'log({event})'`, `f"log({event})"`},
		// Inner double quote: keep single quotes.
		{`f'{got!r} != "expected"'`, `f'{got!r} != "expected"'`},
		// Plain string with braces must not gain an f-prefix.
		{`"%s does not interpolate {name}"`, `"%s does not interpolate {name}"`},
	}
	for _, c := range cases {
		f, err := ParseDefault("test.bzl", []byte("x = "+c.in+"\n"))
		if err != nil {
			t.Errorf("parse %q: %v", c.in, err)
			continue
		}
		out := string(Format(f))
		want := "x = " + c.want + "\n"
		if out != want {
			t.Errorf("round-trip(%q):\n  got:  %q\n  want: %q", c.in, out, want)
		}
	}
}

// When Token can't be reused (e.g. programmatically constructed StringExpr),
// the printer must still emit the "f" prefix from Prefix and escape Value.
func TestFStringPrintFallback(t *testing.T) {
	cases := []struct {
		s    *StringExpr
		want string
	}{
		{&StringExpr{Value: "{name}_test", Prefix: "f"}, `f"{name}_test"`},
		{&StringExpr{Value: `got "{x}"`, Prefix: "f"}, `f"got \"{x}\""`},
		{&StringExpr{Value: "err:\n  {detail}", Prefix: "f", TripleQuote: true}, "f\"\"\"err:\n  {detail}\"\"\""},
		{&StringExpr{Value: "{not_a_field}"}, `"{not_a_field}"`},
		{&StringExpr{Value: "{x}", Prefix: "f", Token: `"{x}"`}, `f"{x}"`},
	}
	for _, c := range cases {
		f := &File{Stmt: []Expr{c.s}}
		out := string(Format(f))
		want := c.want + "\n"
		if out != want {
			t.Errorf("Format fallback:\n  got:  %q\n  want: %q", out, want)
		}
	}
}
