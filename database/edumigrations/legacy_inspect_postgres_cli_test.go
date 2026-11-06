package edumigrations_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/platform/database"
	"github.com/elcomware/edupilot/platform/tenantupgrade"
	"github.com/jackc/pgx/v5"
)

func TestLegacyPlacementInspectionPostgresActualCommand(t *testing.T) {
	drill := newPostgresDatabaseDrill(t)
	ctx, cfg, sslmode, tlsEnvironment := drill.ctx, drill.cfg, drill.sslmode, drill.tlsEnvironment
	shared, west := drill.create("shared"), drill.create("west")
	writers := map[string]*sql.DB{"postgres": shared.db, "tenant_west": west.db}
	markers := map[string]string{"postgres": "private-shared-" + inspectionCLIRandom(t), "tenant_west": "private-west-" + inspectionCLIRandom(t)}
	private := "PRIVATE-INSPECTION-CONTENT-" + inspectionCLIRandom(t)
	execute := func(db *sql.DB, query string, args ...any) {
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal("could not populate the private frozen PostgreSQL fixture")
		}
	}
	for _, registration := range loadStarterRegistry(t).Registrations {
		name := registration.Connection
		if name == "sqlite" {
			name = "postgres"
		}
		if writer := writers[name]; writer != nil {
			for _, statement := range registration.Up {
				execute(writer, strings.ReplaceAll(statement, "'<generated>'", "'"+markers[name]+"'"))
			}
		}
	}
	keys := inspectionKeys()
	if len(keys) != 41 {
		t.Fatal("expected the reviewed 41 work-list registrations")
	}
	tableNames := []string{"students_records", "enrollment_placements", "edu_outbox", "edu_audit_inbox"}
	for _, key := range keys {
		tableNames = append(tableNames, "ws_"+key)
	}
	slices.Sort(tableNames)
	for school, name := range map[string]string{"east": "postgres", "west": "tenant_west"} {
		writer := writers[name]
		execute(shared.db, "INSERT INTO identity_schools VALUES ($1)", school)
		execute(shared.db, "INSERT INTO edu_tenant_placement VALUES ($1,$2)", school, markers[name])
		execute(writer, "INSERT INTO students_records VALUES ($1,'same-record',$2,'Student','active',1)", school, private)
		execute(writer, "INSERT INTO enrollment_placements VALUES ($1,'same-placement','same-record','2026-2027','Grade 7','A','active',1)", school)
		execute(writer, "INSERT INTO edu_outbox(event_id,tenant_id,event_type,aggregate_id,payload,created_at,available_at,published_at) VALUES ('published',$1,'students.registered.v1','same-record',$2,1,1,2)", school, `{"private":"`+private+`"}`)
		execute(writer, "INSERT INTO edu_audit_inbox VALUES ('published',$1,'students.registered.v1','same-record',2)", school)
		for _, key := range keys {
			execute(writer, "INSERT INTO "+pgx.Identifier{"ws_" + key}.Sanitize()+" VALUES ($1,'same-record',$2,'open',1)", school, private)
		}
	}
	binary := filepath.Join(t.TempDir(), "tenant-upgrade")
	build := exec.CommandContext(ctx, "go", "build", "-buildvcs=false", "-o", binary, "../../cmd/tenant-upgrade")
	if _, err := build.CombinedOutput(); err != nil {
		t.Fatal("could not compile the native tenant-upgrade CLI")
	}
	node := database.Connection{Driver: "postgres", Host: cfg.Host, Port: int(cfg.Port), Username: cfg.User, Password: cfg.Password, SSLMode: sslmode}
	node.Tenant, node.Database = "west", west.name
	dedicated, err := json.Marshal([]database.Connection{node})
	if err != nil {
		t.Fatal("could not encode private native connection settings")
	}
	environment := map[string]string{"APP_KEY": "12345678901234567890123456789012", "DB_CONNECTION": "postgres", "DB_HOST": cfg.Host, "DB_PORT": strconv.Itoa(int(cfg.Port)), "DB_USERNAME": cfg.User, "DB_PASSWORD": cfg.Password, "DB_SSLMODE": sslmode, "DB_DATABASE": shared.name, "EDUPILOT_DEDICATED_DATABASES": string(dedicated)}
	for key, value := range tlsEnvironment {
		environment[key] = value
	}
	directory := t.TempDir() // No operator .env or deployment working files.
	secrets := []string{os.Getenv("EDUPILOT_TEST_POSTGRES_DSN"), shared.name, west.name, markers["postgres"], markers["tenant_west"], private, url.UserPassword(cfg.User, cfg.Password).String(), net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port)))}
	if len(cfg.Password) >= 16 {
		secrets = append(secrets, cfg.Password)
	}
	run := func(overrides map[string]string, wantStatus int) ([]byte, []byte) {
		t.Helper()
		command := exec.CommandContext(ctx, binary, "-mode=inspect-placement")
		command.Dir = directory
		values := make(map[string]string)
		for _, entry := range os.Environ() {
			key, value, _ := strings.Cut(entry, "=")
			if key != "EDUPILOT_TEST_POSTGRES_DSN" {
				values[key] = value
			}
		}
		for key, value := range environment {
			values[key] = value
		}
		for key, value := range overrides {
			values[key] = value
		}
		for key, value := range values {
			command.Env = append(command.Env, key+"="+value)
		}
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		status := 0
		if exit, ok := err.(*exec.ExitError); ok {
			status = exit.ExitCode()
		} else if err != nil {
			t.Fatal("native PostgreSQL CLI subprocess could not complete")
		}
		for _, secret := range secrets {
			if secret != "" && strings.Contains(stdout.String()+stderr.String(), secret) {
				t.Fatal("native PostgreSQL CLI exposed private source/configuration input")
			}
		}
		if status != wantStatus {
			t.Fatalf("native PostgreSQL CLI exit status %d; expected %d", status, wantStatus)
		}
		return stdout.Bytes(), stderr.Bytes()
	}
	readReport := func(output, diagnostics []byte) tenantupgrade.PlacementInspection {
		t.Helper()
		report, err := inspectionCLIReport(output)
		if len(diagnostics) != 0 || err != nil || report.Classification != "inspect-only" || report.Version != 1 {
			t.Fatal("native PostgreSQL CLI did not emit only its redacted JSON report")
		}
		return report
	}
	before := inspectionCLISnapshot(t, ctx, writers)
	output, diagnostics := run(nil, 0)
	report := readReport(output, diagnostics)
	if !report.Complete || len(report.Issues) != 0 || len(report.Pools) != 2 || len(report.Schools) != 2 {
		t.Fatal("native PostgreSQL CLI did not recognize the clean mixed placement")
	}
	for i, pool := range report.Pools {
		if pool.Name != []string{"postgres", "tenant_west"}[i] || len(pool.Tables) != 45 {
			t.Fatal("native PostgreSQL CLI omitted a named pool or required table counts")
		}
		for j, table := range pool.Tables {
			if table.Table != tableNames[j] || table.Rows != 1 {
				t.Fatal("native PostgreSQL CLI returned incorrect per-pool counts")
			}
		}
	}
	for i, school := range report.Schools {
		want := map[string]string{"east": "postgres", "west": "tenant_west"}[school.School]
		if school.School != []string{"east", "west"}[i] || want == "" || !school.Identity || !school.Pinned || school.ExpectedPool != want || len(school.Rows) != 45 {
			t.Fatal("native PostgreSQL CLI lost identity, recorded pins or mixed placement")
		}
		for j, rows := range school.Rows {
			if rows.Pool != want || rows.Table != tableNames[j] || rows.Rows != 1 {
				t.Fatal("native PostgreSQL CLI attributed rows to a sibling pool")
			}
		}
	}
	again, diagnostics := run(nil, 0)
	readReport(again, diagnostics)
	if !bytes.Equal(output, again) || !reflect.DeepEqual(before, inspectionCLISnapshot(t, ctx, writers)) {
		t.Fatal("native PostgreSQL CLI changed source rows/table set or produced unstable output")
	}
	// One targeted ownership fault, without changing the marker/pin/data mapping.
	execute(west.db, "UPDATE edu_pool_marker SET owner='tenant:east' WHERE singleton=1")
	damaged := inspectionCLISnapshot(t, ctx, writers)
	output, diagnostics = run(nil, 1)
	report = readReport(output, diagnostics)
	if !report.Complete || len(report.Issues) != 1 || report.Issues[0] != (tenantupgrade.InspectionIssue{Code: "wrong_pool_owner", Pool: "tenant_west", Table: "edu_pool_marker"}) {
		t.Fatal("native PostgreSQL CLI missed or misclassified the ownership fault")
	}
	if !reflect.DeepEqual(damaged, inspectionCLISnapshot(t, ctx, writers)) {
		t.Fatal("native PostgreSQL CLI repaired or mutated a damaged source")
	}
	// Opening an absent random database is read-only. No fixture DDL/cleanup is
	// ever attempted against it, or against the administrator's existing DB.
	missing := "edu_inspect_cli_missing_" + inspectionCLIRandom(t)
	failureSecret := "PRIVATE-CONNECTION-" + inspectionCLIRandom(t)
	secrets = append(secrets, missing, failureSecret)
	output, diagnostics = run(map[string]string{"DB_DATABASE": missing, "DB_PASSWORD": failureSecret, "EDUPILOT_DEDICATED_DATABASES": "[]"}, 1)
	report = readReport(output, diagnostics)
	if report.Complete || !issueCodes(report)["pool_unavailable"] {
		t.Fatal("native PostgreSQL CLI did not redact a failed source connection")
	}
	output, diagnostics = run(map[string]string{"DB_PORT": failureSecret}, 1)
	if len(output) != 0 || string(diagnostics) != "Cannot read native database configuration.\n" {
		t.Fatal("native PostgreSQL CLI did not return the generic configuration diagnostic")
	}
}

