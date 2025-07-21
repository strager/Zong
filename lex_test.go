package main

import (
	"testing"
)

func lexInput(inputStr string) {
	input := []byte(inputStr + "\x00") // trailing null byte
	Init(input)
	NextToken()
}

func TestIntLiteral(t *testing.T) {
	lexInput("12345")
	if CurrTokenType != INT {
		t.Fatalf("expected INT, got %s", CurrTokenType)
	}
	if CurrLiteral != "12345" {
		t.Errorf("expected literal '12345', got %q", CurrLiteral)
	}
	if CurrIntValue != 12345 {
		t.Errorf("expected value 12345, got %d", CurrIntValue)
	}
}

func TestIdentifier(t *testing.T) {
	lexInput("foobar")
	if CurrTokenType != IDENT {
		t.Fatalf("expected IDENT, got %s", CurrTokenType)
	}
	if CurrLiteral != "foobar" {
		t.Errorf("expected literal 'foobar', got %q", CurrLiteral)
	}
}

func TestStringLiteral(t *testing.T) {
	lexInput("\"hello\"")
	if CurrTokenType != STRING {
		t.Fatalf("expected STRING, got %s", CurrTokenType)
	}
	if CurrLiteral != "hello" {
		t.Errorf("expected literal 'hello', got %q", CurrLiteral)
	}
}

func TestCharLiteral(t *testing.T) {
	lexInput("'a'")
	if CurrTokenType != CHAR {
		t.Fatalf("expected CHAR, got %s", CurrTokenType)
	}
	if CurrLiteral != "'a'" {
		t.Errorf("expected literal \"'a'\", got %q", CurrLiteral)
	}
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
		if CurrTokenType != tt.typ {
			t.Errorf("input %q: expected %s, got %s", tt.input, tt.typ, CurrTokenType)
		}
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
		if CurrTokenType != tt.expected {
			t.Errorf("input %q: expected token type %s, got %s", tt.input, tt.expected, CurrTokenType)
		}
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
		if CurrTokenType != tt.typ {
			t.Errorf("input %q: expected token type %s, got %s", tt.input, tt.typ, CurrTokenType)
		}
		if CurrLiteral != tt.input {
			t.Errorf("input %q: expected literal %q, got %q", tt.input, tt.input, CurrLiteral)
		}
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

	for i, expected := range expectedTokens {
		NextToken()
		if CurrTokenType != expected.typ {
			t.Errorf("token %d: expected type %s, got %s", i, expected.typ, CurrTokenType)
		}
		if CurrLiteral != expected.literal {
			t.Errorf("token %d: expected literal %q, got %q", i, expected.literal, CurrLiteral)
		}
		if expected.typ == INT && CurrIntValue != 42 {
			t.Errorf("token %d: expected int value 42, got %d", i, CurrIntValue)
		}
	}
}

func TestLineComment(t *testing.T) {
	input := []byte("x // this is a comment\ny\x00")
	Init(input)

	NextToken()
	if CurrTokenType != IDENT {
		t.Errorf("expected IDENT, got %s", CurrTokenType)
	}
	if CurrLiteral != "x" {
		t.Errorf("expected literal 'x', got %q", CurrLiteral)
	}

	NextToken()
	if CurrTokenType != IDENT {
		t.Errorf("expected IDENT, got %s", CurrTokenType)
	}
	if CurrLiteral != "y" {
		t.Errorf("expected literal 'y', got %q", CurrLiteral)
	}

	NextToken()
	if CurrTokenType != EOF {
		t.Errorf("expected EOF, got %s", CurrTokenType)
	}
}

func TestBlockComment(t *testing.T) {
	input := []byte("x /* this is a\nmultiline comment */ y\x00")
	Init(input)

	NextToken()
	if CurrTokenType != IDENT {
		t.Errorf("expected IDENT, got %s", CurrTokenType)
	}
	if CurrLiteral != "x" {
		t.Errorf("expected literal 'x', got %q", CurrLiteral)
	}

	NextToken()
	if CurrTokenType != IDENT {
		t.Errorf("expected IDENT, got %s", CurrTokenType)
	}
	if CurrLiteral != "y" {
		t.Errorf("expected literal 'y', got %q", CurrLiteral)
	}

	NextToken()
	if CurrTokenType != EOF {
		t.Errorf("expected EOF, got %s", CurrTokenType)
	}
}

