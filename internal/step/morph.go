package step

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Outdated reports whether the index was built by an earlier version of
// the program: without the morphology legends (grammar stays as raw
// codes) or without the lexicon's relations between entries (a related
// entry shows the other entry's number).
func (s *Store) Outdated() bool {
	return !s.hasMorph || !s.hasRelation
}

// Grammar returns the legend's expansion of a word's grammar code, one
// item per segment: a Greek code is one segment ("V-AAI-3S"), a Hebrew
// code one per prefix, word and suffix ("HR/Ncfsa" → the preposition,
// then the noun). It returns nil when the index predates the legends or
// any segment is missing from them, so a code is never half-described.
func (s *Store) Grammar(ctx context.Context, w Word) ([]string, error) {
	if !s.hasMorph || w.Grammar == "" {
		return nil, nil
	}

	var keys []string

	if strings.HasPrefix(w.Strong, "G") {
		code, _, _ := strings.Cut(w.Grammar, " + ")
		keys = []string{"G:" + strings.TrimSpace(code)}
	} else {
		lang := w.Grammar[:1]
		for seg := range strings.SplitSeq(w.Grammar[1:], "/") {
			keys = append(keys, "H:"+lang+seg)
		}
	}

	out := make([]string, 0, len(keys))

	for _, key := range keys {
		var attrs string

		err := s.db.QueryRowContext(ctx, `SELECT attrs FROM morph WHERE code = ?`, key).Scan(&attrs)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		if err != nil {
			return nil, fmt.Errorf("step: reading morphology: %w", err)
		}

		out = append(out, attrs)
	}

	return out, nil
}

// OriginalVerse converts a verse in the translations' numbering (English,
// which Almeida shares) into the numbering of the Hebrew text or Greek
// edition, which the UBS data uses: Jl 2.28 → 3.1, Sl 51.1 → 51.3. A
// verse numbered the same way in both comes back unchanged.
func (s *Store) OriginalVerse(ctx context.Context, book string, chapter, verse int) (int, int, error) {
	var alt string

	err := s.db.QueryRowContext(ctx,
		`SELECT alt_ref FROM word WHERE book = ? AND chapter = ? AND verse = ? AND alt_ref <> '' LIMIT 1`,
		book, chapter, verse,
	).Scan(&alt)
	if errors.Is(err, sql.ErrNoRows) {
		return chapter, verse, nil
	}

	if err != nil {
		return 0, 0, fmt.Errorf("step: reading versification: %w", err)
	}

	c, v, ok := splitAltRef(alt)
	if !ok {
		return chapter, verse, nil
	}

	return c, v, nil
}

// TranslationVerse is the inverse of OriginalVerse: Jl 3.1 (Hebrew) →
// 2.28. A Psalm title, which the original numbers as verse 1 or 2, comes
// back as verse 0.
func (s *Store) TranslationVerse(ctx context.Context, book string, chapter, verse int) (int, int, error) {
	var transChapter, transVerse int

	err := s.db.QueryRowContext(ctx,
		`SELECT chapter, verse FROM word WHERE book = ? AND alt_ref = ? ORDER BY chapter, verse LIMIT 1`,
		book, fmt.Sprintf("%d.%d", chapter, verse),
	).Scan(&transChapter, &transVerse)
	if errors.Is(err, sql.ErrNoRows) {
		return chapter, verse, nil
	}

	if err != nil {
		return 0, 0, fmt.Errorf("step: reading versification: %w", err)
	}

	return transChapter, transVerse, nil
}

func splitAltRef(alt string) (int, int, bool) {
	cs, vs, found := strings.Cut(alt, ".")
	if !found {
		return 0, 0, false
	}

	c, err1 := strconv.Atoi(cs)
	v, err2 := strconv.Atoi(vs)

	return c, v, err1 == nil && err2 == nil
}
