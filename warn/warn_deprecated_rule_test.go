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

import "testing"

func TestDeprecatedRuleViaUseRepoRule(t *testing.T) {
	defer setUpFileReader(map[string]string{
		"test/package/rules.bzl": `
def _my_rule_v1_impl(ctx):
  """
  Some rule.

  Deprecated:
    please use my_rule_v3 instead.
  """
  pass

my_rule_v1 = repository_rule(
  implementation = _my_rule_v1_impl,
)

def _my_rule_v2_impl(ctx):
  """
  Some rule.

  Deprecated:
    please use my_rule_v3 instead.
  """
  pass

my_rule_v2 = repository_rule(
  implementation = _my_rule_v2_impl,
)

def _my_rule_v3_impl(ctx):
  pass

my_rule_v3 = repository_rule(
	implementation = _my_rule_v3_impl,
)
`,
	})()

	checkFindings(t, "deprecated-rule", `
my_rule_v1 = use_repo_rule(":rules.bzl", "my_rule_v1")
my_rule_v3 = use_repo_rule(":rules.bzl", "my_rule_v3")
`,
		[]string{
			`1: The rule "my_rule_v1" defined in "//test/package/rules.bzl" is deprecated.`,
		},
		scopeModule)
}

func TestDeprecatedRuleViaLoad(t *testing.T) {
	defer setUpFileReader(map[string]string{
		"test/package/rules.bzl": `
def _impl(ctx):
  pass

def _deprecated_impl(ctx):
  """
  Some rule.

  Deprecated:
    please use something else instead.
  """
  pass

my_rule_v1 = rule(
  implementation = _impl,
  doc = "Deprecated: please use something else instead.",
)

my_rule_v2 = rule(
  implementation = _deprecated_impl,
)

my_rule_v3 = rule(
  implementation = _impl,
)

my_repo_rule_v1 = repository_rule(
  implementation = _impl,
  doc = "Deprecated: please use something else instead.",
)

my_repo_rule_v2 = repository_rule(
  implementation = _deprecated_impl,
)

my_repo_rule_v3 = repository_rule(
  implementation = _impl,
)

my_materializer_rule_v1 = materializer_rule(
  implementation = _impl,
  doc = "Deprecated: please use v2 instead.",
)

my_materializer_rule_v2 = materializer_rule(
  implementation = _deprecated_impl,
)

my_materializer_rule_v3 = materializer_rule(
  implementation = _impl,
)
`,
	})()

	checkFindings(t, "deprecated-rule", `
load(":rules.bzl", "my_rule_v1", "my_rule_v2", "my_rule_v3", "my_repo_rule_v1", "my_repo_rule_v2", "my_repo_rule_v3", "my_materializer_rule_v1", "my_materializer_rule_v2", "my_materializer_rule_v3")
`,
		[]string{
			`1: The rule "my_rule_v1" defined in "//test/package/rules.bzl" is deprecated.`,
			`1: The rule "my_rule_v2" defined in "//test/package/rules.bzl" is deprecated.`,
			`1: The rule "my_repo_rule_v1" defined in "//test/package/rules.bzl" is deprecated.`,
      `1: The rule "my_repo_rule_v2" defined in "//test/package/rules.bzl" is deprecated.`,
      `1: The rule "my_materializer_rule_v1" defined in "//test/package/rules.bzl" is deprecated.`,
      `1: The rule "my_materializer_rule_v2" defined in "//test/package/rules.bzl" is deprecated.`,
		},
		scopeBuild|scopeBzl)
}
