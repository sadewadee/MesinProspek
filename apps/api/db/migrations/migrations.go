// Package migrations embeds the goose SQL migrations so `msp migrate` runs
// from the binary alone (docs/06 §6).
package migrations

import "embed"

// FS holds the numbered goose migrations of this directory (STD-DB-01).
//
//go:embed *.sql
var FS embed.FS
