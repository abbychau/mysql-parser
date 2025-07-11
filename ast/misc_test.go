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
	"fmt"
	"testing"

	"github.com/abbychau/mysql-parser"
	"github.com/abbychau/mysql-parser/ast"
	"github.com/abbychau/mysql-parser/mysql"
	"github.com/stretchr/testify/require"
)

type visitor struct{}

func (v visitor) Enter(in ast.Node) (ast.Node, bool) {
	return in, false
}

func (v visitor) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

type visitor1 struct {
	visitor
}

func (visitor1) Enter(in ast.Node) (ast.Node, bool) {
	return in, true
}

func TestMiscVisitorCover(t *testing.T) {
	valueExpr := ast.NewValueExpr(42, mysql.DefaultCharset, mysql.DefaultCollationName)
	stmts := []ast.Node{
		&ast.AdminStmt{},
		&ast.AlterUserStmt{},
		&ast.BeginStmt{},
		&ast.BinlogStmt{},
		&ast.CommitStmt{},
		&ast.CompactTableStmt{Table: &ast.TableName{}},
		&ast.CreateUserStmt{},
		&ast.DeallocateStmt{},
		&ast.DoStmt{},
		&ast.ExecuteStmt{UsingVars: []ast.ExprNode{valueExpr}},
		&ast.ExplainStmt{Stmt: &ast.ShowStmt{}},
		&ast.GrantStmt{},
		&ast.PrepareStmt{SQLVar: &ast.VariableExpr{Value: valueExpr}},
		&ast.RollbackStmt{},
		&ast.SetPwdStmt{},
		&ast.SetStmt{Variables: []*ast.VariableAssignment{
			{
				Value: valueExpr,
			},
		}},
		&ast.UseStmt{},
		&ast.AnalyzeTableStmt{
			TableNames: []*ast.TableName{
				{},
			},
		},
		&ast.FlushStmt{},
		&ast.PrivElem{},
		&ast.VariableAssignment{Value: valueExpr},
		&ast.KillStmt{},
		&ast.DropStatsStmt{
			Tables: []*ast.TableName{
				{},
			},
		},
		&ast.ShutdownStmt{},
	}

	for _, v := range stmts {
		v.Accept(visitor{})
		v.Accept(visitor1{})
	}
}

func TestDDLVisitorCoverMisc(t *testing.T) {
	sql := `
create table t (c1 smallint unsigned, c2 int unsigned);
alter table t add column a smallint unsigned after b;
alter table t add column (a int, constraint check (a > 0));
create index t_i on t (id);
create database test character set utf8;
drop database test;
drop index t_i on t;
drop table t;
truncate t;
create table t (
jobAbbr char(4) not null,
constraint foreign key (jobabbr) references ffxi_jobtype (jobabbr) on delete cascade on update cascade
);
`
	parse := parser.New()
	stmts, _, err := parse.Parse(sql, "", "")
	require.NoError(t, err)
	for _, stmt := range stmts {
		stmt.Accept(visitor{})
		stmt.Accept(visitor1{})
	}
}

func TestDMLVistorCover(t *testing.T) {
	sql := `delete from somelog where user = 'jcole' order by timestamp_column limit 1;
delete t1, t2 from t1 inner join t2 inner join t3 where t1.id=t2.id and t2.id=t3.id;
select * from t where exists(select * from t k where t.c = k.c having sum(c) = 1);
insert into t_copy select * from t where t.x > 5;
(select /*+ TIDB_INLJ(t1) */ a from t1 where a=10 and b=1) union (select /*+ TIDB_SMJ(t2) */ a from t2 where a=11 and b=2) order by a limit 10;
update t1 set col1 = col1 + 1, col2 = col1;
show create table t;
load data infile '/tmp/t.csv' into table t fields terminated by 'ab' enclosed by 'b';
import into t from '/file.csv'`

	p := parser.New()
	stmts, _, err := p.Parse(sql, "", "")
	require.NoError(t, err)
	for _, stmt := range stmts {
		stmt.Accept(visitor{})
		stmt.Accept(visitor1{})
	}
}

