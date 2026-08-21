package util

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"booking-system-api/internal/config"
)

var allowedTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"application/pdf": true,
}

// SaveUploadedFile saves a file to uploads/{category}/YYYY/MM/.
// category should be one of: "vehicle", "room", "booking", "profile".
func SaveUploadedFile(fh *multipart.FileHeader, category string) (filePath string, err error) {
	if fh.Size > config.C.MaxFileSizeMB*1024*1024 {
		return "", fmt.Errorf("file size exceeds %dMB limit", config.C.MaxFileSizeMB)
	}

	contentType := fh.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		// fallback: check extension
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".pdf": true}
		if !allowed[ext] {
			return "", fmt.Errorf("file type not allowed")
		}
	}

	if category == "" {
		category = "misc"
	}
	dir := filepath.Join(config.C.UploadDir, category, time.Now().Format("2006/01"))
	if err = os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	uniqueName := GenerateUniqueFilename(fh.Filename)
	dst := filepath.Join(dir, uniqueName)

	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}

	written, copyErr := io.Copy(out, src)
	closeErr := out.Close()

	// io.Copy stops on the first non-EOF error and returns it directly (unlike
	// the old manual read/write loop here, which treated every readErr —
	// including a genuine failed read, e.g. client aborted mid-upload — as a
	// normal end-of-file and returned success with a truncated/empty file on
	// disk while the DB row still got created).
	if copyErr != nil {
		os.Remove(dst)
		return "", fmt.Errorf("failed to save file: %w", copyErr)
	}
	if closeErr != nil {
		os.Remove(dst)
		return "", fmt.Errorf("failed to save file: %w", closeErr)
	}
	if written != fh.Size {
		os.Remove(dst)
		return "", fmt.Errorf("incomplete upload: wrote %d of %d bytes", written, fh.Size)
	}

	// return relative path from uploads root — SELALU pakai forward slash agar
	// valid sebagai URL (di Windows filepath.Rel menghasilkan backslash yang
	// merusak path foto: /uploads/maintenance\2025\07\x.jpg).
	rel, _ := filepath.Rel(config.C.UploadDir, dst)
	return filepath.ToSlash(rel), nil
}

func DeleteUploadedFile(relPath string) {
	abs := filepath.Join(config.C.UploadDir, relPath)
	_ = os.Remove(abs)
}

func GetFileMimeType(fh *multipart.FileHeader) string {
	ct := fh.Header.Get("Content-Type")
	if ct != "" {
		return ct
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	m := map[string]string{
		".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".png": "image/png", ".gif": "image/gif",
		".webp": "image/webp", ".pdf": "application/pdf",
	}
	if v, ok := m[ext]; ok {
		return v
	}
	return "application/octet-stream"
}
