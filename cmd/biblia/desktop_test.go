package main

import (
	"errors"
	"slices"
	"testing"

	"github.com/casara/biblia-cli/internal/bibliaonline"
)

func TestPlanSelection(t *testing.T) {
	tests := []struct {
		sel, mode, format string
		args              []string
		stdin             string
		copy, filter      bool
	}{
		{sel: "Jo 3.16", mode: modeShow, args: []string{"-f", "texto", "Jo 3.16"}},
		{sel: "Jo 3.16", mode: modeOriginal, args: []string{"-f", "texto", "-s", "Jo 3.16"}},
		{
			sel: "Jo 3.16\nRm 8.28", mode: modeShow,
			args: []string{"-f", "texto", "Jo 3.16; Rm 8.28"},
		},
		{
			sel: "Jo 3.16", mode: modeCopy,
			args: []string{"filtro"}, stdin: "Jo 3.16\n", copy: true, filter: true,
		},
		{
			sel: "Jo 3.16", mode: modeCopy, format: "md",
			args: []string{"filtro", "-f", "md"}, stdin: "Jo 3.16\n", copy: true, filter: true,
		},
		// Strong numbers open the lexicon, in every spelling the tool takes.
		{sel: "G5293", mode: modeShow, args: []string{"strong", "G5293", "-c"}},
		{sel: "g25", mode: modeOriginal, args: []string{"strong", "g25", "-c"}},
		{sel: "H7451b", mode: modeShow, args: []string{"strong", "H7451b", "-c"}},
		{sel: "H2603A", mode: modeCopy, args: []string{"strong", "H2603A"}, copy: true},
		// Not Strong numbers: books whose abbreviation starts with G or H.
		{sel: "Gn 1.1", mode: modeShow, args: []string{"-f", "texto", "Gn 1.1"}},
		{sel: "Hb 11", mode: modeShow, args: []string{"-f", "texto", "Hb 11"}},
	}

	for _, tt := range tests {
		plan, err := planSelection(tt.sel, tt.mode, tt.format)
		if err != nil {
			t.Errorf("%q %s: %v", tt.sel, tt.mode, err)

			continue
		}

		if !slices.Equal(plan.args, tt.args) || plan.stdin != tt.stdin ||
			plan.copy != tt.copy || plan.filter != tt.filter {
			t.Errorf("%q %s: got %+v", tt.sel, tt.mode, plan)
		}
	}

	_, err := planSelection("Jo 3.16", "imprimir", "")
	if !errors.Is(err, errUsage) {
		t.Errorf("unknown mode: err = %v", err)
	}
}

func TestPlanSelectionOffersVersionsForReferences(t *testing.T) {
	for _, tt := range []struct {
		sel, mode string
		want      bool
	}{
		{"Jo 3.16", modeShow, true},
		{"Jo 3.16", modeOriginal, true},
		{"Jo 3.16", modeCopy, false},
		{"G5293", modeShow, false},
	} {
		plan, err := planSelection(tt.sel, tt.mode, "")
		if err != nil {
			t.Fatal(err)
		}

		if plan.pickVersions != tt.want {
			t.Errorf("%q %s: pickVersions = %v, want %v", tt.sel, tt.mode, plan.pickVersions, tt.want)
		}
	}
}

func TestVersionChoices(t *testing.T) {
	bibles := []bibliaonline.Bible{
		{ID: "acf", Name: "Almeida Corrigida Fiel", Language: "Português"},
		{ID: "ara", Name: "Almeida Revista e Atualizada", Language: "Português"},
		{ID: "kjv", Name: "King James Version", Language: "English"},
		{ID: "nvi", Name: "Nova Versão Internacional", Language: "Português"},
	}

	// The current ones first, in their own order; an id the catalog lacks
	// is dropped.
	got := versionChoices(bibles, []string{"nvi", "kjv", "xyz"})

	ids := make([]string, 0, len(got))

	for _, choice := range got {
		mark := ""
		if choice.Checked {
			mark = "*"
		}

		ids = append(ids, mark+choice.ID)
	}

	if want := []string{"*nvi", "*kjv", "acf", "ara"}; !slices.Equal(ids, want) {
		t.Errorf("choices = %q, want %q", ids, want)
	}

	if got[1].Group != "English" {
		t.Errorf("language column = %q", got[1].Group)
	}
}
