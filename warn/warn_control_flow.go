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

// Warnings related to the control flow

package warn

import (
	"fmt"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/bzlenv"
	"github.com/bazelbuild/buildtools/edit"
)

// findReturnsWithoutValue searches for return statements without a value, calls `callback` on
// them and returns whether the current list of statements terminates (either by a return or fail()
// statements on the current level in all subbranches.
func findReturnsWithoutValue(stmts []build.Expr, callback func(*build.ReturnStmt)) bool {
	if len(stmts) == 0 {
		// May occur in empty else-clauses
		return false
	}
	terminated := false
	for _, stmt := range stmts {
		switch stmt := stmt.(type) {
		case *build.ReturnStmt:
			if stmt.Result == nil {
				callback(stmt)
			}
			terminated = true
		case *build.CallExpr:
			ident, ok := stmt.X.(*build.Ident)
			if ok && ident.Name == "fail" {
				terminated = true
			}
		case *build.ForStmt:
			// Call recursively to find all return statements without a value there.
			// Even if a for-loop is guaranteed to terminate in each iteration, buildifier still can't
			// check whether the loop is not empty, so we can't say that the statement after the ForStmt
			// is unreachable.
			findReturnsWithoutValue(stmt.Body, callback)
		case *build.IfStmt:
			// Save to separate values to avoid short circuit evaluation
			term1 := findReturnsWithoutValue(stmt.True, callback)
			term2 := findReturnsWithoutValue(stmt.False, callback)
			if term1 && term2 {
				terminated = true
			}
		}
	}
	return terminated
}

// missingReturnValueWarning warns if a function returns both explicit and implicit values.
func missingReturnValueWarning(f *build.File) []*LinterFinding {
	findings := []*LinterFinding{}

	// Collect all def statements in the file
	defStmts := []*build.DefStmt{}
	build.WalkStatements(f, func(expr build.Expr, stack []build.Expr) (err error) {
		if def, ok := expr.(*build.DefStmt); ok {
			defStmts = append(defStmts, def)
		}
		return
	})

	for _, function := range defStmts {
		var hasNonEmptyReturns bool
		build.WalkStatements(function, func(expr build.Expr, stack []build.Expr) (err error) {
			if _, ok := expr.(*build.DefStmt); ok {
				if len(stack) > 0 {
					return &build.StopTraversalError{}
				}
			}

			if ret, ok := expr.(*build.ReturnStmt); ok && ret.Result != nil {
				hasNonEmptyReturns = true
			}
			return err
		})

		if !hasNonEmptyReturns {
			continue
		}
		explicitReturn := findReturnsWithoutValue(function.Body, func(ret *build.ReturnStmt) {
			findings = append(findings,
				makeLinterFinding(ret, fmt.Sprintf("Some but not all execution paths of %q return a value.", function.Name)))
		})
		if !explicitReturn {
			findings = append(findings,
				makeLinterFinding(function, fmt.Sprintf(`Some but not all execution paths of %q return a value.
The function may terminate by an implicit return in the end.`, function.Name)))
		}
	}
	return findings
}

// findUnreachableStatements searches for unreachable statements (i.e. statements that immediately
// follow `return`, `break`, `continue`, and `fail()` statements and calls `callback` on them.
// If there are several consequent unreachable statements, it only reports the first of them.
// Returns whether the execution is terminated explicitly.
func findUnreachableStatements(stmts []build.Expr, callback func(build.Expr)) bool {
	unreachable := false
	for _, stmt := range stmts {
		if unreachable {
			if _, ok := stmt.(*build.CommentBlock); ok {
    			continue
    		}
			callback(stmt)
			return true
		}
		switch stmt := stmt.(type) {
		case *build.ReturnStmt:
			unreachable = true
		case *build.CallExpr:
			ident, ok := stmt.X.(*build.Ident)
			if ok && ident.Name == "fail" {
				unreachable = true
			}
		case *build.BranchStmt:
			if stmt.Token != "pass" {
				// either break or continue
				unreachable = true
			}
		case *build.ForStmt:
			findUnreachableStatements(stmt.Body, callback)
		case *build.IfStmt:
			// Save to separate values to avoid short circuit evaluation
			term1 := findUnreachableStatements(stmt.True, callback)
			term2 := findUnreachableStatements(stmt.False, callback)
			if term1 && term2 {
				unreachable = true
			}
		}
	}
	return unreachable
}

