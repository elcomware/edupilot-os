package tenantupgrade

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/elcomware/edupilot/platform/database"
)

var ErrSourceSchema = errors.New("source schema is incompatible")

const (
	maxSourceRows     = 10_000_000
	maxSourceCellSize = 16 << 20
)

type sourceKind string

const (
	sourceText      sourceKind = "text"
	sourceInteger   sourceKind = "integer"
	sourceBinary    sourceKind = "binary"
	sourceTimestamp sourceKind = "timestamp"
	sourceReal      sourceKind = "real"
	sourceBoolean   sourceKind = "boolean"
)

type sourceColumn struct {
	name     string
	kind     sourceKind
	nullable bool
}

type sourceTable struct {
	name        string
	scope       string
	key         []string
	columns     []sourceColumn
	controlOnly bool
}

// SourceTableInventory exposes only aggregate evidence. Row values and primary
// keys never appear in the report.
type SourceTableInventory struct {
	Name         string `json:"name"`
	Rows         int64  `json:"rows"`
	SchemaSHA256 string `json:"schemaSha256"`
	SHA256       string `json:"sha256"`
}

type sourceSchemaColumn struct {
	name     string
	kind     sourceKind
	nullable bool
	pk       int
}

var sourceIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func inventorySourceTable(ctx context.Context, tx *sql.Tx, driver string, spec sourceTable, allowed map[string]bool) (SourceTableInventory, []string, error) {
	if err := validateSourceTableSpec(spec); err != nil {
		return SourceTableInventory{}, nil, err
	}
	actual, err := inspectSourceSchema(ctx, tx, driver, spec.name)
	if err != nil {
		return SourceTableInventory{}, nil, fmt.Errorf("%w: %s", ErrSourceSchema, spec.name)
	}
	if err := compareSourceSchema(spec, actual); err != nil {
		return SourceTableInventory{}, nil, err
	}
	schemaHash := sha256.New()
	writeFrame(schemaHash, []byte("EDUPILOT-SOURCE-SCHEMA/1"))
	writeFrame(schemaHash, []byte(spec.name))
	for _, column := range spec.columns {
		writeFrame(schemaHash, []byte(column.name))
		writeFrame(schemaHash, []byte(column.kind))
		if column.nullable {
			writeFrame(schemaHash, []byte{1})
		} else {
			writeFrame(schemaHash, []byte{0})
		}
	}
	for _, key := range spec.key {
		writeFrame(schemaHash, []byte(key))
	}

	query := "SELECT " + joinSourceIdentifiers(spec.columns) + " FROM " + quoteSourceIdentifier(spec.name) + " ORDER BY " + joinQuotedIdentifiers(spec.key)
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return SourceTableInventory{}, nil, errors.New("could not read frozen source table")
	}
	defer rows.Close()
	hash := sha256.New()
	writeFrame(hash, []byte("EDUPILOT-SOURCE-TABLE/1"))
	writeFrame(hash, []byte(spec.name))
	unknown := map[string]bool{}
	var count int64
	for rows.Next() {
		count++
		if count > maxSourceRows {
			return SourceTableInventory{}, nil, errors.New("source table exceeds row limit")
		}
		values := make([]any, len(spec.columns))
		dest := make([]any, len(values))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return SourceTableInventory{}, nil, errors.New("could not scan frozen source table")
		}
		writeFrame(hash, []byte("row"))
		for i, column := range spec.columns {
			canonical, err := canonicalSourceValue(values[i], column)
			if err != nil {
				return SourceTableInventory{}, nil, err
			}
			writeFrame(hash, canonical)
			if column.name == spec.scope && canonical != nil {
				school := string(canonical[1:])
				if !allowed[school] {
					unknown[school] = true
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return SourceTableInventory{}, nil, errors.New("could not read frozen source table")
	}
	unknownList := make([]string, 0, len(unknown))
	for school := range unknown {
		unknownList = append(unknownList, school)
	}
	slices.Sort(unknownList)
	return SourceTableInventory{
		Name: spec.name, Rows: count,
		SchemaSHA256: hex.EncodeToString(schemaHash.Sum(nil)),
		SHA256:       hex.EncodeToString(hash.Sum(nil)),
	}, unknownList, nil
}

func validateSourceTableSpec(spec sourceTable) error {
	if !sourceIdentifier.MatchString(spec.name) || len(spec.columns) == 0 || len(spec.key) == 0 {
		return ErrSourceSchema
	}
	seen := map[string]bool{}
	for _, column := range spec.columns {
		if !sourceIdentifier.MatchString(column.name) || seen[column.name] {
			return ErrSourceSchema
		}
		seen[column.name] = true
	}
	keySeen := map[string]bool{}
	for _, key := range spec.key {
		if !seen[key] || keySeen[key] {
			return ErrSourceSchema
		}
		keySeen[key] = true
	}
	if spec.scope != "" {
		validScope := false
		for _, column := range spec.columns {
			if column.name == spec.scope && column.kind == sourceText && !column.nullable {
				validScope = true
			}
		}
		if !validScope {
			return ErrSourceSchema
		}
	}
	return nil
}

func inspectSourceSchema(ctx context.Context, tx *sql.Tx, driver, table string) ([]sourceSchemaColumn, error) {
	if driver == "sqlite" {
		rows, err := tx.QueryContext(ctx, "PRAGMA table_info("+quoteSourceIdentifier(table)+")")
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var columns []sourceSchemaColumn
		for rows.Next() {
			var cid, notNull, pk int
			var name, declared string
			var defaultValue any
			if err := rows.Scan(&cid, &name, &declared, &notNull, &defaultValue, &pk); err != nil {
				return nil, err
			}
			kind, ok := normalizeSourceKind(declared)
			if !ok {
				return nil, ErrSourceSchema
			}
			columns = append(columns, sourceSchemaColumn{name: name, kind: kind, nullable: notNull == 0 && pk == 0, pk: pk})
		}
		return columns, rows.Err()
	}
	if driver != "postgres" {
		return nil, ErrSourceSchema
	}
	rows, err := tx.QueryContext(ctx, `SELECT c.column_name,c.data_type,c.is_nullable,
		COALESCE(k.ordinal_position,0) FROM information_schema.columns c LEFT JOIN (
		SELECT u.column_name,u.ordinal_position FROM information_schema.table_constraints t
		JOIN information_schema.key_column_usage u ON t.constraint_name=u.constraint_name AND t.table_schema=u.table_schema AND t.table_name=u.table_name
		WHERE t.constraint_type='PRIMARY KEY' AND t.table_schema=current_schema() AND t.table_name=$1) k ON k.column_name=c.column_name
		WHERE c.table_schema=current_schema() AND c.table_name=$1 ORDER BY c.ordinal_position`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var columns []sourceSchemaColumn
	for rows.Next() {
		var name, declared, nullable string
		var pk int
		if err := rows.Scan(&name, &declared, &nullable, &pk); err != nil {
			return nil, err
		}
		kind, ok := normalizeSourceKind(declared)
		if !ok {
			return nil, ErrSourceSchema
		}
		columns = append(columns, sourceSchemaColumn{name: name, kind: kind, nullable: nullable == "YES", pk: pk})
	}
	return columns, rows.Err()
}

func compareSourceSchema(spec sourceTable, actual []sourceSchemaColumn) error {
	if len(actual) != len(spec.columns) {
		return fmt.Errorf("%w: %s", ErrSourceSchema, spec.name)
	}
	keys := map[string]int{}
	for i, key := range spec.key {
		keys[key] = i + 1
	}
	for i, expected := range spec.columns {
		column := actual[i]
		if column.name != expected.name || column.kind != expected.kind || column.nullable != expected.nullable || column.pk != keys[expected.name] {
			return fmt.Errorf("%w: %s", ErrSourceSchema, spec.name)
		}
	}
	return nil
}

func normalizeSourceKind(declared string) (sourceKind, bool) {
	t := strings.ToLower(strings.TrimSpace(declared))
	switch {
	case strings.Contains(t, "int"):
		return sourceInteger, true
	case strings.Contains(t, "bytea") || strings.Contains(t, "blob") || strings.Contains(t, "binary"):
		return sourceBinary, true
	case strings.Contains(t, "timestamp") || strings.Contains(t, "datetime") || t == "date":
		return sourceTimestamp, true
	case strings.Contains(t, "double") || strings.Contains(t, "real") || strings.Contains(t, "numeric") || strings.Contains(t, "decimal"):
		return sourceReal, true
	case t == "boolean" || t == "bool":
		return sourceBoolean, true
	case strings.Contains(t, "char") || strings.Contains(t, "text") || strings.Contains(t, "clob") || t == "uuid" || t == "json" || t == "jsonb":
		return sourceText, true
	default:
		return "", false
	}
}

func canonicalSourceValue(value any, column sourceColumn) ([]byte, error) {
	if value == nil {
		if !column.nullable {
			return nil, errors.New("non-null source column contains NULL")
		}
		return []byte{0}, nil
	}
	var text string
	switch column.kind {
	case sourceText:
		switch v := value.(type) {
		case string:
			text = v
		case []byte:
			text = string(v)
		default:
			return nil, errors.New("source text has incompatible storage type")
		}
		if !utf8.ValidString(text) {
			return nil, errors.New("source text is not UTF-8")
		}
	case sourceBinary:
		v, ok := value.([]byte)
		if !ok {
			return nil, errors.New("source binary has incompatible storage type")
		}
		if len(v) > maxSourceCellSize {
			return nil, errors.New("source value exceeds cell limit")
		}
		return append([]byte{byte('b')}, v...), nil
	case sourceInteger:
		var err error
		text, err = canonicalInteger(value)
		if err != nil {
			return nil, err
		}
	case sourceTimestamp:
		parsed, err := canonicalTimestamp(value)
		if err != nil {
			return nil, err
		}
		text = parsed.UTC().Format(time.RFC3339Nano)
	case sourceReal:
		var f float64
		switch v := value.(type) {
		case float64:
			f = v
		case float32:
			f = float64(v)
		default:
			parsed, err := strconv.ParseFloat(fmt.Sprint(v), 64)
			if err != nil {
				return nil, errors.New("source real is invalid")
			}
			f = parsed
		}
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, errors.New("source real is not finite")
		}
		text = strconv.FormatFloat(f, 'g', -1, 64)
	case sourceBoolean:
		switch v := value.(type) {
		case bool:
			text = strconv.FormatBool(v)
		case int64:
			if v != 0 && v != 1 {
				return nil, errors.New("source boolean is invalid")
			}
			text = strconv.FormatBool(v == 1)
		default:
			return nil, errors.New("source boolean has incompatible storage type")
		}
	default:
		return nil, ErrSourceSchema
	}
	if len(text) > maxSourceCellSize {
		return nil, errors.New("source value exceeds cell limit")
	}
	tags := map[sourceKind]byte{sourceText: 's', sourceInteger: 'i', sourceTimestamp: 'z', sourceReal: 'r', sourceBoolean: 'o'}
	return append([]byte{tags[column.kind]}, text...), nil
}

func canonicalInteger(value any) (string, error) {
	switch v := value.(type) {
	case int64:
		return strconv.FormatInt(v, 10), nil
	case int:
		return strconv.Itoa(v), nil
	case []byte:
		if n, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			return strconv.FormatInt(n, 10), nil
		}
	case string:
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return strconv.FormatInt(n, 10), nil
		}
	}
	return "", errors.New("source integer is invalid")
}

