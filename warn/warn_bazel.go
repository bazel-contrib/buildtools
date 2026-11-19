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

// General Bazel-related warnings

package warn

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bazel-contrib/buildtools/v10/build"
)

// locationMakeVariableRe matches deprecated $(location) and $(locations) make variables.
// Note that this is applied to individual string tokens in the parsed file, so we cannot
// require a match for a closing parenthesis if we are to detect constructions like this:
//     args = ["$(location " + src + ")", "$(location " + gen + ")"],
var locationMakeVariableRe = regexp.MustCompile(`\$\(locations?(?:\s[^)]*)?`)

func constantGlobPatternWarning(patterns *build.ListExpr) []*LinterFinding {
	findings := []*LinterFinding{}
	for _, expr := range patterns.List {
		str, ok := expr.(*build.StringExpr)
		if !ok {
			continue
		}
		if !strings.Contains(str.Value, "*") {
			message := fmt.Sprintf(
				`Glob pattern %q has no wildcard ('*'). Constant patterns can be error-prone, move the file outside the glob.`, str.Value)
			findings = append(findings, makeLinterFinding(expr, message))
			return findings // at most one warning per glob
		}
	}
	return findings
}

func constantGlobWarning(f *build.File) []*LinterFinding {
	switch f.Type {
	case build.TypeBuild, build.TypeWorkspace, build.TypeBzl:
	default:
		// Not applicable
		return nil
	}

	findings := []*LinterFinding{}
	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		call, ok := expr.(*build.CallExpr)
		if !ok || len(call.List) == 0 {
			return
		}
		ident, ok := (call.X).(*build.Ident)
		if !ok || ident.Name != "glob" {
			return
		}
		patterns, ok := call.List[0].(*build.ListExpr)
		if ok {
			// first arg is unnamed and is a list
			findings = append(findings, constantGlobPatternWarning(patterns)...)
			return // at most one warning per glob
		}

		// look for named args called include
		for _, arg := range call.List {
			assignExpr, ok := arg.(*build.AssignExpr)
			if !ok {
				continue
			}
			str, ok := assignExpr.LHS.(*build.Ident)
			if !ok || str.Name != "include" {
				continue
			}
			patterns, ok := assignExpr.RHS.(*build.ListExpr)
			if ok {
				findings = append(findings, constantGlobPatternWarning(patterns)...)
				return // at most one warning per glob
			}
		}
	})
	return findings
}

func nativeInBuildFilesWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBuild {
		return nil
	}

	findings := []*LinterFinding{}
	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		// Search for `native.xxx` nodes
		dot, ok := (*expr).(*build.DotExpr)
		if !ok {
			return
		}
		ident, ok := dot.X.(*build.Ident)
		if !ok || ident.Name != "native" {
			return
		}

		findings = append(findings,
			makeLinterFinding(ident,
				`The "native" module shouldn't be used in BUILD files, its members are available as global symbols.`,
				LinterReplacement{expr, &build.Ident{Name: dot.Name}}))
	})
	return findings
}

func nativePackageWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	findings := []*LinterFinding{}
	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		// Search for `native.package()` nodes
		call, ok := expr.(*build.CallExpr)
		if !ok {
			return
		}
		dot, ok := call.X.(*build.DotExpr)
		if !ok || dot.Name != "package" {
			return
		}
		ident, ok := dot.X.(*build.Ident)
		if !ok || ident.Name != "native" {
			return
		}

		findings = append(findings,
			makeLinterFinding(call, `"native.package()" shouldn't be used in .bzl files.`))
	})
	return findings
}

func duplicatedNameWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBuild && f.Type != build.TypeWorkspace {
		// Not applicable to .bzl files.
		return nil
	}

	type namedRule struct {
		branches []conditionalBranch
		line     int
	}

	findings := []*LinterFinding{}
	names := make(map[string][]namedRule) // map from name to previous rules
	msg := `A rule with name %q was already found on line %d. ` +
		`Even if it's valid for Blaze, this may confuse other tools. ` +
		`Please rename it and use different names.`

	// Same traversal as File.Rules, but with access to the stack of each rule.
	for _, stmt := range f.Stmt {
		build.Walk(stmt, func(x build.Expr, stk []build.Expr) {
			call, ok := x.(*build.CallExpr)
			if !ok {
				return
			}
			for _, frame := range stk {
				if _, ok := frame.(*build.CallExpr); ok {
					// Skip nested calls.
					return
				}
			}

			rule := f.Rule(call)
			name := rule.ExplicitName()
			if name == "" {
				return
			}
			start, _ := call.Span()
			current := namedRule{conditionalBranches(x, stk), start.Line}
			for _, previous := range names[name] {
				// Rules in different branches of the same conditional expression,
				// e.g. `foo(name = "x") if cond else bar(name = "x")`, are never
				// both defined.
				if inDifferentBranches(previous.branches, current.branches) {
					continue
				}
				finding := makeLinterFinding(call, fmt.Sprintf(msg, name, previous.line))
				if nameNode := rule.Attr("name"); nameNode != nil {
					finding.Start, finding.End = nameNode.Span()
				}
				findings = append(findings, finding)
				break
			}
			names[name] = append(names[name], current)
		})
	}
	return findings
}