func unreachableStatementWarning(f *build.File) []*LinterFinding {
	findings := []*LinterFinding{}

	build.WalkStatements(f, func(expr build.Expr, stack []build.Expr) (err error) {
		def, ok := expr.(*build.DefStmt)
		if !ok {
			return
		}
		findUnreachableStatements(def.Body, func(expr build.Expr) {
			findings = append(findings,
				makeLinterFinding(expr, `The statement is unreachable.`))
		})
		return
	})
	return findings
}

func noEffectStatementsCheck(body []build.Expr, isTopLevel, isFunc bool, findings []*LinterFinding) []*LinterFinding {
	seenNonComment := false
	for _, stmt := range body {
		if stmt == nil {
			continue
		}

		_, isString := stmt.(*build.StringExpr)
		if isString {
			if !seenNonComment && (isTopLevel || isFunc) {
				// It's a docstring.
				seenNonComment = true
				continue
			}
		}
		if _, ok := stmt.(*build.CommentBlock); !ok {
			seenNonComment = true
		}
		switch s := (stmt).(type) {
		case *build.DefStmt, *build.ForStmt, *build.IfStmt, *build.LoadStmt, *build.ReturnStmt,
			*build.CallExpr, *build.CommentBlock, *build.BranchStmt, *build.AssignExpr:
			continue
		case *build.Comprehension:
			if !isTopLevel || s.Curly {
				// List comprehensions are allowed on top-level.
				findings = append(findings,
					makeLinterFinding(stmt, "Expression result is not used. Use a for-loop instead of a list comprehension."))
			}
			continue
		}

		msg := "Expression result is not used."
		if isString {
			msg += " Docstrings should be the first statements of a file or a function (they may follow comment lines)."
		}
		findings = append(findings, makeLinterFinding(stmt, msg))
	}
	return findings
}

func noEffectWarning(f *build.File) []*LinterFinding {
	findings := []*LinterFinding{}
	findings = noEffectStatementsCheck(f.Stmt, true, false, findings)
	build.WalkStatements(f, func(expr build.Expr, stack []build.Expr) (err error) {
		// Docstrings are valid statements without effects. To detect them we need to
		// analyze blocks of statements rather than single statements.
		switch expr := expr.(type) {
		case *build.ForStmt:
			findings = noEffectStatementsCheck(expr.Body, false, false, findings)
		case *build.DefStmt:
			findings = noEffectStatementsCheck(expr.Function.Body, false, true, findings)
		case *build.IfStmt:
			findings = noEffectStatementsCheck(expr.True, false, false, findings)
			findings = noEffectStatementsCheck(expr.False, false, false, findings)
		}
		return
	})
	return findings
}

var mutatingMethodsReturningNone = map[string]bool{
	"append": true,
	"clear":  true,
	"extend": true,
	"insert": true,
	"remove": true,
	"sort":   true,
	"update": true,
}

