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
	idents  []*Ident
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

//line build/parse.y:1267

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
		Ident: x.(*Ident),
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
	-1, 83,
	6, 63,
	-2, 136,
	-1, 180,
	20, 133,
	-2, 134,
}

const yyPrivate = 57344

const yyLast = 1074

var yyAct = [...]int16{
	22, 33, 238, 239, 277, 291, 262, 112, 253, 237,
	7, 163, 207, 119, 2, 171, 26, 44, 110, 254,
	99, 170, 123, 9, 158, 109, 268, 232, 45, 91,
	92, 93, 94, 185, 41, 23, 97, 102, 105, 40,
	53, 212, 142, 114, 55, 56, 221, 87, 157, 184,
	117, 96, 230, 40, 107, 118, 98, 125, 129, 130,
	131, 132, 133, 134, 135, 136, 137, 138, 139, 140,
	141, 266, 143, 144, 145, 146, 147, 148, 149, 150,
	151, 229, 40, 242, 242, 231, 13, 212, 46, 104,
	51, 15, 23, 101, 127, 244, 244, 245, 245, 236,
	154, 50, 256, 152, 52, 80, 173, 212, 192, 23,
	23, 23, 81, 178, 58, 128, 88, 57, 61, 224,
	62, 176, 59, 174, 114, 300, 179, 23, 182, 173,
	23, 190, 15, 186, 115, 116, 243, 243, 257, 314,
	122, 126, 122, 200, 168, 192, 199, 175, 40, 225,
	219, 89, 113, 191, 60, 76, 77, 58, 208, 192,
	57, 61, 173, 62, 289, 59, 217, 177, 49, 288,
	218, 160, 203, 317, 298, 223, 23, 213, 166, 167,
	223, 206, 226, 228, 216, 58, 220, 90, 57, 83,
	45, 222, 220, 59, 296, 82, 279, 60, 15, 246,
	189, 84, 250, 227, 49, 205, 235, 208, 247, 169,
	47, 260, 261, 197, 124, 263, 15, 249, 193, 258,
	48, 311, 265, 37, 194, 60, 284, 259, 165, 39,
	58, 195, 165, 57, 61, 35, 62, 36, 59, 106,
	307, 49, 15, 264, 267, 233, 209, 278, 74, 215,
	23, 38, 269, 88, 274, 49, 306, 49, 34, 198,
	287, 153, 271, 303, 282, 214, 292, 294, 40, 162,
	60, 76, 77, 43, 297, 7, 310, 159, 234, 283,
	49, 122, 248, 122, 122, 212, 270, 15, 305, 180,
	263, 255, 304, 164, 318, 209, 308, 281, 272, 271,
	204, 302, 183, 202, 278, 95, 313, 312, 108, 188,
	315, 316, 292, 320, 1, 321, 319, 10, 20, 290,
	187, 103, 15, 299, 58, 301, 100, 57, 61, 42,
	62, 54, 59, 122, 63, 21, 64, 280, 12, 309,
	8, 73, 74, 75, 4, 285, 286, 32, 172, 65,
	161, 68, 276, 293, 275, 241, 15, 69, 273, 240,
	122, 66, 67, 120, 60, 76, 77, 121, 70, 196,
	16, 251, 255, 122, 252, 37, 85, 86, 30, 155,
	29, 39, 156, 0, 0, 0, 0, 35, 0, 36,
	0, 0, 122, 0, 31, 122, 122, 6, 0, 293,
	11, 0, 23, 38, 25, 0, 0, 0, 0, 27,
	34, 0, 0, 0, 0, 0, 0, 0, 28, 0,
	40, 24, 14, 17, 18, 19, 37, 295, 0, 30,
	5, 29, 39, 0, 0, 0, 0, 0, 35, 0,
	36, 0, 0, 0, 0, 31, 0, 0, 6, 3,
	0, 11, 0, 23, 38, 25, 0, 0, 0, 0,
	27, 34, 0, 0, 0, 0, 0, 0, 0, 28,
	0, 40, 24, 14, 17, 18, 19, 37, 0, 0,
	30, 5, 29, 39, 0, 0, 0, 0, 0, 35,
	0, 36, 0, 0, 0, 0, 31, 0, 0, 0,
	0, 0, 0, 0, 23, 38, 0, 0, 0, 0,
	0, 27, 34, 0, 0, 0, 0, 0, 0, 0,
	28, 0, 40, 0, 14, 17, 18, 19, 0, 0,
	58, 0, 111, 57, 61, 0, 62, 0, 59, 181,
	63, 0, 64, 0, 0, 0, 0, 73, 74, 75,
	0, 0, 72, 0, 0, 65, 0, 68, 0, 0,
	79, 0, 0, 69, 78, 0, 0, 66, 67, 0,
	60, 76, 77, 58, 70, 71, 57, 61, 0, 62,
	0, 59, 0, 63, 0, 64, 0, 0, 0, 0,
	73, 74, 75, 0, 0, 72, 0, 0, 65, 0,
	68, 0, 0, 79, 201, 0, 69, 78, 0, 0,
	66, 67, 0, 60, 76, 77, 58, 70, 71, 57,
	61, 0, 62, 0, 59, 0, 63, 0, 64, 0,
	0, 0, 0, 73, 74, 75, 0, 0, 72, 0,
	0, 65, 173, 68, 0, 0, 79, 0, 0, 69,
	78, 0, 0, 66, 67, 0, 60, 76, 77, 58,
	70, 71, 57, 61, 0, 62, 0, 59, 0, 63,
	0, 64, 0, 0, 0, 0, 73, 74, 75, 0,
	0, 72, 0, 0, 65, 0, 68, 0, 0, 79,
	0, 0, 69, 78, 0, 0, 66, 67, 0, 60,
	76, 77, 37, 70, 71, 30, 0, 29, 39, 0,
	0, 0, 0, 0, 35, 0, 36, 0, 0, 0,
	0, 31, 0, 0, 0, 0, 0, 0, 0, 23,
	38, 0, 0, 0, 0, 0, 27, 34, 0, 0,
	0, 0, 0, 0, 0, 28, 0, 40, 0, 14,
	17, 18, 19, 58, 0, 0, 57, 61, 0, 62,
	0, 59, 0, 63, 0, 64, 0, 0, 0, 0,
	73, 74, 75, 0, 0, 72, 0, 0, 65, 0,
	68, 0, 0, 0, 0, 0, 69, 78, 0, 0,
	66, 67, 0, 60, 76, 77, 58, 70, 71, 57,
	61, 0, 62, 0, 59, 0, 63, 0, 64, 0,
	0, 0, 0, 73, 74, 75, 0, 0, 72, 0,
	0, 65, 0, 68, 0, 0, 0, 0, 0, 69,
	0, 0, 0, 66, 67, 0, 60, 76, 77, 58,
	70, 71, 57, 61, 0, 62, 0, 59, 0, 63,
	0, 64, 0, 0, 0, 0, 73, 74, 75, 0,
	0, 72, 0, 0, 65, 0, 68, 0, 0, 0,
	0, 0, 69, 0, 0, 0, 66, 67, 0, 60,
	76, 77, 37, 70, 210, 30, 212, 29, 39, 0,
	0, 0, 0, 0, 35, 0, 36, 0, 0, 0,
	0, 31, 0, 0, 0, 0, 0, 0, 0, 23,
	38, 0, 0, 0, 0, 58, 27, 34, 57, 61,
	211, 62, 0, 59, 0, 28, 37, 40, 210, 30,
	0, 29, 39, 74, 75, 0, 0, 0, 35, 0,
	36, 0, 0, 0, 0, 31, 0, 0, 0, 0,
	0, 0, 0, 23, 38, 60, 76, 77, 0, 0,
	27, 34, 37, 0, 211, 30, 212, 29, 39, 28,
	0, 40, 0, 0, 35, 0, 36, 0, 0, 0,
	0, 31, 0, 0, 0, 0, 0, 0, 0, 23,
	38, 0, 0, 0, 0, 0, 27, 34, 37, 0,
	0, 30, 0, 29, 39, 28, 0, 40, 0, 0,
	35, 0, 36, 0, 0, 0, 0, 31, 0, 0,
	0, 0, 0, 0, 0, 23, 38, 0, 0, 0,
	0, 58, 27, 34, 57, 61, 0, 62, 0, 59,
	0, 28, 0, 40, 0, 0, 0, 0, 73, 74,
	75, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 60, 76, 77,
}

