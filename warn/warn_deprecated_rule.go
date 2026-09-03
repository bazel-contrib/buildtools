/*
Copyright 2026 Google LLC

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

// Warnings for using deprecated repository rules

package warn

import (
	"fmt"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/labels"
)

// Checks if the rule definition in `stmt` is deprecated.
func isRuleDefinitionDeprecated(stmt build.Expr, loadedFile *build.File) bool {
	assign, ok := stmt.(*build.AssignExpr)
	if !ok {
		return false
	}

	rhsCall, ok := assign.RHS.(*build.CallExpr)
	if !ok {
		return false
	}

	rhsIdent, ok := rhsCall.X.(*build.Ident)
	if !ok {
		return false
	}

	if rhsIdent.Name != "rule" && rhsIdent.Name != "repository_rule" && rhsIdent.Name != "materializer_rule" {
		return false
	}

	for _, arg := range rhsCall.List {
		assignArg, ok := arg.(*build.AssignExpr)
		if !ok {
			continue
		}
		lhsArg, ok := assignArg.LHS.(*build.Ident)
		if !ok || lhsArg.Name != "doc" {
			continue
		}
		rhsStr, ok := assignArg.RHS.(*build.StringExpr)
		if !ok {
			continue
		}
		if strings.Contains(rhsStr.Value, "Deprecated:") {
			return true
		}
	}

	// Check implementation function
	var implIdent *build.Ident
	for _, arg := range rhsCall.List {
		assignArg, ok := arg.(*build.AssignExpr)
		if !ok {
			continue
		}
		lhsArg, ok := assignArg.LHS.(*build.Ident)
		if !ok || lhsArg.Name != "implementation" {
			continue
		}
		if ident, ok := assignArg.RHS.(*build.Ident); ok {
			implIdent = ident
			break
		}
	}

	if implIdent == nil {
		return false
	}

	for _, defStmt := range loadedFile.Stmt {
		def, ok := defStmt.(*build.DefStmt)
		if !ok || def.Name != implIdent.Name {
			continue
		}

		docstring, ok := getDocstring(def.Body)
		if !ok {
			continue
		}
		str, ok := (*docstring).(*build.StringExpr)
		if !ok {
			continue
		}
		if strings.Contains(str.Value, "Deprecated:") {
			return true
		}
	}

	return false
}

// Checks if each of the loadedSymbols is a deprecated rule.
func checkDeprecatedRule(stmt build.Expr, loadedSymbols map[string]*build.Ident, loadedFile *build.File) *LinterFinding {
	assign, ok := stmt.(*build.AssignExpr)
	if !ok {
		return nil
	}
	lhsIdent, ok := assign.LHS.(*build.Ident)
	if !ok {
		return nil
	}
	node, ok := loadedSymbols[lhsIdent.Name]
	if !ok {
		return nil
	}

	if isRuleDefinitionDeprecated(stmt, loadedFile) {
		return makeLinterFinding(node, fmt.Sprintf("The rule %q defined in %q is deprecated.", lhsIdent.Name, loadedFile.CanonicalPath()))
	}

	return nil
}

// Warns if a BUILD or .bzl file refers to a deprecated rule in a file loaded via a "load".
func checkDeprecatedRuleInLoad(f *build.File, fileReader *FileReader) []*LinterFinding {
	findings := []*LinterFinding{}
	for _, stmt := range f.Stmt {
		if load, ok := stmt.(*build.LoadStmt); ok {
			label := labels.ParseRelative(load.Module.Value, f.Pkg)
			if label.Repository != "" || label.Target == "" {
				continue
			}
			loadedFile := fileReader.GetFile(label.Package, label.Target)
			if loadedFile == nil {
				continue
			}

			loadedSymbols := make(map[string]*build.Ident)
			for _, from := range load.From {
				loadedSymbols[from.Name] = from
			}

			for _, stmt := range loadedFile.Stmt {
				if finding := checkDeprecatedRule(stmt, loadedSymbols, loadedFile); finding != nil {
					findings = append(findings, finding)
				}
			}
			continue
		}
	}
	return findings
}

// Warns if a MODULE.bazel refers to a deprecated rule in a file loaded via a "use_repo_rule".
func checkDeprecatedRuleInModule(f *build.File, fileReader *FileReader) []*LinterFinding {
	findings := []*LinterFinding{}

	type ruleInfo struct {
		name string
		stmt build.Expr
	}
	fileToRules := make(map[string][]ruleInfo)
	fileMap := make(map[string]*build.File)

	for _, stmt := range f.Stmt {
		var call *build.CallExpr
		var ok bool
		if call, ok = stmt.(*build.CallExpr); !ok {
			assign, ok := stmt.(*build.AssignExpr)
			if ok {
				call, ok = assign.RHS.(*build.CallExpr)
			}
		}
		if call == nil {
			continue
		}
		ident, ok := call.X.(*build.Ident)
		if !ok || ident.Name != "use_repo_rule" {
			continue
		}

		args := call.List
		if len(args) < 2 {
			continue
		}
		pathArg, ok := args[0].(*build.StringExpr)
		if !ok {
			continue
		}
		nameArg, ok := args[1].(*build.StringExpr)
		if !ok {
			continue
		}

		label := labels.ParseRelative(pathArg.Value, f.Pkg)
		if label.Repository != "" || label.Target == "" {
			continue
		}
		loadedFile := fileReader.GetFile(label.Package, label.Target)
		if loadedFile == nil {
			continue
		}

		path := loadedFile.CanonicalPath()
		fileToRules[path] = append(fileToRules[path], ruleInfo{name: nameArg.Value, stmt: stmt})
		fileMap[path] = loadedFile
	}

	for path, rules := range fileToRules {
		loadedFile := fileMap[path]
		rulesToFind := make(map[string]build.Expr) // name -> use_repo_rule stmt
		for _, r := range rules {
			rulesToFind[r.name] = r.stmt
		}

		for _, loadedStmt := range loadedFile.Stmt {
			assign, ok := loadedStmt.(*build.AssignExpr)
			if !ok {
				continue
			}
			lhsIdent, ok := assign.LHS.(*build.Ident)
			if !ok {
				continue
			}
			stmt, ok := rulesToFind[lhsIdent.Name]
			if !ok {
				continue
			}

			if isRuleDefinitionDeprecated(loadedStmt, loadedFile) {
				findings = append(findings, makeLinterFinding(stmt, fmt.Sprintf(
					"The rule %q defined in %q is deprecated.",
					lhsIdent.Name, path)))
			}
		}
	}

	return findings
}

// Warns if a deprecated rule is used.
func deprecatedRuleWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if fileReader == nil {
		return nil
	}

	switch f.Type {
	case build.TypeModule:
		return checkDeprecatedRuleInModule(f, fileReader)
	case build.TypeBuild, build.TypeBzl:
		return checkDeprecatedRuleInLoad(f, fileReader)
	}

	return nil
}