func inspectionCLIRandom(t *testing.T) string {
	t.Helper()
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatal("could not allocate random private database identity")
	}
	return hex.EncodeToString(value[:])
}

// Native settings expose one host and explicit SSL mode. Refuse unsupported
// URI options before creating databases, rather than silently weakening TLS or
// dropping settings. Certificate/SNI options use the pgx-supported PGSSL env.
func inspectionCLIConfig(dsn string) (*pgx.ConnConfig, string, map[string]string, error) {
	invalid := errors.New("native PostgreSQL CLI test requires a single-host PostgreSQL URL with explicit supported sslmode and optional TLS parameters")
	uri, err := url.Parse(dsn)
	if err != nil || (uri.Scheme != "postgres" && uri.Scheme != "postgresql") || uri.Hostname() == "" || strings.Contains(uri.Host, ",") || uri.Fragment != "" || os.Getenv("PGSERVICE") != "" {
		return nil, "", nil, invalid
	}
	query, err := url.ParseQuery(uri.RawQuery)
	if err != nil {
		return nil, "", nil, invalid
	}
	sslmode := query.Get("sslmode")
	if !slices.Contains([]string{"disable", "require", "verify-ca", "verify-full"}, sslmode) {
		return nil, "", nil, invalid
	}
	names := map[string]string{"sslrootcert": "PGSSLROOTCERT", "sslcert": "PGSSLCERT", "sslkey": "PGSSLKEY", "sslpassword": "PGSSLPASSWORD", "sslsni": "PGSSLSNI", "sslnegotiation": "PGSSLNEGOTIATION"}
	environment := map[string]string{}
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			environment[name] = value
		}
	}
	for key, values := range query {
		if len(values) != 1 || (key != "sslmode" && names[key] == "") {
			return nil, "", nil, invalid
		}
		if key != "sslmode" {
			environment[names[key]] = values[0]
		}
	}
	// The subprocess has a private working directory. Resolve certificate paths
	// against the same directory used by ParseConfig, retaining their meaning.
	for _, name := range []string{"PGSSLROOTCERT", "PGSSLCERT", "PGSSLKEY"} {
		value := environment[name]
		if value == "" || (name == "PGSSLROOTCERT" && value == "system") {
			continue
		}
		absolute, err := filepath.Abs(value)
		if err != nil {
			return nil, "", nil, invalid
		}
		environment[name] = absolute
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil || cfg.Host != uri.Hostname() || len(cfg.Fallbacks) != 0 {
		return nil, "", nil, invalid
	}
	return cfg, sslmode, environment, nil
}

