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
	"io/fs"
	"path"
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
	if pkg == "" {
		return "", "", "", false
	}
	return prefix, pkg, suffix, true
}

// foldCase maps a name to a key that is shared by all names that only differ
// in case.
func foldCase(name string) string {
	return strings.ToLower(strings.ToUpper(name))
}

// subdirIndex records the names of the subdirectories of a directory in the
// repository.
type subdirIndex struct {
	exact map[string]bool
	// folded maps case-folded names to names. It is only built when needed.
	folded map[string][]string
}

// caseInsensitiveMatches returns the names of all subdirectories whose names
// only differ in case from name.
func (index *subdirIndex) caseInsensitiveMatches(name string) []string {
	if index.folded == nil {
		index.folded = make(map[string][]string)
		for subdir := range index.exact {
			key := foldCase(subdir)
			index.folded[key] = append(index.folded[key], subdir)
		}
	}
	return index.folded[foldCase(name)]
}

// packageSpellingResolver determines how package paths are spelled in the
// repository.
type packageSpellingResolver struct {
	fileReader *FileReader
	dirs       map[string]*subdirIndex
}

func (r *packageSpellingResolver) subdirs(dir string) *subdirIndex {
	if index, ok := r.dirs[dir]; ok {
		return index
	}
	index := &subdirIndex{exact: make(map[string]bool)}
	for _, entry := range r.fileReader.getDir(dir) {
		// Symlinks may point to directories and are followed by Bazel.
		if entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
			index.exact[entry.Name()] = true
		}
	}
	r.dirs[dir] = index
	return index
}

// spelling returns the package path pkg as spelled by the directories in the
// repository. Returns false if a directory along the path doesn't exist or if
// its spelling is ambiguous because there are multiple directories that only
// differ in case and none of them matches exactly.
func (r *packageSpellingResolver) spelling(pkg string) (string, bool) {
	components := strings.Split(pkg, "/")
	dir := ""
	for i, component := range components {
		if component == "" || component == "." || component == ".." {
			return "", false
		}
		index := r.subdirs(dir)
		if !index.exact[component] {
			candidates := index.caseInsensitiveMatches(component)
			if len(candidates) != 1 {
				return "", false
			}
			components[i] = candidates[0]
		}
		dir = path.Join(dir, components[i])
	}
	return dir, true
}

func packageCaseWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if !fileReader.canReadDirs() {
		return nil
	}
	resolver := &packageSpellingResolver{
		fileReader: fileReader,
		dirs:       make(map[string]*subdirIndex),
	}

	// check returns the finding for a string literal with a misspelled package
	// path as well as the string literal with the fixed spelling.
	check := func(str *build.StringExpr) (*LinterFinding, *build.StringExpr) {
		prefix, pkg, suffix, ok := splitMainRepoLabel(str.Value)
		if !ok {
			return nil, nil
		}
		spelling, ok := resolver.spelling(pkg)
		if !ok || spelling == pkg {
			return nil, nil
		}
		fixed := *str
		fixed.Value = prefix + spelling + suffix
		fixed.Token = ""
		return makeLinterFinding(str, fmt.Sprintf(
			"The package %q in %q doesn't match the case of its directory %q. Use %q instead.",
			pkg, str.Value, spelling, fixed.Value)), &fixed
	}

	var findings []*LinterFinding
	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		switch node := (*expr).(type) {
		case *build.LoadStmt:
			finding, fixed := check(node.Module)
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
			finding, fixed := check(node)
			if finding == nil {
				return
			}
			finding.Replacement = []LinterReplacement{{expr, fixed}}
			findings = append(findings, finding)
		}
	})
	return findings
}
