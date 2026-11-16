package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/casara/biblia-cli/internal/bibliaonline"
	"github.com/casara/biblia-cli/internal/cache"
	"github.com/casara/biblia-cli/internal/desktop"
)

// The selection modes, one per desktop shortcut.
const (
	modeShow     = "consultar"
	modeOriginal = "original"
	modeCopy     = "copiar"
)

// maxSelection bounds what "biblia selecao" takes as a reference or a
// Strong number: a longer selection is a page of text selected by
// mistake, not something to send to the server.
const maxSelection = 200

// selectionTitle names the windows and notifications.
const selectionTitle = "Biblia"

// cmdStrongName is the strong subcommand, which a Strong number runs.
const cmdStrongName = "strong"

// strongNumber is a Strong number as "biblia strong" takes it: G25,
// g0025, H7451b, H2603A.
var strongNumber = regexp.MustCompile(`^[HhGg]\d{1,4}[A-Za-z]?$`)

// selectionPlan is what one selection turns into: a run of this same
// program, and what to do with its output.
type selectionPlan struct {
	args  []string
	stdin string
	title string
	// copy puts the output on the clipboard instead of showing it.
	copy bool
	// filter marks a "filtro" run, which echoes lines that are not
	// references: an output equal to the input looked nothing up.
	filter bool
	// pickVersions offers the translations button in the window.
	pickVersions bool
}

// versionsButton is the window's button that picks the translations.
const versionsButton = "Traduções…"

// planSelection decides what the selected text asks for: a Strong number
// opens its lexicon entry, anything else is read as references.
func planSelection(sel, mode, format string) (selectionPlan, error) {
	if strongNumber.MatchString(sel) {
		number := strings.ToUpper(sel)

		switch mode {
		case modeShow, modeOriginal:
			return selectionPlan{args: []string{cmdStrongName, sel, "-c"}, title: number}, nil
		case modeCopy:
			return selectionPlan{args: []string{cmdStrongName, sel}, title: number, copy: true}, nil
		}
	}

	// A selection across lines is one lookup ("Jo 3.16; Rm 8.28").
	refs := strings.Join(strings.Fields(strings.ReplaceAll(sel, "\n", "; ")), " ")

	switch mode {
	case modeShow:
		return selectionPlan{args: []string{"-f", "texto", refs}, title: refs, pickVersions: true}, nil
	case modeOriginal:
		return selectionPlan{
			args:         []string{"-f", "texto", "-s", refs},
			title:        refs + " (original)",
			pickVersions: true,
		}, nil
	case modeCopy:
		args := []string{"filtro"}
		if format != "" {
			args = append(args, "-f", format)
		}

		return selectionPlan{args: args, stdin: sel + "\n", title: refs, copy: true, filter: true}, nil
	}

	return selectionPlan{}, fmt.Errorf("%w: modo %q (consultar, original ou copiar)", errUsage, mode)
}

func newDesktopSession() desktop.Session {
	return desktop.Session{
		Run:     desktop.System{},
		Wayland: os.Getenv("WAYLAND_DISPLAY") != "",
		Timeout: 0,
	}
}

// cmdSelection looks up what is selected in any program; a desktop
// shortcut runs it, so every outcome is told in a window or a
// notification as well as on stderr.
func cmdSelection(ctx context.Context, e *env, args []string) error {
	fs := flag.NewFlagSet("biblia selecao", flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	format := fs.String("f", "", "formato do copiar (padrão: formato_filtro)")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}

	mode := modeShow
	if len(pos) > 0 {
		mode = pos[0]
	}

	desk := newDesktopSession()

	fail := func(msg string) error {
		desk.Alert(ctx, selectionTitle, msg)
		fmt.Fprintln(e.stderr, "biblia:", msg)

		return errReported
	}

	sel, msg := readSelection(ctx, desk)
	if msg != "" {
		return fail(msg)
	}

	plan, err := planSelection(sel, mode, *format)
	if err != nil {
		return err
	}

	if plan.pickVersions {
		msg = browse(ctx, e, desk, plan)
	} else {
		msg = answerOnce(ctx, e, desk, plan, sel)
	}

	if msg != "" {
		return fail(msg)
	}

	return nil
}

