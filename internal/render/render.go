// Package render turns lookup results into text: the terminal view
// ("texto", with a side-by-side comparison when several translations are
// asked for) and template-based formats meant to be pasted into a
// document ("linha", "md", "typ", plus any the reader defines in the
// config file).
package render

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"text/template"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/casara/biblia-cli/internal/lookup"
	"github.com/casara/biblia-cli/internal/ref"
)

// Sentinel errors, matchable with errors.Is.
var (
	// ErrUnknownFormat is returned for a format name with no template.
	ErrUnknownFormat = errors.New("formato desconhecido")
	// ErrReservedName is returned for a template named after the
	// terminal format.
	ErrReservedName = errors.New("é o formato do terminal e não aceita modelo")
)

// Text is the name of the built-in terminal format.
const Text = "texto"

// Builtin holds the default templates. The "typ" one writes the
// "// conferido" line and the #bref call the EBD lesson project checks
// (versão · fonte · data, then «text»).
//
// With parallel passages (-p, -pt, filtro -p) they also say what each
// text is: "typ" in comments, which never reach the PDF; "md" in an
// italic line; "linha" inside its parentheses.
var Builtin = map[string]string{
	"linha": `{{.Text}} ({{.Ref}}, {{.Bible}}{{with .ParallelOf}}; {{.Relation}}{{end}})`,
	"md": `> {{md .Text}}
>
> — {{.Ref}} ({{.Bible}}){{with .ParallelOf}} · *{{.Relation}}*{{end}}{{if .Partial}}
>
> *Trecho parcial: faltam {{.Missing}}. Texto completo: <{{.URL}}>*{{end}}{{if .Parallels}}

*{{parallels .Parallels}}*{{end}}`,
	"typ": `{{with .ParallelOf}}// paralelo: {{$.Ref}} {{.Relation}}
{{end}}{{if .Partial}}// ATENÇÃO: trecho parcial, faltam {{.Missing}} ({{.URL}})
{{end}}// conferido: {{.Bible}} · bibliaonline.com.br/{{.BibleID}} · {{.Date}}
#bref("{{.Ref}}", [«{{typ .Text}}»]){{if .Parallels}}
// {{parallels .Parallels}}{{end}}`,
}

// Kinds of parallel passage, from the point of view of the passage asked
// about: Hc 2.4 is "citado no NT" in Rm 1.17; Rm 3.4 has a "citação do
// AT", Sl 51.4; Mt 6.9 has a "paralelo" in Lc 11.2.
const (
	KindQuotedInNT = "citado no NT"
	KindQuotesOT   = "citação do AT"
	KindParallel   = "paralelo"
)

// ParallelOf tells, on the text of a parallel passage, which passage it
// was found from.
type ParallelOf struct {
	// Ref is the passage asked about ("Hc 2.4"); Kind is one of the Kind
	// constants, from that passage's point of view.
	Ref, Kind string
	// Relation says it from this text's point of view: "cita Hc 2.4",
	// "citado em Rm 3.4", "paralelo de Mt 6.9".
	Relation string
}

// NewParallelOf describes a text found as a parallel of origin.
func NewParallelOf(origin, kind string) *ParallelOf {
	relation := "paralelo de " + origin

	switch kind {
	case KindQuotedInNT:
		relation = "cita " + origin
	case KindQuotesOT:
		relation = "citado em " + origin
	}

	return &ParallelOf{Ref: origin, Kind: kind, Relation: relation}
}

// FormatParallels writes a list of parallels grouped by kind, in order of
// first appearance: "Citado no NT: Rm 1.17 · Gl 3.11 · Hb 10.38". It is
// the "parallels" function of the templates.
func FormatParallels(items []Parallel) string {
	var kinds []string

	byKind := map[string][]string{}

	for _, it := range items {
		if _, ok := byKind[it.Kind]; !ok {
			kinds = append(kinds, it.Kind)
		}

		byKind[it.Kind] = append(byKind[it.Kind], it.Ref)
	}

	groups := make([]string, 0, len(kinds))
	for _, k := range kinds {
		groups = append(groups, k+": "+strings.Join(byKind[k], " · "))
	}

	out := strings.Join(groups, "; ")
	if out == "" {
		return ""
	}

	first, size := utf8.DecodeRuneInString(out)

	return string(unicode.ToUpper(first)) + out[size:]
}

// Data is what a template sees for one result.
type Data struct {
	// Ref is the reference of the text actually obtained, in Brazilian
	// style ("1 Co 13.4-7"). It differs from Requested only when the
	// result is partial: a "Gn 3.1-6" cut at five verses prints as
	// "Gn 3.1-5", so a pasted citation never promises more than it holds.
	Ref string
	// Requested is the reference as asked for.
	Requested string
	// Bible is the translation id in upper case ("ARA"); BibleID is the
	// id as the site uses it ("ara").
	Bible, BibleID string
	// Verses are the verses obtained; absent ones are left out.
	Verses []lookup.Verse
	// Text is the verses joined with a space, without numbers.
	Text string
	// URL links to the passage (to the chapter when Partial).
	URL string
	// Date is when the text was fetched from the site, YYYY-MM-DD.
	Date string
	// Partial is true when some requested verses are missing; Missing
	// lists them ("18-20").
	Partial bool
	Missing string
	// Absent lists verses the translation omits ("21").
	Absent string
	// Parallels lists, on the passage asked about, its parallel passages
	// (with -p or -pt); empty otherwise.
	Parallels []Parallel
	// ParallelOf is set on the text of a parallel passage (-pt,
	// filtro -p): where it was found from.
	ParallelOf *ParallelOf
}

