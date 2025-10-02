package wikimedia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Event is the minimal representation returned to callers.
type Event struct {
	Year int    `json:"year"`
	Text string `json:"text"`
}

// Client provides fetching with an on-disk TTL cache.
type Client struct {
	cacheDir string
	ttl      time.Duration
	client   *http.Client
}

// NewClient creates a new Wikimedia client.
// If cacheDir is empty it defaults to "./.cache/wikimedia".
// ttl controls how long cached responses are considered fresh.
func NewClient(cacheDir string, ttl time.Duration) *Client {
	if cacheDir == "" {
		cacheDir = filepath.Join(".", ".cache", "wikimedia")
	}
	_ = os.MkdirAll(cacheDir, 0o755)

	return &Client{
		cacheDir: cacheDir,
		ttl:      ttl,
		client: &http.Client{
			// Do not set Timeout here; callers should use context with timeout.
			Timeout: 0,
		},
	}
}

// FetchOnThisDay fetches events for the given month and day (MM, DD).
// If bypassCache is false, a fresh cached response (modtime within TTL) will be used.
func (c *Client) FetchOnThisDay(ctx context.Context, month, day string, bypassCache bool) ([]Event, error) {
	if month == "" || day == "" {
		return nil, fmt.Errorf("month and day required")
	}

	cacheFile := filepath.Join(c.cacheDir, fmt.Sprintf("onthisday_%s_%s.json", month, day))

	// Try cache
	if !bypassCache {
		if fi, err := os.Stat(cacheFile); err == nil {
			if time.Since(fi.ModTime()) <= c.ttl {
				if data, err := os.ReadFile(cacheFile); err == nil {
					evs, err := parseEventsFromBody(data)
					if err == nil {
						return evs, nil
					}
					// fallthrough to refetch on parse error
				}
			}
		}
	}

	// Build URL
	url := fmt.Sprintf("https://api.wikimedia.org/feed/v1/wikipedia/en/onthisday/all/%s/%s", month, day)

	// Retry strategy
	const maxAttempts = 3
	backoff := 500 * time.Millisecond

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Respect parent context
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Go Day-in-History BBS Door/1.0 (github.com/robbiew/history)")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Accept-Encoding", "identity")

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("network error: %v", err)
			// retry with jitter unless context cancelled or last attempt
			if attempt < maxAttempts {
				if err := sleepContext(ctx, backoff); err != nil {
					return nil, err
				}
				backoff *= 2
				continue
			}
			return nil, lastErr
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %v", err)
			if attempt < maxAttempts {
				if err := sleepContext(ctx, backoff); err != nil {
					return nil, err
				}
				backoff *= 2
				continue
			}
			return nil, lastErr
		}

		// Handle success
		if resp.StatusCode == http.StatusOK {
			evs, err := parseEventsFromBody(body)
			if err != nil {
				return nil, err
			}

			// Best-effort cache write (atomic)
			_ = writeCacheFileAtomic(cacheFile, body)

			return evs, nil
		}

		// Retry on 429 or 5xx
		if resp.StatusCode == http.StatusTooManyRequests || (resp.StatusCode >= 500 && resp.StatusCode < 600) {
			lastErr = fmt.Errorf("API returned status code: %d", resp.StatusCode)
			if attempt < maxAttempts {
				if err := sleepContext(ctx, backoff); err != nil {
					return nil, err
				}
				// add jitter
				backoff *= 2
				continue
			}
			return nil, lastErr
		}

		// Non-retryable error: include body for diagnostics
		return nil, fmt.Errorf("API returned status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("failed to fetch events: %v", lastErr)
}

// parseEventsFromBody extracts the "events" array from the Wikimedia API payload.
func parseEventsFromBody(body []byte) ([]Event, error) {
	var apiResp struct {
		Events []struct {
			Year int    `json:"year"`
			Text string `json:"text"`
		} `json:"events"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	out := make([]Event, 0, len(apiResp.Events))
	for _, e := range apiResp.Events {
		out = append(out, Event{Year: e.Year, Text: e.Text})
	}
	return out, nil
}

// writeCacheFileAtomic writes data to a temp file and renames it into place.
func writeCacheFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dir, "tmp-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	// Rename is atomic on most platforms
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// sleepContext sleeps for the given duration but returns early if ctx is cancelled.
func sleepContext(ctx context.Context, d time.Duration) error {
	// Add small jitter
	jitter := time.Duration(rand.Int63n(int64(200*time.Millisecond))) - 100*time.Millisecond
	if jitter < 0 {
		jitter = 0
	}
	select {
	case <-time.After(d + jitter):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}