package parser_test

import (
	"strings"
	"testing"

	"github.com/abbychau/mysql-parser"
	"github.com/abbychau/mysql-parser/ast"
	"github.com/abbychau/mysql-parser/format"
	"github.com/abbychau/mysql-parser/mysql"
	"github.com/stretchr/testify/require"
)

func TestSelectIntoVariables(t *testing.T) {
	table := []testCase{
		{"SELECT a INTO @x FROM t WHERE b = 1", true, "SELECT `a` FROM `t` WHERE `b`=1 INTO @`x`"},
		{"SELECT a, b INTO @x, @y FROM t", true, "SELECT `a`,`b` FROM `t` INTO @`x`,@`y`"},
		{"SELECT LAST_INSERT_ID() INTO @id", true, "SELECT LAST_INSERT_ID() INTO @`id`"},
		{"SELECT a FROM t LIMIT 1 INTO @x", true, "SELECT `a` FROM `t` LIMIT 1 INTO @`x`"},
		{"SELECT a INTO @x FROM DUAL", true, "SELECT `a` INTO @`x`"},
		{"SELECT COUNT(*) INTO v FROM t", true, "SELECT COUNT(1) FROM `t` INTO `v`"},
		{"SELECT a INTO @x", true, "SELECT `a` INTO @`x`"},
		{"SELECT a INTO FROM t", false, ""},
	}
	RunTest(t, table, false)

	p := parser.New()
	stmt, err := p.ParseOneStmt("SELECT a + 1 INTO @x FROM t", "", "")
	require.NoError(t, err)
	sel := stmt.(*ast.SelectStmt)
	require.Equal(t, ast.SelectIntoVars, sel.SelectIntoOpt.Tp)
	require.Len(t, sel.SelectIntoOpt.Variables, 1)
	require.Equal(t, "x", sel.SelectIntoOpt.Variables[0].(*ast.VariableExpr).Name)
	require.Equal(t, "a + 1", sel.Fields.Fields[0].Text())
}

func TestMariaDBDDL(t *testing.T) {
	table := []testCase{
		{"CREATE TABLE t (a INT, b INT AS (a + 1) PERSISTENT)", true, "CREATE TABLE `t` (`a` INT,`b` INT GENERATED ALWAYS AS(`a`+1) STORED)"},
		{"ALTER TABLE t DROP FOREIGN KEY IF EXISTS fk_a", true, "ALTER TABLE `t` DROP FOREIGN KEY IF EXISTS `fk_a`"},
		{"ALTER TABLE t ADD UNIQUE IF NOT EXISTS u (a), ADD UNIQUE KEY IF NOT EXISTS u2 (b)", true, "ALTER TABLE `t` ADD UNIQUE IF NOT EXISTS `u`(`a`), ADD UNIQUE IF NOT EXISTS `u2`(`b`)"},
		{"ALTER TABLE IF EXISTS t ADD COLUMN c INT", true, "ALTER TABLE IF EXISTS `t` ADD COLUMN `c` INT"},
		{"ALTER ONLINE TABLE IF EXISTS t ADD COLUMN c INT, ALGORITHM=INPLACE, LOCK=NONE", true, "ALTER ONLINE TABLE IF EXISTS `t` ADD COLUMN `c` INT, ALGORITHM = INPLACE, LOCK = NONE"},
		{"ALTER ONLINE IGNORE TABLE t ADD COLUMN c INT", true, "ALTER ONLINE TABLE `t` ADD COLUMN `c` INT"},
		{"RENAME TABLE IF EXISTS a TO b", true, "RENAME TABLE IF EXISTS `a` TO `b`"},
		{"RENAME TABLE a WAIT 5 TO b, c NOWAIT TO d", true, "RENAME TABLE `a` WAIT 5 TO `b`, `c` NOWAIT TO `d`"},
		{"CREATE OR REPLACE UNIQUE INDEX i ON t (a)", true, "CREATE OR REPLACE UNIQUE INDEX `i` ON `t` (`a`)"},
		{"CREATE ONLINE INDEX i ON t (a)", true, "CREATE ONLINE INDEX `i` ON `t` (`a`)"},
		{"CREATE OFFLINE INDEX IF NOT EXISTS i ON t (a)", true, "CREATE OFFLINE INDEX IF NOT EXISTS `i` ON `t` (`a`)"},
		{"ALTER VIEW v AS SELECT 1", true, "ALTER ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `v` AS SELECT 1"},
		{"ALTER ALGORITHM = MERGE SQL SECURITY INVOKER VIEW v (a) AS SELECT a FROM t WITH LOCAL CHECK OPTION", true, "ALTER ALGORITHM = MERGE DEFINER = CURRENT_USER SQL SECURITY INVOKER VIEW `v` (`a`) AS SELECT `a` FROM `t` WITH LOCAL CHECK OPTION"},
		{"CREATE TABLE t (a CHAR(32) CHARACTER SET utf32 COLLATE utf32_czech_ci)", true, "CREATE TABLE `t` (`a` CHAR(32) CHARACTER SET UTF32 COLLATE utf32_czech_ci)"},
		{"CREATE TABLE t (a VARCHAR(10) COLLATE utf8mb3_czech_ci)", true, "CREATE TABLE `t` (`a` VARCHAR(10) COLLATE utf8_czech_ci)"},
		{"CREATE TABLE t (a INT) CHARSET=nosuchcharset", false, ""},
		{"GRANT SELECT, DELETE HISTORY, SET USER ON *.* TO 'u'@'%'", true, "GRANT SELECT, DELETE HISTORY, SET USER ON *.* TO `u`@`%`"},
	}
	RunTest(t, table, false)
}

