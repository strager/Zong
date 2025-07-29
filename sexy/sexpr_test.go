package sexy

import (
	"testing"

	"github.com/nalgeon/be"
)

func TestParseSymbol(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"test_var", "test_var"},
		{"func-name", "func-name"},
		{"x", "x"},
	}

	for _, test := range tests {
		result, err := Parse(test.input)
		be.Err(t, err, nil)

		be.Equal(t, result.Type, NodeSymbol)
		be.Equal(t, result.Text, test.expected)
		be.Equal(t, result.String(), test.expected)
	}
}

func TestParseString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		output   string
	}{
		{`"hello"`, "hello", `"hello"`},
		{`"hello world"`, "hello world", `"hello world"`},
		{`""`, "", `""`},
		{`"test\"quote"`, `test"quote`, `"test\"quote"`},
		{`"test\\backslash"`, `test\backslash`, `"test\\backslash"`},
	}

	for _, test := range tests {
		result, err := Parse(test.input)
		be.Err(t, err, nil)

		be.Equal(t, result.Type, NodeString)
		be.Equal(t, result.Text, test.expected)
		be.Equal(t, result.String(), test.output)
	}
}

func TestParseInteger(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"42", 42},
		{"0", 0},
		{"-123", -123},
		{"+456", 456},
	}

	for _, test := range tests {
		result, err := Parse(test.input)
		be.Err(t, err, nil)

		be.Equal(t, result.Type, NodeInteger)
		// Integer value is now stored as text, parse if needed
		be.Equal(t, result.Text, test.input)
		be.Equal(t, result.String(), test.input)
	}
}

func TestParseEllipsis(t *testing.T) {
	result, err := Parse("...")
	be.Err(t, err, nil)

	be.Equal(t, result.Type, NodeEllipsis)
	be.Equal(t, result.String(), "...")
}

func TestParseList(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"()", "()"},
		{"(hello)", "(hello)"},
		{"(1 2 3)", "(1 2 3)"},
		{"(binary \"+\" 1 2)", "(binary \"+\" 1 2)"},
		{"(nested (list here))", "(nested (list here))"},
	}

	for _, test := range tests {
		result, err := Parse(test.input)
		be.Err(t, err, nil)

		be.Equal(t, result.Type, NodeList)
		be.Equal(t, result.String(), test.expected)
	}
}

func TestParseArray(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[]", "[]"},
		{"[1]", "[1]"},
		{"[1 2 3]", "[1 2 3]"},
		{"[hello world]", "[hello world]"},
		{"[[nested] array]", "[[nested] array]"},
	}

	for _, test := range tests {
		result, err := Parse(test.input)
		be.Err(t, err, nil)

		be.Equal(t, result.Type, NodeArray)
		be.Equal(t, result.String(), test.expected)
	}
}

func TestParseMap(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"{}", "{}"},
		{"{key: value}", "{key: value}"},
		{"{a: 1, b: 2}", "{a: 1, b: 2}"},
		{"{size: 64}", "{size: 64}"},
		{"{op-loc: \"main.zong:3:13\"}", "{op-loc: \"main.zong:3:13\"}"},
	}

	for _, test := range tests {
		result, err := Parse(test.input)
		be.Err(t, err, nil)

		be.Equal(t, result.Type, NodeMap)
		be.Equal(t, result.String(), test.expected)
	}
}

func TestParseSet(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"{hello world}", "{hello world}"},
		{"{1 2 3}", "{1 2 3}"},
		{"{single}", "{single}"},
	}

	for _, test := range tests {
		result, err := Parse(test.input)
		be.Err(t, err, nil)

		be.Equal(t, result.Type, NodeSet)
		be.Equal(t, result.String(), test.expected)
	}
}

func TestParseLabelDef(t *testing.T) {
	tests := []struct {
		input         string
		expected      string
		expectedLabel string
		expectedType  NodeType
	}{
		{"#x=hello", "#x=hello", "x", NodeSymbol},
		{"#123=world", "#123=world", "123", NodeSymbol},
		{"#I64=(primitive \"I64\" {size: 64})", "#I64=(primitive \"I64\" {size: 64})", "I64", NodeList},
		{"#str=\"test\"", "#str=\"test\"", "str", NodeString},
		{"#num=42", "#num=42", "num", NodeInteger},
		{"#arr=[1 2 3]", "#arr=[1 2 3]", "arr", NodeArray},
		{"#m={key: value}", "#m={key: value}", "m", NodeMap},
	}

	for _, test := range tests {
		result, err := Parse(test.input)
		be.Err(t, err, nil)

		be.Equal(t, result.Type, test.expectedType)
		be.Equal(t, result.Label, test.expectedLabel)
		be.Equal(t, result.String(), test.expected)
	}
}

