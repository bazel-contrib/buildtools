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

//line build/parse.y:1340

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
	-1, 90,
	6, 67,
	-2, 140,
	-1, 197,
	20, 137,
	-2, 138,
}

const yyPrivate = 57344

const yyLast = 1186

var yyAct = [...]int16{
	22, 310, 35, 259, 281, 298, 275, 258, 168, 121,
	2, 128, 7, 9, 119, 181, 174, 108, 276, 49,
	187, 188, 34, 118, 251, 27, 85, 290, 253, 202,
	98, 99, 100, 101, 131, 46, 45, 50, 228, 58,
	106, 111, 114, 260, 169, 150, 264, 94, 265, 228,
	242, 105, 201, 63, 250, 126, 62, 66, 252, 67,
	278, 64, 23, 137, 138, 139, 140, 141, 142, 143,
	144, 145, 146, 147, 148, 149, 131, 151, 152, 153,
	154, 155, 156, 157, 158, 159, 23, 23, 264, 44,
	265, 228, 45, 113, 123, 65, 81, 82, 279, 107,
	127, 123, 133, 116, 23, 182, 45, 96, 56, 51,
	23, 179, 15, 131, 163, 23, 192, 135, 63, 60,
	61, 62, 66, 288, 67, 57, 64, 195, 23, 193,
	122, 44, 203, 23, 45, 78, 79, 80, 95, 136,
	110, 23, 131, 257, 165, 97, 228, 180, 87, 207,
	216, 217, 191, 190, 245, 196, 63, 199, 15, 62,
	65, 81, 82, 209, 64, 190, 132, 240, 132, 182,
	23, 321, 170, 246, 194, 214, 225, 233, 220, 223,
	190, 292, 234, 337, 227, 238, 239, 63, 340, 209,
	62, 66, 244, 67, 232, 64, 162, 244, 65, 247,
	249, 86, 54, 318, 219, 271, 177, 178, 52, 241,
	243, 170, 226, 132, 183, 241, 50, 248, 53, 270,
	332, 256, 267, 209, 186, 235, 236, 182, 15, 65,
	269, 282, 308, 208, 263, 54, 182, 307, 286, 209,
	90, 132, 280, 287, 229, 15, 89, 210, 237, 176,
	328, 284, 91, 211, 304, 160, 176, 327, 268, 54,
	54, 289, 54, 254, 215, 324, 164, 230, 299, 291,
	132, 173, 295, 48, 15, 115, 331, 170, 183, 283,
	285, 303, 54, 228, 221, 95, 311, 263, 197, 315,
	175, 302, 341, 313, 314, 334, 333, 317, 301, 7,
	13, 266, 224, 293, 319, 200, 322, 222, 104, 282,
	325, 103, 300, 329, 102, 55, 263, 117, 330, 132,
	205, 132, 1, 10, 317, 15, 299, 88, 336, 335,
	20, 272, 277, 342, 311, 343, 183, 309, 344, 204,
	320, 112, 323, 263, 109, 183, 132, 263, 326, 47,
	59, 21, 12, 124, 125, 8, 4, 33, 189, 172,
	134, 297, 63, 296, 15, 62, 66, 262, 67, 294,
	64, 338, 339, 41, 132, 261, 129, 130, 132, 43,
	79, 80, 213, 212, 161, 39, 16, 40, 305, 306,
	24, 273, 171, 312, 274, 37, 92, 93, 166, 15,
	167, 23, 42, 132, 65, 81, 82, 0, 38, 0,
	36, 0, 0, 277, 132, 0, 0, 0, 0, 0,
	45, 41, 0, 206, 31, 0, 30, 43, 44, 0,
	132, 0, 0, 39, 132, 40, 0, 132, 132, 0,
	32, 312, 0, 37, 6, 0, 0, 11, 0, 23,
	42, 26, 0, 0, 0, 0, 38, 28, 36, 0,
	0, 0, 0, 0, 0, 0, 29, 0, 45, 25,
	14, 17, 18, 19, 231, 316, 44, 41, 5, 0,
	31, 0, 30, 43, 0, 0, 0, 0, 0, 39,
	0, 40, 0, 0, 0, 0, 32, 0, 0, 37,
	6, 3, 0, 11, 0, 23, 42, 26, 0, 255,
	0, 0, 38, 28, 36, 0, 0, 0, 0, 0,
	0, 0, 29, 0, 45, 25, 14, 17, 18, 19,
	41, 0, 44, 31, 5, 30, 43, 0, 0, 0,
	0, 0, 39, 0, 40, 0, 0, 0, 0, 32,
	0, 0, 37, 0, 0, 0, 0, 0, 23, 42,
	0, 0, 0, 0, 0, 38, 28, 36, 0, 0,
	0, 0, 0, 0, 0, 29, 0, 45, 0, 14,
	17, 18, 19, 41, 0, 44, 31, 120, 30, 43,
	0, 0, 0, 0, 0, 39, 0, 40, 0, 0,
	0, 0, 32, 0, 0, 37, 0, 0, 0, 0,
	0, 23, 42, 0, 0, 0, 0, 0, 38, 28,
	36, 0, 0, 0, 0, 0, 0, 0, 29, 0,
	45, 0, 14, 17, 18, 19, 0, 41, 44, 184,
	31, 228, 30, 43, 0, 0, 0, 0, 0, 39,
	0, 40, 0, 0, 0, 0, 32, 0, 0, 37,
	0, 0, 0, 0, 0, 23, 42, 0, 0, 0,
	0, 0, 38, 28, 36, 0, 0, 185, 0, 0,
	0, 0, 29, 41, 45, 184, 31, 0, 30, 43,
	0, 0, 44, 0, 0, 39, 0, 40, 0, 0,
	0, 0, 32, 0, 0, 37, 0, 0, 0, 0,
	0, 23, 42, 0, 0, 0, 0, 0, 38, 28,
	36, 41, 0, 185, 31, 228, 30, 43, 29, 0,
	45, 0, 0, 39, 0, 40, 0, 0, 44, 0,
	32, 0, 0, 37, 0, 0, 0, 0, 0, 23,
	42, 0, 0, 0, 0, 0, 38, 28, 36, 41,
	0, 0, 31, 0, 30, 43, 29, 0, 45, 0,
	0, 39, 0, 40, 0, 0, 44, 0, 32, 0,
	0, 37, 0, 0, 0, 0, 0, 23, 42, 0,
	0, 0, 0, 0, 38, 28, 36, 0, 0, 63,
	0, 0, 62, 66, 29, 67, 45, 64, 198, 68,
	0, 69, 0, 0, 44, 0, 78, 79, 80, 0,
	0, 77, 0, 0, 0, 70, 0, 73, 0, 0,
	84, 0, 0, 74, 83, 0, 0, 0, 71, 72,
	0, 65, 81, 82, 63, 75, 76, 62, 66, 0,
	67, 0, 64, 0, 68, 0, 69, 0, 0, 0,
	0, 78, 79, 80, 0, 0, 77, 0, 0, 0,
	70, 0, 73, 0, 0, 84, 218, 0, 74, 83,
	0, 0, 0, 71, 72, 0, 65, 81, 82, 63,
	75, 76, 62, 66, 0, 67, 0, 64, 0, 68,
	0, 69, 0, 0, 0, 0, 78, 79, 80, 0,
	0, 77, 0, 0, 0, 70, 190, 73, 0, 0,
	84, 0, 0, 74, 83, 0, 0, 0, 71, 72,
	0, 65, 81, 82, 63, 75, 76, 62, 66, 0,
	67, 0, 64, 0, 68, 0, 69, 0, 0, 0,
	0, 78, 79, 80, 0, 0, 77, 0, 0, 0,
	70, 0, 73, 0, 0, 84, 0, 0, 74, 83,
	0, 0, 0, 71, 72, 0, 65, 81, 82, 63,
	75, 76, 62, 66, 0, 67, 0, 64, 0, 68,
	0, 69, 0, 0, 0, 0, 78, 79, 80, 0,
	0, 77, 0, 0, 0, 70, 0, 73, 0, 0,
	0, 0, 0, 74, 83, 0, 0, 0, 71, 72,
	0, 65, 81, 82, 63, 75, 76, 62, 66, 0,
	67, 0, 64, 0, 68, 0, 69, 0, 0, 0,
	0, 78, 79, 80, 0, 0, 77, 0, 0, 0,
	70, 0, 73, 0, 0, 0, 0, 0, 74, 0,
	0, 0, 0, 71, 72, 0, 65, 81, 82, 63,
	75, 76, 62, 66, 0, 67, 0, 64, 0, 68,
	0, 69, 0, 0, 0, 0, 78, 79, 80, 0,
	0, 77, 0, 0, 0, 70, 0, 73, 0, 0,
	0, 0, 0, 74, 0, 0, 0, 0, 71, 72,
	0, 65, 81, 82, 63, 75, 0, 62, 66, 0,
	67, 0, 64, 0, 68, 0, 69, 0, 0, 0,
	0, 78, 79, 80, 0, 0, 0, 0, 0, 0,
	70, 63, 73, 0, 62, 66, 0, 67, 74, 64,
	0, 0, 0, 71, 72, 0, 65, 81, 82, 79,
	75, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 65, 81, 82,
}

