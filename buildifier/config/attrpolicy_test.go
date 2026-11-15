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

package config

import (
	"strings"
	"testing"

	"github.com/bazelbuild/buildtools/warn"
)

func TestCompileAttrPolicy(t *testing.T) {
	for name, tc := range map[string]struct {
		policy   *AttrPolicy
		wantLen  int
		wantName string
	}{
		"nil uses default shard_count rule": {
			policy:   nil,
			wantLen:  3,
			wantName: "max-shard-count",
		},
		"empty rules uses default shard_count rule": {
			policy:   &AttrPolicy{},
			wantLen:  3,
			wantName: "max-shard-count",
		},
	} {
		t.Run(name, func(t *testing.T) {
			compiled, err := compileAttrPolicy(tc.policy)
			if err != nil {
				t.Fatalf("compileAttrPolicy() error = %v", err)
			}
			if len(compiled) != tc.wantLen {
				t.Fatalf("len(compiled) = %d, want %d", len(compiled), tc.wantLen)
			}
			if compiled[0].Name != tc.wantName {
				t.Fatalf("compiled[0].Name = %q, want %q", compiled[0].Name, tc.wantName)
			}
			if compiled[0].MaxValue == nil || *compiled[0].MaxValue != 50 {
				t.Fatalf("compiled[0].MaxValue = %+v", compiled[0].MaxValue)
			}
		})
	}

	policy := &AttrPolicy{
		Rules: []AttrPolicyRule{
			{
				Name:         "no-eternal-timeout",
				RuleKinds:    []string{"*_test"},
				Attr:         "timeout",
				ForbidValues: []string{"eternal"},
				Allowlist:    []string{"//slow/...", "//foo:big_test"},
			},
			{
				Name:      "max-shard-count",
				RuleKinds: []string{"*_test"},
				Attr:      "shard_count",
				MaxValue:  intPtr(50),
			},
		},
	}
	compiled, err := compileAttrPolicy(policy)
	if err != nil {
		t.Fatalf("compileAttrPolicy() error = %v", err)
	}
	if len(compiled) != 2 {
		t.Fatalf("len(compiled) = %d, want 2", len(compiled))
	}
	if compiled[0].Name != "no-eternal-timeout" || compiled[0].Family != warn.AttrPolicyScalarFamily {
		t.Fatalf("compiled[0] = %+v", compiled[0])
	}
	if compiled[1].MaxValue == nil || *compiled[1].MaxValue != 50 {
		t.Fatalf("compiled[1].MaxValue = %+v", compiled[1].MaxValue)
	}
}

func TestCompileAttrPolicyValidation(t *testing.T) {
	for name, tc := range map[string]struct {
		policy  *AttrPolicy
		wantErr string
	}{
		"duplicate name": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Name: "x", Attr: "timeout", ForbidValues: []string{"eternal"}},
				{Name: "x", Attr: "timeout", ForbidValues: []string{"eternal"}},
			}},
			wantErr: `duplicate name`,
		},
		"missing name": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Attr: "timeout", ForbidValues: []string{"eternal"}},
			}},
			wantErr: `name is required`,
		},
		"missing attr": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Name: "x", ForbidValues: []string{"eternal"}},
			}},
			wantErr: `attr is required`,
		},
		"no constraints": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Name: "x", Attr: "timeout"},
			}},
			wantErr: `at least one constraint`,
		},
		"mixed families": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Name: "x", Attr: "timeout", ForbidValues: []string{"eternal"}, ForbidListItems: []string{"exclusive"}},
			}},
			wantErr: `cannot mix scalar, list, dict, numeric, and presence`,
		},
		"numeric min greater than max": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Name: "x", Attr: "shard_count", MinValue: intPtr(10), MaxValue: intPtr(5)},
			}},
			wantErr: `minValue must be <= maxValue`,
		},
		"bad ruleKinds glob": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Name: "x", Attr: "timeout", ForbidValues: []string{"eternal"}, RuleKinds: []string{"["}},
			}},
			wantErr: `malformed ruleKinds`,
		},
		"bad allowlist": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Name: "x", Attr: "timeout", ForbidValues: []string{"eternal"}, Allowlist: []string{"@repo//foo:bar"}},
			}},
			wantErr: `repository-qualified`,
		},
		"required only": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Name: "x", Attr: "timeout", Required: true},
			}},
		},
		"forbid presence only": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Name: "x", Attr: "licenses", ForbidPresence: true},
			}},
		},
		"allow list items only": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Name: "x", Attr: "tags", AllowListItems: []string{"manual", "assistant-ds*"}},
			}},
		},
		"forbid presence and required": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Name: "x", Attr: "licenses", ForbidPresence: true, Required: true},
			}},
			wantErr: `forbidPresence and required cannot both be true`,
		},
		"forbid presence mixed with scalar": {
			policy: &AttrPolicy{Rules: []AttrPolicyRule{
				{Name: "x", Attr: "timeout", ForbidPresence: true, ForbidValues: []string{"eternal"}},
			}},
			wantErr: `cannot mix scalar, list, dict, numeric, and presence`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := compileAttrPolicy(tc.policy)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestParseAllowlistPattern(t *testing.T) {
	for name, tc := range map[string]struct {
		entry   string
		wantErr string
		kind    warn.AttrPolicyAllowlistKind
		pkg     string
		target  string
	}{
		"all":         {entry: "//...", kind: warn.AttrPolicyAllowAll},
		"exact":       {entry: "//foo:bar", kind: warn.AttrPolicyAllowExact, pkg: "foo", target: "bar"},
		"package":     {entry: "//pkg", kind: warn.AttrPolicyAllowExact, pkg: "pkg", target: "pkg"},
		"package all": {entry: "//pkg:all", kind: warn.AttrPolicyAllowPackageAll, pkg: "pkg"},
		"recursive":   {entry: "//slow/...", kind: warn.AttrPolicyAllowRecursive, pkg: "slow"},
		"repo qualified": {entry: "@repo//foo:bar", wantErr: "repository-qualified"},
		"bare ellipsis":  {entry: "...", wantErr: "bare ..."},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := parseAllowlistPattern(tc.entry)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Kind != tc.kind || got.Pkg != tc.pkg || got.Target != tc.target {
				t.Fatalf("parseAllowlistPattern(%q) = %+v", tc.entry, got)
			}
		})
	}
}

func intPtr(v int) *int {
	return &v
}
