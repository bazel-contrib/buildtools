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

// pathStringKind classifies expressions that return a path string which may contain a
// configuration prefix (such as `bazel-out/k8-fastbuild/bin`) and is therefore not rewritten by
// path mapping.
type pathStringKind int

const (
	notPathString pathStringKind = iota
	filePath                     // `file.path`
	fileDirname                  // `file.dirname`
	fileRootPath                 // `file.root.path`
	ctxDirPath                   // `ctx.bin_dir.path` or `ctx.genfiles_dir.path`
)

// pathStringContext describes where a path string is used.
type pathStringContext int

const (
	argsContext       pathStringContext = iota // `args.add()`, `args.add_all()` or `args.add_joined()`
	executableContext                          // `executable` of `ctx.actions.run()`
	argumentsContext                           // `arguments` of `ctx.actions.run()` or `ctx.actions.run_shell()`
	commandContext                             // `command` of `ctx.actions.run_shell()`
)

// classifyPathString returns the kind of path string the expression evaluates to. The parent is
// used to skip method calls such as `ctx.path(...)` or `paths.dirname(...)`.
func classifyPathString(expr, parent build.Expr) pathStringKind {
	dot, ok := expr.(*build.DotExpr)
	if !ok {
		return notPathString
	}
	if call, ok := parent.(*build.CallExpr); ok && call.X == expr {
		return notPathString
	}
	switch dot.Name {
	case "dirname":
		return fileDirname
	case "path":
		if inner, ok := dot.X.(*build.DotExpr); ok {
			switch inner.Name {
			case "root":
				return fileRootPath
			case "bin_dir", "genfiles_dir":
				return ctxDirPath
			}
		}
		return filePath
	}
	return notPathString
}

// pathStringMessage builds a message for a path string of the given kind used in the given context.
func pathStringMessage(dot *build.DotExpr, kind pathStringKind, context pathStringContext, method string) string {
	expr := build.FormatString(dot)
	receiver := build.FormatString(dot.X)
	msg := fmt.Sprintf("%q is not rewritten by path mapping. ", expr)
	switch kind {
	case fileDirname:
		return msg + fmt.Sprintf(`Add %q to "ctx.actions.args()" with a "map_each" callback that returns "file.dirname" instead.`, receiver)
	case fileRootPath:
		file := build.FormatString(dot.X.(*build.DotExpr).X)
		return msg + fmt.Sprintf(`Add %q to "ctx.actions.args()" with a "map_each" callback that returns "file.root.path" instead.`, file)
	case ctxDirPath:
		return msg + `Compute it from an output "File" in a "map_each" callback via "file.root.path" instead.`
	}
	switch context {
	case argsContext:
		if strings.HasSuffix(method, ".add") {
			return msg + fmt.Sprintf(`Pass %q to "%s()" instead, or use "%s_all([%s], expand_directories = False)" if it is a directory.`,
				receiver, method, method, receiver)
		}
		return msg + fmt.Sprintf(`Pass %q to "%s()" instead (with "expand_directories = False" if it is a directory).`, receiver, method)
	case executableContext:
		return msg + fmt.Sprintf(`Pass %q as "executable" instead.`, receiver)
	case argumentsContext:
		return msg + fmt.Sprintf(`Add %q to "ctx.actions.args()" and pass the "Args" object in "arguments" instead.`, receiver)
	default:
		return msg + fmt.Sprintf(`Add %q to "ctx.actions.args()", pass the "Args" object in "arguments" and refer to it as "$1", "$2", ... in "command" instead.`, receiver)
	}
}

// isCtxActionsArgs reports whether expr is a `ctx.actions.args()` object. Besides the detected
// types, a variable named `args` is assumed to be one, e.g. a parameter of a helper function.
func isCtxActionsArgs(expr build.Expr, types map[build.Expr]Type) bool {
	if types[expr] == CtxActionsArgs {
		return true
	}
	ident, ok := expr.(*build.Ident)
	return ok && ident.Name == "args"
}

// collectDeclaredDirectories returns the names of all variables that are assigned the result of
// `ctx.actions.declare_directory()` anywhere in the file.
func collectDeclaredDirectories(f *build.File) map[string]bool {
	directories := make(map[string]bool)
	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		assign, ok := expr.(*build.AssignExpr)
		if !ok {
			return
		}
		ident, ok := assign.LHS.(*build.Ident)
		if !ok {
			return
		}
		call, ok := assign.RHS.(*build.CallExpr)
		if !ok {
			return
		}
		if dot, ok := call.X.(*build.DotExpr); ok && dot.Name == "declare_directory" {
			directories[ident.Name] = true
		}
	})
	return directories
}