func TestSpatialTypes(t *testing.T) {
	table := []testCase{
		{"CREATE TABLE t (a POINT NOT NULL, b GEOMETRY, c LINESTRING, d POLYGON, e MULTIPOINT, f MULTILINESTRING, g MULTIPOLYGON, h GEOMETRYCOLLECTION, i GEOMCOLLECTION, SPATIAL KEY idx (a))", true,
			"CREATE TABLE `t` (`a` POINT NOT NULL,`b` GEOMETRY,`c` LINESTRING,`d` POLYGON,`e` MULTIPOINT,`f` MULTILINESTRING,`g` MULTIPOLYGON,`h` GEOMETRYCOLLECTION,`i` GEOMCOLLECTION,SPATIAL `idx`(`a`))"},
		{"SELECT POLYGON(LINESTRING(POINT(0, 0), POINT(1, 1), POINT(0, 0)))", true, "SELECT POLYGON(LINESTRING(POINT(0, 0), POINT(1, 1), POINT(0, 0)))"},
		{"SELECT point, polygon, geometry FROM linestring", true, "SELECT `point`,`polygon`,`geometry` FROM `linestring`"},
	}
	RunTest(t, table, false)

	p := parser.New()
	stmt, err := p.ParseOneStmt("CREATE TABLE t (a POINT, b GEOMETRY)", "", "")
	require.NoError(t, err)
	cols := stmt.(*ast.CreateTableStmt).Cols
	require.Equal(t, mysql.TypeGeometry, cols[0].Tp.GetType())
	require.Equal(t, "point", cols[0].Tp.GetGeometryType())
	require.Equal(t, "geometry", cols[1].Tp.GetGeometryType())
}

