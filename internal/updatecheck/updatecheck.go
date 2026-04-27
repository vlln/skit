package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vlln/skit/internal/xdg"
)

const (
	defaultLatestURL = "https://api.github.com/repos/vlln/skit/releases/latest"
	cacheTTL         = 24 * time.Hour
)

type Request struct {
	Current string
	Force   bool
}

type Result struct {
	Current   string
	Latest    string
	Available bool
	URL       string
}

type releaseResponse struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

type cacheFile struct {
	CheckedAt int64  `json:"checkedAt"`
	Latest    string `json:"latest"`
	URL       string `json:"url"`
}

func Check(ctx context.Context, req Request) (Result, error) {
	current := strings.TrimSpace(req.Current)
	result := Result{Current: current}
	if current == "" {
		return result, nil
	}
	if !req.Force && disabled() {
		return result, nil
	}
	if !req.Force && strings.Contains(current, "dev") {
		return result, nil
	}
	if !req.Force {
		if cached, ok := readFreshCache(); ok {
			result.Latest = cached.Latest
			result.URL = cached.URL
			result.Available = isNewer(cached.Latest, current)
			return result, nil
		}
	}

	latest, url, err := fetchLatest(ctx)
	if err != nil {
		return result, err
	}
	result.Latest = latest
	result.URL = url
	result.Available = isNewer(latest, current)
	writeCache(cacheFile{CheckedAt: time.Now().Unix(), Latest: latest, URL: url})
	return result, nil
}

func Message(result Result) string {
	url := result.URL
	if url == "" {
		url = "https://github.com/vlln/skit/releases/latest"
	}
	return fmt.Sprintf("update available: skit %s is available (current %s); see %s", result.Latest, result.Current, url)
}

func disabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SKIT_UPDATE_CHECK"))) {
	case "0", "false", "no", "off":
		return true
	default:
		return false
	}
}

func fetchLatest(ctx context.Context) (string, string, error) {
	url := strings.TrimSpace(os.Getenv("SKIT_UPDATE_CHECK_URL"))
	if url == "" {
		url = defaultLatestURL
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "skit")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", "", fmt.Errorf("update check failed: %s", resp.Status)
	}
	var release releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}
	latest := strings.TrimSpace(release.TagName)
	if latest == "" {
		latest = strings.TrimSpace(release.Name)
	}
	if latest == "" {
		return "", "", fmt.Errorf("update check response did not include a version")
	}
	return latest, strings.TrimSpace(release.HTMLURL), nil
}

func readFreshCache() (cacheFile, bool) {
	var cached cacheFile
	raw, err := os.ReadFile(cachePath())
	if err != nil {
		return cached, false
	}
	if err := json.Unmarshal(raw, &cached); err != nil {
		return cached, false
	}
	if cached.Latest == "" || time.Since(time.Unix(cached.CheckedAt, 0)) > cacheTTL {
		return cached, false
	}
	return cached, true
}

func writeCache(cached cacheFile) {
	path := cachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	raw, err := json.Marshal(cached)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, raw, 0644)
}

func cachePath() string {
	if path := strings.TrimSpace(os.Getenv("SKIT_UPDATE_CHECK_CACHE")); path != "" {
		return path
	}
	return filepath.Join(xdg.CacheHome(), "skit", "update-check.json")
}

func isNewer(latest, current string) bool {
	latestParts, okLatest := versionParts(latest)
	currentParts, okCurrent := versionParts(current)
	if !okLatest || !okCurrent {
		return latest != "" && latest != current
	}
	for i := 0; i < len(latestParts); i++ {
		if latestParts[i] != currentParts[i] {
			return latestParts[i] > currentParts[i]
		}
	}
	return false
}

func versionParts(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.SplitN(v, "-", 2)[0]
	parts := strings.Split(v, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, false
	}
	for i, part := range parts {
		if part == "" {
			return out, false
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
