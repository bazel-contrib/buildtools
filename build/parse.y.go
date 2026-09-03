//line build/parse.y:29
package build

import __yyfmt__ "fmt"

//line build/parse.y:29

//line build/parse.y:34
type yySymType struct {
	yys int
	// input tokens
	tok    string   // raw input syntax
	str    string   // decoding of quoted string
	pos    Position // position of token
	triple bool     // was string triple quoted?

	// partial syntax trees
	expr    Expr
	exprs   []Expr
	kv      *KeyValueExpr
	kvs     []*KeyValueExpr
	string  *StringExpr
	ifstmt  *IfStmt
	loadarg *struct {
		from Ident
		to   Ident
	}
	loadargs []*struct {
		from Ident
		to   Ident
	}
	def_header *DefStmt // partially filled in def statement, without the body

	// supporting information
	comma    Position // position of trailing comma in list, if present
	lastStmt Expr     // most recent rule, to attach line comments to
}

const _AUGM = 57346
const _AND = 57347
const _CAST = 57348
const _COMMENT = 57349
const _EOF = 57350
const _EQ = 57351
const _FOR = 57352
const _GE = 57353
const _IDENT = 57354
const _INT = 57355
const _IF = 57356
const _ELSE = 57357
const _ELIF = 57358
const _IN = 57359
const _IS = 57360
const _ISINSTANCE = 57361
const _LAMBDA = 57362
const _LOAD = 57363
const _LE = 57364
const _NE = 57365
const _STAR_STAR = 57366
const _INT_DIV = 57367
const _BIT_LSH = 57368
const _BIT_RSH = 57369
const _ARROW = 57370
const _NOT = 57371
const _OR = 57372
const _STRING = 57373
const _DEF = 57374
const _RETURN = 57375
const _PASS = 57376
const _BREAK = 57377
const _CONTINUE = 57378
const _INDENT = 57379
const _UNINDENT = 57380
const _ELLIPSIS = 57381
const ShiftInstead = 57382
const _ASSERT = 57383
const _UNARY = 57384

var yyToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"'%'",
	"'('",
	"')'",
	"'*'",
	"'+'",
	"','",
	"'-'",
	"'.'",
	"'/'",
	"':'",
	"'<'",
	"'='",
	"'>'",
	"'['",
	"']'",
	"'{'",
	"'}'",
	"'|'",
	"'&'",
	"'^'",
	"'~'",
	"_AUGM",
	"_AND",
	"_CAST",
	"_COMMENT",
	"_EOF",
	"_EQ",
	"_FOR",
	"_GE",
	"_IDENT",
	"_INT",
	"_IF",
	"_ELSE",
	"_ELIF",
	"_IN",
	"_IS",
	"_ISINSTANCE",
	"_LAMBDA",
	"_LOAD",
	"_LE",
	"_NE",
	"_STAR_STAR",
	"_INT_DIV",
	"_BIT_LSH",
	"_BIT_RSH",
	"_ARROW",
	"_NOT",
	"_OR",
	"_STRING",
	"_DEF",
	"_RETURN",
	"_PASS",
	"_BREAK",
	"_CONTINUE",
	"_INDENT",
	"_UNINDENT",
	"_ELLIPSIS",
	"ShiftInstead",
	"'\\n'",
	"_ASSERT",
	"_UNARY",
	"';'",
}

var yyStatenames = [...]string{}

const yyEofCode = 1
const yyErrCode = 2
const yyInitialStackSize = 16

//line build/parse.y:1300

// Go helper code.

// unary returns a unary expression with the given
// position, operator, and subexpression.
func unary(pos Position, op string, x Expr) Expr {
	return &UnaryExpr{
		OpStart: pos,
		Op:      op,
		X:       x,
	}
}

// binary returns a binary expression with the given
// operands, position, and operator.
func binary(x Expr, pos Position, op string, y Expr) Expr {
	_, xend := x.Span()
	ystart, _ := y.Span()

	switch op {
	case "=", "+=", "-=", "*=", "/=", "//=", "%=", "&=", "|=", "^=", "<<=", ">>=":
		return &AssignExpr{
			LHS:       x,
			OpPos:     pos,
			Op:        op,
			LineBreak: xend.Line < ystart.Line,
			RHS:       y,
		}
	}

	return &BinaryExpr{
		X:         x,
		OpStart:   pos,
		Op:        op,
		LineBreak: xend.Line < ystart.Line,
		Y:         y,
	}
}

// typed returns a TypedIdent expression
func typed(x, y Expr) *TypedIdent {
	return &TypedIdent{
		Ident: x,
		Type:  y,
	}
}

// isSimpleExpression returns whether an expression is simple and allowed to exist in
// compact forms of sequences.
// The formal criteria are the following: an expression is considered simple if it's
// a literal (variable, string or a number), a literal with a unary operator or an empty sequence.
func isSimpleExpression(expr *Expr) bool {
	switch x := (*expr).(type) {
	case *LiteralExpr, *StringExpr, *Ident:
		return true
	case *UnaryExpr:
		_, literal := x.X.(*LiteralExpr)
		_, ident := x.X.(*Ident)
		return literal || ident
	case *ListExpr:
		return len(x.List) == 0
	case *TupleExpr:
		return len(x.List) == 0
	case *DictExpr:
		return len(x.List) == 0
	case *SetExpr:
		return len(x.List) == 0
	case *EllipsisExpr:
		return true
	default:
		return false
	}
}

// forceCompact returns the setting for the ForceCompact field for a call or tuple.
//
// NOTE 1: The field is called ForceCompact, not ForceSingleLine,
// because it only affects the formatting associated with the call or tuple syntax,
// not the formatting of the arguments. For example:
//
//	call([
//		1,
//		2,
//		3,
//	])
//
// is still a compact call even though it runs on multiple lines.
//
// In contrast the multiline form puts a linebreak after the (.
//
//	call(
//		[
//			1,
//			2,
//			3,
//		],
//	)
//
// NOTE 2: Because of NOTE 1, we cannot use start and end on the
// same line as a signal for compact mode: the formatting of an
// embedded list might move the end to a different line, which would
// then look different on rereading and cause buildifier not to be
// idempotent. Instead, we have to look at properties guaranteed
// to be preserved by the reformatting, namely that the opening
// paren and the first expression are on the same line and that
// each subsequent expression begins on the same line as the last
// one ended (no line breaks after comma).
func forceCompact(start Position, list []Expr, end Position) bool {
	if len(list) <= 1 {
		// The call or tuple will probably be compact anyway; don't force it.
		return false
	}

	// If there are any named arguments or non-string, non-literal
	// arguments, cannot force compact mode.
	line := start.Line
	for _, x := range list {
		start, end := x.Span()
		if start.Line != line {
			return false
		}
		line = end.Line
		if !isSimpleExpression(&x) {
			return false
		}
	}
	return end.Line == line
}

// forceMultiLine returns the setting for the ForceMultiLine field.
func forceMultiLine(start Position, list []Expr, end Position) bool {
	if len(list) > 1 {
		// The call will be multiline anyway, because it has multiple elements. Don't force it.
		return false
	}

	if len(list) == 0 {
		// Empty list: use position of brackets.
		return start.Line != end.Line
	}

	// Single-element list.
	// Check whether opening bracket is on different line than beginning of
	// element, or closing bracket is on different line than end of element.
	elemStart, elemEnd := list[0].Span()
	return start.Line != elemStart.Line || end.Line != elemEnd.Line
}

