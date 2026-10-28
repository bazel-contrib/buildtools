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

package utils

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for filename, content := range files {
		path := filepath.Join(root, filepath.FromSlash(filename))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExcluderMatchesGlobSyntax(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		pattern string
		path    string
		want    bool
	}{
		{"vendor", "vendor", true},
		{"vendor", "a/vendor", false},
		{"vendor/", "vendor", true},
		{"**/node_modules", "node_modules", true},
		{"**/node_modules", "a/b/node_modules", true},
		{"**/node_modules", "a/node_modules_x", false},
		{"vendor/*", "vendor/x", true},
		{"vendor/*", "vendor/x/y", false},
		{"*.bzl", "defs.bzl", true},
		{"*.bzl", "pkg/defs.bzl", false},
		{"a/**/b.bzl", "a/b.bzl", true},
		{"a/**/b.bzl", "a/x/y/b.bzl", true},
		{"a/**", "a", true},
		{"a/**", "a/x/y", true},
		{"**", "anything/at/all", true},
		{"?.bzl", "a.bzl", true},
		{"?.bzl", "ab.bzl", false},
	} {
		e, err := newExcluder(root, []string{tc.pattern}, false)
		if err != nil {
			t.Fatalf("newExcluder(%q) returned error: %v", tc.pattern, err)
		}
		if got := e.excludedAbs(filepath.Join(root, filepath.FromSlash(tc.path))); got != tc.want {
			t.Errorf("pattern %q, path %q: excluded = %t, want %t", tc.pattern, tc.path, got, tc.want)
		}
	}
}

func TestExcluderNeverExcludesTheRootOrOutsidePaths(t *testing.T) {
	root := t.TempDir()
	e, err := newExcluder(root, []string{"**"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if e.excludedAbs(root) {
		t.Error("the root itself is excluded")
	}
	if e.excludedAbs(filepath.Dir(root)) {
		t.Error("a path outside the root is excluded")
	}
}

func TestExcluderRejectsInvalidPatterns(t *testing.T) {
	for _, pattern := range []string{"", "/abs", "a//b", "./a", "a/../b", "a**", "**b/c", "["} {
		if _, err := newExcluder(t.TempDir(), []string{pattern}, false); err == nil {
			t.Errorf("newExcluder(%q) succeeded, want an error", pattern)
		} else if !strings.Contains(err.Error(), "invalid exclude pattern") {
			t.Errorf("newExcluder(%q) error = %v, want an invalid exclude pattern error", pattern, err)
		}
	}
}

func TestExcluderRespectsBazelignoreAndIgnoreDirectories(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		".bazelignore": "# comment\n\nignored\n./dotted/\nlit*eral\n",
		"REPO.bazel":   "ignore_directories([\"**/node_modules\"])\n",
	})

	for _, respect := range []bool{true, false} {
		e, err := newExcluder(root, nil, respect)
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			path string
			want bool
		}{
			{"ignored", respect},
			{"dotted", respect},
			{"lit*eral", respect},
			// .bazelignore entries are literal paths, not globs.
			{"litteral", false},
			{"pkg/node_modules", respect},
			{"pkg/ignored", false},
		} {
			if got := e.excludedAbs(filepath.Join(root, filepath.FromSlash(tc.path))); got != tc.want {
				t.Errorf("respect=%t, path %q: excluded = %t, want %t", respect, tc.path, got, tc.want)
			}
		}
	}
}

func TestExpandDirectoriesExcludes(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"MODULE.bazel":                "",
		"BUILD":                       "",
		"included/defs.bzl":           "",
		"excluded/direct.bzl":         "",
		"excluded/nested/BUILD.bazel": "",
		"other/skip.sky":              "",
		"other/keep.star":             "",
		"unreadable/BUILD":            "",
	})
	if runtime.GOOS != "windows" {
		unreadable := filepath.Join(root, "unreadable")
		if err := os.Chmod(unreadable, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(unreadable, 0o755) })
	}
	t.Chdir(root)

	excluder, err := NewExcluder([]string{"excluded", "**/skip.sky", "unreadable"}, true)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"."}
	files, err := ExpandDirectories(&args, excluder)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"BUILD", "MODULE.bazel", filepath.Join("included", "defs.bzl"), filepath.Join("other", "keep.star")}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("ExpandDirectories() = %q, want %q", files, want)
	}
}

func TestExpandDirectoriesExcludesRelativeToTheWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"MODULE.bazel":             "",
		"pkg/BUILD":                "",
		"pkg/excluded/BUILD.bazel": "",
	})
	t.Chdir(filepath.Join(root, "pkg"))

	excluder, err := NewExcluder([]string{"pkg/excluded"}, true)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"."}
	files, err := ExpandDirectories(&args, excluder)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"BUILD"}; !reflect.DeepEqual(files, want) {
		t.Errorf("ExpandDirectories() = %q, want %q", files, want)
	}
}
