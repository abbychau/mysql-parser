# MySQL Parser

A MySQL-compatible SQL parser extracted from TiDB, now available as a standalone Go module with WASM support.

## Features

- **MySQL Compatibility**: Supports MySQL 5.7+ SQL syntax
- **Full AST Support**: Complete Abstract Syntax Tree generation
- **Visitor Pattern**: Easy AST traversal and manipulation
- **WASM Support**: Runs in WebAssembly environments
- **Type Safety**: Proper handling of MySQL data types
- **Cross-Platform**: Optimized for both native and WASM execution

## Module Information

**Module Path:** `github.com/abbychau/mysql-parser`

This parser provides MySQL-compatible SQL parsing with full AST support and two parser driver options:

1. **Test Driver** (`test_driver`) - Lightweight, basic functionality
2. **Full Driver** (`parser_driver`) - Complete functionality with WASM compatibility

## Installation

```bash
go get github.com/abbychau/mysql-parser
```

## Usage Examples

### Basic Usage with Test Driver

```go
package main

import (
    "fmt"
    "github.com/abbychau/mysql-parser"
    "github.com/abbychau/mysql-parser/ast"
    _ "github.com/abbychau/mysql-parser/test_driver"
)

func main() {
    p := parser.New()
    sql := "SELECT name, age FROM users WHERE age > 25"
    
    stmtNodes, _, err := p.ParseSQL(sql)
    if err != nil {
        fmt.Printf("Parse error: %v\n", err)
        return
    }
    
    fmt.Printf("Successfully parsed %d statements\n", len(stmtNodes))
}
```

### Advanced Usage with Full Parser Driver

```go
package main

import (
    "fmt"
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
    p := parser.New()
    sql := "SELECT users.name, orders.total FROM users JOIN orders ON users.id = orders.user_id"
    
    stmtNodes, _, err := p.ParseSQL(sql)
    if err != nil {
        fmt.Printf("Parse error: %v\n", err)
        return
    }

    // Extract column names using visitor pattern
    extractor := &ColumnExtractor{}
    stmtNodes[0].Accept(extractor)
    
    fmt.Printf("Extracted columns: %v\n", extractor.ColNames)
}
```

### WASM Usage

For WebAssembly applications, use the full parser driver:

```go
package main

import (
    "syscall/js"
    "github.com/abbychau/mysql-parser"
    "github.com/abbychau/mysql-parser/ast"
    _ "github.com/abbychau/mysql-parser/parser_driver"
)

func parseSQL(this js.Value, args []js.Value) interface{} {
    sqlText := args[0].String()
    
    p := parser.New()
    stmtNodes, _, err := p.ParseSQL(sqlText)
    if err != nil {
        return map[string]interface{}{
            "success": false,
            "error":   err.Error(),
        }
    }
    
    return map[string]interface{}{
        "success": true,
        "count":   len(stmtNodes),
    }
}

func main() {
    js.Global().Set("parseSQL", js.FuncOf(parseSQL))
    select {} // Keep the program running
}
```

## Project Structure

```
github.com/abbychau/mysql-parser/
├── ast/                    # Abstract Syntax Tree definitions
├── charset/               # Character set support
├── mysql/                 # MySQL constants and utilities
├── parser_driver/         # Full parser driver (WASM-compatible)
├── test_driver/          # Lightweight test driver
├── wasm-demo/            # WebAssembly demo application
└── docs/                 # Documentation
```

## Parser Drivers

### Test Driver (`test_driver`)
- Lightweight and minimal dependencies
- Basic AST node support
- Suitable for simple parsing tasks
- Import: `_ "github.com/abbychau/mysql-parser/test_driver"`

### Full Parser Driver (`parser_driver`)
- Complete functionality with proper type handling
- WASM-compatible (no CGO or unsafe operations)
- Supports all MySQL data types and expressions
- Full type system including Datum, FieldType, MyDecimal, Time
- Comprehensive error handling and type conversions
- Self-contained with no external dependencies
- Import: `_ "github.com/abbychau/mysql-parser/parser_driver"`

## Building for WASM

```bash
cd wasm-demo
GOOS=js GOARCH=wasm go build -o parser.wasm main.go
```

The `parser_driver` package is specifically designed to work in WASM environments. The demo includes a web interface for testing SQL parsing in the browser.



## License

Apache License 2.0 - see LICENSE file for details.

## Original Source

This parser is based on TiDB's SQL parser component, modified for standalone use and WASM compatibility.