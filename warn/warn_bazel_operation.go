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

// Warnings about deprecated Bazel-related operations

package warn

import (
	"fmt"

	"github.com/bazelbuild/buildtools/build"
)

func depsetUnionWarning(f *build.File) []*LinterFinding {
	var findings []*LinterFinding
	addWarning := func(expr build.Expr) {
		findings = append(findings,
			makeLinterFinding(expr, `Depsets should be joined using the "depset()" constructor.`))
	}

	types := DetectTypes(f)
	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		switch expr := expr.(type) {
		case *build.BinaryExpr:
			// `depset1 + depset2` or `depset1 | depset2`
			if types[expr.X] != Depset && types[expr.Y] != Depset {
				return
			}
			switch expr.Op {
			case "+", "|":
				addWarning(expr)
			}
		case *build.AssignExpr:
			// `depset1 += depset2` or `depset1 |= depset2`
			if types[expr.LHS] != Depset && types[expr.RHS] != Depset {
				return
			}
			switch expr.Op {
			case "+=", "|=":
				addWarning(expr)
			}
		case *build.CallExpr:
			// `depset1.union(depset2)`
			if len(expr.List) == 0 {
				return
			}
			dot, ok := expr.X.(*build.DotExpr)
			if !ok {
				return
			}
			if dot.Name != "union" {
				return
			}
			if types[dot.X] != Depset && types[expr.List[0]] != Depset {
				return
			}
			addWarning(expr)
		}
	})
	return findings
}

func depsetIterationWarning(f *build.File) []*LinterFinding {
	var findings []*LinterFinding

	addFinding := func(expr *build.Expr) {
		_, end := (*expr).Span()
		newNode := &build.CallExpr{
			X: &build.DotExpr{
				X:    *expr,
				Name: "to_list",
			},
			End: build.End{Pos: end},
		}
		findings = append(findings,
			makeLinterFinding(*expr, `Depset iteration is deprecated, use the "to_list()" method instead.`, LinterReplacement{expr, newNode}))
	}

	types := DetectTypes(f)
	build.WalkPointers(f, func(e *build.Expr, stack []build.Expr) {
		switch expr := (*e).(type) {
		case *build.ForStmt:
			if types[expr.X] != Depset {
				return
			}
			addFinding(&expr.X)
		case *build.ForClause:
			if types[expr.X] != Depset {
				return
			}
			addFinding(&expr.X)
		case *build.BinaryExpr:
			if expr.Op != "in" && expr.Op != "not in" {
				return
			}
			if types[expr.Y] != Depset {
				return
			}
			addFinding(&expr.Y)
		case *build.CallExpr:
			ident, ok := expr.X.(*build.Ident)
			if !ok {
				return
			}
			switch ident.Name {
			case "all", "any", "depset", "len", "sorted", "max", "min", "list", "tuple":
				if len(expr.List) != 1 {
					return
				}
				if types[expr.List[0]] != Depset {
					return
				}
				addFinding(&expr.List[0])
				if ident.Name == "list" {
					// `list(d.to_list())` can be simplified to just `d.to_list()`
					findings[len(findings)-1].Replacement[0].Old = e
				}
			case "zip":
				for i, arg := range expr.List {
					if types[arg] != Depset {
						continue
					}
					addFinding(&expr.List[i])
				}
			}
		}
		return
	})
	return findings
}

func overlyNestedDepsetWarning(f *build.File) []*LinterFinding {
	var findings []*LinterFinding
	build.WalkStatements(f, func(expr build.Expr, stack []build.Expr) (err error) {
		// Are we inside a for-loop?
		isForLoop := false
		for _, e := range stack {
			if _, ok := e.(*build.ForStmt); ok {
				isForLoop = true
				break
			}
		}
		if !isForLoop {
			return
		}

		// Search for assignment statements
		assign, ok := expr.(*build.AssignExpr)
		if !ok {
			return
		}
		// Is the LHS an ident?
		lhs, ok := assign.LHSIdent()
		if !ok {
			return
		}
		// Is the RHS a depset constructor?
		call, ok := assign.RHS.(*build.CallExpr)
		if !ok {
			return
		}
		if ident, ok := call.X.(*build.Ident); !ok || ident.Name != "depset" {
			return
		}
		_, _, param := getParam(call.List, "transitive")
		if param == nil {
			return
		}
		transitives, ok := param.RHS.(*build.ListExpr)
		if !ok {
			return
		}
		for _, transitive := range transitives.List {
			if ident, ok := transitive.(*build.Ident); ok && ident.Name == lhs.Name {
				findings = append(findings, makeLinterFinding(assign, fmt.Sprintf("Depset %q is potentially overly nested.", lhs.Name)))
				return
			}
		}
		return
	})
	return findings
}
