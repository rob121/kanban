package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rob121/kanban/internal/permissions"
)

type Renderer struct {
	pages    map[string]*template.Template
	partials *template.Template
	mu       sync.RWMutex
}

func NewRenderer(views embed.FS) (*Renderer, error) {
	funcs := template.FuncMap{
		"tagClass": tagColor,
		"tagColor":        tagColor,
		"boardHexColor":   boardHexColor,
		"boardHeaderStyle": boardHeaderStyle,
		"priorityLabel":   priorityLabel,
		"add":             func(a, b int) int { return a + b },
		"sub":             func(a, b int) int { return a - b },
		"initials":        initials,
		"dict":            dict,
		"formatBytes":     formatBytes,
		"dueDateSummary":  dueDateSummary,
		"canRemoveAttachment": func(access permissions.Access, attachmentUserID, currentUserID uint) bool {
			return access.CanRemoveAttachment(attachmentUserID, currentUserID)
		},
	}

	layout, err := views.ReadFile("templates/layouts/base.html")
	if err != nil {
		return nil, fmt.Errorf("read base layout: %w", err)
	}

	r := &Renderer{
		pages:    make(map[string]*template.Template),
		partials: template.New("partials").Funcs(funcs),
	}

	err = fs.WalkDir(views, "templates/partials", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}
		body, err := views.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := r.partials.Parse(string(body)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = fs.WalkDir(views, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}
		if strings.HasPrefix(path, "templates/layouts/") || strings.HasPrefix(path, "templates/partials/") {
			return nil
		}

		page, err := views.ReadFile(path)
		if err != nil {
			return err
		}

		name := strings.TrimPrefix(path, "templates/")
		tmpl := template.New(name).Funcs(funcs)
		if _, err := tmpl.Parse(string(layout)); err != nil {
			return err
		}
		if _, err := tmpl.Parse(string(page)); err != nil {
			return err
		}
		partialFiles, _ := fs.Glob(views, "templates/partials/*.html")
		for _, pf := range partialFiles {
			body, err := views.ReadFile(pf)
			if err != nil {
				return err
			}
			if _, err := tmpl.Parse(string(body)); err != nil {
				return err
			}
		}
		r.pages[name] = tmpl
		return nil
	})
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Renderer) Render(w http.ResponseWriter, name string, data any) error {
	r.mu.RLock()
	tmpl, ok := r.pages[name]
	r.mu.RUnlock()
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return nil
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(w, "base", data)
}

func (r *Renderer) RenderPartial(w http.ResponseWriter, name string, data any) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return r.partials.ExecuteTemplate(w, name, data)
}

func tagColor(color string) string {
	switch strings.ToLower(strings.TrimSpace(color)) {
	case "primary", "danger", "warning", "info", "success", "secondary":
		return "text-bg-" + strings.ToLower(color)
	default:
		return "text-bg-primary"
	}
}

func priorityLabel(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "high":
		return "High"
	case "low":
		return "Low"
	default:
		return "Medium"
	}
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func dueDateSummary(due *time.Time) string {
	return dueDateSummaryAt(due, time.Now().UTC())
}

func dueDateSummaryAt(due *time.Time, now time.Time) string {
	if due == nil {
		return ""
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, time.UTC)
	days := int(dueDay.Sub(today).Hours() / 24)
	dateStr := due.Format("Jan 2")

	if days >= 60 {
		return dateStr
	}

	var relative string
	switch {
	case days == 0:
		relative = "Today"
	case days == 1:
		relative = "1 Day"
	case days > 1:
		relative = fmt.Sprintf("%d Days", days)
	case days == -1:
		relative = "1 Day ago"
	default:
		relative = fmt.Sprintf("%d Days ago", -days)
	}

	return relative + " · " + dateStr
}

func initials(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) == 1 {
		if len(parts[0]) > 0 {
			return strings.ToUpper(string(parts[0][0]))
		}
		return "?"
	}
	return strings.ToUpper(string(parts[0][0]) + string(parts[len(parts)-1][0]))
}

func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict: expected even number of arguments")
	}
	m := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: keys must be strings")
		}
		m[key] = values[i+1]
	}
	return m, nil
}

type PageData struct {
	Title     string
	User      any
	Flash     string
	Error     string
	CSRFToken string
	BrandName  string
	BrandMark  string
	BrandColor string
	UserTheme  string
	Data       any
}
