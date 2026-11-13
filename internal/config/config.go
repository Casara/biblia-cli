// Package config loads the reader's preferences and resolves where the
// program keeps its files. On Linux these follow the XDG base
// directories:
//
//	config  ~/.config/biblia/config.json
//	cache   ~/.cache/biblia/cache.db         (verses already looked up)
//	data    ~/.local/share/biblia/step.db    (STEPBible index) and raw/
//	        ~/.local/share/biblia/ubs.db     (UBS index) and ubs/
//
// On macOS and Windows each base is the system's own (see dataHome).
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// App is the directory name under each XDG base.
const App = "biblia"

// Config holds the reader's preferences. Every field is optional.
type Config struct {
	// Versions are the translation ids used when -v is not given.
	Versions []string `json:"versoes"`
	// Format is the default output format of a lookup.
	Format string `json:"formato"`
	// FilterFormat is the default format of "biblia filtro" (editors).
	FilterFormat string `json:"formato_filtro"`
	// Templates adds or replaces output formats (name → Go template).
	Templates map[string]string `json:"modelos"`
	// Language is the reader's language ("pt", "en", "es"...): it picks
	// the default dictionaries and the language of grammar descriptions.
	Language string `json:"idioma"`
	// Dictionaries are the dictionary ids in order of preference
	// ("ubs-hebraico-pt", "ubs-grego-es", "step"); empty means the
	// defaults for Language.
	Dictionaries []string `json:"dicionarios"`
}

// Default is used for whatever the file does not set.
func Default() Config {
	return Config{
		Versions:     []string{"ara"},
		Format:       "texto",
		FilterFormat: "linha",
		Templates:    map[string]string{},
		Language:     "pt",
		Dictionaries: nil,
	}
}

// Paths are the resolved file locations.
type Paths struct {
	Config, Cache, StepDB, RawDir, UBSDB, UBSRawDir string
}

// Resolve computes the paths. BIBLIA_CONFIG overrides the config file.
func Resolve() (Paths, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("config: %w", err)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return Paths{}, fmt.Errorf("config: %w", err)
	}

	dataDir, err := dataHome(runtime.GOOS)
	if err != nil {
		return Paths{}, fmt.Errorf("config: %w", err)
	}

	p := Paths{
		Config: filepath.Join(cfgDir, App, "config.json"),
		Cache:  filepath.Join(cacheDir, App, "cache.db"),
		StepDB: filepath.Join(dataDir, App, "step.db"),
		RawDir: filepath.Join(dataDir, App, "raw"),
		UBSDB:  filepath.Join(dataDir, App, "ubs.db"),
		// UBSRawDir holds the UBS files as downloaded.
		UBSRawDir: filepath.Join(dataDir, App, "ubs"),
	}

	if env := os.Getenv("BIBLIA_CONFIG"); env != "" {
		p.Config = env
	}

	return p, nil
}

// dataHome is the base directory for the downloaded data and indexes,
// which the standard library has no function for: XDG_DATA_HOME or
// ~/.local/share on Linux and the other Unix systems; "Application
// Support" on macOS, next to the config (os.UserConfigDir); the local,
// not roaming, application data on Windows (%LocalAppData%, which
// os.UserCacheDir returns), since 200 MB of indexes should not follow the
// user between machines.
func dataHome(goos string) (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir, nil
	}

	switch goos {
	case "darwin":
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("data directory: %w", err)
		}

		return dir, nil
	case "windows":
		dir, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("data directory: %w", err)
		}

		return dir, nil
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("data directory: %w", err)
		}

		return filepath.Join(home, ".local", "share"), nil
	}
}

// Load reads the config file over the defaults. A missing file is not
// an error.
func Load(path string) (Config, error) {
	c := Default()

	raw, err := os.ReadFile(path) //nolint:gosec // the reader's own config file
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}

	if err != nil {
		return c, fmt.Errorf("config: reading %s: %w", path, err)
	}

	var file Config
	err = json.Unmarshal(raw, &file)
	if err != nil {
		return c, fmt.Errorf("config: %s: %w", path, err)
	}

	if len(file.Versions) > 0 {
		c.Versions = file.Versions
	}

	if file.Format != "" {
		c.Format = file.Format
	}

	if file.FilterFormat != "" {
		c.FilterFormat = file.FilterFormat
	}

	if file.Templates != nil {
		c.Templates = file.Templates
	}

	if file.Language != "" {
		c.Language = file.Language
	}

	if len(file.Dictionaries) > 0 {
		c.Dictionaries = file.Dictionaries
	}

	return c, nil
}