// answerOnce runs a plan without the translations button (a Strong
// number, or a copy) and delivers its answer. It returns the message of a
// failure.
func answerOnce(ctx context.Context, e *env, desk desktop.Session, plan selectionPlan, sel string) string {
	out, msg := runInner(ctx, e, plan)

	if plan.filter && strings.TrimSpace(out) == sel {
		out = ""
	}

	if strings.TrimSpace(out) == "" {
		if msg == "" {
			msg = "nada encontrado para: " + sel
		}

		return msg
	}

	err := deliver(ctx, desk, plan, out, msg)
	if err != nil {
		return err.Error()
	}

	return ""
}

// readSelection returns the selected text, trimmed, or the message that
// tells why there is nothing to look up.
func readSelection(ctx context.Context, desk desktop.Session) (string, string) {
	sel, err := desk.Selection(ctx)
	if err != nil {
		return "", err.Error()
	}

	sel = strings.TrimSpace(sel)
	size := len([]rune(sel))

	switch {
	case sel == "":
		return "", "nada selecionado: selecione uma referência (Jo 3.16) ou um número de Strong (G5293)"
	case size > maxSelection:
		return "", fmt.Sprintf("a seleção é longa demais para ser uma referência (%d caracteres)", size)
	}

	return sel, ""
}

// deliver copies the answer or shows it in a window. Notes next to an
// answer ("glosas em inglês...") go to a notification.
func deliver(ctx context.Context, desk desktop.Session, plan selectionPlan, out, msg string) error {
	if plan.copy {
		err := desk.Copy(ctx, strings.TrimRight(out, "\n"))
		if err != nil {
			return fmt.Errorf("copiando: %w", err)
		}

		note := "Copiado: " + plan.title
		if msg != "" {
			note += "\n" + msg
		}

		desk.Notify(ctx, selectionTitle, note)

		return nil
	}

	if msg != "" {
		desk.Notify(ctx, selectionTitle, msg)
	}

	err := desk.Show(ctx, selectionTitle+": "+plan.title, out)
	if err != nil {
		return fmt.Errorf("mostrando: %w", err)
	}

	return nil
}

// browse shows a lookup in a window whose "Traduções…" button picks other
// translations and shows it again, until the reader closes it. The choice
// lasts while the window is open; the next lookup starts from the
// configured translations. It returns the message of a failure.
func browse(ctx context.Context, e *env, desk desktop.Session, plan selectionPlan) string {
	versions := e.cfg.Versions

	for {
		run := plan
		run.args = append(slices.Clone(plan.args), "-v", strings.Join(versions, ","))

		out, msg := runInner(ctx, e, run)
		if strings.TrimSpace(out) == "" {
			if msg == "" {
				msg = "nada encontrado para: " + plan.title
			}

			return msg
		}

		if msg != "" {
			desk.Notify(ctx, selectionTitle, msg)
		}

		pressed, err := desk.ShowWithButton(ctx, selectionTitle+": "+plan.title, out, versionsButton)
		if err != nil {
			return err.Error()
		}

		if !pressed {
			return ""
		}

		picked, err := pickVersions(ctx, e, desk, versions)
		if err != nil {
			return err.Error()
		}

		if len(picked) > 0 {
			versions = picked
		}
	}
}

// pickVersions lists every translation of the site, in every language,
// with the current ones checked and first. It returns nil when the reader
// cancels or checks none.
func pickVersions(ctx context.Context, e *env, desk desktop.Session, current []string) ([]string, error) {
	c, err := cache.Open(ctx, e.paths.Cache)
	if err != nil {
		return nil, err
	}
	defer func() { _ = c.Close() }()

	bibles, err := catalog(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("lista de traduções: %w", err)
	}

	picked, ok, err := desk.PickMany(ctx, selectionTitle+": traduções",
		[3]string{"Tradução", "Nome", "Idioma"}, versionChoices(bibles, current))
	if err != nil || !ok {
		return nil, err
	}

	return picked, nil
}