func canonicalTimestamp(value any) (time.Time, error) {
	if v, ok := value.(time.Time); ok {
		return v, nil
	}
	var raw string
	switch v := value.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		return time.Time{}, errors.New("source timestamp has incompatible storage type")
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999Z07:00", "2006-01-02 15:04:05.999999999"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, errors.New("source timestamp is invalid")
}

type hashWriter interface{ Write([]byte) (int, error) }

func writeFrame(writer hashWriter, value []byte) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = writer.Write(size[:])
	_, _ = writer.Write(value)
}

func quoteSourceIdentifier(value string) string { return `"` + value + `"` }

func joinSourceIdentifiers(columns []sourceColumn) string {
	values := make([]string, len(columns))
	for i := range columns {
		values[i] = quoteSourceIdentifier(columns[i].name)
	}
	return strings.Join(values, ",")
}

func joinQuotedIdentifiers(values []string) string {
	quoted := make([]string, len(values))
	for i := range values {
		quoted[i] = quoteSourceIdentifier(values[i])
	}
	return strings.Join(quoted, ",")
}

// SourceInventory is safe to archive as migration evidence: it contains only
// table counts, normalized schema/content digests, configured pool names and
// issue codes. It intentionally omits historical scope IDs and row values.
type SourceInventory struct {
	Version        int                     `json:"version"`
	Classification string                  `json:"classification"`
	Complete       bool                    `json:"complete"`
	ReadyForCopy   bool                    `json:"readyForCopy,omitempty"`
	Schools        int                     `json:"schools"`
	Documents      *SourceDocumentEvidence `json:"documents,omitempty"`
	Pools          []SourcePoolInventory   `json:"pools"`
	Issues         []SourceInventoryIssue  `json:"issues"`
	SHA256         string                  `json:"sha256,omitempty"`
}

