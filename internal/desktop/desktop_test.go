package desktop_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/casara/biblia-cli/internal/desktop"
)

// fakeRunner records every call and answers from a table keyed by the
// program and its arguments ("wl-paste --no-newline --primary").
type fakeRunner struct {
	installed map[string]bool
	answers   map[string]string
	failing   map[string]bool
	// exits makes a call end with that exit status, still printing its
	// answer: zenity prints an extra button's label and exits with 1.
	exits map[string]int
	// hanging calls never answer.
	hanging map[string]bool
	calls   []string
	inputs  []string
}

// exitError fakes *exec.ExitError.
type exitError int

func (e exitError) Error() string { return "exit status" }
func (e exitError) ExitCode() int { return int(e) }

func (f *fakeRunner) Has(name string) bool { return f.installed[name] }

func (f *fakeRunner) Feed(ctx context.Context, stdin, name string, args ...string) error {
	_, err := f.Run(ctx, stdin, name, args...)

	return err
}

func (f *fakeRunner) Run(ctx context.Context, stdin, name string, args ...string) (string, error) {
	call := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, call)
	f.inputs = append(f.inputs, stdin)

	// A hanging program waits until the context gives up, like
	// exec.CommandContext killing it.
	if f.hanging[call] {
		<-ctx.Done()

		return "", fmt.Errorf("fake %s: %w", name, ctx.Err())
	}

	if f.failing[call] {
		return "", errors.New("falhou: " + call)
	}

	if code, ok := f.exits[call]; ok {
		return f.answers[call], exitError(code)
	}

	return f.answers[call], nil
}

func installed(names ...string) map[string]bool {
	m := map[string]bool{}
	for _, n := range names {
		m[n] = true
	}

	return m
}