var yyPact = [...]int16{
	-1000, -1000, 421, -1000, -1000, -1000, -26, -1000, -1000, -1000,
	260, 218, -1000, 195, 993, 77, -1000, -1000, -1000, -1000,
	-7, 9, 655, -1000, 73, 993, 184, 144, 993, 993,
	993, 993, -1000, -1000, 300, 993, 993, 993, 228, 21,
	-1000, -1000, -38, 472, 115, 184, -1000, 993, 993, 993,
	271, 60, 197, 60, 993, 81, -1000, 993, 993, 993,
	993, 993, 993, 993, 993, 993, 993, 993, 993, 993,
	5, 993, 993, 993, 993, 993, 993, 993, 993, 993,
	197, 248, 68, 268, 993, 256, 284, -1000, 217, 60,
	60, -1000, -1000, -1000, -1000, 268, 191, 612, 268, 76,
	147, 280, 526, 268, 296, 655, 16, -1000, -27, 697,
	-1000, -1000, -1000, 993, 218, 271, 271, 749, 138, -1000,
	207, -1000, -1000, 216, 60, 87, 246, 472, -1000, -1000,
	-1000, -1000, -1000, 181, 181, 1027, 1027, 1027, 1027, 1027,
	1027, 1027, 993, 835, 320, 911, 110, 226, 153, 153,
	792, 569, 298, 472, -1000, 294, 268, 921, 276, -1000,
	159, 252, 993, -1000, 144, 993, -1000, -1000, -11, -1000,
	132, 12, -1000, 218, 957, -1000, 99, -1000, 129, 957,
	-1000, 993, 957, -1000, -1000, -1000, -1000, 25, -33, 232,
	184, 993, 60, 67, 79, 60, 273, -1000, 472, -1000,
	1027, 993, 95, -1000, -1000, -1000, 877, -1000, 655, 212,
	993, 993, -1000, -1000, 993, -1000, -1000, 655, 268, -1000,
	12, 993, 34, 655, -1000, -1000, 655, -1000, 526, -1000,
	-34, -1000, -1000, 472, 271, -1000, -1000, 268, -1000, 87,
	-1000, -1000, 292, -1000, 79, -11, 87, 178, 60, -1000,
	749, 291, 270, -1000, -1000, 213, 60, 60, -1000, 993,
	655, 655, 151, 655, 98, 749, 993, 370, -1000, -1000,
	176, 78, -1000, 156, 268, 105, 268, -1000, 250, -1000,
	-1000, -1000, -1000, 95, 60, 243, 227, 655, -1000, 993,
	267, -1000, -1000, 206, 749, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, 32, 79, -1000, 124, 60, 60, 155, 288,
	3, -11, -1000, -1000, 993, 87, 87, -1000, -1000, -1000,
	-1000, 655,
}

