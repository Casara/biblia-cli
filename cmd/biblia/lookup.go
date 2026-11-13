package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/casara/biblia-cli/internal/bibliaonline"
	"github.com/casara/biblia-cli/internal/cache"
	"github.com/casara/biblia-cli/internal/lookup"
	"github.com/casara/biblia-cli/internal/ref"
	"github.com/casara/biblia-cli/internal/render"
	"github.com/casara/biblia-cli/internal/step"
	"github.com/casara/biblia-cli/internal/ubs"
)

type lookupOptions struct {
	versions, format string
	strong, offline  bool
	// parallels lists the parallel passages; parallelText also fetches
	// their text.
	parallels, parallelText bool
}

func lookupFlags(name string, e *env, defaultFormat string) (*flag.FlagSet, *lookupOptions) {
	o := &lookupOptions{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	fs.StringVar(&o.versions, "v", strings.Join(e.cfg.Versions, ","), "traduções, separadas por vírgula")
	fs.StringVar(&o.format, "f", defaultFormat, "formato de saída")
	fs.BoolVar(&o.strong, "s", false, "mostra o original com os números de Strong")
	fs.BoolVar(&o.offline, "offline", false, "usa só o cache")
	fs.BoolVar(&o.parallels, "p", false, "lista as passagens paralelas (UBS)")
	fs.BoolVar(&o.parallelText, "pt", false, "passagens paralelas com o texto")

	return fs, o
}

// session opens what a lookup needs and closes it afterwards.
type session struct {
	svc   *lookup.Service
	tpls  *render.Templates
	orig  *originals
	cache *cache.Cache
	// onlyParallels makes the filter answer each reference with the text
	// of its parallel passages instead of its own (filtro -p).
	onlyParallels bool
}

func openSession(ctx context.Context, e *env, o *lookupOptions) (*session, error) {
	tpls, err := render.NewTemplates(e.cfg.Templates)
	if err != nil {
		return nil, err
	}

	if !tpls.Has(o.format) {
		return nil, fmt.Errorf("%w: formato %q (disponíveis: %s)",
			errUsage, o.format, strings.Join(tpls.Names(), ", "))
	}

	c, err := cache.Open(ctx, e.paths.Cache)
	if err != nil {
		return nil, err
	}

	s := &session{
		svc:   &lookup.Service{Cache: c, Client: bibliaonline.New(), Offline: o.offline},
		tpls:  tpls,
		cache: c,
	}

	if o.strong || o.parallels {
		s.orig, err = openOriginals(ctx, e, o)
		if err != nil {
			_ = c.Close()

			return nil, err
		}
	}

	return s, nil
}

// openOriginals opens the STEPBible and UBS indexes. The interlinear
// needs STEPBible (UBS only improves its glosses); the parallels need
// UBS (STEPBible only converts the Old Testament numbering).
func openOriginals(ctx context.Context, e *env, o *lookupOptions) (*originals, error) {
	chain, err := dictionaryChain(e.cfg)
	if err != nil {
		return nil, err
	}

	orig := &originals{chain: chain, lang: e.cfg.Language}

	orig.step, err = step.Open(ctx, e.paths.StepDB)
	if err != nil && (o.strong || !errors.Is(err, step.ErrNotInstalled)) {
		return nil, err
	}

	orig.ubs, orig.installed, err = openUBS(ctx, e.paths.UBSDB)
	if err != nil {
		orig.close()

		return nil, err
	}

	if o.parallels && orig.ubs == nil {
		orig.close()

		return nil, ubs.ErrNotInstalled
	}

	if o.parallels && orig.step == nil {
		fmt.Fprintln(e.stderr,
			"biblia: sem os dados da STEPBible, a numeração do AT não é convertida (Salmos podem divergir)")
	}

	if o.strong && orig.ubs == nil {
		fmt.Fprintln(e.stderr, "biblia: dicionários da UBS não instalados; glosas em inglês (rode: biblia dados baixar)")
	}

	return orig, nil
}

func (s *session) close() {
	_ = s.cache.Close()

	if s.orig != nil {
		s.orig.close()
	}
}

func cmdLookup(ctx context.Context, e *env, args []string) error {
	fs, o := lookupFlags("biblia", e, e.cfg.Format)

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}

	if len(pos) == 0 {
		return fmt.Errorf("%w: falta a referência", errUsage)
	}

	refs, err := ref.Parse(strings.Join(pos, " "))
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	o.parallels = o.parallels || o.parallelText

	versions := splitList(o.versions)
	if len(versions) == 0 {
		return fmt.Errorf("%w: nenhuma tradução em -v", errUsage)
	}

	s, err := openSession(ctx, e, o)
	if err != nil {
		return err
	}
	defer s.close()

	return s.print(ctx, e, refs, versions, o)
}