func TestSensitiveStatement(t *testing.T) {
	positive := []ast.StmtNode{
		&ast.SetPwdStmt{},
		&ast.CreateUserStmt{},
		&ast.AlterUserStmt{},
		&ast.GrantStmt{},
	}
	for i, stmt := range positive {
		_, ok := stmt.(ast.SensitiveStmtNode)
		require.Truef(t, ok, "%d, %#v fail", i, stmt)
	}

	negative := []ast.StmtNode{
		&ast.DropUserStmt{},
		&ast.RevokeStmt{},
		&ast.AlterTableStmt{},
		&ast.CreateDatabaseStmt{},
		&ast.CreateIndexStmt{},
		&ast.CreateTableStmt{},
		&ast.DropDatabaseStmt{},
		&ast.DropIndexStmt{},
		&ast.DropTableStmt{},
		&ast.RenameTableStmt{},
		&ast.TruncateTableStmt{},
	}
	for _, stmt := range negative {
		_, ok := stmt.(ast.SensitiveStmtNode)
		require.False(t, ok)
	}
}

func TestTableOptimizerHintRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"USE_INDEX(t1 c1)", "USE_INDEX(`t1` `c1`)"},
		{"USE_INDEX(test.t1 c1)", "USE_INDEX(`test`.`t1` `c1`)"},
		{"USE_INDEX(@sel_1 t1 c1)", "USE_INDEX(@`sel_1` `t1` `c1`)"},
		{"USE_INDEX(t1@sel_1 c1)", "USE_INDEX(`t1`@`sel_1` `c1`)"},
		{"USE_INDEX(test.t1@sel_1 c1)", "USE_INDEX(`test`.`t1`@`sel_1` `c1`)"},
		{"USE_INDEX(test.t1@sel_1 partition(p0) c1)", "USE_INDEX(`test`.`t1`@`sel_1` PARTITION(`p0`) `c1`)"},
		{"FORCE_INDEX(t1 c1)", "FORCE_INDEX(`t1` `c1`)"},
		{"FORCE_INDEX(test.t1 c1)", "FORCE_INDEX(`test`.`t1` `c1`)"},
		{"FORCE_INDEX(@sel_1 t1 c1)", "FORCE_INDEX(@`sel_1` `t1` `c1`)"},
		{"FORCE_INDEX(t1@sel_1 c1)", "FORCE_INDEX(`t1`@`sel_1` `c1`)"},
		{"FORCE_INDEX(test.t1@sel_1 c1)", "FORCE_INDEX(`test`.`t1`@`sel_1` `c1`)"},
		{"FORCE_INDEX(test.t1@sel_1 partition(p0) c1)", "FORCE_INDEX(`test`.`t1`@`sel_1` PARTITION(`p0`) `c1`)"},
		{"IGNORE_INDEX(t1 c1)", "IGNORE_INDEX(`t1` `c1`)"},
		{"IGNORE_INDEX(@sel_1 t1 c1)", "IGNORE_INDEX(@`sel_1` `t1` `c1`)"},
		{"IGNORE_INDEX(t1@sel_1 c1)", "IGNORE_INDEX(`t1`@`sel_1` `c1`)"},
		{"IGNORE_INDEX(t1@sel_1 partition(p0, p1) c1)", "IGNORE_INDEX(`t1`@`sel_1` PARTITION(`p0`, `p1`) `c1`)"},
		{"ORDER_INDEX(t1 c1)", "ORDER_INDEX(`t1` `c1`)"},
		{"ORDER_INDEX(test.t1 c1)", "ORDER_INDEX(`test`.`t1` `c1`)"},
		{"ORDER_INDEX(@sel_1 t1 c1)", "ORDER_INDEX(@`sel_1` `t1` `c1`)"},
		{"ORDER_INDEX(t1@sel_1 c1)", "ORDER_INDEX(`t1`@`sel_1` `c1`)"},
		{"ORDER_INDEX(test.t1@sel_1 c1)", "ORDER_INDEX(`test`.`t1`@`sel_1` `c1`)"},
		{"ORDER_INDEX(test.t1@sel_1 partition(p0) c1)", "ORDER_INDEX(`test`.`t1`@`sel_1` PARTITION(`p0`) `c1`)"},
		{"NO_ORDER_INDEX(t1 c1)", "NO_ORDER_INDEX(`t1` `c1`)"},
		{"NO_ORDER_INDEX(test.t1 c1)", "NO_ORDER_INDEX(`test`.`t1` `c1`)"},
		{"NO_ORDER_INDEX(@sel_1 t1 c1)", "NO_ORDER_INDEX(@`sel_1` `t1` `c1`)"},
		{"NO_ORDER_INDEX(t1@sel_1 c1)", "NO_ORDER_INDEX(`t1`@`sel_1` `c1`)"},
		{"NO_ORDER_INDEX(test.t1@sel_1 c1)", "NO_ORDER_INDEX(`test`.`t1`@`sel_1` `c1`)"},
		{"NO_ORDER_INDEX(test.t1@sel_1 partition(p0) c1)", "NO_ORDER_INDEX(`test`.`t1`@`sel_1` PARTITION(`p0`) `c1`)"},
		{"TIDB_SMJ(`t1`)", "TIDB_SMJ(`t1`)"},
		{"TIDB_SMJ(t1)", "TIDB_SMJ(`t1`)"},
		{"TIDB_SMJ(t1,t2)", "TIDB_SMJ(`t1`, `t2`)"},
		{"TIDB_SMJ(@sel1 t1,t2)", "TIDB_SMJ(@`sel1` `t1`, `t2`)"},
		{"TIDB_SMJ(t1@sel1,t2@sel2)", "TIDB_SMJ(`t1`@`sel1`, `t2`@`sel2`)"},
		{"TIDB_INLJ(t1,t2)", "TIDB_INLJ(`t1`, `t2`)"},
		{"TIDB_INLJ(@sel1 t1,t2)", "TIDB_INLJ(@`sel1` `t1`, `t2`)"},
		{"TIDB_INLJ(t1@sel1,t2@sel2)", "TIDB_INLJ(`t1`@`sel1`, `t2`@`sel2`)"},
		{"TIDB_HJ(t1,t2)", "TIDB_HJ(`t1`, `t2`)"},
		{"TIDB_HJ(@sel1 t1,t2)", "TIDB_HJ(@`sel1` `t1`, `t2`)"},
		{"TIDB_HJ(t1@sel1,t2@sel2)", "TIDB_HJ(`t1`@`sel1`, `t2`@`sel2`)"},
		{"MERGE_JOIN(t1,t2)", "MERGE_JOIN(`t1`, `t2`)"},
		{"BROADCAST_JOIN(t1,t2)", "BROADCAST_JOIN(`t1`, `t2`)"},
		{"INL_HASH_JOIN(t1,t2)", "INL_HASH_JOIN(`t1`, `t2`)"},
		{"INL_MERGE_JOIN(t1,t2)", "INL_MERGE_JOIN(`t1`, `t2`)"},
		{"INL_JOIN(t1,t2)", "INL_JOIN(`t1`, `t2`)"},
		{"HASH_JOIN(t1,t2)", "HASH_JOIN(`t1`, `t2`)"},
		{"HASH_JOIN_BUILD(t1)", "HASH_JOIN_BUILD(`t1`)"},
		{"HASH_JOIN_PROBE(t1)", "HASH_JOIN_PROBE(`t1`)"},
		{"LEADING(t1)", "LEADING(`t1`)"},
		{"LEADING(t1, c1)", "LEADING(`t1`, `c1`)"},
		{"LEADING(t1, c1, t2)", "LEADING(`t1`, `c1`, `t2`)"},
		{"LEADING(@sel1 t1, c1)", "LEADING(@`sel1` `t1`, `c1`)"},
		{"LEADING(@sel1 t1)", "LEADING(@`sel1` `t1`)"},
		{"LEADING(@sel1 t1, c1, t2)", "LEADING(@`sel1` `t1`, `c1`, `t2`)"},
		{"LEADING(t1@sel1)", "LEADING(`t1`@`sel1`)"},
		{"LEADING(t1@sel1, c1)", "LEADING(`t1`@`sel1`, `c1`)"},
		{"LEADING(t1@sel1, c1, t2)", "LEADING(`t1`@`sel1`, `c1`, `t2`)"},
		{"MAX_EXECUTION_TIME(3000)", "MAX_EXECUTION_TIME(3000)"},
		{"MAX_EXECUTION_TIME(@sel1 3000)", "MAX_EXECUTION_TIME(@`sel1` 3000)"},
		{"USE_INDEX_MERGE(t1 c1)", "USE_INDEX_MERGE(`t1` `c1`)"},
		{"USE_INDEX_MERGE(@sel1 t1 c1)", "USE_INDEX_MERGE(@`sel1` `t1` `c1`)"},
		{"USE_INDEX_MERGE(t1@sel1 c1)", "USE_INDEX_MERGE(`t1`@`sel1` `c1`)"},
		{"USE_TOJA(TRUE)", "USE_TOJA(TRUE)"},
		{"USE_TOJA(FALSE)", "USE_TOJA(FALSE)"},
		{"USE_TOJA(@sel1 TRUE)", "USE_TOJA(@`sel1` TRUE)"},
		{"USE_CASCADES(TRUE)", "USE_CASCADES(TRUE)"},
		{"USE_CASCADES(FALSE)", "USE_CASCADES(FALSE)"},
		{"USE_CASCADES(@sel1 TRUE)", "USE_CASCADES(@`sel1` TRUE)"},
		{"QUERY_TYPE(OLAP)", "QUERY_TYPE(OLAP)"},
		{"QUERY_TYPE(OLTP)", "QUERY_TYPE(OLTP)"},
		{"QUERY_TYPE(@sel1 OLTP)", "QUERY_TYPE(@`sel1` OLTP)"},
		{"NTH_PLAN(10)", "NTH_PLAN(10)"},
		{"NTH_PLAN(@sel1 30)", "NTH_PLAN(@`sel1` 30)"},
		{"MEMORY_QUOTA(1 GB)", "MEMORY_QUOTA(1024 MB)"},
		{"MEMORY_QUOTA(@sel1 1 GB)", "MEMORY_QUOTA(@`sel1` 1024 MB)"},
		{"HASH_AGG()", "HASH_AGG()"},
		{"HASH_AGG(@sel1)", "HASH_AGG(@`sel1`)"},
		{"STREAM_AGG()", "STREAM_AGG()"},
		{"STREAM_AGG(@sel1)", "STREAM_AGG(@`sel1`)"},
		{"AGG_TO_COP()", "AGG_TO_COP()"},
		{"AGG_TO_COP(@sel_1)", "AGG_TO_COP(@`sel_1`)"},
		{"LIMIT_TO_COP()", "LIMIT_TO_COP()"},
		{"MERGE()", "MERGE()"},
		{"STRAIGHT_JOIN()", "STRAIGHT_JOIN()"},
		{"NO_INDEX_MERGE()", "NO_INDEX_MERGE()"},
		{"NO_INDEX_MERGE(@sel1)", "NO_INDEX_MERGE(@`sel1`)"},
		{"READ_CONSISTENT_REPLICA()", "READ_CONSISTENT_REPLICA()"},
		{"READ_CONSISTENT_REPLICA(@sel1)", "READ_CONSISTENT_REPLICA(@`sel1`)"},
		{"QB_NAME(sel1)", "QB_NAME(`sel1`)"},
		{"READ_FROM_STORAGE(@sel TIFLASH[t1, t2])", "READ_FROM_STORAGE(@`sel` TIFLASH[`t1`, `t2`])"},
		{"READ_FROM_STORAGE(@sel TIFLASH[t1 partition(p0)])", "READ_FROM_STORAGE(@`sel` TIFLASH[`t1` PARTITION(`p0`)])"},
		{"TIME_RANGE('2020-02-02 10:10:10','2020-02-02 11:10:10')", "TIME_RANGE('2020-02-02 10:10:10', '2020-02-02 11:10:10')"},
		{"RESOURCE_GROUP(rg1)", "RESOURCE_GROUP(`rg1`)"},
		{"RESOURCE_GROUP(`default`)", "RESOURCE_GROUP(`default`)"},
	}
	extractNodeFunc := func(node ast.Node) ast.Node {
		return node.(*ast.SelectStmt).TableHints[0]
	}
	runNodeRestoreTest(t, testCases, "select /*+ %s */ * from t1 join t2", extractNodeFunc)
}

