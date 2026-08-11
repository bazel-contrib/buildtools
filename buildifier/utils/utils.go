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

// Package utils contains shared methods that can be used by different implementations of
// buildifier binary
package utils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/warn"
)

func isStarlarkFile(name string) bool {
	ext := filepath.Ext(name)
	switch ext {
	case ".bazel", ".BUILD", ".bzl", ".sky", ".star":
		return true
	}

	switch ext {
	case ".oss":
		return strings.HasPrefix(name, "BUILD.") || strings.HasPrefix(name, "WORKSPACE.")
	}

	return name == "BUILD" || name == "WORKSPACE" || name == "WORKSPACE.bzlmod"
}

func skip(info os.FileInfo) bool {
	return info.IsDir() && info.Name() == ".git"
}

func normalizePathForMatch(value string) string {
	value = filepath.ToSlash(value)
	for strings.HasPrefix(value, "./") {
		value = strings.TrimPrefix(value, "./")
	}
	return value
}

// matchPath uses path.Match syntax, but allows '*' and '?' to match path separators.
func matchPath(pattern, filename string) (bool, error) {
	const separatorPlaceholder = "\x00"
	pattern = strings.ReplaceAll(normalizePathForMatch(pattern), "/", separatorPlaceholder)
	filename = strings.ReplaceAll(normalizePathForMatch(filename), "/", separatorPlaceholder)
	return path.Match(pattern, filename)
}

func isExcluded(filename string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
		matched, err := matchPath(pattern, filename)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

// ExpandDirectories takes a list of file/directory names and returns a list with file names
// by traversing each directory recursively and searching for relevant Starlark files. Paths
// matching any of the optional exclude patterns are skipped. Exclude patterns use path.Match
// syntax and are matched with slash separators on every platform.
func ExpandDirectories(args *[]string, excludePatterns ...string) ([]string, error) {
	for _, pattern := range excludePatterns {
		if _, err := matchPath(pattern, ""); err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", pattern, err)
		}
	}

	files := []string{}
	for _, arg := range *args {
		info, err := os.Stat(arg)
		if err != nil {
			return []string{}, err
		}
		if !info.IsDir() {
			files = append(files, arg)
			continue
		}
		err = filepath.Walk(arg, func(filename string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			excluded, err := isExcluded(filename, excludePatterns)
			if err != nil {
				return err
			}
			if info.IsDir() && (skip(info) || excluded) {
				return filepath.SkipDir
			}
			if excluded {
				return nil
			}
			if !info.IsDir() && isStarlarkFile(info.Name()) {
				// Don't traverse into directory symlinks such as bazel-foo.bzl
				// for a project called foo.bzl.
				if info.Mode()&os.ModeSymlink != 0 {
					stat, err := os.Stat(filename)
					if err != nil || stat.IsDir() {
						return nil
					}
				}
				files = append(files, filename)
			}
			return nil
		})
		if err != nil {
			return []string{}, err
		}
	}
	return files, nil
}

// GetParser returns a parser for a given file type
func GetParser(inputType string) func(filename string, data []byte) (*build.File, error) {
	switch inputType {
	case "build":
		return build.ParseBuild
	case "bzl":
		return build.ParseBzl
	case "auto":
		return build.Parse
	case "workspace":
		return build.ParseWorkspace
	case "module":
		return build.ParseModule
	default:
		return build.ParseDefault
	}
}

// getFileReader returns a *FileReader object that reads files from the local
// filesystem if the workspace root is known.
func getFileReader(workspaceRoot string) *warn.FileReader {
	if workspaceRoot == "" {
		return nil
	}

	readFile := func(filename string) ([]byte, error) {
		// Use OS-specific path separators
		filename = strings.ReplaceAll(filename, "/", string(os.PathSeparator))
		path := filepath.Join(workspaceRoot, filename)

		return os.ReadFile(path)
	}

	return warn.NewFileReader(readFile)
}

// Lint calls the linter and returns a list of unresolved findings
func Lint(f *build.File, lint string, warningsList *[]string, verbose bool) []*warn.Finding {
	fileReader := getFileReader(f.WorkspaceRoot)

	switch lint {
	case "warn":
		return warn.FileWarnings(f, *warningsList, nil, warn.ModeWarn, fileReader)
	case "fix":
		warn.FixWarnings(f, *warningsList, verbose, fileReader)
	}
	return nil
}
