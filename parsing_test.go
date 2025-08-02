// Frontend compiler phase tests
//
// Tests lexing (source text → tokens) and parsing (tokens → AST).
// Covers token recognition, syntax parsing, and type annotation during parsing.

package main

import (
	"strings"
	"testing"

	"github.com/nalgeon/be"
)

// =============================================================================
// LEXING TESTS
// =============================================================================

func lexInput(inputStr string) *Lexer {
	input := []byte(inputStr + "\x00") // trailing null byte
	l := NewLexer(input)
	l.NextToken()
	return l
}

func TestIntLiteral(t *testing.T) {
	l := lexInput("12345")
	be.Equal(t, l.CurrTokenType, INT)
	be.Equal(t, l.CurrLiteral, "12345")
	be.Equal(t, l.CurrIntValue, int64(12345))
}

func TestIdentifier(t *testing.T) {
	l := lexInput("foobar")
	be.Equal(t, l.CurrTokenType, IDENT)
	be.Equal(t, l.CurrLiteral, "foobar")
}

func TestStringLiteral(t *testing.T) {
	l := lexInput("\"hello\"")
	be.Equal(t, l.CurrTokenType, STRING)
	be.Equal(t, l.CurrLiteral, "hello")
}

func TestCharLiteral(t *testing.T) {
	l := lexInput("'a'")
	be.Equal(t, l.CurrTokenType, CHAR)
	be.Equal(t, l.CurrLiteral, "'a'")
}

func TestDelimiters(t *testing.T) {
	tests := []struct {
		input string
		typ   TokenType
	}{
		{"(", LPAREN},
		{")", RPAREN},
		{"{", LBRACE},
		{"}", RBRACE},
		{"[", LBRACKET},
		{"]", RBRACKET},
		{",", COMMA},
		{";", SEMICOLON},
		{":", COLON},
		{".", DOT},
		{"...", ELLIPSIS},
	}

	for _, tt := range tests {
		l := lexInput(tt.input)
		be.Equal(t, l.CurrTokenType, tt.typ)
	}
}

func TestOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"=", ASSIGN},
		{"+", PLUS},
		{"-", MINUS},
		{"!", BANG},
		{"*", ASTERISK},
		{"/", SLASH},
		{"%", PERCENT},
		{"==", EQ},
		{"!=", NOT_EQ},
		{"<", LT},
		{">", GT},
		{"<=", LE},
		{">=", GE},
		{"&&", AND},
		{"||", OR},
		{"&", BIT_AND},
		{"|", BIT_OR},
		{"^", XOR},
		{"<<", SHL},
		{">>", SHR},
		{"&^", AND_NOT},
		{"++", PLUS_PLUS},
		{"--", MINUS_MINUS},
		{":=", DECLARE},
	}

	for _, tt := range tests {
		l := lexInput(tt.input)
		be.Equal(t, l.CurrTokenType, tt.expected)
	}
}

func TestKeywords(t *testing.T) {
	tests := []struct {
		input string
		typ   TokenType
	}{
		{"if", IF},
		{"else", ELSE},
		{"for", FOR},
		{"func", FUNC},
		{"return", RETURN},
		{"var", VAR},
		{"const", CONST},
		{"type", TYPE},
		{"struct", STRUCT},
		{"package", PACKAGE},
		{"import", IMPORT},
		{"break", BREAK},
		{"continue", CONTINUE},
		{"switch", SWITCH},
		{"case", CASE},
		{"default", DEFAULT},
		{"select", SELECT},
		{"go", GO},
		{"defer", DEFER},
		{"fallthrough", FALLTHROUGH},
		{"map", MAP},
		{"range", RANGE},
		{"interface", INTERFACE},
		{"chan", CHAN},
		{"goto", GOTO},
	}

	for _, tt := range tests {
		l := lexInput(tt.input)
		be.Equal(t, l.CurrTokenType, tt.typ)
		be.Equal(t, l.CurrLiteral, tt.input)
	}
}

