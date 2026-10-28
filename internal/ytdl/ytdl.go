package ytdl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

var youtubeRegexp = regexp.MustCompile(`(?i)^(https?://)?(www\.|m\.)?(youtube\.com/(watch\?.*v=|shorts/|live/)|youtu\.be/)[a-zA-Z0-9_-]+`)

// IsYouTubeURL checks if the provided URL is a valid YouTube video/shorts/live URL.
func IsYouTubeURL(rawURL string) bool {
	return youtubeRegexp.MatchString(rawURL)
}

// FormatOption represents a selectable resolution or audio track for the user.
type FormatOption struct {
	FormatID   string `json:"format_id"`
	Resolution string `json:"resolution"`
	Height     int    `json:"height"`
	Ext        string `json:"ext"`
	Filesize   int64  `json:"filesize"`
	IsAudio    bool   `json:"is_audio"`
	HasAudio   bool   `json:"has_audio"`
}

// VideoMetadata contains extracted metadata and selectable download formats.
type VideoMetadata struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Thumbnail   string         `json:"thumbnail"`
	Duration    int            `json:"duration"`
	DurationStr string         `json:"duration_str"`
	Uploader    string         `json:"uploader"`
	Options     []FormatOption `json:"options"`
}

// GetBinaryPath returns the path to yt-dlp, checking PATH first then ~/.candak/bin.
func GetBinaryPath() (string, error) {
	if p, err := exec.LookPath("yt-dlp"); err == nil {
		return p, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home directory not found: %w", err)
	}

	binName := "yt-dlp"
	if runtime.GOOS == "windows" {
		binName = "yt-dlp.exe"
	}

	localPath := filepath.Join(homeDir, ".candak", "bin", binName)
	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	return "", fmt.Errorf("yt-dlp not found")
}

// EnsureBinary checks if yt-dlp is available, downloading it to ~/.candak/bin if missing.
func EnsureBinary(ctx context.Context) (string, error) {
	if p, err := GetBinaryPath(); err == nil {
		return p, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	binDir := filepath.Join(homeDir, ".candak", "bin")
	_ = os.MkdirAll(binDir, 0755)

	binName := "yt-dlp"
	downloadURL := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp"
	if runtime.GOOS == "windows" {
		binName = "yt-dlp.exe"
		downloadURL = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	}

	targetPath := filepath.Join(binDir, binName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download yt-dlp: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("yt-dlp download returned status %d", resp.StatusCode)
	}

	outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create yt-dlp binary: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write yt-dlp binary: %w", err)
	}

	return targetPath, nil
}

type rawFormat struct {
	FormatID   string  `json:"format_id"`
	Ext        string  `json:"ext"`
	Height     int     `json:"height"`
	Filesize   int64   `json:"filesize"`
	FilesizeA  int64   `json:"filesize_approx"`
	VCodec     string  `json:"vcodec"`
	ACodec     string  `json:"acodec"`
	Tbr        float64 `json:"tbr"`
	FormatNote string  `json:"format_note"`
}

type rawJSON struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	Thumbnail string      `json:"thumbnail"`
	Duration  int         `json:"duration"`
	Uploader  string      `json:"uploader"`
	Formats   []rawFormat `json:"formats"`
}