var yyPact = [...]int16{
	-32768, -32768, 472, -32768, -32768, -32768, -27, -32768, -32768, -32768,
	260, 368, -32768, 193, 754, 95, -32768, -32768, -32768, -32768,
	-10, 83, 930, -32768, 184, 115, 754, 235, 100, 754,
	754, 754, 754, -32768, -32768, -32768, 309, 306, 303, 754,
	754, 754, 264, 69, -32768, -32768, -32768, -42, 525, 92,
	235, -32768, 754, 754, 754, 273, 108, -32768, 108, 754,
	104, -32768, 754, 754, 754, 754, 754, 754, 754, 754,
	754, 754, 754, 754, 754, 7, 754, 754, 754, 754,
	754, 754, 754, 754, 754, 240, 77, 184, 253, 111,
	268, 754, 258, 281, -32768, 234, 77, 77, -32768, -32768,
	-32768, -32768, 268, 108, 678, 206, 885, 268, 122, 154,
	279, 795, 268, 299, 930, 18, -32768, -33, 578, -32768,
	-32768, -32768, 754, 368, 273, 273, 975, 218, -32768, 236,
	-32768, 108, -32768, 142, 251, 525, -32768, -32768, -32768, -32768,
	-32768, 152, 152, 114, 114, 114, 114, 114, 114, 114,
	754, 1065, 1110, 358, 49, 1137, 183, 183, 1020, 840,
	108, 275, -32768, 302, 525, -32768, 296, 268, 678, 274,
	-32768, 226, 254, 754, -32768, 100, 754, -32768, -32768, -16,
	202, 268, 930, 233, 754, 754, -32768, 149, 15, -32768,
	368, 716, -32768, 134, -32768, 153, 716, -32768, 754, 716,
	-32768, -32768, -32768, -32768, -4, -34, 250, 235, 754, 108,
	110, 71, 295, 268, 142, 525, -32768, 114, 754, 142,
	187, 77, 53, -32768, -32768, -32768, 632, -32768, -32768, -32768,
	754, -32768, -32768, 930, 268, 632, 137, 754, 930, 930,
	-32768, 15, 754, 85, 930, -32768, -32768, 930, -32768, 795,
	-32768, -35, -32768, -32768, 525, 273, -32768, -32768, 163, -32768,
	142, -32768, -32768, -32768, 71, -16, -32768, -32768, 137, -32768,
	975, -32768, -32768, 292, 272, -32768, -32768, 241, 77, 77,
	-32768, 219, 930, 82, 268, 202, 930, 975, 754, 416,
	-32768, -32768, -32768, 29, 185, 268, 151, 268, -32768, 252,
	142, -32768, -32768, 53, 108, 244, 237, -32768, 754, 267,
	-32768, -32768, 205, 290, 289, 975, -32768, -32768, -32768, -32768,
	29, -32768, -32768, 40, 71, -32768, 168, 108, 108, 170,
	286, 54, -16, -32768, -32768, -32768, -32768, 754, 142, 142,
	-32768, -32768, -32768, -32768, 930,
}

