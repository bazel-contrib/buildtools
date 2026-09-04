package config

import (
	"fmt"
	"path"
	"strings"

	"github.com/bazelbuild/buildtools/labels"
	"github.com/bazelbuild/buildtools/warn"
)

// AttrPolicy is the attrPolicy block in .buildifier.json.
type AttrPolicy struct {
	Rules []AttrPolicyRule `json:"rules,omitempty"`
}

// AttrPolicyRule is a single attribute policy rule.
type AttrPolicyRule struct {
	Name               string            `json:"name"`
	RuleKinds          []string          `json:"ruleKinds,omitempty"`
	Attr               string            `json:"attr"`
	ForbidValues       []string          `json:"forbidValues,omitempty"`
	RequireValues      []string          `json:"requireValues,omitempty"`
	ForbidListItems    []string          `json:"forbidListItems,omitempty"`
	RequireListItems   []string          `json:"requireListItems,omitempty"`
	AllowListItems     []string          `json:"allowListItems,omitempty"`
	ForbidDictEntries  map[string]string `json:"forbidDictEntries,omitempty"`
	RequireDictEntries map[string]string `json:"requireDictEntries,omitempty"`
	ForbidDictKeys     []string          `json:"forbidDictKeys,omitempty"`
	MinValue           *int              `json:"minValue,omitempty"`
	MaxValue           *int              `json:"maxValue,omitempty"`
	Required           bool              `json:"required,omitempty"`
	ForbidPresence     bool              `json:"forbidPresence,omitempty"`
	Allowlist          []string          `json:"allowlist,omitempty"`
	Suppressible       *bool             `json:"suppressible,omitempty"`
	Message            string            `json:"message,omitempty"`
}

func compileAttrPolicy(policy *AttrPolicy) ([]warn.AttrPolicyRuleCompiled, error) {
	if policy == nil || len(policy.Rules) == 0 {
		return warn.DefaultAttrPolicyRules(), nil
	}
	seen := make(map[string]bool)
	var compiled []warn.AttrPolicyRuleCompiled
	for i := range policy.Rules {
		rule := &policy.Rules[i]
		c, err := compileAttrPolicyRule(rule, seen)
		if err != nil {
			return nil, fmt.Errorf("attrPolicy.rules[%d]: %w", i, err)
		}
		compiled = append(compiled, c)
	}
	return compiled, nil
}

func compileAttrPolicyRule(rule *AttrPolicyRule, seen map[string]bool) (warn.AttrPolicyRuleCompiled, error) {
	name := strings.TrimSpace(rule.Name)
	if name == "" {
		return warn.AttrPolicyRuleCompiled{}, fmt.Errorf("attrPolicy rule %q: name is required", rule.Name)
	}
	if seen[name] {
		return warn.AttrPolicyRuleCompiled{}, fmt.Errorf("attrPolicy rule %q: duplicate name", name)
	}
	seen[name] = true

	attr := strings.TrimSpace(rule.Attr)
	if attr == "" {
		return warn.AttrPolicyRuleCompiled{}, fmt.Errorf("attrPolicy rule %q: attr is required", name)
	}

	family, families, err := attrPolicyConstraintFamily(rule)
	if err != nil {
		return warn.AttrPolicyRuleCompiled{}, fmt.Errorf("attrPolicy rule %q: %w", name, err)
	}
	if rule.ForbidPresence && rule.Required {
		return warn.AttrPolicyRuleCompiled{}, fmt.Errorf("attrPolicy rule %q: forbidPresence and required cannot both be true", name)
	}
	if families == 0 && !rule.Required {
		return warn.AttrPolicyRuleCompiled{}, fmt.Errorf("attrPolicy rule %q: at least one constraint, required=true, or forbidPresence=true is required", name)
	}

	for _, kindGlob := range rule.RuleKinds {
		if _, err := path.Match(kindGlob, "x"); err != nil {
			return warn.AttrPolicyRuleCompiled{}, fmt.Errorf("attrPolicy rule %q: malformed ruleKinds glob %q: %w", name, kindGlob, err)
		}
	}

	allowlist, err := compileAllowlistPatterns(name, rule.Allowlist)
	if err != nil {
		return warn.AttrPolicyRuleCompiled{}, err
	}

	suppressible := true
	if rule.Suppressible != nil {
		suppressible = *rule.Suppressible
	}

	return warn.AttrPolicyRuleCompiled{
		Name:               name,
		RuleKinds:          append([]string(nil), rule.RuleKinds...),
		Attr:               attr,
		Family:             family,
		ForbidValues:       append([]string(nil), rule.ForbidValues...),
		RequireValues:      append([]string(nil), rule.RequireValues...),
		ForbidListItems:    append([]string(nil), rule.ForbidListItems...),
		RequireListItems:   append([]string(nil), rule.RequireListItems...),
		AllowListItems:     append([]string(nil), rule.AllowListItems...),
		ForbidDictEntries:  copyStringMap(rule.ForbidDictEntries),
		RequireDictEntries: copyStringMap(rule.RequireDictEntries),
		ForbidDictKeys:     append([]string(nil), rule.ForbidDictKeys...),
		MinValue:           cloneIntPtr(rule.MinValue),
		MaxValue:           cloneIntPtr(rule.MaxValue),
		Required:           rule.Required,
		Allowlist:          allowlist,
		Suppressible:       suppressible,
		Message:            rule.Message,
	}, nil
}

