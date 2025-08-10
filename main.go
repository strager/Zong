package main

import (
	"bytes"
	"fmt"
)

type Error struct {
	message string
}

type ErrorCollection struct {
	errors []Error
}

func (ec *ErrorCollection) Append(error Error) {
	ec.errors = append(ec.errors, error)
}

func (ec *ErrorCollection) HasErrors() bool {
	return len(ec.errors) > 0
}

func (ec *ErrorCollection) Count() int {
	return len(ec.errors)
}

func (ec *ErrorCollection) String() string {
	if len(ec.errors) == 0 {
		return ""
	}

	var result bytes.Buffer
	for i, err := range ec.errors {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(err.message)
	}
	return result.String()
}

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

	// Temporary locals for function call argument evaluation
	TempI64Count  uint32 // Count of I64 temporaries needed for function calls
	TempI32Count  uint32 // Count of I32 temporaries needed for function calls
	TempBaseIndex uint32 // Starting WASM local index for temporaries

	// Current temporary indices for code generation (to avoid collisions in nested calls)
	CurrentTempI32Index uint32 // Next available I32 temporary
	CurrentTempI64Index uint32 // Next available I64 temporary

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

// WASMWriter wraps *bytes.Buffer with WASM opcode methods
type WASMWriter struct {
	buf *bytes.Buffer
}

func NewWASMWriter(buf *bytes.Buffer) *WASMWriter {
	return &WASMWriter{buf: buf}
}

// Arithmetic operations
func (w *WASMWriter) i32_const(value int32) {
	writeByte(w.buf, 0x41)
	writeLEB128Signed(w.buf, int64(value))
}

func (w *WASMWriter) i64_const(value int64) {
	writeByte(w.buf, 0x42)
	writeLEB128Signed(w.buf, value)
}

func (w *WASMWriter) i32_add() {
	writeByte(w.buf, 0x6A)
}

func (w *WASMWriter) i32_sub() {
	writeByte(w.buf, 0x6B)
}

func (w *WASMWriter) i32_mul() {
	writeByte(w.buf, 0x6C)
}

func (w *WASMWriter) i32_div_s() {
	writeByte(w.buf, 0x6D)
}

func (w *WASMWriter) i32_rem_s() {
	writeByte(w.buf, 0x6F)
}

func (w *WASMWriter) i64_add() {
	writeByte(w.buf, 0x7C)
}

func (w *WASMWriter) i64_sub() {
	writeByte(w.buf, 0x7D)
}

func (w *WASMWriter) i64_mul() {
	writeByte(w.buf, 0x7E)
}

func (w *WASMWriter) i64_div_s() {
	writeByte(w.buf, 0x7F)
}

func (w *WASMWriter) i64_rem_s() {
	writeByte(w.buf, 0x81)
}

// Memory operations
func (w *WASMWriter) i32_load(align, offset uint32) {
	writeByte(w.buf, 0x28)
	writeByte(w.buf, byte(align))
	writeLEB128(w.buf, offset)
}

func (w *WASMWriter) i32_load8_u(align, offset uint32) {
	writeByte(w.buf, 0x2D)
	writeByte(w.buf, byte(align))
	writeLEB128(w.buf, offset)
}

func (w *WASMWriter) i32_store(align, offset uint32) {
	writeByte(w.buf, 0x36)
	writeByte(w.buf, byte(align))
	writeLEB128(w.buf, offset)
}

func (w *WASMWriter) i32_store8(align, offset uint32) {
	writeByte(w.buf, 0x3A)
	writeByte(w.buf, byte(align))
	writeLEB128(w.buf, offset)
}

func (w *WASMWriter) i64_load(align, offset uint32) {
	writeByte(w.buf, 0x29)
	writeByte(w.buf, byte(align))
	writeLEB128(w.buf, offset)
}

func (w *WASMWriter) i64_store(align, offset uint32) {
	writeByte(w.buf, 0x37)
	writeByte(w.buf, byte(align))
	writeLEB128(w.buf, offset)
}

// Local and global variables
func (w *WASMWriter) local_get(local uint32) {
	writeByte(w.buf, 0x20)
	writeLEB128(w.buf, local)
}

func (w *WASMWriter) local_set(local uint32) {
	writeByte(w.buf, 0x21)
	writeLEB128(w.buf, local)
}

func (w *WASMWriter) local_tee(local uint32) {
	writeByte(w.buf, 0x22)
	writeLEB128(w.buf, local)
}

func (w *WASMWriter) global_get(global uint32) {
	writeByte(w.buf, 0x23)
	writeLEB128(w.buf, global)
}

func (w *WASMWriter) global_set(global uint32) {
	writeByte(w.buf, 0x24)
	writeLEB128(w.buf, global)
}

// Control flow
func (w *WASMWriter) block(blockType byte) {
	writeByte(w.buf, 0x02)
	writeByte(w.buf, blockType)
}

func (w *WASMWriter) loop(blockType byte) {
	writeByte(w.buf, 0x03)
	writeByte(w.buf, blockType)
}

func (w *WASMWriter) if_stmt(blockType byte) {
	writeByte(w.buf, 0x04)
	writeByte(w.buf, blockType)
}

func (w *WASMWriter) else_stmt() {
	writeByte(w.buf, 0x05)
}

func (w *WASMWriter) end() {
	writeByte(w.buf, 0x0B)
}

func (w *WASMWriter) br(depth uint32) {
	writeByte(w.buf, 0x0C)
	writeLEB128(w.buf, depth)
}

func (w *WASMWriter) br_if(depth uint32) {
	writeByte(w.buf, 0x0D)
	writeLEB128(w.buf, depth)
}

func (w *WASMWriter) call(funcIndex uint32) {
	writeByte(w.buf, 0x10)
	writeLEB128(w.buf, funcIndex)
}

// Comparison operations
func (w *WASMWriter) i32_eq() {
	writeByte(w.buf, 0x46)
}

func (w *WASMWriter) i32_ne() {
	writeByte(w.buf, 0x47)
}

func (w *WASMWriter) i32_lt_s() {
	writeByte(w.buf, 0x48)
}

func (w *WASMWriter) i32_gt_s() {
	writeByte(w.buf, 0x4A)
}

func (w *WASMWriter) i32_le_s() {
	writeByte(w.buf, 0x4C)
}

func (w *WASMWriter) i32_ge_s() {
	writeByte(w.buf, 0x4E)
}

func (w *WASMWriter) i64_eq() {
	writeByte(w.buf, 0x51)
}

func (w *WASMWriter) i64_ne() {
	writeByte(w.buf, 0x52)
}

func (w *WASMWriter) i64_lt_s() {
	writeByte(w.buf, 0x53)
}

func (w *WASMWriter) i64_gt_s() {
	writeByte(w.buf, 0x55)
}

func (w *WASMWriter) i64_le_s() {
	writeByte(w.buf, 0x57)
}

func (w *WASMWriter) i64_ge_s() {
	writeByte(w.buf, 0x59)
}

// Type conversions
func (w *WASMWriter) i32_wrap_i64() {
	writeByte(w.buf, 0xA7)
}

func (w *WASMWriter) i64_extend_i32_s() {
	writeByte(w.buf, 0xAC)
}

func (w *WASMWriter) i64_extend_i32_u() {
	writeByte(w.buf, 0xAD)
}

// Other operations
func (w *WASMWriter) drop() {
	writeByte(w.buf, 0x1A)
}

func (w *WASMWriter) return_() {
	writeByte(w.buf, 0x0F) // RETURN opcode
}

func (w *WASMWriter) memory_copy(dst_mem, src_mem uint32) {
	writeByte(w.buf, 0xFC)          // Multi-byte instruction prefix
	writeLEB128(w.buf, 10)          // memory.copy opcode
	writeByte(w.buf, byte(dst_mem)) // dst memory index
	writeByte(w.buf, byte(src_mem)) // src memory index
}

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
	writeLEB128(&sectionBuf, 3) // 3 imports: print, print_bytes, and read_line functions

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

	// Import 3: read_line function
	// Module name "env"
	writeLEB128(&sectionBuf, 3) // length of "env"
	writeBytes(&sectionBuf, []byte("env"))

	// Import name "read_line"
	writeLEB128(&sectionBuf, 9) // length of "read_line"
	writeBytes(&sectionBuf, []byte("read_line"))

	// Import kind: function (0x00)
	writeByte(&sectionBuf, 0x00)

	// Type index (2)
	writeLEB128(&sectionBuf, 2)

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
	w := NewWASMWriter(&sectionBuf)
	writeLEB128(&sectionBuf, 1) // 1 global: tstack

	// Global type: i32 mutable (0x7F 0x01)
	writeByte(&sectionBuf, 0x7F) // i32
	writeByte(&sectionBuf, 0x01) // mutable

	// Initializer expression: i32.const dataSize + end
	w.i32_const(int32(dataSize))
	w.end()

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

// WASMContext holds state for WASM code generation
type WASMContext struct {
	TypeRegistry     []FunctionType
	TypeMap          map[string]int
	FunctionRegistry []string
	FunctionMap      map[string]int
	StringAddresses  map[string]uint32
}

// NewWASMContext creates a new WASM context
func NewWASMContext() *WASMContext {
	return &WASMContext{
		TypeRegistry:     []FunctionType{},
		TypeMap:          make(map[string]int),
		FunctionRegistry: []string{},
		FunctionMap:      make(map[string]int),
		StringAddresses:  make(map[string]uint32),
	}
}

func (ctx *WASMContext) initTypeRegistry() {
	ctx.TypeRegistry = []FunctionType{}
	ctx.TypeMap = make(map[string]int)

	// Type 0: print function (i64) -> ()
	printType := FunctionType{
		Parameters: []byte{0x7E}, // i64
		Results:    []byte{},     // void
	}
	ctx.TypeRegistry = append(ctx.TypeRegistry, printType)
	ctx.TypeMap["(i64)->()"] = 0

	// Type 1: print_bytes function (i32) -> ()
	printBytesType := FunctionType{
		Parameters: []byte{0x7F}, // i32 (slice pointer)
		Results:    []byte{},     // void
	}
	ctx.TypeRegistry = append(ctx.TypeRegistry, printBytesType)
	ctx.TypeMap["(i32)->()"] = 1

	// Type 2: read_line function (i32) -> ()
	readLineType := FunctionType{
		Parameters: []byte{0x7F}, // i32 (destination address for slice)
		Results:    []byte{},     // void
	}
	ctx.TypeRegistry = append(ctx.TypeRegistry, readLineType)
	ctx.TypeMap["(i32)->()"] = 2
}

func (ctx *WASMContext) initFunctionRegistry(functions []*ASTNode) {
	ctx.FunctionRegistry = []string{}
	ctx.FunctionMap = make(map[string]int)

	// Function 0 is print (imported)
	ctx.FunctionRegistry = append(ctx.FunctionRegistry, "print")
	ctx.FunctionMap["print"] = 0

	// Function 1 is print_bytes (imported)
	ctx.FunctionRegistry = append(ctx.FunctionRegistry, "print_bytes")
	ctx.FunctionMap["print_bytes"] = 1

	// Function 2 is read_line (imported)
	ctx.FunctionRegistry = append(ctx.FunctionRegistry, "read_line")
	ctx.FunctionMap["read_line"] = 2

	// Add user functions starting from index 3
	for _, fn := range functions {
		index := len(ctx.FunctionRegistry)
		ctx.FunctionRegistry = append(ctx.FunctionRegistry, fn.FunctionName)
		ctx.FunctionMap[fn.FunctionName] = index
	}
}

