package upgrader

import (
	"context"
	"fmt"
	"github.com/SymmetricalAI/symctl/internal/logger"
	"github.com/google/go-github/v61/github"
	"io"
	"net/http"
	"os"
	"runtime"
)

const (
	owner = "SymmetricalAI"
	repo  = "symctl"
)

func Upgrade(version string, dryRun bool) {
	logger.Debugf("Upgrade called with version: %s, dry-run: %v\n", version, dryRun)
	ctx := context.TODO()
	client := github.NewClient(nil)
	lrName, lrID := getLatestRelease(ctx, client)
	if lrName == version {
		fmt.Printf("symctl is already up to date\n")
		return
	}
	fmt.Printf("symctl %s -> %s\n", version, lrName)
	goos := getOS()
	goarch := getArch()
	to := getExecutableLocation()
	assetId := findAssetToDownload(ctx, client, lrID, lrName, goos, goarch)
	if dryRun {
		logger.Debugf("Dry run, skipping upgrade\n")
		return
	}
	from := downloadReleaseAsset(ctx, client, assetId)
	copyTempToExecutable(from, to)
}

func copyTempToExecutable(from, to string) {
	logger.Debugf("Copying %s to %s\n", from, to)
	err := os.Rename(from, to)
	if err != nil {
		logger.Fatalf("Error renaming temp file to executable: %v\n", err)
	}
}

func findAssetToDownload(ctx context.Context, client *github.Client, releaseID int64, releaseName, goos, goarch string) *int64 {
	assets, _, err := client.Repositories.ListReleaseAssets(ctx, owner, repo, releaseID, nil)
	if err != nil {
		logger.Fatalf("Error fetching release assets: %v\n", err)
	}
	var assetId *int64
	for _, asset := range assets {
		logger.Debugf("Asset: %v\n", *asset.Name)
		if *asset.Name == fmt.Sprintf("symctl-%s-%s-%s", releaseName, goos, goarch) {
			logger.Debugf("Found asset: %v\n", *asset.BrowserDownloadURL)
			logger.Debugf("Found asset ID: %v\n", *asset.ID)
			assetId = asset.ID
		}
	}
	if assetId == nil {
		logger.Debugf("No asset found for symctl-%s-%s-%s\n", releaseName, goos, goarch)
	}
	return assetId
}

func downloadReleaseAsset(ctx context.Context, client *github.Client, assetId *int64) string {
	f, err := os.CreateTemp("", "symctl-new-")
	if err != nil {
		logger.Fatalf("Error creating temp file: %v\n", err)
	}
	logger.Debugf("Temp file: %s\n", f.Name())
	rc, redirectURL, err := client.Repositories.DownloadReleaseAsset(ctx, owner, repo, *assetId, http.DefaultClient)
	if err != nil {
		logger.Fatalf("Error downloading asset: %v\n", err)
	}
	defer func(rc io.ReadCloser) {
		_ = rc.Close()
	}(rc)
	logger.Debugf("Redirect URL: %s\n", redirectURL)
	_, err = io.Copy(f, rc)
	if err != nil {
		logger.Fatalf("Error copying asset to temp file: %v\n", err)
	}
	err = os.Chmod(f.Name(), 0755)
	if err != nil {
		logger.Fatalf("Error changing temp file permissions: %v\n", err)
	}
	return f.Name()
}

func getLatestRelease(ctx context.Context, client *github.Client) (string, int64) {
	latestRelease, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		logger.Fatalf("Error fetching latest release: %v\n", err)
	}
	logger.Debugf("Latest release: %v\n", *latestRelease.Name)
	return *latestRelease.Name, *latestRelease.ID
}

func getExecutableLocation() string {
	ex, err := os.Executable()
	if err != nil {
		logger.Fatalf("Error getting binary location: %v\n", err)
	}
	logger.Debugf("Binary location: %s\n", ex)
	return ex
}

func getOS() string {
	goos := runtime.GOOS
	logger.Debugf("OS: %s\n", goos)
	return goos
}

func getArch() string {
	goarch := runtime.GOARCH
	logger.Debugf("Arch: %s\n", goarch)
	return goarch
}
