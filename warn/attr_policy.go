package warn

import (
	"path"
	"strings"

	"github.com/bazelbuild/buildtools/labels"
)

// AttrPolicyConstraintFamily identifies which constraint fields apply to a policy rule.
type AttrPolicyConstraintFamily int

const (
	AttrPolicyScalarFamily AttrPolicyConstraintFamily = iota
	AttrPolicyListFamily
	AttrPolicyDictFamily
	AttrPolicyNumericFamily
	AttrPolicyForbidPresenceFamily
)

// AttrPolicyAllowlistKind is a compiled allow-list pattern kind.
type AttrPolicyAllowlistKind int

const (
	AttrPolicyAllowAll AttrPolicyAllowlistKind = iota
	AttrPolicyAllowExact
	AttrPolicyAllowPackageAll
	AttrPolicyAllowRecursive
)

// AttrPolicyAllowlistPattern is a compiled allow-list entry.
type AttrPolicyAllowlistPattern struct {
	Kind   AttrPolicyAllowlistKind
	Pkg    string
	Target string
}

// AttrPolicyRuleCompiled is a compiled attribute policy rule.
type AttrPolicyRuleCompiled struct {
	Name     string
	RuleKinds []string
	Attr     string
	Family   AttrPolicyConstraintFamily

	ForbidValues       []string
	RequireValues      []string
	ForbidListItems    []string
	RequireListItems   []string
	AllowListItems     []string
	ForbidDictEntries  map[string]string
	RequireDictEntries map[string]string
	ForbidDictKeys     []string
	MinValue           *int
	MaxValue           *int

	Required     bool
	Allowlist    []AttrPolicyAllowlistPattern
	Suppressible bool
	Message      string
}

const defaultShardCountMax = 50

var defaultOutputLicensesRuleKinds = []string{
	"genrule",
	"cc_binary",
	"cc_toolchain",
	"java_binary",
	"java_plugin",
}

// DefaultAttrPolicyRules returns the built-in attribute policy rules applied when
// no attrPolicy configuration is present.
func DefaultAttrPolicyRules() []AttrPolicyRuleCompiled {
	maxValue := defaultShardCountMax
	return []AttrPolicyRuleCompiled{
		{
			Name:         "max-shard-count",
			RuleKinds:    []string{"*_test"},
			Attr:         "shard_count",
			Family:       AttrPolicyNumericFamily,
			MaxValue:     &maxValue,
			Message:      "Having more than 50 shards is indicative of poor test organization. Please reduce the number of shards.",
			Suppressible: true,
		},
		{
			Name:         "no-licenses",
			Attr:         "licenses",
			Family:       AttrPolicyForbidPresenceFamily,
			Message:      "The licenses attribute is deprecated; use package(default_applicable_licenses = ...) and applicable_licenses on targets instead (https://github.com/bazelbuild/bazel/issues/188).",
			Suppressible: true,
		},
		{
			Name:         "no-output-licenses",
			RuleKinds:    append([]string(nil), defaultOutputLicensesRuleKinds...),
			Attr:         "output_licenses",
			Family:       AttrPolicyForbidPresenceFamily,
			Message:      "The output_licenses attribute is deprecated; use applicable_licenses instead (https://github.com/bazelbuild/bazel/issues/7444).",
			Suppressible: true,
		},
	}
}

// AttrPolicyConfig is process-global policy, set from buildifier config before linting.
var AttrPolicyConfig []AttrPolicyRuleCompiled

// SetAttrPolicy replaces the active attribute policy rules.
func SetAttrPolicy(rules []AttrPolicyRuleCompiled) {
	AttrPolicyConfig = rules
}

func effectiveAttrPolicyConfig() []AttrPolicyRuleCompiled {
	if len(AttrPolicyConfig) > 0 {
		return AttrPolicyConfig
	}
	return DefaultAttrPolicyRules()
}

func matchesRuleKind(globs []string, kind string) bool {
	if len(globs) == 0 {
		return true
	}
	for _, g := range globs {
		if matched, err := path.Match(g, kind); err == nil && matched {
			return true
		}
	}
	return false
}

func allowlistMatches(patterns []AttrPolicyAllowlistPattern, label, pkg string) bool {
	for _, p := range patterns {
		if allowlistPatternMatches(p, label, pkg) {
			return true
		}
	}
	return false
}

func allowlistPatternMatches(p AttrPolicyAllowlistPattern, label, pkg string) bool {
	switch p.Kind {
	case AttrPolicyAllowAll:
		return true
	case AttrPolicyAllowRecursive:
		l := labels.Parse(label)
		return l.Package == p.Pkg || strings.HasPrefix(l.Package, p.Pkg+"/")
	case AttrPolicyAllowPackageAll:
		return labels.Parse(label).Package == p.Pkg
	case AttrPolicyAllowExact:
		want := labels.Label{Package: p.Pkg, Target: p.Target}.Format()
		return labels.Equal(label, want, pkg)
	default:
		return false
	}
}