func attrPolicyConstraintFamily(rule *AttrPolicyRule) (warn.AttrPolicyConstraintFamily, int, error) {
	scalar := len(rule.ForbidValues) > 0 || len(rule.RequireValues) > 0
	list := len(rule.ForbidListItems) > 0 || len(rule.RequireListItems) > 0 || len(rule.AllowListItems) > 0
	dict := len(rule.ForbidDictEntries) > 0 || len(rule.RequireDictEntries) > 0 || len(rule.ForbidDictKeys) > 0
	numeric := rule.MinValue != nil || rule.MaxValue != nil
	presence := rule.ForbidPresence

	families := 0
	var family warn.AttrPolicyConstraintFamily
	if scalar {
		families++
		family = warn.AttrPolicyScalarFamily
	}
	if list {
		families++
		family = warn.AttrPolicyListFamily
	}
	if dict {
		families++
		family = warn.AttrPolicyDictFamily
	}
	if numeric {
		families++
		family = warn.AttrPolicyNumericFamily
	}
	if presence {
		families++
		family = warn.AttrPolicyForbidPresenceFamily
	}
	if families > 1 {
		return 0, families, fmt.Errorf("cannot mix scalar, list, dict, numeric, and presence constraint families")
	}
	if numeric && rule.MinValue != nil && rule.MaxValue != nil && *rule.MinValue > *rule.MaxValue {
		return 0, families, fmt.Errorf("minValue must be <= maxValue")
	}
	return family, families, nil
}

func compileAllowlistPatterns(ruleName string, entries []string) ([]warn.AttrPolicyAllowlistPattern, error) {
	var patterns []warn.AttrPolicyAllowlistPattern
	for _, entry := range entries {
		p, err := parseAllowlistPattern(entry)
		if err != nil {
			return nil, fmt.Errorf("attrPolicy rule %q: allowlist entry %q: %w", ruleName, entry, err)
		}
		patterns = append(patterns, p)
	}
	return patterns, nil
}

func parseAllowlistPattern(entry string) (warn.AttrPolicyAllowlistPattern, error) {
	if entry == "" {
		return warn.AttrPolicyAllowlistPattern{}, fmt.Errorf("empty pattern")
	}
	if strings.HasPrefix(entry, "@") {
		return warn.AttrPolicyAllowlistPattern{}, fmt.Errorf("repository-qualified entries are not supported")
	}
	if entry == "..." {
		return warn.AttrPolicyAllowlistPattern{}, fmt.Errorf("bare ... is not supported")
	}
	if entry == "//..." {
		return warn.AttrPolicyAllowlistPattern{Kind: warn.AttrPolicyAllowAll}, nil
	}
	if !strings.HasPrefix(entry, "//") {
		return warn.AttrPolicyAllowlistPattern{}, fmt.Errorf("pattern must start with //")
	}
	if strings.HasSuffix(entry, "/...") {
		pkg := strings.TrimPrefix(entry, "//")
		pkg = strings.TrimSuffix(pkg, "/...")
		if pkg == "" {
			return warn.AttrPolicyAllowlistPattern{}, fmt.Errorf("invalid recursive pattern")
		}
		if strings.Contains(pkg, ":") {
			return warn.AttrPolicyAllowlistPattern{}, fmt.Errorf("invalid recursive pattern")
		}
		return warn.AttrPolicyAllowlistPattern{Kind: warn.AttrPolicyAllowRecursive, Pkg: pkg}, nil
	}

	label := labels.Parse(entry)
	if label.Target == "all" || label.Target == "*" {
		if label.Package == "" {
			return warn.AttrPolicyAllowlistPattern{}, fmt.Errorf("invalid package pattern")
		}
		return warn.AttrPolicyAllowlistPattern{Kind: warn.AttrPolicyAllowPackageAll, Pkg: label.Package}, nil
	}
	return warn.AttrPolicyAllowlistPattern{
		Kind:   warn.AttrPolicyAllowExact,
		Pkg:    label.Package,
		Target: label.Target,
	}, nil
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}
