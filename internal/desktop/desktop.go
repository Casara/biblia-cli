// Package desktop reaches the graphical session through its own small
// programs: the selected text and the clipboard (wl-clipboard on Wayland,
// xclip on X11), message windows (zenity), notifications (notify-send)
// and GNOME's custom keyboard shortcuts (gsettings).
//
// GNOME has no context menu for selected text; a global shortcut that
// runs "biblia selecao" takes its place. Every program goes through a
// Runner, so tests replace them all.
package desktop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ErrNoTool is returned when none of the programs a task needs is
// installed; the message says which ones would do.
var ErrNoTool = errors.New("programa necessário não instalado")

// ErrNoAnswer is returned when reading the selection does not finish in
// time. On GNOME, wl-paste reaches the selection through a small window
// that has to take the focus, and it can wait forever when it cannot (a
// locked screen).
var ErrNoAnswer = errors.New("a leitura da seleção não respondeu")

// selectionTimeout bounds each read of the selection; a working one takes
// milliseconds.
const selectionTimeout = 5 * time.Second

// Runner runs a program with stdin as its input and returns its output.
type Runner interface {
	Run(ctx context.Context, stdin, name string, args ...string) (string, error)
	// Feed runs a program only to hand it stdin, without reading its
	// output. wl-copy and xclip -in leave a process behind to serve the
	// clipboard, and that process keeps any captured output open: Run
	// would wait for it forever.
	Feed(ctx context.Context, stdin, name string, args ...string) error
	// Has reports whether the program is installed.
	Has(name string) bool
}

// System runs the programs installed on the machine.
type System struct{}

// Run runs name. A failure carries the program's own message.
func (System) Run(ctx context.Context, stdin, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(stdin)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return stdout.String(), fmt.Errorf("desktop: %s: %w", name, err)
		}

		return stdout.String(), fmt.Errorf("desktop: %s: %w: %s", name, err, msg)
	}

	return stdout.String(), nil
}

// Feed runs name with its output discarded (no pipes to wait on).
func (System) Feed(ctx context.Context, stdin, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(stdin)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("desktop: %s: %w", name, err)
	}

	return nil
}

// Has reports whether name is on the PATH.
func (System) Has(name string) bool {
	_, err := exec.LookPath(name)

	return err == nil
}

// Session is the reader's graphical session.
type Session struct {
	Run Runner
	// Wayland picks wl-clipboard over xclip (WAYLAND_DISPLAY is set).
	Wayland bool
	// Timeout bounds each read of the selection; zero means
	// selectionTimeout.
	Timeout time.Duration
}

// Selection returns the selected text: the PRIMARY selection, what the
// middle mouse button pastes, so selecting is enough. When nothing is
// selected, it returns the clipboard.
func (s Session) Selection(ctx context.Context) (string, error) {
	read := func(primary bool) (string, error) {
		limit := s.Timeout
		if limit == 0 {
			limit = selectionTimeout
		}

		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()

		text, err := s.readOnce(ctx, primary)
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("%w em %s", ErrNoAnswer, limit)
		}

		return text, err
	}

	text, err := read(true)
	if errors.Is(err, ErrNoTool) || errors.Is(err, ErrNoAnswer) {
		return "", err
	}

	// An empty selection makes wl-paste and xclip fail ("Nothing is
	// copied"): that is the case for falling back to the clipboard.
	if err != nil || strings.TrimSpace(text) == "" {
		text, err = read(false)
		if errors.Is(err, ErrNoAnswer) {
			return "", err
		}

		// An empty clipboard is no selection either, not a failure.
		if err != nil {
			text = ""
		}
	}

	return text, nil
}

// Copy puts text on the clipboard.
func (s Session) Copy(ctx context.Context, text string) error {
	if s.Wayland && s.Run.Has("wl-copy") {
		err := s.Run.Feed(ctx, text, "wl-copy")
		if err != nil {
			return fmt.Errorf("desktop: copying: %w", err)
		}

		return nil
	}

	if s.Run.Has("xclip") {
		err := s.Run.Feed(ctx, text, "xclip", "-in", "-selection", "clipboard")
		if err != nil {
			return fmt.Errorf("desktop: copying: %w", err)
		}

		return nil
	}

	return s.missing("wl-clipboard", "xclip")
}

// Show opens text in a window, in a monospaced font: the interlinear and
// the lexicon are aligned in columns.
func (s Session) Show(ctx context.Context, title, text string) error {
	if !s.Run.Has("zenity") {
		return s.missing("zenity")
	}

	_, err := s.Run.Run(ctx, text, "zenity",
		"--text-info",
		"--title="+title,
		"--font=monospace",
		"--width=900",
		"--height=600",
		// The same label as the window with the translations button.
		"--ok-label=Fechar",
	)
	// Cancelar, Esc or the window's X: the reader just closed it.
	if err != nil && !closedDialog(err) {
		return fmt.Errorf("desktop: showing the text: %w", err)
	}

	return nil
}