type SourcePoolInventory struct {
	Name   string                 `json:"name"`
	Tables []SourceTableInventory `json:"tables"`
	SHA256 string                 `json:"sha256,omitempty"`
}

type SourceInventoryIssue struct {
	Code  string `json:"code"`
	Pool  string `json:"pool,omitempty"`
	Table string `json:"table,omitempty"`
}

// InventorySource opens every configured source in read-only mode and obtains
// one repeatable snapshot per physical pool. The operator must freeze all pools;
// no cross-database atomic snapshot is claimed.
func InventorySource(ctx context.Context, settings database.Settings, workspaceKeys []string, selection SourceSelection) SourceInventory {
	return inventorySource(ctx, settings, workspaceKeys, selection, readSnapshots(openInspectionDB))
}

func inventorySource(ctx context.Context, settings database.Settings, workspaceKeys []string, selection SourceSelection, snapshot sourceSnapshots) SourceInventory {
	classification := "frozen-source-sql"
	if selection.Version == 2 {
		classification = "frozen-source-sql-objects"
	}
	report := SourceInventory{Version: selection.Version, Classification: classification, Complete: true, Schools: len(selection.Schools), Pools: []SourcePoolInventory{}, Issues: []SourceInventoryIssue{}}
	issues := map[SourceInventoryIssue]bool{}
	problem := func(code, pool, table string) {
		issue := SourceInventoryIssue{Code: code, Pool: pool, Table: table}
		if !issues[issue] {
			report.Issues = append(report.Issues, issue)
			issues[issue] = true
		}
		report.Complete = false
	}
	if err := selection.Validate(); err != nil || settings.Validate() != nil || len(settings.Nodes()) > 128 {
		problem("invalid_configuration", "", "")
		return report
	}
	tables, err := sourceRegistry(workspaceKeys)
	if err != nil {
		problem("invalid_workspace_registry", "", "")
		return report
	}
	if selection.Documents.Present {
		documents := selection.Documents
		report.Documents = &documents
	}
	allowed := make(map[string]bool, len(selection.Schools))
	for _, school := range selection.Schools {
		allowed[school] = true
	}

	// Placement reconciliation has its own qualified, bounded reader. Reduce
	// its details to safe issue codes; never copy historical school IDs.
	pins := requireRecordedPin
	if selection.Version == 2 {
		pins = allowInitialPlacement
	}
	placement := inspectPlacement(ctx, settings, workspaceKeys, snapshot, pins)
	if !placement.Complete || len(placement.Issues) > 0 {
		for _, issue := range placement.Issues {
			problem("placement_"+issue.Code, issue.Pool, issue.Table)
		}
	}
	observedSchools := make([]string, 0, len(placement.Schools))
	for _, school := range placement.Schools {
		observedSchools = append(observedSchools, school.School)
	}
	slices.Sort(observedSchools)
	if !slices.Equal(observedSchools, selection.Schools) {
		problem("source_school_mismatch", "", "identity_schools")
	}

	nodes := settings.Nodes()
	slices.SortFunc(nodes, func(a, b database.Connection) int { return strings.Compare(a.Name, b.Name) })
	for _, node := range nodes {
		poolReport := SourcePoolInventory{Name: node.Name, Tables: []SourceTableInventory{}}
		poolOK := true
		tx, release, err := snapshot(ctx, node)
		if err != nil {
			problem("pool_unavailable", node.Name, "")
			report.Pools = append(report.Pools, poolReport)
			continue
		}
		for _, table := range tables {
			if table.controlOnly && node.Tenant != "" {
				continue
			}
			entry, unknown, err := inventorySourceTable(ctx, tx, node.Driver, table, allowed)
			if err != nil {
				code := "source_read_failed"
				if errors.Is(err, ErrSourceSchema) {
					code = "source_schema_mismatch"
				}
				problem(code, node.Name, table.name)
				poolOK = false
				continue
			}
			poolReport.Tables = append(poolReport.Tables, entry)
			if len(unknown) > 0 {
				problem("unknown_school_scope", node.Name, table.name)
				poolOK = false
			}
		}
		release()
		if poolOK {
			poolReport.SHA256 = digestSourcePool(poolReport)
		}
		report.Pools = append(report.Pools, poolReport)
	}
	slices.SortFunc(report.Issues, func(a, b SourceInventoryIssue) int {
		if order := strings.Compare(a.Code, b.Code); order != 0 {
			return order
		}
		if order := strings.Compare(a.Pool, b.Pool); order != 0 {
			return order
		}
		return strings.Compare(a.Table, b.Table)
	})
	if report.Complete {
		report.SHA256 = digestSourceReport(report)
		report.ReadyForCopy = report.Version == 2
	}
	return report
}