func TestBRIESecureText(t *testing.T) {
	testCases := []struct {
		input   string
		secured string
	}{
		{
			input:   "restore database * from 'local:///tmp/br01' snapshot = 23333",
			secured: `^\QRESTORE DATABASE * FROM 'local:///tmp/br01' SNAPSHOT = 23333\E$`,
		},
		{
			input:   "backup database * to 's3://bucket/prefix?region=us-west-2'",
			secured: `^\QBACKUP DATABASE * TO 's3://bucket/prefix?region=us-west-2'\E$`,
		},
		{
			// we need to use regexp to match to avoid the random ordering since a map was used.
			// unfortunately Go's regexp doesn't support lookahead assertion, so the test case below
			// has false positives.
			input:   "backup database * to 's3://bucket/prefix?access-key=abcdefghi&secret-access-key=123&force-path-style=true'",
			secured: `^\QBACKUP DATABASE * TO 's3://bucket/prefix?\E((access-key=xxxxxx|force-path-style=true|secret-access-key=xxxxxx)(&|'$)){3}`,
		},
		{
			input:   "backup database * to 'gcs://bucket/prefix?access-key=irrelevant&credentials-file=/home/user/secrets.txt'",
			secured: `^\QBACKUP DATABASE * TO 'gcs://bucket/prefix?\E((access-key=irrelevant|credentials-file=/home/user/secrets\.txt)(&|'$)){2}`,
		},
	}

	p := parser.New()
	for _, tc := range testCases {
		comment := fmt.Sprintf("input = %s", tc.input)
		node, err := p.ParseOneStmt(tc.input, "", "")
		require.NoError(t, err, comment)
		n, ok := node.(ast.SensitiveStmtNode)
		require.True(t, ok, comment)
		require.Regexp(t, tc.secured, n.SecureText(), comment)
	}
}

func TestCompactTableStmtRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"alter table abc compact tiflash replica", "ALTER TABLE `abc` COMPACT TIFLASH REPLICA"},
		{"alter table abc compact", "ALTER TABLE `abc` COMPACT"},
		{"alter table test.abc compact", "ALTER TABLE `test`.`abc` COMPACT"},
	}
	extractNodeFunc := func(node ast.Node) ast.Node {
		return node.(*ast.CompactTableStmt)
	}
	runNodeRestoreTest(t, testCases, "%s", extractNodeFunc)
}

func TestPlanReplayerStmtRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"plan replayer dump with stats as of timestamp '2023-06-28 12:34:00' explain select * from t where a > 10",
			"PLAN REPLAYER DUMP WITH STATS AS OF TIMESTAMP _UTF8MB4'2023-06-28 12:34:00' EXPLAIN SELECT * FROM `t` WHERE `a`>10"},
		{"plan replayer dump explain analyze select * from t where a > 10",
			"PLAN REPLAYER DUMP EXPLAIN ANALYZE SELECT * FROM `t` WHERE `a`>10"},
		{"plan replayer dump with stats as of timestamp 12345 explain analyze select * from t where a > 10",
			"PLAN REPLAYER DUMP WITH STATS AS OF TIMESTAMP 12345 EXPLAIN ANALYZE SELECT * FROM `t` WHERE `a`>10"},
		{"plan replayer dump explain analyze 'test'",
			"PLAN REPLAYER DUMP EXPLAIN ANALYZE 'test'"},
		{"plan replayer dump with stats as of timestamp '12345' explain analyze 'test2'",
			"PLAN REPLAYER DUMP WITH STATS AS OF TIMESTAMP _UTF8MB4'12345' EXPLAIN ANALYZE 'test2'"},
	}
	extractNodeFunc := func(node ast.Node) ast.Node {
		return node.(*ast.PlanReplayerStmt)
	}
	runNodeRestoreTest(t, testCases, "%s", extractNodeFunc)
}

