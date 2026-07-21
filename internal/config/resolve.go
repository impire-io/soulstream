package config

import "os"

// SourceKind names where a resolved value came from.
type SourceKind string

// The places a value can come from, in precedence order.
const (
	SourceFlag    SourceKind = "flag"
	SourceEnv     SourceKind = "env"
	SourceProject SourceKind = "project"
	SourceUser    SourceKind = "user"
	SourceUnset   SourceKind = "unset"
)

// Source is a value's provenance: its kind plus the detail a user needs to find it
// (the environment variable's name, or the config file's path).
type Source struct {
	Kind   SourceKind
	Detail string
}

// Value is one resolved field with its provenance.
type Value struct {
	V      string
	Source Source
}

// Resolved is the full assembly both entry points consume.
type Resolved struct {
	Context  Value
	Realm    Value
	Persona  Value
	KeyFile  Value
	PinsFile Value
}

// Field pairs a canonical field name with its resolved value, for display.
type Field struct {
	Name string
	Value
}

// Fields returns the five values in stable display order.
func (r Resolved) Fields() []Field {
	return []Field{
		{"context", r.Context},
		{"realm", r.Realm},
		{"persona", r.Persona},
		{"key_file", r.KeyFile},
		{"pins_file", r.PinsFile},
	}
}

// envNames maps each field to its environment variable, in Fields() order.
var envNames = [5]string{
	"SOULSTREAM_CONTEXT",
	"SOULSTREAM_REALM",
	"SOULSTREAM_PERSONA",
	"SOULSTREAM_KEY_FILE",
	"SOULSTREAM_PINS_FILE",
}

// Resolve assembles the five fields, each independently, first non-empty source
// wins: explicit (flags the user actually passed — callers fill explicit via
// flag.Visit) > environment > nearest project file walking up from cwd > user file
// > unset. Absent config files are skipped; present-but-broken ones abort with the
// file path in the error. An environment variable that is set but empty counts as
// unset, matching the old os.Getenv defaults.
func Resolve(explicit File, cwd string) (Resolved, error) {
	var project File
	projectPath := ""
	if path, ok := findProjectFile(cwd); ok {
		f, err := loadFile(path)
		if err != nil {
			return Resolved{}, err
		}
		project, projectPath = f, path
	}

	var user File
	userPath := ""
	if path, err := userFile(); err == nil && exists(path) {
		f, err := loadFile(path)
		if err != nil {
			return Resolved{}, err
		}
		user, userPath = f, path
	}

	explicitVals := [5]string{explicit.Context, explicit.Realm, explicit.Persona, explicit.KeyFile, explicit.PinsFile}
	projectVals := [5]string{project.Context, project.Realm, project.Persona, project.KeyFile, project.PinsFile}
	userVals := [5]string{user.Context, user.Realm, user.Persona, user.KeyFile, user.PinsFile}

	var out [5]Value
	for i := range out {
		switch {
		case explicitVals[i] != "":
			out[i] = Value{explicitVals[i], Source{SourceFlag, ""}}
		case os.Getenv(envNames[i]) != "":
			out[i] = Value{os.Getenv(envNames[i]), Source{SourceEnv, envNames[i]}}
		case projectVals[i] != "":
			out[i] = Value{projectVals[i], Source{SourceProject, projectPath}}
		case userVals[i] != "":
			out[i] = Value{userVals[i], Source{SourceUser, userPath}}
		default:
			out[i] = Value{"", Source{SourceUnset, ""}}
		}
	}
	return Resolved{Context: out[0], Realm: out[1], Persona: out[2], KeyFile: out[3], PinsFile: out[4]}, nil
}
