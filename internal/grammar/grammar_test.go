package grammar_test

import (
	"testing"

	"github.com/casara/biblia-cli/internal/grammar"
)

// Expansions copied from TEGMC and TEHMC.
const (
	greekVerb   = "Function=Verb; Tense=Aorist; Voice=Active; Mood=Indicative; Person=3rd; Number=Singular"
	negParticle = "Function=Negative Particle; Extra=Negative"
	possessive  = "Function=Possessive pronoun; Person=2nd Plural; Case=Accusative; Number=Plural; Gender=Feminine"
	hebPrep     = "Function=Preposition "
	hebNoun     = "Function=Noun ; Form=Common ; Gender=Feminine; Number=Singular; State=Absolute"
	hebVerb     = "Function=Verb ; Stem=Qal (hence Action=Simple; Voice=Active); " +
		"Form=Perfect (hence Tense=Past/present; Mood=Indicative); Person=Third; Gender=Masculine; Number=Singular"
)

func TestDescribe(t *testing.T) {
	cases := []struct {
		name     string
		segments []string
		lang     string
		want     string
	}{
		{"greek verb", []string{greekVerb}, "pt", "verbo, aoristo, voz ativa, indicativo, 3ª pessoa, singular"},
		{"repeated term", []string{negParticle}, "pt", "partícula negativa"},
		{"possessor and possessed", []string{possessive}, "pt",
			"pronome possessivo, 2ª pessoa do plural, acusativo, plural, feminino"},
		{"hebrew prefix", []string{hebPrep, hebNoun}, "pt", "preposição + substantivo, comum, feminino, singular, absoluto"},
		{"hence notes dropped", []string{hebVerb}, "pt", "verbo, qal, perfeito, 3ª pessoa, masculino, singular"},
		{"english", []string{greekVerb}, "en", "verb, aorist, active, indicative, 3rd, singular"},
	}

	for _, c := range cases {
		got, ok := grammar.Describe(c.segments, c.lang)
		if !ok || got != c.want {
			t.Errorf("%s: got %q, %v; want %q", c.name, got, ok, c.want)
		}
	}
}

func TestDescribeRefusesUnknownTerms(t *testing.T) {
	for _, segs := range [][]string{
		nil,
		{"Function=Verb; Tense=Gnomic"},
		{hebPrep, "Function=Mystery"},
	} {
		if got, ok := grammar.Describe(segs, "pt"); ok {
			t.Errorf("Describe(%q) = %q, want failure", segs, got)
		}
	}
}