var yyPgo = [...]int16{
	0, 11, 24, 48, 12, 382, 379, 19, 377, 376,
	8, 374, 371, 370, 22, 369, 3, 13, 367, 363,
	359, 358, 9, 2, 355, 354, 352, 4, 0, 6,
	51, 16, 86, 350, 56, 17, 348, 15, 21, 88,
	347, 14, 344, 340, 338, 335, 331, 7, 23, 329,
	20, 326, 321, 1, 18, 320, 5, 319, 318, 317,
	314, 309, 308,
}

var yyR1 = [...]int8{
	0, 60, 54, 54, 61, 61, 55, 55, 55, 41,
	41, 41, 41, 42, 42, 58, 59, 59, 43, 43,
	43, 45, 45, 44, 44, 46, 46, 47, 49, 49,
	48, 48, 48, 48, 48, 48, 48, 48, 48, 48,
	48, 13, 14, 14, 15, 15, 62, 62, 31, 31,
	31, 31, 31, 31, 31, 31, 31, 31, 31, 31,
	31, 31, 31, 6, 6, 5, 5, 4, 4, 4,
	4, 57, 57, 56, 56, 9, 9, 12, 12, 8,
	8, 11, 11, 7, 7, 7, 7, 7, 10, 10,
	10, 10, 10, 32, 32, 33, 33, 28, 28, 28,
	28, 28, 28, 28, 28, 28, 28, 28, 28, 28,
	28, 28, 28, 28, 28, 28, 28, 28, 28, 28,
	28, 28, 28, 28, 28, 28, 34, 34, 29, 29,
	30, 30, 1, 1, 2, 2, 3, 3, 50, 52,
	52, 51, 51, 51, 35, 35, 53, 39, 40, 40,
	40, 40, 36, 37, 37, 38, 38, 16, 16, 17,
	17, 19, 19, 18, 20, 21, 21, 22, 22, 23,
	23, 23, 23, 23, 24, 25, 25, 26, 26, 27,
}

