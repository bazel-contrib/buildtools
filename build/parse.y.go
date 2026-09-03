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
const _COMMENT = 57348
const _EOF = 57349
const _EQ = 57350
const _FOR = 57351
const _GE = 57352
const _IDENT = 57353
const _INT = 57354
const _IF = 57355
const _ELSE = 57356
const _ELIF = 57357
const _IN = 57358
const _IS = 57359
const _LAMBDA = 57360
const _LOAD = 57361
const _LE = 57362
const _NE = 57363
const _STAR_STAR = 57364
const _INT_DIV = 57365
const _BIT_LSH = 57366
const _BIT_RSH = 57367
const _ARROW = 57368
const _NOT = 57369
const _OR = 57370
const _STRING = 57371
const _DEF = 57372
const _RETURN = 57373
const _PASS = 57374
const _BREAK = 57375
const _CONTINUE = 57376
const _INDENT = 57377
const _UNINDENT = 57378
const _ELLIPSIS = 57379
const ShiftInstead = 57380
const _ASSERT = 57381
const _UNARY = 57382

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

//line build/parse.y:1272

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
	-1, 86,
	6, 64,
	-2, 137,
	-1, 184,
	20, 134,
	-2, 135,
}

const yyPrivate = 57344

const yyLast = 1067

var yyAct = [...]int16{
	22, 34, 243, 292, 280, 265, 161, 115, 256, 167,
	7, 241, 2, 211, 102, 174, 122, 242, 175, 162,
	113, 257, 27, 81, 45, 112, 9, 271, 236, 189,
	94, 95, 96, 97, 46, 42, 23, 100, 105, 108,
	41, 54, 143, 56, 57, 225, 188, 110, 216, 216,
	90, 120, 99, 23, 41, 121, 240, 126, 158, 130,
	131, 132, 133, 134, 135, 136, 137, 138, 139, 140,
	141, 142, 23, 144, 145, 146, 147, 148, 149, 150,
	151, 152, 38, 47, 214, 31, 15, 30, 40, 41,
	41, 107, 83, 177, 36, 59, 37, 196, 58, 53,
	234, 32, 117, 60, 13, 172, 117, 156, 179, 23,
	39, 128, 91, 101, 182, 301, 28, 35, 180, 51,
	215, 178, 259, 52, 183, 29, 186, 41, 15, 233,
	269, 84, 129, 235, 116, 61, 125, 229, 125, 190,
	194, 181, 23, 223, 201, 246, 92, 23, 318, 200,
	298, 104, 252, 118, 119, 177, 203, 248, 260, 249,
	127, 246, 212, 315, 204, 216, 155, 209, 173, 196,
	221, 23, 23, 248, 222, 249, 170, 171, 207, 227,
	210, 228, 93, 82, 227, 163, 230, 232, 23, 50,
	220, 177, 164, 224, 273, 48, 15, 50, 247, 224,
	46, 231, 226, 251, 195, 49, 217, 285, 312, 169,
	196, 212, 15, 239, 247, 263, 264, 262, 290, 266,
	250, 193, 38, 289, 261, 197, 268, 86, 40, 308,
	169, 198, 307, 85, 36, 153, 37, 125, 304, 87,
	50, 15, 267, 50, 237, 213, 270, 199, 218, 23,
	39, 281, 91, 166, 44, 50, 109, 35, 272, 157,
	277, 274, 311, 288, 163, 283, 284, 41, 50, 293,
	295, 219, 216, 205, 184, 168, 319, 282, 7, 275,
	125, 208, 125, 15, 299, 187, 302, 206, 306, 253,
	258, 266, 297, 305, 213, 98, 309, 300, 310, 303,
	238, 111, 192, 1, 10, 281, 20, 291, 313, 191,
	316, 317, 106, 293, 321, 320, 322, 103, 297, 43,
	55, 15, 314, 21, 12, 8, 4, 33, 38, 176,
	165, 31, 125, 30, 40, 279, 278, 245, 276, 244,
	36, 123, 37, 286, 287, 124, 154, 32, 16, 24,
	6, 294, 254, 11, 15, 23, 39, 26, 125, 255,
	88, 89, 28, 35, 159, 160, 0, 0, 258, 125,
	0, 29, 0, 41, 25, 14, 17, 18, 19, 0,
	296, 0, 0, 5, 125, 0, 0, 0, 125, 0,
	0, 125, 125, 38, 0, 294, 31, 0, 30, 40,
	0, 0, 0, 0, 0, 36, 0, 37, 0, 0,
	0, 0, 32, 0, 0, 6, 3, 0, 11, 0,
	23, 39, 26, 0, 0, 0, 59, 28, 35, 58,
	62, 0, 63, 0, 60, 0, 29, 0, 41, 25,
	14, 17, 18, 19, 38, 0, 0, 31, 5, 30,
	40, 0, 0, 0, 0, 0, 36, 0, 37, 0,
	0, 0, 0, 32, 0, 0, 61, 77, 78, 0,
	0, 23, 39, 0, 0, 0, 0, 59, 28, 35,
	58, 62, 0, 63, 0, 60, 0, 29, 0, 41,
	0, 14, 17, 18, 19, 0, 0, 59, 0, 114,
	58, 62, 0, 63, 0, 60, 185, 64, 0, 65,
	0, 0, 0, 0, 74, 75, 76, 61, 0, 73,
	0, 0, 66, 0, 69, 0, 0, 80, 0, 0,
	70, 79, 0, 0, 67, 68, 0, 61, 77, 78,
	59, 71, 72, 58, 62, 0, 63, 0, 60, 0,
	64, 0, 65, 0, 0, 0, 0, 74, 75, 76,
	0, 0, 73, 0, 0, 66, 0, 69, 0, 0,
	80, 202, 0, 70, 79, 0, 0, 67, 68, 0,
	61, 77, 78, 59, 71, 72, 58, 62, 0, 63,
	0, 60, 0, 64, 0, 65, 0, 0, 0, 0,
	74, 75, 76, 0, 0, 73, 0, 0, 66, 177,
	69, 0, 0, 80, 0, 0, 70, 79, 0, 0,
	67, 68, 0, 61, 77, 78, 59, 71, 72, 58,
	62, 0, 63, 0, 60, 0, 64, 0, 65, 0,
	0, 0, 0, 74, 75, 76, 0, 0, 73, 0,
	0, 66, 0, 69, 0, 0, 80, 0, 0, 70,
	79, 0, 0, 67, 68, 0, 61, 77, 78, 38,
	71, 72, 31, 0, 30, 40, 0, 0, 0, 0,
	0, 36, 0, 37, 0, 0, 0, 0, 32, 0,
	0, 0, 0, 0, 0, 0, 23, 39, 0, 0,
	0, 0, 0, 28, 35, 0, 0, 0, 0, 0,
	0, 0, 29, 0, 41, 0, 14, 17, 18, 19,
	59, 0, 0, 58, 62, 0, 63, 0, 60, 0,
	64, 0, 65, 0, 0, 0, 0, 74, 75, 76,
	0, 0, 73, 0, 0, 66, 0, 69, 0, 0,
	0, 0, 0, 70, 79, 0, 0, 67, 68, 0,
	61, 77, 78, 59, 71, 72, 58, 62, 0, 63,
	0, 60, 0, 64, 0, 65, 0, 0, 0, 0,
	74, 75, 76, 0, 0, 73, 0, 0, 66, 0,
	69, 0, 0, 0, 0, 0, 70, 0, 0, 0,
	67, 68, 0, 61, 77, 78, 59, 71, 72, 58,
	62, 0, 63, 0, 60, 0, 64, 0, 65, 0,
	0, 0, 0, 74, 75, 76, 0, 0, 73, 0,
	0, 66, 0, 69, 0, 0, 0, 0, 0, 70,
	0, 0, 0, 67, 68, 0, 61, 77, 78, 59,
	71, 0, 58, 62, 0, 63, 0, 60, 0, 64,
	0, 65, 0, 0, 0, 0, 74, 75, 76, 0,
	0, 0, 0, 0, 66, 0, 69, 0, 0, 0,
	0, 0, 70, 0, 0, 0, 67, 68, 0, 61,
	77, 78, 38, 71, 214, 31, 216, 30, 40, 0,
	0, 0, 0, 0, 36, 0, 37, 0, 0, 0,
	0, 32, 0, 0, 0, 0, 0, 0, 0, 23,
	39, 0, 0, 0, 0, 0, 28, 35, 38, 0,
	215, 31, 216, 30, 40, 29, 0, 41, 0, 0,
	36, 0, 37, 0, 0, 0, 0, 32, 0, 0,
	0, 0, 0, 0, 0, 23, 39, 0, 0, 0,
	0, 0, 28, 35, 38, 0, 0, 31, 0, 30,
	40, 29, 0, 41, 0, 0, 36, 0, 37, 0,
	0, 0, 0, 32, 0, 0, 0, 0, 0, 0,
	0, 23, 39, 0, 0, 0, 0, 59, 28, 35,
	58, 62, 0, 63, 0, 60, 0, 29, 0, 41,
	0, 0, 0, 0, 74, 75, 76, 59, 0, 0,
	58, 62, 0, 63, 59, 60, 0, 58, 62, 0,
	63, 0, 60, 0, 0, 75, 76, 61, 77, 78,
	0, 0, 75, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 61, 77, 78,
	0, 0, 0, 0, 61, 77, 78,
}