// extractIdentsFromStmt returns all idents from an AST node representing a
// single statement that are either defined outside the node and used inside,
// or defined inside the node and can be used outside.
// Examples of idents that don't fall into either of the categories:
//   - Named arguments of function calls: `foo` in `f(foo = "bar")`
//   - Iterators of comprehension nodes and its usages: `x` in `[f(x) for x in y]`
//   - Lambda arguments: `x` in `lambda x: x + 1`
//
// Statements that contain other statements (for-loops, if-else blocks) are not
// traversed inside.
func extractIdentsFromStmt(stmt build.Expr, mutatingReceiversCountAsUsed map[string]bool) (assigned, used map[*build.Ident]bool) {
	// The values for `assigned` are `true` if the warning for the variable should
	// be suppressed, and `false` otherwise.
	// It's still important to know that the variable has been assigned in the
	// current scope because it could shadow a variable with the same name from an
	// outer scope.
	assigned = make(map[*build.Ident]bool)
	used = make(map[*build.Ident]bool)

	// Local scopes for comprehensions
	scopes := make(map[build.Expr]map[string]bool)

	// Nodes that are excluded from traversal
	blockedNodes := make(map[build.Expr]bool)

	build.WalkInterruptable(stmt, func(node build.Expr, stack []build.Expr) (err error) {
		// Check if the current node has been blocked
		if _, ok := blockedNodes[node]; ok {
			return &build.StopTraversalError{}
		}

		switch expr := node.(type) {
		case *build.AssignExpr:
			// If it's a top-level assign expression, extract LValues from LHS
			// Otherwise just ignore the LHS, it must be a keyword argument of a
			// function call.
			if node != stmt {
				// The assignment expression is not the statement itself but its child,
				// that means it's a keyword argument for a function call. Its LHS
				// should be ignored.
				blockedNodes[expr.LHS] = true
				return
			}

			hasUnusedComment := edit.ContainsComments(expr, "@unused")

			// LHS may contain both variables that are being used and variables that
			// are being assigned to, e.g. in the following example:
			//     x[i][f(name='foo')], y = 1, 2
			// `x`, `i`, `f` are used, `y` is assigned, `name` should be ignored.
			// Further traversal will ignore `name` but won't know that it's in an LHS
			// of an assign expression, so it'll erroneously collect `y` as used.
			// After the traversal it'll need to be removed from `used`.

			// If some (but not all) variables assigned to by the statements are
			// prefixed with an underscore, suppress the warning on them (i.e. allow
			// them to be unused). That's a common use case for partial unpacking of
			// tuples:
			//
			//     foo, _bar = my_function()  # only `foo` is needed
			//
			// However if all variables are underscored and unused, they should still
			// be reported:
			//
			//     _foo, _bar = my_function()  # LHS can just be removed

			lValues := bzlenv.CollectLValues(expr.LHS)
			allLValuesUnderscored := true
			for _, lValue := range lValues {
				if !strings.HasPrefix(lValue.Name, "_") {
					allLValuesUnderscored = false
					break
				}
			}

			for _, lValue := range bzlenv.CollectLValues(expr.LHS) {
				assigned[lValue] = hasUnusedComment ||
					(!allLValuesUnderscored && strings.HasPrefix(lValue.Name, "_"))
			}

		case *build.CallExpr:
			if dot, ok := expr.X.(*build.DotExpr); ok && mutatingMethodsReturningNone[dot.Name] {
				if receiver, ok := dot.X.(*build.Ident); ok {
					if !mutatingReceiversCountAsUsed[receiver.Name] {
						blockedNodes[dot.X] = true
					}
				}
			}

		case *build.ForStmt:
			// Like AssignExpr, ForStmt too has an analogue of LHS and RHS.
			// Unlike AssignExpr, in this function they may appear only in the root of
			// traversal and shouldn't be traversed inside (the caller of
			// `extractIdentsFromStmt` should be responsible for checking all
			// statements including those that are inside for-loops.

			// It's common to not use all variables (or even not use any of them)
			// after unpacking tuples, suppress the warning an all of them that are
			// prefixed with an underscore:
			//
			//     for _, (_b, c) in iterable:
			//         print(c)
			for _, lValue := range bzlenv.CollectLValues(expr.Vars) {
				assigned[lValue] = strings.HasPrefix(lValue.Name, "_")
			}

			// Don't traverse inside the inner statements (but still traverse into
			// the expressions in `Vars` an `X`).
			for _, substmt := range expr.Body {
				blockedNodes[substmt] = true
			}

		case *build.IfStmt:
			// Nothing special, just don't traverse the inner statements (like with
			// ForStmt nodes).
			for _, substmt := range expr.True {
				blockedNodes[substmt] = true
			}
			for _, substmt := range expr.False {
				blockedNodes[substmt] = true
			}

		case *build.Comprehension:
			// Comprehensions introduce their own visibility scope that shadows the
			// outer scope. Iterators that are defined and used there don't affect
			// the usage of variables with the same name outside the comprehension
			// scope.
			scope := make(map[string]bool)
			for _, clause := range expr.Clauses {
				forClause, ok := clause.(*build.ForClause)
				if !ok {
					// if-clause
					continue
				}
				for _, lValue := range bzlenv.CollectLValues(forClause.Vars) {
					scope[lValue.Name] = true
				}
			}
			scopes[expr] = scope

		case *build.LambdaExpr:
			// Similar to Comprehension nodes
			scope := make(map[string]bool)
			for _, param := range expr.Params {
				for _, lValue := range bzlenv.CollectLValues(param) {
					scope[lValue.Name] = true
				}
			}
			scopes[expr] = scope

		case *build.Ident:
			// If the identifier is defined in an intermediate scope, ignore it.
			for _, node := range stack {
				if scope, ok := scopes[node]; ok {
					if _, ok := scope[expr.Name]; ok {
						return
					}
				}
			}
			used[expr] = true

		default:
			// Do nothing, just traverse further
		}
		return
	})

	for ident := range assigned {
		// If the same ident (not the same variable but the same AST node) is
		// registered as both "assigned" and "used", it means it was in fact just
		// assigned, remove it from "used".
		delete(used, ident)
	}
	return assigned, used
}

