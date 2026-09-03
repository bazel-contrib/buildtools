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

// Warnings for using deprecated functions

package warn

import (
	"fmt"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/labels"
)

func checkDeprecatedFunction(stmt build.Expr, loadedSymbols *map[string]*build.Ident, fullLabel string) *LinterFinding {
	def, ok := stmt.(*build.DefStmt)
	if !ok {
		return nil
	}
	node, ok := (*loadedSymbols)[def.Name]
	if !ok {
		return nil
	}
	if !isFunctionDeprecated(def) {
		return nil
	}

	return makeLinterFinding(node, fmt.Sprintf("The function %q defined in %q is deprecated.", def.Name, fullLabel))
}

func deprecatedFunctionWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if fileReader == nil {
		return nil
	}

	findings := []*LinterFinding{}
	for _, stmt := range f.Stmt {
		load, ok := stmt.(*build.LoadStmt)
		if !ok {
			continue
		}
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
			if finding := checkDeprecatedFunction(stmt, &loadedSymbols, loadedFile.CanonicalPath()); finding != nil {
				findings = append(findings, finding)
			}
		}

	}
	return findings
}

func isFunctionDeprecated(def *build.DefStmt) bool {
	docstring, ok := getDocstring(def.Body)
	if !ok {
		return false
	}
	str, ok := (*docstring).(*build.StringExpr)
	if !ok {
		return false
	}
	docstringInfo := parseFunctionDocstring(str)
	return docstringInfo.deprecated
}
