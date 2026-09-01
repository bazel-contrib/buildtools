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

// Package bzlenv provides function to create and update a static environment.
package bzlenv

import (
	"github.com/bazelbuild/buildtools/build"
)

// ValueKind describes how a binding was declared.
type ValueKind int

// List of ValueKind values.
const (
	Builtin       ValueKind = iota // language builtin
	Imported                       // declared with load()
	Global                         // declared with assignment on top-level
	Function                       // declared with a def
	Parameter                      // function parameter
	Local                          // local variable, defined with assignment or as a loop variable
	TypeParameter                  // type parameter of a type alias or a generic function
)

func (k ValueKind) String() string {
	switch k {
	case Builtin:
		return "builtin"
	case Imported:
		return "imported"
	case Global:
		return "global"
	case Function:
		return "function"
	case Parameter:
		return "parameter"
	case Local:
		return "local"
	case TypeParameter:
		return "type parameter"
	default:
		panic(k)
	}
}

// NameInfo represents information about a symbol name.
type NameInfo struct {
	ID         int    // unique identifier
	Name       string // name of the variable (not unique)
	Kind       ValueKind
	Definition build.Expr // node that defines the value
}

type block map[string]NameInfo

// Environment represents a static environment (e.g. information about all available symbols).
type Environment struct {
	Blocks   []block
	Function *build.DefStmt // enclosing function (or nil on top-level)
	nextID   int            // used to create unique identifiers
	Stack    []build.Expr   // parents of the current node
}

// NewEnvironment creates a new empty Environment.
func NewEnvironment() *Environment {
	sc := block{}
	return &Environment{[]block{sc}, nil, 0, []build.Expr{}}
}

func (e *Environment) enterBlock() {
	e.Blocks = append(e.Blocks, block{})
}

func (e *Environment) exitBlock() {
	if len(e.Blocks) < 1 {
		panic("no block to close")
	}
	e.Blocks = e.Blocks[:len(e.Blocks)-1]
}

func (e *Environment) currentBlock() block {
	return e.Blocks[len(e.Blocks)-1]
}

func (sc *block) declare(name string, kind ValueKind, definition build.Expr, id int) {
	(*sc)[name] = NameInfo{
		ID:         id,
		Name:       name,
		Definition: definition,
		Kind:       kind}
}

// Get resolves the name and resolves information about the binding (or nil if it's not defined).
func (e *Environment) Get(name string) *NameInfo {
	for i := len(e.Blocks) - 1; i >= 0; i-- {
		if ret, ok := e.Blocks[i][name]; ok {
			return &ret
		}
	}
	return nil
}

func (e *Environment) declare(name string, kind ValueKind, node build.Expr) {
	sc := e.currentBlock()
	sc.declare(name, kind, node, e.nextID)
	e.nextID++
}

func declareGlobals(stmts []build.Expr, env *Environment) {
	for _, node := range stmts {
		switch node := node.(type) {
		case *build.LoadStmt:
			for _, ident := range node.To {
				env.declare(ident.Name, Imported, ident)
			}
		case *build.TypeAliasStmt:
			// Type aliases allowed only at top level.
			env.declare(node.TypeName(), Global, node)
		case *build.AssignExpr:
			kind := Local
			if env.Function == nil {
				kind = Global
			}
			for _, id := range CollectLValues(node.LHS) {
				env.declare(id.Name, kind, node)
			}
		case *build.TypedIdent:
			// var statement
			kind := Local
			if env.Function == nil {
				kind = Global
			}
			env.declare(node.Name(), kind, node)
		case *build.DefStmt:
			env.declare(node.Name, Function, node)
		}
	}
}

// CollectLValues returns the list of identifiers that are assigned (assuming that node is a valid
// LValue). For example, it returns `a`, `b` and `c` for the input `a, (b, c)`.
func CollectLValues(node build.Expr) []*build.Ident {
	var result []*build.Ident
	switch node := node.(type) {
	case *build.Ident:
		result = append(result, node)
	case *build.TypedIdent:
		result = CollectLValues(node.Ident)
	case *build.TupleExpr:
		for _, item := range node.List {
			result = append(result, CollectLValues(item)...)
		}
	case *build.ListExpr:
		for _, item := range node.List {
			result = append(result, CollectLValues(item)...)
		}
	}
	return result
}

func declareParams(fct *build.DefStmt, env *Environment) {
	for _, node := range fct.Params {
		name, _ := build.GetParamName(node)
		env.declare(name, Parameter, node)
	}
}

func declareLocalVariables(stmts []build.Expr, env *Environment) {
	for _, stmt := range stmts {
		switch node := stmt.(type) {
		case *build.AssignExpr:
			kind := Local
			if env.Function == nil {
				kind = Global
			}
			for _, id := range CollectLValues(node.LHS) {
				env.declare(id.Name, kind, node)
			}
		case *build.TypedIdent:
			// var statement
			kind := Local
			if env.Function == nil {
				kind = Global
			}
			env.declare(node.Name(), kind, node)
		case *build.IfStmt:
			declareLocalVariables(node.True, env)
			declareLocalVariables(node.False, env)
		case *build.ForStmt:
			for _, id := range CollectLValues(node.Vars) {
				env.declare(id.Name, Local, node)
			}
			declareLocalVariables(node.Body, env)
		}
	}
}

// WalkOnceWithEnvironment calls fct on every child of node, while maintaining the Environment of all available symbols.
func WalkOnceWithEnvironment(node build.Expr, env *Environment, fct func(e *build.Expr, env *Environment)) {
	env.Stack = append(env.Stack, node)
	switch node := node.(type) {
	case *build.File:
		declareGlobals(node.Stmt, env)
		build.WalkOnce(node, func(e *build.Expr) { fct(e, env) })
	case *build.DefStmt:
		if node.TypeParams != nil {
			env.enterBlock()
			for _, id := range CollectLValues(node.TypeParams) {
				env.declare(id.Name, TypeParameter, node)
			}
		}
		env.enterBlock()
		env.Function = node
		declareParams(node, env)
		declareLocalVariables(node.Body, env)
		build.WalkOnce(node, func(e *build.Expr) { fct(e, env) })
		env.Function = nil
		env.exitBlock()
		if node.TypeParams != nil {
			env.exitBlock()
		}
	case *build.TypeAliasStmt:
		if node.TypeParams != nil {
			env.enterBlock()
			for _, id := range CollectLValues(node.TypeParams) {
				env.declare(id.Name, TypeParameter, node)
			}
		}
		build.WalkOnce(node, func(e *build.Expr) { fct(e, env) })
		if node.TypeParams != nil {
			env.exitBlock()
		}
	case *build.Comprehension:
		env.enterBlock()
		for _, clause := range node.Clauses {
			switch clause := clause.(type) {
			case *build.ForClause:
				for _, id := range CollectLValues(clause.Vars) {
					env.declare(id.Name, Local, node)
				}
			}
		}
		build.WalkOnce(node, func(e *build.Expr) { fct(e, env) })
		env.exitBlock()
	default:
		build.WalkOnce(node, func(e *build.Expr) { fct(e, env) })
	}
	env.Stack = env.Stack[:len(env.Stack)-1]
}
