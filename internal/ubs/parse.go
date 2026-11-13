package ubs

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Sense is one meaning of a dictionary entry.
type Sense struct {
	// Dict is the dictionary id ("ubs-hebraico-pt").
	Dict string
	// Lemma is the entry's headword in the original language.
	Lemma string
	// Strongs are the plain Strong numbers of the entry ("H2617"),
	// Aramaic ones (UBS "A4430") filed under their H number.
	Strongs []string
	// Glosses are the short renderings; Definition the sense's
	// definition, with the UBS markup already resolved.
	Glosses    []string
	Definition string
	// Domain is the first lexical semantic domain ("Nomes de Pessoas").
	Domain string
	// AI marks a sense the UBS itself flags as machine-translated with
	// human review recommended.
	AI bool
	// Refs are the verses that use the word in this sense, in the
	// numbering of the original text.
	Refs []Verse
	// InVerse is set by queries that ask about one verse: the verse is
	// among Refs.
	InVerse bool
}

// Verse is a verse in the UBS numbering: canonical book number (1–66),
// chapter and verse of the Hebrew text or Greek edition.
type Verse struct {
	Book, Chapter, Verse int
}

// rawEntry mirrors the parts of a UBS dictionary entry the program uses.
//
//nolint:tagliatelle // field names are UBS's
type rawEntry struct {
	Lemma       string   `json:"Lemma"`
	StrongCodes []string `json:"StrongCodes"`
	BaseForms   []struct {
		LEXMeanings []struct {
			LEXDomains []struct {
				Domain string `json:"Domain"`
			} `json:"LEXDomains"`
			LEXSenses []struct {
				AIGenerated     bool     `json:"AIGenerated"`
				DefinitionShort string   `json:"DefinitionShort"`
				DefinitionLong  string   `json:"DefinitionLong"`
				Glosses         []string `json:"Glosses"`
			} `json:"LEXSenses"`
			LEXReferences []string `json:"LEXReferences"`
		} `json:"LEXMeanings"`
	} `json:"BaseForms"`
}

// ErrFormat is returned for a file that is not what the program expects.
var ErrFormat = errors.New("formato inesperado")

// ParseDictionary reads a UBS dictionary (a JSON array of entries) and
// calls fn with every sense that has a Strong number. It streams the
// array, so the 25 MB files are never held in memory whole.
func ParseDictionary(r io.Reader, dict string, fn func(Sense) error) error {
	dec := json.NewDecoder(skipBOM(r))

	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("ubs: reading %s: %w", dict, err)
	}

	if delim, ok := tok.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("ubs: %s: %w (esperava uma lista)", dict, ErrFormat)
	}

	for dec.More() {
		var e rawEntry

		err := dec.Decode(&e)
		if err != nil {
			return fmt.Errorf("ubs: reading %s: %w", dict, err)
		}

		err = emitSenses(e, dict, fn)
		if err != nil {
			return err
		}
	}

	return nil
}

const utf8BOM = "\uFEFF"

// skipBOM drops a leading byte-order mark, which both the JSON and the
// XML files of the collection carry and neither decoder accepts.
func skipBOM(r io.Reader) io.Reader {
	br := bufio.NewReader(r)

	bom, err := br.Peek(len(utf8BOM))
	if err == nil && string(bom) == utf8BOM {
		_, _ = br.Discard(len(utf8BOM))
	}

	return br
}