var yyPgo = [...]int16{
	0, 16, 44, 8, 15, 400, 398, 18, 397, 396,
	6, 394, 391, 390, 386, 26, 384, 383, 382, 43,
	11, 377, 376, 375, 369, 7, 3, 367, 363, 361,
	5, 0, 4, 51, 25, 300, 359, 99, 19, 358,
	21, 20, 109, 357, 22, 10, 356, 355, 352, 351,
	350, 9, 13, 349, 17, 344, 341, 2, 14, 339,
	1, 337, 330, 323, 322, 320, 317,
}

var yyR1 = [...]int8{
	0, 64, 58, 58, 65, 65, 59, 59, 59, 45,
	45, 45, 45, 46, 46, 62, 63, 63, 47, 47,
	47, 49, 49, 48, 48, 50, 50, 51, 53, 53,
	52, 52, 52, 52, 52, 52, 52, 52, 52, 52,
	52, 13, 14, 15, 15, 16, 16, 66, 66, 34,
	34, 34, 34, 34, 34, 34, 34, 34, 34, 34,
	34, 34, 34, 34, 34, 34, 34, 6, 6, 5,
	5, 4, 4, 4, 4, 61, 61, 60, 60, 9,
	9, 12, 12, 8, 8, 11, 11, 7, 7, 7,
	7, 7, 10, 10, 10, 10, 10, 35, 35, 36,
	36, 31, 31, 31, 31, 31, 31, 31, 31, 31,
	31, 31, 31, 31, 31, 31, 31, 31, 31, 31,
	31, 31, 31, 31, 31, 31, 31, 31, 31, 31,
	37, 37, 32, 32, 33, 33, 1, 1, 2, 2,
	3, 3, 54, 56, 56, 55, 55, 55, 38, 38,
	57, 42, 43, 43, 43, 43, 44, 39, 40, 40,
	41, 41, 17, 17, 18, 18, 19, 19, 20, 20,
	20, 22, 22, 21, 23, 24, 24, 25, 25, 26,
	26, 26, 26, 27, 28, 28, 29, 29, 30,
}

