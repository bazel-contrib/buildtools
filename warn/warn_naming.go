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

package warn

import (
	"fmt"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/bzlenv"
)

var ambiguousNames = map[string]bool{
	"I": true,
	"l": true,
	"O": true,
}

// ambiguousNameCheck checks for the names of idents and functions
func ambiguousNameCheck(expr build.Expr, name string, findings []*LinterFinding) []*LinterFinding {
	if ambiguousNames[name] {
		findings = append(findings,
			makeLinterFinding(expr, `Never use 'l', 'I', or 'O' as names (they're too easily confused with 'I', 'l', or '0').`))
	}
	return findings
}

func confusingNameWarning(f *build.File) []*LinterFinding {
	var findings []*LinterFinding
	// check for global variable names
	for _, ident := range collectLocalVariables(f.Stmt) {
		findings = ambiguousNameCheck(ident, ident.Name, findings)
	}

	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		switch expr := expr.(type) {
		case *build.DefStmt:
			findings = ambiguousNameCheck(expr, expr.Name, findings)
			if expr.TypeParams != nil {
				for _, ident := range bzlenv.CollectLValues(expr.TypeParams) {
					findings = ambiguousNameCheck(ident, ident.Name, findings)
				}
			}
			for _, param := range expr.Params {
				name, _ := build.GetParamName(param)
				findings = ambiguousNameCheck(param, name, findings)
			}
			for _, ident := range collectLocalVariables(expr.Body) {
				findings = ambiguousNameCheck(ident, ident.Name, findings)
			}
		case *build.Comprehension:
			for _, clause := range expr.Clauses {
				forClause, ok := clause.(*build.ForClause)
				if !ok {
					continue
				}
				for _, ident := range bzlenv.CollectLValues(forClause.Vars) {
					findings = ambiguousNameCheck(ident, ident.Name, findings)
				}
			}
		case *build.TypeAliasStmt:
			findings = ambiguousNameCheck(expr, expr.GetIdent().Name, findings)
			if expr.TypeParams != nil {
				for _, ident := range bzlenv.CollectLValues(expr.TypeParams) {
					findings = ambiguousNameCheck(ident, ident.Name, findings)
				}
			}
		}
	})

	return findings
}

func isUpperCamelCase(name string) bool {
	if strings.HasPrefix(name, "_") {
		// Private providers are allowed
		name = name[1:]
	}
	return !strings.ContainsRune(name, '_') && name == strings.Title(name)
}

func isLowerSnakeCase(name string) bool {
	return name == strings.ToLower(name)
}

func isUpperSnakeCase(name string) bool {
	return name == strings.ToUpper(name)
}

func nameConventionsWarning(f *build.File) []*LinterFinding {
	var findings []*LinterFinding

	build.WalkStatements(f, func(stmt build.Expr, stack []build.Expr) (err error) {
		switch stmt := stmt.(type) {
		case *build.AssignExpr:
			// looking for provider declaration statements: `xxx = provider()`
			// note that the code won't trigger on complex assignments, such as `x, y = foo, provider()`
			for _, ident := range bzlenv.CollectLValues(stmt.LHS) {
				findings = appendVariableNameConventionsWarning(ident, findings)
			}
		case *build.TypedIdent:
			ident := stmt.GetIdent()
			findings = appendVariableNameConventionsWarning(ident, findings)
		case *build.TypeAliasStmt:
			ident := stmt.GetIdent()
			findings = appendTypeNameConventionsWarning(ident, findings)
		}
		return
	})

	return findings
}

func appendVariableNameConventionsWarning(ident *build.Ident, findings []*LinterFinding) []*LinterFinding {
	if isLowerSnakeCase(ident.Name) || isUpperSnakeCase(ident.Name) {
		return findings
	}
	if isUpperCamelCase(ident.Name) && strings.HasSuffix(ident.Name, "Info") {
		return findings
	}
	return append(findings,
		makeLinterFinding(ident,
			fmt.Sprintf(`Variable name "%s" should be lower_snake_case (for variables), UPPER_SNAKE_CASE (for constants), or UpperCamelCase ending with 'Info' (for providers).`, ident.Name)))
}

func appendTypeNameConventionsWarning(ident *build.Ident, findings []*LinterFinding) []*LinterFinding {
	if isLowerSnakeCase(ident.Name) || isUpperCamelCase(ident.Name) {
		// Built-in types are either lower_snake (str, dict, list) or UpperCamel (Callable, Sequence).
		// User-defined types and type params should follow the same pattern.
		// Note that providers (expected to be UpperCamelCase) are also types.
		return findings
	}
	return append(findings,
		makeLinterFinding(ident,
			fmt.Sprintf(`Type name "%s" should be lower_snake_case or UpperCamelCase (possibly prefixed with an underscore).`, ident.Name)))
}
