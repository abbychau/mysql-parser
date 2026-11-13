package ast

import (
	"github.com/abbychau/mysql-parser/format"
	"github.com/abbychau/mysql-parser/types"
	"github.com/pingcap/errors"
)

var _ ResultSetNode = &JSONTable{}

// JSONTable is the JSON_TABLE table function used as a table reference.
// See https://dev.mysql.com/doc/refman/8.0/en/json-table-functions.html
// and https://mariadb.com/kb/en/json_table/
//
//	JSON_TABLE(expr, path COLUMNS (column_list)) [AS] alias
type JSONTable struct {
	node

	Expr    ExprNode
	Path    string
	Columns []*JSONTableColumn
}

func (*JSONTable) resultSet() {}

// Restore implements Node interface.
func (n *JSONTable) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("JSON_TABLE")
	ctx.WritePlain("(")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore JSONTable.Expr")
	}
	ctx.WritePlain(", ")
	ctx.WriteString(n.Path)
	if err := restoreJSONTableColumns(ctx, n.Columns); err != nil {
		return err
	}
	ctx.WritePlain(")")
	return nil
}

// Accept implements Node Accept interface.
func (n *JSONTable) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*JSONTable)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	return v.Leave(n)
}

// JSONTableColumnType is the kind of a JSON_TABLE column.
type JSONTableColumnType int

// JSONTableColumnType values.
const (
	// JSONTableColumnPath is "name type PATH path [on_empty] [on_error]".
	JSONTableColumnPath JSONTableColumnType = iota + 1
	// JSONTableColumnOrdinality is "name FOR ORDINALITY".
	JSONTableColumnOrdinality
	// JSONTableColumnExists is "name type EXISTS PATH path".
	JSONTableColumnExists
	// JSONTableColumnNested is "NESTED [PATH] path COLUMNS (column_list)".
	JSONTableColumnNested
)

// JSONTableResponseType is the action of an ON EMPTY or ON ERROR clause.
type JSONTableResponseType int

// JSONTableResponseType values.
const (
	JSONTableResponseNull JSONTableResponseType = iota + 1
	JSONTableResponseError
	JSONTableResponseDefault
)

// JSONTableResponse is an ON EMPTY or ON ERROR clause of a JSON_TABLE column.
type JSONTableResponse struct {
	Tp JSONTableResponseType
	// Default is the value of DEFAULT 'value'.
	Default string
}

func (r *JSONTableResponse) restore(ctx *format.RestoreCtx, on string) {
	ctx.WritePlain(" ")
	switch r.Tp {
	case JSONTableResponseNull:
		ctx.WriteKeyWord("NULL")
	case JSONTableResponseError:
		ctx.WriteKeyWord("ERROR")
	case JSONTableResponseDefault:
		ctx.WriteKeyWord("DEFAULT ")
		ctx.WriteString(r.Default)
	}
	ctx.WriteKeyWord(" ON " + on)
}

// JSONTableColumn is a column definition of JSON_TABLE.
type JSONTableColumn struct {
	Tp            JSONTableColumnType
	Name          CIStr
	FieldType     *types.FieldType
	Path          string
	OnEmpty       *JSONTableResponse
	OnError       *JSONTableResponse
	NestedColumns []*JSONTableColumn
}

// Restore writes the column definition.
func (c *JSONTableColumn) Restore(ctx *format.RestoreCtx) error {
	if c.Tp == JSONTableColumnNested {
		ctx.WriteKeyWord("NESTED PATH ")
		ctx.WriteString(c.Path)
		return restoreJSONTableColumns(ctx, c.NestedColumns)
	}
	ctx.WriteName(c.Name.O)
	if c.Tp == JSONTableColumnOrdinality {
		ctx.WriteKeyWord(" FOR ORDINALITY")
		return nil
	}
	ctx.WritePlain(" ")
	if err := c.FieldType.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore JSONTableColumn.FieldType")
	}
	if c.Tp == JSONTableColumnExists {
		ctx.WriteKeyWord(" EXISTS")
	}
	ctx.WriteKeyWord(" PATH ")
	ctx.WriteString(c.Path)
	if c.OnEmpty != nil {
		c.OnEmpty.restore(ctx, "EMPTY")
	}
	if c.OnError != nil {
		c.OnError.restore(ctx, "ERROR")
	}
	return nil
}

func restoreJSONTableColumns(ctx *format.RestoreCtx, cols []*JSONTableColumn) error {
	ctx.WriteKeyWord(" COLUMNS ")
	ctx.WritePlain("(")
	for i, c := range cols {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := c.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	return nil
}