var yyR2 = [...]int8{
	0, 2, 5, 2, 0, 2, 0, 3, 2, 0,
	2, 2, 3, 1, 1, 6, 1, 3, 3, 6,
	1, 4, 5, 1, 4, 2, 1, 4, 0, 3,
	1, 2, 1, 3, 5, 3, 1, 3, 1, 1,
	1, 2, 4, 0, 4, 1, 3, 0, 1, 1,
	1, 1, 1, 3, 8, 7, 7, 4, 4, 6,
	8, 3, 4, 4, 3, 4, 3, 0, 2, 2,
	3, 1, 3, 2, 2, 1, 3, 1, 3, 0,
	2, 0, 2, 1, 3, 1, 3, 1, 3, 2,
	1, 2, 1, 3, 5, 4, 4, 1, 3, 0,
	1, 1, 4, 2, 2, 2, 2, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	4, 3, 3, 3, 3, 3, 3, 3, 3, 5,
	1, 3, 0, 1, 0, 2, 0, 1, 1, 2,
	0, 1, 3, 1, 3, 0, 1, 2, 1, 3,
	1, 1, 3, 2, 2, 1, 1, 4, 1, 3,
	1, 2, 0, 2, 1, 3, 1, 3, 1, 1,
	3, 1, 3, 4, 3, 0, 2, 1, 3, 1,
	1, 1, 1, 3, 0, 2, 1, 3, 3,
}

