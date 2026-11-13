package grammar_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/casara/biblia-cli/internal/grammar"
)

// TestEveryLegendCodeTranslates checks the Portuguese table against a
// real STEPBible index: every code of the tagged texts must be described
// without falling back to the raw code. It needs the downloaded data, so
// it runs only when BIBLIA_STEP_DB points at a step.db.
func TestEveryLegendCodeTranslates(t *testing.T) {
	path := os.Getenv("BIBLIA_STEP_DB")
	if path == "" {
		t.Skip("BIBLIA_STEP_DB not set")
	}

	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(context.Background(), `SELECT code, attrs FROM morph`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = rows.Close() }()

	var failed []string

	for rows.Next() {
		var code, attrs string

		err := rows.Scan(&code, &attrs)
		if err != nil {
			t.Fatal(err)
		}

		if _, ok := grammar.Describe([]string{attrs}, "pt"); !ok {
			failed = append(failed, code+" = "+attrs)
		}
	}

	err = rows.Err()
	if err != nil {
		t.Fatal(err)
	}

	if len(failed) > 0 {
		t.Errorf("%d codes without a Portuguese description:\n%s", len(failed), strings.Join(failed, "\n"))
	}
}