func TestDefaultExpression(t *testing.T) {
	table := []testCase{
		{"CREATE TABLE t (d DATE NOT NULL DEFAULT (CURDATE() + INTERVAL 9 MONTH))", true, "CREATE TABLE `t` (`d` DATE NOT NULL DEFAULT (DATE_ADD(CURDATE(), INTERVAL 9 MONTH)))"},
		{"CREATE TABLE t (x INT DEFAULT (1 + 2))", true, "CREATE TABLE `t` (`x` INT DEFAULT (1+2))"},
		{"ALTER TABLE t ADD ttl DATE NOT NULL DEFAULT (CURDATE() + INTERVAL 9 MONTH)", true, "ALTER TABLE `t` ADD COLUMN `ttl` DATE NOT NULL DEFAULT (DATE_ADD(CURDATE(), INTERVAL 9 MONTH))"},
		// The parenthesized forms that were accepted before keep their meaning.
		{"CREATE TABLE t (a DATETIME DEFAULT (NOW()), b DATE DEFAULT (CURDATE()), c INT DEFAULT (1), d INT DEFAULT (-1), e VARCHAR(36) DEFAULT (UUID()), f DATETIME(3) DEFAULT (CURRENT_TIMESTAMP(3)), g DATE DEFAULT ((CURDATE())))", true,
			"CREATE TABLE `t` (`a` DATETIME DEFAULT CURRENT_TIMESTAMP(),`b` DATE DEFAULT (CURRENT_DATE()),`c` INT DEFAULT 1,`d` INT DEFAULT -1,`e` VARCHAR(36) DEFAULT (UUID()),`f` DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),`g` DATE DEFAULT (CURRENT_DATE()))"},
	}
	RunTest(t, table, false)
}

func TestTriggers(t *testing.T) {
	table := []testCase{
		{"CREATE TRIGGER tr BEFORE INSERT ON t FOR EACH ROW SET NEW.a = 1, NEW.`b` = OLD.b + 1", true, "CREATE TRIGGER `tr` BEFORE INSERT ON `t` FOR EACH ROW SET `NEW`.`a`=1, `NEW`.`b`=`OLD`.`b`+1"},
		{"CREATE TRIGGER `tr` AFTER UPDATE ON `t` FOR EACH ROW INSERT INTO log SET id = NEW.id", true, "CREATE TRIGGER `tr` AFTER UPDATE ON `t` FOR EACH ROW INSERT INTO `log` SET `id`=`NEW`.`id`"},
		{"CREATE OR REPLACE DEFINER = 'root'@'localhost' TRIGGER IF NOT EXISTS db.tr BEFORE DELETE ON t FOR EACH ROW FOLLOWS other DELETE FROM log WHERE id = OLD.id", true,
			"CREATE OR REPLACE DEFINER = `root`@`localhost` TRIGGER IF NOT EXISTS `db`.`tr` BEFORE DELETE ON `t` FOR EACH ROW FOLLOWS `other` DELETE FROM `log` WHERE `id`=`OLD`.`id`"},
		{"CREATE ALGORITHM = MERGE TRIGGER tr BEFORE INSERT ON t FOR EACH ROW SET NEW.a = 1", false, ""},
		{"CREATE TRIGGER tr BEFORE SELECT ON t FOR EACH ROW SET NEW.a = 1", false, ""},
		{"DROP TRIGGER tr", true, "DROP TRIGGER `tr`"},
		{"DROP TRIGGER IF EXISTS db.tr", true, "DROP TRIGGER IF EXISTS `db`.`tr`"},
		{"SET validate_password.length = 8", true, "SET @@SESSION.`validate_password.length`=8"},
	}
	RunTest(t, table, false)

	// Statements inside a BEGIN ... END body are not visited by Accept (as in
	// stored procedures), so only the restored text is compared.
	p := parser.New()
	src := "CREATE TRIGGER tr AFTER INSERT ON t FOR EACH ROW PRECEDES other BEGIN IF NEW.a > 0 THEN UPDATE s SET b = NEW.a WHERE c = NEW.c; END IF; END"
	stmt, err := p.ParseOneStmt(src, "", "")
	require.NoError(t, err)
	tr := stmt.(*ast.CreateTriggerStmt)
	require.Equal(t, ast.TriggerAfter, tr.Timing)
	require.Equal(t, ast.TriggerInsert, tr.Event)
	require.True(t, tr.Order.Precedes)
	require.Equal(t, "other", tr.Order.OtherTrigger.O)
	require.IsType(t, &ast.ProcedureBlock{}, tr.Body)
	var sb strings.Builder
	require.NoError(t, stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)))
	require.Equal(t, "CREATE TRIGGER `tr` AFTER INSERT ON `t` FOR EACH ROW PRECEDES `other` BEGIN IF `NEW`.`a`>0 THEN UPDATE `s` SET `b`=`NEW`.`a` WHERE `c`=`NEW`.`c`;END IF; END", sb.String())
	_, err = p.ParseOneStmt(sb.String(), "", "")
	require.NoError(t, err)
}

