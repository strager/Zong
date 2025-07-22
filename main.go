package main

import "bytes"

// LocalVarInfo represents information about a local variable
type LocalVarInfo struct {
	Name  string
	Type  string // "I64" or "I64*" (pointers are i64 in WASM)
	Index uint32 // Local variable index in WASM
}

// WASM Binary Encoding Utilities
func writeByte(buf *bytes.Buffer, b byte) {
	buf.WriteByte(b)
}

func writeBytes(buf *bytes.Buffer, data []byte) {
	buf.Write(data)
}

func writeLEB128(buf *bytes.Buffer, val uint32) {
	for val >= 0x80 {
		buf.WriteByte(byte(val&0x7F) | 0x80)
		val >>= 7
	}
	buf.WriteByte(byte(val & 0x7F))
}

func writeLEB128Signed(buf *bytes.Buffer, val int64) {
	for {
		b := byte(val & 0x7F)
		val >>= 7

		if (val == 0 && (b&0x40) == 0) || (val == -1 && (b&0x40) != 0) {
			buf.WriteByte(b)
			break
		}

		buf.WriteByte(b | 0x80)
	}
}

// WASM Opcode Constants
const (
	I64_CONST        = 0x42
	I64_ADD          = 0x7C
	I64_SUB          = 0x7D
	I64_MUL          = 0x7E
	I64_DIV_S        = 0x7F
	I64_REM_S        = 0x81
	I64_EQ           = 0x51
	I64_NE           = 0x52
	I64_LT_S         = 0x53
	I64_GT_S         = 0x55
	I64_LE_S         = 0x57
	I64_GE_S         = 0x59
	I64_EXTEND_I32_S = 0xAC
	LOCAL_GET        = 0x20
	LOCAL_SET        = 0x21
	LOCAL_TEE        = 0x22
	CALL             = 0x10
	END              = 0x0B
)

// WASM Section Emitters
func EmitWASMHeader(buf *bytes.Buffer) {
	// WASM magic number
	writeBytes(buf, []byte{0x00, 0x61, 0x73, 0x6D})
	// WASM version
	writeBytes(buf, []byte{0x01, 0x00, 0x00, 0x00})
}

func EmitImportSection(buf *bytes.Buffer) {
	writeByte(buf, 0x02) // import section id

	// Build section content in temporary buffer to calculate size
	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, 1) // 1 import

	// Module name "env"
	writeLEB128(&sectionBuf, 3) // length of "env"
	writeBytes(&sectionBuf, []byte("env"))

	// Import name "print"
	writeLEB128(&sectionBuf, 5) // length of "print"
	writeBytes(&sectionBuf, []byte("print"))

	// Import kind: function (0x00)
	writeByte(&sectionBuf, 0x00)

	// Type index (0)
	writeLEB128(&sectionBuf, 0)

	// Write section size and content
	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

func EmitTypeSection(buf *bytes.Buffer) {
	writeByte(buf, 0x01) // type section id

	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, 2) // 2 function types

	// Type 0: print function (i64) -> ()
	writeByte(&sectionBuf, 0x60) // func type
	writeLEB128(&sectionBuf, 1)  // 1 param
	writeByte(&sectionBuf, 0x7E) // i64
	writeLEB128(&sectionBuf, 0)  // 0 results

	// Type 1: main function () -> ()
	writeByte(&sectionBuf, 0x60) // func type
	writeLEB128(&sectionBuf, 0)  // 0 params
	writeLEB128(&sectionBuf, 0)  // 0 results

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

func EmitFunctionSection(buf *bytes.Buffer) {
	writeByte(buf, 0x03) // function section id

	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, 1) // 1 function
	writeLEB128(&sectionBuf, 1) // function 0 uses type index 1 (main function type)

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

func EmitExportSection(buf *bytes.Buffer) {
	writeByte(buf, 0x07) // export section id

	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, 1) // 1 export

	// Export name "main"
	writeLEB128(&sectionBuf, 4) // length of "main"
	writeBytes(&sectionBuf, []byte("main"))

	// Export kind: function (0x00)
	writeByte(&sectionBuf, 0x00)

	// Function index (1 - main function comes after import)
	writeLEB128(&sectionBuf, 1)

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

