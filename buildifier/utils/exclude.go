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
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bazel-contrib/buildtools/v10/build"
	"github.com/bazel-contrib/buildtools/v10/wspace"
)

// Excluder decides which paths are skipped during recursive file discovery.
//
// Patterns use the syntax of Bazel's glob(): '*' and '?' match within a single
// path segment, and '**' as a whole segment matches any number of segments.
// Patterns are relative to the workspace root, so that they mean the same
// thing wherever buildifier runs and wherever they are configured, e.g. in
// .buildifier.json. A pattern matching a directory excludes all of its
// contents, and a pattern matching a file excludes that file.
type Excluder struct {
	// root is the absolute directory that patterns are relative to.
	root string
	// patterns holds each pattern split into its path segments.
	patterns [][]string
}

// NewExcluder returns an Excluder for the given patterns, relative to the
// workspace root containing the working directory, or to the working
// directory if there is no workspace. If respectBazelignore is true, the
// paths listed in the workspace's .bazelignore and the patterns passed to
// ignore_directories() in its REPO.bazel are excluded too.
func NewExcluder(patterns []string, respectBazelignore bool) (*Excluder, error) {
	root, err := excludeRoot()
	if err != nil {
		return nil, err
	}
	return newExcluder(root, patterns, respectBazelignore)
}

func excludeRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if root, _ := wspace.FindWorkspaceRoot(wd); root != "" {
		return root, nil
	}
	return wd, nil
}

func newExcluder(root string, patterns []string, respectBazelignore bool) (*Excluder, error) {
	e := &Excluder{root: filepath.Clean(root)}
	for _, pattern := range patterns {
		if err := e.add(pattern); err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", pattern, err)
		}
	}
	if !respectBazelignore {
		return e, nil
	}
	ignored, err := readBazelignore(filepath.Join(e.root, ".bazelignore"))
	if err != nil {
		return nil, err
	}
	for _, dir := range ignored {
		if err := e.add(escapeGlob(dir)); err != nil {
			return nil, fmt.Errorf("invalid .bazelignore entry %q: %w", dir, err)
		}
	}
	ignored, err = readIgnoreDirectories(filepath.Join(e.root, "REPO.bazel"))
	if err != nil {
		return nil, err
	}
	for _, pattern := range ignored {
		if err := e.add(pattern); err != nil {
			return nil, fmt.Errorf("invalid ignore_directories() pattern %q in REPO.bazel: %w", pattern, err)
		}
	}
	return e, nil
}

func (e *Excluder) add(pattern string) error {
	if pattern == "" {
		return errors.New("empty pattern")
	}
	if strings.HasPrefix(pattern, "/") {
		return errors.New("patterns are relative to the workspace root")
	}
	segments := strings.Split(strings.TrimSuffix(pattern, "/"), "/")
	for _, segment := range segments {
		switch {
		case segment == "" || segment == "." || segment == "..":
			return fmt.Errorf("invalid path segment %q", segment)
		case segment != "**" && strings.Contains(segment, "**"):
			return errors.New("'**' must be a whole path segment")
		}
		if _, err := path.Match(segment, ""); err != nil {
			return err
		}
	}
	e.patterns = append(e.patterns, segments)
	return nil
}

// Excluded reports whether a path, absolute or relative to the working
// directory, is excluded.
func (e *Excluder) Excluded(filename string) bool {
	if e == nil || len(e.patterns) == 0 {
		return false
	}
	abs, err := filepath.Abs(filename)
	if err != nil {
		return false
	}
	return e.excludedAbs(abs)
}

// excludedAbs is Excluded for an absolute, clean path.
func (e *Excluder) excludedAbs(abs string) bool {
	if e == nil || len(e.patterns) == 0 {
		return false
	}
	rel, ok := relativeTo(e.root, abs)
	if !ok {
		return false
	}
	segments := strings.Split(rel, "/")
	for _, pattern := range e.patterns {
		if matchSegments(pattern, segments) {
			return true
		}
	}
	return false
}

// relativeTo returns the slash-separated path of abs relative to root, if abs
// is strictly inside root.
func relativeTo(root, abs string) (string, bool) {
	prefix := root
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	if !strings.HasPrefix(abs, prefix) || len(abs) == len(prefix) {
		return "", false
	}
	return filepath.ToSlash(abs[len(prefix):]), true
}

func matchSegments(pattern, segments []string) bool {
	if len(pattern) == 0 {
		return len(segments) == 0
	}
	if pattern[0] == "**" {
		return matchSegments(pattern[1:], segments) ||
			(len(segments) > 0 && matchSegments(pattern, segments[1:]))
	}
	if len(segments) == 0 {
		return false
	}
	// Patterns are validated when added, so there is no error to handle.
	if matched, _ := path.Match(pattern[0], segments[0]); !matched {
		return false
	}
	return matchSegments(pattern[1:], segments[1:])
}

// escapeGlob escapes the glob metacharacters of a literal path.
func escapeGlob(literal string) string {
	var b strings.Builder
	for _, r := range literal {
		switch r {
		case '*', '?', '[', '\\':
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// readBazelignore returns the paths listed in a .bazelignore file, if any.
func readBazelignore(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dirs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for strings.HasPrefix(line, "./") {
			line = strings.TrimPrefix(line, "./")
		}
		if line = strings.TrimSuffix(line, "/"); line != "" {
			dirs = append(dirs, line)
		}
	}
	return dirs, scanner.Err()
}

// readIgnoreDirectories returns the patterns passed to ignore_directories() in
// a REPO.bazel file, if any.
func readIgnoreDirectories(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f, err := build.ParseRepo(filename, data)
	if err != nil {
		return nil, err
	}

	var patterns []string
	for _, stmt := range f.Stmt {
		call, ok := stmt.(*build.CallExpr)
		if !ok {
			continue
		}
		if ident, ok := call.X.(*build.Ident); !ok || ident.Name != "ignore_directories" {
			continue
		}
		for _, arg := range call.List {
			if assign, ok := arg.(*build.AssignExpr); ok {
				arg = assign.RHS
			}
			list, ok := arg.(*build.ListExpr)
			if !ok {
				continue
			}
			for _, item := range list.List {
				if str, ok := item.(*build.StringExpr); ok {
					patterns = append(patterns, str.Value)
				}
			}
		}
	}
	return patterns, nil
}