// Hash schema names and exact stored row values, without logging source data.
func inspectionCLISnapshot(t *testing.T, ctx context.Context, sources map[string]*sql.DB) map[string]string {
	t.Helper()
	result := map[string]string{}
	for pool, db := range sources {
		tables, err := db.QueryContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY table_name")
		if err != nil {
			t.Fatal("could not read private fixture catalog")
		}
		defer tables.Close()
		var names []string
		for tables.Next() {
			var name string
			if err := tables.Scan(&name); err != nil {
				t.Fatal("could not read private fixture table name")
			}
			names = append(names, name)
		}
		if err := tables.Err(); err != nil {
			t.Fatal("private fixture catalog was incomplete")
		}
		tables.Close()
		for _, name := range names {
			rows, err := db.QueryContext(ctx, "SELECT row_to_json(source)::text FROM "+pgx.Identifier{name}.Sanitize()+" source ORDER BY row_to_json(source)::text")
			if err != nil {
				t.Fatal("could not read private fixture rows")
			}
			defer rows.Close()
			hash := sha256.New()
			for rows.Next() {
				var row string
				if err := rows.Scan(&row); err != nil {
					t.Fatal("could not read private fixture row")
				}
				hash.Write([]byte(row + "\n"))
			}
			if err := rows.Err(); err != nil {
				t.Fatal("private fixture rows were incomplete")
			}
			rows.Close()
			result[pool+"/"+name] = hex.EncodeToString(hash.Sum(nil))
		}
	}
	return result
}

