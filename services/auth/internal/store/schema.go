package store

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
)

// Schema travels inside the binary, so a deploy cannot land code that expects
// tables the database does not have. It lives beside the store because this is
// the package that depends on its shape.
//
// The glob matters. This used to embed migrations/001_init.sql by name, which
// means the day a second file was added it would have sat in the repository
// looking applied, passing review, and never running — a whole migration
// missing with nothing to notice it. Filename order is the apply order, which
// is what the numeric prefixes are for.
//
//go:embed migrations/*.sql
var migrations embed.FS

var Schema = concatMigrations()

func concatMigrations() string {
	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		// The pattern is a constant over an embedded filesystem; if this ever
		// fails the binary is not one we should be running.
		panic(err)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		sqlText, err := migrations.ReadFile(name)
		if err != nil {
			panic(err)
		}
		b.Write(sqlText)
		// A file that ends in a comment would otherwise swallow the first line
		// of the next one.
		b.WriteString("\n")
	}
	return b.String()
}
