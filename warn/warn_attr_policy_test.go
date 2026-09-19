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

package warn

import (
	"testing"
)

func attrPolicyTestRules() []AttrPolicyRuleCompiled {
	return []AttrPolicyRuleCompiled{
		{
			Name:         "no-eternal-timeout",
			RuleKinds:    []string{"*_test"},
			Attr:         "timeout",
			Family:       AttrPolicyScalarFamily,
			ForbidValues: []string{"eternal"},
			Allowlist: []AttrPolicyAllowlistPattern{
				{Kind: AttrPolicyAllowExact, Pkg: "test/package", Target: "big_test"},
			},
		},
		{
			Name:            "no-exclusive-tests",
			RuleKinds:       []string{"*_test"},
			Attr:            "tags",
			Family:          AttrPolicyListFamily,
			ForbidListItems: []string{"exclusive"},
		},
		{
			Name:         "no-local-tests",
			RuleKinds:    []string{"*_test"},
			Attr:         "local",
			Family:       AttrPolicyScalarFamily,
			ForbidValues: []string{"True"},
		},
		{
			Name:  "no-no-cache",
			Attr:  "execution_requirements",
			Family: AttrPolicyDictFamily,
			ForbidDictEntries: map[string]string{
				"no-cache": "1",
			},
		},
		{
			Name:      "max-shard-count",
			RuleKinds: []string{"*_test"},
			Attr:      "shard_count",
			Family:    AttrPolicyNumericFamily,
			MaxValue:  intPtr(50),
			Allowlist: []AttrPolicyAllowlistPattern{
				{Kind: AttrPolicyAllowExact, Pkg: "test/package", Target: "massive_test"},
			},
		},
	}
}

func intPtr(v int) *int {
	return &v
}

func TestAttrPolicyWarning(t *testing.T) {
	old := AttrPolicyConfig
	defer func() { SetAttrPolicy(old) }()
	SetAttrPolicy(attrPolicyTestRules())

	checkFindings(t, "attr-policy", `
cc_test(name = "ok", timeout = "short")
cc_test(name = "bad", timeout = "eternal")
cc_test(name = "big_test", timeout = "eternal")
cc_test(name = "exclusive", tags = ["exclusive"])
cc_test(name = "local", local = True)
cc_test(name = "cached", execution_requirements = {"no-cache": "1"})
cc_test(name = "sharded", shard_count = 100)
cc_test(name = "massive_test", shard_count = 100)
cc_library(name = "lib", timeout = "eternal")
`, []string{
		`:2: [no-eternal-timeout] attribute "timeout" must not be "eternal"`,
		`:4: [no-exclusive-tests] attribute "tags" must not contain "exclusive"`,
		`:5: [no-local-tests] attribute "local" must not be "True"`,
		`:6: [no-no-cache] attribute "execution_requirements" must not contain "no-cache": "1"`,
		`:7: [max-shard-count] attribute "shard_count" must be <= 50`,
	}, scopeBuild)

	SetAttrPolicy([]AttrPolicyRuleCompiled{
		{
			Name:         "recursive-allow",
			RuleKinds:    []string{"*_test"},
			Attr:         "timeout",
			Family:       AttrPolicyScalarFamily,
			ForbidValues: []string{"eternal"},
			Allowlist: []AttrPolicyAllowlistPattern{
				{Kind: AttrPolicyAllowRecursive, Pkg: "slow"},
			},
		},
	})
	cleanup := setUpTestPackage("slow/nested")
	defer cleanup()
	checkFindings(t, "attr-policy", `
cc_test(name = "allowed", timeout = "eternal")
`, nil, scopeBuild)

	cleanup2 := setUpTestPackage("fast")
	defer cleanup2()
	checkFindings(t, "attr-policy", `
cc_test(name = "bad", timeout = "eternal")
`, []string{
		`:1: [recursive-allow] attribute "timeout" must not be "eternal"`,
	}, scopeBuild)

	SetAttrPolicy(nil)
	checkFindings(t, "attr-policy", `
cc_test(name = "bad", timeout = "eternal")
cc_test(name = "sharded", shard_count = 100)
cc_test(name = "ok", shard_count = 4)
cc_library(name = "lib", licenses = ["notice"])
cc_binary(name = "bin", licenses = ["notice"], output_licenses = ["notice"])
cc_library(name = "lib2", output_licenses = ["notice"])
`, []string{
		`:2: [max-shard-count] Having more than 50 shards is indicative of poor test organization. Please reduce the number of shards.`,
		`:4: [no-licenses] The licenses attribute is deprecated; use package(default_applicable_licenses = ...) and applicable_licenses on targets instead (https://github.com/bazelbuild/bazel/issues/188).`,
		`:5: [no-licenses] The licenses attribute is deprecated; use package(default_applicable_licenses = ...) and applicable_licenses on targets instead (https://github.com/bazelbuild/bazel/issues/188).`,
		`:5: [no-output-licenses] The output_licenses attribute is deprecated; use applicable_licenses instead (https://github.com/bazelbuild/bazel/issues/7444).`,
	}, scopeBuild)
}