// NewData builds the template data for one result.
func NewData(res lookup.Result) Data {
	d := Data{
		Ref:       obtainedRef(res),
		Requested: res.Ref.String(),
		Bible:     strings.ToUpper(res.Bible),
		BibleID:   res.Bible,
		URL:       res.URL,
		Partial:   res.Partial,
		Missing:   ref.FormatSpans(ref.Compact(res.Missing)),
	}

	if !res.FetchedAt.IsZero() {
		d.Date = res.FetchedAt.Format(time.DateOnly)
	}

	var (
		texts  []string
		absent []int
	)

	for _, verse := range res.Verses {
		if verse.Absent {
			absent = append(absent, verse.Number)

			continue
		}

		d.Verses = append(d.Verses, verse)
		texts = append(texts, verse.Text)
	}

	d.Text = strings.Join(texts, " ")
	d.Absent = ref.FormatSpans(ref.Compact(absent))

	if d.Partial && d.Missing == "" {
		d.Missing = "versículos do capítulo além destes"
	}

	return d
}

// obtainedRef narrows a partial result's reference to the verses that
// came back.
func obtainedRef(res lookup.Result) string {
	if !res.Partial || len(res.Verses) == 0 {
		return res.Ref.String()
	}

	numbers := make([]int, 0, len(res.Verses))
	for _, verse := range res.Verses {
		numbers = append(numbers, verse.Number)
	}

	got := ref.Ref{Book: res.Ref.Book, Chapter: res.Ref.Chapter, Spans: ref.Compact(numbers)}

	return got.String()
}

var funcs = template.FuncMap{
	"typ":       escapeTypst,
	"md":        escapeMarkdown,
	"parallels": FormatParallels,
}

// Templates resolves format names to parsed templates: the built-in ones
// overlaid with the reader's own.
type Templates struct {
	set map[string]*template.Template
}

// NewTemplates parses the built-in templates plus extra (name → source);
// an extra template with a built-in name replaces it.
func NewTemplates(extra map[string]string) (*Templates, error) {
	all := maps.Clone(Builtin)
	maps.Copy(all, extra)

	t := &Templates{set: map[string]*template.Template{}}

	for name, src := range all {
		if name == Text {
			return nil, fmt.Errorf("render: %q %w", Text, ErrReservedName)
		}

		tpl, err := template.New(name).Funcs(funcs).Parse(src)
		if err != nil {
			return nil, fmt.Errorf("render: modelo %q: %w", name, err)
		}

		t.set[name] = tpl
	}

	return t, nil
}

// Names lists every available format, the terminal one included.
func (t *Templates) Names() []string {
	names := slices.Sorted(maps.Keys(t.set))

	return append([]string{Text}, names...)
}

// Execute writes one result with the named template.
func (t *Templates) Execute(w io.Writer, format string, res lookup.Result) error {
	return t.ExecuteData(w, format, NewData(res))
}

// ExecuteData writes prepared template data, for callers that add the
// parallel-passage fields to NewData's result.
func (t *Templates) ExecuteData(w io.Writer, format string, d Data) error {
	tpl, ok := t.set[format]
	if !ok {
		return fmt.Errorf("%w: %q (disponíveis: %s)",
			ErrUnknownFormat, format, strings.Join(t.Names(), ", "))
	}

	err := tpl.Execute(w, d)
	if err != nil {
		return fmt.Errorf("render: %s: %w", format, err)
	}

	return nil
}

// Has reports whether a format exists.
func (t *Templates) Has(format string) bool {
	_, ok := t.set[format]

	return ok || format == Text
}

// escapeTypst protects the characters that mean something in Typst
// markup, so a verse lands in a content block as plain text. Bracketed
// verses ("[A graça ... Amém!]", the NAA's way of marking a verse absent
// from older manuscripts) are the case that actually occurs.
func escapeTypst(s string) string {
	var b strings.Builder

	for _, r := range s {
		if strings.ContainsRune(`\#*_[]$<>@~`+"`", r) {
			b.WriteByte('\\')
		}

		b.WriteRune(r)
	}

	return b.String()
}

func escapeMarkdown(s string) string {
	var b strings.Builder

	for _, r := range s {
		if strings.ContainsRune(`\*_[]<>`+"`", r) {
			b.WriteByte('\\')
		}

		b.WriteRune(r)
	}

	return b.String()
}
