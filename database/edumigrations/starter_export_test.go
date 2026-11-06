package edumigrations

import (
	"fmt"
	"regexp"

	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/goravel/framework/contracts/database/schema"
)

// StarterMigrationDescriptor is test-only access to the exact values passed to
// Register. It is not a production introspection or migration API.
type StarterSQLStatement struct {
	SQL  string
	Args []any
}

type StarterMigrationDescriptor struct {
	Signature, Implementation, PlanID, Connection, Driver, LegacyTenant, Owner string
	ControlOnly                                                                bool
	Up, Down                                                                   []string
	ApplyUp, ApplyDown                                                         []StarterSQLStatement
}

func DescribeStarterMigration(m schema.Migration) StarterMigrationDescriptor {
	d := StarterMigrationDescriptor{Signature: m.Signature(), Implementation: fmt.Sprintf("%T", m)}
	if r, ok := m.(*registered); ok {
		d.PlanID, d.Connection, d.Driver, d.LegacyTenant, d.Owner, d.ControlOnly = r.plan.ID, r.node.Name, r.node.Driver, r.node.Tenant, r.node.Owner(), r.plan.ControlOnly
		d.Up = append([]string{}, r.plan.Statements(r.node, false)...)
		d.Down = append([]string{}, r.plan.Statements(r.node, true)...)
		record := func(down bool) []StarterSQLStatement {
			statements := []StarterSQLStatement{}
			if err := r.apply(down, func(query string, args ...any) (int64, error) {
				statements = append(statements, StarterSQLStatement{SQL: query, Args: append([]any{}, args...)})
				return 1, nil
			}); err != nil {
				panic(err)
			}
			return statements
		}
		d.ApplyUp, d.ApplyDown = record(false), record(true)
		// The original placement DDL generates this value for each invocation.
		generated := regexp.MustCompile(`(VALUES \(1, '[^']+', ')[^']+('\))`)
		if r.plan.placement {
			for i := range d.ApplyUp {
				d.ApplyUp[i].SQL = generated.ReplaceAllString(d.ApplyUp[i].SQL, "${1}<generated>${2}")
			}
			for i := range d.Up {
				d.Up[i] = generated.ReplaceAllString(d.Up[i], "${1}<generated>${2}")
			}
		}
	}
	return d
}

// CloneStarterMigration lets external regression tests alter actual registered
// values without changing historical migration files or production APIs.
func CloneStarterMigration(m schema.Migration, plan func(Plan) Plan, node func(database.Connection) database.Connection) schema.Migration {
	r := m.(*registered)
	copy := *r
	if plan != nil {
		copy.plan = plan(copy.plan)
	}
	if node != nil {
		copy.node = node(copy.node)
	}
	return &copy
}
