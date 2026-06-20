// Package migrate implements the Nucleus version migration planning command.
//
// The package is intentionally CLI-internal: it composes contract inspection,
// validation diagnostics, and local migration rules into auditable output, but
// it does not expose a public migration engine or mutate service code.
package migrate
