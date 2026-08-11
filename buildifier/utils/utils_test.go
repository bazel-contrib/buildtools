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

package utils

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestIsStarlarkFile(t *testing.T) {
	tests := []struct {
		filename string
		ok       bool
	}{
		{
			filename: "BUILD",
			ok:       true,
		},
		{
			filename: "BUILD.bazel",
			ok:       true,
		},
		{
			filename: "BUILD.oss",
			ok:       true,
		},
		{
			filename: "BUILD.foo.bazel",
			ok:       true,
		},
		{
			filename: "BUILD.foo.oss",
			ok:       true,
		},
		{
			filename: "build.foo.bazel",
			ok:       true,
		},
		{
			filename: "foo.BUILD.bazel",
			ok:       true,
		},
		{
			filename: "foo.BUILD",
			ok:       true,
		},
		{
			filename: "build.foo.oss",
			ok:       false,
		},
		{
			filename: "build.oss",
			ok:       false,
		},
		{
			filename: "WORKSPACE",
			ok:       true,
		},
		{
			filename: "WORKSPACE.bazel",
			ok:       true,
		},
		{
			filename: "WORKSPACE.oss",
			ok:       true,
		},
		{
			filename: "WORKSPACE.foo.bazel",
			ok:       true,
		},
		{
			filename: "WORKSPACE.foo.oss",
			ok:       true,
		},
		{
			filename: "workspace.foo.bazel",
			ok:       true,
		},
		{
			filename: "workspace.foo.oss",
			ok:       false,
		},
		{
			filename: "workspace.oss",
			ok:       false,
		},
		{
			filename: "build.gradle",
			ok:       false,
		},
		{
			filename: "workspace.xml",
			ok:       false,
		},
		{
			filename: "foo.bzl",
			ok:       true,
		},
		{
			filename: "foo.BZL",
			ok:       false,
		},
		{
			filename: "build.bzl",
			ok:       true,
		},
		{
			filename: "workspace.sky",
			ok:       true,
		},
		{
			filename: "foo.star",
			ok:       true,
		},
		{
			filename: "foo.bar",
			ok:       false,
		},
		{
			filename: "foo.build",
			ok:       false,
		},
		{
			filename: "foo.workspace",
			ok:       false,
		},
		{
			filename: "MODULE.bazel",
			ok:       true,
		},
		{
			filename: "my.MODULE.bazel",
			ok:       true,
		},
		{
			filename: "MODULE.bazel.other",
			ok:       false,
		},
		{
			filename: "foo.bazel",
			ok:       true,
		},
		{
			filename: "REPO.bazel",
			ok:       true,
		},
	}

	for _, tc := range tests {
		if isStarlarkFile(tc.filename) != tc.ok {
			t.Errorf("Wrong result for %q, want %t", tc.filename, tc.ok)
		}
	}
}

func TestExpandDirectoriesExcludesPaths(t *testing.T) {
	root := t.TempDir()
	for _, filename := range []string{
		"BUILD",
		"included/defs.bzl",
		"excluded/direct.bzl",
		"excluded/nested/BUILD.bazel",
		"other/skip.sky",
		"other/keep.star",
	} {
		path := filepath.Join(root, filepath.FromSlash(filename))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	args := []string{root}
	files, err := ExpandDirectories(
		&args,
		filepath.Join(root, "excluded", "*"),
		filepath.Join(root, "*", "skip.sky"),
	)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		filepath.Join(root, "BUILD"),
		filepath.Join(root, "included", "defs.bzl"),
		filepath.Join(root, "other", "keep.star"),
	}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("ExpandDirectories() = %q, want %q", files, want)
	}
}

func TestMatchPathAllowsWildcardsAcrossSeparators(t *testing.T) {
	for _, tc := range []struct {
		pattern  string
		filename string
		want     bool
	}{
		{pattern: "./vendor/*", filename: "vendor/direct.bzl", want: true},
		{pattern: "./vendor/*", filename: "vendor/nested/defs.bzl", want: true},
		{pattern: "./vendor/*.bzl", filename: "vendor/nested/defs.bzl", want: true},
		{pattern: "./vendor/*.bzl", filename: "third_party/defs.bzl", want: false},
	} {
		got, err := matchPath(tc.pattern, tc.filename)
		if err != nil {
			t.Errorf("matchPath(%q, %q) returned error: %v", tc.pattern, tc.filename, err)
		} else if got != tc.want {
			t.Errorf("matchPath(%q, %q) = %t, want %t", tc.pattern, tc.filename, got, tc.want)
		}
	}
}

func TestExpandDirectoriesRejectsInvalidExcludePattern(t *testing.T) {
	args := []string{t.TempDir()}
	_, err := ExpandDirectories(&args, "[")
	if err == nil || !strings.Contains(err.Error(), "syntax error in pattern") {
		t.Fatalf("ExpandDirectories() error = %v, want invalid pattern error", err)
	}
}
