/*
Copyright 2025 Google LLC

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
	"testing"

	"github.com/bazelbuild/buildtools/build"
	"github.com/google/go-cmp/cmp"
)

func mustFindDefStatement(f *build.File, name string) *build.DefStmt {
	for _, stmt := range f.Stmt {
		if def, ok := stmt.(*build.DefStmt); ok {
			if def.Name == name {
				return def
			}
		}
	}
	panic(fmt.Sprintf("unable to find def statement matching %q", name))
}

func mustFindCallExpression(f *build.File, name string) *build.CallExpr {
	for _, stmt := range f.Stmt {
		if call, ok := stmt.(*build.CallExpr); ok {
			if fnIdent, ok := call.X.(*build.Ident); ok && fnIdent.Name == name {
				return call
			}
		}
	}
	panic(fmt.Sprintf("unable to find call expression matching %q", name))
}

func fileReaderWithFiles(fileContents map[string]string) *FileReader {
	return NewFileReader(func(filename string) ([]byte, error) {
		return []byte(fileContents[filename]), nil
	})
}

func TestAnalyzeFn(t *testing.T) {
	defaultFilename := "BUILD"
	defaultPackage := "//package/path"
	defaultFilepath := fmt.Sprintf("%s/%s", defaultPackage, defaultFilename)
	tests := []struct {
		name                  string
		fileContents          map[string]string
		wantCanProduceTargets bool
		wantStackTrace        string
	}{
		{
			name: "non_macro",
			fileContents: map[string]string{
				defaultFilepath: `
def other_function():
  pass

def test_symbol():
  other_function()
`,
			},
			wantCanProduceTargets: false,
			wantStackTrace:        "",
		},
		{
			name: "with_infinite_recursion",
			fileContents: map[string]string{
				defaultFilepath: `
def first_function():
  test_symbol()

def test_symbol():
  first_function()
`,
			},
			wantCanProduceTargets: false,
			wantStackTrace:        "",
		},
		{
			name: "macro_within_single_file",
			fileContents: map[string]string{
				defaultFilepath: `
macro_def = macro()

def test_symbol():
  macro_def()
`,
			},
			wantCanProduceTargets: true,
			wantStackTrace: `//package/path:BUILD:5 macro_def
//package/path:BUILD:2 macro`,
		},
		{
			name: "macro_through_load_statement",
			fileContents: map[string]string{
				"package/other_path/file.bzl": `
imported_rule = rule()
`,
				defaultFilepath: `
load("//package/other_path:file.bzl", "imported_rule")

def test_symbol():
  imported_rule()
`,
			},
			wantCanProduceTargets: true,
			wantStackTrace: `//package/path:BUILD:5 imported_rule
package/other_path:file.bzl:2 imported_rule
package/other_path:file.bzl:2 rule`,
		},
		{
			name: "with_load_statements_prioritizes_local_statements",
			fileContents: map[string]string{
				"package/other_path/file.bzl": `
imported_rule = rule()
`,
				defaultFilepath: `
load("//package/other_path:file.bzl", "imported_rule")

def test_symbol():
  imported_rule()
  native.cc_library()
`,
			},
			wantCanProduceTargets: true,
			wantStackTrace:        `//package/path:BUILD:6 native.cc_library`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ma := NewMacroAnalyzer(fileReaderWithFiles(tc.fileContents))
			f := ma.fileReader.GetFile(defaultPackage, defaultFilename)
			macroDef := mustFindDefStatement(f, "test_symbol")

			report, err := ma.AnalyzeFn(f, macroDef)

			if err != nil {
				t.Errorf("Got unexpected error %s", err)
			}
			if diff := cmp.Diff(tc.wantCanProduceTargets, report.CanProduceTargets()); diff != "" {
				t.Errorf("AnalyzeFn.CanProduceTargets returned unexpected diff %s", diff)
			}
			if diff := cmp.Diff(tc.wantStackTrace, report.PrintableCallStack()); diff != "" {
				t.Errorf("AnalyzeFn.PrintableCallStack returned unexpected diff %s", diff)
			}
		})
	}
}

func TestAnalyzeFnCall(t *testing.T) {
	defaultFilename := "BUILD"
	defaultPackage := "//package/path"
	defaultFilepath := fmt.Sprintf("%s/%s", defaultPackage, defaultFilename)
	tests := []struct {
		name                  string
		fileContents          map[string]string
		wantCanProduceTargets bool
		wantStackTrace        string
	}{
		{
			name: "non_macro",
			fileContents: map[string]string{
				defaultFilepath: `
def test_symbol():
  pass

test_symbol()
`,
			},
			wantCanProduceTargets: false,
			wantStackTrace:        "",
		},
		{
			name: "with_infinite_recursion",
			fileContents: map[string]string{
				defaultFilepath: `
def test_symbol():
  second_function()

def second_function():
  test_symbol()

test_symbol()
`,
			},
			wantCanProduceTargets: false,
			wantStackTrace:        "",
		},
		{
			name: "macro_within_single_file",
			fileContents: map[string]string{
				defaultFilepath: `
macro_def = macro()

def test_symbol():
  macro_def()

test_symbol()
`,
			},
			wantCanProduceTargets: true,
			wantStackTrace: `//package/path:BUILD:5 macro_def
//package/path:BUILD:2 macro`,
		},
		{
			name: "macro_through_load_statement",
			fileContents: map[string]string{
				"package/other_path/file.bzl": `
imported_rule = rule()
`,
				defaultFilepath: `
load("//package/other_path:file.bzl", "imported_rule")

def test_symbol():
  imported_rule()

test_symbol()
`,
			},
			wantCanProduceTargets: true,
			wantStackTrace: `//package/path:BUILD:5 imported_rule
package/other_path:file.bzl:2 imported_rule
package/other_path:file.bzl:2 rule`,
		},
		{
			name: "with_load_statements_prioritizes_local_statements",
			fileContents: map[string]string{
				"package/other_path/file.bzl": `
imported_rule = rule()
`,
				defaultFilepath: `
load("//package/other_path:file.bzl", "imported_rule")

def test_symbol():
  imported_rule()
  macro()

test_symbol()
`,
			},
			wantCanProduceTargets: true,
			wantStackTrace:        `//package/path:BUILD:6 macro`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ma := NewMacroAnalyzer(fileReaderWithFiles(tc.fileContents))
			f := ma.fileReader.GetFile(defaultPackage, defaultFilename)
			call := mustFindCallExpression(f, "test_symbol")

			report, err := ma.AnalyzeFnCall(f, call)

			if err != nil {
				t.Errorf("Got unexpected error %s", err)
			}
			if diff := cmp.Diff(tc.wantCanProduceTargets, report.CanProduceTargets()); diff != "" {
				t.Errorf("AnalyzeFn.CanProduceTargets returned unexpected diff %s", diff)
			}
			if diff := cmp.Diff(tc.wantStackTrace, report.PrintableCallStack()); diff != "" {
				t.Errorf("AnalyzeFn.PrintableCallStack returned unexpected diff %s", diff)
			}
		})
	}
}
