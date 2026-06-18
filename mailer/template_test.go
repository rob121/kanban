package mailer

import (
	"html/template"
	"strings"
	"testing"
)

func TestRenderTemplateWrapper(t *testing.T) {
	Configure(Settings{
		BaseURL:    "https://kanban.example.com",
		SiteName:   "Team Boards",
		BrandMark:  "TB",
		BrandColor: "#4f46e5",
	})

	args := map[string]interface{}{
		"Body":      template.HTML(`<p style="margin:0;">Hello <strong>world</strong>.</p>`),
		"Preheader": "Quick update from Team Boards",
	}

	out, err := renderTemplate("", args)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"<!DOCTYPE html PUBLIC",
		"Team Boards",
		"TB",
		"#4f46e5",
		"Hello <strong>world</strong>",
		"https://kanban.example.com",
		"email-preheader",
		"Quick update from Team Boards",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered output missing %q\n%s", want, out)
		}
	}
}
