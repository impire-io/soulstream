// Package config assembles the five who-acts-where fields (context, realm, persona,
// key file, pins file) from up to four sources: explicit flags, environment
// variables, the nearest project config file, and the user config file. It is pure
// client-side convenience — it imports no NATS and never touches the wire.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ProjectFileName is the per-project config file, discovered by walking up from the
// working directory (nearest file only — like git finding .git).
const ProjectFileName = ".soulstream.json"

// File is the shape of both config files, and of explicitly-passed flag values.
// Fields name an identity; they can never carry credentials — signing keys live in
// the local keystore, resolved per realm+persona.
type File struct {
	Context  string `json:"context"`
	Realm    string `json:"realm"`
	Persona  string `json:"persona"`
	KeyFile  string `json:"key_file"`
	PinsFile string `json:"pins_file"`
}

// loadFile reads a config file strictly: unknown fields and malformed JSON fail
// with the file path in the error, so a typo can never silently fall through to a
// different identity. Relative key/pins paths are resolved against the file's own
// directory, so a project file's "./keys/ci.ed25519" means the project tree.
func loadFile(path string) (File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("config %s: %w", path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var f File
	if err := dec.Decode(&f); err != nil {
		return File{}, fmt.Errorf("config %s: %w", path, err)
	}
	dir := filepath.Dir(path)
	if f.KeyFile != "" && !filepath.IsAbs(f.KeyFile) {
		f.KeyFile = filepath.Join(dir, f.KeyFile)
	}
	if f.PinsFile != "" && !filepath.IsAbs(f.PinsFile) {
		f.PinsFile = filepath.Join(dir, f.PinsFile)
	}
	return f, nil
}

// findProjectFile walks from dir upward to the filesystem root and returns the
// nearest project config file. Only the nearest file counts — nested project files
// never stack. A missing file is not an error; ok reports whether one was found.
func findProjectFile(dir string) (path string, ok bool) {
	dir = filepath.Clean(dir)
	for {
		candidate := filepath.Join(dir, ProjectFileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// userFile returns the user-level config path: config.json in the Soulstream user
// config dir, beside the keys/ and pins/ the keystore already keeps there.
func userFile() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: locate user config dir: %w", err)
	}
	return filepath.Join(base, "soulstream", "config.json"), nil
}

// exists reports whether path is an existing regular file; used to distinguish
// "absent — skip silently" from "present but unreadable — fail loud".
func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