func (s *session) print(
	ctx context.Context, e *env, refs []ref.Ref, versions []string, o *lookupOptions,
) error {
	color := e.tty && os.Getenv("NO_COLOR") == ""
	tw := render.Terminal{W: e.stdout, Color: color, Width: e.width}

	failed := false

	for i, r := range refs {
		if i > 0 {
			fmt.Fprintln(e.stdout)
		}

		err := s.printRef(ctx, e, tw, r, versions, o)
		if errors.Is(err, errReported) {
			failed = true
		} else if err != nil {
			return err
		}
	}

	if failed {
		return errReported
	}

	return nil
}

// printRef looks one reference up in every translation and prints it,
// with its parallel passages when asked (-p, -pt).
func (s *session) printRef(
	ctx context.Context, e *env, tw render.Terminal, r ref.Ref, versions []string, o *lookupOptions,
) error {
	var (
		found []parallel
		ann   annotation
	)

	if o.parallels {
		var err error

		found, err = s.orig.parallels(ctx, r)
		if err != nil {
			return err
		}

		ann.parallels = parallelItems(found)
	}

	results, failed := s.lookupAll(ctx, e, r, versions)

	if len(results) > 0 {
		err := s.emit(ctx, e, tw, results, o, ann)
		if err != nil {
			return err
		}
	}

	if o.parallels {
		err := s.printParallels(ctx, e, tw, r, found, versions, o)
		if errors.Is(err, errReported) {
			failed = true
		} else if err != nil {
			return err
		}
	}

	if failed {
		return errReported
	}

	return nil
}

// lookupAll looks r up in each translation, reporting failures on stderr.
func (s *session) lookupAll(ctx context.Context, e *env, r ref.Ref, versions []string) ([]lookup.Result, bool) {
	results := make([]lookup.Result, 0, len(versions))
	failed := false

	for _, version := range versions {
		res, err := s.svc.Lookup(ctx, r, version)
		if err != nil {
			failed = true
			fmt.Fprintln(e.stderr, "biblia:", err)

			continue
		}

		results = append(results, res)
	}

	return results, failed
}

// annotation is what the paste formats learn about parallel passages:
// the list, on the passage asked about, or where a parallel's text was
// found from.
type annotation struct {
	parallels []render.Parallel
	of        *render.ParallelOf
}

func parallelItems(found []parallel) []render.Parallel {
	items := make([]render.Parallel, 0, len(found))
	for _, p := range found {
		items = append(items, render.Parallel{Ref: p.String(), Kind: p.kind})
	}

	return items
}

func (s *session) emit(
	ctx context.Context, e *env, tw render.Terminal, results []lookup.Result, o *lookupOptions, ann annotation,
) error {
	if o.format != render.Text {
		for i, res := range results {
			if i > 0 {
				fmt.Fprintln(e.stdout)
			}

			d := render.NewData(res)
			d.Parallels, d.ParallelOf = ann.parallels, ann.of

			err := s.tpls.ExecuteData(e.stdout, o.format, d)
			if err != nil {
				return err
			}

			fmt.Fprintln(e.stdout)
		}

		return nil
	}

	if len(results) == 1 {
		tw.Single(results[0])
	} else {
		tw.Compare(results)
	}

	if o.strong {
		res := results[0]

		verses := make([]int, 0, len(res.Verses))
		for _, v := range res.Verses {
			verses = append(verses, v.Number)
		}

		return s.orig.interlinear(ctx, tw, res.Ref.Book, res.Ref.Chapter, verses)
	}

	return nil
}

// printParallels shows the list of passages parallel to r in the
// terminal view (the paste formats carry it in their template data) and,
// with -pt, looks each one up like the main reference (one server call
// per passage and translation, each under the 5-verse limit).
func (s *session) printParallels(
	ctx context.Context, e *env, tw render.Terminal, r ref.Ref, found []parallel, versions []string, o *lookupOptions,
) error {
	if o.format == render.Text {
		fmt.Fprintln(e.stdout)
		tw.Parallels(r.String(), parallelItems(found))
	}

	if !o.parallelText {
		return nil
	}

	return s.parallelTexts(ctx, e, tw, r, found, versions, o)
}

