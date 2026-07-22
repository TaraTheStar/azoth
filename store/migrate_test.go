// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// openMem opens a scratch database on a temp file (not :memory:, so WAL and
// a real user_version header behave as in production) for a migration test.
func openMem(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// userVersion reads the database's migration cursor.
func userVersion(t *testing.T, db *sql.DB) int {
	t.Helper()
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	return v
}

// tableExists reports whether a table is present — the observable effect of
// a CREATE TABLE migration having run (and committed).
func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&n)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	return n > 0
}

func TestMigrate_AppliesInVersionOrder(t *testing.T) {
	db := openMem(t)
	// Filenames whose lexical order (10 < 2) differs from numeric order, to
	// prove sorting is by parsed int, not string. Migration 10 references a
	// table 2 created — so it only compiles if 2 ran first.
	fsys := fstest.MapFS{
		"m/0002_a.sql": {Data: []byte(`CREATE TABLE a (id INTEGER PRIMARY KEY);`)},
		"m/0010_b.sql": {Data: []byte(`CREATE TABLE b (a_id INTEGER REFERENCES a(id));`)},
	}
	if err := Migrate(db, fsys, "m"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if got := userVersion(t, db); got != 10 {
		t.Fatalf("user_version = %d, want 10", got)
	}
	if !tableExists(t, db, "a") || !tableExists(t, db, "b") {
		t.Fatalf("expected both tables created")
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	db := openMem(t)
	fsys := fstest.MapFS{
		"migrations/0001_init.sql": {Data: []byte(`CREATE TABLE t (id INTEGER PRIMARY KEY);`)},
	}
	if err := Migrate(db, fsys, "migrations"); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	// A second run re-reads the same file but must apply nothing — a
	// re-run of `CREATE TABLE t` (no IF NOT EXISTS) would error if it did.
	if err := Migrate(db, fsys, "migrations"); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if got := userVersion(t, db); got != 1 {
		t.Fatalf("user_version = %d, want 1", got)
	}
}

func TestMigrate_OnlyNewerThanCursor(t *testing.T) {
	db := openMem(t)
	// Simulate a database already at version 1: only migration 2 should run.
	// (Version 1's body would fail if re-run, proving it is skipped.)
	if _, err := db.Exec(`PRAGMA user_version = 1`); err != nil {
		t.Fatalf("seed user_version: %v", err)
	}
	fsys := fstest.MapFS{
		"m/0001_bad.sql":  {Data: []byte(`THIS IS NOT VALID SQL;`)},
		"m/0002_good.sql": {Data: []byte(`CREATE TABLE t (id INTEGER PRIMARY KEY);`)},
	}
	if err := Migrate(db, fsys, "m"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if got := userVersion(t, db); got != 2 {
		t.Fatalf("user_version = %d, want 2", got)
	}
}

func TestMigrate_GapsAreFine(t *testing.T) {
	db := openMem(t)
	fsys := fstest.MapFS{
		"m/0001_a.sql": {Data: []byte(`CREATE TABLE a (id INTEGER PRIMARY KEY);`)},
		"m/0005_b.sql": {Data: []byte(`CREATE TABLE b (id INTEGER PRIMARY KEY);`)},
	}
	if err := Migrate(db, fsys, "m"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if got := userVersion(t, db); got != 5 {
		t.Fatalf("user_version = %d, want 5", got)
	}
}

func TestMigrate_DuplicateVersionErrors(t *testing.T) {
	db := openMem(t)
	fsys := fstest.MapFS{
		"m/0001_a.sql":     {Data: []byte(`CREATE TABLE a (id INTEGER PRIMARY KEY);`)},
		"m/0001_again.sql": {Data: []byte(`CREATE TABLE b (id INTEGER PRIMARY KEY);`)},
	}
	err := Migrate(db, fsys, "m")
	if err == nil || !strings.Contains(err.Error(), "duplicate migration version") {
		t.Fatalf("err = %v, want duplicate-version error", err)
	}
	// Nothing should have been applied — the check runs before any exec.
	if got := userVersion(t, db); got != 0 {
		t.Fatalf("user_version = %d, want 0", got)
	}
}

func TestMigrate_NoNumericPrefixErrors(t *testing.T) {
	db := openMem(t)
	fsys := fstest.MapFS{
		"m/init.sql": {Data: []byte(`CREATE TABLE t (id INTEGER PRIMARY KEY);`)},
	}
	err := Migrate(db, fsys, "m")
	if err == nil || !strings.Contains(err.Error(), "numeric version prefix") {
		t.Fatalf("err = %v, want numeric-prefix error", err)
	}
}

func TestMigrate_SkipsNonSQLAndDirs(t *testing.T) {
	db := openMem(t)
	fsys := fstest.MapFS{
		"m/0001_a.sql": {Data: []byte(`CREATE TABLE a (id INTEGER PRIMARY KEY);`)},
		"m/README.md":  {Data: []byte(`not a migration`)},
		"m/notes.txt":  {Data: []byte(`0002_looks_like_one but isn't sql`)},
		"m/sub/x.sql":  {Data: []byte(`CREATE TABLE should_not_run (id INTEGER);`)},
	}
	if err := Migrate(db, fsys, "m"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if got := userVersion(t, db); got != 1 {
		t.Fatalf("user_version = %d, want 1", got)
	}
	if tableExists(t, db, "should_not_run") {
		t.Fatalf("subdirectory migration was applied but should be skipped")
	}
}

func TestMigrate_AtomicRollbackOnBadBody(t *testing.T) {
	db := openMem(t)
	// Migration 2's second statement is invalid: the whole migration must
	// roll back — table t2 gone, cursor left at 1 (migration 1 committed).
	fsys := fstest.MapFS{
		"m/0001_ok.sql": {Data: []byte(`CREATE TABLE t1 (id INTEGER PRIMARY KEY);`)},
		"m/0002_bad.sql": {Data: []byte(
			`CREATE TABLE t2 (id INTEGER PRIMARY KEY); THIS IS NOT SQL;`)},
	}
	err := Migrate(db, fsys, "m")
	if err == nil || !strings.Contains(err.Error(), "0002_bad.sql") {
		t.Fatalf("err = %v, want failure naming 0002_bad.sql", err)
	}
	if got := userVersion(t, db); got != 1 {
		t.Fatalf("user_version = %d, want 1 (migration 1 committed, 2 rolled back)", got)
	}
	if !tableExists(t, db, "t1") {
		t.Fatalf("migration 1 should have committed (t1 missing)")
	}
	if tableExists(t, db, "t2") {
		t.Fatalf("migration 2 should have rolled back (t2 present)")
	}
}

func TestMigrate_MissingDirErrors(t *testing.T) {
	db := openMem(t)
	err := Migrate(db, fstest.MapFS{}, "migrations")
	if err == nil || !strings.Contains(err.Error(), "read migrations") {
		t.Fatalf("err = %v, want read-migrations error", err)
	}
}
