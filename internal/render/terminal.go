package render

import (
	"fmt"
	"html"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/casara/biblia-cli/internal/lookup"
	"github.com/casara/biblia-cli/internal/ref"
	"github.com/casara/biblia-cli/internal/step"
	"github.com/casara/biblia-cli/internal/ubs"
)

// Terminal writes the human view. Color adds ANSI styling; Width, when
// positive, wraps text with a hanging indent.
type Terminal struct {
	W     io.Writer
	Color bool
	Width int
}

// Layout, in columns.
const (
	labelGap     = 2 // between a label (verse number, translation) and its text
	compareShift = 2 // left margin of the comparison view's lines
	entryIndent  = 4 // definition under a lexicon entry
)

const (
	bold  = "\033[1m"
	dim   = "\033[2m"
	cyan  = "\033[36m"
	yell  = "\033[33m"
	reset = "\033[0m"
)

// Single prints one translation of one reference:
//
//	Jo 3.16-17 · ARA
//	 16  Porque Deus amou ...
//	 17  Porquanto Deus ...
//	     https://www.bibliaonline.com.br/ara/jo/3/16-17
func (t Terminal) Single(res lookup.Result) {
	fmt.Fprintf(t.W, "%s %s\n", t.style(bold, res.Ref.String()), t.style(dim, "· "+strings.ToUpper(res.Bible)))

	numWidth := 0
	for _, v := range res.Verses {
		numWidth = max(numWidth, len(strconv.Itoa(v.Number)))
	}

	for _, v := range res.Verses {
		label := fmt.Sprintf("%*d", numWidth+1, v.Number)
		t.line(label, verseText(v), numWidth+1+labelGap)
	}

	t.footer(res, numWidth+1+labelGap, false)
}

// Compare prints several translations of one reference verse by verse:
//
//	Jo 3.16
//	  ARA  Porque Deus amou ao mundo de tal maneira ...
//	  NVI  Porque Deus tanto amou o mundo ...
func (t Terminal) Compare(results []lookup.Result) {
	if len(results) == 0 {
		return
	}

	tagWidth := 0
	for _, r := range results {
		tagWidth = max(tagWidth, len(r.Bible))
	}

	var order []int

	seen := map[int]bool{}

	for _, r := range results {
		for _, v := range r.Verses {
			if !seen[v.Number] {
				seen[v.Number] = true
				order = append(order, v.Number)
			}
		}
	}

	slices.Sort(order)

	first := results[0].Ref

	for i, n := range order {
		if i > 0 {
			fmt.Fprintln(t.W)
		}

		fmt.Fprintln(t.W, t.style(bold, fmt.Sprintf("%s %d.%d", first.Book.Abbr, first.Chapter, n)))

		for _, r := range results {
			tag := fmt.Sprintf("  %-*s", tagWidth, strings.ToUpper(r.Bible))
			text := t.style(dim, "(não obtido)")

			for _, v := range r.Verses {
				if v.Number == n {
					text = verseText(v)
				}
			}

			t.line(tag, text, compareShift+tagWidth+labelGap)
		}
	}

	for _, r := range results {
		if r.Partial || r.Stale {
			fmt.Fprintln(t.W)
			t.footer(r, compareShift, true)
		}
	}
}

func verseText(v lookup.Verse) string {
	if v.Absent {
		return "— (não consta nesta tradução)"
	}

	return v.Text
}

// WordLine is one word of the interlinear with what the caller found
// about it.
type WordLine struct {
	Word step.Word
	// Gloss is the word's meaning in this verse, from the first
	// dictionary that has it; Foreign marks a gloss that had to come in
	// another language (STEPBible's English) and AI a sense the UBS
	// flags as machine-translated.
	Gloss   string
	Foreign bool
	AI      bool
	// Grammar is the decoded grammar, or the raw code when it could not
	// be decoded.
	Grammar string
}

// Interlinear prints the original-language words of one verse; source
// names the dictionary of the glosses ("UBS grego (es)").
func (t Terminal) Interlinear(label, source string, words []WordLine) {
	if len(words) == 0 {
		fmt.Fprintf(t.W, "  %s\n", t.style(dim, label+": sem dados do texto original"))

		return
	}

	fmt.Fprintln(t.W, "  "+t.style(bold, interlinearHead(label, source, words[0].Word)))

	widths := [3]int{}
	for _, l := range words {
		widths[0] = max(widths[0], utf8.RuneCountInString(l.Word.Original))
		widths[1] = max(widths[1], utf8.RuneCountInString(l.Word.Translit))
		widths[2] = max(widths[2], len(l.Word.Strong))
	}

	// "  † " + three columns, each followed by two spaces.
	const lead, columns = 4, 3

	indent := lead + widths[0] + widths[1] + widths[2] + columns*labelGap
	variant, ai, foreign := false, false, false

	for _, l := range words {
		mark := " "
		if !l.Word.InCriticalText() {
			mark = "†"
			variant = true
		}

		prefix := fmt.Sprintf("  %s %s  %s  %s  ",
			mark,
			padRight(l.Word.Original, widths[0]),
			t.style(dim, padRight(l.Word.Translit, widths[1])),
			t.style(cyan, padRight(l.Word.Strong, widths[2])),
		)

		gloss := l.Gloss
		if l.Foreign {
			gloss += " (en)"
			foreign = true
		}

		if l.AI {
			gloss += " [IA]"
			ai = true
		}

		t.wordLine(prefix, gloss, l.Grammar, indent)
	}

	t.legend(variant, ai, foreign)
}