var yyChk = [...]int16{
	-32768, -64, -45, 29, -46, 62, 28, -51, -47, -52,
	-63, 31, -48, -35, 54, -42, -14, 55, 56, 57,
	-62, -49, -31, 33, -13, 53, 35, -34, 41, 50,
	10, 8, 24, -43, -44, -57, 42, 27, 40, 17,
	19, 5, 34, 11, 60, 52, 62, -53, 13, -38,
	-34, -42, 15, 25, 9, -35, 13, -42, 49, -50,
	36, 37, 7, 4, 12, 46, 8, 10, 14, 16,
	30, 43, 44, 32, 38, 50, 51, 26, 21, 22,
	23, 47, 48, 39, 35, -15, 17, 33, -35, 11,
	5, 17, -9, -8, -7, -42, 7, 45, -31, -31,
	-31, -31, 5, 5, 5, -33, -31, -37, -54, -55,
	-37, -31, -56, -33, -31, 11, 34, -66, 65, -58,
	62, -51, 38, 9, -35, -35, -31, -19, -20, -22,
	-21, 5, -42, -19, -35, 13, 35, -31, -31, -31,
	-31, -31, -31, -31, -31, -31, -31, -31, -31, -31,
	38, -31, -31, -31, -31, -31, -31, -31, -31, -31,
	15, -16, -42, -15, 13, 33, -6, -5, -3, -2,
	9, -35, -36, 13, -1, 9, 15, -42, -42, -3,
	-19, -4, -31, -42, 7, 45, 18, -41, -40, -39,
	31, -2, -3, -41, 20, -1, -2, 9, 13, -2,
	6, 34, 62, -52, -59, -65, -35, -34, 15, 21,
	11, 17, -17, -18, -19, 13, -58, -31, 36, -19,
	-1, 9, 5, -58, 6, -3, -2, -4, 9, 18,
	13, -35, -7, -31, -57, -2, -2, 15, -31, -31,
	18, -40, 35, -38, -31, 20, 20, -31, -54, -31,
	58, 28, 62, 62, 13, -35, -20, 33, -25, -26,
	-19, -23, -27, -44, 17, 19, 6, -3, -2, -58,
	-31, 18, -42, -12, -11, -10, -7, -42, 7, 45,
	-4, -32, -31, -2, -4, -19, -31, -31, 38, -45,
	62, -58, 18, -2, -24, -25, -28, -29, -30, -57,
	-19, 6, -1, 9, 13, -42, -42, 18, 13, -61,
	-60, -57, -42, -3, -3, -31, 59, -26, 18, -3,
	-2, 20, -3, -2, 13, -10, -19, 13, 13, -32,
	-3, 9, 15, 6, 6, -30, -26, 15, -19, -19,
	18, 6, -60, -57, -31,
}

