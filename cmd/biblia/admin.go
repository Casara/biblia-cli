package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/casara/biblia-cli/internal/bibliaonline"
	"github.com/casara/biblia-cli/internal/cache"
	"github.com/casara/biblia-cli/internal/render"
	"github.com/casara/biblia-cli/internal/step"
	"github.com/casara/biblia-cli/internal/ubs"
)

func cmdStrong(ctx context.Context, e *env, args []string) error {
	fs := flag.NewFlagSet("biblia strong", flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	full := fs.Bool("c", false, "mostra também o domínio de cada sentido e o verbete da STEPBible")
	translit := fs.String("tr", "", "busca pela transliteração")
	gloss := fs.String("glosa", "", "busca pela glosa (no idioma dos dicionários)")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}

	chain, err := dictionaryChain(e.cfg)
	if err != nil {
		return err
	}

	store, err := step.Open(ctx, e.paths.StepDB)
	if err != nil {
		return err
	}

	orig := &originals{step: store, chain: chain, lang: e.cfg.Language}
	defer orig.close()

	orig.ubs, orig.installed, err = openUBS(ctx, e.paths.UBSDB)
	if err != nil {
		return err
	}

	entries, err := orig.findEntries(ctx, pos, *translit, *gloss)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return errNothingFound
	}

	return orig.printEntries(ctx, e, entries, *full)
}

// findEntries runs the strong command's search: by number, by
// transliteration, or by gloss (first in the reader's dictionaries, then
// in STEPBible's English glosses).
func (o *originals) findEntries(ctx context.Context, pos []string, translit, gloss string) ([]step.Entry, error) {
	const limit = 20

	switch {
	case translit != "":
		return o.step.SearchTranslit(ctx, translit, limit)
	case gloss != "":
		return o.searchGloss(ctx, gloss, limit)
	case len(pos) == 1:
		return o.step.Lookup(ctx, pos[0])
	default:
		return nil, fmt.Errorf("%w: diga um número de Strong, -tr ou -glosa", errUsage)
	}
}

func (o *originals) searchGloss(ctx context.Context, text string, limit int) ([]step.Entry, error) {
	var dicts []string

	for _, greek := range []bool{false, true} {
		for _, d := range o.usable(greek) {
			dicts = append(dicts, d.ID)
		}
	}

	if o.ubs != nil && len(dicts) > 0 {
		strongs, err := o.ubs.SearchGloss(ctx, dicts, text, limit)
		if err != nil {
			return nil, err
		}

		var out []step.Entry

		seen := map[string]bool{}

		for _, strong := range strongs {
			entries, err := o.step.Lookup(ctx, strong)
			if err != nil {
				return nil, err
			}

			for _, en := range entries {
				if key := en.Strong + " " + en.EStrong; !seen[key] {
					seen[key] = true
					out = append(out, en)
				}
			}
		}

		if len(out) > 0 {
			return out, nil
		}
	}

	return o.step.SearchGloss(ctx, text, limit)
}

// printEntries shows each STEPBible entry and, once per Strong number,
// the senses of the first dictionary of the reader's that has them.
func (o *originals) printEntries(ctx context.Context, e *env, entries []step.Entry, full bool) error {
	tw := render.Terminal{W: e.stdout, Color: e.tty && os.Getenv("NO_COLOR") == "", Width: e.width}
	shown := map[string]bool{}

	const plainLen = 5 // "H2617"

	for i, en := range entries {
		if i > 0 {
			fmt.Fprintln(e.stdout)
		}

		base := en.EStrong
		if len(base) > plainLen {
			base = base[:plainLen]
		}

		senses, source, err := o.lexiconSenses(ctx, base)
		if err != nil {
			return err
		}

		tw.Entry(en, len(senses) == 0, full)

		switch {
		case len(senses) > 0 && !shown[base]:
			shown[base] = true

			tw.Senses(source, senses, full)
		case len(senses) > 0:
			fmt.Fprintf(e.stdout, "  %s\n", tw.Dim("sentidos de "+base+": acima"))
		}
	}

	return nil
}

func (o *originals) lexiconSenses(ctx context.Context, strong string) ([]ubs.Sense, string, error) {
	for _, d := range o.usable(strings.HasPrefix(strong, "G")) {
		senses, err := o.ubs.Senses(ctx, d.ID, strong, nil)
		if err != nil {
			return nil, "", err
		}

		if len(senses) > 0 {
			return senses, d.Label(), nil
		}
	}

	return nil, "", nil
}

const catalogKey = "catalog"

