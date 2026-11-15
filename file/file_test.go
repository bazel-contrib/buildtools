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

package file

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileBeneath(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("subdir", filepath.Join(root, "inside")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err != nil {
		t.Fatal(err)
	}

	if err := writeFileBeneath(root, "inside/file", []byte("data"), 0644); err != nil {
		t.Fatalf("write through symlink inside root: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "subdir", "file")); err != nil || string(got) != "data" {
		t.Fatalf("read written file: got %q, %v", got, err)
	}

	if err := writeFileBeneath(root, "outside/file", []byte("data"), 0644); err == nil {
		t.Fatal("write through symlink outside root succeeded")
	}
	if _, err := os.Stat(filepath.Join(outside, "file")); !os.IsNotExist(err) {
		t.Fatalf("file was created outside root: %v", err)
	}
}
