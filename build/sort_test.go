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

package build

import (
	"testing"

	"github.com/bazelbuild/buildtools/tables"
)

// sortAndFormat parses src, applies sortStringExprs to the list in each
// top-level assignment, and returns the formatted result.
func sortAndFormat(t *testing.T, src string) string {
	t.Helper()
	f, err := ParseBuild("BUILD", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range f.Stmt {
		as, ok := stmt.(*AssignExpr)
		if !ok {
			t.Fatalf("statement is not an assignment: %v", stmt)
		}
		list, ok := as.RHS.(*ListExpr)
		if !ok {
			t.Fatalf("assignment RHS is not a list: %v", as.RHS)
		}
		list.List = sortStringExprs(list.List)
	}
	return string(Format(f))
}

func runSortTests(t *testing.T, cases map[string]struct{ src, want string }) {
	t.Helper()
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := sortAndFormat(t, tc.src)
			if got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestSortStringExprsOrdering(t *testing.T) {
	runSortTests(t, map[string]struct{ src, want string }{
		"separator sorts before dash": {
			src: `deps = [
    ":foo-bar",
    ":foo.bar",
]
`,
			want: `deps = [
    ":foo.bar",
    ":foo-bar",
]
`,
		},
		"separator sorts before plus": {
			src: `deps = [
    ":a+b",
    ":a.b",
]
`,
			want: `deps = [
    ":a.b",
    ":a+b",
]
`,
		},
		"colon separator sorts before digits": {
			src: `deps = [
    ":a5",
    ":a:2",
]
`,
			want: `deps = [
    ":a:2",
    ":a5",
]
`,
		},
		"dot and colon separators tie broken by raw value": {
			src: `deps = [
    ":a:b",
    ":a.b",
]
`,
			want: `deps = [
    ":a.b",
    ":a:b",
]
`,
		},
		"relative phase sorts before absolute": {
			src: `deps = [
    "//x",
    "/x",
]
`,
			want: `deps = [
    "/x",
    "//x",
]
`,
		},
		"empty string sorts first": {
			src: `deps = [
    "x",
    "",
    ":a",
]
`,
			want: `deps = [
    "",
    "x",
    ":a",
]
`,
		},
		"phases order relative then absolute then repo": {
			src: `deps = [
    "@r//x",
    "//x",
    ":x",
    "x",
]
`,
			want: `deps = [
    "x",
    ":x",
    "//x",
    "@r//x",
]
`,
		},
	})
}

func TestSortStringExprsComments(t *testing.T) {
	runSortTests(t, map[string]struct{ src, want string }{
		"before comment on first element is pinned to the top": {
			src: `deps = [
    # comment on b
    ":b",
    ":a",
]
`,
			want: `deps = [
    # comment on b
    ":a",
    ":b",
]
`,
		},
		"before comment on later element starts a new chunk": {
			src: `deps = [
    ":c",
    # comment on b
    ":b",
    ":a",
]
`,
			want: `deps = [
    ":c",
    # comment on b
    ":a",
    ":b",
]
`,
		},
		"suffix comments move with elements": {
			src: `deps = [
    ":b",  # comment on b
    ":a",  # comment on a
]
`,
			want: `deps = [
    ":a",  # comment on a
    ":b",  # comment on b
]
`,
		},
	})
}

func TestSortStringExprsChunks(t *testing.T) {
	runSortTests(t, map[string]struct{ src, want string }{
		"non-string element separates chunks": {
			src: `deps = [
    ":c",
    ":b",
    X,
    ":a",
]
`,
			want: `deps = [
    ":b",
    ":c",
    X,
    ":a",
]
`,
		},
	})
}

func TestSortStringExprsDeduplication(t *testing.T) {
	runSortTests(t, map[string]struct{ src, want string }{
		"duplicates are removed keeping the first occurrence": {
			src: `deps = [
    ":a",  # comment one
    ":b",
    ":a",  # comment two
]
`,
			want: `deps = [
    ":a",  # comment one
    ":b",
]
`,
		},
	})
}

func TestSortStringExprsStripLabelLeadingSlashes(t *testing.T) {
	tables.StripLabelLeadingSlashes = true
	defer func() { tables.StripLabelLeadingSlashes = false }()

	runSortTests(t, map[string]struct{ src, want string }{
		"plain values sort into the absolute phase": {
			src: `deps = [
    "@r//x",
    "x",
    ":a",
]
`,
			want: `deps = [
    ":a",
    "x",
    "@r//x",
]
`,
		},
	})
}
