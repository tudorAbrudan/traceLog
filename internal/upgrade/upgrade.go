package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
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

const (
	owner = "tudorAbrudan"
	repo  = "tracelog"
)

type release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// Run downloads the latest release binary and replaces the current executable.
// currentVersion is the running binary version (e.g. "v0.1.0" or "dev").
func Run(currentVersion string) error {
	client := &http.Client{Timeout: 5 * time.Minute}

	rel, err := fetchLatest(client)
	if err != nil {
		return err
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	cur := strings.TrimPrefix(strings.TrimSpace(currentVersion), "v")
	if cur != "" && cur != "dev" && latest == cur {
		fmt.Printf("Already at latest version: %s\n", rel.TagName)
		return nil
	}

	want := fmt.Sprintf("tracelog_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	var archiveURL, checksumURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case want:
			archiveURL = a.URL
		case "checksums.txt":
			checksumURL = a.URL
		}
	}
	if archiveURL == "" {
		return fmt.Errorf("no release asset %q for this platform (%s/%s)", want, runtime.GOOS, runtime.GOARCH)
	}
	if checksumURL == "" {
		return fmt.Errorf("release missing checksums.txt")
	}

	sumData, err := downloadBytes(client, checksumURL)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	wantHash, err := parseChecksum(sumData, want)
	if err != nil {
		return err
	}

	tgz, err := downloadBytes(client, archiveURL)
	if err != nil {
		return fmt.Errorf("download release: %w", err)
	}
	if hex.EncodeToString(sha256sum(tgz)) != wantHash {
		return fmt.Errorf("checksum mismatch for %s", want)
	}

	bin, err := extractTracelogBinary(tgz)
	if err != nil {
		return fmt.Errorf("extract binary: %w", err)
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	dir := filepath.Dir(self)
	tmp, err := os.CreateTemp(dir, "tracelog-upgrade-*")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	tmp.Close()
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("chmod: %w", err)
	}

	if err := os.Rename(tmpPath, self); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace binary: %w (try: sudo tracelog upgrade)", err)
	}

	fmt.Printf("Upgraded TraceLog %s -> %s\nRestart the service if needed: sudo systemctl restart tracelog\n", currentVersion, rel.TagName)
	return nil
}

func fetchLatest(client *http.Client) (*release, error) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo), nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "tracelog-upgrade")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, fmt.Errorf("github API: %s: %s", res.Status, strings.TrimSpace(string(b)))
	}
	var rel release
	if err := json.NewDecoder(res.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("invalid release response")
	}
	return &rel, nil
}

func downloadBytes(client *http.Client, url string) ([]byte, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "tracelog-upgrade")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", res.Status)
	}
	return io.ReadAll(res.Body)
}

func parseChecksum(data []byte, filename string) (string, error) {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		hash := parts[0]
		name := parts[len(parts)-1]
		if strings.TrimPrefix(name, "*") == filename {
			return strings.ToLower(hash), nil
		}
	}
	return "", fmt.Errorf("no checksum line for %s", filename)
}

func sha256sum(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

func extractTracelogBinary(tgz []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(tgz))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		base := filepath.Base(hdr.Name)
		if base != "tracelog" {
			continue
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != 0 {
			continue
		}
		if hdr.Size <= 0 || hdr.Size > 200<<20 {
			continue
		}
		return io.ReadAll(io.LimitReader(tr, hdr.Size))
	}
	return nil, fmt.Errorf("tracelog binary not found in archive")
}