// isValidArgsFormat reports whether str can be used as a `format` parameter of
// `ctx.actions.args()` methods, i.e. it contains exactly one `%s` and no other `%`.
func isValidArgsFormat(str string) bool {
	return strings.Count(str, "%") == 1 && strings.Contains(str, "%s")
}

// splitFormattedPath checks whether expr formats a single `file.path` expression into a string
// literal, e.g. `"-I%s" % file.path`, `"-I" + file.path`, `file.path + ".d"` or
// `"-I{}".format(file.path)`. It returns the equivalent `format` string for `ctx.actions.args()`
// and the formatted `file.path` expression.
func splitFormattedPath(expr build.Expr) (string, *build.DotExpr, bool) {
	switch node := expr.(type) {
	case *build.ParenExpr:
		return splitFormattedPath(node.X)
	case *build.BinaryExpr:
		switch node.Op {
		case "%":
			str, ok := node.X.(*build.StringExpr)
			if !ok || !isValidArgsFormat(str.Value) {
				return "", nil, false
			}
			dot, ok := node.Y.(*build.DotExpr)
			if !ok || classifyPathString(dot, nil) != filePath {
				return "", nil, false
			}
			return str.Value, dot, true
		case "+":
			var format strings.Builder
			var dot *build.DotExpr
			for _, operand := range flattenConcatenation(node) {
				switch operand := operand.(type) {
				case *build.StringExpr:
					if strings.Contains(operand.Value, "%") {
						return "", nil, false
					}
					format.WriteString(operand.Value)
				case *build.DotExpr:
					if dot != nil || classifyPathString(operand, nil) != filePath {
						return "", nil, false
					}
					dot = operand
					format.WriteString("%s")
				default:
					return "", nil, false
				}
			}
			if dot == nil {
				return "", nil, false
			}
			return format.String(), dot, true
		}
	case *build.CallExpr:
		method, ok := node.X.(*build.DotExpr)
		if !ok || method.Name != "format" || len(node.List) != 1 {
			return "", nil, false
		}
		str, ok := method.X.(*build.StringExpr)
		if !ok || strings.Count(str.Value, "{}") != 1 || strings.Count(str.Value, "{")+strings.Count(str.Value, "}") != 2 || strings.Contains(str.Value, "%") {
			return "", nil, false
		}
		dot, ok := node.List[0].(*build.DotExpr)
		if !ok || classifyPathString(dot, nil) != filePath {
			return "", nil, false
		}
		return strings.Replace(str.Value, "{}", "%s", 1), dot, true
	}
	return "", nil, false
}

// flattenConcatenation returns the operands of a chain of `+` operations.
func flattenConcatenation(expr build.Expr) []build.Expr {
	if binary, ok := expr.(*build.BinaryExpr); ok && binary.Op == "+" {
		return append(flattenConcatenation(binary.X), flattenConcatenation(binary.Y)...)
	}
	if paren, ok := expr.(*build.ParenExpr); ok {
		return flattenConcatenation(paren.X)
	}
	return []build.Expr{expr}
}

// simpleComprehension checks whether expr is a list comprehension of the form `[body for var in seq]`
// and returns the loop variable, the body and the sequence.
func simpleComprehension(expr build.Expr) (*build.Ident, build.Expr, build.Expr, bool) {
	comp, ok := expr.(*build.Comprehension)
	if !ok || comp.Curly || len(comp.Clauses) != 1 {
		return nil, nil, nil, false
	}
	forClause, ok := comp.Clauses[0].(*build.ForClause)
	if !ok {
		return nil, nil, nil, false
	}
	loopVar, ok := forClause.Vars.(*build.Ident)
	if !ok {
		return nil, nil, nil, false
	}
	return loopVar, comp.Body, forClause.X, true
}

// isPathOf reports whether expr is `<ident>.path` for the given identifier.
func isPathOf(expr build.Expr, ident *build.Ident) (*build.DotExpr, bool) {
	dot, ok := expr.(*build.DotExpr)
	if !ok || classifyPathString(dot, nil) != filePath {
		return nil, false
	}
	x, ok := dot.X.(*build.Ident)
	return dot, ok && x.Name == ident.Name
}

