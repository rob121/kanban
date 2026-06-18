package handlers

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/rob121/kanban/internal/auth"
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/permissions"
	"github.com/rob121/kanban/internal/storage"
)

const maxAttachmentBytes = 10 << 20 // 10 MB

type AttachmentHandler struct {
	Render *Renderer
}

func (h *AttachmentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	cardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var card models.Card
	if err := database.DB.First(&card, cardID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	access, ok := requireBoardPerm(w, r, user, card.BoardID, permissions.Access.CanAttach)
	if !ok {
		return
	}

	if err := auth.ParseRequestForm(r); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if header.Size > maxAttachmentBytes {
		http.Error(w, "file too large (max 10 MB)", http.StatusBadRequest)
		return
	}

	filename := sanitizeFilename(header.Filename)
	if filename == "" {
		http.Error(w, "invalid filename", http.StatusBadRequest)
		return
	}
	if blockedAttachmentExt(filename) {
		http.Error(w, "file type not allowed", http.StatusBadRequest)
		return
	}

	storedName, size, err := storage.SaveAttachment(io.LimitReader(file, maxAttachmentBytes+1), filename)
	if err != nil {
		http.Error(w, "could not save file", http.StatusInternalServerError)
		return
	}
	if size > maxAttachmentBytes {
		_ = storage.DeleteAttachment(storedName)
		http.Error(w, "file too large (max 10 MB)", http.StatusBadRequest)
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(filename))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	attachment := models.CardAttachment{
		CardID:      card.ID,
		UserID:      user.ID,
		Filename:    filename,
		StoredName:  storedName,
		ContentType: contentType,
		Size:        size,
	}
	if err := database.DB.Create(&attachment).Error; err != nil {
		_ = storage.DeleteAttachment(storedName)
		http.Error(w, "could not save attachment", http.StatusInternalServerError)
		return
	}
	database.DB.Preload("User").First(&attachment, attachment.ID)

	if wantsPartial(r) {
		_ = h.Render.RenderPartial(w, "attachment", map[string]any{
			"Attachment": attachment,
			"Access":       access,
			"User":         user,
		})
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h *AttachmentHandler) Serve(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	attachmentID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var attachment models.CardAttachment
	if err := database.DB.Preload("Card").First(&attachment, attachmentID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if _, ok := requireBoardPerm(w, r, user, attachment.Card.BoardID, permissions.Access.CanView); !ok {
		return
	}

	path, err := storage.StoredPath(attachment.StoredName)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Disposition", contentDisposition(attachment.Filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, path)
}

func (h *AttachmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	attachmentID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var attachment models.CardAttachment
	if err := database.DB.Preload("Card").First(&attachment, attachmentID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	access, ok := requireBoardPerm(w, r, user, attachment.Card.BoardID, permissions.Access.CanView)
	if !ok {
		return
	}
	if !access.CanRemoveAttachment(attachment.UserID, user.ID) {
		http.NotFound(w, r)
		return
	}

	_ = storage.DeleteAttachment(attachment.StoredName)
	database.DB.Delete(&attachment)

	if wantsPartial(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == '"' || r == '\'' {
			return -1
		}
		return r
	}, name)
	if len(name) > 200 {
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		if len(ext) > 20 {
			ext = ""
		}
		if len(base) > 200-len(ext) {
			base = base[:200-len(ext)]
		}
		name = base + ext
	}
	return strings.TrimSpace(name)
}

func blockedAttachmentExt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".exe", ".bat", ".cmd", ".com", ".msi", ".scr", ".ps1", ".sh", ".dll", ".app":
		return true
	default:
		return false
	}
}

func contentDisposition(filename string) string {
	safe := strings.NewReplacer(`\`, `_`, `"`, `_`).Replace(filename)
	return fmt.Sprintf(`attachment; filename="%s"`, safe)
}

func loadCardAttachments(cardID uint) []models.CardAttachment {
	var attachments []models.CardAttachment
	database.DB.Where("card_id = ?", cardID).
		Preload("User").
		Order("created_at desc").
		Find(&attachments)
	return attachments
}
