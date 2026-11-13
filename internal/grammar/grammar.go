// Package grammar describes a word's grammar in words. Its input is the
// expansion that STEPBible's morphology legends (TEGMC for Greek, TEHMC
// for Hebrew) give each grammar code, "Function=Verb; Tense=Aorist;
// Voice=Active; ...", one item per segment of the word; its output is that
// expansion in the reader's language.
//
// Only grammatical terminology is translated here, from a closed
// vocabulary written by hand: no lexical content. A term missing from the
// table makes Describe fail, so the caller shows the raw code instead of
// a half-translated label.
package grammar

import (
	"regexp"
	"slices"
	"strings"
)

// hence matches the legend's derived notes, "(hence Action=Simple;
// Voice=Active)": they restate what the stem or form already says.
var hence = regexp.MustCompile(`\s*\(hence[^)]*\)`)

// Describe returns the description of a word whose segments have the
// given legend expansions, in lang ("pt" or "en"; any other language gets
// English). Segments are joined with " + " (preposition + noun); ok is
// false when a term has no translation.
func Describe(segments []string, lang string) (string, bool) {
	if len(segments) == 0 {
		return "", false
	}

	parts := make([]string, 0, len(segments))

	for _, seg := range segments {
		terms, ok := describeSegment(seg, lang)
		if !ok {
			return "", false
		}

		parts = append(parts, strings.Join(terms, ", "))
	}

	return strings.Join(parts, " + "), true
}

func describeSegment(attrs, lang string) ([]string, bool) {
	attrs = hence.ReplaceAllString(attrs, "")

	var out []string

	for item := range strings.SplitSeq(attrs, ";") {
		key, value, found := strings.Cut(item, "=")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)

		if !found || key == "" || value == "" {
			continue
		}

		if lang != "pt" {
			out = append(out, strings.ToLower(value))

			continue
		}

		term, ok := portuguese[key+"="+value]
		if !ok {
			return nil, false
		}

		// Only the legend's "Extra=" notes restate the part of speech
		// ("Negative Particle" + "Extra=Negative"); other repeated words
		// carry information, as the number of the possessor and of the
		// thing possessed in "2ª pessoa do plural, ..., plural".
		if term != "" && (key != "Extra" || !redundant(out, term)) {
			out = append(out, term)
		}
	}

	return out, len(out) > 0
}

// redundant reports whether a one-word term repeats what an earlier term
// already says: the legend tags a "Negative Particle" also
// "Extra=Negative", which would read "partícula negativa, negativo".
// The comparison ignores the gender ending (-o/-a).
func redundant(out []string, term string) bool {
	if strings.Contains(term, " ") {
		return slices.Contains(out, term)
	}

	stem := strings.TrimRight(term, "ao")

	for _, t := range out {
		if strings.Contains(t, stem) {
			return true
		}
	}

	return false
}

