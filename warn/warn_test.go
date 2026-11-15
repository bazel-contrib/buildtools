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
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/bazel-contrib/buildtools/v10/build"
	"github.com/bazel-contrib/buildtools/v10/testutils"
)

const (
	scopeBuild       = build.TypeBuild
	scopeBzl         = build.TypeBzl
	scopeWorkspace   = build.TypeWorkspace
	scopeDefault     = build.TypeDefault
	scopeModule      = build.TypeModule
	scopeRepo        = build.TypeRepo
	scopeVendor      = build.TypeVendor
	scopeEverywhere  = scopeBuild | scopeBzl | scopeWorkspace | scopeDefault | scopeModule | scopeRepo | scopeVendor
	scopeBazel       = scopeBuild | scopeBzl | scopeWorkspace | scopeModule | scopeRepo | scopeVendor
	scopeDeclarative = scopeBuild | scopeWorkspace | scopeModule | scopeRepo | scopeVendor
)

// A global FileReader object that can be used by tests. If a test redefines it it must
// reset it when it finishes.
var testFileReader *FileReader

// A global variable containing package name for test cases. Can be overwritten
// but must be reset when the test finishes.
var testPackage string = "test/package"

// fileReaderRequests is used by tests to check which files have actually been requested by testFileReader
var fileReaderRequests []string

func setUpFileReader(data map[string]string) (cleanup func()) {
	readFile := func(filename string) ([]byte, error) {
		fileReaderRequests = append(fileReaderRequests, filename)
		if contents, ok := data[filename]; ok {
			return []byte(contents), nil
		}
		return nil, fmt.Errorf("file not found")
	}
	testFileReader = NewFileReader(readFile)
	fileReaderRequests = nil
	// The cached mapping depends on the file reader, so we need to reset it as well.
	moduleToApparentRepoName = nil

	return func() {
		// Tear down
		testFileReader = nil
		fileReaderRequests = nil
	}
}

func setUpTestPackage(name string) (cleanup func()) {
	oldName := testPackage
	testPackage = name

	return func() {
		// Tear down
		testPackage = oldName
	}
}

func getFilename(fileType build.FileType) string {
	switch fileType {
	case build.TypeBuild:
		return "BUILD"
	case build.TypeWorkspace:
		return "WORKSPACE"
	case build.TypeBzl:
		return "test_file.bzl"
	case build.TypeModule:
		return "MODULE.bazel"
	case build.TypeRepo:
		return "REPO.bazel"
	case build.TypeVendor:
		return "VENDOR.bazel"
	default:
		return "test_file.strlrk"
	}
}

func getFileForTest(input string, fileType build.FileType) *build.File {
	input = strings.TrimLeft(input, "\n")
	filename := getFilename(fileType)
	file, err := build.Parse(testPackage+"/"+filename, []byte(input))
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}
	file.Pkg = testPackage
	file.Label = filename
	file.WorkspaceRoot = "/home/users/foo/bar"
	return file
}

func getFindings(categories, input string, fileType build.FileType) []*Finding {
	file := getFileForTest(input, fileType)
	return FileWarnings(file, strings.Split(categories, ","), nil, ModeWarn, testFileReader)
}

func compareFindings(t *testing.T, categories, input string, expected []string, scope, fileType build.FileType) {
	// If scope doesn't match the file type, no warnings are expected
	if scope&fileType == 0 {
		expected = []string{}
	}

	findings := getFindings(categories, input, fileType)
	// We ensure that there is the expected number of warnings.
	// At the moment, we check only the line numbers.
	if len(expected) != len(findings) {
		t.Errorf("Input: %s", input)
		t.Errorf("number of matches: %d, want %d", len(findings), len(expected))
		for _, e := range expected {
			t.Errorf("expected: %s", e)
		}
		for _, f := range findings {
			t.Errorf("got: %d: %s", f.Start.Line, f.Message)
		}
		return
	}
	for i := range findings {
		msg := fmt.Sprintf(":%d: %s", findings[i].Start.Line, findings[i].Message)
		if !strings.Contains(msg, expected[i]) {
			t.Errorf("Input: %s", input)
			t.Errorf("got:  `%s`,\nwant: `%s`", msg, expected[i])
		}
	}
}

