// Package cache keeps every verse the reader has already looked up, per
// translation, in a local SQLite file, so that a repeated lookup is
// instant and works offline.
//
// It only ever holds what was actually consulted: there is no bulk fill,
// by design (see package bibliaonline).
package cache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // database/sql driver "sqlite"
)

// Entry is one cached verse. Absent marks a verse the translation omits
// (e.g. Mt 17.21 in the NVI): it is cached too, so the server is not
// asked again for something that will never come.
type Entry struct {
	Text      string
	Absent    bool
	URL       string
	FetchedAt time.Time
}

// Cache wraps the SQLite database.
type Cache struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS verse (
	bible      TEXT    NOT NULL,
	book       TEXT    NOT NULL,
	chapter    INTEGER NOT NULL,
	verse      INTEGER NOT NULL,
	text       TEXT    NOT NULL DEFAULT '',
	absent     INTEGER NOT NULL DEFAULT 0,
	url        TEXT    NOT NULL DEFAULT '',
	fetched_at INTEGER NOT NULL,
	PRIMARY KEY (bible, book, chapter, verse)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS kv (
	key        TEXT PRIMARY KEY,
	value      TEXT NOT NULL,
	fetched_at INTEGER NOT NULL
) WITHOUT ROWID;
`

// Open opens (creating if needed) the cache database at path.
func Open(ctx context.Context, path string) (*Cache, error) {
	const dirPerm = 0o750
	err := os.MkdirAll(filepath.Dir(path), dirPerm)
	if err != nil {
		return nil, fmt.Errorf("cache: creating directory: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("cache: opening %s: %w", path, err)
	}

	_, err = db.ExecContext(ctx, schema)
	if err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("cache: creating schema: %w", err)
	}

	return &Cache{db: db}, nil
}

// Close releases the database.
func (c *Cache) Close() error {
	err := c.db.Close()
	if err != nil {
		return fmt.Errorf("cache: closing: %w", err)
	}

	return nil
}

// Get returns the cached entries for the given verses of a chapter,
// keyed by verse number. Verses not in the cache are simply missing
// from the map.
func (c *Cache) Get(
	ctx context.Context, bible, book string, chapter int, verses []int,
) (map[int]Entry, error) {
	out := map[int]Entry{}

	rows, err := c.db.QueryContext(ctx,
		`SELECT verse, text, absent, url, fetched_at FROM verse
		 WHERE bible = ? AND book = ? AND chapter = ?`,
		bible, book, chapter,
	)
	if err != nil {
		return nil, fmt.Errorf("cache: reading: %w", err)
	}
	defer func() { _ = rows.Close() }()

	want := map[int]bool{}
	for _, v := range verses {
		want[v] = true
	}

	for rows.Next() {
		var (
			v       int
			e       Entry
			absent  int
			fetched int64
		)

		err := rows.Scan(&v, &e.Text, &absent, &e.URL, &fetched)
		if err != nil {
			return nil, fmt.Errorf("cache: reading row: %w", err)
		}

		if len(want) > 0 && !want[v] {
			continue
		}

		e.Absent = absent != 0
		e.FetchedAt = time.Unix(fetched, 0)
		out[v] = e
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("cache: reading rows: %w", err)
	}

	return out, nil
}

// Put stores entries for one chapter, replacing any previous version.
func (c *Cache) Put(
	ctx context.Context, bible, book string, chapter int, entries map[int]Entry,
) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cache: starting transaction: %w", err)
	}

	for v, e := range entries {
		absent := 0
		if e.Absent {
			absent = 1
		}

		_, err := tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO verse
			 (bible, book, chapter, verse, text, absent, url, fetched_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			bible, book, chapter, v, e.Text, absent, e.URL, e.FetchedAt.Unix(),
		)
		if err != nil {
			_ = tx.Rollback()

			return fmt.Errorf("cache: writing: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("cache: committing: %w", err)
	}

	return nil
}

// GetValue returns a cached key/value (e.g. the translation catalog) if
// it is younger than maxAge.
func (c *Cache) GetValue(ctx context.Context, key string, maxAge time.Duration) (string, bool, error) {
	var (
		value   string
		fetched int64
	)

	err := c.db.QueryRowContext(ctx,
		`SELECT value, fetched_at FROM kv WHERE key = ?`, key,
	).Scan(&value, &fetched)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}

	if err != nil {
		return "", false, fmt.Errorf("cache: reading %s: %w", key, err)
	}

	if time.Since(time.Unix(fetched, 0)) > maxAge {
		return value, false, nil
	}

	return value, true, nil
}

// PutValue stores a key/value.
func (c *Cache) PutValue(ctx context.Context, key, value string) error {
	_, err := c.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO kv (key, value, fetched_at) VALUES (?, ?, ?)`,
		key, value, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("cache: writing %s: %w", key, err)
	}

	return nil
}

// Stats summarizes the cache: verse count per translation.
func (c *Cache) Stats(ctx context.Context) (map[string]int, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT bible, COUNT(*) FROM verse WHERE absent = 0 GROUP BY bible ORDER BY bible`,
	)
	if err != nil {
		return nil, fmt.Errorf("cache: stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]int{}

	for rows.Next() {
		var (
			bible string
			n     int
		)

		err := rows.Scan(&bible, &n)
		if err != nil {
			return nil, fmt.Errorf("cache: stats row: %w", err)
		}

		out[bible] = n
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("cache: stats rows: %w", err)
	}

	return out, nil
}

// Clear deletes every cached verse (and the catalog), optionally only
// for one translation.
func (c *Cache) Clear(ctx context.Context, bible string) (int64, error) {
	var (
		res sql.Result
		err error
	)

	if bible == "" {
		res, err = c.db.ExecContext(ctx, `DELETE FROM verse`)
		if err == nil {
			_, err = c.db.ExecContext(ctx, `DELETE FROM kv`)
		}
	} else {
		res, err = c.db.ExecContext(ctx, `DELETE FROM verse WHERE bible = ?`, bible)
	}

	if err != nil {
		return 0, fmt.Errorf("cache: clearing: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("cache: counting cleared rows: %w", err)
	}

	return n, nil
}
