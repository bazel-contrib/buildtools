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

// Warnings for incompatible changes in the Bazel API

package warn

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/bzlenv"
	"github.com/bazelbuild/buildtools/edit"
	"github.com/bazelbuild/buildtools/edit/bzlmod"
	"github.com/bazelbuild/buildtools/labels"
	"github.com/bazelbuild/buildtools/tables"
)

// Bazel API-specific warnings

// negateExpression returns an expression which is a negation of the input.
// If it's a boolean literal (true or false), just return the opposite literal.
// If it's a unary expression with a unary `not` operator, just remove it.
// Otherwise, insert a `not` operator.
// It's assumed that input is no longer needed as it may be mutated or reused by the function.
func negateExpression(expr build.Expr) build.Expr {
	paren, ok := expr.(*build.ParenExpr)
	if ok {
		newParen := *paren
		newParen.X = negateExpression(paren.X)
		return &newParen
	}

	unary, ok := expr.(*build.UnaryExpr)
	if ok && unary.Op == "not" {
		return unary.X
	}

	boolean, ok := expr.(*build.Ident)
	if ok {
		newBoolean := *boolean
		if boolean.Name == "True" {
			newBoolean.Name = "False"
		} else {
			newBoolean.Name = "True"
		}
		return &newBoolean
	}

	return &build.UnaryExpr{
		Op: "not",
		X:  expr,
	}
}

// getParam search for a param with a given name in a given list of function arguments
// and returns it with its index
func getParam(attrs []build.Expr, paramName string) (int, *build.Ident, *build.AssignExpr) {
	for i, attr := range attrs {
		as, ok := attr.(*build.AssignExpr)
		if !ok {
			continue
		}
		name, ok := (as.LHS).(*build.Ident)
		if !ok || name.Name != paramName {
			continue
		}
		return i, name, as
	}
	return -1, nil, nil
}

// isFunctionCall checks whether expr is a call of a function with a given name
func isFunctionCall(expr build.Expr, name string) (*build.CallExpr, bool) {
	call, ok := expr.(*build.CallExpr)
	if !ok {
		return nil, false
	}
	if ident, ok := call.X.(*build.Ident); ok && ident.Name == name {
		return call, true
	}
	return nil, false
}

// globalVariableUsageCheck checks whether there's a usage of a given global variable in the file.
// It's ok to shadow the name with a local variable and use it.
func globalVariableUsageCheck(f *build.File, global, alternative string) []*LinterFinding {
	var findings []*LinterFinding

	if f.Type != build.TypeBzl {
		return findings
	}

	var walk func(e *build.Expr, env *bzlenv.Environment)
	walk = func(e *build.Expr, env *bzlenv.Environment) {
		defer bzlenv.WalkOnceWithEnvironment(*e, env, walk)

		ident, ok := (*e).(*build.Ident)
		if !ok {
			return
		}
		if ident.Name != global {
			return
		}
		if binding := env.Get(ident.Name); binding != nil {
			return
		}

		// Fix
		newIdent := *ident
		newIdent.Name = alternative

		findings = append(findings, makeLinterFinding(ident,
			fmt.Sprintf(`Global variable %q is deprecated in favor of %q. Please rename it.`, global, alternative),
			LinterReplacement{e, &newIdent}))
	}
	var expr build.Expr = f
	walk(&expr, bzlenv.NewEnvironment())

	return findings
}

// insertLoad returns a *LinterReplacement object representing a replacement required for inserting
// an additional load statement. Returns nil if nothing needs to be changed.
func insertLoad(f *build.File, module string, symbols []string) *LinterReplacement {
	// Try to find an existing load statement
	for i, stmt := range f.Stmt {
		load, ok := stmt.(*build.LoadStmt)
		if !ok || load.Module.Value != module {
			continue
		}

		// Modify an existing load statement
		newLoad := *load
		if !edit.AppendToLoad(&newLoad, symbols, symbols) {
			return nil
		}
		return &LinterReplacement{&(f.Stmt[i]), &newLoad}
	}

	// Need to insert a new load statement. Can't modify the tree here, so just insert a placeholder
	// nil statement and return a replacement for it.
	i := 0
	for i = range f.Stmt {
		stmt := f.Stmt[i]
		if _, isComment := stmt.(*build.CommentBlock); isComment {
			continue
		}
		if _, isDocString := stmt.(*build.StringExpr); !isDocString {
			// Insert a nil statement here
			break
		}
	}
	stmts := append([]build.Expr{}, f.Stmt[:i]...)
	stmts = append(stmts, nil)
	stmts = append(stmts, f.Stmt[i:]...)
	f.Stmt = stmts

	return &LinterReplacement{&(f.Stmt[i]), edit.NewLoad(module, symbols, symbols)}
}

// Caches the result of bzlmod.ExtractModuleToApparentNameMapping.
var moduleToApparentRepoName func(string) string