// checkFix makes sure that fixed file contents match the expected output
func checkFix(t *testing.T, categories, input, expected string, scope, fileType build.FileType) {
	// If scope doesn't match the file type, no changes are expected
	if scope&fileType == 0 {
		expected = input
	}

	file := getFileForTest(input, fileType)
	goldenFile := getFileForTest(expected, fileType)

	FixWarnings(file, strings.Split(categories, ","), false, testFileReader)
	have := build.Format(file)
	want := build.Format(goldenFile)
	if !bytes.Equal(have, want) {
		t.Errorf("fixed a test (type %s) incorrectly:\ninput:\n%s\ndiff (-expected, +ours)\n",
			fileType, input)
		testutils.Tdiff(t, want, have)
	}
}

// checkFix makes sure that the file contents don't change if a fix is not requested
// (i.e. the warning functions have no side effects modifying the AST)
func checkNoFix(t *testing.T, categories, input string, fileType build.FileType) {
	file := getFileForTest(input, fileType)
	formatted := build.Format(file)

	// No fixes expected
	FileWarnings(file, strings.Split(categories, ","), nil, ModeWarn, testFileReader)
	fixed := build.FormatWithoutRewriting(file)

	if !bytes.Equal(formatted, fixed) {
		t.Errorf("Modified a file (type %s) while getting warnings:\ninput:\n%s\ndiff (-before, +after)\n",
			fileType, input)
		testutils.Tdiff(t, formatted, fixed)
	}
}

func checkFindings(t *testing.T, category, input string, expected []string, scope build.FileType) {
	// The same as checkFindingsAndFix but ensure that fixes don't change the file (except for formatting)
	checkFindingsAndFix(t, category, input, input, expected, scope)
}

func checkFindingsAndFix(t *testing.T, categories, input, output string, expected []string, scope build.FileType) {
	fileTypes := []build.FileType{
		build.TypeDefault,
		build.TypeBuild,
		build.TypeWorkspace,
		build.TypeBzl,
		build.TypeModule,
	}

	for _, fileType := range fileTypes {
		compareFindings(t, categories, input, expected, scope, fileType)
		checkFix(t, categories, input, output, scope, fileType)
		checkFix(t, categories, output, output, scope, fileType)
		checkNoFix(t, categories, input, fileType)
	}
}

func TestCalculateDifference(t *testing.T) {
	tests := []struct {
		before      string
		after       string
		start       int
		end         int
		replacement string
	}{
		{
			before:      "asdf",
			after:       "asxydf",
			start:       2,
			end:         2,
			replacement: "xy",
		},
		{
			before:      "asxydf",
			after:       "asdf",
			start:       2,
			end:         4,
			replacement: "",
		},
		{
			before:      "asxydf",
			after:       "asztdf",
			start:       2,
			end:         4,
			replacement: "zt",
		},
		{
			before:      "",
			after:       "foobar",
			start:       0,
			end:         0,
			replacement: "foobar",
		},
		{
			before:      "foobar",
			after:       "",
			start:       0,
			end:         6,
			replacement: "",
		},
		{
			before:      "qwerty",
			after:       "asdfgh",
			start:       0,
			end:         6,
			replacement: "asdfgh",
		},
		{
			before:      "aa",
			after:       "aaaa",
			start:       2,
			end:         2,
			replacement: "aa",
		},
		{
			before:      "aaaa",
			after:       "aa",
			start:       2,
			end:         4,
			replacement: "",
		},
		{
			before:      "abc",
			after:       "abdbc",
			start:       2,
			end:         2,
			replacement: "db",
		},
		{
			before:      "abdbc",
			after:       "abc",
			start:       2,
			end:         4,
			replacement: "",
		},
	}

	for _, tc := range tests {
		before := []byte(tc.before)
		after := []byte(tc.after)

		start, end, replacement := calculateDifference(&before, &after)

		if start != tc.start || end != tc.end || replacement != tc.replacement {
			t.Errorf("Wrong difference for %q and %q: want %d, %d, %q, got %d, %d, %q",
				tc.before, tc.after, tc.start, tc.end, tc.replacement, start, end, replacement)
		}
	}
}

