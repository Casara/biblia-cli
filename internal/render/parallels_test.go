package render_test

import (
	"strings"
	"testing"

	"github.com/casara/biblia-cli/internal/lookup"
	"github.com/casara/biblia-cli/internal/render"
)

func executeData(t *testing.T, format string, d render.Data) string {
	t.Helper()

	tpls, err := render.NewTemplates(map[string]string{
		"lista": `{{.Ref}}{{range .Parallels}} | {{.Kind}}: {{.Ref}}{{end}}`,
		"de":    `{{.Ref}}{{with .ParallelOf}} ← {{.Ref}} ({{.Kind}}){{end}}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	var b strings.Builder

	err = tpls.ExecuteData(&b, format, d)
	if err != nil {
		t.Fatal(err)
	}

	return b.String()
}

func TestNewParallelOf(t *testing.T) {
	cases := map[string]string{
		render.KindQuotedInNT: "cita Hc 2.4",
		render.KindQuotesOT:   "citado em Hc 2.4",
		render.KindParallel:   "paralelo de Hc 2.4",
	}

	for kind, want := range cases {
		if got := render.NewParallelOf("Hc 2.4", kind).Relation; got != want {
			t.Errorf("%s: relation %q, want %q", kind, got, want)
		}
	}
}

func TestFormatParallels(t *testing.T) {
	items := []render.Parallel{
		{Ref: "Rm 1.17", Kind: render.KindQuotedInNT},
		{Ref: "Sl 18.2", Kind: render.KindParallel},
		{Ref: "Gl 3.11", Kind: render.KindQuotedInNT},
	}

	want := "Citado no NT: Rm 1.17 · Gl 3.11; paralelo: Sl 18.2"
	if got := render.FormatParallels(items); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if got := render.FormatParallels(nil); got != "" {
		t.Errorf("empty list: got %q", got)
	}
}

func TestBuiltinsCarryParallels(t *testing.T) {
	origin := render.NewData(result(t, "Hc 2.4", lookup.Verse{Number: 4, Text: "o justo viverá pela sua fé"}))
	origin.Parallels = []render.Parallel{{Ref: "Rm 1.17", Kind: render.KindQuotedInNT}}

	quote := render.NewData(result(t, "Rm 1.17", lookup.Verse{Number: 17, Text: "O justo viverá por fé"}))
	quote.ParallelOf = render.NewParallelOf("Hc 2.4", render.KindQuotedInNT)

	cases := []struct {
		format string
		data   render.Data
		want   string
	}{
		// The comments stay out of the PDF and leave "// conferido"
		// right above #bref, where the lesson project's checker reads it.
		{"typ", origin, "#bref(\"Hc 2.4\", [«o justo viverá pela sua fé»])\n// Citado no NT: Rm 1.17"},
		{"typ", quote, "// paralelo: Rm 1.17 cita Hc 2.4\n// conferido:"},
		{"md", origin, "\n\n*Citado no NT: Rm 1.17*"},
		{"md", quote, "> — Rm 1.17 (ARA) · *cita Hc 2.4*"},
		{"linha", quote, "O justo viverá por fé (Rm 1.17, ARA; cita Hc 2.4)"},
		{"lista", origin, "Hc 2.4 | citado no NT: Rm 1.17"},
		{"de", quote, "Rm 1.17 ← Hc 2.4 (citado no NT)"},
	}

	for _, c := range cases {
		if got := executeData(t, c.format, c.data); !strings.Contains(got, c.want) {
			t.Errorf("%s: %q does not contain %q", c.format, got, c.want)
		}
	}
}

func TestBuiltinsWithoutParallelsUnchanged(t *testing.T) {
	d := render.NewData(result(t, "Jo 3.16", lookup.Verse{Number: 16, Text: "Porque Deus amou"}))

	for format, want := range map[string]string{
		"linha": "Porque Deus amou (Jo 3.16, ARA)",
		"typ":   "// conferido: ARA · bibliaonline.com.br/ara · 2026-11-12\n#bref(\"Jo 3.16\", [«Porque Deus amou»])",
	} {
		if got := executeData(t, format, d); got != want {
			t.Errorf("%s: got %q, want %q", format, got, want)
		}
	}
}
