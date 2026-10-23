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
	"yaml": yamlScalar,
}).Parse(starterSource))

// yamlScalar renders v on one line; a multi-line value is double-quoted, since a block scalar would need the template's indentation
func yamlScalar(v string) (string, error) {
	var node any = v
	if strings.Contains(v, "\n") {
		node = &yaml.Node{Kind: yaml.ScalarNode, Style: yaml.DoubleQuotedStyle, Value: v}
	}
	out, err := yaml.Marshal(node)
	return strings.TrimSuffix(string(out), "\n"), err
}

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
