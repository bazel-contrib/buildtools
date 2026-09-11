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

//line build/parse.y:1312

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
	6, 65,
	-2, 138,
	-1, 187,
	20, 135,
	-2, 136,
}

const yyPrivate = 57344

const yyLast = 1099

var yyAct = [...]int16{
	22, 300, 248, 273, 287, 35, 164, 264, 117, 34,
	170, 7, 247, 115, 217, 2, 178, 104, 124, 265,
	47, 9, 27, 177, 114, 83, 240, 279, 242, 192,
	96, 97, 98, 99, 48, 44, 23, 43, 102, 107,
	110, 56, 146, 58, 59, 231, 191, 222, 92, 249,
	165, 103, 101, 122, 43, 239, 112, 23, 246, 241,
	161, 133, 134, 135, 136, 137, 138, 139, 140, 141,
	142, 143, 144, 145, 13, 147, 148, 149, 150, 151,
	152, 153, 154, 155, 49, 127, 222, 15, 43, 53,
	106, 127, 109, 119, 131, 222, 85, 253, 267, 254,
	55, 86, 127, 253, 123, 254, 129, 175, 180, 23,
	182, 159, 23, 93, 199, 132, 61, 185, 23, 60,
	64, 277, 65, 23, 62, 120, 121, 43, 183, 23,
	94, 15, 130, 119, 268, 309, 193, 229, 42, 128,
	234, 128, 197, 61, 42, 206, 60, 207, 127, 180,
	180, 62, 222, 54, 181, 23, 63, 186, 235, 189,
	316, 118, 184, 326, 167, 218, 95, 306, 210, 158,
	215, 298, 23, 227, 213, 23, 297, 204, 323, 173,
	174, 228, 233, 63, 199, 260, 52, 233, 166, 236,
	238, 226, 50, 196, 230, 84, 52, 281, 198, 15,
	230, 232, 51, 48, 199, 223, 209, 237, 176, 259,
	256, 252, 128, 293, 216, 172, 15, 218, 245, 258,
	320, 271, 272, 270, 39, 274, 220, 31, 222, 30,
	41, 269, 276, 200, 88, 172, 37, 315, 38, 201,
	87, 128, 156, 32, 225, 15, 89, 312, 224, 219,
	169, 23, 40, 46, 257, 278, 93, 280, 28, 36,
	288, 52, 221, 252, 52, 243, 284, 29, 205, 43,
	111, 296, 52, 244, 291, 319, 160, 42, 303, 275,
	166, 301, 292, 52, 128, 305, 128, 7, 222, 211,
	15, 307, 252, 310, 187, 171, 261, 266, 282, 274,
	313, 219, 317, 327, 290, 255, 318, 289, 214, 190,
	212, 305, 100, 113, 195, 322, 321, 288, 252, 1,
	10, 328, 252, 20, 330, 301, 329, 299, 15, 194,
	108, 105, 45, 57, 39, 308, 21, 311, 128, 12,
	41, 8, 128, 314, 4, 33, 37, 179, 38, 168,
	286, 285, 294, 295, 251, 283, 250, 125, 126, 203,
	302, 23, 40, 15, 202, 324, 325, 128, 157, 36,
	16, 24, 262, 263, 90, 91, 162, 266, 128, 43,
	39, 163, 0, 31, 0, 30, 41, 42, 0, 0,
	0, 0, 37, 128, 38, 0, 0, 128, 0, 32,
	128, 128, 6, 0, 302, 11, 0, 23, 40, 26,
	0, 0, 0, 0, 28, 36, 0, 0, 0, 0,
	0, 0, 0, 29, 0, 43, 25, 14, 17, 18,
	19, 39, 304, 42, 31, 5, 30, 41, 0, 0,
	0, 0, 0, 37, 0, 38, 0, 0, 0, 0,
	32, 0, 0, 6, 3, 0, 11, 0, 23, 40,
	26, 0, 0, 0, 0, 28, 36, 0, 0, 0,
	0, 0, 0, 0, 29, 0, 43, 25, 14, 17,
	18, 19, 39, 0, 42, 31, 5, 30, 41, 0,
	0, 0, 0, 0, 37, 0, 38, 0, 0, 0,
	61, 32, 0, 60, 64, 0, 65, 0, 62, 23,
	40, 0, 0, 0, 0, 0, 28, 36, 77, 78,
	0, 0, 0, 0, 0, 29, 0, 43, 0, 14,
	17, 18, 19, 39, 0, 42, 31, 116, 30, 41,
	63, 79, 80, 0, 0, 37, 0, 38, 0, 0,
	0, 0, 32, 0, 0, 0, 0, 0, 0, 0,
	23, 40, 0, 0, 0, 0, 61, 28, 36, 60,
	64, 0, 65, 0, 62, 0, 29, 0, 43, 0,
	14, 17, 18, 19, 77, 39, 42, 220, 31, 0,
	30, 41, 0, 0, 0, 0, 0, 37, 0, 38,
	0, 0, 0, 0, 32, 0, 63, 79, 80, 0,
	0, 0, 23, 40, 0, 0, 0, 0, 0, 28,
	36, 0, 0, 221, 0, 0, 0, 0, 29, 0,
	43, 61, 0, 0, 60, 64, 0, 65, 42, 62,
	188, 66, 0, 67, 0, 0, 0, 0, 76, 77,
	78, 0, 0, 75, 0, 0, 68, 0, 71, 0,
	0, 82, 0, 0, 72, 81, 0, 0, 69, 70,
	0, 63, 79, 80, 39, 73, 74, 31, 222, 30,
	41, 0, 0, 0, 0, 0, 37, 0, 38, 0,
	0, 0, 0, 32, 0, 0, 0, 0, 0, 0,
	0, 23, 40, 0, 0, 0, 0, 0, 28, 36,
	0, 0, 61, 0, 0, 60, 64, 29, 65, 43,
	62, 0, 66, 0, 67, 0, 0, 42, 0, 76,
	77, 78, 0, 0, 75, 0, 0, 68, 0, 71,
	0, 0, 82, 208, 0, 72, 81, 0, 0, 69,
	70, 0, 63, 79, 80, 61, 73, 74, 60, 64,
	0, 65, 0, 62, 0, 66, 0, 67, 0, 0,
	0, 0, 76, 77, 78, 0, 0, 75, 0, 0,
	68, 180, 71, 0, 0, 82, 0, 0, 72, 81,
	0, 0, 69, 70, 0, 63, 79, 80, 61, 73,
	74, 60, 64, 0, 65, 0, 62, 0, 66, 0,
	67, 0, 0, 0, 0, 76, 77, 78, 0, 0,
	75, 0, 0, 68, 0, 71, 0, 0, 82, 0,
	0, 72, 81, 0, 0, 69, 70, 0, 63, 79,
	80, 39, 73, 74, 31, 0, 30, 41, 0, 0,
	0, 0, 0, 37, 0, 38, 0, 0, 0, 0,
	32, 0, 0, 0, 0, 0, 0, 0, 23, 40,
	0, 0, 0, 0, 0, 28, 36, 0, 0, 61,
	0, 0, 60, 64, 29, 65, 43, 62, 0, 66,
	0, 67, 0, 0, 42, 0, 76, 77, 78, 0,
	0, 75, 0, 0, 68, 0, 71, 0, 0, 0,
	0, 0, 72, 81, 0, 0, 69, 70, 0, 63,
	79, 80, 61, 73, 74, 60, 64, 0, 65, 0,
	62, 0, 66, 0, 67, 0, 0, 0, 0, 76,
	77, 78, 0, 0, 75, 0, 0, 68, 0, 71,
	0, 0, 0, 0, 0, 72, 0, 0, 0, 69,
	70, 0, 63, 79, 80, 61, 73, 74, 60, 64,
	0, 65, 0, 62, 0, 66, 0, 67, 0, 0,
	0, 0, 76, 77, 78, 0, 0, 75, 0, 0,
	68, 0, 71, 0, 0, 0, 0, 0, 72, 0,
	0, 0, 69, 70, 0, 63, 79, 80, 61, 73,
	0, 60, 64, 0, 65, 0, 62, 0, 66, 0,
	67, 0, 0, 0, 0, 76, 77, 78, 0, 0,
	0, 0, 0, 68, 0, 71, 61, 0, 0, 60,
	64, 72, 65, 0, 62, 69, 70, 0, 63, 79,
	80, 0, 73, 76, 77, 78, 61, 0, 0, 60,
	64, 0, 65, 0, 62, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 63, 79, 80, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 63, 79, 80,
}

