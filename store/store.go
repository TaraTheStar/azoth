// SPDX-License-Identifier: AGPL-3.0-or-later

// Package store is the shared SQLite harness: opening a database with the
// standard DSN and pragmas, and running a set of embedded migrations keyed
// on SQLite's PRAGMA user_version. It is deliberately schema-agnostic —
// Open returns a raw *sql.DB and Migrate takes the caller's migration files
// as an fs.FS. Each application keeps its own Store wrapper, its own query
// surface, and its own migrations/ tree; only the open-and-migrate plumbing
// lives here, so a fix to the pragma set or the migration runner lands once.
//
//	db, err := store.Open(dbPath)          // WAL, foreign_keys, busy_timeout
//	if err != nil { ... }
//	err = store.Migrate(db, migrationFS, "migrations")
//
// The package blank-imports the pure-Go modernc.org/sqlite driver, so
// callers don't need to; there is no cgo and tests run anywhere.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// dsn builds the connection string for a database file. The pragmas are the
// shared house policy: WAL for concurrent readers alongside a writer,
// foreign_keys(1) because SQLite leaves enforcement off by default, and a
// 5s busy_timeout so a briefly-locked database waits instead of failing.
func dsn(path string) string {
	return "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
}

// Open opens (creating if needed) the SQLite database at path with the
// standard pragmas, and returns the raw handle. The parent directory is
// created 0700 and its mode clamped to 0700 on every call, so an install
// pre-dating the tightening is upgraded rather than left world-readable.
// The connection is Pinged before returning so a bad path or driver error
// surfaces here, not on the first query.
//
// The returned *sql.DB is the caller's to migrate (see Migrate), wrap, and
// Close.
func Open(path string) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	// MkdirAll only sets the mode when it creates the directory; clamp it
	// on every Open so a pre-existing looser directory is tightened too.
	_ = os.Chmod(dir, 0o700)
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping %s: %w", path, err)
	}
	return db, nil
}
