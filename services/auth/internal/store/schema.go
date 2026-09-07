package store

import _ "embed"

// Schema travels inside the binary, so a deploy cannot land code that expects
// tables the database does not have. It lives beside the store because this is
// the package that depends on its shape.
//
//go:embed migrations/001_init.sql
var Schema string
