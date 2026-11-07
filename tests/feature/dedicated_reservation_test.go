package feature

import (
	"errors"
	"sync"
	"testing"

	"github.com/elcomware/edupilot/app/provisioning"
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
)

func verifyDedicatedReservation(t *testing.T, db *database.Catalog) {
	t.Helper()
	ctx := t.Context()
	id := requireValue(scope.Parse[scope.Tenant](candidateID))
	target := db.Named()["tenant_"+candidateID]
	var marker string
	if err := target.DB.QueryRow("SELECT pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&marker); err != nil {
		t.Fatal(err)
	}
	operation := "ec8ef538-2f74-4788-a305-e1c410de55ca"
	other := "7ab023a8-9a2e-4cbf-afb5-6ae1e0b5d145"
	exec := func(pool database.Pool, query string, args ...any) {
		t.Helper()
		if _, err := pool.DB.Exec(pool.Bind(query), args...); err != nil {
			t.Fatal(err)
		}
	}
	count := func(pool database.Pool, query string, want int) {
		t.Helper()
		var got int
		if err := pool.DB.QueryRow(pool.Bind(query), candidateID).Scan(&got); err != nil || got != want {
			t.Fatalf("count = %d, want %d: %v", got, want, err)
		}
	}
	reserve := func(op, pool string, want bool) {
		t.Helper()
		got, err := provisioning.ReserveDedicatedTenant(ctx, db, id, pool, op)
		if (err == nil) != want || (want && (got.Tenant != id || got.PoolID != marker || got.OperationID != op)) {
			t.Fatalf("reservation = %+v, %v", got, err)
		}
	}
	// The preceding candidate check left a legitimate first-use placement.
	reserve(operation, marker, false)
	exec(db.Control, "DELETE FROM edu_tenant_placement WHERE tenant_id=?", candidateID)
	reserve("", marker, false)
	reserve(operation, "wrong", false)
	exec(db.Control, "INSERT INTO auth_realms (school_id) VALUES (?)", candidateID)
	reserve(operation, marker, false)
	count(db.Control, "SELECT count(*) FROM edu_tenant_placement WHERE tenant_id=?", 0)
	count(db.Control, "SELECT count(*) FROM edu_tenant_reservations WHERE tenant_id=?", 0)
	exec(db.Control, "DELETE FROM auth_realms WHERE school_id=?", candidateID)
	exec(target, "INSERT INTO ws_tenancy (school_id,id,title,status,version) VALUES (?,'reserved','fixture','open',1)", candidateID)
	reserve(operation, marker, false)
	count(db.Control, "SELECT count(*) FROM edu_tenant_placement WHERE tenant_id=?", 0)
	count(db.Control, "SELECT count(*) FROM edu_tenant_reservations WHERE tenant_id=?", 0)
	count(db.Control, "SELECT count(*) FROM auth_realms WHERE school_id=?", 0)
	exec(target, "DELETE FROM ws_tenancy WHERE school_id=?", candidateID)

	// These resets operate only on this fixture's disposable tenant.
	reset := func() {
		t.Helper()
		exec(db.Control, "DELETE FROM edu_tenant_reservations WHERE tenant_id=?", candidateID)
		exec(db.Control, "DELETE FROM edu_tenant_placement WHERE tenant_id=?", candidateID)
		for _, table := range []string{"auth_memberships", "auth_users", "auth_role_permissions", "auth_permissions", "auth_roles", "auth_realms"} {
			exec(db.Control, "DELETE FROM "+table+" WHERE school_id=?", candidateID)
		}
	}
	for _, competitor := range []struct {
		name string
		call func() error
	}{
		{"first use", func() error {
			_, err := db.ForTenantContext(ctx, candidateID)
			return err
		}},
		{"administrator seed", func() error {
			_, err := services.New(&db.Control).Seed(ctx, candidateID, "race@example.test", "Disposable-fixture-password-8")
			return err
		}},
	} {
		for attempt := 0; attempt < 3; attempt++ {
			start := make(chan struct{})
			reserved := make(chan bool, 1)
			competed := make(chan bool, 1)
			go func() {
				<-start
				_, err := provisioning.ReserveDedicatedTenant(ctx, db, id, marker, operation)
				reserved <- err == nil
			}()
			go func() {
				<-start
				competed <- competitor.call() == nil
			}()
			close(start)
			won, lost := <-reserved, <-competed
			if won == lost {
				t.Fatalf("reservation/%s race did not produce one winner: %v/%v", competitor.name, won, lost)
			}
			if won {
				count(db.Control, "SELECT count(*) FROM edu_tenant_placement WHERE tenant_id=? AND pool_id=''", 1)
				count(db.Control, "SELECT count(*) FROM auth_users WHERE school_id=?", 0)
			} else {
				count(db.Control, "SELECT count(*) FROM edu_tenant_reservations WHERE tenant_id=?", 0)
			}
			reset()
		}
	}

	// Competing operations cannot both acquire this tenant.
	var wg sync.WaitGroup
	results := make(chan string, 2)
	for _, op := range []string{operation, other} {
		wg.Go(func() {
			if _, err := provisioning.ReserveDedicatedTenant(ctx, db, id, marker, op); err == nil {
				results <- op
			}
		})
	}
	wg.Wait()
	close(results)
	var winner string
	for op := range results {
		if winner != "" {
			t.Fatal("two operations acquired the same tenant")
		}
		winner = op
	}
	if winner == "" {
		t.Fatal("no operation acquired the fresh tenant")
	}
	loser := operation
	if winner == operation {
		loser = other
	}
	reserve(winner, marker, true)
	reserve(loser, marker, false)
	for _, resolve := range []func(string) (database.Pool, error){
		func(id string) (database.Pool, error) { return db.ForTenantContext(ctx, id) },
		func(id string) (database.Pool, error) { return db.PinnedPool(ctx, id) },
	} {
		if _, err := resolve(candidateID); !errors.Is(err, database.ErrPlacement) {
			t.Fatal("reserved tenant routed", err)
		}
		if _, err := resolve("west"); err != nil {
			t.Fatal("unrelated tenant unavailable", err)
		}
	}
	if _, err := services.New(&db.Control).Seed(ctx, candidateID, "reserve@example.test", "Disposable-fixture-password-8"); !errors.Is(err, services.ErrConflict) {
		t.Fatal("pending reservation allowed administrator bootstrap", err)
	}
	if err := edumigrations.Run(ctx, db); err != nil {
		t.Fatal("reservation prevented installer replay/readiness", err)
	}
	reserve(winner, marker, true)
	count(db.Control, "SELECT count(*) FROM edu_tenant_placement WHERE tenant_id=? AND pool_id=''", 1)
	count(db.Control, "SELECT count(*) FROM edu_tenant_reservations WHERE tenant_id=?", 1)
	count(db.Control, "SELECT count(*) FROM auth_realms WHERE school_id=?", 1)
	count(db.Control, "SELECT count(*) FROM auth_users WHERE school_id=?", 0)
	for _, pool := range db.All() {
		count(pool, "SELECT count(*) FROM tenant_gates WHERE tenant_id=?", 0)
		count(pool, "SELECT count(*) FROM edu_outbox WHERE tenant_id=?", 0)
	}
	// Revalidation detects an unexpected writer without releasing the fence.
	exec(target, "INSERT INTO ws_tenancy (school_id,id,title,status,version) VALUES (?,'unexpected','fixture','open',1)", candidateID)
	reserve(winner, marker, false)
	exec(target, "DELETE FROM ws_tenancy WHERE school_id=?", candidateID)
	reserve(winner, marker, true)
	exec(db.Control, "UPDATE edu_tenant_placement SET pool_id=? WHERE tenant_id=?", marker, candidateID)
	if err := db.CheckReady(ctx); !errors.Is(err, database.ErrPlacement) {
		t.Fatal("active pin accepted with pending reservation", err)
	}
	if _, err := db.ForTenantContext(ctx, candidateID); !errors.Is(err, database.ErrPlacement) {
		t.Fatal("corrupt pending pin routed", err)
	}
	if _, err := db.PinnedPool(ctx, candidateID); !errors.Is(err, database.ErrPlacement) {
		t.Fatal("corrupt pending pin resolved", err)
	}
	exec(db.Control, "UPDATE edu_tenant_placement SET pool_id='' WHERE tenant_id=?", candidateID)
}
