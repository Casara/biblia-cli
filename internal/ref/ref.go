// Package ref parses Bible references as people type them in Portuguese
// ("Jo 3.16", "1co13:4-7", "Rm 8", "Rm 5.8; 6.23", "Jd 3") into a
// structured form, and prints them back in the Brazilian style the rest
// of the program uses ("1 Co 13.4-7").
//
// Parsing is entirely offline: it never asks the server whether a verse
// exists, only whether the chapter is within the book.
package ref

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/casara/biblia-cli/internal/books"
)

// Sentinel errors, matchable with errors.Is.
var (
	ErrEmpty       = errors.New("referência vazia")
	ErrUnknownBook = errors.New("livro não reconhecido")
	ErrSyntax      = errors.New("referência mal formada")
	ErrChapter     = errors.New("capítulo fora do livro")
)

// Span is an inclusive verse range within a chapter. A single verse has
// From == To.
type Span struct {
	From, To int
}

// Ref is one chapter-bounded reference. Spans is empty for a whole
// chapter ("Sl 23").
type Ref struct {
	Book    *books.Book
	Chapter int
	Spans   []Span
}

// WholeChapter reports whether the reference names a chapter with no
// verse selection.
func (r Ref) WholeChapter() bool {
	return len(r.Spans) == 0
}

// Verses expands the spans into the individual verse numbers, in order
// and without duplicates. It returns nil for a whole chapter.
func (r Ref) Verses() []int {
	seen := map[int]bool{}

	var out []int

	for _, s := range r.Spans {
		for v := s.From; v <= s.To; v++ {
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}

	return out
}

// String prints the reference in the Brazilian style: "1 Co 13.4-7,9".
// A single-chapter book keeps the chapter ("Jd 1.3"), which is how the
// printed magazines write it too.
func (r Ref) String() string {
	return r.format(r.Book.Abbr, ".")
}

// Query is the form sent to the Bíblia Online server: the full book name
// and a colon separator ("1 Coríntios 13:4-7,9").
func (r Ref) Query() string {
	return r.format(r.Book.Name, ":")
}

func (r Ref) format(book, sep string) string {
	head := fmt.Sprintf("%s %d", book, r.Chapter)
	if r.WholeChapter() {
		return head
	}

	return head + sep + FormatSpans(r.Spans)
}

// FormatSpans prints spans as "4-7,9".
func FormatSpans(spans []Span) string {
	parts := make([]string, 0, len(spans))

	for _, s := range spans {
		if s.From == s.To {
			parts = append(parts, strconv.Itoa(s.From))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", s.From, s.To))
		}
	}

	return strings.Join(parts, ",")
}

// Compact turns a list of verse numbers into the fewest spans covering
// exactly them, e.g. [4 5 6 9] -> 4-6,9. The input must be sorted.
func Compact(verses []int) []Span {
	var out []Span

	for _, v := range verses {
		if n := len(out); n > 0 && out[n-1].To == v-1 {
			out[n-1].To = v

			continue
		}

		out = append(out, Span{From: v, To: v})
	}

	return out
}

// bookRe splits a segment into the book part and the numbers part. The
// book part may start with a digit or a Roman numeral ("1 Co", "II Rs")
// and is letters after that; the rest must start with a digit.
var bookRe = regexp.MustCompile(`^\s*((?i:[1-3]|i{1,3}\s)?\s*\pL[\pL.]*)\s*(.*)$`)

// chapterRe matches a segment that starts with a chapter number rather
// than a book name ("6.23", "7").
var chapterRe = regexp.MustCompile(`^\s*\d+\s*([.:,\-–]|$)`)

// numbersRe accepts "3", "3.16", "3:16", "3.16-18,20", "3,16" is NOT
// accepted as chapter,verse (the comma lists verses).
var numbersRe = regexp.MustCompile(`^(\d+)(?:\s*[.:]\s*([\d\s,\-–]+))?\s*$`)

// Parse reads one or more references separated by ";". A segment that
// starts with a number reuses the previous segment's book, so
// "Rm 5.8; 6.23" yields two references in Romans.
func Parse(input string) ([]Ref, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, ErrEmpty
	}

	var (
		out  []Ref
		last *books.Book
	)

	for seg := range strings.SplitSeq(input, ";") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		parsed, err := parseSegment(seg, last)
		if err != nil {
			return nil, err
		}

		last = parsed.Book
		out = append(out, parsed)
	}

	if len(out) == 0 {
		return nil, ErrEmpty
	}

	return out, nil
}

func parseSegment(seg string, last *books.Book) (Ref, error) {
	book := last
	numbers := seg

	if m := bookRe.FindStringSubmatch(seg); m != nil && !startsWithChapter(seg) {
		b, ok := books.Lookup(m[1])
		if !ok {
			return Ref{}, fmt.Errorf("%w: %q", ErrUnknownBook, strings.TrimSpace(m[1]))
		}

		book = b
		numbers = m[2]
	}

	if book == nil {
		return Ref{}, fmt.Errorf("%w: %q não diz o livro", ErrSyntax, seg)
	}

	return parseNumbers(book, numbers, seg)
}

// startsWithChapter tells "6.23" (a chapter of the previous book) apart
// from "1 Co 13" or "1co13" (a numbered book).
func startsWithChapter(seg string) bool {
	return chapterRe.MatchString(seg)
}

func parseNumbers(book *books.Book, numbers, seg string) (Ref, error) {
	numbers = strings.TrimSpace(numbers)
	if numbers == "" {
		return Ref{}, fmt.Errorf("%w: %q não diz o capítulo", ErrSyntax, seg)
	}

	m := numbersRe.FindStringSubmatch(numbers)
	if m == nil {
		return Ref{}, fmt.Errorf("%w: %q", ErrSyntax, seg)
	}

	first, err := strconv.Atoi(m[1])
	if err != nil {
		return Ref{}, fmt.Errorf("%w: %q", ErrSyntax, seg)
	}

	out := Ref{Book: book, Chapter: first}

	switch {
	case m[2] != "":
		spans, err := parseSpans(m[2])
		if err != nil {
			return Ref{}, fmt.Errorf("%w: %q", err, seg)
		}

		out.Spans = spans
	case book.SingleChapter():
		// "Jd 3" is verse 3 of the only chapter.
		out.Chapter = 1
		out.Spans = []Span{{From: first, To: first}}
	}

	if out.Chapter < 1 || out.Chapter > book.Chapters {
		return Ref{}, fmt.Errorf(
			"%w: %s tem %d capítulo(s), pedido %d",
			ErrChapter, book.Name, book.Chapters, out.Chapter,
		)
	}

	return out, nil
}

func parseSpans(s string) ([]Span, error) {
	var out []Span

	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(strings.ReplaceAll(part, "–", "-"))
		if part == "" {
			continue
		}

		from, to, isRange := strings.Cut(part, "-")

		lo, err := strconv.Atoi(strings.TrimSpace(from))
		if err != nil || lo < 1 {
			return nil, ErrSyntax
		}

		hi := lo

		if isRange {
			hi, err = strconv.Atoi(strings.TrimSpace(to))
			if err != nil || hi < lo {
				return nil, ErrSyntax
			}
		}

		out = append(out, Span{From: lo, To: hi})
	}

	if len(out) == 0 {
		return nil, ErrSyntax
	}

	return out, nil
}
