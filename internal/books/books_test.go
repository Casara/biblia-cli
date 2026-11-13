package books_test

import (
	"strings"
	"testing"

	"github.com/casara/biblia-cli/internal/books"
)

// TestEverySpellingResolvesToItsOwnBook guards against an alias of one
// book shadowing another (the "jo"/"jó" kind of collision).
func TestEverySpellingResolvesToItsOwnBook(t *testing.T) {
	const canon = 66
	if len(books.All) != canon {
		t.Fatalf("%d books, want %d", len(books.All), canon)
	}

	for i := range books.All {
		want := &books.All[i]

		for _, s := range []string{want.Abbr, want.Slug, want.Name, want.Step} {
			got, ok := books.Lookup(s)
			if !ok || got != want {
				name := "<none>"
				if got != nil {
					name = got.Name
				}

				t.Errorf("Lookup(%q) = %s, want %s", s, name, want.Name)
			}
		}

		if b, ok := books.ByStep(want.Step); !ok || b != want {
			t.Errorf("ByStep(%q) failed", want.Step)
		}

		if b, ok := books.BySlug(want.Slug); !ok || b != want {
			t.Errorf("BySlug(%q) failed", want.Slug)
		}
	}
}

func TestAccentDecides(t *testing.T) {
	cases := map[string]string{
		"jo":   "João",
		"Jo":   "João",
		"jó":   "Jó",
		"Jó":   "Jó",
		"job":  "Jó",
		"jn":   "Jonas",
		"1jo":  "1 João",
		"i jo": "1 João",
		"fp":   "Filipenses",
		"fm":   "Filemom",
	}

	for in, want := range cases {
		got, ok := books.Lookup(in)
		if !ok || got.Name != want {
			t.Errorf("Lookup(%q) = %v, want %s", in, got, want)
		}
	}
}

// TestUSFMCodes checks that the STEPBible codes, upper-cased, are the
// USFM/Paratext book ids in canonical order, as the UBS data uses them
// (its parallel passages name books "2SA", its references number them).
func TestUSFMCodes(t *testing.T) {
	usfm := strings.Fields(`GEN EXO LEV NUM DEU JOS JDG RUT 1SA 2SA 1KI 2KI 1CH 2CH
		EZR NEH EST JOB PSA PRO ECC SNG ISA JER LAM EZK DAN HOS JOL AMO OBA JON
		MIC NAM HAB ZEP HAG ZEC MAL MAT MRK LUK JHN ACT ROM 1CO 2CO GAL EPH PHP
		COL 1TH 2TH 1TI 2TI TIT PHM HEB JAS 1PE 2PE 1JN 2JN 3JN JUD REV`)

	for i, code := range usfm {
		book, ok := books.ByCode(code)
		if !ok {
			t.Errorf("ByCode(%q) failed", code)

			continue
		}

		if book.Number() != i+1 {
			t.Errorf("%s: Number() = %d, want %d", code, book.Number(), i+1)
		}

		if n, ok := books.ByNumber(i + 1); !ok || n != book {
			t.Errorf("ByNumber(%d) is not %s", i+1, code)
		}
	}

	if _, ok := books.ByNumber(0); ok {
		t.Error("ByNumber(0) should fail")
	}
}