var yyPact = [...]int16{
	-1000, -1000, 388, -1000, -1000, -1000, -25, -1000, -1000, -1000,
	241, 217, -1000, 180, 959, 110, -1000, -1000, -1000, -1000,
	-6, 8, 622, -1000, 166, 60, 959, 222, 139, 959,
	959, 959, 959, -1000, -1000, 290, 959, 959, 959, 245,
	14, -1000, -1000, -38, 439, 97, 222, -1000, 959, 959,
	959, 259, 21, -1000, 21, 959, 98, -1000, 959, 959,
	959, 959, 959, 959, 959, 959, 959, 959, 959, 959,
	959, 5, 959, 959, 959, 959, 959, 959, 959, 959,
	959, 220, 21, 166, 246, 26, 255, 959, 240, 266,
	-1000, 215, 21, 21, -1000, -1000, -1000, -1000, 255, 150,
	579, 255, 63, 121, 265, 493, 255, 279, 622, 13,
	-1000, -31, 664, -1000, -1000, -1000, 959, 217, 259, 259,
	716, 189, -1000, 214, -1000, -1000, 76, 234, 439, -1000,
	-1000, -1000, -1000, -1000, 91, 91, 993, 993, 993, 993,
	993, 993, 993, 959, 802, 845, 1013, 422, 1020, 473,
	473, 759, 536, 21, 264, -1000, 282, 439, -1000, 275,
	255, 77, 263, -1000, 188, 235, 959, -1000, 139, 959,
	-1000, -1000, -10, -1000, 125, 11, -1000, 217, 923, -1000,
	161, -1000, 117, 923, -1000, 959, 923, -1000, -1000, -1000,
	-1000, 73, -32, 231, 222, 959, 21, 24, 140, 439,
	-1000, 993, 959, 76, 134, 21, 115, -1000, -1000, -1000,
	887, -1000, 622, 202, 959, 959, -1000, -1000, 959, -1000,
	-1000, 622, 255, -1000, 11, 959, 93, 622, -1000, -1000,
	622, -1000, 493, -1000, -33, -1000, -1000, 439, 259, -1000,
	-1000, 176, -1000, 76, -1000, -1000, 273, -1000, 140, -10,
	-1000, 716, -1000, -1000, 271, 257, -1000, -1000, 194, 21,
	21, -1000, 959, 622, 622, 205, 622, 40, 716, 959,
	323, -1000, -1000, -1000, 156, -1000, 132, 255, 95, 255,
	-1000, 225, -1000, -1000, 115, 21, 219, 216, 622, -1000,
	959, 253, -1000, -1000, 193, 716, -1000, -1000, -1000, -1000,
	156, -1000, -1000, 39, 140, -1000, 148, 21, 21, 130,
	270, 4, -10, -1000, -1000, 959, 76, 76, -1000, -1000,
	-1000, -1000, 622,
}

