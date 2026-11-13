package render_test

import (
	"strings"
	"testing"
	"time"

	"github.com/casara/biblia-cli/internal/lookup"
	"github.com/casara/biblia-cli/internal/ref"
	"github.com/casara/biblia-cli/internal/render"
)

func result(t *testing.T, s string, verses ...lookup.Verse) lookup.Result {
	t.Helper()

	rs, err := ref.Parse(s)
	if err != nil {
		t.Fatal(err)
	}

	return lookup.Result{
		Ref: rs[0], Bible: "ara", Verses: verses,
		URL:       lookup.PassageURL("ara", rs[0]),
		FetchedAt: time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC),
	}
}

func execute(t *testing.T, format string, res lookup.Result) string {
	t.Helper()

	tpls, err := render.NewTemplates(nil)
	if err != nil {
		t.Fatal(err)
	}

	var b strings.Builder
	err = tpls.Execute(&b, format, res)
	if err != nil {
		t.Fatal(err)
	}

	return b.String()
}

func TestTypMatchesLessonProjectConvention(t *testing.T) {
	res := result(t, "1 Tm 2.5", lookup.Verse{Number: 5, Text: "Porquanto há um só Deus"})

	want := "// conferido: ARA · bibliaonline.com.br/ara · 2026-11-12\n" +
		`#bref("1 Tm 2.5", [«Porquanto há um só Deus»])`

	if got := execute(t, "typ", res); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestTypEscapesAndWarnsWhenPartial(t *testing.T) {
	res := result(t, "Rm 16.24-25", lookup.Verse{Number: 24, Text: "[A graça # esteja]"})
	res.Partial = true
	res.Missing = []int{25}

	got := execute(t, "typ", res)

	if !strings.HasPrefix(got, "// ATENÇÃO: trecho parcial, faltam 25") {
		t.Errorf("no partial warning:\n%s", got)
	}

	if !strings.Contains(got, `«\[A graça \# esteja\]»`) {
		t.Errorf("not escaped:\n%s", got)
	}
}

func TestPartialResultNarrowsTheReference(t *testing.T) {
	res := result(t, "Gn 3.1-6",
		lookup.Verse{Number: 1, Text: "a"}, lookup.Verse{Number: 2, Text: "b"},
		lookup.Verse{Number: 3, Text: "c"}, lookup.Verse{Number: 4, Text: "d"},
		lookup.Verse{Number: 5, Text: "e"},
	)
	res.Partial = true
	res.Missing = []int{6}

	d := render.NewData(res)
	if d.Ref != "Gn 3.1-5" || d.Requested != "Gn 3.1-6" {
		t.Errorf("Ref = %q, Requested = %q", d.Ref, d.Requested)
	}

	if got := execute(t, "typ", res); !strings.Contains(got, `#bref("Gn 3.1-5", `) {
		t.Errorf("typ cites more than it holds:\n%s", got)
	}
}

func TestAbsentVersesLeftOutOfText(t *testing.T) {
	res := result(t, "Mt 17.20-22",
		lookup.Verse{Number: 20, Text: "a"},
		lookup.Verse{Number: 21, Absent: true},
		lookup.Verse{Number: 22, Text: "b"},
	)

	d := render.NewData(res)
	if d.Text != "a b" || d.Absent != "21" {
		t.Errorf("got %+v", d)
	}
}

func TestUserTemplates(t *testing.T) {
	tpls, err := render.NewTemplates(map[string]string{"cit": "{{.Ref}}: {{.Text}}"})
	if err != nil {
		t.Fatal(err)
	}

	var b strings.Builder
	err = tpls.Execute(&b, "cit", result(t, "Jo 3.16", lookup.Verse{Number: 16, Text: "x"}))
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != "Jo 3.16: x" {
		t.Errorf("got %q", b.String())
	}

	_, err = render.NewTemplates(map[string]string{"texto": "x"})
	if err == nil {
		t.Error("a template named after the terminal format was accepted")
	}
}

func TestCleanDefinition(t *testing.T) {
	got := render.CleanDefinition("<b>love</b>, goodwill<BR />__1. <i>Of men</i>&amp; God")
	if got != "love, goodwill\n__1. Of men& God" {
		t.Errorf("got %q", got)
	}
}
