// Package report implements the nucleus report CLI subcommand.
//
// The package keeps CLI rendering and report orchestration local to cmd/nucleus
// while using the contract module for service inspection, diagnostics, and the
// report schema reference. Report generation never performs network calls.
package report
