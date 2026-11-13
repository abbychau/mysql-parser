package ast

import (
	"github.com/abbychau/mysql-parser/auth"
	"github.com/abbychau/mysql-parser/format"
	"github.com/pingcap/errors"
)

var (
	_ StmtNode = &CreateTriggerStmt{}
	_ StmtNode = &DropTriggerStmt{}
)

// TriggerTiming is the action time of a trigger (BEFORE or AFTER).
type TriggerTiming int

// TriggerTiming values.
const (
	TriggerBefore TriggerTiming = iota + 1
	TriggerAfter
)

// String implements fmt.Stringer.
func (t TriggerTiming) String() string {
	if t == TriggerAfter {
		return "AFTER"
	}
	return "BEFORE"
}

// TriggerEvent is the statement that activates a trigger.
type TriggerEvent int

// TriggerEvent values.
const (
	TriggerInsert TriggerEvent = iota + 1
	TriggerUpdate
	TriggerDelete
)

// String implements fmt.Stringer.
func (e TriggerEvent) String() string {
	switch e {
	case TriggerUpdate:
		return "UPDATE"
	case TriggerDelete:
		return "DELETE"
	default:
		return "INSERT"
	}
}

// TriggerOrder is the FOLLOWS/PRECEDES clause of CREATE TRIGGER.
type TriggerOrder struct {
	// Precedes is true for PRECEDES and false for FOLLOWS.
	Precedes     bool
	OtherTrigger CIStr
}

// CreateTriggerStmt is a statement to create a trigger.
// See https://dev.mysql.com/doc/refman/8.0/en/create-trigger.html
// and https://mariadb.com/kb/en/create-trigger/
//
//	CREATE [OR REPLACE] [DEFINER = user] TRIGGER [IF NOT EXISTS] trigger_name
//	    {BEFORE | AFTER} {INSERT | UPDATE | DELETE}
//	    ON tbl_name FOR EACH ROW
//	    [{FOLLOWS | PRECEDES} other_trigger_name]
//	    trigger_body
type CreateTriggerStmt struct {
	ddlNode

	OrReplace   bool
	Definer     *auth.UserIdentity
	IfNotExists bool
	TriggerName *TableName
	Timing      TriggerTiming
	Event       TriggerEvent
	Table       *TableName
	Order       *TriggerOrder
	Body        StmtNode
}

// Restore implements Node interface.
func (n *CreateTriggerStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("CREATE ")
	if n.OrReplace {
		ctx.WriteKeyWord("OR REPLACE ")
	}
	if n.Definer != nil && !n.Definer.CurrentUser {
		ctx.WriteKeyWord("DEFINER")
		ctx.WritePlain(" = ")
		ctx.WriteName(n.Definer.Username)
		if n.Definer.Hostname != "" {
			ctx.WritePlain("@")
			ctx.WriteName(n.Definer.Hostname)
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyWord("TRIGGER ")
	if n.IfNotExists {
		ctx.WriteKeyWord("IF NOT EXISTS ")
	}
	if err := n.TriggerName.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateTriggerStmt.TriggerName")
	}
	ctx.WritePlain(" ")
	ctx.WriteKeyWord(n.Timing.String())
	ctx.WritePlain(" ")
	ctx.WriteKeyWord(n.Event.String())
	ctx.WriteKeyWord(" ON ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateTriggerStmt.Table")
	}
	ctx.WriteKeyWord(" FOR EACH ROW ")
	if n.Order != nil {
		if n.Order.Precedes {
			ctx.WriteKeyWord("PRECEDES ")
		} else {
			ctx.WriteKeyWord("FOLLOWS ")
		}
		ctx.WriteName(n.Order.OtherTrigger.O)
		ctx.WritePlain(" ")
	}
	if err := n.Body.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateTriggerStmt.Body")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *CreateTriggerStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CreateTriggerStmt)
	node, ok := n.TriggerName.Accept(v)
	if !ok {
		return n, false
	}
	n.TriggerName = node.(*TableName)
	node, ok = n.Table.Accept(v)
	if !ok {
		return n, false
	}
	n.Table = node.(*TableName)
	node, ok = n.Body.Accept(v)
	if !ok {
		return n, false
	}
	n.Body = node.(StmtNode)
	return v.Leave(n)
}

// DropTriggerStmt is a statement to drop a trigger.
//
//	DROP TRIGGER [IF EXISTS] [schema_name.]trigger_name
type DropTriggerStmt struct {
	ddlNode

	IfExists    bool
	TriggerName *TableName
}

// Restore implements Node interface.
func (n *DropTriggerStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("DROP TRIGGER ")
	if n.IfExists {
		ctx.WriteKeyWord("IF EXISTS ")
	}
	if err := n.TriggerName.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore DropTriggerStmt.TriggerName")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *DropTriggerStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DropTriggerStmt)
	node, ok := n.TriggerName.Accept(v)
	if !ok {
		return n, false
	}
	n.TriggerName = node.(*TableName)
	return v.Leave(n)
}
