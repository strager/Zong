package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func showUsage() {
	fmt.Fprintf(os.Stderr, `Zong - A programming language that compiles to WebAssembly

Usage:
    zong <command> [arguments]

Commands:
    run <file>      Compile and execute a .zong file
    build <file>    Compile a .zong file to WebAssembly
    eval <code>     Evaluate inline Zong code
    check <file>    Parse and type-check a .zong file
    help            Show this help message

Examples:
    zong run examples/prime.zong
    zong build -o program.wasm hello.zong
    zong eval 'print(42)'
    zong check myfile.zong

Use "zong <command> -h" for more information about a command.
`)
}

func runCommand(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	verbose := fs.Bool("v", false, "Show verbose compilation details")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: zong run [-v] <file>\n")
		fmt.Fprintf(os.Stderr, "Compile and execute a .zong file\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Error: expected exactly one file argument\n")
		fs.Usage()
		os.Exit(1)
	}

	filename := fs.Arg(0)

	if *verbose {
		fmt.Printf("Compiling %s...\n", filename)
	}

	// Read the file
	sourceBytes, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filename, err)
		os.Exit(1)
	}

	// Add null terminator as required by lexer
	input := append(sourceBytes, '\x00')

	// Compile
	wasmBytes, err := compileProgram(input, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compilation failed: %v\n", err)
		os.Exit(1)
	}

	// Write WASM to temporary file
	tempWasm := "temp_" + strings.TrimSuffix(filepath.Base(filename), ".zong") + ".wasm"
	err = os.WriteFile(tempWasm, wasmBytes, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing WASM file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tempWasm) // Clean up temporary file

	if *verbose {
		fmt.Printf("Generated %d bytes of WASM\n", len(wasmBytes))
		fmt.Printf("Executing...\n")
	}

	// Execute the WASM
	if err := executeWasmFile(tempWasm); err != nil {
		fmt.Fprintf(os.Stderr, "Execution failed: %v\n", err)
		os.Exit(1)
	}
}

func buildCommand(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	output := fs.String("o", "", "Output file path (default: <filename>.wasm)")
	verbose := fs.Bool("v", false, "Show verbose compilation details")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: zong build [-o output] [-v] <file>\n")
		fmt.Fprintf(os.Stderr, "Compile a .zong file to WebAssembly\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Error: expected exactly one file argument\n")
		fs.Usage()
		os.Exit(1)
	}

	filename := fs.Arg(0)

	// Determine output filename
	outputFile := *output
	if outputFile == "" {
		outputFile = strings.TrimSuffix(filename, ".zong") + ".wasm"
	}

	if *verbose {
		fmt.Printf("Compiling %s to %s...\n", filename, outputFile)
	}

	// Read the file
	sourceBytes, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filename, err)
		os.Exit(1)
	}

	// Add null terminator as required by lexer
	input := append(sourceBytes, '\x00')

	// Compile
	wasmBytes, err := compileProgram(input, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compilation failed: %v\n", err)
		os.Exit(1)
	}

	// Write WASM file
	err = os.WriteFile(outputFile, wasmBytes, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing WASM file %s: %v\n", outputFile, err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s (%d bytes)\n", outputFile, len(wasmBytes))
}

func evalCommand(args []string) {
	fs := flag.NewFlagSet("eval", flag.ExitOnError)
	verbose := fs.Bool("v", false, "Show verbose compilation details")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: zong eval [-v] <code>\n")
		fmt.Fprintf(os.Stderr, "Evaluate inline Zong code\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Error: expected exactly one code argument\n")
		fs.Usage()
		os.Exit(1)
	}

	code := fs.Arg(0)

	if *verbose {
		fmt.Printf("Evaluating: %s\n", code)
	}

	// Add null terminator as required by lexer
	input := []byte(code + "\x00")

	// Compile
	wasmBytes, err := compileProgram(input, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compilation failed: %v\n", err)
		os.Exit(1)
	}

	// Write WASM to temporary file
	tempWasm := "temp_eval.wasm"
	err = os.WriteFile(tempWasm, wasmBytes, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing WASM file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tempWasm) // Clean up temporary file

	if *verbose {
		fmt.Printf("Generated %d bytes of WASM\n", len(wasmBytes))
	}

	// Execute the WASM
	if err := executeWasmFile(tempWasm); err != nil {
		fmt.Fprintf(os.Stderr, "Execution failed: %v\n", err)
		os.Exit(1)
	}
}

