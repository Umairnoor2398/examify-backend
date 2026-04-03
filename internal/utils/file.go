package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

func SaveUploadedFile(file multipart.File, header *multipart.FileHeader, uploadDir, subDir string) (string, error) {
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
