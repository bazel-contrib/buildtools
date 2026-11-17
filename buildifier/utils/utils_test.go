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
	"testing"

	"github.com/bazel-contrib/buildtools/v10/build"
	"github.com/bazel-contrib/buildtools/v10/warn"
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

func TestLinterCachesDirectoryListings(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "MODULE.bazel"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	libDir := filepath.Join(root, "lib")
	if err := os.Mkdir(libDir, 0o755); err != nil {
		t.Fatal(err)
	}

	lint := func(linter *Linter) []*warn.Finding {
		f, err := build.ParseBuild("BUILD", []byte(`foo(deps = ["//Lib:x"])`))
		if err != nil {
			t.Fatal(err)
		}
		f.WorkspaceRoot = root
		return linter.Lint(f, "warn", &[]string{"package-case"}, false)
	}

	linter := NewLinter()
	if findings := lint(linter); len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}

	if err := os.Remove(libDir); err != nil {
		t.Fatal(err)
	}
	// The same Linter still uses the cached directory listing.
	if findings := lint(linter); len(findings) != 1 {
		t.Errorf("got %d findings with the same Linter, want 1", len(findings))
	}
	// A new Linter lists the directory again.
	if findings := lint(NewLinter()); len(findings) != 0 {
		t.Errorf("got %d findings with a new Linter, want 0", len(findings))
	}
}
