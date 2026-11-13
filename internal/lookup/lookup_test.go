package lookup_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/casara/biblia-cli/internal/bibliaonline"
	"github.com/casara/biblia-cli/internal/cache"
	"github.com/casara/biblia-cli/internal/lookup"
	"github.com/casara/biblia-cli/internal/ref"
)

// fakeServer answers like the real one: up to five verses, skipping the
// ones the translation omits, and counts the calls.
type fakeServer struct {
	omitted map[int]bool
	calls   []string
	down    bool
}

var errDown = errors.New("connection refused")

func (f *fakeServer) GetPassage(_ context.Context, reference, _ string) (bibliaonline.Passage, error) {
	f.calls = append(f.calls, reference)

	if f.down {
		return bibliaonline.Passage{}, errDown
	}

	rs, err := ref.Parse(reference)
	if err != nil {
		return bibliaonline.Passage{}, fmt.Errorf("fake: %w", err)
	}

	var p bibliaonline.Passage

	for _, v := range rs[0].Verses() {
		if f.omitted[v] {
			continue
		}

		if len(p.Verses) == bibliaonline.MaxVerses {
			p.Partial = true

			break
		}

		p.Verses = append(p.Verses, bibliaonline.Verse{Number: v, Text: "texto"})
	}

	return p, nil
}

func newService(t *testing.T, f *fakeServer) *lookup.Service {
	t.Helper()

	c, err := cache.Open(context.Background(), filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = c.Close() })

	return &lookup.Service{
		Cache:  c,
		Client: f,
		Now:    func() time.Time { return time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC) },
	}
}

func mustRef(t *testing.T, s string) ref.Ref {
	t.Helper()

	rs, err := ref.Parse(s)
	if err != nil {
		t.Fatal(err)
	}

	return rs[0]
}

func TestCacheAvoidsSecondCall(t *testing.T) {
	f := &fakeServer{}
	s := newService(t, f)
	ctx := context.Background()

	for range 2 {
		res, err := s.Lookup(ctx, mustRef(t, "Jo 3.16-17"), "ara")
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Verses) != 2 || res.Partial {
			t.Fatalf("got %+v", res)
		}
	}

	if len(f.calls) != 1 {
		t.Errorf("server called %d times, want 1: %v", len(f.calls), f.calls)
	}
}

func TestOnlyMissingVersesAreRequested(t *testing.T) {
	f := &fakeServer{}
	s := newService(t, f)
	ctx := context.Background()

	_, err := s.Lookup(ctx, mustRef(t, "Jo 3.16"), "ara")
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Lookup(ctx, mustRef(t, "Jo 3.15-18"), "ara")
	if err != nil {
		t.Fatal(err)
	}

	if got := f.calls[1]; got != "João 3:15,17-18" {
		t.Errorf("second call = %q", got)
	}
}

func TestFiveVerseLimitIsReportedNotWorkedAround(t *testing.T) {
	f := &fakeServer{}
	s := newService(t, f)

	res, err := s.Lookup(context.Background(), mustRef(t, "Gn 3.1-7"), "ara")
	if err != nil {
		t.Fatal(err)
	}

	if len(f.calls) != 1 {
		t.Errorf("server called %d times, want exactly 1", len(f.calls))
	}

	if !res.Partial || len(res.Verses) != 5 || len(res.Missing) != 2 {
		t.Errorf("got %+v", res)
	}

	if res.URL != "https://www.bibliaonline.com.br/ara/gn/3" {
		t.Errorf("partial result should link to the chapter, got %q", res.URL)
	}
}

func TestOmittedVerseIsCachedAsAbsent(t *testing.T) {
	f := &fakeServer{omitted: map[int]bool{21: true}}
	s := newService(t, f)
	ctx := context.Background()

	for range 2 {
		res, err := s.Lookup(ctx, mustRef(t, "Mt 17.20-22"), "nvi")
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Verses) != 3 || !res.Verses[1].Absent || res.Partial {
			t.Fatalf("got %+v", res)
		}
	}

	if len(f.calls) != 1 {
		t.Errorf("absent verse asked again: %v", f.calls)
	}
}

func TestServerDownFallsBackToCache(t *testing.T) {
	f := &fakeServer{}
	s := newService(t, f)
	ctx := context.Background()

	_, err := s.Lookup(ctx, mustRef(t, "Jo 3.16"), "ara")
	if err != nil {
		t.Fatal(err)
	}

	f.down = true

	res, err := s.Lookup(ctx, mustRef(t, "Jo 3.16-17"), "ara")
	if err != nil {
		t.Fatal(err)
	}

	if !res.Stale || !res.Partial || len(res.Verses) != 1 {
		t.Errorf("got %+v", res)
	}

	_, err = s.Lookup(ctx, mustRef(t, "Rm 8.28"), "ara")
	if !errors.Is(err, lookup.ErrNotCached) {
		t.Errorf("uncached lookup with server down: %v", err)
	}
}

func TestOfflineNeverCallsServer(t *testing.T) {
	f := &fakeServer{}
	s := newService(t, f)
	s.Offline = true

	_, err := s.Lookup(context.Background(), mustRef(t, "Jo 3.16"), "ara")
	if !errors.Is(err, lookup.ErrNotCached) {
		t.Errorf("got %v", err)
	}

	if len(f.calls) != 0 {
		t.Errorf("offline lookup called the server: %v", f.calls)
	}
}

func TestPassageURLMatchesSiteCanonicalForm(t *testing.T) {
	cases := map[string]string{
		"Jo 3.16":     "https://www.bibliaonline.com.br/ara/jo/3/16",
		"Jo 3.16-17":  "https://www.bibliaonline.com.br/ara/jo/3/16,17",
		"Jo 3.16-18":  "https://www.bibliaonline.com.br/ara/jo/3/16-18",
		"Sl 23.1-3,6": "https://www.bibliaonline.com.br/ara/sl/23/1-3,6",
		"Jó 1.1":      "https://www.bibliaonline.com.br/ara/j%C3%B3/1/1",
		"Sl 23":       "https://www.bibliaonline.com.br/ara/sl/23",
	}

	for in, want := range cases {
		if got := lookup.PassageURL("ara", mustRef(t, in)); got != want {
			t.Errorf("PassageURL(%q) = %q, want %q", in, got, want)
		}
	}
}
