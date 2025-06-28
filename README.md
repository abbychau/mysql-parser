# Parser

It is a clone of https://github.com/pingcap/tidb/tree/master/pkg/parser .

Reason of cloning:

1. Old https://github.com/pingcap/parser/tree/master is not being maintained.
2. TiDB parser requires "github.com/pingcap/tidb/pkg/types/parser_driver@069631e" as a dependency to work fully.
3. Parser Driver and Parser cannot be compiled without TiDB, and thus cannot target wasm, due to TiDB's dependency on system calls.
