// Package step reads the STEPBible data (www.STEPBible.org, from the
// work of Tyndale House, Cambridge, CC BY 4.0): the tagged original-
// language texts TAHOT (Hebrew OT) and TAGNT (Greek NT), which give every
// word of every verse its extended Strong number, and the brief lexicons
// TBESH/TBESG that define those numbers.
//
// The files are downloaded on demand into the reader's data directory and
// indexed into SQLite; they are never shipped with the program, because
// STEPBible asks that the data not be redistributed (see data/SOURCE.md).
package step

import (
	"regexp"
	"strconv"
	"strings"
)

// Word is one word of a verse in the original language.
type Word struct {
	// Book is the STEPBible code ("Jhn"); Chapter and Verse follow the
	// English (NRSV/KJV) versification, which the Almeida translations
	// share in almost every place. AltRef keeps the Hebrew or Greek-
	// edition reference when it differs ("51.3").
	Book           string
	Chapter, Verse int
	Pos            int
	AltRef         string
	// Type is STEPBible's word-type code: for Hebrew L (Leningrad), Q
	// (Qere), K (Ketiv)...; for Greek the editions letters (N = Nestle-
	// Aland, K = KJV's TR, O = other), lower case for minor variants.
	Type string
	// Original and Translit are the word with prefixes and suffixes
	// joined; English is STEPBible's contextual translation.
	Original, Translit, English string
	// Strong is the main extended Strong number ("H2603A", "G0025");
	// Grammar is STEPBible's morphology code.
	Strong, Grammar string
	// Gloss is the short dictionary gloss of the main word.
	Gloss string
	// Editions lists the Greek editions that contain the word (NT only).
	Editions string
}

// InCriticalText reports whether a Greek word is in the Nestle-Aland
// text (always true for Hebrew).
func (w Word) InCriticalText() bool {
	if !strings.HasPrefix(w.Strong, "G") {
		return true
	}

	return strings.ContainsAny(w.Type, "Nn")
}

// Entry is one lexicon entry.
type Entry struct {
	// Strong is the original Strong number ("H5162"); EStrong the
	// extended one this entry defines ("H5162H").
	Strong, EStrong string
	// Relation and Target tie an entry to another one when the lexicon
	// files it so: H5162H is "a Meaning of" H5162G, H4430 is "in Aramaic
	// of" H4428G. Both are empty for an entry that stands alone.
	Relation, Target string
	Original         string
	Translit         string
	Morph            string
	Gloss            string
	// Definition is the full entry, with the source's HTML markup.
	Definition string
}

// refRe matches the reference column of TAHOT and TAGNT rows:
// "Psa.51.1(51.3)#01=L", "Mat.17.15[17.14]#01=NKO", "Jhn.3.16#01=NKO".
var refRe = regexp.MustCompile(
	`^([1-3]?[A-Z][a-z]{1,2})\.(\d+)\.(\d+)(?:[(\[](\d+\.\d+)[)\]])?#(\d+)=([^\t]*)$`,
)

// braceRe finds the main word's Strong number in a TAHOT dStrongs
// column: "H9003/{H0776G}" -> "H0776G".
var braceRe = regexp.MustCompile(`\{([HG]\d{4}[A-Za-z]?)`)

// glossRe finds the gloss of the braced word in TAHOT's expanded
// column: "{H2603A=חָנַן=be gracious}" -> "be gracious".
var glossRe = regexp.MustCompile(`\{[HG]\d{4}[A-Za-z]?=[^=]*=([^}]*)\}`)

// parseRef reads the reference column into a Word with only its
// position fields set.
func parseRef(col string) (Word, bool) {
	m := refRe.FindStringSubmatch(col)
	if m == nil {
		return Word{}, false
	}

	chapter, _ := strconv.Atoi(m[2])
	verse, _ := strconv.Atoi(m[3])
	pos, _ := strconv.Atoi(m[5])

	return Word{Book: m[1], Chapter: chapter, Verse: verse, Pos: pos, AltRef: m[4], Type: m[6]}, true
}

