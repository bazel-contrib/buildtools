/*
Copyright 2016 Google LLC

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

package edit

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/bazelbuild/buildtools/build"
)

var parseLabelTests = []struct {
	in   string
	repo string
	pkg  string
	rule string
}{
	{"//devtools/buildozer:rule", "", "devtools/buildozer", "rule"},
	{"devtools/buildozer:rule", "", "devtools/buildozer", "rule"},
	{"//devtools/buildozer", "", "devtools/buildozer", "buildozer"},
	{"//base", "", "base", "base"},
	{"//base:", "", "base", "base"},
	{"@r//devtools/buildozer:rule", "r", "devtools/buildozer", "rule"},
	{"@r//devtools/buildozer", "r", "devtools/buildozer", "buildozer"},
	{"@r//base", "r", "base", "base"},
	{"@r//base:", "r", "base", "base"},
	{"@foo", "foo", "", "foo"},
	{":label", "", "", "label"},
	{"label", "", "", "label"},
	{"/abs/path/to/WORKSPACE:rule", "", "/abs/path/to/WORKSPACE", "rule"},
}

func TestParseLabel(t *testing.T) {
	for i, tt := range parseLabelTests {
		repo, pkg, rule := ParseLabel(tt.in)
		if repo != tt.repo || pkg != tt.pkg || rule != tt.rule {
			t.Errorf("%d. ParseLabel(%q) => (%q, %q, %q), want (%q, %q, %q)",
				i, tt.in, repo, pkg, rule, tt.repo, tt.pkg, tt.rule)
		}
	}
}

var shortenLabelTests = []struct {
	in     string
	pkg    string
	result string
}{
	{"//devtools/buildozer:rule", "devtools/buildozer", ":rule"},
	{"@//devtools/buildozer:rule", "devtools/buildozer", ":rule"},
	{"//devtools/buildozer:rule", "devtools", "//devtools/buildozer:rule"},
	{"//base:rule", "devtools", "//base:rule"},
	{"//base:base", "devtools", "//base"},
	{"//base", "base", ":base"},
	{"//devtools/buildozer:buildozer", "", "//devtools/buildozer"},
	{"@r//devtools/buildozer:buildozer", "devtools/buildozer", "@r//devtools/buildozer"},
	{"@r//devtools/buildozer", "devtools/buildozer", "@r//devtools/buildozer"},
	{"@r//devtools", "devtools", "@r//devtools"},
	{"@r:rule", "", "@r:rule"},
	{"@r", "", "@r"},
	{"@foo//:foo", "", "@foo"},
	{"@foo//devtools:foo", "", "@foo//devtools:foo"},
	{"@foo//devtools:foo", "devtools", "@foo//devtools:foo"},
	{"@foo//foo:foo", "", "@foo//foo"},
	{":local", "", ":local"},
	{"something else", "", "something else"},
	{"/path/to/file", "path/to", "/path/to/file"},
}

func TestShortenLabel(t *testing.T) {
	for i, tt := range shortenLabelTests {
		result := ShortenLabel(tt.in, tt.pkg)
		if result != tt.result {
			t.Errorf("%d. ShortenLabel(%q, %q) => %q, want %q",
				i, tt.in, tt.pkg, result, tt.result)
		}
	}
}

var labelsEqualTests = []struct {
	label1   string
	label2   string
	pkg      string
	expected bool
}{
	{"//devtools/buildozer:rule", "rule", "devtools/buildozer", true},
	{"//devtools/buildozer:rule", "rule:jar", "devtools", false},
}

func TestLabelsEqual(t *testing.T) {
	for i, tt := range labelsEqualTests {
		if got := LabelsEqual(tt.label1, tt.label2, tt.pkg); got != tt.expected {
			t.Errorf("%d. LabelsEqual(%q, %q, %q) => %v, want %v",
				i, tt.label1, tt.label2, tt.pkg, got, tt.expected)
		}
	}
}

var splitOnSpacesTests = []struct {
	in  string
	out []string
}{
	{"a", []string{"a"}},
	{"  abc def ", []string{"abc", "def"}},
	{`  abc\ def `, []string{"abc def"}},
	{"  abc def\nghi", []string{"abc", "def", "ghi"}},
}

func TestSplitOnSpaces(t *testing.T) {
	for i, tt := range splitOnSpacesTests {
		result := SplitOnSpaces(tt.in)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("%d. SplitOnSpaces(%q) => %q, want %q",
				i, tt.in, result, tt.out)
		}
	}
}

func TestInsertLoad(t *testing.T) {
	tests := []struct{ input, expected string }{
		{``, `load("location", "symbol")`},
		{`load("location", "symbol")`, `load("location", "symbol")`},
		{`load("location", "other", "symbol")`, `load("location", "other", "symbol")`},
		{`load("location", "other")`, `load("location", "other", "symbol")`},
		{
			`load("other loc", "symbol")`,
			`load("location", "symbol")
load("other loc", "symbol")`,
		},
	}

	for _, tst := range tests {
		bld, err := build.Parse("BUILD", []byte(tst.input))
		if err != nil {
			t.Error(err)
			continue
		}
		bld.Stmt = InsertLoad(bld.Stmt, "location", []string{"symbol"}, []string{"symbol"})
		got := strings.TrimSpace(string(build.Format(bld)))
		if got != tst.expected {
			t.Errorf("maybeInsertLoad(%s): got %s, expected %s", tst.input, got, tst.expected)
		}
	}
}

func TestReplaceLoad(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		location string
		from     []string
		to       []string
		expected string
	}{
		{
			name:     "add_symbol",
			input:    ``,
			location: "new_location",
			from:     []string{"symbol"},
			to:       []string{"symbol"},
			expected: `load("new_location", "symbol")`,
		},
		{
			name:     "replace_location",
			input:    `load("location", "symbol")`,
			location: "new_location",
			from:     []string{"symbol"},
			to:       []string{"symbol"},
			expected: `load("new_location", "symbol")`,
		},
		{
			name:     "replace_location_one_of_multiple",
			input:    `load("location", "other", "symbol")`,
			location: "new_location",
			from:     []string{"symbol"},
			to:       []string{"symbol"},
			expected: `load("location", "other")
load("new_location", "symbol")`,
		},
		{
			name:     "replace_location_alias",
			input:    `load("location", symbol = "other")`,
			location: "new_location",
			from:     []string{"symbol"},
			to:       []string{"symbol"},
			expected: `load("new_location", "symbol")`,
		},
		{
			name: "collapse_duplicate_symbol",
			input: `load("other loc", "symbol")
load("location", "symbol")`,
			location: "new_location",
			from:     []string{"symbol"},
			to:       []string{"symbol"},
			expected: `load("new_location", "symbol")`,
		},
		{
			name:     "replace_multiple_same_location",
			input:    `load("location", "symbol_a", "symbol_b", "symbol_c")`,
			location: "new_location",
			from:     []string{"symbol_a", "symbol_b", "symbol_c"},
			to:       []string{"symbol_a", "symbol_b", "symbol_c"},
			expected: `load("new_location", "symbol_a", "symbol_b", "symbol_c")`,
		},
		{
			name:     "replace_multiple_same_location_out_of_order",
			input:    `load("location", "symbol_a", "symbol_b", "symbol_c")`,
			location: "new_location",
			from:     []string{"symbol_c", "symbol_a", "symbol_b"},
			to:       []string{"symbol_c", "symbol_a", "symbol_b"},
			expected: `load("new_location", "symbol_a", "symbol_b", "symbol_c")`,
		},
		{
			name:     "replace_multiple_same_location_partial",
			input:    `load("location", "symbol_a", "symbol_b", "symbol_c")`,
			location: "new_location",
			from:     []string{"symbol_a", "symbol_b"},
			to:       []string{"symbol_a", "symbol_b"},
			expected: `load("location", "symbol_c")
load("new_location", "symbol_a", "symbol_b")`,
		},
		{
			name:     "replace_multiple_same_location_partial_out_of_order",
			input:    `load("location", "symbol_a", "symbol_b", "symbol_c")`,
			location: "new_location",
			from:     []string{"symbol_b", "symbol_a"},
			to:       []string{"symbol_b", "symbol_a"},
			expected: `load("location", "symbol_c")
load("new_location", "symbol_a", "symbol_b")`,
		},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			bld, err := build.Parse("BUILD", []byte(tst.input))
			if err != nil {
				t.Error(err)
				return
			}
			bld.Stmt = ReplaceLoad(bld.Stmt, tst.location, tst.from, tst.to)
			got := strings.TrimSpace(string(build.Format(bld)))
			if got != tst.expected {
				t.Errorf("ReplaceLoad(%s): got %s, expected %s", tst.input, got, tst.expected)
			}
		})
	}
}

func TestAddValueToListAttribute(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`rule(name="rule")`, `rule(name="rule", attr=["foo"])`},
		{`rule(name="rule", attr=["foo"])`, `rule(name="rule", attr=["foo"])`},
		{`rule(name="rule", attr=IDENT)`, `rule(name="rule", attr=IDENT+["foo"])`},
		{`rule(name="rule", attr=["foo"] + IDENT)`, `rule(name="rule", attr=["foo"] + IDENT)`},
		{`rule(name="rule", attr=["bar"] + IDENT)`, `rule(name="rule", attr=["bar", "foo"] + IDENT)`},
		{`rule(name="rule", attr=IDENT + ["foo"])`, `rule(name="rule", attr=IDENT + ["foo"])`},
		{`rule(name="rule", attr=IDENT + ["bar"])`, `rule(name="rule", attr=IDENT + ["bar", "foo"])`},
	}

	for _, tst := range tests {
		bld, err := build.Parse("BUILD", []byte(tst.input))
		if err != nil {
			t.Error(err)
			continue
		}
		rule := bld.RuleAt(1)
		AddValueToListAttribute(rule, "attr", "", &build.StringExpr{Value: "foo"}, nil)
		got := strings.TrimSpace(string(build.Format(bld)))

		wantBld, err := build.Parse("BUILD", []byte(tst.expected))
		if err != nil {
			t.Error(err)
			continue
		}
		want := strings.TrimSpace(string(build.Format(wantBld)))
		if got != want {
			t.Errorf("AddValueToListAttribute(%s): got %s, expected %s", tst.input, got, want)
		}
	}
}

func TestSelectListsIntersection(t *testing.T) {
	tests := []struct {
		input    string
		expected []build.Expr
	}{
		{`rule(
			name = "rule",
			attr = select()
		)`, nil},
		{`rule(
			name = "rule",
			attr = select({})
		)`, nil},
		{`rule(
			name = "rule",
			attr = select(CONFIGS)
		)`, nil},
		{`rule(
			name = "rule",
			attr = select({
				"config": "string",
				"DEFAULT": "default"
			})
		)`, nil},
		{`rule(
			name = "rule",
			attr = select({
				"config": LIST,
				"DEFAULT": DEFAULT
			})
		)`, nil},
		{`rule(
			name = "rule",
			attr = select({
				"config": ":1 :2 :3".split(" "),
				"DEFAULT": ":2 :3".split(" ")
			})
		)`, nil},
		{`rule(
			name = "rule",
			attr = select({
				"config1": [":1"],
				"config2": [":2"],
				"DEFAULT": []
			})
		)`, []build.Expr{}},
		{`rule(
			name = "rule",
			attr = select({
				"config1": [],
				"config2": [":1"],
				"DEFAULT": [":1"]
			})
		)`, []build.Expr{}},
		{`rule(
			name = "rule",
			attr = select({
				"config1": [":1", ":2", ":3"],
				"config2": [":2"],
				"config3": [":2", ":3"],
				"DEFAULT": [":1", ":2"]
			})
		)`, []build.Expr{&build.StringExpr{Value: ":2"}}},
		{`rule(
			name = "rule",
			attr = select({
				"config1": [":4", ":3", ":1", ":5", ":2", ":6"],
				"config2": [":5", ":2", ":6", ":1"],
				"config3": [":1", ":2", ":3", ":4", ":5", ":6"],
				"config4": [":2", ":1"],
				"DEFAULT": [":3", ":4", ":1", ":2"]
			})
		)`, []build.Expr{&build.StringExpr{Value: ":1"}, &build.StringExpr{Value: ":2"}}},
	}

	for _, tst := range tests {
		bld, err := build.Parse("BUILD", []byte(tst.input))
		if err != nil {
			t.Error(err)
			continue
		}
		rule := bld.RuleAt(1)

		got := SelectListsIntersection(rule.Attr("attr").(*build.CallExpr), "")
		errStr := fmt.Sprintf("TestSelectListsIntersection(%s): got %s, expected %s", tst.input, got, tst.expected)

		if len(got) != len(tst.expected) {
			t.Error(errStr)
		}

		for i := range got {
			if got[i].(*build.StringExpr).Value != tst.expected[i].(*build.StringExpr).Value {
				t.Error(errStr)
			}
		}
	}
}

func TestRemoveEmptySelectsAndConcatLists(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`rule(
			name = "rule",
			attr = select({
				"config1": [],
				"config2": [],
				"DEFAULT": []
			})
		)`, `rule(
			name = "rule",
			attr = []
		)`},
		{`rule(
			name = "rule",
			attr = select({}) + select() + select({
				"config1": [],
				"config2": [],
				"DEFAULT": []
			})
		)`, `rule(
			name = "rule",
			attr = []
		)`},
		{`rule(
			name = "rule",
			attr = select({
				"config1": [],
				"config2": [],
				"DEFAULT": []
			}) + select(CONFIGS)
		)`, `rule(
			name = "rule",
			attr = select(CONFIGS)
		)`},
		{`rule(
			name = "rule",
			attr = [":1"] + select({
				"config1": [],
				"config2": [],
				"DEFAULT": []
			}) + [":2"]
		)`, `rule(
			name = "rule",
			attr = [":1", ":2"]
		)`},
		{`rule(
			name = "rule",
			attr = [":1"] + select({
				"config1": [],
				"config2": [],
				"DEFAULT": []
			}) + LIST + [":2"]
		)`, `rule(
			name = "rule",
			attr = [":1"] + LIST + [":2"]
		)`},
		{`rule(
			name = "rule",
			attr = [":1"] + [":2", ":3"] + select({
				"config1": [":4"],
				"config2": [],
				"DEFAULT": []
			}) + []
		)`, `rule(
			name = "rule",
			attr = [":1", ":2", ":3"] + select({
				"config1": [":4"],
				"config2": [],
				"DEFAULT": []
			})
		)`},
		{`rule(
			name = "rule",
			attr = [":1"] + [":2", ":3"] + select({
				"config1": [":4"],
				"config2": [],
				"DEFAULT": []
			}) + [] + select({
				"config": LIST,
				"DEFAULT": DEFAULT,
			})
		)`, `rule(
			name = "rule",
			attr = [":1", ":2", ":3"] + select({
				"config1": [":4"],
				"config2": [],
				"DEFAULT": []
			}) + select({
				"config": LIST,
				"DEFAULT": DEFAULT,
			})
		)`},
	}

	for _, tst := range tests {
		bld, err := build.Parse("BUILD", []byte(tst.input))
		if err != nil {
			t.Error(err)
			continue
		}
		rule := bld.RuleAt(1)
		rule.SetAttr("attr", RemoveEmptySelectsAndConcatLists(rule.Attr("attr")))
		got := strings.TrimSpace(string(build.Format(bld)))

		wantBld, err := build.Parse("BUILD", []byte(tst.expected))
		if err != nil {
			t.Error(err)
			continue
		}
		want := strings.TrimSpace(string(build.Format(wantBld)))
		if got != want {
			t.Errorf("RemoveEmptySelectsAndConcatLists(%s):\n got: %s,\n expected: %s", tst.input, got, want)
		}
	}
}

func TestResolveAttr(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`rule(
			name = "rule",
			attr = select({
				"config1": [":1"],
				"config2": [":1"],
				"DEFAULT": [":1"]
			})
		)`, `rule(
			name = "rule",
			attr = [":1"]
		)`},
		{`rule(
			name = "rule",
			attr = select({
				"config1": [":1"],
				"config2": [":1"],
				"DEFAULT": [":1"]
			}) + select() + select({})
		)`, `rule(
			name = "rule",
			attr = [":1"]
		)`},
		{`rule(
			name = "rule",
			attr = select({
				"config1": [":1"],
				"config2": [":1"],
				"DEFAULT": [":1"]
			}) + LIST
		)`, `rule(
			name = "rule",
			attr = LIST + [":1"]
		)`},
		{`rule(
			name = "rule",
			attr = select({
				"config1": [":1"],
				"config2": [":1"],
				"DEFAULT": [":1"]
			}) + select({
				"config": LIST,
				"DEFAULT": DEFAULT
			}) + select({
				"config": ":2 :3".split(" "),
				"DEFAULT": ":3".split(" ")
			})
		)`, `rule(
			name = "rule",
			attr = select({
				"config": LIST,
				"DEFAULT": DEFAULT
			}) + select({
				"config": ":2 :3".split(" "),
				"DEFAULT": ":3".split(" ")
			}) + [":1"]
		)`},
		{`rule(
			name = "rule",
			attr = [":1"] + select({
				"config1": [":2"],
				"config2": [":2"],
				"DEFAULT": [":2"]
			}) + [":3"] + select({
				"config1": [":4", ":2"],
				"DEFAULT": [":2"]
			})
		)`, `rule(
			name = "rule",
			attr = [":1", ":2", ":3"] + select({
				"config1": [":4"],
				"DEFAULT": []
			})
		)`},
		{`rule(
			name = "rule",
			attr = [":1"] + select({
				"config1": [":2"],
				"config2": [":2"],
				"DEFAULT": [":2"]
			}) + [":3"] + select({
				"config1": [":4", ":2"],
				"DEFAULT": [":4", ":2"]
			})
		)`, `rule(
			name = "rule",
			attr = [":1", ":2", ":4", ":3"]
		)`},
	}

	for _, tst := range tests {
		bld, err := build.Parse("BUILD", []byte(tst.input))
		if err != nil {
			t.Error(err)
			continue
		}
		rule := bld.RuleAt(1)
		ResolveAttr(rule, "attr", "")
		got := strings.TrimSpace(string(build.Format(bld)))

		wantBld, err := build.Parse("BUILD", []byte(tst.expected))
		if err != nil {
			t.Error(err)
			continue
		}
		want := strings.TrimSpace(string(build.Format(wantBld)))
		if got != want {
			t.Errorf("ResolveAttr(%s):\n got: %s\n expected: %s", tst.input, got, want)
		}
	}
}

func TestListSubstitute(t *testing.T) {
	tests := []struct {
		desc, input, oldPattern, newTemplate, want string
	}{
		{
			desc:        "no_match",
			input:       `["abc"]`,
			oldPattern:  `!!`,
			newTemplate: `xx`,
			want:        `["abc"]`,
		}, {
			desc:        "full_match",
			input:       `["abc"]`,
			oldPattern:  `.*`,
			newTemplate: `xx`,
			want:        `["xx"]`,
		}, {
			desc:        "partial_match",
			input:       `["abcde"]`,
			oldPattern:  `bcd`,
			newTemplate: `xyz`,
			want:        `["axyze"]`,
		}, {
			desc:        "number_group",
			input:       `["abcde"]`,
			oldPattern:  `a(bcd)`,
			newTemplate: `$1 $1`,
			want:        `["bcd bcde"]`,
		}, {
			desc:        "name_group",
			input:       `["abcde"]`,
			oldPattern:  `a(?P<x>bcd)`,
			newTemplate: `$x $x`,
			want:        `["bcd bcde"]`,
		},
	}

	for _, tst := range tests {
		t.Run(tst.desc, func(t *testing.T) {
			f, err := build.ParseBuild("BUILD", []byte(tst.input))
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			lst := f.Stmt[0]
			oldRegexp, err := regexp.Compile(tst.oldPattern)
			if err != nil {
				t.Fatalf("error compiling regexp %q: %v", tst.oldPattern, err)
			}
			ListSubstitute(lst, oldRegexp, tst.newTemplate)
			if got := build.FormatString(lst); got != tst.want {
				t.Errorf("ListSubstitute(%q, %q, %q) = %q ; want %q", tst.input, tst.oldPattern, tst.newTemplate, got, tst.want)
			}
		})
	}
}

func compareKeyValue(a, b build.Expr) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	aKeyVal := a.(*build.KeyValueExpr)
	bKeyVal := b.(*build.KeyValueExpr)
	return aKeyVal.Key.(*build.StringExpr).Value == bKeyVal.Key.(*build.StringExpr).Value &&
		aKeyVal.Value.(*build.StringExpr).Value == bKeyVal.Value.(*build.StringExpr).Value
}

func TestDictionaryDelete(t *testing.T) {
	tests := []struct {
		input, expected string
		expectedReturn  build.Expr
	}{
		{
			`rule(attr = {"deletekey": "value"})`,
			`rule(attr = {})`,
			&build.KeyValueExpr{
				Key:   &build.StringExpr{Value: "deletekey"},
				Value: &build.StringExpr{Value: "value"},
			},
		}, {
			`rule(attr = {"nodeletekey": "value", "deletekey": "value"})`,
			`rule(attr = {"nodeletekey": "value"})`,
			&build.KeyValueExpr{
				Key:   &build.StringExpr{Value: "deletekey"},
				Value: &build.StringExpr{Value: "value"},
			},
		}, {
			`rule(attr = {"nodeletekey": "value"})`,
			`rule(attr = {"nodeletekey": "value"})`,
			nil,
		},
	}

	for _, tst := range tests {
		bld, err := build.ParseBuild("BUILD", []byte(tst.input))
		if err != nil {
			t.Error(err)
			continue
		}
		rule := bld.RuleAt(1)
		dict := rule.Call.List[0].(*build.AssignExpr).RHS.(*build.DictExpr)
		returnVal := DictionaryDelete(dict, "deletekey")
		got := strings.TrimSpace(string(build.Format(bld)))
		wantBld, err := build.Parse("BUILD", []byte(tst.expected))
		if err != nil {
			t.Error(err)
			continue
		}
		want := strings.TrimSpace(string(build.Format(wantBld)))
		if got != want {
			t.Errorf("TestDictionaryDelete(%s): got %s, expected %s", tst.input, got, want)
		}
		if !compareKeyValue(returnVal, tst.expectedReturn) {
			t.Errorf("TestDictionaryDelete(%s): returned %v, expected %v", tst.input, returnVal, tst.expectedReturn)
		}
	}
}

func TestPackageDeclaration(t *testing.T) {
	tests := []struct{ input, expected string }{
		{``, `package(attr = "val")`},
		{`"""Docstring."""

load(":path.bzl", "x")

# package() comes here

x = 2`,
			`"""Docstring."""

load(":path.bzl", "x")

# package() comes here

package(attr = "val")

x = 2`,
		},
	}

	for _, tst := range tests {
		bld, err := build.Parse("BUILD", []byte(tst.input))
		if err != nil {
			t.Error(err)
			continue
		}
		pkg := PackageDeclaration(bld)
		pkg.SetAttr("attr", &build.StringExpr{Value: "val"})
		got := strings.TrimSpace(string(build.Format(bld)))
		want := strings.TrimSpace(tst.expected)

		if got != want {
			t.Errorf("TestPackageDeclaration: got:\n%s\nexpected:\n%s", got, want)
		}
	}
}

type testCase struct {
	inputRoot, inputTarget                       string
	expectedBuildFile, expectedPkg, expectedRule string
}

func runTestInterpretLabelForWorkspaceLocation(t *testing.T, buildFileName string) {
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "a", "b"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "WORKSPACE"), nil, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, buildFileName), nil, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "a", buildFileName), nil, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "a", "b", buildFileName), nil, 0755); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []testCase{
		{tmp, "//", filepath.Join(tmp, buildFileName), "", "."},
		{tmp, "//a", filepath.Join(tmp, "a", buildFileName), "a", "a"},
		{tmp, "//a:a", filepath.Join(tmp, "a", buildFileName), "a", "a"},
		{tmp, "//a/b", filepath.Join(tmp, "a", "b", buildFileName), "a/b", "b"},
		{tmp, "//a/b:b", filepath.Join(tmp, "a", "b", buildFileName), "a/b", "b"},
	} {
		buildFile, _, pkg, rule := InterpretLabelForWorkspaceLocation(tc.inputRoot, tc.inputTarget)
		if buildFile != tc.expectedBuildFile || pkg != tc.expectedPkg || rule != tc.expectedRule {
			t.Errorf("InterpretLabelForWorkspaceLocation(%q, %q) = %q, %q, %q; want %q, %q, %q", tc.inputRoot, tc.inputTarget, buildFile, pkg, rule, tc.expectedBuildFile, tc.expectedPkg, tc.expectedRule)
		}
	}
}

func TestInterpretLabelForWorkspaceLocation(t *testing.T) {
	runTestInterpretLabelForWorkspaceLocation(t, "BUILD")
	runTestInterpretLabelForWorkspaceLocation(t, "BUILD.bazel")
}

func TestFindRuleByName(t *testing.T) {
	input := `load("foo.bzl", "bar")
my_rule(
  name="r1",
	srcs=["a.txt"]
)

a = my_macro(
	name="r2",
)
`

	bld, err := build.Parse("BUILD", []byte(input))
	if err != nil {
		t.Error(err)
		return
	}

	testCases := []struct {
		name       string
		shouldFind bool
	}{
		{"r1", true}, {"r2", true}, {"r3", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rule := FindRuleByName(bld, tc.name)
			found := rule != nil
			if found != tc.shouldFind {
				t.Errorf("FindRuleByName(%q): found = %v, want %v", tc.name, found, tc.shouldFind)
			}
		})
	}
}

func TestContainsDisableComment(t *testing.T) {
	testCases := []struct {
		comment string
		warning string
		want    bool
	}{
		{"# buildifier: disable=warning1", "warning1", true},
		{"# buildifier: disable=warning1", "warning2", false},
		{"# buildifier: disable=warning1,warning2", "warning1", true},
		{"# buildifier: disable=warning1,warning2", "warning2", true},
		{"# buildifier: disable=warning1,warning2", "warning3", false},
		{"# buildifier: disable=warning1, warning2", "warning2", true},
		{"# buildifier: disable=warning1,warning2 (explanation)", "warning1", true},
		{"# buildifier: disable=warn", "warning1", false},
		{"# buildifier: disable=warning1", "warn", true},
		{"# buildifier: disable=warning1,warning2 (warning3)", "warning3", true},
		{"# buildozer: disable=warning1,warning2", "warning2", true},
		{"# othertool: disable=warning1,warning2", "warning2", false},
	}

	for i, tc := range testCases {
		expr := &build.Ident{
			Comments: build.Comments{
				Before: []build.Comment{{Token: tc.comment}},
			},
		}
		if got := ContainsDisableComment(expr, tc.warning); got != tc.want {
			t.Errorf("case %d: ContainsDisableComment(%q, %q) = %v, want %v", i, tc.comment, tc.warning, got, tc.want)
		}
	}
}
