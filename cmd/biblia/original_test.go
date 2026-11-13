package main

import (
	"errors"
	"testing"

	"github.com/casara/biblia-cli/internal/config"
	"github.com/casara/biblia-cli/internal/ubs"
)

func TestDictionaryChain(t *testing.T) {
	cfg := config.Default()

	chain, err := dictionaryChain(cfg)
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(chain))
	for _, src := range chain {
		if src.step {
			got = append(got, stepSource)
		} else {
			got = append(got, src.dict.ID)
		}
	}

	want := []string{"ubs-hebraico-pt", "ubs-grego-es", stepSource}
	if len(got) != len(want) {
		t.Fatalf("chain = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("chain = %v, want %v", got, want)
		}
	}

	cfg.Dictionaries = []string{"ubs-grego-pt"}

	_, err = dictionaryChain(cfg)
	if !errors.Is(err, ubs.ErrUnknownDictionary) || !errors.Is(err, errUsage) {
		t.Errorf("unknown id: err = %v", err)
	}
}

func TestPickGloss(t *testing.T) {
	senses := []ubs.Sense{
		{Glosses: []string{"bondade", "lealdade"}, AI: true},
		{Glosses: []string{"favor"}},
		{Glosses: []string{"vergonha"}},
		{Glosses: []string{"quarto sentido"}},
	}

	gloss, ai := pickGloss(senses)
	if gloss != "bondade / favor / vergonha" || !ai {
		t.Errorf("no sense in the verse: got %q, %v", gloss, ai)
	}

	senses[1].InVerse = true

	gloss, ai = pickGloss(senses)
	if gloss != "favor" || ai {
		t.Errorf("one sense in the verse: got %q, %v", gloss, ai)
	}
}
