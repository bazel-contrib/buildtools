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

// Warnings for using deprecated module extensions and tag classes

package warn

import (
	"fmt"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/labels"
)

type extUsage struct {
	alias string      // e.g. "ext" in `ext = use_extension(...)`
	name  string      // e.g. "non_root"
	file  *build.File // The file defining the extension
	stmt  build.Expr  // The `use_extension` statement
}

// getUsedExtensions scans a MODULE.bazel file for `use_extension` calls
// and resolves them to their defining files.
func getUsedExtensions(f *build.File, fileReader *FileReader) map[string]extUsage {
	aliasToExt := make(map[string]extUsage)

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
		if !ok || ident.Name != "use_extension" {
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

		assign, ok := stmt.(*build.AssignExpr)
		if !ok {
			continue
		}
		lhsIdent, ok := assign.LHS.(*build.Ident)
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

		aliasToExt[lhsIdent.Name] = extUsage{
			alias: lhsIdent.Name,
			name:  nameArg.Value,
			file:  loadedFile,
			stmt:  stmt,
		}
	}
	return aliasToExt
}

// Walks the arguments to a `module_extension` call to find the `tag_classes` dict.
func findTagClassesDict(call *build.CallExpr) *build.DictExpr {
	for _, arg := range call.List {
		assignArg, ok := arg.(*build.AssignExpr)
		if !ok {
			continue
		}
		lhsArg, ok := assignArg.LHS.(*build.Ident)
		if !ok || lhsArg.Name != "tag_classes" {
			continue
		}
		dict, ok := assignArg.RHS.(*build.DictExpr)
		if !ok {
			continue
		}
		return dict
	}
	return nil
}

// Walks the dict for the `tag_classes` to determine if the given tag class is deprecated.
func isTagClassDeprecatedInDict(dict *build.DictExpr, tagName string) bool {
	for _, kv := range dict.List {
		keyStr, ok := kv.Key.(*build.StringExpr)
		if !ok {
			continue
		}
		if keyStr.Value != tagName {
			continue
		}

		tcCall, ok := kv.Value.(*build.CallExpr)
		if !ok {
			continue
		}
		tcIdent, ok := tcCall.X.(*build.Ident)
		if !ok || tcIdent.Name != "tag_class" {
			continue
		}

		for _, arg := range tcCall.List {
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
	}
	return false
}

// Walks the statements in a '.bzl' file to find the definition for a module extension,
// and then determines if `tagName` refers to a deprecated tag class.
func isTagClassDeprecatedInStmt(stmt build.Expr, tagName string) bool {
	assign, ok := stmt.(*build.AssignExpr)
	if !ok {
		return false
	}

	rhsCall, ok := assign.RHS.(*build.CallExpr)
	if !ok {
		return false
	}
	rhsIdent, ok := rhsCall.X.(*build.Ident)
	if !ok || rhsIdent.Name != "module_extension" {
		return false
	}

	dict := findTagClassesDict(rhsCall)
	if dict == nil {
		return false
	}

	return isTagClassDeprecatedInDict(dict, tagName)
}

// Checks if the module extension defined in `stmt` is deprecated.
func isModuleExtensionDeprecatedInStmt(stmt build.Expr, loadedFile *build.File) bool {
	assign, ok := stmt.(*build.AssignExpr)
	if !ok {
		return false
	}

	rhsCall, ok := assign.RHS.(*build.CallExpr)
	if !ok {
		return false
	}
	rhsIdent, ok := rhsCall.X.(*build.Ident)
	if !ok || rhsIdent.Name != "module_extension" {
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

// Warns if a deprecated tag class from a module extension is used.
func deprecatedModuleExtTagWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if fileReader == nil {
		return nil
	}
	if f.Type != build.TypeModule {
		return nil
	}

	findings := []*LinterFinding{}

	aliasToExt := getUsedExtensions(f, fileReader)

	type tagUsage struct {
		tagCall *build.CallExpr
		extName string
	}
	fileToTags := make(map[string][]tagUsage)
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

		dot, ok := call.X.(*build.DotExpr)
		if !ok {
			continue
		}
		ident, ok := dot.X.(*build.Ident)
		if !ok {
			continue
		}

		ext, ok := aliasToExt[ident.Name]
		if !ok {
			continue
		}

		path := ext.file.CanonicalPath()
		fileToTags[path] = append(fileToTags[path], tagUsage{tagCall: call, extName: ext.name})
		fileMap[path] = ext.file
	}

	for path, tags := range fileToTags {
		loadedFile := fileMap[path]

		extToTags := make(map[string][]tagUsage)
		for _, t := range tags {
			extToTags[t.extName] = append(extToTags[t.extName], t)
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

			tagsForExt, ok := extToTags[lhsIdent.Name]
			if !ok {
				continue
			}

			for _, t := range tagsForExt {
				dot := t.tagCall.X.(*build.DotExpr)
				if isTagClassDeprecatedInStmt(loadedStmt, dot.Name) {
					findings = append(findings, makeLinterFinding(t.tagCall, fmt.Sprintf(
						"The tag class %q of module extension %q defined in %q is deprecated.",
						dot.Name, t.extName, path)))
				}
			}
		}
	}

	return findings
}

// Warns if a deprecated module extension is used.
func deprecatedModuleExtWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if fileReader == nil {
		return nil
	}
	if f.Type != build.TypeModule {
		return nil
	}

	findings := []*LinterFinding{}

	aliasToExt := getUsedExtensions(f, fileReader)

	type extInfo struct {
		name string
		stmt build.Expr
	}
	fileToExts := make(map[string][]extInfo)
	fileMap := make(map[string]*build.File)

	for _, ext := range aliasToExt {
		path := ext.file.CanonicalPath()
		fileToExts[path] = append(fileToExts[path], extInfo{name: ext.name, stmt: ext.stmt})
		fileMap[path] = ext.file
	}

	for path, exts := range fileToExts {
		loadedFile := fileMap[path]
		extsToFind := make(map[string]build.Expr) // name -> use_extension stmt
		for _, e := range exts {
			extsToFind[e.name] = e.stmt
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
			stmt, ok := extsToFind[lhsIdent.Name]
			if !ok {
				continue
			}

			if isModuleExtensionDeprecatedInStmt(loadedStmt, loadedFile) {
				findings = append(findings, makeLinterFinding(stmt, fmt.Sprintf(
					"The module extension %q defined in %q is deprecated.",
					lhsIdent.Name, loadedFile.CanonicalPath())))
			}
		}
	}
	return findings
}
