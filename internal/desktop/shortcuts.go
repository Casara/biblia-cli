package desktop

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// The custom keybindings of GNOME's settings daemon: what Settings >
// Keyboard > Custom Shortcuts creates. The list holds one path per
// shortcut; each path is a relocatable schema with name, command and
// binding.
const (
	listSchema = "org.gnome.settings-daemon.plugins.media-keys"
	listKey    = "custom-keybindings"
	itemSchema = listSchema + ".custom-keybinding"
	basePath   = "/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/"
)

// Shortcut is one global key binding that runs "biblia selecao <Mode>".
type Shortcut struct {
	// ID names its settings path (".../custom-keybindings/biblia-consultar/"),
	// so installing again updates it instead of adding a copy.
	ID      string
	Name    string
	Mode    string
	Binding string
}

// Shortcuts are the bindings "biblia atalhos instalar" registers. Super+B
// and its variants are free in a stock GNOME.
func Shortcuts() []Shortcut {
	return []Shortcut{
		{ID: "biblia-consultar", Name: "Biblia: consultar a seleção", Mode: "consultar", Binding: "<Super>b"},
		{ID: "biblia-original", Name: "Biblia: seleção com o original", Mode: "original", Binding: "<Super><Alt>b"},
		{ID: "biblia-copiar", Name: "Biblia: copiar o texto da seleção", Mode: "copiar", Binding: "<Super><Shift>b"},
	}
}

// Label prints a binding the way people write it: "<Super><Alt>b" →
// "Super+Alt+B".
func Label(binding string) string {
	parts := strings.Split(strings.TrimPrefix(binding, "<"), "<")

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		mod, key, found := strings.Cut(p, ">")
		out = append(out, mod)

		if found && key != "" {
			out = append(out, strings.ToUpper(key))
		}
	}

	return strings.Join(out, "+")
}

// Install registers every shortcut to run exe (the absolute path of the
// biblia executable), leaving the reader's other custom shortcuts alone.
func Install(ctx context.Context, run Runner, exe string) error {
	if !run.Has("gsettings") {
		return fmt.Errorf("%w: gsettings (os atalhos são do GNOME)", ErrNoTool)
	}

	paths, err := shortcutPaths(ctx, run)
	if err != nil {
		return err
	}

	for _, sc := range Shortcuts() {
		path := basePath + sc.ID + "/"
		item := itemSchema + ":" + path
		// GNOME runs the command without a shell, splitting it on spaces
		// unless quoted.
		command := fmt.Sprintf("'%s' selecao %s", exe, sc.Mode)

		for _, kv := range [][2]string{{"name", sc.Name}, {"command", command}, {"binding", sc.Binding}} {
			_, err := run.Run(ctx, "", "gsettings", "set", item, kv[0], variantString(kv[1]))
			if err != nil {
				return fmt.Errorf("desktop: registering %s: %w", sc.ID, err)
			}
		}

		if !slices.Contains(paths, path) {
			paths = append(paths, path)
		}
	}

	return setShortcutPaths(ctx, run, paths)
}

// Remove deletes the shortcuts Install registered, and only them.
func Remove(ctx context.Context, run Runner) error {
	if !run.Has("gsettings") {
		return fmt.Errorf("%w: gsettings (os atalhos são do GNOME)", ErrNoTool)
	}

	paths, err := shortcutPaths(ctx, run)
	if err != nil {
		return err
	}

	ours := map[string]bool{}

	for _, sc := range Shortcuts() {
		path := basePath + sc.ID + "/"
		ours[path] = true

		for _, key := range []string{"name", "command", "binding"} {
			_, err := run.Run(ctx, "", "gsettings", "reset", itemSchema+":"+path, key)
			if err != nil {
				return fmt.Errorf("desktop: removing %s: %w", sc.ID, err)
			}
		}
	}

	kept := paths[:0]

	for _, p := range paths {
		if !ours[p] {
			kept = append(kept, p)
		}
	}

	return setShortcutPaths(ctx, run, kept)
}

// quoted matches the items of a GVariant string array: "['/a/', '/b/']".
// Settings paths have no quotes of their own.
var quoted = regexp.MustCompile(`'([^']*)'`)

func shortcutPaths(ctx context.Context, run Runner) ([]string, error) {
	out, err := run.Run(ctx, "", "gsettings", "get", listSchema, listKey)
	if err != nil {
		return nil, fmt.Errorf("desktop: reading the custom shortcuts: %w", err)
	}

	var paths []string

	for _, m := range quoted.FindAllStringSubmatch(out, -1) {
		paths = append(paths, m[1])
	}

	return paths, nil
}

func setShortcutPaths(ctx context.Context, run Runner, paths []string) error {
	items := make([]string, len(paths))
	for i, p := range paths {
		items[i] = "'" + p + "'"
	}

	value := "@as []"
	if len(items) > 0 {
		value = "[" + strings.Join(items, ", ") + "]"
	}

	_, err := run.Run(ctx, "", "gsettings", "set", listSchema, listKey, value)
	if err != nil {
		return fmt.Errorf("desktop: writing the custom shortcuts: %w", err)
	}

	return nil
}

// variantString writes value as GVariant text. gsettings parses a value
// as GVariant first, and "'/path' selecao" is not valid GVariant.
func variantString(value string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)

	return `"` + r.Replace(value) + `"`
}
