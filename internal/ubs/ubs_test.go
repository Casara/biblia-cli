package ubs_test

import (
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/casara/biblia-cli/internal/ubs"
)

// hebrewJSON follows the UBS file layout (BOM included); the definitions
// and references are cut down from the real entries for חֶסֶד and מֶלֶךְ.
const hebrewJSON = "\uFEFF" + `[
{"MainId":"002617000000000","Lemma":"חֶסֶד","StrongCodes":["H2617a"],
 "BaseForms":[{"LEXMeanings":[
  {"LEXDomains":[{"Domain":"Atitude"}],
   "LEXSenses":[{"LanguageCode":"pt","AIGenerated":true,"DefinitionShort":"= estado em que {L:humanos<SDBH:אָדָם>} são leais{N:001}","DefinitionLong":"","Glosses":["bondade","lealdade"]}],
   "LEXReferences":["01903200300012{N:001}","01905100300004"]},
  {"LEXDomains":[],
   "LEXSenses":[{"LanguageCode":"pt","AIGenerated":false,"DefinitionShort":"= um evento que traz desgraça","DefinitionLong":"","Glosses":["vergonha"]}],
   "LEXReferences":["00302001700030"]}]}]},
{"MainId":"004428000000000","Lemma":"מֶלֶךְ","StrongCodes":["H4428","A4430"],
 "BaseForms":[{"LEXMeanings":[
  {"LEXDomains":[],"LEXSenses":[{"LanguageCode":"pt","AIGenerated":false,"DefinitionShort":"= rei","DefinitionLong":"","Glosses":["rei"]}],
   "LEXReferences":["02700200400008"]}]}]},
{"MainId":"x","Lemma":"sem número","StrongCodes":[],"BaseForms":[]}
]`

const parallelsXML = "\uFEFF" + `<?xml version="1.0" encoding="utf-8"?>
<Passages>
  <Passage>
    <Verse HEB="0020052212222">HAB 2:4</Verse>
    <Verse GRK="02000005221222230000000">ROM 1:17</Verse>
    <Verse GRK="0000">GAL 3:11</Verse>
  </Passage>
  <Passage>
    <Verse GRK="0000">ROM 1:17</Verse>
    <Verse GRK="0000">GAL 3:11</Verse>
  </Passage>
  <Passage>
    <Verse HEB="2222">PSA 40:14-18,20</Verse>
    <Verse HEB="2222">PSA 70:2-6</Verse>
  </Passage>
</Passages>`

