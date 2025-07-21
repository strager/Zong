package main

import (
	"testing"

	"github.com/nalgeon/be"
)

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