// useApparentRepoNameIfExternal replaces the module name in a load statement with the apparent repository
// name used by the root Bazel module (if any).
func useApparentRepoNameIfExternal(load string, fileReader *FileReader) string {
	if !strings.HasPrefix(load, "@") || fileReader == nil {
		// Not a load from an external repository or we can't load external files.
		return load
	}
	if moduleToApparentRepoName == nil {
		moduleToApparentRepoName = bzlmod.ExtractModuleToApparentNameMapping(func(relPath string) *build.File {
			return fileReader.GetFile("", relPath)
		})
	}
	l := labels.Parse(load)
	apparentName := moduleToApparentRepoName(l.Repository)
	if apparentName == "" {
		// The module that hosts the load is not a bazel_dep of the root module. We assume that's
		// because it is a WORKSPACE repo, which uses the legacy name.
		apparentName = tables.ModuleToLegacyRepoName[l.Repository]
		if apparentName == "" {
			apparentName = l.Repository
		}
	}
	l.Repository = apparentName
	return l.Format()
}

func notLoadedFunctionUsageCheckInternal(expr *build.Expr, env *bzlenv.Environment, globals []string, loadFrom string, fileReader *FileReader) ([]string, []*LinterFinding) {
	var loads []string
	var findings []*LinterFinding

	call, ok := (*expr).(*build.CallExpr)
	if !ok {
		return loads, findings
	}

	var name string
	var replacements []LinterReplacement
	switch node := call.X.(type) {
	case *build.DotExpr:
		// Maybe native.`global`?
		ident, ok := node.X.(*build.Ident)
		if !ok || ident.Name != "native" {
			return loads, findings
		}

		name = node.Name
		// Replace `native.foo()` with `foo()`
		newCall := *call
		newCall.X = &build.Ident{Name: node.Name}
		replacements = append(replacements, LinterReplacement{expr, &newCall})
	case *build.Ident:
		// Maybe `global`()?
		if binding := env.Get(node.Name); binding != nil {
			return loads, findings
		}
		name = node.Name
	default:
		return loads, findings
	}

	for _, global := range globals {
		if name == global {
			loads = append(loads, name)
			findings = append(findings,
				makeLinterFinding(call.X, fmt.Sprintf(`Function %q is not global anymore and needs to be loaded from %q.`, global, useApparentRepoNameIfExternal(loadFrom, fileReader)), replacements...))
			break
		}
	}

	return loads, findings
}

func notLoadedSymbolUsageCheckInternal(expr *build.Expr, env *bzlenv.Environment, globals []string, loadFrom string, fileReader *FileReader) ([]string, []*LinterFinding) {
	var loads []string
	var findings []*LinterFinding

	ident, ok := (*expr).(*build.Ident)
	if !ok {
		return loads, findings
	}
	if binding := env.Get(ident.Name); binding != nil {
		return loads, findings
	}

	for _, global := range globals {
		if ident.Name == global {
			loads = append(loads, ident.Name)
			findings = append(findings,
				makeLinterFinding(ident, fmt.Sprintf(`Symbol %q is not global anymore and needs to be loaded from %q.`, global, useApparentRepoNameIfExternal(loadFrom, fileReader))))
			break
		}
	}

	return loads, findings
}

// notLoadedUsageCheck checks whether there's a usage of a given not imported function or symbol in the file
// and adds a load statement if necessary.
func notLoadedUsageCheck(f *build.File, fileReader *FileReader, functions, symbols []string, loadFrom string) []*LinterFinding {
	toLoad := make(map[string]bool)
	var findings []*LinterFinding

	var walk func(expr *build.Expr, env *bzlenv.Environment)
	walk = func(expr *build.Expr, env *bzlenv.Environment) {
		defer bzlenv.WalkOnceWithEnvironment(*expr, env, walk)

		functionLoads, functionFindings := notLoadedFunctionUsageCheckInternal(expr, env, functions, loadFrom, fileReader)
		findings = append(findings, functionFindings...)
		for _, load := range functionLoads {
			toLoad[load] = true
		}

		symbolLoads, symbolFindings := notLoadedSymbolUsageCheckInternal(expr, env, symbols, loadFrom, fileReader)
		findings = append(findings, symbolFindings...)
		for _, load := range symbolLoads {
			toLoad[load] = true
		}
	}
	var expr build.Expr = f
	walk(&expr, bzlenv.NewEnvironment())

	if len(toLoad) == 0 {
		return nil
	}

	loadFrom = useApparentRepoNameIfExternal(loadFrom, fileReader)

	loads := []string{}
	for l := range toLoad {
		loads = append(loads, l)
	}

	sort.Strings(loads)
	replacement := insertLoad(f, loadFrom, loads)
	if replacement != nil {
		// Add the same replacement to all relevant findings.
		for _, f := range findings {
			f.Replacement = append(f.Replacement, *replacement)
		}
	}

	return findings
}