// forceMultiLineComprehension returns the setting for the ForceMultiLine field for a comprehension.
func forceMultiLineComprehension(start Position, expr Expr, clauses []Expr, end Position) bool {
	// Return true if there's at least one line break between start, expr, each clause, and end
	exprStart, exprEnd := expr.Span()
	if start.Line != exprStart.Line {
		return true
	}
	previousEnd := exprEnd
	for _, clause := range clauses {
		clauseStart, clauseEnd := clause.Span()
		if previousEnd.Line != clauseStart.Line {
			return true
		}
		previousEnd = clauseEnd
	}
	return previousEnd.Line != end.Line
}

// isBlockStmt reports whether x is a statement with an indentable body
// (def, for, or if). Line comments that follow such statements should form
// standalone CommentBlock statements rather than being attached to the block
// as After-comments, matching how the same comments are parsed when the
// block is written in its expanded form (see extractTrailingComments).
func isBlockStmt(x Expr) bool {
	switch x.(type) {
	case *DefStmt, *ForStmt, *IfStmt:
		return true
	}
	return false
}

// extractTrailingComments extracts trailing comments of an indented block starting with the first
// comment line with indentation less than the block indentation.
// The comments can either belong to CommentBlock statements or to the last non-comment statement
// as After-comments.
func extractTrailingComments(stmt Expr) []Expr {
	body := getLastBody(stmt)
	var comments []Expr
	if body != nil && len(*body) > 0 {
		// Get the current indentation level
		start, _ := (*body)[0].Span()
		indentation := start.LineRune

		// Find the last non-comment statement
		lastNonCommentIndex := -1
		for i, stmt := range *body {
			if _, ok := stmt.(*CommentBlock); !ok {
				lastNonCommentIndex = i
			}
		}
		if lastNonCommentIndex == -1 {
			return comments
		}

		// Iterate over the trailing comments, find the first comment line that's not indented enough,
		// dedent it and all the following comments.
		for i := lastNonCommentIndex; i < len(*body); i++ {
			stmt := (*body)[i]
			if comment := extractDedentedComment(stmt, indentation); comment != nil {
				// This comment and all the following CommentBlock statements are to be extracted.
				comments = append(comments, comment)
				comments = append(comments, (*body)[i+1:]...)
				*body = (*body)[:i+1]
				// If the current statement is a CommentBlock statement without any comment lines
				// it should be removed too.
				if i > lastNonCommentIndex && len(stmt.Comment().After) == 0 {
					*body = (*body)[:i]
				}
			}
		}
	}
	return comments
}

// extractDedentedComment extract the first comment line from `stmt` which indentation is smaller
// than `indentation`, and all following comment lines, and returns them in a newly created
// CommentBlock statement.
func extractDedentedComment(stmt Expr, indentation int) Expr {
	for i, line := range stmt.Comment().After {
		// line.Start.LineRune == 0 can't exist in parsed files, it indicates that the comment line
		// has been added by an AST modification. Don't take such lines into account.
		if line.Start.LineRune > 0 && line.Start.LineRune < indentation {
			// This and all the following lines should be dedented
			cb := &CommentBlock{
				Start:    line.Start,
				Comments: Comments{After: stmt.Comment().After[i:]},
			}
			stmt.Comment().After = stmt.Comment().After[:i]
			return cb
		}
	}
	return nil
}

// getLastBody returns the last body of a block statement (the only body for For- and DefStmt
// objects, the last in a if-elif-else chain
func getLastBody(stmt Expr) *[]Expr {
	switch block := stmt.(type) {
	case *DefStmt:
		return &block.Body
	case *ForStmt:
		return &block.Body
	case *IfStmt:
		if len(block.False) == 0 {
			return &block.True
		} else if len(block.False) == 1 {
			if next, ok := block.False[0].(*IfStmt); ok {
				// Recursively find the last block of the chain
				return getLastBody(next)
			}
		}
		return &block.False
	}
	return nil
}

// Expose lex.ErrorAt to the parser.
type yyLexerWithErrorAt interface {
	ErrorAt(pos Position, s string)
}

func errorAt(yylex yyLexer, pos Position, s string) {
	if lex, ok := yylex.(yyLexerWithErrorAt); ok {
		lex.ErrorAt(pos, s)
	} else {
		yylex.Error(s)
	}
}

//line yacctab:1
var yyExca = [...]int16{
	-1, 1,
	1, -1,
	-2, 0,
	-1, 88,
	6, 66,
	-2, 139,
	-1, 194,
	20, 136,
	-2, 137,
}

const yyPrivate = 57344

const yyLast = 1164