var yyPact = [...]int16{
	-1000, -1000, 426, -1000, -1000, -1000, -25, -1000, -1000, -1000,
	240, 329, -1000, 177, 836, 140, -1000, -1000, -1000, -1000,
	-6, 8, 794, -1000, 178, 64, 836, 229, 123, 836,
	836, 836, 836, -1000, -1000, -1000, 307, 836, 836, 836,
	259, 23, -1000, -1000, -1000, -39, 477, 124, 229, -1000,
	836, 836, 836, 274, 97, -1000, 97, 836, 81, -1000,
	836, 836, 836, 836, 836, 836, 836, 836, 836, 836,
	836, 836, 836, 5, 836, 836, 836, 836, 836, 836,
	836, 836, 836, 227, 25, 178, 263, 28, 271, 836,
	237, 286, -1000, 220, 25, 25, -1000, -1000, -1000, -1000,
	271, 190, 751, 271, 78, 142, 285, 627, 271, 303,
	794, 13, -1000, -31, 528, -1000, -1000, -1000, 836, 329,
	274, 274, 875, 183, -1000, 222, -1000, 97, -1000, 93,
	255, 477, -1000, -1000, -1000, -1000, -1000, 139, 139, 1032,
	1032, 1032, 1032, 1032, 1032, 1032, 836, 961, 1004, 496,
	1052, 562, 112, 112, 918, 708, 97, 280, -1000, 305,
	477, -1000, 302, 271, 580, 279, -1000, 187, 235, 836,
	-1000, 123, 836, -1000, -1000, -13, -1000, 119, 11, -1000,
	329, 669, -1000, 120, -1000, 138, 669, -1000, 836, 669,
	-1000, -1000, -1000, -1000, -1, -32, 252, 229, 836, 97,
	26, 80, 299, 271, 93, 477, -1000, 1032, 836, 93,
	167, 25, 91, -1000, -1000, -1000, 219, -1000, 794, 208,
	836, 836, -1000, -1000, 836, -1000, -1000, 794, 271, -1000,
	11, 836, 84, 794, -1000, -1000, 794, -1000, 627, -1000,
	-33, -1000, -1000, 477, 274, -1000, -1000, 179, -1000, 93,
	-1000, -1000, -1000, 80, -13, -1000, -1000, 143, -1000, 875,
	-1000, -1000, 298, 273, -1000, -1000, 200, 25, 25, -1000,
	836, 794, 794, 158, 794, 77, 875, 836, 375, -1000,
	-1000, -1000, 86, 149, 271, 115, 271, -1000, 234, 93,
	-1000, -1000, 91, 97, 224, 147, 794, -1000, 836, 266,
	-1000, -1000, 205, 875, -1000, -1000, -1000, -1000, 86, -1000,
	-1000, 38, 80, -1000, 163, 97, 97, 145, 297, 4,
	-13, -1000, -1000, 836, 93, 93, -1000, -1000, -1000, -1000,
	794,
}

