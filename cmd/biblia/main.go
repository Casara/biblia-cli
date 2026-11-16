// Command biblia looks up Bible verses in several translations from the
// command line or from an editor, with an optional interlinear of the
// original Hebrew/Greek and Strong's numbers.
//
//	biblia jo 3.16                   # default translation(s)
//	biblia jo 3.16 -v ara,naa,nvi    # side by side
//	biblia jo 3.16 -s                # + original words with Strong numbers
//	biblia jo 3.16 -f typ            # ready to paste, with the "conferido" line
//	echo "Rm 8.28" | biblia filtro   # for editors: reference in, text out
//	biblia strong G0025 -c           # lexicon entry
//	biblia hc 2.4 -p                 # parallel passages
//
// Run "biblia ajuda" for the full usage.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"golang.org/x/term"

	"github.com/casara/biblia-cli/internal/config"
)

const (
	exitOK           = 0
	exitRuntimeError = 1
	exitUsageError   = 2
)

// version is set at build time (make build: git describe).
var version = "dev"

// errUsage marks an error caused by how the program was called.
var errUsage = errors.New("uso")

// errNothingFound is returned by a search with no results.
var errNothingFound = errors.New("nada encontrado")

// errReported means the failures were already printed one by one; only
// the exit status is left to set.
var errReported = errors.New("falhas já relatadas")

// env carries what every subcommand needs.
type env struct {
	stdin          io.Reader
	stdout, stderr io.Writer
	paths          config.Paths
	cfg            config.Config
	tty            bool
	width          int
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := run(ctx, os.Args[1:])

	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string) int {
	e := &env{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}

	paths, err := config.Resolve()
	if err != nil {
		fmt.Fprintln(e.stderr, "biblia:", err)

		return exitRuntimeError
	}

	cfg, err := config.Load(paths.Config)
	if err != nil {
		fmt.Fprintln(e.stderr, "biblia:", err)

		return exitRuntimeError
	}

	e.paths, e.cfg = paths, cfg

	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		e.tty = true
		w, _, err := term.GetSize(fd)
		if err == nil {
			e.width = w
		}
	}

	err = dispatch(ctx, e, args)

	switch {
	case err == nil:
		return exitOK
	case errors.Is(err, flag.ErrHelp):
		return exitOK
	case errors.Is(err, errReported):
		return exitRuntimeError
	case errors.Is(err, errUsage):
		fmt.Fprintln(e.stderr, "biblia:", err)
		fmt.Fprintln(e.stderr, `veja "biblia ajuda"`)

		return exitUsageError
	default:
		fmt.Fprintln(e.stderr, "biblia:", err)

		return exitRuntimeError
	}
}

func dispatch(ctx context.Context, e *env, args []string) error {
	if len(args) == 0 {
		fmt.Fprint(e.stdout, usage)

		return nil
	}

	switch args[0] {
	case "ajuda", "help", "-h", "-help", "--help":
		fmt.Fprint(e.stdout, usage)

		return nil
	case "filtro":
		return cmdFilter(ctx, e, args[1:])
	case "strong":
		return cmdStrong(ctx, e, args[1:])
	case "versoes", "versões":
		return cmdVersions(ctx, e, args[1:])
	case "dados":
		return cmdData(ctx, e, args[1:])
	case "cache":
		return cmdCache(ctx, e, args[1:])
	case "formatos":
		return cmdFormats(e)
	case "selecao", "seleção":
		return cmdSelection(ctx, e, args[1:])
	case "atalhos":
		return cmdShortcuts(ctx, e, args[1:])
	case "versao", "versão", "-version", "--version":
		fmt.Fprintln(e.stdout, "biblia", version)

		return nil
	default:
		return cmdLookup(ctx, e, args)
	}
}

// parseInterspersed lets flags come before or after the positional
// arguments ("biblia jo 3.16 -v nvi" and "biblia -v nvi jo 3.16"), which
// the standard flag package alone does not.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string

	for {
		err := fs.Parse(args)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errUsage, err)
		}

		args = fs.Args()
		if len(args) == 0 {
			return pos, nil
		}

		pos = append(pos, args[0])
		args = args[1:]
	}
}

func splitList(s string) []string {
	var out []string

	for part := range strings.SplitSeq(s, ",") {
		if part = strings.ToLower(strings.TrimSpace(part)); part != "" {
			out = append(out, part)
		}
	}

	return out
}

const usage = `biblia — versículos (Bíblia Online), original e Strong (STEPBible), dicionários e paralelos (UBS)

USO
  biblia <referência> [opções]       consulta
  biblia filtro [opções]             lê referências da entrada padrão (editores);
                                     com -p, devolve só os textos dos paralelos
  biblia strong <número> [-c]        sentidos do número (H2617, G25, H7451b)
  biblia strong -tr <translit>       busca pela transliteração
  biblia strong -glosa <texto>       busca pela glosa (no idioma dos dicionários)
  biblia versoes [-todas]            traduções disponíveis no site
  biblia formatos                    lista os formatos de saída
  biblia selecao [consultar|original|copiar] [-f formato]
                                     consulta o texto selecionado em qualquer
                                     programa: referência ou número de Strong
  biblia atalhos instalar | remover  atalhos do GNOME para o selecao:
                                     Super+B, Super+Alt+B, Super+Shift+B
  biblia dados baixar [step|ubs] [-enxuto]
                                     original, léxicos, dicionários e paralelos;
                                     -enxuto apaga os arquivos baixados após indexar
  biblia dados status
  biblia cache status | limpar [versão]
  biblia versao

REFERÊNCIAS
  jo 3.16    jo 3:16    1co13.4-7    Rm 8    Sl 23.1-3,6    Jd 3
  Rm 5.8; 6.23    (o livro vale para o trecho seguinte)

OPÇÕES DA CONSULTA
  -v ara,naa,nvi   traduções (padrão: as do arquivo de configuração, ou ara)
  -f formato       texto (padrão), linha, md, typ, ou um modelo seu
  -s               mostra as palavras do original: Strong, glosa e gramática
  -p               lista as passagens paralelas (citações do AT no NT, Evangelhos...)
  -pt              as passagens paralelas com o texto
  -offline         usa só o cache, sem acessar o site

O site entrega até 5 versículos por consulta; para trechos maiores a
saída avisa e aponta o link do capítulo.

As glosas do -s e os sentidos do strong vêm dos dicionários da UBS, no
idioma configurado ou no mais próximo (grego: espanhol); [IA] marca a
tradução que a própria UBS diz ter sido feita por IA, a revisar.

Configuração: ~/.config/biblia/config.json (ou BIBLIA_CONFIG)
  {"versoes": ["ara"], "formato": "texto", "formato_filtro": "linha",
   "modelos": {"citacao": "{{.Text}} ({{.Ref}})"},
   "idioma": "pt", "dicionarios": ["ubs-hebraico-pt", "ubs-grego-es", "step"]}

Textos: www.bibliaonline.com.br (cada tradução tem seus titulares de direitos).
Original, léxicos e gramática: www.STEPBible.org, Tyndale House, Cambridge (CC BY 4.0).
Dicionários e paralelos: United Bible Societies (CC BY-SA 4.0).
`
