package ast

import (
	"github.com/abbychau/mysql-parser/format"
)

var _ StmtNode = &CreateLoadableFunctionStmt{}

// CreateLoadableFunctionStmt is a statement to install a loadable function
// (user-defined function implemented in a shared library).
// See https://dev.mysql.com/doc/refman/8.0/en/create-function-loadable.html
// and https://mariadb.com/kb/en/create-function-udf/
//
//	CREATE [OR REPLACE] [AGGREGATE] FUNCTION [IF NOT EXISTS] function_name
//	    RETURNS {STRING | INTEGER | REAL | DECIMAL}
//	    SONAME shared_library_name
type CreateLoadableFunctionStmt struct {
	ddlNode

	OrReplace    bool
	Aggregate    bool
	IfNotExists  bool
	FunctionName CIStr
	// Returns is the return type keyword in upper case.
	Returns string
	SoName  string
}

// Restore implements Node interface.
func (n *CreateLoadableFunctionStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("CREATE ")
	if n.OrReplace {
		ctx.WriteKeyWord("OR REPLACE ")
	}
	if n.Aggregate {
		ctx.WriteKeyWord("AGGREGATE ")
	}
	ctx.WriteKeyWord("FUNCTION ")
	if n.IfNotExists {
		ctx.WriteKeyWord("IF NOT EXISTS ")
	}
	ctx.WriteName(n.FunctionName.O)
	ctx.WriteKeyWord(" RETURNS ")
	ctx.WriteKeyWord(n.Returns)
	ctx.WriteKeyWord(" SONAME ")
	ctx.WriteString(n.SoName)
	return nil
}

// Accept implements Node Accept interface.
func (n *CreateLoadableFunctionStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(newNode.(*CreateLoadableFunctionStmt))
}