var yyPgo = [...]int16{
	0, 10, 50, 6, 14, 381, 376, 19, 375, 374,
	7, 373, 372, 371, 370, 25, 368, 364, 359, 49,
	18, 358, 357, 356, 355, 12, 2, 354, 351, 350,
	4, 0, 3, 52, 22, 74, 349, 51, 20, 347,
	16, 23, 84, 345, 9, 15, 344, 341, 339, 336,
	333, 8, 21, 332, 17, 331, 330, 5, 13, 329,
	1, 327, 323, 320, 319, 314, 313,
}

var yyR1 = [...]int8{
	0, 64, 58, 58, 65, 65, 59, 59, 59, 45,
	45, 45, 45, 46, 46, 62, 63, 63, 47, 47,
	47, 49, 49, 48, 48, 50, 50, 51, 53, 53,
	52, 52, 52, 52, 52, 52, 52, 52, 52, 52,
	52, 13, 14, 15, 15, 16, 16, 66, 66, 34,
	34, 34, 34, 34, 34, 34, 34, 34, 34, 34,
	34, 34, 34, 34, 34, 6, 6, 5, 5, 4,
	4, 4, 4, 61, 61, 60, 60, 9, 9, 12,
	12, 8, 8, 11, 11, 7, 7, 7, 7, 7,
	10, 10, 10, 10, 10, 35, 35, 36, 36, 31,
	31, 31, 31, 31, 31, 31, 31, 31, 31, 31,
	31, 31, 31, 31, 31, 31, 31, 31, 31, 31,
	31, 31, 31, 31, 31, 31, 31, 31, 37, 37,
	32, 32, 33, 33, 1, 1, 2, 2, 3, 3,
	54, 56, 56, 55, 55, 55, 38, 38, 57, 42,
	43, 43, 43, 43, 44, 39, 40, 40, 41, 41,
	17, 17, 18, 18, 19, 19, 20, 20, 20, 22,
	22, 21, 23, 24, 24, 25, 25, 26, 26, 26,
	26, 27, 28, 28, 29, 29, 30,
}

