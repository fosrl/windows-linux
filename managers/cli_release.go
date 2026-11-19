package managers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// The CLI install on each platform asks the Fossorial API for the latest CLI
// version and its assets' checksums, as the CLI's install script (get-cli.sh)
// does, then downloads the asset from that version's GitHub release.
const (
	cliReleaseAPIURL      = "https://api.fossorial.io/api/v1/versions/cli"
	cliReleaseDownloadURL = "https://github.com/fosrl/cli/releases/download"
	cliInstallerUserAgent = "pangolin-client-cli-installer"
)

// cliVersionPattern matches the release versions get-cli.sh accepts, so a
// version can't change the shape of the download URL.
var cliVersionPattern = regexp.MustCompile(`^[0-9A-Za-z.+-]+$`)

type cliRelease struct {
	Version      string
	AssetDigests map[string]string
}

type cliReleaseResponse struct {
	Data struct {
		LatestVersion string            `json:"latestVersion"`
		AssetDigests  map[string]string `json:"assetDigests"`
	} `json:"data"`
}

func getLatestCLIRelease() (*cliRelease, error) {
	req, err := http.NewRequest(http.MethodGet, cliReleaseAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create release request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", cliInstallerUserAgent)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get the latest CLI version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("CLI version API returned %d: %s", resp.StatusCode, string(body))
	}

	var body cliReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode the CLI version response: %w", err)
	}
	version := strings.TrimPrefix(body.Data.LatestVersion, "v")
	if !cliVersionPattern.MatchString(version) {
		return nil, fmt.Errorf("CLI version API returned an invalid version %q", body.Data.LatestVersion)
	}
	return &cliRelease{Version: version, AssetDigests: body.Data.AssetDigests}, nil
}

// asset returns the download URL and published sha256 of the release asset
// called name. An asset without a published checksum isn't installed.
func (r *cliRelease) asset(name string) (url, digest string, err error) {
	digest = strings.ToLower(r.AssetDigests[name])
	if len(digest) != sha256.Size*2 {
		return "", "", fmt.Errorf("CLI %s has no published checksum for %s", r.Version, name)
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", "", fmt.Errorf("CLI %s has an invalid checksum for %s", r.Version, name)
	}
	return fmt.Sprintf("%s/%s/%s", cliReleaseDownloadURL, r.Version, name), digest, nil
}

// downloadCLIAsset saves url to destPath, which it creates, and checks it
// against the published sha256. A file that doesn't match is removed.
func downloadCLIAsset(url, destPath, digest string) (err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", cliInstallerUserAgent)

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download the CLI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("CLI download failed with %d: %s", resp.StatusCode, string(body))
	}

	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create download file: %w", err)
	}
	defer func() {
		out.Close()
		if err != nil {
			os.Remove(destPath)
		}
	}()

	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, hash), resp.Body); err != nil {
		return fmt.Errorf("failed to write download file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to write download file: %w", err)
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != digest {
		return errors.New("the downloaded CLI doesn't match its published checksum; it may be corrupted or tampered with")
	}
	return nil
}
