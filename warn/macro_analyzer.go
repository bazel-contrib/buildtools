/*
Copyright 2025 Google LLC

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

// Analyzer to determine if a function is a macro (produces targets).

package warn

import (
	"fmt"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/labels"
)

// MacroAnalyzer is an object that analyzes the directed graph of functions calling each other,
// determining whether a function produces targets or not.
type MacroAnalyzer struct {
	fileReader *FileReader
	cache      map[symbolRef]*symbolAnalysisResult
}

// NewMacroAnalyzer creates and initiates an instance of macroAnalyzer.
func NewMacroAnalyzer(fileReader *FileReader) MacroAnalyzer {
	if fileReader == nil {
		// If no file reader is provided, a default one is provided which fails on all reads.
		// This can still be used if functions are preloaded via cache.
		fileReader = NewFileReader(func(_ string) ([]byte, error) {
			return nil, fmt.Errorf("tried to read file without file reader")
		})
	}
	return MacroAnalyzer{
		fileReader: fileReader,
		cache:      make(map[symbolRef]*symbolAnalysisResult),
	}
}

// MacroAnalyzerReport defines the results of analyzing a function using the MacroAnalyzer.
type MacroAnalyzerReport struct {
	SelfDescription string
	symbolAnalysis  *symbolAnalysisResult
}

// CanProduceTargets returns true if provided function has any call path which produces a target.
// A function which produces targets is by definition either a rule or a macro.
func (mar *MacroAnalyzerReport) CanProduceTargets() bool {
	if mar.symbolAnalysis == nil {
		return false
	}
	return mar.symbolAnalysis.canProduceTargets
}

// PrintableCallStack returns a user-readable call stack, providing a path for how a function may
// produce targets.
func (mar *MacroAnalyzerReport) PrintableCallStack() string {
	if mar.symbolAnalysis == nil {
		return ""
	}
	return strings.Join(mar.symbolAnalysis.callStackFrames, "\n")
}

// AnalyzeFn analyzes the provided def statement, and returns a report containing whether it produces a target (is a macro) or not.
func (ma *MacroAnalyzer) AnalyzeFn(f *build.File, def *build.DefStmt) (*MacroAnalyzerReport, error) {
	ma.fileReader.AddFileToCache(f)
	call := symbolCall{symbol: &symbolRef{pkg: f.Pkg, label: f.Label, name: def.Name}, line: exprLine(def)}
	report, err := ma.analyzeSymbol(call)
	if err != nil {
		return nil, err
	}
	return &MacroAnalyzerReport{
		SelfDescription: call.asCallStackFrame(),
		symbolAnalysis:  report,
	}, nil
}

// AnalyzeFnCall analyzes a function call to see if it can produce a targets or not.
func (ma *MacroAnalyzer) AnalyzeFnCall(f *build.File, call *build.CallExpr) (*MacroAnalyzerReport, error) {
	ma.fileReader.AddFileToCache(f)
	if symbolName := callExprToString(call); symbolName != "" {
		call := symbolCall{symbol: &symbolRef{pkg: f.Pkg, label: f.Label, name: symbolName}, line: exprLine(call)}
		report, err := ma.analyzeSymbol(call)
		if err != nil {
			return nil, err
		}
		return &MacroAnalyzerReport{
			SelfDescription: call.asCallStackFrame(),
			symbolAnalysis:  report,
		}, nil
	}
	return nil, fmt.Errorf("error checking call for being a macro at %s:%d", f.Path, exprLine(call))
}

// symbolAnalysisResult stores the result of analyzing a symbolRef.
type symbolAnalysisResult struct {
	canProduceTargets bool
	callStackFrames   []string
}

// symbolRef represents a symbol in a specific file.
type symbolRef struct {
	pkg   string
	label string
	name  string
}

// symbolCall represents a call (by line number) to a symbolRef.
type symbolCall struct {
	line   int
	symbol *symbolRef
}

func (sc *symbolCall) asCallStackFrame() string {
	return fmt.Sprintf("%s:%s:%d %s", sc.symbol.pkg, sc.symbol.label, sc.line, sc.symbol.name)
}

// traversalNode is an internal structure to keep track of symbol call hierarchies while traversing symbols.
type traversalNode struct {
	parent     *traversalNode
	symbolCall *symbolCall
}

// analyzeSymbol identifies a given symbol, and traverses its call stack to detect if any downstream calls can generate targets.
func (ma *MacroAnalyzer) analyzeSymbol(sc symbolCall) (*symbolAnalysisResult, error) {
	queue := []*traversalNode{{symbolCall: &sc}}
	visited := make(map[symbolRef]bool)

	var current *traversalNode
	var nodeProducedTarget *traversalNode

	for len(queue) > 0 && nodeProducedTarget == nil {
		current, queue = queue[0], queue[1:]
		visited[*current.symbolCall.symbol] = true

		if producesTarget(current.symbolCall.symbol) {
			nodeProducedTarget = current
		}
		calls, err := ma.expandSymbol(current.symbolCall.symbol)
		if err != nil {
			return nil, err
		}
		for _, call := range calls {
			if _, isVisited := visited[*call.symbol]; isVisited {
				continue
			}
			ref := &traversalNode{parent: current, symbolCall: &call}
			// adding symbol to front/back of queue depending on whether the file is already loaded or not.
			if ma.fileReader.IsCached(call.symbol.pkg, call.symbol.label) {
				queue = append([]*traversalNode{ref}, queue...)
			} else {
				queue = append(queue, ref)
			}
		}
	}
	if nodeProducedTarget == nil {
		// If no node produced a target, all visited nodes can be cached as non-macros.
		for symbol := range visited {
			ma.cache[symbol] = &symbolAnalysisResult{canProduceTargets: false}
		}
	} else {
		// If a node produced a target, the call stack above the node can be cached as producing targets.
		var callStackFrames []string
		node := nodeProducedTarget
		for node != nil {
			ma.cache[*node.symbolCall.symbol] = &symbolAnalysisResult{canProduceTargets: true, callStackFrames: callStackFrames}
			callStackFrames = append([]string{node.symbolCall.asCallStackFrame()}, callStackFrames...)
			node = node.parent
		}
	}
	return ma.cache[*sc.symbol], nil
}

// exprLine returns the start line of an expression
func exprLine(expr build.Expr) int {
	start, _ := expr.Span()
	return start.Line
}

// expandSymbol expands the provided symbol, returning a list of other symbols that it references.
// e.g. if the symbol is an alias, the aliased symbol is returned, or if the symbol is a function, the symbols it calls downstream are returned.
func (ma *MacroAnalyzer) expandSymbol(symbol *symbolRef) ([]symbolCall, error) {
	f := ma.fileReader.GetFile(symbol.pkg, symbol.label)
	if f == nil {
		return nil, fmt.Errorf("unable to find file %s:%s", symbol.pkg, symbol.label)
	}

	for _, stmt := range f.Stmt {
		switch stmt := stmt.(type) {
		case *build.AssignExpr:
			if lhsIdent, ok := stmt.LHS.(*build.Ident); ok && lhsIdent.Name == symbol.name {
				if rhsIdent, ok := stmt.RHS.(*build.Ident); ok {
					return []symbolCall{{
						symbol: &symbolRef{pkg: f.Pkg, label: f.Label, name: rhsIdent.Name},
						line:   exprLine(stmt),
					}}, nil
				}
				if fnName := callExprToString(stmt.RHS); fnName != "" {
					return []symbolCall{{
						symbol: &symbolRef{pkg: f.Pkg, label: f.Label, name: fnName},
						line:   exprLine(stmt),
					}}, nil
				}
			}
		case *build.DefStmt:
			if stmt.Name == symbol.name {
				var calls []symbolCall
				build.Walk(stmt, func(x build.Expr, _ []build.Expr) {
					if fnName := callExprToString(x); fnName != "" {
						calls = append(calls, symbolCall{
							symbol: &symbolRef{pkg: f.Pkg, label: f.Label, name: fnName},
							line:   exprLine(x),
						})
					}
				})
				return calls, nil
			}
		case *build.LoadStmt:
			label := labels.ParseRelative(stmt.Module.Value, f.Pkg)
			if label.Repository != "" || label.Target == "" {
				continue
			}
			for i, from := range stmt.From {
				if stmt.To[i].Name == symbol.name {
					return []symbolCall{{
						symbol: &symbolRef{pkg: label.Package, label: label.Target, name: from.Name},
						line:   exprLine(stmt),
					}}, nil
				}
			}
		}
	}
	return nil, nil
}

// callExprToString converts a callExpr to its "symbol name"
func callExprToString(expr build.Expr) string {
	call, ok := expr.(*build.CallExpr)
	if !ok {
		return ""
	}

	if fnIdent, ok := call.X.(*build.Ident); ok {
		return fnIdent.Name
	}

	// call of the format obj.fn(...), ignores call if anything other than ident.fn().
	if fn, ok := call.X.(*build.DotExpr); ok {
		if obj, ok := fn.X.(*build.Ident); ok {
			return fmt.Sprintf("%s.%s", obj.Name, fn.Name)
		}
	}
	return ""
}

// native functions which do not produce targets (https://bazel.build/rules/lib/toplevel/native).
var nativeRuleExceptions = map[string]bool{
	"native.existing_rule":              true,
	"native.existing_rules":             true,
	"native.exports_files":              true,
	"native.glob":                       true,
	"native.module_name":                true,
	"native.module_version":             true,
	"native.package_default_visibility": true,
	"native.package_group":              true,
	"native.package_name":               true,
	"native.package_relative_label":     true,
	"native.repo_name":                  true,
	"native.repository_name":            true,
	"native.subpackages":                true,
}

// producesTargets returns true if the symbol name is a known generator of a target.
func producesTarget(s *symbolRef) bool {
	// Calls to the macro() symbol produce a symbolic macro (https://bazel.build/extending/macros).
	if s.name == "macro" {
		return true
	}
	// Calls to the rule() symbol define a rule (https://bazel.build/extending/rules).
	if s.name == "rule" {
		return true
	}
	// Calls to native. invokes native rules (except defined list of native helper functions).
	// https://bazel.build/rules/lib/toplevel/native
	if strings.HasPrefix(s.name, "native.") {
		if _, ok := nativeRuleExceptions[s.name]; !ok {
			return true
		}
	}
	return false
}