func checkCommand(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	verbose := fs.Bool("v", false, "Show verbose checking details")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: zong check [-v] <file>\n")
		fmt.Fprintf(os.Stderr, "Parse and type-check a .zong file\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Error: expected exactly one file argument\n")
		fs.Usage()
		os.Exit(1)
	}

	filename := fs.Arg(0)

	if *verbose {
		fmt.Printf("Checking %s...\n", filename)
	}

	// Read the file
	sourceBytes, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filename, err)
		os.Exit(1)
	}

	// Add null terminator as required by lexer
	input := append(sourceBytes, '\x00')

	// Parse and check (but don't compile to WASM)
	l := NewLexer(input)
	l.NextToken()
	ast := ParseProgram(l)

	// Check for parsing errors
	if l.Errors.HasErrors() {
		fmt.Printf("Parsing errors in %s:\n%s\n", filename, l.Errors.String())
		os.Exit(1)
	}

	// Build symbol table for type checking
	symbolTable := BuildSymbolTable(ast)
	if symbolTable.Errors.HasErrors() {
		fmt.Printf("Symbol resolution errors in %s:\n%s\n", filename, symbolTable.Errors.String())
		os.Exit(1)
	}

	// Perform type checking
	typeErrors := CheckProgram(ast, symbolTable.typeTable)
	if typeErrors.HasErrors() {
		fmt.Printf("Type checking errors in %s:\n%s\n", filename, typeErrors.String())
		os.Exit(1)
	}

	fmt.Printf("%s: no errors found\n", filename)

	if *verbose {
		fmt.Printf("AST: %s\n", ToSExpr(ast))
	}
}

// Helper function to compile a program
func compileProgram(input []byte, verbose bool) ([]byte, error) {
	// Parse
	l := NewLexer(input)
	l.NextToken()
	ast := ParseProgram(l)

	// Check for parsing errors
	if l.Errors.HasErrors() {
		return nil, fmt.Errorf("parsing errors:\n%s", l.Errors.String())
	}

	// Build symbol table for type checking
	symbolTable := BuildSymbolTable(ast)
	if symbolTable.Errors.HasErrors() {
		return nil, fmt.Errorf("symbol resolution errors:\n%s", symbolTable.Errors.String())
	}

	// Perform type checking
	typeErrors := CheckProgram(ast, symbolTable.typeTable)
	if typeErrors.HasErrors() {
		return nil, fmt.Errorf("type checking errors:\n%s", typeErrors.String())
	}

	if verbose {
		fmt.Printf("AST: %s\n", ToSExpr(ast))
	}

	// Compile to WASM
	wasmBytes := CompileToWASM(ast)
	return wasmBytes, nil
}

// Helper function to execute WASM using the Rust runtime
func executeWasmFile(wasmFile string) error {
	runtimeBinary := "./wasmruntime/target/release/wasmruntime"

	// Check if runtime exists, build if necessary
	if _, err := os.Stat(runtimeBinary); os.IsNotExist(err) {
		fmt.Println("Building Rust WASM runtime...")
		buildCmd := exec.Command("cargo", "build", "--release")
		buildCmd.Dir = "./wasmruntime"
		buildOutput, buildErr := buildCmd.CombinedOutput()
		if buildErr != nil {
			return fmt.Errorf("failed to build WASM runtime: %v\nOutput: %s", buildErr, buildOutput)
		}
	}

	// Execute the WASM file
	cmd := exec.Command(runtimeBinary, wasmFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "run":
		runCommand(args)
	case "build":
		buildCommand(args)
	case "eval":
		evalCommand(args)
	case "check":
		checkCommand(args)
	case "help", "-h", "--help":
		showUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		showUsage()
		os.Exit(1)
	}
}
