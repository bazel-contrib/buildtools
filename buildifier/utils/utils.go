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
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazel-contrib/buildtools/v10/build"
	"github.com/bazel-contrib/buildtools/v10/warn"
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

// ExpandDirectories takes a list of file/directory names and returns a list with file names
// by traversing each directory recursively and searching for relevant Starlark files.
func ExpandDirectories(args *[]string) ([]string, error) {
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
		err = filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if skip(info) {
				return filepath.SkipDir
			}
			if !info.IsDir() && isStarlarkFile(info.Name()) {
				// Don't traverse into directory symlinks such as bazel-foo.bzl
				// for a project called foo.bzl.
				if info.Mode()&os.ModeSymlink != 0 {
					stat, err := os.Stat(path)
					if err != nil || stat.IsDir() {
						return nil
					}
				}
				files = append(files, path)
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
	case "repo":
		return build.ParseRepo
	case "vendor":
		return build.ParseVendor
	default:
		return build.ParseDefault
	}
}

// Linter calls the linter on files. A Linter is not safe for concurrent use.
type Linter struct {
	// dirs caches directory listings and lookups across all files linted by
	// this Linter. They don't change while buildifier is running, but may
	// change during the lifetime of a long-lived process, so the cache is
	// scoped to a Linter rather than to the process.
	dirs map[string]*dirCache
}

// dirCache caches the entries of a directory and the results of looking up
// names in it.
type dirCache struct {
	entries map[string]fs.DirEntry
	err     error
	lookups map[string]lookupResult
}

type lookupResult struct {
	entry fs.DirEntry
	err   error
}

// NewLinter returns a new Linter.
func NewLinter() *Linter {
	return &Linter{dirs: make(map[string]*dirCache)}
}

// lookup returns the entry of the directory dir in the local filesystem that
// name refers to, with the type of the file it refers to after following
// symlinks and junctions.
func (l *Linter) lookup(dir, name string) (fs.DirEntry, error) {
	d, ok := l.dirs[dir]
	if !ok {
		d = &dirCache{lookups: make(map[string]lookupResult)}
		var entries []fs.DirEntry
		entries, d.err = os.ReadDir(dir)
		d.entries = make(map[string]fs.DirEntry, len(entries))
		for _, entry := range entries {
			d.entries[entry.Name()] = entry
		}
		l.dirs[dir] = d
	}
	if d.err != nil {
		return nil, d.err
	}
	if r, ok := d.lookups[name]; ok {
		return r.entry, r.err
	}
	entry, err := lookupUncached(dir, name, d.entries)
	d.lookups[name] = lookupResult{entry, err}
	return entry, err
}

// lookupUncached returns the entry among the entries of the directory dir that
// name refers to, with the type of the file it refers to after following
// symlinks and junctions.
func lookupUncached(dir, name string, entries map[string]fs.DirEntry) (fs.DirEntry, error) {
	entry, ok := entries[name]
	if !ok {
		// The file system may still resolve name to an entry stored under a
		// different name, e.g. one that differs in case on a case-insensitive
		// file system. Find that entry by comparing file identities, which
		// leaves the rules for matching names to the file system.
		info, err := os.Lstat(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		for candidateName, candidate := range entries {
			candidateInfo, err := os.Lstat(filepath.Join(dir, candidateName))
			if err != nil || !os.SameFile(info, candidateInfo) {
				continue
			}
			if entry != nil {
				return nil, fmt.Errorf("%q refers to multiple entries of %q", name, dir)
			}
			entry = candidate
		}
		if entry == nil {
			return nil, fmt.Errorf("%q doesn't refer to an entry of %q", name, dir)
		}
	}
	// Resolve symlinks as well as Windows junctions, which are reported as
	// irregular files.
	if entry.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
		info, err := os.Stat(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		entry = fs.FileInfoToDirEntry(info)
	}
	return entry, nil
}

// getFileReader returns a *FileReader object that reads files from the local
// filesystem if the workspace root is known.
func (l *Linter) getFileReader(workspaceRoot string) *warn.FileReader {
	if workspaceRoot == "" {
		return nil
	}

	readFile := func(filename string) ([]byte, error) {
		// Use OS-specific path separators
		filename = strings.ReplaceAll(filename, "/", string(os.PathSeparator))
		path := filepath.Join(workspaceRoot, filename)

		return os.ReadFile(path)
	}

	lookup := func(dir, name string) (fs.DirEntry, error) {
		return l.lookup(filepath.Join(workspaceRoot, filepath.FromSlash(dir)), name)
	}

	return warn.NewFileReaderWithLookup(readFile, lookup)
}

// Lint calls the linter and returns a list of unresolved findings
func (l *Linter) Lint(f *build.File, lint string, warningsList *[]string, verbose bool) []*warn.Finding {
	fileReader := l.getFileReader(f.WorkspaceRoot)

	switch lint {
	case "warn":
		return warn.FileWarnings(f, *warningsList, nil, warn.ModeWarn, fileReader)
	case "fix":
		warn.FixWarnings(f, *warningsList, verbose, fileReader)
	}
	return nil
}

// Lint calls the linter and returns a list of unresolved findings
func Lint(f *build.File, lint string, warningsList *[]string, verbose bool) []*warn.Finding {
	return NewLinter().Lint(f, lint, warningsList, verbose)
}
