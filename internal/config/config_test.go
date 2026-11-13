package config_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/casara/biblia-cli/internal/config"
)

func TestResolveHonorsXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join("xdg", "data"))

	paths, err := config.Resolve()
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join("xdg", "data", config.App)
	if filepath.Dir(paths.StepDB) != want || filepath.Dir(paths.UBSDB) != want {
		t.Errorf("indexes in %q and %q, want %q", paths.StepDB, paths.UBSDB, want)
	}
}

func TestResolveDefaultDataHome(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the Linux default; macOS and Windows use the system's folders")
	}

	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/home/leitor")

	paths, err := config.Resolve()
	if err != nil {
		t.Fatal(err)
	}

	if want := "/home/leitor/.local/share/biblia/step.db"; paths.StepDB != want {
		t.Errorf("StepDB = %q, want %q", paths.StepDB, want)
	}
}