// relations translates the lexicon's notes tying an entry to another
// one; a note missing here is shown as the lexicon writes it.
var relations = map[string]string{
	"a meaning of":      "um sentido de",
	"a form of":         "forma de",
	"a spelling of":     "grafia de",
	"a name of":         "nome ligado a",
	"in aramaic of":     "forma aramaica de",
	"in hebrew of":      "forma hebraica de",
	"the greek of":      "forma grega de",
	"a group of":        "grupo de",
	"a group member of": "membro do grupo",
	"a part of":         "parte de",
	"a combination of":  "combinação de",
	"combination of":    "combinação de",
}

// relationNote is "  (um sentido de H5162G)" for a related entry.
func relationNote(e step.Entry) string {
	if e.Relation == "" {
		return ""
	}

	rel, ok := relations[strings.ToLower(e.Relation)]
	if !ok {
		rel = e.Relation
	}

	return fmt.Sprintf("  (%s %s)", rel, e.Target)
}

// Senses prints the senses a dictionary gives one Strong number,
// numbered; full adds each sense's semantic domain.
func (t Terminal) Senses(source string, senses []ubs.Sense, full bool) {
	fmt.Fprintf(t.W, "  %s\n", t.style(dim, source))

	ai := false

	for i, s := range senses {
		head := fmt.Sprintf("%d. %s", i+1, strings.Join(s.Glosses, ", "))
		if s.AI {
			head += " [IA]"
			ai = true
		}

		pad := strings.Repeat(" ", entryIndent)
		t.line("  ", head, labelGap+labelGap)

		text := s.Definition
		if full && s.Domain != "" {
			text += " [" + s.Domain + "]"
		}

		for _, l := range wrap(text, t.Width-entryIndent-labelGap) {
			fmt.Fprintf(t.W, "%s  %s\n", pad, t.style(dim, l))
		}
	}

	if ai {
		fmt.Fprintln(t.W, "  "+t.style(dim, "[IA] tradução feita por IA; a UBS recomenda revisão humana"))
	}
}

// Parallel is one passage of the parallels list: its reference ("Rm
// 1.17") and its kind (KindQuotedInNT, KindQuotesOT, KindParallel).
type Parallel struct {
	Ref, Kind string
}

// Parallels prints the passages parallel to label, grouped by kind.
func (t Terminal) Parallels(label string, items []Parallel) {
	if len(items) == 0 {
		fmt.Fprintf(t.W, "%s\n", t.style(dim, "Paralelos de "+label+": nenhum no banco da UBS"))

		return
	}

	fmt.Fprintln(t.W, t.style(bold, "Paralelos de "+label))

	var kinds []string

	byKind := map[string][]string{}

	for _, it := range items {
		if _, ok := byKind[it.Kind]; !ok {
			kinds = append(kinds, it.Kind)
		}

		byKind[it.Kind] = append(byKind[it.Kind], it.Ref)
	}

	width := 0
	for _, k := range kinds {
		width = max(width, utf8.RuneCountInString(k))
	}

	for _, k := range kinds {
		t.line("  "+padRight(k, width), strings.Join(byKind[k], " · "), compareShift+width+labelGap)
	}
}

// Entry prints one STEPBible lexicon entry. gloss adds its (English)
// gloss, which a dictionary in the reader's language replaces; full adds
// the definition, labelled as English when the gloss line is absent.
func (t Terminal) Entry(e step.Entry, gloss, full bool) {
	num := e.EStrong
	if num == "" {
		num = e.Strong
	}

	fmt.Fprintf(t.W, "%s  %s  %s  %s%s\n",
		t.style(bold, num), e.Original, t.style(dim, e.Translit), t.style(dim, e.Morph),
		t.style(dim, relationNote(e)))

	if gloss {
		fmt.Fprintf(t.W, "  %s\n", e.Gloss)
	}

	if full && e.Definition != "" {
		if !gloss {
			fmt.Fprintf(t.W, "  %s\n", t.style(dim, "STEPBible (inglês): "+e.Gloss))
		}

		pad := strings.Repeat(" ", entryIndent)
		for _, l := range wrap(CleanDefinition(e.Definition), t.Width-entryIndent) {
			fmt.Fprintf(t.W, "%s%s\n", pad, l)
		}
	}
}

// Dim styles s as secondary text.
func (t Terminal) Dim(s string) string {
	return t.style(dim, s)
}