func TestRedactURL(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		args args
		want string
	}{
		{args{""}, ""},
		{args{":"}, ":"},
		{args{"~/file"}, "~/file"},
		{args{"gs://bucket/file"}, "gs://bucket/file"},
		// gs don't have access-key/secret-access-key, so it will NOT be redacted
		{args{"gs://bucket/file?access-key=123"}, "gs://bucket/file?access-key=123"},
		{args{"gs://bucket/file?secret-access-key=123"}, "gs://bucket/file?secret-access-key=123"},
		{args{"s3://bucket/file"}, "s3://bucket/file"},
		{args{"s3://bucket/file?other-key=123"}, "s3://bucket/file?other-key=123"},
		{args{"s3://bucket/file?access-key=123"}, "s3://bucket/file?access-key=xxxxxx"},
		{args{"s3://bucket/file?secret-access-key=123"}, "s3://bucket/file?secret-access-key=xxxxxx"},
		// underline
		{args{"s3://bucket/file?access_key=123"}, "s3://bucket/file?access_key=xxxxxx"},
		{args{"s3://bucket/file?secret_access_key=123"}, "s3://bucket/file?secret_access_key=xxxxxx"},
	}
	for _, tt := range tests {
		t.Run(tt.args.str, func(t *testing.T) {
			got := ast.RedactURL(tt.args.str)
			if got != tt.want {
				t.Errorf("RedactURL() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddQueryWatchStmtRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{
			"QUERY WATCH ADD ACTION KILL SQL TEXT EXACT TO 'select * from test.t2'",
			"QUERY WATCH ADD ACTION = KILL SQL TEXT EXACT TO _UTF8MB4'select * from test.t2'",
		},
		{
			"QUERY WATCH ADD RESOURCE GROUP rg1 SQL TEXT SIMILAR TO 'select * from test.t2'",
			"QUERY WATCH ADD RESOURCE GROUP `rg1` SQL TEXT SIMILAR TO _UTF8MB4'select * from test.t2'",
		},
		{
			"QUERY WATCH ADD RESOURCE GROUP rg1 ACTION COOLDOWN PLAN DIGEST 'd08bc323a934c39dc41948b0a073725be3398479b6fa4f6dd1db2a9b115f7f57'",
			"QUERY WATCH ADD RESOURCE GROUP `rg1` ACTION = COOLDOWN PLAN DIGEST _UTF8MB4'd08bc323a934c39dc41948b0a073725be3398479b6fa4f6dd1db2a9b115f7f57'",
		},
		{
			"QUERY WATCH ADD ACTION SWITCH_GROUP(rg1) SQL TEXT EXACT TO 'select * from test.t1'",
			"QUERY WATCH ADD ACTION = SWITCH_GROUP(`rg1`) SQL TEXT EXACT TO _UTF8MB4'select * from test.t1'",
		},
	}
	extractNodeFunc := func(node ast.Node) ast.Node {
		return node.(*ast.AddQueryWatchStmt)
	}
	runNodeRestoreTest(t, testCases, "%s", extractNodeFunc)
}

func TestRedactTrafficStmt(t *testing.T) {
	testCases := []struct {
		input   string
		secured string
	}{
		{
			input:   "traffic capture to 's3://bucket/prefix?access-key=abcdefghi&secret-access-key=123&force-path-style=true' duration='1m'",
			secured: "TRAFFIC CAPTURE TO 's3://bucket/prefix?access-key=xxxxxx&force-path-style=true&secret-access-key=xxxxxx' DURATION = '1m'",
		},
		{
			input:   "traffic replay from 's3://bucket/prefix?access-key=abcdefghi&secret-access-key=123&force-path-style=true' user='root' password='123456'",
			secured: "TRAFFIC REPLAY FROM 's3://bucket/prefix?access-key=xxxxxx&force-path-style=true&secret-access-key=xxxxxx' USER = 'root' PASSWORD = 'xxxxxx'",
		},
	}

	p := parser.New()
	for _, tc := range testCases {
		node, err := p.ParseOneStmt(tc.input, "", "")
		require.NoError(t, err, tc.input)
		n, ok := node.(ast.SensitiveStmtNode)
		require.True(t, ok, tc.input)
		require.Equal(t, tc.secured, n.SecureText(), tc.input)
	}
}
