// Package books holds the 66-book Protestant canon as the rest of the
// program sees it: the Portuguese name and abbreviation shown to the
// reader, the slug bibliaonline.com.br uses in its URLs, the three-letter
// code STEPBible uses in its tagged texts, and the chapter count used to
// validate references offline.
//
// Deuterocanonical books (present in some Catholic translations on the
// site) are deliberately absent: none of the translations this tool is
// built around carry them, and the STEPBible data does not either.
package books

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Book describes one book of the canon.
type Book struct {
	// Name is the full Portuguese name, e.g. "1 Coríntios". It is also
	// what gets sent to the Bíblia Online server, which resolves it.
	Name string
	// Abbr is the Brazilian abbreviation used when printing a reference,
	// with a space after a leading number ("1 Co", "Jo", "Êx").
	Abbr string
	// Slug is the path segment bibliaonline.com.br uses ("1co", "atos", "jó").
	Slug string
	// Step is the STEPBible book code used in TAHOT/TAGNT ("1Co", "Jhn").
	Step string
	// Chapters is the chapter count in Portuguese (Almeida) versification.
	Chapters int
	// NT reports whether the book belongs to the New Testament.
	NT bool
}

// SingleChapter reports whether a bare number after the book name
// should be read as a verse ("Jd 3") rather than a chapter.
func (b *Book) SingleChapter() bool {
	return b.Chapters == 1
}