// NotLoadedFunctionUsageCheck checks whether there's a usage of a given not imported function in the file
// and adds a load statement if necessary.
func NotLoadedFunctionUsageCheck(f *build.File, fileReader *FileReader, globals []string, loadFrom string) []*LinterFinding {
	return notLoadedUsageCheck(f, fileReader, globals, []string{}, loadFrom)
}

// NotLoadedSymbolUsageCheck checks whether there's a usage of a given not imported function in the file
// and adds a load statement if necessary.
func NotLoadedSymbolUsageCheck(f *build.File, fileReader *FileReader, globals []string, loadFrom string) []*LinterFinding {
	return notLoadedUsageCheck(f, fileReader, []string{}, globals, loadFrom)
}

// makePositional makes the function argument positional (removes the keyword if it exists)
func makePositional(argument build.Expr) build.Expr {
	if binary, ok := argument.(*build.AssignExpr); ok {
		return binary.RHS
	}
	return argument
}

// makeKeyword makes the function argument keyword (adds or edits the keyword name)
func makeKeyword(argument build.Expr, name string) build.Expr {
	assign, ok := argument.(*build.AssignExpr)
	if !ok {
		return &build.AssignExpr{
			LHS: &build.Ident{Name: name},
			Op:  "=",
			RHS: argument,
		}
	}
	ident, ok := assign.LHS.(*build.Ident)
	if ok && ident.Name == name {
		// Nothing to change
		return argument
	}

	// Technically it's possible that the LHS is not an ident, but that is a syntax error anyway.
	newAssign := *assign
	newAssign.LHS = &build.Ident{Name: name}
	return &newAssign
}

func attrConfigurationWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	var findings []*LinterFinding
	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		// Find nodes that match the following pattern: attr.xxxx(..., cfg = "data", ...) and attr.xxxx(..., cfg = "host", ...)
		call, ok := (*expr).(*build.CallExpr)
		if !ok {
			return
		}
		dot, ok := (call.X).(*build.DotExpr)
		if !ok {
			return
		}
		base, ok := dot.X.(*build.Ident)
		if !ok || base.Name != "attr" {
			return
		}
		i, _, param := getParam(call.List, "cfg")
		if param == nil {
			return
		}
		value, ok := (param.RHS).(*build.StringExpr)
		if !ok {
			return
		}

		newCall := *call
		switch value.Value {
		case "data":
			newCall.List = append(newCall.List[:i], newCall.List[i+1:]...)
			findings = append(findings,
				makeLinterFinding(param, `cfg = "data" for attr definitions has no effect and should be removed.`,
					LinterReplacement{expr, &newCall}))

		case "host":
			{
				newCall.List = append([]build.Expr{}, newCall.List...)
				newParam := newCall.List[i].Copy().(*build.AssignExpr)
				newRHS := newParam.RHS.Copy().(*build.StringExpr)
				newRHS.Value = "exec"
				newParam.RHS = newRHS
				newCall.List[i] = newParam
				findings = append(findings,
					makeLinterFinding(param, `cfg = "host" for attr definitions should be replaced by cfg = "exec".`,
						LinterReplacement{expr, &newCall}))
			}

		default:
			// value not matched.
			return
		}
	})
	return findings
}

func depsetItemsWarning(f *build.File) []*LinterFinding {
	var findings []*LinterFinding

	types := DetectTypes(f)
	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		call, ok := (*expr).(*build.CallExpr)
		if !ok {
			return
		}
		base, ok := call.X.(*build.Ident)
		if !ok || base.Name != "depset" {
			return
		}
		if len(call.List) == 0 {
			return
		}
		_, _, param := getParam(call.List, "items")
		if param != nil {
			findings = append(findings,
				makeLinterFinding(param, `Parameter "items" is deprecated, use "direct" and/or "transitive" instead.`))
			return
		}
		if _, ok := call.List[0].(*build.AssignExpr); ok {
			return
		}
		// We have an unnamed first parameter. Check the type.
		if types[call.List[0]] == Depset {
			findings = append(findings,
				makeLinterFinding(call.List[0], `Giving a depset as first unnamed parameter to depset() is deprecated, use the "transitive" parameter instead.`))
		}
	})
	return findings
}

func attrNonEmptyWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	var findings []*LinterFinding
	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		// Find nodes that match the following pattern: attr.xxxx(..., non_empty = ..., ...)
		call, ok := (*expr).(*build.CallExpr)
		if !ok {
			return
		}
		dot, ok := (call.X).(*build.DotExpr)
		if !ok {
			return
		}
		base, ok := dot.X.(*build.Ident)
		if !ok || base.Name != "attr" {
			return
		}
		_, name, param := getParam(call.List, "non_empty")
		if param == nil {
			return
		}

		// Fix
		newName := *name
		newName.Name = "allow_empty"
		negatedRHS := negateExpression(param.RHS)

		findings = append(findings,
			makeLinterFinding(param, "non_empty attributes for attr definitions are deprecated in favor of allow_empty.",
				LinterReplacement{&param.LHS, &newName},
				LinterReplacement{&param.RHS, negatedRHS},
			))
	})
	return findings
}

func attrSingleFileWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	var findings []*LinterFinding
	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		// Find nodes that match the following pattern: attr.xxxx(..., single_file = ..., ...)
		call, ok := (*expr).(*build.CallExpr)
		if !ok {
			return
		}
		dot, ok := (call.X).(*build.DotExpr)
		if !ok {
			return
		}
		base, ok := dot.X.(*build.Ident)
		if !ok || base.Name != "attr" {
			return
		}
		singleFileIndex, singleFileKw, singleFileParam := getParam(call.List, "single_file")
		if singleFileParam == nil {
			return
		}

		// Fix
		newCall := *call
		newCall.List = append([]build.Expr{}, call.List...)

		newSingleFileKw := *singleFileKw
		newSingleFileKw.Name = "allow_single_file"
		singleFileValue := singleFileParam.RHS

		if boolean, ok := singleFileValue.(*build.Ident); ok && boolean.Name == "False" {
			// if the value is `False`, just remove the whole parameter
			newCall.List = append(newCall.List[:singleFileIndex], newCall.List[singleFileIndex+1:]...)
		} else {
			// search for `allow_files` parameter in the same attr definition and remove it
			allowFileIndex, _, allowFilesParam := getParam(call.List, "allow_files")
			if allowFilesParam != nil {
				singleFileValue = allowFilesParam.RHS
				newCall.List = append(newCall.List[:allowFileIndex], newCall.List[allowFileIndex+1:]...)
				if singleFileIndex > allowFileIndex {
					singleFileIndex--
				}
			}
		}
		findings = append(findings,
			makeLinterFinding(singleFileParam, "single_file is deprecated in favor of allow_single_file.",
				LinterReplacement{expr, &newCall},
				LinterReplacement{&singleFileParam.LHS, &newSingleFileKw},
				LinterReplacement{&singleFileParam.RHS, singleFileValue},
			))
	})
	return findings
}

func ctxActionsWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	var findings []*LinterFinding
	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		// Find nodes that match the following pattern: ctx.xxxx(...)
		call, ok := (*expr).(*build.CallExpr)
		if !ok {
			return
		}
		dot, ok := (call.X).(*build.DotExpr)
		if !ok {
			return
		}
		base, ok := dot.X.(*build.Ident)
		if !ok || base.Name != "ctx" {
			return
		}

		switch dot.Name {
		case "new_file", "experimental_new_directory", "file_action", "action", "empty_action", "template_action":
			// fix
		default:
			return
		}

		// Fix
		newCall := *call
		newCall.List = append([]build.Expr{}, call.List...)
		newDot := *dot
		newCall.X = &newDot

		switch dot.Name {
		case "new_file":
			if len(call.List) > 2 {
				// Can't fix automatically because the new API doesn't support the 3 arguments signature
				findings = append(findings,
					makeLinterFinding(dot, fmt.Sprintf(`"ctx.new_file" is deprecated in favor of "ctx.actions.declare_file".`)))
				return
			}
			newDot.Name = "actions.declare_file"
			if len(call.List) == 2 {
				// swap arguments:
				// ctx.new_file(sibling, name) -> ctx.actions.declare_file(name, sibling=sibling)
				newCall.List[0], newCall.List[1] = makePositional(call.List[1]), makeKeyword(call.List[0], "sibling")
			}
		case "experimental_new_directory":
			newDot.Name = "actions.declare_directory"
		case "file_action":
			newDot.Name = "actions.write"
			i, ident, param := getParam(newCall.List, "executable")
			if ident != nil {
				newIdent := *ident
				newIdent.Name = "is_executable"
				newParam := *param
				newParam.LHS = &newIdent
				newCall.List[i] = &newParam
			}
		case "action":
			newDot.Name = "actions.run"
			if _, _, command := getParam(call.List, "command"); command != nil {
				newDot.Name = "actions.run_shell"
			}
		case "empty_action":
			newDot.Name = "actions.do_nothing"
		case "template_action":
			newDot.Name = "actions.expand_template"
			if i, ident, param := getParam(call.List, "executable"); ident != nil {
				newIdent := *ident
				newIdent.Name = "is_executable"
				newParam := *param
				newParam.LHS = &newIdent
				newCall.List[i] = &newParam
			}
		}

		findings = append(findings, makeLinterFinding(dot,
			fmt.Sprintf(`"ctx.%s" is deprecated in favor of "ctx.%s".`, dot.Name, newDot.Name),
			LinterReplacement{expr, &newCall}))
	})
	return findings
}

func fileTypeWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	var findings []*LinterFinding
	var walk func(e *build.Expr, env *bzlenv.Environment)
	walk = func(e *build.Expr, env *bzlenv.Environment) {
		defer bzlenv.WalkOnceWithEnvironment(*e, env, walk)

		call, ok := isFunctionCall(*e, "FileType")
		if !ok {
			return
		}
		if binding := env.Get("FileType"); binding == nil {
			findings = append(findings,
				makeLinterFinding(call, "The FileType function is deprecated."))
		}
	}
	var expr build.Expr = f
	walk(&expr, bzlenv.NewEnvironment())

	return findings
}