// versionChoices lists the catalog with the current translations checked
// and first, in their own order, so the reader sees them without
// scrolling.
func versionChoices(bibles []bibliaonline.Bible, current []string) []desktop.Choice {
	choices := make([]desktop.Choice, 0, len(bibles))

	byID := map[string]bibliaonline.Bible{}
	for _, b := range bibles {
		byID[b.ID] = b
	}

	for _, id := range current {
		if b, ok := byID[id]; ok {
			choices = append(choices, desktop.Choice{ID: b.ID, Name: b.Name, Group: b.Language, Checked: true})
		}
	}

	for _, b := range bibles {
		if !slices.Contains(current, b.ID) {
			choices = append(choices, desktop.Choice{ID: b.ID, Name: b.Name, Group: b.Language, Checked: false})
		}
	}

	return choices
}

// runInner runs plan.args as this program would, capturing what it
// prints. msg joins its messages and its error, without the hints meant
// for a terminal.
func runInner(ctx context.Context, e *env, plan selectionPlan) (string, string) {
	var stdout, stderr bytes.Buffer

	inner := *e
	inner.stdin = strings.NewReader(plan.stdin)
	inner.stdout, inner.stderr = &stdout, &stderr
	inner.tty, inner.width = false, 0

	err := dispatch(ctx, &inner, plan.args)

	var lines []string

	for line := range strings.SplitSeq(stderr.String(), "\n") {
		line = strings.TrimPrefix(strings.TrimSpace(line), "biblia: ")
		if line != "" && line != `veja "biblia ajuda"` {
			lines = append(lines, line)
		}
	}

	switch {
	case errors.Is(err, errNothingFound):
		lines = append(lines, "nada encontrado para: "+plan.title)
	case err != nil && !errors.Is(err, errReported):
		lines = append(lines, err.Error())
	}

	return stdout.String(), strings.Join(lines, "\n")
}

// cmdShortcuts registers or removes the GNOME shortcuts that run
// "biblia selecao".
func cmdShortcuts(ctx context.Context, e *env, args []string) error {
	if len(args) != 1 || (args[0] != "instalar" && args[0] != "remover") {
		return fmt.Errorf("%w: biblia atalhos instalar | remover", errUsage)
	}

	if runtime.GOOS != "linux" {
		return fmt.Errorf("%w: os atalhos existem só para o GNOME (Linux), por enquanto", errUsage)
	}

	run := desktop.System{}

	if args[0] == "remover" {
		err := desktop.Remove(ctx, run)
		if err != nil {
			return err
		}

		fmt.Fprintln(e.stdout, "atalhos do biblia removidos")

		return nil
	}

	exe, err := installedExecutable()
	if err != nil {
		return err
	}

	err = desktop.Install(ctx, run, exe)
	if err != nil {
		return err
	}

	for _, sc := range desktop.Shortcuts() {
		fmt.Fprintf(e.stdout, "%-15s %s\n", desktop.Label(sc.Binding), sc.Name)
	}

	fmt.Fprintf(e.stdout, "\nchamam %s; para mudar as teclas: Configurações › Teclado › Atalhos personalizados\n", exe)

	missing := newDesktopSession().Missing()
	if len(missing) > 0 {
		fmt.Fprintf(e.stderr, "biblia: falta instalar: sudo apt install %s\n", strings.Join(missing, " "))
	}

	return nil
}

// installedExecutable is the absolute path the shortcuts will run: this
// program, unless it is a temporary "go run" build.
func installedExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("localizando o executável: %w", err)
	}

	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("localizando o executável: %w", err)
	}

	if strings.HasPrefix(exe, os.TempDir()) {
		return "", fmt.Errorf(
			"%w: rode o biblia instalado (make install), não o go run: o atalho guarda o caminho %s",
			errUsage, exe,
		)
	}

	return exe, nil
}