var yyAct = [...]int16{
	22, 34, 253, 302, 273, 171, 291, 165, 267, 119,
	252, 126, 7, 2, 178, 106, 9, 117, 47, 184,
	185, 83, 27, 116, 282, 166, 268, 247, 199, 44,
	96, 97, 98, 99, 48, 23, 43, 254, 56, 104,
	109, 112, 147, 222, 58, 59, 257, 236, 198, 114,
	222, 23, 251, 124, 43, 92, 103, 245, 259, 162,
	260, 134, 135, 136, 137, 138, 139, 140, 141, 142,
	143, 144, 145, 146, 23, 148, 149, 150, 151, 152,
	153, 154, 155, 156, 270, 222, 43, 244, 257, 94,
	49, 246, 125, 15, 130, 121, 61, 111, 85, 60,
	259, 258, 260, 179, 62, 222, 55, 160, 176, 23,
	23, 105, 121, 189, 192, 23, 23, 61, 132, 93,
	60, 64, 271, 65, 280, 62, 190, 95, 43, 23,
	187, 188, 206, 200, 193, 77, 196, 15, 63, 177,
	133, 120, 54, 258, 204, 129, 167, 129, 211, 61,
	210, 108, 60, 64, 239, 65, 234, 62, 206, 63,
	79, 80, 23, 313, 214, 187, 179, 329, 240, 187,
	205, 52, 219, 206, 227, 159, 206, 50, 228, 217,
	221, 191, 232, 233, 332, 174, 175, 51, 310, 238,
	220, 63, 129, 180, 238, 213, 241, 243, 263, 226,
	183, 84, 324, 229, 230, 235, 237, 15, 167, 300,
	48, 235, 242, 262, 299, 52, 207, 284, 250, 231,
	320, 179, 208, 15, 223, 274, 173, 261, 13, 296,
	179, 173, 278, 88, 157, 272, 52, 279, 52, 87,
	248, 113, 209, 53, 276, 89, 52, 319, 129, 316,
	161, 224, 15, 170, 275, 86, 180, 46, 281, 323,
	167, 295, 292, 93, 52, 222, 283, 215, 277, 194,
	288, 172, 294, 333, 326, 325, 293, 303, 285, 122,
	123, 307, 286, 218, 305, 306, 131, 197, 309, 216,
	102, 7, 101, 100, 115, 202, 311, 129, 314, 129,
	15, 274, 1, 10, 317, 321, 264, 269, 20, 322,
	301, 180, 201, 110, 312, 309, 315, 292, 168, 328,
	180, 129, 327, 107, 45, 303, 335, 334, 57, 21,
	336, 12, 8, 4, 318, 33, 186, 169, 290, 15,
	289, 256, 40, 287, 255, 127, 128, 158, 42, 203,
	129, 16, 24, 265, 38, 266, 39, 330, 331, 90,
	91, 297, 298, 163, 36, 164, 304, 0, 0, 0,
	23, 41, 15, 0, 0, 0, 129, 37, 0, 35,
	0, 0, 0, 0, 0, 0, 269, 129, 0, 43,
	40, 0, 0, 31, 0, 30, 42, 0, 0, 225,
	0, 0, 38, 129, 39, 0, 0, 129, 0, 32,
	129, 129, 36, 6, 304, 0, 11, 0, 23, 41,
	26, 0, 0, 0, 0, 37, 28, 35, 0, 0,
	0, 0, 0, 0, 249, 29, 0, 43, 25, 14,
	17, 18, 19, 40, 308, 0, 31, 5, 30, 42,
	0, 0, 0, 0, 0, 38, 0, 39, 0, 0,
	0, 0, 32, 0, 0, 36, 6, 3, 0, 11,
	0, 23, 41, 26, 0, 0, 0, 0, 37, 28,
	35, 0, 0, 0, 0, 0, 0, 0, 29, 0,
	43, 25, 14, 17, 18, 19, 40, 0, 0, 31,
	5, 30, 42, 0, 0, 0, 0, 0, 38, 0,
	39, 0, 0, 0, 0, 32, 0, 0, 36, 0,
	0, 0, 0, 0, 23, 41, 0, 0, 0, 0,
	0, 37, 28, 35, 0, 0, 0, 0, 0, 0,
	0, 29, 0, 43, 0, 14, 17, 18, 19, 40,
	0, 0, 31, 118, 30, 42, 0, 0, 0, 0,
	0, 38, 0, 39, 0, 0, 0, 0, 32, 0,
	0, 36, 0, 0, 0, 0, 0, 23, 41, 0,
	0, 0, 0, 0, 37, 28, 35, 0, 0, 0,
	0, 0, 0, 0, 29, 0, 43, 0, 14, 17,
	18, 19, 61, 0, 0, 60, 64, 0, 65, 0,
	62, 195, 66, 0, 67, 0, 0, 0, 0, 76,
	77, 78, 0, 0, 75, 0, 0, 0, 68, 0,
	71, 0, 0, 82, 0, 0, 72, 81, 0, 0,
	0, 69, 70, 0, 63, 79, 80, 61, 73, 74,
	60, 64, 0, 65, 0, 62, 0, 66, 0, 67,
	0, 0, 0, 0, 76, 77, 78, 0, 0, 75,
	0, 0, 0, 68, 0, 71, 0, 0, 82, 212,
	0, 72, 81, 0, 0, 0, 69, 70, 0, 63,
	79, 80, 61, 73, 74, 60, 64, 0, 65, 0,
	62, 0, 66, 0, 67, 0, 0, 0, 0, 76,
	77, 78, 0, 0, 75, 0, 0, 0, 68, 187,
	71, 0, 0, 82, 0, 0, 72, 81, 0, 0,
	0, 69, 70, 0, 63, 79, 80, 61, 73, 74,
	60, 64, 0, 65, 0, 62, 0, 66, 0, 67,
	0, 0, 0, 0, 76, 77, 78, 0, 0, 75,
	0, 0, 0, 68, 0, 71, 0, 0, 82, 0,
	0, 72, 81, 0, 0, 0, 69, 70, 0, 63,
	79, 80, 61, 73, 74, 60, 64, 0, 65, 0,
	62, 0, 66, 0, 67, 0, 0, 0, 0, 76,
	77, 78, 0, 0, 75, 0, 0, 0, 68, 0,
	71, 0, 0, 0, 0, 0, 72, 81, 0, 0,
	0, 69, 70, 0, 63, 79, 80, 61, 73, 74,
	60, 64, 0, 65, 0, 62, 0, 66, 0, 67,
	0, 0, 0, 0, 76, 77, 78, 0, 61, 75,
	0, 60, 64, 68, 65, 71, 62, 0, 0, 0,
	0, 72, 0, 0, 0, 0, 69, 70, 0, 63,
	79, 80, 0, 73, 74, 40, 0, 181, 31, 222,
	30, 42, 0, 0, 0, 0, 0, 38, 0, 39,
	63, 79, 80, 0, 32, 0, 0, 36, 0, 0,
	0, 0, 0, 23, 41, 0, 0, 0, 0, 0,
	37, 28, 35, 61, 0, 182, 60, 64, 0, 65,
	29, 62, 43, 66, 0, 67, 0, 0, 0, 0,
	76, 77, 78, 0, 0, 75, 0, 0, 0, 68,
	0, 71, 0, 0, 0, 0, 0, 72, 0, 0,
	0, 0, 69, 70, 0, 63, 79, 80, 40, 73,
	181, 31, 0, 30, 42, 0, 0, 0, 0, 0,
	38, 0, 39, 0, 0, 0, 0, 32, 0, 0,
	36, 0, 0, 0, 0, 0, 23, 41, 0, 0,
	0, 0, 0, 37, 28, 35, 61, 0, 182, 60,
	64, 0, 65, 29, 62, 43, 66, 0, 67, 0,
	0, 0, 0, 76, 77, 78, 0, 0, 0, 0,
	0, 0, 68, 0, 71, 0, 0, 0, 0, 0,
	72, 0, 0, 0, 0, 69, 70, 0, 63, 79,
	80, 40, 73, 0, 31, 222, 30, 42, 0, 0,
	0, 0, 0, 38, 0, 39, 0, 0, 0, 0,
	32, 40, 0, 36, 31, 0, 30, 42, 0, 23,
	41, 0, 0, 38, 0, 39, 37, 28, 35, 0,
	32, 0, 0, 36, 0, 0, 29, 0, 43, 23,
	41, 0, 0, 0, 0, 0, 37, 28, 35, 61,
	0, 0, 60, 64, 0, 65, 29, 62, 43, 0,
	0, 0, 0, 0, 0, 0, 76, 77, 78, 61,
	0, 0, 60, 64, 0, 65, 0, 62, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 77, 78, 0,
	0, 63, 79, 80, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 63, 79, 80,
}

var yyPact = [...]int16{
	-1000, -1000, 438, -1000, -1000, -1000, -33, -1000, -1000, -1000,
	244, 337, -1000, 162, 1056, 129, -1000, -1000, -1000, -1000,
	-11, 8, 733, -1000, 184, 65, 1056, 228, 82, 1056,
	1056, 1056, 1056, -1000, -1000, 288, 287, 285, 1056, 1056,
	1056, 230, 15, -1000, -1000, -42, 491, 103, 228, -1000,
	1056, 1056, 1056, 255, 18, -1000, 18, 1056, 105, -1000,
	1056, 1056, 1056, 1056, 1056, 1056, 1056, 1056, 1056, 1056,
	1056, 1056, 1056, 4, 1056, 1056, 1056, 1056, 1056, 1056,
	1056, 1056, 1056, 219, 18, 184, 237, 26, 251, 1056,
	240, 262, -1000, 211, 18, 18, -1000, -1000, -1000, -1000,
	251, 18, 953, 182, 688, 251, 99, 161, 260, 598,
	251, 281, 733, 14, -1000, -34, 544, -1000, -1000, -1000,
	1056, 337, 255, 255, 778, 155, -1000, 205, -1000, -1000,
	111, 229, 491, -1000, -1000, -1000, -1000, -1000, 92, 92,
	1095, 1095, 1095, 1095, 1095, 1095, 1095, 1056, 909, 992,
	1115, 844, 113, 145, 145, 823, 643, 18, 258, -1000,
	284, 491, -1000, 277, 251, 953, 256, -1000, 206, 238,
	1056, -1000, 82, 1056, -1000, -1000, -16, 137, 251, 733,
	204, 1056, 1056, -1000, 138, 12, -1000, 337, 1036, -1000,
	134, -1000, 148, 1036, -1000, 1056, 1036, -1000, -1000, -1000,
	-1000, 29, -35, 227, 228, 1056, 18, 19, 83, 491,
	-1000, 1095, 1056, 111, 180, 18, 77, -1000, -1000, -1000,
	870, -1000, -1000, -1000, 1056, -1000, -1000, 733, 251, 870,
	96, 1056, 733, 733, -1000, 12, 1056, 86, 733, -1000,
	-1000, 733, -1000, 598, -1000, -38, -1000, -1000, 491, 255,
	-1000, -1000, 199, -1000, 111, -1000, -1000, 276, -1000, 83,
	-16, -1000, 778, -1000, -1000, 270, 252, -1000, -1000, 216,
	18, 18, -1000, 196, 733, 76, 251, 137, 733, 778,
	1056, 385, -1000, -1000, -1000, 41, -1000, 170, 251, 143,
	251, -1000, 236, -1000, -1000, 77, 18, 234, 207, -1000,
	1056, 250, -1000, -1000, 187, 269, 268, 778, -1000, -1000,
	-1000, -1000, 41, -1000, -1000, 34, 83, -1000, 152, 18,
	18, 166, 267, 2, -16, -1000, -1000, -1000, -1000, 1056,
	111, 111, -1000, -1000, -1000, -1000, 733,
}

