package tenantupgrade

import (
	"context"
	"database/sql"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/elcomware/edupilot-os/platform/database"
)

// PlacementInspection inventories legacy SQL routing only. Complete means all
// required routing tables were read, never that contents or objects are verified.
type PlacementInspection struct {
	Version        int               `json:"version"`
	Classification string            `json:"classification"`
	Complete       bool              `json:"complete"`
	Pools          []InspectedPool   `json:"pools"`
	Schools        []InspectedSchool `json:"schools"`
	Issues         []InspectionIssue `json:"issues"`
}
type InspectedPool struct {
	Name   string           `json:"name"`
	Tables []InspectedTable `json:"tables"`
}
type InspectedTable struct {
	Table string `json:"table"`
	Rows  int64  `json:"rows"`
}
type InspectedSchool struct {
	School       string               `json:"school"`
	Identity     bool                 `json:"identity"`
	Pinned       bool                 `json:"pinned"`
	ExpectedPool string               `json:"expectedPool"`
	Rows         []InspectedScopeRows `json:"rows"`
}
type InspectedScopeRows struct {
	Pool  string `json:"pool"`
	Table string `json:"table"`
	Rows  int64  `json:"rows"`
}
type InspectionIssue struct {
	Code   string `json:"code"`
	Pool   string `json:"pool,omitempty"`
	School string `json:"school,omitempty"`
	Table  string `json:"table,omitempty"`
}
type inspectionTable struct{ name, column string }
type inspectionState struct {
	report       PlacementInspection
	schools      map[string]*InspectedSchool
	pins         map[string]string
	markers      map[string][]string
	owners       map[string][]string
	observed     map[string]map[string]bool
	issueKeys    map[InspectionIssue]bool
	identityRead bool
	pinsRead     bool
	markersRead  map[string]bool
}

// InspectPlacement opens its own read-only connections. Workspace keys must come
// from composition's actual module registrations; platform imports no modules.
// Each pool has a repeatable snapshot, without a cross-database snapshot claim.
func InspectPlacement(ctx context.Context, settings database.Settings, workspaceKeys []string) PlacementInspection {
	return inspectPlacement(ctx, settings, workspaceKeys, readSnapshots(openInspectionDB))
}

