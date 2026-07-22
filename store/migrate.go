// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
)

// Migrate applies every migration in fsys under dir whose version is newer
// than the database's current PRAGMA user_version. A migration is a file
// named `NNNN_description.sql`, where the leading run of digits is its
// version (`0003_add_index.sql` → 3). Files are applied in ascending
// version order — not directory order — so inserting, removing, or gapping
// files can't silently shift a later migration's version and re-run or skip
// it against the wrong schema.
//
// Each migration's body and its matching `PRAGMA user_version = N` bump run
// in ONE transaction, so a failure mid-file rolls back the whole migration
// (user_version is part of the SQLite header and is itself transactional,
// so the cursor reverts too) — never a half-applied schema with a stale
// cursor. Migration N runs only if user_version < N.
//
// fsys is an fs.FS (not a concrete embed.FS) so the runner is unit-testable
// with fstest.MapFS; the caller keeps its own `//go:embed migrations/*.sql`
// and passes the embedded FS in. dir is the subdirectory to read (e.g.
// "migrations"). Non-.sql entries and subdirectories are skipped; a *.sql
// file with no numeric prefix, or two files sharing a version, is a
// programming error in the embedded set and is rejected loudly.
func Migrate(db *sql.DB, fsys fs.FS, dir string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	type migration struct {
		version int
		name    string
	}
	migs := make([]migration, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !hasSQLSuffix(e.Name()) {
			continue
		}
		v, err := migrationVersion(e.Name())
		if err != nil {
			return err
		}
		migs = append(migs, migration{version: v, name: e.Name()})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })
	for i := 1; i < len(migs); i++ {
		if migs[i].version == migs[i-1].version {
			return fmt.Errorf("duplicate migration version %d (%s and %s)", migs[i].version, migs[i-1].name, migs[i].name)
		}
	}

	var current int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&current); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	for _, m := range migs {
		if current >= m.version {
			continue
		}
		body, err := fs.ReadFile(fsys, dir+"/"+m.name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m.name, err)
		}
		if err := applyOne(db, string(body), m.version); err != nil {
			return fmt.Errorf("apply migration %s: %w", m.name, err)
		}
		current = m.version
	}
	return nil
}

// applyOne runs one migration's body and its user_version bump inside a
// single transaction, so the pair is atomic: any failure rolls the whole
// migration back and leaves the cursor put.
func applyOne(db *sql.DB, body string, version int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after a successful Commit
	if _, err := tx.Exec(body); err != nil {
		return err
	}
	// PRAGMA user_version = N — the value must be inlined; placeholders
	// aren't supported on PRAGMA statements. version is an int parsed from
	// a trusted embedded filename, so there is no injection surface.
	if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
		return err
	}
	return tx.Commit()
}

// migrationVersion parses the leading run of digits from a migration
// filename (`0003_messages_agent_id.sql` → 3). A file with no numeric
// prefix is rejected loudly rather than silently skipped.
func migrationVersion(name string) (int, error) {
	i := 0
	for i < len(name) && name[i] >= '0' && name[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("migration %q has no numeric version prefix", name)
	}
	return strconv.Atoi(name[:i])
}

// hasSQLSuffix reports whether name ends in ".sql". Kept local to avoid a
// strings import for one call.
func hasSQLSuffix(name string) bool {
	const ext = ".sql"
	return len(name) >= len(ext) && name[len(name)-len(ext):] == ext
}
