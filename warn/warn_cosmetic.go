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

// Cosmetic warnings (e.g. for improved readability of Starlark files)

package warn

import (
	"regexp"
	"sort"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/edit"
)

// packageOnTopWarning hoists package statements to the top after any comments / docstrings / load statements.
// If applied together with loadOnTopWarning and/or outOfOrderLoadWarning, it should be applied after them.
// This is currently guaranteed by sorting the warning categories names before applying them:
// "load-on-top" < "out-of-order-load" < "package-on-top"
func packageOnTopWarning(f *build.File) []*LinterFinding {
	if f.Type == build.TypeWorkspace || f.Type == build.TypeModule {
		// Not applicable to WORKSPACE or MODULE files
		return nil
	}

	// Collect variables defined by assignment statements and find misplaced packages
	misplacedPackages := make(map[int]*build.CallExpr)
	firstStmtIndex := -1 // index of the first seen non-skippable statement
	definedVars := make(map[string]bool)

	for i := 0; i < len(f.Stmt); i++ {
		stmt := f.Stmt[i]

		// Track variables defined by assignment statements that could be used in the package() call
		if assign, ok := stmt.(*build.AssignExpr); ok {
			if ident, ok := assign.LHS.(*build.Ident); ok {
				definedVars[ident.Name] = true
			}
		}

		_, isString := stmt.(*build.StringExpr) // typically a docstring
		_, isComment := stmt.(*build.CommentBlock)
		_, isLoad := stmt.(*build.LoadStmt)
		_, isLicenses := edit.ExprToRule(stmt, "licenses")
		_, isPackageGroup := edit.ExprToRule(stmt, "package_group")
		if isString || isComment || isLoad || isLicenses || isPackageGroup || stmt == nil {
			continue
		}
		rule, ok := edit.ExprToRule(stmt, "package")
		if !ok {
			if firstStmtIndex == -1 {
				firstStmtIndex = i
			}
			continue
		}
		if firstStmtIndex == -1 {
			continue
		}
		// Check if the package() call uses any of the defined variables.
		// If so, we can't easily move it.
		usedSymbols := edit.UsedSymbols(rule.Call)
		usesDefinedVar := false
		for varName := range definedVars {
			if usedSymbols[varName] {
				usesDefinedVar = true
				break
			}
		}
		if usesDefinedVar {
			continue
		}
		misplacedPackages[i] = rule.Call
	}
	offset := len(misplacedPackages)
	if offset == 0 {
		return nil
	}

	// Calculate a fix by building the new statement order
	if firstStmtIndex == -1 {
		firstStmtIndex = 0
	}

	// Build lists of different statement types
	var beforePkg []build.Expr // statements before firstStmtIndex (comments, docstrings, loads)
	var packages []build.Expr  // misplaced package() calls
	var afterPkg []build.Expr  // other statements (assignments, rules, macros)

	for i, stmt := range f.Stmt {
		if i < firstStmtIndex {
			beforePkg = append(beforePkg, stmt)
			continue
		}
		if _, ok := misplacedPackages[i]; ok {
			packages = append(packages, stmt)
			continue
		}
		afterPkg = append(afterPkg, stmt)
	}

	newOrder := make([]build.Expr, 0, len(f.Stmt))
	newOrder = append(newOrder, beforePkg...)
	newOrder = append(newOrder, packages...)
	newOrder = append(newOrder, afterPkg...)

	var replacements []LinterReplacement
	for i := firstStmtIndex; i < len(f.Stmt); i++ {
		replacements = append(replacements, LinterReplacement{&f.Stmt[i], newOrder[i]})
	}

	var findings []*LinterFinding
	for _, load := range misplacedPackages {
		findings = append(findings, makeLinterFinding(load,
			"Package declaration should be at the top of the file, after the load() statements, "+
				"but before any call to a rule or a macro. "+
				"package_group() and licenses() may be called before package().", replacements...))
	}

	return findings
}

