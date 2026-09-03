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

package edit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bazelbuild/buildtools/build"
	"github.com/google/go-cmp/cmp"
)

func TestMovePackageDeclarationToTheTop(t *testing.T) {
	tests := []struct {
		input, expected string
		shouldMove      bool
	}{
		{`"""Docstring."""

load(":path.bzl", "x")

foo()

package(attr = "val")`,
			`"""Docstring."""

load(":path.bzl", "x")

package(attr = "val")

foo()`,
			true},
		{`"""Docstring."""

load(":path.bzl", "x")

package(attr = "val")

foo()`,
			`"""Docstring."""

load(":path.bzl", "x")

package(attr = "val")

foo()`,
			false},
		{`"""Docstring."""

load(":path.bzl", "x")

foo()`,
			`"""Docstring."""

load(":path.bzl", "x")

foo()`,
			false,
		},
		{`"""Docstring."""

# Some comment 1

load(":path.bzl", "some_target")

# Some comment 2

package_group(name = "my_group")

licenses(["my_license"])

some_target(name = "foo")

some_list = []

[
    some_target(name = "x")
    for x in some_list
]

package(default_visibility = ["//visibility:public"])`,
			`"""Docstring."""

# Some comment 1

load(":path.bzl", "some_target")

# Some comment 2

package_group(name = "my_group")

licenses(["my_license"])

package(default_visibility = ["//visibility:public"])

some_target(name = "foo")

some_list = []

[
    some_target(name = "x")
    for x in some_list
]
`,
			true},
		{`VISIBILITY = ["//visibility:public"]

foo()

package(default_visibility = VISIBILITY)`,
			`VISIBILITY = ["//visibility:public"]

foo()

package(default_visibility = VISIBILITY)`,
			false},
		{`VISIBILITY = "//visibility:public"

foo()

package(default_visibility = [VISIBILITY])`,
			`VISIBILITY = "//visibility:public"

foo()

package(default_visibility = [VISIBILITY])`,
			false},
	}

	for i, tst := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			bld, err := build.Parse("BUILD", []byte(tst.input))
			if err != nil {
				t.Fatalf("Failed to parse %s; %v", tst.input, err)
			}
			if result := movePackageDeclarationToTheTop(bld); result != tst.shouldMove {
				t.Errorf("TestMovePackageDeclarationToTheTop: expected %v, got %v", tst.shouldMove, result)
			}

			got := strings.TrimSpace(string(build.Format(bld)))
			want := strings.TrimSpace(tst.expected)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("TestMovePackageDeclarationToTheTop: (-want +got): %s", diff)
			}
		})
	}
}

// Test cases for the full fix process.
func TestFixAll(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "no empty package",
			input: `load(":path.bzl", "x")

x()
`,
			want: `load(":path.bzl", "x")

x()
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bld, err := build.Parse("BUILD", []byte(tc.input))
			if err != nil {
				t.Fatalf("Failed to parse %s; %v", tc.input, err)
			}
			FixFile(bld, "//who/cares", []string{})
			if diff := cmp.Diff(tc.want, string(build.Format(bld))); diff != "" {
				t.Errorf("%s: (-want +got): %s", tc.name, diff)
			}
		})
	}
}