func TestSelection(t *testing.T) {
	tests := []struct {
		name    string
		wayland bool
		run     *fakeRunner
		want    string
		wantErr error
	}{
		{
			name:    "wayland primary",
			wayland: true,
			run: &fakeRunner{
				installed: installed("wl-paste"),
				answers:   map[string]string{"wl-paste --no-newline --primary": "Jo 3.16"},
			},
			want: "Jo 3.16",
		},
		{
			name:    "empty primary falls back to the clipboard",
			wayland: true,
			run: &fakeRunner{
				installed: installed("wl-paste"),
				failing:   map[string]bool{"wl-paste --no-newline --primary": true},
				answers:   map[string]string{"wl-paste --no-newline": "G5293"},
			},
			want: "G5293",
		},
		{
			name:    "nothing anywhere is no selection, not an error",
			wayland: true,
			run: &fakeRunner{
				installed: installed("wl-paste"),
				failing: map[string]bool{
					"wl-paste --no-newline --primary": true,
					"wl-paste --no-newline":           true,
				},
			},
			want: "",
		},
		{
			name: "x11 uses xclip",
			run: &fakeRunner{
				installed: installed("xclip", "wl-paste"),
				answers:   map[string]string{"xclip -out -selection primary": "Rm 8.28"},
			},
			want: "Rm 8.28",
		},
		{
			name:    "wayland without wl-clipboard uses xclip",
			wayland: true,
			run: &fakeRunner{
				installed: installed("xclip"),
				answers:   map[string]string{"xclip -out -selection primary": "Sl 23"},
			},
			want: "Sl 23",
		},
		{
			name:    "no tool",
			wayland: true,
			run:     &fakeRunner{},
			wantErr: desktop.ErrNoTool,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := desktop.Session{Run: tt.run, Wayland: tt.wayland}

			got, err := s.Selection(t.Context())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("selection = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSelectionGivesUp(t *testing.T) {
	run := &fakeRunner{
		installed: installed("wl-paste"),
		hanging:   map[string]bool{"wl-paste --no-newline --primary": true},
	}

	session := desktop.Session{Run: run, Wayland: true, Timeout: 10 * time.Millisecond}

	_, err := session.Selection(t.Context())
	if !errors.Is(err, desktop.ErrNoAnswer) {
		t.Fatalf("err = %v, want ErrNoAnswer", err)
	}

	// No second wait on the clipboard: it hangs for the same reason.
	if len(run.calls) != 1 {
		t.Errorf("calls = %q", run.calls)
	}
}

func TestCopyAndShow(t *testing.T) {
	run := &fakeRunner{installed: installed("wl-copy", "zenity")}
	session := desktop.Session{Run: run, Wayland: true}

	err := session.Copy(t.Context(), "texto")
	if err != nil {
		t.Fatal(err)
	}

	err = session.Show(t.Context(), "Biblia: Jo 3.16", "interlinear")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"wl-copy",
		"zenity --text-info --title=Biblia: Jo 3.16 --font=monospace --width=900 --height=600 --ok-label=Fechar",
	}
	if !slices.Equal(run.calls, want) {
		t.Errorf("calls = %q, want %q", run.calls, want)
	}

	if !slices.Equal(run.inputs, []string{"texto", "interlinear"}) {
		t.Errorf("inputs = %q", run.inputs)
	}

	err = desktop.Session{Run: &fakeRunner{}}.Show(t.Context(), "t", "x")
	if !errors.Is(err, desktop.ErrNoTool) {
		t.Errorf("show without zenity: err = %v", err)
	}
}

func TestAlertFallsBackToNotification(t *testing.T) {
	run := &fakeRunner{installed: installed("notify-send")}

	desktop.Session{Run: run}.Alert(t.Context(), "Biblia", "nada selecionado")

	want := []string{"notify-send --app-name=Biblia Biblia nada selecionado"}
	if !slices.Equal(run.calls, want) {
		t.Errorf("calls = %q, want %q", run.calls, want)
	}
}

func TestMissing(t *testing.T) {
	got := desktop.Session{Run: &fakeRunner{installed: installed("wl-paste")}, Wayland: true}.Missing()
	if !slices.Equal(got, []string{"zenity", "wl-clipboard"}) {
		t.Errorf("wayland: missing = %q", got)
	}

	got = desktop.Session{Run: &fakeRunner{installed: installed("zenity", "xclip")}}.Missing()
	if len(got) != 0 {
		t.Errorf("x11 with everything: missing = %q", got)
	}
}

func TestLabel(t *testing.T) {
	for binding, want := range map[string]string{
		"<Super>b":        "Super+B",
		"<Super><Alt>b":   "Super+Alt+B",
		"<Super><Shift>b": "Super+Shift+B",
	} {
		if got := desktop.Label(binding); got != want {
			t.Errorf("desktop.Label(%q) = %q, want %q", binding, got, want)
		}
	}
}

const (
	listSchema = "org.gnome.settings-daemon.plugins.media-keys"
	itemSchema = listSchema + ".custom-keybinding"
	basePath   = "/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/"
	other      = basePath + "custom0/"
	consultar  = basePath + "biblia-consultar/"
	getList    = "gsettings get " + listSchema + " custom-keybindings"
	setListPfx = "gsettings set " + listSchema + " custom-keybindings "
)

func TestInstallKeepsOtherShortcuts(t *testing.T) {
	run := &fakeRunner{
		installed: installed("gsettings"),
		// Already installed once: installing again must not add copies.
		answers: map[string]string{getList: "['" + other + "', '" + consultar + "']"},
	}

	// A path with a space: GNOME splits the command on spaces unless quoted.
	err := desktop.Install(t.Context(), run, "/home/leitor/meus programas/biblia")
	if err != nil {
		t.Fatal(err)
	}

	wantCommand := "gsettings set " + itemSchema + ":" + consultar +
		` command "'/home/leitor/meus programas/biblia' selecao consultar"`
	if !slices.Contains(run.calls, wantCommand) {
		t.Errorf("no call %q in %q", wantCommand, run.calls)
	}

	wantList := setListPfx + "['" + other + "', '" + consultar + "', '" +
		basePath + "biblia-original/', '" + basePath + "biblia-copiar/']"
	if last := run.calls[len(run.calls)-1]; last != wantList {
		t.Errorf("list =\n%s\nwant\n%s", last, wantList)
	}
}

func TestRemoveKeepsOtherShortcuts(t *testing.T) {
	run := &fakeRunner{
		installed: installed("gsettings"),
		answers:   map[string]string{getList: "['" + consultar + "', '" + other + "']"},
	}

	err := desktop.Remove(t.Context(), run)
	if err != nil {
		t.Fatal(err)
	}

	if last := run.calls[len(run.calls)-1]; last != setListPfx+"['"+other+"']" {
		t.Errorf("list = %s", last)
	}

	run.answers[getList] = "['" + consultar + "']"

	err = desktop.Remove(t.Context(), run)
	if err != nil {
		t.Fatal(err)
	}

	if last := run.calls[len(run.calls)-1]; last != setListPfx+"@as []" {
		t.Errorf("empty list = %s", last)
	}
}

func TestShortcutsNeedGsettings(t *testing.T) {
	err := desktop.Install(t.Context(), &fakeRunner{}, "/bin/biblia")
	if !errors.Is(err, desktop.ErrNoTool) {
		t.Errorf("err = %v, want desktop.ErrNoTool", err)
	}
}

const textWindow = "zenity --text-info --title=Biblia: Jo 3.16 --font=monospace " +
	"--width=900 --height=600 --ok-label=Fechar --extra-button=Traduções…"

func TestShowWithButton(t *testing.T) {
	tests := []struct {
		name        string
		answer      string
		exit        int
		wantPressed bool
		wantErr     bool
	}{
		{name: "closed with Fechar", exit: -1},
		{name: "extra button", answer: "Traduções…\n", exit: 1, wantPressed: true},
		{name: "closed with Esc", exit: 1},
		{name: "zenity broke", exit: 255, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &fakeRunner{
				installed: installed("zenity"),
				answers:   map[string]string{textWindow: tt.answer},
				exits:     map[string]int{},
			}
			if tt.exit >= 0 {
				run.exits[textWindow] = tt.exit
			}

			pressed, err := desktop.Session{Run: run}.ShowWithButton(
				t.Context(), "Biblia: Jo 3.16", "texto", "Traduções…",
			)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}

			if pressed != tt.wantPressed {
				t.Errorf("pressed = %v, want %v", pressed, tt.wantPressed)
			}

			if !slices.Equal(run.calls, []string{textWindow}) {
				t.Errorf("calls = %q", run.calls)
			}
		})
	}
}