func emitSenses(e rawEntry, dict string, fn func(Sense) error) error {
	strongs := NormalizeCodes(e.StrongCodes)
	if len(strongs) == 0 {
		return nil
	}

	for _, base := range e.BaseForms {
		for _, meaning := range base.LEXMeanings {
			if len(meaning.LEXSenses) == 0 {
				continue
			}

			raw := meaning.LEXSenses[0]

			def := raw.DefinitionShort
			if raw.DefinitionLong != "" {
				def = raw.DefinitionLong
			}

			s := Sense{
				Domain:     "",
				InVerse:    false,
				Dict:       dict,
				Lemma:      e.Lemma,
				Strongs:    strongs,
				Glosses:    raw.Glosses,
				Definition: CleanText(def),
				AI:         raw.AIGenerated,
				Refs:       ParseRefs(meaning.LEXReferences),
			}

			if len(meaning.LEXDomains) > 0 {
				s.Domain = meaning.LEXDomains[0].Domain
			}

			err := fn(s)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// codeRe matches a UBS Strong code: "H2617", "G0025", the Aramaic
// "A4430", with an optional disambiguating letter ("H7451b").
var codeRe = regexp.MustCompile(`^([HGA])(\d{4})`)

// NormalizeCodes reduces UBS Strong codes to plain four-digit numbers,
// the common ground with STEPBible's extended numbers: the letter
// suffixes of the two sources mean different things, and UBS files
// Aramaic words as "A" where Strong (and STEPBible) number them "H".
func NormalizeCodes(codes []string) []string {
	var out []string

	seen := map[string]bool{}

	for _, c := range codes {
		m := codeRe.FindStringSubmatch(strings.TrimSpace(c))
		if m == nil {
			continue
		}

		prefix := m[1]
		if prefix == "A" {
			prefix = "H"
		}

		n := prefix + m[2]
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}

	return out
}

// ParseRefs decodes UBS references, "BBBCCCVVVSSWWW" (book, chapter,
// verse, segment, word), sometimes followed by a footnote mark
// ("{N:001}"), into distinct verses. The word position is dropped: UBS
// counts words differently from STEPBible, so only the verse can be
// matched.
func ParseRefs(refs []string) []Verse {
	const verseDigits = 9

	var out []Verse

	seen := map[Verse]bool{}

	for _, code := range refs {
		if len(code) < verseDigits {
			continue
		}

		book, err1 := strconv.Atoi(code[0:3])
		chapter, err2 := strconv.Atoi(code[3:6])
		verse, err3 := strconv.Atoi(code[6:9])

		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}

		v := Verse{Book: book, Chapter: chapter, Verse: verse}
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}

	return out
}

var (
	// linkRe: "{L:Salomão<SDBH:שְׁלֹמֹה>}" → "Salomão", with an optional
	// trailing "[a]" note letter (Greek dictionary).
	linkRe = regexp.MustCompile(`\{L:([^<}]*)<[^>]*>\}(\[[a-z]\])?`)
	// abbrRe: "{A:NIV}" → "NIV".
	abbrRe = regexp.MustCompile(`\{A:([^}]*)\}`)
	// dropRe: footnotes, Scripture and domain codes, which make no sense
	// without the UBS interface ("{N:001}", "{S:...}", "{D:25.43}").
	dropRe  = regexp.MustCompile(`\{[NSD]:[^}]*\}`)
	spaceRe = regexp.MustCompile(`\s+`)
)

// CleanText resolves the UBS markup of a definition into plain text and
// drops the "= " that opens the Hebrew dictionary's definitions.
func CleanText(s string) string {
	s = linkRe.ReplaceAllString(s, "$1")
	s = abbrRe.ReplaceAllString(s, "$1")
	s = dropRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))

	return strings.TrimSpace(strings.TrimPrefix(s, "= "))
}

// Passage is one member of a group of parallel passages: a book (USFM
// code, "2SA"), a chapter and verse spans, in the numbering of the
// original text. Greek is set for New Testament passages.
type Passage struct {
	Book    string
	Chapter int
	Spans   []Span
	Greek   bool
}

// Span is a verse range; From == To for a single verse.
type Span struct {
	From, To int
}

// Contains reports whether the passage includes a verse.
func (p Passage) Contains(chapter, verse int) bool {
	if chapter != p.Chapter {
		return false
	}

	for _, s := range p.Spans {
		if verse >= s.From && verse <= s.To {
			return true
		}
	}

	return false
}

type xmlPassages struct {
	Passages []struct {
		Verses []struct {
			Heb  string `xml:"HEB,attr"`
			Grk  string `xml:"GRK,attr"`
			Text string `xml:",chardata"`
		} `xml:"Verse"`
	} `xml:"Passage"`
}

// passageRe: "GEN 1:27", "MRK 10:7-8", "1CH 1:5,7", "PSA 40:14-18,20".
var passageRe = regexp.MustCompile(`^([1-3]?[A-Z]{2,3})\s+(\d+):([\d,\-]+)$`)

// ParseParallels reads the UBS parallel passages database into groups.
// A member whose reference cannot be read fails the whole parse: the
// file is fixed data, and a new shape should be noticed, not skipped.
func ParseParallels(r io.Reader) ([][]Passage, error) {
	var doc xmlPassages

	err := xml.NewDecoder(skipBOM(r)).Decode(&doc)
	if err != nil {
		return nil, fmt.Errorf("ubs: reading parallel passages: %w", err)
	}

	groups := make([][]Passage, 0, len(doc.Passages))

	for _, p := range doc.Passages {
		group := make([]Passage, 0, len(p.Verses))

		for _, verse := range p.Verses {
			passage, err := parsePassage(strings.TrimSpace(verse.Text))
			if err != nil {
				return nil, err
			}

			passage.Greek = verse.Grk != ""
			group = append(group, passage)
		}

		groups = append(groups, group)
	}

	return groups, nil
}

func parsePassage(s string) (Passage, error) {
	m := passageRe.FindStringSubmatch(s)
	if m == nil {
		return Passage{}, fmt.Errorf("ubs: parallel passage %q: %w", s, ErrFormat)
	}

	chapter, err := strconv.Atoi(m[2])
	if err != nil {
		return Passage{}, fmt.Errorf("ubs: parallel passage %q: %w", s, ErrFormat)
	}

	p := Passage{Book: m[1], Chapter: chapter, Spans: nil, Greek: false}

	for part := range strings.SplitSeq(m[3], ",") {
		from, to, isRange := strings.Cut(part, "-")
		if !isRange {
			to = from
		}

		first, err1 := strconv.Atoi(from)
		last, err2 := strconv.Atoi(to)

		if err1 != nil || err2 != nil || last < first {
			return Passage{}, fmt.Errorf("ubs: parallel passage %q: %w", s, ErrFormat)
		}

		p.Spans = append(p.Spans, Span{From: first, To: last})
	}

	return p, nil
}