// unusedVariableCheck checks for unused variables inside a given node `stmt` (either *build.File or
// *build.DefStmt) and variables that are used in the current scope or subscopes,
// but not defined here.
func unusedVariableCheck(f *build.File, root build.Expr) (map[string]bool, []*LinterFinding) {
	findings := []*LinterFinding{}

	// Symbols that are defined in the current scope
	definedSymbols := make(map[string]*build.Ident)

	// Functions that are defined in the current scope
	definedFunctions := make(map[string]*build.DefStmt)

	// Symbols that are used in the current and inner scopes
	usedSymbols := make(map[string]bool)

	// Symbols for which the warning should be suppressed
	suppressedWarnings := make(map[string]bool)

	// Symbols that can represent mutable values received from outside the current
	// scope. Mutating method calls on these symbols count as usage unless the
	// symbol has been reassigned.
	mutatingReceiversCountAsUsed := make(map[string]bool)

	// Symbols from outer scopes that are used in the current scope
	usedSymbolsFromOuterScope := make(map[string]bool)

	build.WalkStatements(root, func(expr build.Expr, stack []build.Expr) (err error) {
		switch expr := expr.(type) {
		case *build.File:
			// File nodes don't have anything meaningful, just traverse its subnodes.

		case *build.DefStmt:
			if len(stack) > 0 {
				// Nested def statement. Don't traverse inside, instead call
				// unusedVariableCheck recursively to handle nested scopes properly.

				// The function name is defined in the current scope
				if _, ok := definedFunctions[expr.Name]; !ok {
					definedFunctions[expr.Name] = expr
				}
				if edit.ContainsComments(expr, "@unused") {
					suppressedWarnings[expr.Name] = true
				}

				usedSymbolsInFunction, findingsInFunction := unusedVariableCheck(f, expr)
				findings = append(findings, findingsInFunction...)
				for symbol := range usedSymbolsInFunction {
					usedSymbols[symbol] = true
				}
				return &build.StopTraversalError{}
			}

			// The function is a root for the current scope.
			// Collect its parameters as defined in the current scope.
			for _, param := range expr.Params {
				// Function parameters are defined in the current scope.
				if ident, _ := build.GetParamIdent(param); ident != nil {
					definedSymbols[ident.Name] = ident
					mutatingReceiversCountAsUsed[ident.Name] = true
					if ident.Name == "name" || strings.HasPrefix(ident.Name, "_") || edit.ContainsComments(param, "@unused") {
						// Don't warn about function arguments if they start with "_"
						// or explicitly marked with @unused.
						// Also don't warn about unused "name" arguments, it could be a
						// macro where such argument is encouraged (by `unnamed-macro`)
						// even if not used.
						suppressedWarnings[ident.Name] = true
					}
				}
				// The default variables for the parameters are defined in the outer
				// scope but used here.
				assign, ok := param.(*build.AssignExpr)
				if !ok {
					continue
				}

				// RHS is not a statement, but similar traversal rules should be applied
				// to it. E.g. it may have a comprehension node with its inner scope or
				// a function call with a keyword parameter.
				_, used := extractIdentsFromStmt(assign.RHS, nil)
				for ident := range used {
					// RHS idents in the def statement contains direct references to the outer scope.
					usedSymbolsFromOuterScope[ident.Name] = true
				}
			}

		case *build.LoadStmt:
			// LoadStmt nodes store the loaded symbols as idents, even though in the
			// source code they are strings. These idents may confuse the check,
			// they also shouldn't affect the warnings at all, unused loads are taken
			// care of by another check. It's safe to just ignore them here.
			return

		default:
			assigned, used := extractIdentsFromStmt(expr, mutatingReceiversCountAsUsed)
			newlyDefinedSymbols := make(map[string]*build.Ident)

			for symbol := range used {
				usedSymbols[symbol.Name] = true
			}
			for symbol, isSuppressed := range assigned {
				if _, ok := definedSymbols[symbol.Name]; !ok {
					definedSymbols[symbol.Name] = symbol
					newlyDefinedSymbols[symbol.Name] = symbol
					if isSuppressed {
						suppressedWarnings[symbol.Name] = true
					}
				}
				delete(mutatingReceiversCountAsUsed, symbol.Name)
			}
			if forStmt, ok := expr.(*build.ForStmt); ok {
				for _, lValue := range bzlenv.CollectLValues(forStmt.Vars) {
					if newlyDefinedSymbols[lValue.Name] == lValue {
						mutatingReceiversCountAsUsed[lValue.Name] = true
					}
				}
			}
		}
		return
	})

	// If a variable is defined in an outer scope but also in this scope, it
	// shadows the outer variable. If it's used in the current scope, it doesn't
	// make the variable with the same name from an outer scope also used.
	// Collect variables that are used in the current or inner scopes but are not
	// defined in the current scope.
	for symbol := range usedSymbols {
		if _, ok := definedSymbols[symbol]; ok {
			continue
		}
		if _, ok := definedFunctions[symbol]; ok {
			continue
		}
		usedSymbolsFromOuterScope[symbol] = true
	}

	// Top-level variables defined in .bzl or generic Starlark files
	// can be imported from elsewhere, even if not used in the current file.
	// Do not warn on exportable variables.
	ignoreTopLevel := (f.Type == build.TypeBzl || f.Type == build.TypeDefault) && root == f

	for name, ident := range definedSymbols {
		if _, ok := usedSymbols[name]; ok {
			// The variable is used either in this scope or in a nested scope
			continue
		}
		if _, ok := suppressedWarnings[name]; ok {
			// The variable is explicitly marked with @unused, ignore
			continue
		}
		if ignoreTopLevel && !strings.HasPrefix(name, "_") {
			continue
		}
		findings = append(findings,
			makeLinterFinding(ident, fmt.Sprintf(`Variable %q is unused. Please remove it.`, ident.Name)))
	}

	for name, def := range definedFunctions {
		if _, ok := usedSymbols[name]; ok {
			// The function is used either in this scope or in a nested scope
			continue
		}
		if ignoreTopLevel && !strings.HasPrefix(name, "_") {
			continue
		}
		if _, ok := suppressedWarnings[name]; ok {
			// The function is explicitly marked with @unused, ignore
			continue
		}
		findings = append(findings,
			makeLinterFinding(def, fmt.Sprintf(`Function %q is unused. Please remove it.`, def.Name)))
	}

	return usedSymbolsFromOuterScope, findings
}