// pathMappingFixer collects fixes for a single `args.add()`, `args.add_all()` or `args.add_joined()`
// call. A fix either replaces a single node (e.g. `file.path` with `file`) or rewrites the whole
// call (e.g. to add a `format` parameter).
type pathMappingFixer struct {
	call        *build.CallExpr
	callPtr     *build.Expr
	method      string
	directories map[string]bool
	newList     []build.Expr                         // modified copy of call.List
	newMethod   string                               // method name of the rewritten call, if it changes
	rewriteCall bool                                 // the whole call needs to be replaced
	fixed       map[*build.DotExpr]LinterReplacement // node-level fixes
	broken      bool                                 // a safe fix is not possible
}

func newPathMappingFixer(callPtr *build.Expr, call *build.CallExpr, method string, directories map[string]bool) *pathMappingFixer {
	return &pathMappingFixer{
		call:        call,
		callPtr:     callPtr,
		method:      method,
		directories: directories,
		newList:     append([]build.Expr{}, call.List...),
		newMethod:   method,
		fixed:       make(map[*build.DotExpr]LinterReplacement),
	}
}

func (fx *pathMappingFixer) hasKeyword(name string) bool {
	_, _, param := getParam(fx.call.List, name)
	return param != nil
}

func (fx *pathMappingFixer) isDirectory(expr build.Expr) bool {
	ident, ok := expr.(*build.Ident)
	return ok && fx.directories[ident.Name]
}

func (fx *pathMappingFixer) addKeyword(name string, value build.Expr) {
	fx.newList = append(fx.newList, makeKeyword(value, name))
	fx.rewriteCall = true
}

// fixPositional tries to fix the i-th positional argument of the call.
func (fx *pathMappingFixer) fixPositional(i int) {
	arg := fx.call.List[i]
	switch fx.method {
	case "add":
		// `args.add(file.path)`, `args.add("--flag", file.path)`, `args.add("--flag=%s" % file.path)`
		if dot, ok := arg.(*build.DotExpr); ok && classifyPathString(dot, nil) == filePath {
			fx.newList[i] = dot.X
			fx.fixed[dot] = LinterReplacement{&fx.call.List[i], dot.X}
			if fx.isDirectory(dot.X) {
				// `args.add()` doesn't accept directories, use `args.add_all([...], expand_directories = False)`
				fx.newMethod = "add_all"
				fx.rewriteCall = true
			}
			return
		}
		if format, dot, ok := splitFormattedPath(arg); ok && !fx.hasKeyword("format") && !fx.isDirectory(dot.X) {
			fx.newList[i] = dot.X
			fx.fixed[dot] = LinterReplacement{}
			fx.addKeyword("format", &build.StringExpr{Value: format})
		}
	case "add_all", "add_joined":
		if i != 0 || fx.hasKeyword("map_each") {
			return
		}
		switch values := arg.(type) {
		case *build.ListExpr:
			// `args.add_all([a.path, b.path])`
			newValues := *values
			newValues.List = append([]build.Expr{}, values.List...)
			for j, value := range values.List {
				dot, ok := value.(*build.DotExpr)
				if !ok || classifyPathString(dot, nil) != filePath {
					continue
				}
				newValues.List[j] = dot.X
				fx.fixed[dot] = LinterReplacement{&values.List[j], dot.X}
				if fx.isDirectory(dot.X) && !fx.hasKeyword("expand_directories") {
					fx.rewriteCall = true
				}
			}
			if fx.rewriteCall {
				fx.newList[i] = &newValues
				if !fx.hasKeyword("expand_directories") {
					fx.addKeyword("expand_directories", &build.Ident{Name: "False"})
				}
			}
		case *build.Comprehension:
			// `args.add_all([f.path for f in files])`, `args.add_all(["-I" + f.path for f in files])`
			loopVar, body, seq, ok := simpleComprehension(values)
			if !ok {
				return
			}
			if dot, ok := isPathOf(body, loopVar); ok {
				fx.newList[i] = seq
				fx.fixed[dot] = LinterReplacement{&fx.call.List[i], seq}
				return
			}
			format, dot, ok := splitFormattedPath(body)
			if !ok || fx.hasKeyword("format_each") {
				return
			}
			if _, ok := isPathOf(dot, loopVar); !ok {
				return
			}
			fx.newList[i] = seq
			fx.fixed[dot] = LinterReplacement{}
			fx.addKeyword("format_each", &build.StringExpr{Value: format})
		}
	}
}