func EmitCodeSection(buf *bytes.Buffer, ast *ASTNode) {
	writeByte(buf, 0x0A) // code section id

	// Collect local variables from AST
	locals := collectLocalVariables(ast)

	// Generate function body
	var bodyBuf bytes.Buffer

	// Emit locals declarations
	if len(locals) > 0 {
		// Group locals by type (all I64 in this implementation, including pointers)
		i64Count := 0
		for _, local := range locals {
			if local.Type == "I64" || local.Type == "I64*" {
				i64Count++
			}
		}

		writeLEB128(&bodyBuf, 1)                // 1 local type group
		writeLEB128(&bodyBuf, uint32(i64Count)) // count of I64 locals
		writeByte(&bodyBuf, 0x7E)               // I64 type
	} else {
		writeLEB128(&bodyBuf, 0) // 0 locals (existing behavior)
	}

	// Emit statement bytecode
	EmitStatement(&bodyBuf, ast, locals)
	writeByte(&bodyBuf, END) // end instruction

	// Build section content
	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, 1)                     // 1 function
	writeLEB128(&sectionBuf, uint32(bodyBuf.Len())) // function body size
	writeBytes(&sectionBuf, bodyBuf.Bytes())

	// Write section size and content
	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

// EmitStatement generates WASM bytecode for statements
func EmitStatement(buf *bytes.Buffer, node *ASTNode, locals []LocalVarInfo) {
	if node == nil {
		return
	}

	switch node.Kind {
	case NodeVar:
		// Variable declarations don't generate runtime code
		// (locals are declared in function header)
		break

	case NodeBlock:
		// Emit all statements in the block
		for _, stmt := range node.Children {
			EmitStatement(buf, stmt, locals)
		}

	case NodeCall:
		// Handle expression statements (e.g., print calls)
		EmitExpression(buf, node, locals)

	default:
		// For now, treat unknown statements as expressions
		EmitExpression(buf, node, locals)
	}
}

func EmitExpression(buf *bytes.Buffer, node *ASTNode, locals []LocalVarInfo) {
	switch node.Kind {
	case NodeInteger:
		writeByte(buf, I64_CONST)
		writeLEB128Signed(buf, node.Integer)

	case NodeIdent:
		// Variable reference - emit local.get
		var localIndex uint32
		found := false
		for _, local := range locals {
			if local.Name == node.String {
				localIndex = local.Index
				found = true
				break
			}
		}
		if !found {
			panic("Undefined variable: " + node.String)
		}
		writeByte(buf, LOCAL_GET)
		writeLEB128(buf, localIndex)

	case NodeBinary:
		if node.Op == "=" {
			// Variable assignment
			EmitExpression(buf, node.Children[1], locals) // RHS value

			// Get variable name and emit local.set
			varName := node.Children[0].String
			var localIndex uint32
			found := false
			for _, local := range locals {
				if local.Name == varName {
					localIndex = local.Index
					found = true
					break
				}
			}
			if !found {
				panic("Undefined variable: " + varName)
			}
			writeByte(buf, LOCAL_SET)
			writeLEB128(buf, localIndex)
		} else {
			// Regular binary operations
			EmitExpression(buf, node.Children[0], locals) // left operand
			EmitExpression(buf, node.Children[1], locals) // right operand
			writeByte(buf, getBinaryOpcode(node.Op))

			// Comparison operations return i32, but we need i64 for consistency
			if isComparisonOp(node.Op) {
				writeByte(buf, I64_EXTEND_I32_S) // Convert i32 to i64
			}
		}

	case NodeCall:
		if len(node.Children) > 0 && node.Children[0].Kind == NodeIdent && node.Children[0].String == "print" {
			if len(node.Children) > 1 {
				EmitExpression(buf, node.Children[1], locals) // argument
			}
			writeByte(buf, CALL) // call instruction
			writeLEB128(buf, 0)  // function index 0 (print import)
		}

	case NodeUnary:
		if node.Op == "&" {
			// Address-of operator - not yet implemented
			panic("Address-of operator (&) not yet implemented - requires memory model")
		} else if node.Op == "*" {
			// Dereference operator - not yet implemented
			panic("Dereference operator (*) not yet implemented - requires memory model")
		} else if node.Op == "!" {
			// Handle existing unary not operator
			EmitExpression(buf, node.Children[0], locals)
			// TODO: Implement logical not operation
			panic("Unary not operator (!) not yet implemented")
		}
	}
}

func getBinaryOpcode(op string) byte {
	switch op {
	case "+":
		return I64_ADD
	case "-":
		return I64_SUB
	case "*":
		return I64_MUL
	case "/":
		return I64_DIV_S
	case "%":
		return I64_REM_S
	case "==":
		return I64_EQ
	case "!=":
		return I64_NE
	case "<":
		return I64_LT_S
	case ">":
		return I64_GT_S
	case "<=":
		return I64_LE_S
	case ">=":
		return I64_GE_S
	default:
		panic("Unsupported binary operator: " + op)
	}
}