var yyR2 = [...]int8{
	0, 2, 5, 2, 0, 2, 0, 3, 2, 0,
	2, 2, 3, 1, 1, 6, 1, 3, 3, 6,
	1, 4, 5, 1, 4, 2, 1, 4, 0, 3,
	1, 2, 1, 3, 5, 3, 1, 3, 1, 1,
	1, 5, 0, 4, 1, 3, 0, 1, 1, 1,
	1, 3, 8, 4, 4, 6, 8, 3, 4, 4,
	3, 4, 3, 0, 2, 2, 3, 1, 3, 2,
	2, 1, 3, 1, 3, 0, 2, 0, 2, 1,
	3, 1, 3, 1, 3, 2, 1, 2, 1, 3,
	5, 4, 4, 1, 3, 0, 1, 1, 4, 2,
	2, 2, 2, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 4, 3, 3, 3,
	3, 3, 3, 3, 3, 5, 1, 3, 0, 1,
	0, 2, 0, 1, 1, 2, 0, 1, 3, 1,
	3, 0, 1, 2, 1, 3, 1, 1, 3, 2,
	2, 1, 4, 1, 3, 1, 2, 1, 3, 1,
	1, 1, 3, 5, 3, 0, 2, 1, 3, 1,
	1, 1, 2, 1, 3, 0, 2, 1, 3, 3,
}

var yyChk = [...]int16{
	-1000, -60, -41, 28, -42, 60, 27, -47, -43, -48,
	-59, 30, -44, -32, 52, -39, -13, 53, 54, 55,
	-58, -45, -28, 32, 51, 34, -31, 39, 48, 10,
	8, 24, -40, -53, 40, 17, 19, 5, 33, 11,
	50, 60, -49, 13, -35, -31, -39, 15, 25, 9,
	-32, 13, -39, 47, -46, 35, 36, 7, 4, 12,
	44, 8, 10, 14, 16, 29, 41, 42, 31, 37,
	48, 49, 26, 21, 22, 23, 45, 46, 38, 34,
	32, -32, 11, 5, 17, -9, -8, -7, -39, 7,
	43, -28, -28, -28, -28, 5, -30, -28, -34, -50,
	-51, -34, -28, -52, -30, -28, 11, 33, -62, 63,
	-54, 60, -47, 37, 9, -32, -32, -28, -16, -17,
	-19, -18, -39, -14, 17, -16, -32, 13, 34, -28,
	-28, -28, -28, -28, -28, -28, -28, -28, -28, -28,
	-28, -28, 37, -28, -28, -28, -28, -28, -28, -28,
	-28, -28, -14, 13, 32, -6, -5, -3, -2, 9,
	-32, -33, 13, -1, 9, 15, -39, -39, -3, 18,
	-38, -37, -36, 30, -2, -3, -38, 20, -1, -2,
	9, 13, -2, 6, 33, 60, -48, -55, -61, -32,
	-31, 15, 21, 11, 17, 15, -15, -39, 13, -54,
	-28, 35, 5, -54, 6, -3, -2, -4, -28, -39,
	7, 43, 9, 18, 13, -32, -7, -28, -53, 18,
	-37, 34, -35, -28, 20, 20, -28, -50, -28, 56,
	27, 60, 60, 13, -32, -17, 32, -22, -23, -16,
	-20, -24, 5, 58, 17, 19, -16, -1, 9, -54,
	-28, -12, -11, -10, -7, -39, 7, 43, -4, 15,
	-28, -28, -29, -28, -2, -28, 37, -41, 60, -54,
	-3, -2, 6, -21, -22, -25, -26, -27, -53, 18,
	-39, 6, -1, 9, 13, -39, -39, -28, 18, 13,
	-57, -56, -53, -39, -28, 57, 18, -23, 18, -3,
	20, -3, -2, 13, -10, -16, 13, 13, -29, -3,
	9, 15, -27, -23, 15, -16, -16, 18, 6, -56,
	-53, -28,
}

