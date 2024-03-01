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

func Install(address string, args []string) {
	fmt.Printf("Installing plugin from %s with args %v\n", address, args)

	executablePath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting executable path: %s\n", err)
		return
	}
	fmt.Printf("Executable path: %s\n", executablePath)

	releases, err := downloadReleases(address)
	if err != nil {
		fmt.Printf("Error downloading releases: %s\n", err)
		return
	}

	fmt.Printf("Releases: %v\n", releases)

	url, err := pickReleaseUrl(releases)
	if err != nil {
		fmt.Printf("Error picking release URL: %s\n", err)
		return
	}
	fmt.Printf("Downloading from %s\n", url)

	dir, err := createTempDir()
	if err != nil {
		fmt.Printf("Error creating temporary directory: %s\n", err)
		return
	}

	filePath, err := downloadFile(url, dir)
	if err != nil {
		fmt.Printf("Error downloading file: %s\n", err)
		return
	}

	fmt.Printf("Downloaded file to %s\n", filePath)

	destDir := filepath.Join(dir, "unarchived")
	if err := os.Mkdir(destDir, 0755); err != nil {
		fmt.Printf("Error creating destination directory: %s\n", err)
		return
	}

	fmt.Println("Filepath extension: ", filepath.Ext(url))

	// if url ends with .tar.gz, unarchive it
	if filepath.Ext(url) == ".gz" {
		if err := unarchiveTarGz(filePath, destDir); err != nil {
			fmt.Printf("Error unarchiving file: %s\n", err)
			return
		}

		fmt.Printf("Unarchived file to %s\n", destDir)
	}

	// if url ends with .zip, unzip it
	if filepath.Ext(url) == ".zip" {
		if err := unzip(filePath, destDir); err != nil {
			fmt.Printf("Error unzipping file: %s\n", err)
			return
		}

		fmt.Printf("Unzipped file to %s\n", destDir)
	}

	installDir, err := getInstallDir()
	if err != nil {
		fmt.Printf("Error getting install directory: %s\n", err)
		return
	}

	if err := copyDir(destDir, installDir); err != nil {
		fmt.Printf("Error copying to install directory: %s\n", err)
		return
	}

	fmt.Printf("Installed to %s\n", installDir)
}

func downloadReleases(address string) ([]Release, error) {
	resp, err := http.Get(address)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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

	fmt.Printf("OS: %s\n", runtime.GOOS)
	fmt.Printf("Arch: %s\n", runtime.GOARCH)

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
	fmt.Println("Temporary directory created:", dir)
	return dir, nil
}

func downloadFile(url, dir string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	filename := filepath.Base(resp.Request.URL.Path)
	filePath := filepath.Join(dir, filename)

	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return filePath, err
}

func unzip(zipFile, destDir string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		fPath := filepath.Join(destDir, file.Name)

		if !strings.HasPrefix(fPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", fPath)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(fPath, os.ModePerm)
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
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to check for its error
		if closeErr := outFile.Close(); closeErr != nil {
			rc.Close() // Ignore the error from rc.Close() as we're already handling an error
			return closeErr
		}
		rc.Close()

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
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

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

		fmt.Printf("Unarchiving %s\n", path)

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
				outFile.Close()
				return err
			}
			// ensure mode is taken from source file
			if err := outFile.Chmod(os.FileMode(header.Mode)); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

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

func getInstallDir() (string, error) {
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