// replacement returns the fix for the given node, if any.
func (fx *pathMappingFixer) replacement(dot *build.DotExpr) []LinterReplacement {
	fix, ok := fx.fixed[dot]
	if !ok {
		return nil
	}
	if !fx.rewriteCall {
		return []LinterReplacement{fix}
	}
	if fx.newMethod != fx.method && fx.hasKeyword("format") {
		// `args.add(dir.path, format = ...)` can't be converted to `args.add_all()`
		return nil
	}
	newCall := *fx.call
	newCall.List = fx.newList
	if fx.newMethod != fx.method {
		newDot := *fx.call.X.(*build.DotExpr)
		newDot.Name = fx.newMethod
		newCall.X = &newDot
		newCall.List = []build.Expr{
			&build.ListExpr{List: fx.newList},
			makeKeyword(&build.Ident{Name: "False"}, "expand_directories"),
		}
	}
	return []LinterReplacement{{fx.callPtr, &newCall}}
}

// pathMappingWarning reports path strings used in action command lines. Path mapping
// (https://github.com/bazelbuild/bazel/discussions/22658) can only rewrite `File` objects passed to
// `ctx.actions.args()`, so strings obtained from `File.path`, `File.dirname` etc. break actions
// that opt into it.
func pathMappingWarning(f *build.File) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	types := DetectTypes(f)
	directories := collectDeclaredDirectories(f)
	findings := []*LinterFinding{}

	// report reports every path string expression inside a command line argument.
	report := func(arg build.Expr, context pathStringContext, method string, replacement func(dot *build.DotExpr) []LinterReplacement) {
		build.Walk(arg, func(expr build.Expr, stack []build.Expr) {
			var parent build.Expr
			if len(stack) > 0 {
				parent = stack[len(stack)-1]
			}
			kind := classifyPathString(expr, parent)
			if kind == notPathString {
				return
			}
			dot := expr.(*build.DotExpr)
			findings = append(findings, makeLinterFinding(dot, pathStringMessage(dot, kind, context, method), replacement(dot)...))
		})
	}

	build.WalkPointers(f, func(expr *build.Expr, stack []build.Expr) {
		call, ok := (*expr).(*build.CallExpr)
		if !ok {
			return
		}
		dot, ok := call.X.(*build.DotExpr)
		if !ok {
			return
		}

		switch {
		case isCtxActionsArgs(dot.X, types):
			// `args.add(...)`, `args.add_all(...)`, `args.add_joined(...)`
			switch dot.Name {
			case "add", "add_all", "add_joined":
			default:
				return
			}
			fixer := newPathMappingFixer(expr, call, dot.Name, directories)
			for i, arg := range call.List {
				if _, ok := arg.(*build.AssignExpr); !ok {
					fixer.fixPositional(i)
				}
			}
			for _, arg := range call.List {
				report(makePositional(arg), argsContext, build.FormatString(dot), fixer.replacement)
			}
		case types[dot.X] == CtxActions:
			// `ctx.actions.run(...)` and `ctx.actions.run_shell(...)`
			noFix := func(*build.DotExpr) []LinterReplacement { return nil }
			switch dot.Name {
			case "run":
				if _, _, param := getParam(call.List, "executable"); param != nil {
					report(param.RHS, executableContext, dot.Name, func(dot *build.DotExpr) []LinterReplacement {
						if dot == param.RHS && classifyPathString(dot, nil) == filePath {
							return []LinterReplacement{{&param.RHS, dot.X}}
						}
						return nil
					})
				}
				if _, _, param := getParam(call.List, "arguments"); param != nil {
					report(param.RHS, argumentsContext, dot.Name, noFix)
				}
			case "run_shell":
				if _, _, param := getParam(call.List, "command"); param != nil {
					report(param.RHS, commandContext, dot.Name, noFix)
				}
				if _, _, param := getParam(call.List, "arguments"); param != nil {
					report(param.RHS, argumentsContext, dot.Name, noFix)
				}
			}
		}
	})
	return findings
}