var yyDef = [...]int16{
	9, -2, 0, 1, 10, 11, 0, 13, 14, 28,
	0, 0, 20, 30, 32, 48, 36, 38, 39, 40,
	16, 23, 93, 147, 0, 0, 97, 75, 0, 0,
	0, 0, 49, 50, 0, 130, 141, 130, 151, 0,
	146, 12, 46, 0, 0, 144, 48, 0, 0, 0,
	31, 0, 42, 0, 0, 0, 26, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	42, 0, 0, -2, 95, 0, 132, 79, 83, 86,
	0, 99, 100, 101, 102, 136, 0, 126, 136, 139,
	0, 132, 126, 142, 0, 126, 149, 150, 0, 47,
	18, 6, 4, 0, 0, 33, 37, 94, 35, 157,
	159, 160, 161, 0, 0, 17, 0, 0, 25, 103,
	104, 105, 106, 107, 108, 109, 110, 111, 112, 113,
	114, 115, 0, 117, 118, 119, 120, 121, 122, 123,
	124, 0, 0, 0, 51, 0, 136, 0, 137, 134,
	96, 0, 0, 76, 133, 0, 85, 87, 0, 57,
	0, 155, 153, 0, 137, 131, 0, 60, 0, 0,
	-2, 0, 143, 62, 148, 27, 29, 0, 3, 0,
	145, 0, 0, 0, 0, 0, 132, 44, 0, 24,
	116, 0, 77, 21, 53, 64, 137, 65, 67, 48,
	0, 0, 135, 54, 128, 98, 80, 84, 0, 58,
	156, 0, 0, 127, 59, 61, 138, 140, 0, 9,
	0, 8, 5, 0, 34, 158, 162, 136, 167, 169,
	170, 171, 0, 173, 165, 175, 41, 0, 133, 22,
	125, 0, 132, 81, 88, 83, 86, 0, 66, 0,
	69, 70, 0, 129, 0, 154, 0, 0, 7, 19,
	0, 137, 172, 0, 136, 0, 136, 177, 0, 43,
	45, 15, 78, 133, 0, 85, 87, 68, 55, 128,
	136, 71, 73, 0, 152, 2, 163, 168, 164, 166,
	174, 176, 137, 0, 82, 89, 0, 0, 0, 0,
	134, 0, 178, 179, 0, 91, 92, 56, 52, 72,
	74, 90,
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
				ForceCompact:   forceCompact(yyDollar[4].pos, yyDollar[5].exprs, yyDollar[6].pos),
				ForceMultiLine: forceMultiLine(yyDollar[4].pos, yyDollar[5].exprs, yyDollar[6].pos),
			}
		}
	case 17:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:391
		{
			yyDollar[1].def_header.Type = yyDollar[3].expr
			yyVAL.def_header = yyDollar[1].def_header
		}
	case 18:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:398
		{
			yyDollar[1].def_header.Function.Body = yyDollar[3].exprs
			yyDollar[1].def_header.ColonPos = yyDollar[2].pos
			yyVAL.expr = yyDollar[1].def_header
			yyVAL.lastStmt = yyDollar[3].lastStmt
		}
	case 19:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:405
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
//line build/parse.y:415
		{
			yyVAL.expr = yyDollar[1].ifstmt
			yyVAL.lastStmt = yyDollar[1].lastStmt
		}
	case 21:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:423
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
//line build/parse.y:432
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
//line build/parse.y:453
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
//line build/parse.y:470
		{
			yyVAL.exprs = append([]Expr{yyDollar[1].expr}, yyDollar[2].exprs...)
			yyVAL.lastStmt = yyVAL.exprs[len(yyVAL.exprs)-1]
		}
	case 28:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:476
		{
			yyVAL.exprs = []Expr{}
		}
	case 29:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:480
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 31:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:487
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
				Result: yyDollar[2].expr,
			}
		}
	case 32:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:494
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
			}
		}
	case 33:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:499
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 34:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:500
		{
			yyVAL.expr = binary(typed(yyDollar[1].expr, yyDollar[3].expr), yyDollar[4].pos, yyDollar[4].tok, yyDollar[5].expr)
		}
	case 35:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:501
		{
			yyVAL.expr = typed(yyDollar[1].expr, yyDollar[3].expr)
		}
	case 37:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:503
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 38:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:505
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 39:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:512
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 40:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:519
		{
			yyVAL.expr = &BranchStmt{
				Token:    yyDollar[1].tok,
				TokenPos: yyDollar[1].pos,
			}
		}
	case 41:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:528
		{
			if yyDollar[1].expr.(*Ident).Name != "type" {
				// Match
				_, end := yyDollar[2].expr.Span()
				errorAt(yylex, end, "syntax error near "+yyDollar[2].expr.(*Ident).Name)
			}
			ystart, _ := yyDollar[5].expr.Span()
			yyVAL.expr = &TypeAliasStmt{
				TypePos:    yyDollar[1].expr.(*Ident).NamePos,
				Name:       *yyDollar[2].expr.(*Ident),
				TypeParams: yyDollar[3].expr,
				EqualPos:   yyDollar[4].pos,
				Type:       yyDollar[5].expr,
				LineBreak:  yyDollar[4].pos.Line < ystart.Line,
			}
		}
	case 42:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:546
		{
			yyVAL.expr = nil
		}
	case 43:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:550
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[4].pos),
			}
		}
	case 44:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:561
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 45:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:565
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 50:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:576
		{
			yyVAL.expr = yyDollar[1].string
		}
	case 51:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:580
		{
			yyVAL.expr = &DotExpr{
				X:       yyDollar[1].expr,
				Dot:     yyDollar[2].pos,
				NamePos: yyDollar[3].pos,
				Name:    yyDollar[3].tok,
			}
		}
	case 52:
		yyDollar = yyS[yypt-8 : yypt+1]
