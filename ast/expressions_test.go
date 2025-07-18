// Copyright 2017 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package ast_test

import (
	"testing"

	. "github.com/abbychau/mysql-parser/ast"
	"github.com/abbychau/mysql-parser/format"
	"github.com/abbychau/mysql-parser/mysql"
	"github.com/stretchr/testify/require"
)

type checkVisitor struct{}

func (v checkVisitor) Enter(in Node) (Node, bool) {
	if e, ok := in.(*checkExpr); ok {
		e.enterCnt++
		return in, true
	}
	return in, false
}

func (v checkVisitor) Leave(in Node) (Node, bool) {
	if e, ok := in.(*checkExpr); ok {
		e.leaveCnt++
	}
	return in, true
}

type checkExpr struct {
	ValueExpr

	enterCnt int
	leaveCnt int
}

func (n *checkExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*checkExpr)
	return v.Leave(n)
}

func (n *checkExpr) reset() {
	n.enterCnt = 0
	n.leaveCnt = 0
}

func TestExpresionsVisitorCover(t *testing.T) {
	ce := &checkExpr{}
	stmts :=
		[]struct {
			node             Node
			expectedEnterCnt int
			expectedLeaveCnt int
		}{
			{&BetweenExpr{Expr: ce, Left: ce, Right: ce}, 3, 3},
			{&BinaryOperationExpr{L: ce, R: ce}, 2, 2},
			{&CaseExpr{Value: ce, WhenClauses: []*WhenClause{{Expr: ce, Result: ce},
				{Expr: ce, Result: ce}}, ElseClause: ce}, 6, 6},
			{&ColumnNameExpr{Name: &ColumnName{}}, 0, 0},
			{&CompareSubqueryExpr{L: ce, R: ce}, 2, 2},
			{&DefaultExpr{Name: &ColumnName{}}, 0, 0},
			{&ExistsSubqueryExpr{Sel: ce}, 1, 1},
			{&IsNullExpr{Expr: ce}, 1, 1},
			{&IsTruthExpr{Expr: ce}, 1, 1},
			{NewParamMarkerExpr(0), 0, 0},
			{&ParenthesesExpr{Expr: ce}, 1, 1},
			{&PatternInExpr{Expr: ce, List: []ExprNode{ce, ce, ce}, Sel: ce}, 5, 5},
			{&PatternLikeOrIlikeExpr{Expr: ce, Pattern: ce}, 2, 2},
			{&PatternRegexpExpr{Expr: ce, Pattern: ce}, 2, 2},
			{&PositionExpr{}, 0, 0},
			{&RowExpr{Values: []ExprNode{ce, ce}}, 2, 2},
			{&UnaryOperationExpr{V: ce}, 1, 1},
			{NewValueExpr(0, mysql.DefaultCharset, mysql.DefaultCollationName), 0, 0},
			{&ValuesExpr{Column: &ColumnNameExpr{Name: &ColumnName{}}}, 0, 0},
			{&VariableExpr{Value: ce}, 1, 1},
		}

	for _, v := range stmts {
		ce.reset()
		v.node.Accept(checkVisitor{})
		require.Equal(t, v.expectedEnterCnt, ce.enterCnt)
		require.Equal(t, v.expectedLeaveCnt, ce.leaveCnt)
		v.node.Accept(visitor1{})
	}
}

func TestUnaryOperationExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"++1", "++1"},
		{"--1", "--1"},
		{"-+1", "-+1"},
		{"-1", "-1"},
		{"not true", "NOT TRUE"},
		{"~3", "~3"},
		{"!true", "!TRUE"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	runNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestColumnNameExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"abc", "`abc`"},
		{"`abc`", "`abc`"},
		{"`ab``c`", "`ab``c`"},
		{"sabc.tABC", "`sabc`.`tABC`"},
		{"dabc.sabc.tabc", "`dabc`.`sabc`.`tabc`"},
		{"dabc.`sabc`.tabc", "`dabc`.`sabc`.`tabc`"},
		{"`dABC`.`sabc`.tabc", "`dABC`.`sabc`.`tabc`"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	runNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestIsNullExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"a is null", "`a` IS NULL"},
		{"a is not null", "`a` IS NOT NULL"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	runNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestIsTruthRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"a is true", "`a` IS TRUE"},
		{"a is not true", "`a` IS NOT TRUE"},
		{"a is FALSE", "`a` IS FALSE"},
		{"a is not false", "`a` IS NOT FALSE"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	runNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestBetweenExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"b between 1 and 2", "`b` BETWEEN 1 AND 2"},
		{"b not between 1 and 2", "`b` NOT BETWEEN 1 AND 2"},
		{"b between a and b", "`b` BETWEEN `a` AND `b`"},
		{"b between '' and 'b'", "`b` BETWEEN _UTF8MB4'' AND _UTF8MB4'b'"},
		{"b between '2018-11-01' and '2018-11-02'", "`b` BETWEEN _UTF8MB4'2018-11-01' AND _UTF8MB4'2018-11-02'"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	runNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestCaseExpr(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"case when 1 then 2 end", "CASE WHEN 1 THEN 2 END"},
		{"case when 1 then 'a' when 2 then 'b' end", "CASE WHEN 1 THEN _UTF8MB4'a' WHEN 2 THEN _UTF8MB4'b' END"},
		{"case when 1 then 'a' when 2 then 'b' else 'c' end", "CASE WHEN 1 THEN _UTF8MB4'a' WHEN 2 THEN _UTF8MB4'b' ELSE _UTF8MB4'c' END"},
		{"case when 'a'!=1 then true else false end", "CASE WHEN _UTF8MB4'a'!=1 THEN TRUE ELSE FALSE END"},
		{"case a when 'a' then true else false end", "CASE `a` WHEN _UTF8MB4'a' THEN TRUE ELSE FALSE END"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	runNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestBinaryOperationExpr(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"'a'!=1", "_UTF8MB4'a'!=1"},
		{"a!=1", "`a`!=1"},
		{"3<5", "3<5"},
		{"10>5", "10>5"},
		{"3+5", "3+5"},
		{"3-5", "3-5"},
		{"a<>5", "`a`!=5"},
		{"a=1", "`a`=1"},
		{"a mod 2", "`a`%2"},
		{"a div 2", "`a` DIV 2"},
		{"true and true", "TRUE AND TRUE"},
		{"false or false", "FALSE OR FALSE"},
		{"true xor false", "TRUE XOR FALSE"},
		{"3 & 4", "3&4"},
		{"5 | 6", "5|6"},
		{"7 ^ 8", "7^8"},
		{"9 << 10", "9<<10"},
		{"11 >> 12", "11>>12"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	runNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestBinaryOperationExprWithFlags(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"'a'!=1", "_UTF8MB4'a' != 1"},
		{"a!=1", "`a` != 1"},
		{"3<5", "3 < 5"},
		{"10>5", "10 > 5"},
		{"3+5", "3 + 5"},
		{"3-5", "3 - 5"},
		{"a<>5", "`a` != 5"},
		{"a=1", "`a` = 1"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	flags := format.DefaultRestoreFlags | format.RestoreSpacesAroundBinaryOperation
	runNodeRestoreTestWithFlags(t, testCases, "select %s", extractNodeFunc, flags)
}

func TestParenthesesExpr(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"(1+2)*3", "(1+2)*3"},
		{"1+2*3", "1+2*3"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	runNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestWhenClause(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"when 1 then 2", "WHEN 1 THEN 2"},
		{"when 1 then 'a'", "WHEN 1 THEN _UTF8MB4'a'"},
		{"when 'a'!=1 then true", "WHEN _UTF8MB4'a'!=1 THEN TRUE"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr.(*CaseExpr).WhenClauses[0]
	}
	runNodeRestoreTest(t, testCases, "select case %s end", extractNodeFunc)
}

func TestDefaultExpr(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"default", "DEFAULT"},
		{"default(i)", "DEFAULT(`i`)"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*InsertStmt).Lists[0][0]
	}
	runNodeRestoreTest(t, testCases, "insert into t values(%s)", extractNodeFunc)
}

func TestPatternInExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"'a' in ('b')", "_UTF8MB4'a' IN (_UTF8MB4'b')"},
		{"2 in (0,3,7)", "2 IN (0,3,7)"},
		{"2 not in (0,3,7)", "2 NOT IN (0,3,7)"},
		{"2 in (select 2)", "2 IN (SELECT 2)"},
		{"2 not in (select 2)", "2 NOT IN (SELECT 2)"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	runNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestPatternLikeExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"a like 't1'", "`a` LIKE _UTF8MB4't1'"},
		{"a like 't1%'", "`a` LIKE _UTF8MB4't1%'"},
		{"a like '%t1%'", "`a` LIKE _UTF8MB4'%t1%'"},
		{"a like '%t1_|'", "`a` LIKE _UTF8MB4'%t1_|'"},
		{"a not like 't1'", "`a` NOT LIKE _UTF8MB4't1'"},
		{"a not like 't1%'", "`a` NOT LIKE _UTF8MB4't1%'"},
		{"a not like '%D%v%'", "`a` NOT LIKE _UTF8MB4'%D%v%'"},
		{"a not like '%t1_|'", "`a` NOT LIKE _UTF8MB4'%t1_|'"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	runNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestValuesExpr(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"values(a)", "VALUES(`a`)"},
		{"values(a)+values(b)", "VALUES(`a`)+VALUES(`b`)"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*InsertStmt).OnDuplicate[0].Expr
	}
	runNodeRestoreTest(t, testCases, "insert into t values (1,2,3) on duplicate key update c=%s", extractNodeFunc)
}

func TestPatternRegexpExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"a regexp 't1'", "`a` REGEXP _UTF8MB4't1'"},
		{"a regexp '^[abc][0-9]{11}|ok$'", "`a` REGEXP _UTF8MB4'^[abc][0-9]{11}|ok$'"},
		{"a rlike 't1'", "`a` REGEXP _UTF8MB4't1'"},
		{"a rlike '^[abc][0-9]{11}|ok$'", "`a` REGEXP _UTF8MB4'^[abc][0-9]{11}|ok$'"},
		{"a not regexp 't1'", "`a` NOT REGEXP _UTF8MB4't1'"},
		{"a not regexp '^[abc][0-9]{11}|ok$'", "`a` NOT REGEXP _UTF8MB4'^[abc][0-9]{11}|ok$'"},
		{"a not rlike 't1'", "`a` NOT REGEXP _UTF8MB4't1'"},
		{"a not rlike '^[abc][0-9]{11}|ok$'", "`a` NOT REGEXP _UTF8MB4'^[abc][0-9]{11}|ok$'"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	runNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestRowExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"(1,2)", "ROW(1,2)"},
		{"(col1,col2)", "ROW(`col1`,`col2`)"},
		{"row(1,2)", "ROW(1,2)"},
		{"row(col1,col2)", "ROW(`col1`,`col2`)"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Where.(*BinaryOperationExpr).L
	}
	runNodeRestoreTest(t, testCases, "select 1 from t1 where %s = row(1,2)", extractNodeFunc)
}

func TestMaxValueExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"maxvalue", "MAXVALUE"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*AlterTableStmt).Specs[0].PartDefinitions[0].Clause.(*PartitionDefinitionClauseLessThan).Exprs[0]
	}
	runNodeRestoreTest(t, testCases, "alter table posts add partition ( partition p1 values less than %s)", extractNodeFunc)
}

func TestPositionExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"1", "1"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).OrderBy.Items[0]
	}
	runNodeRestoreTest(t, testCases, "select * from t order by %s", extractNodeFunc)
}

func TestExistsSubqueryExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"EXISTS (SELECT 2)", "EXISTS (SELECT 2)"},
		{"NOT EXISTS (SELECT 2)", "NOT EXISTS (SELECT 2)"},
		{"NOT NOT EXISTS (SELECT 2)", "EXISTS (SELECT 2)"},
		{"NOT NOT NOT EXISTS (SELECT 2)", "NOT EXISTS (SELECT 2)"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Where
	}
	runNodeRestoreTest(t, testCases, "select 1 from t1 where %s", extractNodeFunc)
}

func TestVariableExpr(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"@a>1", "@`a`>1"},
		{"@`aB`+1", "@`aB`+1"},
		{"@'a':=1", "@`a`:=1"},
		{"@`a``b`=4", "@`a``b`=4"},
		{`@"aBC">1`, "@`aBC`>1"},
		{"@`a`+1", "@`a`+1"},
		{"@``", "@``"},
		{"@", "@``"},
		{"@@``", "@@``"},
		{"@@var", "@@`var`"},
		{"@@global.b='foo'", "@@GLOBAL.`b`=_UTF8MB4'foo'"},
		{"@@session.'C'", "@@SESSION.`c`"},
		{`@@local."aBc"`, "@@SESSION.`abc`"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	runNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestMatchAgainstExpr(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{`MATCH(content, title) AGAINST ('search for')`, "MATCH (`content`,`title`) AGAINST (_UTF8MB4'search for')"},
		{`MATCH(content) AGAINST ('search for' IN BOOLEAN MODE)`, "MATCH (`content`) AGAINST (_UTF8MB4'search for' IN BOOLEAN MODE)"},
		{`MATCH(content, title) AGAINST ('search for' WITH QUERY EXPANSION)`, "MATCH (`content`,`title`) AGAINST (_UTF8MB4'search for' WITH QUERY EXPANSION)"},
		{`MATCH(content) AGAINST ('search for' IN NATURAL LANGUAGE MODE WITH QUERY EXPANSION)`, "MATCH (`content`) AGAINST (_UTF8MB4'search for' WITH QUERY EXPANSION)"},
		{`MATCH(content) AGAINST ('search') AND id = 1`, "MATCH (`content`) AGAINST (_UTF8MB4'search') AND `id`=1"},
		{`MATCH(content) AGAINST ('search') OR id = 1`, "MATCH (`content`) AGAINST (_UTF8MB4'search') OR `id`=1"},
		{`MATCH(content) AGAINST (X'40404040' | X'01020304') OR id = 1`, "MATCH (`content`) AGAINST (x'40404040'|x'01020304') OR `id`=1"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Where
	}
	runNodeRestoreTest(t, testCases, "SELECT * FROM t WHERE %s", extractNodeFunc)
}