func isComparisonOp(op string) bool {
	switch op {
	case "==", "!=", "<", ">", "<=", ">=":
		return true
	default:
		return false
	}
}

func CompileToWASM(ast *ASTNode) []byte {
	var buf bytes.Buffer

	// Emit WASM module header and sections in streaming fashion
	EmitWASMHeader(&buf)
	EmitTypeSection(&buf)      // function type definitions
	EmitImportSection(&buf)    // print function import
	EmitFunctionSection(&buf)  // declare main function
	EmitExportSection(&buf)    // export main function
	EmitCodeSection(&buf, ast) // main function body with compiled expression

	return buf.Bytes()
}

// collectLocalVariables traverses AST to find all var declarations
func collectLocalVariables(node *ASTNode) []LocalVarInfo {
	var locals []LocalVarInfo
	var localIndex uint32 = 0

	collectLocalsRecursive(node, &locals, &localIndex)
	return locals
}

func collectLocalsRecursive(node *ASTNode, locals *[]LocalVarInfo, index *uint32) {
	if node == nil {
		return
	}

	switch node.Kind {
	case NodeVar:
		// Extract variable name and type
		varName := node.Children[0].String
		varType := node.Children[1].String

		// Support I64 and I64* (pointers are i64 in WASM)
		if varType == "I64" || varType == "I64*" {
			*locals = append(*locals, LocalVarInfo{
				Name:  varName,
				Type:  varType,
				Index: *index,
			})
			*index++
		}

	case NodeBlock, NodeIf, NodeLoop:
		// Recursively process child statements
		for _, child := range node.Children {
			collectLocalsRecursive(child, locals, index)
		}
	}
}

// Global lexer input state
var (
	input []byte
	pos   int // current reading position in input
)

// Global “current token” state
var (
	CurrTokenType TokenType
	CurrLiteral   string
	CurrIntValue  int64 // only meaningful when CurrTokenType == INT
)

// TokenType is the type of token (identifier, operator, literal, etc.).
type TokenType string

// Definition of token types
const (
	// Special tokens
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers + literals
	IDENT  = "IDENT" // main, foo, _bar
	INT    = "INT"   // 12345
	STRING = "STRING"
	CHAR   = "CHAR"

	// Operators
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"
	PERCENT  = "%"

	LT     = "<"
	GT     = ">"
	EQ     = "=="
	NOT_EQ = "!="
	LE     = "<="
	GE     = ">="

	AND     = "&&"
	OR      = "||"
	BIT_AND = "&"
	BIT_OR  = "|"
	XOR     = "^"
	SHL     = "<<"
	SHR     = ">>"
	AND_NOT = "&^"

	PLUS_PLUS   = "++"
	MINUS_MINUS = "--"
	DECLARE     = ":="

	// Delimiters
	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"
	LPAREN    = "("
	RPAREN    = ")"
	LBRACE    = "{"
	RBRACE    = "}"
	LBRACKET  = "["
	RBRACKET  = "]"
	DOT       = "."
	ELLIPSIS  = "..."

	IF          = "IF"
	ELSE        = "ELSE"
	FOR         = "FOR"
	FUNC        = "FUNC"
	RETURN      = "RETURN"
	VAR         = "VAR"
	CONST       = "CONST"
	TYPE        = "TYPE"
	STRUCT      = "STRUCT"
	PACKAGE     = "PACKAGE"
	IMPORT      = "IMPORT"
	BREAK       = "BREAK"
	CONTINUE    = "CONTINUE"
	SWITCH      = "SWITCH"
	CASE        = "CASE"
	DEFAULT     = "DEFAULT"
	SELECT      = "SELECT"
	GO          = "GO"
	DEFER       = "DEFER"
	FALLTHROUGH = "FALLTHROUGH"
	MAP         = "MAP"
	RANGE       = "RANGE"
	INTERFACE   = "INTERFACE"
	CHAN        = "CHAN"
	GOTO        = "GOTO"
	LOOP        = "LOOP"
)

// NodeKind represents different types of AST nodes
type NodeKind string

