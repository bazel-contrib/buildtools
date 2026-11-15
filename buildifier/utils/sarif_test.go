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
	"encoding/json"
	"strings"
	"testing"

	"github.com/bazel-contrib/buildtools/v10/build"
	"github.com/bazel-contrib/buildtools/v10/warn"
)

func TestFormatSarif(t *testing.T) {
	finding := &warn.Finding{
		Start:    build.Position{Line: 3, LineRune: 5},
		End:      build.Position{Line: 3, LineRune: 10},
		Category: "integer-division",
		Message:  "msg",
		URL:      "https://example.com#integer-division",
	}
	good := NewFileDiagnostics("a/good.bzl", nil)
	warned := NewFileDiagnostics("a/warn.bzl", []*warn.Finding{finding, finding})
	unformatted := NewFileDiagnostics("a/fmt.bzl", nil)
	unformatted.Formatted = false
	invalid := InvalidFileDiagnostics("a/bad.bzl")

	out := NewDiagnostics(good, warned, unformatted, invalid).Format("sarif", false)
	var log sarifLog
	if err := json.Unmarshal([]byte(out), &log); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if log.Version != "2.1.0" || len(log.Runs) != 1 {
		t.Fatalf("unexpected log header: %+v", log)
	}
	run := log.Runs[0]
	if len(run.Results) != 4 {
		t.Fatalf("got %d results, want 4", len(run.Results))
	}
	if len(run.Tool.Driver.Rules) != 3 {
		t.Errorf("got %d rules, want 3 (deduplicated)", len(run.Tool.Driver.Rules))
	}
	r := run.Results[0]
	reg := r.Locations[0].PhysicalLocation.Region
	if r.RuleID != "integer-division" || r.Level != "warning" || reg == nil ||
		reg.StartLine != 3 || reg.StartColumn != 5 || reg.EndColumn != 10 {
		t.Errorf("unexpected warning result: %+v", r)
	}
	if run.Results[2].RuleID != "format" || run.Results[2].Locations[0].PhysicalLocation.Region != nil {
		t.Errorf("unexpected format result: %+v", run.Results[2])
	}
	if run.Results[3].RuleID != "parse-error" || run.Results[3].Level != "error" {
		t.Errorf("unexpected parse result: %+v", run.Results[3])
	}
	for i, res := range run.Results {
		if run.Tool.Driver.Rules[res.RuleIndex].ID != res.RuleID {
			t.Errorf("result %d has inconsistent ruleIndex", i)
		}
	}
}

func TestFormatSarifEmpty(t *testing.T) {
	out := NewDiagnostics(NewFileDiagnostics("a.bzl", nil)).Format("sarif", true)
	if !strings.Contains(out, `"results": []`) || !strings.Contains(out, `"rules": []`) {
		t.Errorf("expected empty arrays, got:\n%s", out)
	}
}

func TestSarifURI(t *testing.T) {
	for in, want := range map[string]string{
		"a/b.bzl":    "a/b.bzl",
		"a b/c.bzl":  "a%20b/c.bzl",
		"<stdin>":    "%3Cstdin%3E",
		"/tmp/x.bzl": "file:///tmp/x.bzl",
		"a:b/c.bzl":  "./a:b/c.bzl",
	} {
		if got := sarifURI(in); got != want {
			t.Errorf("sarifURI(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatSarifEscapesBracesAndSetsColumnKind(t *testing.T) {
	finding := &warn.Finding{
		Start:    build.Position{Line: 1, LineRune: 1},
		End:      build.Position{Line: 1, LineRune: 2},
		Category: "c",
		Message:  `use {"a": 1}`,
	}
	out := NewDiagnostics(NewFileDiagnostics("a.bzl", []*warn.Finding{finding})).Format("sarif", false)
	var log sarifLog
	if err := json.Unmarshal([]byte(out), &log); err != nil {
		t.Fatal(err)
	}
	if got, want := log.Runs[0].Results[0].Message.Text, `use {{"a": 1}}`; got != want {
		t.Errorf("message = %q, want %q", got, want)
	}
	if log.Runs[0].ColumnKind != "unicodeCodePoints" {
		t.Errorf("columnKind = %q", log.Runs[0].ColumnKind)
	}
}
