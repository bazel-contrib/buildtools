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
	"path"
	"regexp"
	"strings"

	"github.com/bazel-contrib/buildtools/v10/build"
	"github.com/bazel-contrib/buildtools/v10/edit"
)

// locationMakeVariableRe matches deprecated $(location) and $(locations) make variables.
// Note that this is applied to individual string tokens in the parsed file, so we cannot
// require a match for a closing parenthesis if we are to detect constructions like this:
//     args = ["$(location " + src + ")", "$(location " + gen + ")"],
var locationMakeVariableRe = regexp.MustCompile(`\$\(locations?(?:\s[^)]*)?`)

func constantGlobPatternWarning(expr build.Expr) []*LinterFinding {
	findings := []*LinterFinding{}
	for _, patterns := range edit.AllLists(expr) {
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
		for _, arg := range call.List {
			assignExpr, ok := arg.(*build.AssignExpr)
			if !ok {
				// first arg is unnamed
				findings = append(findings, constantGlobPatternWarning(arg)...)
				return // at most one warning per glob
			}
			str, ok := assignExpr.LHS.(*build.Ident)
			if !ok || str.Name != "include" {
				continue
			}
			findings = append(findings, constantGlobPatternWarning(assignExpr.RHS)...)
			return // at most one warning per glob
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

	findings := []*LinterFinding{}
	names := make(map[string]int) // map from name to line number
	msg := `A rule with name %q was already found on line %d. ` +
		`Even if it's valid for Blaze, this may confuse other tools. ` +
		`Please rename it and use different names.`

	for _, rule := range f.Rules("") {
		name := rule.ExplicitName()
		if name == "" {
			continue
		}
		start, _ := rule.Call.Span()
		if line, ok := names[name]; ok {
			finding := makeLinterFinding(rule.Call, fmt.Sprintf(msg, name, line))
			if nameNode := rule.Attr("name"); nameNode != nil {
				finding.Start, finding.End = nameNode.Span()
				start = finding.Start
			}
			findings = append(findings, finding)
		} else {
			names[name] = start.Line
		}
	}
	return findings
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

// splitMainRepoLabel splits a string that looks like a label or a target
// pattern referring to the main repository, e.g. "//pkg:target", "@//pkg",
// "@@//pkg" or "-//pkg/...", into the prefix up to and including "//", the
// package path and the remainder of the string. A trailing "/..." wildcard is
// not considered to be part of the package path.
func splitMainRepoLabel(value string) (prefix, pkg, suffix string, ok bool) {
	rest := strings.TrimPrefix(value, "-")
	if strings.HasPrefix(rest, "@@") {
		rest = rest[2:]
	} else if strings.HasPrefix(rest, "@") {
		rest = rest[1:]
	}
	if !strings.HasPrefix(rest, "//") {
		return "", "", "", false
	}
	rest = rest[2:]
	prefix = value[:len(value)-len(rest)]
	pkg = rest
	if i := strings.IndexByte(rest, ':'); i >= 0 {
		pkg, suffix = rest[:i], rest[i:]
	}
	if pkg == "..." {
		return "", "", "", false
	}
	if strings.HasSuffix(pkg, "/...") {
		pkg = strings.TrimSuffix(pkg, "/...")
		suffix = "/..." + suffix
	}
	return prefix, pkg, suffix, true
}

// isValidPathComponent returns whether a component of a path in a label can
// be looked up in a directory.
func isValidPathComponent(component string) bool {
	return component != "" && component != "." && component != ".."
}

// spellDirs returns the path rel of a directory relative to the directory base
// as spelled by the entries stored in the file system. Returns false if rel
// doesn't refer to a directory.
func spellDirs(fileReader *FileReader, base, rel string) (string, bool) {
	if rel == "" {
		return "", true
	}
	components := strings.Split(rel, "/")
	dir := base
	for i, component := range components {
		if !isValidPathComponent(component) {
			return "", false
		}
		entry := fileReader.lookUpEntry(dir, component)
		if entry == nil || !entry.IsDir() {
			return "", false
		}
		components[i] = entry.Name()
		dir = path.Join(dir, components[i])
	}
	return strings.Join(components, "/"), true
}

// spellTargetDirs returns the name of a target in the package pkg with the
// directories it contains spelled as the entries stored in the file system.
// The last component of the target name is never changed as the target may be
// a rule.
func spellTargetDirs(fileReader *FileReader, pkg, target string) string {
	i := strings.LastIndexByte(target, '/')
	if i < 0 {
		return target
	}
	dirs, name := target[:i], target[i+1:]
	spelledDirs, ok := spellDirs(fileReader, pkg, dirs)
	// Rule names may contain slashes and output files don't exist in the source
	// tree, so only change the spelling if the target exists on disk.
	if !ok || !isValidPathComponent(name) || fileReader.lookUpEntry(path.Join(pkg, spelledDirs), name) == nil {
		return target
	}
	return spelledDirs + "/" + name
}

// isPackageSpecification returns whether the innermost node in stack is a
// package specification such as "//pkg" rather than a label, i.e. part of the
// packages attribute of a package_group or of the arguments of visibility() in
// a .bzl file, possibly wrapped in other calls such as sorted(), but not in an
// explicit Label().
func isPackageSpecification(stack []build.Expr) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		call, ok := stack[i].(*build.CallExpr)
		if !ok {
			continue
		}
		switch (&build.Rule{Call: call}).Kind() {
		case "Label":
			return false
		case "visibility":
			return true
		case "package_group", "native.package_group":
			if i+1 < len(stack) {
				if assign, ok := stack[i+1].(*build.AssignExpr); ok {
					if lhs, ok := assign.LHS.(*build.Ident); ok && lhs.Name == "packages" {
						return true
					}
				}
			}
			return false
		}
	}
	return false
}

// specialLabels are labels in the main repository that have a special meaning
// in Bazel and don't refer to a package in the repository, without the leading
// "//".
var specialLabels = map[string]bool{
	"conditions:default": true,
	"visibility:private": true,
	"visibility:public":  true,
}

func labelCaseWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if !fileReader.canLookUp() {
		return nil
	}

	// check returns the finding for a string literal whose spelling doesn't
	// match the file system as well as the string literal with the fixed
	// spelling.
	check := func(str *build.StringExpr, stack []build.Expr) (*LinterFinding, *build.StringExpr) {
		// A string that is concatenated with others is only part of a label,
		// e.g. in "//pkg" + ":target" or "@repo" + "//pkg:target". The full
		// label is unknown, so leave it alone.
		for i := len(stack) - 1; i >= 0; i-- {
			if _, ok := stack[i].(*build.ParenExpr); ok {
				continue
			}
			if concat, ok := stack[i].(*build.BinaryExpr); ok && concat.Op == "+" {
				return nil, nil
			}
			break
		}
		prefix, pkg, suffix, ok := splitMainRepoLabel(str.Value)
		if !ok || specialLabels[pkg+suffix] {
			return nil, nil
		}
		spelledPkg, ok := spellDirs(fileReader, "", pkg)
		if !ok {
			return nil, nil
		}
		if target, ok := strings.CutPrefix(suffix, ":"); ok {
			suffix = ":" + spellTargetDirs(fileReader, spelledPkg, target)
		} else if suffix == "" && path.Base(spelledPkg) != path.Base(pkg) && !isPackageSpecification(stack) {
			// "//pkg" is short for "//pkg:pkg", keep the original target name.
			suffix = ":" + path.Base(pkg)
		}
		value := prefix + spelledPkg + suffix
		// Don't turn a label into one with a special meaning.
		if value == str.Value || specialLabels[spelledPkg+suffix] {
			return nil, nil
		}
		fixed := *str
		fixed.Value = value
		fixed.Token = ""
		return makeLinterFinding(str, fmt.Sprintf(
			"%q doesn't match the spelling of the directories on disk. Use %q instead.",
			str.Value, value)), &fixed
	}

	var findings []*LinterFinding
	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		switch node := (*expr).(type) {
		case *build.LoadStmt:
			finding, fixed := check(node.Module, nil)
			if finding == nil {
				return
			}
			// The module of a load statement can't be replaced via a pointer to
			// an expression, replace the whole statement instead.
			load := *node
			load.Module = fixed
			finding.Replacement = []LinterReplacement{{expr, &load}}
			findings = append(findings, finding)
		case *build.StringExpr:
			if len(stack) > 0 {
				if _, ok := stack[len(stack)-1].(*build.LoadStmt); ok {
					// Already handled above.
					return
				}
			}
			finding, fixed := check(node, stack)
			if finding == nil {
				return
			}
			finding.Replacement = []LinterReplacement{{expr, fixed}}
			findings = append(findings, finding)
		}
	})
	return findings
}