// portuguese maps "Key=Value" from both legends to Brazilian grammatical
// terms. Hebrew stem names follow the transliteration of Portuguese
// grammars (nifal, hifil); an empty term is a value that adds nothing in
// Portuguese and is left out.
var portuguese = map[string]string{
	// Part of speech.
	"Function=Adjective": "adjetivo",
	"Function=Adverb":    "advérbio",
	"Function=Adverb or adverb and particle combined":          "advérbio",
	"Function=Aramaic transliterated word":                     "palavra aramaica transliterada",
	"Function=Conjunction":                                     "conjunção",
	"Function=Correlative or Interrogative pronoun":            "pronome correlativo ou interrogativo",
	"Function=Correlative pronoun":                             "pronome correlativo",
	"Function=Definite article":                                "artigo definido",
	"Function=Demonstrative pronoun":                           "pronome demonstrativo",
	"Function=Demonstrative pronoun+Conjunction":               "pronome demonstrativo + conjunção",
	"Function=Indeclinable Noun of Other type":                 "substantivo indeclinável",
	"Function=Indeclinable Proper Noun":                        "nome próprio indeclinável",
	"Function=Indefinite pronoun":                              "pronome indefinido",
	"Function=Interjection":                                    "interjeição",
	"Function=Interrogative Particle":                          "partícula interrogativa",
	"Function=Interrogative pronoun":                           "pronome interrogativo",
	"Function=Negative Particle":                               "partícula negativa",
	"Function=Noun":                                            "substantivo",
	"Function=Particle":                                        "partícula",
	"Function=Particle or Disjunctive":                         "partícula",
	"Function=Personal pronoun":                                "pronome pessoal",
	"Function=Possessive pronoun":                              "pronome possessivo",
	"Function=Preposition":                                     "preposição",
	"Function=Pronoun":                                         "pronome",
	"Function=Reciprocal pronoun":                              "pronome recíproco",
	"Function=Reflexive pronoun":                               "pronome reflexivo",
	"Function=Relative pronoun":                                "pronome relativo",
	"Function=Suffix":                                          "sufixo",
	"Function=Verb":                                            "verbo",
	"Adj.Numb.=Indeclinable Numeral":                           "numeral indeclinável",
	"Name in Original language=Title Transcribed from Aramaic": "título transcrito do aramaico",
	"Name in Original language=Title Transcribed from Hebrew":  "título transcrito do hebraico",
	"Original language=Title Transcribed from Aramaic":         "título transcrito do aramaico",
	"Original language=Transcribed from Aramaic":               "transcrito do aramaico",
	"Original language=Transcribed from Hebrew":                "transcrito do hebraico",

	// Greek verb.
	"Tense=Present":                    "presente",
	"Tense=Imperfect":                  "imperfeito",
	"Tense=Future":                     "futuro",
	"Tense=Aorist":                     "aoristo",
	"Tense=Perfect":                    "perfeito",
	"Tense=Pluperfect":                 "mais-que-perfeito",
	"Tense=2nd Present":                "presente segundo",
	"Tense=2nd Future":                 "futuro segundo",
	"Tense=2nd Aorist":                 "aoristo segundo",
	"Tense=2nd Perfect":                "perfeito segundo",
	"Tense=2nd Pluperfect":             "mais-que-perfeito segundo",
	"Tense=indefinite tense":           "tempo indefinido",
	"Voice=Active":                     "voz ativa",
	"Voice=Middle":                     "voz média",
	"Voice=Passive":                    "voz passiva",
	"Voice=Middle or Passive":          "voz média ou passiva",
	"Voice=Middle Deponent":            "voz média depoente",
	"Voice=Passive Deponent":           "voz passiva depoente",
	"Voice=Middle or Passive Deponent": "voz média ou passiva depoente",
	"Voice=impersonal active":          "voz ativa impessoal",
	"Voice=indefinite voice":           "voz indefinida",
	"Mood=Indicative":                  "indicativo",
	"Mood=Subjunctive":                 "subjuntivo",
	"Mood=Optative":                    "optativo",
	"Mood=Imperative":                  "imperativo",

	// Greek nominals.
	"Case=Nominative": "nominativo",
	"Case=Genitive":   "genitivo",
	"Case=Dative":     "dativo",
	"Case=Accusative": "acusativo",
	"Case=Vocative":   "vocativo",
	"Gender=Neuter":   "neutro",

	"Extra=Abbreviated":              "forma abreviada",
	"Extra=Abbreviated Numeral":      "numeral abreviado",
	"Extra=Aeolic":                   "forma eólica",
	"Extra=Apocopated form":          "forma apocopada",
	"Extra=Attic Greek form":         "forma ática",
	"Extra=Comparative":              "comparativo",
	"Extra=Contracted form":          "forma contraída",
	"Extra=Indeclinable Letter":      "letra (indeclinável)",
	"Extra=Interrogative":            "interrogativo",
	"Extra=IRRegular or impure form": "forma irregular",
	"Extra=Negative":                 "negativo",
	"Extra=Numeral":                  "numeral",
	"Extra=Superlative":              "superlativo",
	"Extra=Transitive":               "transitivo",

	"Name type=Gentilic":                             "gentílico",
	"Name type=Individual":                           "nome de pessoa",
	"Name type=Individual Gentilic":                  "gentílico de pessoa",
	"Name type=Location":                             "nome de lugar",
	"Name type=Location Gentilic":                    "gentílico de lugar",
	"Name type=Person Gentilic":                      "gentílico de pessoa",
	"Name type=Person name Transcribed from Aramaic": "nome de pessoa transcrito do aramaico",
	"Name type=Title":                                "título",
	"Name type=Title Gentilic":                       "gentílico de título",
	// TEGMC writes "Type" only for A-DSF-T; every other "-T" code is
	// "Title", so it is read as a typo.
	"Name type=Type": "título",

	// Shared by both legends.
	"Form=Infinitive":      "infinitivo",
	"Form=Participle":      "particípio",
	"Gender=Masculine":     "masculino",
	"Gender=Feminine":      "feminino",
	"Gender=Either gender": "masculino ou feminino",
	"Number=Singular":      "singular",
	"Number=Plural":        "plural",
	"Number=Dual":          "dual",
	"Person=1st":           "1ª pessoa",
	"Person=1st Person":    "1ª pessoa",
	"Person=2nd":           "2ª pessoa",
	"Person=2nd Person":    "2ª pessoa",
	"Person=2nd Singular":  "2ª pessoa do singular",
	"Person=2nd Plural":    "2ª pessoa do plural",
	"Person=3rd":           "3ª pessoa",
	"Person=First":         "1ª pessoa",
	"Person=Second":        "2ª pessoa",
	"Person=Third":         "3ª pessoa",

	// Hebrew and Aramaic: form of the word.
	"Form=Common":                     "comum",
	"Form=Proper":                     "próprio",
	"Form=Gentilic":                   "gentílico",
	"Form=Title":                      "título",
	"Form=Personal":                   "pessoal",
	"Form=Demonstrative":              "demonstrativo",
	"Form=Interrogative":              "interrogativo",
	"Form=Relative":                   "relativo",
	"Form=Negative":                   "negação",
	"Form=Conditional":                "condicional",
	"Form=Interjection":               "interjeição",
	"Form=Numerical":                  "numeral cardinal",
	"Form=Numerical position":         "numeral ordinal",
	"Form=Object indicator":           "marca de objeto direto",
	"Form=Directional":                "he direcional",
	"Form=Paragogic Hé":               "he paragógico",
	"Form=Paragogic Nun":              "nun paragógico",
	"Form=Definite":                   "com artigo",
	"Form=Definite article (Hebrew)":  "artigo definido",
	"Form=Definite article (Aramaic)": "artigo definido (aramaico)",
	"Form=Consecutive":                "consecutiva",
	"Form=Perfect":                    "perfeito",
	"Form=Imperfect":                  "imperfeito",
	"Form=Consecutive Perfect":        "perfeito consecutivo",
	"Form=Consecutive Imperfect":      "imperfeito consecutivo",
	"Form=Conjunction+Imperfect":      "imperfeito com conjunção",
	"Form=Imperative":                 "imperativo",
	"Form=Participle passive":         "particípio passivo",
	"Gender=Location":                 "lugar",
	"Gender=Title":                    "título",
	"State=Absolute":                  "absoluto",
	"State=Construct":                 "construto",
	"State=Definite":                  "determinado",

	// Hebrew stems.
	"Stem=Qal":        "qal",
	"Stem=Niphal":     "nifal",
	"Stem=Piel":       "piel",
	"Stem=Pual":       "pual",
	"Stem=Hiphil":     "hifil",
	"Stem=Hophal":     "hofal",
	"Stem=Hithpael":   "hitpael",
	"Stem=Hitpael":    "hitpael",
	"Stem=Polal":      "polal",
	"Stem=Nithpael":   "nitpael",
	"Stem=Hothpaal":   "hotpaal",
	"Stem=Hishtaphel": "hishtafel",
	"Stem=Tiphil":     "tifil",
	// Aramaic stems.
	"Stem=Peal":      "peal",
	"Stem=Peil":      "peil",
	"Stem=Pael":      "pael",
	"Stem=Aphel":     "afel",
	"Stem=Haphel":    "hafel",
	"Stem=Shaphel":   "shafel",
	"Stem=Hitpaal":   "hitpaal",
	"Stem=Hitpeel":   "hitpeel",
	"Stem=Ishtaphel": "ishtafel",
}