func TestMultipleTokens(t *testing.T) {
	input := []byte("func main() { x := 42; }\x00")
	l := NewLexer(input)

	expectedTokens := []struct {
		typ     TokenType
		literal string
	}{
		{FUNC, "func"},
		{IDENT, "main"},
		{LPAREN, "("},
		{RPAREN, ")"},
		{LBRACE, "{"},
		{IDENT, "x"},
		{DECLARE, ":="},
		{INT, "42"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{EOF, ""},
	}

	for _, expected := range expectedTokens {
		l.NextToken()
		be.Equal(t, l.CurrTokenType, expected.typ)
		be.Equal(t, l.CurrLiteral, expected.literal)
		if expected.typ == INT {
			be.Equal(t, l.CurrIntValue, int64(42))
		}
	}
}

func TestLineComment(t *testing.T) {
	input := []byte("x // this is a comment\ny\x00")
	l := NewLexer(input)

	l.NextToken()
	be.Equal(t, l.CurrTokenType, IDENT)
	be.Equal(t, l.CurrLiteral, "x")

	l.NextToken()
	be.Equal(t, l.CurrTokenType, IDENT)
	be.Equal(t, l.CurrLiteral, "y")

	l.NextToken()
	be.Equal(t, l.CurrTokenType, EOF)
}

func TestBlockComment(t *testing.T) {
	input := []byte("x /* this is a\nmultiline comment */ y\x00")
	l := NewLexer(input)

	l.NextToken()
	be.Equal(t, l.CurrTokenType, IDENT)
	be.Equal(t, l.CurrLiteral, "x")

	l.NextToken()
	be.Equal(t, l.CurrTokenType, IDENT)
	be.Equal(t, l.CurrLiteral, "y")

	l.NextToken()
	be.Equal(t, l.CurrTokenType, EOF)
}

func TestCommentsWithTokens(t *testing.T) {
	input := []byte("func // comment\n main() /* comment */ {\x00")
	l := NewLexer(input)

	expectedTokens := []TokenType{FUNC, IDENT, LPAREN, RPAREN, LBRACE, EOF}
	expectedLiterals := []string{"func", "main", "(", ")", "{", ""}

	for i, expected := range expectedTokens {
		l.NextToken()
		be.Equal(t, l.CurrTokenType, expected)
		be.Equal(t, l.CurrLiteral, expectedLiterals[i])
	}
}

func TestWhitespace(t *testing.T) {
	tests := []struct {
		input string
		desc  string
	}{
		{"  x  y  ", "spaces"},
		{"\tx\ty\t", "tabs"},
		{"\nx\ny\n", "newlines"},
		{"\r\nx\r\ny\r\n", "carriage returns"},
		{" \t\n\r x \t\n\r y \t\n\r ", "mixed whitespace"},
	}

	for _, tt := range tests {
		input := []byte(tt.input + "\x00")
		l := NewLexer(input)

		l.NextToken()
		be.Equal(t, l.CurrTokenType, IDENT)
		be.Equal(t, l.CurrLiteral, "x")

		l.NextToken()
		be.Equal(t, l.CurrTokenType, IDENT)
		be.Equal(t, l.CurrLiteral, "y")

		l.NextToken()
		be.Equal(t, l.CurrTokenType, EOF)
	}
}

func TestEOF(t *testing.T) {
	tests := []struct {
		input string
		desc  string
	}{
		{"", "empty input"},
		{" ", "whitespace only"},
		{"\t\n\r", "mixed whitespace"},
		{"// comment", "line comment only"},
		{"/* comment */", "block comment only"},
	}

	for _, tt := range tests {
		l := lexInput(tt.input)
		be.Equal(t, l.CurrTokenType, EOF)
		be.Equal(t, l.CurrLiteral, "")
	}
}

func TestStringEscapes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		desc     string
	}{
		{"\"hello world\"", "hello world", "simple string"},
		{"\"hello\\nworld\"", "hello\\nworld", "newline escape"},
		{"\"hello\\tworld\"", "hello\\tworld", "tab escape"},
		{"\"hello\\\\world\"", "hello\\\\world", "backslash escape"},
		{"\"\"", "", "empty string"},
	}

	for _, tt := range tests {
		l := lexInput(tt.input)
		be.Equal(t, l.CurrTokenType, STRING)
		be.Equal(t, l.CurrLiteral, tt.expected)
	}
}