func unusedVariableWarning(f *build.File) []*LinterFinding {
	_, findings := unusedVariableCheck(f, f)
	return findings
}

func redefinedVariableWarning(f *build.File) []*LinterFinding {
	findings := []*LinterFinding{}
	definedSymbols := make(map[string]bool)

	types := DetectTypes(f)
	for _, s := range f.Stmt {
		// look for all assignments in the scope
		as, ok := s.(*build.AssignExpr)
		if !ok {
			continue
		}
		left, ok := as.LHS.(*build.Ident)
		if !ok {
			continue
		}
		if !definedSymbols[left.Name] {
			definedSymbols[left.Name] = true
			continue
		}

		if as.Op == "+=" && (types[as.LHS] == List || types[as.RHS] == List) {
			// Not a reassignment, just appending to a list
			continue
		}

		findings = append(findings,
			makeLinterFinding(as.LHS, fmt.Sprintf(`Variable %q has already been defined. 
Redefining a global value is discouraged and will be forbidden in the future.
Consider using a new variable instead.`, left.Name)))
	}
	return findings
}

func unusedLoadWarning(f *build.File) []*LinterFinding {
	findings := []*LinterFinding{}
	loaded := make(map[string]struct {
		label, from string
		line        int
	})

	symbols := edit.UsedSymbols(f)
	types := edit.UsedTypes(f)
	for stmtIndex := 0; stmtIndex < len(f.Stmt); stmtIndex++ {
		originalLoad, ok := f.Stmt[stmtIndex].(*build.LoadStmt)
		if !ok {
			continue
		}

		// Findings related to the current load statement
		loadFindings := []*LinterFinding{}

		// Copy the `load` object to provide a replacement if needed
		load := *originalLoad
		load.From = append([]*build.Ident{}, load.From...)
		load.To = append([]*build.Ident{}, load.To...)

		for i := 0; i < len(load.To); i++ {
			from := load.From[i]
			to := load.To[i]
			// Check if the symbol was already loaded
			origin, alreadyLoaded := loaded[to.Name]
			start, _ := from.Span()
			loaded[to.Name] = struct {
				label, from string
				line        int
			}{load.Module.Token, from.Name, start.Line}

			if alreadyLoaded {
				// The same symbol has already been loaded earlier
				if origin.label == load.Module.Token && origin.from == from.Name {
					// Only fix if it's loaded from the same label and variable
					load.To = append(load.To[:i], load.To[i+1:]...)
					load.From = append(load.From[:i], load.From[i+1:]...)
					i--
					loadFindings = append(loadFindings, makeLinterFinding(to,
						fmt.Sprintf("Symbol %q has already been loaded on line %d. Please remove it.", to.Name, origin.line)))
					continue
				}

				loadFindings = append(loadFindings, makeLinterFinding(to,
					fmt.Sprintf("A different symbol %q has already been loaded on line %d. Please use a different local name.", to.Name, origin.line)))
				continue
			}
			_, ok := symbols[to.Name]
			if !ok {
				// Fallback to verify if the symbol is used as a type.
				_, ok = types[to.Name]
			}
			if !ok && !edit.ContainsComments(originalLoad, "@unused") && !edit.ContainsComments(to, "@unused") && !edit.ContainsComments(from, "@unused") {
				// The loaded symbol is not used and is not protected by a special "@unused" comment
				load.To = append(load.To[:i], load.To[i+1:]...)
				load.From = append(load.From[:i], load.From[i+1:]...)
				i--

				loadFindings = append(loadFindings, makeLinterFinding(to,
					fmt.Sprintf("Loaded symbol %q is unused. Please remove it.\nTo disable the warning, add '@unused' in a comment.", to.Name)))
				if f.Type == build.TypeDefault || f.Type == build.TypeBzl {
					loadFindings[len(loadFindings)-1].Message += fmt.Sprintf(`
If you want to re-export a symbol, use the following pattern:

    load(..., _%s = %q, ...)
    %s = _%s
`, to.Name, from.Name, to.Name, to.Name)
				}
			}
		}

		if len(loadFindings) == 0 {
			// No problems with the current load statement
			continue
		}

		build.SortLoadArgs(&load)
		var newStmt build.Expr = &load
		if len(load.To) == 0 {
			// If there are no loaded symbols left remove the entire load statement
			newStmt = nil
		}
		replacement := LinterReplacement{&f.Stmt[stmtIndex], newStmt}

		// Individual replacements can't be combined together: assume we need to remove both loaded
		// symbols from
		//
		//     load(":foo.bzl", "a", "b")
		//
		// Individual replacements are just to remove each of the symbols, but if these replacements
		// are applied together, the result will be incorrect and a syntax error in Bazel:
		//
		//     load(":foo.bzl")
		//
		// A workaround is to attach the full replacement to the first finding.
		loadFindings[0].Replacement = []LinterReplacement{replacement}
		findings = append(findings, loadFindings...)
	}
	return findings
}

