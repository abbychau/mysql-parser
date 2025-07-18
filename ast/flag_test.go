// Copyright 2016 PingCAP, Inc.
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

	"github.com/abbychau/mysql-parser"
	"github.com/abbychau/mysql-parser/ast"
	"github.com/stretchr/testify/require"
)

func TestHasAggFlag(t *testing.T) {
	expr := &ast.BetweenExpr{}
	flagTests := []struct {
		flag   uint64
		hasAgg bool
	}{
		{ast.FlagHasAggregateFunc, true},
		{ast.FlagHasAggregateFunc | ast.FlagHasVariable, true},
		{ast.FlagHasVariable, false},
	}
	for _, tt := range flagTests {
		expr.SetFlag(tt.flag)
		require.Equal(t, tt.hasAgg, ast.HasAggFlag(expr))
	}
}

func TestFlag(t *testing.T) {
	flagTests := []struct {
		expr string
		flag uint64
	}{
		{
			"1 between 0 and 2",
			ast.FlagConstant,
		},
		{
			"case 1 when 1 then 1 else 0 end",
			ast.FlagConstant,
		},
		{
			"case 1 when 1 then 1 else 0 end",
			ast.FlagConstant,
		},
		{
			"case 1 when a > 1 then 1 else 0 end",
			ast.FlagConstant | ast.FlagHasReference,
		},
		{
			"1 = ANY (select 1) OR exists (select 1)",
			ast.FlagHasSubquery,
		},
		{
			"1 in (1) or 1 is true or null is null or 'abc' like 'abc' or 'abc' rlike 'abc'",
			ast.FlagConstant,
		},
		{
			"row (1, 1) = row (1, 1)",
			ast.FlagConstant,
		},
		{
			"(1 + a) > ?",
			ast.FlagHasReference | ast.FlagHasParamMarker,
		},
		{
			"trim('abc ')",
			ast.FlagHasFunc,
		},
		{
			"now() + EXTRACT(YEAR FROM '2009-07-02') + CAST(1 AS UNSIGNED)",
			ast.FlagHasFunc,
		},
		{
			"substring('abc', 1)",
			ast.FlagHasFunc,
		},
		{
			"sum(a)",
			ast.FlagHasAggregateFunc | ast.FlagHasReference,
		},
		{
			"(select 1) as a",
			ast.FlagHasSubquery,
		},
		{
			"@auto_commit",
			ast.FlagHasVariable,
		},
		{
			"default(a)",
			ast.FlagHasDefault,
		},
		{
			"a is null",
			ast.FlagHasReference,
		},
		{
			"1 is true",
			ast.FlagConstant,
		},
		{
			"a in (1, count(*), 3)",
			ast.FlagConstant | ast.FlagHasReference | ast.FlagHasAggregateFunc,
		},
		{
			"'Michael!' REGEXP '.*'",
			ast.FlagConstant,
		},
		{
			"a REGEXP '.*'",
			ast.FlagHasReference,
		},
		{
			"-a",
			ast.FlagHasReference,
		},
	}
	p := parser.New()
	for _, tt := range flagTests {
		stmt, err := p.ParseOneStmt("select "+tt.expr, "", "")
		require.NoError(t, err)
		selectStmt := stmt.(*ast.SelectStmt)
		ast.SetFlag(selectStmt)
		expr := selectStmt.Fields.Fields[0].Expr
		require.Equalf(t, tt.flag, expr.GetFlag(), "For %s", tt.expr)
	}
}