func TestParseLabelRef(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"#x#", "#x#"},
		{"#123#", "#123#"},
		{"#I64#", "#I64#"},
	}

	for _, test := range tests {
		result, err := Parse(test.input)
		be.Err(t, err, nil)

		be.Equal(t, result.Type, NodeLabelRef)
		be.Equal(t, result.String(), test.expected)
	}
}

func TestParseMeta(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"(binary \"+\" ^{op-loc: \"main.zong:3:13\"})", "(^{op-loc: \"main.zong:3:13\"} binary \"+\")"},
		{"(var \"x\" ^{loc: \"main.zong:3:11\"})", "(^{loc: \"main.zong:3:11\"} var \"x\")"},
	}

	for _, test := range tests {
		result, err := Parse(test.input)
		be.Err(t, err, nil)

		be.Equal(t, result.Type, NodeList)
		be.Equal(t, result.String(), test.expected)

		// Check that metadata is present
		be.True(t, len(result.MetaKeys) > 0)
		be.True(t, len(result.MetaItems) > 0)
	}
}

func TestParseMetaMerging(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"Multiple metadata merged",
			"(^{a: 1} ^{b: 2} foo ^{c: 3})",
			"(^{a: 1, b: 2, c: 3} foo)",
		},
		{
			"Three metadata nodes",
			"(^{x: \"val1\"} ^{y: \"val2\"} ^{z: \"val3\"} hello world)",
			"(^{x: \"val1\", y: \"val2\", z: \"val3\"} hello world)",
		},
		{
			"Metadata with overlapping keys - later wins",
			"(^{key: \"first\"} ^{key: \"second\"} item)",
			"(^{key: \"second\"} item)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Parse(test.input)
			be.Err(t, err, nil)

			be.Equal(t, result.Type, NodeList)
			be.Equal(t, result.String(), test.expected)

			// Check that metadata is merged
			be.True(t, len(result.MetaKeys) > 0)
			be.True(t, len(result.MetaItems) > 0)
			be.Equal(t, len(result.MetaKeys), len(result.MetaItems))
		})
	}
}

func TestParseComplexExamples(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"Binary expression AST",
			`(binary "+"
 (var "x")
 (binary "*"
  (var "yyy")
  (integer 2)))`,
			"(binary \"+\" (var \"x\") (binary \"*\" (var \"yyy\") (integer 2)))",
		},
		{
			"Type table",
			`{#I64=(primitive "I64" {size: 64})
 #I64ptr=(pointer "I64*" #I64#)
 #I64slice=(slice "I64[]" #I64#)
 #I64ptrslice=(slice "I64*[]" #I64ptr#)}`,
			"{#I64=(primitive \"I64\" {size: 64}) #I64ptr=(pointer \"I64*\" #I64#) #I64slice=(slice \"I64[]\" #I64#) #I64ptrslice=(slice \"I64*[]\" #I64ptr#)}",
		},
		{
			"Function with symbol resolution",
			`(func "f"
 [#x=(param "x" (type-ref "I64"))]
 [#y=(var-decl "y" (type-ref "I64") #x#)
  (return #y#)])`,
			"(func \"f\" [#x=(param \"x\" (type-ref \"I64\"))] [#y=(var-decl \"y\" (type-ref \"I64\") #x#) (return #y#)])",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Parse(test.input)
			be.Err(t, err, nil)
			be.Equal(t, result.String(), test.expected)
		})
	}
}

func TestRoundTripParsing(t *testing.T) {
	tests := []string{
		"hello",
		`"world"`,
		"42",
		"...",
		"()",
		"(test)",
		"(1 2 3)",
		"[]",
		"[1 2 3]",
		"{}",
		"{key: value}",
		"{hello world}",
		"#x=test",
		"#y#",
		"(binary \"+\" 1 2)",
		"(list ^{meta: data})",
		"{#label=(data here) #ref#}",
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			// Parse once
			result1, err := Parse(test)
			be.Err(t, err, nil)

			// Pretty-print
			output := result1.String()

			// Parse the pretty-printed output
			result2, err := Parse(output)
			be.Err(t, err, nil)

			// Should produce the same pretty-printed output
			be.Equal(t, result2.String(), output)
		})
	}
}

