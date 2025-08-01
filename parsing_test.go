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

func lexInput(inputStr string) {
	input := []byte(inputStr + "\x00") // trailing null byte
	Init(input)
	NextToken()
}

func TestIntLiteral(t *testing.T) {
	lexInput("12345")
	be.Equal(t, CurrTokenType, INT)
	be.Equal(t, CurrLiteral, "12345")
	be.Equal(t, CurrIntValue, int64(12345))
}

func TestIdentifier(t *testing.T) {
	lexInput("foobar")
	be.Equal(t, CurrTokenType, IDENT)
	be.Equal(t, CurrLiteral, "foobar")
}

func TestStringLiteral(t *testing.T) {
	lexInput("\"hello\"")
	be.Equal(t, CurrTokenType, STRING)
	be.Equal(t, CurrLiteral, "hello")
}

func TestCharLiteral(t *testing.T) {
	lexInput("'a'")
	be.Equal(t, CurrTokenType, CHAR)
	be.Equal(t, CurrLiteral, "'a'")
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
		lexInput(tt.input)
		be.Equal(t, CurrTokenType, tt.typ)
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
		lexInput(tt.input)
		be.Equal(t, CurrTokenType, tt.expected)
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
		lexInput(tt.input)
		be.Equal(t, CurrTokenType, tt.typ)
		be.Equal(t, CurrLiteral, tt.input)
	}
}

func TestMultipleTokens(t *testing.T) {
	input := []byte("func main() { x := 42; }\x00")
	Init(input)

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
		NextToken()
		be.Equal(t, CurrTokenType, expected.typ)
		be.Equal(t, CurrLiteral, expected.literal)
		if expected.typ == INT {
			be.Equal(t, CurrIntValue, int64(42))
		}
	}
}

func TestLineComment(t *testing.T) {
	input := []byte("x // this is a comment\ny\x00")
	Init(input)

	NextToken()
	be.Equal(t, CurrTokenType, IDENT)
	be.Equal(t, CurrLiteral, "x")

	NextToken()
	be.Equal(t, CurrTokenType, IDENT)
	be.Equal(t, CurrLiteral, "y")

	NextToken()
	be.Equal(t, CurrTokenType, EOF)
}

func TestBlockComment(t *testing.T) {
	input := []byte("x /* this is a\nmultiline comment */ y\x00")
	Init(input)

	NextToken()
	be.Equal(t, CurrTokenType, IDENT)
	be.Equal(t, CurrLiteral, "x")

	NextToken()
	be.Equal(t, CurrTokenType, IDENT)
	be.Equal(t, CurrLiteral, "y")

	NextToken()
	be.Equal(t, CurrTokenType, EOF)
}

func TestCommentsWithTokens(t *testing.T) {
	input := []byte("func // comment\n main() /* comment */ {\x00")
	Init(input)

	expectedTokens := []TokenType{FUNC, IDENT, LPAREN, RPAREN, LBRACE, EOF}
	expectedLiterals := []string{"func", "main", "(", ")", "{", ""}

	for i, expected := range expectedTokens {
		NextToken()
		be.Equal(t, CurrTokenType, expected)
		be.Equal(t, CurrLiteral, expectedLiterals[i])
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
		Init(input)

		NextToken()
		be.Equal(t, CurrTokenType, IDENT)
		be.Equal(t, CurrLiteral, "x")

		NextToken()
		be.Equal(t, CurrTokenType, IDENT)
		be.Equal(t, CurrLiteral, "y")

		NextToken()
		be.Equal(t, CurrTokenType, EOF)
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
		lexInput(tt.input)
		be.Equal(t, CurrTokenType, EOF)
		be.Equal(t, CurrLiteral, "")
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
		lexInput(tt.input)
		be.Equal(t, CurrTokenType, STRING)
		be.Equal(t, CurrLiteral, tt.expected)
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
		lexInput(tt.input)
		be.Equal(t, CurrTokenType, CHAR)
		be.Equal(t, CurrLiteral, tt.expected)
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
		lexInput(tt.input)
		be.Equal(t, CurrTokenType, INT)
		be.Equal(t, CurrLiteral, tt.input)
		be.Equal(t, CurrIntValue, tt.expectedVal)
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
		Init(input)

		for i, expectedType := range tt.expected {
			NextToken()
			be.Equal(t, CurrTokenType, expectedType)
			be.Equal(t, CurrLiteral, tt.literals[i])
		}

		NextToken()
		be.Equal(t, CurrTokenType, EOF)
	}
}

func TestUnterminatedBlockComment(t *testing.T) {
	lexInput("x /* unterminated comment")
	be.Equal(t, CurrTokenType, IDENT)
	be.Equal(t, CurrLiteral, "x")

	NextToken()
	be.Equal(t, CurrTokenType, EOF)
}

func TestSkipToken(t *testing.T) {
	// Test successful token skip
	t.Run("successful skip", func(t *testing.T) {
		input := []byte("123\x00")
		Init(input)
		NextToken()

		be.Equal(t, INT, CurrTokenType)

		SkipToken(INT) // Should not panic

		be.Equal(t, EOF, CurrTokenType)
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
		Init(input)
		NextToken()

		be.Equal(t, INT, CurrTokenType)

		SkipToken(IDENT) // Should panic - wrong token type
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
	Init(input)
	NextToken()

	be.Equal(t, ILLEGAL, CurrTokenType)
	be.Equal(t, "@", CurrLiteral)
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
		Init([]byte(test.input))
		NextToken()
		result := ParseStatement()

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
	Init(input)
	NextToken()

	stmt := ParseStatement()
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
	Init(input)
	NextToken()

	stmt := ParseStatement()
	be.Equal(t, stmt.Kind, NodeVar)

	// Verify type is slice
	be.Equal(t, stmt.TypeAST.Kind, TypeSlice)
	be.Equal(t, stmt.TypeAST.Child.Kind, TypeBuiltin)
	be.Equal(t, stmt.TypeAST.Child.String, "I64")
}