func digestSourcePool(pool SourcePoolInventory) string {
	hash := sha256.New()
	writeFrame(hash, []byte("EDUPILOT-SOURCE-POOL/1"))
	writeFrame(hash, []byte(pool.Name))
	for _, table := range pool.Tables {
		writeFrame(hash, []byte(table.Name))
		writeFrame(hash, []byte(strconv.FormatInt(table.Rows, 10)))
		writeFrame(hash, []byte(table.SchemaSHA256))
		writeFrame(hash, []byte(table.SHA256))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func digestSourceReport(report SourceInventory) string {
	hash := sha256.New()
	domain := "EDUPILOT-SOURCE-INVENTORY/1"
	if report.Version == 2 {
		domain = "EDUPILOT-SOURCE-INVENTORY/2"
	}
	writeFrame(hash, []byte(domain))
	writeFrame(hash, []byte(strconv.Itoa(report.Schools)))
	for _, pool := range report.Pools {
		writeFrame(hash, []byte(pool.Name))
		writeFrame(hash, []byte(pool.SHA256))
	}
	if report.Documents != nil {
		writeFrame(hash, []byte("documents"))
		if report.Version == 2 {
			writeFrame(hash, []byte(strconv.FormatInt(report.Documents.References, 10)))
			writeFrame(hash, []byte(report.Documents.ReferencesSHA256))
			writeFrame(hash, []byte(report.Documents.InventorySHA256))
		}
		writeFrame(hash, []byte(strconv.FormatInt(report.Documents.Documents, 10)))
		writeFrame(hash, []byte(strconv.FormatInt(report.Documents.ContentBytes, 10)))
		writeFrame(hash, []byte(report.Documents.SHA256))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func sourceRegistry(workspaceKeys []string) ([]sourceTable, error) {
	keys := slices.Clone(workspaceKeys)
	slices.Sort(keys)
	validKey := regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)
	if len(keys) != 41 {
		return nil, errors.New("workspace registry must contain 41 modules")
	}
	for i, key := range keys {
		if !validKey.MatchString(key) || (i > 0 && keys[i-1] == key) {
			return nil, errors.New("invalid workspace registry")
		}
	}
	t := func(name string, nullable bool) sourceColumn {
		return sourceColumn{name: name, kind: sourceText, nullable: nullable}
	}
	i := func(name string, nullable bool) sourceColumn {
		return sourceColumn{name: name, kind: sourceInteger, nullable: nullable}
	}
	ts := func(name string, nullable bool) sourceColumn {
		return sourceColumn{name: name, kind: sourceTimestamp, nullable: nullable}
	}
	tables := []sourceTable{
		{name: "edu_pool_marker", key: []string{"singleton"}, columns: []sourceColumn{i("singleton", false), t("owner", false), t("pool_id", false)}},
		{name: "edu_migration_receipts", key: []string{"signature"}, columns: []sourceColumn{t("signature", false)}},
		{name: "edu_outbox", scope: "tenant_id", key: []string{"event_id"}, columns: []sourceColumn{t("event_id", false), t("tenant_id", false), t("event_type", false), t("aggregate_id", false), t("payload", false), i("created_at", false), i("available_at", false), i("attempts", false), t("lease_id", true), i("lease_until", true), i("published_at", true), i("dead_at", true), t("last_error", true)}},
		{name: "edu_audit_inbox", scope: "tenant_id", key: []string{"event_id"}, columns: []sourceColumn{t("event_id", false), t("tenant_id", false), t("event_type", false), t("aggregate_id", false), i("received_at", false)}},
		{name: "students_records", scope: "school_id", key: []string{"school_id", "id"}, columns: []sourceColumn{t("school_id", false), t("id", false), t("given_name", false), t("family_name", false), t("status", false), i("version", false)}},
		{name: "enrollment_placements", scope: "school_id", key: []string{"school_id", "id"}, columns: []sourceColumn{t("school_id", false), t("id", false), t("student_id", false), t("academic_year", false), t("grade", false), t("course", false), t("status", false), i("version", false)}},
		{name: "edu_tenant_placement", scope: "tenant_id", key: []string{"tenant_id"}, controlOnly: true, columns: []sourceColumn{t("tenant_id", false), t("pool_id", false)}},
		{name: "identity_schools", scope: "school_id", key: []string{"school_id"}, controlOnly: true, columns: []sourceColumn{t("school_id", false)}},
		{name: "identity_accounts", scope: "school_id", key: []string{"school_id", "id"}, controlOnly: true, columns: []sourceColumn{t("school_id", false), t("id", false), t("email", false), t("password_hash", false)}},
		{name: "identity_memberships", scope: "school_id", key: []string{"school_id", "user_id"}, controlOnly: true, columns: []sourceColumn{t("school_id", false), t("user_id", false), t("role", false)}},
		{name: "identity_sessions", scope: "school_id", key: []string{"token_hash"}, controlOnly: true, columns: []sourceColumn{t("token_hash", false), t("school_id", false), t("user_id", false), i("expires_at", false)}},
		{name: "migrations", key: []string{"id"}, controlOnly: true, columns: []sourceColumn{i("id", false), t("migration", false), i("batch", false)}},
		{name: "jobs", key: []string{"id"}, controlOnly: true, columns: []sourceColumn{i("id", false), t("queue", false), t("payload", false), i("attempts", false), ts("reserved_at", true), ts("available_at", false), ts("created_at", false)}},
		{name: "failed_jobs", key: []string{"id"}, controlOnly: true, columns: []sourceColumn{i("id", false), t("uuid", false), t("connection", false), t("queue", false), t("payload", false), t("exception", false), ts("failed_at", false)}},
	}
	for _, key := range keys {
		tables = append(tables, sourceTable{name: "ws_" + key, scope: "school_id", key: []string{"school_id", "id"}, columns: []sourceColumn{t("school_id", false), t("id", false), t("title", false), t("status", false), i("version", false)}})
	}
	slices.SortFunc(tables, func(a, b sourceTable) int { return strings.Compare(a.name, b.name) })
	return tables, nil
}
