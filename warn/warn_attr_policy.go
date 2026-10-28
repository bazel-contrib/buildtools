package warn

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/edit"
	"github.com/bazelbuild/buildtools/labels"
)

func attrPolicyWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBuild {
		return nil
	}
	config := effectiveAttrPolicyConfig()
	var findings []*LinterFinding
	for _, rule := range f.Rules("") {
		kind := rule.Kind()
		label := labels.Label{Package: f.Pkg, Target: rule.Name()}.Format()
		for _, p := range config {
			if !matchesRuleKind(p.RuleKinds, kind) || allowlistMatches(p.Allowlist, label, f.Pkg) {
				continue
			}
			findings = append(findings, attrPolicyCheckRule(rule, p)...)
		}
	}
	return findings
}

func attrPolicyCheckRule(rule *build.Rule, p AttrPolicyRuleCompiled) []*LinterFinding {
	var findings []*LinterFinding
	attrExpr := rule.Attr(p.Attr)

	if p.Required && attrExpr == nil {
		findings = append(findings, makeLinterFinding(rule.Call, attrPolicyMessage(p,
			fmt.Sprintf("attribute %q is required", p.Attr))))
		return findings
	}

	switch p.Family {
	case AttrPolicyScalarFamily:
		if attrExpr == nil {
			return findings
		}
		value := attrScalarString(rule, p.Attr)
		for _, forbidden := range p.ForbidValues {
			if value == forbidden {
				findings = append(findings, makeLinterFinding(attrExpr, attrPolicyMessage(p,
					fmt.Sprintf("attribute %q must not be %q", p.Attr, forbidden))))
				break
			}
		}
		if len(p.RequireValues) > 0 && !slices.Contains(p.RequireValues, value) {
			findings = append(findings, makeLinterFinding(attrExpr, attrPolicyMessage(p,
				fmt.Sprintf("attribute %q must be one of %s", p.Attr, quoteList(p.RequireValues)))))
		}
	case AttrPolicyListFamily:
		if attrExpr == nil {
			return findings
		}
		items := rule.AttrStrings(p.Attr)
		if items == nil {
			return findings
		}
		for _, forbidden := range p.ForbidListItems {
			if slices.Contains(items, forbidden) {
				node := attrExpr
				if itemExpr := listItemExpr(attrExpr, forbidden); itemExpr != nil {
					node = itemExpr
				}
				findings = append(findings, makeLinterFinding(node, attrPolicyMessage(p,
					fmt.Sprintf("attribute %q must not contain %q", p.Attr, forbidden))))
			}
		}
		for _, required := range p.RequireListItems {
			if !slices.Contains(items, required) {
				findings = append(findings, makeLinterFinding(attrExpr, attrPolicyMessage(p,
					fmt.Sprintf("attribute %q must contain %q", p.Attr, required))))
			}
		}
		for _, item := range items {
			if len(p.AllowListItems) > 0 && !allowListItemMatches(p.AllowListItems, item) {
				node := attrExpr
				if itemExpr := listItemExpr(attrExpr, item); itemExpr != nil {
					node = itemExpr
				}
				findings = append(findings, makeLinterFinding(node, attrPolicyMessage(p,
					fmt.Sprintf("attribute %q must not contain %q; allowed values are %s",
						p.Attr, item, quoteList(p.AllowListItems)))))
			}
		}
	case AttrPolicyDictFamily:
		if attrExpr == nil {
			return findings
		}
		dict, ok := attrExpr.(*build.DictExpr)
		if !ok {
			return findings
		}
		for key, forbiddenValue := range p.ForbidDictEntries {
			if valueExpr := edit.DictionaryGet(dict, key); valueExpr != nil {
				if exprScalarString(valueExpr) == forbiddenValue {
					findings = append(findings, makeLinterFinding(valueExpr, attrPolicyMessage(p,
						fmt.Sprintf("attribute %q must not contain %q: %q", p.Attr, key, forbiddenValue))))
				}
			}
		}
		for key, requiredValue := range p.RequireDictEntries {
			valueExpr := edit.DictionaryGet(dict, key)
			if valueExpr == nil || exprScalarString(valueExpr) != requiredValue {
				findings = append(findings, makeLinterFinding(attrExpr, attrPolicyMessage(p,
					fmt.Sprintf("attribute %q must contain %q: %q", p.Attr, key, requiredValue))))
			}
		}
		for _, key := range p.ForbidDictKeys {
			if valueExpr := edit.DictionaryGet(dict, key); valueExpr != nil {
				findings = append(findings, makeLinterFinding(valueExpr, attrPolicyMessage(p,
					fmt.Sprintf("attribute %q must not contain key %q", p.Attr, key))))
			}
		}
	case AttrPolicyNumericFamily:
		if attrExpr == nil {
			return findings
		}
		value, err := strconv.Atoi(rule.AttrLiteral(p.Attr))
		if err != nil {
			return findings
		}
		if p.MinValue != nil && value < *p.MinValue {
			findings = append(findings, makeLinterFinding(attrExpr, attrPolicyMessage(p,
				fmt.Sprintf("attribute %q must be >= %d", p.Attr, *p.MinValue))))
		}
		if p.MaxValue != nil && value > *p.MaxValue {
			findings = append(findings, makeLinterFinding(attrExpr, attrPolicyMessage(p,
				fmt.Sprintf("attribute %q must be <= %d", p.Attr, *p.MaxValue))))
		}
	case AttrPolicyForbidPresenceFamily:
		if attrExpr != nil {
			findings = append(findings, makeLinterFinding(attrExpr, attrPolicyMessage(p,
				fmt.Sprintf("attribute %q must not be set", p.Attr))))
		}
	}
	return findings
}

func attrPolicyMessage(p AttrPolicyRuleCompiled, defaultMsg string) string {
	if p.Message != "" {
		return fmt.Sprintf("[%s] %s", p.Name, p.Message)
	}
	return fmt.Sprintf("[%s] %s", p.Name, defaultMsg)
}

func attrScalarString(rule *build.Rule, key string) string {
	if s := rule.AttrString(key); s != "" {
		return s
	}
	return rule.AttrLiteral(key)
}

func exprScalarString(expr build.Expr) string {
	if expr == nil {
		return ""
	}
	if s, ok := expr.(*build.StringExpr); ok {
		return s.Value
	}
	if i, ok := expr.(*build.Ident); ok {
		return i.Name
	}
	if l, ok := expr.(*build.LiteralExpr); ok {
		return l.Token
	}
	return ""
}

func quoteList(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(quoted, ", ")
}

func listItemExpr(attrExpr build.Expr, item string) build.Expr {
	list, ok := attrExpr.(*build.ListExpr)
	if !ok {
		return nil
	}
	for _, elem := range list.List {
		if str, ok := elem.(*build.StringExpr); ok && str.Value == item {
			return elem
		}
	}
	return nil
}

// allowListItemMatches reports whether item satisfies any allowListItems entry.
// Entries ending in * are prefix patterns; all other entries are exact matches.
func allowListItemMatches(allowed []string, item string) bool {
	for _, pattern := range allowed {
		if strings.HasSuffix(pattern, "*") {
			if strings.HasPrefix(item, strings.TrimSuffix(pattern, "*")) {
				return true
			}
			continue
		}
		if item == pattern {
			return true
		}
	}
	return false
}
