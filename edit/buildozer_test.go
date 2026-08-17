/*
Copyright 2019 Google LLC

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
	"strings"
	"testing"

	"github.com/bazelbuild/buildtools/build"
	"github.com/google/go-cmp/cmp"
)

var removeCommentTests = []struct {
	args      []string
	buildFile string
	expected  string
}{
	{[]string{},
		`# comment
	foo(
		name = "foo",
	)`,
		`foo(
    name = "foo",
)`,
	},
	{[]string{
		"name",
	},
		`foo(
		# comment
		name = "foo",
	)`,
		`foo(
    name = "foo",
)`,
	},
	{[]string{
		"name",
	},
		`foo(
		name = "foo"  # comment,
	)`,
		`foo(
    name = "foo",
)`,
	},
	{[]string{
		"deps", "bar",
	},
		`foo(
		name = "foo",
		deps = [
				# comment
				"bar",
				"baz",
		],
	)`,
		`foo(
    name = "foo",
    deps = [
        "bar",
        "baz",
    ],
)`,
	},
	{[]string{
		"deps", "bar",
	},
		`foo(
		name = "foo",
		deps = [
				"bar",  # comment
				"baz",
		],
	)`,
		`foo(
    name = "foo",
    deps = [
        "bar",
        "baz",
    ],
)`,
	},
}

func TestCmdRemoveComment(t *testing.T) {
	for i, tt := range removeCommentTests {
		bld, err := build.Parse("BUILD", []byte(tt.buildFile))
		if err != nil {
			t.Error(err)
			continue
		}
		rl := bld.Rules("foo")[0]
		env := CmdEnvironment{
			File: bld,
			Rule: rl,
			Args: tt.args,
		}
		bld, _ = cmdRemoveComment(NewOpts(), env)
		got := strings.TrimSpace(string(build.Format(bld)))
		if got != tt.expected {
			t.Errorf("cmdRemoveComment(%d):\ngot:\n%s\nexpected:\n%s", i, got, tt.expected)
		}
	}
}

func TestCmdComment(t *testing.T) {
	var tests = []struct {
		name      string
		args      []string
		buildFile string
		want      string
	}{
		{
			name: "adds_comment_to_rule_call",
			args: []string{"New Comment"},
			buildFile: `foo(
    name = "foo",
)`,
			want: `# New Comment
foo(
    name = "foo",
)`,
		}, {
			name: "adds_comment_to_single_line_attribute",
			args: []string{"deps", "New Comment"},
			buildFile: `foo(
    name = "foo",
    deps = ["//some/dep"],
)`,
			want: `foo(
    name = "foo",
    deps = ["//some/dep"],  # New Comment
)`,
		}, {
			name: "adds_multiline_comment_to_single_line_attribute",
			args: []string{"deps", "New Comment\nWith Multiple Lines"},
			buildFile: `foo(
    name = "foo",
    deps = ["//some/dep"],
)`,
			want: `foo(
    name = "foo",
    # New Comment
    # With Multiple Lines
    deps = ["//some/dep"],
)`,
		}, {
			name: "adds_comment_to_multiline_attribute",
			args: []string{"deps", "New Comment"},
			buildFile: `foo(
    name = "foo",
    deps = [
	    "//some/dep",
	    "//some/other/dep",
    ],
)`,
			want: `foo(
    name = "foo",
    # New Comment
    deps = [
        "//some/dep",
        "//some/other/dep",
    ],
)`,
		}, {
			name: "adds_comment_to_element_in_list",
			args: []string{"deps", "//some/other/dep", "New Comment"},
			buildFile: `foo(
    name = "foo",
    deps = [
	    "//some/dep",
	    "//some/other/dep",
    ],
)`,
			want: `foo(
    name = "foo",
    deps = [
        "//some/dep",
        "//some/other/dep",  # New Comment
    ],
)`,
		}, {
			name: "adds_multiline_comment_to_element_in_list",
			args: []string{"deps", "//some/other/dep", "New Comment\nWith Multiple Lines"},
			buildFile: `foo(
    name = "foo",
    deps = [
	    "//some/dep",
	    "//some/other/dep",
    ],
)`,
			want: `foo(
    name = "foo",
    deps = [
        "//some/dep",
        # New Comment
        # With Multiple Lines
        "//some/other/dep",
    ],
)`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bld, err := build.Parse("BUILD", []byte(tc.buildFile))
			if err != nil {
				t.Error(err)
			}
			rl := bld.Rules("foo")[0]
			env := CmdEnvironment{
				File: bld,
				Rule: rl,
				Args: tc.args,
			}
			bld, _ = cmdComment(NewOpts(), env)
			got := strings.TrimSpace(string(build.Format(bld)))
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Fatalf("cmdComment returned unexpected diff %s", diff)
			}
		})
	}
}

type targetExpressionToBuildFilesTestCase struct {
	rootDir, target string
	buildFiles      []string
}

func setupTestTmpWorkspace(t *testing.T, buildFileName string) (tmp string) {
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	// On MacOS "/tmp" is a symlink to "/private/tmp". Resolve it to make the testing easier
	tmp, err = filepath.EvalSymlinks(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(tmp, "a", "b"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "a", "c"), 0755); err != nil {
		t.Fatal(err)
	}
	// Create additional directories that will be ignored
	if err := os.MkdirAll(filepath.Join(tmp, "ignored"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "a", "ignored"), 0755); err != nil {
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
	if err := os.WriteFile(filepath.Join(tmp, "a", "c", buildFileName), nil, 0755); err != nil {
		t.Fatal(err)
	}
	// Create BUILD files in ignored directories to verify they're skipped
	if err := os.WriteFile(filepath.Join(tmp, "ignored", buildFileName), nil, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "a", "ignored", buildFileName), nil, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .bazelignore file with paths to ignore
	bazelignoreContent := []byte("# Ignore these directories\nignored\na/ignored/\n")
	if err := os.WriteFile(filepath.Join(tmp, ".bazelignore"), bazelignoreContent, 0644); err != nil {
		t.Fatal(err)
	}

	return
}

func runTestTargetExpressionToBuildFiles(t *testing.T, buildFileName string) {
	tmp := setupTestTmpWorkspace(t, buildFileName)
	defer os.RemoveAll(tmp)

	for _, tc := range []targetExpressionToBuildFilesTestCase{
		{tmp, "//", []string{filepath.Join(tmp, buildFileName)}},
		{tmp, "//:foo", []string{filepath.Join(tmp, buildFileName)}},
		{tmp, "//a", []string{filepath.Join(tmp, "a", buildFileName)}},
		{tmp, "//a:foo", []string{filepath.Join(tmp, "a", buildFileName)}},
		{tmp, "//a/b", []string{filepath.Join(tmp, "a", "b", buildFileName)}},
		{tmp, "//a/b:foo", []string{filepath.Join(tmp, "a", "b", buildFileName)}},
		{tmp, "//...", []string{filepath.Join(tmp, buildFileName), filepath.Join(tmp, "a", buildFileName), filepath.Join(tmp, "a", "b", buildFileName), filepath.Join(tmp, "a", "c", buildFileName)}},
		{tmp, "//a/...", []string{filepath.Join(tmp, "a", buildFileName), filepath.Join(tmp, "a", "b", buildFileName), filepath.Join(tmp, "a", "c", buildFileName)}},
		{tmp, "//a/b/...", []string{filepath.Join(tmp, "a", "b", buildFileName)}},
		{tmp, "//a/c/...", []string{filepath.Join(tmp, "a", "c", buildFileName)}},
		{tmp, "//a/c/...:foo", []string{filepath.Join(tmp, "a", "c", buildFileName)}},
		{"", "...:foo", []string{filepath.Join(tmp, buildFileName), filepath.Join(tmp, "a", buildFileName), filepath.Join(tmp, "a", "b", buildFileName), filepath.Join(tmp, "a", "c", buildFileName)}},
	} {
		if tc.rootDir == "" {
			cwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			// buildozer should be able to find the WORKSPACE file in the current wd
			if err := os.Chdir(tmp); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(cwd)
		}

		buildFiles := targetExpressionToBuildFiles(tc.rootDir, tc.target, true)
		expectedBuildFilesMap := make(map[string]bool)
		buildFilesMap := make(map[string]bool)
		for _, buildFile := range buildFiles {
			buildFilesMap[buildFile] = true
		}
		for _, buildFile := range tc.buildFiles {
			expectedBuildFilesMap[buildFile] = true
		}
		if !reflect.DeepEqual(expectedBuildFilesMap, buildFilesMap) {
			t.Errorf("TargetExpressionToBuildFiles(%q, %q) = %q want %q", tc.rootDir, tc.target, buildFiles, tc.buildFiles)
		}
	}
}

func TestTargetExpressionToBuildFiles(t *testing.T) {
	for _, buildFileName := range BuildFileNames {
		runTestTargetExpressionToBuildFiles(t, buildFileName)
	}
}

func runTestAppendCommands(t *testing.T, buildFileName string) {
	tmp := setupTestTmpWorkspace(t, buildFileName)
	defer os.RemoveAll(tmp)

	for _, tc := range []targetExpressionToBuildFilesTestCase{
		{tmp, ".:__pkg__", []string{"./" + buildFileName}},
		{tmp, "a" + ":__pkg__", []string{"a/" + buildFileName}},
		{"", "a" + ":__pkg__", []string{"a/" + buildFileName}},
	} {
		if tc.rootDir == "" {
			cwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			// buildozer should be able to find the WORKSPACE file in the current wd
			if err := os.Chdir(tmp); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(cwd)
		}

		commandsByFile := make(map[string][]commandsForTarget)
		opts := NewOpts()
		opts.RootDir = tc.rootDir
		appendCommands(opts, commandsByFile, tc.buildFiles)
		if len(commandsByFile) != 1 {
			t.Errorf("Expect one target after appendCommands")
		}
		for _, value := range commandsByFile {
			if value[0].target != tc.target {
				t.Errorf("appendCommands for buildfile %s yielded target %s, expected %s", tc.buildFiles, value[0].target, tc.target)
			}
		}
	}
}

func TestAppendCommands(t *testing.T) {
	for _, buildFileName := range BuildFileNames {
		runTestAppendCommands(t, buildFileName)
	}
}

var dictListAddTests = []struct {
	args      []string
	buildFile string
	expected  string
}{
	{[]string{
		"attr", "key1", "value1",
	},
		`foo(
		name = "foo",
	)`,
		`foo(
    name = "foo",
    attr = {"key1": ["value1"]},
)`,
	},
	{[]string{
		"attr", "key1", "value2",
	},
		`foo(
		name = "foo",
		attr = {"key1": ["value1"]},
	)`,
		`foo(
    name = "foo",
    attr = {"key1": [
        "value1",
        "value2",
    ]},
)`,
	},
	{[]string{
		"attr", "key1", "value1", "value2",
	},
		`foo(
		name = "foo",
	)`,
		`foo(
    name = "foo",
    attr = {"key1": [
        "value1",
        "value2",
    ]},
)`,
	},
	{[]string{
		"attr", "key2", "value2",
	},
		`foo(
		name = "foo",
		attr = {"key1": ["value1"]},
	)`,
		`foo(
    name = "foo",
    attr = {
        "key1": ["value1"],
        "key2": ["value2"],
    },
)`,
	},
	{[]string{
		"attr", "key1", "value1",
	},
		`foo(
		name = "foo",
		attr = {"key1": ["value1"]},
	)`,
		`foo(
    name = "foo",
    attr = {"key1": ["value1"]},
)`,
	},
}

func TestCmdDictListAdd(t *testing.T) {
	for i, tt := range dictListAddTests {
		bld, err := build.Parse("BUILD", []byte(tt.buildFile))
		if err != nil {
			t.Error(err)
			continue
		}
		rl := bld.Rules("foo")[0]
		env := CmdEnvironment{
			File: bld,
			Rule: rl,
			Args: tt.args,
		}
		bld, _ = cmdDictListAdd(NewOpts(), env)
		got := strings.TrimSpace(string(build.Format(bld)))
		if got != tt.expected {
			t.Errorf("cmdDictListAdd(%d):\ngot:\n%s\nexpected:\n%s", i, got, tt.expected)
		}
	}
}

var dictReplaceIfEqualTests = []struct {
	args      []string
	buildFile string
	expected  string
}{
	{[]string{
		"attr", "key1", "value1", "value2",
	},
		`foo(
		name = "foo",
		attr = {"key1": "value1"},
	)`,
		`foo(
    name = "foo",
    attr = {"key1": "value2"},
)`,
	},
	{[]string{
		"attr", "key1", "value1", "value2",
	},
		`foo(
		name = "foo",
		attr = {"key1": "x"},
	)`,
		`foo(
    name = "foo",
    attr = {"key1": "x"},
)`,
	},
	{[]string{
		"attr", "key1", "value1", "value2",
	},
		`foo(
		name = "foo",
		attr = {
      "key1": ["value1"],
			"key2": ["value2"],
	},
	)`,
		`foo(
    name = "foo",
    attr = {
			"key1": ["value1"],
			"key2": ["value2"],
    },
)`,
	},
	{[]string{
		"attr", "key1", "value1", "value2",
	},
		`foo(
		name = "foo",
		attr = {
			"key1": "value1",
			"key2": "value2",
	},
	)`,
		`foo(
		name = "foo",
		attr = {
			"key1": "value2",
			"key2": "value2",
},
)`,
	},
	{[]string{
		"attr", "key1", "value1", "value2",
	},
		`foo(
		name = "foo",
		attr = {
			"key1": "value1",
			"key2": "x",
	},
	)`,
		`foo(
    name = "foo",
    attr = {
			"key1": "value2",
			"key2": "x",
		},
)`,
	},
	{[]string{
		"attr", "*", "value1", "value2",
	},
		`foo(
		name = "foo",
		attr = {
			"key1": "value1",
			"key2": "x",
	},
	)`,
		`foo(
    name = "foo",
    attr = {
			"key1": "value2",
			"key2": "x",
		},
)`,
	},
}

func TestCmdDictReplaceIfEqual(t *testing.T) {
	for i, tt := range dictReplaceIfEqualTests {
		bld, err := build.Parse("BUILD", []byte(tt.buildFile))
		if err != nil {
			t.Error(err)
			continue
		}
		expectedBld, err := build.Parse("BUILD", []byte(tt.expected))
		if err != nil {
			t.Error(err)
			continue
		}
		rl := bld.Rules("foo")[0]
		env := CmdEnvironment{
			File: bld,
			Rule: rl,
			Args: tt.args,
		}
		bld, err = cmdDictReplaceIfEqual(NewOpts(), env)
		if err != nil {
			t.Errorf("cmdDictReplaceIfEqual(%d):\ngot error:\n%s", i, err)
		}
		got := strings.TrimSpace(string(build.Format(bld)))
		expected := strings.TrimSpace(string(build.Format(expectedBld)))
		if got != expected {
			t.Errorf("cmdDictReplaceIfEqual(%d):\ngot:\n%s\nexpected:\n%s", i, got, tt.expected)
		}
	}
}

var substituteLoadsTests = []struct {
	args      []string
	buildFile string
	expected  string
}{
	{[]string{
		"^(.*)$", "${1}",
	},
		`load("//foo:foo.bzl", "foo")`,
		`load("//foo:foo.bzl", "foo")`,
	},
	{[]string{
		"^@rules_foo//foo:defs.bzl$", "//build/rules/foo:defs.bzl",
	},
		`load("@rules_bar//bar:defs.bzl", "bar")`,
		`load("@rules_bar//bar:defs.bzl", "bar")`,
	},
	{[]string{
		"^@rules_foo//foo:defs.bzl$", "//build/rules/foo:defs.bzl",
	},
		`load("@rules_foo//foo:defs.bzl", "foo", "foo2")
load("@rules_bar//bar:defs.bzl", "bar")`,
		`load("@rules_bar//bar:defs.bzl", "bar")
load("//build/rules/foo:defs.bzl", "foo", "foo2")`,
	},
	{[]string{
		":foo.bzl$", ":defs.bzl",
	},
		`load("//foo:foo.bzl", "foo")`,
		`load("//foo:defs.bzl", "foo")`,
	},
	{[]string{
		// Keep in sync with the example in `//buildozer:README.md`.
		"^@([^/]*)//([^:].*)$", "//third_party/build_defs/${1}/${2}",
	},
		`load("@rules_foo//foo:defs.bzl", "foo", "foo2")
load("@rules_bar//bar:defs.bzl", "bar")
load("@rules_bar//:defs.bzl", legacy_bar = "bar")`,
		`load("@rules_bar//:defs.bzl", legacy_bar = "bar")
load("//third_party/build_defs/rules_bar/bar:defs.bzl", "bar")
load("//third_party/build_defs/rules_foo/foo:defs.bzl", "foo", "foo2")`,
	},
}

func TestCmdSubstituteLoad(t *testing.T) {
	for i, tt := range substituteLoadsTests {
		bld, err := build.Parse("BUILD", []byte(tt.buildFile))
		if err != nil {
			t.Error(err)
			continue
		}
		env := CmdEnvironment{
			File: bld,
			Args: tt.args,
		}
		bld, _ = cmdSubstituteLoad(NewOpts(), env)
		got := strings.TrimSpace(string(build.Format(bld)))
		if got != tt.expected {
			t.Errorf("cmdSubstituteLoad(%d):\ngot:\n%s\nexpected:\n%s", i, got, tt.expected)
		}
	}
}

func TestCmdSubstitute(t *testing.T) {
	for i, tc := range []struct {
		name      string
		args      []string
		buildFile string
		expected  string
	}{
		{
			name:      "empty_rule",
			args:      []string{"*", "^$", "x"},
			buildFile: `cc_library()`,
			expected:  `cc_library()`,
		},
		{
			name:      "known_attr",
			args:      []string{"*", "^//(.*)$", "//foo/${1}"},
			buildFile: `cc_library(deps = ["//bar/baz:quux"])`,
			expected:  `cc_library(deps = ["//foo/bar/baz:quux"])`,
		},
		{
			name:      "custom_attr",
			args:      []string{"*", "^//(.*)$", "//foo/${1}"},
			buildFile: `cc_library(my_custom_attr = "//bar/baz:quux")`,
			expected:  `cc_library(my_custom_attr = "//foo/bar/baz:quux")`,
		},
		{
			name:      "specific_rule",
			args:      []string{"deps", "^//(.*)$", "//foo/${1}"},
			buildFile: `cc_library(deps = ["//bar"], fancy_deps = ["//bar/baz:quux"])`,
			expected: `cc_library(
    fancy_deps = ["//bar/baz:quux"],
    deps = ["//foo/bar"],
)`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bld, err := build.Parse("BUILD", []byte(tc.buildFile))
			if err != nil {
				t.Error(err)
				return
			}
			env := CmdEnvironment{
				File: bld,
				Args: tc.args,
				Rule: bld.RuleAt(1),
			}
			bld, _ = cmdSubstitute(NewOpts(), env)
			got := strings.TrimSpace(string(build.Format(bld)))
			if got != tc.expected {
				t.Errorf("cmdSubstitute(%d):\ngot:\n%s\nexpected:\n%s", i, got, tc.expected)
			}
		})
	}
}

func TestCmdDictAddSet_missingColon(t *testing.T) {
	for _, tc := range []struct {
		name string
		fun  func(*Options, CmdEnvironment) (*build.File, error)
	}{
		{"dict_add", cmdDictAdd},
		{"dict_set", cmdDictSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bld, err := build.Parse("BUILD", []byte("rule()"))
			if err != nil {
				t.Fatal(err)
			}
			env := CmdEnvironment{
				File: bld,
				Rule: bld.RuleAt(1),
				Args: []string{"attr", "invalid"},
			}
			_, err = tc.fun(NewOpts(), env)
			if err == nil {
				t.Error("succeeded, want error")
			}
		})
	}
}

func TestCmdDictOperations(t *testing.T) {
	tests := []struct {
		name           string
		dictAddArgs    []string
		dictSetArgs    []string
		dictRemoveArgs []string
		input          string
		want           string
	}{
		{
			name:           "dict_add_set_remove",
			dictAddArgs:    []string{"dict_attr", `added_entry:new_value`},
			dictSetArgs:    []string{"dict_attr", `entry_to_change:updated_value`},
			dictRemoveArgs: []string{"dict_attr", `entry_to_remove`},
			input: strings.Join([]string{
				`rule(`,
				`  name = "rule_name",`,
				`  dict_attr = {`,
				`    "entry_to_change": "123",`,
				`    "entry_to_remove": "abc",`,
				`  },`,
				`)`,
			}, "\n"),
			want: strings.Join([]string{
				`rule(`,
				`    name = "rule_name",`,
				`    dict_attr = {`,
				`        "entry_to_change": "updated_value",`,
				`        "added_entry": "new_value",`,
				`    },`,
				`)`,
				``,
			}, "\n"),
		},
		{
			name:           "dict_add_set_remove_with_escaped_colon",
			dictAddArgs:    []string{"dict_attr", `added\:entry:new:value`},
			dictSetArgs:    []string{"dict_attr", `entry\:to_change:updated\:value`},
			dictRemoveArgs: []string{"dict_attr", `entry\:to_remove`},
			input: strings.Join([]string{
				`rule(`,
				`  name = "rule_name",`,
				`  dict_attr = {`,
				`    "entry:to_change": "123",`,
				`    "entry:to_remove": "abc",`,
				`  },`,
				`)`,
			}, "\n"),
			want: strings.Join([]string{
				`rule(`,
				`    name = "rule_name",`,
				`    dict_attr = {`,
				`        "entry:to_change": "updated:value",`,
				`        "added:entry": "new:value",`,
				`    },`,
				`)`,
				``,
			}, "\n"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file, err := build.Parse("BUILD", []byte(tc.input))
			if err != nil {
				t.Fatalf("build.Parse returned err: %s", err)
			}
			rule := file.RuleNamed("rule_name")
			file, err = cmdDictAdd(NewOpts(), CmdEnvironment{File: file, Rule: rule, Args: tc.dictAddArgs})
			if err != nil {
				t.Fatalf("cmdDictAdd returned err: %s", err)
			}

			file, err = cmdDictSet(NewOpts(), CmdEnvironment{File: file, Rule: rule, Args: tc.dictSetArgs})
			if err != nil {
				t.Fatalf("cmdDictSet returned err: %s", err)
			}

			file, err = cmdDictRemove(NewOpts(), CmdEnvironment{File: file, Rule: rule, Args: tc.dictRemoveArgs})
			if err != nil {
				t.Fatalf("cmdDictRemove returned err: %s", err)
			}

			got := string(build.Format(file))

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("dict operations returned diff -want +got %v", diff)
			}
		})
	}
}

func TestCmdSetSelect(t *testing.T) {
	for i, tc := range []struct {
		name      string
		args      []string
		buildFile string
		expected  string
	}{
		{
			name: "select_statement_doesn't_exist",
			args: []string{
				"args",                                   /* attr */
				":use_ci_timeouts", "-test.timeout=123s", /* key, value */
				":use_ci_timeouts", "-test.anotherFlag=flagValue", /* key, value */
				"//conditions:default", "-test.timeout=789s", /* key, value */
			},
			buildFile: `foo(
			name = "foo",
)`,
			expected: `foo(
    name = "foo",
    args = select({
        ":use_ci_timeouts": [
            "-test.timeout=123s",
            "-test.anotherFlag=flagValue",
        ],
        "//conditions:default": ["-test.timeout=789s"],
    }),
)`},
		{
			name: "select_statement_exists",
			args: []string{
				"args",                                   /* attr */
				":use_ci_timeouts", "-test.timeout=543s", /* key, value */
				"//conditions:default", "-test.timeout=876s", /* key, value */
			},
			buildFile: `foo(
    name = "foo",
    args = select({
        ":use_ci_timeouts": [
            "-test.timeout=123s",
            "-test.anotherFlag=flagValue",
        ],
        "//conditions:default": ["-test.timeout=789s"],
    }),
)`,
			expected: `foo(
    name = "foo",
    args = select({
        ":use_ci_timeouts": ["-test.timeout=543s"],
        "//conditions:default": ["-test.timeout=876s"],
    }),
)`},
		{
			name: "attr_exists_but_not_select",
			args: []string{
				"args",                                   /* attr */
				":use_ci_timeouts", "-test.timeout=543s", /* key, value */
				"//conditions:default", "-test.timeout=876s", /* key, value */
			},
			buildFile: `foo(
    name = "foo",
    args = ["-test.timeout=123s"],
)`,
			expected: `foo(
    name = "foo",
    args = select({
        ":use_ci_timeouts": ["-test.timeout=543s"],
        "//conditions:default": ["-test.timeout=876s"],
    }),
)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bld, err := build.Parse("BUILD", []byte(tc.buildFile))
			if err != nil {
				t.Error(err)
			}
			rl := bld.Rules("foo")[0]
			env := CmdEnvironment{
				File: bld,
				Rule: rl,
				Args: tc.args,
			}
			bld, _ = cmdSetSelect(NewOpts(), env)
			got := strings.TrimSpace(string(build.Format(bld)))
			if got != tc.expected {
				t.Errorf("cmdSetSelect(%d):\ngot:\n%s\nexpected:\n%s", i, got, tc.expected)
			}
		})
	}
}

func TestExecuteCommandsOnInlineFile(t *testing.T) {
	tests := []struct {
		name        string
		fileContent []byte
		commands    []string
		wantOutput  []byte
	}{
		{
			name:        "creating_new_target_and_adding_deps",
			fileContent: nil,
			commands: []string{
				"new java_library foo|//package/path/BUILD",
				"add deps :bar|//package/path:foo",
			},
			wantOutput: []byte(strings.Join([]string{
				`java_library(`,
				`    name = "foo",`,
				`    deps = [":bar"],`,
				`)`,
				``}, "\n")),
		},
		{
			name: "adding_deps_to_existing_targets",
			fileContent: []byte(strings.Join([]string{
				`java_library(`,
				`    name = "foo",`,
				`)`,
				``,
				`java_library(`,
				`    name = "fruits",`,
				`    deps = ["//package/fruits:apples"],`,
				`)`,
				``}, "\n")),
			commands: []string{
				"add deps :bar|//package/path:foo",
				"add deps //package/fruits:oranges|//package/path:fruits",
			},
			wantOutput: []byte(strings.Join([]string{
				`java_library(`,
				`    name = "foo",`,
				`    deps = [":bar"],`,
				`)`,
				``,
				`java_library(`,
				`    name = "fruits",`,
				`    deps = [`,
				`        "//package/fruits:apples",`,
				`        "//package/fruits:oranges",`,
				`    ],`,
				`)`,
				``}, "\n")),
		},
		{
			name: "substituting_a_target",
			fileContent: []byte(strings.Join([]string{
				`java_library(`,
				`    name = "fruits",`,
				`    deps = ["//package/fruits:apples"],`,
				`)`,
				``}, "\n")),
			commands: []string{
				"replace deps //package/fruits:apples //package/fruits:oranges|//whatever/package/path:fruits",
			},
			wantOutput: []byte(strings.Join([]string{
				`java_library(`,
				`    name = "fruits",`,
				`    deps = ["//package/fruits:oranges"],`,
				`)`,
				``}, "\n")),
		},
		{
			name: "no_changes_does_not_return_any_diff",
			fileContent: []byte(strings.Join([]string{
				`java_library(`,
				`    name = "foo",`,
				`    deps = [":bar"],`,
				`)`,
				``}, "\n")),
			commands: []string{
				"add deps :bar |//whatever/package/path:foo",
			},
			wantOutput: []byte(strings.Join([]string{
				`java_library(`,
				`    name = "foo",`,
				`    deps = [":bar"],`,
				`)`,
				``}, "\n")),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output, err := ExecuteCommandsOnInlineFile(tc.fileContent, tc.commands)
			if err != nil {
				t.Fatalf("Error, got error %v", err)
			}

			if diff := cmp.Diff(tc.wantOutput, output); diff != "" {
				t.Errorf("%s: (-want +got): %s", tc.name, diff)
			}
		})
	}
}

func TestTestExecuteCommandsOnInlineFileFailed(t *testing.T) {
	tests := []struct {
		name        string
		fileContent []byte
		commands    []string
		wantErr     error
	}{
		{
			name: "target_does_not_exist",
			commands: []string{
				"add deps :foo|//package/path:bar",
			},
			wantErr: fmt.Errorf("rule 'bar' not found"),
		},
		{
			name: "invalid_input",
			commands: []string{
				"completely invalid command",
			},
			wantErr: fmt.Errorf("rule 'completely invalid command' not found"),
		},
		{
			name: "missing_implementation",
			commands: []string{
				"extrapolate packages :foo|//package/path:bar",
			},
			wantErr: fmt.Errorf("invalid input commands, expected all commands to reference a single file"),
		},
		{
			name: "commands_for_multiple_files",
			commands: []string{
				"add deps :foo|//package/path:bar",
				"add deps :foo|//package2/path:bar",
			},
			wantErr: fmt.Errorf("invalid input commands, expected all commands to reference a single file"),
		},
		{
			name: "command_with_unexpected_target_semicolons",
			commands: []string{
				"add deps :foo|//package:path:bar",
			},
			wantErr: fmt.Errorf("invalid target name \"//package:path:bar\""),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output, gotErr := ExecuteCommandsOnInlineFile(tc.fileContent, tc.commands)

			if output != nil {
				t.Fatalf("Error, got response for invalid input %v, expected error", output)
			}

			if diff := cmp.Diff(tc.wantErr.Error(), gotErr.Error()); diff != "" {
				t.Errorf("%s: (-want +got): %s", tc.name, diff)
			}
		})
	}
}

func TestGetIgnoredPrefixes(t *testing.T) {
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	tests := []struct {
		name          string
		bazelignore   string
		expected      []string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid paths",
			bazelignore: `# Ignore these directories
ignored
a/ignored
b/c/d`,
			expected: []string{"ignored", "a/ignored", "b/c/d"},
		},
		{
			name:        "empty file",
			bazelignore: ` `,
			expected:    []string{},
		},
		{
			name: "only comments",
			bazelignore: `# This is a comment
# Another comment`,
			expected: []string{},
		},
		{
			name: "empty lines",
			bazelignore: `ignored

a/ignored

# comment
b/c/d`,
			expected: []string{"ignored", "a/ignored", "b/c/d"},
		},
		{
			name: "absolute paths",
			bazelignore: `/absolute/path
ignored
/another/absolute/path`,
			expected: []string{"ignored"},
		},
		{
			name:        "no file",
			bazelignore: "",
			expected:    []string{},
		},
		{
			name:        "trailing slash should be normalized",
			bazelignore: `ignored/`,
			expected:    []string{"ignored"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new temporary directory for each test case
			testDir, err := os.MkdirTemp(tmp, "")
			if err != nil {
				t.Fatal(err)
			}

			if tt.bazelignore != "" {
				if err := os.WriteFile(filepath.Join(testDir, ".bazelignore"), []byte(tt.bazelignore), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got := getIgnoredPrefixes(testDir)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("getIgnoredPrefixes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestShouldIgnorePath(t *testing.T) {
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	tests := []struct {
		name            string
		path            string
		ignoredPrefixes []string
		want            bool
	}{
		{
			name:            "exact match",
			path:            filepath.Join(tmp, "foo"),
			ignoredPrefixes: []string{"foo"},
			want:            true,
		},
		{
			name:            "subdirectory",
			path:            filepath.Join(tmp, "foo", "bar"),
			ignoredPrefixes: []string{"foo"},
			want:            true,
		},
		{
			name:            "similar prefix but not directory",
			path:            filepath.Join(tmp, "foobar"),
			ignoredPrefixes: []string{"foo"},
			want:            false, // Should not ignore "foobar" when only "foo" is ignored
		},
		{
			name:            "matched with multiple prefixes",
			path:            filepath.Join(tmp, "foobar"),
			ignoredPrefixes: []string{"foo2", "foobar", "baz"},
			want:            true,
		},
		{
			name:            "no match",
			path:            filepath.Join(tmp, "bar"),
			ignoredPrefixes: []string{"foo", "baz"},
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldIgnorePath(tt.path, tmp, tt.ignoredPrefixes)
			if got != tt.want {
				t.Errorf("shouldIgnorePath(%q, %q, %v) = %v, want %v",
					tt.path, tmp, tt.ignoredPrefixes, got, tt.want)
			}
		})
	}
}

func TestSplitOnNonEscaped(t *testing.T) {
	tests := []struct {
		name  string
		input string
		sep   byte
		lim   int
		want  []string
	}{
		{
			name:  "no_split",
			input: "one:two",
			sep:   '|',
			lim:   -1,
			want:  []string{"one:two"},
		},
		{
			name:  "split_to_two",
			input: "one:two",
			sep:   ':',
			lim:   2,
			want:  []string{"one", "two"},
		},
		{
			name:  "split_with_limit",
			input: "one:two:three:four",
			sep:   ':',
			lim:   2,
			want:  []string{"one", "two:three:four"},
		},
		{
			name:  "split_without_limit",
			input: "one:two:three:four",
			sep:   ':',
			lim:   -1,
			want:  []string{"one", "two", "three", "four"},
		},
		{
			name:  "does_not_split_on_escaped",
			input: `one\:two:three`,
			sep:   ':',
			lim:   2,
			want:  []string{`one\:two`, "three"},
		},
		{
			name:  "split_on_pipe",
			input: `one|two|three`,
			sep:   '|',
			lim:   -1,
			want:  []string{"one", "two", "three"},
		},
		{
			name:  "skip_escaped_pipes",
			input: `one\|two|three`,
			sep:   '|',
			lim:   -1,
			want:  []string{`one\|two`, "three"},
		},
		{
			name:  "split_with_unicode",
			input: `😉|❤️|😁`,
			sep:   '|',
			lim:   -1,
			want:  []string{"😉", "❤️", "😁"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitOnNonEscaped(tc.input, tc.sep, tc.lim)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("splitOnNonEscaped returned diff -want +got %v", diff)
			}
		})
	}
}

func TestBuildozerWithTypeAliases(t *testing.T) {
	buildFile := `type Numeric = int | float
type OptionalDict[T, U] = dict[T, U] | None
_: Any # Enable type checking

foo(
    name = "foo",
    deps = ["//a"],
)
`
	bld, err := build.Parse("test.bzl", []byte(buildFile))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	rl := bld.Rules("foo")[0]
	env := CmdEnvironment{
		File: bld,
		Rule: rl,
		Args: []string{"deps", "//b"},
	}
	bld, _ = cmdAdd(NewOpts(), env)
	got := string(build.Format(bld))
	expected := `type Numeric = int | float
type OptionalDict[T, U] = dict[T, U] | None
_: Any  # Enable type checking

foo(
    name = "foo",
    deps = ["//a", "//b"],
)
`
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}