func (ctx *WASMContext) EmitTypeSection(buf *bytes.Buffer, functions []*ASTNode) {
	writeByte(buf, 0x01) // type section id

	var sectionBuf bytes.Buffer

	// Initialize registries
	ctx.initTypeRegistry()
	ctx.initFunctionRegistry(functions)

	if len(functions) == 0 {
		// Legacy path - add main function type (void -> void)
		mainType := FunctionType{
			Parameters: []byte{},
			Results:    []byte{},
		}
		ctx.TypeRegistry = append(ctx.TypeRegistry, mainType)
	} else {
		// Register types for user functions
		for _, fn := range functions {
			sig := generateFunctionSignature(fn)
			if _, exists := ctx.TypeMap[sig]; !exists {
				ctx.TypeMap[sig] = len(ctx.TypeRegistry)
				ctx.TypeRegistry = append(ctx.TypeRegistry, createFunctionType(fn))
			}
		}
	}

	writeLEB128(&sectionBuf, uint32(len(ctx.TypeRegistry)))

	// Emit each function type
	for _, funcType := range ctx.TypeRegistry {
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

func (ctx *WASMContext) EmitFunctionSection(buf *bytes.Buffer, functions []*ASTNode) {
	writeByte(buf, 0x03) // function section id

	var sectionBuf bytes.Buffer

	if len(functions) == 0 {
		// Legacy path - emit single main function
		writeLEB128(&sectionBuf, 1) // 1 function
		writeLEB128(&sectionBuf, 3) // type index 3 (void -> void) - after print (0), print_bytes (1), and read_line (2)
	} else {
		writeLEB128(&sectionBuf, uint32(len(functions))) // number of functions

		// For each function, emit its type index
		for _, fn := range functions {
			typeIndex := ctx.findFunctionTypeIndex(fn)
			writeLEB128(&sectionBuf, uint32(typeIndex))
		}
	}

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

// findFunctionTypeIndex finds the type index for a function
func (ctx *WASMContext) findFunctionTypeIndex(fn *ASTNode) int {
	sig := generateFunctionSignature(fn)
	if index, exists := ctx.TypeMap[sig]; exists {
		return index
	}
	panic("Function type not found in registry: " + sig)
}

// findUserFunctionIndex finds the WASM index for a user-defined function
func (ctx *WASMContext) findUserFunctionIndex(functionName string) int {
	if index, exists := ctx.FunctionMap[functionName]; exists {
		return index
	}
	panic("Function not found in registry: " + functionName)
}

func (ctx *WASMContext) EmitExportSection(buf *bytes.Buffer) {
	writeByte(buf, 0x07) // export section id

	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, 3) // 3 exports: main function, memory, and tstack global

	// Export 1: main function
	writeLEB128(&sectionBuf, 4) // length of "main"
	writeBytes(&sectionBuf, []byte("main"))

	// Export kind: function (0x00)
	writeByte(&sectionBuf, 0x00)

	// Find main function index - fallback to index 1 if main not found
	var mainIndex int
	if ctx.FunctionMap != nil {
		if index, exists := ctx.FunctionMap["main"]; exists {
			mainIndex = index
		} else {
			// Legacy fallback for single-expression tests
			mainIndex = 3 // print=0, print_bytes=1, read_line=2, main=3
		}
	} else {
		// Legacy fallback when no function registry
		mainIndex = 3 // print=0, print_bytes=1, read_line=2, main=3
	}
	writeLEB128(&sectionBuf, uint32(mainIndex))

	// Export 2: memory
	writeLEB128(&sectionBuf, 6) // length of "memory"
	writeBytes(&sectionBuf, []byte("memory"))

	// Export kind: memory (0x02)
	writeByte(&sectionBuf, 0x02)

	// Memory index 0 (first memory)
	writeLEB128(&sectionBuf, 0)

	// Export 3: tstack global
	writeLEB128(&sectionBuf, 6) // length of "tstack"
	writeBytes(&sectionBuf, []byte("tstack"))

	// Export kind: global (0x03)
	writeByte(&sectionBuf, 0x03)

	// Global index 0 (first global - tstack)
	writeLEB128(&sectionBuf, 0)

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

func (ctx *WASMContext) EmitCodeSection(buf *bytes.Buffer, functions []*ASTNode) {
	writeByte(buf, 0x0A) // code section id

	var sectionBuf bytes.Buffer
	writeLEB128(&sectionBuf, uint32(len(functions))) // number of function bodies

	// Emit each function body
	for _, fn := range functions {
		ctx.emitSingleFunction(&sectionBuf, fn)
	}

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

// emitSingleFunction emits the code for a single function
func (ctx *WASMContext) emitSingleFunction(buf *bytes.Buffer, fn *ASTNode) {
	// Check if this is a generated append function
	if isGeneratedAppendFunction(fn.FunctionName) {
		emitAppendFunctionBody(buf, fn)
		return
	}

	// Use unified local management
	localCtx := BuildLocalContext(fn, fn)

	// Generate function body
	var bodyBuf bytes.Buffer

	// Generate WASM locals declaration
	emitLocalDeclarations(&bodyBuf, localCtx)

	// Generate frame setup if needed
	if localCtx.FrameSize > 0 {
		EmitFrameSetupFromContext(&bodyBuf, localCtx)
	}

	// Generate function body
	for _, stmt := range fn.Children {
		ctx.EmitStatement(&bodyBuf, stmt, localCtx)
	}
	w := NewWASMWriter(&bodyBuf)
	w.end() // end instruction

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
	w := NewWASMWriter(&bodyBuf)

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
	w.local_get(0)      // slice_ptr parameter
	w.i64_load(0x03, 8) // alignment, offset to length field
	w.local_tee(4)      // store current_length and keep on stack

	// 2. Calculate copy size: current_length * element_size
	w.i32_wrap_i64()
	w.i32_const(int32(elementSize))
	w.i32_mul()
	w.local_tee(3) // store copy_size and keep on stack

	// 3. Calculate total size needed: copy_size + element_size
	w.i32_const(int32(elementSize))
	w.i32_add()
	// Stack: [total_size]

	// 4. Allocate new space on tstack and store new_items
	w.global_get(0) // tstack
	w.local_tee(2)  // store new_items and keep on stack

	// Update tstack: tstack + total_size
	w.i32_add()     // add total_size (from step 3)
	w.global_set(0) // update tstack

	// 5. Copy existing elements using memory.copy
	// memory.copy(dest, src, size)
	w.local_get(2) // dest: new_items
	// Get old items pointer directly
	w.local_get(0)      // slice_ptr parameter
	w.i32_load(0x02, 0) // alignment, offset to items field (src: old_items)
	w.local_get(3)      // copy_size
	// Stack: [dest, src, size]

	// Emit memory.copy instruction
	w.memory_copy(0, 0) // dst memory index (0), src memory index (0)

	// 6. Store new element at the end
	// addr = new_items + copy_size
	w.local_get(2) // new_items
	w.local_get(3) // copy_size
	w.i32_add()
	// Stack: [new_element_addr]

	w.local_get(1) // value parameter
	// Stack: [new_element_addr, value]

	emitValueStoreToMemory(&bodyBuf, elementType)

	// 7. Update slice.items pointer
	w.local_get(0)       // slice_ptr parameter
	w.local_get(2)       // new_items
	w.i32_store(0x02, 0) // alignment, offset to items field

	// 8. Update slice.length
	w.local_get(0) // slice_ptr parameter
	w.local_get(4) // current_length (stored earlier)
	w.i64_const(1)
	w.i64_add()
	w.i64_store(0x03, 8) // alignment, offset to length field

	w.end() // end instruction

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
	w := NewWASMWriter(&sectionBuf)
	writeLEB128(&sectionBuf, uint32(len(dataSection.Strings))) // number of data segments

	// Emit each string as a data segment
	for _, str := range dataSection.Strings {
		// Segment type: 0x00 = active with memory index
		writeLEB128(&sectionBuf, 0x00)

		// Offset expression (i32.const address + end)
		w.i32_const(int32(str.Address))
		w.end()

		// Data size and content
		writeLEB128(&sectionBuf, str.Length)
		writeBytes(&sectionBuf, []byte(str.Content))
	}

	writeLEB128(buf, uint32(sectionBuf.Len()))
	writeBytes(buf, sectionBuf.Bytes())
}

// EmitStatement generates WASM bytecode for statements
func (ctx *WASMContext) EmitStatement(buf *bytes.Buffer, node *ASTNode, localCtx *LocalContext) {
	w := NewWASMWriter(buf)
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
			ctx.EmitExpression(buf, assignmentNode, localCtx)
		}
		break

	case NodeStruct:
		// Struct declarations don't generate runtime code
		// (they only define types)
		break

	case NodeBlock:
		// Emit all statements in the block
		for _, stmt := range node.Children {
			ctx.EmitStatement(buf, stmt, localCtx)
		}

	case NodeCall:
		// Handle expression statements (e.g., print calls)
		ctx.EmitExpression(buf, node, localCtx)

	case NodeReturn:
		// Return statement
		if len(node.Children) > 0 {
			// Function returns a value - emit the expression
			ctx.EmitExpression(buf, node.Children[0], localCtx)
		}
		// WASM return instruction (implicitly returns the value on stack)
		w.return_() // RETURN opcode

	case NodeIf:
		// If statement compilation
		// Structure: [condition, then_block, condition2?, else_block2?, ...]

		// Emit condition for initial if
		ctx.EmitExpression(buf, node.Children[0], localCtx)
		// Convert I64 Bool conditions to I32 for WASM if instruction
		if TypesEqual(node.Children[0].TypeAST, TypeBool) {
			w.i32_wrap_i64() // Convert I64 to I32
		}

		// Start if block
		w.if_stmt(0x40) // block type: void

		// Increment control depth for entire if statement
		localCtx.ControlDepth++
		// Emit then block
		ctx.EmitStatement(buf, node.Children[1], localCtx)

		// Handle else/else-if clauses
		i := 2
		for i < len(node.Children) {
			w.else_stmt() // else opcode

			// Check if this is an else-if (condition is not nil) or final else (condition is nil)
			if node.Children[i] != nil {
				// else-if: emit condition and start new if block
				ctx.EmitExpression(buf, node.Children[i], localCtx)
				// Convert I64 Bool conditions to I32 for WASM if instruction
				if TypesEqual(node.Children[i].TypeAST, TypeBool) {
					w.i32_wrap_i64() // Convert I64 to I32
				}
				w.if_stmt(0x40) // nested if with block type: void

				// Emit the else-if block
				ctx.EmitStatement(buf, node.Children[i+1], localCtx)
				i += 2
			} else {
				// final else: emit else block directly
				ctx.EmitStatement(buf, node.Children[i+1], localCtx)
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
			w.end() // end opcode
		}

		// Decrement control depth for entire if statement
		localCtx.ControlDepth--

	case NodeBinary:
		// Handle binary operations (mainly assignments)
		ctx.EmitExpression(buf, node, localCtx)

	case NodeLoop:
		// Save current control depth for this loop's break targeting
		localCtx.LoopStack = append(localCtx.LoopStack, localCtx.ControlDepth)

		// Emit WASM: block (for break - outer block)
		w.block(0x40) // void type

		// Emit WASM: loop (for continue - inner loop)
		w.loop(0x40) // void type

		// Emit loop body
		for _, stmt := range node.Children {
			ctx.EmitStatement(buf, stmt, localCtx)
		}

		// Emit branch back to loop start (this makes it an infinite loop until break)
		w.br(0) // branch depth 0 (back to loop start)

		// Emit WASM: end (loop)
		w.end()

		// Emit WASM: end (block)
		w.end()

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

		w.br(uint32(breakDepth))

	case NodeContinue:
		if len(localCtx.LoopStack) == 0 {
			panic("continue statement outside of loop")
		}

		// Emit WASM: br N (continue to inner loop, accounting for nested control structures)
		w.br(uint32(0 + localCtx.ControlDepth)) // branch depth (inner loop + nesting)

	default:
		// For now, treat unknown statements as expressions
		ctx.EmitExpression(buf, node, localCtx)
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
// getStructTypeFromBase extracts the struct type from a base type (handles both direct struct and pointer-to-struct)
func getStructTypeFromBase(baseType *TypeNode) *TypeNode {
	if baseType == nil {
		panic("Base expression has no type information")
	}

	if baseType.Kind == TypeStruct {
		return baseType
	} else if baseType.Kind == TypePointer && baseType.Child.Kind == TypeStruct {
		return baseType.Child
	} else {
		panic("Field access on non-struct type: " + TypeToString(baseType))
	}
}

func (ctx *WASMContext) EmitExpression(buf *bytes.Buffer, node *ASTNode, localCtx *LocalContext) {
	w := NewWASMWriter(buf)
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
				ctx.EmitExpressionR(buf, rhs, localCtx) // RHS value
				w.local_set(targetLocal.Address)
				return
			}
		} else {
			// Check if LHS is a valid assignment target
			if lhs.Kind != NodeUnary && lhs.Kind != NodeDot && lhs.Kind != NodeIndex {
				panic("Invalid assignment target - must be variable, field access, pointer dereference, or slice index")
			}
		}

		// Non-local storage - get address and store based on type
		ctx.EmitExpressionL(buf, lhs, localCtx) // Get address
		ctx.EmitExpressionR(buf, rhs, localCtx) // Get value

		// Stack is now: [address_i32, value]
		emitValueStoreToMemory(buf, lhs.TypeAST)
		return
	}

	// For all non-assignment expressions, delegate to EmitExpressionR
	ctx.EmitExpressionR(buf, node, localCtx)
}

// Precondition: WASM stack should be: [address_i32, value]
// Postcondition: WASM stack is: []
//
// The value on the stack is either the value itself (for primitives) or a pointer to the struct.
func emitValueStoreToMemory(buf *bytes.Buffer, ty *TypeNode) {
	w := NewWASMWriter(buf)
	switch ty.Kind {
	case TypeStruct:
		// Struct assignment (copy) using memory.copy
		structSize := uint32(GetTypeSize(ty))
		w.i32_const(int32(structSize))
		// Stack: [dst_addr, src_addr, size]
		w.memory_copy(0, 0) // dst memory index (0), src memory index (0)
	case TypePointer:
		// Store pointer as i32
		w.i32_store(0x02, 0x00) // alignment (4 bytes = 2^2), offset
	case TypeBuiltin:
		if ty.String == "U8" {
			// Store U8 as single byte
			w.i32_store8(0x00, 0x00) // alignment (1 byte = 2^0), offset
		} else {
			// Store other built-in types as i64
			w.i64_store(0x03, 0x00) // alignment (8 bytes = 2^3), offset
		}
	default:
		// Store regular value as i64
		w.i64_store(0x03, 0x00) // alignment (8 bytes = 2^3), offset
	}
}

// Precondition: WASM stack should be: [address_i32]
// Postcondition: WASM stack is: [value]
//
// The value on the stack upon return is either the value itself (for primitives) or a pointer to the struct.
func emitValueLoadFromMemory(buf *bytes.Buffer, ty *TypeNode) {
	w := NewWASMWriter(buf)
	if ty.Kind == TypeStruct {
		// For struct variables, return the address of the struct (not the value)
	} else {
		// Non-struct stack variable - load from memory
		if ty.Kind == TypePointer {
			// Load pointer as i32
			w.i32_load(0x02, 0x00) // alignment (4 bytes = 2^2), offset
		} else if ty.Kind == TypeBuiltin && ty.String == "U8" {
			// Load U8 as single byte and extend to i32
			w.i32_load8_u(0x00, 0x00) // alignment (1 byte = 2^0), offset
		} else {
			// Load regular value as i64
			w.i64_load(0x03, 0x00) // alignment (8 bytes = 2^3), offset
		}
	}
}

// EmitExpressionL emits code for lvalue expressions (expressions that can be assigned to or addressed)
// These expressions produce an address on the stack where a value can be stored or loaded from
func (ctx *WASMContext) EmitExpressionL(buf *bytes.Buffer, node *ASTNode, localCtx *LocalContext) {
	w := NewWASMWriter(buf)
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
			targetLocal.Symbol.Type.Kind == TypeStruct {
			// Struct parameter - it's stored as a pointer, so just load the pointer and add offset
			w.local_get(targetLocal.Address)
		} else if targetLocal.Storage == VarStorageLocal || targetLocal.Storage == VarStorageParameterLocal {
			// Local variable - can't take address of WASM local
			panic("Cannot take address of local variable: " + node.String)
		} else {
			// Stack variable - emit address
			w.local_get(localCtx.FramePointerIndex)

			// Add variable offset if not zero
			if targetLocal.Address > 0 {
				w.i32_const(int32(targetLocal.Address))
				w.i32_add()
			}
		}

	case NodeUnary:
		if node.Op == "*" {
			// Pointer dereference - the pointer value is the address
			ctx.EmitExpressionR(buf, node.Children[0], localCtx)
		} else {
			panic("Cannot use unary operator " + node.Op + " as lvalue")
		}

	case NodeDot:
		// Field access - emit field address
		baseExpr := node.Children[0]
		fieldName := node.FieldName

		ctx.EmitExpressionL(buf, baseExpr, localCtx)

		// Calculate and add field offset using symbol-based approach
		var fieldOffset uint32
		if node.FieldSymbol == nil {
			panic("Field symbol not found for field access: " + fieldName + " (FieldSymbol should always be populated during semantic analysis)")
		}

		// Get offset from the symbol table entry
		for _, field := range getStructTypeFromBase(baseExpr.TypeAST).Fields {
			if field.Symbol == node.FieldSymbol {
				fieldOffset = field.Offset
				break
			}
		}
		if fieldOffset > 0 {
			w.i32_const(int32(fieldOffset))
			w.i32_add()
		}

	case NodeIndex:
		// Slice subscript operation - compute address of slice element
		// Formula: slice.items + (index * sizeof(elementType))

		sliceExpr := node.Children[0]
		indexExpr := node.Children[1]

		// Get slice base address (the slice struct itself)
		ctx.EmitExpressionL(buf, sliceExpr, localCtx)

		// Load the items field (which is a pointer to the elements)
		// items field is at offset 0 in the slice struct
		w.i32_load(0x02, 0x00) // alignment (4 bytes = 2^2), offset 0 (items field)

		// Get the index value
		ctx.EmitExpressionR(buf, indexExpr, localCtx)

		// Convert index from I64 to I32 for address calculation
		w.i32_wrap_i64()

		// Multiply index by element size
		elementType := node.TypeAST // This should be the element type from type checking
		elementSize := GetTypeSize(elementType)
		w.i32_const(int32(elementSize))
		w.i32_mul()

		// Add to base pointer to get final element address
		w.i32_add()

	case NodeString:
		// String literal creates a slice structure directly on tstack
		// EmitExpressionR already returns the correct address
		ctx.EmitExpressionR(buf, node, localCtx)

	case NodeType:
		panic("Unexpected NodeType in lvalue context: " + TypeToString(node.ReturnType) + ". Type names cannot be assigned to.")

	default:
		// For any other expression (rvalue), create a temporary on tstack
		// Check if this is a struct-returning function call
		if node.Kind == NodeCall && node.TypeAST.Kind == TypeStruct {
			// Function call returning struct - it already returns the correct address
			ctx.EmitExpressionR(buf, node, localCtx)
			return
		}

		// For other rvalues, create a temporary on tstack
		// Save current tstack pointer - this will be the address we return
		w.global_get(0) // tstack global index

		// Evaluate the rvalue to get its value on stack
		ctx.EmitExpressionR(buf, node, localCtx)
		// Stack: [tstack_addr, value]

		// Store the value to tstack based on its type
		if node.TypeAST.Kind == TypePointer {
			// Store pointer as i32
			w.i32_store(0x02, 0x00) // alignment (4 bytes = 2^2), offset

			// Get the address again (where we just stored the value)
			w.global_get(0) // tstack global index (current position)

			// Update tstack pointer (advance by 4 bytes for I32)
			w.global_get(0) // tstack global index
			w.i32_const(4)  // I32 size
			w.i32_add()
			w.global_set(0) // tstack global index
		} else {
			// Store regular value as i64
			w.i64_store(0x03, 0x00) // alignment (8 bytes = 2^3), offset

			// Get the address again (where we just stored the value)
			w.global_get(0) // tstack global index (current position)

			// Update tstack pointer (advance by 8 bytes for I64)
			w.global_get(0) // tstack global index
			w.i32_const(8)  // I64 size
			w.i32_add()
			w.global_set(0) // tstack global index
		}

		// Stack now has the address where we stored the value
	}
}

// EmitExpressionR emits code for rvalue expressions (expressions that produce values)
// These expressions produce a value on the stack that can be consumed
func (ctx *WASMContext) EmitExpressionR(buf *bytes.Buffer, node *ASTNode, localCtx *LocalContext) {
	w := NewWASMWriter(buf)
	if node.TypeAST != nil && node.TypeAST.Kind == TypeInteger {
		panic("Unresolved Integer type in WASM generation: " + ToSExpr(node))
	}
	switch node.Kind {
	case NodeInteger:
		// Check if this integer should be emitted as I32 or I64 based on context
		if node.TypeAST != nil && isWASMI32Type(node.TypeAST) {
			w.i32_const(int32(node.Integer))
		} else {
			w.i64_const(node.Integer)
		}

	case NodeBoolean:
		// Emit boolean as I64 (0 for false, 1 for true)
		if node.Boolean {
			w.i64_const(1)
		} else {
			w.i64_const(0)
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
			w.local_get(targetLocal.Address)
		} else {
			// Stack variable
			ctx.EmitExpressionL(buf, node, localCtx)
			emitValueLoadFromMemory(buf, targetLocal.Symbol.Type)
		}

	case NodeBinary:
		// Assignment is not allowed in rvalue context
		if node.Op == "=" {
			panic("Assignment cannot be used as rvalue")
		}

		if node.Op == "&&" {
			// Logical AND with short-circuiting:
			// if (left) { return right; } else { return 0; }
			ctx.EmitExpressionR(buf, node.Children[0], localCtx) // Evaluate left operand (i64)
			w.i32_wrap_i64()                                     // Convert i64 to i32 for if condition
			w.if_stmt(0x7E)                                      // if (left is truthy), result type: i64
			ctx.EmitExpressionR(buf, node.Children[1], localCtx) // evaluate and return right operand
			w.else_stmt()                                        // else
			w.i64_const(0)                                       // return false (0)
			w.end()                                              // end if
			return
		} else if node.Op == "||" {
			// Logical OR with short-circuiting:
			// if (left) { return 1; } else { return right; }
			ctx.EmitExpressionR(buf, node.Children[0], localCtx) // Evaluate left operand (i64)
			w.i32_wrap_i64()                                     // Convert i64 to i32 for if condition
			w.if_stmt(0x7E)                                      // if (left is truthy), result type: i64
			w.i64_const(1)                                       // return true (1)
			w.else_stmt()                                        // else
			ctx.EmitExpressionR(buf, node.Children[1], localCtx) // evaluate and return right operand
			w.end()                                              // end if
			return
		}

		// Binary operators (non-assignment, non-logical)
		ctx.EmitExpressionR(buf, node.Children[0], localCtx) // LHS
		ctx.EmitExpressionR(buf, node.Children[1], localCtx) // RHS

		// Emit the appropriate operation based on operand types
		leftType := node.Children[0].TypeAST

		// For now, use left operand type to determine operation type
		// Both operands should have the same type after type checking
		if leftType != nil && isWASMI32Type(leftType) {
			// I32 binary operations
			switch node.Op {
			case "+":
				w.i32_add()
			case "-":
				w.i32_sub()
			case "*":
				w.i32_mul()
			case "/":
				w.i32_div_s()
			case "%":
				w.i32_rem_s()
			case "==":
				w.i32_eq()
			case "!=":
				w.i32_ne()
			case "<":
				w.i32_lt_s()
			case ">":
				w.i32_gt_s()
			case "<=":
				w.i32_le_s()
			case ">=":
				w.i32_ge_s()
			default:
				panic("Unsupported I32 binary operator: " + node.Op)
			}
		} else {
			// I64 binary operations
			switch node.Op {
			case "+":
				w.i64_add()
			case "-":
				w.i64_sub()
			case "*":
				w.i64_mul()
			case "/":
				w.i64_div_s()
			case "%":
				w.i64_rem_s()
			case "==":
				w.i64_eq()
			case "!=":
				w.i64_ne()
			case "<":
				w.i64_lt_s()
			case ">":
				w.i64_gt_s()
			case "<=":
				w.i64_le_s()
			case ">=":
				w.i64_ge_s()
			default:
				panic("Unsupported I64 binary operator: " + node.Op)
			}
		}

		// Convert I32 comparison results to I64 for Bool compatibility
		if isComparisonOp(node.Op) {
			w.i64_extend_i32_u() // Convert I32 to I64
		}

	case NodeCall:
		// Function call or struct initialization
		if len(node.Children) == 0 {
			panic("Invalid function call - missing function name")
		}

		// Handle NodeType as callee (struct constructor calls)
		if node.Children[0].Kind == NodeType {
			// This is struct initialization with NodeType callee
			structType := node.Children[0].ReturnType

			// Save current tstack pointer as the struct address (this will be returned)
			w.global_get(0) // tstack global index

			// For each argument in source order, store the provided value at the correct field offset
			for i, paramName := range node.ParameterNames {
				// Find the field for this parameter
				var field *Parameter
				for j := range structType.Fields {
					if structType.Fields[j].Name == paramName {
						field = &structType.Fields[j]
						break
					}
				}
				if field == nil {
					panic("Missing field for parameter " + paramName + " in struct initialization")
				}

				// Duplicate struct base address for this field store
				w.global_get(0) // tstack global index (struct base address)

				// Emit field offset if needed
				if field.Offset > 0 {
					w.i32_const(int32(field.Offset))
					w.i32_add()
				}

				// Emit the argument expression
				ctx.EmitExpressionR(buf, node.Children[i+1], localCtx)

				// Store the value at the memory address (address + offset already computed)
				emitValueStoreToMemory(buf, field.Type)
			}

			// Update tstack to point past the allocated struct
			structSize := GetTypeSize(structType)
			w.global_get(0) // tstack global index
			w.i32_const(int32(structSize))
			w.i32_add()
			w.global_set(0) // tstack global index

			// The struct address is already on the stack from the first GLOBAL_GET
			return
		}

		functionName := node.Children[0].String

		if functionName == "print" {
			// Built-in print function
			if len(node.Children) != 2 {
				panic("print() function expects 1 argument")
			}
			// Emit argument
			arg := node.Children[1]
			ctx.EmitExpressionR(buf, arg, localCtx)

			// Convert i32 results to i64 for print
			if arg.Kind == NodeUnary && arg.Op == "&" {
				// Convert i32 address result to i64
				w.i64_extend_i32_u()
			} else if arg.TypeAST != nil && isWASMI32Type(arg.TypeAST) {
				// Convert U8 (i32) values to i64 for print
				w.i64_extend_i32_u()
			}

			// Call print
			w.call(0) // function index 0 (print import)
		} else if functionName == "print_bytes" {
			// Built-in print_bytes function
			if len(node.Children) != 2 {
				panic("print_bytes() function expects 1 argument")
			}
			// Emit argument (slice address)
			arg := node.Children[1]
			ctx.EmitExpressionL(buf, arg, localCtx) // Get slice address (pointer to slice struct)

			// Call print_bytes
			w.call(1) // function index 1 (print_bytes import)
		} else if functionName == "read_line" {
			// Built-in read_line function
			if len(node.Children) != 1 {
				panic("read_line() function expects no arguments")
			}

			// read_line returns a struct (slice), so allocate space on tstack
			// Get current tstack pointer (this will be the return address)
			w.global_get(0) // tstack global index

			// Duplicate the address for the function call
			w.global_get(0) // tstack global index

			// Update tstack pointer (advance by slice size: 16 bytes)
			w.global_get(0) // tstack global index
			w.i32_const(16) // slice size (pointer + length)
			w.i32_add()
			w.global_set(0) // tstack global index

			// Call read_line with destination address as parameter
			// Stack: [return_addr, dest_addr]
			w.call(2) // function index 2 (read_line import)

			// Stack now has: [return_addr] - the address where the slice was stored
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
			ctx.EmitExpressionR(buf, slicePtrArg, localCtx)
			ctx.EmitExpressionR(buf, valueArg, localCtx)

			// Call the generated append function
			functionIndex, exists := ctx.FunctionMap[appendFunctionName]
			if !exists {
				panic("Generated append function not found: " + appendFunctionName)
			}
			w.call(uint32(functionIndex))
		} else {
			// User-defined function call with source-order evaluation
			args := node.Children[1:]
			if node.Children[0].Symbol == nil || node.Children[0].Symbol.Kind != SymbolFunction {
				panic("Missing resolved function for: " + functionName)
			}
			function := node.Children[0].Symbol.FunctionInfo

			// Phase 1: Evaluate arguments in source order and store in temporaries
			tempIndices := make([]uint32, len(args))
			tempI32Index := localCtx.CurrentTempI32Index
			tempI64Index := localCtx.CurrentTempI64Index

			for i, arg := range args {
				if arg.TypeAST.Kind == TypeStruct {
					// Struct arguments use tstack storage, not locals
					// We'll handle these differently - no temporary needed
					tempIndices[i] = 0 // Mark as tstack
				} else if isWASMI64Type(arg.TypeAST) {
					// Evaluate argument and store in i64 temporary
					ctx.EmitExpressionR(buf, arg, localCtx)
					w.local_set(tempI64Index)
					tempIndices[i] = tempI64Index
					tempI64Index++
				} else if isWASMI32Type(arg.TypeAST) {
					// Evaluate argument and store in i32 temporary
					ctx.EmitExpressionR(buf, arg, localCtx)
					w.local_set(tempI32Index)
					tempIndices[i] = tempI32Index
					tempI32Index++
				}
			}

			// Update current indices after allocating temporaries
			localCtx.CurrentTempI32Index = tempI32Index
			localCtx.CurrentTempI64Index = tempI64Index

			// Phase 2: Build parameter mapping and emit arguments in parameter order
			paramMapping := make([]int, len(args))
			for i, paramName := range node.ParameterNames {
				if paramName == "" {
					// Positional argument
					paramMapping[i] = i
				} else {
					// Named argument - find parameter index
					found := false
					for j, param := range function.Parameters {
						if param.Name == paramName {
							paramMapping[i] = j
							found = true
							break
						}
					}
					if !found {
						panic("Parameter not found: " + paramName)
					}
				}
			}

			// Create parameter-ordered array for function call
			orderedArgs := make([]*ASTNode, len(args))
			orderedTempIndices := make([]uint32, len(args))
			for i := 0; i < len(args); i++ {
				paramIndex := paramMapping[i]
				orderedArgs[paramIndex] = args[i]
				orderedTempIndices[paramIndex] = tempIndices[i]
			}
			// Emit arguments in parameter order using temporaries
			for i, arg := range orderedArgs {
				if arg.TypeAST.Kind == TypeStruct {
					// Struct argument - evaluate again for tstack copy
					structSize := uint32(GetTypeSize(arg.TypeAST))

					// Allocate space on tstack for the struct copy
					w.global_get(0) // tstack global index

					// Save the current tstack pointer (destination address)
					w.global_get(0) // tstack global index

					// Get source address (the struct we're copying)
					ctx.EmitExpressionR(buf, arg, localCtx)

					// Push size for memory.copy
					w.i32_const(int32(structSize))

					// Emit memory.copy instruction to copy struct to tstack
					w.memory_copy(0, 0) // dst memory index (0), src memory index (0)

					// Update tstack pointer
					w.global_get(0) // tstack global index
					w.i32_const(int32(structSize))
					w.i32_add()
					w.global_set(0) // tstack global index

					// Push the copy address as the function argument
					// (we saved it earlier before the memory.copy)
				} else {
					// Load from temporary local
					w.local_get(orderedTempIndices[i])
				}
			}

			// Find function index and call
			functionIndex := ctx.findUserFunctionIndex(functionName)
			w.call(uint32(functionIndex))
		}

	case NodeType:
		panic("Unexpected NodeType in WASM generation: " + TypeToString(node.ReturnType) + ". Type names should be part of constructor calls.")

	case NodeUnary:
		if node.Op == "&" {
			// Address-of operator
			ctx.EmitExpressionL(buf, node.Children[0], localCtx)
			// Address is returned as i32 (standard for pointers in WASM)
		} else if node.Op == "*" {
			// Pointer dereference
			ctx.EmitExpressionR(buf, node.Children[0], localCtx) // Get pointer value (address as i32)
			// Load value from the address (i32 address is already correct for memory operations)
			w.i64_load(0x03, 0x00) // alignment, offset
		} else if node.Op == "!" {
			// Logical NOT operation: 1 - value (where value is 0 or 1)
			w.i64_const(1)
			ctx.EmitExpressionR(buf, node.Children[0], localCtx) // Get boolean value (0 or 1)
			w.i64_sub()                                          // 1 - value gives us the NOT
		}

	case NodeDot:
		// Generate field address using EmitExpressionL
		ctx.EmitExpressionL(buf, node, localCtx)

		// Get the final field type to determine how to load it
		finalFieldType := getFinalFieldType(node)

		if finalFieldType == nil {
			panic("getFinalFieldType returned nil for: " + ToSExpr(node))
		}

		// Load the field value
		if isWASMI64Type(finalFieldType) {
			w.i64_load(0x03, 0x00) // alignment (8 bytes = 2^3), offset
		} else {
			panic("Non-I64 field types not supported in WASM yet: " + TypeToString(finalFieldType))
		}

	case NodeIndex:
		// Slice subscript operation - load value from computed address
		ctx.EmitExpressionL(buf, node, localCtx) // Get address of slice element

		// Load the value from the address based on element type
		elementType := node.TypeAST // TypeAST should be the element type from type checking
		if isWASMI64Type(elementType) {
			w.i64_load(0x03, 0x00) // alignment (8 bytes = 2^3), offset
		} else if isWASMI32Type(elementType) {
			if elementType.Kind == TypeBuiltin && elementType.String == "U8" {
				// Load U8 as single byte and extend to i32
				w.i32_load8_u(0x00, 0x00) // alignment (1 byte = 2^0), offset
			} else {
				w.i32_load(0x02, 0x00) // alignment (4 bytes = 2^2), offset
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
		stringAddress := ctx.findStringAddress(stringContent)
		stringLength := uint32(len(stringContent))

		// Get current tstack pointer (this will be our slice address)
		w.global_get(0) // tstack global index

		// Duplicate tstack pointer for use in store operations
		w.global_get(0) // tstack global index

		// Store items pointer (string address) at offset 0
		w.i32_const(int32(stringAddress))
		w.i32_store(0x02, 0x00) // alignment (4 bytes = 2^2), offset 0 (items field)

		// Get tstack pointer again for length field
		w.global_get(0) // tstack global index

		// Store length at offset 8
		w.i64_const(int64(stringLength))
		w.i64_store(0x03, 0x08) // alignment (8 bytes = 2^3), offset 8 (length field)

		// Advance tstack pointer by 16 bytes (slice size)
		w.global_get(0) // tstack global index
		w.i32_const(16) // slice struct size
		w.i32_add()
		w.global_set(0) // tstack global index

	default:
		panic("Unknown expression node kind: " + string(node.Kind))
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
	if symbolTable.Errors.HasErrors() {
		panic("symbol resolution failed: " + symbolTable.Errors.String())
	}

	// Extract functions from the program first
	functions := extractFunctions(ast)

	// Perform type checking with original slice types
	typeErrors := CheckProgram(ast, symbolTable.typeTable)
	if typeErrors.HasErrors() {
		panic("type checking failed: " + typeErrors.String())
	}

	// Collect slice types before transformation (when they're still slice types)
	collectSliceTypes(ast, symbolTable.typeTable)
	generatedAppendFunctions := generateAllAppendFunctions(symbolTable.typeTable)

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

	// Create WASM context and initialize string addresses
	ctx := NewWASMContext()
	for _, str := range strings {
		ctx.StringAddresses[str.Content] = str.Address
	}

	var buf bytes.Buffer

	// Check if this is legacy expression compilation (no functions)
	if len(functions) == 0 {
		// Legacy path for single expressions
		return compileLegacyExpression(ast, symbolTable.typeTable)
	}

	// Emit WASM module header and sections in streaming fashion
	EmitWASMHeader(&buf)
	ctx.EmitTypeSection(&buf, functions)           // function type definitions
	EmitImportSection(&buf)                        // print function import
	ctx.EmitFunctionSection(&buf, functions)       // declare all functions
	EmitMemorySection(&buf)                        // memory for tstack operations
	EmitGlobalSection(&buf, dataSection.TotalSize) // tstack global with initial value
	ctx.EmitExportSection(&buf)                    // export main function
	ctx.EmitCodeSection(&buf, functions)           // all function bodies
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
func compileLegacyExpression(ast *ASTNode, typeTable *TypeTable) []byte {
	// Run type checking first
	tc := NewTypeChecker(typeTable)
	if isExpressionNode(ast) {
		CheckExpression(ast, tc)
	} else {
		CheckStatement(ast, tc)
	}

	// Use same unified system
	localCtx := BuildLocalContext(ast, nil)

	// Create WASM context for legacy expressions
	ctx := NewWASMContext()

	// Generate function body
	var bodyBuf bytes.Buffer
	w := NewWASMWriter(&bodyBuf)

	// Generate WASM with unified approach
	emitLocalDeclarations(&bodyBuf, localCtx)
	if localCtx.FrameSize > 0 {
		EmitFrameSetupFromContext(&bodyBuf, localCtx)
	}
	ctx.EmitStatement(&bodyBuf, ast, localCtx)
	w.end()

	// Build the full WASM module
	var buf bytes.Buffer
	EmitWASMHeader(&buf)
	ctx.EmitTypeSection(&buf, []*ASTNode{}) // empty functions for legacy
	EmitImportSection(&buf)
	ctx.EmitFunctionSection(&buf, []*ASTNode{}) // empty functions for legacy
	EmitMemorySection(&buf)
	EmitGlobalSection(&buf, 0) // tstack global with initial value 0 for legacy
	ctx.EmitExportSection(&buf)

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

	// Phase 5: Initialize current temporary indices for code generation
	ctx.CurrentTempI32Index = ctx.TempBaseIndex
	// Calculate I64 temp base (after I32 temps + frame pointer + I64 body locals)
	tempI64BaseOffset := ctx.TempI32Count
	if ctx.FrameSize > 0 {
		tempI64BaseOffset++ // Add frame pointer
	}
	tempI64BaseOffset += ctx.countBodyLocalsByType("I64")
	ctx.CurrentTempI64Index = ctx.TempBaseIndex + tempI64BaseOffset

	return ctx
}

// addParameters adds function parameters to the LocalContext using their Symbol field
func (ctx *LocalContext) addParameters(fnNode *ASTNode) {
	// Extract parameter info from function node
	params := fnNode.Parameters

	// Add each parameter to the local context using its Symbol field
	for _, param := range params {
		if param.Symbol != nil {
			ctx.Variables = append(ctx.Variables, LocalVarInfo{
				Symbol:  param.Symbol,
				Storage: VarStorageParameterLocal,
				// Address will be assigned later in assignWASMIndices
			})
			ctx.ParameterCount++
		}
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

		case NodeCall:
			// Analyze function calls to determine temporary storage needs
			ctx.analyzeFunctionCallTemporaries(node)

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

// analyzeFunctionCallTemporaries analyzes a function call to determine temporary storage needs
func (ctx *LocalContext) analyzeFunctionCallTemporaries(node *ASTNode) {
	// Only analyze user-defined function calls that need reordering
	if len(node.Children) == 0 || node.Children[0].Kind != NodeIdent {
		return
	}

	// Skip built-in functions (print, append, etc.) - they don't need reordering
	funcName := node.Children[0].String
	if funcName == "print" || funcName == "print_bytes" || funcName == "read_line" || funcName == "append" {
		return
	}

	// Skip struct initialization calls - they use different logic
	if node.Children[0].Symbol != nil && node.Children[0].Symbol.Kind == SymbolStruct {
		return
	}

	// Count temporaries needed based on argument types
	args := node.Children[1:] // Skip function name
	for _, arg := range args {
		if arg.TypeAST == nil {
			continue // Skip if type not resolved yet
		}

		if arg.TypeAST.Kind == TypeStruct {
			// Struct arguments use tstack storage, not locals
			continue
		} else if isWASMI64Type(arg.TypeAST) {
			ctx.TempI64Count++
		} else if isWASMI32Type(arg.TypeAST) {
			ctx.TempI32Count++
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

	// Step 2: Assign i32 body locals
	for i := range ctx.Variables {
		if ctx.Variables[i].Storage == VarStorageLocal && isWASMI32Type(ctx.Variables[i].Symbol.Type) {
			ctx.Variables[i].Address = wasmIndex
			wasmIndex++
			ctx.I32LocalCount++
		}
	}

	// Step 3: Assign i32 temporary locals
	ctx.TempBaseIndex = wasmIndex
	wasmIndex += ctx.TempI32Count

	// Step 4: Assign frame pointer if needed
	if ctx.FrameSize > 0 {
		ctx.FramePointerIndex = wasmIndex
		wasmIndex++
	}

	// Step 5: Assign i64 body locals
	for i := range ctx.Variables {
		if ctx.Variables[i].Storage == VarStorageLocal && isWASMI64Type(ctx.Variables[i].Symbol.Type) {
			ctx.Variables[i].Address = wasmIndex
			wasmIndex++
			ctx.I64LocalCount++
		}
	}

	// Step 6: Assign i64 temporary locals (after i64 body locals)
	wasmIndex += ctx.TempI64Count
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

	// Add temporary locals to counts
	i32Count += localCtx.TempI32Count
	i64Count += localCtx.TempI64Count

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
	w := NewWASMWriter(buf)
	// Set frame pointer to current tstack pointer: frame_pointer = tstack_pointer
	w.global_get(0) // tstack global index (0)
	w.local_set(framePointerIndex)

	// Advance tstack pointer by frame size: tstack_pointer += frame_size
	w.global_get(0) // tstack global index (0)
	w.i32_const(int32(frameSize))
	w.i32_add()
	w.global_set(0) // tstack global index (0)
}

// EmitFrameSetupFromContext generates frame setup code using LocalContext
func EmitFrameSetupFromContext(buf *bytes.Buffer, localCtx *LocalContext) {
	w := NewWASMWriter(buf)
	// Set frame pointer to current tstack pointer: frame_pointer = tstack_pointer
	w.global_get(0) // tstack global index (0)
	w.local_set(localCtx.FramePointerIndex)

	// Advance tstack pointer by frame size: tstack_pointer += frame_size
	w.global_get(0) // tstack global index (0)
	w.i32_const(int32(localCtx.FrameSize))
	w.i32_add()
	w.global_set(0) // tstack global index (0)
}

// EmitAddressOf generates code for address-of operations
func (ctx *WASMContext) EmitAddressOf(buf *bytes.Buffer, operand *ASTNode, localCtx *LocalContext) {
	w := NewWASMWriter(buf)
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
		w.local_get(localCtx.FramePointerIndex)

		// Add variable offset
		if targetLocal.Address > 0 {
			w.i32_const(int32(targetLocal.Address))
			w.i32_add()
		}
	} else {
		// Rvalue case: &(expression)
		// Save current tstack pointer as result first
		w.global_get(0) // tstack global index (0)

		// Get address for store operation: Stack: [result_addr, store_addr_i32]
		w.global_get(0) // tstack global index (0) -> Stack: [result_addr, store_addr]

		// Evaluate expression to get value: Stack: [result_addr, store_addr_i32, value]
		ctx.EmitExpression(buf, operand, localCtx)

		// Store value at address: i64.store expects [address, value]
		w.i64_store(0x03, 0) // alignment (2^3 = 8 byte alignment), offset (0)

		// Advance tstack pointer by 8 bytes
		w.global_get(0) // tstack global index (0)
		w.i32_const(8)
		w.i32_add()
		w.global_set(0) // tstack global index (0)

		// Stack now has [result_addr] which is what we want to return
	}
}

// Lexer holds the state for tokenizing input
type Lexer struct {
	input []byte
	pos   int // current reading position in input

	// Current token state
	CurrTokenType TokenType
	CurrLiteral   string
	CurrIntValue  int64 // only meaningful when CurrTokenType == INT

	// Error collection
	Errors *ErrorCollection
}

// NewLexer creates a new lexer with the given input (must end with a 0 byte).
func NewLexer(input []byte) *Lexer {
	return &Lexer{
		input:  input,
		pos:    0,
		Errors: &ErrorCollection{},
	}
}

// AddError creates an Error with the given message and appends it to the lexer's ErrorCollection.
func (l *Lexer) AddError(message string) {
	l.Errors.Append(Error{message: message})
}

// TokenType is the type of token (identifier, operator, literal, etc.).
type TokenType string

// Definition of token types
const (
	// Special tokens
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers + literals
	UPPER_IDENT = "UPPER_IDENT" // Main, Point, MyType
	LOWER_IDENT = "LOWER_IDENT" // main, foo, _bar
	INT         = "INT"         // 12345
	STRING      = "STRING"
	CHAR        = "CHAR"

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
	NodeType     NodeKind = "NodeType"
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
	FieldName   string      // Field name for field access (s.field)
	FieldSymbol *SymbolInfo // Resolved field symbol (populated during type checking)
	// NodeFunc:
	FunctionName string      // Function name
	Parameters   []Parameter // Function parameters (unified with struct fields)
	ReturnType   *TypeNode   // Return type (nil for void)
	// NodeStruct:
	StructFields []Parameter // Struct field parameters (parsed, no AST children needed)
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
	Fields []Parameter // Field definitions (only for struct declarations)
}

// Built-in types
var (
	TypeI64         = &TypeNode{Kind: TypeBuiltin, String: "I64"}
	TypeU8          = &TypeNode{Kind: TypeBuiltin, String: "U8"}
	TypeIntegerNode = &TypeNode{Kind: TypeInteger, String: "Integer"}
	TypeBool        = &TypeNode{Kind: TypeBuiltin, String: "Boolean"}
)

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

// ResolveType recursively resolves all type references in a TypeNode
// Returns a new TypeNode with all struct references resolved to their definitions
func ResolveType(typeNode *TypeNode, st *SymbolTable) *TypeNode {
	if typeNode == nil {
		return nil
	}

	switch typeNode.Kind {
	case TypeBuiltin, TypeInteger:
		// Builtin and integer types don't need resolution
		return typeNode

	case TypeStruct:
		// Look up struct definition
		structDef := st.LookupStruct(typeNode.String)
		if structDef != nil {
			return structDef
		}
		// If struct not found, add to unresolved references for error reporting
		// Create a dummy AST node to represent this type reference
		dummyNode := &ASTNode{Kind: NodeIdent, String: typeNode.String}
		st.AddUnresolvedReference(typeNode.String, dummyNode)
		return typeNode

	case TypePointer:
		// Recursively resolve the child type
		resolvedChild := ResolveType(typeNode.Child, st)
		if resolvedChild == typeNode.Child {
			// No change needed
			return typeNode
		}
		// Create new pointer type with resolved child
		return &TypeNode{
			Kind:  TypePointer,
			Child: resolvedChild,
		}

	case TypeSlice:
		// Recursively resolve the element type
		resolvedElement := ResolveType(typeNode.Child, st)
		if resolvedElement == typeNode.Child {
			// No change needed
			return typeNode
		}
		// Create new slice type with resolved element type
		return &TypeNode{
			Kind:  TypeSlice,
			Child: resolvedElement,
		}

	default:
		// Unknown type kind, return as-is
		return typeNode
	}
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

// resolveIntegerType resolves an Integer type to a concrete type based on context
// Returns error if the integer value doesn't fit in the target type
//
// Precondition: node.Kind == NodeInteger
func (tc *TypeChecker) resolveIntegerType(node *ASTNode, targetType *TypeNode) {
	if node.Kind != NodeInteger || node.TypeAST.Kind != TypeInteger {
		panic("resolveIntegerType called with non-constant")
	}

	if !IsIntegerCompatible(node.Integer, targetType) {
		tc.AddError(fmt.Sprintf("error: cannot convert integer %d to %s", node.Integer, TypeToString(targetType)))
	}
	node.TypeAST = targetType
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

// SymbolKind represents the type of symbol
type SymbolKind int

const (
	SymbolVariable SymbolKind = iota
	SymbolFunction
	SymbolStruct
)

// SymbolInfo represents information about a declared symbol (variable, function, or struct)
type SymbolInfo struct {
	Name     string
	Type     *TypeNode
	Kind     SymbolKind
	Assigned bool // tracks if variable has been assigned a value (only relevant for variables)

	// For function symbols
	FunctionInfo *FunctionInfo

	// For struct symbols
	StructType *TypeNode
}

// FunctionInfo represents information about a declared function
type FunctionInfo struct {
	Name       string
	Parameters []Parameter // Unified with struct fields
	ReturnType *TypeNode   // nil for void functions
	WasmIndex  uint32      // WASM function index
}

// Parameter represents a unified parameter/field that can be used for both
// function parameters and struct fields, enabling shared validation logic
type Parameter struct {
	Name    string
	Type    *TypeNode
	IsNamed bool        // true for named parameters, false for positional (function params only)
	Offset  uint32      // byte offset in struct layout (struct fields only)
	Symbol  *SymbolInfo // link to symbol table entry for both function params and struct fields
}

// UnresolvedReference represents a reference to a symbol that hasn't been declared yet
type UnresolvedReference struct {
	Name    string
	ASTNode *ASTNode // Node that needs symbol reference filled in
}

// Scope represents a single scope level in the symbol hierarchy
type Scope struct {
	parent  *Scope                 // Parent scope for scope chain
	symbols map[string]*SymbolInfo // All symbols (variables, functions, structs) in this scope
}

// SliceTypeInfo holds both original and synthesized representations of a slice type
type SliceTypeInfo struct {
	Original    *TypeNode // Original slice type (for codegen)
	Synthesized *TypeNode // Synthesized struct type (for type checking)
}

// TypeTable manages type synthesis and caching
type TypeTable struct {
	sliceTypes map[string]*SliceTypeInfo // Combined registry of slice types
}

// NewTypeTable creates a new type table
func NewTypeTable() *TypeTable {
	return &TypeTable{
		sliceTypes: make(map[string]*SliceTypeInfo),
	}
}

// SymbolTable tracks variable declarations and assignments with hierarchical scoping
type SymbolTable struct {
	currentScope   *Scope
	unresolvedRefs []UnresolvedReference
	allFunctions   []FunctionInfo   // Global list for WASM index assignment
	allScopes      []*Scope         // Keep track of all scopes for traversal
	typeTable      *TypeTable       // Type synthesis and caching
	Errors         *ErrorCollection // Error collection for symbol resolution
}

// TypeChecker holds state for type checking
type TypeChecker struct {
	Errors    *ErrorCollection
	LoopDepth int        // Track loop nesting for break/continue validation
	typeTable *TypeTable // Type synthesis and caching
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
	globalScope := &Scope{
		parent:  nil,
		symbols: make(map[string]*SymbolInfo),
	}

	st := &SymbolTable{
		currentScope:   globalScope,
		unresolvedRefs: make([]UnresolvedReference, 0),
		allFunctions:   make([]FunctionInfo, 0),
		allScopes:      []*Scope{globalScope},
		typeTable:      NewTypeTable(),
		Errors:         &ErrorCollection{},
	}

	// Add built-in functions
	// print function (WASM index 0)
	printFunc := FunctionInfo{
		Name:       "print",
		Parameters: []Parameter{{Name: "value", Type: TypeI64, IsNamed: false}},
		ReturnType: nil, // void
		WasmIndex:  0,
	}
	printSymbol := &SymbolInfo{
		Name:         "print",
		Type:         nil,
		Kind:         SymbolFunction,
		FunctionInfo: &printFunc,
	}
	globalScope.symbols["print"] = printSymbol

	// print_bytes function (WASM index 1)
	printBytesFunc := FunctionInfo{
		Name:       "print_bytes",
		Parameters: []Parameter{{Name: "slice", Type: &TypeNode{Kind: TypeSlice, Child: TypeU8}, IsNamed: false}},
		ReturnType: nil, // void
		WasmIndex:  1,
	}
	printBytesSymbol := &SymbolInfo{
		Name:         "print_bytes",
		Type:         nil,
		Kind:         SymbolFunction,
		FunctionInfo: &printBytesFunc,
	}
	globalScope.symbols["print_bytes"] = printBytesSymbol

	// read_line function (WASM index 2)
	readLineFunc := FunctionInfo{
		Name:       "read_line",
		Parameters: []Parameter{},                             // No parameters
		ReturnType: &TypeNode{Kind: TypeSlice, Child: TypeU8}, // Returns U8[]
		WasmIndex:  2,
	}
	readLineSymbol := &SymbolInfo{
		Name:         "read_line",
		Type:         &TypeNode{Kind: TypeSlice, Child: TypeU8},
		Kind:         SymbolFunction,
		FunctionInfo: &readLineFunc,
	}
	globalScope.symbols["read_line"] = readLineSymbol

	// append function (generic builtin)
	appendFunc := FunctionInfo{
		Name: "append",
		Parameters: []Parameter{
			{Name: "slice_ptr", Type: nil, IsNamed: false}, // Generic type
			{Name: "value", Type: nil, IsNamed: false},     // Generic type
		},
		ReturnType: nil, // void
		WasmIndex:  0,   // Not directly called, generates specific functions
	}
	appendSymbol := &SymbolInfo{
		Name:         "append",
		Type:         nil,
		Kind:         SymbolFunction,
		FunctionInfo: &appendFunc,
	}
	globalScope.symbols["append"] = appendSymbol

	return st
}

// AddError creates an Error with the given message and appends it to the SymbolTable's ErrorCollection
func (st *SymbolTable) AddError(message string) {
	st.Errors.Append(Error{message: message})
}

// PushScope creates a new scope and makes it current
func (st *SymbolTable) PushScope() {
	newScope := &Scope{
		parent:  st.currentScope,
		symbols: make(map[string]*SymbolInfo),
	}
	st.currentScope = newScope
	st.allScopes = append(st.allScopes, newScope)
}

// PopScope removes the current scope and returns to the parent
func (st *SymbolTable) PopScope() {
	if st.currentScope.parent != nil {
		st.currentScope = st.currentScope.parent
	}
}

// DeclareVariable adds a variable declaration to the symbol table
func (st *SymbolTable) DeclareVariable(name string, varType *TypeNode) *SymbolInfo {
	// Check for duplicate declaration in current scope only
	if _, exists := st.currentScope.symbols[name]; exists {
		st.AddError(fmt.Sprintf("error: variable '%s' already declared", name))
		return nil
	}

	// Create new symbol
	symbol := &SymbolInfo{
		Name:     name,
		Type:     varType,
		Kind:     SymbolVariable,
		Assigned: false,
	}

	// Add to current scope
	st.currentScope.symbols[name] = symbol

	// Resolve any unresolved references to this symbol
	st.resolvePendingReferences(name, symbol)

	return symbol
}

// LookupVariable finds a variable in the symbol table, searching through the scope chain
func (st *SymbolTable) LookupVariable(name string) *SymbolInfo {
	// Search through scope chain from current to root
	scope := st.currentScope
	for scope != nil {
		if symbol, exists := scope.symbols[name]; exists && symbol.Kind == SymbolVariable {
			return symbol
		}
		scope = scope.parent
	}
	return nil
}

// checkDuplicateDeclaration checks if a symbol name already exists in the current scope
// Returns an error if the symbol is already declared
func (st *SymbolTable) checkDuplicateDeclaration(name string, symbolType string) bool {
	if _, exists := st.currentScope.symbols[name]; exists {
		st.AddError(fmt.Sprintf("error: %s '%s' already declared", symbolType, name))
		return true
	}
	return false
}

// validateParameterList checks for duplicate parameter names in a parameter list
// Can be used for both function parameters and struct fields
func (st *SymbolTable) validateParameterList(parameters []Parameter, contextName string) {
	seen := make(map[string]bool)
	for _, param := range parameters {
		if seen[param.Name] {
			st.AddError(fmt.Sprintf("error: duplicate parameter '%s' in %s", param.Name, contextName))
		}
		seen[param.Name] = true
	}
}

// validateCallArguments validates arguments against a parameter list for both function calls and struct initialization
// Common validation logic for parameter count, names, duplicates, and types
func validateCallArguments(
	argValues []*ASTNode, // argument expressions
	argNames []string, // parameter names (empty string for positional)
	expectedParams []Parameter, // expected parameters/fields
	callType string, // "function call" or "struct initialization"
	tc *TypeChecker, // for type checking individual arguments and error reporting
) {
	// Check for duplicate argument names first (for better error messages) - use appropriate terminology
	duplicateWord := "parameter"
	if callType == "struct initialization" {
		duplicateWord = "field"
	}
	providedNames := make(map[string]bool)
	for _, argName := range argNames {
		if argName != "" { // Skip positional arguments
			if providedNames[argName] {
				tc.AddError(fmt.Sprintf("error: %s has duplicate %s '%s'", callType, duplicateWord, argName))
				return
			}
			providedNames[argName] = true
		}
	}

	// Check parameter count - use appropriate terminology
	paramWord := "arguments"
	if callType == "struct initialization" {
		paramWord = "fields"
	}
	if len(argValues) != len(expectedParams) {
		tc.AddError(fmt.Sprintf("error: %s expects %d %s, got %d", callType, len(expectedParams), paramWord, len(argValues)))
		return
	}

	// Validate each argument
	for i, argValue := range argValues {
		var expectedParam *Parameter = nil
		argName := ""
		if i < len(argNames) {
			argName = argNames[i]
		}

		if argName != "" {
			// Named parameter - find matching expected parameter
			for j := range expectedParams {
				if expectedParams[j].Name == argName {
					expectedParam = &expectedParams[j]
					break
				}
			}
			if expectedParam == nil {
				unknownWord := "parameter"
				if callType == "struct initialization" {
					unknownWord = "field"
				}
				tc.AddError(fmt.Sprintf("error: %s has unknown %s '%s'", callType, unknownWord, argName))
				return
			}
		} else {
			// Positional parameter
			if i >= len(expectedParams) {
				tc.AddError(fmt.Sprintf("error: too many arguments for %s", callType))
				return
			}
			expectedParam = &expectedParams[i]
		}

		// Type-check the argument
		CheckExpression(argValue, tc)

		// Check that argument type matches expected parameter type
		valueType := argValue.TypeAST
		if valueType.Kind == TypeInteger {
			// Resolve Integer type based on expected parameter type
			tc.resolveIntegerType(argValue, expectedParam.Type)
		} else if !TypesEqual(valueType, expectedParam.Type) {
			tc.AddError(fmt.Sprintf("error: %s field '%s' expects type %s, got %s",
				callType, expectedParam.Name, TypeToString(expectedParam.Type), TypeToString(valueType)))
			return
		}
	}
}

// DeclareStruct adds a struct declaration to the symbol table
func (st *SymbolTable) DeclareStruct(structType *TypeNode) {
	name := structType.String

	// Check for duplicate declaration
	if st.checkDuplicateDeclaration(name, "struct") {
		return
	}

	// Validate struct fields for duplicates
	st.validateParameterList(structType.Fields, "struct "+name)

	// Create new symbol for struct
	symbol := &SymbolInfo{
		Name:       name,
		Type:       structType,
		Kind:       SymbolStruct,
		StructType: structType,
	}

	// Add to current scope
	st.currentScope.symbols[name] = symbol

	// Resolve any unresolved references to this struct
	st.resolvePendingReferences(name, symbol)
}

// LookupStruct finds a struct type by name, searching through the scope chain
func (st *SymbolTable) LookupStruct(name string) *TypeNode {
	// Search through scope chain from current to root
	scope := st.currentScope
	for scope != nil {
		if symbol, exists := scope.symbols[name]; exists && symbol.Kind == SymbolStruct {
			return symbol.StructType
		}
		scope = scope.parent
	}
	return nil
}

// DeclareFunction adds a function declaration to the symbol table
func (st *SymbolTable) DeclareFunction(name string, parameters []Parameter, returnType *TypeNode) {
	// Check for duplicate declaration
	if st.checkDuplicateDeclaration(name, "function") {
		return
	}

	// Validate function parameters for duplicates
	st.validateParameterList(parameters, "function "+name)

	// Assign WASM index (builtin functions like print start at 0, user functions follow)
	wasmIndex := uint32(3 + len(st.allFunctions)) // print=0, print_bytes=1, read_line=2

	// Create function info
	funcInfo := FunctionInfo{
		Name:       name,
		Parameters: parameters,
		ReturnType: returnType,
		WasmIndex:  wasmIndex,
	}

	// Create new symbol for function
	symbol := &SymbolInfo{
		Name:         name,
		Type:         returnType, // Function return type as the symbol type
		Kind:         SymbolFunction,
		FunctionInfo: &funcInfo,
	}

	// Add to current scope
	st.currentScope.symbols[name] = symbol

	// Add to global function list for WASM indexing
	st.allFunctions = append(st.allFunctions, funcInfo)

	// Resolve any unresolved references to this function
	st.resolvePendingReferences(name, symbol)
}

// LookupFunction finds a function by name, searching through the scope chain
func (st *SymbolTable) LookupFunction(name string) *FunctionInfo {
	// Search through scope chain from current to root
	scope := st.currentScope
	for scope != nil {
		if symbol, exists := scope.symbols[name]; exists && symbol.Kind == SymbolFunction {
			return symbol.FunctionInfo
		}
		scope = scope.parent
	}
	return nil
}

// LookupSymbol finds any symbol by name, searching through the scope chain
func (st *SymbolTable) LookupSymbol(name string) *SymbolInfo {
	// Search through scope chain from current to root
	scope := st.currentScope
	for scope != nil {
		if symbol, exists := scope.symbols[name]; exists {
			return symbol
		}
		scope = scope.parent
	}
	return nil
}

// AddUnresolvedReference adds a reference that couldn't be resolved immediately
func (st *SymbolTable) AddUnresolvedReference(name string, node *ASTNode) {
	st.unresolvedRefs = append(st.unresolvedRefs, UnresolvedReference{
		Name:    name,
		ASTNode: node,
	})
}

// resolvePendingReferences resolves any unresolved references to the given symbol
func (st *SymbolTable) resolvePendingReferences(name string, symbol *SymbolInfo) {
	// Find and resolve all matching unresolved references
	remainingRefs := make([]UnresolvedReference, 0)
	for _, ref := range st.unresolvedRefs {
		if ref.Name == name {
			// Resolve this reference
			ref.ASTNode.Symbol = symbol
		} else {
			// Keep this unresolved reference
			remainingRefs = append(remainingRefs, ref)
		}
	}
	st.unresolvedRefs = remainingRefs
}

// ReportUnresolvedSymbols returns errors for any symbols that remain unresolved
func (st *SymbolTable) ReportUnresolvedSymbols() {
	for _, ref := range st.unresolvedRefs {
		st.AddError(fmt.Sprintf("error: undefined symbol '%s'", ref.Name))
	}
}

// GetAllVariables returns all variable symbols from all scopes
func (st *SymbolTable) GetAllVariables() []*SymbolInfo {
	var variables []*SymbolInfo
	// Collect variables from all scopes
	for _, scope := range st.allScopes {
		for _, symbol := range scope.symbols {
			if symbol.Kind == SymbolVariable {
				variables = append(variables, symbol)
			}
		}
	}
	return variables
}

// GetAllStructs returns all struct symbols from all scopes
func (st *SymbolTable) GetAllStructs() []*TypeNode {
	var structs []*TypeNode
	// Collect structs from all scopes
	for _, scope := range st.allScopes {
		for _, symbol := range scope.symbols {
			if symbol.Kind == SymbolStruct {
				structs = append(structs, symbol.StructType)
			}
		}
	}
	return structs
}

// GetAllFunctions returns all function definitions (for WASM compatibility)
func (st *SymbolTable) GetAllFunctions() []FunctionInfo {
	return st.allFunctions
}

// collectSymbolsByKind recursively collects symbols of a specific kind from a scope and all its children
func (st *SymbolTable) collectSymbolsByKind(scope *Scope, kind SymbolKind, result *[]*SymbolInfo) {
	if scope == nil {
		return
	}

	// Collect from current scope
	for _, symbol := range scope.symbols {
		if symbol.Kind == kind {
			*result = append(*result, symbol)
		}
	}

	// For now, we don't have child scope tracking, so we can't traverse child scopes
	// This is a limitation of our current implementation - we only have parent pointers
	// For the current use case, this should be sufficient since most of these symbols
	// will be at the global scope anyway
}

// ConvertStructASTToType converts a struct AST node to a TypeNode with calculated field offsets
func ConvertStructASTToType(structAST *ASTNode) *TypeNode {
	if structAST.Kind != NodeStruct {
		panic("Expected NodeStruct")
	}

	structName := structAST.String
	var fields []Parameter
	var currentOffset uint32 = 0

	// Use parsed field information directly from AST metadata (no AST children processing needed)
	for _, field := range structAST.StructFields {
		fieldSize := GetTypeSize(field.Type)

		fields = append(fields, Parameter{
			Name:    field.Name,
			Type:    field.Type,
			IsNamed: false, // Struct fields don't have named/positional distinction
			Offset:  currentOffset,
			Symbol:  nil, // Will be populated during symbol table building
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
func synthesizeSliceStruct(sliceType *TypeNode, typeTable *TypeTable) *TypeNode {
	if sliceType.Kind != TypeSlice {
		panic("Expected TypeSlice")
	}

	elementType := sliceType.Child
	sliceName := TypeToString(sliceType)

	// Check if we already have a synthesized struct for this slice type
	if sliceInfo, exists := typeTable.sliceTypes[sliceName]; exists {
		return sliceInfo.Synthesized
	}

	// Create the internal struct with symbols
	fields := createSliceFieldsWithSymbols(elementType, nil)

	synthesized := &TypeNode{
		Kind:   TypeStruct,
		String: sliceName,
		Fields: fields,
	}

	// Cache the synthesized struct to ensure symbol consistency
	typeTable.sliceTypes[sliceName] = &SliceTypeInfo{
		Original:    sliceType,
		Synthesized: synthesized,
	}

	return synthesized
}

// transformSlicesToStructs converts all slice types to struct types throughout the AST and symbol table
func transformSlicesToStructs(ast *ASTNode, symbolTable *SymbolTable) {
	// Transform types in symbol table
	for _, symbol := range symbolTable.GetAllVariables() {
		transformTypeNodeSlices(&symbol.Type, symbolTable)
	}
	for _, structDef := range symbolTable.GetAllStructs() {
		for i := range structDef.Fields {
			transformTypeNodeSlices(&structDef.Fields[i].Type, symbolTable)
		}
	}
	for _, funcDef := range symbolTable.GetAllFunctions() {
		transformTypeNodeSlices(&funcDef.ReturnType, symbolTable)
		for i := range funcDef.Parameters {
			transformTypeNodeSlices(&funcDef.Parameters[i].Type, symbolTable)
		}
	}

	// Transform TypeAST nodes in the AST
	transformASTSlices(ast, symbolTable)
}

// transformTypeNodeSlices recursively converts slice types to struct types in a TypeNode tree
func transformTypeNodeSlices(typeNode **TypeNode, symbolTable *SymbolTable) {
	if *typeNode == nil {
		return
	}

	switch (*typeNode).Kind {
	case TypeSlice:
		// First, recursively transform the child type
		transformTypeNodeSlices(&(*typeNode).Child, symbolTable)

		// Use synthesizeSliceStruct to convert slice to struct
		*typeNode = synthesizeSliceStruct(*typeNode, symbolTable.typeTable)
	case TypePointer:
		transformTypeNodeSlices(&(*typeNode).Child, symbolTable)
	case TypeStruct:
		for i := range (*typeNode).Fields {
			transformTypeNodeSlices(&(*typeNode).Fields[i].Type, symbolTable)
		}
	}
}

// createSliceFieldsWithSymbols creates slice struct fields with proper symbol table entries
func createSliceFieldsWithSymbols(elementType *TypeNode, symbolTable *SymbolTable) []Parameter {
	// Create symbols for the slice fields (not added to symbol table, just standalone symbols)
	itemsSymbol := &SymbolInfo{
		Name:     "items",
		Type:     &TypeNode{Kind: TypePointer, Child: elementType},
		Assigned: true,
	}

	lengthSymbol := &SymbolInfo{
		Name:     "length",
		Type:     TypeI64,
		Assigned: true,
	}

	// Create the field parameters with symbols
	fields := []Parameter{
		{
			Name:   "items",
			Type:   &TypeNode{Kind: TypePointer, Child: elementType},
			Offset: 0,
			Symbol: itemsSymbol,
		},
		{
			Name:   "length",
			Type:   TypeI64,
			Offset: 8, // pointer is 8 bytes
			Symbol: lengthSymbol,
		},
	}

	return fields
}

// transformASTSlices recursively converts slice types to struct types in TypeAST fields
func transformASTSlices(node *ASTNode, symbolTable *SymbolTable) {
	if node == nil {
		return
	}

	// Transform TypeAST if present
	if node.TypeAST != nil {
		transformTypeNodeSlices(&node.TypeAST, symbolTable)
	}

	// Recursively transform children
	for _, child := range node.Children {
		transformASTSlices(child, symbolTable)
	}
}

// collectSliceTypes traverses the AST to find all slice types used
func collectSliceTypes(node *ASTNode, typeTable *TypeTable) {
	if node == nil {
		return
	}

	// Collect slice types from the node's type
	if node.TypeAST != nil {
		collectSliceTypesFromType(node.TypeAST, typeTable)
	}

	// Recursively process children
	for _, child := range node.Children {
		collectSliceTypes(child, typeTable)
	}
}

// collectSliceTypesFromType recursively collects slice types from a type node
func collectSliceTypesFromType(typeNode *TypeNode, typeTable *TypeTable) {
	if typeNode == nil {
		return
	}

	switch typeNode.Kind {
	case TypeSlice:
		synthesizeSliceStruct(typeNode, typeTable)
		// Also collect from the element type
		collectSliceTypesFromType(typeNode.Child, typeTable)
	case TypePointer:
		collectSliceTypesFromType(typeNode.Child, typeTable)
	case TypeStruct:
		for _, field := range typeNode.Fields {
			collectSliceTypesFromType(field.Type, typeTable)
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
	if node.Kind == NodeFunc {
		for _, stmt := range node.Children {
			collectStringsFromNode(stmt, stringMap, address)
		}
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
func (ctx *WASMContext) findStringAddress(content string) uint32 {
	if ctx.StringAddresses == nil {
		panic("String addresses not initialized during compilation")
	}
	address, exists := ctx.StringAddresses[content]
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
	parameters := []Parameter{
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
func generateAllAppendFunctions(typeTable *TypeTable) []*ASTNode {
	var generatedFunctions []*ASTNode
	for _, sliceInfo := range typeTable.sliceTypes {
		appendFunc := generateAppendFunction(sliceInfo.Original)
		generatedFunctions = append(generatedFunctions, appendFunc)
	}
	return generatedFunctions
}

// BuildSymbolTable traverses the AST to build a symbol table with variable declarations
// and populates Symbol references in NodeIdent nodes
func BuildSymbolTable(ast *ASTNode) *SymbolTable {
	st := NewSymbolTable()

	// Pass 1: Collect all struct and function declarations (no scoping)
	var collectGlobalDeclarations func(*ASTNode)
	collectGlobalDeclarations = func(node *ASTNode) {
		switch node.Kind {
		case NodeStruct:
			// Convert struct AST to TypeNode and declare it
			structType := ConvertStructASTToType(node)

			// Create symbol table entries for struct fields (similar to function parameters)
			st.PushScope() // Create a scope for struct fields
			for i, field := range structType.Fields {
				symbol := st.DeclareVariable(field.Name, field.Type)
				if symbol != nil {
					// Mark field as assigned (struct fields are always accessible)
					symbol.Assigned = true

					// Populate the Symbol field in the Parameter using the returned symbol
					structType.Fields[i].Symbol = symbol
				}
			}
			st.PopScope() // Close struct field scope

			st.DeclareStruct(structType)

		case NodeFunc:
			// Resolve all type references in function parameters
			for i, param := range node.Parameters {
				node.Parameters[i].Type = ResolveType(param.Type, st)
			}

			// Resolve function return type
			node.ReturnType = ResolveType(node.ReturnType, st)

			// Declare function (in global scope)
			st.DeclareFunction(node.FunctionName, node.Parameters, node.ReturnType)
		}

		// Traverse children for other declarations
		for _, child := range node.Children {
			if child != nil {
				collectGlobalDeclarations(child)
			}
		}
	}

	// Pass 2: Process variables and references with proper scoping
	var processWithScoping func(*ASTNode, bool)
	processWithScoping = func(node *ASTNode, isTopLevel bool) {
		switch node.Kind {
		case NodeStruct:
			// Skip struct processing - already handled in pass 1
			return

		case NodeVar:
			// Extract variable name and type
			varName := node.Children[0].String
			varType := node.TypeAST
			hasInitializer := len(node.Children) > 1

			// Skip variables with no type information
			if varType == nil {
				break
			}

			// Resolve type references
			resolvedVarType := ResolveType(varType, st)

			symbol := st.DeclareVariable(varName, resolvedVarType)
			if symbol != nil {
				node.Children[0].Symbol = symbol

				// Mark variable as assigned if it has an initializer or if it's a struct/slice
				if hasInitializer || resolvedVarType.Kind == TypeStruct || resolvedVarType.Kind == TypeSlice {
					symbol.Assigned = true
				}
			}

			// Process initializer expression if present (Children[1] and beyond)
			for i := 1; i < len(node.Children); i++ {
				if node.Children[i] != nil {
					processWithScoping(node.Children[i], false)
				}
			}
			return

		case NodeFunc:
			// Function already declared in pass 1, now handle scoping
			// Push new scope for function parameters and body
			st.PushScope()

			// Declare function parameters in the function scope
			for i, param := range node.Parameters {
				symbol := st.DeclareVariable(param.Name, param.Type)
				if symbol != nil {
					// Mark parameter as assigned (since it gets its value from the call)
					symbol.Assigned = true

					// Populate the Symbol field in the Parameter using the returned symbol
					node.Parameters[i].Symbol = symbol
				}
			}

			// Process function body with parameters in scope
			// Push an additional scope for the function body to allow variable shadowing
			st.PushScope()
			for _, stmt := range node.Children {
				processWithScoping(stmt, false)
			}
			st.PopScope()

			// Pop scope when done with function
			st.PopScope()
			return

		case NodeBlock:
			// Only push new scope for nested blocks, not the top-level program block
			if !isTopLevel {
				st.PushScope()
			}
			// Process children in the appropriate scope
			for _, child := range node.Children {
				if child != nil {
					processWithScoping(child, false)
				}
			}
			// Only pop scope if we pushed one
			if !isTopLevel {
				st.PopScope()
			}
			return

		case NodeIdent:
			// Handle identifier references
			varName := node.String
			// Try to resolve immediately
			symbol := st.LookupSymbol(varName)
			if symbol != nil {
				node.Symbol = symbol
			} else {
				// Add as unresolved reference for later resolution
				st.AddUnresolvedReference(varName, node)
			}
		}

		// Traverse children for other node types
		for _, child := range node.Children {
			if child != nil {
				processWithScoping(child, false)
			}
		}
	}

	// Execute both passes
	collectGlobalDeclarations(ast)
	processWithScoping(ast, true)

	// Pass 1.5: resolve struct types in variable declarations now that all structs are declared
	var resolveVariableStructTypes func(*ASTNode)
	resolveVariableStructTypes = func(node *ASTNode) {
		if node.Kind == NodeVar && node.TypeAST != nil {
			// Resolve the entire type tree using ResolveType (second pass)
			resolvedType := ResolveType(node.TypeAST, st)
			if resolvedType != node.TypeAST {
				node.TypeAST = resolvedType
				// Update the symbol's type as well
				if node.Children[0].Symbol != nil {
					node.Children[0].Symbol.Type = resolvedType
					node.Children[0].TypeAST = resolvedType
				}
			}
		} else if node.Kind == NodeType && node.ReturnType != nil {
			// Resolve NodeType's type reference to get complete struct information
			resolvedType := ResolveType(node.ReturnType, st)
			if resolvedType != node.ReturnType {
				node.ReturnType = resolvedType
			}
		}

		// Traverse children
		for _, child := range node.Children {
			if child != nil {
				resolveVariableStructTypes(child)
			}
		}
	}
	resolveVariableStructTypes(ast)

	// Pass 1.6: resolve struct field types now that all structs are declared
	for _, structType := range st.GetAllStructs() {
		for i, field := range structType.Fields {
			// Resolve the entire field type tree using ResolveType (second pass)
			resolvedFieldType := ResolveType(field.Type, st)
			if resolvedFieldType != field.Type {
				structType.Fields[i].Type = resolvedFieldType
			}
		}
	}

	// Pass 1.6: recalculate field offsets now that all field types are resolved
	for _, structType := range st.GetAllStructs() {
		var currentOffset uint32 = 0
		for i, field := range structType.Fields {
			// Update the field offset
			structType.Fields[i].Offset = currentOffset

			// Calculate field size with resolved types
			fieldSize := GetTypeSize(field.Type)
			currentOffset += uint32(fieldSize)
		}
	}

	// Pass 3: resolve function calls and struct initialization calls
	var resolveFunctionCalls func(*ASTNode)
	resolveFunctionCalls = func(node *ASTNode) {
		if node.Kind == NodeCall {
			// Get function name from first child
			if len(node.Children) > 0 && node.Children[0].Kind == NodeIdent {
				funcName := node.Children[0].String

				if isUpperCase(funcName) {
					// Look for struct symbol
					structSymbol := st.LookupSymbol(funcName)
					if structSymbol != nil && structSymbol.Kind == SymbolStruct {
						// Set the symbol reference for the struct constructor call
						node.Children[0].Symbol = structSymbol
					}
				} else {
					// Look for function symbol
					functionSymbol := st.LookupSymbol(funcName)
					if functionSymbol != nil && functionSymbol.Kind == SymbolFunction {
						// Set the symbol reference for the function call
						node.Children[0].Symbol = functionSymbol
					}
				}
				// Note: We don't panic on missing functions/structs here since that's handled during type checking
			}
		}

		// Traverse children
		for _, child := range node.Children {
			if child != nil {
				resolveFunctionCalls(child)
			}
		}
	}

	resolveFunctionCalls(ast)

	// Report any unresolved symbols as errors
	st.ReportUnresolvedSymbols()

	return st
}

// NewTypeChecker creates a new type checker with the given type table
func NewTypeChecker(typeTable *TypeTable) *TypeChecker {
	return &TypeChecker{
		Errors:    &ErrorCollection{},
		typeTable: typeTable,
	}
}

// AddError creates an Error with the given message and appends it to the TypeChecker's ErrorCollection.
func (tc *TypeChecker) AddError(message string) {
	tc.Errors.Append(Error{message: message})
}

// CheckProgram performs type checking on the entire AST
func CheckProgram(ast *ASTNode, typeTable *TypeTable) *ErrorCollection {

	tc := NewTypeChecker(typeTable)

	CheckStatement(ast, tc)

	// Return the error collection (may be empty)
	return tc.Errors
}

// CheckStatement validates a statement node
func CheckStatement(stmt *ASTNode, tc *TypeChecker) {

	switch stmt.Kind {
	case NodeVar:
		// Variable declaration - validate type is provided
		varType := stmt.TypeAST
		if varType == nil {
			tc.AddError("error: variable declaration missing type")
			return
		}

		// If there's an initialization expression, type-check it
		if len(stmt.Children) > 1 {
			initExpr := stmt.Children[1]
			CheckExpression(initExpr, tc)

			// Ensure initialization expression type matches variable type or allow implicit conversion
			initType := initExpr.TypeAST
			if initType != nil && !TypesEqual(varType, initType) {
				// Try to resolve Integer type to match variable type
				if initType.Kind == TypeInteger {
					tc.resolveIntegerType(initExpr, varType)
				} else {
					tc.AddError(fmt.Sprintf("error: cannot initialize variable of type %s with value of type %s",
						TypeToString(varType), TypeToString(initType)))
				}
			}
		}

		// Note: We allow unsupported types but only type-check supported ones
		// Unsupported types are simply ignored during WASM generation

	case NodeBlock:
		// Check all statements in the block
		for _, child := range stmt.Children {
			CheckStatement(child, tc)
		}

	case NodeBinary:
		// Check if this is an assignment statement
		if stmt.Op == "=" {
			CheckAssignment(stmt.Children[0], stmt.Children[1], tc)
		} else {
			// Regular expression statement
			CheckExpression(stmt, tc)
		}

	case NodeCall, NodeIdent, NodeInteger, NodeDot, NodeUnary:
		// Expression statement
		CheckExpression(stmt, tc)

	case NodeReturn:
		// TODO: Implement return type checking in the future
		if len(stmt.Children) > 0 {
			CheckExpression(stmt.Children[0], tc)

			// For now, resolve Integer types to I64 in return statements
			// TODO: Use actual function return type when proper return type checking is implemented
			returnValueType := stmt.Children[0].TypeAST
			if returnValueType != nil && returnValueType.Kind == TypeInteger {
				tc.resolveIntegerType(stmt.Children[0], TypeI64)
			}
		}

	case NodeFunc:
		// Function declaration - check the function body
		for _, stmt := range stmt.Children {
			CheckStatement(stmt, tc)
		}

	case NodeIf:
		// If statement type checking
		// Structure: [condition, then_block, condition2?, else_block2?, ...]

		// Check condition (must be Boolean)
		CheckExpression(stmt.Children[0], tc)
		condType := stmt.Children[0].TypeAST
		if condType != nil && !TypesEqual(condType, TypeBool) {
			tc.AddError(fmt.Sprintf("error: if condition must be Boolean, got %s", TypeToString(condType)))
		}

		// Check then block
		CheckStatement(stmt.Children[1], tc)

		// Check else/else-if clauses
		i := 2
		for i < len(stmt.Children) {
			// Check condition (if not nil)
			if stmt.Children[i] != nil {
				// else-if condition
				CheckExpression(stmt.Children[i], tc)
				condType := stmt.Children[i].TypeAST
				if condType != nil && !TypesEqual(condType, TypeBool) {
					tc.AddError(fmt.Sprintf("error: else-if condition must be Boolean, got %s", TypeToString(condType)))
				}
			}

			// Check block
			CheckStatement(stmt.Children[i+1], tc)

			i += 2
		}

	case NodeLoop:
		// Check all statements in loop body
		tc.EnterLoop()
		for _, stmt := range stmt.Children {
			CheckStatement(stmt, tc)
		}
		tc.ExitLoop()

	case NodeBreak:
		if !tc.InLoop() {
			tc.AddError("error: break statement outside of loop")
		}

	case NodeContinue:
		if !tc.InLoop() {
			tc.AddError("error: continue statement outside of loop")
		}

	default:
		// Other statement types are valid for now
	}
}

// CheckExpression validates an expression and stores type in expr.TypeAST (legacy)
func CheckExpression(expr *ASTNode, tc *TypeChecker) {

	switch expr.Kind {
	case NodeInteger:
		expr.TypeAST = TypeIntegerNode

	case NodeBoolean:
		expr.TypeAST = TypeBool

	case NodeString:
		// String literal has type U8[] (slice of U8)
		expr.TypeAST = &TypeNode{
			Kind:  TypeSlice,
			Child: TypeU8,
		}

	case NodeIdent:
		// Variable reference - use cached symbol reference
		if expr.Symbol == nil {
			tc.AddError(fmt.Sprintf("error: variable '%s' used before declaration", expr.String))
			return
		}
		if !expr.Symbol.Assigned {
			tc.AddError(fmt.Sprintf("error: variable '%s' used before assignment", expr.String))
			return
		}
		expr.TypeAST = expr.Symbol.Type

	case NodeBinary:
		if expr.Op == "=" {
			// Assignment expression
			CheckAssignmentExpression(expr.Children[0], expr.Children[1], tc)
			// Assignment expression type is stored in the assignment expression itself
		} else {
			// Binary operation
			CheckExpression(expr.Children[0], tc)
			CheckExpression(expr.Children[1], tc)

			// Get types from the type-checked children
			leftType := expr.Children[0].TypeAST
			rightType := expr.Children[1].TypeAST

			// Resolve Integer types based on the other operand
			var resultType *TypeNode
			if leftType != nil && rightType != nil {
				if leftType.Kind == TypeInteger && rightType.Kind != TypeInteger {
					tc.resolveIntegerType(expr.Children[0], rightType)
					leftType = expr.Children[0].TypeAST
					resultType = rightType
				} else if rightType.Kind == TypeInteger && leftType.Kind != TypeInteger {
					tc.resolveIntegerType(expr.Children[1], leftType)
					rightType = expr.Children[1].TypeAST
					resultType = leftType
				} else if leftType.Kind == TypeInteger && rightType.Kind == TypeInteger {
					// Both are Integer - resolve both to I64 and use I64 as result
					tc.resolveIntegerType(expr.Children[0], TypeI64)
					tc.resolveIntegerType(expr.Children[1], TypeI64)
					leftType = TypeI64
					rightType = TypeI64
					resultType = TypeI64
				} else if !TypesEqual(leftType, rightType) {
					tc.AddError(fmt.Sprintf("error: type mismatch in binary operation: %s vs %s",
						TypeToString(leftType), TypeToString(rightType)))
					return
				} else {
					resultType = leftType
				}

				// Set result type based on operator
				switch expr.Op {
				case "==", "!=", "<", ">", "<=", ">=":
					expr.TypeAST = TypeBool // Comparison operators return Boolean
				case "&&", "||":
					// Logical operators require Boolean operands and return Boolean
					if !TypesEqual(leftType, TypeBool) {
						tc.AddError(fmt.Sprintf("error: logical operator '%s' requires Boolean left operand, got %s", expr.Op, TypeToString(leftType)))
						return
					}
					if !TypesEqual(rightType, TypeBool) {
						tc.AddError(fmt.Sprintf("error: logical operator '%s' requires Boolean right operand, got %s", expr.Op, TypeToString(rightType)))
						return
					}
					expr.TypeAST = TypeBool // Logical operators return Boolean
				case "+", "-", "*", "/", "%":
					expr.TypeAST = resultType // Arithmetic operators return operand type
				default:
					tc.AddError(fmt.Sprintf("error: unsupported binary operator '%s'", expr.Op))
					return
				}
			}
		}

	case NodeCall:
		// Function call or struct initialization validation
		if len(expr.Children) == 0 || (expr.Children[0].Kind != NodeIdent && expr.Children[0].Kind != NodeType) {
			tc.AddError("error: invalid function call")
			return
		}

		// Handle NodeType as callee (struct constructor calls)
		if expr.Children[0].Kind == NodeType {
			// This is a struct constructor call with NodeType callee
			// After symbol resolution, ReturnType should have complete field information
			if expr.Children[0].ReturnType == nil || expr.Children[0].ReturnType.Kind != TypeStruct {
				tc.AddError("error: NodeType must represent a struct type")
				return
			}

			structType := expr.Children[0].ReturnType

			// First, validate that all parameters are named (struct initialization requirement)
			for _, paramName := range expr.ParameterNames {
				if paramName == "" {
					tc.AddError("error: struct initialization requires named parameters for all fields")
					return
				}
			}

			// Use shared validation logic for arguments
			argValues := expr.Children[1:] // Skip struct name
			validateCallArguments(argValues, expr.ParameterNames, structType.Fields, "struct initialization", tc)

			// Additional validation: ensure all struct fields are provided (struct initialization requirement)
			providedFields := make(map[string]bool)
			for _, paramName := range expr.ParameterNames {
				providedFields[paramName] = true
			}
			for _, field := range structType.Fields {
				if !providedFields[field.Name] {
					tc.AddError(fmt.Sprintf("error: struct initialization missing required field '%s'", field.Name))
					return
				}
			}

			// Set the expression type to the struct type
			expr.TypeAST = structType
			return
		}

		funcName := expr.Children[0].String

		if funcName == "print" {
			// Built-in print function
			if len(expr.Children) != 2 {
				tc.AddError("error: print() function expects 1 argument")
				return
			}
			CheckExpression(expr.Children[1], tc)

			// Resolve Integer type to I64 for print function
			argType := expr.Children[1].TypeAST
			if argType != nil && argType.Kind == TypeInteger {
				tc.resolveIntegerType(expr.Children[1], TypeI64)
			}

			expr.TypeAST = TypeI64 // print returns nothing, but use I64 for now
		} else if funcName == "print_bytes" {
			// Built-in print_bytes function
			if len(expr.Children) != 2 {
				tc.AddError("error: print_bytes() function expects 1 argument")
				return
			}
			CheckExpression(expr.Children[1], tc)

			// Check that argument is a slice (U8[])
			argType := expr.Children[1].TypeAST
			if argType != nil && argType.Kind != TypeSlice {
				tc.AddError(fmt.Sprintf("error: print_bytes() expects a slice argument, got %s", TypeToString(argType)))
				return
			}

			expr.TypeAST = TypeI64 // print_bytes returns nothing, but use I64 for now
		} else if funcName == "read_line" {
			// Built-in read_line function
			if len(expr.Children) != 1 {
				tc.AddError("error: read_line() function expects no arguments")
				return
			}

			// read_line returns U8[]
			expr.TypeAST = &TypeNode{Kind: TypeSlice, Child: TypeU8}
		} else if funcName == "append" {
			// Built-in append function
			if len(expr.Children) != 3 {
				tc.AddError("error: append() function expects 2 arguments")
				return
			}

			// Check first argument (slice pointer)
			CheckExpression(expr.Children[1], tc)
			// Check second argument (value to append)
			CheckExpression(expr.Children[2], tc)

			slicePtrType := expr.Children[1].TypeAST
			valueType := expr.Children[2].TypeAST

			// First argument must be a pointer to a slice
			if slicePtrType != nil && (slicePtrType.Kind != TypePointer || slicePtrType.Child.Kind != TypeSlice) {
				tc.AddError(fmt.Sprintf("error: append() first argument must be pointer to slice, got %s", TypeToString(slicePtrType)))
				return
			}

			// Value type must match slice element type or allow implicit conversion
			elementType := slicePtrType.Child.Child
			if !TypesEqual(valueType, elementType) {
				if valueType.Kind == TypeInteger {
					tc.resolveIntegerType(expr.Children[2], elementType)
				} else {
					tc.AddError(fmt.Sprintf("error: append() value type %s does not match slice element type %s",
						TypeToString(valueType), TypeToString(elementType)))
					return
				}
			}

			expr.TypeAST = TypeI64 // append returns nothing, but use I64 for now
		} else {
			// User-defined function
			// Use the symbol-based function resolution
			if expr.Children[0].Symbol == nil || expr.Children[0].Symbol.Kind != SymbolFunction {
				tc.AddError(fmt.Sprintf("error: unknown function '%s'", funcName))
				return
			}
			function := expr.Children[0].Symbol.FunctionInfo

			// Use shared validation logic for function call arguments
			argValues := expr.Children[1:] // Skip function name
			validateCallArguments(argValues, expr.ParameterNames, function.Parameters, "function call", tc)

			// Arguments will be reordered during code generation to preserve evaluation order

			// Return function's return type (or void)
			if function.ReturnType != nil {
				expr.TypeAST = function.ReturnType
			} else {
				expr.TypeAST = TypeI64 // Void functions return I64 for now
			}
		}

	case NodeDot:
		// Field access: struct.field
		if len(expr.Children) != 1 {
			tc.AddError("error: field access expects 1 base expression")
			return
		}

		CheckExpression(expr.Children[0], tc)

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
			structType = synthesizeSliceStruct(baseType, tc.typeTable)
		} else if baseType.Kind == TypePointer && baseType.Child.Kind == TypeSlice {
			// Pointer-to-slice access (slice parameters)
			structType = synthesizeSliceStruct(baseType.Child, tc.typeTable)
		} else {
			tc.AddError(fmt.Sprintf("error: cannot access field of non-struct type %s", TypeToString(baseType)))
			return
		}

		// Find the field in the struct
		fieldName := expr.FieldName

		for _, field := range structType.Fields {
			if field.Name == fieldName {
				expr.TypeAST = field.Type
				expr.FieldSymbol = field.Symbol // Store resolved field symbol
				return
			}
		}

		tc.AddError(fmt.Sprintf("error: struct %s has no field named '%s'", structType.String, fieldName))
		return

	case NodeUnary:
		// Unary operations
		if expr.Op == "&" {
			// Address-of operator
			if len(expr.Children) != 1 {
				tc.AddError("error: address-of operator expects 1 operand")
				return
			}

			CheckExpression(expr.Children[0], tc)

			// Get operand type from the type-checked child
			operandType := expr.Children[0].TypeAST

			// Return pointer type
			expr.TypeAST = &TypeNode{Kind: TypePointer, Child: operandType}
		} else if expr.Op == "*" {
			// Dereference operator
			if len(expr.Children) != 1 {
				tc.AddError("error: dereference operator expects 1 operand")
				return
			}

			CheckExpression(expr.Children[0], tc)

			// Get operand type from the type-checked child
			operandType := expr.Children[0].TypeAST

			// Operand must be a pointer type
			if operandType.Kind != TypePointer {
				tc.AddError(fmt.Sprintf("error: cannot dereference non-pointer type %s", TypeToString(operandType)))
				return
			}

			// Return the pointed-to type
			expr.TypeAST = operandType.Child
		} else if expr.Op == "!" {
			// Logical NOT operator
			if len(expr.Children) != 1 {
				tc.AddError("error: logical NOT operator expects 1 operand")
				return
			}

			CheckExpression(expr.Children[0], tc)

			// Get operand type from the type-checked child
			operandType := expr.Children[0].TypeAST

			// Operand must be Boolean type
			if !TypesEqual(operandType, TypeBool) {
				tc.AddError(fmt.Sprintf("error: logical NOT operator requires Boolean operand, got %s", TypeToString(operandType)))
				return
			}

			// Return Boolean type
			expr.TypeAST = TypeBool
		} else {
			tc.AddError(fmt.Sprintf("error: unsupported unary operator '%s'", expr.Op))
			return
		}

	case NodeIndex:
		// Array/slice subscript operation
		if len(expr.Children) != 2 {
			tc.AddError("error: subscript operator expects 2 operands")
			return
		}

		CheckExpression(expr.Children[0], tc)
		CheckExpression(expr.Children[1], tc)

		// Get base and index types from the type-checked children
		baseType := expr.Children[0].TypeAST
		indexType := expr.Children[1].TypeAST

		// Index must be I64 or resolve Integer to I64
		if !TypesEqual(indexType, TypeI64) {
			if indexType.Kind == TypeInteger {
				tc.resolveIntegerType(expr.Children[1], TypeI64)
			} else {
				tc.AddError(fmt.Sprintf("error: slice index must be I64, got %s", TypeToString(indexType)))
				return
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
			tc.AddError(fmt.Sprintf("error: cannot subscript non-slice type %s", TypeToString(baseType)))
			return
		}

		// Return the element type of the slice
		expr.TypeAST = elementType

	default:
		tc.AddError(fmt.Sprintf("error: unsupported expression type '%s'", expr.Kind))
		return
	}
}

// CheckAssignment validates an assignment statement
func CheckAssignment(lhs, rhs *ASTNode, tc *TypeChecker) {
	// Validate RHS type first
	CheckExpression(rhs, tc)
	rhsType := rhs.TypeAST

	// Validate LHS is assignable
	var lhsType *TypeNode

	if lhs.Kind == NodeIdent {
		// Direct variable assignment - use cached symbol reference
		if lhs.Symbol == nil {
			tc.AddError(fmt.Sprintf("error: variable '%s' used before declaration", lhs.String))
			return
		}
		lhsType = lhs.Symbol.Type

		// Set the TypeAST for code generation
		lhs.TypeAST = lhsType

		// Mark variable as assigned
		lhs.Symbol.Assigned = true

	} else if lhs.Kind == NodeUnary && lhs.Op == "*" {
		// Pointer dereference assignment (e.g., ptr* = value)
		CheckExpression(lhs.Children[0], tc)
		ptrType := lhs.Children[0].TypeAST

		if ptrType != nil && ptrType.Kind != TypePointer {
			tc.AddError(fmt.Sprintf("error: cannot dereference non-pointer type %s", TypeToString(ptrType)))
			return
		}

		if ptrType != nil {
			lhsType = ptrType.Child
		}

		// Set the TypeAST for code generation
		lhs.TypeAST = lhsType

	} else if lhs.Kind == NodeDot {
		// Field assignment (e.g., s.field = value)
		CheckExpression(lhs, tc)
		lhsType = lhs.TypeAST

	} else if lhs.Kind == NodeIndex {
		// Slice index assignment (e.g., slice[0] = value)
		CheckExpression(lhs, tc)
		lhsType = lhs.TypeAST

	} else {
		tc.AddError("error: left side of assignment must be a variable, field access, or dereferenced pointer")
		return
	}

	// Ensure types match or allow implicit conversion
	if lhsType != nil && rhsType != nil && !TypesEqual(lhsType, rhsType) {
		// Try to resolve Integer type to match LHS
		if rhsType.Kind == TypeInteger {
			tc.resolveIntegerType(rhs, lhsType)
		} else {
			tc.AddError(fmt.Sprintf("error: cannot assign %s to %s",
				TypeToString(rhsType), TypeToString(lhsType)))
		}
	}
}

// CheckAssignmentExpression validates an assignment expression
func CheckAssignmentExpression(lhs, rhs *ASTNode, tc *TypeChecker) {
	CheckAssignment(lhs, rhs, tc)

	// Assignment expression returns the type of the assigned value
	// The assignment expression type should be set to the RHS type
	lhs.TypeAST = rhs.TypeAST
}

// NextToken scans the next token and stores it in the lexer's state.
// Call repeatedly until l.CurrTokenType == EOF.
func (l *Lexer) NextToken() {
	l.skipWhitespace()

	c := l.input[l.pos]
	l.CurrIntValue = 0 // reset for non-INT tokens

	if c == '=' {
		if l.input[l.pos+1] == '=' {
			l.CurrTokenType = EQ
			l.CurrLiteral = string(l.input[l.pos : l.pos+2])
			l.pos += 2
		} else {
			l.CurrTokenType = ASSIGN
			l.CurrLiteral = string(c)
			l.pos++ // inlined advance()
		}

	} else if c == '+' {
		if l.input[l.pos+1] == '+' {
			l.CurrTokenType = PLUS_PLUS
			l.CurrLiteral = "++"
			l.pos += 2
		} else {
			l.CurrTokenType = PLUS
			l.CurrLiteral = string(c)
			l.pos++
		}

	} else if c == '-' {
		nxt := l.input[l.pos+1]
		if nxt == '-' {
			l.CurrTokenType = MINUS_MINUS
			l.CurrLiteral = "--"
			l.pos += 2
		} else {
			l.CurrTokenType = MINUS
			l.CurrLiteral = string(c)
			l.pos++
		}

	} else if c == '!' {
		if l.input[l.pos+1] == '=' {
			l.CurrTokenType = NOT_EQ
			l.CurrLiteral = string(l.input[l.pos : l.pos+2])
			l.pos += 2
		} else {
			l.CurrTokenType = BANG
			l.CurrLiteral = string(c)
			l.pos++
		}

	} else if c == '/' {
		nxt := l.input[l.pos+1]
		if nxt == '/' {
			l.skipLineComment()
			l.NextToken()
			return
		} else if nxt == '*' {
			l.skipBlockComment()
			l.NextToken()
			return
		} else {
			l.CurrTokenType = SLASH
			l.CurrLiteral = string(c)
			l.pos++
		}

	} else if c == '*' {
		l.CurrTokenType = ASTERISK
		l.CurrLiteral = string(c)
		l.pos++

	} else if c == '%' {
		l.CurrTokenType = PERCENT
		l.CurrLiteral = string(c)
		l.pos++

	} else if c == '<' {
		if l.input[l.pos+1] == '=' {
			l.CurrTokenType = LE
			l.CurrLiteral = "<="
			l.pos += 2
		} else if l.input[l.pos+1] == '<' {
			l.CurrTokenType = SHL
			l.CurrLiteral = "<<"
			l.pos += 2
		} else {
			l.CurrTokenType = LT
			l.CurrLiteral = string(c)
			l.pos++
		}

	} else if c == '>' {
		if l.input[l.pos+1] == '=' {
			l.CurrTokenType = GE
			l.CurrLiteral = ">="
			l.pos += 2
		} else if l.input[l.pos+1] == '>' {
			l.CurrTokenType = SHR
			l.CurrLiteral = ">>"
			l.pos += 2
		} else {
			l.CurrTokenType = GT
			l.CurrLiteral = string(c)
			l.pos++
		}

	} else if c == '&' {
		nxt := l.input[l.pos+1]
		if nxt == '&' {
			l.CurrTokenType = AND
			l.CurrLiteral = "&&"
			l.pos += 2
		} else if nxt == '^' {
			l.CurrTokenType = AND_NOT
			l.CurrLiteral = "&^"
			l.pos += 2
		} else {
			l.CurrTokenType = BIT_AND
			l.CurrLiteral = string(c)
			l.pos++
		}

	} else if c == '|' {
		if l.input[l.pos+1] == '|' {
			l.CurrTokenType = OR
			l.CurrLiteral = "||"
			l.pos += 2
		} else {
			l.CurrTokenType = BIT_OR
			l.CurrLiteral = string(c)
			l.pos++
		}

	} else if c == '^' {
		l.CurrTokenType = XOR
		l.CurrLiteral = string(c)
		l.pos++

	} else if c == ',' {
		l.CurrTokenType = COMMA
		l.CurrLiteral = string(c)
		l.pos++

	} else if c == ';' {
		l.CurrTokenType = SEMICOLON
		l.CurrLiteral = string(c)
		l.pos++

	} else if c == ':' {
		if l.input[l.pos+1] == '=' {
			l.CurrTokenType = DECLARE
			l.CurrLiteral = ":="
			l.pos += 2
		} else {
			l.CurrTokenType = COLON
			l.CurrLiteral = string(c)
			l.pos++
		}

	} else if c == '(' {
		l.CurrTokenType = LPAREN
		l.CurrLiteral = string(c)
		l.pos++

	} else if c == ')' {
		l.CurrTokenType = RPAREN
		l.CurrLiteral = string(c)
		l.pos++

	} else if c == '{' {
		l.CurrTokenType = LBRACE
		l.CurrLiteral = string(c)
		l.pos++

	} else if c == '}' {
		l.CurrTokenType = RBRACE
		l.CurrLiteral = string(c)
		l.pos++

	} else if c == '[' {
		l.CurrTokenType = LBRACKET
		l.CurrLiteral = string(c)
		l.pos++

	} else if c == ']' {
		l.CurrTokenType = RBRACKET
		l.CurrLiteral = string(c)
		l.pos++

	} else if c == '.' {
		if l.input[l.pos+1] == '.' && l.input[l.pos+2] == '.' {
			l.CurrTokenType = ELLIPSIS
			l.CurrLiteral = "..."
			l.pos += 3
		} else {
			l.CurrTokenType = DOT
			l.CurrLiteral = string(c)
			l.pos++
		}

	} else if c == '"' {
		l.CurrTokenType = STRING
		l.CurrLiteral = l.readString()

	} else if c == '\'' {
		l.CurrTokenType = CHAR
		l.CurrLiteral = l.readCharLiteral()

	} else if c == 0 {
		l.CurrTokenType = EOF
		l.CurrLiteral = ""

	} else {
		if isLetter(c) {
			lit := l.readIdentifier()
			// keyword check
			if lit == "break" {
				l.CurrTokenType = BREAK
			} else if lit == "default" {
				l.CurrTokenType = DEFAULT
			} else if lit == "func" {
				l.CurrTokenType = FUNC
			} else if lit == "interface" {
				l.CurrTokenType = INTERFACE
			} else if lit == "select" {
				l.CurrTokenType = SELECT
			} else if lit == "case" {
				l.CurrTokenType = CASE
			} else if lit == "defer" {
				l.CurrTokenType = DEFER
			} else if lit == "go" {
				l.CurrTokenType = GO
			} else if lit == "map" {
				l.CurrTokenType = MAP
			} else if lit == "struct" {
				l.CurrTokenType = STRUCT
			} else if lit == "chan" {
				l.CurrTokenType = CHAN
			} else if lit == "else" {
				l.CurrTokenType = ELSE
			} else if lit == "goto" {
				l.CurrTokenType = GOTO
			} else if lit == "package" {
				l.CurrTokenType = PACKAGE
			} else if lit == "switch" {
				l.CurrTokenType = SWITCH
			} else if lit == "const" {
				l.CurrTokenType = CONST
			} else if lit == "fallthrough" {
				l.CurrTokenType = FALLTHROUGH
			} else if lit == "if" {
				l.CurrTokenType = IF
			} else if lit == "range" {
				l.CurrTokenType = RANGE
			} else if lit == "type" {
				l.CurrTokenType = TYPE
			} else if lit == "continue" {
				l.CurrTokenType = CONTINUE
			} else if lit == "for" {
				l.CurrTokenType = FOR
			} else if lit == "import" {
				l.CurrTokenType = IMPORT
			} else if lit == "return" {
				l.CurrTokenType = RETURN
			} else if lit == "var" {
				l.CurrTokenType = VAR
			} else if lit == "loop" {
				l.CurrTokenType = LOOP
			} else if lit == "true" {
				l.CurrTokenType = TRUE
			} else if lit == "false" {
				l.CurrTokenType = FALSE
			} else {
				// Determine token type based on first character case
				if isUpperCase(lit) {
					l.CurrTokenType = UPPER_IDENT
				} else {
					l.CurrTokenType = LOWER_IDENT
				}
			}
			l.CurrLiteral = lit

		} else if isDigit(c) {
			lit, val := l.readNumber()
			l.CurrTokenType = INT
			l.CurrLiteral = lit
			l.CurrIntValue = val

		} else {
			l.AddError("error: unexpected character '" + string(c) + "'")
			l.CurrTokenType = ILLEGAL
			l.CurrLiteral = string(c)
			l.pos++
		}
	}
}

func (l *Lexer) skipWhitespace() {
	for {
		c := l.input[l.pos]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return
		}
		l.pos++
	}
}

func (l *Lexer) skipLineComment() {
	for l.input[l.pos] != '\n' && l.input[l.pos] != 0 {
		l.pos++
	}
	if l.input[l.pos] == '\n' {
		l.pos++
	}
}

func (l *Lexer) skipBlockComment() {
	l.pos += 2 // skip /*
	for l.input[l.pos] != 0 && !(l.input[l.pos] == '*' && l.input[l.pos+1] == '/') {
		l.pos++
	}
	if l.input[l.pos] == '*' && l.input[l.pos+1] == '/' {
		l.pos += 2 // skip */
	}
}

// isUpperCase checks if the first character of a name is uppercase
func isUpperCase(name string) bool {
	if len(name) == 0 {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}

func (l *Lexer) readIdentifier() string {
	start := l.pos
	for isLetter(l.input[l.pos]) || isDigit(l.input[l.pos]) {
		l.pos++
	}
	return string(l.input[start:l.pos])
}

func (l *Lexer) readNumber() (string, int64) {
	start := l.pos
	var val int64
	for isDigit(l.input[l.pos]) {
		digit := int64(l.input[l.pos] - '0')
		val = val*10 + digit
		l.pos++
	}
	return string(l.input[start:l.pos]), val
}

func (l *Lexer) readString() string {
	l.pos++ // skip opening "
	var result []byte
	nonASCIIErrorReported := false

	for l.input[l.pos] != '"' && l.input[l.pos] != 0 {
		if l.input[l.pos] == '\\' && l.pos+1 < len(l.input) {
			// Handle escape sequences
			nextChar := l.input[l.pos+1]
			switch nextChar {
			case 'n':
				result = append(result, '\n')
				l.pos += 2 // skip both \ and n
				continue
			case '\\':
				result = append(result, '\\')
				l.pos += 2 // skip both backslashes
				continue
			case '"':
				result = append(result, '"')
				l.pos += 2 // skip both \ and "
				continue
			default:
				// Unknown escape sequence - report error
				l.AddError(fmt.Sprintf("error: unsupported escape sequence '\\%c' in string literal", nextChar))
				// Skip the backslash and invalid escape character
				l.pos += 2
				continue
			}
		}

		// Validate ASCII characters only
		if l.input[l.pos] > 127 && !nonASCIIErrorReported {
			l.AddError("error: non-ASCII characters are not supported in string literals")
			nonASCIIErrorReported = true
			// Continue parsing to recover
		}

		result = append(result, l.input[l.pos])
		l.pos++
	}

	if l.input[l.pos] == 0 {
		l.AddError("error: unterminated string literal")
		return string(result)
	}

	l.pos++ // skip closing "
	return string(result)
}

func (l *Lexer) readCharLiteral() string {
	start := l.pos
	l.pos++ // Skip first '.

	if l.input[l.pos] == 0 {
		l.AddError("error: unterminated character literal")
		return string(l.input[start:l.pos])
	}

	if l.input[l.pos] == '\\' {
		l.pos++
		if l.input[l.pos] == 0 {
			l.AddError("error: unterminated character literal")
			return string(l.input[start:l.pos])
		}
	}
	l.pos++ // Skip the character.

	if l.input[l.pos] != '\'' {
		l.AddError("error: unterminated character literal")
		return string(l.input[start:l.pos])
	}
	l.pos++ // Skip last '.
	lit := string(l.input[start:l.pos])
	return lit
}

// SkipToken advances past the current token, asserting it matches the expected type.
func (l *Lexer) SkipToken(expectedType TokenType) {
	if l.CurrTokenType != expectedType {
		panic("Expected token " + string(expectedType) + " but got " + string(l.CurrTokenType))
	}
	l.NextToken()
}

// PeekToken returns the next token type without advancing the lexer.
func (l *Lexer) PeekToken() TokenType {
	savedPos := l.pos
	savedTokenType := l.CurrTokenType
	savedLiteral := l.CurrLiteral
	savedIntValue := l.CurrIntValue

	l.NextToken()
	nextType := l.CurrTokenType

	// Restore state
	l.pos = savedPos
	l.CurrTokenType = savedTokenType
	l.CurrLiteral = savedLiteral
	l.CurrIntValue = savedIntValue

	return nextType
}

func isLetter(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || c == '_'
}

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
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
		result := "(struct \"" + node.String + "\" ["
		for i, field := range node.StructFields {
			if i > 0 {
				result += " "
			}
			result += "(field \"" + field.Name + "\" \"" + TypeToString(field.Type) + "\")"
		}
		result += "])"
		return result
	case NodeDot:
		base := ToSExpr(node.Children[0])
		return "(dot " + base + " \"" + node.FieldName + "\")"
	case NodeType:
		return "(type \"" + TypeToString(node.ReturnType) + "\")"
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
		result += " (block"
		for _, stmt := range node.Children {
			result += " " + ToSExpr(stmt)
		}
		result += "))"
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

// precedence returns the precedence level for a given token type
func precedence(tokenType TokenType) int {
	switch tokenType {
	case ASSIGN:
		return 1 // assignment has very low precedence
	case OR: // logical OR has low precedence
		return 2
	case AND: // logical AND has higher precedence than OR
		return 3
	case EQ, NOT_EQ, LT, GT, LE, GE:
		return 4
	case PLUS, MINUS:
		return 5
	case ASTERISK, SLASH, PERCENT:
		return 6
	case LBRACKET, LPAREN: // subscript and function call operators
		return 7 // highest precedence (postfix)
	case BIT_AND: // postfix address-of operator
		return 7 // highest precedence (postfix)
	case DOT: // field access operator
		return 7 // highest precedence (postfix)
	default:
		return 0 // not an operator
	}
}

// isOperator returns true if the token is a binary operator
func isOperator(tokenType TokenType) bool {
	return precedence(tokenType) > 0
}

// ParseExpression parses an expression and returns an AST node
func ParseExpression(l *Lexer) *ASTNode {
	return parseExpressionWithPrecedence(l, 0)
}

// parseExpressionWithPrecedence implements precedence climbing
func parseExpressionWithPrecedence(l *Lexer, minPrec int) *ASTNode {
	var left *ASTNode

	// Handle unary operators first
	if l.CurrTokenType == BANG {
		l.SkipToken(BANG)                              // consume '!'
		operand := parseExpressionWithPrecedence(l, 5) // Same as multiplication, less than postfix
		left = &ASTNode{
			Kind:     NodeUnary,
			Op:       "!",
			Children: []*ASTNode{operand},
		}
	} else {
		left = parsePrimary(l)
	}

	for {
		if !isOperator(l.CurrTokenType) || precedence(l.CurrTokenType) < minPrec {
			break
		}

		if l.CurrTokenType == LBRACKET {
			// Handle subscript operator
			l.SkipToken(LBRACKET)
			index := parseExpressionWithPrecedence(l, 0)
			if l.CurrTokenType == RBRACKET {
				l.SkipToken(RBRACKET)
			}
			left = &ASTNode{
				Kind:     NodeIndex,
				Children: []*ASTNode{left, index},
			}
		} else if l.CurrTokenType == LPAREN {
			// Handle function call operator
			l.SkipToken(LPAREN)

			var args []*ASTNode
			var paramNames []string

			for l.CurrTokenType != RPAREN && l.CurrTokenType != EOF {
				var paramName string

				// Check for named parameter (identifier followed by colon)
				if l.CurrTokenType == LOWER_IDENT {
					// Look ahead to see if there's a colon after the identifier
					identName := l.CurrLiteral
					if l.PeekToken() == COLON {
						// This is a named parameter: name: value
						paramName = identName
						l.SkipToken(LOWER_IDENT)
						l.SkipToken(COLON)
					} else {
						paramName = ""
					}
				} else {
					paramName = ""
				}

				paramNames = append(paramNames, paramName)
				expr := parseExpressionWithPrecedence(l, 0)
				args = append(args, expr)

				if l.CurrTokenType == COMMA {
					l.SkipToken(COMMA)
				} else if l.CurrTokenType != RPAREN {
					break
				}
			}

			if l.CurrTokenType == RPAREN {
				l.SkipToken(RPAREN)
			}

			left = &ASTNode{
				Kind:           NodeCall,
				Children:       append([]*ASTNode{left}, args...),
				ParameterNames: paramNames,
			}
		} else if l.CurrTokenType == ASTERISK && minPrec <= 7 {
			// Handle postfix dereference operator: expr*
			// Check if next token suggests this should be binary instead
			nextToken := l.PeekToken()
			if nextToken == LOWER_IDENT || nextToken == UPPER_IDENT || nextToken == INT || nextToken == LPAREN || nextToken == LBRACKET {
				// Treat as binary multiplication - fall through to binary operator handling
				op := l.CurrLiteral
				prec := precedence(l.CurrTokenType)
				l.NextToken()
				right := parseExpressionWithPrecedence(l, prec+1) // left-associative
				left = &ASTNode{
					Kind:     NodeBinary,
					Op:       op,
					Children: []*ASTNode{left, right},
				}
			} else {
				// Treat as postfix dereference
				l.SkipToken(ASTERISK)
				left = &ASTNode{
					Kind:     NodeUnary,
					Op:       "*",
					Children: []*ASTNode{left},
				}
			}
		} else if l.CurrTokenType == BIT_AND {
			// Handle postfix address-of operator: expr&
			l.SkipToken(BIT_AND)
			left = &ASTNode{
				Kind:     NodeUnary,
				Op:       "&",
				Children: []*ASTNode{left},
			}
		} else if l.CurrTokenType == DOT {
			// Handle field access operator: expr.field
			l.SkipToken(DOT)
			if l.CurrTokenType != LOWER_IDENT {
				break // error - expecting field name
			}
			fieldName := l.CurrLiteral
			l.SkipToken(LOWER_IDENT)
			left = &ASTNode{
				Kind:      NodeDot,
				FieldName: fieldName,
				Children:  []*ASTNode{left},
			}
		} else {
			// Handle binary operators
			op := l.CurrLiteral
			prec := precedence(l.CurrTokenType)
			l.NextToken()

			// For assignment (right-associative), use prec instead of prec + 1
			// For other operators (left-associative), use prec + 1
			var right *ASTNode
			if op == "=" {
				right = parseExpressionWithPrecedence(l, prec) // right-associative
			} else {
				right = parseExpressionWithPrecedence(l, prec+1) // left-associative
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
func parsePrimary(l *Lexer) *ASTNode {
	switch l.CurrTokenType {
	case INT:
		node := &ASTNode{
			Kind:    NodeInteger,
			Integer: l.CurrIntValue,
		}
		l.SkipToken(INT)
		return node

	case TRUE:
		node := &ASTNode{
			Kind:    NodeBoolean,
			Boolean: true,
		}
		l.SkipToken(TRUE)
		return node

	case FALSE:
		node := &ASTNode{
			Kind:    NodeBoolean,
			Boolean: false,
		}
		l.SkipToken(FALSE)
		return node

	case STRING:
		node := &ASTNode{
			Kind:   NodeString,
			String: l.CurrLiteral,
		}
		l.SkipToken(STRING)
		return node

	case LOWER_IDENT:
		node := &ASTNode{
			Kind:   NodeIdent,
			String: l.CurrLiteral,
		}
		l.SkipToken(LOWER_IDENT)
		return node

	case UPPER_IDENT:
		// Type name in expression (struct constructor call, etc.)
		typeAST := parseTypeExpression(l)
		node := &ASTNode{
			Kind:       NodeType,
			ReturnType: typeAST,
		}
		return node

	case LPAREN:
		l.SkipToken(LPAREN) // consume '('
		expr := parseExpressionWithPrecedence(l, 0)
		if l.CurrTokenType == RPAREN {
			l.SkipToken(RPAREN)
		} else {
			l.AddError("expected ')' after expression")
		}
		return expr

	default:
		l.AddError("unexpected token '" + string(l.CurrTokenType) + "' in expression")
		// Return empty node for error case
		return &ASTNode{}
	}
}

// parseParameterList parses a parameter list in parentheses and returns a slice of Parameters
// Used by both function parameter parsing and struct field parsing for consistency
// Parameters:
//   - endToken: the token that ends the parameter list (RPAREN for both cases)
//   - allowPositional: whether to allow positional parameters with "_" prefix (true for functions, false for structs)
func parseParameterList(l *Lexer, endToken TokenType, allowPositional bool) []Parameter {
	var parameters []Parameter

	for l.CurrTokenType != endToken && l.CurrTokenType != EOF {
		// Parse parameter: _ name: Type (positional) or name: Type (named)
		isPositional := false
		var paramName string

		if allowPositional && l.CurrTokenType == LOWER_IDENT && l.CurrLiteral == "_" {
			// Positional parameter: _ name: Type (functions only)
			isPositional = true
			l.SkipToken(LOWER_IDENT) // skip the "_"
			if l.CurrTokenType != LOWER_IDENT {
				// Return empty slice on error - caller should check
				return []Parameter{}
			}
			paramName = l.CurrLiteral
			l.SkipToken(LOWER_IDENT)
		} else if l.CurrTokenType == LOWER_IDENT {
			// Named parameter: name: Type
			paramName = l.CurrLiteral
			l.SkipToken(LOWER_IDENT)
		} else {
			// Error: unexpected token where parameter name expected
			// Advance to try to recover and prevent infinite loops
			if l.CurrTokenType != endToken && l.CurrTokenType != EOF {
				l.NextToken()
			}
			// Return empty slice on error - caller should check
			return []Parameter{}
		}

		// Parse colon
		if l.CurrTokenType != COLON {
			// Return empty slice on error - caller should check
			return []Parameter{}
		}
		l.SkipToken(COLON)

		// Parse parameter type
		paramType := parseTypeExpression(l)
		if paramType == nil {
			// Return empty slice on error - caller should check
			return []Parameter{}
		}

		parameters = append(parameters, Parameter{
			Name:    paramName,
			Type:    paramType,
			IsNamed: !isPositional,
			Offset:  0,   // Will be set later for struct fields
			Symbol:  nil, // Will be set later for function parameters
		})

		// Skip optional comma
		if l.CurrTokenType == COMMA {
			l.SkipToken(COMMA)
		}
	}

	return parameters
}

// parseTypeExpression parses a type expression and returns a TypeNode
func parseTypeExpression(l *Lexer) *TypeNode {
	// All types must start with uppercase (built-ins and user-defined)
	if l.CurrTokenType != UPPER_IDENT {
		return nil
	}

	// Parse base type
	baseTypeName := l.CurrLiteral
	l.SkipToken(UPPER_IDENT)

	baseType := getBuiltinType(baseTypeName)
	if baseType == nil {
		// Unknown types assumed to be struct types
		baseType = &TypeNode{Kind: TypeStruct, String: baseTypeName}
	}

	// Handle slice and pointer suffixes
	resultType := baseType

	// Handle slice suffix: Type[]
	if l.CurrTokenType == LBRACKET {
		l.SkipToken(LBRACKET)
		if l.CurrTokenType == RBRACKET {
			l.SkipToken(RBRACKET)
			resultType = &TypeNode{
				Kind:  TypeSlice,
				Child: resultType,
			}
		} else {
			l.AddError("expected ']' after '['")
			return nil
		}
	}

	// Handle pointer suffixes: Type*
	for l.CurrTokenType == ASTERISK {
		l.SkipToken(ASTERISK)
		resultType = &TypeNode{
			Kind:  TypePointer,
			Child: resultType,
		}
	}

	return resultType
}

// parseBlockStatements parses a block of statements between braces and returns a Block AST node
func parseBlockStatements(l *Lexer) *ASTNode {
	if l.CurrTokenType != LBRACE {
		return &ASTNode{} // error
	}
	l.SkipToken(LBRACE)

	var statements []*ASTNode
	for l.CurrTokenType != RBRACE && l.CurrTokenType != EOF {
		stmt := ParseStatement(l)
		statements = append(statements, stmt)
	}

	if l.CurrTokenType == RBRACE {
		l.SkipToken(RBRACE)
	}

	return &ASTNode{
		Kind:     NodeBlock,
		Children: statements,
	}
}

// ParseStatement parses a statement and returns an AST node
func ParseStatement(l *Lexer) *ASTNode {
	switch l.CurrTokenType {
	case STRUCT:
		l.SkipToken(STRUCT)
		if l.CurrTokenType != UPPER_IDENT {
			l.AddError("expected struct name (must start with uppercase letter)")
			return &ASTNode{} // error
		}
		structName := l.CurrLiteral
		l.SkipToken(UPPER_IDENT)
		if l.CurrTokenType != LPAREN {
			l.AddError("expected '(' after struct name")
			return &ASTNode{} // error
		}
		l.SkipToken(LPAREN)

		// Use shared parameter parsing logic for struct fields
		parameters := parseParameterList(l, RPAREN, false) // no positional params
		if len(parameters) == 0 && l.CurrTokenType != RPAREN {
			return &ASTNode{} // error in parsing
		}

		if l.CurrTokenType == RPAREN {
			l.SkipToken(RPAREN)
		}

		if l.CurrTokenType == SEMICOLON {
			l.SkipToken(SEMICOLON)
		}

		// Store field information directly in AST node metadata, no children needed
		return &ASTNode{
			Kind:         NodeStruct,
			String:       structName,
			StructFields: parameters, // Store parsed field parameters directly
			Children:     nil,        // No AST children needed for struct fields
		}

	case IF:
		l.SkipToken(IF)
		children := []*ASTNode{}
		children = append(children, ParseExpression(l)) // if condition
		if l.CurrTokenType != LBRACE {
			return &ASTNode{} // error
		}

		children = append(children, parseBlockStatements(l)) // then block

		for l.CurrTokenType == ELSE {
			l.SkipToken(ELSE)
			if l.CurrTokenType == IF {
				l.SkipToken(IF)
				// else-if block
				children = append(children, ParseExpression(l))      // else condition
				children = append(children, parseBlockStatements(l)) // else block
			} else if l.CurrTokenType == LBRACE {
				// else block
				children = append(children, nil)                     // else condition (nil for final else)
				children = append(children, parseBlockStatements(l)) // else block
				break                                                // final else, no more chaining
			} else {
				return &ASTNode{} // error: expected { after else
			}
		}

		return &ASTNode{
			Kind:     NodeIf,
			Children: children,
		}

	case VAR:
		l.SkipToken(VAR)
		if l.CurrTokenType != LOWER_IDENT {
			l.AddError("expected variable name (must start with lowercase letter)")
			return &ASTNode{} // error
		}
		varName := &ASTNode{
			Kind:   NodeIdent,
			String: l.CurrLiteral,
		}
		l.SkipToken(LOWER_IDENT)
		if l.CurrTokenType != UPPER_IDENT && l.CurrTokenType != LBRACKET {
			l.AddError("expected type name (must start with uppercase letter)")
			return &ASTNode{} // error - expecting type
		}

		// Parse type using new TypeNode system
		varName.TypeAST = parseTypeExpression(l)
		if varName.TypeAST == nil {
			return &ASTNode{} // error - invalid type
		}

		// Check for optional initialization: var x I64 = value;
		if l.CurrTokenType == ASSIGN {
			l.SkipToken(ASSIGN)
			// Parse the initialization expression
			initExpr := ParseExpression(l)

			if l.CurrTokenType == SEMICOLON {
				l.SkipToken(SEMICOLON)
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
			if l.CurrTokenType == SEMICOLON {
				l.SkipToken(SEMICOLON)
			}
			return &ASTNode{
				Kind:     NodeVar,
				Children: []*ASTNode{varName},
				TypeAST:  varName.TypeAST,
			}
		}

	case LBRACE:
		l.SkipToken(LBRACE)
		var statements []*ASTNode
		for l.CurrTokenType != RBRACE && l.CurrTokenType != EOF {
			stmt := ParseStatement(l)
			statements = append(statements, stmt)
		}
		if l.CurrTokenType == RBRACE {
			l.SkipToken(RBRACE)
		}
		return &ASTNode{
			Kind:     NodeBlock,
			Children: statements,
		}

	case RETURN:
		l.SkipToken(RETURN)
		var children []*ASTNode
		// Check if there's an expression after return
		if l.CurrTokenType != SEMICOLON {
			expr := ParseExpression(l)
			children = append(children, expr)
		}
		if l.CurrTokenType == SEMICOLON {
			l.SkipToken(SEMICOLON)
		}
		return &ASTNode{
			Kind:     NodeReturn,
			Children: children,
		}

	case LOOP:
		l.SkipToken(LOOP)
		if l.CurrTokenType != LBRACE {
			return &ASTNode{} // error
		}
		l.SkipToken(LBRACE)
		var statements []*ASTNode
		for l.CurrTokenType != RBRACE && l.CurrTokenType != EOF {
			stmt := ParseStatement(l)
			statements = append(statements, stmt)
		}
		if l.CurrTokenType == RBRACE {
			l.SkipToken(RBRACE)
		}
		return &ASTNode{
			Kind:     NodeLoop,
			Children: statements,
		}

	case BREAK:
		l.SkipToken(BREAK)
		if l.CurrTokenType == SEMICOLON {
			l.SkipToken(SEMICOLON)
		}
		return &ASTNode{
			Kind: NodeBreak,
		}

	case CONTINUE:
		l.SkipToken(CONTINUE)
		if l.CurrTokenType == SEMICOLON {
			l.SkipToken(SEMICOLON)
		}
		return &ASTNode{
			Kind: NodeContinue,
		}

	case FUNC:
		return parseFunctionDeclaration(l)

	default:
		// Expression statement
		expr := ParseExpression(l)
		if l.CurrTokenType == SEMICOLON {
			l.SkipToken(SEMICOLON)
		}
		return expr
	}
}

// parseFunctionDeclaration parses a function declaration
// Syntax: func name(param1: Type, param2: Type): ReturnType { body }
// Or:     func name(param1: Type, param2: Type) { body } // void return
func parseFunctionDeclaration(l *Lexer) *ASTNode {
	l.SkipToken(FUNC) // consume 'func'

	// Parse function name
	var functionName string
	if l.CurrTokenType == LOWER_IDENT {
		functionName = l.CurrLiteral
		l.SkipToken(LOWER_IDENT)
	} else if l.CurrTokenType == UPPER_IDENT {
		l.AddError("expected function name (must start with lowercase letter)")
		functionName = l.CurrLiteral
		l.SkipToken(UPPER_IDENT) // Treat as function name but continue parsing
	} else {
		l.AddError("expected function name (must start with lowercase letter)")
		return &ASTNode{Kind: NodeFunc} // Return placeholder for error recovery
	}

	// Parse parameter list
	if l.CurrTokenType != LPAREN {
		l.AddError("expected '(' after function name")
		return &ASTNode{Kind: NodeFunc}
	}
	l.SkipToken(LPAREN)

	// Use shared parameter parsing logic for function parameters
	paramList := parseParameterList(l, RPAREN, true) // allow positional params
	if len(paramList) == 0 && l.CurrTokenType != RPAREN {
		l.AddError("error parsing function parameters")
	}

	// Use parsed parameters directly (now unified with struct fields)
	parameters := paramList

	if l.CurrTokenType != RPAREN {
		l.AddError("expected ')' after parameter list")
	} else {
		l.SkipToken(RPAREN)
	}

	// Parse optional return type
	var returnType *TypeNode
	if l.CurrTokenType == COLON {
		l.SkipToken(COLON)
		returnType = parseTypeExpression(l)
		if returnType == nil {
			l.AddError("expected return type after ':'")
		}
	}

	// Parse function body
	if l.CurrTokenType != LBRACE {
		l.AddError("expected '{' for function body")
		return &ASTNode{Kind: NodeFunc, FunctionName: functionName, Parameters: parameters, ReturnType: returnType}
	}
	l.SkipToken(LBRACE)
	var statements []*ASTNode
	for l.CurrTokenType != RBRACE && l.CurrTokenType != EOF {
		stmt := ParseStatement(l)
		statements = append(statements, stmt)
	}
	if l.CurrTokenType == RBRACE {
		l.SkipToken(RBRACE)
	}

	return &ASTNode{
		Kind:         NodeFunc,
		FunctionName: functionName,
		Parameters:   parameters,
		ReturnType:   returnType,
		Children:     statements,
	}
}

// ParseProgram parses a complete program (multiple functions and statements)
func ParseProgram(l *Lexer) *ASTNode {
	var statements []*ASTNode

	for l.CurrTokenType != EOF {
		stmt := ParseStatement(l)
		statements = append(statements, stmt)
	}

	return &ASTNode{
		Kind:     NodeBlock,
		Children: statements,
	}
}
