package main

import (
	"fmt"
	"os"

	"github.com/abbychau/mysql-parser"
	"github.com/abbychau/mysql-parser/ast"
	_ "github.com/abbychau/mysql-parser/parser_driver"
)

// ColumnExtractor implements ast.Visitor to extract column names
type ColumnExtractor struct {
	ColNames []string
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

func main() {
	testSQL := "SELECT name, age, email FROM users WHERE age > 25 ORDER BY name"
	if len(os.Args) > 1 {
		testSQL = os.Args[1]
	}

	fmt.Printf("Testing SQL: %s\n", testSQL)

	// Create parser instance
	p := parser.New()

	// Parse the SQL
	stmtNodes, _, err := p.ParseSQL(testSQL)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		return
	}

	if len(stmtNodes) == 0 {
		fmt.Println("No statements found")
		return
	}

	// Extract columns using visitor pattern
	extractor := &ColumnExtractor{}
	stmtNodes[0].Accept(extractor)

	fmt.Printf("✓ Successfully parsed!\n")
	fmt.Printf("Extracted columns: %v\n", extractor.ColNames)
	fmt.Printf("Total columns found: %d\n", len(extractor.ColNames))
}