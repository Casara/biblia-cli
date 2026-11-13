// Package bibliaonline talks to the MCP server Bíblia Online publishes
// for programs (announced in https://www.bibliaonline.com.br/llms.txt),
// rather than scraping the site: the chapter pages load their text from
// /api/, which the site's robots.txt closes to automated clients.
//
// The server returns at most five verses per call and asks that longer
// passages send the reader to the site. This client honors that: one
// call per request, and whatever the server leaves out is reported back
// as missing, never fetched in further calls behind the reader's back.
package bibliaonline

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultEndpoint is the Portuguese MCP endpoint. It accepts every
// translation id on the site, English and Spanish ones included; only
// the language of its messages changes between /pt, /en and /es.
const DefaultEndpoint = "https://mcp.biblebox.com/pt"

// MaxVerses is the server's per-call limit.
const MaxVerses = 5

// Sentinel errors, matchable with errors.Is.
var (
	ErrNotFound      = errors.New("passagem não encontrada")
	ErrUnknownBible  = errors.New("tradução desconhecida")
	ErrUnexpected    = errors.New("resposta inesperada do servidor")
	errNoDataInEvent = errors.New("sem dados no evento")
)

// Verse is one numbered verse as returned by the server.
type Verse struct {
	Number int
	Text   string
}

// Passage is the parsed answer to a get_passage call.
type Passage struct {
	// Verses holds what the server returned, in order.
	Verses []Verse
	// URL is the canonical link to the passage (or, when Partial, to the
	// whole chapter).
	URL string
	// Partial is true when the server cut the passage at MaxVerses.
	Partial bool
}

// Bible is one entry of the translation catalog.
type Bible struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	URL      string `json:"url"`
}

// Client calls the MCP server over plain JSON-RPC (Streamable HTTP
// transport). The server is stateless: no session handshake is needed
// before tools/call.
type Client struct {
	Endpoint string
	HTTP     *http.Client
	// UserAgent identifies the program generically; it never carries
	// anything about the person running it.
	UserAgent string
}

// New returns a client with sensible defaults.
func New() *Client {
	const timeout = 20 * time.Second

	return &Client{
		Endpoint:  DefaultEndpoint,
		HTTP:      &http.Client{Timeout: timeout},
		UserAgent: "biblia-cli",
	}
}

// GetPassage resolves reference (e.g. "João 3:16-18") in the translation
// bibleID ("ara").
func (c *Client) GetPassage(ctx context.Context, reference, bibleID string) (Passage, error) {
	text, err := c.callTool(ctx, "get_passage", map[string]any{
		"reference": reference,
		"bibleId":   bibleID,
	})
	if err != nil {
		return Passage{}, err
	}

	return ParsePassage(text)
}

// ListBibles returns the translation catalog.
func (c *Client) ListBibles(ctx context.Context) ([]Bible, error) {
	text, err := c.callTool(ctx, "list_bibles", map[string]any{})
	if err != nil {
		return nil, err
	}

	return ParseBibles(text), nil
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcResponse struct {
	Result *struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) callTool(ctx context.Context, name string, args map[string]any) (string, error) {
	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  map[string]any{"name": name, "arguments": args},
	})
	if err != nil {
		return "", fmt.Errorf("bibliaonline: encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("bibliaonline: building request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("bibliaonline: calling %s: %w", name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: HTTP %d em %s", ErrUnexpected, resp.StatusCode, name)
	}

	payload, err := readPayload(resp)
	if err != nil {
		return "", err
	}

	var rpc rpcResponse
	err = json.Unmarshal(payload, &rpc)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnexpected, err)
	}

	if rpc.Error != nil {
		return "", fmt.Errorf("%w: %s", ErrUnexpected, rpc.Error.Message)
	}

	if rpc.Result == nil || len(rpc.Result.Content) == 0 {
		return "", fmt.Errorf("%w: resultado vazio", ErrUnexpected)
	}

	return rpc.Result.Content[0].Text, nil
}

// readPayload returns the JSON-RPC message, whether the server answered
// with plain JSON or with a single Server-Sent Event.
func readPayload(resp *http.Response) ([]byte, error) {
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("bibliaonline: reading response: %w", err)
	}

	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		return raw, nil
	}

	var data []string

	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, len(raw)+1), len(raw)+1)

	for sc.Scan() {
		if line, ok := strings.CutPrefix(sc.Text(), "data:"); ok {
			data = append(data, strings.TrimPrefix(line, " "))
		}
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("%w: %w", ErrUnexpected, errNoDataInEvent)
	}

	return []byte(strings.Join(data, "\n")), nil
}

var (
	verseRe   = regexp.MustCompile(`^(\d+)\.\s+(.*)$`)
	siteURLRe = regexp.MustCompile(`https://\S+`)
	bibleRe   = regexp.MustCompile("^- `([^`]+)` — (.+), ([^,:]+): (https://\\S+)$")
)

// ParsePassage reads the markdown-ish text get_passage returns:
//
//	**João 3:16,17** (ARA)
//
//	16. Porque Deus amou ...
//	17. Porquanto Deus ...
//
//	No site: https://www.bibliaonline.com.br/ara/jo/3/16,17
//
// A partial answer ends with "Trecho parcial (limite de 5 versículos).
// Capítulo completo: <url>" instead. Errors come back as plain sentences
// ("Passagem não encontrada em ...", "Não existe tradução com o id ...").
func ParsePassage(text string) (Passage, error) {
	trimmed := strings.TrimSpace(text)

	switch {
	case strings.HasPrefix(trimmed, "Passagem não encontrada"):
		return Passage{}, ErrNotFound
	case strings.HasPrefix(trimmed, "Não existe tradução"):
		return Passage{}, fmt.Errorf("%w (veja: biblia versoes)", ErrUnknownBible)
	}

	var p Passage

	for i, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)

		switch {
		case i == 0 || line == "":
			continue
		case strings.HasPrefix(line, "Trecho parcial"):
			p.Partial = true
			p.URL = siteURLRe.FindString(line)
		case strings.HasPrefix(line, "No site:"):
			p.URL = siteURLRe.FindString(line)
		default:
			m := verseRe.FindStringSubmatch(line)
			if m != nil {
				n, _ := strconv.Atoi(m[1])
				p.Verses = append(p.Verses, Verse{Number: n, Text: m[2]})
			} else if k := len(p.Verses); k > 0 {
				// A verse that spans several lines (poetry).
				p.Verses[k-1].Text += " " + line
			}
		}
	}

	if len(p.Verses) == 0 {
		return Passage{}, fmt.Errorf("%w: nenhum versículo em %q", ErrUnexpected, trimmed)
	}

	return p, nil
}

// ParseBibles reads the list_bibles text, one "- `id` — Name, Language:
// url" line per translation.
func ParseBibles(text string) []Bible {
	var out []Bible

	for line := range strings.SplitSeq(text, "\n") {
		m := bibleRe.FindStringSubmatch(strings.TrimSpace(line))
		if m != nil {
			out = append(out, Bible{ID: m[1], Name: m[2], Language: m[3], URL: m[4]})
		}
	}

	return out
}