func TestPickMany(t *testing.T) {
	list := "zenity --list --checklist --title=Traduções --width=700 --height=600 --separator=, " +
		"--column=  --column=Tradução --column=Nome --column=Idioma " +
		"TRUE ara Almeida Revista e Atualizada Português " +
		"FALSE kjv King James Version English"
	choices := []desktop.Choice{
		{ID: "ara", Name: "Almeida Revista e Atualizada", Group: "Português", Checked: true},
		{ID: "kjv", Name: "King James Version", Group: "English"},
	}
	columns := [3]string{"Tradução", "Nome", "Idioma"}

	run := &fakeRunner{installed: installed("zenity"), answers: map[string]string{list: "ara,kjv\n"}}

	ids, ok, err := desktop.Session{Run: run}.PickMany(t.Context(), "Traduções", columns, choices)
	if err != nil || !ok || !slices.Equal(ids, []string{"ara", "kjv"}) {
		t.Fatalf("got %q, %v, %v; calls %q", ids, ok, err, run.calls)
	}

	run.exits = map[string]int{list: 1}

	ids, ok, err = desktop.Session{Run: run}.PickMany(t.Context(), "Traduções", columns, choices)
	if err != nil || ok || ids != nil {
		t.Errorf("cancel: got %q, %v, %v", ids, ok, err)
	}
}