var yyPgo = [...]int16{
	0, 5, 25, 7, 14, 365, 363, 26, 360, 359,
	8, 355, 353, 352, 351, 21, 347, 37, 11, 346,
	345, 344, 343, 10, 2, 341, 340, 338, 6, 0,
	4, 56, 22, 228, 337, 111, 18, 336, 20, 19,
	90, 335, 13, 333, 332, 331, 329, 328, 9, 16,
	324, 15, 323, 313, 1, 17, 312, 3, 310, 308,
	303, 302, 295, 294,
}

var yyR1 = [...]int8{
	0, 61, 55, 55, 62, 62, 56, 56, 56, 42,
	42, 42, 42, 43, 43, 59, 60, 60, 44, 44,
	44, 46, 46, 45, 45, 47, 47, 48, 50, 50,
	49, 49, 49, 49, 49, 49, 49, 49, 49, 49,
	49, 13, 14, 15, 15, 16, 16, 63, 63, 32,
	32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
	32, 32, 32, 32, 32, 32, 6, 6, 5, 5,
	4, 4, 4, 4, 58, 58, 57, 57, 9, 9,
	12, 12, 8, 8, 11, 11, 7, 7, 7, 7,
	7, 10, 10, 10, 10, 10, 33, 33, 34, 34,
	29, 29, 29, 29, 29, 29, 29, 29, 29, 29,
	29, 29, 29, 29, 29, 29, 29, 29, 29, 29,
	29, 29, 29, 29, 29, 29, 29, 29, 29, 35,
	35, 30, 30, 31, 31, 1, 1, 2, 2, 3,
	3, 51, 53, 53, 52, 52, 52, 36, 36, 54,
	40, 41, 41, 41, 41, 37, 38, 38, 39, 39,
	17, 17, 18, 18, 20, 20, 19, 21, 22, 22,
	23, 23, 24, 24, 24, 24, 24, 25, 26, 26,
	27, 27, 28,
}

var yyR2 = [...]int8{
	0, 2, 5, 2, 0, 2, 0, 3, 2, 0,
	2, 2, 3, 1, 1, 6, 1, 3, 3, 6,
	1, 4, 5, 1, 4, 2, 1, 4, 0, 3,
	1, 2, 1, 3, 5, 3, 1, 3, 1, 1,
	1, 2, 4, 0, 4, 1, 3, 0, 1, 1,
	1, 1, 3, 8, 7, 7, 4, 4, 6, 8,
	3, 4, 4, 3, 4, 3, 0, 2, 2, 3,
	1, 3, 2, 2, 1, 3, 1, 3, 0, 2,
	0, 2, 1, 3, 1, 3, 1, 3, 2, 1,
	2, 1, 3, 5, 4, 4, 1, 3, 0, 1,
	1, 4, 2, 2, 2, 2, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 4,
	3, 3, 3, 3, 3, 3, 3, 3, 5, 1,
	3, 0, 1, 0, 2, 0, 1, 1, 2, 0,
	1, 3, 1, 3, 0, 1, 2, 1, 3, 1,
	1, 3, 2, 2, 1, 4, 1, 3, 1, 2,
	1, 3, 1, 1, 1, 3, 4, 3, 0, 2,
	1, 3, 1, 1, 1, 2, 1, 3, 0, 2,
	1, 3, 3,
}

var yyChk = [...]int16{
	-1000, -61, -42, 29, -43, 62, 28, -48, -44, -49,
	-60, 31, -45, -33, 54, -40, -14, 55, 56, 57,
	-59, -46, -29, 33, -13, 53, 35, -32, 41, 50,
	10, 8, 24, -41, -54, 42, 27, 40, 17, 19,
	5, 34, 11, 52, 62, -50, 13, -36, -32, -40,
	15, 25, 9, -33, 13, -40, 49, -47, 36, 37,
	7, 4, 12, 46, 8, 10, 14, 16, 30, 43,
	44, 32, 38, 50, 51, 26, 21, 22, 23, 47,
	48, 39, 35, -15, 17, 33, -33, 11, 5, 17,
	-9, -8, -7, -40, 7, 45, -29, -29, -29, -29,
	5, 5, 5, -31, -29, -35, -51, -52, -35, -29,
	-53, -31, -29, 11, 34, -63, 65, -55, 62, -48,
	38, 9, -33, -33, -29, -17, -18, -20, -19, -40,
	-17, -33, 13, 35, -29, -29, -29, -29, -29, -29,
	-29, -29, -29, -29, -29, -29, -29, 38, -29, -29,
	-29, -29, -29, -29, -29, -29, -29, 15, -16, -40,
	-15, 13, 33, -6, -5, -3, -2, 9, -33, -34,
	13, -1, 9, 15, -40, -40, -3, -17, -4, -29,
	-40, 7, 45, 18, -39, -38, -37, 31, -2, -3,
	-39, 20, -1, -2, 9, 13, -2, 6, 34, 62,
	-49, -56, -62, -33, -32, 15, 21, 11, 17, 13,
	-55, -29, 36, -17, -1, 9, 5, -55, 6, -3,
	-2, -4, 9, 18, 13, -33, -7, -29, -54, -2,
	-2, 15, -29, -29, 18, -38, 35, -36, -29, 20,
	20, -29, -51, -29, 58, 28, 62, 62, 13, -33,
	-18, 33, -23, -24, -17, -21, -25, 5, 60, 17,
	19, -55, -29, 18, -40, -12, -11, -10, -7, -40,
	7, 45, -4, -30, -29, -2, -4, -17, -29, -29,
	38, -42, 62, -55, 18, -2, 6, -22, -23, -26,
	-27, -28, -54, 6, -1, 9, 13, -40, -40, 18,
	13, -58, -57, -54, -40, -3, -3, -29, 59, -24,
	18, -3, -2, 20, -3, -2, 13, -10, -17, 13,
	13, -30, -3, 9, 15, 6, 6, -28, -24, 15,
	-17, -17, 18, 6, -57, -54, -29,
}

