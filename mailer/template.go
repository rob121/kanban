package mailer

import (
	"bytes"
	"embed"
	"html/template"
	"os"
	"strings"
	"time"
)

//go:embed templates/wrapper.html
var defaultWrapperFS embed.FS

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"year": func() int { return time.Now().Year() },
	}
}

func renderTemplate(path string, args map[string]interface{}) (string, error) {
	var tmplContent string
	path = strings.TrimSpace(path)
	if path != "" {
		body, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		tmplContent = string(body)
	} else {
		body, err := defaultWrapperFS.ReadFile("templates/wrapper.html")
		if err != nil {
			return "", err
		}
		tmplContent = string(body)
	}

	tmpl, err := template.New("email").Funcs(templateFuncs()).Parse(tmplContent)
	if err != nil {
		return "", err
	}

	if args == nil {
		args = make(map[string]interface{})
	}
	mergeDefaultEmailArgs(args)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, args); err != nil {
		return "", err
	}
	return buf.String(), nil
}
