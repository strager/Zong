package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type TestCase struct {
	Name       string
	InputType  string // "zong-expr" or "zong-program"
	Input      string
	Expected   string
	SourceFile string
	FuncName   string
}

type Extractor struct {
	fileSet           *token.FileSet
	cases             []TestCase
	functionsToDelete map[string][]string // filename -> function names
}

func NewExtractor() *Extractor {
	return &Extractor{
		fileSet:           token.NewFileSet(),
		cases:             make([]TestCase, 0),
		functionsToDelete: make(map[string][]string),
	}
}

func (e *Extractor) extractFromTestFiles(pattern string) error {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := e.visitFile(file); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to process %s: %v\n", file, err)
		}
	}

	return nil
}

func (e *Extractor) visitFile(filename string) error {
	src, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	file, err := parser.ParseFile(e.fileSet, filename, src, parser.ParseComments)
	if err != nil {
		return err
	}

	// Visit all function declarations
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if strings.HasPrefix(fn.Name.Name, "Test") {
				e.extractFromFunction(fn, filepath.Base(filename))
			}
		}
	}

	return nil
}

func (e *Extractor) extractFromFunction(fn *ast.FuncDecl, sourceFile string) {
	// Collect all variable assignments and CompileToWASM calls
	var variables = make(map[string]string) // variable name -> string value
	var inputType = "zong-program"          // default
	var input string
	var expected string

	// First pass: collect variable assignments
	ast.Inspect(fn, func(n ast.Node) bool {
		if assign, ok := n.(*ast.AssignStmt); ok {
			if len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
				if ident, ok := assign.Lhs[0].(*ast.Ident); ok {
					if val, ok := e.resolveStringLiteral(assign.Rhs[0]); ok {
						variables[ident.Name] = val
					}
				}
			}
		}
		return true
	})

	// Second pass: look for parsing calls to determine input type and source
	ast.Inspect(fn, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok {
				switch ident.Name {
				case "ParseExpression":
					inputType = "zong-expr"
				case "ParseProgram":
					inputType = "zong-program"
				case "compileExpression":
					inputType = "zong-expr"
					// Look for string argument
					if len(call.Args) >= 2 {
						if val, ok := e.resolveStringLiteral(call.Args[1]); ok {
							input = val
						}
					}
				}
			}
		}
		return true
	})

	// If input not found from compileExpression, look in variables
	if input == "" {
		for varName, varValue := range variables {
			if strings.Contains(varName, "program") || strings.Contains(varName, "source") || strings.Contains(varName, "expression") {
				input = varValue
				break
			}
		}
	}

	// Third pass: look for expected output
	ast.Inspect(fn, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			// Look for be.Equal calls
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "be" && sel.Sel.Name == "Equal" {
					if len(call.Args) >= 3 {
						if val, ok := e.resolveStringLiteral(call.Args[2]); ok {
							expected = val
						}
					}
				}
			}
			// Look for executeWasmAndVerify calls
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "executeWasmAndVerify" {
				if len(call.Args) >= 3 {
					if val, ok := e.resolveStringLiteral(call.Args[2]); ok {
						expected = val
					}
				}
			}
		}
		return true
	})

	// Only create test case if we have meaningful input and expected output
	// Skip tests that expect errors for now
	if input != "" && expected != "" && !strings.Contains(expected, "error:") {
		testCase := TestCase{
			Name:       e.generateTestName(fn.Name.Name),
			InputType:  inputType,
			Input:      e.cleanZongCode(input),
			Expected:   e.cleanExpectedOutput(expected),
			SourceFile: sourceFile,
			FuncName:   fn.Name.Name,
		}

		// Check for duplicates before adding
		if !e.isDuplicate(testCase) {
			e.cases = append(e.cases, testCase)

			// Track this function for deletion
			if e.functionsToDelete[sourceFile] == nil {
				e.functionsToDelete[sourceFile] = make([]string, 0)
			}
			e.functionsToDelete[sourceFile] = append(e.functionsToDelete[sourceFile], fn.Name.Name)
		}
	}
}

func (e *Extractor) cleanZongCode(input string) string {
	// Remove null terminator
	input = strings.TrimSuffix(input, "\x00")

	// Clean up common patterns
	input = strings.TrimSpace(input)

	return input
}

func (e *Extractor) cleanExpectedOutput(expected string) string {
	// Remove trailing newlines that are just formatting
	expected = strings.TrimSuffix(expected, "\n")

	return expected
}

func (e *Extractor) isDuplicate(newTest TestCase) bool {
	for _, existing := range e.cases {
		if existing.Input == newTest.Input && existing.Expected == newTest.Expected && existing.InputType == newTest.InputType {
			return true
		}
	}
	return false
}

func (e *Extractor) generateTestName(funcName string) string {
	// Convert TestFunctionName to "function name"
	name := strings.TrimPrefix(funcName, "Test")

	// Convert CamelCase to space-separated words
	var result []rune
	for i, r := range name {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result = append(result, ' ')
		}
		if i == 0 {
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, unicode.ToLower(r))
		}
	}

	return string(result)
}