func TestCommentsWithTokens(t *testing.T) {
	input := []byte("func // comment\n main() /* comment */ {\x00")
	Init(input)

	expectedTokens := []TokenType{FUNC, IDENT, LPAREN, RPAREN, LBRACE, EOF}
	expectedLiterals := []string{"func", "main", "(", ")", "{", ""}

	for i, expected := range expectedTokens {
		NextToken()
		if CurrTokenType != expected {
			t.Errorf("token %d: expected type %s, got %s", i, expected, CurrTokenType)
		}
		if CurrLiteral != expectedLiterals[i] {
			t.Errorf("token %d: expected literal %q, got %q", i, expectedLiterals[i], CurrLiteral)
		}
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
		if CurrTokenType != IDENT {
			t.Errorf("%s: expected first token IDENT, got %s", tt.desc, CurrTokenType)
		}
		if CurrLiteral != "x" {
			t.Errorf("%s: expected first literal 'x', got %q", tt.desc, CurrLiteral)
		}

		NextToken()
		if CurrTokenType != IDENT {
			t.Errorf("%s: expected second token IDENT, got %s", tt.desc, CurrTokenType)
		}
		if CurrLiteral != "y" {
			t.Errorf("%s: expected second literal 'y', got %q", tt.desc, CurrLiteral)
		}

		NextToken()
		if CurrTokenType != EOF {
			t.Errorf("%s: expected EOF, got %s", tt.desc, CurrTokenType)
		}
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
		if CurrTokenType != EOF {
			t.Errorf("%s: expected EOF, got %s", tt.desc, CurrTokenType)
		}
		if CurrLiteral != "" {
			t.Errorf("%s: expected empty literal, got %q", tt.desc, CurrLiteral)
		}
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
		if CurrTokenType != STRING {
			t.Errorf("%s: expected STRING, got %s", tt.desc, CurrTokenType)
		}
		if CurrLiteral != tt.expected {
			t.Errorf("%s: expected literal %q, got %q", tt.desc, tt.expected, CurrLiteral)
		}
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
		if CurrTokenType != CHAR {
			t.Errorf("%s: expected CHAR, got %s", tt.desc, CurrTokenType)
		}
		if CurrLiteral != tt.expected {
			t.Errorf("%s: expected literal %q, got %q", tt.desc, tt.expected, CurrLiteral)
		}
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
		if CurrTokenType != INT {
			t.Errorf("%s: expected INT, got %s", tt.desc, CurrTokenType)
		}
		if CurrLiteral != tt.input {
			t.Errorf("%s: expected literal %q, got %q", tt.desc, tt.input, CurrLiteral)
		}
		if CurrIntValue != tt.expectedVal {
			t.Errorf("%s: expected value %d, got %d", tt.desc, tt.expectedVal, CurrIntValue)
		}
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
			if CurrTokenType != expectedType {
				t.Errorf("%s token %d: expected type %s, got %s", tt.desc, i, expectedType, CurrTokenType)
			}
			if CurrLiteral != tt.literals[i] {
				t.Errorf("%s token %d: expected literal %q, got %q", tt.desc, i, tt.literals[i], CurrLiteral)
			}
		}

		NextToken()
		if CurrTokenType != EOF {
			t.Errorf("%s: expected EOF after tokens, got %s", tt.desc, CurrTokenType)
		}
	}
}

func TestUnterminatedBlockComment(t *testing.T) {
	lexInput("x /* unterminated comment")
	if CurrTokenType != IDENT {
		t.Errorf("expected IDENT, got %s", CurrTokenType)
	}
	if CurrLiteral != "x" {
		t.Errorf("expected literal 'x', got %q", CurrLiteral)
	}

	NextToken()
	if CurrTokenType != EOF {
		t.Errorf("expected EOF after unterminated comment, got %s", CurrTokenType)
	}
}