func packageNameWarning(f *build.File) []*LinterFinding {
	return globalVariableUsageCheck(f, "PACKAGE_NAME", "native.package_name()")
}

func repositoryNameWarning(f *build.File) []*LinterFinding {
	return globalVariableUsageCheck(f, "REPOSITORY_NAME", "native.repository_name()")
}

func outputGroupWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	var findings []*LinterFinding
	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		// Find nodes that match the following pattern: ctx.attr.xxx.output_group
		outputGroup, ok := (*expr).(*build.DotExpr)
		if !ok || outputGroup.Name != "output_group" {
			return
		}
		dep, ok := (outputGroup.X).(*build.DotExpr)
		if !ok {
			return
		}
		attr, ok := (dep.X).(*build.DotExpr)
		if !ok || attr.Name != "attr" {
			return
		}
		ctx, ok := (attr.X).(*build.Ident)
		if !ok || ctx.Name != "ctx" {
			return
		}

		// Replace `xxx.output_group` with `xxx[OutputGroupInfo]`
		findings = append(findings,
			makeLinterFinding(outputGroup,
				`"ctx.attr.dep.output_group" is deprecated in favor of "ctx.attr.dep[OutputGroupInfo]".`,
				LinterReplacement{expr, &build.IndexExpr{
					X: dep,
					Y: &build.Ident{Name: "OutputGroupInfo"},
				},
				}))
	})
	return findings
}

func nativeGitRepositoryWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}
	return NotLoadedFunctionUsageCheck(f, fileReader, []string{"git_repository", "new_git_repository"}, "@bazel_tools//tools/build_defs/repo:git.bzl")
}

func nativeHTTPArchiveWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}
	return NotLoadedFunctionUsageCheck(f, fileReader, []string{"http_archive"}, "@bazel_tools//tools/build_defs/repo:http.bzl")
}

func nativeAndroidRulesWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if f.Type != build.TypeBzl && f.Type != build.TypeBuild {
		return nil
	}
	return NotLoadedFunctionUsageCheck(f, fileReader, tables.AndroidNativeRules, tables.AndroidLoadPath)
}

// NativeCcRulesWarning produces a warning for missing loads of cc rules
func NativeCcRulesWarning(rule string) func(f *build.File, fileReader *FileReader) []*LinterFinding {
	return func(f *build.File, fileReader *FileReader) []*LinterFinding {
		if f.Type != build.TypeBzl && f.Type != build.TypeBuild {
			return nil
		}
		return NotLoadedFunctionUsageCheck(f, fileReader, []string{rule}, tables.CcLoadPathPrefix+":"+rule+".bzl")
	}
}

// NativeCcToolchainRulesWarning produces a warning for missing loads of cc toolchain rules
func NativeCcToolchainRulesWarning(rule string) func(f *build.File, fileReader *FileReader) []*LinterFinding {
	return func(f *build.File, fileReader *FileReader) []*LinterFinding {
		if f.Type != build.TypeBzl && f.Type != build.TypeBuild {
			return nil
		}
		return NotLoadedFunctionUsageCheck(f, fileReader, []string{rule}, tables.CcLoadPathPrefix+"/toolchains:"+rule+".bzl")
	}
}

// NativeCcSymbolsWarning produces a warning for missing loads of cc top-level symbols
func NativeCcSymbolsWarning(symbol string, bzlfile string) func(f *build.File, fileReader *FileReader) []*LinterFinding {
	return func(f *build.File, fileReader *FileReader) []*LinterFinding {
		if f.Type != build.TypeBzl && f.Type != build.TypeBuild {
			return nil
		}
		return NotLoadedSymbolUsageCheck(f, fileReader, []string{symbol}, tables.CcLoadPathPrefix+"/common:"+bzlfile+".bzl")
	}
}

// NativeJavaRulesWarning produces a warning for missing loads of java rules
func NativeJavaRulesWarning(rule string) func(f *build.File, fileReader *FileReader) []*LinterFinding {
	return func(f *build.File, fileReader *FileReader) []*LinterFinding {
		if f.Type != build.TypeBzl && f.Type != build.TypeBuild {
			return nil
		}
		return NotLoadedFunctionUsageCheck(f, fileReader, []string{rule}, tables.JavaLoadPathPrefix+":"+rule+".bzl")
	}
}

// NativeJavaToolchainRulesWarning produces a warning for missing loads of java toolchain rules
func NativeJavaToolchainRulesWarning(rule string) func(f *build.File, fileReader *FileReader) []*LinterFinding {
	return func(f *build.File, fileReader *FileReader) []*LinterFinding {
		if f.Type != build.TypeBzl && f.Type != build.TypeBuild {
			return nil
		}
		return NotLoadedFunctionUsageCheck(f, fileReader, []string{rule}, tables.JavaLoadPathPrefix+"/toolchains:"+rule+".bzl")
	}
}