var yyPgo = [...]int16{
	0, 9, 19, 6, 13, 365, 364, 21, 361, 360,
	8, 359, 352, 349, 348, 23, 346, 2, 16, 345,
	341, 339, 338, 11, 17, 337, 336, 335, 4, 0,
	5, 52, 22, 104, 330, 113, 24, 329, 18, 15,
	83, 327, 12, 326, 325, 324, 323, 320, 7, 26,
	319, 14, 317, 312, 1, 20, 309, 3, 307, 306,
	304, 303, 302, 301,
}

var yyR1 = [...]int8{
	0, 61, 55, 55, 62, 62, 56, 56, 56, 42,
	42, 42, 42, 43, 43, 59, 60, 60, 44, 44,
	44, 46, 46, 45, 45, 47, 47, 48, 50, 50,
	49, 49, 49, 49, 49, 49, 49, 49, 49, 49,
	49, 13, 14, 15, 15, 16, 16, 63, 63, 32,
	32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
	32, 32, 32, 32, 6, 6, 5, 5, 4, 4,
	4, 4, 58, 58, 57, 57, 9, 9, 12, 12,
	8, 8, 11, 11, 7, 7, 7, 7, 7, 10,
	10, 10, 10, 10, 33, 33, 34, 34, 29, 29,
	29, 29, 29, 29, 29, 29, 29, 29, 29, 29,
	29, 29, 29, 29, 29, 29, 29, 29, 29, 29,
	29, 29, 29, 29, 29, 29, 29, 35, 35, 30,
	30, 31, 31, 1, 1, 2, 2, 3, 3, 51,
	53, 53, 52, 52, 52, 36, 36, 54, 40, 41,
	41, 41, 41, 37, 38, 38, 39, 39, 17, 17,
	18, 18, 20, 20, 19, 21, 22, 22, 23, 23,
	24, 24, 24, 24, 24, 25, 26, 26, 27, 27,
	28,
}

