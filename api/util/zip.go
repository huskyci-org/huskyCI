package util

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// ZipStorageDir is the directory where uploaded zip files are stored
	ZipStorageDir = "/tmp/huskyci-zips"
)

// EnsureZipStorageDir creates the zip storage directory if it doesn't exist
func EnsureZipStorageDir() error {
	if err := os.MkdirAll(ZipStorageDir, 0755); err != nil {
		return fmt.Errorf("failed to create zip storage directory: %w", err)
	}
	return nil
}

// GetZipFilePath returns the path where a zip file for a given RID should be stored
func GetZipFilePath(RID string) string {
	return filepath.Join(ZipStorageDir, fmt.Sprintf("%s.zip", RID))
}

// ExtractZip extracts a zip file to a destination directory
func ExtractZip(zipPath, destDir string) error {
	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open zip file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	// Extract files
	for _, f := range r.File {
		err := extractFile(f, destDir)
		if err != nil {
			return fmt.Errorf("failed to extract file %s: %w", f.Name, err)
		}
	}

	return nil
}

// extractFile extracts a single file from the zip archive
func extractFile(f *zip.File, destDir string) error {
	// Sanitize file path to prevent path traversal
	path := filepath.Join(destDir, f.Name)
	
	// Check for path traversal attempts
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", f.Name)
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Create file
	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Copy file contents
	_, err = io.Copy(outFile, rc)
	return err
}

// CleanupZip removes a zip file and its extracted directory
func CleanupZip(RID string) error {
	zipPath := GetZipFilePath(RID)
	extractedDir := filepath.Join(ZipStorageDir, RID)

	// Remove zip file
	if err := os.Remove(zipPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove zip file: %w", err)
	}

	// Remove extracted directory
	if err := os.RemoveAll(extractedDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove extracted directory: %w", err)
	}

	return nil
}

// GetExtractedDir returns the path where a zip file for a given RID should be extracted
func GetExtractedDir(RID string) string {
	return filepath.Join(ZipStorageDir, RID)
}

// IsFileURL checks if a URL is a file:// URL
func IsFileURL(url string) bool {
	return strings.HasPrefix(url, "file://")
}

// ExtractRIDFromFileURL extracts the RID from a file:// URL
func ExtractRIDFromFileURL(url string) string {
	if !IsFileURL(url) {
		return ""
	}
	// file://<RID> -> extract RID
	parts := strings.Split(url, "://")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}
