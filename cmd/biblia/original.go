package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/casara/biblia-cli/internal/books"
	"github.com/casara/biblia-cli/internal/config"
	"github.com/casara/biblia-cli/internal/grammar"
	"github.com/casara/biblia-cli/internal/ref"
	"github.com/casara/biblia-cli/internal/render"
	"github.com/casara/biblia-cli/internal/step"
	"github.com/casara/biblia-cli/internal/ubs"
)

// stepSource is the dictionary id of STEPBible's own English glosses.
const stepSource = "step"

// dictSource is one entry of the reader's dictionary preferences: a UBS
// dictionary, or STEPBible's English glosses.
type dictSource struct {
	dict ubs.Dictionary
	step bool
}

// dictionaryChain resolves the configured dictionary ids, or the
// defaults for the reader's language, always ending in STEPBible.
func dictionaryChain(cfg config.Config) ([]dictSource, error) {
	ids := cfg.Dictionaries
	if len(ids) == 0 {
		ids = append(ubs.Defaults(cfg.Language), stepSource)
	}

	chain := make([]dictSource, 0, len(ids))

	for _, id := range ids {
		if id == stepSource {
			chain = append(chain, dictSource{step: true})

			continue
		}

		d, ok := ubs.Find(id)
		if !ok {
			known := make([]string, 0, len(ubs.Dictionaries))
			for _, d := range ubs.Dictionaries {
				known = append(known, d.ID)
			}

			return nil, fmt.Errorf("%w: %w %q em \"dicionarios\" (disponíveis: %s, %s)",
				errUsage, ubs.ErrUnknownDictionary, id, strings.Join(known, ", "), stepSource)
		}

		chain = append(chain, dictSource{dict: d})
	}

	return chain, nil
}

// originals holds what the interlinear, the lexicon and the parallels
// need: the STEPBible index, the UBS index (nil when not downloaded) and
// the reader's dictionary preferences.
type originals struct {
	step      *step.Store
	ubs       *ubs.Store
	installed map[string]int
	chain     []dictSource
	lang      string
}

// openUBS opens the UBS index; a missing index is not an error (nil).
func openUBS(ctx context.Context, path string) (*ubs.Store, map[string]int, error) {
	store, err := ubs.Open(path)
	if errors.Is(err, ubs.ErrNotInstalled) {
		return nil, nil, nil
	}

	if err != nil {
		return nil, nil, err
	}

	installed, err := store.Installed(ctx)
	if err != nil {
		_ = store.Close()

		return nil, nil, err
	}

	return store, installed, nil
}

func (o *originals) close() {
	if o.step != nil {
		_ = o.step.Close()
	}

	if o.ubs != nil {
		_ = o.ubs.Close()
	}
}

// usable returns the installed UBS dictionaries of the chain for one
// testament, in order, stopping at STEPBible.
func (o *originals) usable(greek bool) []ubs.Dictionary {
	var out []ubs.Dictionary

	for _, src := range o.chain {
		if src.step {
			break
		}

		if src.dict.Greek != greek || o.ubs == nil {
			continue
		}

		if _, ok := o.installed[src.dict.ID]; ok {
			out = append(out, src.dict)
		}
	}

	return out
}

// source names where the glosses of one testament come from.
func (o *originals) source(greek bool) string {
	if dicts := o.usable(greek); len(dicts) > 0 {
		return dicts[0].Label()
	}

	return "STEPBible (en)"
}

// wordLine finds a word's gloss for the verse it is in and decodes its
// grammar.
func (o *originals) wordLine(ctx context.Context, w step.Word, book *books.Book) (render.WordLine, error) {
	greek := strings.HasPrefix(w.Strong, "G")
	dicts := o.usable(greek)

	// Without any dictionary the header already says the glosses are
	// STEPBible's English; the per-word mark is for the gaps of one.
	line := render.WordLine{Word: w, Gloss: w.Gloss, Foreign: o.lang != "en" && len(dicts) > 0}
	at := ubsVerse(w, book)

	for _, d := range dicts {
		senses, err := o.ubs.Senses(ctx, d.ID, w.Strong, &at)
		if err != nil {
			return line, err
		}

		if len(senses) == 0 {
			continue
		}

		line.Gloss, line.AI = pickGloss(senses)
		line.Foreign = false

		break
	}

	attrs, err := o.step.Grammar(ctx, w)
	if err != nil {
		return line, err
	}

	line.Grammar = w.Grammar
	if desc, ok := grammar.Describe(attrs, o.lang); ok {
		line.Grammar = desc
	}

	return line, nil
}

// ubsVerse is the verse of a word in the UBS numbering: the Hebrew one
// for the Old Testament (Sl 51.1 is 51.3 there), the translations' for
// the New, which matches the UBS Greek references better than the Greek
// editions' own numbers.
func ubsVerse(w step.Word, book *books.Book) ubs.Verse {
	v := ubs.Verse{Book: book.Number(), Chapter: w.Chapter, Verse: w.Verse}

	if !book.NT && w.AltRef != "" {
		var c, n int

		_, err := fmt.Sscanf(w.AltRef, "%d.%d", &c, &n)
		if err == nil {
			v.Chapter, v.Verse = c, n
		}
	}

	return v
}