var yyR2 = [...]int8{
	0, 2, 5, 2, 0, 2, 0, 3, 2, 0,
	2, 2, 3, 1, 1, 6, 1, 3, 3, 6,
	1, 4, 5, 1, 4, 2, 1, 4, 0, 3,
	1, 2, 1, 3, 5, 3, 1, 3, 1, 1,
	1, 2, 4, 0, 4, 1, 3, 0, 1, 1,
	1, 1, 3, 8, 4, 4, 6, 8, 3, 4,
	4, 3, 4, 3, 0, 2, 2, 3, 1, 3,
	2, 2, 1, 3, 1, 3, 0, 2, 0, 2,
	1, 3, 1, 3, 1, 3, 2, 1, 2, 1,
	3, 5, 4, 4, 1, 3, 0, 1, 1, 4,
	2, 2, 2, 2, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 4, 3, 3,
	3, 3, 3, 3, 3, 3, 5, 1, 3, 0,
	1, 0, 2, 0, 1, 1, 2, 0, 1, 3,
	1, 3, 0, 1, 2, 1, 3, 1, 1, 3,
	2, 2, 1, 4, 1, 3, 1, 2, 1, 3,
	1, 1, 1, 3, 4, 3, 0, 2, 1, 3,
	1, 1, 1, 2, 1, 3, 0, 2, 1, 3,
	3,
}

