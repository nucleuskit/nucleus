package main

import (
	"fmt"
	"os"

	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/root"
)

func main() {
	if err := root.New().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
