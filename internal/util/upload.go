package util

import (
	"fmt"
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
	defer out.Close()

	buf := make([]byte, 32*1024)
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return "", werr
			}
		}
		if readErr != nil {
			break
		}
	}

	// return relative path from uploads root
	rel, _ := filepath.Rel(config.C.UploadDir, dst)
	return rel, nil
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
