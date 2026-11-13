// Package ubs reads the open-license data of the United Bible Societies
// (github.com/ubsicap/ubs-open-license, CC BY-SA 4.0):
//
//   - the UBS Dictionary of Biblical Hebrew, adapted from the Semantic
//     Dictionary of Biblical Hebrew, in Portuguese among other languages;
//   - the UBS Dictionary of the Greek New Testament, a revision of Louw &
//     Nida's lexicon based on semantic domains (English, Spanish, French);
//   - the Paratext Parallel Passages database (Gospel parallels, Kings and
//     Chronicles, Old Testament quotations in the New).
//
// Both dictionaries are organized by meaning: one Strong number may have
// several senses, and each sense lists the verses that use it, so the
// sense of a word in a given verse can be found. Their references, like
// the parallel passages', follow the numbering of the original text
// (Hebrew Sl 51.3 is the translations' Sl 51.1).
//
// The files are downloaded on demand and indexed into SQLite, like the
// STEPBible data; this package stands alone, so book codes and numbers
// are plain values here.
package ubs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite" // database/sql driver "sqlite"

	"github.com/casara/biblia-cli/internal/download"
)

// Repo is the UBS open-license repository on GitHub.
const Repo = "ubsicap/ubs-open-license"

// ParallelsPath is the parallel passages database in the repository.
const ParallelsPath = "parallel passages/ParallelPassages.xml"

// Sentinel errors, matchable with errors.Is.
var (
	// ErrNotInstalled is returned when the UBS data has not been
	// downloaded yet.
	ErrNotInstalled = errors.New("dados da UBS não instalados (rode: biblia dados baixar)")
	// ErrUnknownDictionary is returned for an id not in Dictionaries.
	ErrUnknownDictionary = errors.New("dicionário desconhecido")
)

// Dictionary is one downloadable dictionary file.
type Dictionary struct {
	// ID is what the reader writes in the config file.
	ID string
	// Greek is set for the New Testament dictionary.
	Greek bool
	// Lang is the language of glosses and definitions.
	Lang string
	// Path is the file in the repository.
	Path string
}

// Name is the dictionary's title as UBS publishes it.
func (d Dictionary) Name() string {
	if d.Greek {
		return "UBS Dictionary of the Greek New Testament"
	}

	return "UBS Dictionary of Biblical Hebrew"
}

// Label is the short form shown in the program's output.
func (d Dictionary) Label() string {
	if d.Greek {
		return "UBS grego (" + d.Lang + ")"
	}

	return "UBS hebraico (" + d.Lang + ")"
}

// Dictionaries lists the dictionaries the program can use. The Chinese
// editions also exist upstream; they are left out only because nobody
// has asked for them.
var Dictionaries = []Dictionary{
	{ID: "ubs-hebraico-pt", Greek: false, Lang: "pt", Path: "dictionaries/hebrew/JSON/UBSHebrewDic-v0.9.4-pt.JSON"},
	{ID: "ubs-hebraico-es", Greek: false, Lang: "es", Path: "dictionaries/hebrew/JSON/UBSHebrewDic-v0.9.4-es.JSON"},
	{ID: "ubs-hebraico-en", Greek: false, Lang: "en", Path: "dictionaries/hebrew/JSON/UBSHebrewDic-v0.9.4-en.JSON"},
	{ID: "ubs-hebraico-fr", Greek: false, Lang: "fr", Path: "dictionaries/hebrew/JSON/UBSHebrewDic-v0.9.4-fr.JSON"},
	{ID: "ubs-grego-es", Greek: true, Lang: "es", Path: "dictionaries/greek/JSON/UBSGreekNTDic-v1.0-es.JSON"},
	{ID: "ubs-grego-en", Greek: true, Lang: "en", Path: "dictionaries/greek/JSON/UBSGreekNTDic-v1.1-en.JSON"},
	{ID: "ubs-grego-fr", Greek: true, Lang: "fr", Path: "dictionaries/greek/JSON/UBSGreekNTDic-v1.01-fr.JSON"},
}