const (
	NodeIdent    NodeKind = "NodeIdent"
	NodeString   NodeKind = "NodeString"
	NodeInteger  NodeKind = "NodeInteger"
	NodeBinary   NodeKind = "NodeBinary"
	NodeIf       NodeKind = "NodeIf"
	NodeVar      NodeKind = "NodeVar"
	NodeBlock    NodeKind = "NodeBlock"
	NodeReturn   NodeKind = "NodeReturn"
	NodeLoop     NodeKind = "NodeLoop"
	NodeBreak    NodeKind = "NodeBreak"
	NodeContinue NodeKind = "NodeContinue"
	NodeCall     NodeKind = "NodeCall"
	NodeIndex    NodeKind = "NodeIndex"
	NodeUnary    NodeKind = "NodeUnary"
)

// ASTNode represents a node in the Abstract Syntax Tree
type ASTNode struct {
	Kind NodeKind
	// NodeIdent, NodeString:
	String string
	// NodeInteger:
	Integer int64
	// NodeBinary:
	Op       string // "+", "-", "==", "!"
	Children []*ASTNode
	// NodeCall:
	ParameterNames []string
}

// Init initializes the lexer with the given input (must end with a 0 byte).
func Init(in []byte) {
	input = in
	pos = 0
}

// NextToken scans the next token and stores it in the globals.
// Call repeatedly until CurrTokenType == EOF.
func NextToken() {
	skipWhitespace()

	c := input[pos]
	CurrIntValue = 0 // reset for non-INT tokens

	if c == '=' {
		if input[pos+1] == '=' {
			CurrTokenType = EQ
			CurrLiteral = string(input[pos : pos+2])
			pos += 2
		} else {
			CurrTokenType = ASSIGN
			CurrLiteral = string(c)
			pos++ // inlined advance()
		}

	} else if c == '+' {
		if input[pos+1] == '+' {
			CurrTokenType = PLUS_PLUS
			CurrLiteral = "++"
			pos += 2
		} else {
			CurrTokenType = PLUS
			CurrLiteral = string(c)
			pos++
		}

	} else if c == '-' {
		nxt := input[pos+1]
		if nxt == '-' {
			CurrTokenType = MINUS_MINUS
			CurrLiteral = "--"
			pos += 2
		} else {
			CurrTokenType = MINUS
			CurrLiteral = string(c)
			pos++
		}

	} else if c == '!' {
		if input[pos+1] == '=' {
			CurrTokenType = NOT_EQ
			CurrLiteral = string(input[pos : pos+2])
			pos += 2
		} else {
			CurrTokenType = BANG
			CurrLiteral = string(c)
			pos++
		}

	} else if c == '/' {
		nxt := input[pos+1]
		if nxt == '/' {
			skipLineComment()
			NextToken()
			return
		} else if nxt == '*' {
			skipBlockComment()
			NextToken()
			return
		} else {
			CurrTokenType = SLASH
			CurrLiteral = string(c)
			pos++
		}

	} else if c == '*' {
		CurrTokenType = ASTERISK
		CurrLiteral = string(c)
		pos++

	} else if c == '%' {
		CurrTokenType = PERCENT
		CurrLiteral = string(c)
		pos++

	} else if c == '<' {
		if input[pos+1] == '=' {
			CurrTokenType = LE
			CurrLiteral = "<="
			pos += 2
		} else if input[pos+1] == '<' {
			CurrTokenType = SHL
			CurrLiteral = "<<"
			pos += 2
		} else {
			CurrTokenType = LT
			CurrLiteral = string(c)
			pos++
		}

	} else if c == '>' {
		if input[pos+1] == '=' {
			CurrTokenType = GE
			CurrLiteral = ">="
			pos += 2
		} else if input[pos+1] == '>' {
			CurrTokenType = SHR
			CurrLiteral = ">>"
			pos += 2
		} else {
			CurrTokenType = GT
			CurrLiteral = string(c)
			pos++
		}

	} else if c == '&' {
		nxt := input[pos+1]
		if nxt == '&' {
			CurrTokenType = AND
			CurrLiteral = "&&"
			pos += 2
		} else if nxt == '^' {
			CurrTokenType = AND_NOT
			CurrLiteral = "&^"
			pos += 2
		} else {
			CurrTokenType = BIT_AND
			CurrLiteral = string(c)
			pos++
		}

	} else if c == '|' {
		if input[pos+1] == '|' {
			CurrTokenType = OR
			CurrLiteral = "||"
			pos += 2
		} else {
			CurrTokenType = BIT_OR
			CurrLiteral = string(c)
			pos++
		}

	} else if c == '^' {
		CurrTokenType = XOR
		CurrLiteral = string(c)
		pos++

	} else if c == ',' {
		CurrTokenType = COMMA
		CurrLiteral = string(c)
		pos++

	} else if c == ';' {
		CurrTokenType = SEMICOLON
		CurrLiteral = string(c)
		pos++

	} else if c == ':' {
		if input[pos+1] == '=' {
			CurrTokenType = DECLARE
			CurrLiteral = ":="
			pos += 2
		} else {
			CurrTokenType = COLON
			CurrLiteral = string(c)
			pos++
		}

	} else if c == '(' {
		CurrTokenType = LPAREN
		CurrLiteral = string(c)
		pos++

	} else if c == ')' {
		CurrTokenType = RPAREN
		CurrLiteral = string(c)
		pos++

	} else if c == '{' {
		CurrTokenType = LBRACE
		CurrLiteral = string(c)
		pos++

	} else if c == '}' {
		CurrTokenType = RBRACE
		CurrLiteral = string(c)
		pos++

	} else if c == '[' {
		CurrTokenType = LBRACKET
		CurrLiteral = string(c)
		pos++

	} else if c == ']' {
		CurrTokenType = RBRACKET
		CurrLiteral = string(c)
		pos++

	} else if c == '.' {
		if input[pos+1] == '.' && input[pos+2] == '.' {
			CurrTokenType = ELLIPSIS
			CurrLiteral = "..."
			pos += 3
		} else {
			CurrTokenType = DOT
			CurrLiteral = string(c)
			pos++
		}

	} else if c == '"' {
		CurrTokenType = STRING
		CurrLiteral = readString()

	} else if c == '\'' {
		CurrTokenType = CHAR
		CurrLiteral = readCharLiteral()

	} else if c == 0 {
		CurrTokenType = EOF
		CurrLiteral = ""

	} else {
		if isLetter(c) {
			lit := readIdentifier()
			// keyword check
			if lit == "break" {
				CurrTokenType = BREAK
			} else if lit == "default" {
				CurrTokenType = DEFAULT
			} else if lit == "func" {
				CurrTokenType = FUNC
			} else if lit == "interface" {
				CurrTokenType = INTERFACE
			} else if lit == "select" {
				CurrTokenType = SELECT
			} else if lit == "case" {
				CurrTokenType = CASE
			} else if lit == "defer" {
				CurrTokenType = DEFER
			} else if lit == "go" {
				CurrTokenType = GO
			} else if lit == "map" {
				CurrTokenType = MAP
			} else if lit == "struct" {
				CurrTokenType = STRUCT
			} else if lit == "chan" {
				CurrTokenType = CHAN
			} else if lit == "else" {
				CurrTokenType = ELSE
			} else if lit == "goto" {
				CurrTokenType = GOTO
			} else if lit == "package" {
				CurrTokenType = PACKAGE
			} else if lit == "switch" {
				CurrTokenType = SWITCH
			} else if lit == "const" {
				CurrTokenType = CONST
			} else if lit == "fallthrough" {
				CurrTokenType = FALLTHROUGH
			} else if lit == "if" {
				CurrTokenType = IF
			} else if lit == "range" {
				CurrTokenType = RANGE
			} else if lit == "type" {
				CurrTokenType = TYPE
			} else if lit == "continue" {
				CurrTokenType = CONTINUE
			} else if lit == "for" {
				CurrTokenType = FOR
			} else if lit == "import" {
				CurrTokenType = IMPORT
			} else if lit == "return" {
				CurrTokenType = RETURN
			} else if lit == "var" {
				CurrTokenType = VAR
			} else if lit == "loop" {
				CurrTokenType = LOOP
			} else {
				CurrTokenType = IDENT
			}
			CurrLiteral = lit

		} else if isDigit(c) {
			lit, val := readNumber()
			CurrTokenType = INT
			CurrLiteral = lit
			CurrIntValue = val

		} else {
			CurrTokenType = ILLEGAL
			CurrLiteral = string(c)
			pos++
		}
	}
}