func cmdVersions(ctx context.Context, e *env, args []string) error {
	fs := flag.NewFlagSet("biblia versoes", flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	all := fs.Bool("todas", false, "inclui as traduções em outros idiomas")

	_, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}

	c, err := cache.Open(ctx, e.paths.Cache)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	bibles, err := catalog(ctx, c)
	if err != nil {
		return err
	}

	for _, b := range bibles {
		if !*all && b.Language != "Português" {
			continue
		}

		fmt.Fprintf(e.stdout, "%-10s %s (%s)\n", b.ID, b.Name, b.Language)
	}

	if !*all {
		fmt.Fprintln(e.stderr, "(só português; -todas para as demais)")
	}

	return nil
}

// catalog returns the translation list, cached for a week.
func catalog(ctx context.Context, c *cache.Cache) ([]bibliaonline.Bible, error) {
	const maxAge = 7 * 24 * time.Hour

	raw, fresh, err := c.GetValue(ctx, catalogKey, maxAge)
	if err != nil {
		return nil, err
	}

	var bibles []bibliaonline.Bible

	if fresh {
		err := json.Unmarshal([]byte(raw), &bibles)
		if err == nil {
			return bibles, nil
		}
	}

	bibles, err = bibliaonline.New().ListBibles(ctx)
	if err != nil {
		if raw != "" && json.Unmarshal([]byte(raw), &bibles) == nil {
			return bibles, nil
		}

		return nil, err
	}

	enc, err := json.Marshal(bibles)
	if err == nil {
		_ = c.PutValue(ctx, catalogKey, string(enc))
	}

	return bibles, nil
}

func cmdFormats(e *env) error {
	tpls, err := render.NewTemplates(e.cfg.Templates)
	if err != nil {
		return err
	}

	fmt.Fprintln(e.stdout, strings.Join(tpls.Names(), "\n"))

	return nil
}

const dataUsage = "biblia dados baixar [step|ubs] [-enxuto] | status"

func cmdData(ctx context.Context, e *env, args []string) error {
	fs := flag.NewFlagSet("biblia dados", flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	lean := fs.Bool("enxuto", false, "apaga os arquivos baixados depois de indexá-los")

	args, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}

	const maxArgs = 2 // "baixar ubs"

	if len(args) == 0 || len(args) > maxArgs {
		return fmt.Errorf("%w: %s", errUsage, dataUsage)
	}

	switch args[0] {
	case "baixar":
		which := ""
		if len(args) == maxArgs {
			which = args[1]
		}

		if which != "" && which != stepSource && which != "ubs" {
			return fmt.Errorf("%w: %s", errUsage, dataUsage)
		}

		return downloadData(ctx, e, which, *lean)
	case "status":
		return dataStatus(ctx, e)
	default:
		return fmt.Errorf("%w: %s", errUsage, dataUsage)
	}
}

const downloadTimeout = 10 * time.Minute

// downloadData fetches and indexes STEPBible, UBS or both (which == "").
// When lean, each source's downloaded files are deleted once its index
// is in place: the program only reads the indexes, and a later
// "dados baixar" fetches the files again.
func downloadData(ctx context.Context, e *env, which string, lean bool) error {
	if which == "" || which == stepSource {
		err := downloadStep(ctx, e)
		if err != nil {
			return err
		}

		if lean {
			removeRaw(e, e.paths.RawDir)
		}
	}

	if which == "" || which == "ubs" {
		err := downloadUBS(ctx, e)
		if err != nil {
			return err
		}

		if lean {
			removeRaw(e, e.paths.UBSRawDir)
		}
	}

	return nil
}

func removeRaw(e *env, dir string) {
	size := dirSize(dir)

	err := os.RemoveAll(dir)
	if err != nil {
		fmt.Fprintf(e.stderr, "biblia: não consegui apagar %s: %v\n", dir, err)

		return
	}

	fmt.Fprintf(e.stderr, "apagados os arquivos baixados de %s (%s)\n", dir, megabytes(size))
}

// dirSize sums the sizes of the files in dir; a missing dir is 0.
func dirSize(dir string) int64 {
	var total int64

	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr // an unreadable entry just does not count
		}

		info, err := d.Info()
		if err == nil {
			total += info.Size()
		}

		return nil
	})

	return total
}

func megabytes(n int64) string {
	const mb = 1 << 20

	return fmt.Sprintf("%d MB", (n+mb/2)/mb)
}

// rawNote says how much the downloaded files of one source take.
func rawNote(dir string) string {
	size := dirSize(dir)
	if size == 0 {
		return "  arquivos baixados: apagados (o índice basta)"
	}

	return fmt.Sprintf("  arquivos baixados: %s em %s (dados baixar -enxuto os apaga)", megabytes(size), dir)
}