var yyDef = [...]int16{
	9, -2, 0, 1, 10, 11, 0, 13, 14, 28,
	0, 0, 20, 30, 32, 49, 36, 38, 39, 40,
	16, 23, 96, 150, 43, 0, 0, 100, 78, 0,
	0, 0, 0, 50, 51, 0, 0, 0, 133, 144,
	133, 154, 0, 149, 12, 47, 0, 0, 147, 49,
	0, 0, 0, 31, 0, 41, 0, 0, 0, 26,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 43, 0, 0, -2, 98,
	0, 135, 82, 86, 89, 0, 102, 103, 104, 105,
	139, 0, 0, 0, 129, 139, 142, 0, 135, 129,
	145, 0, 129, 152, 153, 0, 48, 18, 6, 4,
	0, 0, 33, 37, 97, 35, 160, 162, 163, 164,
	17, 0, 0, 25, 106, 107, 108, 109, 110, 111,
	112, 113, 114, 115, 116, 117, 118, 0, 120, 121,
	122, 123, 124, 125, 126, 127, 0, 0, 135, 45,
	0, 0, 52, 0, 139, 0, 140, 137, 99, 0,
	0, 79, 136, 0, 88, 90, 0, 0, 0, 70,
	49, 0, 0, 60, 0, 158, 156, 0, 140, 134,
	0, 63, 0, 0, -2, 0, 146, 65, 151, 27,
	29, 0, 3, 0, 148, 0, 0, 0, 0, 0,
	24, 119, 0, 42, 0, 136, 80, 21, 56, 67,
	140, 68, 138, 57, 131, 101, 83, 87, 0, 0,
	0, 0, 72, 73, 61, 159, 0, 0, 130, 62,
	64, 141, 143, 0, 9, 0, 8, 5, 0, 34,
	161, 165, 0, 170, 172, 173, 174, 0, 176, 168,
	178, 22, 128, 44, 46, 0, 135, 84, 91, 86,
	89, 0, 69, 0, 132, 0, 139, 139, 71, 157,
	0, 0, 7, 19, 166, 0, 175, 0, 139, 0,
	139, 180, 0, 15, 81, 136, 0, 88, 90, 58,
	131, 139, 74, 76, 0, 0, 0, 155, 2, 171,
	167, 169, 140, 177, 179, 140, 0, 85, 92, 0,
	0, 0, 0, 137, 0, 54, 55, 181, 182, 0,
	94, 95, 59, 53, 75, 77, 93,
}

var yyTok1 = [...]int8{
	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	62, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 4, 22, 3,
	5, 6, 7, 8, 9, 10, 11, 12, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 13, 65,
	14, 15, 16, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 17, 3, 18, 23, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 19, 21, 20, 24,
}

var yyTok2 = [...]int8{
	2, 3, 25, 26, 27, 28, 29, 30, 31, 32,
	33, 34, 35, 36, 37, 38, 39, 40, 41, 42,
	43, 44, 45, 46, 47, 48, 49, 50, 51, 52,
	53, 54, 55, 56, 57, 58, 59, 60, 61, 63,
	64,
}

var yyTok3 = [...]int8{
	0,
}

var yyErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	yyDebug        = 0
	yyErrorVerbose = false
)

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

type yyParser interface {
	Parse(yyLexer) int
	Lookahead() int
}

type yyParserImpl struct {
	lval  yySymType
	stack [yyInitialStackSize]yySymType
	char  int
}

func (p *yyParserImpl) Lookahead() int {
	return p.char
}

func yyNewParser() yyParser {
	return &yyParserImpl{}
}

const yyFlag = -1000

