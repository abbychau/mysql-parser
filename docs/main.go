package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/abbychau/mysql-parser"
	"github.com/abbychau/mysql-parser/ast"
	_ "github.com/abbychau/mysql-parser/parser_driver"
)

// ColumnExtractor implements ast.Visitor to extract column names
type ColumnExtractor struct {
	ColNames []string `json:"columns"`
}

func (v *ColumnExtractor) Enter(in ast.Node) (ast.Node, bool) {
	if name, ok := in.(*ast.ColumnName); ok {
		v.ColNames = append(v.ColNames, name.Name.O)
	}
	return in, false
}

func (v *ColumnExtractor) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

// ParseResult represents the result of parsing
type ParseResult struct {
	Success bool     `json:"success"`
	Columns []string `json:"columns,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// parseSQL is the main function exposed to JavaScript
func parseSQL(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		result := ParseResult{
			Success: false,
			Error:   "Expected exactly one argument (SQL string)",
		}
		jsonBytes, _ := json.Marshal(result)
		return string(jsonBytes)
	}

	sqlText := args[0].String()
	
	// Create parser instance
	p := parser.New()
	
	// Parse the SQL
	stmtNodes, _, err := p.ParseSQL(sqlText)
	if err != nil {
		result := ParseResult{
			Success: false,
			Error:   fmt.Sprintf("Parse error: %v", err),
		}
		jsonBytes, _ := json.Marshal(result)
		return string(jsonBytes)
	}

	if len(stmtNodes) == 0 {
		result := ParseResult{
			Success: false,
			Error:   "No statements found",
		}
		jsonBytes, _ := json.Marshal(result)
		return string(jsonBytes)
	}

	// Extract columns using visitor pattern
	extractor := &ColumnExtractor{}
	stmtNodes[0].Accept(extractor)

	result := ParseResult{
		Success: true,
		Columns: extractor.ColNames,
	}
	
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}

// version returns the version of the parser
func version(this js.Value, args []js.Value) interface{} {
	return "MySQL Parser WASM Demo v3.0.0 (Enhanced Parser Driver with Full Types)"
}

func main() {
	fmt.Println("MySQL Parser WASM initialized")
	
	// Register functions to be called from JavaScript
	js.Global().Set("parseSQL", js.FuncOf(parseSQL))
	js.Global().Set("parserVersion", js.FuncOf(version))
	
	// Keep the program running
	select {}
}