// conditionalBranch identifies one branch of a conditional expression.
type conditionalBranch struct {
	cond *build.ConditionalExpr
	then bool
}

// conditionalBranches returns the branches of the conditional expressions in stk
// that contain x, from outermost to innermost, or nil if there are none.
func conditionalBranches(x build.Expr, stk []build.Expr) []conditionalBranch {
	var branches []conditionalBranch
	for i, frame := range stk {
		cond, ok := frame.(*build.ConditionalExpr)
		if !ok {
			continue
		}
		inner := x
		if i+1 < len(stk) {
			inner = stk[i+1]
		}
		// Expressions in the condition itself are always evaluated.
		if inner == cond.Then || inner == cond.Else {
			branches = append(branches, conditionalBranch{cond, inner == cond.Then})
		}
	}
	return branches
}

// inDifferentBranches returns whether two expressions are in different branches
// of the same conditional expression, given their conditionalBranches.
func inDifferentBranches(a, b []conditionalBranch) bool {
	// Both lists agree up to the first conditional expression that doesn't
	// contain both expressions in the same branch, which decides the result.
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i].cond != b[i].cond {
			return false
		}
		if a[i].then != b[i].then {
			return true
		}
	}
	return false
}

func positionalArgumentsWarning(f *build.File, fileReader *FileReader) (findings []*LinterFinding) {
	if f.Type != build.TypeBuild {
		return nil
	}
	macroAnalyzer := newMacroAnalyzer(fileReader)
	macroAnalyzer.files[f.Pkg+":"+f.Label] = analyzeFile(f)

	for _, expr := range f.Stmt {
		build.Walk(expr, func(x build.Expr, _ []build.Expr) {
			if fnCall, ok := x.(*build.CallExpr); ok {
				fnIdent, ok := fnCall.X.(*build.Ident)
				if !ok {
					return
				}

				if macroAnalyzer.IsRuleOrMacro(function{pkg: f.Pkg, filename: f.Label, name: fnIdent.Name}).isRuleOrMacro {
					for _, arg := range fnCall.List {
						if _, ok := arg.(*build.AssignExpr); ok || arg == nil {
							continue
						}
						findings = append(findings, makeLinterFinding(fnCall, fmt.Sprintf(
							`All calls to rules or macros should pass arguments by keyword (arg_name=value) syntax.
Found call to rule or macro %q with positional arguments.`,
							fnIdent.Name)))
						return
					}
				}
			}
		})
	}
	return
}

func argsKwargsInBuildFilesWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBuild {
		return nil
	}

	findings := []*LinterFinding{}
	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		// Search for function call nodes
		call, ok := expr.(*build.CallExpr)
		if !ok {
			return
		}
		for _, param := range call.List {
			unary, ok := param.(*build.UnaryExpr)
			if !ok {
				continue
			}
			switch unary.Op {
			case "*":
				findings = append(findings,
					makeLinterFinding(param, `*args are not allowed in BUILD files.`))
			case "**":
				findings = append(findings,
					makeLinterFinding(param, `**kwargs are not allowed in BUILD files.`))
			}
		}
	})
	return findings
}

func printWarning(f *build.File) []*LinterFinding {
	if f.Type == build.TypeDefault {
		// Only applicable to Bazel files
		return nil
	}

	findings := []*LinterFinding{}
	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		call, ok := expr.(*build.CallExpr)
		if !ok {
			return
		}
		ident, ok := (call.X).(*build.Ident)
		if !ok || ident.Name != "print" {
			return
		}
		findings = append(findings,
			makeLinterFinding(expr, `"print()" is a debug function and shouldn't be submitted.`))
	})
	return findings
}

func makeLocationVariableWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBuild && f.Type != build.TypeBzl {
		return nil
	}

	findings := []*LinterFinding{}
	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		stringExpr, ok := expr.(*build.StringExpr)
		if !ok {
			return
		}
		if locationMakeVariableRe.MatchString(stringExpr.Value) {
			findings = append(findings,
				makeLinterFinding(stringExpr,
					`The "$(location)" and "$(locations)" make variables are deprecated. Use "$(execpath ...)" or "$(rootpath ...)" instead.`))
		}
	})
	return findings
}