// zenity's exit status when the reader closes a dialog (Cancel, Esc, the
// window's X) or presses an extra button.
const zenityClosed = 1

// exitCoder is what *exec.ExitError offers; tests fake it.
type exitCoder interface{ ExitCode() int }

// closedDialog reports whether err only says the reader closed the dialog.
func closedDialog(err error) bool {
	var ec exitCoder

	return errors.As(err, &ec) && ec.ExitCode() == zenityClosed
}

// ShowWithButton is Show with one more button beside "Fechar". It reports
// whether the reader pressed that button; closing the window in any other
// way is not an error.
func (s Session) ShowWithButton(ctx context.Context, title, text, button string) (bool, error) {
	if !s.Run.Has("zenity") {
		return false, s.missing("zenity")
	}

	out, err := s.Run.Run(ctx, text, "zenity",
		"--text-info",
		"--title="+title,
		"--font=monospace",
		"--width=900",
		"--height=600",
		"--ok-label=Fechar",
		"--extra-button="+button,
	)

	// An extra button closes the dialog like Cancel does and prints its
	// label: the label tells them apart.
	if strings.TrimSpace(out) == button {
		return true, nil
	}

	if err != nil && !closedDialog(err) {
		return false, fmt.Errorf("desktop: showing the text: %w", err)
	}

	return false, nil
}

// Choice is one row of a PickMany list.
type Choice struct {
	ID, Name, Group string
	Checked         bool
}

// PickMany shows a list with check boxes and returns the IDs checked. ok
// is false when the reader cancels.
func (s Session) PickMany(
	ctx context.Context, title string, columns [3]string, choices []Choice,
) ([]string, bool, error) {
	if !s.Run.Has("zenity") {
		return nil, false, s.missing("zenity")
	}

	args := []string{
		"--list", "--checklist",
		"--title=" + title,
		"--width=700", "--height=600",
		"--separator=,",
		"--column= ",
	}
	for _, c := range columns {
		args = append(args, "--column="+c)
	}

	for _, choice := range choices {
		check := "FALSE"
		if choice.Checked {
			check = "TRUE"
		}

		args = append(args, check, choice.ID, choice.Name, choice.Group)
	}

	out, err := s.Run.Run(ctx, "", "zenity", args...)
	if err != nil {
		if closedDialog(err) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("desktop: listing the choices: %w", err)
	}

	var ids []string

	for id := range strings.SplitSeq(strings.TrimSpace(out), ",") {
		if id != "" {
			ids = append(ids, id)
		}
	}

	return ids, true, nil
}

// Alert shows an error message in a window, or as a notification when
// zenity is missing. It reports nothing: there is nowhere left to.
func (s Session) Alert(ctx context.Context, title, msg string) {
	if s.Run.Has("zenity") {
		err := s.Run.Feed(ctx, "", "zenity", "--error", "--title="+title, "--no-markup", "--text="+msg)
		if err == nil {
			return
		}
	}

	s.Notify(ctx, title, msg)
}

// Notify shows a desktop notification, if notify-send is installed.
func (s Session) Notify(ctx context.Context, title, msg string) {
	if s.Run.Has("notify-send") {
		_ = s.Run.Feed(ctx, "", "notify-send", "--app-name="+title, title, msg)
	}
}

// Missing lists the programs "biblia selecao" needs that are not
// installed, as package names to install (Debian/Ubuntu).
func (s Session) Missing() []string {
	var out []string

	if !s.Run.Has("zenity") {
		out = append(out, "zenity")
	}

	switch {
	case s.Wayland && (!s.Run.Has("wl-paste") || !s.Run.Has("wl-copy")):
		out = append(out, "wl-clipboard")
	case !s.Wayland && !s.Run.Has("xclip"):
		out = append(out, "xclip")
	}

	return out
}

// readOnce reads the PRIMARY selection or the clipboard once.
func (s Session) readOnce(ctx context.Context, primary bool) (string, error) {
	if s.Wayland && s.Run.Has("wl-paste") {
		args := []string{"--no-newline"}
		if primary {
			args = append(args, "--primary")
		}

		return s.read(ctx, "wl-paste", args...)
	}

	if s.Run.Has("xclip") {
		sel := "clipboard"
		if primary {
			sel = "primary"
		}

		return s.read(ctx, "xclip", "-out", "-selection", sel)
	}

	return "", s.missing("wl-clipboard", "xclip")
}

func (s Session) read(ctx context.Context, name string, args ...string) (string, error) {
	out, err := s.Run.Run(ctx, "", name, args...)
	if err != nil {
		return out, fmt.Errorf("desktop: reading the selection: %w", err)
	}

	return out, nil
}

func (s Session) missing(tools ...string) error {
	return fmt.Errorf("%w: %s", ErrNoTool, strings.Join(tools, " ou "))
}
