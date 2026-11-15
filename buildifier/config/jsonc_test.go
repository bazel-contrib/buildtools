/*
Copyright 2022 Google LLC

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

package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseJSONC(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want map[string]any
	}{
		{
			name: "line comment",
			in:   "{\"mode\": \"fix\"} // trailing comment\n",
			want: map[string]any{"mode": "fix"},
		},
		{
			name: "block comment",
			in:   `{"mode": /* inline */ "fix"}`,
			want: map[string]any{"mode": "fix"},
		},
		{
			name: "comment-like sequences in strings",
			in:   `{"url": "http://example.com", "note": "use // sparingly /* ok */"}`,
			want: map[string]any{
				"url":  "http://example.com",
				"note": "use // sparingly /* ok */",
			},
		},
		{
			name: "trailing comma",
			in:   `{"mode": "fix",}`,
			want: map[string]any{"mode": "fix"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := parseJSONC([]byte(tc.in))
			if err != nil {
				t.Fatalf("parseJSONC() error = %v", err)
			}
			if !json.Valid(data) {
				t.Fatalf("parseJSONC() = invalid JSON %q", data)
			}

			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for key, want := range tc.want {
				if got[key] != want {
					t.Fatalf("got[%q] = %v, want %v", key, got[key], want)
				}
			}
		})
	}
}

func TestLoadReaderJSONC(t *testing.T) {
	input := `{
  // Lint settings for the workspace.
  "mode": "check",
  /* Keep lint off until warnings are triaged. */
  "lint": "off",
  "warnings": "native-cc-binary", // single warning for now
}`

	c := New()
	if err := c.LoadReader(strings.NewReader(input)); err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}

	if c.Mode != "check" {
		t.Errorf("Mode = %q, want %q", c.Mode, "check")
	}
	if c.Lint != "off" {
		t.Errorf("Lint = %q, want %q", c.Lint, "off")
	}
	if c.Warnings != "native-cc-binary" {
		t.Errorf("Warnings = %q, want %q", c.Warnings, "native-cc-binary")
	}
}