func TestCharEscapes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		desc     string
	}{
		{"'a'", "'a'", "simple char"},
		{"'Z'", "'Z'", "uppercase char"},
		{"'0'", "'0'", "digit char"},
		{"'\\n'", "'\\n'", "newline escape"},
		{"'\\t'", "'\\t'", "tab escape"},
		{"'\\''", "'\\''", "single quote escape"},
		{"'\\\\'", "'\\\\'", "backslash escape"},
	}

	for _, tt := range tests {
		l := lexInput(tt.input)
		be.Equal(t, l.CurrTokenType, CHAR)
		be.Equal(t, l.CurrLiteral, tt.expected)
	}
}

func TestNumberEdgeCases(t *testing.T) {
	tests := []struct {
		input       string
		expectedVal int64
		desc        string
	}{
		{"0", 0, "zero"},
		{"1", 1, "one"},
		{"999", 999, "three digits"},
		{"1000", 1000, "four digits"},
		{"123456789", 123456789, "large number"},
		{"9223372036854775807", 9223372036854775807, "max int64"},
	}

	for _, tt := range tests {
		l := lexInput(tt.input)
		be.Equal(t, l.CurrTokenType, INT)
		be.Equal(t, l.CurrLiteral, tt.input)
		be.Equal(t, l.CurrIntValue, tt.expectedVal)
	}
}

func TestOperatorBoundaries(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
		literals []string
		desc     string
	}{
		{"+++", []TokenType{PLUS_PLUS, PLUS}, []string{"++", "+"}, "plus plus plus"},
		{"---", []TokenType{MINUS_MINUS, MINUS}, []string{"--", "-"}, "minus minus minus"},
		{"<<=", []TokenType{SHL, ASSIGN}, []string{"<<", "="}, "left shift equals"},
		{">>=", []TokenType{SHR, ASSIGN}, []string{">>", "="}, "right shift equals"},
		{"&&&", []TokenType{AND, BIT_AND}, []string{"&&", "&"}, "logical and bitwise and"},
		{"|||", []TokenType{OR, BIT_OR}, []string{"||", "|"}, "logical or bitwise or"},
		{"!==", []TokenType{NOT_EQ, ASSIGN}, []string{"!=", "="}, "not equals equals"},
		{"===", []TokenType{EQ, ASSIGN}, []string{"==", "="}, "equals equals"},
		{"&^&", []TokenType{AND_NOT, BIT_AND}, []string{"&^", "&"}, "and not and"},
	}

	for _, tt := range tests {
		input := []byte(tt.input + "\x00")
		l := NewLexer(input)

		for i, expectedType := range tt.expected {
			l.NextToken()
			be.Equal(t, l.CurrTokenType, expectedType)
			be.Equal(t, l.CurrLiteral, tt.literals[i])
		}

		l.NextToken()
		be.Equal(t, l.CurrTokenType, EOF)
	}
}

func TestUnterminatedBlockComment(t *testing.T) {
	l := lexInput("x /* unterminated comment")
	be.Equal(t, l.CurrTokenType, IDENT)
	be.Equal(t, l.CurrLiteral, "x")

	l.NextToken()
	be.Equal(t, l.CurrTokenType, EOF)
}