// FetchMetadata invokes yt-dlp to extract clean metadata and selectable format options.
func FetchMetadata(ctx context.Context, rawURL string) (*VideoMetadata, error) {
	binPath, err := EnsureBinary(ctx)
	if err != nil {
		return nil, fmt.Errorf("yt-dlp binary error: %w", err)
	}

	args := []string{"--no-playlist", "--dump-json"}
	// If node or deno is available, enable js runtime for faster/reliable cipher solving
	if nodePath, err := exec.LookPath("node"); err == nil {
		args = append(args, "--js-runtimes", fmt.Sprintf("node:%s", nodePath))
	} else if denoPath, err := exec.LookPath("deno"); err == nil {
		args = append(args, "--js-runtimes", fmt.Sprintf("deno:%s", denoPath))
	}
	args = append(args, rawURL)

	cmd := exec.CommandContext(ctx, binPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("yt-dlp failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var raw rawJSON
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse yt-dlp output: %w", err)
	}

	// Group and deduplicate formats to present clean choices
	videoByHeight := make(map[int]rawFormat)
	var bestAudio *rawFormat

	for _, f := range raw.Formats {
		hasVideo := f.VCodec != "none" && f.VCodec != "" && f.Height > 0
		hasAudio := f.ACodec != "none" && f.ACodec != ""

		if hasVideo {
			existing, ok := videoByHeight[f.Height]
			if !ok || f.Tbr > existing.Tbr {
				videoByHeight[f.Height] = f
			}
		} else if hasAudio {
			if bestAudio == nil || f.Tbr > bestAudio.Tbr {
				c := f
				bestAudio = &c
			}
		}
	}

	var options []FormatOption

	// Add video resolutions descending
	var heights []int
	for h := range videoByHeight {
		heights = append(heights, h)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(heights)))

	for _, h := range heights {
		f := videoByHeight[h]
		label := fmt.Sprintf("%dp", h)
		if h >= 2160 {
			label = "2160p (4K)"
		} else if h >= 1440 {
			label = "1440p (2K)"
		} else if h == 1080 {
			label = "1080p (Full HD)"
		} else if h == 720 {
			label = "720p (HD)"
		}

		sz := f.Filesize
		if sz <= 0 {
			sz = f.FilesizeA
		}

		hasAudio := f.ACodec != "none" && f.ACodec != ""
		formatSelector := f.FormatID
		if !hasAudio {
			// Auto-merge with best audio
			formatSelector = fmt.Sprintf("%s+bestaudio/best", f.FormatID)
		}

		options = append(options, FormatOption{
			FormatID:   formatSelector,
			Resolution: label,
			Height:     h,
			Ext:        "mp4",
			Filesize:   sz,
			IsAudio:    false,
			HasAudio:   true,
		})
	}

	// Add audio option
	if bestAudio != nil {
		sz := bestAudio.Filesize
		if sz <= 0 {
			sz = bestAudio.FilesizeA
		}
		options = append(options, FormatOption{
			FormatID:   bestAudio.FormatID,
			Resolution: "Audio Only (M4A / Best)",
			Height:     0,
			Ext:        "m4a",
			Filesize:   sz,
			IsAudio:    true,
			HasAudio:   true,
		})
	}

	durationStr := ""
	if raw.Duration > 0 {
		m := raw.Duration / 60
		s := raw.Duration % 60
		if m >= 60 {
			h := m / 60
			m = m % 60
			durationStr = fmt.Sprintf("%d:%02d:%02d", h, m, s)
		} else {
			durationStr = fmt.Sprintf("%d:%02d", m, s)
		}
	}

	return &VideoMetadata{
		ID:          raw.ID,
		Title:       raw.Title,
		Thumbnail:   raw.Thumbnail,
		Duration:    raw.Duration,
		DurationStr: durationStr,
		Uploader:    raw.Uploader,
		Options:     options,
	}, nil
}

// ProgressFunc is invoked periodically as download chunks arrive.
type ProgressFunc func(percent float64, speed int64, downloaded int64, total int64, eta int64)

// DownloadVideo executes yt-dlp to download the chosen format to the target output directory.
func DownloadVideo(ctx context.Context, rawURL, formatID, outPath string, onProgress ProgressFunc) error {
	binPath, err := EnsureBinary(ctx)
	if err != nil {
		return err
	}

	args := []string{
		"--no-playlist",
		"-f", formatID,
		"--newline",
		"--progress-template", "candak:%(progress._percent_str)s:%(progress.downloaded_bytes)s:%(progress.total_bytes_estimate)s:%(progress.speed)s:%(progress.eta)s",
		"-o", outPath,
	}

	// Add ffmpeg location if present
	binDir := filepath.Dir(binPath)
	ffmpegName := "ffmpeg"
	if runtime.GOOS == "windows" {
		ffmpegName = "ffmpeg.exe"
	}
	if _, err := os.Stat(filepath.Join(binDir, ffmpegName)); err == nil {
		args = append(args, "--ffmpeg-location", binDir)
	} else if p, err := exec.LookPath("ffmpeg"); err == nil {
		args = append(args, "--ffmpeg-location", filepath.Dir(p))
	}

	if nodePath, err := exec.LookPath("node"); err == nil {
		args = append(args, "--js-runtimes", fmt.Sprintf("node:%s", nodePath))
	} else if denoPath, err := exec.LookPath("deno"); err == nil {
		args = append(args, "--js-runtimes", fmt.Sprintf("deno:%s", denoPath))
	}

	args = append(args, rawURL)

	cmd := exec.CommandContext(ctx, binPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	if onProgress != nil {
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if !strings.HasPrefix(line, "candak:") {
					continue
				}
				parts := strings.Split(line, ":")
				if len(parts) < 6 {
					continue
				}
				// percent
				rawPct := strings.TrimSuffix(strings.TrimSpace(parts[1]), "%")
				pct, _ := strconv.ParseFloat(rawPct, 64)

				// downloaded bytes
				down, _ := strconv.ParseInt(parts[2], 10, 64)

				// total bytes
				tot, _ := strconv.ParseInt(parts[3], 10, 64)

				// speed
				spd, _ := strconv.ParseInt(parts[4], 10, 64)

				// eta
				eta, _ := strconv.ParseInt(parts[5], 10, 64)

				onProgress(pct, spd, down, tot, eta)
			}
		}()
	}

	return cmd.Wait()
}

// CleanVideoTitle sanitizes a video title for safe filesystem use.
func CleanVideoTitle(title string) string {
	invalid := regexp.MustCompile(`[\\/:*?"<>|]`)
	cleaned := invalid.ReplaceAllString(title, "")
	cleaned = strings.TrimSpace(cleaned)
	if len(cleaned) > 120 {
		cleaned = cleaned[:120]
	}
	if cleaned == "" {
		cleaned = "youtube_video"
	}
	return cleaned
}
