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
	"strings"

	"github.com/bazel-contrib/buildtools/v10/build"
	"github.com/bazel-contrib/buildtools/v10/labels"
)

// Internal constant that represents the native module
const nativeModule = "<native>"

// function represents a function identifier, which is a pair (module name, function name).
type function struct {
	pkg      string // package where the function is defined
	filename string // name of a .bzl file relative to the package
	name     string // original name of the function
}

func (f function) label() string {
	return f.pkg + ":" + f.filename
}

// funCall represents a call to another function. It contains information of the function itself as well as some
// information about the environment
type funCall struct {
	function
	nameAlias string // function name alias (it could be loaded with a different name or assigned to a new variable).
	line      int    // line on which the function is being called
}

// acceptsNameArgument checks whether a function can accept a named argument called "name",
// either directly or via **kwargs.
func acceptsNameArgument(def *build.DefStmt) bool {
	for _, param := range def.Params {
		if name, op := build.GetParamName(param); name == "name" || op == "**" {
			return true
		}
	}
	return false
}

// fileData represents information about rules and functions extracted from a file
type fileData struct {
	loadedSymbols map[string]function           // Symbols loaded from other files.
	rulesOrMacros map[string]bool               // all rules or macros defined in the file.
	functions     map[string]map[string]funCall // outer map: all functions defined in the file, inner map: all distinct function calls from the given function
	aliases       map[string]function           // all top-level aliases (e.g. `foo = bar`).
}

// resolvesExternal takes a local function definition and replaces it with an external one if it's been defined
// in another file and loaded
func resolveExternal(fn function, externalSymbols map[string]function) function {
	if external, ok := externalSymbols[fn.name]; ok {
		return external
	}
	return fn
}

// exprLine returns the start line of an expression
func exprLine(expr build.Expr) int {
	start, _ := expr.Span()
	return start.Line
}

// getFunCalls extracts information about functions that are being called from the given function
func getFunCalls(def *build.DefStmt, pkg, filename string, externalSymbols map[string]function) map[string]funCall {
	funCalls := make(map[string]funCall)
	build.Walk(def, func(expr build.Expr, stack []build.Expr) {
		call, ok := expr.(*build.CallExpr)
		if !ok {
			return
		}
		if ident, ok := call.X.(*build.Ident); ok {
			funCalls[ident.Name] = funCall{
				function:  resolveExternal(function{pkg, filename, ident.Name}, externalSymbols),
				nameAlias: ident.Name,
				line:      exprLine(call),
			}
			return
		}
		dot, ok := call.X.(*build.DotExpr)
		if !ok {
			return
		}
		if ident, ok := dot.X.(*build.Ident); !ok || ident.Name != "native" {
			return
		}
		name := "native." + dot.Name
		funCalls[name] = funCall{
			function: function{
				name:     dot.Name,
				filename: nativeModule,
			},
			nameAlias: name,
			line:      exprLine(dot),
		}
	})
	return funCalls
}

// analyzeFile extracts the information about rules and functions defined in the file
func analyzeFile(f *build.File) fileData {
	if f == nil {
		return fileData{}
	}

	report := fileData{
		loadedSymbols: make(map[string]function),
		rulesOrMacros: make(map[string]bool),
		functions:     make(map[string]map[string]funCall),
		aliases:       make(map[string]function),
	}

	// Collect loaded symbols
	for _, stmt := range f.Stmt {
		load, ok := stmt.(*build.LoadStmt)
		if !ok {
			continue
		}
		label := labels.ParseRelative(load.Module.Value, f.Pkg)
		if label.Repository != "" || label.Target == "" {
			continue
		}
		for i, from := range load.From {
			report.loadedSymbols[load.To[i].Name] = function{label.Package, label.Target, from.Name}
		}
	}

	for _, stmt := range f.Stmt {
		switch stmt := stmt.(type) {
		case *build.AssignExpr:
			// Analyze aliases (`foo = bar`) or rule declarations (`foo = rule(...)`)
			lhsIdent, ok := stmt.LHSIdent()
			if !ok {
				continue
			}
			if rhsIdent, ok := stmt.RHS.(*build.Ident); ok {
				report.aliases[lhsIdent.Name] = resolveExternal(function{f.Pkg, f.Label, rhsIdent.Name}, report.loadedSymbols)
				continue
			}

			call, ok := stmt.RHS.(*build.CallExpr)
			if !ok {
				continue
			}
			if ident, ok := call.X.(*build.Ident); ok {
				if ident.Name == "rule" || ident.Name == "macro" {
					report.rulesOrMacros[lhsIdent.Name] = true
					continue
				}
			}
		case *build.DefStmt:
			report.functions[stmt.Name] = getFunCalls(stmt, f.Pkg, f.Label, report.loadedSymbols)
		default:
			continue
		}
	}
	return report
}

// functionReport represents the analysis result of a function
type functionReport struct {
	isRuleOrMacro bool     // whether the function is a macro (or a rule)
	fc            *funCall // a call to the rule or another macro
}

