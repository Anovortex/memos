package postgres

import (
	"context"
	"database/sql"
	"log"

	// Import the PostgreSQL driver.
	_ "github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/store"
)

type DB struct {
	db      *sql.DB
	profile *profile.Profile
}

func NewDB(profile *profile.Profile) (store.Driver, error) {
	if profile == nil {
		return nil, errors.New("profile is nil")
	}

	// Open the PostgreSQL connection
	db, err := sql.Open("postgres", profile.DSN)
	if err != nil {
		log.Printf("Failed to open database: %s", err)
		return nil, errors.Wrapf(err, "failed to open database: %s", profile.DSN)
	}

	var driver store.Driver = &DB{
		db:      db,
		profile: profile,
	}

	// Return the DB struct
	return driver, nil
}

func (d *DB) GetDB() *sql.DB {
	return d.db
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) IsInitialized(ctx context.Context) (bool, error) {
	var exists bool
	// to_regclass resolves through the connection's search_path, so this matches the table the
	// queries actually hit. information_schema.tables would find a 'memo' in *any* schema and
	// wrongly report the DB as initialized, skipping LATEST.sql -> `relation "user" does not exist`.
	err := d.db.QueryRowContext(ctx, "SELECT to_regclass('memo') IS NOT NULL").Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if database is initialized")
	}
	return exists, nil
}

// GetDatabaseSize returns the database size in bytes, or -1 if unavailable.
func (d *DB) GetDatabaseSize(ctx context.Context) (int64, error) {
	var size int64
	const q = `SELECT pg_database_size(current_database())`
	if err := d.db.QueryRowContext(ctx, q).Scan(&size); err != nil {
		return -1, errors.Wrap(err, "failed to query postgres database size")
	}
	return size, nil
}