// Find returns the dictionary with the given id.
func Find(id string) (Dictionary, bool) {
	for _, d := range Dictionaries {
		if d.ID == id {
			return d, true
		}
	}

	return Dictionary{ID: "", Greek: false, Lang: "", Path: ""}, false
}

// nearby lists, for a reading language, the languages to fall back on
// before English when a dictionary does not exist in it: there is no
// Portuguese Greek dictionary, and Spanish is the closest.
var nearby = map[string][]string{
	"pt": {"es"},
	"es": {"pt"},
}

// Defaults returns the dictionary ids for a reading language: one
// Hebrew and one Greek dictionary, in that language or the closest one
// available.
func Defaults(lang string) []string {
	langs := append(append([]string{lang}, nearby[lang]...), "en")

	var out []string

	for _, greek := range []bool{false, true} {
		for _, l := range langs {
			found := false

			for _, d := range Dictionaries {
				if d.Greek == greek && d.Lang == l {
					out = append(out, d.ID)
					found = true

					break
				}
			}

			if found {
				break
			}
		}
	}

	return out
}

// Store is the indexed UBS database.
type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS sense (
	id         INTEGER PRIMARY KEY,
	dict       TEXT    NOT NULL,
	lemma      TEXT    NOT NULL,
	glosses    TEXT    NOT NULL,
	definition TEXT    NOT NULL,
	domain     TEXT    NOT NULL,
	ai         INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sense_strong (
	strong TEXT    NOT NULL,
	sense  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS sense_strong_idx ON sense_strong (strong);

CREATE TABLE IF NOT EXISTS sense_ref (
	sense   INTEGER NOT NULL,
	book    INTEGER NOT NULL,
	chapter INTEGER NOT NULL,
	verse   INTEGER NOT NULL,
	PRIMARY KEY (sense, book, chapter, verse)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS parallel (
	grp     INTEGER NOT NULL,
	pos     INTEGER NOT NULL,
	book    TEXT    NOT NULL,
	chapter INTEGER NOT NULL,
	spans   TEXT    NOT NULL,
	greek   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS parallel_grp ON parallel (grp, pos);

CREATE TABLE IF NOT EXISTS parallel_verse (
	book    TEXT    NOT NULL,
	chapter INTEGER NOT NULL,
	verse   INTEGER NOT NULL,
	grp     INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS parallel_verse_idx ON parallel_verse (book, chapter, verse);

CREATE TABLE IF NOT EXISTS dict (
	id     TEXT PRIMARY KEY,
	senses INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// glossSep separates glosses in one column; no gloss contains it.
const glossSep = "\x1f"

// Open opens an indexed database; it fails with ErrNotInstalled when the
// index does not exist yet.
func Open(dbPath string) (*Store, error) {
	_, err := os.Stat(dbPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotInstalled
	}

	db, err := sql.Open("sqlite", dbPath+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("ubs: opening %s: %w", dbPath, err)
	}

	return &Store{db: db}, nil
}

// Close releases the database.
func (s *Store) Close() error {
	err := s.db.Close()
	if err != nil {
		return fmt.Errorf("ubs: closing: %w", err)
	}

	return nil
}

// Installed returns the ids of the indexed dictionaries and their sense
// counts.
func (s *Store) Installed(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, senses FROM dict`)
	if err != nil {
		return nil, fmt.Errorf("ubs: reading dictionaries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]int{}

	for rows.Next() {
		var (
			id string
			n  int
		)

		err := rows.Scan(&id, &n)
		if err != nil {
			return nil, fmt.Errorf("ubs: reading dictionaries: %w", err)
		}

		out[id] = n
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("ubs: reading dictionaries: %w", err)
	}

	return out, nil
}

// Senses returns the senses a dictionary gives a Strong number ("H2617",
// or STEPBible's extended "H2603A": only the four digits count). When at
// is not nil, each sense's InVerse tells whether that verse (in the
// original numbering) uses the word in that sense.
func (s *Store) Senses(ctx context.Context, dict, strong string, at *Verse) ([]Sense, error) {
	codes := NormalizeCodes([]string{strong})
	if len(codes) == 0 {
		return nil, nil
	}

	var where Verse
	if at != nil {
		where = *at
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT s.lemma, s.glosses, s.definition, s.domain, s.ai,
		        EXISTS (SELECT 1 FROM sense_ref r
		                WHERE r.sense = s.id AND r.book = ? AND r.chapter = ? AND r.verse = ?)
		 FROM sense_strong k JOIN sense s ON s.id = k.sense
		 WHERE k.strong = ? AND s.dict = ?
		 ORDER BY s.id`,
		where.Book, where.Chapter, where.Verse, codes[0], dict,
	)
	if err != nil {
		return nil, fmt.Errorf("ubs: reading senses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Sense

	for rows.Next() {
		sense := Sense{
			Dict: dict, Strongs: codes,
			Lemma: "", Glosses: nil, Definition: "", Domain: "", AI: false, Refs: nil, InVerse: false,
		}

		var glosses string

		err := rows.Scan(&sense.Lemma, &glosses, &sense.Definition, &sense.Domain, &sense.AI, &sense.InVerse)
		if err != nil {
			return nil, fmt.Errorf("ubs: reading sense: %w", err)
		}

		sense.Glosses = splitGlosses(glosses)
		out = append(out, sense)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("ubs: reading senses: %w", err)
	}

	return out, nil
}

// SearchGloss returns the Strong numbers whose senses, in one of the
// given dictionaries, have a gloss containing text, in dictionary order
// and without repetition.
func (s *Store) SearchGloss(ctx context.Context, dicts []string, text string, limit int) ([]string, error) {
	var out []string

	seen := map[string]bool{}

	for _, dict := range dicts {
		found, err := s.searchGloss(ctx, dict, text)
		if err != nil {
			return nil, err
		}

		for _, strong := range found {
			if !seen[strong] && len(out) < limit {
				seen[strong] = true
				out = append(out, strong)
			}
		}
	}

	return out, nil
}

// Parallels returns the passages parallel to a verse (USFM book code,
// chapter and verse in the original numbering), without repetition and
// without the passages that contain the verse itself.
func (s *Store) Parallels(ctx context.Context, book string, chapter, verse int) ([]Passage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT p.book, p.chapter, p.spans, p.greek
		 FROM parallel p
		 WHERE p.grp IN (SELECT grp FROM parallel_verse WHERE book = ? AND chapter = ? AND verse = ?)
		 ORDER BY p.grp, p.pos`,
		strings.ToUpper(book), chapter, verse,
	)
	if err != nil {
		return nil, fmt.Errorf("ubs: reading parallels: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Passage

	seen := map[string]bool{}

	for rows.Next() {
		var (
			p     Passage
			spans string
		)

		err := rows.Scan(&p.Book, &p.Chapter, &spans, &p.Greek)
		if err != nil {
			return nil, fmt.Errorf("ubs: reading parallel: %w", err)
		}

		key := p.Book + " " + strconv.Itoa(p.Chapter) + ":" + spans
		if seen[key] {
			continue
		}

		seen[key] = true

		p.Spans = parseSpans(spans)
		if strings.EqualFold(p.Book, book) && p.Contains(chapter, verse) {
			continue
		}

		out = append(out, p)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("ubs: reading parallels: %w", err)
	}

	return out, nil
}

// Meta returns the provenance recorded at index time.
func (s *Store) Meta(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM meta`)
	if err != nil {
		return nil, fmt.Errorf("ubs: reading meta: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]string{}

	for rows.Next() {
		var key, value string

		err := rows.Scan(&key, &value)
		if err != nil {
			return nil, fmt.Errorf("ubs: reading meta: %w", err)
		}

		out[key] = value
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("ubs: reading meta: %w", err)
	}

	return out, nil
}

func (s *Store) searchGloss(ctx context.Context, dict, text string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT k.strong FROM sense s JOIN sense_strong k ON k.sense = s.id
		 WHERE s.dict = ? AND s.glosses LIKE ? ORDER BY k.strong`,
		dict, "%"+text+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("ubs: searching glosses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []string

	for rows.Next() {
		var strong string

		err := rows.Scan(&strong)
		if err != nil {
			return nil, fmt.Errorf("ubs: searching glosses: %w", err)
		}

		out = append(out, strong)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("ubs: searching glosses: %w", err)
	}

	return out, nil
}

func splitGlosses(s string) []string {
	if s == "" {
		return nil
	}

	return strings.Split(s, glossSep)
}

// formatSpans writes spans the way the database file does: "1-3,5".
func formatSpans(spans []Span) string {
	parts := make([]string, 0, len(spans))

	for _, sp := range spans {
		if sp.From == sp.To {
			parts = append(parts, strconv.Itoa(sp.From))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", sp.From, sp.To))
		}
	}

	return strings.Join(parts, ",")
}

func parseSpans(s string) []Span {
	var out []Span

	for part := range strings.SplitSeq(s, ",") {
		from, to, isRange := strings.Cut(part, "-")
		if !isRange {
			to = from
		}

		f, err1 := strconv.Atoi(from)
		t, err2 := strconv.Atoi(to)

		if err1 == nil && err2 == nil {
			out = append(out, Span{From: f, To: t})
		}
	}

	return out
}

// Installer downloads and indexes the data.
type Installer struct {
	HTTP *http.Client
	// Log receives progress lines.
	Log io.Writer
}

// LatestCommit asks GitHub for the repository's current commit, so the
// download is pinned to one known revision.
func (in *Installer) LatestCommit(ctx context.Context) (string, error) {
	commit, err := download.LatestCommit(ctx, in.HTTP, Repo)
	if err != nil {
		return "", fmt.Errorf("ubs: %w", err)
	}

	return commit, nil
}

// Download fetches the given dictionaries and the parallel passages at a
// commit into rawDir.
func (in *Installer) Download(ctx context.Context, commit, rawDir string, dicts []Dictionary) error {
	const dirPerm = 0o750

	err := os.MkdirAll(rawDir, dirPerm)
	if err != nil {
		return fmt.Errorf("ubs: creating %s: %w", rawDir, err)
	}

	paths := []string{ParallelsPath}
	for _, d := range dicts {
		paths = append(paths, d.Path)
	}

	for _, p := range paths {
		fmt.Fprintf(in.Log, "baixando %s\n", path.Base(p))

		err := download.Fetch(ctx, in.HTTP, download.RawURL(Repo, commit, p),
			filepath.Join(rawDir, path.Base(p)))
		if err != nil {
			return fmt.Errorf("ubs: %w", err)
		}
	}

	return nil
}

// Index builds the database at dbPath from the files in rawDir. It writes
// to a temporary file and swaps it in at the end, so a failed run never
// leaves a half-built index behind.
func (in *Installer) Index(ctx context.Context, rawDir, dbPath, commit string, dicts []Dictionary) error {
	tmp := dbPath + ".part"
	_ = os.Remove(tmp)

	db, err := sql.Open("sqlite", tmp+"?_pragma=journal_mode(OFF)&_pragma=synchronous(OFF)")
	if err != nil {
		return fmt.Errorf("ubs: creating index: %w", err)
	}

	err = in.fill(ctx, db, rawDir, commit, dicts)
	if err != nil {
		_ = db.Close()
		_ = os.Remove(tmp)

		return err
	}

	err = db.Close()
	if err != nil {
		return fmt.Errorf("ubs: closing index: %w", err)
	}

	err = os.Rename(tmp, dbPath)
	if err != nil {
		return fmt.Errorf("ubs: installing index: %w", err)
	}

	return nil
}

func (in *Installer) fill(ctx context.Context, db *sql.DB, rawDir, commit string, dicts []Dictionary) error {
	_, err := db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("ubs: creating schema: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ubs: starting transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, d := range dicts {
		n, err := indexDictionary(ctx, tx, filepath.Join(rawDir, path.Base(d.Path)), d.ID)
		if err != nil {
			return err
		}

		fmt.Fprintf(in.Log, "indexado %s: %d sentidos\n", d.ID, n)
	}

	n, err := indexParallels(ctx, tx, filepath.Join(rawDir, path.Base(ParallelsPath)))
	if err != nil {
		return err
	}

	fmt.Fprintf(in.Log, "indexadas as passagens paralelas: %d grupos\n", n)

	ids := make([]string, 0, len(dicts))
	for _, d := range dicts {
		ids = append(ids, d.ID)
	}

	slices.Sort(ids)

	meta := map[string]string{
		"source":       "https://github.com/" + Repo,
		"commit":       commit,
		"indexed_at":   time.Now().Format(time.RFC3339),
		"dictionaries": strings.Join(ids, ","),
	}

	for k, v := range meta {
		_, err := tx.ExecContext(ctx, `INSERT INTO meta (key, value) VALUES (?, ?)`, k, v)
		if err != nil {
			return fmt.Errorf("ubs: writing meta: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ubs: committing index: %w", err)
	}

	return nil
}

func indexDictionary(ctx context.Context, tx *sql.Tx, name, dict string) (int, error) {
	file, err := os.Open(name) //nolint:gosec // name is under the program's own data directory
	if err != nil {
		return 0, fmt.Errorf("ubs: opening %s: %w", name, err)
	}
	defer func() { _ = file.Close() }()

	n := 0

	err = ParseDictionary(file, dict, func(s Sense) error {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO sense (dict, lemma, glosses, definition, domain, ai) VALUES (?, ?, ?, ?, ?, ?)`,
			dict, s.Lemma, strings.Join(s.Glosses, glossSep), s.Definition, s.Domain, s.AI)
		if err != nil {
			return fmt.Errorf("ubs: inserting sense: %w", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("ubs: inserting sense: %w", err)
		}

		for _, strong := range s.Strongs {
			_, err := tx.ExecContext(ctx, `INSERT INTO sense_strong VALUES (?, ?)`, strong, id)
			if err != nil {
				return fmt.Errorf("ubs: inserting sense: %w", err)
			}
		}

		for _, v := range s.Refs {
			_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO sense_ref VALUES (?, ?, ?, ?)`,
				id, v.Book, v.Chapter, v.Verse)
			if err != nil {
				return fmt.Errorf("ubs: inserting reference: %w", err)
			}
		}

		n++

		return nil
	})
	if err != nil {
		return n, err
	}

	_, err = tx.ExecContext(ctx, `INSERT OR REPLACE INTO dict VALUES (?, ?)`, dict, n)
	if err != nil {
		return n, fmt.Errorf("ubs: recording %s: %w", dict, err)
	}

	return n, nil
}

func indexParallels(ctx context.Context, tx *sql.Tx, name string) (int, error) {
	file, err := os.Open(name) //nolint:gosec // name is under the program's own data directory
	if err != nil {
		return 0, fmt.Errorf("ubs: opening %s: %w", name, err)
	}
	defer func() { _ = file.Close() }()

	groups, err := ParseParallels(file)
	if err != nil {
		return 0, err
	}

	for grp, members := range groups {
		for pos, p := range members {
			_, err := tx.ExecContext(ctx, `INSERT INTO parallel VALUES (?, ?, ?, ?, ?, ?)`,
				grp, pos, p.Book, p.Chapter, formatSpans(p.Spans), p.Greek)
			if err != nil {
				return 0, fmt.Errorf("ubs: inserting parallel: %w", err)
			}

			for _, sp := range p.Spans {
				for v := sp.From; v <= sp.To; v++ {
					_, err := tx.ExecContext(ctx, `INSERT INTO parallel_verse VALUES (?, ?, ?, ?)`,
						p.Book, p.Chapter, v, grp)
					if err != nil {
						return 0, fmt.Errorf("ubs: inserting parallel: %w", err)
					}
				}
			}
		}
	}

	return len(groups), nil
}