var yyDef = [...]int16{
	9, -2, 0, 1, 10, 11, 0, 13, 14, 28,
	0, 0, 20, 30, 32, 49, 36, 38, 39, 40,
	16, 23, 97, 151, 43, 0, 0, 101, 79, 0,
	0, 0, 0, 50, 51, 52, 0, 0, 0, 134,
	145, 134, 155, 0, 156, 150, 12, 47, 0, 0,
	148, 49, 0, 0, 0, 31, 0, 41, 0, 0,
	0, 26, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 43, 0, 0,
	-2, 99, 0, 136, 83, 87, 90, 0, 103, 104,
	105, 106, 140, 0, 0, 0, 130, 140, 143, 0,
	136, 130, 146, 0, 130, 153, 154, 0, 48, 18,
	6, 4, 0, 0, 33, 37, 98, 35, 166, 168,
	169, 162, 171, 17, 0, 0, 25, 107, 108, 109,
	110, 111, 112, 113, 114, 115, 116, 117, 118, 119,
	0, 121, 122, 123, 124, 125, 126, 127, 128, 0,
	0, 136, 45, 0, 0, 53, 0, 140, 0, 141,
	138, 100, 0, 0, 80, 137, 0, 89, 91, 0,
	0, 0, 71, 49, 0, 0, 61, 0, 160, 158,
	0, 141, 135, 0, 64, 0, 0, -2, 0, 147,
	66, 152, 27, 29, 0, 3, 0, 149, 0, 0,
	0, 0, 0, 140, 164, 0, 24, 120, 0, 42,
	0, 137, 81, 21, 57, 68, 141, 69, 139, 58,
	132, 102, 84, 88, 0, 0, 0, 0, 73, 74,
	62, 161, 0, 0, 131, 63, 65, 142, 144, 0,
	9, 0, 8, 5, 0, 34, 167, 172, 0, 177,
	179, 180, 181, 182, 175, 184, 170, 163, 141, 22,
	129, 44, 46, 0, 136, 85, 92, 87, 90, 0,
	70, 0, 133, 0, 140, 140, 72, 159, 0, 0,
	7, 19, 173, 0, 0, 140, 0, 140, 186, 0,
	165, 15, 82, 137, 0, 89, 91, 59, 132, 140,
	75, 77, 0, 0, 0, 157, 2, 178, 174, 176,
	141, 183, 185, 141, 0, 86, 93, 0, 0, 0,
	0, 138, 0, 55, 56, 187, 188, 0, 95, 96,
	60, 54, 76, 78, 94,
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

const yyFlag = -32768

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
//line build/parse.y:240
		{
			yylex.(*input).file = &File{Stmt: yyDollar[1].exprs}
			return 0
		}
	case 2:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:247
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
//line build/parse.y:267
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 6:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:275
		{
			yyVAL.exprs = nil
			yyVAL.lastStmt = nil
		}
	case 7:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:280
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
//line build/parse.y:292
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = nil
		}
	case 9:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:298
		{
			yyVAL.exprs = nil
			yyVAL.lastStmt = nil
		}
	case 10:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:303
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
//line build/parse.y:334
		{
			// Blank line; sever last rule from future comments.
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = nil
		}
	case 12:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:340
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
//line build/parse.y:361
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = yyDollar[1].exprs[len(yyDollar[1].exprs)-1]
		}
	case 14:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:366
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
//line build/parse.y:380
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
//line build/parse.y:397
		{
			yyDollar[1].def_header.Type = yyDollar[3].expr
			yyVAL.def_header = yyDollar[1].def_header
		}
	case 18:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:404
		{
			yyDollar[1].def_header.Function.Body = yyDollar[3].exprs
			yyDollar[1].def_header.ColonPos = &End{Pos: yyDollar[2].pos}
			yyVAL.expr = yyDollar[1].def_header
			yyVAL.lastStmt = yyDollar[3].lastStmt
		}
	case 19:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:411
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
//line build/parse.y:421
		{
			yyVAL.expr = yyDollar[1].ifstmt
			yyVAL.lastStmt = yyDollar[1].lastStmt
		}
	case 21:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:429
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
//line build/parse.y:438
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
//line build/parse.y:459
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
//line build/parse.y:476
		{
			yyVAL.exprs = append([]Expr{yyDollar[1].expr}, yyDollar[2].exprs...)
			yyVAL.lastStmt = yyVAL.exprs[len(yyVAL.exprs)-1]
		}
	case 28:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:482
		{
			yyVAL.exprs = []Expr{}
		}
	case 29:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:486
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 31:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:493
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
				Result: yyDollar[2].expr,
			}
		}
	case 32:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:500
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
			}
		}
	case 33:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:505
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 34:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:506
		{
			yyVAL.expr = binary(typed(yyDollar[1].expr, yyDollar[3].expr), yyDollar[4].pos, yyDollar[4].tok, yyDollar[5].expr)
		}
	case 35:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:507
		{
			yyVAL.expr = typed(yyDollar[1].expr, yyDollar[3].expr)
		}
	case 37:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:509
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 38:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:511
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 39:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:518
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 40:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:525
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 41:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:536
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
//line build/parse.y:551
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
//line build/parse.y:563
		{
			yyVAL.expr = nil
		}
	case 44:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:567
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
//line build/parse.y:578
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 46:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:582
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 52:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:594
		{
			yyVAL.expr = yyDollar[1].string
		}
	case 53:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:598
		{
			yyVAL.expr = &DotExpr{
				X:       yyDollar[1].expr,
				Dot:     yyDollar[2].pos,
				NamePos: yyDollar[3].pos,
				Name:    yyDollar[3].tok,
			}
		}
	case 54:
		yyDollar = yyS[yypt-8 : yypt+1]