// collectLocalVariables traverses statements (e.g. of a function definition) and returns a list
// of idents for variables defined anywhere inside the function.
func collectLocalVariables(stmts []build.Expr) []*build.Ident {
	variables := []*build.Ident{}

	for _, stmt := range stmts {
		switch stmt := stmt.(type) {
		case *build.DefStmt:
			// Don't traverse nested functions
		case *build.ForStmt:
			variables = append(variables, bzlenv.CollectLValues(stmt.Vars)...)
			variables = append(variables, collectLocalVariables(stmt.Body)...)
		case *build.IfStmt:
			variables = append(variables, collectLocalVariables(stmt.True)...)
			variables = append(variables, collectLocalVariables(stmt.False)...)
		case *build.AssignExpr:
			variables = append(variables, bzlenv.CollectLValues(stmt.LHS)...)
		}
	}
	return variables
}

// findUninitializedVariables takes a list of statements (e.g. body of a block statement)
// and a map of previously initialized statements, and calls `callback` on all idents that are not
// initialized. An ident is considered initialized if it's initialized by every possible execution
// path (before or by `stmts`).
// Returns a boolean indicating whether the current list of statements is guaranteed to be
// terminated explicitly (by return or fail() statements) and a map of variables that are guaranteed
// to be defined by `stmts`.
func findUninitializedVariables(stmts []build.Expr, previouslyInitialized map[string]bool, callback func(*build.Ident)) (bool, map[string]bool) {
	// Variables that are guaranteed to be initialized
	locallyInitialized := make(map[string]bool) // in the local block of `stmts`
	initialized := make(map[string]bool)        // anywhere before the current line
	for key := range previouslyInitialized {
		initialized[key] = true
	}

	// findUninitializedIdents traverses an expression (simple statement or a part of it), and calls
	// `callback` on every *build.Ident that's not mentioned in the map of initialized variables
	findUninitializedIdents := func(expr build.Expr, callback func(ident *build.Ident)) {
		// Collect lValues, they shouldn't be taken into account
		// For example, if the expression is `a = foo(b = c)`, only `c` can be an uninitialized variable here.
		lValues := make(map[*build.Ident]bool)
		build.WalkInterruptable(expr, func(expr build.Expr, stack []build.Expr) (err error) {
			switch expr := expr.(type) {
			case *build.DefStmt:
				// Function arguments can't be uninitialized, even if they share the same
				// name with a variable that's not initialized for some execution path
				// in an outer scope.
				for _, param := range expr.Params {
					if ident, _ := build.GetParamIdent(param); ident != nil {
						lValues[ident] = true
					}
				}
				// Don't traverse into nested def statements
				return &build.StopTraversalError{}
			case *build.AssignExpr:
				for _, ident := range bzlenv.CollectLValues(expr.LHS) {
					lValues[ident] = true
				}
			}
			return
		})

		// Check if the ident is really not initialized and call the callback on it.
		callbackIfNeeded := func(ident *build.Ident) {
			if !initialized[ident.Name] && !lValues[ident] {
				callback(ident)
			}
		}

		walkBlockList := map[build.Expr]bool{}
		build.WalkInterruptable(expr, func(expr build.Expr, stack []build.Expr) (err error) {
			if walkBlockList[expr] {
				return &build.StopTraversalError{}
			}
			switch expr := expr.(type) {
			case *build.DefStmt:
				// The header of the DefStmt may contain uninitialized variables (e.g.
				// default values of parameters) and should be traversed.
				// Its body shouldn't be traversed because it has another scope and will
				// be analyzed by another call of `findUninitializedVariables`.
				for _, stmt := range expr.Body {
					walkBlockList[stmt] = true
				}
			case *build.Comprehension, *build.LambdaExpr:
				// Comprehension and Lambda nodes are special, they have their own scope
				// with variables that are only defined inside.
				// Instead of traversing inside stop the traversal and call a special
				// function to retrieve idents from the outer scope that are used inside
				// the comprehension.

				_, used := extractIdentsFromStmt(expr, nil)
				for ident := range used {
					callbackIfNeeded(ident)
				}

				return &build.StopTraversalError{}
			case *build.Ident:
				callbackIfNeeded(expr)
			default:
				// Just traverse further
			}
			return
		})
	}

	for _, stmt := range stmts {
		newlyDefinedVariables := make(map[string]bool)
		switch stmt := stmt.(type) {
		case *build.DefStmt:
			// Don't traverse nested functions
		case *build.CallExpr:
			if _, ok := isFunctionCall(stmt, "fail"); ok {
				return true, locallyInitialized
			}
		case *build.ReturnStmt:
			findUninitializedIdents(stmt, callback)
			return true, locallyInitialized
		case *build.BranchStmt:
			if stmt.Token == "break" || stmt.Token == "continue" {
				return true, locallyInitialized
			}
		case *build.ForStmt:
			// Although loop variables are defined as local variables, buildifier doesn't know whether
			// the collection will be empty or not.

			// Traverse but ignore the result. Even if something is defined inside a for-loop, the loop
			// may be empty and the variable initialization may not happen.
			findUninitializedIdents(stmt.X, callback)

			// The loop can access the variables defined above, and also the for-loop variables.
			initializedInLoop := make(map[string]bool)
			for name := range initialized {
				initializedInLoop[name] = true
			}
			for _, ident := range bzlenv.CollectLValues(stmt.Vars) {
				initializedInLoop[ident.Name] = true
			}
			findUninitializedVariables(stmt.Body, initializedInLoop, callback)
			continue
		case *build.IfStmt:
			findUninitializedIdents(stmt.Cond, callback)
			// Check the variables defined in the if- and else-clauses.
			terminatedTrue, definedInTrue := findUninitializedVariables(stmt.True, initialized, callback)
			terminatedFalse, definedInFalse := findUninitializedVariables(stmt.False, initialized, callback)
			if terminatedTrue && terminatedFalse {
				return true, locallyInitialized
			} else if terminatedTrue {
				// Only take definedInFalse into account
				for key := range definedInFalse {
					locallyInitialized[key] = true
					initialized[key] = true
				}
			} else if terminatedFalse {
				// Only take definedInTrue into account
				for key := range definedInTrue {
					locallyInitialized[key] = true
					initialized[key] = true
				}
			} else {
				// If a variable is defined in both if- and else-clauses, it's considered as defined
				for key := range definedInTrue {
					if definedInFalse[key] {
						locallyInitialized[key] = true
						initialized[key] = true
					}
				}
			}
			continue
		case *build.AssignExpr:
			// Assignment expression. Collect all definitions from the lhs
			for _, ident := range bzlenv.CollectLValues(stmt.LHS) {
				newlyDefinedVariables[ident.Name] = true
			}
		}
		findUninitializedIdents(stmt, callback)
		for name := range newlyDefinedVariables {
			locallyInitialized[name] = true
			initialized[name] = true
		}
	}
	return false, locallyInitialized
}

