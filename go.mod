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
	github.com/nucleuskit/contract v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
)
