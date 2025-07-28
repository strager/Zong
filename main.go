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

	// Loop context
	ControlDepth int   // Track nesting depth of control structures (if, etc.) for branch calculation
	LoopStack    []int // Stack of control depths at each loop entry (for break/continue targeting)
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
	I32_SUB          = 0x6B
	I32_MUL          = 0x6C
	I32_DIV_S        = 0x6D
	I32_REM_S        = 0x6F
	I32_LOAD         = 0x28
	I32_LOAD8_U      = 0x2D
	I32_STORE        = 0x36
	I32_STORE8       = 0x3A
	I32_WRAP_I64     = 0xA7
	I64_CONST        = 0x42
	I64_ADD          = 0x7C
	I64_SUB          = 0x7D
	I64_MUL          = 0x7E
	I64_DIV_S        = 0x7F
	I64_REM_S        = 0x81
	I32_EQ           = 0x46
	I32_NE           = 0x47
	I32_LT_S         = 0x48
	I32_LE_S         = 0x4C
	I32_GE_S         = 0x4E
	I64_EQ           = 0x51
	I64_NE           = 0x52
	I64_LT_S         = 0x53
	I64_GT_S         = 0x55
	I64_LE_S         = 0x57
	I64_GE_S         = 0x59
	I64_EXTEND_I32_S = 0xAC
	I64_EXTEND_I32_U = 0xAD
	I64_LOAD         = 0x29
	I64_STORE        = 0x37
	GLOBAL_GET       = 0x23
	GLOBAL_SET       = 0x24
	LOCAL_GET        = 0x20
	LOCAL_SET        = 0x21
	LOCAL_TEE        = 0x22
	CALL             = 0x10
	END              = 0x0B
	WASM_BLOCK       = 0x02
	WASM_LOOP        = 0x03
	WASM_BR          = 0x0C
	WASM_BR_IF       = 0x0D
	I32_GT_S         = 0x4A
	WASM_IF          = 0x04
	WASM_ELSE        = 0x05
	DROP             = 0x1A
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
	writeLEB128(&sectionBuf, 2) // 2 imports: print and print_bytes functions

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

	// Import 2: print_bytes function
	// Module name "env"
	writeLEB128(&sectionBuf, 3) // length of "env"
	writeBytes(&sectionBuf, []byte("env"))

	// Import name "print_bytes"
	writeLEB128(&sectionBuf, 11) // length of "print_bytes"
	writeBytes(&sectionBuf, []byte("print_bytes"))

	// Import kind: function (0x00)
	writeByte(&sectionBuf, 0x00)

	// Type index (1)
	writeLEB128(&sectionBuf, 1)

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

func EmitGlobalSection(buf *bytes.Buffer, dataSize uint32) {
	writeByte(buf, 0x06) // global section id

	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, 1) // 1 global: tstack

	// Global type: i32 mutable (0x7F 0x01)
	writeByte(&sectionBuf, 0x7F) // i32
	writeByte(&sectionBuf, 0x01) // mutable

	// Initializer expression: i32.const dataSize + end
	writeByte(&sectionBuf, I32_CONST)
	writeLEB128Signed(&sectionBuf, int64(dataSize))
	writeByte(&sectionBuf, END)

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

// Global type registry for consistent type indices
var globalTypeRegistry []FunctionType
var globalTypeMap map[string]int

// Global function registry for consistent function indices
var globalFunctionRegistry []string
var globalFunctionMap map[string]int

// Global slice type registry for generating append functions
var globalSliceTypes map[string]*TypeNode
var generatedAppendFunctions []*ASTNode

func initTypeRegistry() {
	globalTypeRegistry = []FunctionType{}
	globalTypeMap = make(map[string]int)
	globalSliceTypes = make(map[string]*TypeNode)
	generatedAppendFunctions = []*ASTNode{}

	// Type 0: print function (i64) -> ()
	printType := FunctionType{
		Parameters: []byte{0x7E}, // i64
		Results:    []byte{},     // void
	}
	globalTypeRegistry = append(globalTypeRegistry, printType)
	globalTypeMap["(i64)->()"] = 0

	// Type 1: print_bytes function (i32) -> ()
	printBytesType := FunctionType{
		Parameters: []byte{0x7F}, // i32 (slice pointer)
		Results:    []byte{},     // void
	}
	globalTypeRegistry = append(globalTypeRegistry, printBytesType)
	globalTypeMap["(i32)->()"] = 1
}