// Freeze the permitted wire fields independently of production report DTOs.
// Adding a diagnostic/debug field to any production DTO must not silently
// widen this redaction oracle, including for incomplete/failed-source reports.
func inspectionCLIReport(output []byte) (tenantupgrade.PlacementInspection, error) {
	var permitted struct {
		Version        int    `json:"version"`
		Classification string `json:"classification"`
		Complete       bool   `json:"complete"`
		Pools          []struct {
			Name   string `json:"name"`
			Tables []struct {
				Table string `json:"table"`
				Rows  int64  `json:"rows"`
			} `json:"tables"`
		} `json:"pools"`
		Schools []struct {
			School       string `json:"school"`
			Identity     bool   `json:"identity"`
			Pinned       bool   `json:"pinned"`
			ExpectedPool string `json:"expectedPool"`
			Rows         []struct {
				Pool  string `json:"pool"`
				Table string `json:"table"`
				Rows  int64  `json:"rows"`
			} `json:"rows"`
		} `json:"schools"`
		Issues []struct {
			Code   string `json:"code"`
			Pool   string `json:"pool"`
			School string `json:"school"`
			Table  string `json:"table"`
		} `json:"issues"`
	}
	invalid := errors.New("native CLI report violates permitted output schema")
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&permitted) != nil || decoder.Decode(new(any)) != io.EOF {
		return tenantupgrade.PlacementInspection{}, invalid
	}
	var report tenantupgrade.PlacementInspection
	if json.Unmarshal(output, &report) != nil {
		return tenantupgrade.PlacementInspection{}, invalid
	}
	return report, nil
}