//line build/parse.y:589
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
	case 53:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:603
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
	case 54:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:614
		{
			yyVAL.expr = &IndexExpr{
				X:          yyDollar[1].expr,
				IndexStart: yyDollar[2].pos,
				Y:          yyDollar[3].expr,
				End:        yyDollar[4].pos,
			}
		}
	case 55:
		yyDollar = yyS[yypt-6 : yypt+1]
//line build/parse.y:623
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
	case 56:
		yyDollar = yyS[yypt-8 : yypt+1]
//line build/parse.y:634
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
	case 57:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:647
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 58:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:656
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
	case 59:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:667
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
	case 60:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:678
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
	case 61:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:691
		{
			yyVAL.expr = &SetExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[4].pos),
			}
		}
	case 62:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:700
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
	case 63:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:721
		{
			yyVAL.exprs = nil
		}
	case 64:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:725
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 65:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:731
		{
			yyVAL.exprs = []Expr{yyDollar[2].expr}
		}
	case 66:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:735
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 68:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:742
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 69:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:746
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 70:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:750
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 71:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:755
		{
			yyVAL.loadargs = []*struct {
				from Ident
				to   Ident
			}{yyDollar[1].loadarg}
		}
	case 72:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:759
		{
			yyDollar[1].loadargs = append(yyDollar[1].loadargs, yyDollar[3].loadarg)
			yyVAL.loadargs = yyDollar[1].loadargs
		}
	case 73:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:765
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
	case 74:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:782
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
	case 75:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:797
		{
			yyVAL.exprs = nil
		}
	case 76:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:801
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 77:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:806
		{
			yyVAL.exprs = nil
		}
	case 78:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:810
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 79:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:816
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 80:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:820
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 81:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:827
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 82:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:831
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 84:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:838
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 85:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:842
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 86:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:846
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, nil)
		}
	case 87:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:850
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 89:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:859
		{
			yyVAL.expr = typed(yyDollar[1].expr, yyDollar[3].expr)
		}
	case 90:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:863
		{
			yyVAL.expr = binary(typed(yyDollar[1].expr, yyDollar[3].expr), yyDollar[4].pos, yyDollar[4].tok, yyDollar[5].expr)
		}
	case 91:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:867
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, typed(yyDollar[2].expr, yyDollar[4].expr))
		}
	case 92:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:871
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, typed(yyDollar[2].expr, yyDollar[4].expr))
		}
	case 94:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:878
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
	case 95:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:893
		{
			yyVAL.expr = nil
		}
	case 98:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:901
		{
			yyVAL.expr = &LambdaExpr{
				Function: Function{
					StartPos: yyDollar[1].pos,
					Params:   yyDollar[2].exprs,
					Body:     []Expr{yyDollar[4].expr},
				},
			}
		}
	case 99:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:910
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 100:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:911
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 101:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:912
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 102:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:913
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 103:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:914
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 104:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:915
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 105:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:916
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 106:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:917
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 107:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:918
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 108:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:919
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 109:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:920
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 110:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:921
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 111:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:922
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 112:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:923
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 113:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:924
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 114:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:925
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 115:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:926
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 116:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:927
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "not in", yyDollar[4].expr)
		}
	case 117:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:928
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 118:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:929
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 119:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:930
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 120:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:931
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 121:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:932
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 122:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:933
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 123:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:934
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 124:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:936
		{
			if b, ok := yyDollar[3].expr.(*UnaryExpr); ok && b.Op == "not" {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "is not", b.X)
			} else {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
			}
		}
	case 125:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:944
		{
			yyVAL.expr = &ConditionalExpr{
				Then:      yyDollar[1].expr,
				IfStart:   yyDollar[2].pos,
				Test:      yyDollar[3].expr,
				ElseStart: yyDollar[4].pos,
				Else:      yyDollar[5].expr,
			}
		}
	case 126:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:956
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 127:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:960
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 128:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:965
		{
			yyVAL.expr = nil
		}
	case 130:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:971
		{
			yyVAL.exprs, yyVAL.comma = nil, Position{}
		}
	case 131:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:975
		{
			yyVAL.exprs, yyVAL.comma = yyDollar[1].exprs, yyDollar[2].pos
		}
	case 132:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:985
		{
			yyVAL.pos = Position{}
		}
	case 135:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:996
		{
			yyVAL.pos = yyDollar[1].pos
		}
	case 136:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1004
		{
			yyVAL.pos = Position{}
		}
	case 138:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1011
		{
			yyVAL.kv = &KeyValueExpr{
				Key:   yyDollar[1].expr,
				Colon: yyDollar[2].pos,
				Value: yyDollar[3].expr,
			}
		}
	case 139:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1021
		{
			yyVAL.kvs = []*KeyValueExpr{yyDollar[1].kv}
		}
	case 140:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1025
		{
			yyVAL.kvs = append(yyDollar[1].kvs, yyDollar[3].kv)
		}
	case 141:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1030
		{
			yyVAL.kvs = nil
		}
	case 142:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1034
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 143:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1038
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 145:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1045
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
	case 146:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1061
		{
			yyVAL.string = &StringExpr{
				Start:       yyDollar[1].pos,
				Value:       yyDollar[1].str,
				TripleQuote: yyDollar[1].triple,
				End:         yyDollar[1].pos.add(yyDollar[1].tok),
				Token:       yyDollar[1].tok,
			}
		}
	case 147:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1073
		{
			yyVAL.expr = &Ident{NamePos: yyDollar[1].pos, Name: yyDollar[1].tok}
		}
	case 148:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1079
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok + "." + yyDollar[3].tok}
		}
	case 149:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1083
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok + "."}
		}
	case 150:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1087
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: "." + yyDollar[2].tok}
		}
	case 151:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1091
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok}
		}
	case 152:
		yyDollar = yyS[yypt-4 : yypt+1]
