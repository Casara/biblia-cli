package ref_test

import (
	"errors"
	"testing"

	"github.com/casara/biblia-cli/internal/ref"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in    string
		want  []string // String() of each ref
		query string   // Query() of the first ref
	}{
		{in: "jo 3.16", want: []string{"Jo 3.16"}, query: "João 3:16"},
		{in: "Jo 3:16", want: []string{"Jo 3.16"}},
		{in: "jó 1.1", want: []string{"Jó 1.1"}, query: "Jó 1:1"},
		{in: "job 1.1", want: []string{"Jó 1.1"}},
		{in: "1co13:4-7", want: []string{"1 Co 13.4-7"}, query: "1 Coríntios 13:4-7"},
		{in: "1 Co 13.4-7,9", want: []string{"1 Co 13.4-7,9"}},
		{in: "I Co 13.4", want: []string{"1 Co 13.4"}},
		{in: "II Rs 4.40", want: []string{"2 Rs 4.40"}},
		{in: "Rm 8", want: []string{"Rm 8"}},
		{in: "Sl 23.1-3, 6", want: []string{"Sl 23.1-3,6"}},
		{in: "Jd 3", want: []string{"Jd 1.3"}, query: "Judas 1:3"},
		{in: "Jd 1.3", want: []string{"Jd 1.3"}},
		{in: "Ob 1", want: []string{"Ob 1.1"}},
		{in: "Rm 5.8; 6.23", want: []string{"Rm 5.8", "Rm 6.23"}},
		{in: "Jo 3.16; Rm 5.8", want: []string{"Jo 3.16", "Rm 5.8"}},
		{in: "Êx 20.3", want: []string{"Êx 20.3"}},
		{in: "ex 20.3", want: []string{"Êx 20.3"}},
		{in: "genesis 1.1", want: []string{"Gn 1.1"}},
		{in: "Gênesis 1.1", want: []string{"Gn 1.1"}},
		{in: "romano 8.28", want: []string{"Rm 8.28"}},
		{in: "atos 2.38", want: []string{"At 2.38"}},
		{in: "Jhn 3.16", want: []string{"Jo 3.16"}},
		{in: "Ap 22.20–21", want: []string{"Ap 22.20-21"}},
	}

	for _, tc := range cases {
		got, err := ref.Parse(tc.in)
		if err != nil {
			t.Errorf("Parse(%q): %v", tc.in, err)

			continue
		}

		if len(got) != len(tc.want) {
			t.Errorf("Parse(%q) = %d refs, want %d", tc.in, len(got), len(tc.want))

			continue
		}

		for i, r := range got {
			if r.String() != tc.want[i] {
				t.Errorf("Parse(%q)[%d] = %q, want %q", tc.in, i, r.String(), tc.want[i])
			}
		}

		if tc.query != "" && got[0].Query() != tc.query {
			t.Errorf("Parse(%q).Query() = %q, want %q", tc.in, got[0].Query(), tc.query)
		}
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		in   string
		want error
	}{
		{in: "", want: ref.ErrEmpty},
		{in: "xyz 3.1", want: ref.ErrUnknownBook},
		{in: "fil 1.1", want: ref.ErrUnknownBook}, // Filipenses or Filemom
		{in: "Jo", want: ref.ErrSyntax},
		{in: "3.16", want: ref.ErrSyntax}, // no book to carry over
		{in: "Jo 3.18-16", want: ref.ErrSyntax},
		{in: "Jo 22.1", want: ref.ErrChapter},
		{in: "Jo 3.16-4.2", want: ref.ErrSyntax},
	}

	for _, tc := range cases {
		_, err := ref.Parse(tc.in)
		if !errors.Is(err, tc.want) {
			t.Errorf("Parse(%q) error = %v, want %v", tc.in, err, tc.want)
		}
	}
}

func TestVersesAndCompact(t *testing.T) {
	rs, err := ref.Parse("Sl 23.1-3,2,6")
	if err != nil {
		t.Fatal(err)
	}

	got := rs[0].Verses()
	want := []int{1, 2, 3, 6}

	if len(got) != len(want) {
		t.Fatalf("Verses() = %v, want %v", got, want)
	}

	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("Verses() = %v, want %v", got, want)
		}
	}

	if s := ref.FormatSpans(ref.Compact([]int{4, 5, 6, 9, 11, 12})); s != "4-6,9,11-12" {
		t.Errorf("Compact = %q", s)
	}
}