func yyTokname(c int) string {
	if c >= 1 && c-1 < len(yyToknames) {
		if yyToknames[c-1] != "" {
			return yyToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yyErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !yyErrorVerbose {
		return "syntax error"
	}

	for _, e := range yyErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + yyTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := int(yyPact[state])
	for tok := TOKSTART; tok-1 < len(yyToknames); tok++ {
		if n := base + tok; n >= 0 && n < yyLast && int(yyChk[int(yyAct[n])]) == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if yyDef[state] == -2 {
		i := 0
		for yyExca[i] != -1 || int(yyExca[i+1]) != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; yyExca[i] >= 0; i += 2 {
			tok := int(yyExca[i])
			if tok < TOKSTART || yyExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if yyExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += yyTokname(tok)
	}
	return res
}

func yylex1(lex yyLexer, lval *yySymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = int(yyTok1[0])
		goto out
	}
	if char < len(yyTok1) {
		token = int(yyTok1[char])
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			token = int(yyTok2[char-yyPrivate])
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		token = int(yyTok3[i+0])
		if token == char {
			token = int(yyTok3[i+1])
			goto out
		}
	}

out:
	if token == 0 {
		token = int(yyTok2[1]) /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(token), uint(char))
	}
	return char, token
}

func yyParse(yylex yyLexer) int {
	return yyNewParser().Parse(yylex)
}

func (yyrcvr *yyParserImpl) Parse(yylex yyLexer) int {
	var yyn int
	var yyVAL yySymType
	var yyDollar []yySymType
	_ = yyDollar // silence set and not used
	yyS := yyrcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yyrcvr.char = -1
	yytoken := -1 // yyrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		yystate = -1
		yyrcvr.char = -1
		yytoken = -1
	}()
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yytoken), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = int(yyPact[yystate])
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yyrcvr.char < 0 {
		yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
	}
	yyn += yytoken
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = int(yyAct[yyn])
	if int(yyChk[yyn]) == yytoken { /* valid shift */
		yyrcvr.char = -1
		yytoken = -1
		yyVAL = yyrcvr.lval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = int(yyDef[yystate])
	if yyn == -2 {
		if yyrcvr.char < 0 {
			yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && int(yyExca[xi+1]) == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = int(yyExca[xi+0])
			if yyn < 0 || yyn == yytoken {
				break
			}
		}
		yyn = int(yyExca[xi+1])
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error(yyErrorMessage(yystate, yytoken))
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yytoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = int(yyPact[yyS[yyp].yys]) + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = int(yyAct[yyn]) /* simulate a shift of "error" */
					if int(yyChk[yystate]) == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yytoken))
			}
			if yytoken == yyEofCode {
				goto ret1
			}
			yyrcvr.char = -1
			yytoken = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= int(yyR2[yyn])
	// yyp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if yyp+1 >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = int(yyR1[yyn])
	yyg := int(yyPgo[yyn])
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = int(yyAct[yyg])
	} else {
		yystate = int(yyAct[yyj])
		if int(yyChk[yystate]) != -yyn {
			yystate = int(yyAct[yyg])
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:237
		{
			yylex.(*input).file = &File{Stmt: yyDollar[1].exprs}
			return 0
		}
	case 2:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:244
		{
			statements := yyDollar[4].exprs
			if yyDollar[2].exprs != nil {
				// $2 can only contain *CommentBlock objects, each of them contains a non-empty After slice
				cb := yyDollar[2].exprs[len(yyDollar[2].exprs)-1].(*CommentBlock)
				// $4 can't be empty and can't start with a comment
				stmt := yyDollar[4].exprs[0]
				start, _ := stmt.Span()
				if start.Line-cb.After[len(cb.After)-1].Start.Line == 1 {
					// The first statement of $4 starts on the next line after the last comment of $2.
					// Attach the last comment to the first statement
					stmt.Comment().Before = cb.After
					yyDollar[2].exprs = yyDollar[2].exprs[:len(yyDollar[2].exprs)-1]
				}
				statements = append(yyDollar[2].exprs, yyDollar[4].exprs...)
			}
			yyVAL.exprs = statements
			yyVAL.lastStmt = yyDollar[4].lastStmt
		}
	case 3:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:264
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 6:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:272
		{
			yyVAL.exprs = nil
			yyVAL.lastStmt = nil
		}
	case 7:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:277
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = yyDollar[1].lastStmt
			if yyVAL.lastStmt == nil {
				cb := &CommentBlock{Start: yyDollar[2].pos}
				yyVAL.exprs = append(yyVAL.exprs, cb)
				yyVAL.lastStmt = cb
			}
			com := yyVAL.lastStmt.Comment()
			com.After = append(com.After, Comment{Start: yyDollar[2].pos, Token: yyDollar[2].tok})
		}
	case 8:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:289
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = nil
		}
	case 9:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:295
		{
			yyVAL.exprs = nil
			yyVAL.lastStmt = nil
		}
	case 10:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:300
		{
			// If this statement follows a comment block,
			// attach the comments to the statement.
			if cb, ok := yyDollar[1].lastStmt.(*CommentBlock); ok {
				yyVAL.exprs = append(yyDollar[1].exprs[:len(yyDollar[1].exprs)-1], yyDollar[2].exprs...)
				yyDollar[2].exprs[0].Comment().Before = cb.After
				yyVAL.lastStmt = yyDollar[2].lastStmt
				break
			}

			// Otherwise add to list.
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[2].exprs...)
			yyVAL.lastStmt = yyDollar[2].lastStmt

			// Consider this input:
			//
			//	foo()
			//	# bar
			//	baz()
			//
			// If we've just parsed baz(), the # bar is attached to
			// foo() as an After comment. Make it a Before comment
			// for baz() instead.
			if x := yyDollar[1].lastStmt; x != nil {
				com := x.Comment()
				// stmt is never empty
				yyDollar[2].exprs[0].Comment().Before = com.After
				com.After = nil
			}
		}
	case 11:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:331
		{
			// Blank line; sever last rule from future comments.
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = nil
		}
	case 12:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:337
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = yyDollar[1].lastStmt
			if yyVAL.lastStmt == nil || isBlockStmt(yyVAL.lastStmt) {
				// Comments after a block statement (e.g. a compact `def f(): pass`)
				// must become a standalone CommentBlock, matching how the same
				// comment is parsed when the block is written in its expanded,
				// indented form (see extractTrailingComments). Attaching it to the
				// block's After list instead makes the printer emit it without the
				// blank line that separates a block from a trailing comment, so
				// formatting would need a second pass to become stable.
				cb := &CommentBlock{Start: yyDollar[2].pos}
				yyVAL.exprs = append(yyVAL.exprs, cb)
				yyVAL.lastStmt = cb
			}
			com := yyVAL.lastStmt.Comment()
			com.After = append(com.After, Comment{Start: yyDollar[2].pos, Token: yyDollar[2].tok})
		}
	case 13:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:358
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = yyDollar[1].exprs[len(yyDollar[1].exprs)-1]
		}
	case 14:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:363
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
			yyVAL.lastStmt = yyDollar[1].expr
			if cbs := extractTrailingComments(yyDollar[1].expr); len(cbs) > 0 {
				yyVAL.exprs = append(yyVAL.exprs, cbs...)
				yyVAL.lastStmt = cbs[len(cbs)-1]
				if yyDollar[1].lastStmt == nil {
					yyVAL.lastStmt = nil
				}
			}
		}
	case 15:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:377
		{
			yyVAL.def_header = &DefStmt{
				Function: Function{
					StartPos: yyDollar[1].pos,
					Params:   yyDollar[5].exprs,
				},
				Name:           yyDollar[2].tok,
				TypeParams:     yyDollar[3].expr,
				ParamsEnd:      &End{Pos: yyDollar[6].pos},
				ForceCompact:   forceCompact(yyDollar[4].pos, yyDollar[5].exprs, yyDollar[6].pos),
				ForceMultiLine: forceMultiLine(yyDollar[4].pos, yyDollar[5].exprs, yyDollar[6].pos),
			}
		}
	case 17:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:394
		{
			yyDollar[1].def_header.Type = yyDollar[3].expr
			yyVAL.def_header = yyDollar[1].def_header
		}
	case 18:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:401
		{
			yyDollar[1].def_header.Function.Body = yyDollar[3].exprs
			yyDollar[1].def_header.ColonPos = &End{Pos: yyDollar[2].pos}
			yyVAL.expr = yyDollar[1].def_header
			yyVAL.lastStmt = yyDollar[3].lastStmt
		}
	case 19:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:408
		{
			yyVAL.expr = &ForStmt{
				For:  yyDollar[1].pos,
				Vars: yyDollar[2].expr,
				X:    yyDollar[4].expr,
				Body: yyDollar[6].exprs,
			}
			yyVAL.lastStmt = yyDollar[6].lastStmt
		}
	case 20:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:418
		{
			yyVAL.expr = yyDollar[1].ifstmt
			yyVAL.lastStmt = yyDollar[1].lastStmt
		}
	case 21:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:426
		{
			yyVAL.ifstmt = &IfStmt{
				If:   yyDollar[1].pos,
				Cond: yyDollar[2].expr,
				True: yyDollar[4].exprs,
			}
			yyVAL.lastStmt = yyDollar[4].lastStmt
		}
	case 22:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:435
		{
			yyVAL.ifstmt = yyDollar[1].ifstmt
			inner := yyDollar[1].ifstmt
			for len(inner.False) == 1 {
				inner = inner.False[0].(*IfStmt)
			}
			inner.ElsePos = End{Pos: yyDollar[2].pos}
			inner.False = []Expr{
				&IfStmt{
					If:   yyDollar[2].pos,
					Cond: yyDollar[3].expr,
					True: yyDollar[5].exprs,
				},
			}
			yyVAL.lastStmt = yyDollar[5].lastStmt
		}
	case 24:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:456
		{
			yyVAL.ifstmt = yyDollar[1].ifstmt
			inner := yyDollar[1].ifstmt
			for len(inner.False) == 1 {
				inner = inner.False[0].(*IfStmt)
			}
			inner.ElsePos = End{Pos: yyDollar[2].pos}
			inner.False = yyDollar[4].exprs
			yyVAL.lastStmt = yyDollar[4].lastStmt
		}
	case 27:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:473
		{
			yyVAL.exprs = append([]Expr{yyDollar[1].expr}, yyDollar[2].exprs...)
			yyVAL.lastStmt = yyVAL.exprs[len(yyVAL.exprs)-1]
		}
	case 28:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:479
		{
			yyVAL.exprs = []Expr{}
		}
	case 29:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:483
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 31:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:490
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
				Result: yyDollar[2].expr,
			}
		}
	case 32:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:497
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
			}
		}
	case 33:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:502
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 34:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:503
		{
			yyVAL.expr = binary(typed(yyDollar[1].expr, yyDollar[3].expr), yyDollar[4].pos, yyDollar[4].tok, yyDollar[5].expr)
		}
	case 35:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:504
		{
			yyVAL.expr = typed(yyDollar[1].expr, yyDollar[3].expr)
		}
	case 37:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:506
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 38:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:508
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 39:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:515
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 40:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:522
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 41:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:533
		{
			if yyDollar[1].expr.(*Ident).Name != "type" {
				// two idents can be adjacent only if the first one is `type`.
				_, end := yyDollar[2].expr.Span()
				errorAt(yylex, end, "syntax error near "+yyDollar[2].expr.(*Ident).Name)
			}
			yyVAL.expr = &TypeAliasStmt{
				TypePos: yyDollar[1].expr.(*Ident).NamePos,
				Name:    yyDollar[2].expr,
				// Rest of fields will be filled in by type_alias_stmt
			}
		}
	case 42:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:548
		{
			typeStart, _ := yyDollar[4].expr.Span()
			// Modify $1 in-place to fill in the remaining fields.
			typeAlisStmt := yyDollar[1].expr.(*TypeAliasStmt)
			typeAlisStmt.TypeParams = yyDollar[2].expr
			typeAlisStmt.EqualPos = yyDollar[3].pos
			typeAlisStmt.Type = yyDollar[4].expr
			typeAlisStmt.LineBreak = yyDollar[3].pos.Line < typeStart.Line
			yyVAL.expr = typeAlisStmt
		}
	case 43:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:560
		{
			yyVAL.expr = nil
		}
	case 44:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:564
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[4].pos),
			}
		}
	case 45:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:575
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 46:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:579
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 51:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:590
		{
			yyVAL.expr = yyDollar[1].string
		}
	case 52:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:594
		{
			yyVAL.expr = &DotExpr{
				X:       yyDollar[1].expr,
				Dot:     yyDollar[2].pos,
				NamePos: yyDollar[3].pos,
				Name:    yyDollar[3].tok,
			}
		}
	case 53:
		yyDollar = yyS[yypt-8 : yypt+1]
