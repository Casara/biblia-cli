// Package lookup answers "give me this reference in this translation",
// combining the local cache with at most one call to the Bíblia Online
// server per (reference, translation).
package lookup

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"time"

	"github.com/casara/biblia-cli/internal/bibliaonline"
	"github.com/casara/biblia-cli/internal/cache"
	"github.com/casara/biblia-cli/internal/ref"
)

// ErrNotCached is returned offline (or with the site unreachable) for a
// reference the cache does not hold.
var ErrNotCached = errors.New("não está no cache")

// SiteBase is the root of the human-facing site the links point to.
const SiteBase = "https://www.bibliaonline.com.br"

// Verse is one verse of a result. Absent marks a verse the translation
// omits (the number exists in other translations).
type Verse struct {
	Number int
	Text   string
	Absent bool
}

// Result is the answer for one reference in one translation.
type Result struct {
	Ref   ref.Ref
	Bible string
	// Verses holds what could be obtained, in verse order.
	Verses []Verse
	// Missing lists requested verses that could not be obtained: cut by
	// the server's five-verse limit, or unavailable offline.
	Missing []int
	// Partial is true when the text is incomplete: Missing is not empty,
	// or a whole chapter was cut by the server.
	Partial bool
	// URL links to the passage on the site (to the chapter when Partial).
	URL string
	// FetchedAt is when the oldest verse shown was fetched.
	FetchedAt time.Time
	// Stale is true when the server could not be reached and the result
	// came from the cache only.
	Stale bool
}

// Passager is the part of the bibliaonline client lookup needs; tests
// replace it with a fake.
type Passager interface {
	GetPassage(ctx context.Context, reference, bibleID string) (bibliaonline.Passage, error)
}

// Service performs lookups.
type Service struct {
	Cache   *cache.Cache
	Client  Passager
	Offline bool
	Now     func() time.Time
}

// Lookup resolves r in translation bible.
func (s *Service) Lookup(ctx context.Context, r ref.Ref, bible string) (Result, error) {
	if r.WholeChapter() {
		return s.chapter(ctx, r, bible)
	}

	return s.verses(ctx, r, bible)
}

func (s *Service) verses(ctx context.Context, r ref.Ref, bible string) (Result, error) {
	wanted := r.Verses()

	have, err := s.Cache.Get(ctx, bible, r.Book.Step, r.Chapter, wanted)
	if err != nil {
		return Result{}, fmt.Errorf("lookup: %w", err)
	}

	var missing []int

	for _, v := range wanted {
		_, ok := have[v]
		if !ok {
			missing = append(missing, v)
		}
	}

	res := Result{Ref: r, Bible: bible, URL: PassageURL(bible, r)}

	if len(missing) > 0 && !s.Offline {
		fetched, err := s.fetch(ctx, r, bible, missing)

		switch {
		case err == nil:
			maps.Copy(have, fetched)
		case isRemote(err):
			res.Stale = true
		default:
			return Result{}, err
		}
	}

	for _, v := range wanted {
		e, ok := have[v]
		if !ok {
			res.Missing = append(res.Missing, v)

			continue
		}

		res.Verses = append(res.Verses, Verse{Number: v, Text: e.Text, Absent: e.Absent})
		res.FetchedAt = oldest(res.FetchedAt, e.FetchedAt)
	}

	if len(res.Missing) > 0 {
		res.Partial = true
		res.URL = ChapterURL(bible, r)
	}

	if len(res.Verses) == 0 && len(res.Missing) > 0 && (s.Offline || res.Stale) {
		return res, fmt.Errorf("%s (%s): %w", r, bible, ErrNotCached)
	}

	return res, nil
}

