/*
Copyright 2021 Google LLC

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

package build

// GetParamName extracts the param name from an item of function params.
func GetParamName(param Expr) (name, op string) {
	ident, op := GetParamIdent(param)
	if ident == nil {
		return "", ""
	}
	return ident.Name, op
}

// GetParamIdent extracts the param identifier from an item of function params.
func GetParamIdent(param Expr) (ident *Ident, op string) {
	switch param := param.(type) {
	case *Ident:
		return param, ""
	case *TypedIdent:
		ident, _ := GetParamIdent(param.Ident)
		return ident, ""
	case *AssignExpr:
		// keyword parameter
		return GetParamIdent(param.LHS)
	case *UnaryExpr:
		// *args, **kwargs, or *
		if param.X == nil {
			// An asterisk separating position and keyword-only arguments
			break
		}
		ident, _ := GetParamIdent(param.X)
		return ident, param.Op
	}
	return nil, ""
}

// GetTypes returns the list of types defined by the a given expression.
// Examples:
//
// List[tuple[bool, int]] should return [List, Tuple, bool, int]
// str should return str
func GetTypes(t Expr) []string {
	switch t := t.(type) {
	case *TypeAliasStmt:
		return GetTypes(t.Type)
	case *TypedIdent:
		return GetTypes(t.Type)
	case *TypeExpr:
		var ret []string
		for _, x := range t.List {
			ret = append(ret, GetTypes(x)...)
		}
		return ret
	case *TypeAppExpr:
		ret := GetTypes(t.Type)
		for _, x := range t.Args {
			ret = append(ret, GetTypes(x)...)
		}
		return ret
	case *EllipsisExpr:
		return []string{}
	case *Ident:
		return []string{t.Name}
	case *DefStmt:
		ret := GetTypes(t.Type)
		params := make([]string, 0)
		for _, p := range t.Params {
			params = append(params, GetTypes(p)...)
		}
		return append(ret, params...)
	case *IndexExpr:
		left := GetTypes(t.X)
		right := GetTypes(t.Y)
		return append(left, right...)
	case *DotExpr:
		return GetTypes(t.X)
	case *BinaryExpr:
		left := GetTypes(t.X)
		right := GetTypes(t.Y)
		return append(left, right...)
	case *ListExpr:
		var ret []string
		for _, x := range t.List {
			ret = append(ret, GetTypes(x)...)
		}
		return ret
	case *DictExpr:
		var ret []string
		for _, kv := range t.List {
			ret = append(ret, GetTypes(kv.Value)...)
		}
		return ret
	case *TupleExpr:
		var ret []string
		for _, x := range t.List {
			ret = append(ret, GetTypes(x)...)
		}
		return ret
	default:
		return []string{}
	}
}

// IsMultiLine returns whether an Expr is multiline or not.
func IsMultiLine(param Expr) bool {
	start, end := param.Span()
	if end.Line == 0 && end.Byte == 0 && end.LineRune == 0 {
		// If the end position is unknown, we assume it's single line (for backwards compatibility).
		return false
	}
	return start.Line != end.Line
}