var yyR2 = [...]int8{
	0, 2, 5, 2, 0, 2, 0, 3, 2, 0,
	2, 2, 3, 1, 1, 6, 1, 3, 3, 6,
	1, 4, 5, 1, 4, 2, 1, 4, 0, 3,
	1, 2, 1, 3, 5, 3, 1, 3, 1, 1,
	1, 2, 4, 0, 4, 1, 3, 0, 1, 1,
	1, 1, 1, 3, 8, 4, 4, 6, 8, 3,
	4, 4, 3, 4, 3, 0, 2, 2, 3, 1,
	3, 2, 2, 1, 3, 1, 3, 0, 2, 0,
	2, 1, 3, 1, 3, 1, 3, 2, 1, 2,
	1, 3, 5, 4, 4, 1, 3, 0, 1, 1,
	4, 2, 2, 2, 2, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 4, 3,
	3, 3, 3, 3, 3, 3, 3, 5, 1, 3,
	0, 1, 0, 2, 0, 1, 1, 2, 0, 1,
	3, 1, 3, 0, 1, 2, 1, 3, 1, 1,
	3, 2, 2, 1, 1, 4, 1, 3, 1, 2,
	0, 2, 1, 3, 1, 3, 1, 1, 3, 1,
	3, 4, 3, 0, 2, 1, 3, 1, 1, 1,
	1, 3, 0, 2, 1, 3, 3,
}

var yyChk = [...]int16{
	-1000, -64, -45, 28, -46, 60, 27, -51, -47, -52,
	-63, 30, -48, -35, 52, -42, -14, 53, 54, 55,
	-62, -49, -31, 32, -13, 51, 34, -34, 39, 48,
	10, 8, 24, -43, -44, -57, 40, 17, 19, 5,
	33, 11, 58, 50, 60, -53, 13, -38, -34, -42,
	15, 25, 9, -35, 13, -42, 47, -50, 35, 36,
	7, 4, 12, 44, 8, 10, 14, 16, 29, 41,
	42, 31, 37, 48, 49, 26, 21, 22, 23, 45,
	46, 38, 34, -15, 17, 32, -35, 11, 5, 17,
	-9, -8, -7, -42, 7, 43, -31, -31, -31, -31,
	5, -33, -31, -37, -54, -55, -37, -31, -56, -33,
	-31, 11, 33, -66, 63, -58, 60, -51, 37, 9,
	-35, -35, -31, -19, -20, -22, -21, 5, -42, -19,
	-35, 13, 34, -31, -31, -31, -31, -31, -31, -31,
	-31, -31, -31, -31, -31, -31, 37, -31, -31, -31,
	-31, -31, -31, -31, -31, -31, 15, -16, -42, -15,
	13, 32, -6, -5, -3, -2, 9, -35, -36, 13,
	-1, 9, 15, -42, -42, -3, 18, -41, -40, -39,
	30, -2, -3, -41, 20, -1, -2, 9, 13, -2,
	6, 33, 60, -52, -59, -65, -35, -34, 15, 21,
	11, 17, -17, -18, -19, 13, -58, -31, 35, -19,
	-1, 9, 5, -58, 6, -3, -2, -4, -31, -42,
	7, 43, 9, 18, 13, -35, -7, -31, -57, 18,
	-40, 34, -38, -31, 20, 20, -31, -54, -31, 56,
	27, 60, 60, 13, -35, -20, 32, -25, -26, -19,
	-23, -27, -44, 17, 19, 6, -3, -2, -58, -31,
	18, -42, -12, -11, -10, -7, -42, 7, 43, -4,
	15, -31, -31, -32, -31, -2, -31, 37, -45, 60,
	-58, 18, -2, -24, -25, -28, -29, -30, -57, -19,
	6, -1, 9, 13, -42, -42, -31, 18, 13, -61,
	-60, -57, -42, -31, 57, -26, 18, -3, -2, 20,
	-3, -2, 13, -10, -19, 13, 13, -32, -3, 9,
	15, -30, -26, 15, -19, -19, 18, 6, -60, -57,
	-31,
}

