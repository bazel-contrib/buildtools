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

package warn

import (
	"io/fs"

	"github.com/bazel-contrib/buildtools/v10/build"
)

// FileReader is a class that can read an arbitrary Starlark file
// from the repository and cache the results.
type FileReader struct {
	cache    map[string]*build.File
	readFile func(string) ([]byte, error)
	lookup   func(dir, name string) (fs.DirEntry, error)
}

// NewFileReader creates and initializes a FileReader instance with a
// custom readFile function that can read an arbitrary file in the
// repository using a path relative to the workspace root
// (OS-independent, with forward slashes).
func NewFileReader(readFile func(string) ([]byte, error)) *FileReader {
	return &FileReader{
		cache:    make(map[string]*build.File),
		readFile: readFile,
	}
}

// NewFileReaderWithLookup is like NewFileReader, but additionally accepts a
// custom lookup function that returns the entry of a directory in the
// repository that a name refers to according to the file system. The directory
// is given as a path relative to the workspace root (OS-independent, with
// forward slashes, empty for the workspace root itself). The returned entry has
// the name stored in the directory, which can differ from the name that was
// looked up, e.g. in case on a case-insensitive file system, and the type of
// the file it refers to after following symlinks. lookup returns an error if
// the name doesn't refer to exactly one entry.
func NewFileReaderWithLookup(readFile func(string) ([]byte, error), lookup func(dir, name string) (fs.DirEntry, error)) *FileReader {
	fr := NewFileReader(readFile)
	fr.lookup = lookup
	return fr
}

// retrieveFile reads a Starlark file using only the readFile method
// (without using the cache).
func (fr *FileReader) retrieveFile(filename string) *build.File {
	contents, err := fr.readFile(filename)
	if err != nil {
		return nil
	}

	file, err := build.Parse(filename, contents)
	if err != nil {
		return nil
	}
	file.Path = filename

	return file
}

// GetFile reads a Starlark file from the repository or the cache.
// Returns nil if the file is not found or not valid.
func (fr *FileReader) GetFile(pkg, label string) *build.File {
	filename := label
	if pkg != "" {
		filename = pkg + "/" + label
	}

	// Try to retrieve from the cache
	if file, ok := fr.cache[filename]; ok {
		return file
	}
	file := fr.retrieveFile(filename)
	if file != nil {
		file.Pkg = pkg
		file.Label = label
	}
	fr.cache[filename] = file
	return file
}

// canLookUp returns whether the FileReader is able to look up directory
// entries.
func (fr *FileReader) canLookUp() bool {
	return fr != nil && fr.lookup != nil
}

// lookUpEntry returns the entry of a directory in the repository that name
// refers to according to the file system, or nil if there is no such entry.
func (fr *FileReader) lookUpEntry(dir, name string) fs.DirEntry {
	if !fr.canLookUp() {
		return nil
	}
	entry, err := fr.lookup(dir, name)
	if err != nil {
		return nil
	}
	return entry
}