// fetch makes the single server call for the missing verses and caches
// what comes back, including verses the translation omits.
func (s *Service) fetch(
	ctx context.Context, r ref.Ref, bible string, missing []int,
) (map[int]cache.Entry, error) {
	q := ref.Ref{Book: r.Book, Chapter: r.Chapter, Spans: ref.Compact(missing)}

	p, err := s.Client.GetPassage(ctx, q.Query(), bible)
	if err != nil {
		return nil, fmt.Errorf("%s (%s): %w", r, bible, err)
	}

	now := s.now()
	out := map[int]cache.Entry{}
	last := 0

	for _, v := range p.Verses {
		out[v.Number] = cache.Entry{Text: v.Text, URL: p.URL, FetchedAt: now}
		last = max(last, v.Number)
	}

	// A requested verse the server skipped does not exist in the
	// translation (omitted, like Mt 17.21 in the NVI, or past the end of
	// the chapter: the server answers "João 3:35-37" with 35 and 36),
	// unless it lies past the cut of a partial answer.
	for _, v := range missing {
		if _, ok := out[v]; ok {
			continue
		}

		if !p.Partial || v < last {
			out[v] = cache.Entry{Absent: true, URL: p.URL, FetchedAt: now}
		}
	}

	err = s.Cache.Put(ctx, bible, r.Book.Step, r.Chapter, out)
	if err != nil {
		return nil, fmt.Errorf("lookup: %w", err)
	}

	return out, nil
}

func (s *Service) chapter(ctx context.Context, r ref.Ref, bible string) (Result, error) {
	res := Result{Ref: r, Bible: bible, URL: ChapterURL(bible, r)}

	if !s.Offline {
		p, err := s.Client.GetPassage(ctx, r.Query(), bible)

		switch {
		case err == nil:
			now := s.now()
			entries := map[int]cache.Entry{}

			for _, v := range p.Verses {
				entries[v.Number] = cache.Entry{Text: v.Text, URL: p.URL, FetchedAt: now}
				res.Verses = append(res.Verses, Verse{Number: v.Number, Text: v.Text})
			}

			err := s.Cache.Put(ctx, bible, r.Book.Step, r.Chapter, entries)
			if err != nil {
				return Result{}, fmt.Errorf("lookup: %w", err)
			}

			res.Partial = p.Partial
			res.FetchedAt = now

			return res, nil
		case isRemote(err):
			res.Stale = true
		default:
			return Result{}, fmt.Errorf("%s (%s): %w", r, bible, err)
		}
	}

	// Offline, or the server is unreachable: whatever the cache has.
	have, err := s.Cache.Get(ctx, bible, r.Book.Step, r.Chapter, nil)
	if err != nil {
		return Result{}, fmt.Errorf("lookup: %w", err)
	}

	if len(have) == 0 {
		return res, fmt.Errorf("%s (%s): %w", r, bible, ErrNotCached)
	}

	keys := make([]int, 0, len(have))
	for v := range have {
		keys = append(keys, v)
	}

	slices.Sort(keys)

	for _, v := range keys {
		e := have[v]
		res.Verses = append(res.Verses, Verse{Number: v, Text: e.Text, Absent: e.Absent})
		res.FetchedAt = oldest(res.FetchedAt, e.FetchedAt)
	}

	// Without the server there is no telling whether the cache holds the
	// whole chapter.
	res.Partial = true

	return res, nil
}

// isRemote tells a network failure (fall back to the cache) from an
// answer the server gave (report it).
func isRemote(err error) bool {
	return !errors.Is(err, bibliaonline.ErrNotFound) &&
		!errors.Is(err, bibliaonline.ErrUnknownBible) &&
		!errors.Is(err, context.Canceled)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}

	return time.Now()
}

func oldest(a, b time.Time) time.Time {
	if a.IsZero() || (!b.IsZero() && b.Before(a)) {
		return b
	}

	return a
}

// PassageURL is the site link for the exact verses of r.
func PassageURL(bible string, r ref.Ref) string {
	if r.WholeChapter() {
		return ChapterURL(bible, r)
	}

	return fmt.Sprintf("%s/%s", ChapterURL(bible, r), urlSpans(r.Spans))
}

// urlSpans prints spans the way the site's canonical URLs do: a range of
// exactly two verses becomes a list ("16,17"); the site redirects
// "16-17" there.
func urlSpans(spans []ref.Span) string {
	out := make([]ref.Span, 0, len(spans))

	for _, span := range spans {
		if span.To == span.From+1 {
			out = append(out, ref.Span{From: span.From, To: span.From}, ref.Span{From: span.To, To: span.To})

			continue
		}

		out = append(out, span)
	}

	return ref.FormatSpans(out)
}

// ChapterURL is the site link for the chapter of r.
func ChapterURL(bible string, r ref.Ref) string {
	return fmt.Sprintf("%s/%s/%s/%d",
		SiteBase, url.PathEscape(bible), url.PathEscape(r.Book.Slug), r.Chapter,
	)
}