// parallelTexts looks up each parallel passage (-pt).
func (s *session) parallelTexts(
	ctx context.Context, e *env, tw render.Terminal, origin ref.Ref, found []parallel, versions []string,
	o *lookupOptions,
) error {
	failed := false
	inner := *o
	inner.strong, inner.parallels, inner.parallelText = false, false, false

	for _, p := range found {
		if p.ref.WholeChapter() {
			continue
		}

		results, lookupFailed := s.lookupAll(ctx, e, p.ref, versions)
		failed = failed || lookupFailed

		if len(results) == 0 {
			continue
		}

		fmt.Fprintln(e.stdout)

		ann := annotation{parallels: nil, of: render.NewParallelOf(origin.String(), p.kind)}

		err := s.emit(ctx, e, tw, results, &inner, ann)
		if err != nil {
			return err
		}
	}

	if failed {
		return errReported
	}

	return nil
}

// cmdFilter reads lines from standard input and replaces each reference
// with its text, for editors: select "Jo 3.16", pipe it through
// "biblia filtro", get the verse back. A line that is not a reference is
// passed through unchanged and silently, so a filter run over a whole
// selection never loses text; a reference that fails to resolve is also
// kept, and reported on stderr.
//
// With -p (or -pt) each reference is answered with the text of its
// parallel passages instead, and nothing else is written: editors insert
// that output after the selection, so echoing the input would duplicate
// it.
func cmdFilter(ctx context.Context, e *env, args []string) error {
	fs, o := lookupFlags("biblia filtro", e, e.cfg.FilterFormat)

	_, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}

	if o.format == render.Text {
		o.format = "linha"
	}

	o.strong = false
	o.parallels = o.parallels || o.parallelText
	o.parallelText = false

	versions := splitList(o.versions)

	s, err := openSession(ctx, e, o)
	if err != nil {
		return err
	}
	defer s.close()

	s.onlyParallels = o.parallels

	return s.filter(ctx, e, versions, o.format)
}

func (s *session) filter(ctx context.Context, e *env, versions []string, format string) error {
	sc := bufio.NewScanner(e.stdin)
	failed := false

	for sc.Scan() {
		line := sc.Text()
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

		out, err := s.filterLine(ctx, strings.TrimSpace(line), versions, format)
		if err != nil {
			failed = true
			fmt.Fprintln(e.stderr, "biblia:", err)

			if !s.onlyParallels {
				fmt.Fprintln(e.stdout, line)
			}

			continue
		}

		if s.onlyParallels && out == "" {
			continue
		}

		for l := range strings.SplitSeq(out, "\n") {
			fmt.Fprintln(e.stdout, indent+l)
		}
	}

	err := sc.Err()
	if err != nil {
		return fmt.Errorf("lendo a entrada: %w", err)
	}

	if failed {
		return errReported
	}

	return nil
}

func (s *session) filterLine(ctx context.Context, line string, versions []string, format string) (string, error) {
	if line == "" {
		return "", nil
	}

	refs, err := ref.Parse(line)
	if err != nil {
		if s.onlyParallels {
			return "", nil
		}

		return line, nil // not a reference: pass the line through
	}

	targets := make([]filterTarget, 0, len(refs))
	for _, r := range refs {
		targets = append(targets, filterTarget{ref: r, of: nil})
	}

	if s.onlyParallels {
		targets, err = s.parallelTargets(ctx, refs)
		if err != nil {
			return "", err
		}
	}

	var parts []string

	for _, target := range targets {
		for _, v := range versions {
			res, err := s.svc.Lookup(ctx, target.ref, v)
			if err != nil {
				return "", err
			}

			d := render.NewData(res)
			d.ParallelOf = target.of

			var b strings.Builder

			err = s.tpls.ExecuteData(&b, format, d)
			if err != nil {
				return "", err
			}

			parts = append(parts, b.String())
		}
	}

	return strings.Join(parts, "\n"), nil
}

// filterTarget is one reference the filter looks up; of is set when it
// is a parallel passage of what the reader selected.
type filterTarget struct {
	ref ref.Ref
	of  *render.ParallelOf
}

// parallelTargets returns the passages parallel to any of refs, each
// once. A parallel that is only a Psalm title has no verse to look up and
// is left out.
func (s *session) parallelTargets(ctx context.Context, refs []ref.Ref) ([]filterTarget, error) {
	var out []filterTarget

	seen := map[string]bool{}

	for _, r := range refs {
		found, err := s.orig.parallels(ctx, r)
		if err != nil {
			return nil, err
		}

		for _, p := range found {
			if p.ref.WholeChapter() || seen[p.ref.String()] {
				continue
			}

			seen[p.ref.String()] = true
			out = append(out, filterTarget{ref: p.ref, of: render.NewParallelOf(r.String(), p.kind)})
		}
	}

	return out, nil
}