//line build/parse.y:607
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
	case 55:
		yyDollar = yyS[yypt-7 : yypt+1]
//line build/parse.y:621
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
	case 56:
		yyDollar = yyS[yypt-7 : yypt+1]
//line build/parse.y:634
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
	case 57:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:647
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
	case 58:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:658
		{
			yyVAL.expr = &IndexExpr{
				X:          yyDollar[1].expr,
				IndexStart: yyDollar[2].pos,
				Y:          yyDollar[3].expr,
				End:        yyDollar[4].pos,
			}
		}
	case 59:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:667
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
	case 60:
		yyDollar = yyS[yypt-8 : yypt+1]
//line build/parse.y:678
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
	case 61:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:691
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 62:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:700
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
	case 63:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:711
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
	case 64:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:722
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
	case 65:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:735
		{
			yyVAL.expr = &SetExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[4].pos),
			}
		}
	case 66:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:744
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
	case 67:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:765
		{
			yyVAL.exprs = nil
		}
	case 68:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:769
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 69:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:775
		{
			yyVAL.exprs = []Expr{yyDollar[2].expr}
		}
	case 70:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:779
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 72:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:786
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 73:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:790
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 74:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:794
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 75:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:799
		{
			yyVAL.loadargs = []*struct {
				from Ident
				to   Ident
			}{yyDollar[1].loadarg}
		}
	case 76:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:803
		{
			yyDollar[1].loadargs = append(yyDollar[1].loadargs, yyDollar[3].loadarg)
			yyVAL.loadargs = yyDollar[1].loadargs
		}
	case 77:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:809
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
	case 78:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:826
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
	case 79:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:841
		{
			yyVAL.exprs = nil
		}
	case 80:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:845
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 81:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:850
		{
			yyVAL.exprs = nil
		}
	case 82:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:854
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 83:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:860
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 84:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:864
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 85:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:871
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 86:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:875
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 88:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:882
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 89:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:886
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 90:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:890
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, nil)
		}
	case 91:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:894
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 93:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:903
		{
			yyVAL.expr = typed(yyDollar[1].expr, yyDollar[3].expr)
		}
	case 94:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:907
		{
			yyVAL.expr = binary(typed(yyDollar[1].expr, yyDollar[3].expr), yyDollar[4].pos, yyDollar[4].tok, yyDollar[5].expr)
		}
	case 95:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:911
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, typed(yyDollar[2].expr, yyDollar[4].expr))
		}
	case 96:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:915
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, typed(yyDollar[2].expr, yyDollar[4].expr))
		}
	case 98:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:922
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
	case 99:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:937
		{
			yyVAL.expr = nil
		}
	case 102:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:945
		{
			yyVAL.expr = &LambdaExpr{
				Function: Function{
					StartPos: yyDollar[1].pos,
					Params:   yyDollar[2].exprs,
					Body:     []Expr{yyDollar[4].expr},
				},
			}
		}
	case 103:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:954
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 104:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:955
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 105:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:956
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 106:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:957
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 107:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:958
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 108:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:959
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 109:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:960
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 110:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:961
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 111:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:962
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 112:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:963
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 113:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:964
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 114:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:965
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 115:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:966
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 116:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:967
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 117:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:968
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 118:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:969
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 119:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:970
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 120:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:971
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "not in", yyDollar[4].expr)
		}
	case 121:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:972
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 122:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:973
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 123:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:974
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 124:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:975
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 125:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:976
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 126:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:977
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 127:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:978
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 128:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:980
		{
			if b, ok := yyDollar[3].expr.(*UnaryExpr); ok && b.Op == "not" {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "is not", b.X)
			} else {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
			}
		}
	case 129:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:988
		{
			yyVAL.expr = &ConditionalExpr{
				Then:      yyDollar[1].expr,
				IfStart:   yyDollar[2].pos,
				Test:      yyDollar[3].expr,
				ElseStart: yyDollar[4].pos,
				Else:      yyDollar[5].expr,
			}
		}
	case 130:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1000
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 131:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1004
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 132:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1009
		{
			yyVAL.expr = nil
		}
	case 134:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1015
		{
			yyVAL.exprs, yyVAL.comma = nil, Position{}
		}
	case 135:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1019
		{
			yyVAL.exprs, yyVAL.comma = yyDollar[1].exprs, yyDollar[2].pos
		}
	case 136:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1029
		{
			yyVAL.pos = Position{}
		}
	case 139:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1040
		{
			yyVAL.pos = yyDollar[1].pos
		}
	case 140:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1048
		{
			yyVAL.pos = Position{}
		}
	case 142:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1055
		{
			yyVAL.kv = &KeyValueExpr{
				Key:   yyDollar[1].expr,
				Colon: yyDollar[2].pos,
				Value: yyDollar[3].expr,
			}
		}
	case 143:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1065
		{
			yyVAL.kvs = []*KeyValueExpr{yyDollar[1].kv}
		}
	case 144:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1069
		{
			yyVAL.kvs = append(yyDollar[1].kvs, yyDollar[3].kv)
		}
	case 145:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1074
		{
			yyVAL.kvs = nil
		}
	case 146:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1078
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 147:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1082
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 149:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1089
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
	case 150:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1105
		{
			yyVAL.string = &StringExpr{
				Start:       yyDollar[1].pos,
				Value:       yyDollar[1].str,
				TripleQuote: yyDollar[1].triple,
				End:         yyDollar[1].pos.add(yyDollar[1].tok),
				Token:       yyDollar[1].tok,
			}
		}
	case 151:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1117
		{
			yyVAL.expr = &Ident{NamePos: yyDollar[1].pos, Name: yyDollar[1].tok}
		}
	case 152:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1123
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok + "." + yyDollar[3].tok}
		}
	case 153:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1127
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok + "."}
		}
	case 154:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1131
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: "." + yyDollar[2].tok}
		}
	case 155:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1135
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok}
		}
	case 156:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1141
		{
			yyVAL.expr = &EllipsisExpr{
				Pos: yyDollar[1].pos,
			}
		}
	case 157:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:1149
		{
			yyVAL.expr = &ForClause{
				For:  yyDollar[1].pos,
				Vars: yyDollar[2].expr,
				In:   yyDollar[3].pos,
				X:    yyDollar[4].expr,
			}
		}
	case 158:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1160
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 159:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1164
		{
			yyVAL.exprs = append(yyDollar[1].exprs, &IfClause{
				If:   yyDollar[2].pos,
				Cond: yyDollar[3].expr,
			})
		}
	case 160:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1173
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 161:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1177
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[2].exprs...)
		}
	case 162:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1182
		{
			yyVAL.exprs = nil
		}
	case 163:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1186
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 164:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1192
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 165:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1196
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 166:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1202
		{
			yyVAL.expr = yyDollar[1].expr
		}
	case 167:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1206
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 170:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1214
		{
			if len(yyDollar[2].exprs) == 1 && yyDollar[2].comma.Line == 0 {
				// Just a parenthesized type expression, not a tuple;
				// useless in type syntax, but permitted.
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
	case 172:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1238
		{
			yyVAL.expr = &DotExpr{
				X:       yyDollar[1].expr,
				Dot:     yyDollar[2].pos,
				NamePos: yyDollar[3].pos,
				Name:    yyDollar[3].tok,
			}
		}
	case 173:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:1249
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
	case 174:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1262
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 175:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1272
		{
			yyVAL.exprs = nil
		}
	case 176:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1276
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 177:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1282
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 178:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1286
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 183:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1298
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
	case 184:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1312
		{
			yyVAL.kvs = nil
		}
	case 185:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1316
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 186:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1322
		{
			yyVAL.kvs = []*KeyValueExpr{yyDollar[1].kv}
		}
	case 187:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1326
		{
			yyVAL.kvs = append(yyDollar[1].kvs, yyDollar[3].kv)
		}
	case 188:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1332
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
