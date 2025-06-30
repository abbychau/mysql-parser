// Package parser_driver_init initializes the parser_driver for the main parser
package parser

import (
	"fmt"
	
	"github.com/abbychau/mysql-parser/ast"
	"github.com/abbychau/mysql-parser/charset"
	myformat "github.com/abbychau/mysql-parser/format"
	"github.com/abbychau/mysql-parser/parser_driver"
	"github.com/abbychau/mysql-parser/types"
)

func init() {
	// Register the parser driver functions with the AST
	ast.NewValueExpr = func(value interface{}, charset string, collate string) ast.ValueExpr {
		// Create using the parser_driver but return as ast.ValueExpr interface
		ve := parser_driver.NewValueExpr(value, charset, collate)
		if pvExpr, ok := ve.(*parser_driver.ValueExpr); ok {
			return &astValueExpr{pvExpr}
		}
		return nil
	}
	
	ast.NewParamMarkerExpr = func(offset int) ast.ParamMarkerExpr {
		// Create using the parser_driver but return as ast.ParamMarkerExpr interface
		pme := parser_driver.NewParamMarkerExpr(offset)
		if ppExpr, ok := pme.(*parser_driver.ParamMarkerExpr); ok {
			return &astParamMarkerExpr{ppExpr}
		}
		return nil
	}
	
	ast.NewDecimal = func(str string) (interface{}, error) {
		return parser_driver.NewDecimal(str)
	}
	ast.NewHexLiteral = func(str string) (interface{}, error) {
		return parser_driver.NewHexLiteral(str)
	}
	ast.NewBitLiteral = func(str string) (interface{}, error) {
		return parser_driver.NewBitLiteral(str)
	}
}

// astValueExpr wraps parser_driver.ValueExpr to implement ast.ValueExpr
type astValueExpr struct {
	*parser_driver.ValueExpr
}

// Implement ast.ValueExpr interface methods that aren't in parser_driver.ValueExpr
func (v *astValueExpr) Accept(visitor ast.Visitor) (ast.Node, bool) {
	// Simple implementation for AST traversal
	newNode, skipChildren := visitor.Enter(v)
	if skipChildren {
		return visitor.Leave(newNode)
	}
	return visitor.Leave(newNode)
}

func (v *astValueExpr) Restore(ctx *myformat.RestoreCtx) error {
	// Simple restore implementation - just write the value
	switch v.Kind() {
	case parser_driver.KindNull:
		ctx.WriteKeyWord("NULL")
	case parser_driver.KindInt64:
		ctx.WritePlain(fmt.Sprintf("%d", v.GetInt64()))
	case parser_driver.KindUint64:
		ctx.WritePlain(fmt.Sprintf("%d", v.GetUint64()))
	case parser_driver.KindFloat32, parser_driver.KindFloat64:
		ctx.WritePlain(fmt.Sprintf("%g", v.GetFloat64()))
	case parser_driver.KindString:
		ctx.WriteString(v.GetString())
	default:
		ctx.WritePlain(v.GetString())
	}
	return nil
}

func (v *astValueExpr) Text() string {
	return v.GetString()
}

func (v *astValueExpr) OriginalText() string {
	return v.GetString() 
}

func (v *astValueExpr) SetText(enc charset.Encoding, text string) {
	// No-op for simplicity
}

func (v *astValueExpr) SetOriginTextPosition(offset int) {
	// No-op for simplicity
}

func (v *astValueExpr) OriginTextPosition() int {
	return 0
}

func (v *astValueExpr) GetFlag() uint64 {
	return 0
}

func (v *astValueExpr) SetFlag(flag uint64) {
	// No-op for simplicity
}

func (v *astValueExpr) GetType() *types.FieldType {
	// Convert from parser_driver.FieldType to types.FieldType
	// For simplicity, create a basic FieldType
	return &types.FieldType{}
}

func (v *astValueExpr) SetType(tp *types.FieldType) {
	// No-op for simplicity
}

// astParamMarkerExpr wraps parser_driver.ParamMarkerExpr to implement ast.ParamMarkerExpr
type astParamMarkerExpr struct {
	*parser_driver.ParamMarkerExpr
}

func (p *astParamMarkerExpr) Accept(visitor ast.Visitor) (ast.Node, bool) {
	newNode, skipChildren := visitor.Enter(p)
	if skipChildren {
		return visitor.Leave(newNode)
	}
	return visitor.Leave(newNode)
}

func (p *astParamMarkerExpr) Restore(ctx *myformat.RestoreCtx) error {
	ctx.WritePlain("?")
	return nil
}

func (p *astParamMarkerExpr) Text() string {
	return "?"
}

func (p *astParamMarkerExpr) OriginalText() string {
	return "?"
}

func (p *astParamMarkerExpr) SetText(enc charset.Encoding, text string) {
	// No-op
}

func (p *astParamMarkerExpr) SetOriginTextPosition(offset int) {
	// No-op
}

func (p *astParamMarkerExpr) OriginTextPosition() int {
	return 0
}

func (p *astParamMarkerExpr) GetFlag() uint64 {
	return 0
}

func (p *astParamMarkerExpr) SetFlag(flag uint64) {
	// No-op for simplicity
}

func (p *astParamMarkerExpr) GetType() *types.FieldType {
	return &types.FieldType{}
}

func (p *astParamMarkerExpr) SetType(tp *types.FieldType) {
	// No-op for simplicity
}