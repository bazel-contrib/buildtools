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
	"os/exec"
	"path/filepath"
	"runtime"
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

// isCaseInsensitive returns whether the file system containing dir resolves
// names case-insensitively.
func isCaseInsensitive(t *testing.T, dir string) bool {
	t.Helper()
	probe := filepath.Join(dir, "case-probe")
	if err := os.Mkdir(probe, 0o755); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(probe)
	_, err := os.Lstat(filepath.Join(dir, "CASE-PROBE"))
	return err == nil
}

func TestLinterLookup(t *testing.T) {
	root := t.TempDir()
	caseInsensitive := isCaseInsensitive(t, root)
	if err := os.Mkdir(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// The same name in Unicode normalization forms D and C.
	decomposed, composed := "e\u0301", "\u00e9"
	if err := os.Mkdir(filepath.Join(root, decomposed), 0o755); err != nil {
		t.Fatal(err)
	}

	linter := NewLinter()
	// check looks up name and verifies the name and type of the resulting entry.
	// An empty wantName means that the lookup is expected to fail.
	check := func(name, wantName string, wantDir bool) {
		t.Helper()
		entry, err := linter.lookup(root, name)
		if wantName == "" {
			if err == nil {
				t.Errorf("lookup(%q) = %q, want error", name, entry.Name())
			}
			return
		}
		if err != nil {
			t.Errorf("lookup(%q) failed: %v", name, err)
			return
		}
		if entry.Name() != wantName || entry.IsDir() != wantDir {
			t.Errorf("lookup(%q) = %q (directory: %t), want %q (directory: %t)", name, entry.Name(), entry.IsDir(), wantName, wantDir)
		}
	}

	check("lib", "lib", true)
	check("file.txt", "file.txt", false)
	check("missing", "", false)
	if caseInsensitive {
		// Different wrong spellings of the same name are looked up independently.
		check("Lib", "lib", true)
		check("LIB", "lib", true)
		check("File.TXT", "file.txt", false)
	} else {
		check("Lib", "", false)
		check("LIB", "", false)
	}

	// Matching names is left to the file system, which may also ignore
	// differences in Unicode normalization (e.g. APFS on macOS).
	if _, err := os.Lstat(filepath.Join(root, composed)); err == nil {
		check(composed, decomposed, true)
	} else {
		check(composed, "", false)
	}

	// Symlinks keep their own name, but have the type of the file they point to.
	if err := os.Symlink("lib", filepath.Join(root, "linked")); err != nil {
		t.Logf("Skipping symlink checks: %v", err)
		return
	}
	if err := os.Symlink("file.txt", filepath.Join(root, "filelink")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing", filepath.Join(root, "broken")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("loop", filepath.Join(root, "loop")); err != nil {
		t.Fatal(err)
	}
	linter = NewLinter()
	check("linked", "linked", true)
	check("filelink", "filelink", false)
	check("broken", "", false)
	check("loop", "", false)
	if caseInsensitive {
		check("LINKED", "linked", true)
	}
}

func TestLinterLookupJunction(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("junctions only exist on Windows")
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Unlike symlinks, junctions can be created without special privileges.
	junction := filepath.Join(root, "linked")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", junction, filepath.Join(root, "lib")).CombinedOutput(); err != nil {
		t.Fatalf("failed to create junction: %v\n%s", err, out)
	}

	entry, err := NewLinter().lookup(root, "LINKED")
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if entry.Name() != "linked" || !entry.IsDir() {
		t.Errorf("lookup = %q (directory: %t), want %q (directory: true)", entry.Name(), entry.IsDir(), "linked")
	}
}

func TestLinterCachesLookups(t *testing.T) {
	root := t.TempDir()
	if !isCaseInsensitive(t, root) {
		t.Skip("requires a case-insensitive file system")
	}
	if err := os.WriteFile(filepath.Join(root, "MODULE.bazel"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	libDir := filepath.Join(root, "lib")
	if err := os.Mkdir(libDir, 0o755); err != nil {
		t.Fatal(err)
	}

	lint := func(linter *Linter) []*warn.Finding {
		f, err := build.ParseBuild("BUILD", []byte(`foo(deps = ["//Lib:x", "//LIB:y"])`))
		if err != nil {
			t.Fatal(err)
		}
		f.WorkspaceRoot = root
		return linter.Lint(f, "warn", &[]string{"label-case"}, false)
	}

	linter := NewLinter()
	if findings := lint(linter); len(findings) != 2 {
		t.Fatalf("got %d findings, want 2", len(findings))
	}

	if err := os.Remove(libDir); err != nil {
		t.Fatal(err)
	}
	// The same Linter still uses the cached lookups of both spellings.
	if findings := lint(linter); len(findings) != 2 {
		t.Errorf("got %d findings with the same Linter, want 2", len(findings))
	}
	// A new Linter looks up the directory again.
	if findings := lint(NewLinter()); len(findings) != 0 {
		t.Errorf("got %d findings with a new Linter, want 0", len(findings))
	}
}