// macroAnalyzer is an object that analyzes the directed graph of functions calling each other,
// loading other files lazily if necessary.
type macroAnalyzer struct {
	fileReader *FileReader
	files      map[string]fileData
	cache      map[function]functionReport
}

// getFileData retrieves a file using the fileReader object and extracts information about functions and rules
// defined in the file.
func (ma macroAnalyzer) getFileData(pkg, label string) fileData {
	filename := pkg + ":" + label
	if fd, ok := ma.files[filename]; ok {
		return fd
	}
	if ma.fileReader == nil {
		fd := fileData{}
		ma.files[filename] = fd
		return fd
	}
	f := ma.fileReader.GetFile(pkg, label)
	fd := analyzeFile(f)
	ma.files[filename] = fd
	return fd
}

// IsMacro is a public function that checks whether the given function is a macro
func (ma macroAnalyzer) IsRuleOrMacro(fn function) (report functionReport) {
	// Check the cache first
	if cached, ok := ma.cache[fn]; ok {
		return cached
	}
	// Write a negative result to the cache before analyzing. This will prevent stack overflow crashes
	// if the input data contains recursion.
	ma.cache[fn] = report
	defer func() {
		// Update the cache with the actual result
		ma.cache[fn] = report
	}()

	// Check for native rules
	if fn.filename == nativeModule {
		switch fn.name {
		case "glob", "existing_rule", "existing_rules", "package_name",
			"repository_name", "exports_files":
			// Not a rule
		default:
			report.isRuleOrMacro = true
		}
		return
	}

	fileData := ma.getFileData(fn.pkg, fn.filename)

	// Check whether fn.name is an alias for another function
	if alias, ok := fileData.aliases[fn.name]; ok {
		if ma.IsRuleOrMacro(alias).isRuleOrMacro {
			report.isRuleOrMacro = true
		}
		return
	}

	// Check whether fn.name is a rule or macro
	if fileData.rulesOrMacros[fn.name] {
		report.isRuleOrMacro = true
		return
	}

	// Check whether fn.name is a loaded symbol from another file
	if externalFn, ok := fileData.loadedSymbols[fn.name]; ok {
		if ma.IsRuleOrMacro(externalFn).isRuleOrMacro {
			report.isRuleOrMacro = true
			return
		}
	}

	// Check whether fn.name is an ordinary function
	funCalls, ok := fileData.functions[fn.name]
	if !ok {
		return
	}

	// Prioritize function calls from already loaded files. If some of the function calls are from the same file
	// (or another file that has been loaded already), check them first.
	var knownFunCalls, newFunCalls []funCall
	for _, fc := range funCalls {
		if _, ok := ma.files[fc.function.pkg+":"+fc.function.filename]; ok || fc.function.filename == nativeModule {
			knownFunCalls = append(knownFunCalls, fc)
		} else {
			newFunCalls = append(newFunCalls, fc)
		}
	}

	for _, fc := range append(knownFunCalls, newFunCalls...) {
		if ma.IsRuleOrMacro(fc.function).isRuleOrMacro {
			report.isRuleOrMacro = true
			report.fc = &fc
			return
		}
	}

	return
}

// newMacroAnalyzer creates and initiates an instance of macroAnalyzer.
func newMacroAnalyzer(fileReader *FileReader) macroAnalyzer {
	return macroAnalyzer{
		fileReader: fileReader,
		files:      make(map[string]fileData),
		cache:      make(map[function]functionReport),
	}
}

func unnamedMacroWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	macroAnalyzer := newMacroAnalyzer(fileReader)
	macroAnalyzer.files[f.Pkg+":"+f.Label] = analyzeFile(f)

	findings := []*LinterFinding{}
	for _, stmt := range f.Stmt {
		def, ok := stmt.(*build.DefStmt)
		if !ok {
			continue
		}

		if strings.HasPrefix(def.Name, "_") || acceptsNameArgument(def) {
			continue
		}

		report := macroAnalyzer.IsRuleOrMacro(function{f.Pkg, f.Label, def.Name})
		if !report.isRuleOrMacro {
			continue
		}
		msg := fmt.Sprintf(`The macro %q should have a keyword argument called "name".`, def.Name)
		if report.fc != nil {
			// fc shouldn't be nil because that's the only node that can be found inside a function.
			msg += fmt.Sprintf(`

It is considered a macro because it calls a rule or another macro %q on line %d.

By convention, every public macro needs a "name" argument (even if it doesn't use it).
This is important for tooling and automation.

  * If this function is a helper function that's not supposed to be used outside of this file,
    please make it private (e.g. rename it to "_%s").
  * Otherwise, add a "name" argument. If possible, use that name when calling other macros/rules.`, report.fc.nameAlias, report.fc.line, def.Name)
		}
		finding := makeLinterFinding(def, msg)
		_, finding.End = def.HeaderSpan()
		findings = append(findings, finding)
	}

	return findings
}
