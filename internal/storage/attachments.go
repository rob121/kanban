package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rob121/kanban/internal/config"
)

func AttachmentsDir() string {
	dir := strings.TrimSpace(config.C.AttachmentsDir)
	if dir == "" {
		dir = "data/attachments"
	}
	return dir
}

func EnsureAttachmentsDir() error {
	return os.MkdirAll(AttachmentsDir(), 0o750)
}

func StoredPath(storedName string) (string, error) {
	if storedName == "" || strings.Contains(storedName, "..") || strings.ContainsAny(storedName, `/\`) {
		return "", fmt.Errorf("invalid stored name")
	}
	return filepath.Join(AttachmentsDir(), storedName), nil
}

func SaveAttachment(src io.Reader, originalName string) (storedName string, size int64, err error) {
	if err := EnsureAttachmentsDir(); err != nil {
		return "", 0, err
	}

	ext := filepath.Ext(originalName)
	if len(ext) > 16 {
		ext = ""
	}
	storedName, err = randomStoredName(ext)
	if err != nil {
		return "", 0, err
	}
	path, err := StoredPath(storedName)
	if err != nil {
		return "", 0, err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o640)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	written, err := io.Copy(f, src)
	if err != nil {
		_ = os.Remove(path)
		return "", 0, err
	}
	return storedName, written, nil
}

func DeleteAttachment(storedName string) error {
	path, err := StoredPath(storedName)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func randomStoredName(ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + strings.ToLower(ext), nil
}