// NativeJavaSymbolsWarning produces a warning for missing loads of java top-level symbols
func NativeJavaSymbolsWarning(symbol string, bzlfile string) func(f *build.File, fileReader *FileReader) []*LinterFinding {
	return func(f *build.File, fileReader *FileReader) []*LinterFinding {
		if f.Type != build.TypeBzl && f.Type != build.TypeBuild {
			return nil
		}
		return NotLoadedSymbolUsageCheck(f, fileReader, []string{symbol}, tables.JavaLoadPathPrefix+"/common:"+bzlfile+".bzl")
	}
}

func nativePyRulesWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if f.Type != build.TypeBzl && f.Type != build.TypeBuild {
		return nil
	}
	return NotLoadedFunctionUsageCheck(f, fileReader, tables.PyNativeRules, tables.PyLoadPath)
}

// NativeProtoRulesWarning produces a warning for missing loads of proto rules
func NativeProtoRulesWarning(rule string) func(f *build.File, fileReader *FileReader) []*LinterFinding {
	return func(f *build.File, fileReader *FileReader) []*LinterFinding {
		if f.Type != build.TypeBzl && f.Type != build.TypeBuild {
			return nil
		}
		return NotLoadedFunctionUsageCheck(f, fileReader, []string{rule}, tables.ProtoLoadPathPrefix+":"+rule+".bzl")
	}
}

func nativeProtoLangToolchainWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if f.Type != build.TypeBzl && f.Type != build.TypeBuild {
		return nil
	}
	return NotLoadedFunctionUsageCheck(f, fileReader, []string{"proto_lang_toolchain"}, tables.ProtoLoadPathPrefix+"/toolchains:proto_lang_toolchain.bzl")
}

func nativeProtoSymbolsWarning(symbol string, bzlfile string) func(f *build.File, fileReader *FileReader) []*LinterFinding {
	return func(f *build.File, fileReader *FileReader) []*LinterFinding {
		if f.Type != build.TypeBzl && f.Type != build.TypeBuild {
			return nil
		}
		return NotLoadedSymbolUsageCheck(f, fileReader, []string{symbol}, tables.ProtoLoadPathPrefix+"/common:"+bzlfile)
	}
}

// NativeShellRulesWarning produces a warning for missing loads of shell rules
func NativeShellRulesWarning(rule string) func(f *build.File, fileReader *FileReader) []*LinterFinding {
	return func(f *build.File, fileReader *FileReader) []*LinterFinding {
		if f.Type != build.TypeBzl && f.Type != build.TypeBuild {
			return nil
		}
		return NotLoadedFunctionUsageCheck(f, fileReader, []string{rule}, tables.ShellLoadPathPrefix+":"+rule+".bzl")
	}
}

func contextArgsAPIWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	var findings []*LinterFinding
	types := DetectTypes(f)

	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		// Search for `<ctx.actions.args>.add()` nodes
		call, ok := (*expr).(*build.CallExpr)
		if !ok {
			return
		}
		dot, ok := call.X.(*build.DotExpr)
		if !ok || dot.Name != "add" || types[dot.X] != CtxActionsArgs {
			return
		}

		// If neither before_each nor join_with nor map_fn is specified, the node is ok.
		// Otherwise if join_with is specified, use `.add_joined` instead.
		// Otherwise use `.add_all` instead.

		_, beforeEachKw, beforeEach := getParam(call.List, "before_each")
		_, _, joinWith := getParam(call.List, "join_with")
		_, mapFnKw, mapFn := getParam(call.List, "map_fn")
		if beforeEach == nil && joinWith == nil && mapFn == nil {
			// No deprecated API detected
			return
		}

		// Fix
		var replacements []LinterReplacement

		newDot := *dot
		newDot.Name = "add_all"
		replacements = append(replacements, LinterReplacement{&call.X, &newDot})

		if joinWith != nil {
			newDot.Name = "add_joined"
			if beforeEach != nil {
				// `add_joined` doesn't have a `before_each` parameter, replace it with `format_each`:
				// `before_each = foo` -> `format_each = foo + "%s"`
				newBeforeEachKw := *beforeEachKw
				newBeforeEachKw.Name = "format_each"

				replacements = append(replacements, LinterReplacement{&beforeEach.LHS, &newBeforeEachKw})
				replacements = append(replacements, LinterReplacement{&beforeEach.RHS, &build.BinaryExpr{
					X:  beforeEach.RHS,
					Op: "+",
					Y:  &build.StringExpr{Value: "%s"},
				}})
			}
		}
		if mapFnKw != nil {
			// Replace `map_fn = ...` with `map_each = ...`
			newMapFnKw := *mapFnKw
			newMapFnKw.Name = "map_each"
			replacements = append(replacements, LinterReplacement{&mapFn.LHS, &newMapFnKw})
		}

		findings = append(findings,
			makeLinterFinding(call,
				`"ctx.actions.args().add()" for multiple arguments is deprecated in favor of "add_all()" or "add_joined()".`,
				replacements...))

	})
	return findings
}

func attrOutputDefaultWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	var findings []*LinterFinding
	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		// Find nodes that match the following pattern: attr.output(..., default = ...)
		call, ok := expr.(*build.CallExpr)
		if !ok {
			return
		}
		dot, ok := (call.X).(*build.DotExpr)
		if !ok || dot.Name != "output" {
			return
		}
		base, ok := dot.X.(*build.Ident)
		if !ok || base.Name != "attr" {
			return
		}
		_, _, param := getParam(call.List, "default")
		if param == nil {
			return
		}
		findings = append(findings,
			makeLinterFinding(param, `The "default" parameter for attr.output() is deprecated.`))
	})
	return findings
}

func attrLicenseWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	var findings []*LinterFinding
	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		// Find nodes that match the following pattern: attr.license(...)
		call, ok := expr.(*build.CallExpr)
		if !ok {
			return
		}
		dot, ok := (call.X).(*build.DotExpr)
		if !ok || dot.Name != "license" {
			return
		}
		base, ok := dot.X.(*build.Ident)
		if !ok || base.Name != "attr" {
			return
		}
		findings = append(findings,
			makeLinterFinding(expr, `"attr.license()" is deprecated and shouldn't be used.`))
	})
	return findings
}

// ruleImplReturnWarning checks whether a rule implementation function returns an old-style struct
func ruleImplReturnWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	var findings []*LinterFinding

	// iterate over rules and collect rule implementation function names
	implNames := make(map[string]bool)
	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		call, ok := isFunctionCall(expr, "rule")
		if !ok {
			return
		}

		// Try to get the implementaton parameter either by name or as the first argument
		var impl build.Expr
		_, _, param := getParam(call.List, "implementation")
		if param != nil {
			impl = param.RHS
		} else if len(call.List) > 0 {
			impl = call.List[0]
		}
		if name, ok := impl.(*build.Ident); ok {
			implNames[name.Name] = true
		}
	})

	// iterate over functions
	for _, stmt := range f.Stmt {
		def, ok := stmt.(*build.DefStmt)
		if !ok || !implNames[def.Name] {
			// either not a function or not used in the file as a rule implementation function
			continue
		}
		// traverse the function and find all of its return statements
		build.Walk(def, func(expr build.Expr, stack []build.Expr) {
			ret, ok := expr.(*build.ReturnStmt)
			if !ok {
				return
			}
			// check whether it returns a struct
			if _, ok := isFunctionCall(ret.Result, "struct"); ok {
				findings = append(findings, makeLinterFinding(ret, `Avoid using the legacy provider syntax.`))
			}
		})
	}

	return findings
}

type signature struct {
	Positional []string // These parameters are typePositional-only
	Keyword    []string // These parameters are typeKeyword-only
}

var signatures = map[string]signature{
	"all":     {[]string{"elements"}, []string{}},
	"any":     {[]string{"elements"}, []string{}},
	"tuple":   {[]string{"x"}, []string{}},
	"list":    {[]string{"x"}, []string{}},
	"len":     {[]string{"x"}, []string{}},
	"str":     {[]string{"x"}, []string{}},
	"repr":    {[]string{"x"}, []string{}},
	"bool":    {[]string{"x"}, []string{}},
	"int":     {[]string{"x"}, []string{}},
	"dir":     {[]string{"x"}, []string{}},
	"type":    {[]string{"x"}, []string{}},
	"hasattr": {[]string{"x", "name"}, []string{}},
	"getattr": {[]string{"x", "name", "default"}, []string{}},
	"select":  {[]string{"x"}, []string{}},
}

// functionName returns the name of the given function if it's a direct function call (e.g.
// `foo(...)` or `native.foo(...)`, but not `foo.bar(...)` or `x[3](...)`
func functionName(call *build.CallExpr) (string, bool) {
	if ident, ok := call.X.(*build.Ident); ok {
		return ident.Name, true
	}
	// Also check for `native.<name>`
	dot, ok := call.X.(*build.DotExpr)
	if !ok {
		return "", false
	}
	if ident, ok := dot.X.(*build.Ident); !ok || ident.Name != "native" {
		return "", false
	}
	return dot.Name, true
}

const (
	typePositional int = iota
	typeKeyword
	typeArgs
	typeKwargs
)

// paramType returns the type of the param. If it's a typeKeyword param, also returns its name
func paramType(param build.Expr) (int, string) {
	switch param := param.(type) {
	case *build.AssignExpr:
		if param.Op == "=" {
			ident, ok := param.LHSIdent()
			if ok {
				return typeKeyword, ident.Name
			}
			return typeKeyword, ""
		}
	case *build.UnaryExpr:
		switch param.Op {
		case "*":
			return typeArgs, ""
		case "**":
			return typeKwargs, ""
		}
	}
	return typePositional, ""
}

