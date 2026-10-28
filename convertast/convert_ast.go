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

// This file contains functions to convert from one AST to the other.
// Input: AST from go.starlark.net/syntax
// Output: AST from github.com/bazel-contrib/buildtools/v10/build

package convertast

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bazel-contrib/buildtools/v10/build"
	"go.starlark.net/syntax"
)

func ConvFile(f *syntax.File) *build.File {
	stmts := []build.Expr{}
	for _, stmt := range f.Stmts {
		stmts = append(stmts, convStmt(stmt))
	}

	return &build.File{
		Type:     build.TypeDefault,
		Stmt:     stmts,
		Comments: convComments(f.Comments()),
	}
}

func convStmt(stmt syntax.Stmt) build.Expr {
	switch stmt := stmt.(type) {
	case *syntax.ExprStmt:
		s := convExpr(stmt.X)
		*s.Comment() = convComments(stmt.Comments())
		return s
	case *syntax.BranchStmt:
		return &build.BranchStmt{
			Token:    stmt.Token.String(),
			Comments: convComments(stmt.Comments()),
		}
	case *syntax.LoadStmt:
		load := &build.LoadStmt{
			Module:       convExpr(stmt.Module).(*build.StringExpr),
			ForceCompact: singleLine(stmt),
		}
		for _, ident := range stmt.From {
			load.From = append(load.From, convExpr(ident).(*build.Ident))
		}
		for _, ident := range stmt.To {
			load.To = append(load.To, convExpr(ident).(*build.Ident))
		}
		return load
	case *syntax.AssignStmt:
		return &build.AssignExpr{
			Op:       stmt.Op.String(),
			LHS:      convExpr(stmt.LHS),
			RHS:      convExpr(stmt.RHS),
			Comments: convComments(stmt.Comments()),
		}
	case *syntax.IfStmt:
		return &build.IfStmt{
			Cond:     convExpr(stmt.Cond),
			True:     convStmts(stmt.True),
			False:    convStmts(stmt.False),
			Comments: convComments(stmt.Comments()),
		}
	case *syntax.DefStmt:
		return &build.DefStmt{
			Name:     stmt.Name.Name,
			Comments: convComments(stmt.Comments()),
			Function: build.Function{
				Params: convExprs(stmt.Params),
				Body:   convStmts(stmt.Body),
			},
		}
	case *syntax.ForStmt:
		return &build.ForStmt{
			Vars:     convExpr(stmt.Vars),
			X:        convExpr(stmt.X),
			Comments: convComments(stmt.Comments()),
			Body:     convStmts(stmt.Body),
		}
	case *syntax.ReturnStmt:
		return &build.ReturnStmt{
			Comments: convComments(stmt.Comments()),
			Result:   convExpr(stmt.Result),
		}
	}
	panic("unreachable")
}

func convStmts(list []syntax.Stmt) []build.Expr {
	res := []build.Expr{}
	for _, i := range list {
		res = append(res, convStmt(i))
	}
	return res
}

func convExprs(list []syntax.Expr) []build.Expr {
	res := []build.Expr{}
	for _, i := range list {
		res = append(res, convExpr(i))
	}
	return res
}

func convCommentList(list []syntax.Comment, txt string) []build.Comment {
	res := []build.Comment{}
	for _, c := range list {
		res = append(res, build.Comment{Token: c.Text})
	}
	return res
}

func convComments(c *syntax.Comments) build.Comments {
	if c == nil {
		return build.Comments{}
	}
	return build.Comments{
		Before: convCommentList(c.Before, "before"),
		Suffix: convCommentList(c.Suffix, "suffix"),
		After:  convCommentList(c.After, "after"),
	}
}

// singleLine returns true if the node fits on a single line.
func singleLine(n syntax.Node) bool {
	start, end := n.Span()
	return start.Line == end.Line
}

func convClauses(list []syntax.Node) []build.Expr {
	res := []build.Expr{}
	for _, c := range list {
		switch stmt := c.(type) {
		case *syntax.ForClause:
			res = append(res, &build.ForClause{
				Vars: convExpr(stmt.Vars),
				X:    convExpr(stmt.X),
			})
		case *syntax.IfClause:
			res = append(res, &build.IfClause{
				Cond: convExpr(stmt.Cond),
			})
		}
	}
	return res
}