func inspectPlacement(ctx context.Context, settings database.Settings, workspaceKeys []string, snapshot sourceSnapshots) PlacementInspection {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	state := inspectionState{report: PlacementInspection{Version: 1, Classification: "inspect-only", Complete: true, Pools: []InspectedPool{}, Schools: []InspectedSchool{}, Issues: []InspectionIssue{}}, schools: map[string]*InspectedSchool{}, pins: map[string]string{}, markers: map[string][]string{}, owners: map[string][]string{}, observed: map[string]map[string]bool{}, issueKeys: map[InspectionIssue]bool{}, markersRead: map[string]bool{}}
	keys := slices.Clone(workspaceKeys)
	slices.Sort(keys)
	validKey := regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)
	if settings.Validate() != nil || len(settings.Nodes()) > 128 || len(keys) != 41 {
		state.problem("invalid_configuration", "", "", "", true)
		return state.report
	}
	for i, key := range keys {
		if !validKey.MatchString(key) || (i > 0 && key == keys[i-1]) {
			state.problem("invalid_workspace_registry", "", "", "", true)
			return state.report
		}
	}
	tables := []inspectionTable{{"students_records", "school_id"}, {"enrollment_placements", "school_id"}, {"edu_outbox", "tenant_id"}, {"edu_audit_inbox", "tenant_id"}}
	for _, key := range keys {
		tables = append(tables, inspectionTable{"ws_" + key, "school_id"})
	}
	slices.SortFunc(tables, func(a, b inspectionTable) int { return strings.Compare(a.name, b.name) })
	nodes := settings.Nodes()
	slices.SortFunc(nodes, func(a, b database.Connection) int { return strings.Compare(a.Name, b.Name) })
	now := time.Now().UnixMilli()
	for _, node := range nodes {
		state.inspectPool(ctx, node, tables, now, snapshot)
	}
	state.reconcile(settings)
	for _, school := range state.schools {
		state.report.Schools = append(state.report.Schools, *school)
	}
	slices.SortFunc(state.report.Schools, func(a, b InspectedSchool) int { return strings.Compare(a.School, b.School) })
	slices.SortFunc(state.report.Issues, func(a, b InspectionIssue) int {
		for _, pair := range [][2]string{{a.Code, b.Code}, {a.Pool, b.Pool}, {a.School, b.School}, {a.Table, b.Table}} {
			if cmp := strings.Compare(pair[0], pair[1]); cmp != 0 {
				return cmp
			}
		}
		return 0
	})
	return state.report
}
func (s *inspectionState) problem(code, pool, school, table string, incomplete bool) {
	issue := InspectionIssue{Code: code, Pool: pool, School: school, Table: table}
	if !s.issueKeys[issue] {
		s.report.Issues = append(s.report.Issues, issue)
		s.issueKeys[issue] = true
	}
	if incomplete {
		s.report.Complete = false
	}
}
func (s *inspectionState) school(id, pool, table string) *InspectedSchool {
	if !database.ValidTenant(id) {
		s.problem("invalid_school_scope", pool, "", table, false)
		return nil
	}
	if existing := s.schools[id]; existing != nil {
		return existing
	}
	if len(s.schools) >= maxSchools {
		s.problem("school_limit_exceeded", pool, "", table, true)
		return nil
	}
	school := &InspectedSchool{School: id, Rows: []InspectedScopeRows{}}
	s.schools[id] = school
	return school
}
func (s *inspectionState) inspectPool(ctx context.Context, node database.Connection, tables []inspectionTable, now int64, snapshot sourceSnapshots) {
	pool := InspectedPool{Name: node.Name, Tables: []InspectedTable{}}
	defer func() { s.report.Pools = append(s.report.Pools, pool) }()
	tx, release, err := snapshot(ctx, node)
	if err != nil {
		s.problem("pool_unavailable", node.Name, "", "", true)
		return
	}
	defer release()
	available, err := inspectionTables(ctx, tx, node.Driver)
	if err != nil {
		s.problem("schema_unavailable", node.Name, "", "", true)
		return
	}
	has := func(table string) bool {
		if !available[table] {
			s.problem("missing_legacy_table", node.Name, "", table, true)
			return false
		}
		return true
	}
	if node.Tenant != "" {
		for _, table := range []string{"identity_schools", "identity_accounts", "identity_memberships", "identity_sessions", "edu_tenant_placement", "migrations", "jobs", "failed_jobs"} {
			if available[table] {
				s.problem("unexpected_control_table", node.Name, "", table, false)
			}
		}
	}
	if has("edu_pool_marker") {
		rows, err := tx.QueryContext(ctx, "SELECT singleton,"+boundedInspectionText("owner", 71, node.Driver)+","+boundedInspectionText("pool_id", 129, node.Driver)+" FROM edu_pool_marker ORDER BY singleton,owner,pool_id LIMIT 2")
		if err != nil {
			s.problem("routing_query_failed", node.Name, "", "edu_pool_marker", true)
			return
		}
		count, valid := 0, 0
		for rows.Next() {
			var singleton int
			var owner, marker sql.NullString
			if err := rows.Scan(&singleton, &owner, &marker); err != nil {
				rows.Close()
				s.problem("invalid_pool_marker", node.Name, "", "edu_pool_marker", true)
				return
			}
			count++
			if singleton != 1 || !owner.Valid || !validLegacyOwner(owner.String) || !marker.Valid || !legacyMarkerValid(marker.String) {
				s.problem("invalid_pool_marker", node.Name, "", "edu_pool_marker", false)
				continue
			}
			valid++
			s.markers[marker.String] = append(s.markers[marker.String], node.Name)
			s.owners[owner.String] = append(s.owners[owner.String], node.Name)
			if owner.String != node.Owner() {
				s.problem("wrong_pool_owner", node.Name, "", "edu_pool_marker", false)
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			s.problem("routing_query_failed", node.Name, "", "edu_pool_marker", true)
			return
		}
		if count != 1 {
			s.problem("invalid_pool_marker", node.Name, "", "edu_pool_marker", count > 1)
		}
		s.markersRead[node.Name] = count == 1 && valid == 1
	}
	if node.Tenant == "" {
		if has("identity_schools") {
			rows, err := tx.QueryContext(ctx, "SELECT "+boundedInspectionText("school_id", 64, node.Driver)+" FROM identity_schools ORDER BY school_id LIMIT 4097")
			if err != nil {
				s.problem("routing_query_failed", node.Name, "", "identity_schools", true)
				return
			}
			count := 0
			for rows.Next() {
				var id sql.NullString
				if err := rows.Scan(&id); err != nil {
					rows.Close()
					s.problem("routing_query_failed", node.Name, "", "identity_schools", true)
					return
				}
				count++
				if school := s.school(id.String, node.Name, "identity_schools"); school != nil {
					school.Identity = true
				}
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				s.problem("routing_query_failed", node.Name, "", "identity_schools", true)
				return
			}
			if count > maxSchools {
				s.problem("school_limit_exceeded", node.Name, "", "identity_schools", true)
			} else {
				s.identityRead = true
			}
		}
		if has("edu_tenant_placement") {
			rows, err := tx.QueryContext(ctx, "SELECT "+boundedInspectionText("tenant_id", 64, node.Driver)+","+boundedInspectionText("pool_id", 129, node.Driver)+" FROM edu_tenant_placement ORDER BY tenant_id,pool_id LIMIT 4097")
			if err != nil {
				s.problem("routing_query_failed", node.Name, "", "edu_tenant_placement", true)
				return
			}
			count := 0
			for rows.Next() {
				var id, pin sql.NullString
				if err := rows.Scan(&id, &pin); err != nil {
					rows.Close()
					s.problem("routing_query_failed", node.Name, "", "edu_tenant_placement", true)
					return
				}
				count++
				if school := s.school(id.String, node.Name, "edu_tenant_placement"); school != nil {
					if school.Pinned {
						s.problem("duplicate_school_pin", node.Name, id.String, "edu_tenant_placement", false)
					}
					school.Pinned = true
					if pin.Valid && legacyMarkerValid(pin.String) {
						s.pins[id.String] = pin.String
					} else {
						s.problem("invalid_school_pin", node.Name, id.String, "edu_tenant_placement", false)
					}
				}
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				s.problem("routing_query_failed", node.Name, "", "edu_tenant_placement", true)
				return
			}
			if count > maxSchools {
				s.problem("school_limit_exceeded", node.Name, "", "edu_tenant_placement", true)
			} else {
				s.pinsRead = true
			}
		}
	}
	for _, table := range tables {
		if !has(table.name) {
			continue
		}
		query := "SELECT " + boundedInspectionText(table.column, 64, node.Driver) + ",COUNT(*) FROM " + table.name + " GROUP BY " + table.column + " ORDER BY " + table.column + " LIMIT 4097"
		args := []any{}
		if table.name == "edu_outbox" {
			query = "SELECT " + boundedInspectionText("tenant_id", 64, node.Driver) + ",COUNT(*),SUM(CASE WHEN lease_until>? AND lease_id IS NOT NULL AND published_at IS NULL AND dead_at IS NULL THEN 1 ELSE 0 END) FROM edu_outbox GROUP BY tenant_id ORDER BY tenant_id LIMIT 4097"
			args = append(args, now)
		}
		rows, err := tx.QueryContext(ctx, (database.Pool{Driver: node.Driver}).Bind(query), args...)
		if err != nil {
			s.problem("scope_query_failed", node.Name, "", table.name, true)
			return
		}
		var total int64
		groups := 0
		for rows.Next() {
			var id sql.NullString
			var count, leases int64
			dest := []any{&id, &count}
			if table.name == "edu_outbox" {
				dest = append(dest, &leases)
			}
			if err := rows.Scan(dest...); err != nil {
				rows.Close()
				s.problem("scope_query_failed", node.Name, "", table.name, true)
				return
			}
			groups++
			total += count
			school := s.school(id.String, node.Name, table.name)
			if school == nil {
				continue
			}
			school.Rows = append(school.Rows, InspectedScopeRows{Pool: node.Name, Table: table.name, Rows: count})
			if s.observed[id.String] == nil {
				s.observed[id.String] = map[string]bool{}
			}
			s.observed[id.String][node.Name] = true
			if leases > 0 {
				s.problem("active_outbox_lease", node.Name, id.String, table.name, false)
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			s.problem("scope_query_failed", node.Name, "", table.name, true)
			return
		}
		if groups > maxSchools {
			s.problem("school_limit_exceeded", node.Name, "", table.name, true)
		}
		pool.Tables = append(pool.Tables, InspectedTable{Table: table.name, Rows: total})
	}
}
func legacyMarkerValid(value string) bool { return marker.MatchString(value) }
func validLegacyOwner(value string) bool {
	if value == "shared" {
		return true
	}
	scope, ok := strings.CutPrefix(value, "tenant:")
	return ok && database.ValidTenant(scope)
}
func (s *inspectionState) reconcile(settings database.Settings) {
	markersComplete := true
	for _, node := range settings.Nodes() {
		if !s.markersRead[node.Name] {
			markersComplete = false
		}
	}
	for _, node := range settings.Dedicated {
		s.school(node.Tenant, node.Name, "")
	}
	for _, pools := range s.markers {
		if len(pools) > 1 {
			for _, pool := range pools {
				s.problem("duplicate_pool_marker", pool, "", "edu_pool_marker", false)
			}
		}
	}
	for _, pools := range s.owners {
		if len(pools) > 1 {
			for _, pool := range pools {
				s.problem("duplicate_pool_owner", pool, "", "edu_pool_marker", false)
			}
		}
	}
	for id, school := range s.schools {
		expected := settings.Control.Name
		for _, node := range settings.Dedicated {
			if node.Tenant == id {
				expected = node.Name
				break
			}
		}
		school.ExpectedPool = expected
		if s.identityRead && !school.Identity {
			s.problem("missing_identity", "", id, "identity_schools", false)
			if len(s.observed[id]) > 0 {
				s.problem("unknown_scope", "", id, "", false)
			}
		}
		if !school.Pinned && s.pinsRead {
			s.problem("unpinned_school", "", id, "edu_tenant_placement", false)
		} else if school.Pinned {
			pools := s.markers[s.pins[id]]
			if len(pools) > 1 {
				s.problem("ambiguous_pin_pool", "", id, "edu_tenant_placement", false)
			} else if !markersComplete {
				s.problem("pin_pool_unverified", "", id, "edu_tenant_placement", false)
			} else if len(pools) == 0 {
				s.problem("unknown_pin_pool", "", id, "edu_tenant_placement", false)
			} else if pools[0] != expected {
				s.problem("wrong_pin_pool", pools[0], id, "edu_tenant_placement", false)
			}
		}
		observed := s.observed[id]
		if len(observed) > 1 {
			s.problem("duplicate_school_ownership", "", id, "", false)
		}
		for pool := range observed {
			if pool != expected {
				s.problem("unexpected_pool_rows", pool, id, "", false)
			}
			if school.Pinned {
				pools := s.markers[s.pins[id]]
				if len(pools) == 1 && pool != pools[0] {
					s.problem("rows_outside_recorded_pool", pool, id, "", false)
				}
			}
		}
	}
}