func TestSkipToken(t *testing.T) {
	// Test successful token skip
	t.Run("successful skip", func(t *testing.T) {
		input := []byte("123\x00")
		l := NewLexer(input)
		l.NextToken()

		be.Equal(t, INT, l.CurrTokenType)

		l.SkipToken(INT) // Should not panic

		be.Equal(t, EOF, l.CurrTokenType)
	})

	// Test panic on wrong token type
	t.Run("panic on wrong token", func(t *testing.T) {
		defer func() {
			r := recover()
			be.True(t, r != nil)
			if r != nil {
				be.True(t, strings.Contains(r.(string), "Expected token"))
			}
		}()

		input := []byte("123\x00")
		l := NewLexer(input)
		l.NextToken()

		be.Equal(t, INT, l.CurrTokenType)

		l.SkipToken(IDENT) // Should panic - wrong token type
	})
}

func TestIntToString(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		expected string
	}{
		{
			name:     "positive number",
			value:    42,
			expected: "42",
		},
		{
			name:     "zero",
			value:    0,
			expected: "0",
		},
		{
			name:     "negative number",
			value:    -42,
			expected: "-42",
		},
		{
			name:     "large positive",
			value:    9223372036854775807, // max int64
			expected: "9223372036854775807",
		},
		{
			name:     "large negative",
			value:    -9223372036854775808, // min int64
			expected: "-9223372036854775808",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := intToString(test.value)
			be.Equal(t, test.expected, result)
		})
	}
}

func TestNextTokenIllegalCharacter(t *testing.T) {
	// Test handling of illegal/unknown characters
	input := []byte("@#$\x00") // Special characters not handled by lexer
	l := NewLexer(input)
	l.NextToken()

	be.Equal(t, ILLEGAL, l.CurrTokenType)
	be.Equal(t, "@", l.CurrLiteral)
}

// =============================================================================
// PARSING TESTS
// =============================================================================

func TestVarTypeAST(t *testing.T) {
	tests := []struct {
		input        string
		expectedType *TypeNode
	}{
		{
			input:        "var x I64;\x00",
			expectedType: TypeI64,
		},
		{
			input:        "var flag Boolean;\x00",
			expectedType: TypeBool,
		},
		{
			input:        "var ptr I64*;\x00",
			expectedType: &TypeNode{Kind: TypePointer, Child: TypeI64},
		},
		{
			input:        "var ptrPtr I64**;\x00",
			expectedType: &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypePointer, Child: TypeI64}},
		},
		{
			input:        "var boolPtr Boolean*;\x00",
			expectedType: &TypeNode{Kind: TypePointer, Child: TypeBool},
		},
	}

	for _, test := range tests {
		l := NewLexer([]byte(test.input))
		l.NextToken()
		result := ParseStatement(l)

		be.Equal(t, NodeVar, result.Kind)
		if result.Kind != NodeVar {
			continue
		}

		be.True(t, result.TypeAST != nil)
		if result.TypeAST == nil {
			continue
		}

		be.True(t, TypesEqual(result.TypeAST, test.expectedType))
	}
}

func TestSliceTypeParsing(t *testing.T) {
	// Test basic slice type parsing directly
	input := []byte("var nums I64[];\x00")
	l := NewLexer(input)
	l.NextToken()

	stmt := ParseStatement(l)
	be.Equal(t, stmt.Kind, NodeVar)

	expectedType := &TypeNode{
		Kind:  TypeSlice,
		Child: TypeI64,
	}
	be.True(t, TypesEqual(stmt.TypeAST, expectedType))
}

func TestSliceBasicDeclaration(t *testing.T) {
	// Test basic slice variable declaration
	input := []byte("var nums I64[];\x00")
	l := NewLexer(input)
	l.NextToken()

	stmt := ParseStatement(l)
	be.Equal(t, stmt.Kind, NodeVar)

	// Verify type is slice
	be.Equal(t, stmt.TypeAST.Kind, TypeSlice)
	be.Equal(t, stmt.TypeAST.Child.Kind, TypeBuiltin)
	be.Equal(t, stmt.TypeAST.Child.String, "I64")
}