func skipWhitespace() {
	for {
		c := input[pos]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return
		}
		pos++
	}
}

func skipLineComment() {
	for input[pos] != '\n' && input[pos] != 0 {
		pos++
	}
	if input[pos] == '\n' {
		pos++
	}
}

func skipBlockComment() {
	pos += 2 // skip /*
	for input[pos] != 0 && !(input[pos] == '*' && input[pos+1] == '/') {
		pos++
	}
	if input[pos] == '*' && input[pos+1] == '/' {
		pos += 2 // skip */
	}
}

func isLetter(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || c == '_'
}

func readIdentifier() string {
	start := pos
	for isLetter(input[pos]) || isDigit(input[pos]) {
		pos++
	}
	return string(input[start:pos])
}

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func readNumber() (string, int64) {
	start := pos
	var val int64
	for isDigit(input[pos]) {
		val = val*10 + int64(input[pos]-'0')
		pos++
	}
	return string(input[start:pos]), val
}

func readString() string {
	pos++ // skip opening "
	start := pos
	for input[pos] != '"' {
		pos++
	}
	lit := string(input[start:pos])
	pos++
	return lit
}

func readCharLiteral() string {
	start := pos
	pos++ // Skip first '.
	if input[pos] == '\\' {
		pos++
	}
	pos++ // Skip the character.
	pos++ // Skip last '.
	lit := string(input[start:pos])
	return lit
}

