package main

import (
	"bytes"
	"fmt"
)

// VarStorage represents how a variable is stored
type VarStorage int

const (
	VarStorageLocal          VarStorage = iota // Variable stored in WASM local
	VarStorageParameterLocal                   // Variable stored as function parameter (like VarStorageLocal but allocated differently)
	VarStorageTStack                           // Variable stored on the stack (addressed)
)

// LocalVarInfo represents information about a local variable
type LocalVarInfo struct {
	Symbol  *SymbolInfo // Link to symbol table entry for this variable
	Storage VarStorage  // How the variable is stored
	Address uint32      // For VarStorageLocal: WASM local index; For VarStorageTStack: byte offset in stack frame
}

// LocalContext represents unified local variable management for both legacy and function compilation paths
type LocalContext struct {
	// Variable registry
	Variables []LocalVarInfo

	// Layout configuration
	ParameterCount    uint32
	I32LocalCount     uint32
	I64LocalCount     uint32
	FramePointerIndex uint32
	FrameSize         uint32
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
	I32_CONST        = 0x41
	I32_ADD          = 0x6A
	I32_WRAP_I64     = 0xA7
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
	I64_LOAD         = 0x29
	I64_STORE        = 0x37
	GLOBAL_GET       = 0x23
	GLOBAL_SET       = 0x24
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
	writeLEB128(&sectionBuf, 2) // 2 imports: print function + tstack global

	// Import 1: print function
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

	// Import 2: tstack global
	// Module name "env"
	writeLEB128(&sectionBuf, 3) // length of "env"
	writeBytes(&sectionBuf, []byte("env"))

	// Import name "tstack"
	writeLEB128(&sectionBuf, 6) // length of "tstack"
	writeBytes(&sectionBuf, []byte("tstack"))

	// Import kind: global (0x03)
	writeByte(&sectionBuf, 0x03)

	// Global type: i32 mutable (0x7F 0x01)
	writeByte(&sectionBuf, 0x7F) // i32
	writeByte(&sectionBuf, 0x01) // mutable

	// Write section size and content
	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

func EmitMemorySection(buf *bytes.Buffer) {
	writeByte(buf, 0x05) // memory section id

	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, 1) // 1 memory

	// Memory limits: initial=1 page (64KB), no maximum
	writeByte(&sectionBuf, 0x00) // limits flags (no maximum)
	writeLEB128(&sectionBuf, 1)  // initial pages (1 page = 64KB)

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

// Global type registry for consistent type indices
var globalTypeRegistry []FunctionType
var globalTypeMap map[string]int

// Global function registry for consistent function indices
var globalFunctionRegistry []string
var globalFunctionMap map[string]int

func initTypeRegistry() {
	globalTypeRegistry = []FunctionType{}
	globalTypeMap = make(map[string]int)

	// Type 0: print function (i64) -> ()
	printType := FunctionType{
		Parameters: []byte{0x7E}, // i64
		Results:    []byte{},     // void
	}
	globalTypeRegistry = append(globalTypeRegistry, printType)
	globalTypeMap["(i64)->()"] = 0
}

func initFunctionRegistry(functions []*ASTNode) {
	globalFunctionRegistry = []string{}
	globalFunctionMap = make(map[string]int)

	// Function 0 is print (imported)
	globalFunctionRegistry = append(globalFunctionRegistry, "print")
	globalFunctionMap["print"] = 0

	// Add user functions starting from index 1
	for _, fn := range functions {
		index := len(globalFunctionRegistry)
		globalFunctionRegistry = append(globalFunctionRegistry, fn.FunctionName)
		globalFunctionMap[fn.FunctionName] = index
	}
}

func EmitTypeSection(buf *bytes.Buffer, functions []*ASTNode) {
	writeByte(buf, 0x01) // type section id

	var sectionBuf bytes.Buffer

	// Initialize registries
	initTypeRegistry()
	initFunctionRegistry(functions)

	if len(functions) == 0 {
		// Legacy path - add main function type (void -> void)
		mainType := FunctionType{
			Parameters: []byte{},
			Results:    []byte{},
		}
		globalTypeRegistry = append(globalTypeRegistry, mainType)
	} else {
		// Register types for user functions
		for _, fn := range functions {
			sig := generateFunctionSignature(fn)
			if _, exists := globalTypeMap[sig]; !exists {
				globalTypeMap[sig] = len(globalTypeRegistry)
				globalTypeRegistry = append(globalTypeRegistry, createFunctionType(fn))
			}
		}
	}

	writeLEB128(&sectionBuf, uint32(len(globalTypeRegistry)))

	// Emit each function type
	for _, funcType := range globalTypeRegistry {
		writeByte(&sectionBuf, 0x60) // func type
		writeLEB128(&sectionBuf, uint32(len(funcType.Parameters)))
		for _, param := range funcType.Parameters {
			writeByte(&sectionBuf, param)
		}
		writeLEB128(&sectionBuf, uint32(len(funcType.Results)))
		for _, result := range funcType.Results {
			writeByte(&sectionBuf, result)
		}
	}

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

// FunctionType represents a WASM function type signature
type FunctionType struct {
	Parameters []byte
	Results    []byte
}

// generateFunctionSignature creates a string signature for deduplication
func generateFunctionSignature(fn *ASTNode) string {
	sig := "("
	for i, param := range fn.Parameters {
		if i > 0 {
			sig += ","
		}
		sig += wasmTypeString(param.Type)
	}
	sig += ")->("
	if fn.ReturnType != nil {
		sig += wasmTypeString(fn.ReturnType)
	}
	sig += ")"
	return sig
}

// createFunctionType creates a FunctionType from a function AST node
func createFunctionType(fn *ASTNode) FunctionType {
	var params []byte
	for _, param := range fn.Parameters {
		params = append(params, wasmTypeByte(param.Type))
	}

	var results []byte
	if fn.ReturnType != nil {
		results = append(results, wasmTypeByte(fn.ReturnType))
	}

	return FunctionType{
		Parameters: params,
		Results:    results,
	}
}

// wasmTypeByte returns the WASM type byte for a TypeNode
func wasmTypeByte(typeNode *TypeNode) byte {
	if typeNode.Kind == TypeBuiltin && typeNode.String == "I64" {
		return 0x7E // i64
	}
	if typeNode.Kind == TypePointer {
		return 0x7F // i32 (pointer)
	}
	if typeNode.Kind == TypeStruct {
		return 0x7F // i32 (struct parameters are passed as pointers)
	}
	panic("Unsupported type for WASM: " + TypeToString(typeNode))
}

// wasmTypeString returns the WASM type string for a TypeNode
func wasmTypeString(typeNode *TypeNode) string {
	if typeNode.Kind == TypeBuiltin && typeNode.String == "I64" {
		return "i64"
	}
	if typeNode.Kind == TypePointer {
		return "i32"
	}
	if typeNode.Kind == TypeStruct {
		// Struct parameters are passed as i32 pointers in WASM
		return "i32"
	}
	panic("Unsupported type for WASM: " + TypeToString(typeNode))
}

func EmitFunctionSection(buf *bytes.Buffer, functions []*ASTNode) {
	writeByte(buf, 0x03) // function section id

	var sectionBuf bytes.Buffer

	if len(functions) == 0 {
		// Legacy path - emit single main function
		writeLEB128(&sectionBuf, 1) // 1 function
		writeLEB128(&sectionBuf, 1) // type index 1 (void -> void)
	} else {
		writeLEB128(&sectionBuf, uint32(len(functions))) // number of functions

		// For each function, emit its type index
		for _, fn := range functions {
			typeIndex := findFunctionTypeIndex(fn)
			writeLEB128(&sectionBuf, uint32(typeIndex))
		}
	}

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

// findFunctionTypeIndex finds the type index for a function
func findFunctionTypeIndex(fn *ASTNode) int {
	sig := generateFunctionSignature(fn)
	if index, exists := globalTypeMap[sig]; exists {
		return index
	}
	panic("Function type not found in registry: " + sig)
}

// findUserFunctionIndex finds the WASM index for a user-defined function
func findUserFunctionIndex(functionName string) int {
	if index, exists := globalFunctionMap[functionName]; exists {
		return index
	}
	panic("Function not found in registry: " + functionName)
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

	// Find main function index - fallback to index 1 if main not found
	var mainIndex int
	if globalFunctionMap != nil {
		if index, exists := globalFunctionMap["main"]; exists {
			mainIndex = index
		} else {
			// Legacy fallback for single-expression tests
			mainIndex = 1
		}
	} else {
		// Legacy fallback when no function registry
		mainIndex = 1
	}
	writeLEB128(&sectionBuf, uint32(mainIndex))

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

func EmitCodeSection(buf *bytes.Buffer, functions []*ASTNode, symbolTable *SymbolTable) {
	writeByte(buf, 0x0A) // code section id

	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, uint32(len(functions))) // number of function bodies

	// Emit each function body
	for _, fn := range functions {
		emitSingleFunction(&sectionBuf, fn, symbolTable)
	}

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

// emitSingleFunction emits the code for a single function
func emitSingleFunction(buf *bytes.Buffer, fn *ASTNode, symbolTable *SymbolTable) {
	// Use unified local management
	localCtx := BuildLocalContext(fn.Body, fn.Parameters, symbolTable)

	// Generate function body
	var bodyBuf bytes.Buffer

	// Generate WASM locals declaration
	emitLocalDeclarations(&bodyBuf, localCtx)

	// Generate frame setup if needed
	if localCtx.FrameSize > 0 {
		EmitFrameSetupFromContext(&bodyBuf, localCtx)
	}

	// Generate function body
	EmitStatement(&bodyBuf, fn.Body, localCtx)
	writeByte(&bodyBuf, END) // end instruction

	// Write function body size and content
	writeLEB128(buf, uint32(bodyBuf.Len())) // function body size
	writeBytes(buf, bodyBuf.Bytes())
}

// EmitStatement generates WASM bytecode for statements
func EmitStatement(buf *bytes.Buffer, node *ASTNode, localCtx *LocalContext) {
	switch node.Kind {
	case NodeVar:
		// Variable declarations don't generate runtime code
		// (locals are declared in function header)
		break

	case NodeBlock:
		// Emit all statements in the block
		for _, stmt := range node.Children {
			EmitStatement(buf, stmt, localCtx)
		}

	case NodeCall:
		// Handle expression statements (e.g., print calls)
		EmitExpression(buf, node, localCtx)

	case NodeReturn:
		// Return statement
		if len(node.Children) > 0 {
			// Function returns a value - emit the expression
			EmitExpression(buf, node.Children[0], localCtx)
		}
		// WASM return instruction (implicitly returns the value on stack)
		writeByte(buf, 0x0F) // RETURN opcode

	case NodeIf:
		// If statement compilation
		// Structure: [condition, then_block, condition2?, else_block2?, ...]

		// Emit condition for initial if
		EmitExpression(buf, node.Children[0], localCtx)

		// Start if block
		writeByte(buf, 0x04) // if opcode
		writeByte(buf, 0x40) // block type: void

		// Emit then block
		EmitStatement(buf, node.Children[1], localCtx)

		// Handle else/else-if clauses
		i := 2
		for i < len(node.Children) {
			writeByte(buf, 0x05) // else opcode

			// Check if this is an else-if (condition is not nil) or final else (condition is nil)
			if node.Children[i] != nil {
				// else-if: emit condition and start new if block
				EmitExpression(buf, node.Children[i], localCtx)
				writeByte(buf, 0x04) // nested if opcode
				writeByte(buf, 0x40) // block type: void

				// Emit the else-if block
				EmitStatement(buf, node.Children[i+1], localCtx)
				i += 2
			} else {
				// final else: emit else block directly
				EmitStatement(buf, node.Children[i+1], localCtx)
				break
			}
		}

		// End all if blocks (one end for each if we opened)
		ifCount := 1 // Initial if
		for j := 2; j < i; j += 2 {
			if node.Children[j] != nil {
				ifCount++ // else-if adds another if
			}
		}
		for k := 0; k < ifCount; k++ {
			writeByte(buf, 0x0B) // end opcode
		}

	default:
		// For now, treat unknown statements as expressions
		EmitExpression(buf, node, localCtx)
	}
}

// Emit WASM code to calculate struct field address and leave it on stack
func emitStructFieldAddress(buf *bytes.Buffer, targetLocal *LocalVarInfo, fieldOffset uint32, localCtx *LocalContext) {
	if targetLocal.Storage == VarStorageParameterLocal {
		// Struct parameter - load pointer from parameter
		writeByte(buf, LOCAL_GET)
		writeLEB128(buf, targetLocal.Address)

		if fieldOffset > 0 {
			writeByte(buf, I32_CONST)
			writeLEB128Signed(buf, int64(fieldOffset))
			writeByte(buf, I32_ADD)
		}
	} else {
		// Local struct variable - calculate from frame pointer
		writeByte(buf, LOCAL_GET)
		writeLEB128(buf, localCtx.FramePointerIndex)

		totalOffset := targetLocal.Address + fieldOffset
		if totalOffset > 0 {
			writeByte(buf, I32_CONST)
			writeLEB128Signed(buf, int64(totalOffset))
			writeByte(buf, I32_ADD)
		}
	}
}

func EmitExpression(buf *bytes.Buffer, node *ASTNode, localCtx *LocalContext) {
	switch node.Kind {
	case NodeInteger:
		writeByte(buf, I64_CONST)
		writeLEB128Signed(buf, node.Integer)

	case NodeIdent:
		// Variable reference
		if node.Symbol == nil {
			panic("Undefined variable: " + node.String)
		}
		targetLocal := localCtx.FindVariable(node.Symbol)
		if targetLocal == nil {
			panic("Variable not found in local context: " + node.String)
		}

		if targetLocal.Storage == VarStorageLocal || targetLocal.Storage == VarStorageParameterLocal {
			// Local variable - emit local.get
			writeByte(buf, LOCAL_GET)
			writeLEB128(buf, targetLocal.Address)
		} else {
			// Stack variable
			if targetLocal.Symbol.Type.Kind == TypeStruct {
				// For struct variables, return the address of the struct (not the value)
				writeByte(buf, LOCAL_GET)
				writeLEB128(buf, localCtx.FramePointerIndex)

				// Add variable offset if not zero
				if targetLocal.Address > 0 {
					writeByte(buf, I64_CONST)
					writeLEB128Signed(buf, int64(targetLocal.Address))
					writeByte(buf, I64_ADD)
				}
				// Don't load - leave address on stack for struct operations
			} else {
				// Non-struct stack variable - load from memory
				writeByte(buf, LOCAL_GET)
				writeLEB128(buf, localCtx.FramePointerIndex)

				// Add variable offset if not zero
				if targetLocal.Address > 0 {
					writeByte(buf, I32_CONST)
					writeLEB128Signed(buf, int64(targetLocal.Address))
					writeByte(buf, I32_ADD)
				}

				// Load the value from memory
				writeByte(buf, I64_LOAD) // Load i64 from memory
				writeByte(buf, 0x03)     // alignment (8 bytes = 2^3)
				writeByte(buf, 0x00)     // offset
			}
		}

	case NodeBinary:
		if node.Op == "=" {
			// Assignment: LHS = RHS
			lhs := node.Children[0]
			rhs := node.Children[1]

			if lhs.Kind == NodeIdent {
				// Variable assignment: var = value
				if lhs.Symbol == nil {
					panic("Undefined variable: " + lhs.String)
				}
				targetLocal := localCtx.FindVariable(lhs.Symbol)
				if targetLocal == nil {
					panic("Variable not found in local context: " + lhs.String)
				}

				if targetLocal.Symbol.Type.Kind == TypeStruct {
					// Struct assignment (copy) using memory.copy
					structSize := uint32(GetTypeSize(targetLocal.Symbol.Type))

					// Get destination address
					writeByte(buf, LOCAL_GET)
					writeLEB128(buf, localCtx.FramePointerIndex)

					// Add variable offset if not zero
					if targetLocal.Address > 0 {
						writeByte(buf, I32_CONST)
						writeLEB128Signed(buf, int64(targetLocal.Address))
						writeByte(buf, I32_ADD)
					}

					// Get source address (RHS should evaluate to struct address)
					EmitExpression(buf, rhs, localCtx) // RHS address

					// Push size for memory.copy
					writeByte(buf, I32_CONST)
					writeLEB128(buf, structSize)

					// Emit memory.copy instruction
					// Stack: [dst_addr, src_addr, size]
					writeByte(buf, 0xFC) // Multi-byte instruction prefix
					writeLEB128(buf, 10) // memory.copy opcode
					writeByte(buf, 0x00) // dst memory index (0)
					writeByte(buf, 0x00) // src memory index (0)

				} else if targetLocal.Storage == VarStorageLocal || targetLocal.Storage == VarStorageParameterLocal {
					// Local variable - emit local.set
					EmitExpression(buf, rhs, localCtx) // RHS value
					writeByte(buf, LOCAL_SET)
					writeLEB128(buf, targetLocal.Address)
				} else {
					// Stack variable - store to memory
					// First get the address (before evaluating RHS)
					writeByte(buf, LOCAL_GET)
					writeLEB128(buf, localCtx.FramePointerIndex)

					// Add variable offset if not zero
					if targetLocal.Address > 0 {
						writeByte(buf, I32_CONST)
						writeLEB128Signed(buf, int64(targetLocal.Address))
						writeByte(buf, I32_ADD)
					}

					// Now evaluate the RHS value
					EmitExpression(buf, rhs, localCtx) // RHS value

					// Stack is now: [address_i32, value] - perfect for i64.store
					writeByte(buf, I64_STORE) // Store i64 to memory
					writeByte(buf, 0x03)      // alignment (8 bytes = 2^3)
					writeByte(buf, 0x00)      // offset
				}
			} else if lhs.Kind == NodeUnary && lhs.Op == "*" {
				// Pointer dereference assignment: ptr* = value
				// First get the address where to store
				EmitExpression(buf, lhs.Children[0], localCtx) // Get pointer value (i32)

				// Now evaluate the RHS value
				EmitExpression(buf, rhs, localCtx) // RHS value

				// Stack is now: [address_i32, value] - perfect for i64.store
				writeByte(buf, I64_STORE) // Store i64 to memory
				writeByte(buf, 0x03)      // alignment (8 bytes = 2^3)
				writeByte(buf, 0x00)      // offset
			} else if lhs.Kind == NodeDot {
				// Field assignment: struct.field = value
				baseExpr := lhs.Children[0]
				fieldName := lhs.FieldName

				// Get the base struct variable
				if baseExpr.Kind != NodeIdent {
					panic("Only direct variable field assignment supported currently")
				}

				if baseExpr.Symbol == nil {
					panic("Undefined struct variable: " + baseExpr.String)
				}
				targetLocal := localCtx.FindVariable(baseExpr.Symbol)
				if targetLocal == nil {
					panic("Variable not found in local context: " + baseExpr.String)
				}

				// Determine struct type
				var structType *TypeNode
				if targetLocal.Symbol.Type.Kind == TypeStruct {
					// Direct struct variable
					structType = targetLocal.Symbol.Type
				} else if targetLocal.Symbol.Type.Kind == TypePointer && targetLocal.Symbol.Type.Child.Kind == TypeStruct {
					// Struct parameter (pointer to struct)
					structType = targetLocal.Symbol.Type.Child
				} else {
					panic("Field assignment on non-struct variable")
				}

				// Find field in struct definition
				var fieldOffset uint32
				var fieldType *TypeNode
				found := false
				for _, field := range structType.Fields {
					if field.Name == fieldName {
						fieldOffset = field.Offset
						fieldType = field.Type
						found = true
						break
					}
				}
				if !found {
					panic("Field not found in struct: " + fieldName)
				}

				// Generate address calculation (before evaluating RHS)
				emitStructFieldAddress(buf, targetLocal, fieldOffset, localCtx)

				// Now evaluate the RHS value
				EmitExpression(buf, rhs, localCtx) // RHS value

				// Store the field value
				if isWASMI64Type(fieldType) {
					writeByte(buf, I64_STORE) // Store i64 to memory
					writeByte(buf, 0x03)      // alignment (8 bytes = 2^3)
					writeByte(buf, 0x00)      // offset
				} else {
					panic("Non-I64 field types not supported in WASM yet")
				}
			} else {
				panic("Invalid assignment target - must be variable, field access, or pointer dereference")
			}
		} else {
			// Regular binary operations
			EmitExpression(buf, node.Children[0], localCtx) // left operand
			EmitExpression(buf, node.Children[1], localCtx) // right operand
			writeByte(buf, getBinaryOpcode(node.Op))

			// Comparison operations return i32 (Boolean)
			// Don't extend to i64 since Boolean type should be i32
		}

	case NodeCall:
		if len(node.Children) > 0 && node.Children[0].Kind == NodeIdent {
			functionName := node.Children[0].String

			if functionName == "print" {
				// Built-in print function
				if len(node.Children) > 1 {
					arg := node.Children[1]
					EmitExpression(buf, arg, localCtx) // argument

					// If the argument is a pointer type, we need to widen it from i32 to i64
					// because print() expects i64
					if isWASMI32Type(arg.TypeAST) {
						writeByte(buf, I64_EXTEND_I32_S) // Convert i32 pointer to i64
					}
				}
				writeByte(buf, CALL) // call instruction
				writeLEB128(buf, 0)  // function index 0 (print import)
			} else {
				// User-defined function call
				// Emit arguments in order
				for i := 1; i < len(node.Children); i++ {
					arg := node.Children[i]

					// Check if this argument is a struct that needs to be copied
					if arg.Kind == NodeIdent && arg.Symbol != nil {
						argLocal := localCtx.FindVariable(arg.Symbol)
						if argLocal != nil && argLocal.Symbol.Type.Kind == TypeStruct {
							// This is a struct argument - we need to copy it
							// First allocate space on the stack for the copy
							structSize := uint32(GetTypeSize(argLocal.Symbol.Type))

							// Get current tstack pointer as destination address
							writeByte(buf, GLOBAL_GET)
							writeLEB128(buf, 0) // tstack global index

							// Get source address
							EmitExpression(buf, arg, localCtx)

							// Copy size
							writeByte(buf, I32_CONST)
							writeLEB128(buf, structSize)

							// Emit memory.copy to copy struct
							writeByte(buf, 0xFC) // Multi-byte instruction prefix
							writeLEB128(buf, 10) // memory.copy opcode
							writeByte(buf, 0x00) // dst memory index (0)
							writeByte(buf, 0x00) // src memory index (0)

							// Push the copy address as the function argument
							writeByte(buf, GLOBAL_GET)
							writeLEB128(buf, 0) // tstack global index (points to the copy)

							// Advance tstack pointer past the copy
							writeByte(buf, GLOBAL_GET)
							writeLEB128(buf, 0) // tstack global index
							writeByte(buf, I32_CONST)
							writeLEB128(buf, structSize)
							writeByte(buf, I32_ADD)
							writeByte(buf, GLOBAL_SET)
							writeLEB128(buf, 0) // tstack global index

							continue
						}
					}

					// Regular argument (not a struct needing copy)
					EmitExpression(buf, arg, localCtx)
				}

				// Find function index
				functionIndex := findUserFunctionIndex(functionName)
				writeByte(buf, CALL)
				writeLEB128(buf, uint32(functionIndex))
			}
		}

	case NodeUnary:
		if node.Op == "&" {
			// Address-of operator
			EmitAddressOf(buf, node.Children[0], localCtx)
		} else if node.Op == "*" {
			// Dereference operator
			EmitExpression(buf, node.Children[0], localCtx) // Get the pointer value (i32)
			writeByte(buf, I64_LOAD)                        // Load i64 from memory using i32 address
			writeByte(buf, 0x03)                            // alignment (8 bytes = 2^3)
			writeByte(buf, 0x00)                            // offset
		} else if node.Op == "!" {
			// Handle existing unary not operator
			EmitExpression(buf, node.Children[0], localCtx)
			// TODO: Implement logical not operation
			panic("Unary not operator (!) not yet implemented")
		}

	case NodeDot:
		// Field access: struct.field
		baseExpr := node.Children[0]
		fieldName := node.FieldName

		// Get the base struct variable
		if baseExpr.Kind != NodeIdent {
			panic("Only direct variable field access supported currently")
		}

		if baseExpr.Symbol == nil {
			panic("Undefined struct variable: " + baseExpr.String)
		}
		targetLocal := localCtx.FindVariable(baseExpr.Symbol)
		if targetLocal == nil {
			panic("Variable not found in local context: " + baseExpr.String)
		}

		// Determine struct type
		var structType *TypeNode
		if targetLocal.Symbol.Type.Kind == TypeStruct {
			// Direct struct variable
			structType = targetLocal.Symbol.Type
		} else if targetLocal.Symbol.Type.Kind == TypePointer && targetLocal.Symbol.Type.Child.Kind == TypeStruct {
			// Struct parameter (pointer to struct)
			structType = targetLocal.Symbol.Type.Child
		} else {
			panic("Field access on non-struct variable")
		}

		// Find field in struct definition
		var fieldOffset uint32
		var fieldType *TypeNode
		found := false
		for _, field := range structType.Fields {
			if field.Name == fieldName {
				fieldOffset = field.Offset
				fieldType = field.Type
				found = true
				break
			}
		}
		if !found {
			panic("Field not found in struct: " + fieldName)
		}

		// Generate address calculation
		emitStructFieldAddress(buf, targetLocal, fieldOffset, localCtx)

		// Load the field value
		if isWASMI64Type(fieldType) {
			writeByte(buf, I64_LOAD) // Load i64 from memory
			writeByte(buf, 0x03)     // alignment (8 bytes = 2^3)
			writeByte(buf, 0x00)     // offset
		} else {
			panic("Non-I64 field types not supported in WASM yet")
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
	// Build symbol table
	symbolTable := BuildSymbolTable(ast)

	// Perform type checking
	err := CheckProgram(ast, symbolTable)
	if err != nil {
		panic(err.Error())
	}

	// Extract functions from the program
	functions := extractFunctions(ast)

	var buf bytes.Buffer

	// Check if this is legacy expression compilation (no functions)
	if len(functions) == 0 {
		// Legacy path for single expressions
		return compileLegacyExpression(ast, symbolTable)
	}

	// Emit WASM module header and sections in streaming fashion
	EmitWASMHeader(&buf)
	EmitTypeSection(&buf, functions)              // function type definitions
	EmitImportSection(&buf)                       // print function + tstack global import
	EmitFunctionSection(&buf, functions)          // declare all functions
	EmitMemorySection(&buf)                       // memory for tstack operations
	EmitExportSection(&buf)                       // export main function
	EmitCodeSection(&buf, functions, symbolTable) // all function bodies

	return buf.Bytes()
}

// extractFunctions finds all function declarations in the program
func extractFunctions(ast *ASTNode) []*ASTNode {
	var functions []*ASTNode

	// For single statements, check if it's a function
	if ast.Kind == NodeFunc {
		return []*ASTNode{ast}
	}

	// For programs (blocks), find all functions
	if ast.Kind == NodeBlock {
		for _, child := range ast.Children {
			if child.Kind == NodeFunc {
				functions = append(functions, child)
			}
		}
	}

	return functions
}

// compileLegacyExpression compiles single expressions (backward compatibility)
func compileLegacyExpression(ast *ASTNode, symbolTable *SymbolTable) []byte {
	// Use same unified system
	localCtx := BuildLocalContext(ast, []FunctionParameter{}, symbolTable)

	// Generate function body
	var bodyBuf bytes.Buffer

	// Generate WASM with unified approach
	emitLocalDeclarations(&bodyBuf, localCtx)
	if localCtx.FrameSize > 0 {
		EmitFrameSetupFromContext(&bodyBuf, localCtx)
	}
	EmitStatement(&bodyBuf, ast, localCtx)
	writeByte(&bodyBuf, END) // end instruction

	// Build the full WASM module
	var buf bytes.Buffer
	EmitWASMHeader(&buf)
	EmitTypeSection(&buf, []*ASTNode{}) // empty functions for legacy
	EmitImportSection(&buf)
	EmitFunctionSection(&buf, []*ASTNode{}) // empty functions for legacy
	EmitMemorySection(&buf)
	EmitExportSection(&buf)

	// Emit code section with single function
	writeByte(&buf, 0x0A) // code section id
	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, 1)                     // 1 function
	writeLEB128(&sectionBuf, uint32(bodyBuf.Len())) // function body size
	writeBytes(&sectionBuf, bodyBuf.Bytes())
	writeLEB128(&buf, uint32(sectionBuf.Len()))
	writeBytes(&buf, sectionBuf.Bytes())

	return buf.Bytes()
}

// BuildLocalContext creates a unified LocalContext for both legacy and function compilation paths
func BuildLocalContext(ast *ASTNode, params []FunctionParameter, symbolTable *SymbolTable) *LocalContext {
	ctx := &LocalContext{}

	// Phase 1: Add parameters
	ctx.addParameters(params, symbolTable)

	// Phase 2: Collect body variables
	ctx.collectBodyVariables(ast, symbolTable)

	// Phase 3: Calculate frame pointer (if needed)
	ctx.calculateFramePointer()

	// Phase 4: Assign final WASM indices
	ctx.assignWASMIndices()

	return ctx
}

// addParameters adds function parameters to the LocalContext
func (ctx *LocalContext) addParameters(params []FunctionParameter, symbolTable *SymbolTable) {
	for _, param := range params {
		// Look up the parameter in the symbol table
		// Parameters are now added to the symbol table by BuildSymbolTable
		symbol := symbolTable.LookupVariable(param.Name)
		if symbol == nil {
			panic("Parameter " + param.Name + " not found in symbol table")
		}

		ctx.Variables = append(ctx.Variables, LocalVarInfo{
			Symbol:  symbol,
			Storage: VarStorageParameterLocal,
			// Address will be assigned later in assignWASMIndices
		})
		ctx.ParameterCount++
	}
}

// collectBodyVariables traverses AST to find all var declarations and address-of operations
func (ctx *LocalContext) collectBodyVariables(node *ASTNode, symbolTable *SymbolTable) {
	var frameOffset uint32 = 0

	var traverse func(*ASTNode)
	traverse = func(node *ASTNode) {
		switch node.Kind {
		case NodeStruct:
			// Don't traverse struct children - field declarations are not local variables
			return
		case NodeVar:
			// Extract variable name
			varName := node.Children[0].String

			// Skip variables with no type information
			if node.TypeAST == nil {
				break
			}

			resolvedType := node.TypeAST

			// Support I64, I64* (pointers are i32 in WASM), and other types
			if isWASMI32Type(resolvedType) || isWASMI64Type(resolvedType) {
				// Look up the variable in the symbol table
				symbol := symbolTable.LookupVariable(varName)
				if symbol == nil {
					panic("Variable not found in symbol table: " + varName)
				}

				ctx.Variables = append(ctx.Variables, LocalVarInfo{
					Symbol:  symbol,
					Storage: VarStorageLocal,
					// Address will be allocated later.
				})
			} else if resolvedType.Kind == TypeStruct {
				// Struct variables are always stored on tstack (addressed)
				structSize := uint32(GetTypeSize(resolvedType))

				// Look up the variable in the symbol table
				symbol := symbolTable.LookupVariable(varName)
				if symbol == nil {
					panic("Variable not found in symbol table: " + varName)
				}

				ctx.Variables = append(ctx.Variables, LocalVarInfo{
					Symbol:  symbol,
					Storage: VarStorageTStack,
					Address: frameOffset,
				})
				frameOffset += structSize
			}

		case NodeUnary:
			if node.Op == "&" {
				// This is an address-of operation
				child := node.Children[0]
				if child.Kind == NodeIdent {
					// Address of a variable - mark it as addressed
					varName := child.String
					ctx.markVariableAsAddressed(varName, &frameOffset)
				}
			}
		}

		// Recursively traverse children
		for _, child := range node.Children {
			if child != nil {
				traverse(child)
			}
		}
	}

	traverse(node)
	ctx.FrameSize = frameOffset
}

// markVariableAsAddressed converts a variable from VarStorageLocal to VarStorageTStack
func (ctx *LocalContext) markVariableAsAddressed(varName string, frameOffset *uint32) {
	for i := range ctx.Variables {
		if ctx.Variables[i].Symbol.Name == varName && ctx.Variables[i].Storage == VarStorageLocal {
			// Convert to addressed storage
			ctx.Variables[i].Storage = VarStorageTStack
			ctx.Variables[i].Address = *frameOffset
			*frameOffset += uint32(GetTypeSize(ctx.Variables[i].Symbol.Type))
			break
		}
	}
}

// calculateFramePointer determines if we need a frame pointer and reserves space
func (ctx *LocalContext) calculateFramePointer() {
	// Frame pointer is needed if we have any addressed variables
	if ctx.FrameSize > 0 {
		// Frame pointer will be assigned an index in assignWASMIndices
		ctx.I32LocalCount++ // Frame pointer is always i32
	}
}

// assignWASMIndices assigns WASM local indices according to the unified layout
func (ctx *LocalContext) assignWASMIndices() {
	wasmIndex := uint32(0)

	// Step 1: Assign parameter indices (parameters come first in WASM)
	for i := range ctx.Variables {
		if ctx.Variables[i].Storage == VarStorageParameterLocal {
			ctx.Variables[i].Address = wasmIndex
			wasmIndex++
		}
	}

	// Step 2: Assign i32 body locals (including eventual frame pointer space)
	for i := range ctx.Variables {
		if ctx.Variables[i].Storage == VarStorageLocal && isWASMI32Type(ctx.Variables[i].Symbol.Type) {
			ctx.Variables[i].Address = wasmIndex
			wasmIndex++
			ctx.I32LocalCount++
		}
	}

	// Step 3: Assign frame pointer if needed
	if ctx.FrameSize > 0 {
		ctx.FramePointerIndex = wasmIndex
		wasmIndex++
	}

	// Step 4: Assign i64 body locals
	for i := range ctx.Variables {
		if ctx.Variables[i].Storage == VarStorageLocal && isWASMI64Type(ctx.Variables[i].Symbol.Type) {
			ctx.Variables[i].Address = wasmIndex
			wasmIndex++
			ctx.I64LocalCount++
		}
	}
}

// FindVariable looks up a variable by symbol in the LocalContext
func (ctx *LocalContext) FindVariable(symbol *SymbolInfo) *LocalVarInfo {
	if symbol == nil {
		return nil
	}
	for i := range ctx.Variables {
		if ctx.Variables[i].Symbol == symbol {
			return &ctx.Variables[i]
		}
	}
	return nil
}

// emitLocalDeclarations generates WASM local variable declarations
func emitLocalDeclarations(buf *bytes.Buffer, localCtx *LocalContext) {
	// Count locals by type (excluding parameters)
	i32Count := localCtx.countBodyLocalsByType("I32")
	i64Count := localCtx.countBodyLocalsByType("I64")

	// Add frame pointer to i32 count if needed
	if localCtx.FrameSize > 0 {
		i32Count++
	}

	// Emit local declarations
	groupCount := 0
	if i32Count > 0 {
		groupCount++
	}
	if i64Count > 0 {
		groupCount++
	}

	writeLEB128(buf, uint32(groupCount))

	if i32Count > 0 {
		writeLEB128(buf, uint32(i32Count))
		writeByte(buf, 0x7F) // i32
	}

	if i64Count > 0 {
		writeLEB128(buf, uint32(i64Count))
		writeByte(buf, 0x7E) // i64
	}
}

// countBodyLocalsByType counts how many body locals (not parameters) are of a given type
func (ctx *LocalContext) countBodyLocalsByType(typeName string) uint32 {
	count := uint32(0)
	for _, local := range ctx.Variables {
		if local.Storage == VarStorageLocal {
			if (typeName == "I32" && isWASMI32Type(local.Symbol.Type)) ||
				(typeName == "I64" && isWASMI64Type(local.Symbol.Type)) {
				count++
			}
		}
	}
	return count
}

// collectLocalVariables traverses AST once to find all var declarations and address-of operations
// Returns the locals list and the total frame size for addressed variables
func collectLocalVariables(node *ASTNode) ([]LocalVarInfo, uint32) {
	var locals []LocalVarInfo
	var frameOffset uint32 = 0

	var traverse func(*ASTNode)
	traverse = func(node *ASTNode) {
		switch node.Kind {
		case NodeStruct:
			// Don't traverse struct children - field declarations are not local variables
			return
		case NodeVar:
			// Extract variable name
			varName := node.Children[0].String

			// Skip variables with no type information
			if node.TypeAST == nil {
				break
			}

			resolvedType := node.TypeAST

			// Support I64, I64* (pointers are i32 in WASM), and other types
			if isWASMI32Type(resolvedType) || isWASMI64Type(resolvedType) {
				// Create a temporary SymbolInfo for testing purposes
				symbol := &SymbolInfo{
					Name:     varName,
					Type:     resolvedType,
					Assigned: false,
				}

				locals = append(locals, LocalVarInfo{
					Symbol:  symbol,
					Storage: VarStorageLocal,
					// Address will be allocated later.
				})
			} else if resolvedType.Kind == TypeStruct {
				// Struct variables are always stored on tstack (addressed)
				structSize := uint32(GetTypeSize(resolvedType))

				// Create a temporary SymbolInfo for testing purposes
				symbol := &SymbolInfo{
					Name:     varName,
					Type:     resolvedType,
					Assigned: false,
				}

				locals = append(locals, LocalVarInfo{
					Symbol:  symbol,
					Storage: VarStorageTStack,
					Address: frameOffset,
				})
				frameOffset += structSize
			}

		case NodeUnary:
			if node.Op == "&" {
				// This is an address-of operation
				child := node.Children[0]
				if child.Kind == NodeIdent {
					// Address of a variable - mark it as addressed
					varName := child.String
					for i := range locals {
						if locals[i].Symbol.Name == varName {
							if locals[i].Storage == VarStorageLocal {
								// Change from local to stack storage and assign frame offset
								locals[i].Storage = VarStorageTStack
								locals[i].Address = frameOffset
								frameOffset += 8 // Each I64 value takes 8 bytes
							}
							break
						}
					}
				}
			}
			// Continue scanning the operand
		}
		for _, child := range node.Children {
			if child != nil {
				traverse(child)
			}
		}
	}

	traverse(node)

	// Reassign WASM local indices: i32 locals first, then i64 locals
	var i32Index uint32 = 0
	var i64Index uint32 = 0

	// Count i32 locals to know where i64 locals start
	i32Count := uint32(0)
	for i := range locals {
		if locals[i].Storage == VarStorageLocal && isWASMI32Type(locals[i].Symbol.Type) {
			i32Count++
		}
	}

	// Calculate total i32 locals
	// Note: frame pointer is handled separately by the compilation phase
	totalI32Locals := i32Count

	// Assign correct indices
	for i := range locals {
		if locals[i].Storage == VarStorageLocal {
			if isWASMI32Type(locals[i].Symbol.Type) {
				locals[i].Address = i32Index
				i32Index++
			} else if isWASMI64Type(locals[i].Symbol.Type) {
				locals[i].Address = totalI32Locals + i64Index
				i64Index++
			}
		}
	}

	return locals, frameOffset
}

// EmitFrameSetup generates frame setup code at function entry
func EmitFrameSetup(buf *bytes.Buffer, locals []LocalVarInfo, frameSize uint32, framePointerIndex uint32) {
	// Set frame pointer to current tstack pointer: frame_pointer = tstack_pointer
	writeByte(buf, GLOBAL_GET) // global.get $tstack_pointer
	writeLEB128(buf, 0)        // tstack global index (0)
	writeByte(buf, LOCAL_SET)  // local.set $frame_pointer
	writeLEB128(buf, framePointerIndex)

	// Advance tstack pointer by frame size: tstack_pointer += frame_size
	writeByte(buf, GLOBAL_GET) // global.get $tstack_pointer
	writeLEB128(buf, 0)        // tstack global index (0)
	writeByte(buf, I32_CONST)  // i32.const frame_size
	writeLEB128Signed(buf, int64(frameSize))
	writeByte(buf, I32_ADD)    // i32.add
	writeByte(buf, GLOBAL_SET) // global.set $tstack_pointer
	writeLEB128(buf, 0)        // tstack global index (0)
}

// EmitFrameSetupFromContext generates frame setup code using LocalContext
func EmitFrameSetupFromContext(buf *bytes.Buffer, localCtx *LocalContext) {
	// Set frame pointer to current tstack pointer: frame_pointer = tstack_pointer
	writeByte(buf, GLOBAL_GET) // global.get $tstack_pointer
	writeLEB128(buf, 0)        // tstack global index (0)
	writeByte(buf, LOCAL_SET)  // local.set $frame_pointer
	writeLEB128(buf, localCtx.FramePointerIndex)

	// Advance tstack pointer by frame size: tstack_pointer += frame_size
	writeByte(buf, GLOBAL_GET) // global.get $tstack_pointer
	writeLEB128(buf, 0)        // tstack global index (0)
	writeByte(buf, I32_CONST)  // i32.const frame_size
	writeLEB128Signed(buf, int64(localCtx.FrameSize))
	writeByte(buf, I32_ADD)    // i32.add
	writeByte(buf, GLOBAL_SET) // global.set $tstack_pointer
	writeLEB128(buf, 0)        // tstack global index (0)
}

// EmitAddressOf generates code for address-of operations
func EmitAddressOf(buf *bytes.Buffer, operand *ASTNode, localCtx *LocalContext) {
	if operand.Kind == NodeIdent {
		// Lvalue case: &variable
		// Find the variable in locals
		if operand.Symbol == nil {
			panic("Undefined variable in address-of: " + operand.String)
		}
		targetLocal := localCtx.FindVariable(operand.Symbol)
		if targetLocal == nil {
			panic("Variable not found in local context: " + operand.String)
		}

		if targetLocal.Storage != VarStorageTStack {
			panic("Variable " + operand.String + " is not addressed but address-of is used")
		}

		// Load frame pointer
		writeByte(buf, LOCAL_GET)
		writeLEB128(buf, localCtx.FramePointerIndex)

		// Add variable offset
		if targetLocal.Address > 0 {
			writeByte(buf, I32_CONST)
			writeLEB128Signed(buf, int64(targetLocal.Address))
			writeByte(buf, I32_ADD)
		}
	} else {
		// Rvalue case: &(expression)
		// Save current tstack pointer as result first
		writeByte(buf, GLOBAL_GET) // global.get $tstack_pointer
		writeLEB128(buf, 0)        // tstack global index (0)

		// Get address for store operation: Stack: [result_addr, store_addr_i32]
		writeByte(buf, GLOBAL_GET) // global.get $tstack_pointer -> Stack: [result_addr, store_addr]
		writeLEB128(buf, 0)        // tstack global index (0)

		// Evaluate expression to get value: Stack: [result_addr, store_addr_i32, value]
		EmitExpression(buf, operand, localCtx)

		// Store value at address: i64.store expects [address, value]
		writeByte(buf, I64_STORE) // i64.store -> Stack: [result_addr]
		writeByte(buf, 0x03)      // alignment (2^3 = 8 byte alignment)
		writeLEB128(buf, 0)       // offset (0)

		// Advance tstack pointer by 8 bytes
		writeByte(buf, GLOBAL_GET) // global.get $tstack_pointer
		writeLEB128(buf, 0)        // tstack global index (0)
		writeByte(buf, I32_CONST)  // i32.const 8
		writeLEB128Signed(buf, 8)
		writeByte(buf, I32_ADD)    // i32.add
		writeByte(buf, GLOBAL_SET) // global.set $tstack_pointer
		writeLEB128(buf, 0)        // tstack global index (0)

		// Stack now has [result_addr] which is what we want to return
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
	NodeStruct   NodeKind = "NodeStruct"
	NodeDot      NodeKind = "NodeDot"
	NodeFunc     NodeKind = "NodeFunc"
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
	// NodeVar:
	TypeAST *TypeNode // Type information for variable declarations
	// NodeIdent (variable references):
	Symbol *SymbolInfo // Direct reference to symbol in symbol table
	// NodeDot:
	FieldName string // Field name for field access (s.field)
	// NodeFunc:
	FunctionName string              // Function name
	Parameters   []FunctionParameter // Function parameters
	ReturnType   *TypeNode           // Return type (nil for void)
	Body         *ASTNode            // Function body (block statement)
}

// TypeKind represents different kinds of types
type TypeKind string

const (
	TypeBuiltin TypeKind = "TypeBuiltin" // I64, Bool
	TypePointer TypeKind = "TypePointer" // *T
	TypeStruct  TypeKind = "TypeStruct"  // MyStruct
)

// TypeNode represents a type in the type system
type TypeNode struct {
	Kind TypeKind

	// For TypeBuiltin, TypeStruct
	String string // "I64", "Bool", name of struct

	// For TypePointer
	Child *TypeNode

	// For TypeStruct
	Fields []StructField // Field definitions (only for struct declarations)
}

// StructField represents a field in a struct
type StructField struct {
	Name   string
	Type   *TypeNode
	Offset uint32 // Byte offset in struct layout
}

// Built-in types
var (
	TypeI64  = &TypeNode{Kind: TypeBuiltin, String: "I64"}
	TypeBool = &TypeNode{Kind: TypeBuiltin, String: "Bool"}
)

// Type utility functions

// TypesEqual checks if two TypeNodes are equal
func TypesEqual(a, b *TypeNode) bool {
	if a.Kind != b.Kind {
		return false
	}

	switch a.Kind {
	case TypeBuiltin:
		return a.String == b.String
	case TypePointer:
		return TypesEqual(a.Child, b.Child)
	case TypeStruct:
		return a.String == b.String
	}
	// unreachable with current TypeKind values
	panic("Unknown TypeKind: " + string(a.Kind))
}

// GetTypeSize returns the size in bytes for WASM code generation
func GetTypeSize(t *TypeNode) int {
	switch t.Kind {
	case TypeBuiltin:
		switch t.String {
		case "I64":
			return 8
		case "Bool":
			return 1
		default:
			return 8 // default to 8 bytes
		}
	case TypePointer:
		return 8 // pointers are always 64-bit
	case TypeStruct:
		// Calculate struct size from fields
		if len(t.Fields) == 0 {
			return 0
		}
		lastField := t.Fields[len(t.Fields)-1]
		return int(lastField.Offset) + GetTypeSize(lastField.Type)
	}
	// unreachable with current TypeKind values
	panic("Unknown TypeKind: " + string(t.Kind))
}

// TypeToString converts TypeNode to string for display/debugging
func TypeToString(t *TypeNode) string {
	switch t.Kind {
	case TypeBuiltin:
		return t.String
	case TypePointer:
		return TypeToString(t.Child) + "*"
	case TypeStruct:
		return t.String
	}
	// unreachable with current TypeKind values
	panic("Unknown TypeKind: " + string(t.Kind))
}

// getBuiltinType returns the built-in type for a given name
func getBuiltinType(name string) *TypeNode {
	switch name {
	case "I64":
		return TypeI64
	case "Bool":
		return TypeBool
	default:
		return nil
	}
}

// isKnownUnsupportedType checks if a type name is a known unsupported built-in type
func isKnownUnsupportedType(name string) bool {
	switch name {
	case "string", "int", "float64", "byte", "rune", "uint64", "int32", "uint32":
		return true
	default:
		return false
	}
}

// isWASMI64Type checks if a TypeNode represents a type that maps to WASM I64
func isWASMI64Type(t *TypeNode) bool {
	if t == nil {
		return false
	}
	switch t.Kind {
	case TypeBuiltin:
		// Only I64 and Bool are known to map to WASM I64
		// Other types like "int", "string" are not supported in WASM generation
		return t.String == "I64" || t.String == "Bool"
	case TypePointer:
		return false // pointers are I32 in WASM
	case TypeStruct:
		return false // structs are stored in memory, not as I64 locals
	}
	// unreachable with current TypeKind values
	panic("Unknown TypeKind: " + string(t.Kind))
}

// isWASMI32Type checks if a TypeNode represents a type that maps to WASM I32
func isWASMI32Type(t *TypeNode) bool {
	if t == nil {
		return false
	}
	switch t.Kind {
	case TypeBuiltin:
		return t.String == "Bool" // Boolean type maps to I32 in WASM
	case TypePointer:
		return true // all pointers are I32 in WASM
	case TypeStruct:
		return false // structs are stored in memory, not as I32 locals
	}
	// unreachable with current TypeKind values
	panic("Unknown TypeKind: " + string(t.Kind))
}

// SymbolInfo represents information about a declared variable
type SymbolInfo struct {
	Name     string
	Type     *TypeNode
	Assigned bool // tracks if variable has been assigned a value
}

// FunctionInfo represents information about a declared function
type FunctionInfo struct {
	Name       string
	Parameters []FunctionParameter
	ReturnType *TypeNode // nil for void functions
	WasmIndex  uint32    // WASM function index
}

// FunctionParameter represents a function parameter
type FunctionParameter struct {
	Name    string
	Type    *TypeNode
	IsNamed bool // true for named parameters, false for positional
}

// SymbolTable tracks variable declarations and assignments
type SymbolTable struct {
	variables []SymbolInfo
	structs   []*TypeNode    // struct type definitions
	functions []FunctionInfo // function declarations
}

// TypeChecker holds state for type checking
type TypeChecker struct {
	errors      []string
	symbolTable *SymbolTable
}

// NewSymbolTable creates a new empty symbol table
func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		variables: make([]SymbolInfo, 0),
		structs:   make([]*TypeNode, 0),
		functions: make([]FunctionInfo, 0),
	}
}

// DeclareVariable adds a variable declaration to the symbol table
func (st *SymbolTable) DeclareVariable(name string, varType *TypeNode) error {
	// Check for duplicate declaration
	for _, v := range st.variables {
		if v.Name == name {
			return fmt.Errorf("error: variable '%s' already declared", name)
		}
	}

	st.variables = append(st.variables, SymbolInfo{
		Name:     name,
		Type:     varType,
		Assigned: false,
	})
	return nil
}

// AssignVariable marks a variable as assigned
func (st *SymbolTable) AssignVariable(name string) {
	for i := range st.variables {
		if st.variables[i].Name == name {
			st.variables[i].Assigned = true
			return
		}
	}
	panic(fmt.Sprintf("error: variable '%s' used before declaration", name))
}

// LookupVariable finds a variable in the symbol table
func (st *SymbolTable) LookupVariable(name string) *SymbolInfo {
	for i := range st.variables {
		if st.variables[i].Name == name {
			return &st.variables[i]
		}
	}
	return nil
}

// DeclareStruct adds a struct declaration to the symbol table
func (st *SymbolTable) DeclareStruct(structType *TypeNode) error {
	name := structType.String
	// Check for duplicate declaration
	for _, existingStruct := range st.structs {
		if existingStruct.String == name {
			return fmt.Errorf("error: struct '%s' already declared", name)
		}
	}

	st.structs = append(st.structs, structType)
	return nil
}

// LookupStruct finds a struct type by name
func (st *SymbolTable) LookupStruct(name string) *TypeNode {
	for _, structType := range st.structs {
		if structType.String == name {
			return structType
		}
	}
	return nil
}

// DeclareFunction adds a function declaration to the symbol table
func (st *SymbolTable) DeclareFunction(name string, parameters []FunctionParameter, returnType *TypeNode) error {
	// Check for duplicate declaration
	for _, fn := range st.functions {
		if fn.Name == name {
			return fmt.Errorf("function '%s' already declared", name)
		}
	}

	// Assign WASM index (builtin functions like print start at 0, user functions follow)
	wasmIndex := uint32(1 + len(st.functions)) // print is at index 0

	st.functions = append(st.functions, FunctionInfo{
		Name:       name,
		Parameters: parameters,
		ReturnType: returnType,
		WasmIndex:  wasmIndex,
	})
	return nil
}

// LookupFunction finds a function by name
func (st *SymbolTable) LookupFunction(name string) *FunctionInfo {
	for i := range st.functions {
		if st.functions[i].Name == name {
			return &st.functions[i]
		}
	}
	return nil
}

// ConvertStructASTToType converts a struct AST node to a TypeNode with calculated field offsets
func ConvertStructASTToType(structAST *ASTNode) *TypeNode {
	if structAST.Kind != NodeStruct {
		panic("Expected NodeStruct")
	}

	structName := structAST.String
	var fields []StructField
	var currentOffset uint32 = 0

	// Process field declarations
	for _, fieldAST := range structAST.Children {
		if fieldAST.Kind != NodeVar {
			continue // skip non-field declarations
		}

		fieldName := fieldAST.Children[0].String
		fieldType := fieldAST.TypeAST
		fieldSize := GetTypeSize(fieldType)

		fields = append(fields, StructField{
			Name:   fieldName,
			Type:   fieldType,
			Offset: currentOffset,
		})

		currentOffset += uint32(fieldSize)
	}

	return &TypeNode{
		Kind:   TypeStruct,
		String: structName,
		Fields: fields,
	}
}

// BuildSymbolTable traverses the AST to build a symbol table with variable declarations
// and populates Symbol references in NodeIdent nodes
func BuildSymbolTable(ast *ASTNode) *SymbolTable {
	st := NewSymbolTable()

	// Pass 1: collect all struct and variable declarations
	// (must be done first to handle variables declared in nested scopes)
	var collectDeclarations func(*ASTNode)
	collectDeclarations = func(node *ASTNode) {
		switch node.Kind {
		case NodeStruct:
			// Convert struct AST to TypeNode and declare it
			structType := ConvertStructASTToType(node)
			err := st.DeclareStruct(structType)
			if err != nil {
				panic(err.Error())
			}

		case NodeVar:
			// Extract variable name and type
			varName := node.Children[0].String
			varType := node.TypeAST

			// Skip variables with no type information
			if varType == nil {
				break
			}

			// Only add supported types to symbol table for type checking
			// This includes I64, Bool, pointers, and struct types
			if isWASMI64Type(varType) || isWASMI32Type(varType) || varType.Kind == TypeStruct {
				// For struct types, resolve the struct name to actual struct type
				if varType.Kind == TypeStruct {
					// Look up the struct definition
					structDef := st.LookupStruct(varType.String)
					if structDef != nil {
						// Use the complete struct definition
						varType = structDef
						node.TypeAST = varType
					}
				}

				err := st.DeclareVariable(varName, varType)
				if err != nil {
					panic(err.Error())
				}

				// Struct variables are "assigned" when declared (they have allocated memory)
				if varType.Kind == TypeStruct {
					st.AssignVariable(varName)
				}
			}

		case NodeFunc:
			// Resolve struct types in function parameters and update the AST node
			for i, param := range node.Parameters {
				resolvedType := param.Type

				// For pointer-to-struct parameters, resolve the child struct type
				if resolvedType.Kind == TypePointer && resolvedType.Child.Kind == TypeStruct {
					structDef := st.LookupStruct(resolvedType.Child.String)
					if structDef != nil {
						// Create new pointer type with resolved struct child
						resolvedType = &TypeNode{
							Kind:  TypePointer,
							Child: structDef,
						}
						// Update the AST node with resolved type
						node.Parameters[i].Type = resolvedType
					}
				}
			}

			// Declare function with resolved parameter types (from updated AST node)
			err := st.DeclareFunction(node.FunctionName, node.Parameters, node.ReturnType)
			if err != nil {
				panic(err.Error())
			}

			// Add function parameters to the symbol table
			// Since we made parameter names unique across all functions, they can be
			// added to the global symbol table without conflicts.
			for _, param := range node.Parameters {
				// Create a variable symbol for each parameter
				err := st.DeclareVariable(param.Name, param.Type)
				if err != nil {
					panic(err.Error())
				}
				// Parameters are considered assigned when declared
				st.AssignVariable(param.Name)
			}

			// Only traverse the function body, not the children (which would include parameters)
			if node.Body != nil {
				collectDeclarations(node.Body)
			}
			return // Don't traverse children normally for functions
		}

		// Traverse children
		for _, child := range node.Children {
			if child != nil {
				collectDeclarations(child)
			}
		}
	}

	// Pass 2: populate Symbol references for all NodeIdent nodes
	var populateReferences func(*ASTNode)
	populateReferences = func(node *ASTNode) {
		if node.Kind == NodeIdent {
			varName := node.String
			symbol := st.LookupVariable(varName)
			if symbol != nil {
				node.Symbol = symbol
			}
		}

		// Special handling for function nodes - declare parameters and traverse the body with scoping
		if node.Kind == NodeFunc {
			// Store the original symbol table variables count to restore later
			originalVarCount := len(st.variables)

			// Declare function parameters in symbol table
			for _, param := range node.Parameters {
				err := st.DeclareVariable(param.Name, param.Type)
				if err != nil {
					// If parameter conflicts with global variable, that's okay for now
					// We'll handle proper scoping in a future improvement
				} else {
					// Mark parameter as assigned (since it gets its value from the call)
					st.AssignVariable(param.Name)
				}
			}

			// Populate references in function body with parameters in scope
			if node.Body != nil {
				populateReferences(node.Body)
			}

			// Remove function parameters from symbol table to avoid conflicts with other functions
			st.variables = st.variables[:originalVarCount]
		} else {
			// Traverse children
			for _, child := range node.Children {
				if child != nil {
					populateReferences(child)
				}
			}
		}
	}

	// Execute both passes
	collectDeclarations(ast)
	populateReferences(ast)

	return st
}

// NewTypeChecker creates a new type checker with the given symbol table
func NewTypeChecker(symbolTable *SymbolTable) *TypeChecker {
	return &TypeChecker{
		errors:      make([]string, 0),
		symbolTable: symbolTable,
	}
}

// CheckProgram performs type checking on the entire AST
func CheckProgram(ast *ASTNode, symbolTable *SymbolTable) error {

	tc := NewTypeChecker(symbolTable)

	err := CheckStatement(ast, tc)
	if err != nil {
		return err
	}

	// Return any accumulated errors
	if len(tc.errors) > 0 {
		return fmt.Errorf("type checking failed: %s", tc.errors[0])
	}

	return nil
}

// CheckStatement validates a statement node
func CheckStatement(stmt *ASTNode, tc *TypeChecker) error {

	switch stmt.Kind {
	case NodeVar:
		// Variable declaration - validate type is provided
		varType := stmt.TypeAST
		if varType == nil {
			return fmt.Errorf("error: variable declaration missing type")
		}
		// Note: We allow unsupported types but only type-check supported ones
		// Unsupported types are simply ignored during WASM generation

	case NodeBlock:
		// Check all statements in the block
		for _, child := range stmt.Children {
			err := CheckStatement(child, tc)
			if err != nil {
				return err
			}
		}

	case NodeBinary:
		// Check if this is an assignment statement
		if stmt.Op == "=" {
			return CheckAssignment(stmt.Children[0], stmt.Children[1], tc)
		} else {
			// Regular expression statement
			err := CheckExpression(stmt, tc)
			if err != nil {
				return err
			}
		}

	case NodeCall, NodeIdent, NodeInteger, NodeDot:
		// Expression statement
		err := CheckExpression(stmt, tc)
		if err != nil {
			return err
		}

	case NodeReturn:
		// TODO: Implement return type checking in the future
		if len(stmt.Children) > 0 {
			err := CheckExpression(stmt.Children[0], tc)
			if err != nil {
				return err
			}
		}

	case NodeFunc:
		// Function declaration - check the function body
		if stmt.Body != nil {
			err := CheckStatement(stmt.Body, tc)
			if err != nil {
				return err
			}
		}

	case NodeIf:
		// If statement type checking
		// Structure: [condition, then_block, condition2?, else_block2?, ...]

		// Check condition (must be Boolean)
		err := CheckExpression(stmt.Children[0], tc)
		if err != nil {
			return err
		}
		condType := stmt.Children[0].TypeAST
		if !TypesEqual(condType, TypeBool) {
			return fmt.Errorf("error: if condition must be Boolean, got %s", TypeToString(condType))
		}

		// Check then block
		err = CheckStatement(stmt.Children[1], tc)
		if err != nil {
			return err
		}

		// Check else/else-if clauses
		i := 2
		for i < len(stmt.Children) {
			// Check condition (if not nil)
			if stmt.Children[i] != nil {
				// else-if condition
				err := CheckExpression(stmt.Children[i], tc)
				if err != nil {
					return err
				}
				condType := stmt.Children[i].TypeAST
				if !TypesEqual(condType, TypeBool) {
					return fmt.Errorf("error: else-if condition must be Boolean, got %s", TypeToString(condType))
				}
			}

			// Check block
			err = CheckStatement(stmt.Children[i+1], tc)
			if err != nil {
				return err
			}

			i += 2
		}

	default:
		// Other statement types are valid for now
	}

	return nil
}

// CheckExpression validates an expression and stores type in expr.TypeAST

func CheckExpression(expr *ASTNode, tc *TypeChecker) error {

	switch expr.Kind {
	case NodeInteger:
		expr.TypeAST = TypeI64
		return nil

	case NodeIdent:
		// Variable reference - use cached symbol reference
		if expr.Symbol == nil {
			return fmt.Errorf("error: variable '%s' used before declaration", expr.String)
		}
		if !expr.Symbol.Assigned {
			return fmt.Errorf("error: variable '%s' used before assignment", expr.String)
		}
		expr.TypeAST = expr.Symbol.Type
		return nil

	case NodeBinary:
		if expr.Op == "=" {
			// Assignment expression
			err := CheckAssignmentExpression(expr.Children[0], expr.Children[1], tc)
			if err != nil {
				return err
			}
			// Assignment expression type is stored in the assignment expression itself
			return nil
		} else {
			// Binary operation
			err := CheckExpression(expr.Children[0], tc)
			if err != nil {
				return err
			}
			err = CheckExpression(expr.Children[1], tc)
			if err != nil {
				return err
			}

			// Get types from the type-checked children
			leftType := expr.Children[0].TypeAST
			rightType := expr.Children[1].TypeAST

			// Ensure operand types match
			if !TypesEqual(leftType, rightType) {
				return fmt.Errorf("error: type mismatch in binary operation")
			}

			// Set result type based on operator
			switch expr.Op {
			case "==", "!=", "<", ">", "<=", ">=":
				expr.TypeAST = TypeBool // Comparison operators return Boolean
			case "+", "-", "*", "/", "%":
				expr.TypeAST = leftType // Arithmetic operators return operand type
			default:
				return fmt.Errorf("error: unsupported binary operator '%s'", expr.Op)
			}
			return nil
		}

	case NodeCall:
		// Function call validation
		if len(expr.Children) == 0 || expr.Children[0].Kind != NodeIdent {
			return fmt.Errorf("error: invalid function call")
		}
		funcName := expr.Children[0].String

		if funcName == "print" {
			// Built-in print function
			if len(expr.Children) != 2 {
				return fmt.Errorf("error: print() function expects 1 argument")
			}
			err := CheckExpression(expr.Children[1], tc)
			if err != nil {
				return err
			}
			expr.TypeAST = TypeI64 // print returns nothing, but use I64 for now
			return nil
		} else {
			// User-defined function
			if tc.symbolTable == nil {
				return fmt.Errorf("error: no symbol table for function validation")
			}

			// Look up function in symbol table
			function := tc.symbolTable.LookupFunction(funcName)
			if function == nil {
				return fmt.Errorf("error: unknown function '%s'", funcName)
			}

			// Validate and match parameters
			err := validateFunctionCall(expr, function, tc)
			if err != nil {
				return err
			}

			// Return function's return type (or void)
			if function.ReturnType != nil {
				expr.TypeAST = function.ReturnType
			} else {
				expr.TypeAST = TypeI64 // Void functions return I64 for now
			}
			return nil
		}

	case NodeDot:
		// Field access: struct.field
		if len(expr.Children) != 1 {
			return fmt.Errorf("error: field access expects 1 base expression")
		}

		err := CheckExpression(expr.Children[0], tc)
		if err != nil {
			return err
		}

		// Get base type from the type-checked child
		baseType := expr.Children[0].TypeAST

		// Handle both direct struct and pointer-to-struct access
		var structType *TypeNode
		if baseType.Kind == TypeStruct {
			// Direct struct access
			structType = baseType
		} else if baseType.Kind == TypePointer && baseType.Child.Kind == TypeStruct {
			// Pointer-to-struct access (struct parameters)
			structType = baseType.Child
		} else {
			return fmt.Errorf("error: cannot access field of non-struct type %s", TypeToString(baseType))
		}

		// Find the field in the struct
		fieldName := expr.FieldName
		for _, field := range structType.Fields {
			if field.Name == fieldName {
				expr.TypeAST = field.Type
				return nil
			}
		}

		return fmt.Errorf("error: struct %s has no field named '%s'", structType.String, fieldName)

	case NodeUnary:
		// Unary operations
		if expr.Op == "&" {
			// Address-of operator
			if len(expr.Children) != 1 {
				return fmt.Errorf("error: address-of operator expects 1 operand")
			}

			err := CheckExpression(expr.Children[0], tc)
			if err != nil {
				return err
			}

			// Get operand type from the type-checked child
			operandType := expr.Children[0].TypeAST

			// Return pointer type
			expr.TypeAST = &TypeNode{Kind: TypePointer, Child: operandType}
			return nil
		} else if expr.Op == "*" {
			// Dereference operator
			if len(expr.Children) != 1 {
				return fmt.Errorf("error: dereference operator expects 1 operand")
			}

			err := CheckExpression(expr.Children[0], tc)
			if err != nil {
				return err
			}

			// Get operand type from the type-checked child
			operandType := expr.Children[0].TypeAST

			// Operand must be a pointer type
			if operandType.Kind != TypePointer {
				return fmt.Errorf("error: cannot dereference non-pointer type %s", TypeToString(operandType))
			}

			// Return the pointed-to type
			expr.TypeAST = operandType.Child
			return nil
		} else {
			return fmt.Errorf("error: unsupported unary operator '%s'", expr.Op)
		}

	default:
		return fmt.Errorf("error: unsupported expression type '%s'", expr.Kind)
	}
}

// validateFunctionCall validates a function call and reorders parameters if necessary
func validateFunctionCall(callExpr *ASTNode, function *FunctionInfo, tc *TypeChecker) error {
	args := callExpr.Children[1:] // Skip function name
	paramNames := callExpr.ParameterNames

	// Check parameter count
	if len(args) != len(function.Parameters) {
		return fmt.Errorf("error: function '%s' expects %d arguments, got %d",
			function.Name, len(function.Parameters), len(args))
	}

	// Separate positional and named parameters
	var positionalArgs []*ASTNode
	var namedArgs []*ASTNode
	var namedArgNames []string

	for i, arg := range args {
		if i < len(paramNames) && paramNames[i] != "" {
			// Named parameter
			namedArgs = append(namedArgs, arg)
			namedArgNames = append(namedArgNames, paramNames[i])
		} else {
			// Positional parameter
			positionalArgs = append(positionalArgs, arg)
		}
	}

	// Validate that positional parameters come before named parameters
	if len(positionalArgs) > 0 && len(namedArgs) > 0 {
		// Find the first named parameter position
		firstNamedPos := -1
		for i, name := range paramNames {
			if name != "" {
				firstNamedPos = i
				break
			}
		}
		// Check that all positional args come before first named arg
		if firstNamedPos >= 0 && len(positionalArgs) > firstNamedPos {
			return fmt.Errorf("error: positional arguments must come before named arguments")
		}
	}

	// Validate positional parameters match function signature
	for i, arg := range positionalArgs {
		if i >= len(function.Parameters) {
			return fmt.Errorf("error: too many positional arguments")
		}
		if function.Parameters[i].IsNamed {
			return fmt.Errorf("error: cannot pass positional argument to named parameter '%s'",
				function.Parameters[i].Name)
		}

		// Type check the argument
		err := CheckExpression(arg, tc)
		if err != nil {
			return err
		}
	}

	// Validate named parameters
	usedParams := make(map[string]bool)
	for i, argName := range namedArgNames {
		// Check for duplicate parameter names
		if usedParams[argName] {
			return fmt.Errorf("error: duplicate parameter name '%s'", argName)
		}
		usedParams[argName] = true

		// Find the parameter in function signature
		paramIndex := -1
		for j, param := range function.Parameters {
			if param.Name == argName {
				paramIndex = j
				break
			}
		}

		if paramIndex == -1 {
			return fmt.Errorf("error: unknown parameter name '%s' for function '%s'",
				argName, function.Name)
		}

		if !function.Parameters[paramIndex].IsNamed {
			return fmt.Errorf("error: parameter '%s' is positional, not named", argName)
		}

		// Type check the argument
		err := CheckExpression(namedArgs[i], tc)
		if err != nil {
			return err
		}
	}

	// Check that all required parameters are provided
	providedPositional := len(positionalArgs)
	for i, param := range function.Parameters {
		if i < providedPositional {
			continue // Already handled by positional args
		}

		// Check if this named parameter was provided
		found := false
		for _, argName := range namedArgNames {
			if argName == param.Name {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("error: missing required parameter '%s' for function '%s'",
				param.Name, function.Name)
		}
	}

	// Reorder arguments to match function signature
	reorderFunctionCallArguments(callExpr, function)

	return nil
}

// reorderFunctionCallArguments reorders the arguments in a function call to match the function signature
func reorderFunctionCallArguments(callExpr *ASTNode, function *FunctionInfo) {
	if len(callExpr.ParameterNames) == 0 {
		// No named parameters, no reordering needed
		return
	}

	args := callExpr.Children[1:] // Skip function name
	paramNames := callExpr.ParameterNames
	newArgs := make([]*ASTNode, len(function.Parameters))

	// First, place positional arguments
	positionalCount := 0
	for i, name := range paramNames {
		if name == "" {
			newArgs[i] = args[i]
			positionalCount++
		}
	}

	// Then, place named arguments in correct positions
	for i, name := range paramNames {
		if name != "" {
			// Find the parameter index in function signature
			for j, param := range function.Parameters {
				if param.Name == name {
					newArgs[j] = args[i]
					break
				}
			}
		}
	}

	// Update the call expression with reordered arguments
	callExpr.Children = append([]*ASTNode{callExpr.Children[0]}, newArgs...)
	// Clear parameter names since arguments are now in order
	callExpr.ParameterNames = make([]string, len(newArgs))
}

// CheckAssignment validates an assignment statement
func CheckAssignment(lhs, rhs *ASTNode, tc *TypeChecker) error {
	// Validate RHS type first
	err := CheckExpression(rhs, tc)
	if err != nil {
		return err
	}
	rhsType := rhs.TypeAST

	// Validate LHS is assignable
	var lhsType *TypeNode

	if lhs.Kind == NodeIdent {
		// Direct variable assignment - use cached symbol reference
		if lhs.Symbol == nil {
			return fmt.Errorf("error: variable '%s' used before declaration", lhs.String)
		}
		lhsType = lhs.Symbol.Type

		// Mark variable as assigned
		lhs.Symbol.Assigned = true

	} else if lhs.Kind == NodeUnary && lhs.Op == "*" {
		// Pointer dereference assignment (e.g., ptr* = value)
		err := CheckExpression(lhs.Children[0], tc)
		if err != nil {
			return err
		}
		ptrType := lhs.Children[0].TypeAST

		if ptrType.Kind != TypePointer {
			return fmt.Errorf("error: cannot dereference non-pointer type %s", TypeToString(ptrType))
		}

		lhsType = ptrType.Child

	} else if lhs.Kind == NodeDot {
		// Field assignment (e.g., s.field = value)
		err = CheckExpression(lhs, tc)
		if err != nil {
			return err
		}
		lhsType = lhs.TypeAST

	} else {
		return fmt.Errorf("error: left side of assignment must be a variable, field access, or dereferenced pointer")
	}

	// Ensure types match
	if !TypesEqual(lhsType, rhsType) {
		return fmt.Errorf("error: cannot assign %s to %s",
			TypeToString(rhsType), TypeToString(lhsType))
	}

	return nil
}

// CheckAssignmentExpression validates an assignment expression
func CheckAssignmentExpression(lhs, rhs *ASTNode, tc *TypeChecker) error {
	err := CheckAssignment(lhs, rhs, tc)
	if err != nil {
		return err
	}

	// Assignment expression returns the type of the assigned value
	// The assignment expression type should be set to the RHS type
	if rhs.TypeAST != nil {
		lhs.TypeAST = rhs.TypeAST
	}
	return nil
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
	if node == nil {
		return "_"
	}
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
		typeStr := "(ident \"" + TypeToString(node.TypeAST) + "\")"
		return "(var " + name + " " + typeStr + ")"
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
	case NodeStruct:
		result := "(struct \"" + node.String + "\""
		for _, child := range node.Children {
			result += " " + ToSExpr(child)
		}
		result += ")"
		return result
	case NodeDot:
		base := ToSExpr(node.Children[0])
		return "(dot " + base + " \"" + node.FieldName + "\")"
	case NodeFunc:
		result := "(func \"" + node.FunctionName + "\" ("
		for i, param := range node.Parameters {
			if i > 0 {
				result += " "
			}
			paramType := "positional"
			if param.IsNamed {
				paramType = "named"
			}
			result += "(param \"" + param.Name + "\" \"" + TypeToString(param.Type) + "\" " + paramType + ")"
		}
		result += ")"
		if node.ReturnType != nil {
			result += " \"" + TypeToString(node.ReturnType) + "\""
		} else {
			result += " void"
		}
		result += " " + ToSExpr(node.Body) + ")"
		return result
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
		// Handle special case of minimum int64 to avoid overflow
		if n == -9223372036854775808 {
			return "-9223372036854775808"
		}
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
	case DOT: // field access operator
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
		} else if CurrTokenType == DOT {
			// Handle field access operator: expr.field
			SkipToken(DOT)
			if CurrTokenType != IDENT {
				break // error - expecting field name
			}
			fieldName := CurrLiteral
			SkipToken(IDENT)
			left = &ASTNode{
				Kind:      NodeDot,
				FieldName: fieldName,
				Children:  []*ASTNode{left},
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

// parseTypeExpression parses a type expression and returns a TypeNode
func parseTypeExpression() *TypeNode {
	if CurrTokenType != IDENT {
		return nil
	}

	// Parse base type
	baseTypeName := CurrLiteral
	SkipToken(IDENT)

	baseType := getBuiltinType(baseTypeName)
	if baseType == nil {
		if isKnownUnsupportedType(baseTypeName) {
			// Known unsupported built-in types like "string", "int", etc.
			baseType = &TypeNode{Kind: TypeBuiltin, String: baseTypeName}
		} else {
			// Unknown types assumed to be struct types
			baseType = &TypeNode{Kind: TypeStruct, String: baseTypeName}
		}
	}

	// Handle pointer suffixes
	resultType := baseType
	for CurrTokenType == ASTERISK {
		SkipToken(ASTERISK)
		resultType = &TypeNode{
			Kind:  TypePointer,
			Child: resultType,
		}
	}

	return resultType
}

// parseBlockStatements parses a block of statements between braces and returns a Block AST node
func parseBlockStatements() *ASTNode {
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
		Kind:     NodeBlock,
		Children: statements,
	}
}

// ParseStatement parses a statement and returns an AST node
func ParseStatement() *ASTNode {
	switch CurrTokenType {
	case STRUCT:
		SkipToken(STRUCT)
		if CurrTokenType != IDENT {
			return &ASTNode{} // error
		}
		structName := CurrLiteral
		SkipToken(IDENT)
		if CurrTokenType != LBRACE {
			return &ASTNode{} // error
		}
		SkipToken(LBRACE)

		var fields []*ASTNode
		for CurrTokenType != RBRACE && CurrTokenType != EOF {
			// Parse field declaration: var fieldName Type;
			if CurrTokenType != VAR {
				break // error
			}
			fieldDecl := ParseStatement() // Parse the var declaration
			fields = append(fields, fieldDecl)
		}

		if CurrTokenType == RBRACE {
			SkipToken(RBRACE)
		}

		return &ASTNode{
			Kind:     NodeStruct,
			String:   structName,
			Children: fields,
		}

	case IF:
		SkipToken(IF)
		children := []*ASTNode{}
		children = append(children, ParseExpression()) // if condition
		if CurrTokenType != LBRACE {
			return &ASTNode{} // error
		}

		children = append(children, parseBlockStatements()) // then block

		for CurrTokenType == ELSE {
			SkipToken(ELSE)
			if CurrTokenType == IF {
				SkipToken(IF)
				// else-if block
				children = append(children, ParseExpression())      // else condition
				children = append(children, parseBlockStatements()) // else block
			} else if CurrTokenType == LBRACE {
				// else block
				children = append(children, nil)                    // else condition (nil for final else)
				children = append(children, parseBlockStatements()) // else block
				break                                               // final else, no more chaining
			} else {
				return &ASTNode{} // error: expected { after else
			}
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

		// Parse type using new TypeNode system
		typeAST := parseTypeExpression()
		if typeAST == nil {
			return &ASTNode{} // error - invalid type
		}

		if CurrTokenType == SEMICOLON {
			SkipToken(SEMICOLON)
		}
		return &ASTNode{
			Kind:     NodeVar,
			Children: []*ASTNode{varName},
			TypeAST:  typeAST,
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

	case FUNC:
		return parseFunctionDeclaration()

	default:
		// Expression statement
		expr := ParseExpression()
		if CurrTokenType == SEMICOLON {
			SkipToken(SEMICOLON)
		}
		return expr
	}
}

// parseFunctionDeclaration parses a function declaration
// Syntax: func name(param1: Type, param2: Type): ReturnType { body }
// Or:     func name(param1: Type, param2: Type) { body } // void return
func parseFunctionDeclaration() *ASTNode {
	SkipToken(FUNC) // consume 'func'

	// Parse function name
	if CurrTokenType != IDENT {
		panic("Expected function name")
	}
	functionName := CurrLiteral
	SkipToken(IDENT)

	// Parse parameter list
	if CurrTokenType != LPAREN {
		panic("Expected '(' after function name")
	}
	SkipToken(LPAREN)

	var parameters []FunctionParameter
	for CurrTokenType != RPAREN && CurrTokenType != EOF {
		// Parse parameter: _ name: Type (positional) or name: Type (named)
		isPositional := false
		var paramName string

		if CurrTokenType == IDENT && CurrLiteral == "_" {
			// Positional parameter: _ name: Type
			isPositional = true
			SkipToken(IDENT) // skip the "_"
			if CurrTokenType != IDENT {
				panic("Expected parameter name after '_'")
			}
			paramName = CurrLiteral
			SkipToken(IDENT)
		} else if CurrTokenType == IDENT {
			// Named parameter: name: Type
			paramName = CurrLiteral
			SkipToken(IDENT)
		} else {
			panic("Expected parameter name")
		}

		// Parse colon
		if CurrTokenType != COLON {
			panic("Expected ':' after parameter name")
		}
		SkipToken(COLON)

		// Parse parameter type
		paramType := parseTypeExpression()
		if paramType == nil {
			panic("Expected parameter type")
		}

		// Convert struct parameters to pointer types (per Phase 3 spec)
		finalParamType := paramType
		if paramType.Kind == TypeStruct {
			finalParamType = &TypeNode{
				Kind:  TypePointer,
				Child: paramType,
			}
		}

		parameters = append(parameters, FunctionParameter{
			Name:    paramName,
			Type:    finalParamType,
			IsNamed: !isPositional,
		})

		// Check for comma or end of parameters
		if CurrTokenType == COMMA {
			SkipToken(COMMA)
		} else if CurrTokenType != RPAREN {
			panic("Expected ',' or ')' in parameter list")
		}
	}

	if CurrTokenType != RPAREN {
		panic("Expected ')' after parameter list")
	}
	SkipToken(RPAREN)

	// Parse optional return type
	var returnType *TypeNode
	if CurrTokenType == COLON {
		SkipToken(COLON)
		returnType = parseTypeExpression()
		if returnType == nil {
			panic("Expected return type after ':'")
		}
	}

	// Parse function body
	if CurrTokenType != LBRACE {
		panic("Expected '{' for function body")
	}
	body := ParseStatement() // This will parse the block statement

	return &ASTNode{
		Kind:         NodeFunc,
		FunctionName: functionName,
		Parameters:   parameters,
		ReturnType:   returnType,
		Body:         body,
	}
}

// ParseProgram parses a complete program (multiple functions and statements)
func ParseProgram() *ASTNode {
	var statements []*ASTNode

	for CurrTokenType != EOF {
		stmt := ParseStatement()
		statements = append(statements, stmt)
	}

	return &ASTNode{
		Kind:     NodeBlock,
		Children: statements,
	}
}
