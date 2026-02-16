package util

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/huskyci-org/huskyCI/cli/config"
	"github.com/huskyci-org/huskyCI/cli/errorcli"
)

// GetAllAllowedFilesAndDirsFromPath returns a list of all files and dirs allowed to be zipped
func GetAllAllowedFilesAndDirsFromPath(path string) ([]string, error) {

	var allFilesAndDirNames []string

	filesAndDirs, err := ioutil.ReadDir(path)
	if err != nil {
		return allFilesAndDirNames, err
	}
	for _, file := range filesAndDirs {
		fileName := file.Name()
		if err := checkFileExtension(fileName); err != nil {
			continue
		} else {
			// Return full path for zip creation
			fullPath := filepath.Join(path, fileName)
			allFilesAndDirNames = append(allFilesAndDirNames, fullPath)
		}
	}

	return allFilesAndDirNames, nil
}

// CompressFiles compress all files into a zip and return its full path and an error
func CompressFiles(allFilesAndDirNames []string) (string, error) {

	var fullFilePath string

	fullFilePath, err := config.GetHuskyZipFilePath()
	if err != nil {
		return fullFilePath, err
	}

	// Create zip file using standard library (more secure, no path traversal vulnerability)
	zipFile, err := os.Create(fullFilePath)
	if err != nil {
		return fullFilePath, err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add each file/directory to the zip
	for _, filePath := range allFilesAndDirNames {
		if err := addToZip(zipWriter, filePath); err != nil {
			return fullFilePath, err
		}
	}

	return fullFilePath, nil
}

// addToZip adds a file or directory to the zip archive
func addToZip(zipWriter *zip.Writer, filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		// Recursively add directory contents
		return filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip the directory itself, only add files
			if info.IsDir() {
				return nil
			}

			// Create relative path for zip entry (prevents path traversal)
			relPath, err := filepath.Rel(filepath.Dir(filePath), path)
			if err != nil {
				return err
			}

			// Sanitize path to prevent path traversal
			relPath = filepath.ToSlash(relPath)
			// Check for path traversal attempts
			if filepath.IsAbs(relPath) || relPath == ".." || strings.HasPrefix(relPath, "../") {
				return fmt.Errorf("illegal file path: %s", relPath)
			}

			return addFileToZip(zipWriter, path, relPath)
		})
	}

	// Add single file
	relPath := filepath.Base(filePath)
	return addFileToZip(zipWriter, filePath, relPath)
}

// addFileToZip adds a single file to the zip archive
func addFileToZip(zipWriter *zip.Writer, filePath, zipPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := zipWriter.Create(zipPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// GetZipFriendlySize returns the size of a friendly zip file size based on its destination
func GetZipFriendlySize(destination string) (string, error) {

	var friendlySize string

	file, err := os.Open(destination) // #nosec -> this destination is always "$HOME/.huskyci/compressed-code.zip"
	if err != nil {
		return friendlySize, err
	}

	fi, err := file.Stat()
	if err != nil {
		return friendlySize, err
	}

	if err := file.Close(); err != nil {
		return friendlySize, err
	}

	friendlySize = byteCountSI(fi.Size())
	return friendlySize, nil
}

// DeleteHuskyFile will delete the huskyCI file present at "$HOME/.huskyci/compressed-code.zip"
func DeleteHuskyFile(destination string) error {
	return os.Remove(destination)
}

func checkFileExtension(file string) error {
	extensionFound := filepath.Ext(file)
	switch extensionFound {
	case "":
		return nil
	case ".jpg", ".png", ".gif", ".webp", ".tiff", ".psd", ".raw", ".bmp", ".heif", ".indd", ".jpeg", ".svg", ".ai", ".eps", ".pdf":
		return errorcli.ErrInvalidExtension
	case ".webm", ".mpg", ".mp2", ".mpeg", ".mpe", ".mpv", ".ogg", ".mp4", ".m4p", ".m4v", ".avi", ".wmv", ".mov", ".qt", ".flv", ".swf", ".avchd":
		return errorcli.ErrInvalidExtension
	default:
		return nil
	}
}

func byteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

// AppendIfMissing will append an item in a slice if it is missing
func AppendIfMissing(slice []string, s string) []string {
	for _, ele := range slice {
		if ele == s {
			return slice
		}
	}
	return append(slice, s)
}

// NormalizeURL removes trailing slashes from a URL
func NormalizeURL(url string) string {
	if strings.HasSuffix(url, "/") {
		return url[:len(url)-1]
	}
	return url
}
