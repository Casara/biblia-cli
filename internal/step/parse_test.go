package step_test

import (
	"testing"

	"github.com/casara/biblia-cli/internal/step"
)

// Rows copied from the STEPBible files (tabs preserved).
const (
	hebrewRow = "Psa.51.1(51.3)#01=L\tחָנֵּ֣/נִי\tcho.ne./ni\tshow favor to/ me\t{H2603A}/H9030\tHVqv2ms/Sp1bs" +
		"\t\t\tH2603A\t\t\t{H2603A=חָנַן=be gracious}/H9030=Os1c=me\t\t\t\t\t"
	hebrewPrefixRow = "Job.1.1#03=L\tבְ/אֶֽרֶץ\\־\tve.'e.retz-\tin/ [the] land of\tH9003/{H0776G}\\H9014\tHR/Ncfsc" +
		"\t\t\tH0776G\t\t\tH9003=ב=in/{H0776G=אֶרֶץ=: country;_planet»land:2_country}\\H9014=־=link\t\t\t\t\t"
	greekRow = "Jhn.3.16#03=NKO\tἠγάπησεν (ēgapēsen)\tloved\tG0025=V-AAI-3S\tἀγαπάω=to love" +
		"\tNA28+NA27+Tyn+SBL+WH+Treg+TR+Byz\t\t\tamó\tto love\t#03\tG0025\t\t\t\t\t"
	greekVariantRow = "Mat.17.21#01=KO\tΤοῦτο (Touto)\tthis\tG3778=D-NSN\tοὗτος=this/he/she/it" +
		"\tTreg+TR+Byz\t\t\testo\tthis\t#01»06:G1607\tG3778\tG5124\t^ \t\t\t"
	lexiconRow = "H2603a\tH2603A =\tH2603A\tחָנַן\tcha.nan\tH:V\tbe gracious\t1) to be gracious, show favour"
	meaningRow = "H5162\tH5162H = a Meaning of\tH5162G\tנָחַם\tna.cham\tH:V\tto be sorry: relent\t1) to be sorry"
)

func TestParseHebrewRow(t *testing.T) {
	w, ok := step.ParseHebrewRow(hebrewRow)
	if !ok {
		t.Fatal("row not parsed")
	}

	want := step.Word{
		Book: "Psa", Chapter: 51, Verse: 1, Pos: 1, AltRef: "51.3", Type: "L",
		Original: "חָנֵּ֣נִי", Translit: "choneni", English: "show favor to me",
		Strong: "H2603A", Grammar: "HVqv2ms/Sp1bs", Gloss: "be gracious",
	}

	if w != want {
		t.Errorf("got  %+v\nwant %+v", w, want)
	}

	w, ok = step.ParseHebrewRow(hebrewPrefixRow)
	if !ok {
		t.Fatal("prefix row not parsed")
	}

	if w.Strong != "H0776G" || w.Gloss != "country; planet" || w.Original != "בְאֶֽרֶץ־" {
		t.Errorf("prefix row: %+v", w)
	}
}

func TestParseGreekRow(t *testing.T) {
	w, ok := step.ParseGreekRow(greekRow)
	if !ok {
		t.Fatal("row not parsed")
	}

	if w.Original != "ἠγάπησεν" || w.Translit != "ēgapēsen" || w.Strong != "G0025" ||
		w.Grammar != "V-AAI-3S" || w.Gloss != "to love" || !w.InCriticalText() {
		t.Errorf("got %+v", w)
	}

	w, ok = step.ParseGreekRow(greekVariantRow)
	if !ok {
		t.Fatal("variant row not parsed")
	}

	if w.InCriticalText() || w.Editions != "Treg+TR+Byz" {
		t.Errorf("variant row: %+v", w)
	}
}

func TestParseLexiconRow(t *testing.T) {
	e, ok := step.ParseLexiconRow(lexiconRow)
	if !ok {
		t.Fatal("row not parsed")
	}

	if e.Strong != "H2603a" || e.EStrong != "H2603A" || e.Gloss != "be gracious" || e.Translit != "cha.nan" {
		t.Errorf("got %+v", e)
	}

	// The second meaning of nacham points at the first one; its own
	// number is H5162H, not the target's H5162G.
	e, ok = step.ParseLexiconRow(meaningRow)
	if !ok || e.EStrong != "H5162H" || e.Relation != "a Meaning of" || e.Target != "H5162G" {
		t.Errorf("related entry: got %+v", e)
	}

	if e, _ := step.ParseLexiconRow(lexiconRow); e.Relation != "" || e.Target != "" {
		t.Errorf("standalone entry has a relation: %+v", e)
	}

	if _, ok := step.ParseLexiconRow("TBESH - Translators Brief lexicon"); ok {
		t.Error("header line parsed as an entry")
	}
}

func TestNormalizeStrong(t *testing.T) {
	cases := []struct{ in, want string }{
		{"g25", "G0025"},
		{"G0025", "G0025"},
		{"H7451b", "H7451b"},
		{" h5162", "H5162"}, // surrounding spaces are tolerated
	}

	for _, tc := range cases {
		got, ok := step.NormalizeStrong(tc.in)
		if !ok || got != tc.want {
			t.Errorf("NormalizeStrong(%q) = %q, %v; want %q", tc.in, got, ok, tc.want)
		}
	}

	if _, ok := step.NormalizeStrong("X12"); ok {
		t.Error("X12 accepted")
	}
}

func TestParseLegendRow(t *testing.T) {
	cases := []struct {
		line   string
		greek  bool
		code   string
		wantOK bool
	}{
		{"V-AAI-3S\tFunction=Verb; Tense=Aorist", true, "G:V-AAI-3S", true},
		{"ADV\tFunction=Adverb", true, "G:ADV", true},
		{"HVqp3ms\tFunction=Verb ; Stem=Qal", false, "H:HVqp3ms", true},
		{"HR\tFunction=Preposition ", false, "H:HR", true},
		// Brief lexicon codes and prose lines are not legend rows.
		{"H:A-F\tfemale\tHebrew Adjective (Feminine)", false, "", false},
		{"G:N-M\tFunction=Noun", true, "", false},
		{"Code\tExample in English\tMeaning", true, "", false},
	}

	for _, c := range cases {
		code, _, ok := step.ParseLegendRow(c.line, c.greek)
		if ok != c.wantOK || code != c.code {
			t.Errorf("ParseLegendRow(%q) = %q, %v; want %q, %v", c.line, code, ok, c.code, c.wantOK)
		}
	}
}