// All lists the books in canonical order.
var All = []Book{
	{Name: "Gênesis", Abbr: "Gn", Slug: "gn", Step: "Gen", Chapters: 50},
	{Name: "Êxodo", Abbr: "Êx", Slug: "ex", Step: "Exo", Chapters: 40},
	{Name: "Levítico", Abbr: "Lv", Slug: "lv", Step: "Lev", Chapters: 27},
	{Name: "Números", Abbr: "Nm", Slug: "nm", Step: "Num", Chapters: 36},
	{Name: "Deuteronômio", Abbr: "Dt", Slug: "dt", Step: "Deu", Chapters: 34},
	{Name: "Josué", Abbr: "Js", Slug: "js", Step: "Jos", Chapters: 24},
	{Name: "Juízes", Abbr: "Jz", Slug: "jz", Step: "Jdg", Chapters: 21},
	{Name: "Rute", Abbr: "Rt", Slug: "rt", Step: "Rut", Chapters: 4},
	{Name: "1 Samuel", Abbr: "1 Sm", Slug: "1sm", Step: "1Sa", Chapters: 31},
	{Name: "2 Samuel", Abbr: "2 Sm", Slug: "2sm", Step: "2Sa", Chapters: 24},
	{Name: "1 Reis", Abbr: "1 Rs", Slug: "1rs", Step: "1Ki", Chapters: 22},
	{Name: "2 Reis", Abbr: "2 Rs", Slug: "2rs", Step: "2Ki", Chapters: 25},
	{Name: "1 Crônicas", Abbr: "1 Cr", Slug: "1cr", Step: "1Ch", Chapters: 29},
	{Name: "2 Crônicas", Abbr: "2 Cr", Slug: "2cr", Step: "2Ch", Chapters: 36},
	{Name: "Esdras", Abbr: "Ed", Slug: "ed", Step: "Ezr", Chapters: 10},
	{Name: "Neemias", Abbr: "Ne", Slug: "ne", Step: "Neh", Chapters: 13},
	{Name: "Ester", Abbr: "Et", Slug: "et", Step: "Est", Chapters: 10},
	{Name: "Jó", Abbr: "Jó", Slug: "jó", Step: "Job", Chapters: 42},
	{Name: "Salmos", Abbr: "Sl", Slug: "sl", Step: "Psa", Chapters: 150},
	{Name: "Provérbios", Abbr: "Pv", Slug: "pv", Step: "Pro", Chapters: 31},
	{Name: "Eclesiastes", Abbr: "Ec", Slug: "ec", Step: "Ecc", Chapters: 12},
	{Name: "Cantares", Abbr: "Ct", Slug: "ct", Step: "Sng", Chapters: 8},
	{Name: "Isaías", Abbr: "Is", Slug: "is", Step: "Isa", Chapters: 66},
	{Name: "Jeremias", Abbr: "Jr", Slug: "jr", Step: "Jer", Chapters: 52},
	{Name: "Lamentações", Abbr: "Lm", Slug: "lm", Step: "Lam", Chapters: 5},
	{Name: "Ezequiel", Abbr: "Ez", Slug: "ez", Step: "Ezk", Chapters: 48},
	{Name: "Daniel", Abbr: "Dn", Slug: "dn", Step: "Dan", Chapters: 12},
	{Name: "Oseias", Abbr: "Os", Slug: "os", Step: "Hos", Chapters: 14},
	{Name: "Joel", Abbr: "Jl", Slug: "jl", Step: "Jol", Chapters: 3},
	{Name: "Amós", Abbr: "Am", Slug: "am", Step: "Amo", Chapters: 9},
	{Name: "Obadias", Abbr: "Ob", Slug: "ob", Step: "Oba", Chapters: 1},
	{Name: "Jonas", Abbr: "Jn", Slug: "jn", Step: "Jon", Chapters: 4},
	{Name: "Miqueias", Abbr: "Mq", Slug: "mq", Step: "Mic", Chapters: 7},
	{Name: "Naum", Abbr: "Na", Slug: "na", Step: "Nam", Chapters: 3},
	{Name: "Habacuque", Abbr: "Hc", Slug: "hc", Step: "Hab", Chapters: 3},
	{Name: "Sofonias", Abbr: "Sf", Slug: "sf", Step: "Zep", Chapters: 3},
	{Name: "Ageu", Abbr: "Ag", Slug: "ag", Step: "Hag", Chapters: 2},
	{Name: "Zacarias", Abbr: "Zc", Slug: "zc", Step: "Zec", Chapters: 14},
	{Name: "Malaquias", Abbr: "Ml", Slug: "ml", Step: "Mal", Chapters: 4},
	{Name: "Mateus", Abbr: "Mt", Slug: "mt", Step: "Mat", Chapters: 28, NT: true},
	{Name: "Marcos", Abbr: "Mc", Slug: "mc", Step: "Mrk", Chapters: 16, NT: true},
	{Name: "Lucas", Abbr: "Lc", Slug: "lc", Step: "Luk", Chapters: 24, NT: true},
	{Name: "João", Abbr: "Jo", Slug: "jo", Step: "Jhn", Chapters: 21, NT: true},
	{Name: "Atos", Abbr: "At", Slug: "atos", Step: "Act", Chapters: 28, NT: true},
	{Name: "Romanos", Abbr: "Rm", Slug: "rm", Step: "Rom", Chapters: 16, NT: true},
	{Name: "1 Coríntios", Abbr: "1 Co", Slug: "1co", Step: "1Co", Chapters: 16, NT: true},
	{Name: "2 Coríntios", Abbr: "2 Co", Slug: "2co", Step: "2Co", Chapters: 13, NT: true},
	{Name: "Gálatas", Abbr: "Gl", Slug: "gl", Step: "Gal", Chapters: 6, NT: true},
	{Name: "Efésios", Abbr: "Ef", Slug: "ef", Step: "Eph", Chapters: 6, NT: true},
	{Name: "Filipenses", Abbr: "Fp", Slug: "fp", Step: "Php", Chapters: 4, NT: true},
	{Name: "Colossenses", Abbr: "Cl", Slug: "cl", Step: "Col", Chapters: 4, NT: true},
	{Name: "1 Tessalonicenses", Abbr: "1 Ts", Slug: "1ts", Step: "1Th", Chapters: 5, NT: true},
	{Name: "2 Tessalonicenses", Abbr: "2 Ts", Slug: "2ts", Step: "2Th", Chapters: 3, NT: true},
	{Name: "1 Timóteo", Abbr: "1 Tm", Slug: "1tm", Step: "1Ti", Chapters: 6, NT: true},
	{Name: "2 Timóteo", Abbr: "2 Tm", Slug: "2tm", Step: "2Ti", Chapters: 4, NT: true},
	{Name: "Tito", Abbr: "Tt", Slug: "tt", Step: "Tit", Chapters: 3, NT: true},
	{Name: "Filemom", Abbr: "Fm", Slug: "fm", Step: "Phm", Chapters: 1, NT: true},
	{Name: "Hebreus", Abbr: "Hb", Slug: "hb", Step: "Heb", Chapters: 13, NT: true},
	{Name: "Tiago", Abbr: "Tg", Slug: "tg", Step: "Jas", Chapters: 5, NT: true},
	{Name: "1 Pedro", Abbr: "1 Pe", Slug: "1pe", Step: "1Pe", Chapters: 5, NT: true},
	{Name: "2 Pedro", Abbr: "2 Pe", Slug: "2pe", Step: "2Pe", Chapters: 3, NT: true},
	{Name: "1 João", Abbr: "1 Jo", Slug: "1jo", Step: "1Jn", Chapters: 5, NT: true},
	{Name: "2 João", Abbr: "2 Jo", Slug: "2jo", Step: "2Jn", Chapters: 1, NT: true},
	{Name: "3 João", Abbr: "3 Jo", Slug: "3jo", Step: "3Jn", Chapters: 1, NT: true},
	{Name: "Judas", Abbr: "Jd", Slug: "jd", Step: "Jud", Chapters: 1, NT: true},
	{Name: "Apocalipse", Abbr: "Ap", Slug: "ap", Step: "Rev", Chapters: 22, NT: true},
}