//line build/parse.y:1097
		{
			yyVAL.expr = &ForClause{
				For:  yyDollar[1].pos,
				Vars: yyDollar[2].expr,
				In:   yyDollar[3].pos,
				X:    yyDollar[4].expr,
			}
		}
	case 153:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1108
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 154:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1112
		{
			yyVAL.exprs = append(yyDollar[1].exprs, &IfClause{
				If:   yyDollar[2].pos,
				Cond: yyDollar[3].expr,
			})
		}
	case 155:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1121
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 156:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1125
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[2].exprs...)
		}
	case 157:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1131
		{
			yyVAL.expr = yyDollar[1].expr
		}
	case 158:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1135
		{
			if te, ok := yyDollar[1].expr.(*TypeExpr); ok {
				te.List = append(te.List, yyDollar[3].expr)
				yyVAL.expr = te
			} else {
				yyVAL.expr = &TypeExpr{
					List: []Expr{yyDollar[1].expr, yyDollar[3].expr},
				}
			}
		}
	case 162:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1153
		{
			yyVAL.expr = &DotExpr{
				X:       yyDollar[1].expr,
				Dot:     yyDollar[2].pos,
				NamePos: yyDollar[3].pos,
				Name:    yyDollar[3].tok,
			}
		}
	case 163:
		yyDollar = yyS[yypt-5 : yypt+1]