// keywordPositionalParametersWarning checks for deprecated typeKeyword parameters of builtins
func keywordPositionalParametersWarning(f *build.File) []*LinterFinding {
	var findings []*LinterFinding

	// Check for legacy typeKeyword parameters
	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		call, ok := (*expr).(*build.CallExpr)
		if !ok || len(call.List) == 0 {
			return
		}
		function, ok := functionName(call)
		if !ok {
			return
		}

		// Findings and replacements for the current call expression
		var callFindings []*LinterFinding
		var callReplacements []LinterReplacement

		signature, ok := signatures[function]
		if !ok {
			return
		}

		var paramTypes []int // types of the parameters (typeKeyword or not) after the replacements has been applied.
		for i, parameter := range call.List {
			pType, name := paramType(parameter)
			paramTypes = append(paramTypes, pType)

			if pType == typeKeyword && i < len(signature.Positional) && signature.Positional[i] == name {
				// The parameter should be typePositional
				callFindings = append(callFindings, makeLinterFinding(
					parameter,
					fmt.Sprintf(`Keyword parameter %q for %q should be positional.`, signature.Positional[i], function),
				))
				callReplacements = append(callReplacements, LinterReplacement{&call.List[i], makePositional(parameter)})
				paramTypes[i] = typePositional
			}

			if pType == typePositional && i >= len(signature.Positional) && i < len(signature.Positional)+len(signature.Keyword) {
				// The parameter should be typeKeyword
				keyword := signature.Keyword[i-len(signature.Positional)]
				callFindings = append(callFindings, makeLinterFinding(
					parameter,
					fmt.Sprintf(`Parameter at the position %d for %q should be keyword (%s = ...).`, i+1, function, keyword),
				))
				callReplacements = append(callReplacements, LinterReplacement{&call.List[i], makeKeyword(parameter, keyword)})
				paramTypes[i] = typeKeyword
			}
		}

		if len(callFindings) == 0 {
			return
		}

		// Only apply the replacements if the signature is correct after they have been applied
		// (i.e. the order of the parameters is typePositional, typeKeyword, typeArgs, typeKwargs)
		// Otherwise the signature will be not correct, probably it was incorrect initially.
		// All the replacements should be applied to the first finding for the current node.

		if sort.IntsAreSorted(paramTypes) {
			// It's possible that the parameter list had `ForceCompact` set to true because it only contained
			// positional arguments, and now it has keyword arguments as well. Reset the flag to let the
			// printer decide how the function call should be formatted.
			for _, t := range paramTypes {
				if t == typeKeyword {
					// There's at least one keyword argument
					newCall := *call
					newCall.ForceCompact = false
					callFindings[0].Replacement = append(callFindings[0].Replacement, LinterReplacement{expr, &newCall})
					break
				}
			}
			// Attach all the parameter replacements to the first finding
			callFindings[0].Replacement = append(callFindings[0].Replacement, callReplacements...)
		}

		findings = append(findings, callFindings...)
	})

	return findings
}

func providerParamsWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	var findings []*LinterFinding
	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		call, ok := isFunctionCall(expr, "provider")
		if !ok {
			return
		}

		_, _, fields := getParam(call.List, "fields")
		_, _, doc := getParam(call.List, "doc")
		// doc can also be the first positional argument
		hasPositional := false
		if len(call.List) > 0 {
			if _, ok := call.List[0].(*build.AssignExpr); !ok {
				hasPositional = true
			}
		}
		msg := ""
		if fields == nil {
			msg = "a list of fields"
		}
		if doc == nil && !hasPositional {
			if msg != "" {
				msg += " and "
			}
			msg += "a documentation"
		}
		if msg != "" {
			findings = append(findings, makeLinterFinding(call,
				`Calls to 'provider' should provide `+msg+`:\n`+
					`  provider("description", fields = [...])`))
		}
	})
	return findings
}

func attrNameWarning(f *build.File, names []string) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	var findings []*LinterFinding
	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		// Find nodes that match "attrs = {..., "license", ...}"
		dict, ok := (*expr).(*build.DictExpr)
		if !ok {
			return
		}
		for _, item := range dict.List {
			// include only string literal keys into consideration
			value, ok := item.Key.(*build.StringExpr)
			if !ok {
				continue
			}
			for _, name := range names {
				if value.Value == name {
					findings = append(findings, makeLinterFinding(dict,
						fmt.Sprintf(`Do not use '%s' as an attribute name.`+
							` It may cause unexpected behavior.`, value.Value)))

				}
			}
		}
	})
	return findings
}

func attrLicensesWarning(f *build.File) []*LinterFinding {
	return attrNameWarning(f, []string{"licenses"})
}

func attrApplicableLicensesWarning(f *build.File) []*LinterFinding {
	return attrNameWarning(f, []string{"applicable_licenses", "package_metadata"})
}
