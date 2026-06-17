// Package scenario implements the nucleus scenario CLI subcommand.
//
// The package keeps CLI rendering and HTTP execution local to cmd/nucleus while
// reusing the public contract module for OpenAPI routes, error catalogs, and
// inspection flow facts.
package scenario
