package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// isolate redirects the user config dir into a temp home and blanks the five
// environment variables, so tests never see the developer's real configuration.
// It returns the user config file path (not yet created).
func isolate(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)                                      // darwin: ~/Library/Application Support
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".xdgcfg")) // linux: honoured directly
	for _, name := range envNames {
		t.Setenv(name, "")
	}
	path, err := userFile()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveNoSourcesIsAllUnset(t *testing.T) {
	isolate(t)
	r, err := Resolve(File{}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range r.Fields() {
		if f.V != "" || f.Source.Kind != SourceUnset {
			t.Errorf("%s = %+v, want unset", f.Name, f.Value)
		}
	}
}

func TestResolvePerFieldChain(t *testing.T) {
	userPath := isolate(t)
	cwd := t.TempDir()

	// User file supplies the machine-wide context; project file names realm+persona.
	write(t, userPath, `{"context":"personal","realm":"user-realm"}`)
	projectPath := filepath.Join(cwd, ProjectFileName)
	write(t, projectPath, `{"realm":"impire","persona":"daan"}`)

	r, err := Resolve(File{}, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if r.Context.V != "personal" || r.Context.Source.Kind != SourceUser || r.Context.Source.Detail != userPath {
		t.Errorf("context = %+v, want personal from user file", r.Context)
	}
	if r.Realm.V != "impire" || r.Realm.Source.Kind != SourceProject || r.Realm.Source.Detail != projectPath {
		t.Errorf("realm = %+v, want impire from project file (project beats user)", r.Realm)
	}
	if r.Persona.V != "daan" || r.Persona.Source.Kind != SourceProject {
		t.Errorf("persona = %+v, want daan from project file", r.Persona)
	}

	// Environment beats the files...
	t.Setenv("SOULSTREAM_REALM", "from-env")
	if r, err = Resolve(File{}, cwd); err != nil {
		t.Fatal(err)
	}
	if r.Realm.V != "from-env" || r.Realm.Source.Kind != SourceEnv || r.Realm.Source.Detail != "SOULSTREAM_REALM" {
		t.Errorf("realm = %+v, want from-env via env", r.Realm)
	}

	// ...and an explicit flag beats everything.
	if r, err = Resolve(File{Realm: "from-flag"}, cwd); err != nil {
		t.Fatal(err)
	}
	if r.Realm.V != "from-flag" || r.Realm.Source.Kind != SourceFlag {
		t.Errorf("realm = %+v, want from-flag via flag", r.Realm)
	}
}

func TestResolveNearestProjectFileDoesNotStack(t *testing.T) {
	isolate(t)
	root := t.TempDir()
	deep := filepath.Join(root, "sub")
	write(t, filepath.Join(root, ProjectFileName), `{"realm":"outer","context":"outer-ctx"}`)
	write(t, filepath.Join(deep, ProjectFileName), `{"realm":"inner"}`)

	r, err := Resolve(File{}, deep)
	if err != nil {
		t.Fatal(err)
	}
	if r.Realm.V != "inner" {
		t.Errorf("realm = %q, want inner", r.Realm.V)
	}
	// The outer file is not consulted at all — even for fields the inner one lacks.
	if r.Context.Source.Kind != SourceUnset {
		t.Errorf("context = %+v, want unset (no stacking across project files)", r.Context)
	}
}

func TestResolveEmptyEnvCountsAsUnset(t *testing.T) {
	isolate(t)
	cwd := t.TempDir()
	write(t, filepath.Join(cwd, ProjectFileName), `{"persona":"from-file"}`)
	t.Setenv("SOULSTREAM_PERSONA", "") // set but empty — must not mask the file

	r, err := Resolve(File{}, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if r.Persona.V != "from-file" || r.Persona.Source.Kind != SourceProject {
		t.Errorf("persona = %+v, want from-file", r.Persona)
	}
}

func TestResolveBrokenFileFailsLoud(t *testing.T) {
	isolate(t)
	cwd := t.TempDir()
	path := filepath.Join(cwd, ProjectFileName)
	write(t, path, `{"presona":"typo"}`)

	if _, err := Resolve(File{}, cwd); err == nil || !strings.Contains(err.Error(), path) {
		t.Errorf("err = %v, want unknown-field error naming %s", err, path)
	}
}