func TestSuggestions(t *testing.T) {
	// Suggestions are not generated by individual warning functions but by the warnings framework.

	contents := `foo()

attr.bar(name = "bar", cfg = "data")

attr.baz("baz", cfg = "data")
`
	f, err := build.ParseBzl("file.bzl", []byte(contents))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	findings := FileWarnings(f, []string{"attr-cfg"}, nil, ModeSuggest, testFileReader)
	want := []struct {
		start       int
		end         int
		replacement string
	}{
		{
			start:       28,
			end:         42,
			replacement: "",
		},
		{
			start:       59,
			end:         73,
			replacement: "",
		},
	}

	if len(findings) != len(want) {
		t.Errorf("Expected %d findings, got %d", len(want), len(findings))
	}

	for i, f := range findings {
		w := want[i]
		if f.Replacement == nil {
			t.Errorf("No replacement for finding %d", i)
		}
		r := f.Replacement
		if r.Start != w.start || r.End != w.end || r.Content != w.replacement {
			t.Errorf("Wrong replacement #%d, want %d, %d, %q, got %d, %d, %q",
				i, w.start, w.end, w.replacement, r.Start, r.End, r.Content)
		}
	}
}

func TestDisabledWarning(t *testing.T) {
	contents := `foo()

# buildifier: disable=depset-iteration
for x in depset([1, 2, 3]):
    print(x)  # buildozer: disable=print

for y in "foobar":  # buildozer: disable=string-iteration
    # buildifier: disable=no-effect
    y

# buildifier: disable=duplicated-name-2
cc_library(
   name = "foo",  # buildifier: disable=duplicated-name-1
)

# buildifier: disable=skylark-comment
# some comment mentioning skylark

def foo(x, y): # buildifier: disable=unused-variable
    pass
`

	f, err := build.ParseBzl("file.bzl", []byte(contents))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	tests := []struct {
		start    int
		end      int
		category string
	}{
		{
			start:    3,
			end:      5,
			category: "depset-iteration",
		},
		{
			start:    5,
			end:      5,
			category: "print",
		},
		{
			start:    7,
			end:      7,
			category: "string-iteration",
		},
		{
			start:    8,
			end:      9,
			category: "no-effect",
		},
		{
			start:    13,
			end:      13,
			category: "duplicated-name-1",
		},
		{
			start:    11,
			end:      14,
			category: "duplicated-name-2",
		},
		{
			start:    16,
			end:      17,
			category: "skylark-comment",
		},
		{
			start:    19,
			end:      19,
			category: "unused-variable",
		},
	}

	linesCount := strings.Count(contents, "\n")

	for _, tc := range tests {
		for line := 1; line <= linesCount; line++ {
			disabled := DisabledWarning(f, line, tc.category)
			shouldBeDisabled := line >= tc.start && line <= tc.end
			if disabled != shouldBeDisabled {
				t.Errorf("Wrong disabled status for the category %q, want %t, got %t", tc.category, shouldBeDisabled, disabled)
			}
		}
	}
}

func TestDisabledWarningMultiple(t *testing.T) {
	contents := `foo()

# buildifier: disable=depset-iteration,print
for x in depset([1, 2, 3]):
    print(x)

# buildozer: disable=string-iteration, no-effect (with explanation)
for y in "foobar":
    y
`
	f, err := build.ParseBzl("file.bzl", []byte(contents))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	tests := []struct {
		start    int
		end      int
		category string
	}{
		{
			start:    3,
			end:      5,
			category: "depset-iteration",
		},
		{
			start:    3,
			end:      5,
			category: "print",
		},
		{
			start:    7,
			end:      9,
			category: "string-iteration",
		},
		{
			start:    7,
			end:      9,
			category: "no-effect",
		},
	}

	linesCount := strings.Count(contents, "\n")

	for _, tc := range tests {
		for line := 1; line <= linesCount; line++ {
			disabled := DisabledWarning(f, line, tc.category)
			shouldBeDisabled := line >= tc.start && line <= tc.end
			if disabled != shouldBeDisabled {
				t.Errorf("Wrong disabled status for category %q at line %d: want %t, got %t", tc.category, line, shouldBeDisabled, disabled)
			}
		}
	}
}