// extraAliases maps additional spellings (already normalized, see Key)
// to a book's Abbr. Abbreviations, slugs, full names and STEPBible codes
// are indexed automatically; these are the common variants on top.
var extraAliases = map[string]string{
	"exo":        "Êx",
	"cantico":    "Ct",
	"canticos":   "Ct",
	"cantaresde": "Ct",
	"ctc":        "Ct",
	"sal":        "Sl",
	"salmo":      "Sl",
	"prov":       "Pv",
	"ecl":        "Ec",
	"lam":        "Lm",
	"mat":        "Mt",
	"mar":        "Mc",
	"luc":        "Lc",
	"joao":       "Jo",
	"jao":        "Jo",
	"john":       "Jo",
	"acts":       "At",
	"apoc":       "Ap",
	"rev":        "Ap",
	"revelation": "Ap",
	"psalm":      "Sl",
	"psalms":     "Sl",
	"genesis":    "Gn",
	"romans":     "Rm",
	"hebrews":    "Hb",
	"james":      "Tg",
	"jude":       "Jd",
}

// accented holds the keys whose meaning depends on the accent, checked
// before accents are stripped: "jó" is Job, while "jo" is John.
var accented = map[string]string{
	"jó": "Jó",
}

var (
	byKey    = map[string]*Book{}
	byStep   = map[string]*Book{}
	bySlug   = map[string]*Book{}
	byCode   = map[string]*Book{}
	byNumber = map[*Book]int{}
)

func init() {
	byAbbr := map[string]*Book{}

	for i := range All {
		book := &All[i]
		byAbbr[book.Abbr] = book
		byStep[book.Step] = book
		bySlug[book.Slug] = book
		byCode[strings.ToUpper(book.Step)] = book
		byNumber[book] = i + 1

		for _, k := range []string{book.Abbr, book.Slug, book.Name, book.Step} {
			byKey[Key(k)] = book
		}
	}

	for k, abbr := range extraAliases {
		byKey[k] = byAbbr[abbr]
	}

	for k, abbr := range accented {
		byKey[k] = byAbbr[abbr]
	}
}

// Key normalizes a book name for lookup: lower case, no accents, no
// spaces or dots, and a leading Roman numeral (I, II, III) turned into
// its digit, so that "I Co", "1co" and "1 Coríntios" all compare equal
// on their prefix.
func Key(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	for roman, digit := range map[string]string{"iii ": "3", "ii ": "2", "i ": "1"} {
		if strings.HasPrefix(s, roman) {
			s = digit + s[len(roman):]

			break
		}
	}

	if _, ok := accented[s]; ok {
		return s
	}

	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

	plain, _, err := transform.String(t, s)
	if err != nil {
		plain = s
	}

	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '.' || r == '-' {
			return -1
		}

		return r
	}, plain)
}

// Lookup resolves a book name as typed by the reader. An exact match on
// any known spelling wins; otherwise the name is accepted when it is an
// unambiguous prefix (at least three characters) of exactly one full
// name, so "gene" or "romano" also work.
func Lookup(name string) (*Book, bool) {
	raw := strings.ToLower(strings.TrimSpace(name))
	if abbr, ok := accented[raw]; ok {
		return byKey[Key(abbr)], true
	}

	key := Key(name)
	if book, ok := byKey[key]; ok {
		return book, true
	}

	const minPrefix = 3
	if len(key) < minPrefix {
		return nil, false
	}

	var found *Book

	for i := range All {
		if strings.HasPrefix(Key(All[i].Name), key) {
			if found != nil {
				return nil, false
			}

			found = &All[i]
		}
	}

	return found, found != nil
}

// ByStep returns the book with the given STEPBible code ("Jhn").
func ByStep(code string) (*Book, bool) {
	book, ok := byStep[code]

	return book, ok
}

// BySlug returns the book with the given bibliaonline.com.br slug ("1co").
func BySlug(slug string) (*Book, bool) {
	book, ok := bySlug[slug]

	return book, ok
}

// Number returns the book's position in the canon, from 1 (Gênesis) to
// 66 (Apocalipse): the book numbers of USFM/Paratext, which the UBS data
// uses in its references.
func (b *Book) Number() int {
	return byNumber[b]
}

// ByNumber returns the book at a canonical position (1 to 66).
func ByNumber(n int) (*Book, bool) {
	if n < 1 || n > len(All) {
		return nil, false
	}

	return &All[n-1], true
}

// ByCode returns the book with the given USFM/Paratext code ("2SA",
// "JHN"), case-insensitively; STEPBible's codes are the same in title
// case.
func ByCode(code string) (*Book, bool) {
	book, ok := byCode[strings.ToUpper(strings.TrimSpace(code))]

	return book, ok
}