// interlinearHead is "Sl 51.1 · hebraico (no original: 51.3) · glosas:
// UBS hebraico (pt)".
func interlinearHead(label, source string, first step.Word) string {
	lang := "hebraico"
	if strings.HasPrefix(first.Strong, "G") {
		lang = "grego"
	}

	head := fmt.Sprintf("%s · %s", label, lang)
	if first.AltRef != "" {
		head += fmt.Sprintf(" (no original: %s)", first.AltRef)
	}

	if source != "" {
		head += " · glosas: " + source
	}

	return head
}

// wordLine prints an interlinear row: the fixed columns, then gloss and
// grammar wrapped under the gloss column.
func (t Terminal) wordLine(prefix, gloss, grammar string, indent int) {
	text := gloss
	if grammar != "" {
		text += " · " + grammar
	}

	for i, row := range wrap(text, t.Width-indent) {
		// Dim the grammar part, which starts after the gloss.
		styled := row
		if i == 0 && grammar != "" && strings.HasPrefix(row, gloss) {
			styled = row[:len(gloss)] + t.style(dim, row[len(gloss):])
		} else if i > 0 {
			styled = t.style(dim, row)
		}

		if i == 0 {
			fmt.Fprintf(t.W, "%s%s\n", prefix, styled)
		} else {
			fmt.Fprintf(t.W, "%s%s\n", strings.Repeat(" ", indent), styled)
		}
	}
}

func (t Terminal) legend(variant, ai, foreign bool) {
	if variant {
		fmt.Fprintln(t.W, "  "+t.style(dim, "† fora do texto crítico (Nestle-Aland); presente no Textus Receptus/Bizantino"))
	}

	if ai {
		fmt.Fprintln(t.W, "  "+t.style(dim, "[IA] tradução feita por IA; a UBS recomenda revisão humana"))
	}

	if foreign {
		fmt.Fprintln(t.W, "  "+t.style(dim, "(en) sem verbete no dicionário escolhido: glosa da STEPBible, em inglês"))
	}
}

func (t Terminal) style(code, s string) string {
	if !t.Color || s == "" {
		return s
	}

	return code + s + reset
}

// line prints label + text, wrapping the text under itself.
func (t Terminal) line(label, text string, indent int) {
	label = t.style(cyan, label)
	lines := wrap(text, t.Width-indent)

	for i, l := range lines {
		if i == 0 {
			fmt.Fprintf(t.W, "%s  %s\n", label, l)
		} else {
			fmt.Fprintf(t.W, "%s%s\n", strings.Repeat(" ", indent), l)
		}
	}
}

func (t Terminal) footer(res lookup.Result, indent int, tagged bool) {
	pad := strings.Repeat(" ", indent)

	if res.Stale {
		fmt.Fprintln(t.W, pad+t.style(yell, "sem conexão com o site: texto do cache"))
	}

	if res.Partial {
		msg := "trecho parcial: o site entrega até 5 versículos por consulta"
		if len(res.Missing) > 0 {
			msg = "trecho parcial: faltam " + ref.FormatSpans(ref.Compact(res.Missing))
		}

		if tagged {
			msg = strings.ToUpper(res.Bible) + " · " + msg
		}

		fmt.Fprintln(t.W, pad+t.style(yell, msg))
	}

	fmt.Fprintln(t.W, pad+t.style(dim, res.URL))
}

var (
	tagRe   = regexp.MustCompile(`<[^>]+>`)
	brRe    = regexp.MustCompile(`(?i)<br\s*/?>`)
	spaceRe = regexp.MustCompile(`[ \t]+`)
)

// CleanDefinition turns the lexicon's HTML into plain text, keeping its
// line breaks.
func CleanDefinition(s string) string {
	s = brRe.ReplaceAllString(s, "\n")
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)

	var lines []string

	for l := range strings.SplitSeq(s, "\n") {
		l = strings.TrimSpace(spaceRe.ReplaceAllString(l, " "))
		if l != "" {
			lines = append(lines, l)
		}
	}

	return strings.Join(lines, "\n")
}

// wrap splits text into lines of at most width runes, on spaces,
// keeping existing line breaks. A non-positive width disables wrapping.
func wrap(text string, width int) []string {
	const minWidth = 20

	var out []string

	for para := range strings.SplitSeq(text, "\n") {
		if width < minWidth {
			out = append(out, para)

			continue
		}

		var cur strings.Builder

		for word := range strings.FieldsSeq(para) {
			if cur.Len() > 0 && utf8.RuneCountInString(cur.String())+1+utf8.RuneCountInString(word) > width {
				out = append(out, cur.String())
				cur.Reset()
			}

			if cur.Len() > 0 {
				cur.WriteByte(' ')
			}

			cur.WriteString(word)
		}

		out = append(out, cur.String())
	}

	return out
}

func padRight(s string, n int) string {
	if pad := n - utf8.RuneCountInString(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}

	return s
}