var yyDef = [...]int16{
	9, -2, 0, 1, 10, 11, 0, 13, 14, 28,
	0, 0, 20, 30, 32, 49, 36, 38, 39, 40,
	16, 23, 95, 149, 43, 0, 0, 99, 77, 0,
	0, 0, 0, 50, 51, 52, 0, 132, 143, 132,
	153, 0, 154, 148, 12, 47, 0, 0, 146, 49,
	0, 0, 0, 31, 0, 41, 0, 0, 0, 26,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 43, 0, 0, -2, 97,
	0, 134, 81, 85, 88, 0, 101, 102, 103, 104,
	138, 0, 128, 138, 141, 0, 134, 128, 144, 0,
	128, 151, 152, 0, 48, 18, 6, 4, 0, 0,
	33, 37, 96, 35, 164, 166, 167, 160, 169, 17,
	0, 0, 25, 105, 106, 107, 108, 109, 110, 111,
	112, 113, 114, 115, 116, 117, 0, 119, 120, 121,
	122, 123, 124, 125, 126, 0, 0, 134, 45, 0,
	0, 53, 0, 138, 0, 139, 136, 98, 0, 0,
	78, 135, 0, 87, 89, 0, 59, 0, 158, 156,
	0, 139, 133, 0, 62, 0, 0, -2, 0, 145,
	64, 150, 27, 29, 0, 3, 0, 147, 0, 0,
	0, 0, 0, 138, 162, 0, 24, 118, 0, 42,
	0, 135, 79, 21, 55, 66, 139, 67, 69, 49,
	0, 0, 137, 56, 130, 100, 82, 86, 0, 60,
	159, 0, 0, 129, 61, 63, 140, 142, 0, 9,
	0, 8, 5, 0, 34, 165, 170, 0, 175, 177,
	178, 179, 180, 173, 182, 168, 161, 139, 22, 127,
	44, 46, 0, 134, 83, 90, 85, 88, 0, 68,
	0, 71, 72, 0, 131, 0, 157, 0, 0, 7,
	19, 171, 0, 0, 138, 0, 138, 184, 0, 163,
	15, 80, 135, 0, 87, 89, 70, 57, 130, 138,
	73, 75, 0, 155, 2, 176, 172, 174, 139, 181,
	183, 139, 0, 84, 91, 0, 0, 0, 0, 136,
	0, 185, 186, 0, 93, 94, 58, 54, 74, 76,
	92,
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
//line build/parse.y:238
		{
			yylex.(*input).file = &File{Stmt: yyDollar[1].exprs}
			return 0
		}
	case 2:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:245
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
//line build/parse.y:265
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 6:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:273
		{
			yyVAL.exprs = nil
			yyVAL.lastStmt = nil
		}
	case 7:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:278
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
//line build/parse.y:290
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = nil
		}
	case 9:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:296
		{
			yyVAL.exprs = nil
			yyVAL.lastStmt = nil
		}
	case 10:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:301
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
//line build/parse.y:332
		{
			// Blank line; sever last rule from future comments.
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = nil
		}
	case 12:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:338
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
//line build/parse.y:359
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = yyDollar[1].exprs[len(yyDollar[1].exprs)-1]
		}
	case 14:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:364
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
//line build/parse.y:378
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
//line build/parse.y:395
		{
			yyDollar[1].def_header.Type = yyDollar[3].expr
			yyVAL.def_header = yyDollar[1].def_header
		}
	case 18:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:402
		{
			yyDollar[1].def_header.Function.Body = yyDollar[3].exprs
			yyDollar[1].def_header.ColonPos = &End{Pos: yyDollar[2].pos}
			yyVAL.expr = yyDollar[1].def_header
			yyVAL.lastStmt = yyDollar[3].lastStmt
		}
	case 19:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:409
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
//line build/parse.y:419
		{
			yyVAL.expr = yyDollar[1].ifstmt
			yyVAL.lastStmt = yyDollar[1].lastStmt
		}
	case 21:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:427
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
//line build/parse.y:436
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
//line build/parse.y:457
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
//line build/parse.y:474
		{
			yyVAL.exprs = append([]Expr{yyDollar[1].expr}, yyDollar[2].exprs...)
			yyVAL.lastStmt = yyVAL.exprs[len(yyVAL.exprs)-1]
		}
	case 28:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:480
		{
			yyVAL.exprs = []Expr{}
		}
	case 29:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:484
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 31:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:491
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
				Result: yyDollar[2].expr,
			}
		}
	case 32:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:498
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
			}
		}
	case 33:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:503
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 34:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:504
		{
			yyVAL.expr = binary(typed(yyDollar[1].expr, yyDollar[3].expr), yyDollar[4].pos, yyDollar[4].tok, yyDollar[5].expr)
		}
	case 35:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:505
		{
			yyVAL.expr = typed(yyDollar[1].expr, yyDollar[3].expr)
		}
	case 37:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:507
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 38:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:509
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 39:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:516
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 40:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:523
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 41:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:534
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
//line build/parse.y:549
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
//line build/parse.y:561
		{
			yyVAL.expr = nil
		}
	case 44:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:565
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
//line build/parse.y:576
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 46:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:580
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 52:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:592
		{
			yyVAL.expr = yyDollar[1].string
		}
	case 53:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:596
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
//line build/parse.y:605
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
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:619
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
	case 56:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:630
		{
			yyVAL.expr = &IndexExpr{
				X:          yyDollar[1].expr,
				IndexStart: yyDollar[2].pos,
				Y:          yyDollar[3].expr,
				End:        yyDollar[4].pos,
			}
		}
	case 57:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:639
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
	case 58:
		yyDollar = yyS[yypt-8 : yypt+1]