func TestParseComments(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"; comment\nhello", "hello"},
		{"hello ; trailing comment", "hello"},
		{"; AST for expression\n(binary \"+\" 1 2)", "(binary \"+\" 1 2)"},
		{"(test ; inline comment\n world)", "(test world)"},
	}

	for _, test := range tests {
		result, err := Parse(test.input)
		be.Err(t, err, nil)
		be.Equal(t, result.String(), test.expected)
	}
}

func TestLexerErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"unterminated string`, "unterminated string"},
		{`"invalid \escape"`, "invalid escape sequence"},
		{".", "unexpected character '.'"},
		{"@", "unexpected character '@'"},
		{"$", "unexpected character '$'"},
		{"%", "unexpected character '%'"},
		{"&", "unexpected character '&'"},
		{"?", "unexpected character '?'"},
		{"`", "unexpected character '`'"},
		{"~", "unexpected character '~'"},
	}

	for _, test := range tests {
		_, err := Parse(test.input)
		be.True(t, err != nil)
		be.True(t, len(err.Error()) > 0)
	}
}

func TestParserErrors(t *testing.T) {
	tests := []string{
		"(",           // unclosed list
		"[",           // unclosed array
		"{",           // unclosed map/set
		"(hello",      // unclosed list with content
		"^",           // invalid meta syntax
		"^hello",      // invalid meta syntax
		"hello world", // extra tokens after main expression
		"42 extra",    // extra tokens after integer
		"(test) more", // extra tokens after list
		"[] trailing", // extra tokens after array
	}

	for _, test := range tests {
		_, err := Parse(test)
		be.True(t, err != nil)
	}
}

func TestSyntaxErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"Single dot should be syntax error",
			".",
			"unexpected character '.'",
		},
		{
			"Unknown character @ should be syntax error",
			"@",
			"unexpected character '@'",
		},
		{
			"Unknown character $ should be syntax error",
			"$",
			"unexpected character '$'",
		},
		{
			"Unknown character % should be syntax error",
			"%",
			"unexpected character '%'",
		},
		{
			"Unknown character & should be syntax error",
			"&",
			"unexpected character '&'",
		},
		{
			"Three dots should still work as ellipsis",
			"...",
			"", // no error expected
		},
		{
			"Single dot within list should be syntax error",
			"(1 2 3 . 4)",
			"unexpected character '.'",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Parse(test.input)

			if test.expected == "" {
				// Should succeed
				be.Err(t, err, nil)
				if test.input == "..." {
					be.Equal(t, result.Type, NodeEllipsis)
					be.Equal(t, result.String(), "...")
				}
			} else {
				// Should fail with expected error
				be.True(t, err != nil)
				be.Equal(t, err.Error(), test.expected)
				be.True(t, result == nil)
			}
		})
	}
}

func TestNodeTypeHelpers(t *testing.T) {
	// Test atomic nodes
	symbol := NewSymbol("test")
	be.True(t, symbol.IsAtom())
	be.True(t, !symbol.IsLabeled())

	str := NewString("hello")
	be.True(t, str.IsAtom())
	be.True(t, !str.IsLabeled())

	integer := NewInteger("42")
	be.True(t, integer.IsAtom())
	be.True(t, !integer.IsLabeled())

	ellipsis := NewEllipsis()
	be.True(t, ellipsis.IsAtom())
	be.True(t, !ellipsis.IsLabeled())

	// Test collection nodes
	list := NewList([]*Node{symbol})
	be.True(t, !list.IsAtom())
	be.True(t, !list.IsLabeled())

	set := NewSet([]*Node{symbol})
	be.True(t, !set.IsAtom())
	be.True(t, !set.IsLabeled())

	// Test labeled nodes
	labeledSymbol := NewSymbol("test")
	labeledSymbol.SetLabel("x")
	be.True(t, labeledSymbol.IsAtom())
	be.True(t, labeledSymbol.IsLabeled()) // Labeled nodes are not unlabeled

	labelRef := NewLabelRef("x")
	be.True(t, !labelRef.IsAtom())
	be.True(t, !labelRef.IsLabeled()) // Label references themselves are unlabeled

	// Test map nodes (which are used for metadata)
	metaMap := NewMap([]string{"key"}, []*Node{NewString("value")})
	be.True(t, !metaMap.IsAtom())
	be.True(t, !metaMap.IsLabeled())
}
