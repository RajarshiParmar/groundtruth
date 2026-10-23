package config

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

//go:embed init.yaml.tmpl
var starterSource string

// yaml quotes a value only when YAML needs it, so names like O'Brien: Jr survive the round trip
var starterTemplate = template.Must(template.New("init").Funcs(template.FuncMap{
	"yaml": func(v string) (string, error) {
		out, err := yaml.Marshal(v)
		return strings.TrimSuffix(string(out), "\n"), err
	},
}).Parse(starterSource))

// StarterValues fills in the config written by groundtruth init
type StarterValues struct {
	Path, Name, Email string
	BaseDir           bool
}

// Starter renders the starter config; config.example.yaml is this with placeholder values
func Starter(v StarterValues) []byte {
	var buf bytes.Buffer
	if err := starterTemplate.Execute(&buf, v); err != nil {
		// The template is fixed and its values are plain strings, so this cannot fail at run time
		panic(err)
	}
	return buf.Bytes()
}