func initFunctionRegistry(functions []*ASTNode) {
	globalFunctionRegistry = []string{}
	globalFunctionMap = make(map[string]int)

	// Function 0 is print (imported)
	globalFunctionRegistry = append(globalFunctionRegistry, "print")
	globalFunctionMap["print"] = 0

	// Function 1 is print_bytes (imported)
	globalFunctionRegistry = append(globalFunctionRegistry, "print_bytes")
	globalFunctionMap["print_bytes"] = 1

	// Add user functions starting from index 2
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
	if typeNode.Kind == TypeInteger {
		panic("TypeInteger should be resolved before WASM generation")
	}
	if typeNode.Kind == TypeBuiltin && typeNode.String == "I64" {
		return 0x7E // i64
	}
	if typeNode.Kind == TypeBuiltin && typeNode.String == "U8" {
		return 0x7F // i32 (U8 maps to i32 in WASM)
	}
	if typeNode.Kind == TypeBuiltin && typeNode.String == "Boolean" {
		return 0x7E // i64 (Boolean maps to i64 in WASM)
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
	if typeNode.Kind == TypeBuiltin && typeNode.String == "U8" {
		return "i32" // U8 maps to i32 in WASM
	}
	if typeNode.Kind == TypeBuiltin && typeNode.String == "Boolean" {
		return "i64" // Boolean maps to i64 in WASM
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
		writeLEB128(&sectionBuf, 2) // type index 2 (void -> void) - after print (0) and print_bytes (1)
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
	writeLEB128(&sectionBuf, 2) // 2 exports: main function and memory

	// Export 1: main function
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
			mainIndex = 2 // print=0, print_bytes=1, main=2
		}
	} else {
		// Legacy fallback when no function registry
		mainIndex = 2 // print=0, print_bytes=1, main=2
	}
	writeLEB128(&sectionBuf, uint32(mainIndex))

	// Export 2: memory
	writeLEB128(&sectionBuf, 6) // length of "memory"
	writeBytes(&sectionBuf, []byte("memory"))

	// Export kind: memory (0x02)
	writeByte(&sectionBuf, 0x02)

	// Memory index 0 (first memory)
	writeLEB128(&sectionBuf, 0)

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

func EmitCodeSection(buf *bytes.Buffer, functions []*ASTNode) {
	writeByte(buf, 0x0A) // code section id

	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, uint32(len(functions))) // number of function bodies

	// Emit each function body
	for _, fn := range functions {
		emitSingleFunction(&sectionBuf, fn)
	}

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

// emitSingleFunction emits the code for a single function
func emitSingleFunction(buf *bytes.Buffer, fn *ASTNode) {
	// Check if this is a generated append function
	if isGeneratedAppendFunction(fn.FunctionName) {
		emitAppendFunctionBody(buf, fn)
		return
	}

	// Use unified local management
	localCtx := BuildLocalContext(fn.Body, fn)

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

// isGeneratedAppendFunction checks if a function name is a generated append function
func isGeneratedAppendFunction(functionName string) bool {
	return len(functionName) > 7 && functionName[:7] == "append_"
}

// emitAppendFunctionBody generates the WASM code for a generated append function
func emitAppendFunctionBody(buf *bytes.Buffer, fn *ASTNode) {
	// Determine element type from the function parameters
	slicePtrParam := fn.Parameters[0] // slice_ptr: T[]*

	sliceType := slicePtrParam.Type.Child // T[] (or struct after transformation)

	// Get element type - handle both original slice and transformed struct
	var elementType *TypeNode
	if sliceType.Kind == TypeSlice {
		// Before transformation
		elementType = sliceType.Child
	} else if sliceType.Kind == TypeStruct {
		// After transformation - get element type from items field
		for _, field := range sliceType.Fields {
			if field.Name == "items" {
				// items field is TypePointer with Child as element type
				elementType = field.Type.Child
				break
			}
		}
	}

	elementSize := uint32(GetTypeSize(elementType))

	var bodyBuf bytes.Buffer

	// Declare locals for this function
	// Parameters are:
	// - param 0: slice_ptr (i32)
	// - param 1: value (i64 or i32)
	// Locals are:
	// - local 2: i32 (new items pointer) - needed for multiple uses
	// - local 3: i32 (copy size) - needed for multiple uses
	// - local 4: i64 (current length) - needed for length update
	writeLEB128(&bodyBuf, 2) // 2 local entries

	// local 2: i32 (new_items)
	// local 3: i32 (copy_size)
	writeLEB128(&bodyBuf, 2)  // count
	writeByte(&bodyBuf, 0x7F) // i32

	// local 4: i64 (current_length)
	writeLEB128(&bodyBuf, 1)  // count
	writeByte(&bodyBuf, 0x7E) // i64

	// IMPROVED APPEND IMPLEMENTATION that copies existing elements

	// 1. Get current length and store for later use
	writeByte(&bodyBuf, LOCAL_GET)
	writeLEB128(&bodyBuf, 0) // slice_ptr parameter
	writeByte(&bodyBuf, I64_LOAD)
	writeByte(&bodyBuf, 0x03) // alignment
	writeLEB128(&bodyBuf, 8)  // offset to length field
	writeByte(&bodyBuf, LOCAL_TEE)
	writeLEB128(&bodyBuf, 4) // store current_length and keep on stack

	// 2. Calculate copy size: current_length * element_size
	writeByte(&bodyBuf, I32_WRAP_I64)
	writeByte(&bodyBuf, I32_CONST)
	writeLEB128(&bodyBuf, elementSize)
	writeByte(&bodyBuf, I32_MUL)
	writeByte(&bodyBuf, LOCAL_TEE)
	writeLEB128(&bodyBuf, 3) // store copy_size and keep on stack

	// 3. Calculate total size needed: copy_size + element_size
	writeByte(&bodyBuf, I32_CONST)
	writeLEB128(&bodyBuf, elementSize)
	writeByte(&bodyBuf, I32_ADD)
	// Stack: [total_size]

	// 4. Allocate new space on tstack and store new_items
	writeByte(&bodyBuf, GLOBAL_GET)
	writeLEB128(&bodyBuf, 0) // tstack
	writeByte(&bodyBuf, LOCAL_TEE)
	writeLEB128(&bodyBuf, 2) // store new_items and keep on stack

	// Update tstack: tstack + total_size
	writeByte(&bodyBuf, I32_ADD) // add total_size (from step 3)
	writeByte(&bodyBuf, GLOBAL_SET)
	writeLEB128(&bodyBuf, 0) // update tstack

	// 5. Copy existing elements using memory.copy
	// memory.copy(dest, src, size)
	writeByte(&bodyBuf, LOCAL_GET)
	writeLEB128(&bodyBuf, 2) // dest: new_items
	// Get old items pointer directly
	writeByte(&bodyBuf, LOCAL_GET)
	writeLEB128(&bodyBuf, 0) // slice_ptr parameter
	writeByte(&bodyBuf, I32_LOAD)
	writeByte(&bodyBuf, 0x02) // alignment
	writeLEB128(&bodyBuf, 0)  // offset to items field (src: old_items)
	writeByte(&bodyBuf, LOCAL_GET)
	writeLEB128(&bodyBuf, 3) // copy_size
	// Stack: [dest, src, size]

	// Emit memory.copy instruction
	writeByte(&bodyBuf, 0xFC) // Multi-byte instruction prefix
	writeLEB128(&bodyBuf, 10) // memory.copy opcode
	writeByte(&bodyBuf, 0x00) // dst memory index (0)
	writeByte(&bodyBuf, 0x00) // src memory index (0)

	// 6. Store new element at the end
	// addr = new_items + copy_size
	writeByte(&bodyBuf, LOCAL_GET)
	writeLEB128(&bodyBuf, 2) // new_items
	writeByte(&bodyBuf, LOCAL_GET)
	writeLEB128(&bodyBuf, 3) // copy_size
	writeByte(&bodyBuf, I32_ADD)
	// Stack: [new_element_addr]

	writeByte(&bodyBuf, LOCAL_GET)
	writeLEB128(&bodyBuf, 1) // value parameter
	// Stack: [new_element_addr, value]

	emitValueStoreToMemory(&bodyBuf, elementType)

	// 7. Update slice.items pointer
	writeByte(&bodyBuf, LOCAL_GET)
	writeLEB128(&bodyBuf, 0) // slice_ptr parameter
	writeByte(&bodyBuf, LOCAL_GET)
	writeLEB128(&bodyBuf, 2) // new_items
	writeByte(&bodyBuf, I32_STORE)
	writeByte(&bodyBuf, 0x02) // alignment
	writeLEB128(&bodyBuf, 0)  // offset to items field

	// 8. Update slice.length
	writeByte(&bodyBuf, LOCAL_GET)
	writeLEB128(&bodyBuf, 0) // slice_ptr parameter
	writeByte(&bodyBuf, LOCAL_GET)
	writeLEB128(&bodyBuf, 4) // current_length (stored earlier)
	writeByte(&bodyBuf, I64_CONST)
	writeLEB128Signed(&bodyBuf, 1)
	writeByte(&bodyBuf, I64_ADD)
	writeByte(&bodyBuf, I64_STORE)
	writeByte(&bodyBuf, 0x03) // alignment
	writeLEB128(&bodyBuf, 8)  // offset to length field

	writeByte(&bodyBuf, END) // end instruction

	// Write function body size and content
	writeLEB128(buf, uint32(bodyBuf.Len())) // function body size
	writeBytes(buf, bodyBuf.Bytes())
}

// EmitDataSection emits the WASM data section with string literals
func EmitDataSection(buf *bytes.Buffer, dataSection *DataSection) {
	if len(dataSection.Strings) == 0 {
		return // No data section needed if no strings
	}

	writeByte(buf, 0x0B) // data section id

	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, uint32(len(dataSection.Strings))) // number of data segments

	// Emit each string as a data segment
	for _, str := range dataSection.Strings {
		// Segment type: 0x00 = active with memory index
		writeLEB128(&sectionBuf, 0x00)

		// Offset expression (i32.const address + end)
		writeByte(&sectionBuf, I32_CONST)
		writeLEB128Signed(&sectionBuf, int64(str.Address))
		writeByte(&sectionBuf, END)

		// Data size and content
		writeLEB128(&sectionBuf, str.Length)
		writeBytes(&sectionBuf, []byte(str.Content))
	}

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

// EmitStatement generates WASM bytecode for statements
func EmitStatement(buf *bytes.Buffer, node *ASTNode, localCtx *LocalContext) {
	switch node.Kind {
	case NodeVar:
		// Variable declarations don't generate runtime code for the declaration itself
		// (locals are declared in function header)
		// However, if there's an initialization expression, generate assignment code
		if len(node.Children) > 1 {
			// Has initialization: var x I64 = value;
			// Generate equivalent assignment: x = value;
			varName := node.Children[0]
			initExpr := node.Children[1]

			// Ensure varName has proper Symbol and TypeAST information
			if varName.Symbol == nil {
				panic("Variable symbol not found during initialization: " + varName.String)
			}

			// Create a synthetic assignment node
			assignmentNode := &ASTNode{
				Kind:     NodeBinary,
				Op:       "=",
				Children: []*ASTNode{varName, initExpr},
			}

			// Emit the assignment
			EmitExpression(buf, assignmentNode, localCtx)
		}
		break

	case NodeStruct:
		// Struct declarations don't generate runtime code
		// (they only define types)
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
		// Convert I64 Bool conditions to I32 for WASM if instruction
		if TypesEqual(node.Children[0].TypeAST, TypeBool) {
			writeByte(buf, I32_WRAP_I64) // Convert I64 to I32
		}

		// Start if block
		writeByte(buf, 0x04) // if opcode
		writeByte(buf, 0x40) // block type: void

		// Increment control depth for entire if statement
		localCtx.ControlDepth++
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
				// Convert I64 Bool conditions to I32 for WASM if instruction
				if TypesEqual(node.Children[i].TypeAST, TypeBool) {
					writeByte(buf, I32_WRAP_I64) // Convert I64 to I32
				}
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

		// Decrement control depth for entire if statement
		localCtx.ControlDepth--

	case NodeBinary:
		// Handle binary operations (mainly assignments)
		EmitExpression(buf, node, localCtx)

	case NodeLoop:
		// Save current control depth for this loop's break targeting
		localCtx.LoopStack = append(localCtx.LoopStack, localCtx.ControlDepth)

		// Emit WASM: block (for break - outer block)
		writeByte(buf, WASM_BLOCK) // block opcode
		writeByte(buf, 0x40)       // void type

		// Emit WASM: loop (for continue - inner loop)
		writeByte(buf, WASM_LOOP) // loop opcode
		writeByte(buf, 0x40)      // void type

		// Emit loop body
		for _, stmt := range node.Children {
			EmitStatement(buf, stmt, localCtx)
		}

		// Emit branch back to loop start (this makes it an infinite loop until break)
		writeByte(buf, WASM_BR) // br opcode
		writeLEB128(buf, 0)     // branch depth 0 (back to loop start)

		// Emit WASM: end (loop)
		writeByte(buf, END) // end opcode

		// Emit WASM: end (block)
		writeByte(buf, END) // end opcode

		// Pop from loop stack
		localCtx.LoopStack = localCtx.LoopStack[:len(localCtx.LoopStack)-1]

	case NodeBreak:
		if len(localCtx.LoopStack) == 0 {
			panic("break statement outside of loop")
		}

		// Calculate break depth: current control depth minus control depth when entering current loop, plus 1 for the loop itself
		currentLoopControlDepth := localCtx.LoopStack[len(localCtx.LoopStack)-1]
		nestedControlDepth := localCtx.ControlDepth - currentLoopControlDepth
		breakDepth := 1 + nestedControlDepth // 1 to exit loop, plus nested controls within this loop

		writeByte(buf, WASM_BR) // br opcode
		writeLEB128(buf, uint32(breakDepth))

	case NodeContinue:
		if len(localCtx.LoopStack) == 0 {
			panic("continue statement outside of loop")
		}

		// Emit WASM: br N (continue to inner loop, accounting for nested control structures)
		writeByte(buf, WASM_BR)                           // br opcode
		writeLEB128(buf, uint32(0+localCtx.ControlDepth)) // branch depth (inner loop + nesting)

	default:
		// For now, treat unknown statements as expressions
		EmitExpression(buf, node, localCtx)
	}
}

// emitFieldAddressRecursive generates code to put the address of any field expression on the stack
// getFinalFieldType returns the type of the final field in a dot expression chain
func getFinalFieldType(node *ASTNode) *TypeNode {
	if node.Kind != NodeDot {
		return node.TypeAST
	}

	baseExpr := node.Children[0]
	fieldName := node.FieldName

	// Get the struct type
	var structType *TypeNode
	baseType := baseExpr.TypeAST
	if baseType == nil {
		panic("Base expression has no type information")
	}

	if baseType.Kind == TypeStruct {
		structType = baseType
	} else if baseType.Kind == TypePointer && baseType.Child.Kind == TypeStruct {
		structType = baseType.Child
	} else {
		panic("Field access on non-struct type: " + TypeToString(baseType))
	}

	// Find the field type
	for _, field := range structType.Fields {
		if field.Name == fieldName {
			return field.Type
		}
	}

	panic("Field not found in struct: " + fieldName)
}

// getFieldOffset returns the byte offset of a field within a struct
func getFieldOffset(baseType *TypeNode, fieldName string) uint32 {
	// Get the struct type
	var structType *TypeNode
	if baseType == nil {
		panic("Base expression has no type information")
	}

	if baseType.Kind == TypeStruct {
		structType = baseType
	} else if baseType.Kind == TypePointer && baseType.Child.Kind == TypeStruct {
		structType = baseType.Child
	} else {
		panic("Field access on non-struct type: " + TypeToString(baseType))
	}

	// Find the field offset
	for _, field := range structType.Fields {
		if field.Name == fieldName {
			return field.Offset
		}
	}

	panic("Field not found in struct: " + fieldName)
}

func EmitExpression(buf *bytes.Buffer, node *ASTNode, localCtx *LocalContext) {
	// Handle assignment specially, then delegate to EmitExpressionR for other cases
	if node.Kind == NodeBinary && node.Op == "=" {
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

			if targetLocal.Storage == VarStorageLocal || targetLocal.Storage == VarStorageParameterLocal {
				// Local variable - emit local.set
				EmitExpressionR(buf, rhs, localCtx) // RHS value
				writeByte(buf, LOCAL_SET)
				writeLEB128(buf, targetLocal.Address)
				return
			}
		} else {
			// Check if LHS is a valid assignment target
			if lhs.Kind != NodeUnary && lhs.Kind != NodeDot && lhs.Kind != NodeIndex {
				panic("Invalid assignment target - must be variable, field access, pointer dereference, or slice index")
			}
		}

		// Non-local storage - get address and store based on type
		EmitExpressionL(buf, lhs, localCtx) // Get address
		EmitExpressionR(buf, rhs, localCtx) // Get value

		// Stack is now: [address_i32, value]
		emitValueStoreToMemory(buf, lhs.TypeAST)
		return
	}

	// For all non-assignment expressions, delegate to EmitExpressionR
	EmitExpressionR(buf, node, localCtx)
}

// Precondition: WASM stack should be: [address_i32, value]
// Postcondition: WASM stack is: []
//
// The value on the stack is either the value itself (for primitives) or a pointer to the struct.
func emitValueStoreToMemory(buf *bytes.Buffer, ty *TypeNode) {
	switch ty.Kind {
	case TypeStruct:
		// Struct assignment (copy) using memory.copy
		structSize := uint32(GetTypeSize(ty))
		writeByte(buf, I32_CONST)
		writeLEB128(buf, structSize)
		// Stack: [dst_addr, src_addr, size]
		writeByte(buf, 0xFC) // Multi-byte instruction prefix
		writeLEB128(buf, 10) // memory.copy opcode
		writeByte(buf, 0x00) // dst memory index (0)
		writeByte(buf, 0x00) // src memory index (0)
	case TypePointer:
		// Store pointer as i32
		writeByte(buf, I32_STORE) // Store i32 to memory
		writeByte(buf, 0x02)      // alignment (4 bytes = 2^2)
		writeByte(buf, 0x00)      // offset
	case TypeBuiltin:
		if ty.String == "U8" {
			// Store U8 as single byte
			writeByte(buf, I32_STORE8) // Store i32 as 8-bit to memory
			writeByte(buf, 0x00)       // alignment (1 byte = 2^0)
			writeByte(buf, 0x00)       // offset
		} else {
			// Store other built-in types as i64
			writeByte(buf, I64_STORE) // Store i64 to memory
			writeByte(buf, 0x03)      // alignment (8 bytes = 2^3)
			writeByte(buf, 0x00)      // offset
		}
	default:
		// Store regular value as i64
		writeByte(buf, I64_STORE) // Store i64 to memory
		writeByte(buf, 0x03)      // alignment (8 bytes = 2^3)
		writeByte(buf, 0x00)      // offset
	}
}

// Precondition: WASM stack should be: [address_i32]
// Postcondition: WASM stack is: [value]
//
// The value on the stack upon return is either the value itself (for primitives) or a pointer to the struct.
func emitValueLoadFromMemory(buf *bytes.Buffer, ty *TypeNode) {
	if ty.Kind == TypeStruct {
		// For struct variables, return the address of the struct (not the value)
	} else {
		// Non-struct stack variable - load from memory
		if ty.Kind == TypePointer {
			// Load pointer as i32
			writeByte(buf, I32_LOAD) // Load i32 from memory
			writeByte(buf, 0x02)     // alignment (4 bytes = 2^2)
			writeByte(buf, 0x00)     // offset
		} else if ty.Kind == TypeBuiltin && ty.String == "U8" {
			// Load U8 as single byte and extend to i32
			writeByte(buf, I32_LOAD8_U) // Load 8-bit unsigned to i32
			writeByte(buf, 0x00)        // alignment (1 byte = 2^0)
			writeByte(buf, 0x00)        // offset
		} else {
			// Load regular value as i64
			writeByte(buf, I64_LOAD) // Load i64 from memory
			writeByte(buf, 0x03)     // alignment (8 bytes = 2^3)
			writeByte(buf, 0x00)     // offset
		}
	}
}

// EmitExpressionL emits code for lvalue expressions (expressions that can be assigned to or addressed)
// These expressions produce an address on the stack where a value can be stored or loaded from
func EmitExpressionL(buf *bytes.Buffer, node *ASTNode, localCtx *LocalContext) {
	switch node.Kind {
	case NodeIdent:
		// Variable reference - emit address
		if node.Symbol == nil {
			panic("Undefined variable: " + node.String)
		}
		targetLocal := localCtx.FindVariable(node.Symbol)
		if targetLocal == nil {
			panic("Variable not found in local context: " + node.String)
		}

		if targetLocal.Storage == VarStorageParameterLocal &&
			targetLocal.Symbol.Type.Kind == TypePointer &&
			targetLocal.Symbol.Type.Child.Kind == TypeStruct {
			// Struct parameter - it's stored as a pointer, so just load the pointer and add offset
			writeByte(buf, LOCAL_GET)
			writeLEB128(buf, targetLocal.Address)
		} else if targetLocal.Storage == VarStorageLocal || targetLocal.Storage == VarStorageParameterLocal {
			// Local variable - can't take address of WASM local
			panic("Cannot take address of local variable: " + node.String)
		} else {
			// Stack variable - emit address
			writeByte(buf, LOCAL_GET)
			writeLEB128(buf, localCtx.FramePointerIndex)

			// Add variable offset if not zero
			if targetLocal.Address > 0 {
				writeByte(buf, I32_CONST)
				writeLEB128Signed(buf, int64(targetLocal.Address))
				writeByte(buf, I32_ADD)
			}
		}

	case NodeUnary:
		if node.Op == "*" {
			// Pointer dereference - the pointer value is the address
			EmitExpressionR(buf, node.Children[0], localCtx)
		} else {
			panic("Cannot use unary operator " + node.Op + " as lvalue")
		}

	case NodeDot:
		// Field access - emit field address
		baseExpr := node.Children[0]
		fieldName := node.FieldName

		EmitExpressionL(buf, baseExpr, localCtx)

		// Calculate and add field offset
		fieldOffset := getFieldOffset(baseExpr.TypeAST, fieldName)
		if fieldOffset > 0 {
			writeByte(buf, I32_CONST)
			writeLEB128Signed(buf, int64(fieldOffset))
			writeByte(buf, I32_ADD)
		}

	case NodeIndex:
		// Slice subscript operation - compute address of slice element
		// Formula: slice.items + (index * sizeof(elementType))

		sliceExpr := node.Children[0]
		indexExpr := node.Children[1]

		// Get slice base address (the slice struct itself)
		EmitExpressionL(buf, sliceExpr, localCtx)

		// Load the items field (which is a pointer to the elements)
		// items field is at offset 0 in the slice struct
		writeByte(buf, I32_LOAD) // Load i32 pointer from memory
		writeByte(buf, 0x02)     // alignment (4 bytes = 2^2)
		writeByte(buf, 0x00)     // offset 0 (items field)

		// Get the index value
		EmitExpressionR(buf, indexExpr, localCtx)

		// Convert index from I64 to I32 for address calculation
		writeByte(buf, I32_WRAP_I64)

		// Multiply index by element size
		elementType := node.TypeAST // This should be the element type from type checking
		elementSize := GetTypeSize(elementType)
		writeByte(buf, I32_CONST)
		writeLEB128(buf, uint32(elementSize))
		writeByte(buf, I32_MUL)

		// Add to base pointer to get final element address
		writeByte(buf, I32_ADD)

	default:
		// For any other expression (rvalue), create a temporary on tstack
		// Check if this is a struct-returning function call
		if node.Kind == NodeCall && node.TypeAST.Kind == TypeStruct {
			// Function call returning struct - it already returns the correct address
			EmitExpressionR(buf, node, localCtx)
			return
		}

		// For other rvalues, create a temporary on tstack
		// Save current tstack pointer - this will be the address we return
		writeByte(buf, GLOBAL_GET)
		writeLEB128(buf, 0) // tstack global index

		// Evaluate the rvalue to get its value on stack
		EmitExpressionR(buf, node, localCtx)
		// Stack: [tstack_addr, value]

		// Store the value to tstack based on its type
		if node.TypeAST.Kind == TypePointer {
			// Store pointer as i32
			writeByte(buf, I32_STORE)
			writeByte(buf, 0x02) // alignment (4 bytes = 2^2)
			writeByte(buf, 0x00) // offset

			// Get the address again (where we just stored the value)
			writeByte(buf, GLOBAL_GET)
			writeLEB128(buf, 0) // tstack global index (current position)

			// Update tstack pointer (advance by 4 bytes for I32)
			writeByte(buf, GLOBAL_GET)
			writeLEB128(buf, 0) // tstack global index
			writeByte(buf, I32_CONST)
			writeLEB128(buf, 4) // I32 size
			writeByte(buf, I32_ADD)
			writeByte(buf, GLOBAL_SET)
			writeLEB128(buf, 0) // tstack global index
		} else {
			// Store regular value as i64
			writeByte(buf, I64_STORE)
			writeByte(buf, 0x03) // alignment (8 bytes = 2^3)
			writeByte(buf, 0x00) // offset

			// Get the address again (where we just stored the value)
			writeByte(buf, GLOBAL_GET)
			writeLEB128(buf, 0) // tstack global index (current position)

			// Update tstack pointer (advance by 8 bytes for I64)
			writeByte(buf, GLOBAL_GET)
			writeLEB128(buf, 0) // tstack global index
			writeByte(buf, I32_CONST)
			writeLEB128(buf, 8) // I64 size
			writeByte(buf, I32_ADD)
			writeByte(buf, GLOBAL_SET)
			writeLEB128(buf, 0) // tstack global index
		}

		// Stack now has the address where we stored the value
	}
}

// EmitExpressionR emits code for rvalue expressions (expressions that produce values)
// These expressions produce a value on the stack that can be consumed
func EmitExpressionR(buf *bytes.Buffer, node *ASTNode, localCtx *LocalContext) {
	if node.TypeAST != nil && node.TypeAST.Kind == TypeInteger {
		panic("Unresolved Integer type in WASM generation: " + ToSExpr(node))
	}
	switch node.Kind {
	case NodeInteger:
		// Check if this integer should be emitted as I32 or I64 based on context
		if node.TypeAST != nil && isWASMI32Type(node.TypeAST) {
			writeByte(buf, I32_CONST)
			writeLEB128Signed(buf, node.Integer)
		} else {
			writeByte(buf, I64_CONST)
			writeLEB128Signed(buf, node.Integer)
		}

	case NodeBoolean:
		// Emit boolean as I64 (0 for false, 1 for true)
		writeByte(buf, I64_CONST)
		if node.Boolean {
			writeLEB128Signed(buf, 1)
		} else {
			writeLEB128Signed(buf, 0)
		}

	case NodeIdent:
		// Variable reference - emit value
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
			EmitExpressionL(buf, node, localCtx)
			emitValueLoadFromMemory(buf, targetLocal.Symbol.Type)
		}

	case NodeBinary:
		// Assignment is not allowed in rvalue context
		if node.Op == "=" {
			panic("Assignment cannot be used as rvalue")
		}

		// Binary operators (non-assignment)
		EmitExpressionR(buf, node.Children[0], localCtx) // LHS
		EmitExpressionR(buf, node.Children[1], localCtx) // RHS

		// Emit the appropriate operation based on operand types
		leftType := node.Children[0].TypeAST

		// For now, use left operand type to determine operation type
		// Both operands should have the same type after type checking
		var opcode byte
		if leftType != nil && isWASMI32Type(leftType) {
			opcode = getBinaryOpcodeI32(node.Op)
		} else {
			opcode = getBinaryOpcode(node.Op)
		}
		writeByte(buf, opcode)

		// Convert I32 comparison results to I64 for Bool compatibility
		if isComparisonOp(node.Op) {
			writeByte(buf, I64_EXTEND_I32_U) // Convert I32 to I64
		}

	case NodeCall:
		// Function call
		if len(node.Children) == 0 {
			panic("Invalid function call - missing function name")
		}
		functionName := node.Children[0].String

		if functionName == "print" {
			// Built-in print function
			if len(node.Children) != 2 {
				panic("print() function expects 1 argument")
			}
			// Emit argument
			arg := node.Children[1]
			EmitExpressionR(buf, arg, localCtx)

			// Convert i32 results to i64 for print
			if arg.Kind == NodeUnary && arg.Op == "&" {
				// Convert i32 address result to i64
				writeByte(buf, I64_EXTEND_I32_U)
			} else if arg.TypeAST != nil && isWASMI32Type(arg.TypeAST) {
				// Convert U8 (i32) values to i64 for print
				writeByte(buf, I64_EXTEND_I32_U)
			}

			// Call print
			writeByte(buf, CALL)
			writeLEB128(buf, 0) // function index 0 (print import)
		} else if functionName == "print_bytes" {
			// Built-in print_bytes function
			if len(node.Children) != 2 {
				panic("print_bytes() function expects 1 argument")
			}
			// Emit argument (slice address)
			arg := node.Children[1]
			EmitExpressionL(buf, arg, localCtx) // Get slice address (pointer to slice struct)

			// Call print_bytes
			writeByte(buf, CALL)
			writeLEB128(buf, 1) // function index 1 (print_bytes import)
		} else if functionName == "append" {
			// Call the generated append function for this slice type
			if len(node.Children) != 3 {
				panic("append() function expects 2 arguments")
			}

			slicePtrArg := node.Children[1]
			valueArg := node.Children[2]

			// Get slice type to determine which append function to call
			sliceType := slicePtrArg.TypeAST.Child // Slice type from pointer to slice

			// After transformation, sliceType is a struct with an "items" field
			var elementType *TypeNode
			if sliceType.Kind == TypeSlice {
				// Before transformation
				elementType = sliceType.Child
			} else if sliceType.Kind == TypeStruct {
				// After transformation - get element type from items field
				for _, field := range sliceType.Fields {
					if field.Name == "items" {
						// items field is TypePointer with Child as element type
						elementType = field.Type.Child
						break
					}
				}
			}

			appendFunctionName := "append_" + sanitizeTypeName(TypeToString(elementType))

			// Emit arguments
			EmitExpressionR(buf, slicePtrArg, localCtx)
			EmitExpressionR(buf, valueArg, localCtx)

			// Call the generated append function
			functionIndex, exists := globalFunctionMap[appendFunctionName]
			if !exists {
				panic("Generated append function not found: " + appendFunctionName)
			}
			writeByte(buf, CALL)
			writeLEB128(buf, uint32(functionIndex))
		} else {
			// User-defined function call
			args := node.Children[1:]

			// Emit arguments (including struct copies for struct parameters)
			for _, arg := range args {
				if arg.TypeAST.Kind == TypeStruct {
					// Struct argument - need to copy to a temporary location
					structSize := uint32(GetTypeSize(arg.TypeAST))

					// Allocate space on tstack for the struct copy
					writeByte(buf, GLOBAL_GET)
					writeLEB128(buf, 0) // tstack global index

					// Save the current tstack pointer (destination address)
					writeByte(buf, GLOBAL_GET)
					writeLEB128(buf, 0) // tstack global index

					// Get source address (the struct we're copying)
					EmitExpressionR(buf, arg, localCtx) // This should emit struct address for struct types

					// Push size for memory.copy
					writeByte(buf, I32_CONST)
					writeLEB128(buf, structSize)

					// Emit memory.copy instruction to copy struct to tstack
					writeByte(buf, 0xFC) // Multi-byte instruction prefix
					writeLEB128(buf, 10) // memory.copy opcode
					writeByte(buf, 0x00) // dst memory index (0)
					writeByte(buf, 0x00) // src memory index (0)

					// Update tstack pointer
					writeByte(buf, GLOBAL_GET)
					writeLEB128(buf, 0) // tstack global index
					writeByte(buf, I32_CONST)
					writeLEB128(buf, structSize)
					writeByte(buf, I32_ADD)
					writeByte(buf, GLOBAL_SET)
					writeLEB128(buf, 0) // tstack global index

					// Push the copy address as the function argument
					// (we saved it earlier before the memory.copy)
				} else {
					// Non-struct argument
					EmitExpressionR(buf, arg, localCtx)
				}
			}

			// Find function index
			functionIndex := findUserFunctionIndex(functionName)
			writeByte(buf, CALL)
			writeLEB128(buf, uint32(functionIndex))
		}

	case NodeUnary:
		if node.Op == "&" {
			// Address-of operator
			EmitExpressionL(buf, node.Children[0], localCtx)
			// Address is returned as i32 (standard for pointers in WASM)
		} else if node.Op == "*" {
			// Pointer dereference
			EmitExpressionR(buf, node.Children[0], localCtx) // Get pointer value (address as i32)
			// Load value from the address (i32 address is already correct for memory operations)
			writeByte(buf, I64_LOAD)
			writeByte(buf, 0x03) // alignment
			writeByte(buf, 0x00) // offset
		} else if node.Op == "!" {
			// Logical not operation
			panic("Unary not operator (!) not yet implemented")
		}

	case NodeDot:
		// Generate field address using EmitExpressionL
		EmitExpressionL(buf, node, localCtx)

		// Get the final field type to determine how to load it
		finalFieldType := getFinalFieldType(node)

		if finalFieldType == nil {
			panic("getFinalFieldType returned nil for: " + ToSExpr(node))
		}

		// Load the field value
		if isWASMI64Type(finalFieldType) {
			writeByte(buf, I64_LOAD) // Load i64 from memory
			writeByte(buf, 0x03)     // alignment (8 bytes = 2^3)
			writeByte(buf, 0x00)     // offset
		} else {
			panic("Non-I64 field types not supported in WASM yet: " + TypeToString(finalFieldType))
		}

	case NodeIndex:
		// Slice subscript operation - load value from computed address
		EmitExpressionL(buf, node, localCtx) // Get address of slice element

		// Load the value from the address based on element type
		elementType := node.TypeAST // TypeAST should be the element type from type checking
		if isWASMI64Type(elementType) {
			writeByte(buf, I64_LOAD) // Load i64 from memory
			writeByte(buf, 0x03)     // alignment (8 bytes = 2^3)
			writeByte(buf, 0x00)     // offset
		} else if isWASMI32Type(elementType) {
			if elementType.Kind == TypeBuiltin && elementType.String == "U8" {
				// Load U8 as single byte and extend to i32
				writeByte(buf, I32_LOAD8_U) // Load 8-bit unsigned to i32
				writeByte(buf, 0x00)        // alignment (1 byte = 2^0)
				writeByte(buf, 0x00)        // offset
			} else {
				writeByte(buf, I32_LOAD) // Load i32 from memory
				writeByte(buf, 0x02)     // alignment (4 bytes = 2^2)
				writeByte(buf, 0x00)     // offset
			}
		} else if elementType.Kind == TypeStruct {
			// For struct types, return the address (already computed by EmitExpressionL)
			// The address is already on the stack, no additional load needed
		} else {
			panic("Unsupported slice element type for WASM: " + TypeToString(elementType))
		}

	case NodeStruct:
		// Struct declarations should not appear in expression context
		panic("Struct declaration cannot be used as expression: " + ToSExpr(node))

	case NodeString:
		// String literal creates a slice structure on tstack
		stringContent := node.String
		stringAddress := findStringAddress(stringContent)
		stringLength := uint32(len(stringContent))

		// Get current tstack pointer (this will be our slice address)
		writeByte(buf, GLOBAL_GET)
		writeLEB128(buf, 0) // tstack global index

		// Duplicate tstack pointer for use in store operations
		writeByte(buf, GLOBAL_GET)
		writeLEB128(buf, 0) // tstack global index

		// Store items pointer (string address) at offset 0
		writeByte(buf, I32_CONST)
		writeLEB128(buf, stringAddress)
		writeByte(buf, I32_STORE)
		writeByte(buf, 0x02) // alignment (4 bytes = 2^2)
		writeByte(buf, 0x00) // offset 0 (items field)

		// Get tstack pointer again for length field
		writeByte(buf, GLOBAL_GET)
		writeLEB128(buf, 0) // tstack global index

		// Store length at offset 8
		writeByte(buf, I64_CONST)
		writeLEB128Signed(buf, int64(stringLength))
		writeByte(buf, I64_STORE)
		writeByte(buf, 0x03) // alignment (8 bytes = 2^3)
		writeByte(buf, 0x08) // offset 8 (length field)

		// Advance tstack pointer by 16 bytes (slice size)
		writeByte(buf, GLOBAL_GET)
		writeLEB128(buf, 0) // tstack global index
		writeByte(buf, I32_CONST)
		writeLEB128(buf, 16) // slice struct size
		writeByte(buf, I32_ADD)
		writeByte(buf, GLOBAL_SET)
		writeLEB128(buf, 0) // tstack global index

	default:
		panic("Unknown expression node kind: " + string(node.Kind))
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

func getBinaryOpcodeI32(op string) byte {
	switch op {
	case "+":
		return I32_ADD
	case "-":
		return I32_SUB
	case "*":
		return I32_MUL
	case "/":
		return I32_DIV_S
	case "%":
		return I32_REM_S
	case "==":
		return I32_EQ
	case "!=":
		return I32_NE
	case "<":
		return I32_LT_S
	case ">":
		return I32_GT_S
	case "<=":
		return I32_LE_S
	case ">=":
		return I32_GE_S
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

	// Extract functions from the program first
	functions := extractFunctions(ast)

	// Perform type checking with original slice types
	err := CheckProgram(ast, symbolTable)
	if err != nil {
		panic(err.Error())
	}

	// Initialize globals for slice type collection before transformation
	globalSliceTypes = make(map[string]*TypeNode)
	generatedAppendFunctions = []*ASTNode{}

	// Collect slice types before transformation (when they're still slice types)
	collectSliceTypes(ast)
	generateAllAppendFunctions()

	// Apply slice-to-struct transformation pass after type checking and slice collection
	transformSlicesToStructs(ast, symbolTable)

	// Add generated append functions to the functions list
	functions = append(functions, generatedAppendFunctions...)

	// Collect string literals and create data section
	strings := collectStringLiterals(ast)
	dataSection := &DataSection{
		Strings:   strings,
		TotalSize: calculateDataSectionSize(strings),
	}

	// Initialize global string addresses map
	globalStringAddresses = make(map[string]uint32)
	for _, str := range strings {
		globalStringAddresses[str.Content] = str.Address
	}

	var buf bytes.Buffer

	// Check if this is legacy expression compilation (no functions)
	if len(functions) == 0 {
		// Legacy path for single expressions
		return compileLegacyExpression(ast, symbolTable)
	}

	// Emit WASM module header and sections in streaming fashion
	EmitWASMHeader(&buf)
	EmitTypeSection(&buf, functions)               // function type definitions
	EmitImportSection(&buf)                        // print function import
	EmitFunctionSection(&buf, functions)           // declare all functions
	EmitMemorySection(&buf)                        // memory for tstack operations
	EmitGlobalSection(&buf, dataSection.TotalSize) // tstack global with initial value
	EmitExportSection(&buf)                        // export main function
	EmitCodeSection(&buf, functions)               // all function bodies
	EmitDataSection(&buf, dataSection)             // string literal data

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

// isExpressionNode checks if a node is an expression (vs statement)
func isExpressionNode(node *ASTNode) bool {
	switch node.Kind {
	case NodeInteger, NodeBoolean, NodeIdent, NodeBinary, NodeCall, NodeDot, NodeUnary, NodeIndex:
		return true
	case NodeVar, NodeBlock, NodeReturn, NodeIf, NodeLoop, NodeBreak, NodeContinue, NodeFunc:
		return false
	default:
		return false
	}
}

// compileLegacyExpression compiles single expressions (backward compatibility)
func compileLegacyExpression(ast *ASTNode, symbolTable *SymbolTable) []byte {
	// Run type checking first
	tc := NewTypeChecker(symbolTable)
	var err error
	if isExpressionNode(ast) {
		err = CheckExpression(ast, tc)
	} else {
		err = CheckStatement(ast, tc)
	}
	if err != nil {
		panic(err) // For backward compatibility, panic on type errors
	}

	// Use same unified system
	localCtx := BuildLocalContext(ast, nil)

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
	EmitGlobalSection(&buf, 0) // tstack global with initial value 0 for legacy
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
func BuildLocalContext(ast *ASTNode, fnNode *ASTNode) *LocalContext {
	ctx := &LocalContext{}

	// Phase 1: Add parameters (if this is a function)
	if fnNode != nil {
		ctx.addParameters(fnNode)
	}

	// Phase 2: Collect body variables
	ctx.collectBodyVariables(ast)

	// Phase 3: Calculate frame pointer (if needed)
	ctx.calculateFramePointer()

	// Phase 4: Assign final WASM indices
	ctx.assignWASMIndices()

	return ctx
}

// addParameters adds function parameters to the LocalContext by finding them in the function body
func (ctx *LocalContext) addParameters(fnNode *ASTNode) {
	// Extract parameter info from function node
	params := fnNode.Parameters

	// Create a set of parameter names for lookup
	paramNames := make(map[string]*TypeNode)
	for _, param := range params {
		paramNames[param.Name] = param.Type
	}

	// Find parameter references in the function body and collect their symbols
	var findParamSymbols func(*ASTNode)
	findParamSymbols = func(node *ASTNode) {
		if node == nil {
			return
		}
		if node.Kind == NodeIdent {
			if _, isParam := paramNames[node.String]; isParam {
				// This is a parameter reference
				if node.Symbol != nil {
					// Check if we already have this parameter
					alreadyExists := false
					for _, existing := range ctx.Variables {
						if existing.Symbol == node.Symbol {
							alreadyExists = true
							break
						}
					}

					if !alreadyExists {
						ctx.Variables = append(ctx.Variables, LocalVarInfo{
							Symbol:  node.Symbol,
							Storage: VarStorageParameterLocal,
							// Address will be assigned later in assignWASMIndices
						})
						ctx.ParameterCount++
					}
				}
			}
		}

		// Recursively process children
		for _, child := range node.Children {
			findParamSymbols(child)
		}
	}

	// Traverse the function body to find parameter references
	if fnNode.Body != nil {
		findParamSymbols(fnNode.Body)
	}
}

// collectBodyVariables traverses AST to find all var declarations and address-of operations
func (ctx *LocalContext) collectBodyVariables(node *ASTNode) {
	var frameOffset uint32 = 0

	var traverse func(*ASTNode)
	traverse = func(node *ASTNode) {
		switch node.Kind {
		case NodeStruct:
			// Don't traverse struct children - field declarations are not local variables
			return
		case NodeVar:
			// Extract variable identifier node and name
			varIdent := node.Children[0]
			varName := varIdent.String

			resolvedType := node.TypeAST

			// Skip variables with unsupported types (same filter as BuildSymbolTable)
			if !(isWASMI64Type(resolvedType) || isWASMI32Type(resolvedType) || resolvedType.Kind == TypeStruct || resolvedType.Kind == TypeSlice) {
				// Skip unsupported types like string
				return
			}

			// Use the symbol from the identifier node
			if varIdent.Symbol == nil {
				panic("Variable identifier has no symbol information: " + varName)
			}

			// Support I64, I64* (pointers are i32 in WASM), and other types
			if isWASMI32Type(resolvedType) || isWASMI64Type(resolvedType) {
				ctx.Variables = append(ctx.Variables, LocalVarInfo{
					Symbol:  varIdent.Symbol,
					Storage: VarStorageLocal,
					// Address will be allocated later.
				})
			} else if resolvedType.Kind == TypeStruct {
				// Struct variables are always stored on tstack (addressed)
				structSize := uint32(GetTypeSize(resolvedType))

				ctx.Variables = append(ctx.Variables, LocalVarInfo{
					Symbol:  varIdent.Symbol,
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
	TRUE        = "TRUE"
	FALSE       = "FALSE"
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
	NodeBoolean  NodeKind = "NodeBoolean"
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
	// NodeBoolean:
	Boolean bool
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
	TypeBuiltin TypeKind = "TypeBuiltin" // I64, U8, Bool
	TypeInteger TypeKind = "TypeInteger" // Compile-time integer constants
	TypePointer TypeKind = "TypePointer" // *T
	TypeStruct  TypeKind = "TypeStruct"  // MyStruct
	TypeSlice   TypeKind = "TypeSlice"   // T[]
)

// TypeNode represents a type in the type system
type TypeNode struct {
	Kind TypeKind

	// For TypeBuiltin, TypeStruct
	String string // "I64", "Boolean", name of struct

	// For TypePointer, TypeSlice (element type)
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
	TypeI64         = &TypeNode{Kind: TypeBuiltin, String: "I64"}
	TypeU8          = &TypeNode{Kind: TypeBuiltin, String: "U8"}
	TypeIntegerNode = &TypeNode{Kind: TypeInteger, String: "Integer"}
	TypeBool        = &TypeNode{Kind: TypeBuiltin, String: "Boolean"}
)

// Global map to store string addresses during compilation
var globalStringAddresses map[string]uint32

// String literal data structures for WASM data section
type StringLiteral struct {
	Content string
	Address uint32
	Length  uint32
}

type DataSection struct {
	Strings   []StringLiteral
	TotalSize uint32
}

// Type utility functions

// TypesEqual checks if two TypeNodes are equal
func TypesEqual(a, b *TypeNode) bool {
	if a.Kind != b.Kind {
		return false
	}

	switch a.Kind {
	case TypeBuiltin:
		return a.String == b.String
	case TypeInteger:
		return true // All Integer types are equal
	case TypePointer:
		return TypesEqual(a.Child, b.Child)
	case TypeStruct:
		return a.String == b.String
	case TypeSlice:
		return TypesEqual(a.Child, b.Child)
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
		case "U8":
			return 1
		case "Boolean":
			return 8
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
	case TypeSlice:
		return 16 // slice is a struct with items pointer (8 bytes) + length (8 bytes)
	}
	// unreachable with current TypeKind values
	panic("Unknown TypeKind: " + string(t.Kind))
}

// TypeToString converts TypeNode to string for display/debugging
func TypeToString(t *TypeNode) string {
	if t == nil {
		return "<nil type>"
	}
	switch t.Kind {
	case TypeBuiltin:
		return t.String
	case TypeInteger:
		return t.String
	case TypePointer:
		return TypeToString(t.Child) + "*"
	case TypeStruct:
		return t.String
	case TypeSlice:
		return TypeToString(t.Child) + "[]"
	}
	// unreachable with current TypeKind values
	panic("Unknown TypeKind: " + string(t.Kind))
}

// getBuiltinType returns the built-in type for a given name
func getBuiltinType(name string) *TypeNode {
	switch name {
	case "I64":
		return TypeI64
	case "U8":
		return TypeU8
	case "Boolean":
		return TypeBool
	default:
		return nil
	}
}

// IsIntegerCompatible checks if an Integer type can be converted to targetType
func IsIntegerCompatible(integerValue int64, targetType *TypeNode) bool {
	switch targetType.Kind {
	case TypeBuiltin:
		switch targetType.String {
		case "I64":
			return true // I64 can hold any value we support
		case "U8":
			return integerValue >= 0 && integerValue <= 255
		case "Boolean":
			return false // No integer→Boolean conversion
		}
	}
	return false
}

// ResolveIntegerType resolves an Integer type to a concrete type based on context
// Returns error if the integer value doesn't fit in the target type
//
// Precondition: node.Kind == NodeInteger
func ResolveIntegerType(node *ASTNode, targetType *TypeNode) error {
	if node.Kind != NodeInteger || node.TypeAST.Kind != TypeInteger {
		panic("ResolveIntegerType called with non-constant")
	}

	if !IsIntegerCompatible(node.Integer, targetType) {
		return fmt.Errorf("cannot convert integer %d to %s", node.Integer, TypeToString(targetType))
	}
	node.TypeAST = targetType
	return nil
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
		// Only I64 and Boolean are known to map to WASM I64
		// Other types like "int", "string" are not supported in WASM generation
		return t.String == "I64" || t.String == "Boolean"
	case TypePointer:
		return false // pointers are I32 in WASM
	case TypeStruct:
		return false // structs are stored in memory, not as I64 locals
	case TypeSlice:
		return false // slices are stored in memory, not as I64 locals
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
		return t.String == "U8" // U8 maps to I32 in WASM
	case TypePointer:
		return true // all pointers are I32 in WASM
	case TypeStruct:
		return false // structs are stored in memory, not as I32 locals
	case TypeSlice:
		return false // slices are stored in memory, not as I32 locals
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
	LoopDepth   int // Track loop nesting for break/continue validation
}

func (tc *TypeChecker) EnterLoop() {
	tc.LoopDepth++
}

func (tc *TypeChecker) ExitLoop() {
	tc.LoopDepth--
}

func (tc *TypeChecker) InLoop() bool {
	return tc.LoopDepth > 0
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
	wasmIndex := uint32(2 + len(st.functions)) // print is at index 0, print_bytes is at index 1

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

// synthesizeSliceStruct converts a slice type to its internal struct representation
func synthesizeSliceStruct(sliceType *TypeNode) *TypeNode {
	if sliceType.Kind != TypeSlice {
		panic("Expected TypeSlice")
	}

	elementType := sliceType.Child
	sliceName := TypeToString(sliceType)

	// Create the internal struct: { var items ElementType*; var length I64; }
	fields := []StructField{
		{
			Name:   "items",
			Type:   &TypeNode{Kind: TypePointer, Child: elementType},
			Offset: 0,
		},
		{
			Name:   "length",
			Type:   TypeI64,
			Offset: 8, // pointer is 8 bytes
		},
	}

	return &TypeNode{
		Kind:   TypeStruct,
		String: sliceName,
		Fields: fields,
	}
}

// transformSlicesToStructs converts all slice types to struct types throughout the AST and symbol table
func transformSlicesToStructs(ast *ASTNode, symbolTable *SymbolTable) {
	// Transform types in symbol table
	for _, symbol := range symbolTable.variables {
		transformTypeNodeSlices(symbol.Type)
	}
	for _, structDef := range symbolTable.structs {
		for i := range structDef.Fields {
			transformTypeNodeSlices(structDef.Fields[i].Type)
		}
	}
	for _, funcDef := range symbolTable.functions {
		transformTypeNodeSlices(funcDef.ReturnType)
		for i := range funcDef.Parameters {
			transformTypeNodeSlices(funcDef.Parameters[i].Type)
		}
	}

	// Transform TypeAST nodes in the AST
	transformASTSlices(ast)
}

// transformTypeNodeSlices recursively converts slice types to struct types in a TypeNode tree
func transformTypeNodeSlices(typeNode *TypeNode) {
	if typeNode == nil {
		return
	}

	switch typeNode.Kind {
	case TypeSlice:
		// First, recursively transform the child type
		transformTypeNodeSlices(typeNode.Child)

		// Convert slice to its synthesized struct representation
		elementType := typeNode.Child
		sliceName := TypeToString(typeNode)

		// Create the internal struct: { var items ElementType*; var length I64; }
		fields := []StructField{
			{
				Name:   "items",
				Type:   &TypeNode{Kind: TypePointer, Child: elementType},
				Offset: 0,
			},
			{
				Name:   "length",
				Type:   TypeI64,
				Offset: 8, // pointer is 8 bytes
			},
		}

		// Update the typeNode in-place to become a struct
		typeNode.Kind = TypeStruct
		typeNode.String = sliceName
		typeNode.Fields = fields
		typeNode.Child = nil // Clear the Child since it's now a struct
	case TypePointer:
		transformTypeNodeSlices(typeNode.Child)
	case TypeStruct:
		for i := range typeNode.Fields {
			transformTypeNodeSlices(typeNode.Fields[i].Type)
		}
	}
}

// transformASTSlices recursively converts slice types to struct types in TypeAST fields
func transformASTSlices(node *ASTNode) {
	if node == nil {
		return
	}

	// Transform TypeAST if present
	if node.TypeAST != nil {
		transformTypeNodeSlices(node.TypeAST)
	}

	// Recursively transform children
	for _, child := range node.Children {
		transformASTSlices(child)
	}
}

// collectSliceTypes traverses the AST to find all slice types used
func collectSliceTypes(node *ASTNode) {
	if node == nil {
		return
	}

	// Collect slice types from the node's type
	if node.TypeAST != nil {
		collectSliceTypesFromType(node.TypeAST)
	}

	// For functions, also check the body
	if node.Kind == NodeFunc && node.Body != nil {
		collectSliceTypes(node.Body)
	}

	// Recursively process children
	for _, child := range node.Children {
		collectSliceTypes(child)
	}
}

// collectSliceTypesFromType recursively collects slice types from a type node
func collectSliceTypesFromType(typeNode *TypeNode) {
	if typeNode == nil {
		return
	}

	switch typeNode.Kind {
	case TypeSlice:
		typeKey := TypeToString(typeNode)
		if _, exists := globalSliceTypes[typeKey]; !exists {
			globalSliceTypes[typeKey] = typeNode
		}
		// Also collect from the element type
		collectSliceTypesFromType(typeNode.Child)
	case TypePointer:
		collectSliceTypesFromType(typeNode.Child)
	case TypeStruct:
		for _, field := range typeNode.Fields {
			collectSliceTypesFromType(field.Type)
		}
	}
}

// collectStringLiterals walks the AST and collects all string literals
func collectStringLiterals(node *ASTNode) []StringLiteral {
	if node == nil {
		return nil
	}

	var strings []StringLiteral
	var address uint32 = 0

	// Use map for deduplication as per plan
	stringMap := make(map[string]uint32)

	// Walk the AST and collect strings
	collectStringsFromNode(node, stringMap, &address)

	// Convert map to slice
	for content, addr := range stringMap {
		strings = append(strings, StringLiteral{
			Content: content,
			Address: addr,
			Length:  uint32(len(content)),
		})
	}

	return strings
}

// collectStringsFromNode recursively searches for NodeString nodes
func collectStringsFromNode(node *ASTNode, stringMap map[string]uint32, address *uint32) {
	if node == nil {
		return
	}

	if node.Kind == NodeString {
		content := node.String
		if _, exists := stringMap[content]; !exists {
			stringMap[content] = *address
			*address += uint32(len(content))
		}
	}

	// Recursively process children
	for _, child := range node.Children {
		collectStringsFromNode(child, stringMap, address)
	}

	// Process function bodies
	if node.Kind == NodeFunc && node.Body != nil {
		collectStringsFromNode(node.Body, stringMap, address)
	}
}

// calculateDataSectionSize computes the total size of the data section
func calculateDataSectionSize(strings []StringLiteral) uint32 {
	var total uint32 = 0
	for _, str := range strings {
		total += str.Length
	}
	return total
}

// findStringAddress looks up a string's address in the global string map
func findStringAddress(content string) uint32 {
	if globalStringAddresses == nil {
		panic("String addresses not initialized during compilation")
	}
	address, exists := globalStringAddresses[content]
	if !exists {
		panic("String address not found: " + content)
	}
	return address
}

// generateAppendFunction creates an append function for a specific slice type
func generateAppendFunction(sliceType *TypeNode) *ASTNode {
	elementType := sliceType.Child
	functionName := "append_" + sanitizeTypeName(TypeToString(elementType))

	// Function signature: func append_T(slice_ptr: T[]*, value: T): void
	parameters := []FunctionParameter{
		{
			Name: "slice_ptr",
			Type: &TypeNode{Kind: TypePointer, Child: sliceType},
		},
		{
			Name: "value",
			Type: elementType,
		},
	}

	// Create function body - for now, we'll use the existing append logic as a template
	// The actual implementation will be in the WASM generation
	functionNode := &ASTNode{
		Kind:         NodeFunc,
		FunctionName: functionName,
		Parameters:   parameters,
		ReturnType:   nil,          // void function
		Children:     []*ASTNode{}, // Empty body - implementation is in WASM generation
	}

	return functionNode
}

// sanitizeTypeName converts a type name to a valid function name suffix
func sanitizeTypeName(typeName string) string {
	// Replace problematic characters
	result := ""
	for _, r := range typeName {
		switch r {
		case '[', ']', '*', ' ':
			result += "_"
		default:
			result += string(r)
		}
	}
	return result
}

// generateAllAppendFunctions creates append functions for all collected slice types
func generateAllAppendFunctions() {
	for _, sliceType := range globalSliceTypes {
		appendFunc := generateAppendFunction(sliceType)
		generatedAppendFunctions = append(generatedAppendFunctions, appendFunc)
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
			hasInitializer := len(node.Children) > 1 // Check if there's an initialization expression

			// Skip variables with no type information
			if varType == nil {
				break
			}

			// Only add supported types to symbol table for type checking
			// This includes I64, Bool, pointers, struct types, and slice types
			if isWASMI64Type(varType) || isWASMI32Type(varType) || varType.Kind == TypeStruct || varType.Kind == TypeSlice {
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
				// For slice types, resolve the element type if it's a struct
				if varType.Kind == TypeSlice && varType.Child.Kind == TypeStruct {
					// Look up the struct definition for the element type
					elementStructDef := st.LookupStruct(varType.Child.String)
					if elementStructDef != nil {
						// Create new slice type with resolved element type
						varType = &TypeNode{
							Kind:  TypeSlice,
							Child: elementStructDef,
						}
						node.TypeAST = varType
					}
				}

				err := st.DeclareVariable(varName, varType)
				if err != nil {
					panic(err.Error())
				}

				// Mark variable as assigned if it has an initializer, or if it's a struct/slice
				if hasInitializer || varType.Kind == TypeStruct || varType.Kind == TypeSlice {
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

			// Resolve struct types in function return type
			if node.ReturnType != nil && node.ReturnType.Kind == TypeStruct {
				structDef := st.LookupStruct(node.ReturnType.String)
				if structDef != nil {
					// Use the complete struct definition as return type
					node.ReturnType = structDef
				}
			}

			// Declare function with resolved parameter types and return type
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

	// Execute all passes
	collectDeclarations(ast)

	// Pass 1.5: resolve struct field types now that all structs are declared
	for _, structType := range st.structs {
		for i, field := range structType.Fields {
			if field.Type.Kind == TypeStruct {
				// Look up the struct definition for this field
				fieldStructDef := st.LookupStruct(field.Type.String)
				if fieldStructDef != nil {
					// Update the field to use the complete struct definition
					structType.Fields[i].Type = fieldStructDef
				}
			}
		}
	}

	// Pass 1.6: recalculate field offsets now that all field types are resolved
	for _, structType := range st.structs {
		var currentOffset uint32 = 0
		for i, field := range structType.Fields {
			// Update the field offset
			structType.Fields[i].Offset = currentOffset

			// Calculate field size with resolved types
			fieldSize := GetTypeSize(field.Type)
			currentOffset += uint32(fieldSize)
		}
	}

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

		// If there's an initialization expression, type-check it
		if len(stmt.Children) > 1 {
			initExpr := stmt.Children[1]
			err := CheckExpression(initExpr, tc)
			if err != nil {
				return err
			}

			// Ensure initialization expression type matches variable type or allow implicit conversion
			initType := initExpr.TypeAST
			if !TypesEqual(varType, initType) {
				// Try to resolve Integer type to match variable type
				if initType.Kind == TypeInteger {
					err := ResolveIntegerType(initExpr, varType)
					if err != nil {
						return err
					}
					// Type resolution succeeded, continue
				} else {
					return fmt.Errorf("error: cannot initialize variable of type %s with value of type %s",
						TypeToString(varType), TypeToString(initType))
				}
			}
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

	case NodeCall, NodeIdent, NodeInteger, NodeDot, NodeUnary:
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

			// For now, resolve Integer types to I64 in return statements
			// TODO: Use actual function return type when proper return type checking is implemented
			returnValueType := stmt.Children[0].TypeAST
			if returnValueType.Kind == TypeInteger {
				err := ResolveIntegerType(stmt.Children[0], TypeI64)
				if err != nil {
					return fmt.Errorf("error: return statement %v", err)
				}
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

	case NodeLoop:
		// Check all statements in loop body
		tc.EnterLoop()
		for _, stmt := range stmt.Children {
			err := CheckStatement(stmt, tc)
			if err != nil {
				tc.ExitLoop()
				return err
			}
		}
		tc.ExitLoop()
		return nil

	case NodeBreak:
		if !tc.InLoop() {
			return fmt.Errorf("error: break statement outside of loop")
		}
		return nil

	case NodeContinue:
		if !tc.InLoop() {
			return fmt.Errorf("error: continue statement outside of loop")
		}
		return nil

	default:
		// Other statement types are valid for now
	}

	return nil
}

// CheckExpression validates an expression and stores type in expr.TypeAST

func CheckExpression(expr *ASTNode, tc *TypeChecker) error {

	switch expr.Kind {
	case NodeInteger:
		expr.TypeAST = TypeIntegerNode
		return nil

	case NodeBoolean:
		expr.TypeAST = TypeBool
		return nil

	case NodeString:
		// String literal has type U8[] (slice of U8)
		expr.TypeAST = &TypeNode{
			Kind:  TypeSlice,
			Child: TypeU8,
		}
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

			// Resolve Integer types based on the other operand
			var resultType *TypeNode
			if leftType.Kind == TypeInteger && rightType.Kind != TypeInteger {
				err := ResolveIntegerType(expr.Children[0], rightType)
				if err != nil {
					return err
				}
				leftType = expr.Children[0].TypeAST
				resultType = rightType
			} else if rightType.Kind == TypeInteger && leftType.Kind != TypeInteger {
				err := ResolveIntegerType(expr.Children[1], leftType)
				if err != nil {
					return err
				}
				rightType = expr.Children[1].TypeAST
				resultType = leftType
			} else if leftType.Kind == TypeInteger && rightType.Kind == TypeInteger {
				// Both are Integer - resolve both to I64 and use I64 as result
				err := ResolveIntegerType(expr.Children[0], TypeI64)
				if err != nil {
					return err
				}
				err = ResolveIntegerType(expr.Children[1], TypeI64)
				if err != nil {
					return err
				}
				leftType = TypeI64
				rightType = TypeI64
				resultType = TypeI64
			} else if !TypesEqual(leftType, rightType) {
				return fmt.Errorf("error: type mismatch in binary operation: %s vs %s",
					TypeToString(leftType), TypeToString(rightType))
			} else {
				resultType = leftType
			}

			// Set result type based on operator
			switch expr.Op {
			case "==", "!=", "<", ">", "<=", ">=":
				expr.TypeAST = TypeBool // Comparison operators return Boolean
			case "+", "-", "*", "/", "%":
				expr.TypeAST = resultType // Arithmetic operators return operand type
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

			// Resolve Integer type to I64 for print function
			argType := expr.Children[1].TypeAST
			if argType.Kind == TypeInteger {
				err := ResolveIntegerType(expr.Children[1], TypeI64)
				if err != nil {
					return fmt.Errorf("error: print() %v", err)
				}
			}

			expr.TypeAST = TypeI64 // print returns nothing, but use I64 for now
			return nil
		} else if funcName == "print_bytes" {
			// Built-in print_bytes function
			if len(expr.Children) != 2 {
				return fmt.Errorf("error: print_bytes() function expects 1 argument")
			}
			err := CheckExpression(expr.Children[1], tc)
			if err != nil {
				return err
			}

			// Check that argument is a slice (U8[])
			argType := expr.Children[1].TypeAST
			if argType.Kind != TypeSlice {
				return fmt.Errorf("error: print_bytes() expects a slice argument, got %s", TypeToString(argType))
			}

			expr.TypeAST = TypeI64 // print_bytes returns nothing, but use I64 for now
			return nil
		} else if funcName == "append" {
			// Built-in append function
			if len(expr.Children) != 3 {
				return fmt.Errorf("error: append() function expects 2 arguments")
			}

			// Check first argument (slice pointer)
			err := CheckExpression(expr.Children[1], tc)
			if err != nil {
				return err
			}
			// Check second argument (value to append)
			err = CheckExpression(expr.Children[2], tc)
			if err != nil {
				return err
			}

			slicePtrType := expr.Children[1].TypeAST
			valueType := expr.Children[2].TypeAST

			// First argument must be a pointer to a slice
			if slicePtrType.Kind != TypePointer || slicePtrType.Child.Kind != TypeSlice {
				return fmt.Errorf("error: append() first argument must be pointer to slice, got %s", TypeToString(slicePtrType))
			}

			// Value type must match slice element type or allow implicit conversion
			elementType := slicePtrType.Child.Child
			if !TypesEqual(valueType, elementType) {
				if valueType.Kind == TypeInteger {
					err := ResolveIntegerType(expr.Children[2], elementType)
					if err != nil {
						return fmt.Errorf("error: append() %v", err)
					}
				} else {
					return fmt.Errorf("error: append() value type %s does not match slice element type %s",
						TypeToString(valueType), TypeToString(elementType))
				}
			}

			expr.TypeAST = TypeI64 // append returns nothing, but use I64 for now
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

		// Handle direct struct, pointer-to-struct, slice, and pointer-to-slice access
		var structType *TypeNode
		if baseType.Kind == TypeStruct {
			// Direct struct access
			structType = baseType
		} else if baseType.Kind == TypePointer && baseType.Child.Kind == TypeStruct {
			// Pointer-to-struct access (struct parameters)
			structType = baseType.Child
		} else if baseType.Kind == TypeSlice {
			// Slice access - synthesize the internal struct representation
			structType = synthesizeSliceStruct(baseType)
		} else if baseType.Kind == TypePointer && baseType.Child.Kind == TypeSlice {
			// Pointer-to-slice access (slice parameters)
			structType = synthesizeSliceStruct(baseType.Child)
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

	case NodeIndex:
		// Array/slice subscript operation
		if len(expr.Children) != 2 {
			return fmt.Errorf("error: subscript operator expects 2 operands")
		}

		err := CheckExpression(expr.Children[0], tc)
		if err != nil {
			return err
		}
		err = CheckExpression(expr.Children[1], tc)
		if err != nil {
			return err
		}

		// Get base and index types from the type-checked children
		baseType := expr.Children[0].TypeAST
		indexType := expr.Children[1].TypeAST

		// Index must be I64 or resolve Integer to I64
		if !TypesEqual(indexType, TypeI64) {
			if indexType.Kind == TypeInteger {
				err := ResolveIntegerType(expr.Children[1], TypeI64)
				if err != nil {
					return fmt.Errorf("error: slice index %v", err)
				}
			} else {
				return fmt.Errorf("error: slice index must be I64, got %s", TypeToString(indexType))
			}
		}

		// Base must be a slice type or pointer-to-slice (for function parameters)
		var elementType *TypeNode
		if baseType.Kind == TypeSlice {
			// Direct slice access
			elementType = baseType.Child
		} else if baseType.Kind == TypePointer && baseType.Child.Kind == TypeSlice {
			// Pointer-to-slice access (slice parameters)
			elementType = baseType.Child.Child
		} else {
			return fmt.Errorf("error: cannot subscript non-slice type %s", TypeToString(baseType))
		}

		// Return the element type of the slice
		expr.TypeAST = elementType
		return nil

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

	// Type check arguments after reordering
	args = callExpr.Children[1:] // Get the reordered arguments
	for i, arg := range args {
		expectedType := function.Parameters[i].Type
		actualType := arg.TypeAST

		if actualType.Kind == TypeInteger {
			err := ResolveIntegerType(arg, expectedType)
			if err != nil {
				return fmt.Errorf("error: argument %d: %v", i+1, err)
			}
		} else if !TypesEqual(actualType, expectedType) {
			// Special case: allow struct argument for struct pointer parameter (copy semantics)
			if expectedType.Kind == TypePointer && expectedType.Child.Kind == TypeStruct &&
				actualType.Kind == TypeStruct &&
				TypesEqual(actualType, expectedType.Child) {
				// This is allowed: passing struct S to parameter S* (copy semantics)
			} else if expectedType.Kind == TypePointer && expectedType.Child.Kind == TypeSlice &&
				actualType.Kind == TypeSlice &&
				TypesEqual(actualType, expectedType.Child) {
				// This is allowed: passing slice T[] to parameter T[]* (copy semantics)
			} else {
				return fmt.Errorf("error: argument %d type mismatch: expected %s, got %s",
					i+1, TypeToString(expectedType), TypeToString(actualType))
			}
		}
	}

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

		// Set the TypeAST for code generation
		lhs.TypeAST = lhsType

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

		// Set the TypeAST for code generation
		lhs.TypeAST = lhsType

	} else if lhs.Kind == NodeDot {
		// Field assignment (e.g., s.field = value)
		err = CheckExpression(lhs, tc)
		if err != nil {
			return err
		}
		lhsType = lhs.TypeAST

	} else if lhs.Kind == NodeIndex {
		// Slice index assignment (e.g., slice[0] = value)
		err = CheckExpression(lhs, tc)
		if err != nil {
			return err
		}
		lhsType = lhs.TypeAST

	} else {
		return fmt.Errorf("error: left side of assignment must be a variable, field access, or dereferenced pointer")
	}

	// Ensure types match or allow implicit conversion
	if !TypesEqual(lhsType, rhsType) {
		// Try to resolve Integer type to match LHS
		if rhsType.Kind == TypeInteger {
			err := ResolveIntegerType(rhs, lhsType)
			if err != nil {
				return err
			}
			// Type resolution succeeded, continue
		} else {
			return fmt.Errorf("error: cannot assign %s to %s",
				TypeToString(rhsType), TypeToString(lhsType))
		}
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
	lhs.TypeAST = rhs.TypeAST
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
			} else if lit == "true" {
				CurrTokenType = TRUE
			} else if lit == "false" {
				CurrTokenType = FALSE
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
		// Validate ASCII characters only
		if input[pos] > 127 {
			panic("error: non-ASCII characters are not supported in string literals")
		}
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
	case NodeBoolean:
		if node.Boolean {
			return "(boolean true)"
		} else {
			return "(boolean false)"
		}
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
		if len(node.Children) > 1 {
			// Has initialization expression
			initExpr := ToSExpr(node.Children[1])
			return "(var " + name + " " + typeStr + " " + initExpr + ")"
		}
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

	case TRUE:
		node := &ASTNode{
			Kind:    NodeBoolean,
			Boolean: true,
		}
		SkipToken(TRUE)
		return node

	case FALSE:
		node := &ASTNode{
			Kind:    NodeBoolean,
			Boolean: false,
		}
		SkipToken(FALSE)
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

	// Handle slice and pointer suffixes
	resultType := baseType

	// Handle slice suffix: Type[]
	if CurrTokenType == LBRACKET {
		SkipToken(LBRACKET)
		if CurrTokenType == RBRACKET {
			SkipToken(RBRACKET)
			resultType = &TypeNode{
				Kind:  TypeSlice,
				Child: resultType,
			}
		} else {
			// Error: expected ']' after '['
			return nil
		}
	}

	// Handle pointer suffixes: Type*
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
		if CurrTokenType != IDENT && CurrTokenType != LBRACKET {
			return &ASTNode{} // error - expecting type
		}

		// Parse type using new TypeNode system
		varName.TypeAST = parseTypeExpression()
		if varName.TypeAST == nil {
			return &ASTNode{} // error - invalid type
		}

		// Check for optional initialization: var x I64 = value;
		if CurrTokenType == ASSIGN {
			SkipToken(ASSIGN)
			// Parse the initialization expression
			initExpr := ParseExpression()

			if CurrTokenType == SEMICOLON {
				SkipToken(SEMICOLON)
			}

			// Return a variable declaration with initialization
			// This will be semantically equivalent to: var x I64; x = value;
			return &ASTNode{
				Kind:     NodeVar,
				Children: []*ASTNode{varName, initExpr}, // Second child is initialization expression
				TypeAST:  varName.TypeAST,
			}
		} else {
			// Regular variable declaration without initialization
			if CurrTokenType == SEMICOLON {
				SkipToken(SEMICOLON)
			}
			return &ASTNode{
				Kind:     NodeVar,
				Children: []*ASTNode{varName},
				TypeAST:  varName.TypeAST,
			}
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

		// Convert struct and slice parameters to pointer types (per Phase 3 spec)
		finalParamType := paramType
		if paramType.Kind == TypeStruct || paramType.Kind == TypeSlice {
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
