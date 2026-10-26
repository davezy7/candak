// Package updater checks GitHub Releases for newer Candak versions and
// resolves the release asset matching the current platform.
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Version is the compiled application version.
// Keep in sync with wails.json info.productVersion.
const Version = "0.2.0"

const releasesAPI = "https://api.github.com/repos/davezy7/candak/releases/latest"

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Asset is a downloadable release artifact.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Release is the subset of the GitHub release payload Candak consumes.
type Release struct {
	TagName     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	Assets      []Asset   `json:"assets"`
}

// CheckResult summarizes an update check for the frontend.
type CheckResult struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	PublishedAt     string `json:"publishedAt,omitempty"`
	ReleaseNotes    string `json:"releaseNotes,omitempty"`
	ReleaseURL      string `json:"releaseUrl,omitempty"`
	AssetName       string `json:"assetName,omitempty"`
	AssetURL        string `json:"assetUrl,omitempty"`
	AssetSize       int64  `json:"assetSize,omitempty"`
}

// Check queries the latest GitHub release and compares it against the
// compiled version, resolving the installer asset for the current platform.
func Check(ctx context.Context) (*CheckResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no published release found")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("failed to decode release payload: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("release payload missing tag_name")
	}

	latest := strings.TrimSpace(rel.TagName)
	latest = strings.TrimPrefix(latest, "v")
	latest = strings.TrimPrefix(latest, "V")
	result := &CheckResult{
		CurrentVersion:  Version,
		LatestVersion:   latest,
		UpdateAvailable: compareSemver(latest, Version) > 0,
		PublishedAt:     rel.PublishedAt.UTC().Format(time.RFC3339),
		ReleaseNotes:    rel.Body,
		ReleaseURL:      rel.HTMLURL,
	}
	if asset := pickAsset(rel.Assets, runtime.GOOS, runtime.GOARCH); asset != nil {
		result.AssetName = asset.Name
		result.AssetURL = asset.BrowserDownloadURL
		result.AssetSize = asset.Size
	}
	return result, nil
}

// pickAsset selects the release asset matching the platform and architecture.
// Windows prefers the installer exe; macOS prefers the dmg for the exact arch.
func pickAsset(assets []Asset, goos, arch string) *Asset {
	switch goos {
	case "windows":
		var installer, plain *Asset
		for i := range assets {
			name := strings.ToLower(assets[i].Name)
			if !strings.HasSuffix(name, ".exe") {
				continue
			}
			if strings.Contains(name, "installer") && installer == nil {
				installer = &assets[i]
			}
			if plain == nil {
				plain = &assets[i]
			}
		}
		if installer != nil {
			return installer
		}
		return plain
	case "darwin":
		var exact, anyDmg *Asset
		for i := range assets {
			name := strings.ToLower(assets[i].Name)
			if !strings.HasSuffix(name, ".dmg") {
				continue
			}
			if anyDmg == nil {
				anyDmg = &assets[i]
			}
			if strings.Contains(name, arch) && exact == nil {
				exact = &assets[i]
			}
		}
		if exact != nil {
			return exact
		}
		return anyDmg
	default:
		return nil
	}
}

// compareSemver compares dotted numeric versions, ignoring a leading "v" and
// any pre-release/metadata suffix. Returns -1 if a < b, 0 if equal, 1 if a > b.
func compareSemver(a, b string) int {
	pa, pb := semverParts(a), semverParts(b)
	for i := range pa {
		if pa[i] != pb[i] {
			if pa[i] < pb[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func semverParts(v string) [3]int {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var parts [3]int
	for i, s := range strings.SplitN(v, ".", 3) {
		if i >= 3 {
			break
		}
		n := 0
		for _, r := range s {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		parts[i] = n
	}
	return parts
}
