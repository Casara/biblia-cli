package step

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	_ "modernc.org/sqlite" // database/sql driver "sqlite"

	"github.com/casara/biblia-cli/internal/download"
)

// Sentinel errors, matchable with errors.Is.
var (
	// ErrNotInstalled is returned when the STEPBible data has not been
	// downloaded yet.
	ErrNotInstalled = errors.New("dados da STEPBible não instalados (rode: biblia dados baixar)")
	// ErrBadStrong is returned for text that is not a Strong number.
	ErrBadStrong = errors.New("não é um número de Strong (ex.: H5162, G0025)")
)

// Repo is the STEPBible data repository on GitHub.
const Repo = "STEPBible/STEPBible-Data"

// File is one upstream file: its path in the repository and the kind of
// rows it holds.
type File struct {
	Path string
	Kind Kind
}

// Kind tells the indexer how to read a file.
type Kind int

// File kinds.
const (
	Hebrew Kind = iota
	Greek
	Lexicon
	// GreekLegend and HebrewLegend are the morphology legends (TEGMC,
	// TEHMC): each grammar code of the tagged texts expanded into
	// "Function=Verb; Tense=Aorist; ...".
	GreekLegend
	HebrewLegend
)

const amalgamated = "Translators Amalgamated OT+NT/"

// Files lists everything the program needs from the repository.
var Files = []File{
	{amalgamated + "TAHOT Gen-Deu - Translators Amalgamated Hebrew OT - STEPBible.org CC BY.txt", Hebrew},
	{amalgamated + "TAHOT Jos-Est - Translators Amalgamated Hebrew OT - STEPBible.org CC BY.txt", Hebrew},
	{amalgamated + "TAHOT Job-Sng - Translators Amalgamated Hebrew OT - STEPBible.org CC BY.txt", Hebrew},
	{amalgamated + "TAHOT Isa-Mal - Translators Amalgamated Hebrew OT - STEPBible.org CC BY.txt", Hebrew},
	{amalgamated + "TAGNT Mat-Jhn - Translators Amalgamated Greek NT - STEPBible.org CC-BY.txt", Greek},
	{amalgamated + "TAGNT Act-Rev - Translators Amalgamated Greek NT - STEPBible.org CC-BY.txt", Greek},
	{"Lexicons/TBESH - Translators Brief lexicon of Extended Strongs for Hebrew - STEPBible.org CC BY.txt", Lexicon},
	{"Lexicons/TBESG - Translators Brief lexicon of Extended Strongs for Greek - STEPBible.org CC BY.txt", Lexicon},
	{"Morphology codes/TEGMC - Translators Expansion of Greek Morphhology Codes - STEPBible.org CC BY.txt", GreekLegend},
	{"Morphology codes/TEHMC - Translators Expansion of Hebrew Morphology Codes - STEPBible.org CC BY.txt", HebrewLegend},
}

// Store is the indexed STEPBible database.
type Store struct {
	db *sql.DB
	// hasMorph is false for an index built before the morphology legends
	// were part of the download; grammar codes then stay undecoded.
	hasMorph bool
	// hasRelation is false for an index built before lexicon entries
	// kept their relation to other entries; such an index filed a
	// related entry under the other entry's number.
	hasRelation bool
}

const schema = `
CREATE TABLE IF NOT EXISTS word (
	book     TEXT    NOT NULL,
	chapter  INTEGER NOT NULL,
	verse    INTEGER NOT NULL,
	pos      INTEGER NOT NULL,
	alt_ref  TEXT    NOT NULL,
	type     TEXT    NOT NULL,
	original TEXT    NOT NULL,
	translit TEXT    NOT NULL,
	english  TEXT    NOT NULL,
	strong   TEXT    NOT NULL,
	grammar  TEXT    NOT NULL,
	gloss    TEXT    NOT NULL,
	editions TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS word_ref ON word (book, chapter, verse, pos);

CREATE TABLE IF NOT EXISTS lexicon (
	strong     TEXT NOT NULL,
	estrong    TEXT NOT NULL,
	relation   TEXT NOT NULL,
	target     TEXT NOT NULL,
	original   TEXT NOT NULL,
	translit   TEXT NOT NULL,
	morph      TEXT NOT NULL,
	gloss      TEXT NOT NULL,
	definition TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS lexicon_strong ON lexicon (strong);
CREATE INDEX IF NOT EXISTS lexicon_estrong ON lexicon (estrong);
CREATE INDEX IF NOT EXISTS lexicon_target ON lexicon (target);

CREATE INDEX IF NOT EXISTS word_alt ON word (book, alt_ref);

CREATE TABLE IF NOT EXISTS morph (
	code  TEXT PRIMARY KEY,
	attrs TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// Open opens an indexed database; it fails with ErrNotInstalled when the
// index does not exist yet.
func Open(ctx context.Context, path string) (*Store, error) {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotInstalled
	}

	db, err := sql.Open("sqlite", path+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("step: opening %s: %w", path, err)
	}

	var n int

	err = db.QueryRowContext(ctx,
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'morph'`).Scan(&n)
	if err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("step: opening %s: %w", path, err)
	}

	var relation int

	err = db.QueryRowContext(ctx,
		`SELECT count(*) FROM pragma_table_info('lexicon') WHERE name = 'relation'`).Scan(&relation)
	if err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("step: opening %s: %w", path, err)
	}

	return &Store{db: db, hasMorph: n > 0, hasRelation: relation > 0}, nil
}

