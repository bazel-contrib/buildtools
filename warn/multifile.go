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
	"github.com/bazel-contrib/buildtools/v10/build"
)

// FileReader is a class that can read an arbitrary Starlark file
// from the repository and cache the results.
type FileReader struct {
	cache    map[string]*build.File
	readFile func(string) ([]byte, error)
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
