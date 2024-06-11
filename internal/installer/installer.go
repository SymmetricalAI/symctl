package installer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/SymmetricalAI/symctl/internal/logger"
)

type Url struct {
	Platform string `json:"platform"`
	Os       string `json:"os"`
	Url      string `json:"url"`
}

type Release struct {
	Name        string    `json:"name"`
	BinaryName  string    `json:"binaryName"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Urls        []Url     `json:"urls"`
	Created     time.Time `json:"created"`
}

func Install(address string) {
	logger.Debugf("Installing plugin from %s\n", address)

	executablePath, err := os.Executable()
	if err != nil {
		logger.Fatalf("Error getting executable path: %s\n", err)
	}
	logger.Debugf("Executable path: %s\n", executablePath)

	releases, err := downloadReleases(address)
	if err != nil {
		logger.Fatalf("Error downloading releases: %s\n", err)
	}

	logger.Debugf("Releases: %v\n", releases)

	url, err := pickReleaseUrl(releases)
	if err != nil {
		logger.Fatalf("Error picking release URL: %s\n", err)
	}
	logger.Debugf("Downloading from %s\n", url)

	dir, err := createTempDir()
	if err != nil {
		logger.Fatalf("Error creating temporary directory: %s\n", err)
	}

	filePath, err := downloadFile(url, dir)
	if err != nil {
		logger.Fatalf("Error downloading file: %s\n", err)
	}

	logger.Debugf("Downloaded file to %s\n", filePath)

	destDir := filepath.Join(dir, "unarchived")
	if err := os.Mkdir(destDir, 0755); err != nil {
		logger.Fatalf("Error creating destination directory: %s\n", err)
	}

	logger.Debugf("Filepath extension: %s\n", filepath.Ext(url))

	// if url ends with .gz, unarchive it
	if filepath.Ext(url) == ".gz" {
		if err := unarchiveTarGz(filePath, destDir); err != nil {
			logger.Fatalf("Error unarchiving file: %s\n", err)
		}

		logger.Debugf("Unarchived file to %s\n", destDir)
	}

	// if url ends with .zip, unzip it
	if filepath.Ext(url) == ".zip" {
		if err := unzip(filePath, destDir); err != nil {
			logger.Fatalf("Error unzipping file: %s\n", err)
		}

		logger.Debugf("Unzipped file to %s\n", destDir)
	}

	installDir, err := GetInstallDir()
	if err != nil {
		logger.Fatalf("Error getting install directory: %s\n", err)
	}

	if err := copyDir(destDir, installDir); err != nil {
		logger.Fatalf("Error copying to install directory: %s\n", err)
	}

	logger.Debugf("Installed to %s\n", installDir)
}

func downloadReleases(address string) ([]Release, error) {
	resp, err := http.Get(address)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func pickReleaseUrl(releases []Release) (string, error) {
	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found")
	}
	pickedRelease := releases[0]
	for _, release := range releases {
		if release.Created.After(pickedRelease.Created) {
			pickedRelease = release
		}
	}

	logger.Debugf("OS: %s\n", runtime.GOOS)
	logger.Debugf("Arch: %s\n", runtime.GOARCH)

	var pickedUrl *Url
	for _, url := range pickedRelease.Urls {
		if url.Platform == runtime.GOARCH && url.Os == runtime.GOOS {
			pickedUrl = &url
			break
		}
	}
	// if picked url is nil check if there is url with any platform and os
	if pickedUrl == nil {
		for _, url := range pickedRelease.Urls {
			if url.Platform == "any" && url.Os == "any" {
				pickedUrl = &url
				break
			}
		}
	}

	if pickedUrl == nil {
		return "", fmt.Errorf("no suitable URL found")
	}
	return pickedUrl.Url, nil
}

func createTempDir() (string, error) {
	dir, err := os.MkdirTemp("", "symctl-")
	if err != nil {
		return "", err
	}
	logger.Debugf("Temporary directory created: %s\n", dir)
	return dir, nil
}

func downloadFile(url, dir string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	filename := filepath.Base(resp.Request.URL.Path)
	filePath := filepath.Join(dir, filename)

	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = out.Close()
	}()

	_, err = io.Copy(out, resp.Body)
	return filePath, err
}

func unzip(zipFile, destDir string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer func() {
		_ = zipReader.Close()
	}()

	for _, file := range zipReader.File {
		fPath := filepath.Join(destDir, file.Name)

		if !strings.HasPrefix(fPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", fPath)
		}

		if file.FileInfo().IsDir() {
			err := os.MkdirAll(fPath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			_ = outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to check for its error
		if closeErr := outFile.Close(); closeErr != nil {
			_ = rc.Close() // Ignore the error from rc.Close() as we're already handling an error
			return closeErr
		}
		_ = rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func unarchiveTarGz(tarGzPath, destDir string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer func() {
		_ = gzr.Close()
	}()

	tarReader := tar.NewReader(gzr)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}

		path := filepath.Join(destDir, header.Name)

		logger.Debugf("Unarchiving %s\n", path)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				_ = outFile.Close()
				return err
			}
			// ensure mode is taken from source file
			if err := outFile.Chmod(os.FileMode(header.Mode)); err != nil {
				_ = outFile.Close()
				return err
			}
			_ = outFile.Close()
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = dstFile.Close()
	}()

	_, err = io.Copy(dstFile, sourceFile)
	if err != nil {
		return err
	}
	// ensure mode is taken from source file
	return dstFile.Chmod(getMode(src))
}

func getMode(src string) os.FileMode {
	info, err := os.Stat(src)
	if err != nil {
		return 0
	}
	return info.Mode()
}

// copyDir recursively copies a directory tree, overwriting existing files if they exist.
// Source directory must exist.
func copyDir(src string, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Use MkdirAll to create the directory if it doesn't exist (no error if it already exists)
			return os.MkdirAll(dstPath, info.Mode())
		}

		// For files, just call copyFile which overwrites by default
		return copyFile(path, dstPath)
	})

	return err
}

func GetInstallDir() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	executableDir := filepath.Dir(executablePath)
	// if executable dir ends with "bin", assume we're in a "bin" directory and go up one level
	if filepath.Base(executableDir) == "bin" {
		executableDir = filepath.Dir(executableDir)
	}
	return executableDir, nil
}