//line build/parse.y:1164
		{
			yyVAL.expr = &TypeAppExpr{
				Type:           yyDollar[1].expr,
				ArgsStart:      yyDollar[2].pos,
				Args:           yyDollar[3].exprs,
				End:            End{Pos: yyDollar[5].pos},
				ForceCompact:   forceCompact(yyDollar[2].pos, yyDollar[3].exprs, yyDollar[5].pos),
				ForceMultiLine: forceMultiLine(yyDollar[2].pos, yyDollar[3].exprs, yyDollar[5].pos),
			}
		}
	case 164:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1177
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 165:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1187
		{
			yyVAL.exprs = nil
		}
	case 166:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1191
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 167:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1197
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 168:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1201
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 172:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1210
		{
			yyVAL.expr = &TupleExpr{
				Start: yyDollar[1].pos,
				End:   End{Pos: yyDollar[2].pos},
			}
		}
	case 173:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1217
		{
			yyVAL.expr = &EllipsisExpr{
				Pos: yyDollar[1].pos,
			}
		}
	case 174:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1225
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
	case 175:
		yyDollar = yyS[yypt-0 : yypt+1]
//line build/parse.y:1239
		{
			yyVAL.kvs = nil
		}
	case 176:
		yyDollar = yyS[yypt-2 : yypt+1]
//line build/parse.y:1243
		{
			yyVAL.kvs = yyDollar[1].kvs
		}
	case 177:
		yyDollar = yyS[yypt-1 : yypt+1]
//line build/parse.y:1249
		{
			yyVAL.kvs = []*KeyValueExpr{yyDollar[1].kv}
		}
	case 178:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1253
		{
			yyVAL.kvs = append(yyDollar[1].kvs, yyDollar[3].kv)
		}
	case 179:
		yyDollar = yyS[yypt-3 : yypt+1]
//line build/parse.y:1259
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