func TestAttrPolicyForbidPresence(t *testing.T) {
	old := AttrPolicyConfig
	defer func() { SetAttrPolicy(old) }()
	SetAttrPolicy([]AttrPolicyRuleCompiled{
		{
			Name:   "no-licenses",
			Attr:   "licenses",
			Family: AttrPolicyForbidPresenceFamily,
		},
		{
			Name:      "no-output-licenses",
			RuleKinds: append([]string(nil), defaultOutputLicensesRuleKinds...),
			Attr:      "output_licenses",
			Family:    AttrPolicyForbidPresenceFamily,
		},
	})

	checkFindings(t, "attr-policy", `
cc_library(name = "lib", licenses = ["notice"])
cc_binary(name = "bin", output_licenses = ["notice"])
cc_library(name = "lib2", output_licenses = ["notice"])
cc_library(name = "clean")
`, []string{
		`:1: [no-licenses] attribute "licenses" must not be set`,
		`:2: [no-output-licenses] attribute "output_licenses" must not be set`,
	}, scopeBuild)
}

func TestAttrPolicyRequired(t *testing.T) {
	old := AttrPolicyConfig
	defer func() { SetAttrPolicy(old) }()
	SetAttrPolicy([]AttrPolicyRuleCompiled{
		{
			Name:     "needs-timeout",
			Attr:     "timeout",
			Family:   AttrPolicyScalarFamily,
			Required: true,
		},
	})
	checkFindings(t, "attr-policy", `
cc_test(name = "missing")
cc_test(name = "present", timeout = "short")
`, []string{
		`:1: [needs-timeout] attribute "timeout" is required`,
	}, scopeBuild)
}

func TestAttrPolicyRuleKindGlob(t *testing.T) {
	old := AttrPolicyConfig
	defer func() { SetAttrPolicy(old) }()
	SetAttrPolicy([]AttrPolicyRuleCompiled{
		{
			Name:         "tests-only",
			RuleKinds:    []string{"*_test"},
			Attr:         "timeout",
			Family:       AttrPolicyScalarFamily,
			ForbidValues: []string{"eternal"},
		},
	})
	checkFindings(t, "attr-policy", `
cc_test(name = "bad", timeout = "eternal")
cc_library(name = "lib", timeout = "eternal")
`, []string{
		`:1: [tests-only] attribute "timeout" must not be "eternal"`,
	}, scopeBuild)
}

func TestAttrPolicyAllowListItems(t *testing.T) {
	old := AttrPolicyConfig
	defer func() { SetAttrPolicy(old) }()
	SetAttrPolicy([]AttrPolicyRuleCompiled{
		{
			Name:           "allowed-tags",
			Attr:           "tags",
			Family:         AttrPolicyListFamily,
			AllowListItems: []string{"manual", "assistant-ds*", "exclusive"},
		},
	})

	checkFindings(t, "attr-policy", `
cc_test(name = "ok", tags = ["manual", "assistant-ds-community", "exclusive"])
cc_test(name = "bad", tags = ["my-new-tag"])
`, []string{
		`:2: [allowed-tags] attribute "tags" must not contain "my-new-tag"; allowed values are "manual", "assistant-ds*", "exclusive"`,
	}, scopeBuild)
}

func TestAllowListItemMatches(t *testing.T) {
	tests := []struct {
		allowed []string
		item    string
		want    bool
	}{
		{[]string{"manual", "exclusive"}, "manual", true},
		{[]string{"manual", "exclusive"}, "exclusive", true},
		{[]string{"manual", "exclusive"}, "other", false},
		{[]string{"assistant-ds*"}, "assistant-ds", true},
		{[]string{"assistant-ds*"}, "assistant-ds-community", true},
		{[]string{"assistant-ds*"}, "assistant-hero", false},
	}
	for _, tc := range tests {
		if got := allowListItemMatches(tc.allowed, tc.item); got != tc.want {
			t.Errorf("allowListItemMatches(%v, %q) = %v, want %v", tc.allowed, tc.item, got, tc.want)
		}
	}
}

func TestAllowlistPatternMatches(t *testing.T) {
	tests := []struct {
		pattern AttrPolicyAllowlistPattern
		label   string
		pkg     string
		want    bool
	}{
		{AttrPolicyAllowlistPattern{Kind: AttrPolicyAllowAll}, "//any:target", "any", true},
		{AttrPolicyAllowlistPattern{Kind: AttrPolicyAllowExact, Pkg: "foo", Target: "bar"}, "//foo:bar", "foo", true},
		{AttrPolicyAllowlistPattern{Kind: AttrPolicyAllowExact, Pkg: "foo", Target: "bar"}, "//foo:baz", "foo", false},
		{AttrPolicyAllowlistPattern{Kind: AttrPolicyAllowPackageAll, Pkg: "foo"}, "//foo:bar", "foo", true},
		{AttrPolicyAllowlistPattern{Kind: AttrPolicyAllowPackageAll, Pkg: "foo"}, "//bar:baz", "bar", false},
		{AttrPolicyAllowlistPattern{Kind: AttrPolicyAllowRecursive, Pkg: "slow"}, "//slow:big", "slow", true},
		{AttrPolicyAllowlistPattern{Kind: AttrPolicyAllowRecursive, Pkg: "slow"}, "//slow/nested:big", "slow/nested", true},
		{AttrPolicyAllowlistPattern{Kind: AttrPolicyAllowRecursive, Pkg: "slow"}, "//fast:big", "fast", false},
	}
	for _, tc := range tests {
		if got := allowlistPatternMatches(tc.pattern, tc.label, tc.pkg); got != tc.want {
			t.Errorf("allowlistPatternMatches(%+v, %q, %q) = %v, want %v", tc.pattern, tc.label, tc.pkg, got, tc.want)
		}
	}
}