var yyChk = [...]int16{
	-1000, -61, -42, 28, -43, 60, 27, -48, -44, -49,
	-60, 30, -45, -33, 52, -40, -14, 53, 54, 55,
	-59, -46, -29, 32, -13, 51, 34, -32, 39, 48,
	10, 8, 24, -41, -54, 40, 17, 19, 5, 33,
	11, 50, 60, -50, 13, -36, -32, -40, 15, 25,
	9, -33, 13, -40, 47, -47, 35, 36, 7, 4,
	12, 44, 8, 10, 14, 16, 29, 41, 42, 31,
	37, 48, 49, 26, 21, 22, 23, 45, 46, 38,
	34, -15, 17, 32, -33, 11, 5, 17, -9, -8,
	-7, -40, 7, 43, -29, -29, -29, -29, 5, -31,
	-29, -35, -51, -52, -35, -29, -53, -31, -29, 11,
	33, -63, 63, -55, 60, -48, 37, 9, -33, -33,
	-29, -17, -18, -20, -19, -40, -17, -33, 13, 34,
	-29, -29, -29, -29, -29, -29, -29, -29, -29, -29,
	-29, -29, -29, 37, -29, -29, -29, -29, -29, -29,
	-29, -29, -29, 15, -16, -40, -15, 13, 32, -6,
	-5, -3, -2, 9, -33, -34, 13, -1, 9, 15,
	-40, -40, -3, 18, -39, -38, -37, 30, -2, -3,
	-39, 20, -1, -2, 9, 13, -2, 6, 33, 60,
	-49, -56, -62, -33, -32, 15, 21, 11, 17, 13,
	-55, -29, 35, -17, -1, 9, 5, -55, 6, -3,
	-2, -4, -29, -40, 7, 43, 9, 18, 13, -33,
	-7, -29, -54, 18, -38, 34, -36, -29, 20, 20,
	-29, -51, -29, 56, 27, 60, 60, 13, -33, -18,
	32, -23, -24, -17, -21, -25, 5, 58, 17, 19,
	-55, -29, 18, -40, -12, -11, -10, -7, -40, 7,
	43, -4, 15, -29, -29, -30, -29, -2, -29, 37,
	-42, 60, -55, 18, -2, 6, -22, -23, -26, -27,
	-28, -54, 6, -1, 9, 13, -40, -40, -29, 18,
	13, -58, -57, -54, -40, -29, 57, -24, 18, -3,
	-2, 20, -3, -2, 13, -10, -17, 13, 13, -30,
	-3, 9, 15, -28, -24, 15, -17, -17, 18, 6,
	-57, -54, -29,
}

var yyDef = [...]int16{
	9, -2, 0, 1, 10, 11, 0, 13, 14, 28,
	0, 0, 20, 30, 32, 49, 36, 38, 39, 40,
	16, 23, 94, 148, 43, 0, 0, 98, 76, 0,
	0, 0, 0, 50, 51, 0, 131, 142, 131, 152,
	0, 147, 12, 47, 0, 0, 145, 49, 0, 0,
	0, 31, 0, 41, 0, 0, 0, 26, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 43, 0, 0, -2, 96, 0, 133,
	80, 84, 87, 0, 100, 101, 102, 103, 137, 0,
	127, 137, 140, 0, 133, 127, 143, 0, 127, 150,
	151, 0, 48, 18, 6, 4, 0, 0, 33, 37,
	95, 35, 158, 160, 161, 162, 17, 0, 0, 25,
	104, 105, 106, 107, 108, 109, 110, 111, 112, 113,
	114, 115, 116, 0, 118, 119, 120, 121, 122, 123,
	124, 125, 0, 0, 133, 45, 0, 0, 52, 0,
	137, 0, 138, 135, 97, 0, 0, 77, 134, 0,
	86, 88, 0, 58, 0, 156, 154, 0, 138, 132,
	0, 61, 0, 0, -2, 0, 144, 63, 149, 27,
	29, 0, 3, 0, 146, 0, 0, 0, 0, 0,
	24, 117, 0, 42, 0, 134, 78, 21, 54, 65,
	138, 66, 68, 49, 0, 0, 136, 55, 129, 99,
	81, 85, 0, 59, 157, 0, 0, 128, 60, 62,
	139, 141, 0, 9, 0, 8, 5, 0, 34, 159,
	163, 0, 168, 170, 171, 172, 0, 174, 166, 176,
	22, 126, 44, 46, 0, 133, 82, 89, 84, 87,
	0, 67, 0, 70, 71, 0, 130, 0, 155, 0,
	0, 7, 19, 164, 0, 173, 0, 137, 0, 137,
	178, 0, 15, 79, 134, 0, 86, 88, 69, 56,
	129, 137, 72, 74, 0, 153, 2, 169, 165, 167,
	138, 175, 177, 138, 0, 83, 90, 0, 0, 0,
	0, 135, 0, 179, 180, 0, 92, 93, 57, 53,
	73, 75, 91,
}