//line build/parse.y:650
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
	case 59:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:663
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 60:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:672
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
	case 61:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:683
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
	case 62:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:694
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
	case 63:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:707
		{
			yyVAL.expr = &SetExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[4].pos),
			}
		}
	case 64:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:716
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
	case 65:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:737
		{
			yyVAL.exprs = nil
		}
	case 66:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:741
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 67:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:747
		{
			yyVAL.exprs = []Expr{yyDollar[2].expr}
		}
	case 68:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:751
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 70:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:758
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 71:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:762
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 72:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:766
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 73:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:771
		{
			yyVAL.loadargs = []*struct {
				from Ident
				to   Ident
			}{yyDollar[1].loadarg}
		}
	case 74:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:775
		{
			yyDollar[1].loadargs = append(yyDollar[1].loadargs, yyDollar[3].loadarg)
			yyVAL.loadargs = yyDollar[1].loadargs
		}
	case 75:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:781
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
	case 76:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:798
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
	case 77:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:813
		{
			yyVAL.exprs = nil
		}
	case 78:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:817
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 79:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:822
		{
			yyVAL.exprs = nil
		}
	case 80:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:826
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 81:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:832
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 82:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:836
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 83:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:843
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 84:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:847
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 86:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:854
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 87:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:858
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 88:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:862
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, nil)
		}
	case 89:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:866
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 91:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:875
		{
			yyVAL.expr = typed(yyDollar[1].expr, yyDollar[3].expr)
		}
	case 92:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:879
		{
			yyVAL.expr = binary(typed(yyDollar[1].expr, yyDollar[3].expr), yyDollar[4].pos, yyDollar[4].tok, yyDollar[5].expr)
		}
	case 93:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:883
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, typed(yyDollar[2].expr, yyDollar[4].expr))
		}
	case 94:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:887
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, typed(yyDollar[2].expr, yyDollar[4].expr))
		}
	case 96:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:894
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
	case 97:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:909
		{
			yyVAL.expr = nil
		}
	case 100:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:917
		{
			yyVAL.expr = &LambdaExpr{
				Function: Function{
					StartPos: yyDollar[1].pos,
					Params:   yyDollar[2].exprs,
					Body:     []Expr{yyDollar[4].expr},
				},
			}
		}
	case 101:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:926
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 102:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:927
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 103:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:928
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 104:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:929
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 105:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:930
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 106:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:931
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 107:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:932
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 108:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:933
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 109:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:934
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 110:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:935
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 111:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:936
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 112:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:937
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 113:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:938
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 114:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:939
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 115:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:940
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 116:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:941
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 117:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:942
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 118:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:943
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "not in", yyDollar[4].expr)
		}
	case 119:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:944
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 120:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:945
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 121:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:946
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 122:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:947
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 123:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:948
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 124:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:949
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 125:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:950
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 126:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:952
		{
			if b, ok := yyDollar[3].expr.(*UnaryExpr); ok && b.Op == "not" {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "is not", b.X)
			} else {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
			}
		}
	case 127:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:960
		{
			yyVAL.expr = &ConditionalExpr{
				Then:      yyDollar[1].expr,
				IfStart:   yyDollar[2].pos,
				Test:      yyDollar[3].expr,
				ElseStart: yyDollar[4].pos,
				Else:      yyDollar[5].expr,
			}
		}
	case 128:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:972
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 129:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:976
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 130:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:981
		{
			yyVAL.expr = nil
		}
	case 132:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:987
		{
			yyVAL.exprs, yyVAL.comma = nil, Position{}
		}
	case 133:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:991
		{
			yyVAL.exprs, yyVAL.comma = yyDollar[1].exprs, yyDollar[2].pos
		}
	case 134:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1001
		{
			yyVAL.pos = Position{}
		}
	case 137:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1012
		{
			yyVAL.pos = yyDollar[1].pos
		}
	case 138:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1020
		{
			yyVAL.pos = Position{}
		}
	case 140:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1027
		{
			yyVAL.kv = &KeyValueExpr{
				Key:   yyDollar[1].expr,
				Colon: yyDollar[2].pos,
				Value: yyDollar[3].expr,
			}
		}
	case 141:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1037
		{
			yyVAL.kvs = []*KeyValueExpr{yyDollar[1].kv}
		}
	case 142:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1041
		{
			yyVAL.kvs = append(yyDollar[1].kvs, yyDollar[3].kv)
		}
	case 143:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1046
		{
			yyVAL.kvs = nil
		}
	case 144:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1050
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 145:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1054
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 147:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1061
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
	case 148:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1077
		{
			yyVAL.string = &StringExpr{
				Start:       yyDollar[1].pos,
				Value:       yyDollar[1].str,
				TripleQuote: yyDollar[1].triple,
				End:         yyDollar[1].pos.add(yyDollar[1].tok),
				Token:       yyDollar[1].tok,
			}
		}
	case 149:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1089
		{
			yyVAL.expr = &Ident{NamePos: yyDollar[1].pos, Name: yyDollar[1].tok}
		}
	case 150:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1095
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok + "." + yyDollar[3].tok}
		}
	case 151:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1099
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok + "."}
		}
	case 152:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1103
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: "." + yyDollar[2].tok}
		}
	case 153:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1107
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok}
		}
	case 154:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1113
		{
			yyVAL.expr = &EllipsisExpr{
				Pos: yyDollar[1].pos,
			}
		}
	case 155:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:1121
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
//line build/parse.y:1132
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 157:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1136
		{
			yyVAL.exprs = append(yyDollar[1].exprs, &IfClause{
				If:   yyDollar[2].pos,
				Cond: yyDollar[3].expr,
			})
		}
	case 158:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1145
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 159:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1149
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[2].exprs...)
		}
	case 160:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1154
		{
			yyVAL.exprs = nil
		}
	case 161:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1158
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 162:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1164
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 163:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1168
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 164:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1174
		{
			yyVAL.expr = yyDollar[1].expr
		}
	case 165:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1178
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 168:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1186
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
	case 170:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1210
		{
			yyVAL.expr = &DotExpr{
				X:       yyDollar[1].expr,
				Dot:     yyDollar[2].pos,
				NamePos: yyDollar[3].pos,
				Name:    yyDollar[3].tok,
			}
		}
	case 171:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:1221
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
	case 172:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1234
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 173:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1244
		{
			yyVAL.exprs = nil
		}
	case 174:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1248
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 175:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1254
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 176:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1258
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 181:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1270
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
	case 182:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1284
		{
			yyVAL.kvs = nil
		}
	case 183:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1288
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 184:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1294
		{
			yyVAL.kvs = []*KeyValueExpr{yyDollar[1].kv}
		}
	case 185:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1298
		{
			yyVAL.kvs = append(yyDollar[1].kvs, yyDollar[3].kv)
		}
	case 186:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1304
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
