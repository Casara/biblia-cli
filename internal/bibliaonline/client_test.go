package bibliaonline_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casara/biblia-cli/internal/bibliaonline"
)

// Answers captured from the live server.
const (
	full = "**João 3:16,17** (ARA)\n\n" +
		"16. Porque Deus amou ao mundo de tal maneira que deu o seu Filho unigênito.\n" +
		"17. Porquanto Deus enviou o seu Filho ao mundo.\n\n" +
		"No site: https://www.bibliaonline.com.br/ara/jo/3/16,17"
	partial = "**Salmos 23:1-5** (ARA)\n\n" +
		"1. O Senhor é o meu pastor; nada me faltará.\n" +
		"2. Ele me faz repousar em pastos verdejantes.\n" +
		"3. refrigera-me a alma.\n" +
		"4. Ainda que eu ande pelo vale da sombra da morte,\n" +
		"5. Preparas-me uma mesa\n\n" +
		"Trecho parcial (limite de 5 versículos). Capítulo completo: https://www.bibliaonline.com.br/ara/sl/23"
	gap = "**Mateus 17:20,22** (NVI)\n\n" +
		"20. Ele respondeu: — Porque a fé que vocês têm é pequena.\n" +
		"22. Reunindo-se eles na Galileia, Jesus lhes disse.\n\n" +
		"No site: https://www.bibliaonline.com.br/nvi/mt/17/20,22"
)

func TestParsePassage(t *testing.T) {
	p, err := bibliaonline.ParsePassage(full)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Verses) != 2 || p.Verses[0].Number != 16 || p.Verses[1].Number != 17 || p.Partial {
		t.Errorf("full: %+v", p)
	}

	if p.URL != "https://www.bibliaonline.com.br/ara/jo/3/16,17" {
		t.Errorf("full URL = %q", p.URL)
	}

	p, err = bibliaonline.ParsePassage(partial)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Verses) != 5 || !p.Partial || p.URL != "https://www.bibliaonline.com.br/ara/sl/23" {
		t.Errorf("partial: %+v", p)
	}

	p, err = bibliaonline.ParsePassage(gap)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Verses) != 2 || p.Verses[1].Number != 22 {
		t.Errorf("gap: %+v", p)
	}
}

func TestParsePassageErrors(t *testing.T) {
	_, err := bibliaonline.ParsePassage("Passagem não encontrada em ARA: João 99:1.")
	if !errors.Is(err, bibliaonline.ErrNotFound) {
		t.Errorf("not found: %v", err)
	}

	_, err = bibliaonline.ParsePassage(`Não existe tradução com o id "xyz". Chame list_bibles.`)
	if !errors.Is(err, bibliaonline.ErrUnknownBible) {
		t.Errorf("unknown bible: %v", err)
	}
}

func TestParseBibles(t *testing.T) {
	text := "24 traduções disponíveis:\n\n" +
		"- `ara` — Almeida Revista e Atualizada (ARA), Português: https://www.bibliaonline.com.br/ara\n" +
		"- `vfl` — Bíblia Sagrada: Versão Fácil de Ler (VFL), Português: https://www.bibliaonline.com.br/vfl\n" +
		"- `mznvi` — Nova Versão Internacional, Português de Moçambique (MZNVI), Português: " +
		"https://www.bibliaonline.com.br/mznvi\n"

	got := bibliaonline.ParseBibles(text)
	if len(got) != 3 {
		t.Fatalf("got %d bibles: %+v", len(got), got)
	}

	if got[1].Name != "Bíblia Sagrada: Versão Fácil de Ler (VFL)" || got[2].Language != "Português" {
		t.Errorf("got %+v", got)
	}
}

// TestClientSSE checks the transport: JSON-RPC request in, one Server-
// Sent Event out.
func TestClientSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") == "" || r.Method != http.MethodPost {
			http.Error(w, "bad request", http.StatusBadRequest)

			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: message\ndata: %s\n\n",
			`{"result":{"content":[{"type":"text","text":"**João 3:16** (ARA)\n\n16. Porque Deus amou.\n\n`+
				`No site: https://www.bibliaonline.com.br/ara/jo/3/16"}]},"jsonrpc":"2.0","id":1}`)
	}))
	defer srv.Close()

	c := bibliaonline.New()
	c.Endpoint = srv.URL

	p, err := c.GetPassage(context.Background(), "João 3:16", "ara")
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Verses) != 1 || p.Verses[0].Text != "Porque Deus amou." {
		t.Errorf("got %+v", p)
	}
}
