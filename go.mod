module github.com/nucleuskit/nucleus

go 1.26.3

replace (
	github.com/nucleuskit/bridge => ./bridge
	github.com/nucleuskit/cap => ./cap
	github.com/nucleuskit/contract => ./contract
	github.com/nucleuskit/core => ./core
	github.com/nucleuskit/grpc => ./runtime/grpc
	github.com/nucleuskit/http => ./runtime/http
	github.com/nucleuskit/worker => ./runtime/worker
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
)