var yyTok1 = [...]int8{
	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	60, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 4, 22, 3,
	5, 6, 7, 8, 9, 10, 11, 12, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 13, 63,
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
	53, 54, 55, 56, 57, 58, 59, 61, 62,
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
//line build/parse.y:235
		{
			yylex.(*input).file = &File{Stmt: yyDollar[1].exprs}
			return 0
		}
	case 2:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:242
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
//line build/parse.y:262
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 6:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:270
		{
			yyVAL.exprs = nil
			yyVAL.lastStmt = nil
		}
	case 7:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:275
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
//line build/parse.y:287
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = nil
		}
	case 9:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:293
		{
			yyVAL.exprs = nil
			yyVAL.lastStmt = nil
		}
	case 10:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:298
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
//line build/parse.y:329
		{
			// Blank line; sever last rule from future comments.
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = nil
		}
	case 12:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:335
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
//line build/parse.y:356
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = yyDollar[1].exprs[len(yyDollar[1].exprs)-1]
		}
	case 14:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:361
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
//line build/parse.y:375
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
//line build/parse.y:392
		{
			yyDollar[1].def_header.Type = yyDollar[3].expr
			yyVAL.def_header = yyDollar[1].def_header
		}
	case 18:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:399
		{
			yyDollar[1].def_header.Function.Body = yyDollar[3].exprs
			yyDollar[1].def_header.ColonPos = &End{Pos: yyDollar[2].pos}
			yyVAL.expr = yyDollar[1].def_header
			yyVAL.lastStmt = yyDollar[3].lastStmt
		}
	case 19:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:406
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
//line build/parse.y:416
		{
			yyVAL.expr = yyDollar[1].ifstmt
			yyVAL.lastStmt = yyDollar[1].lastStmt
		}
	case 21:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:424
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
//line build/parse.y:433
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
//line build/parse.y:454
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
//line build/parse.y:471
		{
			yyVAL.exprs = append([]Expr{yyDollar[1].expr}, yyDollar[2].exprs...)
			yyVAL.lastStmt = yyVAL.exprs[len(yyVAL.exprs)-1]
		}
	case 28:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:477
		{
			yyVAL.exprs = []Expr{}
		}
	case 29:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:481
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 31:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:488
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
				Result: yyDollar[2].expr,
			}
		}
	case 32:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:495
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
			}
		}
	case 33:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:500
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 34:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:501
		{
			yyVAL.expr = binary(typed(yyDollar[1].expr, yyDollar[3].expr), yyDollar[4].pos, yyDollar[4].tok, yyDollar[5].expr)
		}
	case 35:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:502
		{
			yyVAL.expr = typed(yyDollar[1].expr, yyDollar[3].expr)
		}
	case 37:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:504
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 38:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:506
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 39:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:513
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 40:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:520
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 41:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:531
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
//line build/parse.y:546
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
//line build/parse.y:558
		{
			yyVAL.expr = nil
		}
	case 44:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:562
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
//line build/parse.y:573
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 46:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:577
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 51:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:588
		{
			yyVAL.expr = yyDollar[1].string
		}
	case 52:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:592
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
//line build/parse.y:601
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
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:615
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
	case 55:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:626
		{
			yyVAL.expr = &IndexExpr{
				X:          yyDollar[1].expr,
				IndexStart: yyDollar[2].pos,
				Y:          yyDollar[3].expr,
				End:        yyDollar[4].pos,
			}
		}
	case 56:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:635
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
	case 57:
		yyDollar = yyS[yypt-8 : yypt+1]
