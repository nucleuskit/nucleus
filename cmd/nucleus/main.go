package main

import (
	"os"

	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/root"
)

func main() {
	if err := root.New().Execute(); err != nil {
		os.Exit(1)
	}
}