func TestParseDictionary(t *testing.T) {
	var got []ubs.Sense

	err := ubs.ParseDictionary(strings.NewReader(hebrewJSON), "ubs-hebraico-pt", func(s ubs.Sense) error {
		got = append(got, s)

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 3 {
		t.Fatalf("%d senses, want 3 (the entry without a Strong number is skipped)", len(got))
	}

	first := got[0]
	if first.Definition != "estado em que humanos são leais" {
		t.Errorf("definition = %q", first.Definition)
	}

	if !first.AI || got[1].AI {
		t.Error("AI flag not carried per sense")
	}

	wantRefs := []ubs.Verse{{Book: 19, Chapter: 32, Verse: 3}, {Book: 19, Chapter: 51, Verse: 3}}
	if !reflect.DeepEqual(first.Refs, wantRefs) {
		t.Errorf("refs = %v, want %v", first.Refs, wantRefs)
	}

	if want := []string{"H4428", "H4430"}; !reflect.DeepEqual(got[2].Strongs, want) {
		t.Errorf("strongs = %v, want %v (Aramaic A filed under H)", got[2].Strongs, want)
	}
}

func TestParseParallels(t *testing.T) {
	groups, err := ubs.ParseParallels(strings.NewReader(parallelsXML))
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 3 {
		t.Fatalf("%d groups, want 3", len(groups))
	}

	psa := groups[2][0]
	want := []ubs.Span{{From: 14, To: 18}, {From: 20, To: 20}}

	if psa.Book != "PSA" || psa.Chapter != 40 || !reflect.DeepEqual(psa.Spans, want) || psa.Greek {
		t.Errorf("PSA 40:14-18,20 parsed as %+v", psa)
	}

	if !groups[0][1].Greek {
		t.Error("ROM 1:17 should be Greek")
	}
}

func TestParseParallelsRejectsUnknownShape(t *testing.T) {
	bad := `<Passages><Passage><Verse HEB="0">GEN 1:1-2:3</Verse></Passage></Passages>`

	_, err := ubs.ParseParallels(strings.NewReader(bad))
	if err == nil {
		t.Error("a cross-chapter reference should be reported, not skipped")
	}
}

func TestDefaults(t *testing.T) {
	cases := map[string][]string{
		"pt": {"ubs-hebraico-pt", "ubs-grego-es"},
		"es": {"ubs-hebraico-es", "ubs-grego-es"},
		"de": {"ubs-hebraico-en", "ubs-grego-en"},
	}

	for lang, want := range cases {
		if got := ubs.Defaults(lang); !reflect.DeepEqual(got, want) {
			t.Errorf("Defaults(%q) = %v, want %v", lang, got, want)
		}
	}
}

// newStore indexes the fixtures into a temporary database.
func newStore(t *testing.T) (*ubs.Store, ubs.Dictionary) {
	t.Helper()

	dir := t.TempDir()
	dict, _ := ubs.Find("ubs-hebraico-pt")

	writeFile(t, filepath.Join(dir, path.Base(dict.Path)), hebrewJSON)
	writeFile(t, filepath.Join(dir, path.Base(ubs.ParallelsPath)), parallelsXML)

	dbPath := filepath.Join(dir, "ubs.db")
	in := &ubs.Installer{Log: io.Discard}

	err := in.Index(context.Background(), dir, dbPath, "abc123", []ubs.Dictionary{dict})
	if err != nil {
		t.Fatal(err)
	}

	store, err := ubs.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store, dict
}

func TestStoreSenses(t *testing.T) {
	ctx := context.Background()
	store, dict := newStore(t)

	// Sl 51.3 in the Hebrew numbering is the translations' Sl 51.1.
	senses, err := store.Senses(ctx, dict.ID, "H2617A", &ubs.Verse{Book: 19, Chapter: 51, Verse: 3})
	if err != nil {
		t.Fatal(err)
	}

	if len(senses) != 2 || !senses[0].InVerse || senses[1].InVerse {
		t.Errorf("senses = %+v, want 2 with only the first in the verse", senses)
	}

	aramaic, err := store.Senses(ctx, dict.ID, "H4430", nil)
	if err != nil || len(aramaic) != 1 || aramaic[0].Glosses[0] != "rei" {
		t.Errorf("H4430 = %+v, %v", aramaic, err)
	}

	found, err := store.SearchGloss(ctx, []string{dict.ID}, "lealdade", 10)
	if err != nil || !reflect.DeepEqual(found, []string{"H2617"}) {
		t.Errorf("SearchGloss = %v, %v", found, err)
	}

	installed, err := store.Installed(ctx)
	if err != nil || installed[dict.ID] != 3 {
		t.Errorf("Installed = %v, %v", installed, err)
	}
}

func TestStoreParallels(t *testing.T) {
	store, _ := newStore(t)

	parallels, err := store.Parallels(context.Background(), "ROM", 1, 17)
	if err != nil {
		t.Fatal(err)
	}

	refs := make([]string, 0, len(parallels))
	for _, p := range parallels {
		refs = append(refs, p.Book)
	}

	// GAL 3:11 is in both groups but listed once; ROM 1:17 itself is left out.
	if want := []string{"HAB", "GAL"}; !reflect.DeepEqual(refs, want) {
		t.Errorf("parallels of ROM 1:17 = %v, want %v", refs, want)
	}
}

func writeFile(t *testing.T, name, content string) {
	t.Helper()

	err := os.WriteFile(name, []byte(content), 0o600)
	if err != nil {
		t.Fatal(err)
	}
}