// pickGloss returns the first gloss of each sense the verse uses; when
// the dictionary lists none for this verse, the first gloss of each of
// the first senses, so the reader sees the range of meanings instead of
// a guess.
func pickGloss(senses []ubs.Sense) (string, bool) {
	const maxSenses = 3

	var chosen []ubs.Sense

	for _, s := range senses {
		if s.InVerse {
			chosen = append(chosen, s)
		}
	}

	if len(chosen) == 0 {
		chosen = senses[:min(len(senses), maxSenses)]
	}

	var (
		glosses []string
		ai      bool
	)

	seen := map[string]bool{}

	for _, s := range chosen {
		if len(s.Glosses) == 0 || seen[s.Glosses[0]] {
			continue
		}

		seen[s.Glosses[0]] = true
		glosses = append(glosses, s.Glosses[0])
		ai = ai || s.AI
	}

	return strings.Join(glosses, " / "), ai
}

// interlinear prints the original words of every verse shown.
func (o *originals) interlinear(
	ctx context.Context, tw render.Terminal, book *books.Book, chapter int, verses []int,
) error {
	source := o.source(book.NT)

	for _, verse := range verses {
		words, err := o.step.Words(ctx, book.Step, chapter, verse)
		if err != nil {
			return err
		}

		lines := make([]render.WordLine, 0, len(words))

		for _, w := range words {
			line, err := o.wordLine(ctx, w, book)
			if err != nil {
				return err
			}

			lines = append(lines, line)
		}

		fmt.Fprintln(tw.W)
		tw.Interlinear(fmt.Sprintf("%s %d.%d", book.Abbr, chapter, verse), source, lines)
	}

	return nil
}

// maxVerse bounds the verse numbers tried for a whole chapter (Sl 119
// has 176).
const maxVerse = 176

// parallel is one passage parallel to the reference asked about.
type parallel struct {
	ref   ref.Ref
	title bool
	kind  string
}

func (p parallel) String() string {
	if p.title && p.ref.WholeChapter() {
		return p.ref.String() + " (título)"
	}

	if p.title {
		return p.ref.String() + " (com o título)"
	}

	return p.ref.String()
}

// parallels finds the passages parallel to any verse of r, converting
// between the translations' numbering and the original's (which the UBS
// database uses) when the STEPBible index is available.
func (o *originals) parallels(ctx context.Context, r ref.Ref) ([]parallel, error) {
	verses := r.Verses()
	if r.WholeChapter() {
		for v := 1; v <= maxVerse; v++ {
			verses = append(verses, v)
		}
	}

	var out []parallel

	seen := map[string]bool{}

	for _, v := range verses {
		chapter, verse, err := o.toOriginal(ctx, r.Book, r.Chapter, v)
		if err != nil {
			return nil, err
		}

		found, err := o.ubs.Parallels(ctx, r.Book.Step, chapter, verse)
		if err != nil {
			return nil, err
		}

		for _, p := range found {
			item, ok, err := o.fromPassage(ctx, p, r.Book.NT)
			if err != nil {
				return nil, err
			}

			if !ok || seen[item.String()] || sameRef(item.ref, r) {
				continue
			}

			seen[item.String()] = true
			out = append(out, item)
		}
	}

	return out, nil
}

func sameRef(a, b ref.Ref) bool {
	return a.Book == b.Book && a.String() == b.String()
}

func (o *originals) toOriginal(ctx context.Context, book *books.Book, chapter, verse int) (int, int, error) {
	if book.NT || o.step == nil {
		return chapter, verse, nil
	}

	return o.step.OriginalVerse(ctx, book.Step, chapter, verse)
}

// fromPassage turns a UBS passage into a reference in the translations'
// numbering. A Psalm title (verse 0 after conversion) is folded into the
// passage and flagged.
func (o *originals) fromPassage(ctx context.Context, p ubs.Passage, fromNT bool) (parallel, bool, error) {
	book, ok := books.ByCode(p.Book)
	if !ok {
		return parallel{}, false, nil
	}

	item := parallel{kind: render.KindParallel}

	switch {
	case !fromNT && p.Greek:
		item.kind = render.KindQuotedInNT
	case fromNT && !p.Greek:
		item.kind = render.KindQuotesOT
	}

	r := ref.Ref{Book: book, Chapter: p.Chapter}

	for _, sp := range p.Spans {
		fromCh, from, err := o.fromOriginal(ctx, book, p.Chapter, sp.From)
		if err != nil {
			return parallel{}, false, err
		}

		toCh, to, err := o.fromOriginal(ctx, book, p.Chapter, sp.To)
		if err != nil {
			return parallel{}, false, err
		}

		if toCh != fromCh {
			to = from
		}

		if from == 0 {
			item.title = true
			from = 1
		}

		if to < from {
			continue
		}

		r.Chapter = fromCh
		r.Spans = append(r.Spans, ref.Span{From: from, To: to})
	}

	item.ref = r

	return item, item.title || len(r.Spans) > 0, nil
}

func (o *originals) fromOriginal(ctx context.Context, book *books.Book, chapter, verse int) (int, int, error) {
	if book.NT || o.step == nil {
		return chapter, verse, nil
	}

	return o.step.TranslationVerse(ctx, book.Step, chapter, verse)
}