func TestLegacyPlacementInspectionCLIRejectsPrivateOutputFields(t *testing.T) {
	const clean = `{"version":1,"classification":"inspect-only","complete":true,"pools":[{"name":"postgres","tables":[{"table":"students_records","rows":1}]}],"schools":[{"school":"localhost","identity":true,"pinned":true,"expectedPool":"postgres","rows":[{"pool":"postgres","table":"students_records","rows":1}]}],"issues":[]}`
	const failed = `{"version":1,"classification":"inspect-only","complete":false,"pools":[{"name":"postgres","tables":[]}],"schools":[],"issues":[{"code":"pool_unavailable","pool":"postgres"}]}`
	const private = "PRIVATE-REDACTION-REGRESSION"
	first := func(value any) map[string]any { return value.([]any)[0].(map[string]any) }
	for name, raw := range map[string]string{"clean": clean, "failed_source": failed} {
		t.Run(name, func(t *testing.T) {
			if _, err := inspectionCLIReport([]byte(raw)); err != nil {
				t.Fatal("permitted native report rejected")
			}
			locations := map[string]func(map[string]any) map[string]any{
				"report": func(root map[string]any) map[string]any { return root },
				"pool":   func(root map[string]any) map[string]any { return first(root["pools"]) },
			}
			if name == "clean" {
				locations["table"] = func(root map[string]any) map[string]any { return first(first(root["pools"])["tables"]) }
				locations["school"] = func(root map[string]any) map[string]any { return first(root["schools"]) }
				locations["school_rows"] = func(root map[string]any) map[string]any { return first(first(root["schools"])["rows"]) }
			} else {
				locations["issue"] = func(root map[string]any) map[string]any { return first(root["issues"]) }
			}
			for location, selectObject := range locations {
				for detail, value := range map[string]any{"scalar": private, "nested": map[string]any{"password": private}} {
					t.Run(location+"/"+detail, func(t *testing.T) {
						var root map[string]any
						if json.Unmarshal([]byte(raw), &root) != nil {
							t.Fatal("invalid synthetic native report")
						}
						selectObject(root)["debug"] = value
						output, err := json.Marshal(root)
						if err != nil {
							t.Fatal("could not encode synthetic native report")
						}
						if _, err = inspectionCLIReport(output); err == nil || strings.Contains(err.Error(), private) {
							t.Fatal("unknown private output accepted or echoed in diagnostic")
						}
					})
				}
			}
		})
	}
}

func TestLegacyPlacementInspectionCLIConfigPreservesTLS(t *testing.T) {
	t.Setenv("PGSERVICE", "")
	for _, name := range []string{"PGSSLROOTCERT", "PGSSLCERT", "PGSSLKEY", "PGSSLPASSWORD", "PGSSLSNI", "PGSSLNEGOTIATION"} {
		t.Setenv(name, "")
	}
	for _, mode := range []string{"disable", "require", "verify-ca", "verify-full"} {
		cfg, got, env, err := inspectionCLIConfig("postgres://fixture:private@localhost:5432/fixture?sslmode=" + mode + "&sslsni=0")
		if err != nil || got != mode || env["PGSSLSNI"] != "0" || (mode == "disable") != (cfg.TLSConfig == nil) {
			t.Fatal("native PostgreSQL CLI test changed configured TLS policy")
		}
	}
	for _, dsn := range []string{"host=localhost password=PRIVATE", "postgres://localhost/fixture", "postgres://localhost/fixture?sslmode=prefer", "postgres://localhost/fixture?sslmode=disable&sslmode=require", "postgres://localhost/fixture?sslmode=disable&options=PRIVATE"} {
		if _, _, _, err := inspectionCLIConfig(dsn); err == nil || strings.Contains(err.Error(), "PRIVATE") {
			t.Fatal("unsupported native test settings accepted or exposed")
		}
	}
}