// ToSExpr converts an AST node to s-expression string representation
func ToSExpr(node *ASTNode) string {
	switch node.Kind {
	case NodeIdent:
		return "(ident \"" + node.String + "\")"
	case NodeString:
		return "(string \"" + node.String + "\")"
	case NodeInteger:
		return "(integer " + intToString(node.Integer) + ")"
	case NodeBinary:
		left := ToSExpr(node.Children[0])
		right := ToSExpr(node.Children[1])
		return "(binary \"" + node.Op + "\" " + left + " " + right + ")"
	case NodeIf:
		cond := ToSExpr(node.Children[0])
		result := "(if " + cond
		for i := 1; i < len(node.Children); i++ {
			result += " " + ToSExpr(node.Children[i])
		}
		result += ")"
		return result
	case NodeVar:
		name := ToSExpr(node.Children[0])
		varType := ToSExpr(node.Children[1])
		return "(var " + name + " " + varType + ")"
	case NodeBlock:
		result := "(block"
		for _, child := range node.Children {
			result += " " + ToSExpr(child)
		}
		result += ")"
		return result
	case NodeReturn:
		if len(node.Children) == 0 {
			return "(return)"
		}
		expr := ToSExpr(node.Children[0])
		return "(return " + expr + ")"
	case NodeLoop:
		result := "(loop"
		for _, child := range node.Children {
			result += " " + ToSExpr(child)
		}
		result += ")"
		return result
	case NodeBreak:
		return "(break)"
	case NodeContinue:
		return "(continue)"
	case NodeCall:
		result := "(call " + ToSExpr(node.Children[0])
		for i := 1; i < len(node.Children); i++ {
			if i-1 < len(node.ParameterNames) && node.ParameterNames[i-1] != "" {
				result += " \"" + node.ParameterNames[i-1] + "\""
			}
			result += " " + ToSExpr(node.Children[i])
		}
		result += ")"
		return result
	case NodeIndex:
		array := ToSExpr(node.Children[0])
		index := ToSExpr(node.Children[1])
		return "(idx " + array + " " + index + ")"
	case NodeUnary:
		operand := ToSExpr(node.Children[0])
		return "(unary \"" + node.Op + "\" " + operand + ")"
	default:
		return ""
	}
}

// intToString converts an int64 to string
func intToString(n int64) string {
	if n == 0 {
		return "0"
	}

	var result string
	negative := n < 0
	if negative {
		n = -n
	}

	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}

	if negative {
		result = "-" + result
	}

	return result
}

// PeekToken returns the next token type without advancing the lexer.
// Useful for lookahead parsing decisions.
func PeekToken() TokenType {
	savedPos := pos
	savedTokenType := CurrTokenType
	savedLiteral := CurrLiteral
	savedIntValue := CurrIntValue

	NextToken()
	nextType := CurrTokenType

	// Restore state
	pos = savedPos
	CurrTokenType = savedTokenType
	CurrLiteral = savedLiteral
	CurrIntValue = savedIntValue

	return nextType
}

// SkipToken advances past the current token, asserting it matches the expected type.
//
// Panics if the current token doesn't match the expected type.
func SkipToken(expectedType TokenType) {
	if CurrTokenType != expectedType {
		panic("Expected token " + string(expectedType) + " but got " + string(CurrTokenType))
	}
	NextToken()
}

// precedence returns the precedence level for a given token type
func precedence(tokenType TokenType) int {
	switch tokenType {
	case ASSIGN:
		return 1 // assignment has very low precedence
	case EQ, NOT_EQ, LT, GT, LE, GE:
		return 2
	case PLUS, MINUS:
		return 3
	case ASTERISK, SLASH, PERCENT:
		return 4
	case LBRACKET, LPAREN: // subscript and function call operators
		return 5 // highest precedence (postfix)
	case BIT_AND: // postfix address-of operator
		return 5 // highest precedence (postfix)
	default:
		return 0 // not an operator
	}
}