func (e *Extractor) resolveStringLiteral(expr ast.Expr) (string, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			// Parse the Go string literal
			if val, err := strconv.Unquote(e.Value); err == nil {
				return val, true
			}
		}
	}
	return "", false
}

func (e *Extractor) generateSexyMarkdown() string {
	if len(e.cases) == 0 {
		return "# No test cases found\n"
	}

	// Sort test cases by source file and function name
	sort.Slice(e.cases, func(i, j int) bool {
		if e.cases[i].SourceFile != e.cases[j].SourceFile {
			return e.cases[i].SourceFile < e.cases[j].SourceFile
		}
		return e.cases[i].FuncName < e.cases[j].FuncName
	})

	var sb strings.Builder
	sb.WriteString("# Extracted execution tests\n\n")
	sb.WriteString("Generated from existing Go test files.\n\n")

	currentFile := ""
	for _, tc := range e.cases {
		if tc.SourceFile != currentFile {
			currentFile = tc.SourceFile
			sb.WriteString(fmt.Sprintf("## Tests from %s\n\n", currentFile))
		}

		sb.WriteString(fmt.Sprintf("### Test: %s\n", tc.Name))
		sb.WriteString(fmt.Sprintf("```%s\n", tc.InputType))
		sb.WriteString(tc.Input)
		sb.WriteString("\n```\n")
		sb.WriteString("```execute\n")
		sb.WriteString(tc.Expected)
		sb.WriteString("\n```\n\n")
	}

	return sb.String()
}

func (e *Extractor) deleteExtractedFunctions() error {
	if len(e.functionsToDelete) == 0 {
		fmt.Fprintf(os.Stderr, "No functions to delete\n")
		return nil
	}

	for filename, functionNames := range e.functionsToDelete {
		if err := e.modifyFile(filename, functionNames); err != nil {
			return fmt.Errorf("failed to modify %s: %w", filename, err)
		}
		fmt.Fprintf(os.Stderr, "Deleted %d functions from %s: %v\n", len(functionNames), filename, functionNames)
	}
	return nil
}

func (e *Extractor) modifyFile(filename string, functionsToDelete []string) error {
	// Read and parse the file again
	src, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	file, err := parser.ParseFile(e.fileSet, filename, src, parser.ParseComments)
	if err != nil {
		return err
	}

	// Filter out the functions to delete
	var newDecls []ast.Decl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if !e.contains(functionsToDelete, fn.Name.Name) {
				newDecls = append(newDecls, decl)
			}
		} else {
			newDecls = append(newDecls, decl)
		}
	}
	file.Decls = newDecls

	// Pretty-print and write back
	return e.writeFormattedFile(filename, file)
}

func (e *Extractor) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (e *Extractor) writeFormattedFile(filename string, file *ast.File) error {
	// Remove unused imports before writing
	e.removeUnusedImports(file)

	var buf bytes.Buffer
	if err := format.Node(&buf, e.fileSet, file); err != nil {
		return err
	}

	return os.WriteFile(filename, buf.Bytes(), 0644)
}

// removeUnusedImports removes imports that are no longer used after function deletion
func (e *Extractor) removeUnusedImports(file *ast.File) {
	if len(file.Decls) == 0 {
		return
	}

	// Check which imports are actually used
	usedImports := make(map[string]bool)

	// Walk through all remaining code to see what's used
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			// For qualified identifiers like "be.Equal"
			if ident, ok := node.X.(*ast.Ident); ok {
				usedImports[ident.Name] = true
			}
		case *ast.Ident:
			// For unqualified identifiers that might be imported
			usedImports[node.Name] = true
		}
		return true
	})

	// Remove import declarations that aren't used
	for i, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			var newSpecs []ast.Spec
			for _, spec := range genDecl.Specs {
				if importSpec, ok := spec.(*ast.ImportSpec); ok {
					importName := ""
					if importSpec.Name != nil {
						importName = importSpec.Name.Name
					} else {
						// Extract package name from import path
						path := strings.Trim(importSpec.Path.Value, "\"")
						parts := strings.Split(path, "/")
						importName = parts[len(parts)-1]
					}

					// Keep the import if it's used
					if usedImports[importName] || e.isEssentialImport(importName) {
						newSpecs = append(newSpecs, spec)
					}
				}
			}

			if len(newSpecs) > 0 {
				genDecl.Specs = newSpecs
			} else {
				// Remove the entire import declaration if empty
				file.Decls = append(file.Decls[:i], file.Decls[i+1:]...)
			}
		}
	}
}

// isEssentialImport checks if an import should be kept regardless of usage analysis
// This is a safety measure for imports that might be used in ways we don't detect
func (e *Extractor) isEssentialImport(importName string) bool {
	essential := []string{"fmt", "os", "strings"}
	for _, name := range essential {
		if name == importName {
			return true
		}
	}
	return false
}

func main() {
	extractor := NewExtractor()

	// Extract from test files
	if err := extractor.extractFromTestFiles("*_test.go"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Generate and output Sexy markdown
	output := extractor.generateSexyMarkdown()
	fmt.Print(output)

	// Delete the original functions from source files
	if err := extractor.deleteExtractedFunctions(); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting functions: %v\n", err)
		os.Exit(1)
	}
}