func unsortedDictItemsWarning(f *build.File) []*LinterFinding {
	var findings []*LinterFinding

	compareItems := func(item1, item2 *build.KeyValueExpr) bool {
		key1 := item1.Key.(*build.StringExpr).Value
		key2 := item2.Key.(*build.StringExpr).Value
		// regular keys should precede private ones (start with "_")
		if strings.HasPrefix(key1, "_") {
			return strings.HasPrefix(key2, "_") && key1 < key2
		}
		if strings.HasPrefix(key2, "_") {
			return true
		}

		// "//conditions:default" should always be the last
		const conditionsDefault = "//conditions:default"
		if key1 == conditionsDefault {
			return false
		} else if key2 == conditionsDefault {
			return true
		}

		return key1 < key2
	}

	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		dict, ok := (*expr).(*build.DictExpr)

		mustSkipCheck := func(expr build.Expr) bool {
			return edit.ContainsComments(expr, "@unsorted-dict-items")
		}

		if !ok || mustSkipCheck(dict) {
			return
		}
		// do not process dictionaries nested within expressions that do not
		// want dict items to be sorted
		for i := len(stack) - 1; i >= 0; i-- {
			if mustSkipCheck(stack[i]) {
				return
			}
		}
		var sortedItems []*build.KeyValueExpr
		for _, item := range dict.List {
			// include only string literal keys into consideration
			if _, ok = item.Key.(*build.StringExpr); !ok {
				continue
			}
			sortedItems = append(sortedItems, item)
		}

		// Fix
		comp := func(i, j int) bool {
			return compareItems(sortedItems[i], sortedItems[j])
		}

		var misplacedItems []*build.KeyValueExpr
		for i := 1; i < len(sortedItems); i++ {
			if comp(i, i-1) {
				misplacedItems = append(misplacedItems, sortedItems[i])
			}
		}

		if len(misplacedItems) == 0 {
			// Already sorted
			return
		}
		newDict := *dict
		newDict.List = append([]*build.KeyValueExpr{}, dict.List...)

		sort.SliceStable(sortedItems, comp)
		sortedItemIndex := 0
		for originalItemIndex := 0; originalItemIndex < len(dict.List); originalItemIndex++ {
			item := dict.List[originalItemIndex]
			if _, ok := item.Key.(*build.StringExpr); !ok {
				continue
			}
			newDict.List[originalItemIndex] = sortedItems[sortedItemIndex]
			sortedItemIndex++
		}

		for _, item := range misplacedItems {
			findings = append(findings, makeLinterFinding(item,
				"Dictionary items are out of their lexicographical order.",
				LinterReplacement{expr, &newDict}))
		}
	})
	return findings
}

// skylarkToStarlark converts a string "skylark" in different cases to "starlark"
// trying to preserve the same case style, e.g. capitalized "Skylark" becomes "Starlark".
func skylarkToStarlark(s string) string {
	switch {
	case s == "SKYLARK":
		return "STARLARK"
	case strings.HasPrefix(s, "S"):
		return "Starlark"
	default:
		return "starlark"
	}
}

var skylarkRegex = regexp.MustCompile("(?i)skylark")

// replaceSkylark replaces a substring "skylark" (case-insensitive) with a
// similar cased string "starlark". Doesn't replace it if the previous or the
// next symbol is '/', which may indicate it's a part of a URL.
// Normally that should be done with look-ahead and look-behind assertions in a
// regular expression, but negative look-aheads and look-behinds are not
// supported by Go regexp module.
func replaceSkylark(s string) (newString string, changed bool) {
	newString = s
	for _, r := range skylarkRegex.FindAllStringIndex(s, -1) {
		if r[0] > 0 && s[r[0]-1] == '/' {
			continue
		}
		if r[1] < len(s)-1 && s[r[1]+1] == '/' {
			continue
		}
		newString = newString[:r[0]] + skylarkToStarlark(newString[r[0]:r[1]]) + newString[r[1]:]
	}
	return newString, newString != s
}

func skylarkCommentWarning(f *build.File) []*LinterFinding {
	var findings []*LinterFinding
	msg := `"Skylark" is an outdated name of the language, please use "starlark" instead.`

	// Check comments
	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		comments := (*expr).Comment()
		newComments := build.Comments{
			Before: append([]build.Comment{}, comments.Before...),
			Suffix: append([]build.Comment{}, comments.Suffix...),
			After:  append([]build.Comment{}, comments.After...),
		}
		isModified := false
		var start, end build.Position

		for _, block := range []*[]build.Comment{&newComments.Before, &newComments.Suffix, &newComments.After} {
			for i, comment := range *block {
				// Don't trigger on disabling comments
				if strings.Contains(comment.Token, "disable=skylark-docstring") {
					continue
				}
				newValue, changed := replaceSkylark(comment.Token)
				(*block)[i] = build.Comment{
					Start: comment.Start,
					Token: newValue,
				}
				if changed {
					isModified = true
					start, end = comment.Span()
				}
			}
		}
		if !isModified {
			return
		}
		newExpr := (*expr).Copy()
		newExpr.Comment().Before = newComments.Before
		newExpr.Comment().Suffix = newComments.Suffix
		newExpr.Comment().After = newComments.After
		finding := makeLinterFinding(*expr, msg, LinterReplacement{expr, newExpr})
		finding.Start = start
		finding.End = end
		findings = append(findings, finding)
	})

	return findings
}

func checkSkylarkDocstring(stmts []build.Expr) *LinterFinding {
	msg := `"Skylark" is an outdated name of the language, please use "starlark" instead.`

	doc, ok := getDocstring(stmts)
	if !ok {
		return nil
	}
	docString := (*doc).(*build.StringExpr)
	newValue, updated := replaceSkylark(docString.Value)
	if !updated {
		return nil
	}
	newDocString := *docString
	newDocString.Value = newValue
	return makeLinterFinding(docString, msg, LinterReplacement{doc, &newDocString})
}

func skylarkDocstringWarning(f *build.File) []*LinterFinding {
	var findings []*LinterFinding

	// File docstring
	if finding := checkSkylarkDocstring(f.Stmt); finding != nil {
		findings = append(findings, finding)
	}

	// Function docstrings
	for _, stmt := range f.Stmt {
		def, ok := stmt.(*build.DefStmt)
		if !ok {
			continue
		}
		if finding := checkSkylarkDocstring(def.Body); finding != nil {
			findings = append(findings, finding)
		}
	}

	return findings
}