// ParseHebrewRow parses one TAHOT data row. Columns: ref, Hebrew,
// transliteration, translation, dStrongs, grammar, meaning variants,
// spelling variants, root dStrong, alternative Strongs, conjoin,
// expanded Strong tags.
func ParseHebrewRow(line string) (Word, bool) {
	const minCols = 12

	cols := strings.Split(line, "\t")
	if len(cols) < minCols {
		return Word{}, false
	}

	w, ok := parseRef(cols[0])
	if !ok {
		return Word{}, false
	}

	w.Original = strings.NewReplacer("/", "", `\`, "").Replace(cols[1])
	w.Translit = strings.NewReplacer("/", "", ".", "", `\`, "").Replace(cols[2])
	w.English = strings.TrimSpace(strings.NewReplacer("/ ", " ", "/", " ", `\`, " ").Replace(cols[3]))
	w.Grammar = cols[5]

	m := braceRe.FindStringSubmatch(cols[4])
	if m != nil {
		w.Strong = m[1]
	} else {
		w.Strong, _, _ = strings.Cut(strings.Trim(cols[4], "{}"), "/")
	}

	const expandedCol = 11

	m = glossRe.FindStringSubmatch(cols[expandedCol])
	if m != nil {
		w.Gloss = cleanGloss(m[1])
	}

	return w, true
}

// ParseGreekRow parses one TAGNT data row. Columns: ref & type, Greek
// "(translit)", English, "dStrong=grammar", "dictionary form=gloss",
// editions, meaning variants, spelling variants, Spanish, sub-meaning,
// conjoin, sStrong+instance, alt Strongs.
func ParseGreekRow(line string) (Word, bool) {
	const minCols = 6

	cols := strings.Split(line, "\t")
	if len(cols) < minCols {
		return Word{}, false
	}

	w, ok := parseRef(cols[0])
	if !ok {
		return Word{}, false
	}

	w.English = strings.TrimSpace(cols[2])
	w.Editions = cols[5]

	greek, translit, _ := strings.Cut(cols[1], "(")
	w.Original = strings.TrimSpace(greek)
	w.Translit = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(translit), ")"))

	tag := cols[3]
	if i := strings.LastIndex(tag, "«"); i >= 0 {
		tag = tag[i+len("«"):]
	}

	w.Strong, w.Grammar, _ = strings.Cut(tag, "=")

	_, gloss, found := strings.Cut(cols[4], "=")
	if found {
		w.Gloss = cleanGloss(gloss)
	}

	return w, true
}

// cleanGloss trims STEPBible's sub-meaning notation: ": man»man:1_man"
// -> "man".
func cleanGloss(s string) string {
	s, _, _ = strings.Cut(s, "»")
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), ":"))

	return strings.ReplaceAll(s, "_", " ")
}

// entryRe matches a lexicon data row's first column ("H0034", "G0025",
// and the disambiguated "H2603a").
var entryRe = regexp.MustCompile(`^[HG]\d{4}[A-Za-z]?$`)

// ParseLexiconRow parses one TBESH/TBESG row: Strong; the entry's own
// extended number with its relation to another entry ("H5162H = a
// Meaning of", or just "H5162G ="); that other entry's number (the
// entry's own when it stands alone); original, transliteration,
// morphology, gloss, definition.
func ParseLexiconRow(line string) (Entry, bool) {
	const minCols = 7

	cols := strings.Split(line, "\t")
	if len(cols) < minCols || !entryRe.MatchString(cols[0]) {
		return Entry{}, false
	}

	own, relation, _ := strings.Cut(cols[1], "=")
	own, relation = strings.TrimSpace(own), strings.TrimSpace(relation)

	if own == "" {
		own = cols[0]
	}

	e := Entry{
		Strong: cols[0], EStrong: own, Original: cols[3],
		Translit: cols[4], Morph: cols[5], Gloss: cols[6],
	}

	if relation != "" {
		e.Relation, e.Target = relation, strings.TrimSpace(cols[2])
	}

	const defCol = 7
	if len(cols) > defCol {
		e.Definition = cols[defCol]
	}

	return e, true
}

// Legend code shapes: Greek full codes ("V-AAI-3S", "ADV", "S-1SNSM") and
// Hebrew segment codes with their language letter ("HVqp3ms", "ANcmsd",
// "HR"). The brief lexicon codes ("H:N-M") have a colon and are skipped.
var (
	greekLegendRe  = regexp.MustCompile(`^[A-Z0-9]+(-[A-Z0-9]+)*$`)
	hebrewLegendRe = regexp.MustCompile(`^[HA][A-Za-z0-9]+$`)
)

// ParseLegendRow parses one row of a morphology legend (TEGMC when greek,
// TEHMC otherwise): the code and its expansion. The code comes back
// prefixed with "G:" or "H:" because both legends share one table and a
// Greek code such as "ADV" is also a valid shape for Hebrew.
func ParseLegendRow(line string, greek bool) (string, string, bool) {
	cols := strings.Split(line, "\t")
	if len(cols) < 2 || !strings.Contains(cols[1], "=") {
		return "", "", false
	}

	code := strings.TrimSpace(cols[0])
	attrs := strings.TrimSpace(cols[1])

	if greek && greekLegendRe.MatchString(code) {
		return "G:" + code, attrs, true
	}

	if !greek && hebrewLegendRe.MatchString(code) {
		return "H:" + code, attrs, true
	}

	return "", "", false
}