func downloadStep(ctx context.Context, e *env) error {
	in := &step.Installer{HTTP: &http.Client{Timeout: downloadTimeout}, Log: e.stderr}

	commit, err := in.LatestCommit(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintf(e.stderr, "STEPBible-Data, commit %s\n", commit)

	err = in.Download(ctx, commit, e.paths.RawDir)
	if err != nil {
		return err
	}

	err = in.Index(ctx, e.paths.RawDir, e.paths.StepDB, commit)
	if err != nil {
		return err
	}

	fmt.Fprintf(e.stderr, "pronto: %s\n", e.paths.StepDB)

	return nil
}

// downloadUBS fetches the dictionaries of the reader's preferences (or
// the defaults for their language) and the parallel passages.
func downloadUBS(ctx context.Context, e *env) error {
	chain, err := dictionaryChain(e.cfg)
	if err != nil {
		return err
	}

	var dicts []ubs.Dictionary

	for _, src := range chain {
		if !src.step {
			dicts = append(dicts, src.dict)
		}
	}

	in := &ubs.Installer{HTTP: &http.Client{Timeout: downloadTimeout}, Log: e.stderr}

	commit, err := in.LatestCommit(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintf(e.stderr, "ubs-open-license, commit %s\n", commit)

	err = in.Download(ctx, commit, e.paths.UBSRawDir, dicts)
	if err != nil {
		return err
	}

	err = in.Index(ctx, e.paths.UBSRawDir, e.paths.UBSDB, commit, dicts)
	if err != nil {
		return err
	}

	fmt.Fprintf(e.stderr, "pronto: %s\n", e.paths.UBSDB)

	return nil
}

func dataStatus(ctx context.Context, e *env) error {
	err := stepStatus(ctx, e)
	if err != nil {
		return err
	}

	err = ubsStatus(ctx, e)
	if err != nil {
		return err
	}

	chain, err := dictionaryChain(e.cfg)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(chain))
	for _, src := range chain {
		if src.step {
			names = append(names, stepSource)
		} else {
			names = append(names, src.dict.ID)
		}
	}

	fmt.Fprintf(e.stdout, "dicionários em uso (idioma %s): %s\n", e.cfg.Language, strings.Join(names, ", "))

	return nil
}

func cmdCache(ctx context.Context, e *env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: biblia cache status | limpar [versão]", errUsage)
	}

	c, err := cache.Open(ctx, e.paths.Cache)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	switch args[0] {
	case "status":
		stats, err := c.Stats(ctx)
		if err != nil {
			return err
		}

		fmt.Fprintln(e.stdout, e.paths.Cache)

		for _, bible := range slices.Sorted(maps.Keys(stats)) {
			fmt.Fprintf(e.stdout, "  %-10s %d versículo(s)\n", bible, stats[bible])
		}

		return nil
	case "limpar":
		bible := ""
		if len(args) > 1 {
			bible = strings.ToLower(args[1])
		}

		n, err := c.Clear(ctx, bible)
		if err != nil {
			return err
		}

		fmt.Fprintf(e.stdout, "%d registro(s) apagado(s)\n", n)

		return nil
	default:
		return fmt.Errorf("%w: biblia cache status | limpar [versão]", errUsage)
	}
}

func stepStatus(ctx context.Context, e *env) error {
	store, err := step.Open(ctx, e.paths.StepDB)
	if err != nil {
		fmt.Fprintf(e.stdout, "STEPBible: %v\n", err)
	} else {
		meta, err := store.Meta(ctx)
		outdated := store.Outdated()
		_ = store.Close()

		if err != nil {
			return err
		}

		fmt.Fprintf(e.stdout, "STEPBible\n  índice:  %s\n  fonte:   %s\n  commit:  %s\n  gerado:  %s\n",
			e.paths.StepDB, meta["source"], meta["commit"], meta["indexed_at"])

		if outdated {
			fmt.Fprintln(e.stdout, "  índice de uma versão anterior do programa (rode: biblia dados baixar step)")
		}

		fmt.Fprintln(e.stdout, rawNote(e.paths.RawDir))
	}

	return nil
}

func ubsStatus(ctx context.Context, e *env) error {
	us, installed, err := openUBS(ctx, e.paths.UBSDB)
	if err != nil {
		return err
	}

	if us == nil {
		fmt.Fprintf(e.stdout, "UBS: %v\n", ubs.ErrNotInstalled)
	} else {
		meta, err := us.Meta(ctx)
		_ = us.Close()

		if err != nil {
			return err
		}

		fmt.Fprintf(e.stdout, "UBS\n  índice:  %s\n  fonte:   %s\n  commit:  %s\n  gerado:  %s\n",
			e.paths.UBSDB, meta["source"], meta["commit"], meta["indexed_at"])

		for _, id := range slices.Sorted(maps.Keys(installed)) {
			fmt.Fprintf(e.stdout, "  %-16s %d sentidos\n", id, installed[id])
		}

		fmt.Fprintln(e.stdout, rawNote(e.paths.UBSRawDir))
	}

	return nil
}
