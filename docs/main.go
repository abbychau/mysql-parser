package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"syscall/js"

	parser "github.com/abbychau/mysql-parser"
	"github.com/abbychau/mysql-parser/ast"
	_ "github.com/abbychau/mysql-parser/parser_driver"
)

// InfoExtractor implements ast.Visitor to extract column and table names
type InfoExtractor struct {
	ColNames   []string
	TableNames []string
}

func (v *InfoExtractor) Enter(in ast.Node) (ast.Node, bool) {
	if name, ok := in.(*ast.ColumnName); ok {
		v.ColNames = append(v.ColNames, name.Name.O)
	}
	if name, ok := in.(*ast.TableName); ok {
		fullTableName := name.Name.O
		if name.Schema.O != "" {
			fullTableName = name.Schema.O + "." + name.Name.O
		}
		v.TableNames = append(v.TableNames, fullTableName)
	}
	return in, false
}

func (v *InfoExtractor) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

// ParseResult represents the result of parsing
type ParseResult struct {
	Success  bool     `json:"success"`
	Columns  []string `json:"columns,omitempty"`
	Tables   []string `json:"tables,omitempty"`
	StmtType string   `json:"stmtType,omitempty"`
	Error    string   `json:"error,omitempty"`
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

	// Extract info using visitor pattern
	extractor := &InfoExtractor{}
	stmtNodes[0].Accept(extractor)

	// Determine statement type
	stmtType := "UNKNOWN"
	if len(stmtNodes) > 0 {
		t := reflect.TypeOf(stmtNodes[0])
		if t.Kind() == reflect.Ptr {
			stmtType = t.Elem().Name()
		} else {
			stmtType = t.Name()
		}
		stmtType = strings.TrimSuffix(stmtType, "Stmt")
		stmtType = strings.ToUpper(stmtType)
	}

	result := ParseResult{
		Success:  true,
		Columns:  extractor.ColNames,
		Tables:   extractor.TableNames,
		StmtType: stmtType,
	}

	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}

// version returns the version of the parser
func version(this js.Value, args []js.Value) interface{} {
	return "MySQL Parser WASM Demo v3.1.0 (Enhanced Parser Driver with Full Types)"
}

func main() {
	fmt.Println("MySQL Parser WASM initialized")

	// Register functions to be called from JavaScript
	js.Global().Set("parseSQL", js.FuncOf(parseSQL))
	js.Global().Set("parserVersion", js.FuncOf(version))

	// Keep the program running
	select {}
}
