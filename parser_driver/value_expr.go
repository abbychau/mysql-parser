// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser_driver

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Initialize the parser driver functions for AST integration
func init() {
	// These will be set by the main parser module when it imports this driver
	// We use a simple function-based approach to avoid circular dependencies
}

// ValueExpr is the simple value expression for WASM-compatible driver
type ValueExpr struct {
	Datum
	Type             FieldType
	projectionOffset int
}

// SetValue implements interface of ValueExpr
func (n *ValueExpr) SetValue(res interface{}) {
	n.Datum.SetValueWithDefaultCollation(res)
}

// GetDatumString implements the ValueExpr interface
func (n *ValueExpr) GetDatumString() string {
	return n.GetString()
}

// Format the ExprNode into a Writer
func (n *ValueExpr) Format(w io.Writer) {
	var s string
	switch n.Kind() {
	case KindNull:
		s = "NULL"
	case KindInt64:
		s = strconv.FormatInt(n.GetInt64(), 10)
	case KindUint64:
		s = strconv.FormatUint(n.GetUint64(), 10)
	case KindFloat32:
		s = strconv.FormatFloat(n.GetFloat64(), 'e', -1, 32)
	case KindFloat64:
		s = strconv.FormatFloat(n.GetFloat64(), 'e', -1, 64)
	case KindString, KindBytes:
		s = WrapInSingleQuotes(n.GetString())
	case KindMysqlDecimal:
		s = n.GetMysqlDecimal().String()
	default:
		panic("Can't format to string")
	}
	fmt.Fprint(w, s)
}

// WrapInSingleQuotes escapes single quotes and backslashes and adds single quotes around the string
func WrapInSingleQuotes(inStr string) string {
	s := strings.ReplaceAll(inStr, "\\", "\\\\")
	s = strings.ReplaceAll(s, `'`, `''`)
	return fmt.Sprintf("'%s'", s)
}

// UnwrapFromSingleQuotes the reverse of WrapInSingleQuotes
func UnwrapFromSingleQuotes(inStr string) string {
	if len(inStr) < 2 || inStr[:1] != "'" || inStr[len(inStr)-1:] != "'" {
		return inStr
	}
	s := strings.ReplaceAll(inStr[1:len(inStr)-1], "\\\\", "\\")
	return strings.ReplaceAll(s, `''`, `'`)
}

// SetProjectionOffset sets ValueExpr.projectionOffset for logical plan builder
func (n *ValueExpr) SetProjectionOffset(offset int) {
	n.projectionOffset = offset
}

// GetProjectionOffset returns ValueExpr.projectionOffset
func (n *ValueExpr) GetProjectionOffset() int {
	return n.projectionOffset
}

// ParamMarkerExpr expression holds a place for another expression
type ParamMarkerExpr struct {
	ValueExpr
	Offset    int
	Order     int
	InExecute bool
	UseAsValueInGbyByClause bool
}

// SetOrder implements the ParamMarkerExpr interface
func (n *ParamMarkerExpr) SetOrder(order int) {
	n.Order = order
}

// NewValueExpr creates a ValueExpr with value, and sets default field type
func NewValueExpr(value interface{}, charset string, collate string) interface{} {
	if ve, ok := value.(*ValueExpr); ok {
		return ve
	}
	ve := &ValueExpr{}
	
	// Set default type for value
	DefaultTypeForValue(value, &ve.Type, charset, collate)
	ve.SetValue(value)
	ve.projectionOffset = -1
	return ve
}

// NewParamMarkerExpr creates a new parameter marker expression
func NewParamMarkerExpr(offset int) interface{} {
	return &ParamMarkerExpr{
		Offset: offset,
	}
}

// NewDecimal creates a new decimal value
func NewDecimal(str string) (interface{}, error) {
	dec := new(MyDecimal)
	err := dec.FromString([]byte(str))
	if err == ErrTruncated {
		err = nil
	}
	return dec, err
}

// NOTE: NewHexLiteral and NewBitLiteral are commented out to avoid duplicates with binary_literal.go
// NewHexLiteral creates a new hex literal
// func NewHexLiteral(str string) (interface{}, error) {
// 	h, err := NewHexLiteral(str)
// 	return h, err
// }

// NewBitLiteral creates a new bit literal
// func NewBitLiteral(str string) (interface{}, error) {
// 	b, err := NewBitLiteral(str)
// 	return b, err
// }