// Close releases the database.
func (s *Store) Close() error {
	err := s.db.Close()
	if err != nil {
		return fmt.Errorf("step: closing: %w", err)
	}

	return nil
}

// Words returns the words of a verse in order.
func (s *Store) Words(ctx context.Context, book string, chapter, verse int) ([]Word, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT book, chapter, verse, pos, alt_ref, type, original, translit,
		        english, strong, grammar, gloss, editions
		 FROM word WHERE book = ? AND chapter = ? AND verse = ? ORDER BY pos, rowid`,
		book, chapter, verse,
	)
	if err != nil {
		return nil, fmt.Errorf("step: reading words: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Word

	for rows.Next() {
		var w Word

		err := rows.Scan(&w.Book, &w.Chapter, &w.Verse, &w.Pos, &w.AltRef, &w.Type,
			&w.Original, &w.Translit, &w.English, &w.Strong, &w.Grammar, &w.Gloss, &w.Editions)
		if err != nil {
			return nil, fmt.Errorf("step: reading word: %w", err)
		}

		out = append(out, w)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("step: reading words: %w", err)
	}

	return out, nil
}

// strongRe splits "H7451b" or "G25" into prefix, number and suffix.
var strongRe = regexp.MustCompile(`^([HGhg])0*(\d{1,4})([A-Za-z]?)$`)

// NormalizeStrong pads a Strong number to STEPBible's four digits and
// upper-cases the prefix: "g25" -> "G0025", "H7451b" -> "H7451b".
func NormalizeStrong(s string) (string, bool) {
	m := strongRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return "", false
	}

	num, err := strconv.Atoi(m[2])
	if err != nil {
		return "", false
	}

	return fmt.Sprintf("%s%04d%s", strings.ToUpper(m[1]), num, m[3]), true
}

// Lookup returns the lexicon entries for a Strong number, trying in
// turn: the extended number the tagged texts use ("H2603A", with the
// entries the lexicon ties to it, such as H2589 "a form of" H2603A), the
// number exactly as the lexicon files it ("H7451b", "H5162"), and finally
// every entry whose number starts with the plain four digits ("H2603").
func (s *Store) Lookup(ctx context.Context, strong string) ([]Entry, error) {
	num, ok := NormalizeStrong(strong)
	if !ok {
		return nil, fmt.Errorf("step: %q %w", strong, ErrBadStrong)
	}

	extended := `WHERE upper(estrong) = upper(?) ORDER BY strong`
	if s.hasRelation {
		// The entry itself, then those the lexicon ties to it ("H2589 =
		// a form of H2603A").
		extended = `WHERE upper(estrong) = upper(?1) OR upper(target) = upper(?1)
		            ORDER BY upper(estrong) <> upper(?1), strong`
	}

	for _, where := range []string{extended, `WHERE upper(strong) = upper(?) ORDER BY estrong`} {
		entries, err := s.query(ctx, where, num)
		if err != nil || len(entries) > 0 {
			return entries, err
		}
	}

	const plainLen = 5 // "H2603"

	return s.query(ctx,
		`WHERE substr(strong, 1, 5) = ? OR substr(estrong, 1, 5) = ? ORDER BY estrong`,
		num[:plainLen], num[:plainLen])
}

// SearchTranslit finds entries by transliteration, ignoring case, dots
// and diacritics ("agape" finds "agapē", "chanan" finds "cha.nan"):
// exact matches first, otherwise up to limit substring matches.
func (s *Store) SearchTranslit(ctx context.Context, tr string, limit int) ([]Entry, error) {
	all, err := s.query(ctx, `ORDER BY estrong`)
	if err != nil {
		return nil, err
	}

	key := foldTranslit(tr)

	var exact, partial []Entry

	for _, e := range all {
		folded := foldTranslit(e.Translit)

		switch {
		case folded == key:
			exact = append(exact, e)
		case strings.Contains(folded, key) && len(partial) < limit:
			partial = append(partial, e)
		}
	}

	if len(exact) > 0 {
		return exact, nil
	}

	return partial, nil
}

// foldTranslit lower-cases and strips dots, apostrophes and diacritics.
func foldTranslit(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

	plain, _, err := transform.String(t, strings.ToLower(s))
	if err != nil {
		plain = strings.ToLower(s)
	}

	return strings.Map(func(r rune) rune {
		if r == '.' || r == '\'' || r == '’' || r == 'ʼ' || r == '-' {
			return -1
		}

		return r
	}, plain)
}

// SearchGloss finds entries whose (English) gloss contains text.
func (s *Store) SearchGloss(ctx context.Context, text string, limit int) ([]Entry, error) {
	return s.query(ctx, `WHERE gloss LIKE ? LIMIT ?`, "%"+text+"%", limit)
}

// Meta returns the provenance recorded at index time.
func (s *Store) Meta(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM meta`)
	if err != nil {
		return nil, fmt.Errorf("step: reading meta: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]string{}

	for rows.Next() {
		var key, value string

		err := rows.Scan(&key, &value)
		if err != nil {
			return nil, fmt.Errorf("step: reading meta: %w", err)
		}

		out[key] = value
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("step: reading meta: %w", err)
	}

	return out, nil
}

func (s *Store) query(ctx context.Context, where string, args ...any) ([]Entry, error) {
	columns := `SELECT strong, estrong, relation, target, original, translit, morph, gloss, definition FROM lexicon `
	if !s.hasRelation {
		columns = `SELECT strong, estrong, '', '', original, translit, morph, gloss, definition FROM lexicon `
	}

	// where is always one of this file's constant clauses.
	rows, err := s.db.QueryContext(ctx, columns+where, args...)
	if err != nil {
		return nil, fmt.Errorf("step: reading lexicon: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Entry

	for rows.Next() {
		var e Entry

		err := rows.Scan(&e.Strong, &e.EStrong, &e.Relation, &e.Target,
			&e.Original, &e.Translit, &e.Morph, &e.Gloss, &e.Definition)
		if err != nil {
			return nil, fmt.Errorf("step: reading entry: %w", err)
		}

		out = append(out, e)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("step: reading lexicon: %w", err)
	}

	return out, nil
}

// Installer downloads and indexes the data.
type Installer struct {
	HTTP *http.Client
	// Log receives progress lines.
	Log io.Writer
}

// LatestCommit asks GitHub for the current commit of the repository, so
// the download is pinned to one known revision.
func (in *Installer) LatestCommit(ctx context.Context) (string, error) {
	commit, err := download.LatestCommit(ctx, in.HTTP, Repo)
	if err != nil {
		return "", fmt.Errorf("step: %w", err)
	}

	return commit, nil
}

// Download fetches every file at the given commit into rawDir.
func (in *Installer) Download(ctx context.Context, commit, rawDir string) error {
	const dirPerm = 0o750
	err := os.MkdirAll(rawDir, dirPerm)
	if err != nil {
		return fmt.Errorf("step: creating %s: %w", rawDir, err)
	}

	for _, file := range Files {
		fmt.Fprintf(in.Log, "baixando %s\n", filepath.Base(file.Path))

		err := download.Fetch(ctx, in.HTTP, download.RawURL(Repo, commit, file.Path),
			filepath.Join(rawDir, filepath.Base(file.Path)))
		if err != nil {
			return fmt.Errorf("step: %w", err)
		}
	}

	return nil
}

// Index builds the database at dbPath from the files in rawDir. It
// writes to a temporary file and swaps it in at the end, so a failed
// run never leaves a half-built index behind.
func (in *Installer) Index(ctx context.Context, rawDir, dbPath, commit string) error {
	tmp := dbPath + ".part"
	_ = os.Remove(tmp)

	db, err := sql.Open("sqlite", tmp+"?_pragma=journal_mode(OFF)&_pragma=synchronous(OFF)")
	if err != nil {
		return fmt.Errorf("step: creating index: %w", err)
	}

	err = in.fill(ctx, db, rawDir, commit)
	if err != nil {
		_ = db.Close()
		_ = os.Remove(tmp)

		return err
	}

	err = db.Close()
	if err != nil {
		return fmt.Errorf("step: closing index: %w", err)
	}

	err = os.Rename(tmp, dbPath)
	if err != nil {
		return fmt.Errorf("step: installing index: %w", err)
	}

	return nil
}

func (in *Installer) fill(ctx context.Context, db *sql.DB, rawDir, commit string) error {
	_, err := db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("step: creating schema: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("step: starting transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, file := range Files {
		n, err := indexFile(ctx, tx, filepath.Join(rawDir, filepath.Base(file.Path)), file.Kind)
		if err != nil {
			return err
		}

		fmt.Fprintf(in.Log, "indexado %s: %d linhas\n", filepath.Base(file.Path), n)
	}

	meta := map[string]string{
		"source":     "https://github.com/" + Repo,
		"commit":     commit,
		"indexed_at": time.Now().Format(time.RFC3339),
	}

	for k, v := range meta {
		_, err := tx.ExecContext(ctx, `INSERT INTO meta (key, value) VALUES (?, ?)`, k, v)
		if err != nil {
			return fmt.Errorf("step: writing meta: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("step: committing index: %w", err)
	}

	return nil
}

func indexFile(ctx context.Context, tx *sql.Tx, path string, kind Kind) (int, error) {
	file, err := os.Open(path) //nolint:gosec // path is under the program's own data directory
	if err != nil {
		return 0, fmt.Errorf("step: opening %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	ins, err := newInserter(ctx, tx)
	if err != nil {
		return 0, err
	}
	defer ins.close()

	const maxLine = 1 << 20

	sc := bufio.NewScanner(file)
	sc.Buffer(make([]byte, 0, maxLine), maxLine)

	n := 0

	for sc.Scan() {
		ok, err := ins.row(ctx, kind, strings.TrimPrefix(sc.Text(), "\uFEFF"))
		if err != nil {
			return n, fmt.Errorf("step: indexing %s: %w", filepath.Base(path), err)
		}

		if ok {
			n++
		}
	}

	err = sc.Err()
	if err != nil {
		return n, fmt.Errorf("step: reading %s: %w", path, err)
	}

	return n, nil
}

// inserter holds the prepared statements of one indexing transaction.
type inserter struct {
	word, lexicon, morph *sql.Stmt
}

func newInserter(ctx context.Context, tx *sql.Tx) (*inserter, error) {
	word, err := tx.PrepareContext(ctx,
		`INSERT INTO word VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, fmt.Errorf("step: preparing: %w", err)
	}

	lexicon, err := tx.PrepareContext(ctx,
		`INSERT INTO lexicon VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = word.Close() //nolint:sqlclosecheck // the statements outlive this function; inserter.close closes them

		return nil, fmt.Errorf("step: preparing: %w", err)
	}

	morph, err := tx.PrepareContext(ctx, `INSERT OR REPLACE INTO morph VALUES (?, ?)`)
	if err != nil {
		_ = word.Close()
		_ = lexicon.Close() //nolint:sqlclosecheck // see above

		return nil, fmt.Errorf("step: preparing: %w", err)
	}

	return &inserter{word: word, lexicon: lexicon, morph: morph}, nil
}

// row inserts one line of a file of the given kind; ok is false for a
// line that holds no data (headers, notes, blank lines).
func (in *inserter) row(ctx context.Context, kind Kind, line string) (bool, error) {
	var err error

	switch kind {
	case Hebrew, Greek:
		parse := ParseHebrewRow
		if kind == Greek {
			parse = ParseGreekRow
		}

		w, ok := parse(line)
		if !ok {
			return false, nil
		}

		_, err = in.word.ExecContext(ctx, w.Book, w.Chapter, w.Verse, w.Pos, w.AltRef,
			w.Type, w.Original, w.Translit, w.English, w.Strong, w.Grammar, w.Gloss, w.Editions)
	case Lexicon:
		e, ok := ParseLexiconRow(line)
		if !ok {
			return false, nil
		}

		_, err = in.lexicon.ExecContext(ctx, e.Strong, e.EStrong, e.Relation, e.Target,
			e.Original, e.Translit, e.Morph, e.Gloss, e.Definition)
	case GreekLegend, HebrewLegend:
		code, attrs, ok := ParseLegendRow(line, kind == GreekLegend)
		if !ok {
			return false, nil
		}

		_, err = in.morph.ExecContext(ctx, code, attrs)
	}

	if err != nil {
		return false, fmt.Errorf("step: inserting: %w", err)
	}

	return true, nil
}

func (in *inserter) close() {
	_ = in.word.Close()
	_ = in.lexicon.Close()
	_ = in.morph.Close()
}
