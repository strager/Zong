package main

import (
	"bytes"
	"fmt"
)

// VarStorage represents how a variable is stored
type VarStorage int

const (
	VarStorageLocal  VarStorage = iota // Variable stored in WASM local
	VarStorageTStack                   // Variable stored on the stack (addressed)
)

// LocalVarInfo represents information about a local variable
type LocalVarInfo struct {
	Name    string
	Type    *TypeNode  // TypeNode representation of the type
	Storage VarStorage // How the variable is stored
	Address uint32     // For VarStorageLocal: WASM local index; For VarStorageTStack: byte offset in stack frame
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

func EmitCodeSection(buf *bytes.Buffer, ast *ASTNode, symbolTable *SymbolTable) {
	writeByte(buf, 0x0A) // code section id

	// Collect local variables from AST and calculate frame size
	locals, frameSize := collectLocalVariables(ast, symbolTable)

	// Generate function body
	var bodyBuf bytes.Buffer

	// Emit locals declarations (include frame pointer)
	i32LocalCount := 0
	i64LocalCount := 0
	for _, local := range locals {
		if local.Storage == VarStorageLocal {
			if isWASMI32Type(local.Type) {
				i32LocalCount++
			} else if isWASMI64Type(local.Type) {
				i64LocalCount++
			}
		}
	}

	if frameSize > 0 {
		i32LocalCount++ // Add frame pointer local (now i32)
	}

	// Emit local type groups
	groupCount := 0
	if i32LocalCount > 0 {
		groupCount++
	}
	if i64LocalCount > 0 {
		groupCount++
	}

	writeLEB128(&bodyBuf, uint32(groupCount))

	if i32LocalCount > 0 {
		writeLEB128(&bodyBuf, uint32(i32LocalCount)) // count of I32 locals
		writeByte(&bodyBuf, 0x7F)                    // I32 type
	}
	if i64LocalCount > 0 {
		writeLEB128(&bodyBuf, uint32(i64LocalCount)) // count of I64 locals
		writeByte(&bodyBuf, 0x7E)                    // I64 type
	}

	// Emit frame setup code if we have addressed variables
	if frameSize > 0 {
		EmitFrameSetup(&bodyBuf, locals, frameSize)
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
		// Variable reference
		var targetLocal *LocalVarInfo
		for i := range locals {
			if locals[i].Name == node.String {
				targetLocal = &locals[i]
				break
			}
		}
		if targetLocal == nil {
			panic("Undefined variable: " + node.String)
		}

		if targetLocal.Storage == VarStorageLocal {
			// Local variable - emit local.get
			writeByte(buf, LOCAL_GET)
			writeLEB128(buf, targetLocal.Address)
		} else {
			// Stack variable
			if targetLocal.Type.Kind == TypeStruct {
				// For struct variables, return the address of the struct (not the value)
				framePointerIndex := getFramePointerIndex(locals)
				writeByte(buf, LOCAL_GET)
				writeLEB128(buf, framePointerIndex)

				// Add variable offset if not zero
				if targetLocal.Address > 0 {
					writeByte(buf, I64_CONST)
					writeLEB128Signed(buf, int64(targetLocal.Address))
					writeByte(buf, I64_ADD)
				}
				// Don't load - leave address on stack for struct operations
			} else {
				// Non-struct stack variable - load from memory
				framePointerIndex := getFramePointerIndex(locals)
				writeByte(buf, LOCAL_GET)
				writeLEB128(buf, framePointerIndex)

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
				varName := lhs.String
				var targetLocal *LocalVarInfo
				for i := range locals {
					if locals[i].Name == varName {
						targetLocal = &locals[i]
						break
					}
				}
				if targetLocal == nil {
					panic("Undefined variable: " + varName)
				}

				if targetLocal.Type.Kind == TypeStruct {
					// Struct assignment (copy)
					structSize := uint32(GetTypeSize(targetLocal.Type))

					// Get destination address
					framePointerIndex := getFramePointerIndex(locals)
					writeByte(buf, LOCAL_GET)
					writeLEB128(buf, framePointerIndex)

					// Add variable offset if not zero
					if targetLocal.Address > 0 {
						writeByte(buf, I32_CONST)
						writeLEB128Signed(buf, int64(targetLocal.Address))
						writeByte(buf, I32_ADD)
					}

					// Get source address (RHS should evaluate to struct address)
					EmitExpression(buf, rhs, locals) // RHS address

					// Perform memory copy (simplified - copy 8 bytes at a time)
					// Stack: [dst_addr_i32, src_addr_i32]
					for offset := uint32(0); offset < structSize; offset += 8 {
						// Load from source: [dst, src] -> [dst, src, value]
						writeByte(buf, LOCAL_GET) // Duplicate source address (assume src is in local)
						// Note: This is a simplification - real implementation would need proper stack management
					}

					// For now, just do a single 8-byte copy to avoid complex stack management
					writeByte(buf, I64_LOAD)  // Load from source
					writeByte(buf, 0x03)      // alignment
					writeByte(buf, 0x00)      // offset
					writeByte(buf, I64_STORE) // Store to destination
					writeByte(buf, 0x03)      // alignment
					writeByte(buf, 0x00)      // offset

				} else if targetLocal.Storage == VarStorageLocal {
					// Local variable - emit local.set
					EmitExpression(buf, rhs, locals) // RHS value
					writeByte(buf, LOCAL_SET)
					writeLEB128(buf, targetLocal.Address)
				} else {
					// Stack variable - store to memory
					// First get the address (before evaluating RHS)
					framePointerIndex := getFramePointerIndex(locals)
					writeByte(buf, LOCAL_GET)
					writeLEB128(buf, framePointerIndex)

					// Add variable offset if not zero
					if targetLocal.Address > 0 {
						writeByte(buf, I32_CONST)
						writeLEB128Signed(buf, int64(targetLocal.Address))
						writeByte(buf, I32_ADD)
					}

					// Now evaluate the RHS value
					EmitExpression(buf, rhs, locals) // RHS value

					// Stack is now: [address_i32, value] - perfect for i64.store
					writeByte(buf, I64_STORE) // Store i64 to memory
					writeByte(buf, 0x03)      // alignment (8 bytes = 2^3)
					writeByte(buf, 0x00)      // offset
				}
			} else if lhs.Kind == NodeUnary && lhs.Op == "*" {
				// Pointer dereference assignment: ptr* = value
				// First get the address where to store
				EmitExpression(buf, lhs.Children[0], locals) // Get pointer value (i32)

				// Now evaluate the RHS value
				EmitExpression(buf, rhs, locals) // RHS value

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

				varName := baseExpr.String
				var targetLocal *LocalVarInfo
				for i := range locals {
					if locals[i].Name == varName {
						targetLocal = &locals[i]
						break
					}
				}
				if targetLocal == nil {
					panic("Undefined struct variable: " + varName)
				}

				if targetLocal.Type.Kind != TypeStruct {
					panic("Field assignment on non-struct variable")
				}

				// Find field in struct definition
				var fieldOffset uint32
				var fieldType *TypeNode
				found := false
				for _, field := range targetLocal.Type.Fields {
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
				framePointerIndex := getFramePointerIndex(locals)
				writeByte(buf, LOCAL_GET)
				writeLEB128(buf, framePointerIndex)

				// Add struct base offset + field offset
				totalOffset := targetLocal.Address + fieldOffset
				if totalOffset > 0 {
					writeByte(buf, I32_CONST)
					writeLEB128Signed(buf, int64(totalOffset))
					writeByte(buf, I32_ADD)
				}

				// Now evaluate the RHS value
				EmitExpression(buf, rhs, locals) // RHS value

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
				arg := node.Children[1]
				EmitExpression(buf, arg, locals) // argument

				// If the argument is a pointer type, we need to widen it from i32 to i64
				// because print() expects i64
				if needsI32ToI64Widening(arg, locals) {
					writeByte(buf, I64_EXTEND_I32_S) // Convert i32 pointer to i64
				}
			}
			writeByte(buf, CALL) // call instruction
			writeLEB128(buf, 0)  // function index 0 (print import)
		}

	case NodeUnary:
		if node.Op == "&" {
			// Address-of operator
			EmitAddressOf(buf, node.Children[0], locals)
		} else if node.Op == "*" {
			// Dereference operator
			EmitExpression(buf, node.Children[0], locals) // Get the pointer value (i32)
			writeByte(buf, I64_LOAD)                      // Load i64 from memory using i32 address
			writeByte(buf, 0x03)                          // alignment (8 bytes = 2^3)
			writeByte(buf, 0x00)                          // offset
		} else if node.Op == "!" {
			// Handle existing unary not operator
			EmitExpression(buf, node.Children[0], locals)
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

		varName := baseExpr.String
		var targetLocal *LocalVarInfo
		for i := range locals {
			if locals[i].Name == varName {
				targetLocal = &locals[i]
				break
			}
		}
		if targetLocal == nil {
			panic("Undefined struct variable: " + varName)
		}

		if targetLocal.Type.Kind != TypeStruct {
			panic("Field access on non-struct variable")
		}

		// Find field in struct definition
		var fieldOffset uint32
		var fieldType *TypeNode
		found := false
		for _, field := range targetLocal.Type.Fields {
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
		framePointerIndex := getFramePointerIndex(locals)
		writeByte(buf, LOCAL_GET)
		writeLEB128(buf, framePointerIndex)

		// Add struct base offset + field offset
		totalOffset := targetLocal.Address + fieldOffset
		if totalOffset > 0 {
			writeByte(buf, I32_CONST)
			writeLEB128Signed(buf, int64(totalOffset))
			writeByte(buf, I32_ADD)
		}

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

	var buf bytes.Buffer

	// Emit WASM module header and sections in streaming fashion
	EmitWASMHeader(&buf)
	EmitTypeSection(&buf)                   // function type definitions
	EmitImportSection(&buf)                 // print function + tstack global import
	EmitFunctionSection(&buf)               // declare main function
	EmitMemorySection(&buf)                 // memory for tstack operations
	EmitExportSection(&buf)                 // export main function
	EmitCodeSection(&buf, ast, symbolTable) // main function body with compiled expression

	return buf.Bytes()
}

// collectLocalVariables traverses AST once to find all var declarations and address-of operations
// Returns the locals list and the total frame size for addressed variables
func collectLocalVariables(node *ASTNode, symbolTable *SymbolTable) ([]LocalVarInfo, uint32) {
	var locals []LocalVarInfo
	var frameOffset uint32 = 0

	var traverse func(*ASTNode)
	traverse = func(node *ASTNode) {
		switch node.Kind {
		case NodeVar:
			// Extract variable name
			varName := node.Children[0].String

			// Skip variables with no type information
			if node.TypeAST == nil {
				break
			}

			// Get the resolved type from symbol table if available
			resolvedType := node.TypeAST
			if symbolTable != nil {
				if symbol := symbolTable.LookupVariable(varName); symbol != nil {
					resolvedType = symbol.Type
				}
			}

			// Support I64, I64* (pointers are i32 in WASM), and other types
			if isWASMI32Type(resolvedType) || isWASMI64Type(resolvedType) {
				locals = append(locals, LocalVarInfo{
					Name:    varName,
					Type:    resolvedType,
					Storage: VarStorageLocal,
					// Address will be allocated later.
				})
			} else if resolvedType.Kind == TypeStruct {
				// Struct variables are always stored on tstack (addressed)
				structSize := uint32(GetTypeSize(resolvedType))
				locals = append(locals, LocalVarInfo{
					Name:    varName,
					Type:    resolvedType,
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
						if locals[i].Name == varName {
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
			traverse(child)
		}
	}

	traverse(node)

	// Reassign WASM local indices: i32 locals first, then i64 locals
	var i32Index uint32 = 0
	var i64Index uint32 = 0

	// Count i32 locals to know where i64 locals start
	i32Count := uint32(0)
	for i := range locals {
		if locals[i].Storage == VarStorageLocal && isWASMI32Type(locals[i].Type) {
			i32Count++
		}
	}

	// Calculate total i32 locals including frame pointer
	totalI32Locals := i32Count
	if frameOffset > 0 { // frameOffset > 0 means we need a frame pointer
		totalI32Locals++ // Add 1 for frame pointer
	}

	// Assign correct indices
	for i := range locals {
		if locals[i].Storage == VarStorageLocal {
			if isWASMI32Type(locals[i].Type) {
				locals[i].Address = i32Index
				i32Index++
			} else if isWASMI64Type(locals[i].Type) {
				locals[i].Address = totalI32Locals + i64Index
				i64Index++
			}
		}
	}

	return locals, frameOffset
}

// EmitFrameSetup generates frame setup code at function entry
func EmitFrameSetup(buf *bytes.Buffer, locals []LocalVarInfo, frameSize uint32) {
	// Get frame pointer local index (last local variable)
	framePointerIndex := getFramePointerIndex(locals)

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

// getFramePointerIndex returns the local index for the frame pointer
func getFramePointerIndex(locals []LocalVarInfo) uint32 {
	// Frame pointer is the last local (after all VarStorageLocal variable locals)
	// Count both i32 and i64 locals
	i32Count := uint32(0)
	i64Count := uint32(0)
	for _, local := range locals {
		if local.Storage == VarStorageLocal {
			if isWASMI32Type(local.Type) {
				i32Count++
			} else if isWASMI64Type(local.Type) {
				i64Count++
			}
		}
	}
	// Frame pointer is the last i32 local (comes after user i32 locals, before i64 locals)
	return i32Count
}

// EmitAddressOf generates code for address-of operations
func EmitAddressOf(buf *bytes.Buffer, operand *ASTNode, locals []LocalVarInfo) {
	if operand.Kind == NodeIdent {
		// Lvalue case: &variable
		varName := operand.String

		// Find the variable in locals
		var targetLocal *LocalVarInfo
		for i := range locals {
			if locals[i].Name == varName {
				targetLocal = &locals[i]
				break
			}
		}

		if targetLocal == nil {
			panic("Undefined variable in address-of: " + varName)
		}

		if targetLocal.Storage != VarStorageTStack {
			panic("Variable " + varName + " is not addressed but address-of is used")
		}

		// Load frame pointer
		framePointerIndex := getFramePointerIndex(locals)
		writeByte(buf, LOCAL_GET)
		writeLEB128(buf, framePointerIndex)

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
		EmitExpression(buf, operand, locals)

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
		return false // builtin types are not I32
	case TypePointer:
		return true // all pointers are I32 in WASM
	case TypeStruct:
		return false // structs are stored in memory, not as I32 locals
	}
	// unreachable with current TypeKind values
	panic("Unknown TypeKind: " + string(t.Kind))
}

// needsI32ToI64Widening checks if an expression produces i32 and needs widening to i64
func needsI32ToI64Widening(expr *ASTNode, locals []LocalVarInfo) bool {
	switch expr.Kind {
	case NodeIdent:
		// Look up variable in locals
		varName := expr.String
		for _, local := range locals {
			if local.Name == varName && isWASMI32Type(local.Type) {
				return true
			}
		}
		return false

	case NodeUnary:
		if expr.Op == "&" {
			// Address-of operator produces a pointer (i32)
			return true
		}
		return false

	default:
		// Other expressions don't produce i32 values that need widening
		return false
	}
}

// SymbolInfo represents information about a declared variable
type SymbolInfo struct {
	Name     string
	Type     *TypeNode
	Assigned bool // tracks if variable has been assigned a value
}

// SymbolTable tracks variable declarations and assignments
type SymbolTable struct {
	variables []SymbolInfo
	structs   []*TypeNode // struct type definitions
}

// TypeChecker holds state for type checking
type TypeChecker struct {
	errors []string
}

// NewSymbolTable creates a new empty symbol table
func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		variables: make([]SymbolInfo, 0),
		structs:   make([]*TypeNode, 0),
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
		}

		// Traverse children
		for _, child := range node.Children {
			collectDeclarations(child)
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

		// Traverse children
		for _, child := range node.Children {
			populateReferences(child)
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
		errors: make([]string, 0),
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
			_, err := CheckExpression(stmt, tc)
			if err != nil {
				return err
			}
		}

	case NodeCall, NodeIdent, NodeInteger, NodeDot:
		// Expression statement
		_, err := CheckExpression(stmt, tc)
		if err != nil {
			return err
		}

	case NodeReturn:
		// TODO: Implement return type checking in the future
		if len(stmt.Children) > 0 {
			_, err := CheckExpression(stmt.Children[0], tc)
			if err != nil {
				return err
			}
		}

	default:
		// Other statement types are valid for now
	}

	return nil
}

// CheckExpression validates an expression and returns its type
func CheckExpression(expr *ASTNode, tc *TypeChecker) (*TypeNode, error) {
	switch expr.Kind {
	case NodeInteger:
		return TypeI64, nil

	case NodeIdent:
		// Variable reference - use cached symbol reference
		if expr.Symbol == nil {
			return nil, fmt.Errorf("error: variable '%s' used before declaration", expr.String)
		}
		if !expr.Symbol.Assigned {
			return nil, fmt.Errorf("error: variable '%s' used before assignment", expr.String)
		}
		return expr.Symbol.Type, nil

	case NodeBinary:
		if expr.Op == "=" {
			// Assignment expression
			return CheckAssignmentExpression(expr.Children[0], expr.Children[1], tc)
		} else {
			// Binary operation
			leftType, err := CheckExpression(expr.Children[0], tc)
			if err != nil {
				return nil, err
			}
			rightType, err := CheckExpression(expr.Children[1], tc)
			if err != nil {
				return nil, err
			}

			// Ensure operand types match
			if !TypesEqual(leftType, rightType) {
				return nil, fmt.Errorf("error: type mismatch in binary operation")
			}

			// Return result type based on operator
			switch expr.Op {
			case "==", "!=", "<", ">", "<=", ">=":
				return TypeI64, nil // Comparison operators return integers (0 or 1)
			case "+", "-", "*", "/", "%":
				return leftType, nil // Arithmetic operators return operand type
			default:
				return nil, fmt.Errorf("error: unsupported binary operator '%s'", expr.Op)
			}
		}

	case NodeCall:
		// Function call - for now only validate print() function
		if len(expr.Children) == 0 || expr.Children[0].Kind != NodeIdent {
			return nil, fmt.Errorf("error: invalid function call")
		}
		funcName := expr.Children[0].String
		if funcName == "print" {
			// Validate arguments
			if len(expr.Children) != 2 {
				return nil, fmt.Errorf("error: print() function expects 1 argument")
			}
			_, err := CheckExpression(expr.Children[1], tc)
			if err != nil {
				return nil, err
			}
			return TypeI64, nil // print returns nothing, but use I64 for now
		} else {
			return nil, fmt.Errorf("error: unknown function '%s'", funcName)
		}

	case NodeDot:
		// Field access: struct.field
		if len(expr.Children) != 1 {
			return nil, fmt.Errorf("error: field access expects 1 base expression")
		}

		baseType, err := CheckExpression(expr.Children[0], tc)
		if err != nil {
			return nil, err
		}

		// Base must be a struct type
		if baseType.Kind != TypeStruct {
			return nil, fmt.Errorf("error: cannot access field of non-struct type %s", TypeToString(baseType))
		}

		// Find the field in the struct
		fieldName := expr.FieldName
		for _, field := range baseType.Fields {
			if field.Name == fieldName {
				return field.Type, nil
			}
		}

		return nil, fmt.Errorf("error: struct %s has no field named '%s'", baseType.String, fieldName)

	case NodeUnary:
		// Unary operations
		if expr.Op == "&" {
			// Address-of operator
			if len(expr.Children) != 1 {
				return nil, fmt.Errorf("error: address-of operator expects 1 operand")
			}

			operandType, err := CheckExpression(expr.Children[0], tc)
			if err != nil {
				return nil, err
			}

			// Return pointer type
			return &TypeNode{Kind: TypePointer, Child: operandType}, nil
		} else if expr.Op == "*" {
			// Dereference operator
			if len(expr.Children) != 1 {
				return nil, fmt.Errorf("error: dereference operator expects 1 operand")
			}

			operandType, err := CheckExpression(expr.Children[0], tc)
			if err != nil {
				return nil, err
			}

			// Operand must be a pointer type
			if operandType.Kind != TypePointer {
				return nil, fmt.Errorf("error: cannot dereference non-pointer type %s", TypeToString(operandType))
			}

			// Return the pointed-to type
			return operandType.Child, nil
		} else {
			return nil, fmt.Errorf("error: unsupported unary operator '%s'", expr.Op)
		}

	default:
		return nil, fmt.Errorf("error: unsupported expression type '%s'", expr.Kind)
	}
}

// CheckAssignment validates an assignment statement
func CheckAssignment(lhs, rhs *ASTNode, tc *TypeChecker) error {
	// Validate RHS type first
	rhsType, err := CheckExpression(rhs, tc)
	if err != nil {
		return err
	}

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
		ptrType, err := CheckExpression(lhs.Children[0], tc)
		if err != nil {
			return err
		}

		if ptrType.Kind != TypePointer {
			return fmt.Errorf("error: cannot dereference non-pointer type %s", TypeToString(ptrType))
		}

		lhsType = ptrType.Child

	} else if lhs.Kind == NodeDot {
		// Field assignment (e.g., s.field = value)
		lhsType, err = CheckExpression(lhs, tc)
		if err != nil {
			return err
		}

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

// CheckAssignmentExpression validates an assignment expression and returns its type
func CheckAssignmentExpression(lhs, rhs *ASTNode, tc *TypeChecker) (*TypeNode, error) {
	err := CheckAssignment(lhs, rhs, tc)
	if err != nil {
		return nil, err
	}

	// Assignment expression returns the type of the assigned value
	return CheckExpression(rhs, tc)
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

	default:
		// Expression statement
		expr := ParseExpression()
		if CurrTokenType == SEMICOLON {
			SkipToken(SEMICOLON)
		}
		return expr
	}
}