//line build/parse.y:603
		{
			load := &LoadStmt{
				Load:         yyDollar[1].pos,
				Module:       yyDollar[4].string,
				Rparen:       End{Pos: yyDollar[8].pos},
				ForceCompact: yyDollar[2].pos.Line == yyDollar[8].pos.Line,
			}
			for _, arg := range yyDollar[6].loadargs {
				load.From = append(load.From, &arg.from)
				load.To = append(load.To, &arg.to)
			}
			yyVAL.expr = load
		}
	case 54:
		yyDollar = yyS[yypt-7 : yypt+1]
//line build/parse.y:617
		{
			args := []Expr{yyDollar[3].expr, yyDollar[5].expr}
			yyVAL.expr = &CallExpr{
				CallExprKind:   CallExprCast,
				X:              &Ident{NamePos: yyDollar[1].pos, Name: yyDollar[1].tok},
				ListStart:      yyDollar[2].pos,
				List:           args,
				End:            End{Pos: yyDollar[7].pos},
				ForceCompact:   forceCompact(yyDollar[2].pos, args, yyDollar[7].pos),
				ForceMultiLine: forceMultiLine(yyDollar[2].pos, args, yyDollar[7].pos),
			}
		}
	case 55:
		yyDollar = yyS[yypt-7 : yypt+1]
//line build/parse.y:630
		{
			args := []Expr{yyDollar[3].expr, yyDollar[5].expr}
			yyVAL.expr = &CallExpr{
				CallExprKind:   CallExprIsInstance,
				X:              &Ident{NamePos: yyDollar[1].pos, Name: yyDollar[1].tok},
				ListStart:      yyDollar[2].pos,
				List:           args,
				End:            End{Pos: yyDollar[7].pos},
				ForceCompact:   forceCompact(yyDollar[2].pos, args, yyDollar[7].pos),
				ForceMultiLine: forceMultiLine(yyDollar[2].pos, args, yyDollar[7].pos),
			}
		}
	case 56:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:643
		{
			yyVAL.expr = &CallExpr{
				X:              yyDollar[1].expr,
				ListStart:      yyDollar[2].pos,
				List:           yyDollar[3].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceCompact:   forceCompact(yyDollar[2].pos, yyDollar[3].exprs, yyDollar[4].pos),
				ForceMultiLine: forceMultiLine(yyDollar[2].pos, yyDollar[3].exprs, yyDollar[4].pos),
			}
		}
	case 57:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:654
		{
			yyVAL.expr = &IndexExpr{
				X:          yyDollar[1].expr,
				IndexStart: yyDollar[2].pos,
				Y:          yyDollar[3].expr,
				End:        yyDollar[4].pos,
			}
		}
	case 58:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:663
		{
			yyVAL.expr = &SliceExpr{
				X:          yyDollar[1].expr,
				SliceStart: yyDollar[2].pos,
				From:       yyDollar[3].expr,
				FirstColon: yyDollar[4].pos,
				To:         yyDollar[5].expr,
				End:        yyDollar[6].pos,
			}
		}
	case 59:
		yyDollar = yyS[yypt-8 : yypt+1]
