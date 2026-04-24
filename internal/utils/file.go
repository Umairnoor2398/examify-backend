package utils

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

var AllowedDocMIMEs = []string{"application/pdf"}
var AllowedImageMIMEs = []string{"image/jpeg", "image/png", "image/webp"}

func ValidateSquareDimensions(file multipart.File) error {
	cfg, _, err := image.DecodeConfig(file)
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}
	if err != nil {
		// Unknown format (e.g. WebP) — skip dimension check; MIME validation already ran
		return nil
	}
	if cfg.Width != cfg.Height {
		return fmt.Errorf("image must be square (%dx%d provided)", cfg.Width, cfg.Height)
	}
	return nil
}

func ValidateFileMIME(file multipart.File, allowed []string) error {
	mtype, err := mimetype.DetectReader(file)
	if err != nil {
		return fmt.Errorf("could not detect file type")
	}
	// Reset the reader so the file can be saved afterward.
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}
	for _, m := range allowed {
		if mtype.Is(m) {
			return nil
		}
	}
	return fmt.Errorf("file type %s is not allowed", mtype.String())
}

func SaveUploadedFile(file multipart.File, header *multipart.FileHeader, uploadDir, subDir string, maxSize int64) (string, error) {
	if maxSize > 0 && header.Size > maxSize {
		return "", fmt.Errorf("file size %d bytes exceeds the %d byte limit", header.Size, maxSize)
	}

	dir := filepath.Join(uploadDir, subDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s-%d%s", NewUUID(), time.Now().Unix(), ext)
	dstPath := filepath.Join(dir, filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}

	return "/uploads/" + subDir + "/" + filename, nil
}