// isOperator returns true if the token is a binary operator
func isOperator(tokenType TokenType) bool {
	return precedence(tokenType) > 0
}

// ParseExpression parses an expression and returns an AST node
func ParseExpression() *ASTNode {
	return parseExpressionWithPrecedence(0)
}

// parseExpressionWithPrecedence implements precedence climbing
func parseExpressionWithPrecedence(minPrec int) *ASTNode {
	var left *ASTNode

	// Handle unary operators first
	if CurrTokenType == BANG {
		SkipToken(BANG)                             // consume '!'
		operand := parseExpressionWithPrecedence(3) // Same as multiplication, less than postfix
		left = &ASTNode{
			Kind:     NodeUnary,
			Op:       "!",
			Children: []*ASTNode{operand},
		}
	} else {
		left = parsePrimary()
	}

	for {
		if !isOperator(CurrTokenType) || precedence(CurrTokenType) < minPrec {
			break
		}

		if CurrTokenType == LBRACKET {
			// Handle subscript operator
			SkipToken(LBRACKET)
			index := parseExpressionWithPrecedence(0)
			if CurrTokenType == RBRACKET {
				SkipToken(RBRACKET)
			}
			left = &ASTNode{
				Kind:     NodeIndex,
				Children: []*ASTNode{left, index},
			}
		} else if CurrTokenType == LPAREN {
			// Handle function call operator
			SkipToken(LPAREN)

			var args []*ASTNode
			var paramNames []string

			for CurrTokenType != RPAREN && CurrTokenType != EOF {
				var paramName string

				// Check for named parameter (identifier followed by colon)
				if CurrTokenType == IDENT {
					// Look ahead to see if there's a colon after the identifier
					identName := CurrLiteral
					if PeekToken() == COLON {
						// This is a named parameter: name: value
						paramName = identName
						SkipToken(IDENT)
						SkipToken(COLON)
					} else {
						paramName = ""
					}
				} else {
					paramName = ""
				}

				paramNames = append(paramNames, paramName)
				expr := parseExpressionWithPrecedence(0)
				args = append(args, expr)

				if CurrTokenType == COMMA {
					SkipToken(COMMA)
				} else if CurrTokenType != RPAREN {
					break
				}
			}

			if CurrTokenType == RPAREN {
				SkipToken(RPAREN)
			}

			left = &ASTNode{
				Kind:           NodeCall,
				Children:       append([]*ASTNode{left}, args...),
				ParameterNames: paramNames,
			}
		} else if CurrTokenType == ASTERISK && minPrec <= 5 {
			// Handle postfix dereference operator: expr*
			// Check if next token suggests this should be binary instead
			nextToken := PeekToken()
			if nextToken == IDENT || nextToken == INT || nextToken == LPAREN || nextToken == LBRACKET {
				// Treat as binary multiplication - fall through to binary operator handling
				op := CurrLiteral
				prec := precedence(CurrTokenType)
				NextToken()
				right := parseExpressionWithPrecedence(prec + 1) // left-associative
				left = &ASTNode{
					Kind:     NodeBinary,
					Op:       op,
					Children: []*ASTNode{left, right},
				}
			} else {
				// Treat as postfix dereference
				SkipToken(ASTERISK)
				left = &ASTNode{
					Kind:     NodeUnary,
					Op:       "*",
					Children: []*ASTNode{left},
				}
			}
		} else if CurrTokenType == BIT_AND {
			// Handle postfix address-of operator: expr&
			SkipToken(BIT_AND)
			left = &ASTNode{
				Kind:     NodeUnary,
				Op:       "&",
				Children: []*ASTNode{left},
			}
		} else {
			// Handle binary operators
			op := CurrLiteral
			prec := precedence(CurrTokenType)
			NextToken()

			// For assignment (right-associative), use prec instead of prec + 1
			// For other operators (left-associative), use prec + 1
			var right *ASTNode
			if op == "=" {
				right = parseExpressionWithPrecedence(prec) // right-associative
			} else {
				right = parseExpressionWithPrecedence(prec + 1) // left-associative
			}

			left = &ASTNode{
				Kind:     NodeBinary,
				Op:       op,
				Children: []*ASTNode{left, right},
			}
		}
	}

	return left
}