//line build/parse.y:646
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
	case 58:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:659
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 59:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:668
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
	case 60:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:679
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
	case 61:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:690
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
	case 62:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:703
		{
			yyVAL.expr = &SetExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[4].pos),
			}
		}
	case 63:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:712
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
	case 64:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:733
		{
			yyVAL.exprs = nil
		}
	case 65:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:737
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 66:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:743
		{
			yyVAL.exprs = []Expr{yyDollar[2].expr}
		}
	case 67:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:747
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 69:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:754
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 70:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:758
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 71:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:762
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 72:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:767
		{
			yyVAL.loadargs = []*struct {
				from Ident
				to   Ident
			}{yyDollar[1].loadarg}
		}
	case 73:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:771
		{
			yyDollar[1].loadargs = append(yyDollar[1].loadargs, yyDollar[3].loadarg)
			yyVAL.loadargs = yyDollar[1].loadargs
		}
	case 74:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:777
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
	case 75:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:794
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
	case 76:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:809
		{
			yyVAL.exprs = nil
		}
	case 77:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:813
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 78:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:818
		{
			yyVAL.exprs = nil
		}
	case 79:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:822
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 80:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:828
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 81:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:832
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 82:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:839
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 83:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:843
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 85:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:850
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 86:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:854
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 87:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:858
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, nil)
		}
	case 88:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:862
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 90:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:871
		{
			yyVAL.expr = typed(yyDollar[1].expr, yyDollar[3].expr)
		}
	case 91:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:875
		{
			yyVAL.expr = binary(typed(yyDollar[1].expr, yyDollar[3].expr), yyDollar[4].pos, yyDollar[4].tok, yyDollar[5].expr)
		}
	case 92:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:879
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, typed(yyDollar[2].expr, yyDollar[4].expr))
		}
	case 93:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:883
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, typed(yyDollar[2].expr, yyDollar[4].expr))
		}
	case 95:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:890
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
	case 96:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:905
		{
			yyVAL.expr = nil
		}
	case 99:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:913
		{
			yyVAL.expr = &LambdaExpr{
				Function: Function{
					StartPos: yyDollar[1].pos,
					Params:   yyDollar[2].exprs,
					Body:     []Expr{yyDollar[4].expr},
				},
			}
		}
	case 100:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:922
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 101:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:923
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 102:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:924
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 103:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:925
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 104:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:926
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 105:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:927
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 106:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:928
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 107:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:929
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 108:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:930
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 109:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:931
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 110:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:932
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 111:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:933
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 112:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:934
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 113:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:935
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 114:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:936
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 115:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:937
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 116:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:938
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 117:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:939
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "not in", yyDollar[4].expr)
		}
	case 118:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:940
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 119:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:941
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 120:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:942
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 121:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:943
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 122:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:944
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 123:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:945
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 124:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:946
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 125:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:948
		{
			if b, ok := yyDollar[3].expr.(*UnaryExpr); ok && b.Op == "not" {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "is not", b.X)
			} else {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
			}
		}
	case 126:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:956
		{
			yyVAL.expr = &ConditionalExpr{
				Then:      yyDollar[1].expr,
				IfStart:   yyDollar[2].pos,
				Test:      yyDollar[3].expr,
				ElseStart: yyDollar[4].pos,
				Else:      yyDollar[5].expr,
			}
		}
	case 127:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:968
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 128:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:972
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 129:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:977
		{
			yyVAL.expr = nil
		}
	case 131:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:983
		{
			yyVAL.exprs, yyVAL.comma = nil, Position{}
		}
	case 132:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:987
		{
			yyVAL.exprs, yyVAL.comma = yyDollar[1].exprs, yyDollar[2].pos
		}
	case 133:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:997
		{
			yyVAL.pos = Position{}
		}
	case 136:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1008
		{
			yyVAL.pos = yyDollar[1].pos
		}
	case 137:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1016
		{
			yyVAL.pos = Position{}
		}
	case 139:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1023
		{
			yyVAL.kv = &KeyValueExpr{
				Key:   yyDollar[1].expr,
				Colon: yyDollar[2].pos,
				Value: yyDollar[3].expr,
			}
		}
	case 140:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1033
		{
			yyVAL.kvs = []*KeyValueExpr{yyDollar[1].kv}
		}
	case 141:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1037
		{
			yyVAL.kvs = append(yyDollar[1].kvs, yyDollar[3].kv)
		}
	case 142:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1042
		{
			yyVAL.kvs = nil
		}
	case 143:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1046
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 144:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1050
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 146:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1057
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
	case 147:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1073
		{
			yyVAL.string = &StringExpr{
				Start:       yyDollar[1].pos,
				Value:       yyDollar[1].str,
				TripleQuote: yyDollar[1].triple,
				End:         yyDollar[1].pos.add(yyDollar[1].tok),
				Token:       yyDollar[1].tok,
			}
		}
	case 148:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1085
		{
			yyVAL.expr = &Ident{NamePos: yyDollar[1].pos, Name: yyDollar[1].tok}
		}
	case 149:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1091
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok + "." + yyDollar[3].tok}
		}
	case 150:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1095
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok + "."}
		}
	case 151:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1099
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: "." + yyDollar[2].tok}
		}
	case 152:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1103
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok}
		}
	case 153:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:1109
		{
			yyVAL.expr = &ForClause{
				For:  yyDollar[1].pos,
				Vars: yyDollar[2].expr,
				In:   yyDollar[3].pos,
				X:    yyDollar[4].expr,
			}
		}
	case 154:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1120
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 155:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1124
		{
			yyVAL.exprs = append(yyDollar[1].exprs, &IfClause{
				If:   yyDollar[2].pos,
				Cond: yyDollar[3].expr,
			})
		}
	case 156:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1133
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 157:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1137
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[2].exprs...)
		}
	case 158:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1143
		{
			yyVAL.expr = yyDollar[1].expr
		}
	case 159:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1147
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 163:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1158
		{
			yyVAL.expr = &DotExpr{
				X:       yyDollar[1].expr,
				Dot:     yyDollar[2].pos,
				NamePos: yyDollar[3].pos,
				Name:    yyDollar[3].tok,
			}
		}
	case 164:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:1169
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
	case 165:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1182
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 166:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1192
		{
			yyVAL.exprs = nil
		}
	case 167:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1196
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 168:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1202
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 169:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1206
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 173:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1215
		{
			yyVAL.expr = &TupleExpr{
				Start: yyDollar[1].pos,
				End:   End{Pos: yyDollar[2].pos},
			}
		}
	case 174:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1222
		{
			yyVAL.expr = &EllipsisExpr{
				Pos: yyDollar[1].pos,
			}
		}
	case 175:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1230
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
	case 176:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1244
		{
			yyVAL.kvs = nil
		}
	case 177:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1248
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 178:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1254
		{
			yyVAL.kvs = []*KeyValueExpr{yyDollar[1].kv}
		}
	case 179:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1258
		{
			yyVAL.kvs = append(yyDollar[1].kvs, yyDollar[3].kv)
		}
	case 180:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1264
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
