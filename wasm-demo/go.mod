module wasm-demo

go 1.23

replace github.com/abbychau/mysql-parser => ../

replace github.com/abbychau/mysql-parser/parser_driver => ../parser_driver

require (
	github.com/abbychau/mysql-parser v0.0.0-00010101000000-000000000000
	github.com/abbychau/mysql-parser/parser_driver v0.0.0-00010101000000-000000000000
)

require (
	github.com/pingcap/errors v0.11.5-0.20250523034308-74f78ae071ee // indirect
	github.com/pingcap/failpoint v0.0.0-20240528011301-b51a646c7c86 // indirect
	github.com/pingcap/log v1.1.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)