// parsePrimary handles primary expressions (literals, identifiers, parentheses)
func parsePrimary() *ASTNode {
	switch CurrTokenType {
	case INT:
		node := &ASTNode{
			Kind:    NodeInteger,
			Integer: CurrIntValue,
		}
		SkipToken(INT)
		return node

	case STRING:
		node := &ASTNode{
			Kind:   NodeString,
			String: CurrLiteral,
		}
		SkipToken(STRING)
		return node

	case IDENT:
		node := &ASTNode{
			Kind:   NodeIdent,
			String: CurrLiteral,
		}
		SkipToken(IDENT)
		return node

	case LPAREN:
		SkipToken(LPAREN) // consume '('
		expr := parseExpressionWithPrecedence(0)
		if CurrTokenType == RPAREN {
			SkipToken(RPAREN)
		}
		return expr

	default:
		// Return empty node for error case
		return &ASTNode{}
	}
}

// ParseStatement parses a statement and returns an AST node
func ParseStatement() *ASTNode {
	switch CurrTokenType {
	case IF:
		SkipToken(IF)
		cond := ParseExpression()
		if CurrTokenType != LBRACE {
			return &ASTNode{} // error
		}
		SkipToken(LBRACE)
		var children []*ASTNode = []*ASTNode{cond}
		for CurrTokenType != RBRACE && CurrTokenType != EOF {
			stmt := ParseStatement()
			children = append(children, stmt)
		}
		if CurrTokenType == RBRACE {
			SkipToken(RBRACE)
		}
		return &ASTNode{
			Kind:     NodeIf,
			Children: children,
		}

	case VAR:
		SkipToken(VAR)
		if CurrTokenType != IDENT {
			return &ASTNode{} // error
		}
		varName := &ASTNode{
			Kind:   NodeIdent,
			String: CurrLiteral,
		}
		SkipToken(IDENT)
		if CurrTokenType != IDENT {
			return &ASTNode{} // error - expecting type
		}

		// Parse type with potential pointer suffix
		typeName := CurrLiteral
		SkipToken(IDENT)

		// Check for pointer suffix
		if CurrTokenType == ASTERISK {
			typeName += "*"
			SkipToken(ASTERISK)
		}

		varType := &ASTNode{
			Kind:   NodeIdent,
			String: typeName,
		}

		if CurrTokenType == SEMICOLON {
			SkipToken(SEMICOLON)
		}
		return &ASTNode{
			Kind:     NodeVar,
			Children: []*ASTNode{varName, varType},
		}

	case LBRACE:
		SkipToken(LBRACE)
		var statements []*ASTNode
		for CurrTokenType != RBRACE && CurrTokenType != EOF {
			stmt := ParseStatement()
			statements = append(statements, stmt)
		}
		if CurrTokenType == RBRACE {
			SkipToken(RBRACE)
		}
		return &ASTNode{
			Kind:     NodeBlock,
			Children: statements,
		}

	case RETURN:
		SkipToken(RETURN)
		var children []*ASTNode
		// Check if there's an expression after return
		if CurrTokenType != SEMICOLON {
			expr := ParseExpression()
			children = append(children, expr)
		}
		if CurrTokenType == SEMICOLON {
			SkipToken(SEMICOLON)
		}
		return &ASTNode{
			Kind:     NodeReturn,
			Children: children,
		}

	case LOOP:
		SkipToken(LOOP)
		if CurrTokenType != LBRACE {
			return &ASTNode{} // error
		}
		SkipToken(LBRACE)
		var statements []*ASTNode
		for CurrTokenType != RBRACE && CurrTokenType != EOF {
			stmt := ParseStatement()
			statements = append(statements, stmt)
		}
		if CurrTokenType == RBRACE {
			SkipToken(RBRACE)
		}
		return &ASTNode{
			Kind:     NodeLoop,
			Children: statements,
		}

	case BREAK:
		SkipToken(BREAK)
		if CurrTokenType == SEMICOLON {
			SkipToken(SEMICOLON)
		}
		return &ASTNode{
			Kind: NodeBreak,
		}

	case CONTINUE:
		SkipToken(CONTINUE)
		if CurrTokenType == SEMICOLON {
			SkipToken(SEMICOLON)
		}
		return &ASTNode{
			Kind: NodeContinue,
		}

	default:
		// Expression statement
		expr := ParseExpression()
		if CurrTokenType == SEMICOLON {
			SkipToken(SEMICOLON)
		}
		return expr
	}
}