//line build/parse.y:674
		{
			yyVAL.expr = &SliceExpr{
				X:           yyDollar[1].expr,
				SliceStart:  yyDollar[2].pos,
				From:        yyDollar[3].expr,
				FirstColon:  yyDollar[4].pos,
				To:          yyDollar[5].expr,
				SecondColon: yyDollar[6].pos,
				Step:        yyDollar[7].expr,
				End:         yyDollar[8].pos,
			}
		}
	case 60:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:687
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 61:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:696
		{
			yyVAL.expr = &Comprehension{
				Curly:          false,
				Lbrack:         yyDollar[1].pos,
				Body:           yyDollar[2].expr,
				Clauses:        yyDollar[3].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLineComprehension(yyDollar[1].pos, yyDollar[2].expr, yyDollar[3].exprs, yyDollar[4].pos),
			}
		}
	case 62:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:707
		{
			yyVAL.expr = &Comprehension{
				Curly:          true,
				Lbrack:         yyDollar[1].pos,
				Body:           yyDollar[2].kv,
				Clauses:        yyDollar[3].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLineComprehension(yyDollar[1].pos, yyDollar[2].kv, yyDollar[3].exprs, yyDollar[4].pos),
			}
		}
	case 63:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:718
		{
			exprValues := make([]Expr, 0, len(yyDollar[2].kvs))
			for _, kv := range yyDollar[2].kvs {
				exprValues = append(exprValues, Expr(kv))
			}
			yyVAL.expr = &DictExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].kvs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, exprValues, yyDollar[3].pos),
			}
		}
	case 64:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:731
		{
			yyVAL.expr = &SetExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[4].pos),
			}
		}
	case 65:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:740
		{
			if len(yyDollar[2].exprs) == 1 && yyDollar[2].comma.Line == 0 {
				// Just a parenthesized expression, not a tuple.
				yyVAL.expr = &ParenExpr{
					Start:          yyDollar[1].pos,
					X:              yyDollar[2].exprs[0],
					End:            End{Pos: yyDollar[3].pos},
					ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
				}
			} else {
				yyVAL.expr = &TupleExpr{
					Start:          yyDollar[1].pos,
					List:           yyDollar[2].exprs,
					End:            End{Pos: yyDollar[3].pos},
					ForceCompact:   forceCompact(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
					ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
				}
			}
		}
	case 66:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:761
		{
			yyVAL.exprs = nil
		}
	case 67:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:765
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 68:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:771
		{
			yyVAL.exprs = []Expr{yyDollar[2].expr}
		}
	case 69:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:775
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 71:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:782
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 72:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:786
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 73:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:790
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 74:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:795
		{
			yyVAL.loadargs = []*struct {
				from Ident
				to   Ident
			}{yyDollar[1].loadarg}
		}
	case 75:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:799
		{
			yyDollar[1].loadargs = append(yyDollar[1].loadargs, yyDollar[3].loadarg)
			yyVAL.loadargs = yyDollar[1].loadargs
		}
	case 76:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:805
		{
			start := yyDollar[1].string.Start.add("'")
			if yyDollar[1].string.TripleQuote {
				start = start.add("''")
			}
			yyVAL.loadarg = &struct {
				from Ident
				to   Ident
			}{
				from: Ident{
					Name:    yyDollar[1].string.Value,
					NamePos: start,
				},
				to: Ident{
					Name:    yyDollar[1].string.Value,
					NamePos: start,
				},
			}
		}
	case 77:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:822
		{
			start := yyDollar[3].string.Start.add("'")
			if yyDollar[3].string.TripleQuote {
				start = start.add("''")
			}
			yyVAL.loadarg = &struct {
				from Ident
				to   Ident
			}{
				from: Ident{
					Name:    yyDollar[3].string.Value,
					NamePos: start,
				},
				to: *yyDollar[1].expr.(*Ident),
			}
		}
	case 78:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:837
		{
			yyVAL.exprs = nil
		}
	case 79:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:841
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 80:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:846
		{
			yyVAL.exprs = nil
		}
	case 81:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:850
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 82:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:856
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 83:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:860
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 84:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:867
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 85:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:871
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 87:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:878
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 88:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:882
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 89:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:886
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, nil)
		}
	case 90:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:890
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 92:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:899
		{
			yyVAL.expr = typed(yyDollar[1].expr, yyDollar[3].expr)
		}
	case 93:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:903
		{
			yyVAL.expr = binary(typed(yyDollar[1].expr, yyDollar[3].expr), yyDollar[4].pos, yyDollar[4].tok, yyDollar[5].expr)
		}
	case 94:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:907
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, typed(yyDollar[2].expr, yyDollar[4].expr))
		}
	case 95:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:911
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, typed(yyDollar[2].expr, yyDollar[4].expr))
		}
	case 97:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:918
		{
			tuple, ok := yyDollar[1].expr.(*TupleExpr)
			if !ok || !tuple.NoBrackets {
				tuple = &TupleExpr{
					List:           []Expr{yyDollar[1].expr},
					NoBrackets:     true,
					ForceCompact:   true,
					ForceMultiLine: false,
				}
			}
			tuple.List = append(tuple.List, yyDollar[3].expr)
			yyVAL.expr = tuple
		}
	case 98:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:933
		{
			yyVAL.expr = nil
		}
	case 101:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:941
		{
			yyVAL.expr = &LambdaExpr{
				Function: Function{
					StartPos: yyDollar[1].pos,
					Params:   yyDollar[2].exprs,
					Body:     []Expr{yyDollar[4].expr},
				},
			}
		}
	case 102:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:950
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 103:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:951
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 104:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:952
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 105:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:953
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 106:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:954
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 107:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:955
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 108:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:956
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 109:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:957
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 110:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:958
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 111:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:959
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 112:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:960
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 113:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:961
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 114:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:962
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 115:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:963
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 116:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:964
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 117:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:965
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 118:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:966
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 119:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:967
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "not in", yyDollar[4].expr)
		}
	case 120:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:968
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 121:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:969
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 122:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:970
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 123:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:971
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 124:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:972
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 125:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:973
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 126:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:974
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 127:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:976
		{
			if b, ok := yyDollar[3].expr.(*UnaryExpr); ok && b.Op == "not" {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "is not", b.X)
			} else {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
			}
		}
	case 128:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:984
		{
			yyVAL.expr = &ConditionalExpr{
				Then:      yyDollar[1].expr,
				IfStart:   yyDollar[2].pos,
				Test:      yyDollar[3].expr,
				ElseStart: yyDollar[4].pos,
				Else:      yyDollar[5].expr,
			}
		}
	case 129:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:996
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 130:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1000
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 131:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1005
		{
			yyVAL.expr = nil
		}
	case 133:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1011
		{
			yyVAL.exprs, yyVAL.comma = nil, Position{}
		}
	case 134:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1015
		{
			yyVAL.exprs, yyVAL.comma = yyDollar[1].exprs, yyDollar[2].pos
		}
	case 135:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1025
		{
			yyVAL.pos = Position{}
		}
	case 138:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1036
		{
			yyVAL.pos = yyDollar[1].pos
		}
	case 139:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1044
		{
			yyVAL.pos = Position{}
		}
	case 141:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1051
		{
			yyVAL.kv = &KeyValueExpr{
				Key:   yyDollar[1].expr,
				Colon: yyDollar[2].pos,
				Value: yyDollar[3].expr,
			}
		}
	case 142:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1061
		{
			yyVAL.kvs = []*KeyValueExpr{yyDollar[1].kv}
		}
	case 143:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1065
		{
			yyVAL.kvs = append(yyDollar[1].kvs, yyDollar[3].kv)
		}
	case 144:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1070
		{
			yyVAL.kvs = nil
		}
	case 145:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1074
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 146:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1078
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 148:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1085
		{
			tuple, ok := yyDollar[1].expr.(*TupleExpr)
			if !ok || !tuple.NoBrackets {
				tuple = &TupleExpr{
					List:           []Expr{yyDollar[1].expr},
					NoBrackets:     true,
					ForceCompact:   true,
					ForceMultiLine: false,
				}
			}
			tuple.List = append(tuple.List, yyDollar[3].expr)
			yyVAL.expr = tuple
		}
	case 149:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1101
		{
			yyVAL.string = &StringExpr{
				Start:       yyDollar[1].pos,
				Value:       yyDollar[1].str,
				TripleQuote: yyDollar[1].triple,
				End:         yyDollar[1].pos.add(yyDollar[1].tok),
				Token:       yyDollar[1].tok,
			}
		}
	case 150:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1113
		{
			yyVAL.expr = &Ident{NamePos: yyDollar[1].pos, Name: yyDollar[1].tok}
		}
	case 151:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1119
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok + "." + yyDollar[3].tok}
		}
	case 152:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1123
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok + "."}
		}
	case 153:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1127
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: "." + yyDollar[2].tok}
		}
	case 154:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1131
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok}
		}
	case 155:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:1137
		{
			yyVAL.expr = &ForClause{
				For:  yyDollar[1].pos,
				Vars: yyDollar[2].expr,
				In:   yyDollar[3].pos,
				X:    yyDollar[4].expr,
			}
		}
	case 156:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1148
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 157:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1152
		{
			yyVAL.exprs = append(yyDollar[1].exprs, &IfClause{
				If:   yyDollar[2].pos,
				Cond: yyDollar[3].expr,
			})
		}
	case 158:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1161
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 159:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1165
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[2].exprs...)
		}
	case 160:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1171
		{
			yyVAL.expr = yyDollar[1].expr
		}
	case 161:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1175
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 165:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1186
		{
			yyVAL.expr = &DotExpr{
				X:       yyDollar[1].expr,
				Dot:     yyDollar[2].pos,
				NamePos: yyDollar[3].pos,
				Name:    yyDollar[3].tok,
			}
		}
	case 166:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:1197
		{
			yyVAL.expr = &TypeAppExpr{
				Type:           yyDollar[1].expr,
				ArgsStart:      yyDollar[2].pos,
				Args:           yyDollar[3].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceCompact:   forceCompact(yyDollar[2].pos, yyDollar[3].exprs, yyDollar[4].pos),
				ForceMultiLine: forceMultiLine(yyDollar[2].pos, yyDollar[3].exprs, yyDollar[4].pos),
			}
		}
	case 167:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1210
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 168:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1220
		{
			yyVAL.exprs = nil
		}
	case 169:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1224
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 170:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1230
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 171:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1234
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 175:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1243
		{
			yyVAL.expr = &TupleExpr{
				Start: yyDollar[1].pos,
				End:   End{Pos: yyDollar[2].pos},
			}
		}
	case 176:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1250
		{
			yyVAL.expr = &EllipsisExpr{
				Pos: yyDollar[1].pos,
			}
		}
	case 177:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1258
		{
			exprValues := make([]Expr, 0, len(yyDollar[2].kvs))
			for _, kv := range yyDollar[2].kvs {
				exprValues = append(exprValues, Expr(kv))
			}
			yyVAL.expr = &DictExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].kvs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, exprValues, yyDollar[3].pos),
			}
		}
	case 178:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1272
		{
			yyVAL.kvs = nil
		}
	case 179:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1276
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 180:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1282
		{
			yyVAL.kvs = []*KeyValueExpr{yyDollar[1].kv}
		}
	case 181:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1286
		{
			yyVAL.kvs = append(yyDollar[1].kvs, yyDollar[3].kv)
		}
	case 182:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1292
		{
			yyVAL.kv = &KeyValueExpr{
				Key:   yyDollar[1].string,
				Colon: yyDollar[2].pos,
				Value: yyDollar[3].expr,
			}
		}
	}
	goto yystack /* stack new state and value */
}