// uninitializedVariableWarning warns about usages of values that may not have been initialized.
func uninitializedVariableWarning(f *build.File) []*LinterFinding {
	findings := []*LinterFinding{}
	build.WalkStatements(f, func(expr build.Expr, stack []build.Expr) (err error) {
		def, ok := expr.(*build.DefStmt)
		if !ok {
			return
		}

		// Get all variables defined in the function body.
		// If a variable is not defined there, it can be builtin, global, or loaded.
		localVars := make(map[string]bool)
		for _, ident := range collectLocalVariables(def.Body) {
			localVars[ident.Name] = true
		}

		// Function parameters are guaranteed to be defined everywhere in the function, even if they
		// are redefined inside the function body. They shouldn't be taken into consideration.
		for _, param := range def.Params {
			if name, _ := build.GetParamName(param); name != "" {
				delete(localVars, name)
			}
		}

		// Search for all potentially initialized variables in the function body
		findUninitializedVariables(def.Body, make(map[string]bool), func(ident *build.Ident) {
			// Check that the found ident represents a local variable
			if localVars[ident.Name] {
				findings = append(findings,
					makeLinterFinding(ident, fmt.Sprintf(`Variable "%s" may not have been initialized.`, ident.Name)))
			}
		})
		return
	})
	return findings
}
