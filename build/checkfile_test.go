/*
Copyright 2016 Google LLC

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

package build

import (
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/bazel-contrib/buildtools/v10/testutils"
)

func TestFilesMatch(t *testing.T) {
	testdata := path.Join(os.Getenv("TEST_SRCDIR"), os.Getenv("TEST_WORKSPACE"), "build")

	generated, err := os.ReadFile(path.Join(testdata, "parse.y.baz.go"))
	if err != nil {
		t.Fatalf("ReadFile(%q) = %v", "parse.y.baz.go", err)
	}
	// The first line contains a path to the generator. We want to skip it
	// (as well as the next character, another '\n') to make the test more
	// robust.
	generated = generated[bytes.IndexByte(generated, '\n')+2:]

	checkedIn, err := os.ReadFile(path.Join(testdata, "parse.y.go"))
	if err != nil {
		t.Fatalf("ReadFile(%q) = %v", "parse.y.go", err)
	}

	d, err := testutils.Diff(generated, checkedIn)
	if err != nil {
		t.Fatalf("diff(generated, checkedIn) = %v", err)
	}
	if len(d) != 0 {
		t.Errorf("diff(generated, checkedIn) = %v", string(d))
	}
}