func TestLoadableFunction(t *testing.T) {
	table := []testCase{
		{"CREATE OR REPLACE FUNCTION uuid_v4 RETURNS string SONAME 'libudf_uuid.so'", true, "CREATE OR REPLACE FUNCTION `uuid_v4` RETURNS STRING SONAME 'libudf_uuid.so'"},
		{"CREATE FUNCTION IF NOT EXISTS f RETURNS INTEGER SONAME 'lib.so'", true, "CREATE FUNCTION IF NOT EXISTS `f` RETURNS INTEGER SONAME 'lib.so'"},
		{"CREATE FUNCTION f RETURNS REAL SONAME 'lib.so'", true, "CREATE FUNCTION `f` RETURNS REAL SONAME 'lib.so'"},
		{"CREATE FUNCTION f RETURNS DECIMAL SONAME 'lib.so'", true, "CREATE FUNCTION `f` RETURNS DECIMAL SONAME 'lib.so'"},
		{"CREATE FUNCTION f RETURNS BLOB SONAME 'lib.so'", false, ""},
	}
	RunTest(t, table, false)
}

func TestJSONTable(t *testing.T) {
	table := []testCase{
		{"SELECT * FROM JSON_TABLE('[1,2]', '$[*]' COLUMNS (v INT PATH '$')) AS j", true, "SELECT * FROM JSON_TABLE('[1,2]', '$[*]' COLUMNS (`v` INT PATH '$')) AS `j`"},
		{"SELECT j.v FROM t, JSON_TABLE(t.doc, '$[*]' COLUMNS (rn FOR ORDINALITY, v INT UNSIGNED PATH '$' NULL ON EMPTY ERROR ON ERROR, e INT EXISTS PATH '$.a', NESTED PATH '$.b[*]' COLUMNS (x VARCHAR(10) PATH '$' DEFAULT '0' ON EMPTY))) j", true,
			"SELECT `j`.`v` FROM (`t`) JOIN JSON_TABLE(`t`.`doc`, '$[*]' COLUMNS (`rn` FOR ORDINALITY, `v` INT UNSIGNED PATH '$' NULL ON EMPTY ERROR ON ERROR, `e` INT EXISTS PATH '$.a', NESTED PATH '$.b[*]' COLUMNS (`x` VARCHAR(10) PATH '$' DEFAULT '0' ON EMPTY))) AS `j`"},
		{"SELECT * FROM JSON_TABLE('[1]', '$[*]' COLUMNS (v INT PATH '$'))", false, ""},
		{"SELECT json_table, path, nested, ordinality FROM empty", true, "SELECT `json_table`,`path`,`nested`,`ordinality` FROM `empty`"},
	}
	RunTest(t, table, false)
}

func TestMariaDBMisc(t *testing.T) {
	table := []testCase{
		{"SELECT JSON_ARRAYAGG(DISTINCT a) FROM t", true, "SELECT JSON_ARRAYAGG(DISTINCT `a`) FROM `t`"},
		{"INSERT INTO t (`from`, until) SELECT a, until FROM s", true, "INSERT INTO `t` (`from`,`until`) SELECT `a`,`until` FROM `s`"},
		{"/*!999999\\- enable the sandbox mode */ SELECT 1", true, "SELECT 1"},
		{"/*!40101 SET @a = 1 */", true, "SET @`a`=1"},
		{"/*!100100 SET @a = 1 */", true, "SET @`a`=1"},
	}
	RunTest(t, table, false)
}
