// Package schemafixture builds isolated SQL fixtures with the Goravel SQLite grammar.
package schemafixture

import (
	"database/sql"

	"github.com/goravel/framework/contracts/database/driver"
	"github.com/goravel/framework/contracts/database/schema"
	native "github.com/goravel/framework/database/schema"
	"github.com/goravel/postgres"
	"github.com/goravel/sqlite"
)

type Builder struct {
	DB     *sql.DB
	Driver string
}

func (s Builder) Create(name string, define func(schema.Blueprint)) error {
	b := native.NewBlueprint(nil, "", name)
	b.Create()
	define(b)
	var grammar driver.Grammar = sqlite.NewGrammar(nil, "")
	if s.Driver == "postgres" {
		grammar = postgres.NewGrammar("")
	}
	stmts, err := b.ToSql(grammar)
	if err != nil {
		return err
	}
	for _, stmt := range stmts {
		if stmt == "" {
			continue
		}
		if _, err = s.DB.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
