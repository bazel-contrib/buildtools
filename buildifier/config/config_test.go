/*
Copyright 2022 Google LLC

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

package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ExampleNew() {
	c := New()
	fmt.Print(c.String())
	// Output:
	// {
	//   "type": "auto"
	// }
}

func ExampleExample() {
	c := Example()
	fmt.Print(c.String())
	// Output:
	// {
	//   "type": "auto",
	//   "mode": "fix",
	//   "lint": "fix",
	//   "warningsList": [
	//     "allowed-symbol-load-locations",
	//     "attr-applicable_licenses",
	//     "attr-cfg",
	//     "attr-license",
	//     "attr-licenses",
	//     "attr-non-empty",
	//     "attr-output-default",
	//     "attr-single-file",
	//     "build-args-kwargs",
	//     "bzl-visibility",
	//     "canonical-repository",
	//     "confusing-name",
	//     "constant-glob",
	//     "ctx-actions",
	//     "ctx-args",
	//     "deprecated-function",
	//     "depset-items",
	//     "depset-iteration",
	//     "depset-union",
	//     "dict-concatenation",
	//     "dict-method-named-arg",
	//     "duplicated-name",
	//     "external-path",
	//     "filetype",
	//     "function-docstring",
	//     "function-docstring-args",
	//     "function-docstring-header",
	//     "function-docstring-return",
	//     "git-repository",
	//     "http-archive",
	//     "integer-division",
	//     "keyword-positional-params",
	//     "list-append",
	//     "load",
	//     "module-docstring",
	//     "name-conventions",
	//     "native-android",
	//     "native-build",
	//     "native-cc-binary",
	//     "native-cc-common",
	//     "native-cc-debug-package-info",
	//     "native-cc-fdo-prefetch-hints",
	//     "native-cc-fdo-profile",
	//     "native-cc-import",
	//     "native-cc-info",
	//     "native-cc-library",
	//     "native-cc-memprof-profile",
	//     "native-cc-objc-import",
	//     "native-cc-objc-library",
	//     "native-cc-propeller-optimize",
	//     "native-cc-proto",
	//     "native-cc-shared-library",
	//     "native-cc-shared-library-hint-info",
	//     "native-cc-shared-library-info",
	//     "native-cc-test",
	//     "native-cc-toolchain",
	//     "native-cc-toolchain-suite",
	//     "native-java-binary",
	//     "native-java-common",
	//     "native-java-import",
	//     "native-java-info",
	//     "native-java-library",
	//     "native-java-lite-proto",
	//     "native-java-package-config",
	//     "native-java-plugin",
	//     "native-java-plugin-info",
	//     "native-java-proto",
	//     "native-java-runtime",
	//     "native-java-test",
	//     "native-java-toolchain",
	//     "native-package",
	//     "native-proto",
	//     "native-proto-common",
	//     "native-proto-info",
	//     "native-proto-lang-toolchain",
	//     "native-proto-lang-toolchain-info",
	//     "native-py",
	//     "native-sh-binary",
	//     "native-sh-library",
	//     "native-sh-test",
	//     "no-effect",
	//     "output-group",
	//     "overly-nested-depset",
	//     "package-name",
	//     "package-on-top",
	//     "positional-args",
	//     "print",
	//     "provider-params",
	//     "redefined-variable",
	//     "repository-name",
	//     "return-value",
	//     "rule-impl-return",
	//     "skylark-comment",
	//     "skylark-docstring",
	//     "string-iteration",
	//     "uninitialized",
	//     "unnamed-macro",
	//     "unreachable",
	//     "unsorted-dict-items",
	//     "unused-variable"
	//   ]
	// }
}

func ExampleFlagSet() {
	c := New()
	flags := c.FlagSet("buildifier", flag.ExitOnError)
	flags.VisitAll(func(f *flag.Flag) {
		fmt.Printf("%s: %s (%q)\n", f.Name, f.Usage, f.DefValue)
	})
	// Output:
	// add_tables: path to JSON file with custom table definitions which will be merged with the built-in tables ("")
	// allowsort: additional sort contexts to treat as safe ("")
	// buildifier_disable: list of buildifier rewrites to disable ("")
	// config: path to .buildifier.json config file ("")
	// d: alias for -mode=diff ("false")
	// diff_command: command to run when the formatting mode is diff (default uses the BUILDIFIER_DIFF, BUILDIFIER_MULTIDIFF, and DISPLAY environment variables to create the diff command) ("")
	// exclude: path pattern to exclude when finding starlark files recursively ("")
	// format: diagnostics format: text or json (default text) ("")
	// help: print usage information ("false")
	// lint: lint mode: off, warn, or fix (default off) ("")
	// mode: formatting mode: check, diff, or fix (default fix) ("")
	// multi_diff: the command specified by the -diff_command flag can diff multiple files in the style of tkdiff (default false) ("false")
	// path: assume BUILD file has this path relative to the workspace directory ("")
	// r: find starlark files recursively ("false")
	// tables: path to JSON file with custom table definitions which will replace the built-in tables ("")
	// type: Input file type: build (for BUILD files), bzl (for .bzl files), workspace (for WORKSPACE files), module (for MODULE.bazel files), default (for generic Starlark files) or auto (default, based on the filename) ("auto")
	// v: print verbose information to standard error ("false")
	// version: print the version of buildifier ("false")
	// warnings: comma-separated warnings used in the lint mode or "all" ("")
}

func ExampleFlagSet_parse() {
	c := New()
	flags := c.FlagSet("buildifier", flag.ExitOnError)
	flags.Parse([]string{
		"--add_tables=/path/to/add_tables.json",
		"--allowsort=proto_library.deps",
		"--allowsort=proto_library.srcs",
		"--buildifier_disable=unsafesort",
		"--config=/path/to/.buildifier.json",
		"-d",
		"--diff_command=diff",
		"--exclude=vendor/*",
		"--exclude=third_party/*",
		"--format=json",
		"--help",
		"--lint=fix",
		"--mode=fix",
		"--multi_diff=true",
		"--path=pkg/foo",
		"-r",
		"--tables=/path/to/tables.json",
		"--type=default",
		"-v",
		"--version",
		"--warnings=+print,-no-effect",
	})
	fmt.Println("help:", c.Help)
	fmt.Println("version:", c.Version)
	fmt.Println("configPath:", c.ConfigPath)
	fmt.Print(c.String())
	// Output:
	// help: true
	// version: true
	// configPath: /path/to/.buildifier.json
	// {
	//   "type": "default",
	//   "format": "json",
	//   "mode": "fix",
	//   "diffMode": true,
	//   "lint": "fix",
	//   "warnings": "+print,-no-effect",
	//   "recursive": true,
	//   "exclude": [
	//     "vendor/*",
	//     "third_party/*"
	//   ],
	//   "verbose": true,
	//   "diffCommand": "diff",
	//   "multiDiff": true,
	//   "tables": "/path/to/tables.json",
	//   "addTables": "/path/to/add_tables.json",
	//   "path": "pkg/foo",
	//   "buildifier_disable": [
	//     "unsafesort"
	//   ],
	//   "allowsort": [
	//     "proto_library.deps",
	//     "proto_library.srcs"
	//   ]
	// }
}

func TestValidate(t *testing.T) {
	for name, tc := range map[string]struct {
		options      string
		args         string
		wantErr      error
		wantMode     string   // optional
		wantLint     string   // optional
		wantWarnings []string // optional
	}{
		"mode not set":          {wantMode: "fix"},
		"mode check":            {options: "--mode=check", wantMode: "check"},
		"mode diff":             {options: "--mode=diff", wantMode: "diff"},
		"mode d":                {options: "-d", wantMode: "diff"},
		"mode d error":          {options: "--mode=diff -d", wantErr: fmt.Errorf("cannot specify both -d and -mode flags")},
		"mode fix":              {options: "--mode=fix", wantMode: "fix"},
		"mode print_if_changed": {options: "--mode=print_if_changed", wantMode: "print_if_changed"},
		"mode error":            {options: "--mode=foo", wantErr: fmt.Errorf("unrecognized mode foo; valid modes are check, diff, fix, print_if_changed")},
		"lint not set":          {wantLint: "off"},
		"lint off":              {options: "--lint=off", wantLint: "off"},
		"lint warn":             {options: "--lint=warn", wantLint: "warn"},
		"lint fix":              {options: "--lint=fix", wantLint: "fix"},
		"lint fix error":        {options: "--lint=fix --mode=check", wantErr: fmt.Errorf("--lint=fix is only compatible with --mode=fix")},
		"format mode error":     {options: "--mode=fix --format=text", wantErr: fmt.Errorf("cannot specify --format without --mode=check")},
		"format text":           {options: "--mode=check --format=text"},
		"format json":           {options: "--mode=check --format=json"},
		"format error":          {options: "--mode=check --format=foo", wantErr: fmt.Errorf("unrecognized format foo; valid types are text, json")},
		"type build":            {options: "--type=build"},
		"type bzl":              {options: "--type=bzl"},
		"type workspace":        {options: "--type=workspace"},
		"type default":          {options: "--type=default"},
		"type module":           {options: "--type=module"},
		"type auto":             {options: "--type=auto"},
		"type error":            {options: "--type=foo", wantErr: fmt.Errorf("unrecognized input type foo; valid types are build, bzl, workspace, default, module, auto")},
		"warnings all": {options: "--warnings=all", wantWarnings: []string{
			"allowed-symbol-load-locations",
			"attr-applicable_licenses",
			"attr-cfg",
			"attr-license",
			"attr-licenses",
			"attr-non-empty",
			"attr-output-default",
			"attr-single-file",
			"build-args-kwargs",
			"bzl-visibility",
			"canonical-repository",
			"confusing-name",
			"constant-glob",
			"ctx-actions",
			"ctx-args",
			"deprecated-function",
			"depset-items",
			"depset-iteration",
			"depset-union",
			"dict-concatenation",
			"dict-method-named-arg",
			"duplicated-name",
			"external-path",
			"filetype",
			"function-docstring",
			"function-docstring-args",
			"function-docstring-header",
			"function-docstring-return",
			"git-repository",
			"http-archive",
			"integer-division",
			"keyword-positional-params",
			"list-append",
			"load",
			"module-docstring",
			"name-conventions",
			"native-android",
			"native-build",
			"native-cc-binary",
			"native-cc-common",
			"native-cc-debug-package-info",
			"native-cc-fdo-prefetch-hints",
			"native-cc-fdo-profile",
			"native-cc-import",
			"native-cc-info",
			"native-cc-library",
			"native-cc-memprof-profile",
			"native-cc-objc-import",
			"native-cc-objc-library",
			"native-cc-propeller-optimize",
			"native-cc-proto",
			"native-cc-shared-library",
			"native-cc-shared-library-hint-info",
			"native-cc-shared-library-info",
			"native-cc-test",
			"native-cc-toolchain",
			"native-cc-toolchain-suite",
			"native-java-binary",
			"native-java-common",
			"native-java-import",
			"native-java-info",
			"native-java-library",
			"native-java-lite-proto",
			"native-java-package-config",
			"native-java-plugin",
			"native-java-plugin-info",
			"native-java-proto",
			"native-java-runtime",
			"native-java-test",
			"native-java-toolchain",
			"native-package",
			"native-proto",
			"native-proto-common",
			"native-proto-info",
			"native-proto-lang-toolchain",
			"native-proto-lang-toolchain-info",
			"native-py",
			"native-sh-binary",
			"native-sh-library",
			"native-sh-test",
			"no-effect",
			"output-group",
			"overly-nested-depset",
			"package-name",
			"package-on-top",
			"positional-args",
			"print",
			"provider-params",
			"redefined-variable",
			"repository-name",
			"return-value",
			"rule-impl-return",
			"skylark-comment",
			"skylark-docstring",
			"string-iteration",
			"uninitialized",
			"unnamed-macro",
			"unreachable",
			"unsorted-dict-items",
			"unused-variable",
		}},
		"warnings default": {options: "--warnings=default", wantWarnings: []string{
			"allowed-symbol-load-locations",
			"attr-applicable_licenses",
			"attr-cfg",
			"attr-license",
			"attr-licenses",
			"attr-non-empty",
			"attr-output-default",
			"attr-single-file",
			"build-args-kwargs",
			"bzl-visibility",
			"canonical-repository",
			"confusing-name",
			"constant-glob",
			"ctx-actions",
			"ctx-args",
			"deprecated-function",
			"depset-items",
			"depset-iteration",
			"depset-union",
			"dict-concatenation",
			"dict-method-named-arg",
			"duplicated-name",
			"external-path",
			"filetype",
			"function-docstring",
			"function-docstring-args",
			"function-docstring-header",
			"function-docstring-return",
			"git-repository",
			"http-archive",
			"integer-division",
			"keyword-positional-params",
			"list-append",
			"load",
			"module-docstring",
			"name-conventions",
			"native-android",
			"native-build",
			"native-cc-binary",
			"native-cc-common",
			"native-cc-debug-package-info",
			"native-cc-fdo-prefetch-hints",
			"native-cc-fdo-profile",
			"native-cc-import",
			"native-cc-info",
			"native-cc-library",
			"native-cc-memprof-profile",
			"native-cc-objc-import",
			"native-cc-objc-library",
			"native-cc-propeller-optimize",
			"native-cc-proto",
			"native-cc-shared-library",
			"native-cc-shared-library-hint-info",
			"native-cc-shared-library-info",
			"native-cc-test",
			"native-cc-toolchain",
			"native-cc-toolchain-suite",
			"native-java-binary",
			"native-java-common",
			"native-java-import",
			"native-java-info",
			"native-java-library",
			"native-java-lite-proto",
			"native-java-package-config",
			"native-java-plugin",
			"native-java-plugin-info",
			"native-java-proto",
			"native-java-runtime",
			"native-java-test",
			"native-java-toolchain",
			"native-package",
			"native-proto",
			"native-proto-common",
			"native-proto-info",
			"native-proto-lang-toolchain",
			"native-proto-lang-toolchain-info",
			"native-py",
			"native-sh-binary",
			"native-sh-library",
			"native-sh-test",
			"no-effect",
			"output-group",
			"overly-nested-depset",
			"package-name",
			"package-on-top",
			"positional-args",
			"print",
			"provider-params",
			"redefined-variable",
			"repository-name",
			"return-value",
			"rule-impl-return",
			"skylark-comment",
			"skylark-docstring",
			"string-iteration",
			"uninitialized",
			"unnamed-macro",
			"unreachable",
			// "unsorted-dict-items",
			"unused-variable",
		}},
		"warnings plus/minus": {options: "--warnings=+unsorted-dict-items,-print,-deprecated-function", wantWarnings: []string{
			"allowed-symbol-load-locations",
			"attr-applicable_licenses",
			"attr-cfg",
			"attr-license",
			"attr-licenses",
			"attr-non-empty",
			"attr-output-default",
			"attr-single-file",
			"build-args-kwargs",
			"bzl-visibility",
			"canonical-repository",
			"confusing-name",
			"constant-glob",
			"ctx-actions",
			"ctx-args",
			// "deprecated-function",
			"depset-items",
			"depset-iteration",
			"depset-union",
			"dict-concatenation",
			"dict-method-named-arg",
			"duplicated-name",
			"external-path",
			"filetype",
			"function-docstring",
			"function-docstring-args",
			"function-docstring-header",
			"function-docstring-return",
			"git-repository",
			"http-archive",
			"integer-division",
			"keyword-positional-params",
			"list-append",
			"load",
			"module-docstring",
			"name-conventions",
			"native-android",
			"native-build",
			"native-cc-binary",
			"native-cc-common",
			"native-cc-debug-package-info",
			"native-cc-fdo-prefetch-hints",
			"native-cc-fdo-profile",
			"native-cc-import",
			"native-cc-info",
			"native-cc-library",
			"native-cc-memprof-profile",
			"native-cc-objc-import",
			"native-cc-objc-library",
			"native-cc-propeller-optimize",
			"native-cc-proto",
			"native-cc-shared-library",
			"native-cc-shared-library-hint-info",
			"native-cc-shared-library-info",
			"native-cc-test",
			"native-cc-toolchain",
			"native-cc-toolchain-suite",
			"native-java-binary",
			"native-java-common",
			"native-java-import",
			"native-java-info",
			"native-java-library",
			"native-java-lite-proto",
			"native-java-package-config",
			"native-java-plugin",
			"native-java-plugin-info",
			"native-java-proto",
			"native-java-runtime",
			"native-java-test",
			"native-java-toolchain",
			"native-package",
			"native-proto",
			"native-proto-common",
			"native-proto-info",
			"native-proto-lang-toolchain",
			"native-proto-lang-toolchain-info",
			"native-py",
			"native-sh-binary",
			"native-sh-library",
			"native-sh-test",
			"no-effect",
			"output-group",
			"overly-nested-depset",
			"package-name",
			"package-on-top",
			"positional-args",
			// "print",
			"provider-params",
			"redefined-variable",
			"repository-name",
			"return-value",
			"rule-impl-return",
			"skylark-comment",
			"skylark-docstring",
			"string-iteration",
			"uninitialized",
			"unnamed-macro",
			"unreachable",
			"unsorted-dict-items",
			"unused-variable",
		}},
		"warnings no duplicates": {options: "--warnings=+unused-variable", wantWarnings: []string{
			"allowed-symbol-load-locations",
			"attr-applicable_licenses",
			"attr-cfg",
			"attr-license",
			"attr-licenses",
			"attr-non-empty",
			"attr-output-default",
			"attr-single-file",
			"build-args-kwargs",
			"bzl-visibility",
			"canonical-repository",
			"confusing-name",
			"constant-glob",
			"ctx-actions",
			"ctx-args",
			"deprecated-function",
			"depset-items",
			"depset-iteration",
			"depset-union",
			"dict-concatenation",
			"dict-method-named-arg",
			"duplicated-name",
			"external-path",
			"filetype",
			"function-docstring",
			"function-docstring-args",
			"function-docstring-header",
			"function-docstring-return",
			"git-repository",
			"http-archive",
			"integer-division",
			"keyword-positional-params",
			"list-append",
			"load",
			"module-docstring",
			"name-conventions",
			"native-android",
			"native-build",
			"native-cc-binary",
			"native-cc-common",
			"native-cc-debug-package-info",
			"native-cc-fdo-prefetch-hints",
			"native-cc-fdo-profile",
			"native-cc-import",
			"native-cc-info",
			"native-cc-library",
			"native-cc-memprof-profile",
			"native-cc-objc-import",
			"native-cc-objc-library",
			"native-cc-propeller-optimize",
			"native-cc-proto",
			"native-cc-shared-library",
			"native-cc-shared-library-hint-info",
			"native-cc-shared-library-info",
			"native-cc-test",
			"native-cc-toolchain",
			"native-cc-toolchain-suite",
			"native-java-binary",
			"native-java-common",
			"native-java-import",
			"native-java-info",
			"native-java-library",
			"native-java-lite-proto",
			"native-java-package-config",
			"native-java-plugin",
			"native-java-plugin-info",
			"native-java-proto",
			"native-java-runtime",
			"native-java-test",
			"native-java-toolchain",
			"native-package",
			"native-proto",
			"native-proto-common",
			"native-proto-info",
			"native-proto-lang-toolchain",
			"native-proto-lang-toolchain-info",
			"native-py",
			"native-sh-binary",
			"native-sh-library",
			"native-sh-test",
			"no-effect",
			"output-group",
			"overly-nested-depset",
			"package-name",
			"package-on-top",
			"positional-args",
			"print",
			"provider-params",
			"redefined-variable",
			"repository-name",
			"return-value",
			"rule-impl-return",
			"skylark-comment",
			"skylark-docstring",
			"string-iteration",
			"uninitialized",
			"unnamed-macro",
			"unreachable",
			"unused-variable",
		}},
		"warnings error": {options: "--warnings=native-py,-print,-deprecated-function", wantErr: fmt.Errorf(`warning categories with modifiers ("+" or "-") can't be mixed with raw warning categories`)},
	} {
		t.Run(name, func(t *testing.T) {
			c := New()
			flags := c.FlagSet("buildifier", flag.ExitOnError)
			flags.Parse(strings.Fields(tc.options))
			got := c.Validate(strings.Fields(tc.args))
			if tc.wantMode != "" && tc.wantMode != c.Mode {
				t.Fatalf("--mode mismatch: want %v, got %v", tc.wantMode, c.Mode)
			}
			if tc.wantLint != "" && tc.wantLint != c.Lint {
				t.Fatalf("--lint mismatch: want %v, got %v", tc.wantLint, c.Lint)
			}
			if len(tc.wantWarnings) > 0 {
				if len(tc.wantWarnings) != len(c.LintWarnings) {
					t.Fatalf("--warnings mismatch: want %v, got %v", tc.wantWarnings, c.LintWarnings)
				}
				for i, wantWarning := range tc.wantWarnings {
					gotWarning := c.LintWarnings[i]
					if wantWarning != gotWarning {
						t.Errorf("warning mismatch at list position %d: want %s, got %s", i, wantWarning, gotWarning)
					}
				}
			}
			if tc.wantErr == nil && got == nil {
				return
			}
			if tc.wantErr == nil && got != nil {
				t.Fatalf("unexpected error: %v", got)
			}
			if tc.wantErr != nil && got == nil {
				t.Fatalf("expected error did not occur: %v", tc.wantErr)
			}
			if tc.wantErr.Error() != got.Error() {
				t.Fatalf("error mismatch: want %v, got %v", tc.wantErr.Error(), got.Error())
			}
		})
	}
}

func TestFindConfigPath(t *testing.T) {
	for name, tc := range map[string]struct {
		files map[string]string
		env   map[string]string
		want  string
	}{
		"no-config-file": {
			want: "",
		},
		"default": {
			files: map[string]string{
				".buildifier.json": "{}",
			},
			want: ".buildifier.json",
		},
		"BUILDIFIER_CONFIG-override": {
			env: map[string]string{
				"BUILDIFIER_CONFIG": ".buildifier_x.json",
			},
			want: ".buildifier_x.json",
		},
	} {
		t.Run(name, func(t *testing.T) {
			for k, v := range tc.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			tmp, err := os.MkdirTemp("", name+"*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmp)

			if err := os.Chdir(tmp); err != nil {
				t.Fatal(err)
			}

			t.Log("tmp:", tmp)

			for rel, content := range tc.files {
				dir := filepath.Join(tmp, filepath.Dir(rel))
				if dir != "." {
					if err := os.MkdirAll(dir, os.ModePerm); err != nil {
						t.Fatal(err)
					}
				}
				filename := filepath.Join(dir, rel)
				if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got := FindConfigPath(tmp)
			got = strings.TrimPrefix(got, tmp)
			got = strings.TrimPrefix(got, "/")

			if tc.want != got {
				t.Errorf("FindConfigPath: want %q, got %q", tc.want, got)
			}
		})
	}
}

func TestFindTablePath(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		files   []string
		wd      string
		want    string
		wantErr error
	}{
		{
			name:    "default",
			file:    ".buildifier-tables.json",
			files:   []string{".buildifier-tables.json"},
			wd:      "",
			want:    ".buildifier-tables.json",
			wantErr: nil,
		},
		{
			name:    "working-dir-is-subdir",
			file:    ".buildifier-tables.json",
			files:   []string{".buildifier-tables.json", "foo/BUILD.bazel"},
			wd:      "foo",
			want:    ".buildifier-tables.json",
			wantErr: nil,
		},
		{
			name:    "relative-subdir",
			file:    "bar/.buildifier-tables.json",
			files:   []string{"bar/.buildifier-tables.json", "foo/BUILD.bazel"},
			wd:      "foo",
			want:    "bar/.buildifier-tables.json",
			wantErr: nil,
		},
		{
			name:    "file-not-found",
			file:    "nonexistentFile.json",
			files:   []string{".buildifier-tables.json"},
			wd:      "",
			want:    "nonexistentFile.json",
			wantErr: os.ErrNotExist,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()

			// On MacOS "/tmp" is a symlink to "/private/tmp". Resolve it to make the testing easier
			tmp, err := filepath.EvalSymlinks(tmp)
			if err != nil {
				t.Fatalf("failed to resolve symlink for temporary directory: %v", err)
			}
			t.Log("tmp:", tmp)

			if tc.wd != "" {
				if err := os.MkdirAll(filepath.Join(tmp, tc.wd), os.ModePerm); err != nil {
					t.Fatalf("failed to create working directory: %v", err)
				}
				if err := os.Chdir(filepath.Join(tmp, tc.wd)); err != nil {
					t.Fatalf("failed to change working directory: %v", err)
				}
			} else {
				if err := os.Chdir(tmp); err != nil {
					t.Fatalf("failed to change working directory: %v", err)
				}
			}

			for _, file := range tc.files {
				filePath := filepath.Join(tmp, file)
				dir := filepath.Dir(filePath)
				if err := os.MkdirAll(dir, os.ModePerm); err != nil {
					t.Fatalf("failed to create directory %v: %v", dir, err)
				}
				if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
					t.Fatalf("failed to create file %v: %v", filePath, err)
				}
			}

			got, err := findTablesPath(tc.file)
			got = strings.TrimPrefix(got, tmp)
			got = strings.TrimPrefix(got, "/")

			if (err != nil) != (tc.wantErr != nil) || (err != nil && tc.wantErr.Error() != err.Error()) {
				t.Errorf("FindTablePath wantErr = %q, error = %q", tc.wantErr, err)
			}

			if tc.want != got {
				t.Errorf("FindTablePath want = %q, got = %q", tc.want, got)
			}
		})
	}
}