func convExpr(e syntax.Expr) build.Expr {
	if e == nil {
		return nil
	}
	switch e := e.(type) {
	case *syntax.Literal:
		switch e.Token {
		case syntax.INT:
			return &build.LiteralExpr{
				Token:    strconv.FormatInt(e.Value.(int64), 10),
				Comments: convComments(e.Comments())}
		case syntax.FLOAT:
			return &build.LiteralExpr{
				Token:    e.Raw,
				Comments: convComments(e.Comments())}
		case syntax.STRING:
			return &build.StringExpr{
				Value:       e.Value.(string),
				TripleQuote: strings.HasPrefix(e.Raw, "\"\"\""),
				Comments:    convComments(e.Comments())}
		}
	case *syntax.Ident:
		return &build.Ident{Name: e.Name, Comments: convComments(e.Comments())}
	case *syntax.BinaryExpr:
		_, lhsEnd := e.X.Span()
		rhsBegin, _ := e.Y.Span()
		if e.Op.String() == "=" {
			return &build.AssignExpr{
				LHS:       convExpr(e.X),
				RHS:       convExpr(e.Y),
				Op:        e.Op.String(),
				LineBreak: lhsEnd.Line != rhsBegin.Line,
				Comments:  convComments(e.Comments())}
		}
		return &build.BinaryExpr{
			X:         convExpr(e.X),
			Y:         convExpr(e.Y),
			Op:        e.Op.String(),
			LineBreak: lhsEnd.Line != rhsBegin.Line,
			Comments:  convComments(e.Comments())}
	case *syntax.UnaryExpr:
		return &build.UnaryExpr{Op: e.Op.String(), X: convExpr(e.X)}
	case *syntax.SliceExpr:
		return &build.SliceExpr{X: convExpr(e.X), From: convExpr(e.Lo), To: convExpr(e.Hi), Step: convExpr(e.Step)}
	case *syntax.DotExpr:
		return &build.DotExpr{X: convExpr(e.X), Name: e.Name.Name}
	case *syntax.CallExpr:
		args := []build.Expr{}
		for _, a := range e.Args {
			args = append(args, convExpr(a))
		}
		return &build.CallExpr{
			X:            convExpr(e.Fn),
			List:         args,
			ForceCompact: singleLine(e),
		}
	case *syntax.ListExpr:
		list := []build.Expr{}
		for _, i := range e.List {
			list = append(list, convExpr(i))
		}
		return &build.ListExpr{List: list, Comments: convComments(e.Comments())}
	case *syntax.DictExpr:
		list := []*build.KeyValueExpr{}
		for i := range e.List {
			entry := e.List[i].(*syntax.DictEntry)
			list = append(list, &build.KeyValueExpr{
				Key:      convExpr(entry.Key),
				Value:    convExpr(entry.Value),
				Comments: convComments(entry.Comments()),
			})
		}
		return &build.DictExpr{List: list, Comments: convComments(e.Comments())}
	case *syntax.CondExpr:
		return &build.ConditionalExpr{
			Then:     convExpr(e.True),
			Test:     convExpr(e.Cond),
			Else:     convExpr(e.False),
			Comments: convComments(e.Comments()),
		}
	case *syntax.Comprehension:
		return &build.Comprehension{
			Body:     convExpr(e.Body),
			Clauses:  convClauses(e.Clauses),
			Comments: convComments(e.Comments()),
			Curly:    e.Curly,
		}
	case *syntax.ParenExpr:
		return &build.ParenExpr{
			X:        convExpr(e.X),
			Comments: convComments(e.Comments()),
		}
	case *syntax.TupleExpr:
		return &build.TupleExpr{
			List:         convExprs(e.List),
			NoBrackets:   !e.Lparen.IsValid(),
			Comments:     convComments(e.Comments()),
			ForceCompact: singleLine(e),
		}
	case *syntax.IndexExpr:
		return &build.IndexExpr{
			X:        convExpr(e.X),
			Y:        convExpr(e.Y),
			Comments: convComments(e.Comments()),
		}
	case *syntax.LambdaExpr:
		return &build.LambdaExpr{
			Comments: convComments(e.Comments()),
			Function: build.Function{
				Params: convExprs(e.Params),
				Body:   []build.Expr{convExpr(e.Body)},
			},
		}
	default:
		panic(fmt.Sprintf("other expr: %T %+v", e, e))
	}
	panic("unreachable")
}
