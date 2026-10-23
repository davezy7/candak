package prober

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"candak/internal/models"
)

// URLProber inspects a remote URL to gather size, resumability, and filename metadata.
type URLProber interface {
	// Probe sends an inspection request and returns the ProbeResult.
	Probe(ctx context.Context, rawURL string) (*models.ProbeResult, error)
	// ProbeWithHeaders sends an inspection request with custom headers and returns the ProbeResult.
	ProbeWithHeaders(ctx context.Context, rawURL string, headers map[string]string) (*models.ProbeResult, error)
}

// HTTPProber implements URLProber using standard HTTP client requests.
type HTTPProber struct {
	client    *http.Client
	userAgent string
}

// Option configures an HTTPProber.
type Option func(*HTTPProber)

// WithHTTPClient overrides the default HTTP client used by HTTPProber.
func WithHTTPClient(client *http.Client) Option {
	return func(p *HTTPProber) {
		p.client = client
	}
}

// WithUserAgent sets a custom User-Agent string for probe requests.
func WithUserAgent(ua string) Option {
	return func(p *HTTPProber) {
		p.userAgent = ua
	}
}

// NewHTTPProber constructs an HTTPProber with configurable options.
func NewHTTPProber(opts ...Option) *HTTPProber {
	p := &HTTPProber{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		userAgent: "Candak-Download-Manager/1.0 (Mozilla/5.0)",
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Probe inspects a remote URL via HEAD or range GET fallback without custom headers.
func (p *HTTPProber) Probe(ctx context.Context, rawURL string) (*models.ProbeResult, error) {
	return p.ProbeWithHeaders(ctx, rawURL, nil)
}

// ProbeWithHeaders inspects a remote URL via HEAD or range GET fallback with optional headers.
func (p *HTTPProber) ProbeWithHeaders(ctx context.Context, rawURL string, headers map[string]string) (*models.ProbeResult, error) {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrInvalidURL, err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("%w: scheme must be http or https", models.ErrInvalidURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", p.userAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	// If HEAD request fails or receives an error status code, fallback to range GET (bytes=0-0)
	if err != nil || resp.StatusCode >= 400 {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}

		getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
		if err != nil {
			return nil, err
		}
		getReq.Header.Set("User-Agent", p.userAgent)
		getReq.Header.Set("Range", "bytes=0-0")
		for k, v := range headers {
			if strings.EqualFold(k, "Range") {
				continue
			}
			getReq.Header.Set(k, v)
		}

		resp, err = p.client.Do(getReq)
		if err != nil {
			return nil, fmt.Errorf("probe request failed: %w", err)
		}
		defer resp.Body.Close()
	} else {
		defer resp.Body.Close()
	}

	totalSize := resp.ContentLength
	resumable := false

	if cr := resp.Header.Get("Content-Range"); cr != "" {
		resumable = true
		// Content-Range: bytes 0-0/104857600
		if slash := strings.LastIndex(cr, "/"); slash != -1 && slash+1 < len(cr) {
			sizeStr := strings.TrimSpace(cr[slash+1:])
			if sizeStr != "*" {
				if parsedSize, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && parsedSize > 0 {
					totalSize = parsedSize
				}
			}
		}
	} else if resp.Header.Get("Accept-Ranges") == "bytes" {
		resumable = true
	}

	filename := extractFilename(resp.Header.Get("Content-Disposition"), parsedURL.Path)
	filename = SanitizeFilename(filename)

	return &models.ProbeResult{
		URL:         rawURL,
		Filename:    filename,
		TotalSize:   totalSize,
		Resumable:   resumable,
		ContentType: resp.Header.Get("Content-Type"),
		Headers:     headers,
	}, nil
}

func extractFilename(contentDisposition, urlPath string) string {
	if contentDisposition != "" {
		if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
			if fn, ok := params["filename*"]; ok {
				parts := strings.SplitN(fn, "''", 2)
				if len(parts) == 2 {
					if unesc, err := url.PathUnescape(parts[1]); err == nil {
						return unesc
					}
				}
			}
			if fn, ok := params["filename"]; ok && fn != "" {
				return fn
			}
		}
	}

	base := filepath.Base(urlPath)
	if base == "." || base == "/" || base == "" {
		return "download.bin"
	}
	if unesc, err := url.PathUnescape(base); err == nil {
		return unesc
	}
	return base
}

// SanitizeFilename cleans invalid filesystem characters from filenames.
func SanitizeFilename(name string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	clean := name
	for _, c := range invalidChars {
		clean = strings.ReplaceAll(clean, c, "_")
	}
	clean = strings.TrimSpace(clean)
	if clean == "" || clean == "." || clean == ".." || strings.HasPrefix(clean, "..") {
		return "download.bin"
	}
	return clean
}
